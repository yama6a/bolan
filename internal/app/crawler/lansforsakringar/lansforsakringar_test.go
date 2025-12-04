//nolint:revive,nolintlint,dupl // package name matches the package being tested; test patterns intentionally similar across crawlers
package lansforsakringar

import (
	"errors"
	"os"
	"testing"
	"time"

	crawlertest "github.com/yama6a/bolan-compare/internal/app/crawler"
	"github.com/yama6a/bolan-compare/internal/pkg/http/httpmock"
	"github.com/yama6a/bolan-compare/internal/pkg/model"
	"go.uber.org/zap"
)

// assertLFListRateFields validates common fields for Länsförsäkringar list rate results.
func assertLFListRateFields(t *testing.T, r model.InterestSet, crawlTime time.Time) {
	t.Helper()

	if r.Bank != lfBankName {
		t.Errorf("Bank = %q, want %q", r.Bank, lfBankName)
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

func TestLansforsakringarCrawler_Crawl(t *testing.T) {
	t.Parallel()

	ratesHTML := crawlertest.LoadGoldenFile(t, "testdata/lansforsakringar_rates.html")
	ratesPDF, err := os.ReadFile("testdata/lansforsakringar_avg_rates.pdf")
	if err != nil {
		t.Fatalf("failed to load PDF golden file: %v", err)
	}
	logger := zap.NewNop()

	tests := []struct {
		name             string
		mockFetch        func(url string, headers map[string]string) (string, error)
		mockFetchRaw     func(url string, headers map[string]string) ([]byte, error)
		wantListRates    int
		wantAvgRatesMin  int
		wantTotalMinimum int
	}{
		{
			name: "successful crawl extracts list and average rates",
			mockFetch: func(_ string, _ map[string]string) (string, error) {
				return ratesHTML, nil
			},
			mockFetchRaw: func(_ string, _ map[string]string) ([]byte, error) {
				return ratesPDF, nil
			},
			wantListRates:    8,   // 3 mån, 1-5 år, 7 år, 10 år
			wantAvgRatesMin:  100, // PDF has ~130 rows × 8 terms, some empty
			wantTotalMinimum: 100,
		},
		{
			name: "fetch error returns no results",
			mockFetch: func(_ string, _ map[string]string) (string, error) {
				return "", errors.New("network error")
			},
			mockFetchRaw: func(_ string, _ map[string]string) ([]byte, error) {
				return nil, errors.New("network error")
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
				FetchFunc:    tt.mockFetch,
				FetchRawFunc: tt.mockFetchRaw,
			}

			crawler := NewLansforsakringarCrawler(mockClient, logger)
			resultChan := make(chan model.InterestSet, 2000)

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

			crawlertest.AssertBankName(t, results, lfBankName)
		})
	}
}

func TestLansforsakringarCrawler_extractListRates(t *testing.T) {
	t.Parallel()

	goldenHTML := crawlertest.LoadGoldenFile(t, "testdata/lansforsakringar_rates.html")
	logger := zap.NewNop()
	crawler := &LansforsakringarCrawler{logger: logger}
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
		assertLFListRateFields(t, r, crawlTime)
	}

	for term, found := range expectedTerms {
		if !found {
			t.Errorf("missing term %q in results", term)
		}
	}

	// Verify change date is parsed correctly (most recent date in test data)
	for _, r := range results {
		if r.ChangedOn == nil {
			t.Errorf("ChangedOn is nil for term %q", r.Term)
		}
	}
}

func TestLansforsakringarCrawler_parsePDF(t *testing.T) {
	t.Parallel()

	pdfContent, err := os.ReadFile("testdata/lansforsakringar_avg_rates.pdf")
	if err != nil {
		t.Fatalf("failed to load PDF golden file: %v", err)
	}
	logger := zap.NewNop()
	crawler := &LansforsakringarCrawler{logger: logger}
	crawlTime := time.Date(2025, 12, 1, 10, 0, 0, 0, time.UTC)

	results, err := crawler.parsePDF(pdfContent, crawlTime)
	if err != nil {
		t.Fatalf("parsePDF() error = %v", err)
	}

	// PDF has ~130 rows with 8 terms each, but some cells are empty
	// Expect at least 700 average rate results
	if len(results) < 700 {
		t.Errorf("parsePDF() returned %d results, want at least 700", len(results))
	}

	// Count unique months and verify terms
	uniqueMonths := countUniqueMonths(results)
	if len(uniqueMonths) < 100 {
		t.Errorf("parsePDF() found %d unique months, want at least 100", len(uniqueMonths))
	}

	// Verify expected terms are present
	foundTerms := collectFoundTerms(results)
	expectedTerms := []model.Term{
		model.Term3months, model.Term1year, model.Term2years, model.Term3years,
		model.Term4years, model.Term5years, model.Term7years, model.Term10years,
	}
	for _, term := range expectedTerms {
		if !foundTerms[term] {
			t.Errorf("missing term %q in results", term)
		}
	}

	// Validate first result as sample
	validatePDFResult(t, results[0])
}

func countUniqueMonths(results []model.InterestSet) map[string]bool {
	uniqueMonths := make(map[string]bool)
	for _, r := range results {
		if r.AverageReferenceMonth != nil {
			key := r.AverageReferenceMonth.Month.String() + string(rune(r.AverageReferenceMonth.Year))
			uniqueMonths[key] = true
		}
	}
	return uniqueMonths
}

func collectFoundTerms(results []model.InterestSet) map[model.Term]bool {
	found := make(map[model.Term]bool)
	for _, r := range results {
		found[r.Term] = true
	}
	return found
}

func validatePDFResult(t *testing.T, r model.InterestSet) {
	t.Helper()
	if r.Type != model.TypeAverageRate {
		t.Errorf("Type = %q, want %q", r.Type, model.TypeAverageRate)
	}
	if r.Bank != lfBankName {
		t.Errorf("Bank = %q, want %q", r.Bank, lfBankName)
	}
	if r.AverageReferenceMonth == nil {
		t.Error("AverageReferenceMonth is nil, want non-nil")
	}
	if r.NominalRate <= 0 {
		t.Errorf("NominalRate = %f, want positive value", r.NominalRate)
	}
}

func TestParseLFRate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    float32
		wantErr bool
	}{
		{
			name:    "Swedish format with comma and percent",
			input:   "3,84 %",
			want:    3.84,
			wantErr: false,
		},
		{
			name:    "Swedish format with comma",
			input:   "2,70",
			want:    2.70,
			wantErr: false,
		},
		{
			name:    "with extra whitespace",
			input:   "  3.34 %  ",
			want:    3.34,
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
			name:    "dash (no data)",
			input:   "-",
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

			got, err := parseLFRate(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseLFRate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("parseLFRate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseLFListDate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    time.Time
		wantErr bool
	}{
		{
			name:    "standard YYYY-MM-DD format",
			input:   "2025-10-02",
			want:    time.Date(2025, time.October, 2, 0, 0, 0, 0, time.UTC),
			wantErr: false,
		},
		{
			name:    "another valid date",
			input:   "2025-07-11",
			want:    time.Date(2025, time.July, 11, 0, 0, 0, 0, time.UTC),
			wantErr: false,
		},
		{
			name:    "with surrounding whitespace",
			input:   "  2024-01-15  ",
			want:    time.Date(2024, time.January, 15, 0, 0, 0, 0, time.UTC),
			wantErr: false,
		},
		{
			name:    "invalid format",
			input:   "02-10-2025",
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

			got, err := parseLFListDate(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseLFListDate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !got.Equal(tt.want) {
				t.Errorf("parseLFListDate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseLFAvgMonth(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     string
		wantMonth time.Month
		wantYear  uint
		wantErr   bool
	}{
		{
			name:      "October 2025 format",
			input:     "Genomsnittlig ränta oktober 2025",
			wantMonth: time.October,
			wantYear:  2025,
			wantErr:   false,
		},
		{
			name:      "November 2024 format",
			input:     "Genomsnittlig ränta november 2024",
			wantMonth: time.November,
			wantYear:  2024,
			wantErr:   false,
		},
		{
			name:      "January format",
			input:     "Genomsnittlig ränta januari 2025",
			wantMonth: time.January,
			wantYear:  2025,
			wantErr:   false,
		},
		{
			name:      "with extra spaces",
			input:     "  Genomsnittlig   ränta   mars  2024  ",
			wantMonth: time.March,
			wantYear:  2024,
			wantErr:   false,
		},
		{
			name:    "invalid format - missing month",
			input:   "Genomsnittlig ränta 2024",
			wantErr: true,
		},
		{
			name:    "invalid month",
			input:   "Genomsnittlig ränta invalidmonth 2024",
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

			got, err := parseLFAvgMonth(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseLFAvgMonth() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got.Month != tt.wantMonth {
					t.Errorf("parseLFAvgMonth() month = %v, want %v", got.Month, tt.wantMonth)
				}
				if got.Year != tt.wantYear {
					t.Errorf("parseLFAvgMonth() year = %v, want %v", got.Year, tt.wantYear)
				}
			}
		})
	}
}

func TestLfSwedishMonthToTime(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input  string
		want   time.Month
		wantOK bool
	}{
		{"januari", time.January, true},
		{"februari", time.February, true},
		{"mars", time.March, true},
		{"april", time.April, true},
		{"maj", time.May, true},
		{"juni", time.June, true},
		{"juli", time.July, true},
		{"augusti", time.August, true},
		{"september", time.September, true},
		{"oktober", time.October, true},
		{"november", time.November, true},
		{"december", time.December, true},
		{"invalid", 0, false},
		{"", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()

			got, ok := lfSwedishMonthToTime(tt.input)
			if ok != tt.wantOK {
				t.Errorf("lfSwedishMonthToTime() ok = %v, want %v", ok, tt.wantOK)
			}
			if tt.wantOK && got != tt.want {
				t.Errorf("lfSwedishMonthToTime() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewLansforsakringarCrawler(t *testing.T) {
	t.Parallel()

	mockClient := &httpmock.ClientMock{}
	logger := zap.NewNop()

	crawler := NewLansforsakringarCrawler(mockClient, logger)

	if crawler == nil {
		t.Fatal("NewLansforsakringarCrawler() returned nil")
	}
	if crawler.httpClient != mockClient {
		t.Error("httpClient not set correctly")
	}
	if crawler.logger != logger {
		t.Error("logger not set correctly")
	}
}

func TestLansforsakringarCrawler_InterfaceCompliance(t *testing.T) {
	t.Parallel()

	// Compile-time check that LansforsakringarCrawler implements SiteCrawler
	var _ crawlertest.SiteCrawler = &LansforsakringarCrawler{}
}

func TestParseLFPDFDate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     string
		wantMonth time.Month
		wantYear  uint
		wantOK    bool
	}{
		{
			name:      "October 2025",
			input:     "20251031",
			wantMonth: time.October,
			wantYear:  2025,
			wantOK:    true,
		},
		{
			name:      "January 2024",
			input:     "20240115",
			wantMonth: time.January,
			wantYear:  2024,
			wantOK:    true,
		},
		{
			name:      "December 2020",
			input:     "20201231",
			wantMonth: time.December,
			wantYear:  2020,
			wantOK:    true,
		},
		{
			name:   "invalid month 13",
			input:  "20251331",
			wantOK: false,
		},
		{
			name:   "invalid month 00",
			input:  "20250031",
			wantOK: false,
		},
		{
			name:   "too short",
			input:  "2025103",
			wantOK: false,
		},
		{
			name:   "too long",
			input:  "202510311",
			wantOK: false,
		},
		{
			name:   "non-numeric",
			input:  "2025abcd",
			wantOK: false,
		},
		{
			name:   "empty",
			input:  "",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, ok := parseLFPDFDate(tt.input)
			if ok != tt.wantOK {
				t.Errorf("parseLFPDFDate() ok = %v, want %v", ok, tt.wantOK)
				return
			}
			if tt.wantOK {
				if got.Month != tt.wantMonth {
					t.Errorf("parseLFPDFDate() month = %v, want %v", got.Month, tt.wantMonth)
				}
				if got.Year != tt.wantYear {
					t.Errorf("parseLFPDFDate() year = %v, want %v", got.Year, tt.wantYear)
				}
			}
		})
	}
}

func TestExtractLFTermsFromHeader(t *testing.T) {
	t.Parallel()

	// Simulated PDF header text
	headerText := "Ränta procent 3 Månader 1 År 2 År 3 År 4 År 5 År 7 År 10 År"

	terms := extractLFTermsFromHeader(headerText)

	expectedTerms := []model.Term{
		model.Term3months,
		model.Term1year,
		model.Term2years,
		model.Term3years,
		model.Term4years,
		model.Term5years,
		model.Term7years,
		model.Term10years,
	}

	if len(terms) != len(expectedTerms) {
		t.Errorf("extractLFTermsFromHeader() returned %d terms, want %d", len(terms), len(expectedTerms))
	}

	for i, expected := range expectedTerms {
		if i >= len(terms) {
			break
		}
		if terms[i] != expected {
			t.Errorf("extractLFTermsFromHeader() term[%d] = %v, want %v", i, terms[i], expected)
		}
	}
}

func TestCollectLFPDFRates(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		tokens   []string
		startIdx int
		maxRates int
		want     []float32
	}{
		{
			name:     "typical row with all rates",
			tokens:   []string{"20251031", "2,70", "2,66", "3,07", "3,16", "3,31", "3,39", "3,85"},
			startIdx: 1,
			maxRates: 8,
			want:     []float32{2.70, 2.66, 3.07, 3.16, 3.31, 3.39, 3.85},
		},
		{
			name:     "stops at next date",
			tokens:   []string{"20251031", "2,70", "2,66", "20250930", "3,04"},
			startIdx: 1,
			maxRates: 8,
			want:     []float32{2.70, 2.66},
		},
		{
			name:     "respects maxRates",
			tokens:   []string{"20251031", "2,70", "2,66", "3,07", "3,16", "3,31"},
			startIdx: 1,
			maxRates: 3,
			want:     []float32{2.70, 2.66, 3.07},
		},
		{
			name:     "empty tokens",
			tokens:   []string{},
			startIdx: 0,
			maxRates: 8,
			want:     []float32{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := collectLFPDFRates(tt.tokens, tt.startIdx, tt.maxRates)

			if len(got) != len(tt.want) {
				t.Errorf("collectLFPDFRates() returned %d rates, want %d", len(got), len(tt.want))
				return
			}

			for i, wantRate := range tt.want {
				if got[i] != wantRate {
					t.Errorf("collectLFPDFRates() rate[%d] = %v, want %v", i, got[i], wantRate)
				}
			}
		})
	}
}
