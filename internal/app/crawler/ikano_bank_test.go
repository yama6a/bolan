//nolint:revive,nolintlint,dupl // package name matches the package being tested; test patterns intentionally similar across crawlers
package crawler

import (
	"errors"
	"testing"
	"time"

	"github.com/yama6a/bolan-compare/internal/pkg/http/httpmock"
	"github.com/yama6a/bolan-compare/internal/pkg/model"
	"go.uber.org/zap"
)

//nolint:cyclop // table-driven test with multiple cases
func TestIkanoBankCrawler_Crawl(t *testing.T) {
	t.Parallel()

	listRatesJSON := loadGoldenFile(t, "testdata/ikano_bank_list_rates.json")
	avgRatesHTML := loadGoldenFile(t, "testdata/ikano_bank_avg_rates.html")
	logger := zap.NewNop()

	tests := []struct {
		name             string
		mockFetch        func(url string, headers map[string]string) (string, error)
		wantListRates    int
		wantAvgRates     bool
		wantTotalMinimum int
	}{
		{
			name: "successful crawl extracts list and average rates",
			mockFetch: func(url string, _ map[string]string) (string, error) {
				if url == ikanoBankListRateURL {
					return listRatesJSON, nil
				}
				return avgRatesHTML, nil
			},
			wantListRates:    8, // 3 mån, 1 år, 2 år, 3 år, 4 år, 5 år, 7 år, 10 år
			wantAvgRates:     true,
			wantTotalMinimum: 30, // at least 30 interest sets (8 list + many average)
		},
		{
			name: "list rates fetch error still returns average rates",
			mockFetch: func(url string, _ map[string]string) (string, error) {
				if url == ikanoBankListRateURL {
					return "", errors.New("network error")
				}
				return avgRatesHTML, nil
			},
			wantListRates:    0,
			wantAvgRates:     true,
			wantTotalMinimum: 20,
		},
		{
			name: "average rates fetch error still returns list rates",
			mockFetch: func(url string, _ map[string]string) (string, error) {
				if url == ikanoBankListRateURL {
					return listRatesJSON, nil
				}
				return "", errors.New("network error")
			},
			wantListRates:    8,
			wantAvgRates:     false,
			wantTotalMinimum: 8,
		},
		{
			name: "both fetch errors return no results",
			mockFetch: func(_ string, _ map[string]string) (string, error) {
				return "", errors.New("network error")
			},
			wantListRates:    0,
			wantAvgRates:     false,
			wantTotalMinimum: 0,
		},
		{
			name: "invalid JSON returns no list rates",
			mockFetch: func(url string, _ map[string]string) (string, error) {
				if url == ikanoBankListRateURL {
					return testInvalidJSON, nil
				}
				return avgRatesHTML, nil
			},
			wantListRates:    0,
			wantAvgRates:     true,
			wantTotalMinimum: 20,
		},
		{
			name: "API success=false returns no list rates",
			mockFetch: func(url string, _ map[string]string) (string, error) {
				if url == ikanoBankListRateURL {
					return `{"success":false,"listData":[]}`, nil
				}
				return avgRatesHTML, nil
			},
			wantListRates:    0,
			wantAvgRates:     true,
			wantTotalMinimum: 20,
		},
		{
			name: "empty list data returns no list rates",
			mockFetch: func(url string, _ map[string]string) (string, error) {
				if url == ikanoBankListRateURL {
					return `{"success":true,"listData":[]}`, nil
				}
				return avgRatesHTML, nil
			},
			wantListRates:    0,
			wantAvgRates:     true,
			wantTotalMinimum: 20,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockClient := &httpmock.ClientMock{
				FetchFunc: tt.mockFetch,
			}

			crawler := NewIkanoBankCrawler(mockClient, logger)
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

			if !tt.wantAvgRates && avgRateCount > 0 {
				t.Errorf("expected no average rates but got %d", avgRateCount)
			}

			if len(results) < tt.wantTotalMinimum {
				t.Errorf("total results = %d, want at least %d", len(results), tt.wantTotalMinimum)
			}

			assertBankName(t, results, ikanoBankName)
		})
	}
}

func TestIkanoBankCrawler_fetchListRates(t *testing.T) {
	t.Parallel()

	listRatesJSON := loadGoldenFile(t, "testdata/ikano_bank_list_rates.json")
	logger := zap.NewNop()
	crawlTime := time.Date(2025, 12, 1, 10, 0, 0, 0, time.UTC)

	mockClient := &httpmock.ClientMock{
		FetchFunc: func(_ string, _ map[string]string) (string, error) {
			return listRatesJSON, nil
		},
	}

	crawler := &IkanoBankCrawler{httpClient: mockClient, logger: logger}

	results, err := crawler.fetchListRates(crawlTime)
	if err != nil {
		t.Fatalf("fetchListRates() error = %v", err)
	}

	if len(results) != 8 {
		t.Errorf("fetchListRates() returned %d results, want 8", len(results))
	}

	// Verify expected terms are present.
	expectedTerms := map[model.Term]bool{
		model.Term3months: false,
		model.Term1year:   false,
		model.Term2years:  false,
		model.Term3years:  false,
		model.Term4years:  false,
		model.Term5years:  false,
		model.Term7years:  false,
		model.Term10years: false,
	}

	for _, r := range results {
		if _, ok := expectedTerms[r.Term]; ok {
			expectedTerms[r.Term] = true
		}
		assertIkanoBankListRateFields(t, r, crawlTime)
	}

	for term, found := range expectedTerms {
		if !found {
			t.Errorf("missing term %q in results", term)
		}
	}
}

func TestIkanoBankCrawler_fetchAverageRates(t *testing.T) {
	t.Parallel()

	avgRatesHTML := loadGoldenFile(t, "testdata/ikano_bank_avg_rates.html")
	logger := zap.NewNop()
	crawlTime := time.Date(2025, 12, 1, 10, 0, 0, 0, time.UTC)

	mockClient := &httpmock.ClientMock{
		FetchFunc: func(_ string, _ map[string]string) (string, error) {
			return avgRatesHTML, nil
		},
	}

	crawler := &IkanoBankCrawler{httpClient: mockClient, logger: logger}

	results, err := crawler.fetchAverageRates(crawlTime)
	if err != nil {
		t.Fatalf("fetchAverageRates() error = %v", err)
	}

	if len(results) == 0 {
		t.Fatal("fetchAverageRates() returned no results")
	}

	// Verify we got multiple months of data.
	months := make(map[string]bool)
	for _, r := range results {
		if r.AverageReferenceMonth != nil {
			key := r.AverageReferenceMonth.Month.String() + string(rune(r.AverageReferenceMonth.Year))
			months[key] = true
		}
		assertIkanoBankAverageRateFields(t, r, crawlTime)
	}

	// Should have data for multiple months.
	if len(months) < 10 {
		t.Errorf("got data for %d months, want at least 10", len(months))
	}
}

func TestParseIkanoBankRate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    float32
		wantErr bool
	}{
		{
			name:    "API format - 3.4800",
			input:   "3.4800",
			want:    3.48,
			wantErr: false,
		},
		{
			name:    "HTML format with comma - 3,61 %",
			input:   "3,61 %",
			want:    3.61,
			wantErr: false,
		},
		{
			name:    "HTML format without space - 3,61%",
			input:   "3,61%",
			want:    3.61,
			wantErr: false,
		},
		{
			name:    "simple decimal - 2.5",
			input:   "2.5",
			want:    2.5,
			wantErr: false,
		},
		{
			name:    "invalid - not a number",
			input:   "abc",
			wantErr: true,
		},
		{
			name:    "invalid - empty string",
			input:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := parseIkanoBankRate(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseIkanoBankRate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("parseIkanoBankRate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseIkanoBankAvgMonth(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     string
		wantYear  uint
		wantMonth time.Month
		wantErr   bool
	}{
		{
			name:      "valid - November 2024",
			input:     "2024 11",
			wantYear:  2024,
			wantMonth: time.November,
			wantErr:   false,
		},
		{
			name:      "valid - January 2025",
			input:     "2025 01",
			wantYear:  2025,
			wantMonth: time.January,
			wantErr:   false,
		},
		{
			name:      "valid - December 2024",
			input:     "2024 12",
			wantYear:  2024,
			wantMonth: time.December,
			wantErr:   false,
		},
		{
			name:    "invalid - single digit month",
			input:   "2024 1",
			wantErr: true,
		},
		{
			name:    "invalid - wrong format",
			input:   "2024-01",
			wantErr: true,
		},
		{
			name:    "invalid - not a number",
			input:   "abc",
			wantErr: true,
		},
		{
			name:    "invalid - empty string",
			input:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := parseIkanoBankAvgMonth(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseIkanoBankAvgMonth() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got.Year != tt.wantYear {
					t.Errorf("parseIkanoBankAvgMonth() year = %v, want %v", got.Year, tt.wantYear)
				}
				if got.Month != tt.wantMonth {
					t.Errorf("parseIkanoBankAvgMonth() month = %v, want %v", got.Month, tt.wantMonth)
				}
			}
		})
	}
}

func TestNewIkanoBankCrawler(t *testing.T) {
	t.Parallel()

	mockClient := &httpmock.ClientMock{}
	logger := zap.NewNop()

	crawler := NewIkanoBankCrawler(mockClient, logger)

	if crawler == nil {
		t.Fatal("NewIkanoBankCrawler() returned nil")
	}
	if crawler.httpClient != mockClient {
		t.Error("httpClient not set correctly")
	}
	if crawler.logger != logger {
		t.Error("logger not set correctly")
	}
}

func TestIkanoBankCrawler_InterfaceCompliance(t *testing.T) {
	t.Parallel()

	// Compile-time check that IkanoBankCrawler implements SiteCrawler.
	var _ SiteCrawler = &IkanoBankCrawler{}
}

// assertIkanoBankListRateFields validates common fields for Ikano Bank list rate results.
func assertIkanoBankListRateFields(t *testing.T, r model.InterestSet, crawlTime time.Time) {
	t.Helper()

	if r.Bank != ikanoBankName {
		t.Errorf("Bank = %q, want %q", r.Bank, ikanoBankName)
	}
	if r.Type != model.TypeListRate {
		t.Errorf("Type = %q, want %q", r.Type, model.TypeListRate)
	}
	if r.NominalRate <= 0 {
		t.Errorf("NominalRate = %f, want positive value", r.NominalRate)
	}
	if r.LastCrawledAt != crawlTime {
		t.Errorf("LastCrawledAt = %v, want %v", r.LastCrawledAt, crawlTime)
	}
}

// assertIkanoBankAverageRateFields validates common fields for Ikano Bank average rate results.
func assertIkanoBankAverageRateFields(t *testing.T, r model.InterestSet, crawlTime time.Time) {
	t.Helper()

	if r.Bank != ikanoBankName {
		t.Errorf("Bank = %q, want %q", r.Bank, ikanoBankName)
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
