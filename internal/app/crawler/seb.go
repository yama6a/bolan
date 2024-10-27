package crawler

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ymakhloufi/bolan-compare/internal/pkg/model"
	"go.uber.org/zap"
	"regexp"
	"strings"
	"time"
)

const (
	sebBankName              model.Bank = "SEB"
	sebAvgCurrentHtmlUrl     string     = "https://pricing-portal-web-public.clouda.sebgroup.com/mortgage/averageratecurrent"
	sebApiKeyJsFileUrlPrefix string     = "https://pricing-portal-web-public.clouda.sebgroup.com/"
	sebListRateUrl           string     = "https://pricing-portal-api-public.clouda.sebgroup.com/public/mortgage/listrate/current"
	sebAverageRatesUrl       string     = "https://pricing-portal-api-public.clouda.sebgroup.com/public/mortgage/averagerate/current"
)

var (
	_           SiteCrawler = &SebBankCrawler{}
	jsFileRegex             = regexp.MustCompile(`main\.[a-zA-Z0-9]+\.js`)
	apiKeyRegex             = regexp.MustCompile(`x-api-key":"(.*?)"`)
)

type SebBankCrawler struct {
	logger *zap.Logger
}

func NewSebBankCrawler(logger *zap.Logger) *SebBankCrawler {
	return &SebBankCrawler{logger: logger}
}

func (c *SebBankCrawler) Crawl(channel chan<- model.InterestSet) {
	crawlTime := time.Now().UTC()

	apiKey, err := c.fetchApiKey()
	if err != nil {
		c.logger.Error("failed fetching SEB API key", zap.Error(err))
		return
	}

	listRates, err := c.fetchListRates(apiKey, crawlTime)
	if err != nil {
		c.logger.Error("failed fetching SEB list rates", zap.Error(err))
	}

	//avgRates, err := c.fetchAverageRates(apiKey)
	//if err != nil {
	//	c.logger.Error("failed fetching SEB average rates", zap.Error(err))
	//}

	for _, set := range listRates {
		channel <- set
	}
}

func (c *SebBankCrawler) fetchApiKey() (string, error) {
	rawHtml, err := fetchRawContentFromUrl(sebAvgCurrentHtmlUrl, DecoderUtf8, nil)
	if err != nil {
		return "", fmt.Errorf("failed reading SEB website that references JS file that contains API key: %w", err)
	}

	jsFileName := jsFileRegex.FindString(rawHtml)
	if jsFileName == "" {
		return "", errors.New("failed finding file name for JS file that contains API key")
	}

	jsFileUrl := sebApiKeyJsFileUrlPrefix + jsFileName
	rawJs, err := fetchRawContentFromUrl(jsFileUrl, DecoderUtf8, nil)
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

type sebListRates struct {
	AdjustmentTerm string  `json:"adjustmentTerm"`
	Change         float32 `json:"change"`
	StartDate      string  `json:"startDate"`
	Value          float32 `json:"value"`
}

type sebListRatesResponse []sebListRates

func (c *SebBankCrawler) fetchListRates(apiKey string, crawlTime time.Time) ([]model.InterestSet, error) {
	rawJson, err := c.fetchRawContentFromUrl(sebListRateUrl, DecoderUtf8, apiKey)
	if err != nil {
		return nil, fmt.Errorf("failed reading SEB list rates API: %w", err)
	}

	var listRates []sebListRates
	if err := json.Unmarshal([]byte(rawJson), &listRates); err != nil {
		c.logger.Error("failed unmarshalling SEB list rates", zap.Error(err), zap.String("rawJson", rawJson))
		return nil, fmt.Errorf("failed unmarshalling SEB list rates: %w", err)
	}

	interestSets := []model.InterestSet{}
	for _, rate := range listRates {
		term, err := parseTerm(rate.AdjustmentTerm)
		if err != nil {
			c.logger.Warn("SEB list rate term not supported - skipping", zap.Any("rateObj", rate), zap.Error(err))
			continue
		}

		changeDate, err := parseChangeDate(rate.StartDate, isoDateRegex)
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

func (c *SebBankCrawler) fetchRawContentFromUrl(url string, decoder Decoder, apiKey string) (string, error) {
	origin, _ := strings.CutSuffix(sebApiKeyJsFileUrlPrefix, "/")
	headers := map[string]string{
		"X-API-Key": apiKey,
		"Referer":   sebApiKeyJsFileUrlPrefix,
		"Origin":    origin,
	}

	return fetchRawContentFromUrl(url, decoder, headers)
}
