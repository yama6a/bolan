//nolint:revive,nolintlint,dupl // package name matches the package being tested; test patterns intentionally similar across crawlers
package svea

import (
	"errors"
	"testing"
	"time"

	crawlertest "github.com/yama6a/bolan-compare/internal/app/crawler"
	"github.com/yama6a/bolan-compare/internal/pkg/http/httpmock"
	"github.com/yama6a/bolan-compare/internal/pkg/model"
	"go.uber.org/zap"
)

//nolint:cyclop // table-driven test with multiple cases
func TestSveaCrawler_Crawl(t *testing.T) {
	t.Parallel()

	listRatesHTML := crawlertest.LoadGoldenFile(t, "testdata/svea_list_rates.html")
	avgRatesHTML := crawlertest.LoadGoldenFile(t, "testdata/svea_avg_rates.html")
	logger := zap.NewNop()

	tests := []struct {
		name             string
		mockFetch        func(url string, headers map[string]string) (string, error)
		wantListRates    bool
		wantAvgRates     bool
		wantTotalMinimum int
	}{
		{
			name: "successful crawl extracts list and average rates",
			mockFetch: func(url string, _ map[string]string) (string, error) {
				if url == sveaListRatesURL {
					return listRatesHTML, nil
				}
				return avgRatesHTML, nil
			},
			wantListRates:    true,
			wantAvgRates:     true,
			wantTotalMinimum: 11, // 1 list rate + at least 10 months of avg data
		},
		{
			name: "fetch error for list rates still returns average rates",
			mockFetch: func(url string, _ map[string]string) (string, error) {
				if url == sveaListRatesURL {
					return "", errors.New("network error")
				}
				return avgRatesHTML, nil
			},
			wantListRates:    false,
			wantAvgRates:     true,
			wantTotalMinimum: 10,
		},
		{
			name: "fetch error for both returns no results",
			mockFetch: func(_ string, _ map[string]string) (string, error) {
				return "", errors.New("network error")
			},
			wantListRates:    false,
			wantAvgRates:     false,
			wantTotalMinimum: 0,
		},
		{
			name: "invalid HTML returns no results",
			mockFetch: func(_ string, _ map[string]string) (string, error) {
				return "<html><body>No tables here</body></html>", nil
			},
			wantListRates:    false,
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

			crawler := NewSveaCrawler(mockClient, logger)
			resultChan := make(chan model.InterestSet, 100)

			crawler.Crawl(resultChan)
			close(resultChan)

			var results []model.InterestSet
			for set := range resultChan {
				results = append(results, set)
			}

			listRateCount, avgRateCount := crawlertest.CountRatesByType(results)

			if tt.wantListRates && listRateCount == 0 {
				t.Error("expected list rates but got none")
			}

			if !tt.wantListRates && listRateCount > 0 {
				t.Errorf("expected no list rates but got %d", listRateCount)
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

			crawlertest.AssertBankName(t, results, sveaBankName)
		})
	}
}

func TestSveaCrawler_extractListRate(t *testing.T) {
	t.Parallel()

	listRatesHTML := crawlertest.LoadGoldenFile(t, "testdata/svea_list_rates.html")
	logger := zap.NewNop()
	crawlTime := time.Date(2025, 12, 1, 10, 0, 0, 0, time.UTC)

	crawler := &SveaCrawler{logger: logger}

	result, err := crawler.extractListRate(listRatesHTML, crawlTime)
	if err != nil {
		t.Fatalf("extractListRate() error = %v", err)
	}

	// Validate list rate fields.
	if result.Bank != sveaBankName {
		t.Errorf("Bank = %q, want %q", result.Bank, sveaBankName)
	}
	if result.Type != model.TypeListRate {
		t.Errorf("Type = %q, want TypeListRate", result.Type)
	}
	if result.Term != model.Term3months {
		t.Errorf("Term = %q, want Term3months", result.Term)
	}
	if result.NominalRate <= 0 {
		t.Errorf("NominalRate = %f, want positive value", result.NominalRate)
	}
	if result.LastCrawledAt != crawlTime {
		t.Errorf("LastCrawledAt = %v, want %v", result.LastCrawledAt, crawlTime)
	}
	// List rates should not have AverageReferenceMonth.
	if result.AverageReferenceMonth != nil {
		t.Error("AverageReferenceMonth should be nil for list rate")
	}
}

func TestSveaCrawler_extractAverageRates(t *testing.T) {
	t.Parallel()

	avgRatesHTML := crawlertest.LoadGoldenFile(t, "testdata/svea_avg_rates.html")
	logger := zap.NewNop()
	crawlTime := time.Date(2025, 12, 1, 10, 0, 0, 0, time.UTC)

	crawler := &SveaCrawler{logger: logger}

	results, err := crawler.extractAverageRates(avgRatesHTML, crawlTime)
	if err != nil {
		t.Fatalf("extractAverageRates() error = %v", err)
	}

	// Should have at least 10 months of data.
	if len(results) < 10 {
		t.Errorf("extractAverageRates() returned %d results, want at least 10", len(results))
	}

	// All results should be average rates for 3-month term.
	for _, r := range results {
		assertSveaAvgRateFields(t, r, crawlTime)
	}
}

func TestSveaCrawler_parseSveaRate(t *testing.T) {
	t.Parallel()

	logger := zap.NewNop()
	crawler := &SveaCrawler{logger: logger}

	tests := []struct {
		name    string
		input   string
		want    float32
		wantErr bool
	}{
		{
			name:    "rate with comma and percent",
			input:   "6,10 %",
			want:    6.10,
			wantErr: false,
		},
		{
			name:    "rate with period",
			input:   "6.33%",
			want:    6.33,
			wantErr: false,
		},
		{
			name:    "rate without percent",
			input:   "5,86",
			want:    5.86,
			wantErr: false,
		},
		{
			name:    "rate with spaces",
			input:   "  6,51  %  ",
			want:    6.51,
			wantErr: false,
		},
		{
			name:    "dash means no data",
			input:   "-",
			want:    0,
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			want:    0,
			wantErr: true,
		},
		{
			name:    "invalid text",
			input:   "invalid",
			want:    0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := crawler.parseSveaRate(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseSveaRate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				diff := got - tt.want
				if diff > 0.001 || diff < -0.001 {
					t.Errorf("parseSveaRate() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestSveaCrawler_parseSveaMonth(t *testing.T) {
	t.Parallel()

	logger := zap.NewNop()
	crawler := &SveaCrawler{logger: logger}

	tests := []struct {
		name      string
		input     string
		wantYear  uint
		wantMonth time.Month
		wantErr   bool
	}{
		{
			name:      "November 2025",
			input:     "November 2025",
			wantYear:  2025,
			wantMonth: time.November,
			wantErr:   false,
		},
		{
			name:      "Oktober 2025",
			input:     "Oktober 2025",
			wantYear:  2025,
			wantMonth: time.October,
			wantErr:   false,
		},
		{
			name:      "December 2024",
			input:     "December 2024",
			wantYear:  2024,
			wantMonth: time.December,
			wantErr:   false,
		},
		{
			name:      "Januari 2025 lowercase",
			input:     "januari 2025",
			wantYear:  2025,
			wantMonth: time.January,
			wantErr:   false,
		},
		{
			name:      "with extra spaces",
			input:     "  Mars   2025  ",
			wantYear:  2025,
			wantMonth: time.March,
			wantErr:   false,
		},
		{
			name:    "invalid format",
			input:   "2025-11",
			wantErr: true,
		},
		{
			name:    "unknown month",
			input:   "Måndag 2025",
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

			got, err := crawler.parseSveaMonth(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseSveaMonth() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got.Year != tt.wantYear {
					t.Errorf("parseSveaMonth() year = %v, want %v", got.Year, tt.wantYear)
				}
				if got.Month != tt.wantMonth {
					t.Errorf("parseSveaMonth() month = %v, want %v", got.Month, tt.wantMonth)
				}
			}
		})
	}
}

func TestNewSveaCrawler(t *testing.T) {
	t.Parallel()

	mockClient := &httpmock.ClientMock{}
	logger := zap.NewNop()

	crawler := NewSveaCrawler(mockClient, logger)

	if crawler == nil {
		t.Fatal("NewSveaCrawler() returned nil")
	}
	if crawler.httpClient != mockClient {
		t.Error("httpClient not set correctly")
	}
	if crawler.logger != logger {
		t.Error("logger not set correctly")
	}
}

func TestSveaCrawler_InterfaceCompliance(t *testing.T) {
	t.Parallel()

	// Compile-time check that SveaCrawler implements SiteCrawler.
	var _ crawlertest.SiteCrawler = &SveaCrawler{}
}

// assertSveaAvgRateFields validates common fields for Svea average rate results.
func assertSveaAvgRateFields(t *testing.T, r model.InterestSet, crawlTime time.Time) {
	t.Helper()

	if r.Bank != sveaBankName {
		t.Errorf("Bank = %q, want %q", r.Bank, sveaBankName)
	}
	if r.Type != model.TypeAverageRate {
		t.Errorf("Type = %q, want TypeAverageRate", r.Type)
	}
	// Svea only offers variable rate (3 months).
	if r.Term != model.Term3months {
		t.Errorf("Term = %q, want Term3months", r.Term)
	}
	if r.NominalRate <= 0 {
		t.Errorf("NominalRate = %f, want positive value", r.NominalRate)
	}
	if r.AverageReferenceMonth == nil {
		t.Error("AverageReferenceMonth is nil for average rate")
	}
	if r.LastCrawledAt != crawlTime {
		t.Errorf("LastCrawledAt = %v, want %v", r.LastCrawledAt, crawlTime)
	}
}
