package jak

import (
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
	jakRatesURL = "https://www.jak.se/snittranta/"
	jakBankName = model.Bank("JAK Medlemsbank")
)

var (
	_             crawler.SiteCrawler = &JAKCrawler{}
	jakRateRegex                      = regexp.MustCompile(`^(\d+[,.]?\d*)\s*%?$`)
	jakMonthRegex                     = regexp.MustCompile(`^(\d{4})\s+(\d{1,2})$`)
	// JAK's HTML is malformed in two ways:
	// 1. Rows start with <td> directly instead of <tr><td>.
	// 2. Cells start with <td> without closing the previous <td> with </td>.
	jakMalformedRowRegex = regexp.MustCompile(`(</tr>|<tbody>)\s*<td`)
	jakUnclosedCellRegex = regexp.MustCompile(`<td([^>]*)>\s*([^<]*)\s*<td`)
)

// JAKCrawler crawls JAK Medlemsbank's rates page.
// JAK is an ethical/cooperative bank with a unique "sparlånesystem".
// They only offer 2 binding periods: 3 månader and 12 månader.
//
//nolint:revive // Bank name prefix is intentional for clarity
type JAKCrawler struct {
	httpClient http.Client
	logger     *zap.Logger
}

func NewJAKCrawler(httpClient http.Client, logger *zap.Logger) *JAKCrawler {
	return &JAKCrawler{httpClient: httpClient, logger: logger}
}

func (c *JAKCrawler) Crawl(channel chan<- model.InterestSet) {
	crawlTime := time.Now().UTC()

	rawHTML, err := c.httpClient.Fetch(jakRatesURL, nil)
	if err != nil {
		c.logger.Error("failed fetching JAK rates page", zap.Error(err))
		return
	}

	interestSets, err := c.extractRates(rawHTML, crawlTime)
	if err != nil {
		c.logger.Error("failed parsing JAK rates", zap.Error(err))
		return
	}

	for _, set := range interestSets {
		channel <- set
	}
}

// extractRates parses both list and average rates from JAK's HTML page.
// JAK has two tables: one for "3 månader" and one for "12 månader".
// Each table contains both list rates (Listränta) and average rates (Snittränta).
func (c *JAKCrawler) extractRates(rawHTML string, crawlTime time.Time) ([]model.InterestSet, error) {
	interestSets := make([]model.InterestSet, 0, 50)

	// Extract 3 månader rates
	threeMonthSets, err := c.extractTermRates(rawHTML, "List- och snittränta 3 månader", model.Term3months, crawlTime)
	if err != nil {
		c.logger.Warn("failed to extract 3 month rates", zap.Error(err))
	} else {
		interestSets = append(interestSets, threeMonthSets...)
	}

	// Extract 12 månader rates (which is 1 year)
	oneYearSets, err := c.extractTermRates(rawHTML, "List- och snittränta 12 månader", model.Term1year, crawlTime)
	if err != nil {
		c.logger.Warn("failed to extract 12 month rates", zap.Error(err))
	} else {
		interestSets = append(interestSets, oneYearSets...)
	}

	if len(interestSets) == 0 {
		return nil, fmt.Errorf("no rates extracted from JAK page")
	}

	return interestSets, nil
}

// extractTermRates extracts rates for a specific term from the table.
func (c *JAKCrawler) extractTermRates(rawHTML, tableMarker string, term model.Term, crawlTime time.Time) ([]model.InterestSet, error) {
	// Fix JAK's malformed HTML where rows start with <td> instead of <tr><td>
	fixedHTML := fixMalformedTableHTML(rawHTML)

	tokenizer, err := utils.FindTokenizedTableByTextBeforeTable(fixedHTML, tableMarker)
	if err != nil {
		return nil, fmt.Errorf("failed to find table for %s: %w", tableMarker, err)
	}

	table, err := utils.ParseTable(tokenizer)
	if err != nil {
		return nil, fmt.Errorf("failed to parse table for %s: %w", tableMarker, err)
	}

	return c.parseRateTable(table, term, crawlTime), nil
}

// parseRateTable parses a JAK rate table.
// Table structure: Tidsperiod | Listränta | Snittränta | Sparkrav
// First row is the current month's list rate.
func (c *JAKCrawler) parseRateTable(table utils.Table, term model.Term, crawlTime time.Time) []model.InterestSet {
	interestSets := make([]model.InterestSet, 0, len(table.Rows)*2)

	for i, row := range table.Rows {
		if len(row) < 3 {
			continue
		}

		// Parse month from first column (format: "YYYY MM")
		refMonth, err := c.parseJAKMonth(row[0])
		if err != nil {
			c.logger.Warn("failed to parse JAK month", zap.String("month", row[0]), zap.Error(err))
			continue
		}

		// Parse list rate (second column) - only for the first/current row
		if i == 0 {
			listRate, err := c.parseJAKRate(row[1])
			if err == nil {
				interestSets = append(interestSets, model.InterestSet{
					Bank:          jakBankName,
					Type:          model.TypeListRate,
					Term:          term,
					NominalRate:   listRate,
					LastCrawledAt: crawlTime,

					ChangedOn:               nil,
					RatioDiscountBoundaries: nil,
					UnionDiscount:           false,
					AverageReferenceMonth:   nil,
				})
			}
		}

		// Parse average rate (third column)
		avgRate, err := c.parseJAKRate(row[2])
		if err != nil {
			// Skip rows with missing average rates (like "-")
			continue
		}

		refMonthCopy := refMonth
		interestSets = append(interestSets, model.InterestSet{
			Bank:                  jakBankName,
			Type:                  model.TypeAverageRate,
			Term:                  term,
			NominalRate:           avgRate,
			LastCrawledAt:         crawlTime,
			AverageReferenceMonth: &refMonthCopy,

			ChangedOn:               nil,
			RatioDiscountBoundaries: nil,
			UnionDiscount:           false,
		})
	}

	return interestSets
}

// parseJAKRate parses a rate string like "3,58 %" or "3.24%".
func (c *JAKCrawler) parseJAKRate(rateStr string) (float32, error) {
	rateStr = utils.NormalizeSpaces(rateStr)
	rateStr = strings.ReplaceAll(rateStr, ",", ".")
	rateStr = strings.ReplaceAll(rateStr, " ", "")
	rateStr = strings.TrimSuffix(rateStr, "%")

	// Handle double percent signs like "3,58 % %"
	rateStr = strings.TrimSuffix(rateStr, "%")
	rateStr = strings.TrimSpace(rateStr)

	// Skip empty or dash values
	if rateStr == "" || rateStr == "-" {
		return 0, fmt.Errorf("empty or missing rate")
	}

	matches := jakRateRegex.FindStringSubmatch(rateStr)
	if len(matches) != 2 {
		return 0, fmt.Errorf("failed to match rate regex: %s", rateStr)
	}

	rate, err := strconv.ParseFloat(matches[1], 32)
	if err != nil {
		return 0, fmt.Errorf("failed to parse rate: %w", err)
	}

	return float32(rate), nil
}

// fixMalformedTableHTML fixes JAK's malformed HTML where:
// 1. Table rows start with <td> directly instead of <tr><td>
// 2. Table cells start with <td> without closing the previous cell with </td>
// This is a workaround for broken HTML from jak.se.
func fixMalformedTableHTML(htmlStr string) string {
	// First fix: unclosed cells like "<td>content<td>" -> "<td>content</td><td>"
	// We need to loop because the regex can't handle overlapping matches
	for {
		newHTML := jakUnclosedCellRegex.ReplaceAllString(htmlStr, "<td$1>$2</td><td")
		if newHTML == htmlStr {
			break
		}
		htmlStr = newHTML
	}

	// Second fix: missing <tr> tags like "</tr><td" -> "</tr><tr><td"
	htmlStr = jakMalformedRowRegex.ReplaceAllStringFunc(htmlStr, func(match string) string {
		if strings.HasPrefix(match, "</tr>") {
			return "</tr><tr><td"
		}
		return "<tbody><tr><td"
	})

	return htmlStr
}

// parseJAKMonth parses a month string like "2025 11" to AvgMonth.
func (c *JAKCrawler) parseJAKMonth(monthStr string) (model.AvgMonth, error) {
	monthStr = utils.NormalizeSpaces(monthStr)

	matches := jakMonthRegex.FindStringSubmatch(monthStr)
	if len(matches) != 3 {
		return model.AvgMonth{}, fmt.Errorf("failed to parse month: %s", monthStr)
	}

	year, err := strconv.Atoi(matches[1])
	if err != nil {
		return model.AvgMonth{}, fmt.Errorf("failed to parse year: %w", err)
	}

	month, err := strconv.Atoi(matches[2])
	if err != nil {
		return model.AvgMonth{}, fmt.Errorf("failed to parse month number: %w", err)
	}

	if month < 1 || month > 12 {
		return model.AvgMonth{}, fmt.Errorf("invalid month: %d", month)
	}

	return model.AvgMonth{
		Month: time.Month(month),
		Year:  uint(year),
	}, nil
}
