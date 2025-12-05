//nolint:revive,nolintlint,dupl // package name matches the package being tested; test patterns intentionally similar across crawlers
package jak

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
func TestJAKCrawler_Crawl(t *testing.T) {
	t.Parallel()

	ratesHTML := crawlertest.LoadGoldenFile(t, "testdata/jak_rates.html")
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
				return ratesHTML, nil
			},
			wantListRates:    2, // 3 månader and 12 månader (1 år)
			wantAvgRates:     true,
			wantTotalMinimum: 10, // 2 list + average rates from both tables
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
		{
			name: "invalid HTML returns no results",
			mockFetch: func(_ string, _ map[string]string) (string, error) {
				return "<html><body>No tables here</body></html>", nil
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

			crawler := NewJAKCrawler(mockClient, logger)
			resultChan := make(chan model.InterestSet, 100)

			crawler.Crawl(resultChan)
			close(resultChan)

			var results []model.InterestSet
			for set := range resultChan {
				results = append(results, set)
			}

			listRateCount, avgRateCount := crawlertest.CountRatesByType(results)

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

			crawlertest.AssertBankName(t, results, jakBankName)
		})
	}
}

func TestJAKCrawler_extractRates(t *testing.T) {
	t.Parallel()

	ratesHTML := crawlertest.LoadGoldenFile(t, "testdata/jak_rates.html")
	logger := zap.NewNop()
	crawlTime := time.Date(2025, 12, 1, 10, 0, 0, 0, time.UTC)

	crawler := &JAKCrawler{logger: logger}

	results, err := crawler.extractRates(ratesHTML, crawlTime)
	if err != nil {
		t.Fatalf("extractRates() error = %v", err)
	}

	// Should have list rates and average rates for both terms
	if len(results) < 10 {
		t.Errorf("extractRates() returned %d results, want at least 10", len(results))
	}

	// Verify expected terms are present
	expectedTerms := map[model.Term]bool{
		model.Term3months: false,
		model.Term1year:   false,
	}

	for _, r := range results {
		if _, ok := expectedTerms[r.Term]; ok {
			expectedTerms[r.Term] = true
		}
		assertJAKRateFields(t, r, crawlTime)
	}

	for term, found := range expectedTerms {
		if !found {
			t.Errorf("missing term %q in results", term)
		}
	}
}

func TestJAKCrawler_parseJAKRate(t *testing.T) {
	t.Parallel()

	logger := zap.NewNop()
	crawler := &JAKCrawler{logger: logger}

	tests := []struct {
		name    string
		input   string
		want    float32
		wantErr bool
	}{
		{
			name:    "rate with comma and percent",
			input:   "3,58 %",
			want:    3.58,
			wantErr: false,
		},
		{
			name:    "rate with period",
			input:   "3.24%",
			want:    3.24,
			wantErr: false,
		},
		{
			name:    "rate without percent",
			input:   "4,08",
			want:    4.08,
			wantErr: false,
		},
		{
			name:    "rate with double percent (malformed data)",
			input:   "3,58 % %",
			want:    3.58,
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

			got, err := crawler.parseJAKRate(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseJAKRate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				// Compare with tolerance for float comparison
				diff := got - tt.want
				if diff > 0.001 || diff < -0.001 {
					t.Errorf("parseJAKRate() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestJAKCrawler_parseJAKMonth(t *testing.T) {
	t.Parallel()

	logger := zap.NewNop()
	crawler := &JAKCrawler{logger: logger}

	tests := []struct {
		name      string
		input     string
		wantYear  uint
		wantMonth time.Month
		wantErr   bool
	}{
		{
			name:      "valid month November 2025",
			input:     "2025 11",
			wantYear:  2025,
			wantMonth: time.November,
			wantErr:   false,
		},
		{
			name:      "valid month January 2025",
			input:     "2025 01",
			wantYear:  2025,
			wantMonth: time.January,
			wantErr:   false,
		},
		{
			name:      "valid month December 2024",
			input:     "2024 12",
			wantYear:  2024,
			wantMonth: time.December,
			wantErr:   false,
		},
		{
			name:      "single digit month",
			input:     "2025 9",
			wantYear:  2025,
			wantMonth: time.September,
			wantErr:   false,
		},
		{
			name:    "invalid format - with day",
			input:   "2025-11-30",
			wantErr: true,
		},
		{
			name:    "invalid month 00",
			input:   "2025 00",
			wantErr: true,
		},
		{
			name:    "invalid month 13",
			input:   "2025 13",
			wantErr: true,
		},
		{
			name:    "invalid - not a date",
			input:   "abcdefg",
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

			got, err := crawler.parseJAKMonth(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseJAKMonth() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got.Year != tt.wantYear {
					t.Errorf("parseJAKMonth() year = %v, want %v", got.Year, tt.wantYear)
				}
				if got.Month != tt.wantMonth {
					t.Errorf("parseJAKMonth() month = %v, want %v", got.Month, tt.wantMonth)
				}
			}
		})
	}
}

func TestNewJAKCrawler(t *testing.T) {
	t.Parallel()

	mockClient := &httpmock.ClientMock{}
	logger := zap.NewNop()

	crawler := NewJAKCrawler(mockClient, logger)

	if crawler == nil {
		t.Fatal("NewJAKCrawler() returned nil")
	}
	if crawler.httpClient != mockClient {
		t.Error("httpClient not set correctly")
	}
	if crawler.logger != logger {
		t.Error("logger not set correctly")
	}
}

func TestJAKCrawler_InterfaceCompliance(t *testing.T) {
	t.Parallel()

	// Compile-time check that JAKCrawler implements SiteCrawler
	var _ crawlertest.SiteCrawler = &JAKCrawler{}
}

// assertJAKRateFields validates common fields for JAK rate results.
func assertJAKRateFields(t *testing.T, r model.InterestSet, crawlTime time.Time) {
	t.Helper()

	if r.Bank != jakBankName {
		t.Errorf("Bank = %q, want %q", r.Bank, jakBankName)
	}
	if r.Type != model.TypeListRate && r.Type != model.TypeAverageRate {
		t.Errorf("Type = %q, want TypeListRate or TypeAverageRate", r.Type)
	}
	if r.NominalRate <= 0 {
		t.Errorf("NominalRate = %f, want positive value", r.NominalRate)
	}
	if r.Type == model.TypeAverageRate && r.AverageReferenceMonth == nil {
		t.Error("AverageReferenceMonth is nil for average rate")
	}
	if r.LastCrawledAt != crawlTime {
		t.Errorf("LastCrawledAt = %v, want %v", r.LastCrawledAt, crawlTime)
	}
}
