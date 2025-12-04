package skandia

import (
	"encoding/json"
	"fmt"
	"html"
	"regexp"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/yama6a/bolan-compare/internal/app/crawler"
	"github.com/yama6a/bolan-compare/internal/pkg/http"
	"github.com/yama6a/bolan-compare/internal/pkg/model"
)

const (
	skandiaBankName     model.Bank = "Skandia"
	skandiaListRatesURL string     = "https://www.skandia.se/lana/bolan/bolanerantor/"
	skandiaAvgRatesURL  string     = "https://www.skandia.se/lana/bolan/bolanerantor/snittrantor/"
)

var (
	_ crawler.SiteCrawler = &SkandiaCrawler{}

	// Regex to extract SKB.pageContent JSON from HTML.
	skandiaPageContentRgx = regexp.MustCompile(`SKB\.pageContent\s*=\s*(\{[\s\S]*?\});?\s*(?:SKB\.|</script>)`)

	// Regex to extract rate values like "3,45 %" from HTML content.
	skandiaRateRgx = regexp.MustCompile(`([0-9]+,[0-9]+)\s*%`)

	// Regex to extract term like "3 mån" or "1 år" from HTML content.
	skandiaTermRgx = regexp.MustCompile(`([0-9]+)\s*(mån|år)`)

	// Regex to extract month and year from header like "Snitträntor november 2025".
	skandiaMonthYearRgx = regexp.MustCompile(`(?i)(januari|februari|mars|april|maj|juni|juli|augusti|september|oktober|november|december)\s+(\d{4})`)
)

// skandiaMonthMap maps Swedish month names to month numbers.
//
//nolint:gochecknoglobals // constant map for month name conversion
var skandiaMonthMap = map[string]int{
	"januari":   1,
	"februari":  2,
	"mars":      3,
	"april":     4,
	"maj":       5,
	"juni":      6,
	"juli":      7,
	"augusti":   8,
	"september": 9,
	"oktober":   10,
	"november":  11,
	"december":  12,
}

//nolint:revive // Bank name prefix is intentional for clarity
type SkandiaCrawler struct {
	httpClient http.Client
	logger     *zap.Logger
}

// skandiaPageContent represents the top-level SKB.pageContent JSON structure.
type skandiaPageContent struct {
	SectionContent1 []skandiaSectionContent `json:"sectionContent1"`
	SectionContent2 []skandiaSectionContent `json:"sectionContent2"`
}

type skandiaSectionContent struct {
	ContentLink struct {
		Expanded *skandiaExpandedBlock `json:"expanded"`
	} `json:"contentLink"`
}

type skandiaExpandedBlock struct {
	Name        string   `json:"name"`
	ContentType []string `json:"contentType"`
	Header      *struct {
		HeaderText string `json:"headerText"`
	} `json:"header"`
	Columns   []skandiaColumn         `json:"columns"`
	Items     []skandiaItem           `json:"items"`     // For accordion items in avg rates page
	Questions []skandiaFAQItem        `json:"questions"` // For FAQ block items
	Blocks    []skandiaSectionContent `json:"blocks"`    // For nested blocks in FAQ items
}

type skandiaColumn struct {
	ContentLink struct {
		Expanded *skandiaColumnExpanded `json:"expanded"`
	} `json:"contentLink"`
}

type skandiaColumnExpanded struct {
	Name       string   `json:"name"`
	CellHeader string   `json:"cellHeader"`
	Cells      []string `json:"cells"`
}

type skandiaItem struct {
	ContentLink struct {
		Expanded *skandiaItemExpanded `json:"expanded"`
	} `json:"contentLink"`
}

type skandiaItemExpanded struct {
	Name        string   `json:"name"`
	ContentType []string `json:"contentType"`
	Header      *struct {
		HeaderText string `json:"headerText"`
	} `json:"header"`
	Columns []skandiaColumn `json:"columns"`
}

type skandiaFAQItem struct {
	ContentLink struct {
		Expanded *skandiaFAQItemExpanded `json:"expanded"`
	} `json:"contentLink"`
}

type skandiaFAQItemExpanded struct {
	Name        string                  `json:"name"`
	ContentType []string                `json:"contentType"`
	Question    string                  `json:"question"`
	Blocks      []skandiaSectionContent `json:"blocks"`
}

func NewSkandiaCrawler(httpClient http.Client, logger *zap.Logger) *SkandiaCrawler {
	return &SkandiaCrawler{httpClient: httpClient, logger: logger}
}

func (c *SkandiaCrawler) Crawl(channel chan<- model.InterestSet) {
	crawlTime := time.Now().UTC()

	listRates, err := c.fetchListRates(crawlTime)
	if err != nil {
		c.logger.Error("failed fetching Skandia list rates", zap.Error(err))
	}

	avgRates, err := c.fetchAverageRates(crawlTime)
	if err != nil {
		c.logger.Error("failed fetching Skandia average rates", zap.Error(err))
	}

	for _, set := range append(listRates, avgRates...) {
		channel <- set
	}
}

func (c *SkandiaCrawler) fetchListRates(crawlTime time.Time) ([]model.InterestSet, error) {
	html, err := c.httpClient.Fetch(skandiaListRatesURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed reading Skandia list rates page: %w", err)
	}

	pageContent, err := extractSkandiaPageContent(html)
	if err != nil {
		return nil, fmt.Errorf("failed extracting Skandia page content: %w", err)
	}

	interestSets := []model.InterestSet{}

	// Find the Listräntor table in sectionContent2
	for _, section := range pageContent.SectionContent2 {
		if section.ContentLink.Expanded == nil {
			continue
		}

		block := section.ContentLink.Expanded

		// Check if this is a TableBlock with "Listräntor" in the name
		if !isTableBlock(block.ContentType) {
			continue
		}

		if block.Header == nil || !strings.Contains(block.Header.HeaderText, "Listräntor") {
			continue
		}

		// Extract list rates from the table columns
		rates, err := c.parseSkandiaListRateTable(block.Columns, crawlTime)
		if err != nil {
			c.logger.Warn("failed parsing Skandia list rate table", zap.Error(err))
			continue
		}

		interestSets = append(interestSets, rates...)
	}

	if len(interestSets) == 0 {
		return nil, fmt.Errorf("no list rates found in Skandia page")
	}

	return interestSets, nil
}

//nolint:gocognit,cyclop // complex logic for parsing nested JSON structure
func (c *SkandiaCrawler) fetchAverageRates(crawlTime time.Time) ([]model.InterestSet, error) {
	html, err := c.httpClient.Fetch(skandiaAvgRatesURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed reading Skandia average rates page: %w", err)
	}

	pageContent, err := extractSkandiaPageContent(html)
	if err != nil {
		return nil, fmt.Errorf("failed extracting Skandia page content: %w", err)
	}

	interestSets := []model.InterestSet{}

	// Find snitträntor tables in both sectionContent1 and sectionContent2
	allSections := make([]skandiaSectionContent, 0, len(pageContent.SectionContent1)+len(pageContent.SectionContent2))
	allSections = append(allSections, pageContent.SectionContent1...)
	allSections = append(allSections, pageContent.SectionContent2...)

	for _, section := range allSections {
		if section.ContentLink.Expanded == nil {
			continue
		}

		block := section.ContentLink.Expanded

		// Check for TableBlock (current month's snitträntor)
		if isTableBlock(block.ContentType) && block.Header != nil && strings.Contains(block.Header.HeaderText, "Snitträntor") {
			avgMonth, err := parseSkandiaMonthYear(block.Header.HeaderText)
			if err != nil {
				c.logger.Warn("failed parsing Skandia snitträntor header", zap.String("header", block.Header.HeaderText), zap.Error(err))
				continue
			}

			rates, err := c.parseSkandiaAvgRateTable(block.Columns, avgMonth, crawlTime)
			if err != nil {
				c.logger.Warn("failed parsing Skandia avg rate table", zap.Error(err))
				continue
			}

			interestSets = append(interestSets, rates...)
		}

		// Check for AccordionBlock (previous months' snitträntor)
		if isAccordionBlock(block.ContentType) && len(block.Items) > 0 {
			for _, item := range block.Items {
				if item.ContentLink.Expanded == nil {
					continue
				}

				itemBlock := item.ContentLink.Expanded
				if !isTableBlock(itemBlock.ContentType) || itemBlock.Header == nil {
					continue
				}

				avgMonth, err := parseSkandiaMonthYear(itemBlock.Header.HeaderText)
				if err != nil {
					c.logger.Warn("failed parsing Skandia accordion header", zap.String("header", itemBlock.Header.HeaderText), zap.Error(err))
					continue
				}

				rates, err := c.parseSkandiaAvgRateTable(itemBlock.Columns, avgMonth, crawlTime)
				if err != nil {
					c.logger.Warn("failed parsing Skandia accordion avg rate table", zap.Error(err))
					continue
				}

				interestSets = append(interestSets, rates...)
			}
		}

		// Check for FAQBlock (historical months in accordion/FAQ format)
		//nolint:nestif // FAQ processing requires nested iteration
		if isFAQBlock(block.ContentType) && len(block.Questions) > 0 {
			for _, faqItem := range block.Questions {
				if faqItem.ContentLink.Expanded == nil {
					continue
				}

				faqExpanded := faqItem.ContentLink.Expanded
				// The FAQ item question contains the month name (e.g. "Oktober 2025")
				monthStr := faqExpanded.Question
				avgMonth, err := parseSkandiaMonthYear(monthStr)
				if err != nil {
					c.logger.Warn("failed parsing Skandia FAQ month", zap.String("question", monthStr), zap.Error(err))
					continue
				}

				// Each FAQ item has blocks array containing the TableBlock
				for _, nestedSection := range faqExpanded.Blocks {
					if nestedSection.ContentLink.Expanded == nil {
						continue
					}

					tableBlock := nestedSection.ContentLink.Expanded
					if !isTableBlock(tableBlock.ContentType) {
						continue
					}

					rates, err := c.parseSkandiaAvgRateTable(tableBlock.Columns, avgMonth, crawlTime)
					if err != nil {
						c.logger.Warn("failed parsing Skandia FAQ avg rate table", zap.Error(err))
						continue
					}

					interestSets = append(interestSets, rates...)
				}
			}
		}
	}

	return interestSets, nil
}

//nolint:cyclop // switch-case for multiple column types
func (c *SkandiaCrawler) parseSkandiaListRateTable(columns []skandiaColumn, crawlTime time.Time) ([]model.InterestSet, error) {
	if len(columns) < 2 {
		return nil, fmt.Errorf("expected at least 2 columns, got %d", len(columns))
	}

	// Find the term column and rate column
	var termCells, rateCells, dateCells []string

	for _, col := range columns {
		if col.ContentLink.Expanded == nil {
			continue
		}

		header := strings.ToLower(col.ContentLink.Expanded.CellHeader)
		header = strings.TrimSpace(header)

		switch {
		case strings.Contains(header, "bindningstid"):
			termCells = col.ContentLink.Expanded.Cells
		case strings.Contains(header, "listränta"):
			rateCells = col.ContentLink.Expanded.Cells
		case strings.Contains(header, "ändrad"):
			dateCells = col.ContentLink.Expanded.Cells
		}
	}

	if len(termCells) == 0 || len(rateCells) == 0 {
		return nil, fmt.Errorf("could not find term or rate columns")
	}

	interestSets := []model.InterestSet{}

	for i := 0; i < len(termCells) && i < len(rateCells); i++ {
		term, err := parseSkandiaHTMLTerm(termCells[i])
		if err != nil {
			c.logger.Warn("failed parsing Skandia term", zap.String("cell", termCells[i]), zap.Error(err))
			continue
		}

		rate, err := parseSkandiaHTMLRate(rateCells[i])
		if err != nil {
			c.logger.Warn("failed parsing Skandia rate", zap.String("cell", rateCells[i]), zap.Error(err))
			continue
		}

		var changedOn *time.Time
		if i < len(dateCells) {
			if parsed, err := parseSkandiaHTMLDate(dateCells[i]); err == nil {
				changedOn = &parsed
			}
		}

		interestSets = append(interestSets, model.InterestSet{
			Bank:          skandiaBankName,
			Type:          model.TypeListRate,
			Term:          term,
			NominalRate:   rate,
			ChangedOn:     changedOn,
			LastCrawledAt: crawlTime,
		})
	}

	return interestSets, nil
}

//nolint:cyclop // switch-case for multiple column types
func (c *SkandiaCrawler) parseSkandiaAvgRateTable(columns []skandiaColumn, avgMonth model.AvgMonth, crawlTime time.Time) ([]model.InterestSet, error) {
	if len(columns) < 2 {
		return nil, fmt.Errorf("expected at least 2 columns, got %d", len(columns))
	}

	// Find the term column and rate column
	var termCells, rateCells []string

	for _, col := range columns {
		if col.ContentLink.Expanded == nil {
			continue
		}

		header := strings.ToLower(col.ContentLink.Expanded.CellHeader)
		header = strings.TrimSpace(header)

		switch {
		case strings.Contains(header, "bindningstid"):
			termCells = col.ContentLink.Expanded.Cells
		case strings.Contains(header, "snittränta"):
			rateCells = col.ContentLink.Expanded.Cells
		}
	}

	if len(termCells) == 0 || len(rateCells) == 0 {
		return nil, fmt.Errorf("could not find term or rate columns")
	}

	interestSets := []model.InterestSet{}

	for i := 0; i < len(termCells) && i < len(rateCells); i++ {
		term, err := parseSkandiaHTMLTerm(termCells[i])
		if err != nil {
			c.logger.Warn("failed parsing Skandia term", zap.String("cell", termCells[i]), zap.Error(err))
			continue
		}

		rate, err := parseSkandiaHTMLRate(rateCells[i])
		if err != nil {
			c.logger.Warn("failed parsing Skandia rate", zap.String("cell", rateCells[i]), zap.Error(err))
			continue
		}

		interestSets = append(interestSets, model.InterestSet{
			Bank:                  skandiaBankName,
			Type:                  model.TypeAverageRate,
			Term:                  term,
			NominalRate:           rate,
			AverageReferenceMonth: &avgMonth,
			LastCrawledAt:         crawlTime,
		})
	}

	return interestSets, nil
}

func extractSkandiaPageContent(html string) (*skandiaPageContent, error) {
	matches := skandiaPageContentRgx.FindStringSubmatch(html)
	if len(matches) < 2 {
		return nil, fmt.Errorf("could not find SKB.pageContent in HTML")
	}

	jsonStr := matches[1]

	var pageContent skandiaPageContent
	if err := json.Unmarshal([]byte(jsonStr), &pageContent); err != nil {
		return nil, fmt.Errorf("failed unmarshalling SKB.pageContent: %w", err)
	}

	return &pageContent, nil
}

//nolint:cyclop // switch-case for multiple term types
func parseSkandiaHTMLTerm(htmlCell string) (model.Term, error) {
	// Decode HTML entities (e.g., &aring; -> å)
	decoded := html.UnescapeString(htmlCell)
	matches := skandiaTermRgx.FindStringSubmatch(decoded)
	if len(matches) < 3 {
		return "", fmt.Errorf("could not parse term from %q", htmlCell)
	}

	num, err := strconv.Atoi(matches[1])
	if err != nil {
		return "", fmt.Errorf("invalid term number: %w", err)
	}

	unit := strings.ToLower(matches[2])

	switch unit {
	case "mån":
		if num == 3 {
			return model.Term3months, nil
		}
		return "", fmt.Errorf("unsupported month term: %d", num)
	case "år":
		switch num {
		case 1:
			return model.Term1year, nil
		case 2:
			return model.Term2years, nil
		case 3:
			return model.Term3years, nil
		case 4:
			return model.Term4years, nil
		case 5:
			return model.Term5years, nil
		case 6:
			return model.Term6years, nil
		case 7:
			return model.Term7years, nil
		case 8:
			return model.Term8years, nil
		case 9:
			return model.Term9years, nil
		case 10:
			return model.Term10years, nil
		default:
			return "", fmt.Errorf("unsupported year term: %d", num)
		}
	}

	return "", fmt.Errorf("unknown term unit: %s", unit)
}

func parseSkandiaHTMLRate(htmlCell string) (float32, error) {
	matches := skandiaRateRgx.FindStringSubmatch(htmlCell)
	if len(matches) < 2 {
		return 0, fmt.Errorf("could not parse rate from %q", htmlCell)
	}

	// Replace Swedish decimal comma with dot
	rateStr := strings.ReplaceAll(matches[1], ",", ".")

	rate, err := strconv.ParseFloat(rateStr, 32)
	if err != nil {
		return 0, fmt.Errorf("failed parsing rate %q: %w", rateStr, err)
	}

	return float32(rate), nil
}

func parseSkandiaHTMLDate(htmlCell string) (time.Time, error) {
	// Extract date in format YYYY-MM-DD from HTML like "<p>2025-09-30</p>"
	dateRgx := regexp.MustCompile(`(\d{4}-\d{2}-\d{2})`)
	matches := dateRgx.FindStringSubmatch(htmlCell)
	if len(matches) < 2 {
		return time.Time{}, fmt.Errorf("could not parse date from %q", htmlCell)
	}

	parsed, err := time.Parse("2006-01-02", matches[1])
	if err != nil {
		return time.Time{}, fmt.Errorf("failed parsing date %q: %w", matches[1], err)
	}

	return parsed, nil
}

func parseSkandiaMonthYear(text string) (model.AvgMonth, error) {
	matches := skandiaMonthYearRgx.FindStringSubmatch(text)
	if len(matches) < 3 {
		return model.AvgMonth{}, fmt.Errorf("could not parse month/year from %q", text)
	}

	monthName := strings.ToLower(matches[1])
	monthNum, ok := skandiaMonthMap[monthName]
	if !ok {
		return model.AvgMonth{}, fmt.Errorf("unknown month: %s", monthName)
	}

	year, err := strconv.Atoi(matches[2])
	if err != nil {
		return model.AvgMonth{}, fmt.Errorf("invalid year: %w", err)
	}

	return model.AvgMonth{
		Year:  uint(year),
		Month: time.Month(monthNum),
	}, nil
}

func isTableBlock(contentType []string) bool {
	for _, ct := range contentType {
		if ct == "TableBlock" {
			return true
		}
	}
	return false
}

func isAccordionBlock(contentType []string) bool {
	for _, ct := range contentType {
		if ct == "AccordionBlock" {
			return true
		}
	}
	return false
}

func isFAQBlock(contentType []string) bool {
	for _, ct := range contentType {
		if ct == "FAQBlock" {
			return true
		}
	}
	return false
}
