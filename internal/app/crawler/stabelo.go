package crawler

import (
	"bytes"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ledongthuc/pdf"
	"github.com/yama6a/bolan-compare/internal/pkg/http"
	"github.com/yama6a/bolan-compare/internal/pkg/model"
	"github.com/yama6a/bolan-compare/internal/pkg/utils"
	"go.uber.org/zap"
	"golang.org/x/net/html"
)

const (
	stabeloBankName     model.Bank = "Stabelo"
	stabeloRateTableURL string     = "https://api.stabelo.se/rate-table/"
	stabeloAvgRatesURL  string     = "https://www.stabelo.se/bolanerantor"
)

var _ SiteCrawler = &StabeloCrawler{}

// StabeloCrawler crawls Stabelo mortgage rates from their rate table and PDF documents.
// Stabelo is a digital mortgage bank using a Remix.js framework with turbo-stream data.
//
// Data sources:
//   - List rates: Extracted from turbo-stream data (worst-case rates, no LTV discount)
//   - LTV discounted rates: Extracted from turbo-stream data (rates per LTV tier)
//   - Average rates: Extracted from PDF document linked on bolanerantor page
type StabeloCrawler struct {
	httpClient http.Client
	logger     *zap.Logger
}

// NewStabeloCrawler creates a new StabeloCrawler instance.
func NewStabeloCrawler(httpClient http.Client, logger *zap.Logger) *StabeloCrawler {
	return &StabeloCrawler{httpClient: httpClient, logger: logger}
}

// Crawl fetches and parses Stabelo mortgage rates from all sources.
func (c *StabeloCrawler) Crawl(channel chan<- model.InterestSet) {
	crawlTime := time.Now().UTC()
	var allRates []model.InterestSet

	// Fetch list rates and LTV-discounted rates
	rates, err := c.fetchRates(crawlTime)
	if err != nil {
		c.logger.Error("failed fetching Stabelo rates", zap.Error(err))
	} else {
		allRates = append(allRates, rates...)
		for _, set := range rates {
			channel <- set
		}
	}

	// Fetch average rates from PDF
	avgRates, err := c.fetchAverageRates(crawlTime)
	if err != nil {
		c.logger.Warn("failed fetching Stabelo average rates", zap.Error(err))
	} else {
		allRates = append(allRates, avgRates...)
		for _, set := range avgRates {
			channel <- set
		}
	}
}

// fetchRates fetches the rate table and extracts list rates and LTV-discounted rates.
func (c *StabeloCrawler) fetchRates(crawlTime time.Time) ([]model.InterestSet, error) {
	rawHTML, err := c.httpClient.Fetch(stabeloRateTableURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed reading Stabelo rate table: %w", err)
	}

	return c.extractRates(rawHTML, crawlTime)
}

// extractRates parses the HTML and extracts both list rates and LTV-discounted rates.
// The Remix.js turbo-stream contains all rate combinations encoded in the page.
func (c *StabeloCrawler) extractRates(rawHTML string, crawlTime time.Time) ([]model.InterestSet, error) {
	var results []model.InterestSet

	// Try to extract list rates from turbo-stream data
	listRates, err := c.extractListRatesFromTurboStream(rawHTML, crawlTime)
	if err != nil {
		c.logger.Debug("could not extract list rates from turbo-stream", zap.Error(err))
	} else {
		results = append(results, listRates...)
	}

	// Extract LTV-discounted rates from HTML buttons (default 50% LTV config)
	ltvRates, err := c.extractLTVRatesFromHTML(rawHTML, crawlTime)
	if err != nil {
		c.logger.Warn("failed to extract LTV rates from HTML", zap.Error(err))
	} else {
		results = append(results, ltvRates...)
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no rates extracted from Stabelo page")
	}

	return results, nil
}

// extractListRatesFromTurboStream attempts to extract true list rates from the turbo-stream data.
// List rates are the worst-case rates (no LTV discount, no green loan discount, no amount discount).
// These are identified in the turbo-stream by entries with no LTV, no EPC, and product_amount=0.
func (c *StabeloCrawler) extractListRatesFromTurboStream(rawHTML string, crawlTime time.Time) ([]model.InterestSet, error) {
	// The turbo-stream format uses a flat array with reference-based encoding.
	// We extract rates by finding patterns in the raw data.
	//
	// From analysis:
	// - List rate entries have no "ltv" field
	// - Rate fixations: "3M", "1Y", "2Y", "3Y", "5Y", "10Y"
	// - Rates are in basis points (333 = 3.33%)

	// Extract all rate basis point values near rate fixation strings
	results := []model.InterestSet{}

	// Pattern to find rate fixation followed by associated bps value in the turbo-stream
	// The structure is: "rate_fixation" ... "3M" ... bps value nearby
	// We look for patterns like: ,"3M", ... followed by numbers that could be rates

	termRates := c.findListRatesInStream(rawHTML)
	if len(termRates) == 0 {
		return nil, fmt.Errorf("could not find list rates in turbo-stream")
	}

	for term, rateBps := range termRates {
		modelTerm, err := parseStabeloTerm(term)
		if err != nil {
			continue
		}

		rate := float32(rateBps) / 100.0

		results = append(results, model.InterestSet{
			Bank:          stabeloBankName,
			Type:          model.TypeListRate,
			Term:          modelTerm,
			NominalRate:   rate,
			LastCrawledAt: crawlTime,
		})
	}

	return results, nil
}

// findListRatesInStream searches the turbo-stream data for list rate values.
// Based on analysis, list rates (no LTV) are typically the highest rates for each term.
func (c *StabeloCrawler) findListRatesInStream(rawHTML string) map[string]int {
	result := make(map[string]int)
	terms := []string{"3M", "1Y", "2Y", "3Y", "5Y", "10Y"}

	for _, term := range terms {
		maxRate := c.findMaxRateForTerm(rawHTML, term)
		if maxRate > 0 {
			result[term] = maxRate
		}
	}

	return result
}

// findMaxRateForTerm finds the highest rate value near a term string in the turbo-stream.
func (c *StabeloCrawler) findMaxRateForTerm(rawHTML, term string) int {
	termPattern := regexp.MustCompile(fmt.Sprintf(`"%s"[^"]*?(\d{3})`, term))
	termMatches := termPattern.FindAllStringSubmatch(rawHTML, -1)

	maxRate := 0
	for _, match := range termMatches {
		val, err := strconv.Atoi(match[1])
		if err != nil {
			continue
		}
		if val >= 200 && val <= 600 && val > maxRate {
			maxRate = val
		}
	}

	return maxRate
}

// extractLTVRatesFromHTML extracts rates from the HTML buttons.
// These buttons display rates for the default configuration: 2M SEK loan on 4M property (50% LTV).
// Since 50% LTV is below Stabelo's minimum tier (60%), these represent the best available rates.
func (c *StabeloCrawler) extractLTVRatesFromHTML(rawHTML string, crawlTime time.Time) ([]model.InterestSet, error) {
	doc, err := html.Parse(strings.NewReader(rawHTML))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	rateButtons := findStabeloRateButtons(doc)
	if len(rateButtons) == 0 {
		return nil, fmt.Errorf("no rate buttons found in HTML")
	}

	results := make([]model.InterestSet, 0, len(rateButtons))

	for _, btn := range rateButtons {
		termStr := getAttr(btn, "value")
		if termStr == "" {
			continue
		}

		term, err := parseStabeloTerm(termStr)
		if err != nil {
			c.logger.Debug("skipping button with unsupported term",
				zap.String("value", termStr),
				zap.Error(err))
			continue
		}

		rate, err := extractRateFromButton(btn)
		if err != nil {
			c.logger.Warn("failed to extract rate from button",
				zap.String("value", termStr),
				zap.Error(err))
			continue
		}

		// These are LTV-discounted rates (≤60% LTV tier)
		// The HTML shows rates for 50% LTV which falls into the ≤60% tier
		results = append(results, model.InterestSet{
			Bank:        stabeloBankName,
			Type:        model.TypeRatioDiscounted,
			Term:        term,
			NominalRate: rate,
			RatioDiscountBoundaries: &model.RatioDiscountBoundary{
				MinRatio: 0,
				MaxRatio: 0.60, // ≤60% LTV tier
			},
			LastCrawledAt: crawlTime,
		})
	}

	return results, nil
}

// fetchAverageRates fetches and parses the average rates PDF from Stabelo's website.
func (c *StabeloCrawler) fetchAverageRates(crawlTime time.Time) ([]model.InterestSet, error) {
	// First, fetch the bolanerantor page to find the PDF link
	pageHTML, err := c.httpClient.Fetch(stabeloAvgRatesURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch bolanerantor page: %w", err)
	}

	// Find the PDF link
	pdfURL := c.findPDFLink(pageHTML)
	if pdfURL == "" {
		return nil, fmt.Errorf("could not find average rates PDF link")
	}

	c.logger.Debug("found average rates PDF", zap.String("url", pdfURL))

	// Fetch the PDF
	pdfContent, err := c.httpClient.FetchRaw(pdfURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch PDF: %w", err)
	}

	// Parse the PDF
	return c.parsePDF(pdfContent, crawlTime)
}

// findPDFLink searches the HTML for a link to the average rates PDF.
// Robust matching: looks for any PDF link containing average rate keywords.
func (c *StabeloCrawler) findPDFLink(pageHTML string) string {
	// Find all PDF links in the page
	pdfPattern := regexp.MustCompile(`href="([^"]*\.pdf)"`)
	allMatches := pdfPattern.FindAllStringSubmatch(pageHTML, -1)

	for _, matches := range allMatches {
		if len(matches) < 2 {
			continue
		}
		pdfURL := matches[1]
		lowerURL := strings.ToLower(pdfURL)

		// Look for average rates PDF - matches various Swedish spellings
		// Genomsnittsräntor, genomsnittsrantor, snitträntor, etc.
		if strings.Contains(lowerURL, "genomsnitt") ||
			strings.Contains(lowerURL, "snitt") ||
			strings.Contains(lowerURL, "stabelo") {
			return c.normalizePDFURL(pdfURL)
		}
	}

	// Fallback: return the first PDF link found
	if len(allMatches) > 0 && len(allMatches[0]) > 1 {
		c.logger.Warn("using fallback PDF link", zap.String("url", allMatches[0][1]))
		return c.normalizePDFURL(allMatches[0][1])
	}

	return ""
}

// normalizePDFURL converts relative PDF URLs to absolute URLs with proper encoding.
func (c *StabeloCrawler) normalizePDFURL(pdfURL string) string {
	var baseURL string

	if strings.HasPrefix(pdfURL, "http") {
		// Full URL - parse and re-encode
		parsed, err := url.Parse(pdfURL)
		if err != nil {
			return pdfURL
		}
		return parsed.String()
	}

	// Relative URL - build absolute URL
	if strings.HasPrefix(pdfURL, "/") {
		baseURL = "https://www.stabelo.se"
	} else {
		baseURL = "https://www.stabelo.se/"
	}

	// Parse the full URL to properly encode the path
	fullURL := baseURL + pdfURL
	parsed, err := url.Parse(fullURL)
	if err != nil {
		return fullURL
	}
	return parsed.String()
}

// parsePDF extracts average rates from the Stabelo PDF document.
// The PDF contains a table with historical average rates for different binding periods.
func (c *StabeloCrawler) parsePDF(pdfContent []byte, crawlTime time.Time) ([]model.InterestSet, error) {
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

	return c.parseAverageRatesText(allText.String(), crawlTime)
}

// parseAverageRatesText extracts average rates from the PDF text content.
// The PDF table format has columns for different binding periods parsed from the header.
func (c *StabeloCrawler) parseAverageRatesText(text string, crawlTime time.Time) ([]model.InterestSet, error) {
	// Extract terms from the PDF header
	terms := extractTermsFromHeader(text)
	if len(terms) == 0 {
		return nil, fmt.Errorf("could not find terms in PDF header")
	}

	c.logger.Debug("extracted terms from PDF header", zap.Int("count", len(terms)))

	// Find ALL month/year and their associated rate data
	// PDF format: "oktober 2025 2,61% 2,52%...september 2025 2,90%..."
	allRateData := findAllRateData(text)
	if len(allRateData) == 0 {
		return nil, fmt.Errorf("could not find any rate data in PDF")
	}

	c.logger.Debug("extracted rate data", zap.Int("months", len(allRateData)))

	var results []model.InterestSet
	for _, rd := range allRateData {
		monthResults := c.buildAverageRateResults(rd.rateLine, rd.month, rd.year, crawlTime, terms)
		results = append(results, monthResults...)
	}

	return results, nil
}

// extractTermsFromHeader parses the binding period terms from the PDF header.
// Header format: "Bindningstid3 mån1 år2 år3 år5 år10 år".
func extractTermsFromHeader(text string) []model.Term {
	// Pattern to find the header with terms
	// Matches: "Bindningstid" followed by term patterns like "3 mån", "1 år", etc.
	headerPattern := regexp.MustCompile(`[Bb]indningstid((?:\d+\s*(?:mån|år))+)`)
	match := headerPattern.FindStringSubmatch(text)
	if match == nil {
		return nil
	}

	termsStr := match[1]

	// Extract individual terms: "3 mån", "1 år", "2 år", etc.
	termPattern := regexp.MustCompile(`(\d+\s*(?:mån|år))`)
	termMatches := termPattern.FindAllString(termsStr, -1)

	terms := make([]model.Term, 0, len(termMatches))
	for _, termStr := range termMatches {
		term, err := utils.ParseTerm(termStr)
		if err != nil {
			continue
		}
		terms = append(terms, term)
	}

	return terms
}

// rateData holds parsed rate information for a single month.
type rateData struct {
	month    time.Month
	year     int
	rateLine string
}

// findAllRateData extracts all month/year combinations with their rate data from the PDF text.
// The PDF format has all data on one line: "oktober 2025 2,61% 2,52%...september 2025 2,90%...".
func findAllRateData(text string) []rateData {
	// Swedish month names for regex
	monthNames := `januari|februari|mars|april|maj|juni|juli|augusti|september|oktober|november|december`

	// Pattern to find month+year followed by rates
	// Captures: (month)(year)(rates until next month or end)
	pattern := regexp.MustCompile(`(?i)(` + monthNames + `)\s*(20\d{2})\s*([^a-z]+?)(?:(?:` + monthNames + `)|$)`)

	var results []rateData
	remaining := text

	for {
		match := pattern.FindStringSubmatchIndex(remaining)
		if match == nil {
			break
		}

		monthStr := strings.ToLower(remaining[match[2]:match[3]])
		yearStr := remaining[match[4]:match[5]]
		ratePart := strings.TrimSpace(remaining[match[6]:match[7]])

		month := parseSwedishMonth(monthStr)
		year, _ := strconv.Atoi(yearStr)

		// Only add if we have valid rate data (contains at least one percentage)
		if strings.Contains(ratePart, "%") {
			results = append(results, rateData{
				month:    month,
				year:     year,
				rateLine: ratePart,
			})
		}

		// Move past this match to find the next one
		// Start from where the rate part ends (before next month name)
		remaining = remaining[match[6]:]
	}

	return results
}

// parseSwedishMonth converts a Swedish month name to time.Month.
func parseSwedishMonth(name string) time.Month {
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
	return monthMap[name]
}

// buildAverageRateResults creates InterestSet results from parsed rate data.
func (c *StabeloCrawler) buildAverageRateResults(
	dataLine string, month time.Month, year int, crawlTime time.Time, terms []model.Term,
) []model.InterestSet {
	ratePattern := regexp.MustCompile(`(\d+[,\.]\d+)\s*%?`)
	rates := ratePattern.FindAllString(dataLine, -1)

	results := make([]model.InterestSet, 0, len(terms))
	for i, rateStr := range rates {
		if i >= len(terms) {
			break
		}

		rate, err := parseStabeloRate(rateStr)
		if err != nil {
			c.logger.Warn("failed to parse rate from PDF", zap.String("rate", rateStr), zap.Error(err))
			continue
		}

		results = append(results, model.InterestSet{
			Bank:                  stabeloBankName,
			Type:                  model.TypeAverageRate,
			Term:                  terms[i],
			NominalRate:           rate,
			AverageReferenceMonth: &model.AvgMonth{Month: month, Year: uint(year)},
			LastCrawledAt:         crawlTime,
		})
	}

	return results
}

// findStabeloRateButtons finds all rate button elements in the HTML.
// Rate buttons are <button> elements with a value attribute matching rate fixation patterns.
func findStabeloRateButtons(n *html.Node) []*html.Node {
	var buttons []*html.Node

	var traverse func(*html.Node)
	traverse = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "button" {
			value := getAttr(n, "value")
			// Check if this looks like a rate fixation value (3M, 1Y, 2Y, etc.)
			if isRateFixationValue(value) {
				buttons = append(buttons, n)
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			traverse(c)
		}
	}
	traverse(n)

	return buttons
}

// isRateFixationValue checks if a string looks like a Stabelo rate fixation value.
func isRateFixationValue(s string) bool {
	switch s {
	case "3M", "1Y", "2Y", "3Y", "5Y", "10Y":
		return true
	default:
		return false
	}
}

// extractRateFromButton extracts the interest rate percentage from a button element.
// The rate is in a span element with format "X,XX %".
func extractRateFromButton(btn *html.Node) (float32, error) {
	var spans []string

	var collectSpans func(*html.Node)
	collectSpans = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "span" {
			text := getTextContent(n)
			if text != "" {
				spans = append(spans, text)
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			collectSpans(c)
		}
	}
	collectSpans(btn)

	// Find the span containing the rate (ends with " %")
	for _, text := range spans {
		if strings.HasSuffix(text, " %") || strings.HasSuffix(text, "%") {
			return parseStabeloRate(text)
		}
	}

	return 0, fmt.Errorf("no rate found in button spans: %v", spans)
}

// getTextContent returns the text content of a node and its children.
func getTextContent(n *html.Node) string {
	var sb strings.Builder

	var traverse func(*html.Node)
	traverse = func(n *html.Node) {
		if n.Type == html.TextNode {
			sb.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			traverse(c)
		}
	}
	traverse(n)

	return strings.TrimSpace(sb.String())
}

// getAttr returns the value of an attribute on a node.
func getAttr(n *html.Node, key string) string {
	for _, attr := range n.Attr {
		if attr.Key == key {
			return attr.Val
		}
	}
	return ""
}

// parseStabeloRate parses a Swedish-format rate string like "2,54 %" to float32.
func parseStabeloRate(s string) (float32, error) {
	// Trim surrounding whitespace first
	s = strings.TrimSpace(s)

	// Remove "%" suffix (with or without space before it)
	s = strings.TrimSuffix(s, "%")
	s = strings.TrimSpace(s)

	// Replace Swedish decimal comma with period
	s = strings.Replace(s, ",", ".", 1)

	rate, err := strconv.ParseFloat(s, 32)
	if err != nil {
		return 0, fmt.Errorf("failed to parse rate %q: %w", s, err)
	}

	return float32(rate), nil
}

// parseStabeloTerm converts Stabelo's rate fixation format to model.Term.
// Rate fixation format: "3M", "1Y", "2Y", "3Y", "5Y", "10Y".
func parseStabeloTerm(rateFixation string) (model.Term, error) {
	switch rateFixation {
	case "3M":
		return model.Term3months, nil
	case "1Y":
		return model.Term1year, nil
	case "2Y":
		return model.Term2years, nil
	case "3Y":
		return model.Term3years, nil
	case "5Y":
		return model.Term5years, nil
	case "10Y":
		return model.Term10years, nil
	default:
		return "", fmt.Errorf("unsupported Stabelo rate fixation: %s", rateFixation)
	}
}
