package bluestep

import (
	"fmt"
	"html"
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
	bluestepListRatesURL = "https://www.bluestep.se/bolan/borantor/"
	bluestepAvgRatesURL  = "https://www.bluestep.se/bolan/borantor/genomsnittsrantor/"
	bluestepBankName     = model.Bank("Bluestep")
)

var (
	// bluestepRateRegex matches rates like "4,45%" or "5.68%".
	bluestepRateRegex = regexp.MustCompile(`^(\d+[,.]?\d*)\s*%?$`)
	// bluestepMonthRegex matches "YYYY MM" format like "2025 11".
	bluestepMonthRegex = regexp.MustCompile(`^(\d{4})\s+(\d{1,2})$`)
	// bluestepTermRegex extracts term from header like "Rörlig 3 månader" or "Fast 3 år".
	bluestepTermRegex = regexp.MustCompile(`(\d+)\s*(mån|år)`)
)

var _ crawler.SiteCrawler = &BluestepCrawler{}

//nolint:revive // Bank name prefix is intentional for clarity
type BluestepCrawler struct {
	httpClient http.Client
	logger     *zap.Logger
}

func NewBluestepCrawler(httpClient http.Client, logger *zap.Logger) *BluestepCrawler {
	return &BluestepCrawler{httpClient: httpClient, logger: logger}
}

func (c *BluestepCrawler) Crawl(channel chan<- model.InterestSet) {
	interestSets := []model.InterestSet{}
	crawlTime := time.Now().UTC()

	// Fetch and parse list rates
	listHTML, err := c.httpClient.Fetch(bluestepListRatesURL, nil)
	if err != nil {
		c.logger.Error("failed reading Bluestep list rates page", zap.Error(err))
	} else {
		listRates, err := c.extractListRates(listHTML, crawlTime)
		if err != nil {
			c.logger.Error("failed parsing Bluestep list rates", zap.Error(err))
		} else {
			interestSets = append(interestSets, listRates...)
		}
	}

	// Fetch and parse average rates
	avgHTML, err := c.httpClient.Fetch(bluestepAvgRatesURL, nil)
	if err != nil {
		c.logger.Error("failed reading Bluestep average rates page", zap.Error(err))
	} else {
		avgRates, err := c.extractAverageRates(avgHTML, crawlTime)
		if err != nil {
			c.logger.Error("failed parsing Bluestep average rates", zap.Error(err))
		} else {
			interestSets = append(interestSets, avgRates...)
		}
	}

	for _, set := range interestSets {
		channel <- set
	}
}

// extractListRates parses the list rates from Bluestep's HTML page.
// The table structure has terms in the first row (as <td><strong>...</strong></td>)
// and rates in the second row.
func (c *BluestepCrawler) extractListRates(rawHTML string, crawlTime time.Time) ([]model.InterestSet, error) {
	table, err := c.findListRatesTable(rawHTML)
	if err != nil {
		return nil, err
	}

	headerRow, rateRow, err := c.getHeaderAndRateRows(table)
	if err != nil {
		return nil, err
	}

	return c.buildListRateInterestSets(headerRow, rateRow, crawlTime), nil
}

// findListRatesTable locates and parses the list rates table from HTML.
func (c *BluestepCrawler) findListRatesTable(rawHTML string) (*utils.Table, error) {
	// Find the first table after "Bolån*" heading.
	tokenizer, err := utils.FindTokenizedTableByTextBeforeTable(rawHTML, "Bolån*")
	if err != nil {
		// Fallback: try finding with HTML encoded text
		tokenizer, err = utils.FindTokenizedTableByTextBeforeTable(rawHTML, "Bol&aring;n*")
		if err != nil {
			return nil, fmt.Errorf("failed to find list rates table: %w", err)
		}
	}

	table, err := utils.ParseTable(tokenizer)
	if err != nil {
		return nil, fmt.Errorf("failed to parse list rates table: %w", err)
	}

	return &table, nil
}

// getHeaderAndRateRows extracts header and rate rows from the table.
func (c *BluestepCrawler) getHeaderAndRateRows(table *utils.Table) ([]string, []string, error) {
	// The table uses <tbody> without <thead>, so both term and rate rows are in Rows.
	switch {
	case len(table.Header) > 0 && len(table.Rows) > 0:
		// Header from <th> tags, rates from first data row
		return table.Header, table.Rows[0], nil
	case len(table.Rows) >= 2:
		// Both rows are in Rows (no <th> tags used)
		return table.Rows[0], table.Rows[1], nil
	default:
		return nil, nil, fmt.Errorf("list rates table has insufficient rows: header=%d, rows=%d",
			len(table.Header), len(table.Rows))
	}
}

// buildListRateInterestSets creates InterestSet objects from header and rate rows.
func (c *BluestepCrawler) buildListRateInterestSets(headerRow, rateRow []string, crawlTime time.Time) []model.InterestSet {
	interestSets := []model.InterestSet{}

	for i, termStr := range headerRow {
		if i >= len(rateRow) {
			break
		}

		term, err := c.parseBluestepTerm(termStr)
		if err != nil {
			c.logger.Warn("failed to parse term", zap.String("term", termStr), zap.Error(err))
			continue
		}

		rate, err := c.parseBluestepRate(rateRow[i])
		if err != nil {
			c.logger.Warn("failed to parse rate", zap.String("rate", rateRow[i]), zap.Error(err))
			continue
		}

		interestSets = append(interestSets, model.InterestSet{
			Bank:          bluestepBankName,
			Type:          model.TypeListRate,
			Term:          term,
			NominalRate:   rate,
			LastCrawledAt: crawlTime,

			ChangedOn:               nil,
			RatioDiscountBoundaries: nil,
			UnionDiscount:           false,
			AverageReferenceMonth:   nil,
		})
	}

	return interestSets
}

// extractAverageRates parses the average rates from Bluestep's historical rates page.
func (c *BluestepCrawler) extractAverageRates(rawHTML string, crawlTime time.Time) ([]model.InterestSet, error) {
	// The average rates table has a header row with terms and data rows with month + rates.
	tokenizer, err := utils.FindTokenizedTableByTextBeforeTable(rawHTML, "Genomsnittsräntor")
	if err != nil {
		return nil, fmt.Errorf("failed to find average rates table: %w", err)
	}

	table, err := utils.ParseTable(tokenizer)
	if err != nil {
		return nil, fmt.Errorf("failed to parse average rates table: %w", err)
	}

	// Parse header to get terms (skip first column which is "Månad")
	terms := c.extractTermsFromHeader(table.Header)

	interestSets := []model.InterestSet{}
	for _, row := range table.Rows {
		if len(row) == 0 {
			continue
		}

		refMonth, err := c.parseBluestepMonth(row[0])
		if err != nil {
			c.logger.Warn("failed to parse month", zap.String("month", row[0]), zap.Error(err))
			continue
		}

		for i, cell := range row {
			if i == 0 {
				continue // skip month column
			}

			term, ok := terms[i]
			if !ok {
				continue
			}

			rate, err := c.parseBluestepRate(cell)
			if err != nil {
				continue // skip empty or invalid rates
			}

			interestSets = append(interestSets, model.InterestSet{
				Bank:                  bluestepBankName,
				Type:                  model.TypeAverageRate,
				Term:                  term,
				NominalRate:           rate,
				LastCrawledAt:         crawlTime,
				AverageReferenceMonth: &refMonth,

				ChangedOn:               nil,
				RatioDiscountBoundaries: nil,
				UnionDiscount:           false,
			})
		}
	}

	return interestSets, nil
}

// extractTermsFromHeader parses the header row to extract term mappings.
// Returns a map of column index to Term.
func (c *BluestepCrawler) extractTermsFromHeader(header []string) map[int]model.Term {
	terms := make(map[int]model.Term)

	for i, h := range header {
		term, err := utils.ParseTerm(h)
		if err != nil {
			continue // skip non-term columns like "Månad"
		}
		terms[i] = term
	}

	return terms
}

// parseBluestepTerm parses a term string like "Rörlig 3 månader" or "Fast 3 år".
func (c *BluestepCrawler) parseBluestepTerm(termStr string) (model.Term, error) {
	// Decode HTML entities (e.g., &aring; -> å)
	termStr = html.UnescapeString(termStr)
	termStr = utils.NormalizeSpaces(termStr)

	// Extract the numeric part and unit using regex
	matches := bluestepTermRegex.FindStringSubmatch(strings.ToLower(termStr))
	if len(matches) != 3 {
		return "", fmt.Errorf("failed to parse term: %s", termStr)
	}

	num := matches[1]
	unit := matches[2]

	// Reconstruct in standard format for ParseTerm
	var standardTerm string
	if unit == "mån" {
		standardTerm = num + " mån"
	} else {
		standardTerm = num + " år"
	}

	term, err := utils.ParseTerm(standardTerm)
	if err != nil {
		return "", fmt.Errorf("failed to parse term %q: %w", standardTerm, err)
	}

	return term, nil
}

// parseBluestepRate parses a rate string like "4,45%" or "5.68%".
func (c *BluestepCrawler) parseBluestepRate(rateStr string) (float32, error) {
	rateStr = html.UnescapeString(rateStr)
	rateStr = utils.NormalizeSpaces(rateStr)
	rateStr = strings.ReplaceAll(rateStr, ",", ".") // Swedish decimal separator
	rateStr = strings.ReplaceAll(rateStr, " ", "")

	matches := bluestepRateRegex.FindStringSubmatch(rateStr)
	if len(matches) != 2 {
		return 0, fmt.Errorf("failed to match rate regex: %s", rateStr)
	}

	rate, err := strconv.ParseFloat(matches[1], 32)
	if err != nil {
		return 0, fmt.Errorf("failed to parse rate: %w", err)
	}

	return float32(rate), nil
}

// parseBluestepMonth parses a month string like "2025 11" to AvgMonth.
func (c *BluestepCrawler) parseBluestepMonth(monthStr string) (model.AvgMonth, error) {
	monthStr = utils.NormalizeSpaces(monthStr)

	matches := bluestepMonthRegex.FindStringSubmatch(monthStr)
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
