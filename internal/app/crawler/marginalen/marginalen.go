package marginalen

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/yama6a/bolan-compare/internal/app/crawler"
	"github.com/yama6a/bolan-compare/internal/pkg/http"
	"github.com/yama6a/bolan-compare/internal/pkg/model"
	"github.com/yama6a/bolan-compare/internal/pkg/utils"
	"go.uber.org/zap"
)

const (
	// Marginalen uses Episerver CMS with a content delivery API.
	// The Vue.js frontend fetches content from this API endpoint.
	marginalenAPIURL   = "https://www.marginalen.se/api/episerver/v3.0/content?contentUrl=%2Fprivat%2Fbanktjanster%2Flan%2Fflytta-eller-utoka-bolan%2Fgenomsnittlig-bolaneranta%2F&matchExact=true&expand=*"
	marginalenBankName = model.Bank("Marginalen Bank")
)

var (
	// marginalenRateRegex matches rates like "5,92 %" or "6.35%".
	marginalenRateRegex = regexp.MustCompile(`^(\d+[,.]?\d*)\s*%?$`)
	// marginalenPeriodRegex matches "YYYYMM" format like "202412".
	marginalenPeriodRegex = regexp.MustCompile(`^(\d{4})(\d{2})$`)
)

var _ crawler.SiteCrawler = &MarginalenCrawler{}

// episerverResponse represents the Episerver CMS API response structure.
type episerverResponse []struct {
	MainContentArea []struct {
		MainContentArea []struct {
			Body string `json:"body"`
		} `json:"mainContentArea"`
	} `json:"mainContentArea"`
}

//nolint:revive // Bank name prefix is intentional for clarity
type MarginalenCrawler struct {
	httpClient http.Client
	logger     *zap.Logger
}

func NewMarginalenCrawler(httpClient http.Client, logger *zap.Logger) *MarginalenCrawler {
	return &MarginalenCrawler{httpClient: httpClient, logger: logger}
}

func (c *MarginalenCrawler) Crawl(channel chan<- model.InterestSet) {
	crawlTime := time.Now().UTC()

	// Note: Marginalen Bank does not publish list rates per term.
	// They only publish a rate range (4,41% - 10,50%) which varies based on
	// individual credit assessment. Therefore, we only extract average rates.
	//
	// Marginalen Bank uses Vue.js with Episerver CMS. The website's frontend
	// fetches content via the Episerver Content Delivery API, which returns
	// JSON with HTML embedded in the "body" field.

	jsonData, err := c.httpClient.Fetch(marginalenAPIURL, nil)
	if err != nil {
		c.logger.Error("failed fetching Marginalen API", zap.Error(err))
		return
	}

	// Parse the Episerver API JSON response
	var apiResponse episerverResponse
	if err := json.Unmarshal([]byte(jsonData), &apiResponse); err != nil {
		c.logger.Error("failed parsing Marginalen API JSON", zap.Error(err))
		return
	}

	// Extract the HTML content from the JSON structure
	htmlContent, err := c.extractHTMLFromAPI(apiResponse)
	if err != nil {
		c.logger.Error("failed extracting HTML from Marginalen API response", zap.Error(err))
		return
	}

	avgRates, err := c.extractAverageRates(htmlContent, crawlTime)
	if err != nil {
		c.logger.Error("failed parsing Marginalen average rates", zap.Error(err))
		return
	}

	for _, set := range avgRates {
		channel <- set
	}
}

// extractHTMLFromAPI extracts the HTML body content from Episerver API response.
// The API returns JSON with nested structure: [0].mainContentArea[0].mainContentArea[0].body.
func (c *MarginalenCrawler) extractHTMLFromAPI(response episerverResponse) (string, error) {
	if len(response) == 0 {
		return "", fmt.Errorf("empty API response")
	}

	if len(response[0].MainContentArea) == 0 {
		return "", fmt.Errorf("no main content area in API response")
	}

	if len(response[0].MainContentArea[0].MainContentArea) == 0 {
		return "", fmt.Errorf("no content blocks in API response")
	}

	htmlBody := response[0].MainContentArea[0].MainContentArea[0].Body
	if htmlBody == "" {
		return "", fmt.Errorf("empty HTML body in API response")
	}

	return htmlBody, nil
}

// extractAverageRates parses average rates from Marginalen's HTML page.
// The table has columns: Månad | 3 Mån | 6 Mån | 1 år | 2 år | 3 år
// Missing values are shown as "-".
//
//nolint:cyclop // Table parsing with validation requires multiple conditionals
func (c *MarginalenCrawler) extractAverageRates(rawHTML string, crawlTime time.Time) ([]model.InterestSet, error) {
	// Find table by looking for "Genomsnittlig bolåneränta" heading
	tokenizer, err := utils.FindTokenizedTableByTextBeforeTable(rawHTML, "Genomsnittlig bolåneränta")
	if err != nil {
		return nil, fmt.Errorf("failed to find average rates table: %w", err)
	}

	table, err := utils.ParseTable(tokenizer)
	if err != nil {
		return nil, fmt.Errorf("failed to parse average rates table: %w", err)
	}

	if len(table.Header) == 0 {
		return nil, fmt.Errorf("average rates table has no header row")
	}

	// Extract terms from header (skip first column which is "Månad")
	terms, err := c.extractTermsFromHeader(table.Header[1:])
	if err != nil {
		return nil, fmt.Errorf("failed to extract terms from header: %w", err)
	}

	// Parse each data row
	interestSets := []model.InterestSet{}
	for _, row := range table.Rows {
		if len(row) < 2 {
			c.logger.Warn("skipping row with insufficient columns", zap.Int("columns", len(row)))
			continue
		}

		// First column is period (YYYYMM format)
		period := strings.TrimSpace(row[0])
		validFrom, err := c.parseMarginalenPeriod(period)
		if err != nil {
			c.logger.Warn("failed to parse period", zap.String("period", period), zap.Error(err))
			continue
		}

		// Process each term column
		for i, termStr := range row[1:] {
			if i >= len(terms) {
				break
			}

			// Skip missing values (shown as "-")
			trimmedRate := strings.TrimSpace(termStr)
			if trimmedRate == "-" || trimmedRate == "" {
				continue
			}

			rate, err := c.parseMarginalenRate(trimmedRate)
			if err != nil {
				c.logger.Warn("failed to parse rate",
					zap.String("rate", termStr),
					zap.String("period", period),
					zap.String("term", string(terms[i])),
					zap.Error(err))
				continue
			}

			avgMonth := &model.AvgMonth{
				Month: validFrom.Month(),
				Year:  uint(validFrom.Year()),
			}

			interestSets = append(interestSets, model.InterestSet{
				Bank:                  marginalenBankName,
				Type:                  model.TypeAverageRate,
				Term:                  terms[i],
				NominalRate:           float32(rate),
				LastCrawledAt:         crawlTime,
				AverageReferenceMonth: avgMonth,
			})
		}
	}

	if len(interestSets) == 0 {
		return nil, fmt.Errorf("no average rates found in table")
	}

	return interestSets, nil
}

// extractTermsFromHeader extracts term codes from header row.
// Header format: ["3 Mån", "6 Mån", "1 år", "2 år", "3 år"].
func (c *MarginalenCrawler) extractTermsFromHeader(headers []string) ([]model.Term, error) {
	terms := make([]model.Term, 0, len(headers))

	for _, header := range headers {
		termStr := strings.TrimSpace(header)
		term, err := utils.ParseTerm(termStr)
		if err != nil {
			c.logger.Warn("failed to parse term from header", zap.String("header", termStr), zap.Error(err))
			continue
		}
		terms = append(terms, term)
	}

	if len(terms) == 0 {
		return nil, fmt.Errorf("no valid terms found in header")
	}

	return terms, nil
}

// parseMarginalenRate parses a rate string like "5,92 %" or "6.35%" to a float64.
func (c *MarginalenCrawler) parseMarginalenRate(rateStr string) (float64, error) {
	trimmed := strings.TrimSpace(rateStr)
	matches := marginalenRateRegex.FindStringSubmatch(trimmed)
	if matches == nil {
		return 0, fmt.Errorf("rate does not match expected format: %s", rateStr)
	}

	// Replace comma with period for parsing
	normalized := strings.ReplaceAll(matches[1], ",", ".")
	rate, err := strconv.ParseFloat(normalized, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse rate as float: %w", err)
	}

	return rate, nil
}

// parseMarginalenPeriod parses period string "YYYYMM" to time.Time.
// Example: "202412" -> December 2024.
func (c *MarginalenCrawler) parseMarginalenPeriod(period string) (time.Time, error) {
	matches := marginalenPeriodRegex.FindStringSubmatch(period)
	if matches == nil {
		return time.Time{}, fmt.Errorf("period does not match YYYYMM format: %s", period)
	}

	year, err := strconv.Atoi(matches[1])
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse year: %w", err)
	}

	month, err := strconv.Atoi(matches[2])
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse month: %w", err)
	}

	if month < 1 || month > 12 {
		return time.Time{}, fmt.Errorf("invalid month: %d", month)
	}

	// Use first day of the month
	return time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC), nil
}
