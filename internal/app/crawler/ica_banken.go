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
	icaBankenURL             = "https://www.icabanken.se/lana/bolan/bolanerantor/"
	icaBankenName model.Bank = "ICA Banken"
)

var (
	_ SiteCrawler = &ICABankenCrawler{}

	// ICA Banken date format: YYYY-MM-DD.
	icaDateRegex = regexp.MustCompile(`^(\d{4})-(0[1-9]|1[0-2])-(0[1-9]|[12]\d|3[01])$`)
	// ICA Banken average month format: "2025 11" or "2024 12".
	icaAvgMonthRegex = regexp.MustCompile(`^(\d{4})\s+(\d{1,2})$`)
)

type ICABankenCrawler struct {
	httpClient http.Client
	logger     *zap.Logger
}

func NewICABankenCrawler(httpClient http.Client, logger *zap.Logger) *ICABankenCrawler {
	return &ICABankenCrawler{httpClient: httpClient, logger: logger}
}

func (c *ICABankenCrawler) Crawl(channel chan<- model.InterestSet) {
	interestSets := []model.InterestSet{}

	crawlTime := time.Now().UTC()
	rawHTML, err := c.httpClient.Fetch(icaBankenURL, nil)
	if err != nil {
		c.logger.Error("failed reading ICA Banken website", zap.Error(err))
		return
	}

	listRates, err := c.extractListRates(rawHTML, crawlTime)
	if err != nil {
		c.logger.Error("failed parsing ICA Banken List Rates", zap.Error(err))
	} else {
		interestSets = append(interestSets, listRates...)
	}

	averageRates, err := c.extractAverageRates(rawHTML, crawlTime)
	if err != nil {
		c.logger.Error("failed parsing ICA Banken Average Rates", zap.Error(err))
	} else {
		interestSets = append(interestSets, averageRates...)
	}

	for _, set := range interestSets {
		channel <- set
	}
}

//nolint:dupl // Each bank crawler is intentionally independent for maintainability
func (c *ICABankenCrawler) extractListRates(rawHTML string, crawlTime time.Time) ([]model.InterestSet, error) {
	// Search for "Aktuella bolåneräntor" which appears before the list rates table
	tokenizer, err := utils.FindTokenizedTableByTextBeforeTable(rawHTML, "Aktuella bolåneräntor")
	if err != nil {
		return nil, fmt.Errorf("failed to find table by text 'Aktuella bolåneräntor': %w", err)
	}

	table, err := utils.ParseTable(tokenizer)
	if err != nil {
		return nil, fmt.Errorf("failed to parse table: %w", err)
	}

	// Table structure: Bindningstid | Ränta | Senast ändrad
	interestSets := []model.InterestSet{}
	for _, row := range table.Rows {
		if len(row) < 3 {
			c.logger.Warn("skipping row with insufficient columns", zap.Strings("row", row))
			continue
		}

		term, err := utils.ParseTerm(row[0])
		if err != nil {
			c.logger.Warn("failed to parse term", zap.String("term", row[0]), zap.Error(err))
			continue
		}

		rate, err := parseICARate(row[1])
		if err != nil {
			c.logger.Warn("failed to parse rate", zap.String("rate", row[1]), zap.Error(err))
			continue
		}

		changedOn, err := parseICADate(row[2])
		if err != nil {
			c.logger.Warn("failed to parse change date", zap.String("date", row[2]), zap.Error(err))
			continue
		}

		interestSets = append(interestSets, model.InterestSet{
			Bank:          icaBankenName,
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
func (c *ICABankenCrawler) extractAverageRates(rawHTML string, crawlTime time.Time) ([]model.InterestSet, error) {
	// Search for "Snitträntor för bolån" which appears before the average rates table
	tokenizer, err := utils.FindTokenizedTableByTextBeforeTable(rawHTML, "Snitträntor för bolån")
	if err != nil {
		return nil, fmt.Errorf("failed to find table by text 'Snitträntor för bolån': %w", err)
	}

	table, err := utils.ParseTable(tokenizer)
	if err != nil {
		return nil, fmt.Errorf("failed to parse table: %w", err)
	}

	// Table structure: Månad | 3 mån | 1 år | 2 år | 3 år | 4 år | 5 år | 7 år | 10 år
	// Header maps column index to term
	termMap := map[int]model.Term{
		1: model.Term3months,
		2: model.Term1year,
		3: model.Term2years,
		4: model.Term3years,
		5: model.Term4years,
		6: model.Term5years,
		7: model.Term7years,
		8: model.Term10years,
	}

	interestSets := []model.InterestSet{}
	for _, row := range table.Rows {
		if len(row) < 2 {
			continue
		}

		// Parse month from first column (format: "2025 11")
		avgMonth, err := parseICAAvgMonth(row[0])
		if err != nil {
			c.logger.Warn("failed to parse average month", zap.String("month", row[0]), zap.Error(err))
			continue
		}

		// Parse rates for each term column
		for colIdx, term := range termMap {
			if colIdx >= len(row) {
				continue
			}

			rateStr := row[colIdx]
			// Skip "-*" entries (insufficient data)
			if strings.Contains(rateStr, "-") || strings.Contains(rateStr, "*") {
				continue
			}

			rate, err := parseICARate(rateStr)
			if err != nil {
				c.logger.Warn("failed to parse average rate",
					zap.String("rate", rateStr),
					zap.String("term", string(term)),
					zap.Error(err))
				continue
			}

			interestSets = append(interestSets, model.InterestSet{
				Bank:                  icaBankenName,
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

func parseICARate(rateStr string) (float32, error) {
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

func parseICADate(dateStr string) (time.Time, error) {
	str := utils.NormalizeSpaces(dateStr)
	matches := icaDateRegex.FindStringSubmatch(str)
	if matches == nil {
		return time.Time{}, fmt.Errorf("date %q does not match expected format YYYY-MM-DD", dateStr)
	}

	year, _ := strconv.Atoi(matches[1])
	month, _ := strconv.Atoi(matches[2])
	day, _ := strconv.Atoi(matches[3])

	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC), nil
}

func parseICAAvgMonth(monthStr string) (*model.AvgMonth, error) {
	str := utils.NormalizeSpaces(monthStr)
	matches := icaAvgMonthRegex.FindStringSubmatch(str)
	if matches == nil {
		return nil, fmt.Errorf("month %q does not match expected format 'YYYY MM'", monthStr)
	}

	year, _ := strconv.Atoi(matches[1])
	month, _ := strconv.Atoi(matches[2])

	return &model.AvgMonth{
		Year:  uint(year), //nolint:gosec // year is validated by regex to be 4 digits (0000-9999)
		Month: time.Month(month),
	}, nil
}
