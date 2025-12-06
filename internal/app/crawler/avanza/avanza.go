package avanza

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/yama6a/bolan-compare/internal/app/crawler"
	"github.com/yama6a/bolan-compare/internal/pkg/http"
	"github.com/yama6a/bolan-compare/internal/pkg/model"
	"go.uber.org/zap"
)

const (
	avanzaBankName        model.Bank = "Avanza"
	avanzaStabeloRatesURL string     = "https://www.avanza.se/_api/external-mortgage-stabelo/interest-table"
	avanzaLHBRatesURL     string     = "https://www.avanza.se/_api/external-mortgage-lhb/interest-table"
)

var _ crawler.SiteCrawler = &AvanzaCrawler{}

// AvanzaCrawler crawls Avanza's mortgage rate APIs.
// Avanza offers mortgages via two partners: Stabelo and Landshypotek (LHB).
// Only list rates are available; no average rates are published.
//
//nolint:revive // Bank name prefix is intentional for clarity
type AvanzaCrawler struct {
	httpClient http.Client
	logger     *zap.Logger
}

// avanzaRatesResponse represents the JSON response from Avanza's rate APIs.
type avanzaRatesResponse struct {
	Rows []avanzaRateRow `json:"rows"`
}

type avanzaRateRow struct {
	MinLoanToValue float64              `json:"minLoanToValue"`
	MinLoanAmount  int                  `json:"minLoanAmount"`
	InterestRates  []avanzaInterestRate `json:"interestRates"`
}

type avanzaInterestRate struct {
	BindingPeriod string  `json:"bindingPeriod"` // THREE_MONTHS, ONE_YEAR, etc.
	Effective     float64 `json:"effective"`
	Nominal       float64 `json:"nominal"`
}

func NewAvanzaCrawler(httpClient http.Client, logger *zap.Logger) *AvanzaCrawler {
	return &AvanzaCrawler{httpClient: httpClient, logger: logger}
}

func (c *AvanzaCrawler) Crawl(channel chan<- model.InterestSet) {
	crawlTime := time.Now().UTC()

	// Fetch rates from both partners
	stabeloRates, err := c.fetchRates(avanzaStabeloRatesURL, "Stabelo", crawlTime)
	if err != nil {
		c.logger.Error("failed fetching Avanza Stabelo rates", zap.Error(err))
	}

	lhbRates, err := c.fetchRates(avanzaLHBRatesURL, "Landshypotek", crawlTime)
	if err != nil {
		c.logger.Error("failed fetching Avanza Landshypotek rates", zap.Error(err))
	}

	for _, set := range append(stabeloRates, lhbRates...) {
		channel <- set
	}
}

func (c *AvanzaCrawler) fetchRates(url, partner string, crawlTime time.Time) ([]model.InterestSet, error) {
	rawJSON, err := c.httpClient.Fetch(url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed reading Avanza %s rates API: %w", partner, err)
	}

	var response avanzaRatesResponse
	if err := json.Unmarshal([]byte(rawJSON), &response); err != nil {
		c.logger.Error("failed unmarshalling Avanza rates",
			zap.String("partner", partner),
			zap.Error(err),
			zap.String("rawJSON", rawJSON))
		return nil, fmt.Errorf("failed unmarshalling Avanza %s rates: %w", partner, err)
	}

	// Find the base rate row (minLoanToValue=0, minLoanAmount=0)
	// This represents the "list rate" - the worst-case rate without volume/LTV discounts
	baseRow := c.findBaseRateRow(response.Rows)
	if baseRow == nil {
		c.logger.Warn("no base rate row found for Avanza partner", zap.String("partner", partner))
		return nil, nil
	}

	interestSets := []model.InterestSet{}
	for _, rate := range baseRow.InterestRates {
		term, err := parseAvanzaBindingPeriod(rate.BindingPeriod)
		if err != nil {
			c.logger.Warn("Avanza binding period not supported - skipping",
				zap.String("partner", partner),
				zap.String("bindingPeriod", rate.BindingPeriod),
				zap.Error(err))
			continue
		}

		interestSets = append(interestSets, model.InterestSet{
			Bank:          avanzaBankName,
			Type:          model.TypeListRate,
			Term:          term,
			NominalRate:   float32(rate.Nominal),
			LastCrawledAt: crawlTime,

			ChangedOn:               nil, // Avanza API doesn't provide change dates
			RatioDiscountBoundaries: nil,
			UnionDiscount:           false,
			AverageReferenceMonth:   nil,
		})
	}

	return interestSets, nil
}

// findBaseRateRow finds the row with minLoanToValue=0 and minLoanAmount=0.
// This is the "list rate" row without any discounts applied.
func (c *AvanzaCrawler) findBaseRateRow(rows []avanzaRateRow) *avanzaRateRow {
	for i := range rows {
		if rows[i].MinLoanToValue == 0 && rows[i].MinLoanAmount == 0 {
			return &rows[i]
		}
	}
	return nil
}

// parseAvanzaBindingPeriod converts Avanza's binding period strings to model.Term.
func parseAvanzaBindingPeriod(period string) (model.Term, error) {
	switch period {
	case "THREE_MONTHS":
		return model.Term3months, nil
	case "ONE_YEAR":
		return model.Term1year, nil
	case "TWO_YEARS":
		return model.Term2years, nil
	case "THREE_YEARS":
		return model.Term3years, nil
	case "FOUR_YEARS":
		return model.Term4years, nil
	case "FIVE_YEARS":
		return model.Term5years, nil
	case "TEN_YEARS":
		return model.Term10years, nil
	default:
		return "", fmt.Errorf("unsupported binding period: %s", period)
	}
}
