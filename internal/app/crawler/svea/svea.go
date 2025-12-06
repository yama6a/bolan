package svea

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
	"golang.org/x/net/html"
)

const (
	sveaListRatesURL = "https://www.svea.com/sv-se/privat/låna/bolån"
	sveaAvgRatesURL  = "https://www.svea.com/sv-se/privat/låna/bolån/snitträntor"
	sveaBankName     = model.Bank("Svea Bank")
)

var (
	_              crawler.SiteCrawler = &SveaCrawler{}
	sveaRateRegex                      = regexp.MustCompile(`^(\d+[,.]?\d*)\s*%?$`)
	sveaMonthRegex                     = regexp.MustCompile(`(?i)^(\w+)\s+(\d{4})$`)
	// Regex to find the table containing "Månad för utbetalning" text.
	sveaTableRegex = regexp.MustCompile(`(?s)<table[^>]*>.*?Månad för utbetalning.*?</table>`)
	// Regex to extract list rate from "Bolån från X,XX %" (handles &nbsp; as well).
	sveaListRateRegex = regexp.MustCompile(`Bolån från (\d+[,.]?\d*)\s*(?:&nbsp;)?%`)

	sveaSwedishMonthMap = map[string]time.Month{ //nolint: gochecknoglobals
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
)

// SveaCrawler crawls Svea Bank's rates page.
// Svea Bank is a specialty/non-prime lender that only publishes average rates.
// They only offer variable rate (rörlig ränta) mortgages.
//
//nolint:revive // Bank name prefix is intentional for clarity
type SveaCrawler struct {
	httpClient http.Client
	logger     *zap.Logger
}

func NewSveaCrawler(httpClient http.Client, logger *zap.Logger) *SveaCrawler {
	return &SveaCrawler{httpClient: httpClient, logger: logger}
}

func (c *SveaCrawler) Crawl(channel chan<- model.InterestSet) {
	crawlTime := time.Now().UTC()

	// Fetch list rates from main bolån page.
	listHTML, err := c.httpClient.Fetch(sveaListRatesURL, nil)
	if err != nil {
		c.logger.Error("failed fetching Svea list rates page", zap.Error(err))
	} else {
		listRate, err := c.extractListRate(listHTML, crawlTime)
		if err != nil {
			c.logger.Error("failed parsing Svea list rate", zap.Error(err))
		} else {
			channel <- listRate
		}
	}

	// Fetch average rates from snitträntor page.
	avgHTML, err := c.httpClient.Fetch(sveaAvgRatesURL, nil)
	if err != nil {
		c.logger.Error("failed fetching Svea avg rates page", zap.Error(err))
		return
	}

	interestSets, err := c.extractAverageRates(avgHTML, crawlTime)
	if err != nil {
		c.logger.Error("failed parsing Svea avg rates", zap.Error(err))
		return
	}

	for _, set := range interestSets {
		channel <- set
	}
}

// extractListRate parses the list rate from Svea's main bolån page.
// The rate is displayed in the page header as "Bolån från X,XX %".
// Svea only offers variable rate (rörlig ränta) mortgages.
func (c *SveaCrawler) extractListRate(rawHTML string, crawlTime time.Time) (model.InterestSet, error) {
	matches := sveaListRateRegex.FindStringSubmatch(rawHTML)
	if len(matches) != 2 {
		return model.InterestSet{}, fmt.Errorf("failed to find list rate in page")
	}

	rateStr := strings.ReplaceAll(matches[1], ",", ".")
	rate, err := strconv.ParseFloat(rateStr, 32)
	if err != nil {
		return model.InterestSet{}, fmt.Errorf("failed to parse list rate: %w", err)
	}

	return model.InterestSet{
		Bank:          sveaBankName,
		Type:          model.TypeListRate,
		Term:          model.Term3months, // Svea only offers variable rate (3 månader).
		NominalRate:   float32(rate),
		LastCrawledAt: crawlTime,

		ChangedOn:               nil,
		AverageReferenceMonth:   nil,
		RatioDiscountBoundaries: nil,
		UnionDiscount:           false,
	}, nil
}

// extractAverageRates parses average rates from Svea's HTML page.
// Svea only publishes average rates (snitträntor), not list rates.
// Table structure: Månad för utbetalning | Räntesats.
func (c *SveaCrawler) extractAverageRates(rawHTML string, crawlTime time.Time) ([]model.InterestSet, error) {
	// Use regex to find the table containing "Månad för utbetalning".
	// Standard utils.FindTokenizedTableByTextBeforeTable doesn't work here because
	// Svea's HTML has headers inside the table, not before it.
	tableHTML := sveaTableRegex.FindString(rawHTML)
	if tableHTML == "" {
		return nil, fmt.Errorf("failed to find avg rates table with regex")
	}

	tokenizer := html.NewTokenizer(strings.NewReader(tableHTML))
	table, err := utils.ParseTable(tokenizer)
	if err != nil {
		return nil, fmt.Errorf("failed to parse avg rates table: %w", err)
	}

	interestSets := make([]model.InterestSet, 0, len(table.Rows))

	for _, row := range table.Rows {
		if len(row) < 2 {
			continue
		}

		// Parse month from first column (e.g., "November 2025").
		refMonth, err := c.parseSveaMonth(row[0])
		if err != nil {
			c.logger.Warn("failed to parse Svea month", zap.String("month", row[0]), zap.Error(err))
			continue
		}

		// Parse rate from second column (e.g., "6,10 %").
		rate, err := c.parseSveaRate(row[1])
		if err != nil {
			c.logger.Warn("failed to parse Svea rate", zap.String("rate", row[1]), zap.Error(err))
			continue
		}

		refMonthCopy := refMonth
		interestSets = append(interestSets, model.InterestSet{
			Bank:                  sveaBankName,
			Type:                  model.TypeAverageRate,
			Term:                  model.Term3months, // Svea only offers variable rate (3 månader).
			NominalRate:           rate,
			LastCrawledAt:         crawlTime,
			AverageReferenceMonth: &refMonthCopy,

			ChangedOn:               nil,
			RatioDiscountBoundaries: nil,
			UnionDiscount:           false,
		})
	}

	if len(interestSets) == 0 {
		return nil, fmt.Errorf("no average rates extracted from Svea page")
	}

	return interestSets, nil
}

// parseSveaRate parses a rate string like "6,10 %" or "6.10%".
func (c *SveaCrawler) parseSveaRate(rateStr string) (float32, error) {
	rateStr = utils.NormalizeSpaces(rateStr)
	rateStr = strings.ReplaceAll(rateStr, ",", ".")
	rateStr = strings.ReplaceAll(rateStr, " ", "")
	rateStr = strings.TrimSuffix(rateStr, "%")
	rateStr = strings.TrimSpace(rateStr)

	if rateStr == "" || rateStr == "-" {
		return 0, fmt.Errorf("empty or missing rate")
	}

	matches := sveaRateRegex.FindStringSubmatch(rateStr)
	if len(matches) != 2 {
		return 0, fmt.Errorf("failed to match rate regex: %s", rateStr)
	}

	rate, err := strconv.ParseFloat(matches[1], 32)
	if err != nil {
		return 0, fmt.Errorf("failed to parse rate: %w", err)
	}

	return float32(rate), nil
}

// parseSveaMonth parses a month string like "November 2025" to AvgMonth.
func (c *SveaCrawler) parseSveaMonth(monthStr string) (model.AvgMonth, error) {
	monthStr = utils.NormalizeSpaces(monthStr)

	matches := sveaMonthRegex.FindStringSubmatch(monthStr)
	if len(matches) != 3 {
		return model.AvgMonth{}, fmt.Errorf("failed to parse month: %s", monthStr)
	}

	monthName := strings.ToLower(matches[1])
	year, err := strconv.Atoi(matches[2])
	if err != nil {
		return model.AvgMonth{}, fmt.Errorf("failed to parse year: %w", err)
	}

	month, ok := sveaSwedishMonthMap[monthName]
	if !ok {
		return model.AvgMonth{}, fmt.Errorf("unknown Swedish month: %s", monthName)
	}

	return model.AvgMonth{
		Month: month,
		Year:  uint(year),
	}, nil
}
