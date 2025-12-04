//nolint:revive,nolintlint,dupl // package name matches the package being tested; test patterns intentionally similar across crawlers
package skandia

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
func TestSkandiaCrawler_Crawl(t *testing.T) {
	t.Parallel()

	listRatesHTML := crawlertest.LoadGoldenFile(t, "testdata/skandia_list_rates.html")
	avgRatesHTML := crawlertest.LoadGoldenFile(t, "testdata/skandia_avg_rates.html")
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
			mockFetch: func(url string, _ map[string]string) (string, error) {
				if url == skandiaListRatesURL {
					return listRatesHTML, nil
				}
				return avgRatesHTML, nil
			},
			wantListRates:    5, // 3 mån, 1 år, 2 år, 3 år, 5 år
			wantAvgRates:     true,
			wantTotalMinimum: 10, // 5 list + at least 5 average rates (1 month × 5 terms)
		},
		{
			name: "list rates fetch error still returns average rates",
			mockFetch: func(url string, _ map[string]string) (string, error) {
				if url == skandiaListRatesURL {
					return "", errors.New("network error")
				}
				return avgRatesHTML, nil
			},
			wantListRates:    0,
			wantAvgRates:     true,
			wantTotalMinimum: 5, // at least 5 average rates (1 month × 5 terms)
		},
		{
			name: "average rates fetch error still returns list rates",
			mockFetch: func(url string, _ map[string]string) (string, error) {
				if url == skandiaListRatesURL {
					return listRatesHTML, nil
				}
				return "", errors.New("network error")
			},
			wantListRates:    5,
			wantAvgRates:     false,
			wantTotalMinimum: 5,
		},
		{
			name: "both fetch errors return no results",
			mockFetch: func(_ string, _ map[string]string) (string, error) {
				return "", errors.New("network error")
			},
			wantListRates:    0,
			wantAvgRates:     false,
			wantTotalMinimum: 0,
		},
		{
			name: "invalid HTML returns no list rates",
			mockFetch: func(url string, _ map[string]string) (string, error) {
				if url == skandiaListRatesURL {
					return "<html><body>No SKB.pageContent here</body></html>", nil
				}
				return avgRatesHTML, nil
			},
			wantListRates:    0,
			wantAvgRates:     true,
			wantTotalMinimum: 5, // at least 5 average rates (1 month × 5 terms)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockClient := &httpmock.ClientMock{
				FetchFunc: tt.mockFetch,
			}

			crawler := NewSkandiaCrawler(mockClient, logger)
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

			crawlertest.AssertBankName(t, results, skandiaBankName)
		})
	}
}

func TestSkandiaCrawler_fetchListRates(t *testing.T) {
	t.Parallel()

	listRatesHTML := crawlertest.LoadGoldenFile(t, "testdata/skandia_list_rates.html")
	logger := zap.NewNop()
	crawlTime := time.Date(2025, 12, 1, 10, 0, 0, 0, time.UTC)

	mockClient := &httpmock.ClientMock{
		FetchFunc: func(_ string, _ map[string]string) (string, error) {
			return listRatesHTML, nil
		},
	}

	crawler := &SkandiaCrawler{httpClient: mockClient, logger: logger}

	results, err := crawler.fetchListRates(crawlTime)
	if err != nil {
		t.Fatalf("fetchListRates() error = %v", err)
	}

	if len(results) != 5 {
		t.Errorf("fetchListRates() returned %d results, want 5", len(results))
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
		assertSkandiaListRateFields(t, r, crawlTime)
	}

	for term, found := range expectedTerms {
		if !found {
			t.Errorf("missing term %q in results", term)
		}
	}
}

func TestSkandiaCrawler_fetchAverageRates(t *testing.T) {
	t.Parallel()

	avgRatesHTML := crawlertest.LoadGoldenFile(t, "testdata/skandia_avg_rates.html")
	logger := zap.NewNop()
	crawlTime := time.Date(2025, 12, 1, 10, 0, 0, 0, time.UTC)

	mockClient := &httpmock.ClientMock{
		FetchFunc: func(_ string, _ map[string]string) (string, error) {
			return avgRatesHTML, nil
		},
	}

	crawler := &SkandiaCrawler{httpClient: mockClient, logger: logger}

	results, err := crawler.fetchAverageRates(crawlTime)
	if err != nil {
		t.Fatalf("fetchAverageRates() error = %v", err)
	}

	if len(results) == 0 {
		t.Fatal("fetchAverageRates() returned no results")
	}

	// Verify we got multiple months of data
	months := make(map[string]bool)
	for _, r := range results {
		if r.AverageReferenceMonth != nil {
			key := r.AverageReferenceMonth.Month.String() + string(rune(r.AverageReferenceMonth.Year))
			months[key] = true
		}
		assertSkandiaAverageRateFields(t, r, crawlTime)
	}

	// Should have data for at least one month (current month)
	if len(months) < 1 {
		t.Errorf("got data for %d months, want at least 1", len(months))
	}
}

func TestParseSkandiaHTMLTerm(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		htmlCell string
		want     model.Term
		wantErr  bool
	}{
		{
			name:     "3 months",
			htmlCell: "<p>3 mån</p>",
			want:     model.Term3months,
			wantErr:  false,
		},
		{
			name:     "1 year",
			htmlCell: "<p>1 år</p>",
			want:     model.Term1year,
			wantErr:  false,
		},
		{
			name:     "2 years",
			htmlCell: "<p>2 år</p>",
			want:     model.Term2years,
			wantErr:  false,
		},
		{
			name:     "3 years",
			htmlCell: "<p>3 år</p>",
			want:     model.Term3years,
			wantErr:  false,
		},
		{
			name:     "5 years",
			htmlCell: "<p>5 år</p>",
			want:     model.Term5years,
			wantErr:  false,
		},
		{
			name:     "unsupported month term",
			htmlCell: "<p>6 mån</p>",
			wantErr:  true,
		},
		{
			name:     "unsupported year term",
			htmlCell: "<p>15 år</p>",
			wantErr:  true,
		},
		{
			name:     "no term found",
			htmlCell: "<p>something else</p>",
			wantErr:  true,
		},
		{
			name:     "empty string",
			htmlCell: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := parseSkandiaHTMLTerm(tt.htmlCell)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseSkandiaHTMLTerm() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("parseSkandiaHTMLTerm() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseSkandiaHTMLRate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		htmlCell string
		want     float32
		wantErr  bool
	}{
		{
			name:     "valid rate with comma",
			htmlCell: "<p>3,45 %</p>",
			want:     3.45,
			wantErr:  false,
		},
		{
			name:     "valid rate without whitespace",
			htmlCell: "<p>2,95%</p>",
			want:     2.95,
			wantErr:  false,
		},
		{
			name:     "valid rate with multiple spaces",
			htmlCell: "<p>4,20  %</p>",
			want:     4.20,
			wantErr:  false,
		},
		{
			name:     "integer rate",
			htmlCell: "<p>3,00 %</p>",
			want:     3.00,
			wantErr:  false,
		},
		{
			name:     "no percent sign",
			htmlCell: "<p>3,45</p>",
			wantErr:  true,
		},
		{
			name:     "no rate found",
			htmlCell: "<p>something else</p>",
			wantErr:  true,
		},
		{
			name:     "empty string",
			htmlCell: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := parseSkandiaHTMLRate(tt.htmlCell)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseSkandiaHTMLRate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("parseSkandiaHTMLRate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseSkandiaMonthYear(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		text      string
		wantYear  uint
		wantMonth time.Month
		wantErr   bool
	}{
		{
			name:      "valid november 2025",
			text:      "Snitträntor november 2025",
			wantYear:  2025,
			wantMonth: time.November,
			wantErr:   false,
		},
		{
			name:      "valid januari 2025",
			text:      "Snitträntor januari 2025",
			wantYear:  2025,
			wantMonth: time.January,
			wantErr:   false,
		},
		{
			name:      "valid december 2024",
			text:      "Snitträntor december 2024",
			wantYear:  2024,
			wantMonth: time.December,
			wantErr:   false,
		},
		{
			name:      "case insensitive",
			text:      "SNITTRÄNTOR NOVEMBER 2025",
			wantYear:  2025,
			wantMonth: time.November,
			wantErr:   false,
		},
		{
			name:    "no month found",
			text:    "Snitträntor 2025",
			wantErr: true,
		},
		{
			name:    "invalid month name",
			text:    "Snitträntor invalidmonth 2025",
			wantErr: true,
		},
		{
			name:    "no year found",
			text:    "Snitträntor november",
			wantErr: true,
		},
		{
			name:    "empty string",
			text:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := parseSkandiaMonthYear(tt.text)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseSkandiaMonthYear() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got.Year != tt.wantYear {
					t.Errorf("parseSkandiaMonthYear() year = %v, want %v", got.Year, tt.wantYear)
				}
				if got.Month != tt.wantMonth {
					t.Errorf("parseSkandiaMonthYear() month = %v, want %v", got.Month, tt.wantMonth)
				}
			}
		})
	}
}

func TestNewSkandiaCrawler(t *testing.T) {
	t.Parallel()

	mockClient := &httpmock.ClientMock{}
	logger := zap.NewNop()

	crawler := NewSkandiaCrawler(mockClient, logger)

	if crawler == nil {
		t.Fatal("NewSkandiaCrawler() returned nil")
	}
	if crawler.httpClient != mockClient {
		t.Error("httpClient not set correctly")
	}
	if crawler.logger != logger {
		t.Error("logger not set correctly")
	}
}

func TestSkandiaCrawler_InterfaceCompliance(t *testing.T) {
	t.Parallel()

	// Compile-time check that SkandiaCrawler implements SiteCrawler
	var _ crawlertest.SiteCrawler = &SkandiaCrawler{}
}

// assertSkandiaListRateFields validates common fields for Skandia list rate results.
func assertSkandiaListRateFields(t *testing.T, r model.InterestSet, crawlTime time.Time) {
	t.Helper()

	if r.Bank != skandiaBankName {
		t.Errorf("Bank = %q, want %q", r.Bank, skandiaBankName)
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

// assertSkandiaAverageRateFields validates common fields for Skandia average rate results.
func assertSkandiaAverageRateFields(t *testing.T, r model.InterestSet, crawlTime time.Time) {
	t.Helper()

	if r.Bank != skandiaBankName {
		t.Errorf("Bank = %q, want %q", r.Bank, skandiaBankName)
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
