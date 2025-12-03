package crawler

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"time"

	"github.com/yama6a/bolan-compare/internal/pkg/http"
	"github.com/yama6a/bolan-compare/internal/pkg/model"
	"go.uber.org/zap"
)

const (
	handelsbankenBankName    model.Bank = "Handelsbanken"
	handelsbankenListRateURL string     = "https://www.handelsbanken.se/tron/slana/slan/service/mortgagerates/v1/interestrates"
	handelsbankenAvgRatesURL string     = "https://www.handelsbanken.se/tron/slana/slan/service/mortgagerates/v1/averagerates"
)

var (
	_                            SiteCrawler = &HandelsbankenCrawler{}
	handelsbankenPeriodYYYYMMRgx             = regexp.MustCompile(`^(\d{4})(0[1-9]|1[0-2])$`) // YYYYMM
)

type HandelsbankenCrawler struct {
	httpClient http.Client
	logger     *zap.Logger
}

// handelsbankenListRatesResponse represents the JSON response for list rates.
type handelsbankenListRatesResponse struct {
	InterestRates []handelsbankenInterestRate `json:"interestRates"`
}

type handelsbankenInterestRate struct {
	EffectiveRateValue handelsbankenRateValue `json:"effectiveRateValue"`
	PeriodBasisType    string                 `json:"periodBasisType"` // "3" = months, "4" = years
	RateValue          handelsbankenRateValue `json:"rateValue"`
	Term               string                 `json:"term"` // number of months/years
}

type handelsbankenRateValue struct {
	Value    string  `json:"value"`    // formatted string "3,84"
	ValueRaw float32 `json:"valueRaw"` // numeric value 3.84
}

// handelsbankenAvgRatesResponse represents the JSON response for average rates.
type handelsbankenAvgRatesResponse struct {
	AverageRatePeriods []handelsbankenAvgRatePeriod `json:"averageRatePeriods"`
}

type handelsbankenAvgRatePeriod struct {
	Period string                      `json:"period"` // YYYYMM format
	Rates  []handelsbankenInterestRate `json:"rates"`
}

func NewHandelsbankenCrawler(httpClient http.Client, logger *zap.Logger) *HandelsbankenCrawler {
	return &HandelsbankenCrawler{httpClient: httpClient, logger: logger}
}

func (c *HandelsbankenCrawler) Crawl(channel chan<- model.InterestSet) {
	crawlTime := time.Now().UTC()

	listRates, err := c.fetchListRates(crawlTime)
	if err != nil {
		c.logger.Error("failed fetching Handelsbanken list rates", zap.Error(err))
	}

	avgRates, err := c.fetchAverageRates(crawlTime)
	if err != nil {
		c.logger.Error("failed fetching Handelsbanken average rates", zap.Error(err))
	}

	for _, set := range append(listRates, avgRates...) {
		channel <- set
	}
}

func (c *HandelsbankenCrawler) fetchListRates(crawlTime time.Time) ([]model.InterestSet, error) {
	rawJSON, err := c.httpClient.Fetch(handelsbankenListRateURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed reading Handelsbanken list rates API: %w", err)
	}

	var response handelsbankenListRatesResponse
	if err := json.Unmarshal([]byte(rawJSON), &response); err != nil {
		c.logger.Error("failed unmarshalling Handelsbanken list rates", zap.Error(err), zap.String("rawJSON", rawJSON))
		return nil, fmt.Errorf("failed unmarshalling Handelsbanken list rates: %w", err)
	}

	interestSets := []model.InterestSet{}
	for _, rate := range response.InterestRates {
		term, err := parseHandelsbankenTerm(rate.PeriodBasisType, rate.Term)
		if err != nil {
			c.logger.Warn("Handelsbanken list rate term not supported - skipping",
				zap.String("periodBasisType", rate.PeriodBasisType),
				zap.String("term", rate.Term),
				zap.Error(err))
			continue
		}

		interestSets = append(interestSets, model.InterestSet{
			Bank:          handelsbankenBankName,
			Type:          model.TypeListRate,
			Term:          term,
			NominalRate:   rate.RateValue.ValueRaw,
			LastCrawledAt: crawlTime,

			ChangedOn:               nil, // Handelsbanken API doesn't provide change dates
			RatioDiscountBoundaries: nil,
			UnionDiscount:           false,
			AverageReferenceMonth:   nil,
		})
	}

	return interestSets, nil
}

func (c *HandelsbankenCrawler) fetchAverageRates(crawlTime time.Time) ([]model.InterestSet, error) {
	rawJSON, err := c.httpClient.Fetch(handelsbankenAvgRatesURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed reading Handelsbanken average rates API: %w", err)
	}

	var response handelsbankenAvgRatesResponse
	if err := json.Unmarshal([]byte(rawJSON), &response); err != nil {
		c.logger.Error("failed unmarshalling Handelsbanken average rates", zap.Error(err), zap.String("rawJSON", rawJSON))
		return nil, fmt.Errorf("failed unmarshalling Handelsbanken average rates: %w", err)
	}

	interestSets := []model.InterestSet{}
	for _, period := range response.AverageRatePeriods {
		avgMonth, err := parseHandelsbankenPeriod(period.Period)
		if err != nil {
			c.logger.Warn("failed parsing Handelsbanken average rate period", zap.String("period", period.Period), zap.Error(err))
			continue
		}

		for _, rate := range period.Rates {
			term, err := parseHandelsbankenTerm(rate.PeriodBasisType, rate.Term)
			if err != nil {
				c.logger.Warn("Handelsbanken average rate term not supported - skipping",
					zap.String("periodBasisType", rate.PeriodBasisType),
					zap.String("term", rate.Term),
					zap.Error(err))
				continue
			}

			interestSets = append(interestSets, model.InterestSet{
				AverageReferenceMonth: &avgMonth,
				Bank:                  handelsbankenBankName,
				Type:                  model.TypeAverageRate,
				Term:                  term,
				NominalRate:           rate.RateValue.ValueRaw,
				LastCrawledAt:         crawlTime,

				RatioDiscountBoundaries: nil,
				UnionDiscount:           false,
				ChangedOn:               nil,
			})
		}
	}

	return interestSets, nil
}

// handelsbankenYearTermMap maps year numbers to model.Term for Handelsbanken.
//
//nolint:gochecknoglobals // constant lookup map to reduce cyclomatic complexity
var handelsbankenYearTermMap = map[int]model.Term{
	1:  model.Term1year,
	2:  model.Term2years,
	3:  model.Term3years,
	4:  model.Term4years,
	5:  model.Term5years,
	6:  model.Term6years,
	7:  model.Term7years,
	8:  model.Term8years,
	9:  model.Term9years,
	10: model.Term10years,
}

// parseHandelsbankenTerm converts Handelsbanken's periodBasisType and term to model.Term.
// periodBasisType "3" = months, "4" = years.
func parseHandelsbankenTerm(periodBasisType, term string) (model.Term, error) {
	termNum, err := strconv.Atoi(term)
	if err != nil {
		return "", fmt.Errorf("failed to parse term number: %w", err)
	}

	switch periodBasisType {
	case "3": // months
		if termNum == 3 {
			return model.Term3months, nil
		}
		return "", fmt.Errorf("unsupported month term: %d", termNum)
	case "4": // years
		if t, ok := handelsbankenYearTermMap[termNum]; ok {
			return t, nil
		}
		return "", fmt.Errorf("unsupported year term: %d", termNum)
	default:
		return "", fmt.Errorf("unsupported periodBasisType: %s", periodBasisType)
	}
}

// parseHandelsbankenPeriod converts a YYYYMM period string to model.AvgMonth.
func parseHandelsbankenPeriod(period string) (model.AvgMonth, error) {
	matches := handelsbankenPeriodYYYYMMRgx.FindStringSubmatch(period)
	if len(matches) != 3 {
		return model.AvgMonth{}, fmt.Errorf("failed to match period format YYYYMM: %s", period)
	}

	year, err := strconv.Atoi(matches[1])
	if err != nil {
		return model.AvgMonth{}, fmt.Errorf("failed to parse year: %w", err)
	}

	month, err := strconv.Atoi(matches[2])
	if err != nil {
		return model.AvgMonth{}, fmt.Errorf("failed to parse month: %w", err)
	}

	return model.AvgMonth{
		Year:  uint(year), //nolint:gosec // year validated by regex
		Month: time.Month(month),
	}, nil
}
