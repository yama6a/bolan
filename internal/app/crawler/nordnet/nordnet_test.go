//nolint:revive,nolintlint,dupl // package name matches the package being tested; test patterns intentionally similar across crawlers
package nordnet

import (
	"errors"
	"regexp"
	"testing"
	"time"

	crawlertest "github.com/yama6a/bolan-compare/internal/app/crawler"
	"github.com/yama6a/bolan-compare/internal/pkg/http/httpmock"
	"github.com/yama6a/bolan-compare/internal/pkg/model"
	"go.uber.org/zap"
)

//nolint:cyclop // table-driven test with multiple cases
func TestNordnetCrawler_Crawl(t *testing.T) {
	t.Parallel()

	ratesJSON := crawlertest.LoadGoldenFile(t, "testdata/nordnet_rates.json")
	logger := zap.NewNop()

	tests := []struct {
		name          string
		mockFetch     func(url string, headers map[string]string) (string, error)
		wantListRates int
	}{
		{
			name: "successful crawl extracts list rates",
			mockFetch: func(_ string, _ map[string]string) (string, error) {
				return ratesJSON, nil
			},
			wantListRates: 6, // 3 mån, 1 år, 2 år, 3 år, 5 år, 10 år
		},
		{
			name: "fetch error returns no results",
			mockFetch: func(_ string, _ map[string]string) (string, error) {
				return "", errors.New("network error")
			},
			wantListRates: 0,
		},
		{
			name: "invalid JSON returns no results",
			mockFetch: func(_ string, _ map[string]string) (string, error) {
				return "{invalid}", nil
			},
			wantListRates: 0,
		},
		{
			name: "empty response returns no results",
			mockFetch: func(_ string, _ map[string]string) (string, error) {
				return `{"includes":{"Entry":[]}}`, nil
			},
			wantListRates: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockClient := &httpmock.ClientMock{
				FetchFunc: tt.mockFetch,
			}

			crawler := NewNordnetCrawler(mockClient, logger)
			resultChan := make(chan model.InterestSet, 100)

			crawler.Crawl(resultChan)
			close(resultChan)

			var results []model.InterestSet
			for set := range resultChan {
				results = append(results, set)
			}

			listRateCount, _ := crawlertest.CountRatesByType(results)

			if listRateCount != tt.wantListRates {
				t.Errorf("list rate count = %d, want %d", listRateCount, tt.wantListRates)
			}

			crawlertest.AssertBankName(t, results, nordnetBankName)
		})
	}
}

func TestNordnetCrawler_fetchListRates(t *testing.T) {
	t.Parallel()

	ratesJSON := crawlertest.LoadGoldenFile(t, "testdata/nordnet_rates.json")
	logger := zap.NewNop()
	crawlTime := time.Date(2025, 12, 1, 10, 0, 0, 0, time.UTC)

	mockClient := &httpmock.ClientMock{
		FetchFunc: func(_ string, _ map[string]string) (string, error) {
			return ratesJSON, nil
		},
	}

	crawler := &NordnetCrawler{httpClient: mockClient, logger: logger}

	results, err := crawler.fetchListRates(crawlTime)
	if err != nil {
		t.Fatalf("fetchListRates() error = %v", err)
	}

	if len(results) != 6 {
		t.Errorf("fetchListRates() returned %d results, want 6", len(results))
	}

	// Verify expected terms are present
	expectedTerms := map[model.Term]bool{
		model.Term3months: false,
		model.Term1year:   false,
		model.Term2years:  false,
		model.Term3years:  false,
		model.Term5years:  false,
		model.Term10years: false,
	}

	for _, r := range results {
		if _, ok := expectedTerms[r.Term]; ok {
			expectedTerms[r.Term] = true
		}
		assertNordnetListRateFields(t, r, crawlTime)
	}

	for term, found := range expectedTerms {
		if !found {
			t.Errorf("missing term %q in results", term)
		}
	}
}

func TestParseNordnetRate(t *testing.T) {
	t.Parallel()

	rateRegex := regexp.MustCompile(`^(\d+),(\d+)`)

	tests := []struct {
		name    string
		input   string
		want    float32
		wantErr bool
	}{
		{
			name:    "simple rate with effective rate",
			input:   "2,54 (2,57)",
			want:    2.54,
			wantErr: false,
		},
		{
			name:    "rate with two decimal places",
			input:   "3,00 (3,04)",
			want:    3.00,
			wantErr: false,
		},
		{
			name:    "rate with higher value",
			input:   "4,21 (4,29)",
			want:    4.21,
			wantErr: false,
		},
		{
			name:    "invalid format",
			input:   "invalid",
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

			got, err := parseNordnetRate(tt.input, rateRegex)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseNordnetRate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				// Allow small floating point differences
				if got < tt.want-0.01 || got > tt.want+0.01 {
					t.Errorf("parseNordnetRate() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestNewNordnetCrawler(t *testing.T) {
	t.Parallel()

	mockClient := &httpmock.ClientMock{}
	logger := zap.NewNop()

	crawler := NewNordnetCrawler(mockClient, logger)

	if crawler == nil {
		t.Fatal("NewNordnetCrawler() returned nil")
	}
	if crawler.httpClient != mockClient {
		t.Error("httpClient not set correctly")
	}
	if crawler.logger != logger {
		t.Error("logger not set correctly")
	}
}

func TestNordnetCrawler_InterfaceCompliance(t *testing.T) {
	t.Parallel()

	// Compile-time check that NordnetCrawler implements SiteCrawler
	var _ crawlertest.SiteCrawler = &NordnetCrawler{}
}

// assertNordnetListRateFields validates common fields for Nordnet list rate results.
func assertNordnetListRateFields(t *testing.T, r model.InterestSet, crawlTime time.Time) {
	t.Helper()

	if r.Bank != nordnetBankName {
		t.Errorf("Bank = %q, want %q", r.Bank, nordnetBankName)
	}
	if r.Type != model.TypeListRate {
		t.Errorf("Type = %q, want %q", r.Type, model.TypeListRate)
	}
	if r.NominalRate <= 0 {
		t.Errorf("NominalRate = %f, want positive value", r.NominalRate)
	}
	// Nordnet API doesn't provide change dates
	if r.ChangedOn != nil {
		t.Error("ChangedOn is not nil, want nil for Nordnet")
	}
	if r.LastCrawledAt != crawlTime {
		t.Errorf("LastCrawledAt = %v, want %v", r.LastCrawledAt, crawlTime)
	}
}
