//nolint:revive // Nordnet prefix is intentional for clarity
package nordnet

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
	nordnetBankName model.Bank = "Nordnet"

	// Nordnet uses Contentful CMS API to serve their rate data.
	nordnetRatesURL = "https://api.prod.nntech.io/cms/v1/contentful-cache/spaces/main_se/environments/master/entries?include=5&sys.id=36p8FGv6CCUfUIiXPjPBJy"
)

// NordnetCrawler crawls mortgage rates from Nordnet.
// Nordnet offers mortgages via Stabelo with different LTV-based rate tiers.
type NordnetCrawler struct {
	httpClient http.Client
	logger     *zap.Logger
}

var _ crawler.SiteCrawler = &NordnetCrawler{}

// NewNordnetCrawler creates a new Nordnet crawler.
func NewNordnetCrawler(httpClient http.Client, logger *zap.Logger) *NordnetCrawler {
	return &NordnetCrawler{httpClient: httpClient, logger: logger}
}

// Crawl fetches and parses mortgage rates from Nordnet.
func (c *NordnetCrawler) Crawl(channel chan<- model.InterestSet) {
	crawlTime := time.Now().UTC()

	listRates, err := c.fetchListRates(crawlTime)
	if err != nil {
		c.logger.Error("failed fetching Nordnet list rates", zap.Error(err))
		return
	}

	for _, set := range listRates {
		channel <- set
	}
}

// nordnetCMSResponse represents the Contentful CMS response structure.
type nordnetCMSResponse struct {
	Includes struct {
		Entry []nordnetEntry `json:"Entry"` //nolint:tagliatelle // Contentful API uses uppercase "Entry"
	} `json:"includes"`
}

type nordnetEntry struct {
	Sys struct {
		ID          string `json:"id"`
		ContentType struct {
			Sys struct {
				ID string `json:"id"`
			} `json:"sys"`
		} `json:"contentType"`
	} `json:"sys"`
	Fields struct {
		InternalName string `json:"internalName"`
		TableData    struct {
			Rows [][]nordnetTableCell `json:"rows"`
		} `json:"tableData"`
	} `json:"fields"`
}

type nordnetTableCell struct {
	Value string `json:"value"`
}

// fetchListRates fetches list rates from Nordnet's CMS API.
func (c *NordnetCrawler) fetchListRates(crawlTime time.Time) ([]model.InterestSet, error) {
	rawJSON, err := c.httpClient.Fetch(nordnetRatesURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed reading Nordnet rates API: %w", err)
	}

	var response nordnetCMSResponse
	if err := json.Unmarshal([]byte(rawJSON), &response); err != nil {
		c.logger.Error("failed unmarshalling Nordnet rates", zap.Error(err), zap.String("rawJSON", rawJSON))
		return nil, fmt.Errorf("failed unmarshalling Nordnet rates: %w", err)
	}

	// Find the rate table entry
	var tableData *[][]nordnetTableCell
	for _, entry := range response.Includes.Entry {
		if entry.Sys.ContentType.Sys.ID == "componentTable" &&
			strings.Contains(entry.Fields.InternalName, "Räntor") &&
			strings.Contains(entry.Fields.InternalName, "Tabell") {
			tableData = &entry.Fields.TableData.Rows
			break
		}
	}

	if tableData == nil {
		return nil, fmt.Errorf("rate table not found in Nordnet CMS response")
	}

	return c.parseRateTable(*tableData, crawlTime)
}

// parseRateTable parses the rate table from Nordnet's CMS response.
// The table has headers in row 0 and rate data in subsequent rows.
// For list rates, we use the highest LTV tier (last column: 80-85%).
func (c *NordnetCrawler) parseRateTable(rows [][]nordnetTableCell, crawlTime time.Time) ([]model.InterestSet, error) {
	if len(rows) < 2 {
		return nil, fmt.Errorf("rate table has insufficient rows: %d", len(rows))
	}

	// First row is headers, skip it
	// Rate format: "2,54 (2,57)" where first number is nominal rate
	rateRegex := regexp.MustCompile(`^(\d+),(\d+)`)

	var interestSets []model.InterestSet

	for i := 1; i < len(rows); i++ {
		row := rows[i]
		if len(row) < 4 {
			c.logger.Warn("Nordnet rate row has insufficient columns, skipping",
				zap.Int("row", i),
				zap.Int("columns", len(row)))
			continue
		}

		// First column is the term (e.g., "3 mån", "1 år")
		termStr := row[0].Value
		term, err := utils.ParseTerm(termStr)
		if err != nil {
			c.logger.Warn("failed parsing Nordnet term, skipping",
				zap.String("term", termStr),
				zap.Error(err))
			continue
		}

		// Use the last column (highest LTV tier: 80-85%) for list rate
		rateStr := row[len(row)-1].Value
		rate, err := parseNordnetRate(rateStr, rateRegex)
		if err != nil {
			c.logger.Warn("failed parsing Nordnet rate, skipping",
				zap.String("rate", rateStr),
				zap.Error(err))
			continue
		}

		interestSets = append(interestSets, model.InterestSet{
			Bank:          nordnetBankName,
			Type:          model.TypeListRate,
			Term:          term,
			NominalRate:   rate,
			LastCrawledAt: crawlTime,

			ChangedOn:               nil, // Nordnet API doesn't provide change dates
			RatioDiscountBoundaries: nil,
			UnionDiscount:           false,
			AverageReferenceMonth:   nil,
		})
	}

	return interestSets, nil
}

// parseNordnetRate parses a rate string like "2,54 (2,57)" and returns the nominal rate.
func parseNordnetRate(rateStr string, rateRegex *regexp.Regexp) (float32, error) {
	matches := rateRegex.FindStringSubmatch(rateStr)
	if len(matches) != 3 {
		return 0, fmt.Errorf("invalid rate format: %s", rateStr)
	}

	intPart, err := strconv.ParseFloat(matches[1], 32)
	if err != nil {
		return 0, fmt.Errorf("failed parsing rate integer part: %w", err)
	}

	decPart, err := strconv.ParseFloat(matches[2], 32)
	if err != nil {
		return 0, fmt.Errorf("failed parsing rate decimal part: %w", err)
	}

	// Calculate the rate value (e.g., "2,54" -> 2.54)
	decDivisor := 10.0
	for decPart >= decDivisor {
		decDivisor *= 10.0
	}

	rate := float32(intPart + decPart/decDivisor)
	return rate, nil
}
