//nolint:revive,nolintlint,dupl // package name matches the package being tested; test patterns intentionally similar across crawlers
package nordea

import (
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	crawlertest "github.com/yama6a/bolan-compare/internal/app/crawler"
	"github.com/yama6a/bolan-compare/internal/pkg/http/httpmock"
	"github.com/yama6a/bolan-compare/internal/pkg/model"
	"go.uber.org/zap"
)

// assertNordeaListRateFields validates common fields for Nordea list rate results.
func assertNordeaListRateFields(t *testing.T, r model.InterestSet, crawlTime time.Time) {
	t.Helper()

	if r.Bank != nordeaBankName {
		t.Errorf("Bank = %q, want %q", r.Bank, nordeaBankName)
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

// assertNordeaAvgRateFields validates common fields for Nordea average rate results.
func assertNordeaAvgRateFields(t *testing.T, r model.InterestSet, crawlTime time.Time) {
	t.Helper()

	if r.Bank != nordeaBankName {
		t.Errorf("Bank = %q, want %q", r.Bank, nordeaBankName)
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

// loadGoldenFileBytes loads a binary golden file for testing.
func loadGoldenFileBytes(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read golden file %s: %v", path, err)
	}
	return data
}

func TestNordeaCrawler_Crawl_Success(t *testing.T) {
	t.Parallel()

	listRatesHTML := crawlertest.LoadGoldenFile(t, "testdata/nordea_list_rates.html")
	historicRatesPageHTML := crawlertest.LoadGoldenFile(t, "testdata/nordea_historic_rates_page.html")
	historicRatesXLSX := loadGoldenFileBytes(t, "testdata/nordea_historic_rates.xlsx")

	mockClient := &httpmock.ClientMock{
		FetchFunc: func(url string, _ map[string]string) (string, error) {
			if url == nordeaListRatesURL {
				return listRatesHTML, nil
			}
			if url == nordeaHistoricRatesURL {
				return historicRatesPageHTML, nil
			}
			return "", errors.New("unexpected URL: " + url)
		},
		FetchRawFunc: func(url string, _ map[string]string) ([]byte, error) {
			if strings.HasSuffix(url, ".xlsx") {
				return historicRatesXLSX, nil
			}
			return nil, errors.New("unexpected URL: " + url)
		},
	}

	crawler := NewNordeaCrawler(mockClient, zap.NewNop())
	results := runNordeaCrawl(t, crawler)

	// Should have many results (7 list rates + thousands of historic rates)
	if len(results) < 100 {
		t.Errorf("total results = %d, want at least 100", len(results))
	}

	crawlertest.AssertBankName(t, results, nordeaBankName)
}

func TestNordeaCrawler_Crawl_FetchError(t *testing.T) {
	t.Parallel()

	mockClient := &httpmock.ClientMock{
		FetchFunc: func(_ string, _ map[string]string) (string, error) {
			return "", errors.New("network error")
		},
		FetchRawFunc: func(_ string, _ map[string]string) ([]byte, error) {
			return nil, errors.New("network error")
		},
	}

	crawler := NewNordeaCrawler(mockClient, zap.NewNop())
	results := runNordeaCrawl(t, crawler)

	if len(results) != 0 {
		t.Errorf("expected no results on error, got %d", len(results))
	}
}

func TestNordeaCrawler_Crawl_ListRatesOnly(t *testing.T) {
	t.Parallel()

	listRatesHTML := crawlertest.LoadGoldenFile(t, "testdata/nordea_list_rates.html")

	mockClient := &httpmock.ClientMock{
		FetchFunc: func(url string, _ map[string]string) (string, error) {
			if url == nordeaListRatesURL {
				return listRatesHTML, nil
			}
			return "", errors.New("network error")
		},
		FetchRawFunc: func(_ string, _ map[string]string) ([]byte, error) {
			return nil, errors.New("network error")
		},
	}

	crawler := NewNordeaCrawler(mockClient, zap.NewNop())
	results := runNordeaCrawl(t, crawler)

	// Should have list rates only (7 terms)
	if len(results) != 7 {
		t.Errorf("list rate count = %d, want 7", len(results))
	}

	crawlertest.AssertBankName(t, results, nordeaBankName)
}

func runNordeaCrawl(t *testing.T, crawler *NordeaCrawler) []model.InterestSet {
	t.Helper()
	resultChan := make(chan model.InterestSet, 10000)
	crawler.Crawl(resultChan)
	close(resultChan)

	results := make([]model.InterestSet, 0, len(resultChan))
	for set := range resultChan {
		results = append(results, set)
	}
	return results
}

func TestNordeaCrawler_extractListRates(t *testing.T) {
	t.Parallel()

	goldenHTML := crawlertest.LoadGoldenFile(t, "testdata/nordea_list_rates.html")
	logger := zap.NewNop()
	crawler := &NordeaCrawler{logger: logger}
	crawlTime := time.Date(2025, 12, 1, 10, 0, 0, 0, time.UTC)

	results, err := crawler.extractListRates(goldenHTML, crawlTime)
	if err != nil {
		t.Fatalf("extractListRates() error = %v", err)
	}

	if len(results) != 7 {
		t.Errorf("extractListRates() returned %d results, want 7", len(results))
	}

	// Verify expected terms are present (Nordea has 8 år instead of 7 år/10 år)
	expectedTerms := map[model.Term]bool{
		model.Term3months: false,
		model.Term1year:   false,
		model.Term2years:  false,
		model.Term3years:  false,
		model.Term4years:  false,
		model.Term5years:  false,
		model.Term8years:  false,
	}

	for _, r := range results {
		if _, ok := expectedTerms[r.Term]; ok {
			expectedTerms[r.Term] = true
		}
		assertNordeaListRateFields(t, r, crawlTime)
	}

	for term, found := range expectedTerms {
		if !found {
			t.Errorf("missing term %q in results", term)
		}
	}
}

func TestNordeaCrawler_parseHistoricRatesXLSX(t *testing.T) {
	t.Parallel()

	xlsxData := loadGoldenFileBytes(t, "testdata/nordea_historic_rates.xlsx")
	logger := zap.NewNop()
	crawler := &NordeaCrawler{logger: logger}
	crawlTime := time.Date(2025, 12, 1, 10, 0, 0, 0, time.UTC)

	results, err := crawler.parseHistoricRatesXLSX(xlsxData, crawlTime)
	if err != nil {
		t.Fatalf("parseHistoricRatesXLSX() error = %v", err)
	}

	if len(results) == 0 {
		t.Fatal("parseHistoricRatesXLSX() returned no results")
	}

	// Verify we got multiple years of data (data spans 1990-2025)
	years := make(map[int]bool)
	for _, r := range results {
		if r.AverageReferenceMonth != nil {
			years[int(r.AverageReferenceMonth.Year)] = true
		}
		assertNordeaAvgRateFields(t, r, crawlTime)
	}

	// Should have data spanning many years
	if len(years) < 10 {
		t.Errorf("got data for %d years, want at least 10 (historic data spans 1990-2025)", len(years))
	}

	// Verify terms are present (dynamically parsed from XLSX header)
	termsSeen := make(map[model.Term]bool)
	for _, r := range results {
		termsSeen[r.Term] = true
	}

	// Should have multiple terms parsed from the XLSX header
	if len(termsSeen) < 5 {
		t.Errorf("got %d unique terms, want at least 5", len(termsSeen))
	}
}

func TestParseNordeaRate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    float32
		wantErr bool
	}{
		{
			name:    "Swedish format with comma and percent",
			input:   "3,33 %",
			want:    3.33,
			wantErr: false,
		},
		{
			name:    "Swedish format with comma",
			input:   "3,74",
			want:    3.74,
			wantErr: false,
		},
		{
			name:    "with percent sign no space",
			input:   "3.74%",
			want:    3.74,
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

			got, err := parseNordeaRate(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseNordeaRate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("parseNordeaRate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseNordeaDate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    time.Time
		wantErr bool
	}{
		{
			name:    "valid date YYYY-MM-DD",
			input:   "2025-10-06",
			want:    time.Date(2025, time.October, 6, 0, 0, 0, 0, time.UTC),
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

			got, err := parseNordeaDate(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseNordeaDate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !got.Equal(tt.want) {
				t.Errorf("parseNordeaDate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseNordeaHistoricDate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    time.Time
		wantErr bool
	}{
		{
			name:    "valid date MM-DD-YY (2000s)",
			input:   "06-15-24",
			want:    time.Date(2024, time.June, 15, 0, 0, 0, 0, time.UTC),
			wantErr: false,
		},
		{
			name:    "valid date MM-DD-YY (1990s)",
			input:   "03-20-95",
			want:    time.Date(1995, time.March, 20, 0, 0, 0, 0, time.UTC),
			wantErr: false,
		},
		{
			name:    "valid date boundary year 90",
			input:   "01-01-90",
			want:    time.Date(1990, time.January, 1, 0, 0, 0, 0, time.UTC),
			wantErr: false,
		},
		{
			name:    "valid date boundary year 89 (2089)",
			input:   "12-31-89",
			want:    time.Date(2089, time.December, 31, 0, 0, 0, 0, time.UTC),
			wantErr: false,
		},
		{
			name:    "invalid format YYYY-MM-DD",
			input:   "2024-06-15",
			wantErr: true,
		},
		{
			name:    "text instead of date",
			input:   "test-data",
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

			got, err := parseNordeaHistoricDate(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseNordeaHistoricDate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !got.Equal(tt.want) {
				t.Errorf("parseNordeaHistoricDate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewNordeaCrawler(t *testing.T) {
	t.Parallel()

	mockClient := &httpmock.ClientMock{}
	logger := zap.NewNop()

	crawler := NewNordeaCrawler(mockClient, logger)

	if crawler == nil {
		t.Fatal("NewNordeaCrawler() returned nil")
	}
	if crawler.httpClient != mockClient {
		t.Error("httpClient not set correctly")
	}
	if crawler.logger != logger {
		t.Error("logger not set correctly")
	}
}

func TestNordeaCrawler_InterfaceCompliance(t *testing.T) {
	t.Parallel()

	// Compile-time check that NordeaCrawler implements SiteCrawler
	var _ crawlertest.SiteCrawler = &NordeaCrawler{}
}
