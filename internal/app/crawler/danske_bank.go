package crawler

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/ymakhloufi/bolan-compare/internal/pkg/model"
	"go.uber.org/zap"
	"golang.org/x/net/html"
)

const (
	danskeUrl = "https://danskebank.se/privat/produkter/bolan/relaterat/aktuella-bolanerantor"

	danskeBankName model.Bank = "Danske Bank"
)

var (
	_ SiteCrawler = &DanskeBankCrawler{}
)

type DanskeBankCrawler struct {
	logger *zap.Logger
}

func NewDanskeBankCrawler(logger *zap.Logger) *DanskeBankCrawler {
	return &DanskeBankCrawler{logger: logger}
}

func (c *DanskeBankCrawler) Crawl(channel chan<- model.InterestSet) {
	interestSets := []model.InterestSet{}

	crawlTime := time.Now().UTC()
	rawHtml, err := fetchRawContentFromUrl(danskeUrl, DecoderUtf8, nil)
	if err != nil {
		c.logger.Error("failed reading Danske website for ListRates", zap.Error(err))
		return
	}

	listInterestSets, err := c.parseListRates(rawHtml, crawlTime)
	if err != nil {
		c.logger.Error("failed parsing Danske List Rates website", zap.Error(err))
	} else {
		interestSets = append(interestSets, listInterestSets...)
	}

	//discountedInterest, err := c.parseDiscountedRates(rawHtml, crawlTime)
	//if err != nil {
	//	c.logger.Error("failed parsing Danske Avg Rates website", zap.Error(err))
	//} else {
	//	interestSets = append(interestSets, discountedInterest...)
	//}

	//avgInterest, err := c.parseAverageRates(sebAverageRatesUrl, model.TypeAverageRate)
	//if err != nil {
	//	c.logger.Error("failed parsing Danske Avg Rates website", zap.Error(err))
	//} else {
	//	interestSets = append(interestSets, avgInterest...)
	//}
	//
	//unionInterest, err := c.parseUnionRates(sebAverageRatesUrl, model.TypeAverageRate)
	//if err != nil {
	//	c.logger.Error("failed parsing Danske Avg Rates website", zap.Error(err))
	//} else {
	//	interestSets = append(interestSets, unionInterest...)
	//}

	for _, set := range interestSets {
		channel <- set
	}
}

func (c *DanskeBankCrawler) parseListRates(rawHtml string, crawlTime time.Time) ([]model.InterestSet, error) {
	interestSets := []model.InterestSet{}
	tokenizer, err := findTokenizedTableByTextBeforeTable(rawHtml, "Bankens aktuella")
	if err != nil {
		return nil, fmt.Errorf("failed to find table by text 'Bankens aktuella' before table: %w", err)
	}

	for node := tokenizer.Next(); node != html.ErrorToken; node = tokenizer.Next() {
		switch node {
		case html.EndTagToken:
			tkn := tokenizer.Token()
			if tkn.Data == "table" {
				return interestSets, nil
			}
		case html.StartTagToken:
			tkn := tokenizer.Token()
			if tkn.Data != "tr" {
				continue
			}

			interestSet, err := c.extractListInterestSetFromRow(tokenizer, crawlTime)
			if err != nil {
				if errors.Is(err, ErrNoInterestSetFound) {
					continue // ignore rows for which no interest set could be extracted, e.g. table header rows
				}
				return nil, err
			}
			interestSets = append(interestSets, interestSet)
		}
	}

	if tokenizer.Err() == io.EOF {
		return nil, fmt.Errorf("no closing table-tag found, EOF: %w", ErrNoInterestSetFound)
	}
	return nil, tokenizer.Err()
}

func (c *DanskeBankCrawler) extractListInterestSetFromRow(tokenizer *html.Tokenizer, crawlTime time.Time) (model.InterestSet, error) {
	interestSet := model.InterestSet{
		Bank:          danskeBankName,
		Type:          model.TypeListRate,
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

			for {
				innerNode := tokenizer.Next()
				if innerNode == html.ErrorToken {
					return model.InterestSet{}, tokenizer.Err()
				}

				if innerNode == html.EndTagToken {
					data := tokenizer.Token().Data
					if data == "td" {
						c.logger.Debug("TD with no useful data, skipping to next TD!", zap.Any("token", data))
						continue loop
					}
				}

				if innerNode != html.TextToken {
					continue
				}

				innerToken := tokenizer.Token()
				c.logger.Debug("next token is TEXT", zap.Any("token", innerToken.Data))

				if strings.Contains(innerToken.Data, "Premium") {
					return model.InterestSet{}, ErrNoInterestSetFound // Danske Bank offers multiple "Premium" interest rates, which are not relevant for this crawler
				}

				term, err := parseTerm(innerToken.Data)
				c.logger.Debug("extracted term", zap.Any("term", term), zap.Any("error", err))
				if err == nil {
					interestSet.Term = term
					continue loop
				}

				nominalRate, err := parseNominalRate(innerToken.Data)
				c.logger.Debug("extracted nominal rate", zap.Any("nominalRate", nominalRate), zap.Any("error", err))
				if err == nil {
					interestSet.NominalRate = nominalRate
					continue loop
				}

				changedDate, err := parseChangeDate(innerToken.Data, swedishDashedDateRegex)
				c.logger.Debug("extracted change date", zap.Any("changedDate", changedDate), zap.Any("error", err))
				if err == nil {
					interestSet.ChangedOn = &changedDate
					continue loop
				}
			}
		}
	}

	if interestSet.Term == "" || interestSet.NominalRate == 0 {
		c.logger.Debug("no interest set found", zap.Any("interestSet", interestSet), zap.String("term", string(interestSet.Term)))
		return model.InterestSet{}, ErrNoInterestSetFound
	}

	return interestSet, nil
}
