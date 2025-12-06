//nolint:revive,nolintlint,dupl // package name matches the package being tested; test patterns intentionally similar across crawlers
package nordax

import (
	"errors"
	"testing"
	"time"

	crawlertest "github.com/yama6a/bolan-compare/internal/app/crawler"
	"github.com/yama6a/bolan-compare/internal/pkg/http/httpmock"
	"github.com/yama6a/bolan-compare/internal/pkg/model"
	"go.uber.org/zap"
)

func TestNordaxCrawler_Crawl(t *testing.T) {
	t.Parallel()

	avgRatesHTML := crawlertest.LoadGoldenFile(t, "testdata/nordax_avg_rates.html")
	logger := zap.NewNop()

	tests := []struct {
		name             string
		mockFetch        func(url string, headers map[string]string) (string, error)
		wantAvgRates     bool
		wantTotalMinimum int
	}{
		{
			name: "successful crawl extracts average rates",
			mockFetch: func(_ string, _ map[string]string) (string, error) {
				return avgRatesHTML, nil
			},
			wantAvgRates:     true,
			wantTotalMinimum: 10, // At least 10 months of data
		},
		{
			name: "fetch error returns no results",
			mockFetch: func(_ string, _ map[string]string) (string, error) {
				return "", errors.New("network error")
			},
			wantAvgRates:     false,
			wantTotalMinimum: 0,
		},
		{
			name: "invalid HTML returns no results",
			mockFetch: func(_ string, _ map[string]string) (string, error) {
				return "<html><body>No data here</body></html>", nil
			},
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

			crawler := NewNordaxCrawler(mockClient, logger)
			results := make(chan model.InterestSet, 100)

			crawler.Crawl(results)
			close(results)

			var collectedResults []model.InterestSet
			for result := range results {
				collectedResults = append(collectedResults, result)
			}

			if len(collectedResults) < tt.wantTotalMinimum {
				t.Errorf("expected at least %d results, got %d", tt.wantTotalMinimum, len(collectedResults))
			}

			if tt.wantAvgRates {
				_, avgCount := crawlertest.CountRatesByType(collectedResults)
				if avgCount == 0 {
					t.Error("expected average rates but got none")
				}
			}

			// Verify all results have correct bank name.
			crawlertest.AssertBankName(t, collectedResults, nordaxBankName)
		})
	}
}

//nolint:cyclop // test validation requires multiple checks
func TestNordaxCrawler_extractAverageRates(t *testing.T) {
	t.Parallel()

	avgRatesHTML := crawlertest.LoadGoldenFile(t, "testdata/nordax_avg_rates.html")
	logger := zap.NewNop()
	crawler := NewNordaxCrawler(nil, logger)
	crawlTime := time.Date(2025, 12, 1, 10, 0, 0, 0, time.UTC)

	results, err := crawler.extractAverageRates(avgRatesHTML, crawlTime)
	if err != nil {
		t.Fatalf("extractAverageRates() error = %v", err)
	}

	if len(results) == 0 {
		t.Fatal("extractAverageRates() returned no results")
	}

	// Verify we have data for 3 months term.
	threeMonthsCount := 0
	for _, result := range results {
		if result.Term == model.Term3months {
			threeMonthsCount++
		}
	}

	if threeMonthsCount == 0 {
		t.Error("expected rates for 3 months term but got none")
	}

	// Verify all results have correct bank name.
	crawlertest.AssertBankName(t, results, nordaxBankName)

	// Verify all results have correct fields.
	for _, result := range results {
		if result.Type != model.TypeAverageRate {
			t.Errorf("expected Type=%v, got %v", model.TypeAverageRate, result.Type)
		}

		if result.NominalRate <= 0 {
			t.Errorf("expected positive rate, got %f", result.NominalRate)
		}

		if result.AverageReferenceMonth == nil {
			t.Error("expected valid AverageReferenceMonth, got nil")
		}

		if result.Term != model.Term3months && result.Term != model.Term3years && result.Term != model.Term5years {
			t.Errorf("unexpected term: %v", result.Term)
		}
	}
}

func TestParseNordaxTerm(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    model.Term
		wantErr bool
	}{
		{
			name:    "3 månaders",
			input:   "3 månaders",
			want:    model.Term3months,
			wantErr: false,
		},
		{
			name:    "36 månaders",
			input:   "36 månaders",
			want:    model.Term3years,
			wantErr: false,
		},
		{
			name:    "60 månaders",
			input:   "60 månaders",
			want:    model.Term5years,
			wantErr: false,
		},
		{
			name:    "with extra spaces",
			input:   "  3  månaders  ",
			want:    model.Term3months,
			wantErr: false,
		},
		{
			name:    "uppercase",
			input:   "3 MÅNADERS",
			want:    model.Term3months,
			wantErr: false,
		},
		{
			name:    "unsupported term",
			input:   "12 månaders",
			want:    "",
			wantErr: true,
		},
		{
			name:    "invalid format",
			input:   "invalid",
			want:    "",
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := parseNordaxTerm(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseNordaxTerm() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if got != tt.want {
				t.Errorf("parseNordaxTerm() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseNordaxRate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    float64
		wantErr bool
	}{
		{
			name:    "Swedish format with comma",
			input:   "4,66%",
			want:    4.66,
			wantErr: false,
		},
		{
			name:    "Swedish format without percent",
			input:   "4,66",
			want:    4.66,
			wantErr: false,
		},
		{
			name:    "decimal format with period",
			input:   "4.66%",
			want:    4.66,
			wantErr: false,
		},
		{
			name:    "with spaces",
			input:   "  4,66 %  ",
			want:    4.66,
			wantErr: false,
		},
		{
			name:    "integer rate",
			input:   "5%",
			want:    5.0,
			wantErr: false,
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

			got, err := parseNordaxRate(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseNordaxRate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if got != tt.want {
				t.Errorf("parseNordaxRate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseNordaxMonth(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "valid YYYY-MM",
			input:   "2025-11",
			wantErr: false,
		},
		{
			name:    "valid YYYY-MM beginning of year",
			input:   "2025-01",
			wantErr: false,
		},
		{
			name:    "valid YYYY-MM end of year",
			input:   "2024-12",
			wantErr: false,
		},
		{
			name:    "with spaces",
			input:   "  2025-11  ",
			wantErr: false,
		},
		{
			name:    "invalid format",
			input:   "Nov 2025",
			wantErr: true,
		},
		{
			name:    "invalid month",
			input:   "2025-13",
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

			got, err := parseNordaxMonth(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseNordaxMonth() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && got.IsZero() {
				t.Error("parseNordaxMonth() returned zero time for valid input")
			}
		})
	}
}

func TestNewNordaxCrawler(t *testing.T) {
	t.Parallel()

	crawler := NewNordaxCrawler(nil, nil)
	if crawler == nil {
		t.Fatal("NewNordaxCrawler() returned nil")
	}
}

func TestNordaxCrawler_InterfaceCompliance(t *testing.T) {
	t.Parallel()

	var _ crawlertest.SiteCrawler = &NordaxCrawler{}
}
