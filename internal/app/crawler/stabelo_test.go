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

func TestStabeloCrawler_Crawl(t *testing.T) {
	t.Parallel()

	rateTableHTML := loadGoldenFile(t, "testdata/stabelo_rate_table.html")
	logger := zap.NewNop()

	tests := []struct {
		name            string
		mockFetch       func(url string, headers map[string]string) (string, error)
		mockFetchRaw    func(url string, headers map[string]string) ([]byte, error)
		wantLTVRates    int  // LTV discounted rates (from HTML buttons)
		wantMinimum     int  // minimum total results expected
		expectSomeRates bool // whether we expect any rates at all
	}{
		{
			name: "successful crawl extracts LTV rates from HTML buttons",
			mockFetch: func(url string, _ map[string]string) (string, error) {
				// Rate table returns HTML, average rates page returns error (no PDF testing)
				if url == stabeloRateTableURL {
					return rateTableHTML, nil
				}
				return "", errors.New("not found")
			},
			mockFetchRaw: func(_ string, _ map[string]string) ([]byte, error) {
				return nil, errors.New("PDF not available in test")
			},
			wantLTVRates:    6, // 3 mån, 1 år, 2 år, 3 år, 5 år, 10 år (LTV discounted)
			wantMinimum:     6,
			expectSomeRates: true,
		},
		{
			name: "fetch error returns no results",
			mockFetch: func(_ string, _ map[string]string) (string, error) {
				return "", errors.New("network error")
			},
			mockFetchRaw: func(_ string, _ map[string]string) ([]byte, error) {
				return nil, errors.New("network error")
			},
			wantLTVRates:    0,
			wantMinimum:     0,
			expectSomeRates: false,
		},
		{
			name: "empty HTML returns no results",
			mockFetch: func(_ string, _ map[string]string) (string, error) {
				return "", nil
			},
			mockFetchRaw: func(_ string, _ map[string]string) ([]byte, error) {
				return nil, errors.New("not available")
			},
			wantLTVRates:    0,
			wantMinimum:     0,
			expectSomeRates: false,
		},
		{
			name: "invalid HTML returns no results",
			mockFetch: func(_ string, _ map[string]string) (string, error) {
				return "<html><body>No rate table here</body></html>", nil
			},
			mockFetchRaw: func(_ string, _ map[string]string) ([]byte, error) {
				return nil, errors.New("not available")
			},
			wantLTVRates:    0,
			wantMinimum:     0,
			expectSomeRates: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockClient := &httpmock.ClientMock{
				FetchFunc:    tt.mockFetch,
				FetchRawFunc: tt.mockFetchRaw,
			}

			crawler := NewStabeloCrawler(mockClient, logger)
			resultChan := make(chan model.InterestSet, 50)

			crawler.Crawl(resultChan)
			close(resultChan)

			var results []model.InterestSet
			for set := range resultChan {
				results = append(results, set)
			}

			// Count LTV discounted rates
			ltvRateCount := 0
			for _, r := range results {
				if r.Type == model.TypeRatioDiscounted {
					ltvRateCount++
				}
			}

			if ltvRateCount != tt.wantLTVRates {
				t.Errorf("LTV rate count = %d, want %d", ltvRateCount, tt.wantLTVRates)
			}

			if len(results) < tt.wantMinimum {
				t.Errorf("total results = %d, want at least %d", len(results), tt.wantMinimum)
			}

			if tt.expectSomeRates && len(results) > 0 {
				assertBankName(t, results, stabeloBankName)
			}
		})
	}
}

func TestStabeloCrawler_fetchRates(t *testing.T) {
	t.Parallel()

	rateTableHTML := loadGoldenFile(t, "testdata/stabelo_rate_table.html")
	logger := zap.NewNop()
	crawlTime := time.Date(2025, 12, 1, 10, 0, 0, 0, time.UTC)

	mockClient := &httpmock.ClientMock{
		FetchFunc: func(_ string, _ map[string]string) (string, error) {
			return rateTableHTML, nil
		},
	}

	crawler := &StabeloCrawler{httpClient: mockClient, logger: logger}

	results, err := crawler.fetchRates(crawlTime)
	if err != nil {
		t.Fatalf("fetchRates() error = %v", err)
	}

	// Should have at least LTV rates from HTML buttons
	if len(results) < 6 {
		t.Errorf("fetchRates() returned %d results, want at least 6", len(results))
	}

	// Verify expected terms are present for LTV rates
	expectedTerms := map[model.Term]bool{
		model.Term3months: false,
		model.Term1year:   false,
		model.Term2years:  false,
		model.Term3years:  false,
		model.Term5years:  false,
		model.Term10years: false,
	}

	for _, r := range results {
		if r.Type == model.TypeRatioDiscounted {
			if _, ok := expectedTerms[r.Term]; ok {
				expectedTerms[r.Term] = true
			}
			assertStabeloLTVRateFields(t, r, crawlTime)
		}
	}

	for term, found := range expectedTerms {
		if !found {
			t.Errorf("missing term %q in LTV rate results", term)
		}
	}
}

func TestStabeloCrawler_extractLTVRatesFromHTML(t *testing.T) {
	t.Parallel()

	logger := zap.NewNop()
	crawlTime := time.Date(2025, 12, 1, 10, 0, 0, 0, time.UTC)

	tests := []struct {
		name      string
		html      string
		wantCount int
		wantErr   bool
	}{
		{
			name: "valid rate buttons",
			html: `<html><body>
				<button type="button" value="3M"><span>3 mån</span><span>2,54 %</span></button>
				<button type="button" value="1Y"><span>1 år</span><span>2,73 %</span></button>
			</body></html>`,
			wantCount: 2,
			wantErr:   false,
		},
		{name: "empty HTML", html: "", wantCount: 0, wantErr: true},
		{name: "no rate buttons", html: "<html><body>No rates here</body></html>", wantCount: 0, wantErr: true},
		{name: "button without rate span logs warning and continues", html: `<html><body><button type="button" value="3M"><span>3 mån</span></button></body></html>`, wantCount: 0, wantErr: false},
		{
			name: "button with invalid value ignored",
			html: `<html><body>
				<button type="button" value="invalid"><span>test</span><span>2,54 %</span></button>
				<button type="button" value="3M"><span>3 mån</span><span>2,54 %</span></button>
			</body></html>`,
			wantCount: 1,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			crawler := &StabeloCrawler{httpClient: nil, logger: logger}
			results, err := crawler.extractLTVRatesFromHTML(tt.html, crawlTime)

			if (err != nil) != tt.wantErr {
				t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(results) != tt.wantCount {
				t.Errorf("returned %d results, want %d", len(results), tt.wantCount)
			}

			for _, r := range results {
				assertLTVRateType(t, r)
			}
		})
	}
}

func assertLTVRateType(t *testing.T, r model.InterestSet) {
	t.Helper()
	if r.Type != model.TypeRatioDiscounted {
		t.Errorf("Type = %q, want %q", r.Type, model.TypeRatioDiscounted)
	}
	if r.RatioDiscountBoundaries == nil {
		t.Error("RatioDiscountBoundaries should not be nil")
	} else if r.RatioDiscountBoundaries.MaxRatio != 0.60 {
		t.Errorf("MaxRatio = %v, want 0.60", r.RatioDiscountBoundaries.MaxRatio)
	}
}

func TestStabeloCrawler_parseAverageRatesText(t *testing.T) {
	t.Parallel()

	logger := zap.NewNop()
	crawlTime := time.Date(2025, 12, 1, 10, 0, 0, 0, time.UTC)

	tests := []struct {
		name           string
		text           string
		wantCount      int
		wantErr        bool
		wantFirstMonth time.Month
		wantFirstYear  int
	}{
		{
			name:           "valid PDF text with multiple months",
			text:           `Stabelos genomsnittsräntorHär ser du våra genomsnittsräntor.Bindningstid3 mån1 år2 år3 år5 år10 årnovember 2025 2,50% 2,60% 2,70% 2,80% 3,00% 3,50%oktober 2025 2,61% 2,52% 2,89%`,
			wantCount:      9, // 6 rates from Nov + 3 rates from Oct
			wantErr:        false,
			wantFirstMonth: time.November,
			wantFirstYear:  2025,
		},
		{
			name:           "single month with all rates",
			text:           `Bindningstid3 mån1 år2 år3 år5 år10 årjanuari 2025 1,50% 1,60% 1,70% 1,80% 2,00% 2,50%`,
			wantCount:      6,
			wantErr:        false,
			wantFirstMonth: time.January,
			wantFirstYear:  2025,
		},
		{
			name:           "rates with missing values (dashes)",
			text:           `Bindningstid3 mån1 år2 år3 år5 år10 åroktober 2025 2,61% 2,52% 2,89% 2,96% 3,20%-september 2025 2,90%`,
			wantCount:      6, // 5 rates from Oct + 1 rate from Sep
			wantErr:        false,
			wantFirstMonth: time.October,
			wantFirstYear:  2025,
		},
		{
			name:      "no rate data",
			text:      "Some random text without rate data",
			wantCount: 0,
			wantErr:   true,
		},
		{
			name:      "month without percentage rates",
			text:      "november 2025 no actual rates here",
			wantCount: 0,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			crawler := &StabeloCrawler{httpClient: nil, logger: logger}

			results, err := crawler.parseAverageRatesText(tt.text, crawlTime)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseAverageRatesText() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(results) != tt.wantCount {
				t.Errorf("parseAverageRatesText() returned %d results, want %d", len(results), tt.wantCount)
			}

			verifyAvgRateResults(t, results, tt.wantFirstMonth, tt.wantFirstYear)
		})
	}
}

func verifyAvgRateResults(t *testing.T, results []model.InterestSet, wantFirstMonth time.Month, wantFirstYear int) {
	t.Helper()

	// Check that all results are average rate type
	for _, r := range results {
		if r.Type != model.TypeAverageRate {
			t.Errorf("Type = %q, want %q", r.Type, model.TypeAverageRate)
		}
		if r.AverageReferenceMonth == nil {
			t.Error("AverageReferenceMonth should not be nil")
		}
	}

	// Check first result has expected month/year (first month in PDF)
	if len(results) > 0 && wantFirstMonth != 0 {
		first := results[0]
		if first.AverageReferenceMonth.Month != wantFirstMonth {
			t.Errorf("First result month = %v, want %v", first.AverageReferenceMonth.Month, wantFirstMonth)
		}
		if first.AverageReferenceMonth.Year != uint(wantFirstYear) {
			t.Errorf("First result year = %v, want %v", first.AverageReferenceMonth.Year, wantFirstYear)
		}
	}
}

func TestStabeloCrawler_findPDFLink(t *testing.T) {
	t.Parallel()

	logger := zap.NewNop()
	crawler := &StabeloCrawler{httpClient: nil, logger: logger}

	tests := []struct {
		name    string
		html    string
		wantURL string
	}{
		{
			name:    "finds PDF link with full URL",
			html:    `<a href="https://www.stabelo.se/documents/StabeloGenomsnittsrantorNovember2025.pdf">Download</a>`,
			wantURL: "https://www.stabelo.se/documents/StabeloGenomsnittsrantorNovember2025.pdf",
		},
		{
			name:    "finds PDF link with relative path",
			html:    `<a href="/documents/StabeloGenomsnittsrantorOktober2025.pdf">Download</a>`,
			wantURL: "https://www.stabelo.se/documents/StabeloGenomsnittsrantorOktober2025.pdf",
		},
		{
			name:    "no PDF link found",
			html:    `<html><body>No PDF here</body></html>`,
			wantURL: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := crawler.findPDFLink(tt.html)
			if got != tt.wantURL {
				t.Errorf("findPDFLink() = %q, want %q", got, tt.wantURL)
			}
		})
	}
}

func TestParseStabeloTerm(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		term    string
		want    model.Term
		wantErr bool
	}{
		{name: "3 months", term: "3M", want: model.Term3months, wantErr: false},
		{name: "1 year", term: "1Y", want: model.Term1year, wantErr: false},
		{name: "2 years", term: "2Y", want: model.Term2years, wantErr: false},
		{name: "3 years", term: "3Y", want: model.Term3years, wantErr: false},
		{name: "5 years", term: "5Y", want: model.Term5years, wantErr: false},
		{name: "10 years", term: "10Y", want: model.Term10years, wantErr: false},
		{name: "unsupported 4 years", term: "4Y", wantErr: true},
		{name: "unsupported 6 months", term: "6M", wantErr: true},
		{name: "empty string", term: "", wantErr: true},
		{name: "invalid format", term: "3 months", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := parseStabeloTerm(tt.term)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseStabeloTerm() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("parseStabeloTerm() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseStabeloRate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    float32
		wantErr bool
	}{
		{name: "rate 2,54%", input: "2,54 %", want: 2.54, wantErr: false},
		{name: "rate 10,00%", input: "10,00 %", want: 10.00, wantErr: false},
		{name: "rate without space before %", input: "3,20%", want: 3.20, wantErr: false},
		{name: "rate with extra spaces", input: "  4,00 %  ", want: 4.00, wantErr: false},
		{name: "rate without percent sign", input: "2,50", want: 2.50, wantErr: false},
		{name: "invalid - no number", input: "abc %", wantErr: true},
		{name: "empty string", input: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := parseStabeloRate(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseStabeloRate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("parseStabeloRate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewStabeloCrawler(t *testing.T) {
	t.Parallel()

	mockClient := &httpmock.ClientMock{}
	logger := zap.NewNop()

	crawler := NewStabeloCrawler(mockClient, logger)

	if crawler == nil {
		t.Fatal("NewStabeloCrawler() returned nil")
	}
	if crawler.httpClient != mockClient {
		t.Error("httpClient not set correctly")
	}
	if crawler.logger != logger {
		t.Error("logger not set correctly")
	}
}

func TestStabeloCrawler_InterfaceCompliance(t *testing.T) {
	t.Parallel()

	// Compile-time check that StabeloCrawler implements SiteCrawler
	var _ SiteCrawler = &StabeloCrawler{}
}

// assertStabeloLTVRateFields validates common fields for Stabelo LTV discounted rate results.
func assertStabeloLTVRateFields(t *testing.T, r model.InterestSet, crawlTime time.Time) {
	t.Helper()

	if r.Bank != stabeloBankName {
		t.Errorf("Bank = %q, want %q", r.Bank, stabeloBankName)
	}
	if r.Type != model.TypeRatioDiscounted {
		t.Errorf("Type = %q, want %q", r.Type, model.TypeRatioDiscounted)
	}
	if r.NominalRate <= 0 {
		t.Errorf("NominalRate = %f, want positive value", r.NominalRate)
	}
	if r.LastCrawledAt != crawlTime {
		t.Errorf("LastCrawledAt = %v, want %v", r.LastCrawledAt, crawlTime)
	}
	if r.RatioDiscountBoundaries == nil {
		t.Error("RatioDiscountBoundaries should not be nil for LTV rates")
	}
	// Stabelo doesn't provide change dates
	if r.ChangedOn != nil {
		t.Error("ChangedOn should be nil for Stabelo")
	}
}
