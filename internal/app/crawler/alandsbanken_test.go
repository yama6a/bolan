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

// assertAlandsbankListRateFields validates common fields for Ålandsbanken list rate results.
func assertAlandsbankListRateFields(t *testing.T, r model.InterestSet, crawlTime time.Time) {
	t.Helper()

	if r.Bank != alandsbankBankName {
		t.Errorf("Bank = %q, want %q", r.Bank, alandsbankBankName)
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

// assertAlandsbankAverageRateFields validates common fields for Ålandsbanken average rate results.
func assertAlandsbankAverageRateFields(t *testing.T, r model.InterestSet, crawlTime time.Time) {
	t.Helper()

	if r.Bank != alandsbankBankName {
		t.Errorf("Bank = %q, want %q", r.Bank, alandsbankBankName)
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

func TestAlandsbankCrawler_Crawl(t *testing.T) {
	t.Parallel()

	html := loadGoldenFile(t, "testdata/alandsbanken.html")
	logger := zap.NewNop()

	tests := []struct {
		name             string
		mockFetch        func(url string, headers map[string]string) (string, error)
		wantListRates    int
		wantAvgRatesMin  int
		wantTotalMinimum int
	}{
		{
			name: "successful crawl extracts list and average rates",
			mockFetch: func(_ string, _ map[string]string) (string, error) {
				return html, nil
			},
			wantListRates:    8,  // 3 mån, 1-7 år, 10 år
			wantAvgRatesMin:  12, // 3 mån only, 12+ months of data
			wantTotalMinimum: 20, // 8 list rates + at least 12 average rates
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

			crawler := NewAlandsbankCrawler(mockClient, logger)
			resultChan := make(chan model.InterestSet, 1000)

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

			if avgRateCount < tt.wantAvgRatesMin {
				t.Errorf("average rate count = %d, want at least %d", avgRateCount, tt.wantAvgRatesMin)
			}

			if len(results) < tt.wantTotalMinimum {
				t.Errorf("total results = %d, want at least %d", len(results), tt.wantTotalMinimum)
			}

			assertBankName(t, results, alandsbankBankName)
		})
	}
}

func TestAlandsbankCrawler_extractListRates(t *testing.T) {
	t.Parallel()

	goldenHTML := loadGoldenFile(t, "testdata/alandsbanken.html")
	logger := zap.NewNop()
	crawler := &AlandsbankCrawler{logger: logger}
	crawlTime := time.Date(2025, 12, 1, 10, 0, 0, 0, time.UTC)

	rates, err := crawler.extractListRates(goldenHTML, crawlTime)
	if err != nil {
		t.Fatalf("extractListRates failed: %v", err)
	}

	if len(rates) != 8 {
		t.Errorf("extractListRates returned %d rates, want 8", len(rates))
	}

	// Validate all list rates have required fields
	for _, r := range rates {
		assertAlandsbankListRateFields(t, r, crawlTime)
	}

	// Validate specific known values from golden file
	expectedRates := map[model.Term]float32{
		model.Term3months: 3.85,
		model.Term1year:   3.45,
		model.Term2years:  3.60,
		model.Term3years:  3.70,
		model.Term4years:  3.70,
		model.Term5years:  3.90,
		model.Term7years:  4.05,
		model.Term10years: 4.30,
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

	// Validate change date
	expectedDate := time.Date(2025, 10, 3, 0, 0, 0, 0, time.UTC)
	for _, r := range rates {
		if r.ChangedOn == nil {
			t.Errorf("ChangedOn is nil for term %q", r.Term)
			continue
		}
		if *r.ChangedOn != expectedDate {
			t.Errorf("ChangedOn for term %q = %v, want %v", r.Term, *r.ChangedOn, expectedDate)
		}
	}
}

func TestAlandsbankCrawler_extractAverageRates(t *testing.T) {
	t.Parallel()

	goldenHTML := loadGoldenFile(t, "testdata/alandsbanken.html")
	logger := zap.NewNop()
	crawler := &AlandsbankCrawler{logger: logger}
	crawlTime := time.Date(2025, 12, 1, 10, 0, 0, 0, time.UTC)

	rates, err := crawler.extractAverageRates(goldenHTML, crawlTime)
	if err != nil {
		t.Fatalf("extractAverageRates failed: %v", err)
	}

	if len(rates) < 12 {
		t.Errorf("extractAverageRates returned %d rates, want at least 12", len(rates))
	}

	// Validate all average rates have required fields and correct term
	for _, r := range rates {
		assertAlandsbankAverageRateFields(t, r, crawlTime)
		if r.Term != model.Term3months {
			t.Errorf("unexpected term in average rates: %q, want %q", r.Term, model.Term3months)
		}
	}

	// Validate specific known value from golden file (Oktober 2025 = 2.59%)
	validateOktoberRate(t, rates)
}

func validateOktoberRate(t *testing.T, rates []model.InterestSet) {
	t.Helper()
	for _, r := range rates {
		if r.AverageReferenceMonth != nil &&
			r.AverageReferenceMonth.Year == 2025 &&
			r.AverageReferenceMonth.Month == time.October {
			if r.NominalRate != 2.59 {
				t.Errorf("Oktober 2025 rate = %f, want 2.59", r.NominalRate)
			}
			return
		}
	}
	t.Error("did not find Oktober 2025 in average rates")
}

func TestParseAlandsbankRate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    float32
		wantErr bool
	}{
		{
			name:  "valid rate with comma",
			input: "3,85 %",
			want:  3.85,
		},
		{
			name:  "valid rate with period",
			input: "3.45%",
			want:  3.45,
		},
		{
			name:  "valid rate with spaces",
			input: "  2,59  %  ",
			want:  2.59,
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
			name:    "non-numeric",
			input:   "abc",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := parseAlandsbankRate(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseAlandsbankRate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("parseAlandsbankRate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseAlandsbankListDate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    time.Time
		wantErr bool
	}{
		{
			name:  "valid date",
			input: "2025.10.03",
			want:  time.Date(2025, 10, 3, 0, 0, 0, 0, time.UTC),
		},
		{
			name:  "valid date with spaces",
			input: "  2025.12.01  ",
			want:  time.Date(2025, 12, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name:    "invalid format",
			input:   "2025-10-03",
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

			got, err := parseAlandsbankListDate(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseAlandsbankListDate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("parseAlandsbankListDate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseAlandsbankAvgMonth(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     string
		wantYear  uint
		wantMonth time.Month
		wantErr   bool
	}{
		{
			name:      "valid month",
			input:     "Oktober 2025",
			wantYear:  2025,
			wantMonth: time.October,
		},
		{
			name:      "valid month lowercase",
			input:     "september 2025",
			wantYear:  2025,
			wantMonth: time.September,
		},
		{
			name:      "valid month with spaces",
			input:     "  Augusti 2025  ",
			wantYear:  2025,
			wantMonth: time.August,
		},
		{
			name:    "invalid format",
			input:   "2025 Oktober",
			wantErr: true,
		},
		{
			name:    "unknown month",
			input:   "Foobar 2025",
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

			got, err := parseAlandsbankAvgMonth(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseAlandsbankAvgMonth() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got.Year != tt.wantYear {
					t.Errorf("parseAlandsbankAvgMonth() Year = %d, want %d", got.Year, tt.wantYear)
				}
				if got.Month != tt.wantMonth {
					t.Errorf("parseAlandsbankAvgMonth() Month = %v, want %v", got.Month, tt.wantMonth)
				}
			}
		})
	}
}

func TestNewAlandsbankCrawler(t *testing.T) {
	t.Parallel()

	client := &httpmock.ClientMock{}
	logger := zap.NewNop()

	crawler := NewAlandsbankCrawler(client, logger)

	if crawler == nil {
		t.Fatal("NewAlandsbankCrawler returned nil")
	}
	if crawler.httpClient == nil {
		t.Error("httpClient is nil")
	}
	if crawler.logger == nil {
		t.Error("logger is nil")
	}
}

func TestAlandsbankCrawler_InterfaceCompliance(t *testing.T) {
	t.Parallel()

	var _ SiteCrawler = &AlandsbankCrawler{}
}
