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
func TestHandelsbankenCrawler_Crawl(t *testing.T) {
	t.Parallel()

	listRatesJSON := loadGoldenFile(t, "testdata/handelsbanken_list_rates.json")
	avgRatesJSON := loadGoldenFile(t, "testdata/handelsbanken_avg_rates.json")
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
				if url == handelsbankenListRateURL {
					return listRatesJSON, nil
				}
				return avgRatesJSON, nil
			},
			wantListRates:    7, // 3 mån, 1 år, 2 år, 3 år, 5 år, 8 år, 10 år
			wantAvgRates:     true,
			wantTotalMinimum: 50, // at least 50 interest sets (7 list + many average)
		},
		{
			name: "list rates fetch error still returns average rates",
			mockFetch: func(url string, _ map[string]string) (string, error) {
				if url == handelsbankenListRateURL {
					return "", errors.New("network error")
				}
				return avgRatesJSON, nil
			},
			wantListRates:    0,
			wantAvgRates:     true,
			wantTotalMinimum: 50,
		},
		{
			name: "average rates fetch error still returns list rates",
			mockFetch: func(url string, _ map[string]string) (string, error) {
				if url == handelsbankenListRateURL {
					return listRatesJSON, nil
				}
				return "", errors.New("network error")
			},
			wantListRates:    7,
			wantAvgRates:     false,
			wantTotalMinimum: 7,
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
				if url == handelsbankenListRateURL {
					return testInvalidJSON, nil
				}
				return avgRatesJSON, nil
			},
			wantListRates:    0,
			wantAvgRates:     true,
			wantTotalMinimum: 50,
		},
		{
			name: "empty response returns no results",
			mockFetch: func(url string, _ map[string]string) (string, error) {
				if url == handelsbankenListRateURL {
					return `{"interestRates":[]}`, nil
				}
				return `{"averageRatePeriods":[]}`, nil
			},
			wantListRates:    0,
			wantAvgRates:     false,
			wantTotalMinimum: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockClient := &httpmock.ClientMock{
				FetchFunc: tt.mockFetch,
			}

			crawler := NewHandelsbankenCrawler(mockClient, logger)
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

			assertBankName(t, results, handelsbankenBankName)
		})
	}
}

func TestHandelsbankenCrawler_fetchListRates(t *testing.T) {
	t.Parallel()

	listRatesJSON := loadGoldenFile(t, "testdata/handelsbanken_list_rates.json")
	logger := zap.NewNop()
	crawlTime := time.Date(2025, 12, 1, 10, 0, 0, 0, time.UTC)

	mockClient := &httpmock.ClientMock{
		FetchFunc: func(_ string, _ map[string]string) (string, error) {
			return listRatesJSON, nil
		},
	}

	crawler := &HandelsbankenCrawler{httpClient: mockClient, logger: logger}

	results, err := crawler.fetchListRates(crawlTime)
	if err != nil {
		t.Fatalf("fetchListRates() error = %v", err)
	}

	if len(results) != 7 {
		t.Errorf("fetchListRates() returned %d results, want 7", len(results))
	}

	// Verify expected terms are present (Handelsbanken has 8 år instead of 7 år)
	expectedTerms := map[model.Term]bool{
		model.Term3months: false,
		model.Term1year:   false,
		model.Term2years:  false,
		model.Term3years:  false,
		model.Term5years:  false,
		model.Term8years:  false,
		model.Term10years: false,
	}

	for _, r := range results {
		if _, ok := expectedTerms[r.Term]; ok {
			expectedTerms[r.Term] = true
		}
		assertHandelsbankenListRateFields(t, r, crawlTime)
	}

	for term, found := range expectedTerms {
		if !found {
			t.Errorf("missing term %q in results", term)
		}
	}
}

func TestHandelsbankenCrawler_fetchAverageRates(t *testing.T) {
	t.Parallel()

	avgRatesJSON := loadGoldenFile(t, "testdata/handelsbanken_avg_rates.json")
	logger := zap.NewNop()
	crawlTime := time.Date(2025, 12, 1, 10, 0, 0, 0, time.UTC)

	mockClient := &httpmock.ClientMock{
		FetchFunc: func(_ string, _ map[string]string) (string, error) {
			return avgRatesJSON, nil
		},
	}

	crawler := &HandelsbankenCrawler{httpClient: mockClient, logger: logger}

	results, err := crawler.fetchAverageRates(crawlTime)
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
		assertHandelsbankenAverageRateFields(t, r, crawlTime)
	}

	// Should have data for multiple months
	if len(months) < 10 {
		t.Errorf("got data for %d months, want at least 10", len(months))
	}
}

func TestParseHandelsbankenTerm(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		periodBasisType string
		term            string
		want            model.Term
		wantErr         bool
	}{
		{
			name:            "3 months",
			periodBasisType: "3",
			term:            "3",
			want:            model.Term3months,
			wantErr:         false,
		},
		{
			name:            "1 year",
			periodBasisType: "4",
			term:            "1",
			want:            model.Term1year,
			wantErr:         false,
		},
		{
			name:            "2 years",
			periodBasisType: "4",
			term:            "2",
			want:            model.Term2years,
			wantErr:         false,
		},
		{
			name:            "3 years",
			periodBasisType: "4",
			term:            "3",
			want:            model.Term3years,
			wantErr:         false,
		},
		{
			name:            "5 years",
			periodBasisType: "4",
			term:            "5",
			want:            model.Term5years,
			wantErr:         false,
		},
		{
			name:            "8 years",
			periodBasisType: "4",
			term:            "8",
			want:            model.Term8years,
			wantErr:         false,
		},
		{
			name:            "10 years",
			periodBasisType: "4",
			term:            "10",
			want:            model.Term10years,
			wantErr:         false,
		},
		{
			name:            "unsupported month term",
			periodBasisType: "3",
			term:            "6",
			wantErr:         true,
		},
		{
			name:            "unsupported year term",
			periodBasisType: "4",
			term:            "11",
			wantErr:         true,
		},
		{
			name:            "unsupported periodBasisType",
			periodBasisType: "5",
			term:            "1",
			wantErr:         true,
		},
		{
			name:            "invalid term number",
			periodBasisType: "4",
			term:            "abc",
			wantErr:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := parseHandelsbankenTerm(tt.periodBasisType, tt.term)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseHandelsbankenTerm() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("parseHandelsbankenTerm() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseHandelsbankenPeriod(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     string
		wantYear  uint
		wantMonth time.Month
		wantErr   bool
	}{
		{
			name:      "valid period December 2024",
			input:     "202412",
			wantYear:  2024,
			wantMonth: time.December,
			wantErr:   false,
		},
		{
			name:      "valid period January 2025",
			input:     "202501",
			wantYear:  2025,
			wantMonth: time.January,
			wantErr:   false,
		},
		{
			name:      "valid period October 2025",
			input:     "202510",
			wantYear:  2025,
			wantMonth: time.October,
			wantErr:   false,
		},
		{
			name:    "invalid format - too short",
			input:   "20251",
			wantErr: true,
		},
		{
			name:    "invalid format - too long",
			input:   "2025101",
			wantErr: true,
		},
		{
			name:    "invalid month 00",
			input:   "202500",
			wantErr: true,
		},
		{
			name:    "invalid month 13",
			input:   "202513",
			wantErr: true,
		},
		{
			name:    "invalid - not a number",
			input:   "abcdef",
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

			got, err := parseHandelsbankenPeriod(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseHandelsbankenPeriod() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got.Year != tt.wantYear {
					t.Errorf("parseHandelsbankenPeriod() year = %v, want %v", got.Year, tt.wantYear)
				}
				if got.Month != tt.wantMonth {
					t.Errorf("parseHandelsbankenPeriod() month = %v, want %v", got.Month, tt.wantMonth)
				}
			}
		})
	}
}

func TestNewHandelsbankenCrawler(t *testing.T) {
	t.Parallel()

	mockClient := &httpmock.ClientMock{}
	logger := zap.NewNop()

	crawler := NewHandelsbankenCrawler(mockClient, logger)

	if crawler == nil {
		t.Fatal("NewHandelsbankenCrawler() returned nil")
	}
	if crawler.httpClient != mockClient {
		t.Error("httpClient not set correctly")
	}
	if crawler.logger != logger {
		t.Error("logger not set correctly")
	}
}

func TestHandelsbankenCrawler_InterfaceCompliance(t *testing.T) {
	t.Parallel()

	// Compile-time check that HandelsbankenCrawler implements SiteCrawler
	var _ SiteCrawler = &HandelsbankenCrawler{}
}

// assertHandelsbankenListRateFields validates common fields for Handelsbanken list rate results.
func assertHandelsbankenListRateFields(t *testing.T, r model.InterestSet, crawlTime time.Time) {
	t.Helper()

	if r.Bank != handelsbankenBankName {
		t.Errorf("Bank = %q, want %q", r.Bank, handelsbankenBankName)
	}
	if r.Type != model.TypeListRate {
		t.Errorf("Type = %q, want %q", r.Type, model.TypeListRate)
	}
	if r.NominalRate <= 0 {
		t.Errorf("NominalRate = %f, want positive value", r.NominalRate)
	}
	// Handelsbanken API doesn't provide change dates
	if r.ChangedOn != nil {
		t.Error("ChangedOn is not nil, want nil for Handelsbanken")
	}
	if r.LastCrawledAt != crawlTime {
		t.Errorf("LastCrawledAt = %v, want %v", r.LastCrawledAt, crawlTime)
	}
}

// assertHandelsbankenAverageRateFields validates common fields for Handelsbanken average rate results.
func assertHandelsbankenAverageRateFields(t *testing.T, r model.InterestSet, crawlTime time.Time) {
	t.Helper()

	if r.Bank != handelsbankenBankName {
		t.Errorf("Bank = %q, want %q", r.Bank, handelsbankenBankName)
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
