//nolint:revive,nolintlint,dupl // package name matches the package being tested; test patterns intentionally similar across crawlers
package crawler

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/yama6a/bolan-compare/internal/pkg/http/httpmock"
	"github.com/yama6a/bolan-compare/internal/pkg/model"
	"go.uber.org/zap"
)

const sebTestHTMLWithJSRef = `<html><script src="main.test123.js"></script></html>`

// assertSEBListRateFields validates common fields for SEB list rate results.
func assertSEBListRateFields(t *testing.T, r model.InterestSet, crawlTime time.Time) {
	t.Helper()

	if r.Bank != sebBankName {
		t.Errorf("Bank = %q, want %q", r.Bank, sebBankName)
	}
	if r.Type != model.TypeListRate {
		t.Errorf("Type = %q, want %q", r.Type, model.TypeListRate)
	}
	if r.NominalRate <= 0 {
		t.Errorf("NominalRate = %f, want positive value", r.NominalRate)
	}
	if r.ChangedOn == nil {
		t.Error("ChangedOn is nil, want non-nil")
	}
	if r.LastCrawledAt != crawlTime {
		t.Errorf("LastCrawledAt = %v, want %v", r.LastCrawledAt, crawlTime)
	}
}

// assertSEBAverageRateFields validates common fields for SEB average rate results.
func assertSEBAverageRateFields(t *testing.T, r model.InterestSet, crawlTime time.Time) {
	t.Helper()

	if r.Bank != sebBankName {
		t.Errorf("Bank = %q, want %q", r.Bank, sebBankName)
	}
	if r.Type != model.TypeAverageRate {
		t.Errorf("Type = %q, want %q", r.Type, model.TypeAverageRate)
	}
	if r.NominalRate <= 0 {
		t.Errorf("NominalRate = %f, want positive value", r.NominalRate)
	}
	if r.AverageReferenceMonth == nil {
		t.Error("AverageReferenceMonth is nil, want non-nil")
	}
	if r.LastCrawledAt != crawlTime {
		t.Errorf("LastCrawledAt = %v, want %v", r.LastCrawledAt, crawlTime)
	}
}

// createSEBMockFetch creates a mock fetch function that routes requests based on URL.
func createSEBMockFetch(t *testing.T) func(url string, headers map[string]string) (string, error) {
	t.Helper()

	pageHTML := loadGoldenFile(t, "testdata/seb_page.html")
	mainJS := loadGoldenFile(t, "testdata/seb_main.js")
	listRatesJSON := loadGoldenFile(t, "testdata/seb_list_rates.json")
	avgRatesJSON := loadGoldenFile(t, "testdata/seb_avg_rates.json")

	return func(url string, headers map[string]string) (string, error) {
		switch {
		case url == sebAvgCurrentHTMLURL:
			return pageHTML, nil
		case strings.HasPrefix(url, sebAPIKeyJsFileURLPrefix) && strings.HasSuffix(url, ".js"):
			return mainJS, nil
		case url == sebListRateURL:
			// Verify API key header is present
			if headers["X-API-Key"] == "" {
				return "", errors.New("missing X-API-Key header")
			}
			return listRatesJSON, nil
		case url == sebAverageRatesURL:
			// Verify API key header is present
			if headers["X-API-Key"] == "" {
				return "", errors.New("missing X-API-Key header")
			}
			return avgRatesJSON, nil
		default:
			return "", errors.New("unexpected URL: " + url)
		}
	}
}

//nolint:cyclop // table-driven test with multiple cases
func TestSebBankCrawler_Crawl(t *testing.T) {
	t.Parallel()

	logger := zap.NewNop()

	tests := []struct {
		name             string
		mockFetch        func(url string, headers map[string]string) (string, error)
		wantListRates    int
		wantAvgRates     bool
		wantTotalMinimum int
	}{
		{
			name:             "successful crawl extracts list and average rates",
			mockFetch:        createSEBMockFetch(t),
			wantListRates:    7, // 3mo, 1yr, 2yr, 3yr, 5yr, 7yr, 10yr
			wantAvgRates:     true,
			wantTotalMinimum: 30, // 7 list rates + many average rates
		},
		{
			name: "API key fetch failure returns no results",
			mockFetch: func(_ string, _ map[string]string) (string, error) {
				return "", errors.New("network error")
			},
			wantListRates:    0,
			wantAvgRates:     false,
			wantTotalMinimum: 0,
		},
		{
			name: "JS file not found returns no results",
			mockFetch: func(url string, _ map[string]string) (string, error) {
				if url == sebAvgCurrentHTMLURL {
					return "<html><body>No JS reference</body></html>", nil
				}
				return "", errors.New("not found")
			},
			wantListRates:    0,
			wantAvgRates:     false,
			wantTotalMinimum: 0,
		},
		{
			name: "API key not in JS file returns no results",
			mockFetch: func(url string, _ map[string]string) (string, error) {
				if url == sebAvgCurrentHTMLURL {
					return `<html><script src="main.abc123.js"></script></html>`, nil
				}
				if strings.HasSuffix(url, ".js") {
					return "// no api key here", nil
				}
				return "", errors.New("not found")
			},
			wantListRates:    0,
			wantAvgRates:     false,
			wantTotalMinimum: 0,
		},
		{
			name: "list rates API failure still returns average rates",
			mockFetch: func(url string, headers map[string]string) (string, error) {
				pageHTML := loadGoldenFile(t, "testdata/seb_page.html")
				mainJS := loadGoldenFile(t, "testdata/seb_main.js")
				avgRatesJSON := loadGoldenFile(t, "testdata/seb_avg_rates.json")

				switch {
				case url == sebAvgCurrentHTMLURL:
					return pageHTML, nil
				case strings.HasSuffix(url, ".js"):
					return mainJS, nil
				case url == sebListRateURL:
					return "", errors.New("list rates API error")
				case url == sebAverageRatesURL:
					return avgRatesJSON, nil
				default:
					return "", errors.New("unexpected URL")
				}
			},
			wantListRates:    0,
			wantAvgRates:     true,
			wantTotalMinimum: 25, // only average rates
		},
		{
			name: "average rates API failure still returns list rates",
			mockFetch: func(url string, headers map[string]string) (string, error) {
				pageHTML := loadGoldenFile(t, "testdata/seb_page.html")
				mainJS := loadGoldenFile(t, "testdata/seb_main.js")
				listRatesJSON := loadGoldenFile(t, "testdata/seb_list_rates.json")

				switch {
				case url == sebAvgCurrentHTMLURL:
					return pageHTML, nil
				case strings.HasSuffix(url, ".js"):
					return mainJS, nil
				case url == sebListRateURL:
					return listRatesJSON, nil
				case url == sebAverageRatesURL:
					return "", errors.New("average rates API error")
				default:
					return "", errors.New("unexpected URL")
				}
			},
			wantListRates:    7,
			wantAvgRates:     false,
			wantTotalMinimum: 7,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockClient := &httpmock.ClientMock{
				FetchFunc: tt.mockFetch,
			}

			crawler := NewSebBankCrawler(mockClient, logger)
			resultChan := make(chan model.InterestSet, 200)

			crawler.Crawl(resultChan)
			close(resultChan)

			var results []model.InterestSet
			for set := range resultChan {
				results = append(results, set)
			}

			listRateCount, avgRateCount := countRatesByType(results)

			if listRateCount != tt.wantListRates {
				t.Errorf("list rate count = %d, want %d", listRateCount, tt.wantListRates)
			}

			if tt.wantAvgRates && avgRateCount == 0 {
				t.Error("expected average rates but got none")
			}

			if len(results) < tt.wantTotalMinimum {
				t.Errorf("total results = %d, want at least %d", len(results), tt.wantTotalMinimum)
			}

			assertBankName(t, results, sebBankName)
		})
	}
}

//nolint:cyclop // table-driven test with multiple cases
func TestSebBankCrawler_fetchAPIKey(t *testing.T) {
	t.Parallel()

	logger := zap.NewNop()

	tests := []struct {
		name      string
		mockFetch func(url string, headers map[string]string) (string, error)
		wantKey   string
		wantErr   bool
	}{
		{
			name: "successful API key extraction",
			mockFetch: func(url string, _ map[string]string) (string, error) {
				if url == sebAvgCurrentHTMLURL {
					return sebTestHTMLWithJSRef, nil
				}
				if strings.HasSuffix(url, ".js") {
					return `const config = {"x-api-key":"my-secret-key-123"};`, nil
				}
				return "", errors.New("unexpected URL")
			},
			wantKey: "my-secret-key-123",
			wantErr: false,
		},
		{
			name: "HTML fetch error",
			mockFetch: func(_ string, _ map[string]string) (string, error) {
				return "", errors.New("network error")
			},
			wantKey: "",
			wantErr: true,
		},
		{
			name: "JS filename not found in HTML",
			mockFetch: func(url string, _ map[string]string) (string, error) {
				if url == sebAvgCurrentHTMLURL {
					return `<html><body>No script tag</body></html>`, nil
				}
				return "", errors.New("should not be called")
			},
			wantKey: "",
			wantErr: true,
		},
		{
			name: "JS file fetch error",
			mockFetch: func(url string, _ map[string]string) (string, error) {
				if url == sebAvgCurrentHTMLURL {
					return sebTestHTMLWithJSRef, nil
				}
				return "", errors.New("JS fetch failed")
			},
			wantKey: "",
			wantErr: true,
		},
		{
			name: "API key not found in JS",
			mockFetch: func(url string, _ map[string]string) (string, error) {
				if url == sebAvgCurrentHTMLURL {
					return sebTestHTMLWithJSRef, nil
				}
				if strings.HasSuffix(url, ".js") {
					return `const config = {"other": "value"};`, nil
				}
				return "", errors.New("unexpected URL")
			},
			wantKey: "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockClient := &httpmock.ClientMock{
				FetchFunc: tt.mockFetch,
			}

			crawler := NewSebBankCrawler(mockClient, logger)

			got, err := crawler.fetchAPIKey()
			if (err != nil) != tt.wantErr {
				t.Errorf("fetchAPIKey() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.wantKey {
				t.Errorf("fetchAPIKey() = %v, want %v", got, tt.wantKey)
			}
		})
	}
}

func TestSebBankCrawler_fetchListRates(t *testing.T) {
	t.Parallel()

	listRatesJSON := loadGoldenFile(t, "testdata/seb_list_rates.json")
	logger := zap.NewNop()
	crawlTime := time.Date(2025, 12, 1, 10, 0, 0, 0, time.UTC)

	mockClient := &httpmock.ClientMock{
		FetchFunc: func(_ string, _ map[string]string) (string, error) {
			return listRatesJSON, nil
		},
	}

	crawler := NewSebBankCrawler(mockClient, logger)
	results, err := crawler.fetchListRates("test-api-key", crawlTime)
	if err != nil {
		t.Fatalf("fetchListRates() error = %v", err)
	}

	if len(results) != 7 {
		t.Errorf("fetchListRates() returned %d results, want 7", len(results))
	}

	// Verify expected terms are present
	expectedTerms := map[model.Term]bool{
		model.Term3months: false,
		model.Term1year:   false,
		model.Term2years:  false,
		model.Term3years:  false,
		model.Term5years:  false,
		model.Term7years:  false,
		model.Term10years: false,
	}

	for _, r := range results {
		if _, ok := expectedTerms[r.Term]; ok {
			expectedTerms[r.Term] = true
		}
		assertSEBListRateFields(t, r, crawlTime)
	}

	for term, found := range expectedTerms {
		if !found {
			t.Errorf("missing term %q in results", term)
		}
	}
}

func TestSebBankCrawler_fetchAverageRates(t *testing.T) {
	t.Parallel()

	avgRatesJSON := loadGoldenFile(t, "testdata/seb_avg_rates.json")
	logger := zap.NewNop()
	crawlTime := time.Date(2025, 12, 1, 10, 0, 0, 0, time.UTC)

	mockClient := &httpmock.ClientMock{
		FetchFunc: func(_ string, _ map[string]string) (string, error) {
			return avgRatesJSON, nil
		},
	}

	crawler := NewSebBankCrawler(mockClient, logger)
	results, err := crawler.fetchAverageRates("test-api-key", crawlTime)
	if err != nil {
		t.Fatalf("fetchAverageRates() error = %v", err)
	}

	if len(results) == 0 {
		t.Fatal("fetchAverageRates() returned no results")
	}

	// Verify we got multiple months of data
	months := make(map[string]bool)
	for _, r := range results {
		if r.AverageReferenceMonth != nil {
			key := r.AverageReferenceMonth.Month.String() + string(rune(r.AverageReferenceMonth.Year))
			months[key] = true
		}
		assertSEBAverageRateFields(t, r, crawlTime)
	}

	// Should have data for multiple months (our test data has 5 periods)
	if len(months) < 4 {
		t.Errorf("got data for %d months, want at least 4", len(months))
	}
}

func TestParseReferenceMonth(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     uint
		wantMonth time.Month
		wantYear  uint
		wantErr   bool
	}{
		{
			name:      "October 2025",
			input:     2510,
			wantMonth: time.October,
			wantYear:  2025,
			wantErr:   false,
		},
		{
			name:      "November 2024",
			input:     2411,
			wantMonth: time.November,
			wantYear:  2024,
			wantErr:   false,
		},
		{
			name:      "January 2025",
			input:     2501,
			wantMonth: time.January,
			wantYear:  2025,
			wantErr:   false,
		},
		{
			name:      "December 2039 (last valid 21st century)",
			input:     3912,
			wantMonth: time.December,
			wantYear:  2039,
			wantErr:   false,
		},
		{
			name:      "January 1940 (first 20th century)",
			input:     4001,
			wantMonth: time.January,
			wantYear:  1940,
			wantErr:   false,
		},
		{
			name:    "invalid format - too short",
			input:   251,
			wantErr: true,
		},
		{
			name:    "invalid format - too long",
			input:   25101,
			wantErr: true,
		},
		{
			name:    "invalid month - 00",
			input:   2500,
			wantErr: true,
		},
		{
			name:    "invalid month - 13",
			input:   2513,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := parseReferenceMonth(tt.input, yearMonthReferenceDate)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseReferenceMonth() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got.Month != tt.wantMonth {
					t.Errorf("parseReferenceMonth() month = %v, want %v", got.Month, tt.wantMonth)
				}
				if got.Year != tt.wantYear {
					t.Errorf("parseReferenceMonth() year = %v, want %v", got.Year, tt.wantYear)
				}
			}
		})
	}
}

func TestParseSEBChangeDate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    time.Time
		wantErr bool
	}{
		{
			name:    "ISO format with timezone",
			input:   "2025-09-25T04:00:00Z",
			want:    time.Date(2025, time.September, 25, 0, 0, 0, 0, time.UTC),
			wantErr: false,
		},
		{
			name:    "ISO format date only",
			input:   "2025-07-10",
			want:    time.Date(2025, time.July, 10, 0, 0, 0, 0, time.UTC),
			wantErr: false,
		},
		{
			name:    "with extra whitespace",
			input:   "  2025-03-26T05:00:00Z  ",
			want:    time.Date(2025, time.March, 26, 0, 0, 0, 0, time.UTC),
			wantErr: false,
		},
		{
			name:    "invalid format",
			input:   "25-09-2025",
			wantErr: true,
		},
		{
			name:    "invalid month",
			input:   "2025-13-01",
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := parseSEBChangeDate(tt.input, isoDateRegex)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseSEBChangeDate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !got.Equal(tt.want) {
				t.Errorf("parseSEBChangeDate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewSebBankCrawler(t *testing.T) {
	t.Parallel()

	mockClient := &httpmock.ClientMock{}
	logger := zap.NewNop()

	crawler := NewSebBankCrawler(mockClient, logger)

	if crawler == nil {
		t.Fatal("NewSebBankCrawler() returned nil")
	}
	if crawler.httpClient != mockClient {
		t.Error("httpClient not set correctly")
	}
	if crawler.logger != logger {
		t.Error("logger not set correctly")
	}
}

func TestSebBankCrawler_InterfaceCompliance(t *testing.T) {
	t.Parallel()

	// Compile-time check that SebBankCrawler implements SiteCrawler
	var _ SiteCrawler = &SebBankCrawler{}
}
