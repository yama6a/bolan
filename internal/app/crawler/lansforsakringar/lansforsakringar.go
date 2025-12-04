package lansforsakringar

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ledongthuc/pdf"
	"github.com/yama6a/bolan-compare/internal/app/crawler"
	"github.com/yama6a/bolan-compare/internal/pkg/http"
	"github.com/yama6a/bolan-compare/internal/pkg/model"
	"github.com/yama6a/bolan-compare/internal/pkg/utils"
	"go.uber.org/zap"
)

const (
	// Länsförsäkringar uses regional URLs but rates are the same across regions.
	// Using Stockholm as the default region.
	lfRatesURL                  = "https://www.lansforsakringar.se/stockholm/privat/bank/bolan/bolaneranta/"
	lfAvgRatesPDFURL            = "https://www.lansforsakringar.se/osfiles/00000-bolanerantor-genomsnittliga.pdf"
	lfBankName       model.Bank = "Länsförsäkringar"
)

var (
	_ crawler.SiteCrawler = &LansforsakringarCrawler{}

	// lfAvgMonthRegex matches "Genomsnittlig ränta oktober 2025" or similar in table header.
	lfAvgMonthRegex = regexp.MustCompile(`(?i)genomsnittlig\s+ränta\s+(\w+)\s+(\d{4})`)
	// lfListDateRegex matches "YYYY-MM-DD" format used in list rates table.
	lfListDateRegex = regexp.MustCompile(`(\d{4})-(\d{2})-(\d{2})`)
	// lfPDFDateRegex matches "YYYYMMDD" format used in PDF (e.g., "20251031" = October 31, 2025).
	lfPDFDateRegex = regexp.MustCompile(`^(20\d{2})(\d{2})(\d{2})$`)
	// lfPDFRateRegex matches Swedish decimal rates like "2,70" or "3,84".
	lfPDFRateRegex = regexp.MustCompile(`^\d+,\d+$`)
)

// lfSwedishMonthMap maps Swedish month names to time.Month.
//
//nolint:gochecknoglobals // constant lookup map to reduce cyclomatic complexity
var lfSwedishMonthMap = map[string]time.Month{
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

//nolint:revive // Bank name prefix is intentional for clarity
type LansforsakringarCrawler struct {
	httpClient http.Client
	logger     *zap.Logger
}

func NewLansforsakringarCrawler(httpClient http.Client, logger *zap.Logger) *LansforsakringarCrawler {
	return &LansforsakringarCrawler{httpClient: httpClient, logger: logger}
}

func (c *LansforsakringarCrawler) Crawl(channel chan<- model.InterestSet) {
	crawlTime := time.Now().UTC()

	rawHTML, err := c.httpClient.Fetch(lfRatesURL, nil)
	if err != nil {
		c.logger.Error("failed reading Länsförsäkringar rates page", zap.Error(err))
		return
	}

	// Extract list rates from HTML
	listRates, err := c.extractListRates(rawHTML, crawlTime)
	if err != nil {
		c.logger.Error("failed parsing Länsförsäkringar list rates", zap.Error(err))
	} else {
		for _, set := range listRates {
			channel <- set
		}
	}

	// Extract historical average rates from PDF
	avgRates, err := c.fetchAverageRatesFromPDF(crawlTime)
	if err != nil {
		c.logger.Error("failed fetching Länsförsäkringar average rates from PDF", zap.Error(err))
	} else {
		for _, set := range avgRates {
			channel <- set
		}
	}
}

// lfColumnIndices holds the column indices for rate and date in the list rates table.
type lfColumnIndices struct {
	rate int
	date int
}

// findListRateColumns finds the rate and date column indices in the list rates table header.
func findListRateColumns(header []string) lfColumnIndices {
	indices := lfColumnIndices{rate: -1, date: -1}
	for i, col := range header {
		colLower := strings.ToLower(col)
		if strings.Contains(colLower, "ränta") && !strings.Contains(colLower, "ändring") {
			indices.rate = i
		}
		if strings.Contains(colLower, "datum") {
			indices.date = i
		}
	}
	return indices
}

func (c *LansforsakringarCrawler) extractListRates(rawHTML string, crawlTime time.Time) ([]model.InterestSet, error) {
	tokenizer, err := utils.FindTokenizedTableByTextBeforeTable(rawHTML, "Bindningstid")
	if err != nil {
		return nil, fmt.Errorf("failed to find list rates table: %w", err)
	}

	table, err := utils.ParseTable(tokenizer)
	if err != nil {
		return nil, fmt.Errorf("failed to parse list rates table: %w", err)
	}

	cols := findListRateColumns(table.Header)
	if cols.rate == -1 {
		return nil, fmt.Errorf("could not find rate column in list rates table")
	}

	return c.parseListRateRows(table.Rows, cols, crawlTime), nil
}

// parseListRateRows parses the list rate rows from the table.
func (c *LansforsakringarCrawler) parseListRateRows(rows [][]string, cols lfColumnIndices, crawlTime time.Time) []model.InterestSet {
	interestSets := []model.InterestSet{}
	for _, row := range rows {
		set, ok := c.parseListRateRow(row, cols, crawlTime)
		if ok {
			interestSets = append(interestSets, set)
		}
	}
	return interestSets
}

// parseListRateRow parses a single list rate row.
func (c *LansforsakringarCrawler) parseListRateRow(row []string, cols lfColumnIndices, crawlTime time.Time) (model.InterestSet, bool) {
	if len(row) <= cols.rate {
		c.logger.Warn("skipping row with insufficient columns", zap.Strings("row", row))
		return model.InterestSet{}, false
	}

	term, err := utils.ParseTerm(row[0])
	if err != nil {
		c.logger.Warn("failed to parse term", zap.String("term", row[0]), zap.Error(err))
		return model.InterestSet{}, false
	}

	rate, err := parseLFRate(row[cols.rate])
	if err != nil {
		c.logger.Warn("failed to parse rate", zap.String("rate", row[cols.rate]), zap.Error(err))
		return model.InterestSet{}, false
	}

	var changedOn *time.Time
	if cols.date >= 0 && cols.date < len(row) {
		if parsed, err := parseLFListDate(row[cols.date]); err == nil {
			changedOn = &parsed
		}
	}

	return model.InterestSet{
		Bank:          lfBankName,
		Type:          model.TypeListRate,
		Term:          term,
		NominalRate:   rate,
		ChangedOn:     changedOn,
		LastCrawledAt: crawlTime,
	}, true
}

// parseLFRate parses a Swedish format rate like "3,84 %" to float32.
func parseLFRate(rateStr string) (float32, error) {
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

// parseLFListDate parses a date in YYYY-MM-DD format.
func parseLFListDate(dateStr string) (time.Time, error) {
	str := utils.NormalizeSpaces(dateStr)
	matches := lfListDateRegex.FindStringSubmatch(str)
	if matches == nil {
		return time.Time{}, fmt.Errorf("date %q does not match expected format YYYY-MM-DD", dateStr)
	}

	year, _ := strconv.Atoi(matches[1])
	month, _ := strconv.Atoi(matches[2])
	day, _ := strconv.Atoi(matches[3])

	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC), nil
}

// parseLFAvgMonth parses the average month from header text like "Genomsnittlig ränta oktober 2025".
func parseLFAvgMonth(headerStr string) (*model.AvgMonth, error) {
	str := utils.NormalizeSpaces(headerStr)
	str = strings.ToLower(str)

	matches := lfAvgMonthRegex.FindStringSubmatch(str)
	if matches == nil {
		return nil, fmt.Errorf("header %q does not match expected format 'Genomsnittlig ränta month YYYY'", headerStr)
	}

	monthName := matches[1]
	year, _ := strconv.Atoi(matches[2])

	month, ok := lfSwedishMonthToTime(monthName)
	if !ok {
		return nil, fmt.Errorf("unknown Swedish month: %q", monthName)
	}

	return &model.AvgMonth{
		Year:  uint(year),
		Month: month,
	}, nil
}

// lfSwedishMonthToTime converts Swedish month names to time.Month.
func lfSwedishMonthToTime(monthName string) (time.Month, bool) {
	m, ok := lfSwedishMonthMap[monthName]
	return m, ok
}

// fetchAverageRatesFromPDF fetches and parses the historical average rates PDF.
func (c *LansforsakringarCrawler) fetchAverageRatesFromPDF(crawlTime time.Time) ([]model.InterestSet, error) {
	pdfContent, err := c.httpClient.FetchRaw(lfAvgRatesPDFURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch PDF: %w", err)
	}

	return c.parsePDF(pdfContent, crawlTime)
}

// parsePDF extracts average rates from the Länsförsäkringar PDF document.
// PDF format: First column is date (YYYYMMDD), followed by rate columns for each term.
// Header: "Ränta procent", "3 Månader", "1 År", "2 År", "3 År", "4 År", "5 År", "7 År", "10 År".
func (c *LansforsakringarCrawler) parsePDF(pdfContent []byte, crawlTime time.Time) ([]model.InterestSet, error) {
	reader, err := pdf.NewReader(bytes.NewReader(pdfContent), int64(len(pdfContent)))
	if err != nil {
		return nil, fmt.Errorf("failed to create PDF reader: %w", err)
	}

	// Extract text from all pages
	var allText strings.Builder
	numPages := reader.NumPage()
	for i := 1; i <= numPages; i++ {
		page := reader.Page(i)
		if page.V.IsNull() {
			continue
		}
		text, err := page.GetPlainText(nil)
		if err != nil {
			c.logger.Warn("failed to extract text from PDF page", zap.Int("page", i), zap.Error(err))
			continue
		}
		allText.WriteString(text)
		allText.WriteString("\n")
	}

	return c.parseAverageRatesPDFText(allText.String(), crawlTime)
}

// parseAverageRatesPDFText extracts average rates from the PDF text content.
func (c *LansforsakringarCrawler) parseAverageRatesPDFText(text string, crawlTime time.Time) ([]model.InterestSet, error) {
	// Extract terms from header
	terms := extractLFTermsFromHeader(text)
	if len(terms) == 0 {
		return nil, fmt.Errorf("could not find terms in PDF header")
	}

	c.logger.Debug("extracted terms from PDF header", zap.Int("count", len(terms)))

	// Parse all data rows
	rows := c.parsePDFDataRows(text, terms, crawlTime)
	if len(rows) == 0 {
		return nil, fmt.Errorf("no rate data found in PDF")
	}

	return rows, nil
}

// extractLFTermsFromHeader parses terms from PDF header line.
// Expected: "3 Månader", "1 År", "2 År", "3 År", "4 År", "5 År", "7 År", "10 År".
func extractLFTermsFromHeader(text string) []model.Term {
	// Look for the header pattern with terms
	termPattern := regexp.MustCompile(`(\d+)\s*(Månader|År|månader|år)`)
	matches := termPattern.FindAllStringSubmatch(text, -1)

	seen := make(map[model.Term]bool)
	terms := make([]model.Term, 0, 8)

	for _, match := range matches {
		termStr := match[1] + " " + strings.ToLower(match[2])
		// Normalize "månader" to "mån"
		termStr = strings.Replace(termStr, "månader", "mån", 1)

		term, err := utils.ParseTerm(termStr)
		if err != nil {
			continue
		}

		// Only add each term once, preserving order
		if !seen[term] {
			seen[term] = true
			terms = append(terms, term)
		}
	}

	return terms
}

// parsePDFDataRows parses all data rows from PDF text.
func (c *LansforsakringarCrawler) parsePDFDataRows(text string, terms []model.Term, crawlTime time.Time) []model.InterestSet {
	var results []model.InterestSet

	// Split text into lines/tokens
	tokens := strings.Fields(text)

	for i, token := range tokens {
		// Check if this token is a date (YYYYMMDD format)
		avgMonth, ok := parseLFPDFDate(token)
		if !ok {
			continue
		}

		// Collect rate values following the date
		rates := collectLFPDFRates(tokens, i+1, len(terms))
		if len(rates) == 0 {
			continue
		}

		// Create InterestSet for each term with a valid rate
		for j, rate := range rates {
			if j >= len(terms) {
				break
			}
			if rate < 0 {
				continue // Skip empty rates
			}

			results = append(results, model.InterestSet{
				Bank:                  lfBankName,
				Type:                  model.TypeAverageRate,
				Term:                  terms[j],
				NominalRate:           rate,
				AverageReferenceMonth: &model.AvgMonth{Year: avgMonth.Year, Month: avgMonth.Month},
				LastCrawledAt:         crawlTime,
			})
		}
	}

	return results
}

// parseLFPDFDate parses a date in YYYYMMDD format to year and month.
func parseLFPDFDate(dateStr string) (*model.AvgMonth, bool) {
	matches := lfPDFDateRegex.FindStringSubmatch(dateStr)
	if matches == nil {
		return nil, false
	}

	year, _ := strconv.Atoi(matches[1])
	month, _ := strconv.Atoi(matches[2])

	if month < 1 || month > 12 {
		return nil, false
	}

	return &model.AvgMonth{
		Year:  uint(year),
		Month: time.Month(month),
	}, true
}

// collectLFPDFRates collects rate values from tokens starting at the given index.
// Returns -1 for empty/missing rates.
func collectLFPDFRates(tokens []string, startIdx int, maxRates int) []float32 {
	rates := make([]float32, 0, maxRates)

	for i := startIdx; i < len(tokens) && len(rates) < maxRates; i++ {
		token := tokens[i]

		// Stop if we hit another date (next row)
		if lfPDFDateRegex.MatchString(token) {
			break
		}

		// Check if this is a rate value
		if lfPDFRateRegex.MatchString(token) {
			rate, err := parseLFRate(token)
			if err != nil {
				rates = append(rates, -1) // Mark as invalid
			} else {
				rates = append(rates, rate)
			}
		}
	}

	return rates
}
