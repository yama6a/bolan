//nolint:revive,nolintlint,dupl // package name matches the package being tested; test patterns intentionally similar across crawlers
package avanza

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
func TestAvanzaCrawler_Crawl(t *testing.T) {
	t.Parallel()

	stabeloJSON := crawlertest.LoadGoldenFile(t, "testdata/avanza_stabelo_rates.json")
	lhbJSON := crawlertest.LoadGoldenFile(t, "testdata/avanza_lhb_rates.json")
	logger := zap.NewNop()

	tests := []struct {
		name             string
		mockFetch        func(url string, headers map[string]string) (string, error)
		wantMinResults   int
		wantStabeloTerms int
		wantLHBTerms     int
	}{
		{
			name: "successful crawl extracts rates from both partners",
			mockFetch: func(url string, _ map[string]string) (string, error) {
				if url == avanzaStabeloRatesURL {
					return stabeloJSON, nil
				}
				return lhbJSON, nil
			},
			wantMinResults:   10, // 6 Stabelo + 6 LHB terms (some overlap)
			wantStabeloTerms: 6,  // 3 mån, 1 år, 2 år, 3 år, 5 år, 10 år
			wantLHBTerms:     6,  // 3 mån, 1 år, 2 år, 3 år, 4 år, 5 år
		},
		{
			name: "Stabelo fetch error still returns LHB rates",
			mockFetch: func(url string, _ map[string]string) (string, error) {
				if url == avanzaStabeloRatesURL {
					return "", errors.New("network error")
				}
				return lhbJSON, nil
			},
			wantMinResults:   6,
			wantStabeloTerms: 0,
			wantLHBTerms:     6,
		},
		{
			name: "LHB fetch error still returns Stabelo rates",
			mockFetch: func(url string, _ map[string]string) (string, error) {
				if url == avanzaStabeloRatesURL {
					return stabeloJSON, nil
				}
				return "", errors.New("network error")
			},
			wantMinResults:   6,
			wantStabeloTerms: 6,
			wantLHBTerms:     0,
		},
		{
			name: "both fetch errors return no results",
			mockFetch: func(_ string, _ map[string]string) (string, error) {
				return "", errors.New("network error")
			},
			wantMinResults:   0,
			wantStabeloTerms: 0,
			wantLHBTerms:     0,
		},
		{
			name: "invalid JSON returns no Stabelo rates",
			mockFetch: func(url string, _ map[string]string) (string, error) {
				if url == avanzaStabeloRatesURL {
					return "{invalid}", nil
				}
				return lhbJSON, nil
			},
			wantMinResults:   6,
			wantStabeloTerms: 0,
			wantLHBTerms:     6,
		},
		{
			name: "empty response returns no results",
			mockFetch: func(_ string, _ map[string]string) (string, error) {
				return `{"rows":[]}`, nil
			},
			wantMinResults:   0,
			wantStabeloTerms: 0,
			wantLHBTerms:     0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockClient := &httpmock.ClientMock{
				FetchFunc: tt.mockFetch,
			}

			crawler := NewAvanzaCrawler(mockClient, logger)
			resultChan := make(chan model.InterestSet, 100)

			crawler.Crawl(resultChan)
			close(resultChan)

			var results []model.InterestSet
			for set := range resultChan {
				results = append(results, set)
			}

			if len(results) < tt.wantMinResults {
				t.Errorf("got %d results, want at least %d", len(results), tt.wantMinResults)
			}

			crawlertest.AssertBankName(t, results, avanzaBankName)

			// All results should be list rates (no average rates from Avanza)
			listCount, avgCount := crawlertest.CountRatesByType(results)
			if avgCount != 0 {
				t.Errorf("got %d average rates, want 0", avgCount)
			}
			if listCount != len(results) {
				t.Errorf("got %d list rates out of %d total, want all to be list rates", listCount, len(results))
			}
		})
	}
}

func TestAvanzaCrawler_fetchRates(t *testing.T) {
	t.Parallel()

	stabeloJSON := crawlertest.LoadGoldenFile(t, "testdata/avanza_stabelo_rates.json")
	logger := zap.NewNop()
	crawlTime := time.Date(2025, 12, 6, 10, 0, 0, 0, time.UTC)

	mockClient := &httpmock.ClientMock{
		FetchFunc: func(_ string, _ map[string]string) (string, error) {
			return stabeloJSON, nil
		},
	}

	crawler := &AvanzaCrawler{httpClient: mockClient, logger: logger}

	results, err := crawler.fetchRates(avanzaStabeloRatesURL, "Stabelo", crawlTime)
	if err != nil {
		t.Fatalf("fetchRates() error = %v", err)
	}

	listCount, avgCount := crawlertest.CountRatesByType(results)
	if listCount != 6 {
		t.Errorf("fetchRates() returned %d list rates, want 6", listCount)
	}
	if avgCount != 0 {
		t.Errorf("fetchRates() returned %d average rates, want 0", avgCount)
	}

	// Verify expected terms are present for Stabelo
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
		assertAvanzaListRateFields(t, r, crawlTime)
	}

	for term, found := range expectedTerms {
		if !found {
			t.Errorf("missing term %q in Stabelo results", term)
		}
	}
}

func TestAvanzaCrawler_fetchRates_LHB(t *testing.T) {
	t.Parallel()

	lhbJSON := crawlertest.LoadGoldenFile(t, "testdata/avanza_lhb_rates.json")
	logger := zap.NewNop()
	crawlTime := time.Date(2025, 12, 6, 10, 0, 0, 0, time.UTC)

	mockClient := &httpmock.ClientMock{
		FetchFunc: func(_ string, _ map[string]string) (string, error) {
			return lhbJSON, nil
		},
	}

	crawler := &AvanzaCrawler{httpClient: mockClient, logger: logger}

	results, err := crawler.fetchRates(avanzaLHBRatesURL, "Landshypotek", crawlTime)
	if err != nil {
		t.Fatalf("fetchRates() error = %v", err)
	}

	listCount, avgCount := crawlertest.CountRatesByType(results)
	if listCount != 6 {
		t.Errorf("fetchRates() returned %d list rates, want 6", listCount)
	}
	if avgCount != 0 {
		t.Errorf("fetchRates() returned %d average rates, want 0", avgCount)
	}

	// Verify expected terms are present for Landshypotek
	expectedTerms := map[model.Term]bool{
		model.Term3months: false,
		model.Term1year:   false,
		model.Term2years:  false,
		model.Term3years:  false,
		model.Term4years:  false,
		model.Term5years:  false,
	}

	for _, r := range results {
		if _, ok := expectedTerms[r.Term]; ok {
			expectedTerms[r.Term] = true
		}
		assertAvanzaListRateFields(t, r, crawlTime)
	}

	for term, found := range expectedTerms {
		if !found {
			t.Errorf("missing term %q in Landshypotek results", term)
		}
	}
}

func TestParseAvanzaBindingPeriod(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		period  string
		want    model.Term
		wantErr bool
	}{
		{
			name:    "THREE_MONTHS",
			period:  "THREE_MONTHS",
			want:    model.Term3months,
			wantErr: false,
		},
		{
			name:    "ONE_YEAR",
			period:  "ONE_YEAR",
			want:    model.Term1year,
			wantErr: false,
		},
		{
			name:    "TWO_YEARS",
			period:  "TWO_YEARS",
			want:    model.Term2years,
			wantErr: false,
		},
		{
			name:    "THREE_YEARS",
			period:  "THREE_YEARS",
			want:    model.Term3years,
			wantErr: false,
		},
		{
			name:    "FOUR_YEARS",
			period:  "FOUR_YEARS",
			want:    model.Term4years,
			wantErr: false,
		},
		{
			name:    "FIVE_YEARS",
			period:  "FIVE_YEARS",
			want:    model.Term5years,
			wantErr: false,
		},
		{
			name:    "TEN_YEARS",
			period:  "TEN_YEARS",
			want:    model.Term10years,
			wantErr: false,
		},
		{
			name:    "unsupported period",
			period:  "SEVEN_YEARS",
			wantErr: true,
		},
		{
			name:    "empty string",
			period:  "",
			wantErr: true,
		},
		{
			name:    "lowercase",
			period:  "three_months",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := parseAvanzaBindingPeriod(tt.period)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseAvanzaBindingPeriod() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("parseAvanzaBindingPeriod() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAvanzaCrawler_findBaseRateRow(t *testing.T) {
	t.Parallel()

	logger := zap.NewNop()
	crawler := &AvanzaCrawler{logger: logger}

	tests := []struct {
		name     string
		rows     []avanzaRateRow
		wantNil  bool
		wantLTV  float64
		wantLoan int
	}{
		{
			name: "finds base rate row",
			rows: []avanzaRateRow{
				{MinLoanToValue: 60.01, MinLoanAmount: 0},
				{MinLoanToValue: 0, MinLoanAmount: 0},
				{MinLoanToValue: 0, MinLoanAmount: 500000},
			},
			wantNil:  false,
			wantLTV:  0,
			wantLoan: 0,
		},
		{
			name: "base rate row is first",
			rows: []avanzaRateRow{
				{MinLoanToValue: 0, MinLoanAmount: 0},
				{MinLoanToValue: 60.01, MinLoanAmount: 0},
			},
			wantNil:  false,
			wantLTV:  0,
			wantLoan: 0,
		},
		{
			name: "no base rate row",
			rows: []avanzaRateRow{
				{MinLoanToValue: 60.01, MinLoanAmount: 0},
				{MinLoanToValue: 0, MinLoanAmount: 500000},
			},
			wantNil: true,
		},
		{
			name:    "empty rows",
			rows:    []avanzaRateRow{},
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := crawler.findBaseRateRow(tt.rows)

			if tt.wantNil {
				if got != nil {
					t.Errorf("findBaseRateRow() = %v, want nil", got)
				}
				return
			}

			if got == nil {
				t.Fatal("findBaseRateRow() returned nil, want non-nil")
			}

			if got.MinLoanToValue != tt.wantLTV {
				t.Errorf("MinLoanToValue = %v, want %v", got.MinLoanToValue, tt.wantLTV)
			}
			if got.MinLoanAmount != tt.wantLoan {
				t.Errorf("MinLoanAmount = %v, want %v", got.MinLoanAmount, tt.wantLoan)
			}
		})
	}
}

func TestNewAvanzaCrawler(t *testing.T) {
	t.Parallel()

	mockClient := &httpmock.ClientMock{}
	logger := zap.NewNop()

	crawler := NewAvanzaCrawler(mockClient, logger)

	if crawler == nil {
		t.Fatal("NewAvanzaCrawler() returned nil")
	}
	if crawler.httpClient != mockClient {
		t.Error("httpClient not set correctly")
	}
	if crawler.logger != logger {
		t.Error("logger not set correctly")
	}
}

func TestAvanzaCrawler_InterfaceCompliance(t *testing.T) {
	t.Parallel()

	// Compile-time check that AvanzaCrawler implements SiteCrawler
	var _ crawlertest.SiteCrawler = &AvanzaCrawler{}
}

// assertAvanzaListRateFields validates common fields for Avanza list rate results.
func assertAvanzaListRateFields(t *testing.T, r model.InterestSet, crawlTime time.Time) {
	t.Helper()

	if r.Bank != avanzaBankName {
		t.Errorf("Bank = %q, want %q", r.Bank, avanzaBankName)
	}
	if r.Type != model.TypeListRate {
		t.Errorf("Type = %q, want %q", r.Type, model.TypeListRate)
	}
	if r.NominalRate <= 0 {
		t.Errorf("NominalRate = %f, want positive value", r.NominalRate)
	}
	// Avanza API doesn't provide change dates
	if r.ChangedOn != nil {
		t.Error("ChangedOn is not nil, want nil for Avanza")
	}
	// Avanza doesn't have average rates
	if r.AverageReferenceMonth != nil {
		t.Error("AverageReferenceMonth is not nil, want nil for Avanza")
	}
	if r.LastCrawledAt != crawlTime {
		t.Errorf("LastCrawledAt = %v, want %v", r.LastCrawledAt, crawlTime)
	}
}
