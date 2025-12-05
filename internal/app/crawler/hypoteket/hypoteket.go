package hypoteket

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"time"

	"github.com/yama6a/bolan-compare/internal/app/crawler"
	"github.com/yama6a/bolan-compare/internal/pkg/http"
	"github.com/yama6a/bolan-compare/internal/pkg/model"
	"go.uber.org/zap"
)

const (
	hypoteketBankName model.Bank = "Hypoteket"
	hypoteketRatesURL string     = "https://hypoteket.com/borantor/_payload.json"
)

var (
	_                       crawler.SiteCrawler = &HypoteketCrawler{}
	hypoteketPeriodYYYYMMRx                     = regexp.MustCompile(`^(\d{4})-(\d{2})$`) // YYYY-MM
)

// HypoteketCrawler crawls Hypoteket's Nuxt.js payload for mortgage rates.
//
//nolint:revive // Bank name prefix is intentional for clarity
type HypoteketCrawler struct {
	httpClient http.Client
	logger     *zap.Logger
}

func NewHypoteketCrawler(httpClient http.Client, logger *zap.Logger) *HypoteketCrawler {
	return &HypoteketCrawler{httpClient: httpClient, logger: logger}
}

func (c *HypoteketCrawler) Crawl(channel chan<- model.InterestSet) {
	crawlTime := time.Now().UTC()

	rawJSON, err := c.httpClient.Fetch(hypoteketRatesURL, nil)
	if err != nil {
		c.logger.Error("failed fetching Hypoteket rates payload", zap.Error(err))
		return
	}

	listRates, err := c.parseListRates(rawJSON, crawlTime)
	if err != nil {
		c.logger.Error("failed parsing Hypoteket list rates", zap.Error(err))
	}

	avgRates, err := c.parseAverageRates(rawJSON, crawlTime)
	if err != nil {
		c.logger.Error("failed parsing Hypoteket average rates", zap.Error(err))
	}

	for _, set := range append(listRates, avgRates...) {
		channel <- set
	}
}

// parseListRates extracts list rates from the Nuxt.js payload.
// The payload structure uses array indices as references.
func (c *HypoteketCrawler) parseListRates(rawJSON string, crawlTime time.Time) ([]model.InterestSet, error) {
	var payload []any
	if err := json.Unmarshal([]byte(rawJSON), &payload); err != nil {
		return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	rateIndices, err := findListRateIndices(payload)
	if err != nil {
		return nil, err
	}

	interestSets := []model.InterestSet{}
	for _, rateIdxRaw := range rateIndices {
		set, ok := c.extractListRateEntry(payload, rateIdxRaw, crawlTime)
		if ok {
			interestSets = append(interestSets, set)
		}
	}

	return interestSets, nil
}

// findListRateIndices locates the list rate entry indices in the payload.
func findListRateIndices(payload []any) ([]any, error) {
	if len(payload) < 3 {
		return nil, fmt.Errorf("payload too short, expected at least 3 elements")
	}

	dataMap, ok := payload[2].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("expected object at index 2, got %T", payload[2])
	}

	interestRatesIdxRaw, ok := dataMap["interest-rates"]
	if !ok {
		return nil, fmt.Errorf("interest-rates key not found in data object")
	}

	interestRatesIdx, ok := interestRatesIdxRaw.(float64)
	if !ok {
		return nil, fmt.Errorf("interest-rates value is not a number: %T", interestRatesIdxRaw)
	}

	// The value at interest-rates index is ["Reactive", arrayIndex]
	reactiveWrapper, ok := payload[int(interestRatesIdx)].([]any)
	if !ok || len(reactiveWrapper) != 2 || reactiveWrapper[0] != "Reactive" {
		return nil, fmt.Errorf("expected Reactive wrapper at index %d", int(interestRatesIdx))
	}

	ratesArrayIdx, ok := reactiveWrapper[1].(float64)
	if !ok {
		return nil, fmt.Errorf("expected array index in Reactive wrapper")
	}

	// Get the array of rate entry indices
	rateIndices, ok := payload[int(ratesArrayIdx)].([]any)
	if !ok {
		return nil, fmt.Errorf("expected array of rate indices at index %d", int(ratesArrayIdx))
	}

	return rateIndices, nil
}

// extractListRateEntry extracts a single list rate entry from the payload.
func (c *HypoteketCrawler) extractListRateEntry(payload []any, rateIdxRaw any, crawlTime time.Time) (model.InterestSet, bool) {
	rateIdx, ok := rateIdxRaw.(float64)
	if !ok {
		return model.InterestSet{}, false
	}

	rateEntry, ok := payload[int(rateIdx)].(map[string]any)
	if !ok {
		return model.InterestSet{}, false
	}

	term, rate, ok := extractTermAndRate(payload, rateEntry)
	if !ok {
		return model.InterestSet{}, false
	}

	termModel, err := parseHypoteketTermToTerm(term)
	if err != nil {
		c.logger.Warn("Hypoteket list rate term not supported - skipping",
			zap.String("term", term),
			zap.Error(err))
		return model.InterestSet{}, false
	}

	changedOn := extractValidFromDate(payload, rateEntry)

	return model.InterestSet{
		Bank:          hypoteketBankName,
		Type:          model.TypeListRate,
		Term:          termModel,
		NominalRate:   float32(rate),
		ChangedOn:     changedOn,
		LastCrawledAt: crawlTime,

		RatioDiscountBoundaries: nil,
		UnionDiscount:           false,
		AverageReferenceMonth:   nil,
	}, true
}

// extractTermAndRate extracts the term string and rate value from a rate entry.
func extractTermAndRate(payload []any, rateEntry map[string]any) (string, float64, bool) {
	termIdx, ok := rateEntry["interestTerm"].(float64)
	if !ok {
		return "", 0, false
	}
	termStr, ok := payload[int(termIdx)].(string)
	if !ok {
		return "", 0, false
	}

	rateValIdx, ok := rateEntry["rate"].(float64)
	if !ok {
		return "", 0, false
	}
	rate, ok := payload[int(rateValIdx)].(float64)
	if !ok {
		return "", 0, false
	}

	return termStr, rate, true
}

// extractValidFromDate extracts the validFrom date from a rate entry.
func extractValidFromDate(payload []any, rateEntry map[string]any) *time.Time {
	validFromIdx, ok := rateEntry["validFrom"].(float64)
	if !ok {
		return nil
	}

	validFromStr, ok := payload[int(validFromIdx)].(string)
	if !ok {
		return nil
	}

	// Format: 2025-11-10T00:00:00.000Z
	if t, err := time.Parse(time.RFC3339, validFromStr); err == nil {
		return &t
	}
	if t, err := time.Parse("2006-01-02T15:04:05.000Z", validFromStr); err == nil {
		return &t
	}

	return nil
}

// parseAverageRates extracts historical average rates from the Nuxt.js payload.
func (c *HypoteketCrawler) parseAverageRates(rawJSON string, crawlTime time.Time) ([]model.InterestSet, error) {
	var payload []any
	if err := json.Unmarshal([]byte(rawJSON), &payload); err != nil {
		return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	interestSets := []model.InterestSet{}

	for i, item := range payload {
		entryMap, ok := item.(map[string]any)
		if !ok {
			continue
		}

		avgMonth, ok := c.extractAvgMonthFromEntry(payload, entryMap, i)
		if !ok {
			continue
		}

		sets := extractAvgRatesFromEntry(payload, entryMap, avgMonth, crawlTime)
		interestSets = append(interestSets, sets...)
	}

	return interestSets, nil
}

// extractAvgMonthFromEntry extracts the average month from a payload entry.
func (c *HypoteketCrawler) extractAvgMonthFromEntry(payload []any, entryMap map[string]any, index int) (model.AvgMonth, bool) {
	monthPeriodIdxRaw, ok := entryMap["monthPeriod"]
	if !ok {
		return model.AvgMonth{}, false
	}

	monthPeriodIdx, ok := monthPeriodIdxRaw.(float64)
	if !ok {
		return model.AvgMonth{}, false
	}

	monthPeriodStr, ok := payload[int(monthPeriodIdx)].(string)
	if !ok {
		return model.AvgMonth{}, false
	}

	avgMonth, err := parseHypoteketPeriod(monthPeriodStr)
	if err != nil {
		c.logger.Warn("failed parsing Hypoteket average rate period",
			zap.String("period", monthPeriodStr),
			zap.Int("index", index),
			zap.Error(err))
		return model.AvgMonth{}, false
	}

	return avgMonth, true
}

// extractAvgRatesFromEntry extracts all term rates from an average rate entry.
func extractAvgRatesFromEntry(payload []any, entryMap map[string]any, avgMonth model.AvgMonth, crawlTime time.Time) []model.InterestSet {
	termFields := []struct {
		field string
		term  model.Term
	}{
		{"threeMonth", model.Term3months},
		{"oneYear", model.Term1year},
		{"twoYear", model.Term2years},
		{"threeYear", model.Term3years},
		{"fiveYear", model.Term5years},
	}

	sets := make([]model.InterestSet, 0, len(termFields))
	for _, tf := range termFields {
		rate, ok := extractRateValue(payload, entryMap, tf.field)
		if !ok {
			continue
		}

		avgMonthCopy := avgMonth
		sets = append(sets, model.InterestSet{
			AverageReferenceMonth: &avgMonthCopy,
			Bank:                  hypoteketBankName,
			Type:                  model.TypeAverageRate,
			Term:                  tf.term,
			NominalRate:           float32(rate),
			LastCrawledAt:         crawlTime,

			RatioDiscountBoundaries: nil,
			UnionDiscount:           false,
			ChangedOn:               nil,
		})
	}

	return sets
}

// extractRateValue extracts a rate value from a payload entry.
func extractRateValue(payload []any, entryMap map[string]any, field string) (float64, bool) {
	rateIdxRaw, ok := entryMap[field]
	if !ok {
		return 0, false
	}

	rateIdx, ok := rateIdxRaw.(float64)
	if !ok {
		return 0, false
	}

	rateVal := payload[int(rateIdx)]
	switch v := rateVal.(type) {
	case float64:
		return v, true
	case string:
		if v == "-" || v == "" {
			return 0, false // Skip missing values
		}
		// Try parsing as number
		parsed, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return 0, false
		}
		return parsed, true
	default:
		return 0, false
	}
}

// parseHypoteketTermToTerm converts Hypoteket's term names to model.Term.
func parseHypoteketTermToTerm(term string) (model.Term, error) {
	switch term {
	case "threeMonth":
		return model.Term3months, nil
	case "oneYear":
		return model.Term1year, nil
	case "twoYear":
		return model.Term2years, nil
	case "threeYear":
		return model.Term3years, nil
	case "fiveYear":
		return model.Term5years, nil
	default:
		return "", fmt.Errorf("unsupported Hypoteket term: %s", term)
	}
}

// parseHypoteketPeriod converts a YYYY-MM period string to model.AvgMonth.
func parseHypoteketPeriod(period string) (model.AvgMonth, error) {
	matches := hypoteketPeriodYYYYMMRx.FindStringSubmatch(period)
	if len(matches) != 3 {
		return model.AvgMonth{}, fmt.Errorf("failed to match period format YYYY-MM: %s", period)
	}

	year, err := strconv.Atoi(matches[1])
	if err != nil {
		return model.AvgMonth{}, fmt.Errorf("failed to parse year: %w", err)
	}

	month, err := strconv.Atoi(matches[2])
	if err != nil {
		return model.AvgMonth{}, fmt.Errorf("failed to parse month: %w", err)
	}

	if month < 1 || month > 12 {
		return model.AvgMonth{}, fmt.Errorf("invalid month: %d", month)
	}

	return model.AvgMonth{
		Year:  uint(year),
		Month: time.Month(month),
	}, nil
}
