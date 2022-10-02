package crawler

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/ymakhloufi/bolan-compare/internal/pkg/model"
	"go.uber.org/zap"
	"golang.org/x/net/html"
)

const (
	sebListRatesUrl               = "https://seb.se/pow/apps/Borantor/bo_rantor.asp"
	sebAverageRatesUrl            = "https://seb.se/pow/apps/genomsnittsrantor/genomsnittsranta.aspx"
	sebBankName        model.Bank = "SEB"
)

var _ SiteCrawler = &SebBankCrawler{}

type SebBankCrawler struct {
	logger *zap.Logger
}

func NewSebBankCrawler(logger *zap.Logger) *SebBankCrawler {
	return &SebBankCrawler{logger: logger}
}

func (c SebBankCrawler) Crawl(channel chan<- model.InterestSet) {
	interestSets, err := c.parseRates(sebListRatesUrl, DecoderWindows1252, model.TypeListRate)
	if err != nil {
		c.logger.Error("failed parsing SEB List Rates website", zap.Error(err))
	}

	avgInterest, err := c.parseRates(sebAverageRatesUrl, DecoderUtf8, model.TypeAverageRate)
	if err != nil {
		c.logger.Error("failed parsing SEB Avg Rates website", zap.Error(err))
	}

	for _, set := range append(avgInterest, interestSets...) {
		channel <- set
	}
}

//goland:noinspection ALL
func (c SebBankCrawler) parseRates(url string, decoder Decoder, t model.Type) ([]model.InterestSet, error) {
	crawlTime := time.Now().UTC()
	rawHtml, err := fetchHtmlFromUrl(url, decoder)
	if err != nil {
		c.logger.Error("failed reading SEB website for ListRates", zap.Error(err))
		return nil, fmt.Errorf("failed reading SEB website for ListRates: %w", err)
	}

	interestSets := []model.InterestSet{}
	tokenizer := html.NewTokenizer(strings.NewReader(rawHtml))
	for {
		node := tokenizer.Next()
		switch node {
		case html.ErrorToken:
			if tokenizer.Err() == io.EOF {
				return interestSets, nil
			}
			return nil, tokenizer.Err()
		case html.StartTagToken:
			tkn := tokenizer.Token()
			if tkn.Data == "tr" {
				interestSet, err := c.extractInterestSetFromRow(t, tokenizer, crawlTime)
				if err != nil {
					if err == ErrNoInterestSetFound {
						continue // ignore rows for which no interest set could be extracted
					}
					return nil, err
				}
				interestSets = append(interestSets, interestSet)
			}
		}
	}

	return interestSets, nil

}

func (c SebBankCrawler) extractInterestSetFromRow(t model.Type, tokenizer *html.Tokenizer, crawlTime time.Time) (model.InterestSet, error) {
	interestSet := model.InterestSet{
		Bank:          sebBankName,
		Type:          t,
		LastCrawledAt: crawlTime,
	}

loop:
	for {
		switch tokenizer.Next() {

		case html.ErrorToken:
			err := tokenizer.Err()
			c.logger.Debug("error token", zap.Any("error", err))
			if err == io.EOF {
				return interestSet, nil
			}
			return model.InterestSet{}, err

		case html.EndTagToken:
			data := tokenizer.Token().Data
			if data == "tr" {
				c.logger.Debug("end TR tag, breaking loop!", zap.Any("token", data))
				break loop
			}

		case html.StartTagToken:
			token := tokenizer.Token()
			if token.Data != "td" {
				c.logger.Debug("start tag token, not TD, continue loop", zap.Any("token", token.Data))
				continue
			}

			innerNode := tokenizer.Next()
			innerToken := tokenizer.Token()
			if innerNode != html.TextToken {
				c.logger.Debug("next token not TEXT, continue loop", zap.Any("token", innerToken))
				continue
			}
			c.logger.Debug("next token is TEXT", zap.Any("token", innerToken.Data))

			avgMonth, err := c.extractAvgMonth(innerToken.Data)
			if err == nil {
				interestSet.AverageReferenceMonth = &avgMonth
				continue
			}

			term, err := parseTerm(innerToken.Data)
			c.logger.Debug("extracted term", zap.Any("term", term), zap.Any("error", err))
			if err == nil {
				interestSet.Term = term
				continue
			}

			nominalRate, err := parseNominalRate(innerToken.Data)
			c.logger.Debug("extracted nominal rate", zap.Any("nominalRate", nominalRate), zap.Any("error", err))
			if err == nil {
				interestSet.NominalRate = nominalRate
				continue
			}

			changedDate, err := parseChangeDate(innerToken.Data, isoDateRegex)
			c.logger.Debug("extracted change date", zap.Any("changedDate", changedDate), zap.Any("error", err))
			if err == nil {
				interestSet.ChangedOn = &changedDate
				continue
			}

		}
	}

	if interestSet.Term == "" || interestSet.NominalRate == 0 {
		c.logger.Debug("no interest set found", zap.Any("interestSet", interestSet), zap.String("term", string(interestSet.Term)))
		return model.InterestSet{}, ErrNoInterestSetFound
	}

	return interestSet, nil
}

func (c SebBankCrawler) extractAvgMonth(data string) (model.AvgMonth, error) {
	data = normalizeString(data)
	data = strings.ToLower(data)

	Month := model.AvgMonth{
		Year: uint(time.Now().Year()),
	}
	switch data {
	case "jan", "januari", "january":
		Month.Month = time.January
	case "feb", "februari", "february":
		Month.Month = time.February
	case "mar", "mars", "march":
		Month.Month = time.March
	case "apr", "april":
		Month.Month = time.April
	case "may", "maj":
		Month.Month = time.May
	case "jun", "juni", "june":
		Month.Month = time.June
	case "jul", "juli", "july":
		Month.Month = time.July
	case "aug", "augusti", "august":
		Month.Month = time.August
	case "sep", "september":
		Month.Month = time.September
	case "okt", "oktober", "october":
		Month.Month = time.October
	case "nov", "november":
		Month.Month = time.November
	case "dec", "december":
		Month.Month = time.December
		if time.Now().Month() == time.January {
			Month.Year -= 1
		}
	default:
		c.logger.Debug("no month found", zap.Any("data", data))
		return Month, ErrUnsupportedAvgMonth
	}

	return Month, nil
}
