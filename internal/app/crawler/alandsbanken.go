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
	alandsbankRatesURL            = "https://www.alandsbanken.se/banktjanster/lana-pengar/bolan"
	alandsbankBankName model.Bank = "Ålandsbanken"
)

var (
	_ SiteCrawler = &AlandsbankCrawler{}

	// Ålandsbanken date format in list rates: "2025.10.03".
	alandsbankListDateRegex = regexp.MustCompile(`^\d{4}\.\d{2}\.\d{2}$`)
)

// alandsbankSwedishMonthToTime converts Swedish month names to time.Month.
func alandsbankSwedishMonthToTime(monthName string) (time.Month, bool) {
	months := map[string]time.Month{
		"januari":   time.January,
		"februari":  time.February,
		"mars":      time.March,
		"april":     time.April,
		"maj":       time.May,
		"juni":      time.June,
		"juli":      time.July,
		"augusti":   time.August,
		"september": time.September,
		"oktober":   time.October,
		"november":  time.November,
		"december":  time.December,
	}
	m, ok := months[strings.ToLower(monthName)]
	return m, ok
}

type AlandsbankCrawler struct {
	httpClient http.Client
	logger     *zap.Logger
}

func NewAlandsbankCrawler(httpClient http.Client, logger *zap.Logger) *AlandsbankCrawler {
	return &AlandsbankCrawler{httpClient: httpClient, logger: logger}
}

func (c *AlandsbankCrawler) Crawl(channel chan<- model.InterestSet) {
	interestSets := []model.InterestSet{}
	crawlTime := time.Now().UTC()

	// Fetch rates page (contains both list and average rates)
	html, err := c.httpClient.Fetch(alandsbankRatesURL, nil)
	if err != nil {
		c.logger.Error("failed reading Ålandsbanken rates page", zap.Error(err))
		return
	}

	// Extract list rates
	listRates, err := c.extractListRates(html, crawlTime)
	if err != nil {
		c.logger.Error("failed parsing Ålandsbanken List Rates", zap.Error(err))
	} else {
		interestSets = append(interestSets, listRates...)
	}

	// Extract average rates
	avgRates, err := c.extractAverageRates(html, crawlTime)
	if err != nil {
		c.logger.Error("failed parsing Ålandsbanken Average Rates", zap.Error(err))
	} else {
		interestSets = append(interestSets, avgRates...)
	}

	for _, set := range interestSets {
		channel <- set
	}
}

func (c *AlandsbankCrawler) extractListRates(rawHTML string, crawlTime time.Time) ([]model.InterestSet, error) {
	// Find the table that follows the text "Aktuella räntor:"
	tokenizer, err := utils.FindTokenizedTableByTextBeforeTable(rawHTML, "Aktuella räntor:")
	if err != nil {
		return nil, fmt.Errorf("failed to find list rates table: %w", err)
	}

	table, err := utils.ParseTable(tokenizer)
	if err != nil {
		return nil, fmt.Errorf("failed to parse list rates table: %w", err)
	}

	// Table structure: Bindningstid | Räntesats % | Senaste ränteförändring | Förändring %
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

		rate, err := parseAlandsbankRate(row[1])
		if err != nil {
			c.logger.Warn("failed to parse rate", zap.String("rate", row[1]), zap.Error(err))
			continue
		}

		// Parse change date from third column (format: "2025.10.03")
		var changedOn *time.Time
		if len(row) >= 3 {
			parsed, err := parseAlandsbankListDate(row[2])
			if err != nil {
				c.logger.Warn("failed to parse change date", zap.String("date", row[2]), zap.Error(err))
			} else {
				changedOn = &parsed
			}
		}

		interestSets = append(interestSets, model.InterestSet{
			Bank:          alandsbankBankName,
			Type:          model.TypeListRate,
			Term:          term,
			NominalRate:   rate,
			ChangedOn:     changedOn,
			LastCrawledAt: crawlTime,
		})
	}

	return interestSets, nil
}

func (c *AlandsbankCrawler) extractAverageRates(rawHTML string, crawlTime time.Time) ([]model.InterestSet, error) {
	// Find the table with header containing "Genomsnittlig bolåneränta"
	tokenizer, err := utils.FindTokenizedTableByTextBeforeTable(rawHTML, "Genomsnittlig bolåneränta")
	if err != nil {
		return nil, fmt.Errorf("failed to find average rates table: %w", err)
	}

	table, err := utils.ParseTable(tokenizer)
	if err != nil {
		return nil, fmt.Errorf("failed to parse average rates table: %w", err)
	}

	// Table structure: Bindningstid | Genomsnittlig bolåneränta | Månad
	// Note: Ålandsbanken only publishes average rates for 3 mån
	interestSets := []model.InterestSet{}
	for _, row := range table.Rows {
		if len(row) < 3 {
			continue
		}

		term, err := utils.ParseTerm(row[0])
		if err != nil {
			c.logger.Warn("failed to parse term", zap.String("term", row[0]), zap.Error(err))
			continue
		}

		rate, err := parseAlandsbankRate(row[1])
		if err != nil {
			continue // Skip empty or invalid rates
		}

		// Parse month from third column (format: "Oktober 2025")
		avgMonth, err := parseAlandsbankAvgMonth(row[2])
		if err != nil {
			c.logger.Warn("failed to parse average month", zap.String("month", row[2]), zap.Error(err))
			continue
		}

		interestSets = append(interestSets, model.InterestSet{
			Bank:                  alandsbankBankName,
			Type:                  model.TypeAverageRate,
			Term:                  term,
			NominalRate:           rate,
			LastCrawledAt:         crawlTime,
			AverageReferenceMonth: avgMonth,
		})
	}

	return interestSets, nil
}

func parseAlandsbankRate(rateStr string) (float32, error) {
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

func parseAlandsbankListDate(dateStr string) (time.Time, error) {
	str := utils.NormalizeSpaces(dateStr)
	str = strings.TrimSpace(str)

	// Format: "2025.10.03"
	if !alandsbankListDateRegex.MatchString(str) {
		return time.Time{}, fmt.Errorf("date %q does not match expected format 'YYYY.MM.DD'", dateStr)
	}

	parts := strings.Split(str, ".")
	if len(parts) != 3 {
		return time.Time{}, fmt.Errorf("date %q does not have expected format 'YYYY.MM.DD'", dateStr)
	}

	year, _ := strconv.Atoi(parts[0])
	month, _ := strconv.Atoi(parts[1])
	day, _ := strconv.Atoi(parts[2])

	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC), nil
}

func parseAlandsbankAvgMonth(monthStr string) (*model.AvgMonth, error) {
	str := utils.NormalizeSpaces(monthStr)
	str = strings.TrimSpace(str)

	// Format: "Oktober 2025"
	parts := strings.Split(str, " ")
	if len(parts) != 2 {
		return nil, fmt.Errorf("month %q does not match expected format 'Month YYYY'", monthStr)
	}

	monthName := parts[0]
	year, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, fmt.Errorf("failed to parse year from %q: %w", monthStr, err)
	}

	month, ok := alandsbankSwedishMonthToTime(monthName)
	if !ok {
		return nil, fmt.Errorf("unknown Swedish month: %q", monthName)
	}

	return &model.AvgMonth{
		Year:  uint(year),
		Month: month,
	}, nil
}
