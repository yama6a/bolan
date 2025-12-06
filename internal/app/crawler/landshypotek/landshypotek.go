package landshypotek

import (
	"fmt"
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
	landshypotekRatesURL            = "https://www.landshypotek.se/lana/bolanerantor/"
	landshypotekBankName model.Bank = "Landshypotek"
)

var _ crawler.SiteCrawler = &LandshypotekCrawler{}

//nolint:revive // Bank name prefix is intentional for clarity
type LandshypotekCrawler struct {
	httpClient http.Client
	logger     *zap.Logger
}

func NewLandshypotekCrawler(httpClient http.Client, logger *zap.Logger) *LandshypotekCrawler {
	return &LandshypotekCrawler{httpClient: httpClient, logger: logger}
}

func (c *LandshypotekCrawler) Crawl(channel chan<- model.InterestSet) {
	interestSets := []model.InterestSet{}
	crawlTime := time.Now().UTC()

	// Fetch rates page
	html, err := c.httpClient.Fetch(landshypotekRatesURL, nil)
	if err != nil {
		c.logger.Error("failed reading Landshypotek rates page", zap.Error(err))
		return
	}

	// Extract discounted rates (visible on page load) - 60% LTV tier
	var discounted60Rates, discounted75Rates, listRates, avgRates, historicalRates []model.InterestSet

	discounted60Rates, err = c.extractDiscountedRates(html, crawlTime, "belåningsgrad 60", 0, 60)
	if err != nil {
		c.logger.Error("failed parsing Landshypotek discounted rates (60% LTV)", zap.Error(err))
	} else {
		interestSets = append(interestSets, discounted60Rates...)
	}

	// Extract discounted rates (visible on page load) - 75% LTV tier
	discounted75Rates, err = c.extractDiscountedRates(html, crawlTime, "belåningsgrad 75", 60, 75)
	if err != nil {
		c.logger.Error("failed parsing Landshypotek discounted rates (75% LTV)", zap.Error(err))
	} else {
		interestSets = append(interestSets, discounted75Rates...)
	}

	// Extract list rates (in accordion, before discount)
	listRates, err = c.extractListRates(html, crawlTime)
	if err != nil {
		c.logger.Error("failed parsing Landshypotek list rates", zap.Error(err))
	} else {
		interestSets = append(interestSets, listRates...)
	}

	// Extract average rates for current month (in accordion)
	avgRates, err = c.extractCurrentMonthAverageRates(html, crawlTime)
	if err != nil {
		c.logger.Error("failed parsing Landshypotek current month average rates", zap.Error(err))
	} else {
		interestSets = append(interestSets, avgRates...)
	}

	// Extract historical average rates (in accordion)
	historicalRates, err = c.extractHistoricalAverageRates(html, crawlTime)
	if err != nil {
		c.logger.Error("failed parsing Landshypotek historical average rates", zap.Error(err))
	} else {
		interestSets = append(interestSets, historicalRates...)
	}

	// Log individual rates
	for _, set := range interestSets {
		channel <- set
	}
}

func (c *LandshypotekCrawler) extractDiscountedRates(
	rawHTML string,
	crawlTime time.Time,
	searchText string,
	minRatio, maxRatio float32,
) ([]model.InterestSet, error) {
	// Find the table with caption containing the search text (e.g., "belåningsgrad 60")
	tokenizer, err := utils.FindTokenizedTableByTextInCaption(rawHTML, searchText)
	if err != nil {
		return nil, fmt.Errorf("failed to find discounted rates table: %w", err)
	}

	table, err := utils.ParseTable(tokenizer)
	if err != nil {
		return nil, fmt.Errorf("failed to parse discounted rates table: %w", err)
	}

	// Table structure: Bindningstid | Ränta | Effektiv ränta
	interestSets := []model.InterestSet{}
	for _, row := range table.Rows {
		if len(row) < 2 {
			c.logger.Warn("skipping row with insufficient columns", zap.Strings("row", row))
			continue
		}

		term, err := utils.ParseTerm(row[0])
		if err != nil {
			c.logger.Warn("failed to parse term", zap.String("term", row[0]), zap.Error(err))
			continue
		}

		rate, err := parseLandshypotekRate(row[1])
		if err != nil {
			c.logger.Warn("failed to parse rate", zap.String("rate", row[1]), zap.Error(err))
			continue
		}

		interestSets = append(interestSets, model.InterestSet{
			Bank:        landshypotekBankName,
			Type:        model.TypeRatioDiscounted,
			Term:        term,
			NominalRate: rate,
			RatioDiscountBoundaries: &model.RatioDiscountBoundary{
				MinRatio: minRatio,
				MaxRatio: maxRatio,
			},
			LastCrawledAt: crawlTime,
		})
	}

	return interestSets, nil
}

func (c *LandshypotekCrawler) extractListRates(rawHTML string, crawlTime time.Time) ([]model.InterestSet, error) {
	// List rates are in an accordion item with title containing "Listräntor för bolån"
	table, err := c.findListRatesTable(rawHTML)
	if err != nil {
		return nil, err
	}

	return c.parseListRatesTable(&table, crawlTime), nil
}

func (c *LandshypotekCrawler) findListRatesTable(rawHTML string) (utils.Table, error) {
	startIdx := strings.Index(rawHTML, "Listr&#xE4;ntor f&#xF6;r bol&#xE5;n")
	if startIdx == -1 {
		return utils.Table{}, fmt.Errorf("could not find list rates section")
	}

	relevantHTML := rawHTML[startIdx:]

	tokenizer, err := utils.FindTokenizedTableByTextBeforeTable(relevantHTML, "Bindningstid")
	if err != nil {
		return utils.Table{}, fmt.Errorf("failed to find list rates table: %w", err)
	}

	table, err := utils.ParseTable(tokenizer)
	if err != nil {
		return utils.Table{}, fmt.Errorf("failed to parse list rates table: %w", err)
	}

	return table, nil
}

func (c *LandshypotekCrawler) parseListRatesTable(table *utils.Table, crawlTime time.Time) []model.InterestSet {
	interestSets := []model.InterestSet{}

	for _, row := range table.Rows {
		if len(row) < 2 {
			c.logger.Warn("skipping row with insufficient columns", zap.Strings("row", row))
			continue
		}

		term, err := utils.ParseTerm(row[0])
		if err != nil {
			c.logger.Warn("failed to parse term", zap.String("term", row[0]), zap.Error(err))
			continue
		}

		rate, err := parseLandshypotekRate(row[1])
		if err != nil {
			c.logger.Warn("failed to parse rate", zap.String("rate", row[1]), zap.Error(err))
			continue
		}

		interestSet := model.InterestSet{
			Bank:          landshypotekBankName,
			Type:          model.TypeListRate,
			Term:          term,
			NominalRate:   rate,
			LastCrawledAt: crawlTime,
		}

		if len(row) >= 4 && row[3] != "" {
			if changeDate, err := parseLandshypotekChangeDate(row[3]); err == nil {
				interestSet.ChangedOn = &changeDate
			}
		}

		interestSets = append(interestSets, interestSet)
	}

	return interestSets
}

func (c *LandshypotekCrawler) extractCurrentMonthAverageRates(
	rawHTML string,
	crawlTime time.Time,
) ([]model.InterestSet, error) {
	avgMonth, table, err := c.findCurrentMonthAverageRatesTable(rawHTML, crawlTime)
	if err != nil {
		return nil, err
	}

	return c.parseAverageRatesTable(&table, avgMonth, crawlTime), nil
}

func (c *LandshypotekCrawler) findCurrentMonthAverageRatesTable(
	rawHTML string,
	crawlTime time.Time,
) (model.AvgMonth, utils.Table, error) {
	startIdx := strings.Index(rawHTML, "Snittr&#xE4;ntor f&#xF6;r bol&#xE5;n senaste m&#xE5;naden")
	if startIdx == -1 {
		return model.AvgMonth{}, utils.Table{}, fmt.Errorf("could not find current month average rates section")
	}

	relevantHTML := rawHTML[startIdx:]

	monthStr, err := extractMonthHeader(relevantHTML)
	if err != nil {
		return model.AvgMonth{}, utils.Table{}, err
	}

	avgMonth, err := parseLandshypotekAvgMonth(monthStr, crawlTime)
	if err != nil {
		return model.AvgMonth{}, utils.Table{}, fmt.Errorf("failed to parse month: %w", err)
	}

	tokenizer, err := utils.FindTokenizedTableByTextBeforeTable(relevantHTML, "Bindningstid")
	if err != nil {
		return model.AvgMonth{}, utils.Table{}, fmt.Errorf("failed to find average rates table: %w", err)
	}

	table, err := utils.ParseTable(tokenizer)
	if err != nil {
		return model.AvgMonth{}, utils.Table{}, fmt.Errorf("failed to parse average rates table: %w", err)
	}

	return avgMonth, table, nil
}

func (c *LandshypotekCrawler) parseAverageRatesTable(
	table *utils.Table,
	avgMonth model.AvgMonth,
	crawlTime time.Time,
) []model.InterestSet {
	interestSets := []model.InterestSet{}

	for _, row := range table.Rows {
		if len(row) < 2 {
			c.logger.Warn("skipping row with insufficient columns", zap.Strings("row", row))
			continue
		}

		term, err := utils.ParseTerm(row[0])
		if err != nil {
			c.logger.Warn("failed to parse term", zap.String("term", row[0]), zap.Error(err))
			continue
		}

		rate, err := parseLandshypotekRate(row[1])
		if err != nil {
			continue
		}

		interestSets = append(interestSets, model.InterestSet{
			Bank:                  landshypotekBankName,
			Type:                  model.TypeAverageRate,
			Term:                  term,
			NominalRate:           rate,
			AverageReferenceMonth: &avgMonth,
			LastCrawledAt:         crawlTime,
		})
	}

	return interestSets
}

func extractMonthHeader(html string) (string, error) {
	monthStart := strings.Index(html, "<h4>")
	if monthStart == -1 {
		return "", fmt.Errorf("could not find month header")
	}
	monthEnd := strings.Index(html[monthStart:], "</h4>")
	if monthEnd == -1 {
		return "", fmt.Errorf("could not find month header end")
	}
	return html[monthStart+4 : monthStart+monthEnd], nil
}

func (c *LandshypotekCrawler) extractHistoricalAverageRates(
	rawHTML string,
	crawlTime time.Time,
) ([]model.InterestSet, error) {
	table, err := c.findHistoricalAverageRatesTable(rawHTML)
	if err != nil {
		return nil, err
	}

	terms := c.parseTermsFromHeader(table.Header)
	if len(terms) == 0 {
		return nil, fmt.Errorf("no valid terms found in header")
	}

	return c.parseHistoricalRatesTable(&table, terms, crawlTime), nil
}

func (c *LandshypotekCrawler) findHistoricalAverageRatesTable(rawHTML string) (utils.Table, error) {
	startIdx := strings.Index(rawHTML, "Historisk snittr&#xE4;nta f&#xF6;r bol&#xE5;n")
	if startIdx == -1 {
		return utils.Table{}, fmt.Errorf("could not find historical average rates section")
	}

	relevantHTML := rawHTML[startIdx:]

	tokenizer, err := utils.FindTokenizedTableByTextBeforeTable(relevantHTML, "bindningstid")
	if err != nil {
		return utils.Table{}, fmt.Errorf("failed to find historical rates table: %w", err)
	}

	table, err := utils.ParseTable(tokenizer)
	if err != nil {
		return utils.Table{}, fmt.Errorf("failed to parse historical rates table: %w", err)
	}

	if len(table.Header) < 3 {
		return utils.Table{}, fmt.Errorf("invalid table header: expected at least 3 columns, got %d", len(table.Header))
	}

	return table, nil
}

func (c *LandshypotekCrawler) parseTermsFromHeader(header []string) []model.Term {
	terms := []model.Term{}
	for i := 2; i < len(header); i++ {
		term, err := utils.ParseTerm(header[i])
		if err != nil {
			c.logger.Warn("failed to parse term from header", zap.String("header", header[i]), zap.Error(err))
			continue
		}
		terms = append(terms, term)
	}
	return terms
}

func (c *LandshypotekCrawler) parseHistoricalRatesTable(
	table *utils.Table,
	terms []model.Term,
	crawlTime time.Time,
) []model.InterestSet {
	interestSets := []model.InterestSet{}

	for _, row := range table.Rows {
		if len(row) < 3 {
			c.logger.Warn("skipping row with insufficient columns", zap.Strings("row", row))
			continue
		}

		avgMonth, err := parseLandshypotekHistoricalMonth(row[0], row[1])
		if err != nil {
			c.logger.Warn("failed to parse month", zap.String("year", row[0]), zap.String("month", row[1]), zap.Error(err))
			continue
		}

		for i, term := range terms {
			colIdx := i + 2
			if colIdx >= len(row) {
				break
			}

			rate, err := parseLandshypotekRate(row[colIdx])
			if err != nil {
				continue
			}

			interestSets = append(interestSets, model.InterestSet{
				Bank:                  landshypotekBankName,
				Type:                  model.TypeAverageRate,
				Term:                  term,
				NominalRate:           rate,
				AverageReferenceMonth: &avgMonth,
				LastCrawledAt:         crawlTime,
			})
		}
	}

	return interestSets
}

func parseLandshypotekRate(rateStr string) (float32, error) {
	str := utils.NormalizeSpaces(rateStr)
	str = strings.ReplaceAll(str, "%", "")
	str = strings.ReplaceAll(str, ",", ".")
	str = strings.ReplaceAll(str, "\u00a0", "") // Remove non-breaking spaces
	str = strings.TrimSpace(str)

	// Handle n/a or empty values
	if str == "" || str == "-" || strings.ToLower(str) == "n/a" {
		return 0, fmt.Errorf("empty or invalid rate: %q", rateStr)
	}

	rate, err := strconv.ParseFloat(str, 32)
	if err != nil {
		return 0, fmt.Errorf("failed to parse rate %q: %w", rateStr, err)
	}

	return float32(rate), nil
}

func parseLandshypotekChangeDate(dateStr string) (time.Time, error) {
	// Format: "1 oktober 2025" or "20 november 2025"
	str := utils.NormalizeSpaces(dateStr)
	str = strings.TrimSpace(str)

	// Parse using Swedish month names
	monthMap := map[string]time.Month{
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

	parts := strings.Fields(str)
	if len(parts) != 3 {
		return time.Time{}, fmt.Errorf("invalid date format: %q", dateStr)
	}

	day, err := strconv.Atoi(parts[0])
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid day: %q", parts[0])
	}

	monthName := strings.ToLower(parts[1])
	month, ok := monthMap[monthName]
	if !ok {
		return time.Time{}, fmt.Errorf("invalid month: %q", parts[1])
	}

	year, err := strconv.Atoi(parts[2])
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid year: %q", parts[2])
	}

	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC), nil
}

func parseLandshypotekAvgMonth(monthStr string, crawlTime time.Time) (model.AvgMonth, error) {
	monthMap := map[string]time.Month{
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

	monthName := strings.ToLower(strings.TrimSpace(monthStr))
	month, ok := monthMap[monthName]
	if !ok {
		return model.AvgMonth{}, fmt.Errorf("invalid month: %q", monthStr)
	}

	// Infer year based on current time
	year := uint(crawlTime.Year())
	// If the month is in the future compared to crawl time, it's from last year
	if month > crawlTime.Month() {
		year--
	}

	return model.AvgMonth{
		Month: month,
		Year:  year,
	}, nil
}

func parseLandshypotekHistoricalMonth(yearStr, monthStr string) (model.AvgMonth, error) {
	monthMap := map[string]time.Month{
		"Januari":   time.January,
		"Februari":  time.February,
		"Mars":      time.March,
		"April":     time.April,
		"Maj":       time.May,
		"Juni":      time.June,
		"Juli":      time.July,
		"Augusti":   time.August,
		"September": time.September,
		"Oktober":   time.October,
		"November":  time.November,
		"December":  time.December,
	}

	month, ok := monthMap[strings.TrimSpace(monthStr)]
	if !ok {
		return model.AvgMonth{}, fmt.Errorf("invalid month: %q", monthStr)
	}

	year, err := strconv.ParseUint(strings.TrimSpace(yearStr), 10, 32)
	if err != nil {
		return model.AvgMonth{}, fmt.Errorf("invalid year: %q", yearStr)
	}

	return model.AvgMonth{
		Month: month,
		Year:  uint(year),
	}, nil
}
