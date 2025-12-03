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

// assertICAListRateFields validates common fields for ICA Banken list rate results.
func assertICAListRateFields(t *testing.T, r model.InterestSet, crawlTime time.Time) {
	t.Helper()

	if r.Bank != icaBankenName {
		t.Errorf("Bank = %q, want %q", r.Bank, icaBankenName)
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

// assertICAAverageRateFields validates common fields for ICA Banken average rate results.
func assertICAAverageRateFields(t *testing.T, r model.InterestSet, crawlTime time.Time) {
	t.Helper()

	if r.Bank != icaBankenName {
		t.Errorf("Bank = %q, want %q", r.Bank, icaBankenName)
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

func TestICABankenCrawler_Crawl(t *testing.T) {
	t.Parallel()

	goldenHTML := loadGoldenFile(t, "testdata/ica_banken.html")
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
			mockFetch: func(_ string, _ map[string]string) (string, error) {
				return goldenHTML, nil
			},
			wantListRates:    8, // 3 mån, 1 år, 2 år, 3 år, 4 år, 5 år, 7 år, 10 år
			wantAvgRates:     true,
			wantTotalMinimum: 30, // at least 30 interest sets (8 list + average rates)
		},
		{
			name: "fetch error returns no results",
			mockFetch: func(_ string, _ map[string]string) (string, error) {
				return "", errors.New("network error")
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

			crawler := NewICABankenCrawler(mockClient, logger)
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

			assertBankName(t, results, icaBankenName)
		})
	}
}

func TestICABankenCrawler_extractListRates(t *testing.T) {
	t.Parallel()

	goldenHTML := loadGoldenFile(t, "testdata/ica_banken.html")
	logger := zap.NewNop()
	crawler := &ICABankenCrawler{logger: logger}
	crawlTime := time.Date(2025, 12, 1, 10, 0, 0, 0, time.UTC)

	results, err := crawler.extractListRates(goldenHTML, crawlTime)
	if err != nil {
		t.Fatalf("extractListRates() error = %v", err)
	}

	if len(results) != 8 {
		t.Errorf("extractListRates() returned %d results, want 8", len(results))
	}

	// Verify expected terms are present
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
		assertICAListRateFields(t, r, crawlTime)
	}

	for term, found := range expectedTerms {
		if !found {
			t.Errorf("missing term %q in results", term)
		}
	}
}

func TestICABankenCrawler_extractAverageRates(t *testing.T) {
	t.Parallel()

	goldenHTML := loadGoldenFile(t, "testdata/ica_banken.html")
	logger := zap.NewNop()
	crawler := &ICABankenCrawler{logger: logger}
	crawlTime := time.Date(2025, 12, 1, 10, 0, 0, 0, time.UTC)

	results, err := crawler.extractAverageRates(goldenHTML, crawlTime)
	if err != nil {
		t.Fatalf("extractAverageRates() error = %v", err)
	}

	if len(results) == 0 {
		t.Fatal("extractAverageRates() returned no results")
	}

	// Verify we got multiple months of data
	months := make(map[string]bool)
	for _, r := range results {
		if r.AverageReferenceMonth != nil {
			key := r.AverageReferenceMonth.Month.String() + string(rune(r.AverageReferenceMonth.Year))
			months[key] = true
		}
		assertICAAverageRateFields(t, r, crawlTime)
	}

	// Should have data for multiple months
	if len(months) < 10 {
		t.Errorf("got data for %d months, want at least 10", len(months))
	}
}

func TestParseICARate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    float32
		wantErr bool
	}{
		{
			name:    "Swedish format with comma and percent",
			input:   "3,33 %",
			want:    3.33,
			wantErr: false,
		},
		{
			name:    "Swedish format with comma",
			input:   "3,74",
			want:    3.74,
			wantErr: false,
		},
		{
			name:    "with percent sign no space",
			input:   "3.74%",
			want:    3.74,
			wantErr: false,
		},
		{
			name:    "simple decimal",
			input:   "2.58",
			want:    2.58,
			wantErr: false,
		},
		{
			name:    "invalid text",
			input:   "abc",
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

			got, err := parseICARate(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseICARate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("parseICARate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseICADate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    time.Time
		wantErr bool
	}{
		{
			name:    "valid date YYYY-MM-DD",
			input:   "2025-10-06",
			want:    time.Date(2025, time.October, 6, 0, 0, 0, 0, time.UTC),
			wantErr: false,
		},
		{
			name:    "valid date beginning of year",
			input:   "2024-01-15",
			want:    time.Date(2024, time.January, 15, 0, 0, 0, 0, time.UTC),
			wantErr: false,
		},
		{
			name:    "valid date end of year",
			input:   "2023-12-31",
			want:    time.Date(2023, time.December, 31, 0, 0, 0, 0, time.UTC),
			wantErr: false,
		},
		{
			name:    "invalid format",
			input:   "26-09-2025",
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
		{
			name:    "invalid day",
			input:   "2025-01-00",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := parseICADate(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseICADate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !got.Equal(tt.want) {
				t.Errorf("parseICADate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseICAAvgMonth(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     string
		wantMonth time.Month
		wantYear  uint
		wantErr   bool
	}{
		{
			name:      "November 2025 format",
			input:     "2025 11",
			wantMonth: time.November,
			wantYear:  2025,
			wantErr:   false,
		},
		{
			name:      "December 2024 format",
			input:     "2024 12",
			wantMonth: time.December,
			wantYear:  2024,
			wantErr:   false,
		},
		{
			name:      "January format",
			input:     "2025 1",
			wantMonth: time.January,
			wantYear:  2025,
			wantErr:   false,
		},
		{
			name:      "with extra spaces",
			input:     "  2024   03  ",
			wantMonth: time.March,
			wantYear:  2024,
			wantErr:   false,
		},
		{
			name:    "invalid format - month name",
			input:   "November 2024",
			wantErr: true,
		},
		{
			name:    "invalid format - no space",
			input:   "202411",
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

			got, err := parseICAAvgMonth(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseICAAvgMonth() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got.Month != tt.wantMonth {
					t.Errorf("parseICAAvgMonth() month = %v, want %v", got.Month, tt.wantMonth)
				}
				if got.Year != tt.wantYear {
					t.Errorf("parseICAAvgMonth() year = %v, want %v", got.Year, tt.wantYear)
				}
			}
		})
	}
}

func TestNewICABankenCrawler(t *testing.T) {
	t.Parallel()

	mockClient := &httpmock.ClientMock{}
	logger := zap.NewNop()

	crawler := NewICABankenCrawler(mockClient, logger)

	if crawler == nil {
		t.Fatal("NewICABankenCrawler() returned nil")
	}
	if crawler.httpClient != mockClient {
		t.Error("httpClient not set correctly")
	}
	if crawler.logger != logger {
		t.Error("logger not set correctly")
	}
}

func TestICABankenCrawler_InterfaceCompliance(t *testing.T) {
	t.Parallel()

	// Compile-time check that ICABankenCrawler implements SiteCrawler
	var _ SiteCrawler = &ICABankenCrawler{}
}
