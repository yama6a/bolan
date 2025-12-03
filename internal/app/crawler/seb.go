package crawler

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/yama6a/bolan-compare/internal/pkg/http"
	"github.com/yama6a/bolan-compare/internal/pkg/model"
	"github.com/yama6a/bolan-compare/internal/pkg/utils"
	"go.uber.org/zap"
)

const (
	sebBankName              model.Bank = "SEB"
	sebAvgCurrentHTMLURL     string     = "https://pricing-portal-web-public.clouda.sebgroup.com/mortgage/averageratehistoric"
	sebAPIKeyJsFileURLPrefix string     = "https://pricing-portal-web-public.clouda.sebgroup.com/" //nolint:gosec // does not contain secret
	sebListRateURL           string     = "https://pricing-portal-api-public.clouda.sebgroup.com/public/mortgage/listrate/current"
	sebAverageRatesURL       string     = "https://pricing-portal-api-public.clouda.sebgroup.com/public/mortgage/averagerate/historic"
)

var (
	_                      SiteCrawler = &SebBankCrawler{}
	jsFileRegex                        = regexp.MustCompile(`main\.[a-zA-Z0-9]+\.js`)
	apiKeyRegex                        = regexp.MustCompile(`x-api-key":"(.*?)"`)
	isoDateRegex                       = regexp.MustCompile(`^((\d{4})-(0[1-9]|1[0-2])-([0-2][1-9]|[1-3]0|3[01])).*$`) // YYYY-MM-DD
	yearMonthReferenceDate             = regexp.MustCompile(`^(\d{2})(0[1-9]|1[0-2])$`)                                // YYMM
)

type SebBankCrawler struct {
	httpClient http.Client
	logger     *zap.Logger
}

type sebListRatesResponseItem struct {
	AdjustmentTerm string  `json:"adjustmentTerm"`
	Change         float32 `json:"change"`
	StartDate      string  `json:"startDate"`
	Value          float32 `json:"value"`
}

type sebAverageRatesResponse struct {
	Period uint               `json:"period"`
	Rates  map[string]float32 `json:"rates"`
}

func NewSebBankCrawler(httpClient http.Client, logger *zap.Logger) *SebBankCrawler {
	return &SebBankCrawler{httpClient: httpClient, logger: logger}
}

func (c *SebBankCrawler) Crawl(channel chan<- model.InterestSet) {
	crawlTime := time.Now().UTC()

	apiKey, err := c.fetchAPIKey()
	if err != nil {
		c.logger.Error("failed fetching SEB API key", zap.Error(err))
		return
	}

	listRates, err := c.fetchListRates(apiKey, crawlTime)
	if err != nil {
		c.logger.Error("failed fetching SEB list rates", zap.Error(err))
	}

	avgRates, err := c.fetchAverageRates(apiKey, crawlTime)
	if err != nil {
		c.logger.Error("failed fetching SEB average rates", zap.Error(err))
	}

	for _, set := range append(listRates, avgRates...) {
		channel <- set
	}
}

func (c *SebBankCrawler) fetchAPIKey() (string, error) {
	rawHTML, err := c.httpClient.Fetch(sebAvgCurrentHTMLURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed reading SEB website that references JS file that contains API key: %w", err)
	}

	jsFileName := jsFileRegex.FindString(rawHTML)
	if jsFileName == "" {
		return "", errors.New("failed finding file name for JS file that contains API key")
	}

	jsFileURL := sebAPIKeyJsFileURLPrefix + jsFileName
	rawJs, err := c.httpClient.Fetch(jsFileURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed reading SEB JS file for API key: %w", err)
	}

	apiKeyResult := apiKeyRegex.FindStringSubmatch(rawJs)
	if len(apiKeyResult) != 2 {
		return "", errors.New("failed finding API key in JS file")
	}
	apiKey := apiKeyResult[1]

	return apiKey, nil
}

func (c *SebBankCrawler) fetchListRates(apiKey string, crawlTime time.Time) ([]model.InterestSet, error) {
	rawJSON, err := c.fetchWithAPIKey(sebListRateURL, apiKey)
	if err != nil {
		return nil, fmt.Errorf("failed reading SEB list rates API: %w", err)
	}

	var listRates []sebListRatesResponseItem
	if err := json.Unmarshal([]byte(rawJSON), &listRates); err != nil {
		c.logger.Error("failed unmarshalling SEB list rates", zap.Error(err), zap.String("rawJSON", rawJSON))
		return nil, fmt.Errorf("failed unmarshalling SEB list rates: %w", err)
	}

	interestSets := []model.InterestSet{}
	for _, rate := range listRates {
		term, err := utils.ParseTerm(rate.AdjustmentTerm)
		if err != nil {
			c.logger.Warn("SEB list rate term not supported - skipping", zap.Any("rateObj", rate), zap.Error(err))
			continue
		}

		changeDate, err := parseSEBChangeDate(rate.StartDate, isoDateRegex)
		if err != nil {
			c.logger.Warn("failed parsing SEB list rate change date", zap.Any("rateObj", rate), zap.Error(err))
			continue
		}

		interestSets = append(interestSets, model.InterestSet{
			Bank:          sebBankName,
			Type:          model.TypeListRate,
			Term:          term,
			NominalRate:   rate.Value,
			ChangedOn:     &changeDate,
			LastCrawledAt: crawlTime,

			RatioDiscountBoundaries: nil,
			UnionDiscount:           false,
			AverageReferenceMonth:   nil,
		})
	}

	return interestSets, nil
}

func (c *SebBankCrawler) fetchWithAPIKey(url string, apiKey string) (string, error) {
	origin, _ := strings.CutSuffix(sebAPIKeyJsFileURLPrefix, "/")
	headers := map[string]string{
		"X-API-Key": apiKey,
		"Referer":   sebAPIKeyJsFileURLPrefix,
		"Origin":    origin,
	}

	return c.httpClient.Fetch(url, headers) //nolint:wrapcheck
}

func (c *SebBankCrawler) fetchAverageRates(apiKey string, crawlTime time.Time) ([]model.InterestSet, error) {
	rawJSON, err := c.fetchWithAPIKey(sebAverageRatesURL, apiKey)
	if err != nil {
		return nil, fmt.Errorf("failed reading SEB average rates API: %w", err)
	}

	var avgRates []sebAverageRatesResponse
	if err := json.Unmarshal([]byte(rawJSON), &avgRates); err != nil {
		c.logger.Error("failed unmarshalling SEB average rates", zap.Error(err), zap.String("rawJSON", rawJSON))
		return nil, fmt.Errorf("failed unmarshalling SEB average rates: %w", err)
	}

	interestSets := []model.InterestSet{}
	for _, rate := range avgRates {
		period, err := parseReferenceMonth(rate.Period, yearMonthReferenceDate)
		if err != nil {
			c.logger.Warn("failed parsing SEB average rate reference month", zap.Uint("period", rate.Period), zap.Error(err))
			continue
		}

		for termStr, rate := range rate.Rates {
			term, err := utils.ParseTerm(termStr)
			if err != nil {
				if errors.Is(err, utils.ErrTermHeader) {
					continue // skip header
				}
				c.logger.Warn("SEB average rate term not supported - skipping", zap.String("term", termStr), zap.Error(err))
				continue
			}

			interestSets = append(interestSets, model.InterestSet{
				AverageReferenceMonth: &period,
				Bank:                  sebBankName,
				Type:                  model.TypeAverageRate,
				Term:                  term,
				NominalRate:           rate,
				LastCrawledAt:         crawlTime,

				RatioDiscountBoundaries: nil,
				UnionDiscount:           false,
				ChangedOn:               nil,
			})
		}
	}

	return interestSets, nil
}

func parseReferenceMonth(data uint, regex *regexp.Regexp) (model.AvgMonth, error) {
	matches := regex.FindStringSubmatch(fmt.Sprintf("%d", data))
	if len(matches) != 3 {
		return model.AvgMonth{}, fmt.Errorf("failed to match reference month: %d", data)
	}

	year, err := strconv.Atoi(matches[1])
	if err != nil || year < 0 {
		return model.AvgMonth{}, fmt.Errorf("failed to parse year: %w", err)
	}

	month, err := strconv.Atoi(matches[2])
	if err != nil {
		return model.AvgMonth{}, fmt.Errorf("failed to parse month: %w", err)
	}

	// assume all double-digit year numbers lower than 40 are from the 21st century, otherwise 20th century. This will
	// ensure that this function works until the year 2039 and assumes we don't get historical data from before 1940
	// presented in this format.
	if year < 40 {
		year += 2000
	} else {
		year += 1900
	}

	return model.AvgMonth{
		Year:  uint(year), //nolint:gosec // year will never be negative due to the above checks
		Month: time.Month(month),
	}, nil
}

func parseSEBChangeDate(str string, regex *regexp.Regexp) (time.Time, error) {
	str = utils.NormalizeSpaces(str)

	matches := regex.FindStringSubmatch(str)
	if len(matches) != 5 {
		return time.Time{}, fmt.Errorf("failed to match change date")
	}

	date, err := time.Parse("2006-01-02", matches[1])
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse change date: %w", err)
	}

	return date, nil
}
