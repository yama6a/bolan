package crawler

import (
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
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

var (
	_ SiteCrawler = &SebBankCrawler{}

	ErrNoInterestSetFound      = errors.New("no interest set found")
	ErrUnsupportedTerm         = errors.New("unsupported term")
	ErrUnsupportedInterestRate = errors.New("unsupported interest rate")
	ErrUnsupportedChangeDate   = errors.New("unsupported change date")
	ErrUnsupportedAvgMonth     = errors.New("unsupported avg month")

	interestRegex    = regexp.MustCompile(`^(\d+\.\d+ ?)%$`)
	changedDateRegex = regexp.MustCompile(`^(\d{4})-([0][1-9]|1[0-2])-([0-2][1-9]|[1-3]0|3[01])$`)
)

type SebBankCrawler struct {
	logger *zap.Logger
}

func NewSebBankCrawler(logger *zap.Logger) *SebBankCrawler {
	return &SebBankCrawler{logger: logger}
}

func (d SebBankCrawler) Crawl(channel chan<- model.InterestSet) {
	interestSets, err := d.parseRates(sebListRatesUrl, DecoderWindows1252, model.TypeListRate)
	if err != nil {
		d.logger.Error("failed parsing SEB List Rates website", zap.Error(err))
	}

	avgInterest, err := d.parseRates(sebAverageRatesUrl, DecoderUtf8, model.TypeAverageRate)
	if err != nil {
		d.logger.Error("failed parsing SEB Avg Rates website", zap.Error(err))
	}

	for _, set := range append(avgInterest, interestSets...) {
		channel <- set
	}
}

//goland:noinspection ALL
func (d SebBankCrawler) parseRates(url string, decoder Decoder, t model.Type) ([]model.InterestSet, error) {
	crawlTime := time.Now().UTC()
	rawHtml, err := fetchHtmlFromUrl(url, decoder)
	if err != nil {
		d.logger.Error("failed reading SEB website for ListRates", zap.Error(err))
		return nil, fmt.Errorf("failed reading SEB website for ListRates: %w", err)
	}
	d.logger.Debug("successfully read SEB List Rates website", zap.Any("rawHtml", rawHtml))

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
				interestSet, err := d.extractInterestSetFromRow(t, tokenizer, crawlTime)
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

func (d SebBankCrawler) extractInterestSetFromRow(t model.Type, tokenizer *html.Tokenizer, crawlTime time.Time) (model.InterestSet, error) {
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
			d.logger.Debug("error token", zap.Any("error", err))
			if err == io.EOF {
				return interestSet, nil
			}
			return model.InterestSet{}, err

		case html.EndTagToken:
			data := tokenizer.Token().Data
			if data == "tr" {
				d.logger.Debug("end TR tag, breaking loop!", zap.Any("token", data))
				break loop
			}

		case html.StartTagToken:
			token := tokenizer.Token()
			if token.Data != "td" {
				d.logger.Debug("start tag token, not TD, continue loop", zap.Any("token", token.Data))
				continue
			}

			innerNode := tokenizer.Next()
			innerToken := tokenizer.Token()
			if innerNode != html.TextToken {
				d.logger.Debug("next token not TEXT, continue loop", zap.Any("token", innerToken))
				continue
			}
			d.logger.Debug("next token is TEXT", zap.Any("token", innerToken.Data))

			avgMonth, err := d.extractAvgMonth(innerToken.Data)
			if err == nil {
				interestSet.AverageReferenceMonth = &avgMonth
				continue
			}

			term, err := d.extractTerm(innerToken.Data)
			d.logger.Debug("extracted term", zap.Any("term", term), zap.Any("error", err))
			if err == nil {
				interestSet.Term = term
				continue
			}

			nominalRate, err := d.extractNominalRate(innerToken.Data)
			d.logger.Debug("extracted nominal rate", zap.Any("nominalRate", nominalRate), zap.Any("error", err))
			if err == nil {
				interestSet.NominalRate = nominalRate
				continue
			}

			changedDate, err := d.extractChangeDate(innerToken.Data)
			d.logger.Debug("extracted change date", zap.Any("changedDate", changedDate), zap.Any("error", err))
			if err == nil {
				interestSet.ChangedOn = &changedDate
				continue
			}

		}
	}

	if interestSet.Term == "" || interestSet.NominalRate == 0 {
		d.logger.Debug("no interest set found", zap.Any("interestSet", interestSet), zap.String("term", string(interestSet.Term)))
		return model.InterestSet{}, ErrNoInterestSetFound
	}

	return interestSet, nil
}

func (d SebBankCrawler) extractAvgMonth(data string) (model.AvgMonth, error) {
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
		d.logger.Debug("no month found", zap.Any("data", data))
		return Month, ErrUnsupportedAvgMonth
	}

	return Month, nil
}

func (d SebBankCrawler) extractTerm(data string) (model.Term, error) {
	data = normalizeString(data)

	switch data {
	case "3mån":
		return model.Term3months, nil
	case "1år":
		return model.Term1year, nil
	case "2år":
		return model.Term2years, nil
	case "3år":
		return model.Term3years, nil
	case "4år":
		return model.Term4years, nil
	case "5år":
		return model.Term5years, nil
	case "6år":
		return model.Term6years, nil
	case "7år":
		return model.Term7years, nil
	case "8år":
		return model.Term8years, nil
	case "9år":
		return model.Term9years, nil
	case "10år":
		return model.Term10years, nil
	default:
		d.logger.Debug("no term found", zap.Any("data", data))
		return "", ErrUnsupportedTerm
	}
}

func (d SebBankCrawler) extractNominalRate(data string) (float32, error) {
	data = normalizeString(data)
	data = strings.Replace(data, ",", ".", -1) // replace Swedish decimal separator with dot

	matches := interestRegex.FindStringSubmatch(data)
	if len(matches) != 2 {
		d.logger.Debug("regex for nominal rate did not match", zap.Any("data", data), zap.Any("matches", matches))
		return 0, ErrUnsupportedInterestRate
	}

	rate, err := strconv.ParseFloat(matches[1], 32)
	if err != nil {
		d.logger.Warn("failed to parse interest rate to float", zap.Error(err), zap.String("data", data), zap.Any("regexMatches", matches))
		return 0, fmt.Errorf("failed to parse interest rate: %w", err)
	}

	return float32(rate), nil
}

func (d SebBankCrawler) extractChangeDate(str string) (time.Time, error) {
	str = normalizeString(str)

	matches := changedDateRegex.FindStringSubmatch(str)
	if len(matches) != 4 {
		return time.Time{}, ErrUnsupportedChangeDate
	}

	date, err := time.Parse("2006-01-02", matches[0])
	if err != nil {
		d.logger.Warn("failed to parse change date", zap.Error(err), zap.String("str", str), zap.Any("regexMatches", matches))
		return time.Time{}, fmt.Errorf("failed to parse change date: %w", err)
	}

	return date, nil
}
