//nolint:revive,nolintlint,dupl // package name matches the package being tested; test patterns intentionally similar across crawlers
package hypoteket

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
func TestHypoteketCrawler_Crawl(t *testing.T) {
	t.Parallel()

	payloadJSON := crawlertest.LoadGoldenFile(t, "testdata/hypoteket_rates.json")
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
				return payloadJSON, nil
			},
			wantListRates:    5, // 3 mån, 1 år, 2 år, 3 år, 5 år
			wantAvgRates:     true,
			wantTotalMinimum: 40, // at least 40 interest sets (5 list + many average)
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
			name: "invalid JSON returns no results",
			mockFetch: func(_ string, _ map[string]string) (string, error) {
				return "{invalid}", nil
			},
			wantListRates:    0,
			wantAvgRates:     false,
			wantTotalMinimum: 0,
		},
		{
			name: "empty array returns no results",
			mockFetch: func(_ string, _ map[string]string) (string, error) {
				return "[]", nil
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

			crawler := NewHypoteketCrawler(mockClient, logger)
			resultChan := make(chan model.InterestSet, 200)

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

			crawlertest.AssertBankName(t, results, hypoteketBankName)
		})
	}
}

func TestHypoteketCrawler_parseListRates(t *testing.T) {
	t.Parallel()

	payloadJSON := crawlertest.LoadGoldenFile(t, "testdata/hypoteket_rates.json")
	logger := zap.NewNop()
	crawlTime := time.Date(2025, 12, 1, 10, 0, 0, 0, time.UTC)

	crawler := &HypoteketCrawler{logger: logger}

	results, err := crawler.parseListRates(payloadJSON, crawlTime)
	if err != nil {
		t.Fatalf("parseListRates() error = %v", err)
	}

	if len(results) != 5 {
		t.Errorf("parseListRates() returned %d results, want 5", len(results))
	}

	// Verify expected terms are present
	expectedTerms := map[model.Term]bool{
		model.Term3months: false,
		model.Term1year:   false,
		model.Term2years:  false,
		model.Term3years:  false,
		model.Term5years:  false,
	}

	for _, r := range results {
		if _, ok := expectedTerms[r.Term]; ok {
			expectedTerms[r.Term] = true
		}
		assertHypoteketListRateFields(t, r, crawlTime)
	}

	for term, found := range expectedTerms {
		if !found {
			t.Errorf("missing term %q in results", term)
		}
	}
}

func TestHypoteketCrawler_parseAverageRates(t *testing.T) {
	t.Parallel()

	payloadJSON := crawlertest.LoadGoldenFile(t, "testdata/hypoteket_rates.json")
	logger := zap.NewNop()
	crawlTime := time.Date(2025, 12, 1, 10, 0, 0, 0, time.UTC)

	crawler := &HypoteketCrawler{logger: logger}

	results, err := crawler.parseAverageRates(payloadJSON, crawlTime)
	if err != nil {
		t.Fatalf("parseAverageRates() error = %v", err)
	}

	if len(results) == 0 {
		t.Fatal("parseAverageRates() returned no results")
	}

	// Verify we got multiple months of data
	months := make(map[string]bool)
	for _, r := range results {
		if r.AverageReferenceMonth != nil {
			key := r.AverageReferenceMonth.Month.String() + string(rune(r.AverageReferenceMonth.Year))
			months[key] = true
		}
		assertHypoteketAverageRateFields(t, r, crawlTime)
	}

	// Should have data for at least 10 months
	if len(months) < 10 {
		t.Errorf("got data for %d months, want at least 10", len(months))
	}
}

func TestParseHypoteketTermToTerm(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		term    string
		want    model.Term
		wantErr bool
	}{
		{
			name:    "3 months",
			term:    "threeMonth",
			want:    model.Term3months,
			wantErr: false,
		},
		{
			name:    "1 year",
			term:    "oneYear",
			want:    model.Term1year,
			wantErr: false,
		},
		{
			name:    "2 years",
			term:    "twoYear",
			want:    model.Term2years,
			wantErr: false,
		},
		{
			name:    "3 years",
			term:    "threeYear",
			want:    model.Term3years,
			wantErr: false,
		},
		{
			name:    "5 years",
			term:    "fiveYear",
			want:    model.Term5years,
			wantErr: false,
		},
		{
			name:    "unsupported term fourYear",
			term:    "fourYear",
			wantErr: true,
		},
		{
			name:    "unsupported term sevenYear",
			term:    "sevenYear",
			wantErr: true,
		},
		{
			name:    "unsupported term tenYear",
			term:    "tenYear",
			wantErr: true,
		},
		{
			name:    "invalid format",
			term:    "3_MONTHS",
			wantErr: true,
		},
		{
			name:    "empty string",
			term:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := parseHypoteketTermToTerm(tt.term)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseHypoteketTermToTerm() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("parseHypoteketTermToTerm() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseHypoteketPeriod(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     string
		wantYear  uint
		wantMonth time.Month
		wantErr   bool
	}{
		{
			name:      "valid period November 2025",
			input:     "2025-11",
			wantYear:  2025,
			wantMonth: time.November,
			wantErr:   false,
		},
		{
			name:      "valid period January 2025",
			input:     "2025-01",
			wantYear:  2025,
			wantMonth: time.January,
			wantErr:   false,
		},
		{
			name:      "valid period December 2024",
			input:     "2024-12",
			wantYear:  2024,
			wantMonth: time.December,
			wantErr:   false,
		},
		{
			name:    "invalid format - with day",
			input:   "2025-11-30",
			wantErr: true,
		},
		{
			name:    "invalid format - wrong separator",
			input:   "2025/11",
			wantErr: true,
		},
		{
			name:    "invalid month 00",
			input:   "2025-00",
			wantErr: true,
		},
		{
			name:    "invalid month 13",
			input:   "2025-13",
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

			got, err := parseHypoteketPeriod(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseHypoteketPeriod() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got.Year != tt.wantYear {
					t.Errorf("parseHypoteketPeriod() year = %v, want %v", got.Year, tt.wantYear)
				}
				if got.Month != tt.wantMonth {
					t.Errorf("parseHypoteketPeriod() month = %v, want %v", got.Month, tt.wantMonth)
				}
			}
		})
	}
}

func TestNewHypoteketCrawler(t *testing.T) {
	t.Parallel()

	mockClient := &httpmock.ClientMock{}
	logger := zap.NewNop()

	crawler := NewHypoteketCrawler(mockClient, logger)

	if crawler == nil {
		t.Fatal("NewHypoteketCrawler() returned nil")
	}
	if crawler.httpClient != mockClient {
		t.Error("httpClient not set correctly")
	}
	if crawler.logger != logger {
		t.Error("logger not set correctly")
	}
}

func TestHypoteketCrawler_InterfaceCompliance(t *testing.T) {
	t.Parallel()

	// Compile-time check that HypoteketCrawler implements SiteCrawler
	var _ crawlertest.SiteCrawler = &HypoteketCrawler{}
}

// assertHypoteketListRateFields validates common fields for Hypoteket list rate results.
func assertHypoteketListRateFields(t *testing.T, r model.InterestSet, crawlTime time.Time) {
	t.Helper()

	if r.Bank != hypoteketBankName {
		t.Errorf("Bank = %q, want %q", r.Bank, hypoteketBankName)
	}
	if r.Type != model.TypeListRate {
		t.Errorf("Type = %q, want %q", r.Type, model.TypeListRate)
	}
	if r.NominalRate <= 0 {
		t.Errorf("NominalRate = %f, want positive value", r.NominalRate)
	}
	// Hypoteket API provides change dates (validFrom)
	if r.ChangedOn == nil {
		t.Error("ChangedOn is nil, want non-nil for Hypoteket")
	}
	if r.LastCrawledAt != crawlTime {
		t.Errorf("LastCrawledAt = %v, want %v", r.LastCrawledAt, crawlTime)
	}
}

// assertHypoteketAverageRateFields validates common fields for Hypoteket average rate results.
func assertHypoteketAverageRateFields(t *testing.T, r model.InterestSet, crawlTime time.Time) {
	t.Helper()

	if r.Bank != hypoteketBankName {
		t.Errorf("Bank = %q, want %q", r.Bank, hypoteketBankName)
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
