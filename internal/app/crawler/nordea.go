package crawler

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
	"github.com/yama6a/bolan-compare/internal/pkg/http"
	"github.com/yama6a/bolan-compare/internal/pkg/model"
	"github.com/yama6a/bolan-compare/internal/pkg/utils"
	"go.uber.org/zap"
)

const (
	nordeaListRatesURL      = "https://www.nordea.se/privat/produkter/bolan/listrantor.html"
	nordeaHistoricRatesURL  = "https://www.nordea.se/privat/produkter/bolan/historiska-bolanerantor.html"
	nordeaBankName          = model.Bank("Nordea")
	nordeaHistoricSheetName = "Ränteändringsdagar 1990-2025"
	nordeaHistoricHeaderRow = 6 // 0-indexed row containing column headers
	nordeaHistoricDataStart = 7 // 0-indexed first data row
)

var (
	_ SiteCrawler = &NordeaCrawler{}

	// Nordea date format: YYYY-MM-DD.
	nordeaDateRegex = regexp.MustCompile(`^(\d{4})-(0[1-9]|1[0-2])-(0[1-9]|[12]\d|3[01])$`)
	// Nordea historic date format in XLSX: MM-DD-YY.
	nordeaHistoricDateRegex = regexp.MustCompile(`^(\d{2})-(\d{2})-(\d{2})$`)
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

	// Fetch list rates from HTML
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

	// Fetch historic rates from XLSX file
	historicRates, err := c.fetchHistoricRates(crawlTime)
	if err != nil {
		c.logger.Error("failed fetching Nordea historic rates", zap.Error(err))
	} else {
		interestSets = append(interestSets, historicRates...)
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

// fetchHistoricRates fetches the historic rates page, finds the XLSX link dynamically,
// downloads the file, and parses the historical rate data.
func (c *NordeaCrawler) fetchHistoricRates(crawlTime time.Time) ([]model.InterestSet, error) {
	// Fetch the historic rates page to find the XLSX link
	pageHTML, err := c.httpClient.Fetch(nordeaHistoricRatesURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch historic rates page: %w", err)
	}

	// Find the XLSX link dynamically (robust to filename changes)
	xlsxURL := c.findXLSXLink(pageHTML)
	if xlsxURL == "" {
		return nil, fmt.Errorf("could not find XLSX link on historic rates page")
	}

	c.logger.Debug("found historic rates XLSX", zap.String("url", xlsxURL))

	// Download the XLSX file
	xlsxData, err := c.httpClient.FetchRaw(xlsxURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to download XLSX file: %w", err)
	}

	// Parse the XLSX file
	return c.parseHistoricRatesXLSX(xlsxData, crawlTime)
}

// findXLSXLink searches the HTML for any link to an XLSX file.
// This is robust to filename changes - it finds any .xlsx link on the page.
func (c *NordeaCrawler) findXLSXLink(pageHTML string) string {
	// Find any link to an xlsx file
	xlsxPattern := regexp.MustCompile(`href="([^"]*\.xlsx)"`)
	matches := xlsxPattern.FindAllStringSubmatch(pageHTML, -1)

	if len(matches) == 0 {
		return ""
	}

	// Use the first XLSX link found (there should typically be only one)
	xlsxPath := matches[0][1]

	// Convert relative URL to absolute
	if strings.HasPrefix(xlsxPath, "/") {
		return "https://www.nordea.se" + xlsxPath
	}
	if strings.HasPrefix(xlsxPath, "http") {
		return xlsxPath
	}
	return "https://www.nordea.se/" + xlsxPath
}

// parseHistoricRatesXLSX parses the Nordea historic rates XLSX file.
// Terms are read dynamically from the header row, not hardcoded.
//
//nolint:cyclop // Complexity is inherent in multi-column XLSX parsing with error handling
func (c *NordeaCrawler) parseHistoricRatesXLSX(xlsxData []byte, crawlTime time.Time) ([]model.InterestSet, error) {
	f, err := excelize.OpenReader(bytes.NewReader(xlsxData))
	if err != nil {
		return nil, fmt.Errorf("failed to open XLSX: %w", err)
	}
	defer func() { _ = f.Close() }()

	// Find the data sheet - look for any sheet containing rate data
	sheetName := c.findHistoricDataSheet(f)
	if sheetName == "" {
		return nil, fmt.Errorf("could not find historic rates sheet in XLSX")
	}

	rows, err := f.GetRows(sheetName)
	if err != nil {
		return nil, fmt.Errorf("failed to read rows from sheet %q: %w", sheetName, err)
	}

	// Find the header row dynamically (contains "Ränteändringsdag" or similar)
	headerRowIdx, headerRow := c.findHeaderRow(rows)
	if headerRowIdx < 0 {
		return nil, fmt.Errorf("could not find header row in XLSX")
	}

	// Parse terms from header row (column 1+)
	termColumns := c.parseTermsFromHeader(headerRow)
	if len(termColumns) == 0 {
		return nil, fmt.Errorf("could not find any valid terms in XLSX header")
	}

	c.logger.Debug("parsed XLSX header", zap.Int("termCount", len(termColumns)))

	// Find the date column index (column containing "Ränteändringsdag" in header)
	dateColIdx := 0
	for i, cell := range headerRow {
		if strings.Contains(strings.ToLower(cell), "ränteändringsdag") ||
			strings.Contains(strings.ToLower(cell), "datum") {
			dateColIdx = i
			break
		}
	}

	// Parse data rows (starting after header)
	interestSets := make([]model.InterestSet, 0, (len(rows)-headerRowIdx-1)*len(termColumns))
	for i := headerRowIdx + 1; i < len(rows); i++ {
		row := rows[i]
		if len(row) <= dateColIdx {
			continue
		}

		// Parse date from the identified date column
		dateStr := strings.TrimSpace(row[dateColIdx])
		rateDate, err := parseNordeaHistoricDate(dateStr)
		if err != nil {
			continue // Skip rows without valid dates
		}

		// Parse rates for each term column
		for colIdx, term := range termColumns {
			if colIdx >= len(row) {
				continue
			}

			rateStr := strings.TrimSpace(row[colIdx])
			rate, err := parseNordeaRate(rateStr)
			if err != nil {
				continue // Skip empty or invalid rates
			}

			interestSets = append(interestSets, model.InterestSet{
				Bank:        nordeaBankName,
				Type:        model.TypeAverageRate,
				Term:        term,
				NominalRate: rate,
				AverageReferenceMonth: &model.AvgMonth{
					Month: rateDate.Month(),
					Year:  uint(rateDate.Year()),
				},
				LastCrawledAt: crawlTime,
			})
		}
	}

	return interestSets, nil
}

// findHistoricDataSheet finds the sheet containing historic rate data.
// Looks for sheets with rate-related names, robust to name changes.
func (c *NordeaCrawler) findHistoricDataSheet(f *excelize.File) string {
	sheets := f.GetSheetList()

	// First, try to find a sheet with rate-related keywords
	for _, sheet := range sheets {
		lowerSheet := strings.ToLower(sheet)
		if strings.Contains(lowerSheet, "ränteändring") ||
			strings.Contains(lowerSheet, "ranteandring") ||
			strings.Contains(lowerSheet, "historisk") {
			return sheet
		}
	}

	// Fallback: use the first sheet that's not a diagram
	for _, sheet := range sheets {
		lowerSheet := strings.ToLower(sheet)
		if !strings.Contains(lowerSheet, "diagram") {
			return sheet
		}
	}

	// Last resort: first sheet
	if len(sheets) > 0 {
		return sheets[0]
	}
	return ""
}

// findHeaderRow finds the row containing column headers.
// Returns the row index and the header row content.
// Note: XLSX may have empty first column, so we check multiple columns.
func (c *NordeaCrawler) findHeaderRow(rows [][]string) (int, []string) {
	for i, row := range rows {
		if len(row) == 0 {
			continue
		}
		// Check first few columns for header text (XLSX may have empty first column)
		for j := 0; j < len(row) && j < 3; j++ {
			cell := strings.ToLower(strings.TrimSpace(row[j]))
			if strings.Contains(cell, "ränteändringsdag") ||
				strings.Contains(cell, "ranteandring") ||
				strings.Contains(cell, "datum") {
				return i, row
			}
		}
	}
	return -1, nil
}

// parseTermsFromHeader extracts terms from the XLSX header row.
// Terms are read dynamically using utils.ParseTerm, not hardcoded.
func (c *NordeaCrawler) parseTermsFromHeader(headerRow []string) map[int]model.Term {
	termColumns := make(map[int]model.Term)

	for i := 1; i < len(headerRow); i++ { // Skip first column (date)
		termStr := strings.TrimSpace(headerRow[i])
		if termStr == "" {
			continue
		}

		term, err := utils.ParseTerm(termStr)
		if err != nil {
			c.logger.Debug("skipping XLSX header column", zap.String("column", termStr), zap.Error(err))
			continue
		}
		termColumns[i] = term
	}

	return termColumns
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

func parseNordeaHistoricDate(dateStr string) (time.Time, error) {
	str := utils.NormalizeSpaces(dateStr)
	matches := nordeaHistoricDateRegex.FindStringSubmatch(str)
	if matches == nil {
		return time.Time{}, fmt.Errorf("date %q does not match expected format MM-DD-YY", dateStr)
	}

	month, _ := strconv.Atoi(matches[1])
	day, _ := strconv.Atoi(matches[2])
	year, _ := strconv.Atoi(matches[3])

	// Convert 2-digit year to 4-digit (assume 90+ is 1990s, otherwise 2000s)
	if year >= 90 {
		year += 1900
	} else {
		year += 2000
	}

	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC), nil
}
