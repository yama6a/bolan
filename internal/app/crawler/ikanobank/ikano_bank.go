package ikanobank

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
	ikanoBankName        model.Bank = "Ikano Bank"
	ikanoBankListRateURL string     = "https://ikanobank.se/api/interesttable/gettabledata"
	ikanoBankAvgRatesURL string     = "https://ikanobank.se/bolan/bolanerantor"
)

var (
	_ crawler.SiteCrawler = &IkanoBankCrawler{}

	// Ikano Bank average month format: "2025 01" or "2024 12".
	ikanoAvgMonthRegex = regexp.MustCompile(`^(\d{4})\s+(\d{2})$`)
)

// IkanoBankCrawler crawls Ikano Bank mortgage rates.
//
//nolint:revive // Bank name prefix is intentional for clarity
type IkanoBankCrawler struct {
	httpClient http.Client
	logger     *zap.Logger
}

// ikanoBankListRatesResponse represents the JSON response for list rates.
type ikanoBankListRatesResponse struct {
	Success  bool                    `json:"success"`
	ListData []ikanoBankListRateItem `json:"listData"`
}

type ikanoBankListRateItem struct {
	RateFixationPeriod    string `json:"rateFixationPeriod"`    // "3 mån", "1 år", etc.
	ListPriceInterestRate string `json:"listPriceInterestRate"` // "3.4800"
	EffectiveInterestRate string `json:"effectiveInterestRate"` // "3.5400"
}

// NewIkanoBankCrawler creates a new Ikano Bank crawler.
func NewIkanoBankCrawler(httpClient http.Client, logger *zap.Logger) *IkanoBankCrawler {
	return &IkanoBankCrawler{httpClient: httpClient, logger: logger}
}

// Crawl fetches Ikano Bank mortgage rates and sends them to the channel.
func (c *IkanoBankCrawler) Crawl(channel chan<- model.InterestSet) {
	crawlTime := time.Now().UTC()

	listRates, err := c.fetchListRates(crawlTime)
	if err != nil {
		c.logger.Error("failed fetching Ikano Bank list rates", zap.Error(err))
	}

	avgRates, err := c.fetchAverageRates(crawlTime)
	if err != nil {
		c.logger.Error("failed fetching Ikano Bank average rates", zap.Error(err))
	}

	for _, set := range append(listRates, avgRates...) {
		channel <- set
	}
}

func (c *IkanoBankCrawler) fetchListRates(crawlTime time.Time) ([]model.InterestSet, error) {
	rawJSON, err := c.httpClient.Fetch(ikanoBankListRateURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed reading Ikano Bank list rates API: %w", err)
	}

	var response ikanoBankListRatesResponse
	if err := json.Unmarshal([]byte(rawJSON), &response); err != nil {
		c.logger.Error("failed unmarshalling Ikano Bank list rates", zap.Error(err), zap.String("rawJSON", rawJSON))
		return nil, fmt.Errorf("failed unmarshalling Ikano Bank list rates: %w", err)
	}

	if !response.Success {
		return nil, fmt.Errorf("ikano bank API returned success=false")
	}

	interestSets := []model.InterestSet{}
	for _, item := range response.ListData {
		term, err := utils.ParseTerm(item.RateFixationPeriod)
		if err != nil {
			c.logger.Warn("Ikano Bank list rate term not supported - skipping",
				zap.String("term", item.RateFixationPeriod),
				zap.Error(err))
			continue
		}

		rate, err := parseIkanoBankRate(item.ListPriceInterestRate)
		if err != nil {
			c.logger.Warn("failed to parse Ikano Bank rate",
				zap.String("rate", item.ListPriceInterestRate),
				zap.Error(err))
			continue
		}

		interestSets = append(interestSets, model.InterestSet{
			Bank:          ikanoBankName,
			Type:          model.TypeListRate,
			Term:          term,
			NominalRate:   rate,
			LastCrawledAt: crawlTime,
		})
	}

	return interestSets, nil
}

//nolint:cyclop // Complexity is inherent in multi-column table parsing with error handling
func (c *IkanoBankCrawler) fetchAverageRates(crawlTime time.Time) ([]model.InterestSet, error) {
	rawHTML, err := c.httpClient.Fetch(ikanoBankAvgRatesURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed reading Ikano Bank average rates page: %w", err)
	}

	// Search for "Snitträntor för bolån" which appears before the average rates table.
	// The HTML is encoded so we need to search for the encoded version.
	tokenizer, err := utils.FindTokenizedTableByTextBeforeTable(rawHTML, "Snitträntor för bolån")
	if err != nil {
		return nil, fmt.Errorf("failed to find table by text 'Snitträntor för bolån': %w", err)
	}

	table, err := utils.ParseTable(tokenizer)
	if err != nil {
		return nil, fmt.Errorf("failed to parse table: %w", err)
	}

	// Dynamically parse terms from header row.
	// Table structure: Månad | 3 mån | 1 år | 2 år | ... (terms may change)
	termMap := make(map[int]model.Term)
	for colIdx, header := range table.Header {
		if colIdx == 0 {
			// Skip first column (Månad).
			continue
		}
		term, err := utils.ParseTerm(header)
		if err != nil {
			c.logger.Warn("failed to parse term from header",
				zap.String("header", header),
				zap.Error(err))
			continue
		}
		termMap[colIdx] = term
	}

	if len(termMap) == 0 {
		return nil, fmt.Errorf("no valid terms found in table header")
	}

	interestSets := []model.InterestSet{}
	for _, row := range table.Rows {
		if len(row) < 2 {
			continue
		}

		// Parse month from first column (format: "2025 01").
		avgMonth, err := parseIkanoBankAvgMonth(row[0])
		if err != nil {
			c.logger.Warn("failed to parse average month", zap.String("month", row[0]), zap.Error(err))
			continue
		}

		// Parse rates for each term column.
		for colIdx, term := range termMap {
			if colIdx >= len(row) {
				continue
			}

			rateStr := row[colIdx]
			// Skip "-" entries (insufficient data).
			if strings.Contains(rateStr, "-") {
				continue
			}

			rate, err := parseIkanoBankRate(rateStr)
			if err != nil {
				c.logger.Warn("failed to parse average rate",
					zap.String("rate", rateStr),
					zap.String("term", string(term)),
					zap.Error(err))
				continue
			}

			interestSets = append(interestSets, model.InterestSet{
				Bank:                  ikanoBankName,
				Type:                  model.TypeAverageRate,
				Term:                  term,
				NominalRate:           rate,
				LastCrawledAt:         crawlTime,
				AverageReferenceMonth: avgMonth,
			})
		}
	}

	return interestSets, nil
}

// parseIkanoBankRate parses a rate string from Ikano Bank.
// Handles both API format ("3.4800") and HTML format ("3,61 %").
func parseIkanoBankRate(rateStr string) (float32, error) {
	str := utils.NormalizeSpaces(rateStr)
	str = strings.ReplaceAll(str, "%", "")
	str = strings.ReplaceAll(str, ",", ".")
	str = strings.TrimSpace(str)

	rate, err := strconv.ParseFloat(str, 32)
	if err != nil {
		return 0, fmt.Errorf("failed to parse rate %q: %w", rateStr, err)
	}

	return float32(rate), nil
}

// parseIkanoBankAvgMonth parses a month string from Ikano Bank average rates table.
// Format: "2025 01" or "2024 12".
func parseIkanoBankAvgMonth(monthStr string) (*model.AvgMonth, error) {
	str := utils.NormalizeSpaces(monthStr)
	matches := ikanoAvgMonthRegex.FindStringSubmatch(str)
	if matches == nil {
		return nil, fmt.Errorf("month %q does not match expected format 'YYYY MM'", monthStr)
	}

	year, _ := strconv.Atoi(matches[1])
	month, _ := strconv.Atoi(matches[2])

	return &model.AvgMonth{
		Year:  uint(year),
		Month: time.Month(month),
	}, nil
}
