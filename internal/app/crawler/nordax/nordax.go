package nordax

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
	"go.uber.org/zap"
)

const (
	nordaxAvgRatesURL = "https://www.nordax.se/lana/bolan/genomsnittsrantor"
	nordaxBankName    = model.Bank("Nordax Bank")
)

var (
	_                   crawler.SiteCrawler = &NordaxCrawler{}
	nordaxRateRegex                         = regexp.MustCompile(`^(\d+[,.]?\d*)\s*%?$`)
	nordaxNextDataRegex                     = regexp.MustCompile(`<script id="__NEXT_DATA__" type="application/json">(.+?)</script>`)
)

// NordaxCrawler crawls Nordax Bank's rates page.
// Nordax Bank is a specialty/non-prime lender (NOBA Bank Group) that only publishes average rates (snitträntor).
// The data is embedded in Next.js __NEXT_DATA__ JSON in the HTML page.
//
//nolint:revive // Bank name prefix is intentional for clarity
type NordaxCrawler struct {
	httpClient http.Client
	logger     *zap.Logger
}

// nextDataPage represents the page structure in Next.js __NEXT_DATA__.
type nextDataPage struct {
	Content []pageContent `json:"content"`
}

// pageContent represents a content block in the page.
type pageContent struct {
	ExpandableContent []expandableContent `json:"expandableContent"`
}

// expandableContent represents expandable content blocks.
type expandableContent struct {
	Body []contentBody `json:"body"`
}

// contentBody represents body content which can be a table.
type contentBody struct {
	Type    string       `json:"_type"` //nolint:tagliatelle // Nordax uses _type in JSON
	Content tableContent `json:"content"`
}

// tableContent represents the table structure.
type tableContent struct {
	Rows []tableRow `json:"rows"`
}

// tableRow represents a single row in the table.
type tableRow struct {
	Cells []string `json:"cells"`
}

// nextDataResponse represents the full Next.js __NEXT_DATA__ structure.
type nextDataResponse struct {
	Props nextDataProps `json:"props"`
}

// nextDataProps represents the props section.
type nextDataProps struct {
	PageProps nextDataPageProps `json:"pageProps"`
}

// nextDataPageProps represents the page props section.
type nextDataPageProps struct {
	Page nextDataPage `json:"page"`
}

func NewNordaxCrawler(httpClient http.Client, logger *zap.Logger) *NordaxCrawler {
	return &NordaxCrawler{httpClient: httpClient, logger: logger}
}

func (c *NordaxCrawler) Crawl(channel chan<- model.InterestSet) {
	crawlTime := time.Now().UTC()

	// Fetch average rates page.
	avgHTML, err := c.httpClient.Fetch(nordaxAvgRatesURL, nil)
	if err != nil {
		c.logger.Error("failed fetching Nordax avg rates page", zap.Error(err))
		return
	}

	interestSets, err := c.extractAverageRates(avgHTML, crawlTime)
	if err != nil {
		c.logger.Error("failed parsing Nordax avg rates", zap.Error(err))
		return
	}

	for _, set := range interestSets {
		channel <- set
	}
}

// extractAverageRates parses the average rates from the Next.js JSON data.
//
//nolint:cyclop // JSON navigation and parsing requires multiple checks
func (c *NordaxCrawler) extractAverageRates(rawHTML string, crawlTime time.Time) ([]model.InterestSet, error) {
	// Extract __NEXT_DATA__ JSON from HTML.
	matches := nordaxNextDataRegex.FindStringSubmatch(rawHTML)
	if len(matches) < 2 {
		return nil, fmt.Errorf("could not find __NEXT_DATA__ in HTML")
	}

	var nextData nextDataResponse
	if err := json.Unmarshal([]byte(matches[1]), &nextData); err != nil {
		return nil, fmt.Errorf("failed to parse __NEXT_DATA__ JSON: %w", err)
	}

	// Navigate to the table data.
	if len(nextData.Props.PageProps.Page.Content) == 0 {
		return nil, fmt.Errorf("no content found in page")
	}

	if len(nextData.Props.PageProps.Page.Content[0].ExpandableContent) == 0 {
		return nil, fmt.Errorf("no expandable content found")
	}

	if len(nextData.Props.PageProps.Page.Content[0].ExpandableContent[0].Body) == 0 {
		return nil, fmt.Errorf("no body content found")
	}

	tableBody := nextData.Props.PageProps.Page.Content[0].ExpandableContent[0].Body[0]
	if tableBody.Type != "table" {
		return nil, fmt.Errorf("expected table type, got: %s", tableBody.Type)
	}

	rows := tableBody.Content.Rows
	if len(rows) < 2 {
		return nil, fmt.Errorf("table has insufficient rows")
	}

	// First row is the header: ["Datum", "3 månaders", "36 månaders", "60 månaders"].
	header := rows[0].Cells
	if len(header) < 2 {
		return nil, fmt.Errorf("header row has insufficient columns")
	}

	// Map header columns to terms.
	termMap := make(map[int]model.Term)
	for i := 1; i < len(header); i++ {
		term, err := parseNordaxTerm(header[i])
		if err != nil {
			c.logger.Warn("failed to parse term from header", zap.String("header", header[i]), zap.Error(err))
			continue
		}
		termMap[i] = term
	}

	if len(termMap) == 0 {
		return nil, fmt.Errorf("no valid terms found in header")
	}

	var results []model.InterestSet

	// Process data rows (skip header row).
	for _, row := range rows[1:] {
		if len(row.Cells) == 0 {
			continue
		}

		// First cell is the date in "YYYY-MM" format.
		dateStr := strings.TrimSpace(row.Cells[0])
		month, err := parseNordaxMonth(dateStr)
		if err != nil {
			c.logger.Warn("failed to parse month", zap.String("date", dateStr), zap.Error(err))
			continue
		}

		// Process each term column.
		for colIdx, term := range termMap {
			if colIdx >= len(row.Cells) {
				continue
			}

			rateStr := strings.TrimSpace(row.Cells[colIdx])
			if rateStr == "" {
				// Empty cell - no data for this term in this month.
				continue
			}

			rate, err := parseNordaxRate(rateStr)
			if err != nil {
				c.logger.Warn("failed to parse rate", zap.String("rate", rateStr), zap.Error(err))
				continue
			}

			results = append(results, model.InterestSet{
				Bank:        nordaxBankName,
				Term:        term,
				Type:        model.TypeAverageRate,
				NominalRate: float32(rate),
				AverageReferenceMonth: &model.AvgMonth{
					Month: month.Month(),
					Year:  uint(month.Year()),
				},
				LastCrawledAt: crawlTime,
			})
		}
	}

	return results, nil
}

// parseNordaxTerm parses Swedish term strings like "3 månaders", "36 månaders", "60 månaders".
func parseNordaxTerm(termStr string) (model.Term, error) {
	termStr = strings.TrimSpace(strings.ToLower(termStr))

	// Extract number of months.
	if strings.Contains(termStr, "månader") {
		parts := strings.Fields(termStr)
		if len(parts) < 1 {
			return "", fmt.Errorf("invalid term format: %s", termStr)
		}

		months, err := strconv.Atoi(parts[0])
		if err != nil {
			return "", fmt.Errorf("failed to parse months: %w", err)
		}

		switch months {
		case 3:
			return model.Term3months, nil
		case 36:
			return model.Term3years, nil
		case 60:
			return model.Term5years, nil
		default:
			return "", fmt.Errorf("unsupported term: %d months", months)
		}
	}

	return "", fmt.Errorf("unrecognized term format: %s", termStr)
}

// parseNordaxRate parses Swedish rate format like "4,66%" or "4.66%".
func parseNordaxRate(rateStr string) (float64, error) {
	rateStr = strings.TrimSpace(rateStr)

	matches := nordaxRateRegex.FindStringSubmatch(rateStr)
	if len(matches) < 2 {
		return 0, fmt.Errorf("invalid rate format: %s", rateStr)
	}

	// Replace comma with period for parsing.
	rateNumStr := strings.ReplaceAll(matches[1], ",", ".")
	rate, err := strconv.ParseFloat(rateNumStr, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse rate: %w", err)
	}

	return rate, nil
}

// parseNordaxMonth parses date strings in "YYYY-MM" format.
func parseNordaxMonth(dateStr string) (time.Time, error) {
	dateStr = strings.TrimSpace(dateStr)

	// Parse "YYYY-MM" format.
	t, err := time.Parse("2006-01", dateStr)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse date %s: %w", dateStr, err)
	}

	return t, nil
}
