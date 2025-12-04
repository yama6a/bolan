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
	swedbankListRatesURL                = "https://www.swedbank.se/privat/boende-och-bolan/bolanerantor.html"
	swedbankHistoricRatesURL            = "https://www.swedbank.se/privat/boende-och-bolan/bolanerantor/historiska-genomsnittsrantor.html"
	swedbankBankName         model.Bank = "Swedbank"
)

var (
	_ SiteCrawler = &SwedbankCrawler{}

	// Swedbank date format in list rates header: "senast ändrad 25 september 2025".
	swedbankListDateRegex = regexp.MustCompile(`senast ändrad (\d{1,2}) (\w+) (\d{4})`)
	// Swedbank month format in average rates header: "november 2025".
	swedbankAvgMonthRegex = regexp.MustCompile(`(\w+) (\d{4})$`)
)

// swedbankSwedishMonthToTime converts Swedish month names to time.Month.
// Supports both full names (januari) and abbreviated names (jan.).
func swedbankSwedishMonthToTime(monthName string) (time.Month, bool) {
	months := map[string]time.Month{
		"januari":   time.January,
		"jan":       time.January,
		"februari":  time.February,
		"feb":       time.February,
		"mars":      time.March,
		"mar":       time.March,
		"april":     time.April,
		"apr":       time.April,
		"maj":       time.May,
		"juni":      time.June,
		"jun":       time.June,
		"juli":      time.July,
		"jul":       time.July,
		"augusti":   time.August,
		"aug":       time.August,
		"september": time.September,
		"sep":       time.September,
		"oktober":   time.October,
		"okt":       time.October,
		"november":  time.November,
		"nov":       time.November,
		"december":  time.December,
		"dec":       time.December,
	}
	m, ok := months[monthName]
	return m, ok
}

type SwedbankCrawler struct {
	httpClient http.Client
	logger     *zap.Logger
}

func NewSwedbankCrawler(httpClient http.Client, logger *zap.Logger) *SwedbankCrawler {
	return &SwedbankCrawler{httpClient: httpClient, logger: logger}
}

//nolint:dupl // Crawl pattern intentionally similar across crawlers
func (c *SwedbankCrawler) Crawl(channel chan<- model.InterestSet) {
	interestSets := []model.InterestSet{}
	crawlTime := time.Now().UTC()

	// Fetch list rates from main page
	listHTML, err := c.httpClient.Fetch(swedbankListRatesURL, nil)
	if err != nil {
		c.logger.Error("failed reading Swedbank list rates page", zap.Error(err))
	} else {
		listRates, err := c.extractListRates(listHTML, crawlTime)
		if err != nil {
			c.logger.Error("failed parsing Swedbank List Rates", zap.Error(err))
		} else {
			interestSets = append(interestSets, listRates...)
		}
	}

	// Fetch average rates from historic page
	historicHTML, err := c.httpClient.Fetch(swedbankHistoricRatesURL, nil)
	if err != nil {
		c.logger.Error("failed reading Swedbank historic rates page", zap.Error(err))
	} else {
		avgRates, err := c.extractHistoricAverageRates(historicHTML, crawlTime)
		if err != nil {
			c.logger.Error("failed parsing Swedbank Historic Average Rates", zap.Error(err))
		} else {
			interestSets = append(interestSets, avgRates...)
		}
	}

	for _, set := range interestSets {
		channel <- set
	}
}

func (c *SwedbankCrawler) extractListRates(rawHTML string, crawlTime time.Time) ([]model.InterestSet, error) {
	// Find the table that follows the heading "Aktuella bolåneräntor – listpris"
	tokenizer, err := utils.FindTokenizedTableByTextBeforeTable(rawHTML, "Aktuella bolåneräntor – listpris")
	if err != nil {
		return nil, fmt.Errorf("failed to find list rates table: %w", err)
	}

	table, err := utils.ParseTable(tokenizer)
	if err != nil {
		return nil, fmt.Errorf("failed to parse list rates table: %w", err)
	}

	// Extract the change date from the header (e.g., "Ränta, senast ändrad 25 september 2025")
	var changedOn *time.Time
	if len(table.Header) >= 2 {
		parsed, err := parseSwedbankListDate(table.Header[1])
		if err != nil {
			c.logger.Warn("failed to parse change date from header", zap.String("header", table.Header[1]), zap.Error(err))
		} else {
			changedOn = &parsed
		}
	}

	// Table structure: Bindningstid | Ränta
	interestSets := []model.InterestSet{}
	for _, row := range table.Rows {
		if len(row) < 2 {
			c.logger.Warn("skipping row with insufficient columns", zap.Strings("row", row))
			continue
		}

		// Skip Banklån rows
		if strings.Contains(strings.ToLower(row[0]), "banklån") {
			continue
		}

		term, err := utils.ParseTerm(row[0])
		if err != nil {
			c.logger.Warn("failed to parse term", zap.String("term", row[0]), zap.Error(err))
			continue
		}

		rate, err := parseSwedbankRate(row[1])
		if err != nil {
			c.logger.Warn("failed to parse rate", zap.String("rate", row[1]), zap.Error(err))
			continue
		}

		interestSets = append(interestSets, model.InterestSet{
			Bank:          swedbankBankName,
			Type:          model.TypeListRate,
			Term:          term,
			NominalRate:   rate,
			ChangedOn:     changedOn,
			LastCrawledAt: crawlTime,
		})
	}

	return interestSets, nil
}

func (c *SwedbankCrawler) extractHistoricAverageRates(rawHTML string, crawlTime time.Time) ([]model.InterestSet, error) {
	// Find the table with caption "Våra historiska genomsnittsräntor"
	tokenizer, err := utils.FindTokenizedTableByTextInCaption(rawHTML, "Våra historiska genomsnittsräntor")
	if err != nil {
		return nil, fmt.Errorf("failed to find historic average rates table: %w", err)
	}

	table, err := utils.ParseTable(tokenizer)
	if err != nil {
		return nil, fmt.Errorf("failed to parse historic average rates table: %w", err)
	}

	// Parse term columns from header: Bindningstid, 3 månader, 1 år, 2 år, ..., 10 år, Banklån*
	termColumns, err := c.parseHistoricTableHeader(table.Header)
	if err != nil {
		return nil, err
	}

	return c.parseHistoricRateRows(table.Rows, termColumns, crawlTime), nil
}

// parseHistoricTableHeader parses the header row and returns a mapping of column index to term.
func (c *SwedbankCrawler) parseHistoricTableHeader(header []string) (map[int]model.Term, error) {
	termColumns := make(map[int]model.Term)

	for i, col := range header {
		if i == 0 { // Skip "Bindningstid" column
			continue
		}
		if strings.Contains(strings.ToLower(col), "banklån") {
			continue
		}

		term, err := utils.ParseTerm(col)
		if err != nil {
			c.logger.Debug("skipping header column", zap.String("column", col), zap.Error(err))
			continue
		}
		termColumns[i] = term
	}

	if len(termColumns) == 0 {
		return nil, fmt.Errorf("no valid term columns found in header")
	}

	return termColumns, nil
}

// parseHistoricRateRows processes rows from the historic average rates table.
// Each row contains: month (nov. 2025), rate for 3m, rate for 1y, ..., rate for 10y, rate for Banklån.
func (c *SwedbankCrawler) parseHistoricRateRows(rows [][]string, termColumns map[int]model.Term, crawlTime time.Time) []model.InterestSet {
	interestSets := []model.InterestSet{}

	for _, row := range rows {
		if len(row) < 2 {
			continue
		}

		// First column is the month (e.g., "nov. 2025")
		avgMonth, err := parseSwedbankHistoricMonth(row[0])
		if err != nil {
			c.logger.Warn("failed to parse historic month", zap.String("month", row[0]), zap.Error(err))
			continue
		}

		// Parse rates for each term column
		for colIndex, term := range termColumns {
			if colIndex >= len(row) {
				continue
			}

			rate, err := parseSwedbankRate(row[colIndex])
			if err != nil {
				continue // Skip empty or invalid rates (e.g., 9 år often has no data)
			}

			interestSets = append(interestSets, model.InterestSet{
				Bank:                  swedbankBankName,
				Type:                  model.TypeAverageRate,
				Term:                  term,
				NominalRate:           rate,
				LastCrawledAt:         crawlTime,
				AverageReferenceMonth: avgMonth,
			})
		}
	}

	return interestSets
}

func parseSwedbankRate(rateStr string) (float32, error) {
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

func parseSwedbankListDate(headerStr string) (time.Time, error) {
	str := utils.NormalizeSpaces(headerStr)
	str = strings.ToLower(str)

	matches := swedbankListDateRegex.FindStringSubmatch(str)
	if matches == nil {
		return time.Time{}, fmt.Errorf("date in %q does not match expected format 'senast ändrad DD month YYYY'", headerStr)
	}

	day, _ := strconv.Atoi(matches[1])
	monthName := matches[2]
	year, _ := strconv.Atoi(matches[3])

	month, ok := swedbankSwedishMonthToTime(monthName)
	if !ok {
		return time.Time{}, fmt.Errorf("unknown Swedish month: %q", monthName)
	}

	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC), nil
}

func parseSwedbankAvgMonth(headerStr string) (*model.AvgMonth, error) {
	str := utils.NormalizeSpaces(headerStr)
	str = strings.ToLower(str)

	matches := swedbankAvgMonthRegex.FindStringSubmatch(str)
	if matches == nil {
		return nil, fmt.Errorf("month in %q does not match expected format 'month YYYY'", headerStr)
	}

	monthName := matches[1]
	year, _ := strconv.Atoi(matches[2])

	month, ok := swedbankSwedishMonthToTime(monthName)
	if !ok {
		return nil, fmt.Errorf("unknown Swedish month: %q", monthName)
	}

	return &model.AvgMonth{
		Year:  uint(year),
		Month: month,
	}, nil
}

// parseSwedbankHistoricMonth parses abbreviated Swedish month format like "nov. 2025" or "okt. 2025".
func parseSwedbankHistoricMonth(monthStr string) (*model.AvgMonth, error) {
	str := utils.NormalizeSpaces(monthStr)
	str = strings.ToLower(str)
	str = strings.ReplaceAll(str, ".", "") // Remove period after abbreviation

	matches := swedbankAvgMonthRegex.FindStringSubmatch(str)
	if matches == nil {
		return nil, fmt.Errorf("month in %q does not match expected format 'mon. YYYY'", monthStr)
	}

	monthName := matches[1]
	year, _ := strconv.Atoi(matches[2])

	month, ok := swedbankSwedishMonthToTime(monthName)
	if !ok {
		return nil, fmt.Errorf("unknown Swedish month: %q", monthName)
	}

	return &model.AvgMonth{
		Year:  uint(year),
		Month: month,
	}, nil
}
