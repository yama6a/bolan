package crawler

import (
	"fmt"
	"io"
	"math/rand"
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
	isoDateRegex           = regexp.MustCompile(`^(\d{4})-(0[1-9]|1[0-2])-([0-2][1-9]|[1-3]0|3[01])$`)   // YYYY-MM-DD
	swedishDashedDateRegex = regexp.MustCompile(`^([0-2][1-9]|[1-3]0|3[01])-(0[1-9]|1[0-2])-(\d{2})$`)   // DD-MM-YY
	swedishDottedDateRegex = regexp.MustCompile(`^([0-2][1-9]|[1-3]0|3[01])\.(0[1-9]|1[0-2])\.(\d{2})$`) // DD.MM.YY
	yearMonthReferenceDate = regexp.MustCompile(`^(\d{2})(0[1-9]|1[0-2])$`)                              // YYMM

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

func fetchRawContentFromUrl(url string, decoder Decoder, headers map[string]string) (string, error) {
	client := http.Client{Timeout: 30 * time.Second}
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	defaultHeaders := map[string]string{
		"Accept":          "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9,application/json",
		"Accept-Language": "sv-SE,sv;q=0.5",
		"Connection":      "keep-alive",
		"User-Agent":      randomUserAgent(),
		"Cache-Control":   "no-cache",
	}
	for key, value := range defaultHeaders {
		req.Header.Set(key, value)
	}

	for key, value := range headers { // overwrites default headers if same key
		req.Header.Set(key, value)
	}

	resp, err := client.Do(req)
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
	str = strings.TrimSpace(str)
	str = strings.ReplaceAll(str, " ", "")
	str = strings.ReplaceAll(str, "*", "")
	str = strings.ReplaceAll(str, "\t", "")
	str = strings.ReplaceAll(str, "\n", "")
	str = strings.ReplaceAll(str, "\r", "")
	str = strings.ReplaceAll(str, "&nbsp;", "")
	str = strings.ReplaceAll(str, "\u00A0", "") // no-break space

	return str
}
func parseNominalRate(data string) (float32, error) {
	data = normalizeString(data)
	data = strings.ReplaceAll(data, ",", ".") // replace Swedish decimal separator with dot

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

func parseReferenceMonth(data uint, regex *regexp.Regexp) (model.AvgMonth, error) {
	matches := regex.FindStringSubmatch(fmt.Sprintf("%d", data))
	if len(matches) != 3 {
		return model.AvgMonth{}, ErrUnsupportedAvgMonth
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
		Year:  uint(year),
		Month: time.Month(month),
	}, nil
}

func parseTerm(data string) (model.Term, error) {
	str := normalizeString(data)

	switch {
	case strings.Contains(str, "3mån"), strings.Contains(str, "3mo"):
		return model.Term3months, nil
	case strings.Contains(str, "1år"), strings.Contains(str, "1yr"):
		return model.Term1year, nil
	case strings.Contains(str, "2år"), strings.Contains(str, "2yr"):
		return model.Term2years, nil
	case strings.Contains(str, "3år"), strings.Contains(str, "3yr"):
		return model.Term3years, nil
	case strings.Contains(str, "4år"), strings.Contains(str, "4yr"):
		return model.Term4years, nil
	case strings.Contains(str, "5år"), strings.Contains(str, "5yr"):
		return model.Term5years, nil
	case strings.Contains(str, "6år"), strings.Contains(str, "6yr"):
		return model.Term6years, nil
	case strings.Contains(str, "7år"), strings.Contains(str, "7yr"):
		return model.Term7years, nil
	case strings.Contains(str, "8år"), strings.Contains(str, "8yr"):
		return model.Term8years, nil
	case strings.Contains(str, "9år"), strings.Contains(str, "9yr"):
		return model.Term9years, nil
	case strings.Contains(str, "10år"), strings.Contains(str, "10yr"):
		return model.Term10years, nil

	}

	return "", ErrUnsupportedTerm
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
					default:
						continue
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

// randomUserAgent generates a random User-Agent string for different browsers and platforms.
func randomUserAgent() string {
	rand.Seed(time.Now().UnixNano())

	browsers := []string{"Chrome", "Firefox", "Safari", "Edge"}
	platforms := []string{
		"Windows NT 10.0; Win64; x64",
		"Macintosh; Intel Mac OS X %d_%d_%d",
		"iPhone; CPU iPhone OS %d_%d like Mac OS X",
		"Linux; Android %d; Pixel 3",
	}

	// Randomly select browser and platform
	browser := browsers[rand.Intn(len(browsers))]
	platformTemplate := platforms[rand.Intn(len(platforms))]

	// Generate random version numbers
	majorVersion := rand.Intn(20) + 80 // Major version for browser (e.g., Chrome 80-99)
	minorVersion := rand.Intn(10)      // Minor version
	patchVersion := rand.Intn(1000)    // Patch version

	// Generate random OS versions
	macMajor := rand.Intn(6) + 10      // macOS version 10-15
	macMinor := rand.Intn(10)          // Minor version for macOS
	macPatch := rand.Intn(10)          // Patch version for macOS
	iosMajor := rand.Intn(3) + 13      // iOS version 13-15
	iosMinor := rand.Intn(5)           // Minor version for iOS
	androidVersion := rand.Intn(6) + 7 // Android version 7-12

	// Fill in the platform template with random OS versions
	var platform string
	switch platformTemplate {
	case "Macintosh; Intel Mac OS X %d_%d_%d":
		platform = fmt.Sprintf(platformTemplate, macMajor, macMinor, macPatch)
	case "iPhone; CPU iPhone OS %d_%d like Mac OS X":
		platform = fmt.Sprintf(platformTemplate, iosMajor, iosMinor)
	case "Linux; Android %d; Pixel 3":
		platform = fmt.Sprintf(platformTemplate, androidVersion)
	default:
		platform = platformTemplate // Windows platform doesn't require formatting
	}

	// Construct User-Agent based on browser type
	var userAgent string
	switch browser {
	case "Chrome":
		userAgent = fmt.Sprintf("Mozilla/5.0 (%s) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/%d.0.%d.%d Safari/537.36", platform, majorVersion, minorVersion, patchVersion)
	case "Firefox":
		userAgent = fmt.Sprintf("Mozilla/5.0 (%s; rv:%d.0) Gecko/20100101 Firefox/%d.0", platform, majorVersion, majorVersion)
	case "Safari":
		userAgent = fmt.Sprintf("Mozilla/5.0 (%s) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/%d.0 Safari/605.1.15", platform, majorVersion)
	case "Edge":
		userAgent = fmt.Sprintf("Mozilla/5.0 (%s) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/%d.0.%d.%d Safari/537.36 Edg/%d.0.%d.%d", platform, majorVersion, minorVersion, patchVersion, majorVersion, minorVersion, patchVersion)
	}

	return userAgent
}
