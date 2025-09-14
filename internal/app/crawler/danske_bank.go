package crawler

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/yama6a/bolan-compare/internal/pkg/utils"

	"github.com/yama6a/bolan-compare/internal/pkg/model"
	"go.uber.org/zap"
)

const (
	danskeUrl                 = "https://danskebank.se/privat/produkter/bolan/relaterat/aktuella-bolanerantor"
	danskeBankName model.Bank = "Danske Bank"
)

var (
	_ SiteCrawler = &DanskeBankCrawler{}

	// YY-MM-DD
	changedDateRegex = regexp.MustCompile(`^(\d{4})-(0[1-9]|1[0-2])-([0-2][1-9]|[1-3]0|3[01])$`)
	interestRegex    = regexp.MustCompile(`^(\d+\.\d+ ?)%?$`)

	swedishMonthMap = map[string]time.Month{
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
		"jan":       time.January,
		"feb":       time.February,
		"mar":       time.March,
		"apr":       time.April,
		"jun":       time.June,
		"jul":       time.July,
		"aug":       time.August,
		"sep":       time.September,
		"oct":       time.October,
		"nov":       time.November,
		"dec":       time.December,
	}
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
	rawHtml, err := utils.FetchRawContentFromUrl(danskeUrl, utils.DecoderUtf8, nil)
	if err != nil {
		c.logger.Error("failed reading Danske website for ListRates", zap.Error(err))
		return
	}

	listInterestSets, err := c.extractListRates(rawHtml, crawlTime)
	if err != nil {
		c.logger.Error("failed parsing Danske List Rates website", zap.Error(err))
	} else {
		interestSets = append(interestSets, listInterestSets...)
	}

	avgInterest, err := c.extractAverageRates(rawHtml, crawlTime)
	if err != nil {
		c.logger.Error("failed parsing Danske Avg Rates website", zap.Error(err))
	} else {
		interestSets = append(interestSets, avgInterest...)
	}

	for _, set := range interestSets {
		channel <- set
	}
}

func (c *DanskeBankCrawler) extractListRates(rawHtml string, crawlTime time.Time) ([]model.InterestSet, error) {
	tokenizer, err := utils.FindTokenizedTableByTextBeforeTable(rawHtml, "Bankens aktuella listräntor")
	if err != nil {
		return nil, fmt.Errorf("failed to find table by text 'Bankens aktuella' before table: %w", err)
	}

	table, err := utils.ParseTable(tokenizer)
	if err != nil {
		return nil, fmt.Errorf("failed to parse table: %w", err)
	}

	interestSets := []model.InterestSet{}
	for _, row := range table.Rows {
		term, err := utils.ParseTerm(row[0])
		if err != nil {
			c.logger.Warn("failed to parse term", zap.String("term", row[0]), zap.Error(err))
			continue
		}

		changedOn, err := parseDanskeBankChangeDate(row[1])
		if err != nil {
			c.logger.Warn("failed to parse change date", zap.String("date", row[1]), zap.Error(err))
			continue
		}

		rate, err := parseNominalRate(row[3])
		if err != nil {
			c.logger.Warn("failed to parse rate", zap.String("rate", row[3]), zap.Error(err))
			continue
		}

		interestSets = append(interestSets, model.InterestSet{
			Bank:          danskeBankName,
			Type:          model.TypeListRate,
			Term:          term,
			NominalRate:   rate,
			ChangedOn:     &changedOn,
			LastCrawledAt: crawlTime,

			RatioDiscountBoundaries: nil,
			UnionDiscount:           false,
			AverageReferenceMonth:   nil,
		})
	}

	return interestSets, nil
}

func (c *DanskeBankCrawler) extractAverageRates(rawHtml string, crawlTime time.Time) ([]model.InterestSet, error) {
	tokenizer, err := utils.FindTokenizedTableByTextBeforeTable(rawHtml, "Genomsnittlig historisk")
	if err != nil {
		return nil, fmt.Errorf("failed to find table by text 'Genomsnittlig historisk' before table: %w", err)
	}

	table, err := utils.ParseTable(tokenizer)
	if err != nil {
		return nil, fmt.Errorf("failed to parse table: %w", err)
	}

	table, err = c.sanitizeAvgRows(table)

	// parse header strings to terms
	terms := map[int]model.Term{}
	for i, t := range table.Header {
		term, err := utils.ParseTerm(t)
		if err == nil {
			terms[i] = term
		} else {
			if errors.Is(err, utils.ErrTermHeader) {
				continue // skip header
			}
			c.logger.Warn("failed to parse term", zap.String("term", t), zap.Error(err))
		}
	}

	interestSets := []model.InterestSet{}
	for _, row := range table.Rows {
		refMonth, err := c.parseReferenceMonth(row[0])
		if err != nil {
			continue
		}

		for i, cell := range row {
			if term, ok := terms[i]; ok {
				rate, err := parseNominalRate(cell)
				if err != nil {
					continue
				}

				interestSets = append(interestSets, model.InterestSet{
					Bank:                  danskeBankName,
					Type:                  model.TypeAverageRate,
					Term:                  term,
					NominalRate:           rate,
					LastCrawledAt:         crawlTime,
					AverageReferenceMonth: &refMonth,

					RatioDiscountBoundaries: nil,
					UnionDiscount:           false,
					ChangedOn:               nil,
				})
			}
		}
	}

	return interestSets, nil
}

func (c *DanskeBankCrawler) sanitizeAvgRows(table utils.Table) (utils.Table, error) {
	resultRows := [][]string{}
	for i, row := range table.Rows {
		// skip empty rows
		if len(row) == 0 {
			continue
		}

		// fix wonky (and inconsistent) Danske Bank table layout, e.g. row split like this:
		// <tr><td>Augusti 2021</td><td>1,23</td><td>1,44</td><td>1,66</td></tr>
		// <tr><td>Juli 2021</td></tr>
		// <tr><td>1,23</td><td>1,44</td><td>1,66</td></tr>
		if len(row) == 1 {
			_, err := c.parseReferenceMonth(row[0])
			if err != nil {
				// first cell contains no reference month, dunno what to do with this row, skip it
				continue
			}

			// If the current row has no values, and only the month in the first cell, then the next row
			// should contain the values for that month (because Dansek Bank doesn't know how to make html tables).
			resultRows = append(resultRows, append(row, table.Rows[i+1]...))
			table.Rows[i+1] = []string{} // clear the next row, so it doesn't get processed again
			continue
		}

		resultRows = append(resultRows, row)
	}

	return utils.Table{
		Header: table.Header,
		Rows:   resultRows,
	}, nil
}

//nolint:cyclop // complexity is ok for this simple function
func parseDanskeBankChangeDate(data string) (time.Time, error) {
	matches := changedDateRegex.FindStringSubmatch(data)
	if len(matches) != 4 {
		return time.Time{}, utils.ErrUnsupportedChangeDate
	}

	year, err := strconv.Atoi(matches[1])
	if err != nil || year < 0 {
		return time.Time{}, fmt.Errorf("failed to parse year: %w", err)
	}

	month, err := strconv.Atoi(matches[2])
	if err != nil || month < 1 || month > 12 {
		return time.Time{}, fmt.Errorf("failed to parse month: %w", err)
	}

	day, err := strconv.Atoi(matches[3])
	if err != nil || day < 1 || day > 31 {
		return time.Time{}, fmt.Errorf("failed to parse day: %w", err)
	}

	// assume all double-digit year numbers lower than 40 are from the 21st century, otherwise 20th century. This will
	// ensure that this function works until the year 2039 and assumes we don't get historical data from before 1940
	// presented in this format.
	switch true {
	case year < 40:
		year += 2000
	case year < 100:
		year += 1900
	}

	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC), nil
}

// E.g. "Augusti 2021" -> AvgMonth{Month: time.August,   Year: 2021}
// E.g. "Feb 1955"     -> AvgMonth{Month: time.February, Year: 1955}
func (c *DanskeBankCrawler) parseReferenceMonth(data string) (model.AvgMonth, error) {
	data = strings.ToLower(data)
	data = strings.TrimSpace(data)

	parts := strings.Fields(data)
	if len(parts) < 2 {
		return model.AvgMonth{}, utils.ErrUnsupportedAvgMonth
	}

	var month time.Month
	for swedishMonth, monthObject := range swedishMonthMap {
		rawMonth := utils.NormalizeSpaces(parts[0])
		if strings.Contains(rawMonth, swedishMonth) {
			month = monthObject
			break
		}
	}
	if month == 0 {
		return model.AvgMonth{}, fmt.Errorf("failed to parse month: %w", utils.ErrUnsupportedAvgMonth)
	}

	yearInt, err := strconv.Atoi(parts[1])
	if err != nil {
		return model.AvgMonth{}, fmt.Errorf("failed to parse year: %w", err)
	}
	if yearInt < 1940 || yearInt > 2100 {
		return model.AvgMonth{}, fmt.Errorf("year out of range: %w", utils.ErrUnsupportedAvgMonth)
	}

	return model.AvgMonth{
		Month: month,
		Year:  uint(yearInt),
	}, nil

}

func parseNominalRate(data string) (float32, error) {
	data = utils.NormalizeSpaces(data)
	data = strings.ReplaceAll(data, ",", ".") // replace Swedish decimal separator with dot
	data = strings.ReplaceAll(data, " ", "")  // remove spaces

	matches := interestRegex.FindStringSubmatch(data)
	if len(matches) != 2 {
		return 0, utils.ErrUnsupportedInterestRate
	}

	rate, err := strconv.ParseFloat(matches[1], 32)
	if err != nil {
		return 0, fmt.Errorf("failed to parse interest rate: %w", err)
	}

	return float32(rate), nil
}
