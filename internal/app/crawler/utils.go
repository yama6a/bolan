package crawler

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ymakhloufi/bolan-compare/internal/pkg/model"
	"golang.org/x/net/html"
)

var (
	interestRegex          = regexp.MustCompile(`^(\d+\.\d+ ?)%$`)
	isoDateRegex           = regexp.MustCompile(`^(\d{4})-([0][1-9]|1[0-2])-([0-2][1-9]|[1-3]0|3[01])$`)   // YYYY-MM-DD
	swedishDashedDateRegex = regexp.MustCompile(`^([0-2][1-9]|[1-3]0|3[01])-([0][1-9]|1[0-2])-(\d{2})$`)   // DD-MM-YY
	swedishDottedDateRegex = regexp.MustCompile(`^([0-2][1-9]|[1-3]0|3[01])\.([0][1-9]|1[0-2])\.(\d{2})$`) // DD.MM.YY

	DecoderWindows1252 Decoder = func(runes []byte) (str string) {
		for _, r := range runes {
			str += string(r)
		}
		return
	}
	DecoderUtf8 Decoder = func(runes []byte) string {
		return string(runes)
	}
)

type Decoder func([]byte) string

func fetchHtmlFromUrl(url string, decoder Decoder) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return decoder(body), nil
}

func normalizeString(str string) string {
	str = strings.Replace(str, " ", "", -1)
	str = strings.Replace(str, "*", "", -1)
	str = strings.Replace(str, "\n", "", -1)
	str = strings.Replace(str, "\r", "", -1)
	str = strings.Replace(str, "&nbsp;", "", -1)
	str = strings.Replace(str, "\u00A0", "", -1) // no-break space

	return str
}
func parseNominalRate(data string) (float32, error) {
	data = normalizeString(data)
	data = strings.Replace(data, ",", ".", -1) // replace Swedish decimal separator with dot

	matches := interestRegex.FindStringSubmatch(data)
	if len(matches) != 2 {
		return 0, ErrUnsupportedInterestRate
	}

	rate, err := strconv.ParseFloat(matches[1], 32)
	if err != nil {
		return 0, fmt.Errorf("failed to parse interest rate: %w", err)
	}

	return float32(rate), nil
}

func parseTerm(data string) (model.Term, error) {
	str := normalizeString(data)

	println(str)
	switch {
	case strings.Contains(str, "3mån"):
		return model.Term3months, nil
	case strings.Contains(str, "1år"):
		return model.Term1year, nil
	case strings.Contains(str, "2år"):
		return model.Term2years, nil
	case strings.Contains(str, "3år"):
		return model.Term3years, nil
	case strings.Contains(str, "4år"):
		return model.Term4years, nil
	case strings.Contains(str, "5år"):
		return model.Term5years, nil
	case strings.Contains(str, "6år"):
		return model.Term6years, nil
	case strings.Contains(str, "7år"):
		return model.Term7years, nil
	case strings.Contains(str, "8år"):
		return model.Term8years, nil
	case strings.Contains(str, "9år"):
		return model.Term9years, nil
	case strings.Contains(str, "10år"):
		return model.Term10years, nil
	default:
		return "", ErrUnsupportedTerm
	}
}

func parseChangeDate(str string, regex *regexp.Regexp) (time.Time, error) {
	str = normalizeString(str)

	matches := regex.FindStringSubmatch(str)
	if len(matches) != 4 {
		return time.Time{}, ErrUnsupportedChangeDate
	}

	date, err := time.Parse("2006-01-02", matches[0])
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse change date: %w", err)
	}

	return date, nil
}

// findTokenizedTableByTextBeforeTable returns the tokenizer for a table which occurs after a given text in an xml document
func findTokenizedTableByTextBeforeTable(rawHtml, stringToFind string) (*html.Tokenizer, error) {
	tokenizer := html.NewTokenizer(strings.NewReader(rawHtml))
	for node := tokenizer.Next(); node != html.ErrorToken; node = tokenizer.Next() {
		if node == html.TextToken {
			tkn := tokenizer.Token()
			if strings.Contains(tkn.Data, stringToFind) {
				for {
					node = tokenizer.Next()
					switch node {
					case html.ErrorToken:
						if tokenizer.Err() == io.EOF {
							return nil, fmt.Errorf("failed finding section on Danske List Rates website, EOF: %w", ErrNoInterestSetFound)
						}
						return nil, tokenizer.Err()
					case html.StartTagToken:
						tkn := tokenizer.Token()
						if tkn.Data == "table" {
							return tokenizer, nil
						}
					}
				}
			}
		}
	}
	if tokenizer.Err() == io.EOF {
		return nil, fmt.Errorf("failed finding section on Danske Bank website, EOF: %w", ErrNoInterestSetFound)
	}
	return nil, tokenizer.Err()
}
