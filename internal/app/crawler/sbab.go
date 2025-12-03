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
	sbabBankName     model.Bank = "SBAB"
	sbabListRatesURL string     = "https://www.sbab.se/api/interest-mortgage-service/api/external/v1/interest"
	sbabAvgRatesURL  string     = "https://www.sbab.se/api/historical-average-interest-rate-service/interest-rate/average-interest-rate-last-twelve-months-by-period"
)

var (
	_                    SiteCrawler = &SBABCrawler{}
	sbabPeriodYYYYMMDDRx             = regexp.MustCompile(`^(\d{4})-(\d{2})-(\d{2})$`) // YYYY-MM-DD
)

type SBABCrawler struct {
	httpClient http.Client
	logger     *zap.Logger
}

// sbabListRatesResponse represents the JSON response for SBAB list rates.
type sbabListRatesResponse struct {
	ListInterests []sbabListInterest `json:"listInterests"`
}

type sbabListInterest struct {
	Period       string `json:"period"`       // e.g., "P_3_MONTHS", "P_1_YEAR"
	InterestRate string `json:"interestRate"` // e.g., "3.05"
	ValidFrom    string `json:"validFrom"`    // e.g., "2025-09-29"
}

// sbabAvgRatesResponse represents the JSON response for SBAB average rates.
type sbabAvgRatesResponse struct {
	AverageInterestRateLast12Months []sbabAvgRatePeriod `json:"average_interest_rate_last_twelve_months"` //nolint:tagliatelle // external API uses snake_case
}

type sbabAvgRatePeriod struct {
	Period      string   `json:"period"`       // YYYY-MM-DD format (last day of month)
	ThreeMonths *float32 `json:"three_months"` //nolint:tagliatelle // external API uses snake_case
	OneYear     *float32 `json:"one_year"`     //nolint:tagliatelle // external API uses snake_case
	TwoYears    *float32 `json:"two_years"`    //nolint:tagliatelle // external API uses snake_case
	ThreeYears  *float32 `json:"three_years"`  //nolint:tagliatelle // external API uses snake_case
	FourYears   *float32 `json:"four_years"`   //nolint:tagliatelle // external API uses snake_case
	FiveYears   *float32 `json:"five_years"`   //nolint:tagliatelle // external API uses snake_case
	SevenYears  *float32 `json:"seven_years"`  //nolint:tagliatelle // external API uses snake_case
	TenYears    *float32 `json:"ten_years"`    //nolint:tagliatelle // external API uses snake_case
}

func NewSBABCrawler(httpClient http.Client, logger *zap.Logger) *SBABCrawler {
	return &SBABCrawler{httpClient: httpClient, logger: logger}
}

func (c *SBABCrawler) Crawl(channel chan<- model.InterestSet) {
	crawlTime := time.Now().UTC()

	listRates, err := c.fetchListRates(crawlTime)
	if err != nil {
		c.logger.Error("failed fetching SBAB list rates", zap.Error(err))
	}

	avgRates, err := c.fetchAverageRates(crawlTime)
	if err != nil {
		c.logger.Error("failed fetching SBAB average rates", zap.Error(err))
	}

	for _, set := range append(listRates, avgRates...) {
		channel <- set
	}
}

func (c *SBABCrawler) fetchListRates(crawlTime time.Time) ([]model.InterestSet, error) {
	rawJSON, err := c.httpClient.Fetch(sbabListRatesURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed reading SBAB list rates API: %w", err)
	}

	var response sbabListRatesResponse
	if err := json.Unmarshal([]byte(rawJSON), &response); err != nil {
		c.logger.Error("failed unmarshalling SBAB list rates", zap.Error(err), zap.String("rawJSON", rawJSON))
		return nil, fmt.Errorf("failed unmarshalling SBAB list rates: %w", err)
	}

	interestSets := []model.InterestSet{}
	for _, rate := range response.ListInterests {
		term, err := parseSBABPeriodToTerm(rate.Period)
		if err != nil {
			c.logger.Warn("SBAB list rate term not supported - skipping",
				zap.String("period", rate.Period),
				zap.Error(err))
			continue
		}

		nominalRate, err := strconv.ParseFloat(rate.InterestRate, 32)
		if err != nil {
			c.logger.Warn("failed to parse SBAB interest rate",
				zap.String("interestRate", rate.InterestRate),
				zap.Error(err))
			continue
		}

		var changedOn *time.Time
		if rate.ValidFrom != "" {
			parsedTime, err := time.Parse("2006-01-02", rate.ValidFrom)
			if err != nil {
				c.logger.Warn("failed to parse SBAB validFrom date",
					zap.String("validFrom", rate.ValidFrom),
					zap.Error(err))
			} else {
				changedOn = &parsedTime
			}
		}

		interestSets = append(interestSets, model.InterestSet{
			Bank:          sbabBankName,
			Type:          model.TypeListRate,
			Term:          term,
			NominalRate:   float32(nominalRate),
			ChangedOn:     changedOn,
			LastCrawledAt: crawlTime,

			RatioDiscountBoundaries: nil,
			UnionDiscount:           false,
			AverageReferenceMonth:   nil,
		})
	}

	return interestSets, nil
}

func (c *SBABCrawler) fetchAverageRates(crawlTime time.Time) ([]model.InterestSet, error) {
	rawJSON, err := c.httpClient.Fetch(sbabAvgRatesURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed reading SBAB average rates API: %w", err)
	}

	var response sbabAvgRatesResponse
	if err := json.Unmarshal([]byte(rawJSON), &response); err != nil {
		c.logger.Error("failed unmarshalling SBAB average rates", zap.Error(err), zap.String("rawJSON", rawJSON))
		return nil, fmt.Errorf("failed unmarshalling SBAB average rates: %w", err)
	}

	interestSets := []model.InterestSet{}
	for _, period := range response.AverageInterestRateLast12Months {
		avgMonth, err := parseSBABAvgPeriod(period.Period)
		if err != nil {
			c.logger.Warn("failed parsing SBAB average rate period", zap.String("period", period.Period), zap.Error(err))
			continue
		}

		// Add rates for each term if available (not null)
		termRates := []struct {
			term model.Term
			rate *float32
		}{
			{model.Term3months, period.ThreeMonths},
			{model.Term1year, period.OneYear},
			{model.Term2years, period.TwoYears},
			{model.Term3years, period.ThreeYears},
			{model.Term4years, period.FourYears},
			{model.Term5years, period.FiveYears},
			{model.Term7years, period.SevenYears},
			{model.Term10years, period.TenYears},
		}

		for _, tr := range termRates {
			if tr.rate == nil {
				continue // Skip null values
			}

			interestSets = append(interestSets, model.InterestSet{
				AverageReferenceMonth: &avgMonth,
				Bank:                  sbabBankName,
				Type:                  model.TypeAverageRate,
				Term:                  tr.term,
				NominalRate:           *tr.rate,
				LastCrawledAt:         crawlTime,

				RatioDiscountBoundaries: nil,
				UnionDiscount:           false,
				ChangedOn:               nil,
			})
		}
	}

	return interestSets, nil
}

// parseSBABPeriodToTerm converts SBAB's period format to model.Term.
// Period format examples: "P_3_MONTHS", "P_1_YEAR", "P_2_YEARS", etc.
func parseSBABPeriodToTerm(period string) (model.Term, error) {
	switch period {
	case "P_3_MONTHS":
		return model.Term3months, nil
	case "P_1_YEAR":
		return model.Term1year, nil
	case "P_2_YEARS":
		return model.Term2years, nil
	case "P_3_YEARS":
		return model.Term3years, nil
	case "P_4_YEARS":
		return model.Term4years, nil
	case "P_5_YEARS":
		return model.Term5years, nil
	case "P_7_YEARS":
		return model.Term7years, nil
	case "P_10_YEARS":
		return model.Term10years, nil
	default:
		return "", fmt.Errorf("unsupported SBAB period: %s", period)
	}
}

// parseSBABAvgPeriod converts a YYYY-MM-DD period string to model.AvgMonth.
// The day is ignored as SBAB uses the last day of the month.
func parseSBABAvgPeriod(period string) (model.AvgMonth, error) {
	matches := sbabPeriodYYYYMMDDRx.FindStringSubmatch(period)
	if len(matches) != 4 {
		return model.AvgMonth{}, fmt.Errorf("failed to match period format YYYY-MM-DD: %s", period)
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
		Year:  uint(year), //nolint:gosec // year validated by regex
		Month: time.Month(month),
	}, nil
}
