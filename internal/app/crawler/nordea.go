package crawler

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/yama6a/bolan-compare/internal/pkg/http"
	"github.com/yama6a/bolan-compare/internal/pkg/model"
	"github.com/yama6a/bolan-compare/internal/pkg/utils"
	"go.uber.org/zap"
)

const (
	nordeaListRatesURL            = "https://www.nordea.se/privat/produkter/bolan/listrantor.html"
	nordeaAvgRatesURL             = "https://www.nordea.se/privat/produkter/bolan/snittrantor.html"
	nordeaBankName     model.Bank = "Nordea"
)

var (
	_ SiteCrawler = &NordeaCrawler{}

	// Nordea date format: YYYY-MM-DD.
	nordeaDateRegex = regexp.MustCompile(`^(\d{4})-(0[1-9]|1[0-2])-(0[1-9]|[12]\d|3[01])$`)
	// Nordea average month format in header: "202511" (YYYYMM).
	nordeaAvgMonthRegex = regexp.MustCompile(`^(\d{4})(0[1-9]|1[0-2])$`)
)

type NordeaCrawler struct {
	httpClient http.Client
	logger     *zap.Logger
}

func NewNordeaCrawler(httpClient http.Client, logger *zap.Logger) *NordeaCrawler {
	return &NordeaCrawler{httpClient: httpClient, logger: logger}
}

func (c *NordeaCrawler) Crawl(channel chan<- model.InterestSet) {
	interestSets := []model.InterestSet{}

	crawlTime := time.Now().UTC()

	// Fetch list rates
	listRatesHTML, err := c.httpClient.Fetch(nordeaListRatesURL, nil)
	if err != nil {
		c.logger.Error("failed reading Nordea list rates website", zap.Error(err))
	} else {
		listRates, err := c.extractListRates(listRatesHTML, crawlTime)
		if err != nil {
			c.logger.Error("failed parsing Nordea List Rates", zap.Error(err))
		} else {
			interestSets = append(interestSets, listRates...)
		}
	}

	// Fetch average rates (different page)
	avgRatesHTML, err := c.httpClient.Fetch(nordeaAvgRatesURL, nil)
	if err != nil {
		c.logger.Error("failed reading Nordea average rates website", zap.Error(err))
	} else {
		avgRates, err := c.extractAverageRates(avgRatesHTML, crawlTime)
		if err != nil {
			c.logger.Error("failed parsing Nordea Average Rates", zap.Error(err))
		} else {
			interestSets = append(interestSets, avgRates...)
		}
	}

	for _, set := range interestSets {
		channel <- set
	}
}

//nolint:dupl // Each bank crawler is intentionally independent for maintainability
func (c *NordeaCrawler) extractListRates(rawHTML string, crawlTime time.Time) ([]model.InterestSet, error) {
	// Find the table with caption "Listräntor för bolån"
	tokenizer, err := utils.FindTokenizedTableByTextBeforeTable(rawHTML, "Listräntor för bolån")
	if err != nil {
		return nil, fmt.Errorf("failed to find list rates table: %w", err)
	}

	table, err := utils.ParseTable(tokenizer)
	if err != nil {
		return nil, fmt.Errorf("failed to parse list rates table: %w", err)
	}

	// Table structure: Bindningstid | Ränta | Ändring | Senast ändrad
	interestSets := []model.InterestSet{}
	for _, row := range table.Rows {
		if len(row) < 4 {
			c.logger.Warn("skipping row with insufficient columns", zap.Strings("row", row))
			continue
		}

		term, err := utils.ParseTerm(row[0])
		if err != nil {
			c.logger.Warn("failed to parse term", zap.String("term", row[0]), zap.Error(err))
			continue
		}

		rate, err := parseNordeaRate(row[1])
		if err != nil {
			c.logger.Warn("failed to parse rate", zap.String("rate", row[1]), zap.Error(err))
			continue
		}

		changedOn, err := parseNordeaDate(row[3])
		if err != nil {
			c.logger.Warn("failed to parse change date", zap.String("date", row[3]), zap.Error(err))
			continue
		}

		interestSets = append(interestSets, model.InterestSet{
			Bank:          nordeaBankName,
			Type:          model.TypeListRate,
			Term:          term,
			NominalRate:   rate,
			ChangedOn:     &changedOn,
			LastCrawledAt: crawlTime,
		})
	}

	return interestSets, nil
}

//nolint:cyclop // Complexity is inherent in multi-column table parsing with error handling
func (c *NordeaCrawler) extractAverageRates(rawHTML string, crawlTime time.Time) ([]model.InterestSet, error) {
	// Find the table with title "Snitträntor" - text appears before the table
	tokenizer, err := utils.FindTokenizedTableByTextBeforeTable(rawHTML, "respektive månad") //nolint:misspell // Swedish text, not English
	if err != nil {
		return nil, fmt.Errorf("failed to find average rates table: %w", err)
	}

	table, err := utils.ParseTable(tokenizer)
	if err != nil {
		return nil, fmt.Errorf("failed to parse average rates table: %w", err)
	}

	// Parse headers to get months (columns 1+)
	// Header format: Månad | 202511 | 202510 | 202509 | ...
	monthColumns := make(map[int]*model.AvgMonth)
	for i, header := range table.Header {
		if i == 0 {
			continue // skip "Månad" column
		}
		avgMonth, err := parseNordeaAvgMonth(header)
		if err != nil {
			c.logger.Warn("failed to parse average month header", zap.String("header", header), zap.Error(err))
			continue
		}
		monthColumns[i] = avgMonth
	}

	// Rows: term | rate1 | rate2 | rate3 | ...
	interestSets := []model.InterestSet{}
	for _, row := range table.Rows {
		if len(row) < 2 {
			continue
		}

		// First column is the term (e.g., "3 mån", "1 år")
		termStr := row[0]
		// Skip "Banklån*" row - it's not a standard term
		if strings.Contains(strings.ToLower(termStr), "banklån") {
			continue
		}

		term, err := utils.ParseTerm(termStr)
		if err != nil {
			c.logger.Warn("failed to parse average rate term", zap.String("term", termStr), zap.Error(err))
			continue
		}

		// Parse rates for each month column
		for colIdx, avgMonth := range monthColumns {
			if colIdx >= len(row) {
				continue
			}

			rateStr := row[colIdx]
			rate, err := parseNordeaRate(rateStr)
			if err != nil {
				// Skip cells that can't be parsed (might be empty or "-")
				continue
			}

			interestSets = append(interestSets, model.InterestSet{
				Bank:                  nordeaBankName,
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

func parseNordeaRate(rateStr string) (float32, error) {
	str := utils.NormalizeSpaces(rateStr)
	str = strings.ReplaceAll(str, "%", "")
	str = strings.ReplaceAll(str, ",", ".")
	str = strings.TrimSpace(str)

	if str == "" || str == "-" {
		return 0, fmt.Errorf("empty or invalid rate: %q", rateStr)
	}

	rate, err := strconv.ParseFloat(str, 32)
	if err != nil {
		return 0, fmt.Errorf("failed to parse rate %q: %w", rateStr, err)
	}

	return float32(rate), nil
}

func parseNordeaDate(dateStr string) (time.Time, error) {
	str := utils.NormalizeSpaces(dateStr)
	matches := nordeaDateRegex.FindStringSubmatch(str)
	if matches == nil {
		return time.Time{}, fmt.Errorf("date %q does not match expected format YYYY-MM-DD", dateStr)
	}

	year, _ := strconv.Atoi(matches[1])
	month, _ := strconv.Atoi(matches[2])
	day, _ := strconv.Atoi(matches[3])

	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC), nil
}

func parseNordeaAvgMonth(monthStr string) (*model.AvgMonth, error) {
	str := utils.NormalizeSpaces(monthStr)
	matches := nordeaAvgMonthRegex.FindStringSubmatch(str)
	if matches == nil {
		return nil, fmt.Errorf("month %q does not match expected format 'YYYYMM'", monthStr)
	}

	year, _ := strconv.Atoi(matches[1])
	month, _ := strconv.Atoi(matches[2])

	return &model.AvgMonth{
		Year:  uint(year), //nolint:gosec // year is validated by regex to be 4 digits (0000-9999)
		Month: time.Month(month),
	}, nil
}
