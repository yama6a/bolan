//nolint:revive,nolintlint,dupl // package name matches the package being tested; test patterns intentionally similar across crawlers
package crawler

import (
	"errors"
	"os"
	"testing"
	"time"

	"github.com/yama6a/bolan-compare/internal/pkg/http/httpmock"
	"github.com/yama6a/bolan-compare/internal/pkg/model"
	"go.uber.org/zap"
)

// testInvalidJSON is a shared test constant for mocking invalid JSON responses.
const testInvalidJSON = "invalid json"

func loadGoldenFile(t *testing.T, filename string) string {
	t.Helper()

	data, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("failed to load golden file %s: %v", filename, err)
	}

	return string(data)
}

// countRatesByType counts list rates and average rates from results.
func countRatesByType(results []model.InterestSet) (listCount, avgCount int) {
	for _, r := range results {
		switch r.Type {
		case model.TypeListRate:
			listCount++
		case model.TypeAverageRate:
			avgCount++
		case model.TypeRatioDiscounted, model.TypeUnionDiscounted:
			// Not counted in this helper
		}
	}

	return listCount, avgCount
}

// assertBankName verifies all results have the expected bank name.
func assertBankName(t *testing.T, results []model.InterestSet, wantBank model.Bank) {
	t.Helper()

	for _, r := range results {
		if r.Bank != wantBank {
			t.Errorf("bank = %q, want %q", r.Bank, wantBank)
		}
	}
}

// assertListRateFields validates common fields for list rate results.
func assertListRateFields(t *testing.T, r model.InterestSet, crawlTime time.Time) {
	t.Helper()

	if r.Bank != danskeBankName {
		t.Errorf("Bank = %q, want %q", r.Bank, danskeBankName)
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

// assertAverageRateFields validates common fields for average rate results.
func assertAverageRateFields(t *testing.T, r model.InterestSet, crawlTime time.Time) {
	t.Helper()

	if r.Bank != danskeBankName {
		t.Errorf("Bank = %q, want %q", r.Bank, danskeBankName)
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

func TestDanskeBankCrawler_Crawl(t *testing.T) {
	t.Parallel()

	goldenHTML := loadGoldenFile(t, "testdata/danske_bank.html")
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
				return goldenHTML, nil
			},
			wantListRates:    8, // 3 mån, 1 år, 2 år, 3 år, 4 år, 5 år, 6 år, 10 år
			wantAvgRates:     true,
			wantTotalMinimum: 50, // at least 50 interest sets (8 list + many average)
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockClient := &httpmock.ClientMock{
				FetchFunc: tt.mockFetch,
			}

			crawler := NewDanskeBankCrawler(mockClient, logger)
			resultChan := make(chan model.InterestSet, 200)

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

			if tt.wantAvgRates && avgRateCount == 0 {
				t.Error("expected average rates but got none")
			}

			if len(results) < tt.wantTotalMinimum {
				t.Errorf("total results = %d, want at least %d", len(results), tt.wantTotalMinimum)
			}

			assertBankName(t, results, danskeBankName)
		})
	}
}

func TestDanskeBankCrawler_extractListRates(t *testing.T) {
	t.Parallel()

	goldenHTML := loadGoldenFile(t, "testdata/danske_bank.html")
	logger := zap.NewNop()
	crawler := &DanskeBankCrawler{logger: logger}
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
		model.Term6years:  false,
		model.Term10years: false,
	}

	for _, r := range results {
		if _, ok := expectedTerms[r.Term]; ok {
			expectedTerms[r.Term] = true
		}
		assertListRateFields(t, r, crawlTime)
	}

	for term, found := range expectedTerms {
		if !found {
			t.Errorf("missing term %q in results", term)
		}
	}
}

func TestDanskeBankCrawler_extractAverageRates(t *testing.T) {
	t.Parallel()

	goldenHTML := loadGoldenFile(t, "testdata/danske_bank.html")
	logger := zap.NewNop()
	crawler := &DanskeBankCrawler{logger: logger}
	crawlTime := time.Date(2025, 12, 1, 10, 0, 0, 0, time.UTC)

	results, err := crawler.extractAverageRates(goldenHTML, crawlTime)
	if err != nil {
		t.Fatalf("extractAverageRates() error = %v", err)
	}

	if len(results) == 0 {
		t.Fatal("extractAverageRates() returned no results")
	}

	// Verify we got multiple months of data
	months := make(map[string]bool)
	for _, r := range results {
		if r.AverageReferenceMonth != nil {
			key := r.AverageReferenceMonth.Month.String() + string(rune(r.AverageReferenceMonth.Year))
			months[key] = true
		}
		assertAverageRateFields(t, r, crawlTime)
	}

	// Should have data for multiple months
	if len(months) < 10 {
		t.Errorf("got data for %d months, want at least 10", len(months))
	}
}

func TestParseDanskeBankChangeDate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    time.Time
		wantErr bool
	}{
		{
			name:    "valid date YYYY-MM-DD",
			input:   "2025-09-26",
			want:    time.Date(2025, time.September, 26, 0, 0, 0, 0, time.UTC),
			wantErr: false,
		},
		{
			name:    "valid date beginning of year",
			input:   "2024-01-15",
			want:    time.Date(2024, time.January, 15, 0, 0, 0, 0, time.UTC),
			wantErr: false,
		},
		{
			name:    "valid date end of year",
			input:   "2023-12-31",
			want:    time.Date(2023, time.December, 31, 0, 0, 0, 0, time.UTC),
			wantErr: false,
		},
		{
			name:    "invalid format",
			input:   "26-09-2025",
			wantErr: true,
		},
		{
			name:    "invalid month",
			input:   "2025-13-01",
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "invalid day",
			input:   "2025-01-00",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := parseDanskeBankChangeDate(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseDanskeBankChangeDate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !got.Equal(tt.want) {
				t.Errorf("parseDanskeBankChangeDate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseNominalRate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    float32
		wantErr bool
	}{
		{
			name:    "Swedish format with comma",
			input:   "3,74",
			want:    3.74,
			wantErr: false,
		},
		{
			name:    "with percent sign",
			input:   "3.74%",
			want:    3.74,
			wantErr: false,
		},
		{
			name:    "with space and percent",
			input:   "3.74 %",
			want:    3.74,
			wantErr: false,
		},
		{
			name:    "Swedish format with space and percent",
			input:   "2,58 %",
			want:    2.58,
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

			got, err := parseNominalRate(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseNominalRate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("parseNominalRate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDanskeBankCrawler_parseReferenceMonth(t *testing.T) {
	t.Parallel()

	logger := zap.NewNop()
	crawler := &DanskeBankCrawler{logger: logger}

	tests := []struct {
		name      string
		input     string
		wantMonth time.Month
		wantYear  uint
		wantErr   bool
	}{
		{
			name:      "November 2025",
			input:     "November 2025",
			wantMonth: time.November,
			wantYear:  2025,
			wantErr:   false,
		},
		{
			name:      "Augusti 2024",
			input:     "Augusti 2024",
			wantMonth: time.August,
			wantYear:  2024,
			wantErr:   false,
		},
		{
			name:      "lowercase januari",
			input:     "januari 2023",
			wantMonth: time.January,
			wantYear:  2023,
			wantErr:   false,
		},
		{
			name:      "with extra spaces",
			input:     "  Mars   2024  ",
			wantMonth: time.March,
			wantYear:  2024,
			wantErr:   false,
		},
		{
			name:      "short month name feb",
			input:     "feb 2024",
			wantMonth: time.February,
			wantYear:  2024,
			wantErr:   false,
		},
		{
			name:    "invalid month",
			input:   "InvalidMonth 2024",
			wantErr: true,
		},
		{
			name:    "missing year",
			input:   "November",
			wantErr: true,
		},
		{
			name:    "year out of range low",
			input:   "November 1900",
			wantErr: true,
		},
		{
			name:    "year out of range high",
			input:   "November 2200",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := crawler.parseReferenceMonth(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseReferenceMonth() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got.Month != tt.wantMonth {
					t.Errorf("parseReferenceMonth() month = %v, want %v", got.Month, tt.wantMonth)
				}
				if got.Year != tt.wantYear {
					t.Errorf("parseReferenceMonth() year = %v, want %v", got.Year, tt.wantYear)
				}
			}
		})
	}
}

func TestDanskeBankCrawler_sanitizeAvgRows(t *testing.T) {
	t.Parallel()

	logger := zap.NewNop()
	crawler := &DanskeBankCrawler{logger: logger}

	tests := []struct {
		name     string
		input    [][]string
		wantRows int
	}{
		{
			name: "normal rows unchanged",
			input: [][]string{
				{"November 2025", "2,58", "2,76", "2,80"},
				{"Oktober 2025", "2,58", "2,80", "2,83"},
			},
			wantRows: 2,
		},
		{
			name: "empty rows removed",
			input: [][]string{
				{"November 2025", "2,58", "2,76", "2,80"},
				{},
				{"Oktober 2025", "2,58", "2,80", "2,83"},
			},
			wantRows: 2,
		},
		{
			name: "split rows merged",
			input: [][]string{
				{"November 2025", "2,58", "2,76", "2,80"},
				{"Oktober 2025"},
				{"2,58", "2,80", "2,83"},
			},
			wantRows: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create a copy to avoid mutation issues
			inputCopy := make([][]string, len(tt.input))
			for i, row := range tt.input {
				inputCopy[i] = make([]string, len(row))
				copy(inputCopy[i], row)
			}

			table := crawler.sanitizeAvgRows(struct {
				Header []string
				Rows   [][]string
			}{
				Header: []string{"Månad", "3 mån", "1 år", "2 år"},
				Rows:   inputCopy,
			})

			if len(table.Rows) != tt.wantRows {
				t.Errorf("sanitizeAvgRows() returned %d rows, want %d", len(table.Rows), tt.wantRows)
			}
		})
	}
}

func TestNewDanskeBankCrawler(t *testing.T) {
	t.Parallel()

	mockClient := &httpmock.ClientMock{}
	logger := zap.NewNop()

	crawler := NewDanskeBankCrawler(mockClient, logger)

	if crawler == nil {
		t.Fatal("NewDanskeBankCrawler() returned nil")
	}
	if crawler.httpClient != mockClient {
		t.Error("httpClient not set correctly")
	}
	if crawler.logger != logger {
		t.Error("logger not set correctly")
	}
}

func TestDanskeBankCrawler_InterfaceCompliance(t *testing.T) {
	t.Parallel()

	// Compile-time check that DanskeBankCrawler implements SiteCrawler
	var _ SiteCrawler = &DanskeBankCrawler{}
}
