//nolint:revive,nolintlint,dupl // package name matches the package being tested; test patterns intentionally similar across crawlers
package landshypotek

import (
	"errors"
	"testing"
	"time"

	crawlertest "github.com/yama6a/bolan-compare/internal/app/crawler"
	"github.com/yama6a/bolan-compare/internal/pkg/http/httpmock"
	"github.com/yama6a/bolan-compare/internal/pkg/model"
	"go.uber.org/zap"
)

// assertLandshypotekListRateFields validates common fields for Landshypotek list rate results.
func assertLandshypotekListRateFields(t *testing.T, r model.InterestSet, crawlTime time.Time) {
	t.Helper()

	if r.Bank != landshypotekBankName {
		t.Errorf("Bank = %q, want %q", r.Bank, landshypotekBankName)
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

func TestLandshypotekCrawler_Crawl(t *testing.T) {
	t.Parallel()

	html := crawlertest.LoadGoldenFile(t, "testdata/landshypotek_rates.html")
	logger := zap.NewNop()

	tests := []struct {
		name             string
		mockFetch        func(url string, headers map[string]string) (string, error)
		wantListRates    int
		wantAvgRatesMin  int
		wantTotalMinimum int
	}{
		{
			name: "successful crawl extracts list rates",
			mockFetch: func(_ string, _ map[string]string) (string, error) {
				return html, nil
			},
			wantListRates:    6, // 3 mån, 1-5 år (6 terms total)
			wantAvgRatesMin:  0, // Average rates require JavaScript
			wantTotalMinimum: 6, // 6 list rates
		},
		{
			name: "fetch error returns no results",
			mockFetch: func(_ string, _ map[string]string) (string, error) {
				return "", errors.New("network error")
			},
			wantListRates:    0,
			wantAvgRatesMin:  0,
			wantTotalMinimum: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockClient := &httpmock.ClientMock{
				FetchFunc: tt.mockFetch,
			}

			crawler := NewLandshypotekCrawler(mockClient, logger)
			resultChan := make(chan model.InterestSet, 1000)

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

			if avgRateCount < tt.wantAvgRatesMin {
				t.Errorf("average rate count = %d, want at least %d", avgRateCount, tt.wantAvgRatesMin)
			}

			if len(results) < tt.wantTotalMinimum {
				t.Errorf("total results = %d, want at least %d", len(results), tt.wantTotalMinimum)
			}

			crawlertest.AssertBankName(t, results, landshypotekBankName)
		})
	}
}

func TestLandshypotekCrawler_extractListRates(t *testing.T) {
	t.Parallel()

	goldenHTML := crawlertest.LoadGoldenFile(t, "testdata/landshypotek_rates.html")
	logger := zap.NewNop()
	crawler := &LandshypotekCrawler{logger: logger}
	crawlTime := time.Date(2025, 12, 1, 10, 0, 0, 0, time.UTC)

	rates, err := crawler.extractListRates(goldenHTML, crawlTime)
	if err != nil {
		t.Fatalf("extractListRates failed: %v", err)
	}

	if len(rates) != 6 {
		t.Errorf("extractListRates returned %d rates, want 6", len(rates))
	}

	// Validate all list rates have required fields
	for _, r := range rates {
		assertLandshypotekListRateFields(t, r, crawlTime)
	}

	// Validate specific known values from golden file (list rates from accordion)
	expectedRates := map[model.Term]float32{
		model.Term3months: 3.04,
		model.Term1year:   3.19,
		model.Term2years:  3.40,
		model.Term3years:  3.50,
		model.Term4years:  3.65,
		model.Term5years:  3.75,
	}

	for _, r := range rates {
		expectedRate, ok := expectedRates[r.Term]
		if !ok {
			t.Errorf("unexpected term in results: %q", r.Term)
			continue
		}
		if r.NominalRate != expectedRate {
			t.Errorf("rate for term %q = %f, want %f", r.Term, r.NominalRate, expectedRate)
		}
	}
}

func TestParseLandshypotekRate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    float32
		wantErr bool
	}{
		{
			name:  "valid rate with comma",
			input: "2,54 %",
			want:  2.54,
		},
		{
			name:  "valid rate with period",
			input: "3.15%",
			want:  3.15,
		},
		{
			name:  "valid rate with spaces",
			input: "  2,71  %  ",
			want:  2.71,
		},
		{
			name:    "empty rate",
			input:   "",
			wantErr: true,
		},
		{
			name:    "dash",
			input:   "-",
			wantErr: true,
		},
		{
			name:    "n/a",
			input:   "n/a",
			wantErr: true,
		},
		{
			name:    "non-numeric",
			input:   "abc",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := parseLandshypotekRate(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseLandshypotekRate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("parseLandshypotekRate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewLandshypotekCrawler(t *testing.T) {
	t.Parallel()

	client := &httpmock.ClientMock{}
	logger := zap.NewNop()

	crawler := NewLandshypotekCrawler(client, logger)

	if crawler == nil {
		t.Fatal("NewLandshypotekCrawler returned nil")
	}
	if crawler.httpClient == nil {
		t.Error("httpClient is nil")
	}
	if crawler.logger == nil {
		t.Error("logger is nil")
	}
}

func TestLandshypotekCrawler_InterfaceCompliance(t *testing.T) {
	t.Parallel()

	var _ crawlertest.SiteCrawler = &LandshypotekCrawler{}
}
