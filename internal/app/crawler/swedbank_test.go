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

// assertSwedbankListRateFields validates common fields for Swedbank list rate results.
func assertSwedbankListRateFields(t *testing.T, r model.InterestSet, crawlTime time.Time) {
	t.Helper()

	if r.Bank != swedbankBankName {
		t.Errorf("Bank = %q, want %q", r.Bank, swedbankBankName)
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

// assertSwedbankAverageRateFields validates common fields for Swedbank average rate results.
func assertSwedbankAverageRateFields(t *testing.T, r model.InterestSet, crawlTime time.Time) {
	t.Helper()

	if r.Bank != swedbankBankName {
		t.Errorf("Bank = %q, want %q", r.Bank, swedbankBankName)
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

func TestSwedbankCrawler_Crawl(t *testing.T) {
	t.Parallel()

	listRatesHTML := loadGoldenFile(t, "testdata/swedbank.html")
	historicRatesHTML := loadGoldenFile(t, "testdata/swedbank_historic.html")
	logger := zap.NewNop()

	tests := []struct {
		name             string
		mockFetch        func(url string, headers map[string]string) (string, error)
		wantListRates    int
		wantAvgRatesMin  int // minimum average rates (historic page has many months)
		wantTotalMinimum int
	}{
		{
			name: "successful crawl extracts list and historic average rates",
			mockFetch: func(url string, _ map[string]string) (string, error) {
				if url == swedbankHistoricRatesURL {
					return historicRatesHTML, nil
				}
				return listRatesHTML, nil
			},
			wantListRates:    11,  // 3 månader, 1-10 år (excluding Banklån)
			wantAvgRatesMin:  100, // Many months * 11 terms (minus missing data)
			wantTotalMinimum: 111, // 11 list rates + at least 100 average rates
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
		{
			name: "list rates only when historic fetch fails",
			mockFetch: func(url string, _ map[string]string) (string, error) {
				if url == swedbankHistoricRatesURL {
					return "", errors.New("network error")
				}
				return listRatesHTML, nil
			},
			wantListRates:    11,
			wantAvgRatesMin:  0,
			wantTotalMinimum: 11,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockClient := &httpmock.ClientMock{
				FetchFunc: tt.mockFetch,
			}

			crawler := NewSwedbankCrawler(mockClient, logger)
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

			assertBankName(t, results, swedbankBankName)
		})
	}
}

func TestSwedbankCrawler_extractListRates(t *testing.T) {
	t.Parallel()

	goldenHTML := loadGoldenFile(t, "testdata/swedbank.html")
	logger := zap.NewNop()
	crawler := &SwedbankCrawler{logger: logger}
	crawlTime := time.Date(2025, 12, 1, 10, 0, 0, 0, time.UTC)

	results, err := crawler.extractListRates(goldenHTML, crawlTime)
	if err != nil {
		t.Fatalf("extractListRates() error = %v", err)
	}

	if len(results) != 11 {
		t.Errorf("extractListRates() returned %d results, want 11", len(results))
	}

	// Verify expected terms are present (Swedbank has full range 3 månader, 1-10 år)
	expectedTerms := map[model.Term]bool{
		model.Term3months: false,
		model.Term1year:   false,
		model.Term2years:  false,
		model.Term3years:  false,
		model.Term4years:  false,
		model.Term5years:  false,
		model.Term6years:  false,
		model.Term7years:  false,
		model.Term8years:  false,
		model.Term9years:  false,
		model.Term10years: false,
	}

	for _, r := range results {
		if _, ok := expectedTerms[r.Term]; ok {
			expectedTerms[r.Term] = true
		}
		assertSwedbankListRateFields(t, r, crawlTime)
	}

	for term, found := range expectedTerms {
		if !found {
			t.Errorf("missing term %q in results", term)
		}
	}

	// Verify change date is parsed correctly (25 september 2025)
	if results[0].ChangedOn != nil {
		expectedDate := time.Date(2025, time.September, 25, 0, 0, 0, 0, time.UTC)
		if !results[0].ChangedOn.Equal(expectedDate) {
			t.Errorf("ChangedOn = %v, want %v", results[0].ChangedOn, expectedDate)
		}
	}
}

func TestSwedbankCrawler_extractHistoricAverageRates(t *testing.T) {
	t.Parallel()

	historicHTML := loadGoldenFile(t, "testdata/swedbank_historic.html")
	logger := zap.NewNop()
	crawler := &SwedbankCrawler{logger: logger}
	crawlTime := time.Date(2025, 12, 1, 10, 0, 0, 0, time.UTC)

	results, err := crawler.extractHistoricAverageRates(historicHTML, crawlTime)
	if err != nil {
		t.Fatalf("extractHistoricAverageRates() error = %v", err)
	}

	// Should have many results (many months * 11 terms minus missing data)
	if len(results) < 100 {
		t.Errorf("extractHistoricAverageRates() returned %d results, want at least 100", len(results))
	}

	// Verify all results have valid average rate fields
	for _, r := range results {
		assertSwedbankAverageRateFields(t, r, crawlTime)
	}

	// Verify we have data for multiple months
	monthsSeen := make(map[string]bool)
	for _, r := range results {
		key := r.AverageReferenceMonth.Month.String() + string(rune(r.AverageReferenceMonth.Year))
		monthsSeen[key] = true
	}
	if len(monthsSeen) < 10 {
		t.Errorf("only found %d different months, expected at least 10", len(monthsSeen))
	}

	// Verify November 2025 data exists (first row in the table)
	foundNov2025 := false
	for _, r := range results {
		if r.AverageReferenceMonth.Month == time.November && r.AverageReferenceMonth.Year == 2025 {
			foundNov2025 = true
			break
		}
	}
	if !foundNov2025 {
		t.Error("expected to find November 2025 data")
	}
}

func TestParseSwedbankRate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    float32
		wantErr bool
	}{
		{
			name:    "Swedish format with comma and percent",
			input:   "3,79 %",
			want:    3.79,
			wantErr: false,
		},
		{
			name:    "Swedish format with comma",
			input:   "2,63",
			want:    2.63,
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

			got, err := parseSwedbankRate(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseSwedbankRate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("parseSwedbankRate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseSwedbankListDate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    time.Time
		wantErr bool
	}{
		{
			name:    "valid header format",
			input:   "Ränta, senast ändrad 25 september 2025",
			want:    time.Date(2025, time.September, 25, 0, 0, 0, 0, time.UTC),
			wantErr: false,
		},
		{
			name:    "january",
			input:   "senast ändrad 1 januari 2024",
			want:    time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC),
			wantErr: false,
		},
		{
			name:    "december",
			input:   "Ränta, senast ändrad 31 december 2023",
			want:    time.Date(2023, time.December, 31, 0, 0, 0, 0, time.UTC),
			wantErr: false,
		},
		{
			name:    "missing pattern",
			input:   "Ränta 2025",
			wantErr: true,
		},
		{
			name:    "invalid month",
			input:   "senast ändrad 1 invalidmonth 2024",
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

			got, err := parseSwedbankListDate(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseSwedbankListDate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !got.Equal(tt.want) {
				t.Errorf("parseSwedbankListDate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseSwedbankAvgMonth(t *testing.T) {
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
			input:     "Genomsnittsränta, november 2025",
			wantMonth: time.November,
			wantYear:  2025,
			wantErr:   false,
		},
		{
			name:      "December 2024 format",
			input:     "Genomsnittsränta, december 2024",
			wantMonth: time.December,
			wantYear:  2024,
			wantErr:   false,
		},
		{
			name:      "January format",
			input:     "januari 2025",
			wantMonth: time.January,
			wantYear:  2025,
			wantErr:   false,
		},
		{
			name:      "with extra spaces",
			input:     "  Genomsnittsränta,   mars  2024  ",
			wantMonth: time.March,
			wantYear:  2024,
			wantErr:   false,
		},
		{
			name:    "invalid format - missing year",
			input:   "november",
			wantErr: true,
		},
		{
			name:    "invalid month",
			input:   "invalidmonth 2024",
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

			got, err := parseSwedbankAvgMonth(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseSwedbankAvgMonth() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got.Month != tt.wantMonth {
					t.Errorf("parseSwedbankAvgMonth() month = %v, want %v", got.Month, tt.wantMonth)
				}
				if got.Year != tt.wantYear {
					t.Errorf("parseSwedbankAvgMonth() year = %v, want %v", got.Year, tt.wantYear)
				}
			}
		})
	}
}

func TestParseSwedbankHistoricMonth(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     string
		wantMonth time.Month
		wantYear  uint
		wantErr   bool
	}{
		{
			name:      "November abbreviated",
			input:     "nov. 2025",
			wantMonth: time.November,
			wantYear:  2025,
			wantErr:   false,
		},
		{
			name:      "October abbreviated",
			input:     "okt. 2025",
			wantMonth: time.October,
			wantYear:  2025,
			wantErr:   false,
		},
		{
			name:      "September abbreviated",
			input:     "sep. 2025",
			wantMonth: time.September,
			wantYear:  2025,
			wantErr:   false,
		},
		{
			name:      "January abbreviated",
			input:     "jan. 2024",
			wantMonth: time.January,
			wantYear:  2024,
			wantErr:   false,
		},
		{
			name:      "December abbreviated",
			input:     "dec. 2023",
			wantMonth: time.December,
			wantYear:  2023,
			wantErr:   false,
		},
		{
			name:      "April abbreviated",
			input:     "apr. 2024",
			wantMonth: time.April,
			wantYear:  2024,
			wantErr:   false,
		},
		{
			name:      "February abbreviated",
			input:     "feb. 2025",
			wantMonth: time.February,
			wantYear:  2025,
			wantErr:   false,
		},
		{
			name:      "August abbreviated",
			input:     "aug. 2024",
			wantMonth: time.August,
			wantYear:  2024,
			wantErr:   false,
		},
		{
			name:    "invalid month",
			input:   "xyz. 2024",
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

			got, err := parseSwedbankHistoricMonth(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseSwedbankHistoricMonth() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got.Month != tt.wantMonth {
					t.Errorf("parseSwedbankHistoricMonth() month = %v, want %v", got.Month, tt.wantMonth)
				}
				if got.Year != tt.wantYear {
					t.Errorf("parseSwedbankHistoricMonth() year = %v, want %v", got.Year, tt.wantYear)
				}
			}
		})
	}
}

func TestNewSwedbankCrawler(t *testing.T) {
	t.Parallel()

	mockClient := &httpmock.ClientMock{}
	logger := zap.NewNop()

	crawler := NewSwedbankCrawler(mockClient, logger)

	if crawler == nil {
		t.Fatal("NewSwedbankCrawler() returned nil")
	}
	if crawler.httpClient != mockClient {
		t.Error("httpClient not set correctly")
	}
	if crawler.logger != logger {
		t.Error("logger not set correctly")
	}
}

func TestSwedbankCrawler_InterfaceCompliance(t *testing.T) {
	t.Parallel()

	// Compile-time check that SwedbankCrawler implements SiteCrawler
	var _ SiteCrawler = &SwedbankCrawler{}
}
