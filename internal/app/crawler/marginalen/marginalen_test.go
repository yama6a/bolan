package marginalen

import (
	"errors"
	"testing"
	"time"

	crawlertest "github.com/yama6a/bolan-compare/internal/app/crawler"
	"github.com/yama6a/bolan-compare/internal/pkg/http/httpmock"
	"github.com/yama6a/bolan-compare/internal/pkg/model"
	"go.uber.org/zap"
)

func TestNewMarginalenCrawler(t *testing.T) {
	t.Parallel()

	mockClient := &httpmock.ClientMock{}
	logger := zap.NewNop()

	crawler := NewMarginalenCrawler(mockClient, logger)

	if crawler == nil {
		t.Fatal("expected crawler to be non-nil")
	}

	if crawler.httpClient != mockClient {
		t.Error("expected httpClient to be set")
	}

	if crawler.logger != logger {
		t.Error("expected logger to be set")
	}
}

func TestMarginalenCrawler_InterfaceCompliance(t *testing.T) {
	t.Parallel()

	var _ crawlertest.SiteCrawler = &MarginalenCrawler{}
}

func TestMarginalenCrawler_Crawl(t *testing.T) {
	t.Parallel()

	apiJSON := crawlertest.LoadGoldenFile(t, "testdata/marginalen_api_response.json")
	logger := zap.NewNop()

	tests := []struct {
		name             string
		mockFetch        func(url string, headers map[string]string) (string, error)
		wantAvgRates     bool
		wantTotalMinimum int
	}{
		{
			name: "successful crawl extracts average rates",
			mockFetch: func(_ string, _ map[string]string) (string, error) {
				return apiJSON, nil
			},
			wantAvgRates:     true,
			wantTotalMinimum: 10, // At least 10 average rates from various months/terms
		},
		{
			name: "fetch error returns no results",
			mockFetch: func(_ string, _ map[string]string) (string, error) {
				return "", errors.New("network error")
			},
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

			crawler := NewMarginalenCrawler(mockClient, logger)
			channel := make(chan model.InterestSet, 100)

			go func() {
				crawler.Crawl(channel)
				close(channel)
			}()

			results := make([]model.InterestSet, 0, 100)
			for set := range channel {
				results = append(results, set)
			}

			if len(results) < tt.wantTotalMinimum {
				t.Errorf("expected at least %d results, got %d", tt.wantTotalMinimum, len(results))
			}

			hasAvg := false
			for _, r := range results {
				if r.Type == model.TypeAverageRate {
					hasAvg = true
					break
				}
			}

			if hasAvg != tt.wantAvgRates {
				t.Errorf("wantAvgRates=%v, but hasAvg=%v", tt.wantAvgRates, hasAvg)
			}

			// Verify all results are from Marginalen Bank
			crawlertest.AssertBankName(t, results, marginalenBankName)
		})
	}
}

//nolint:gocognit // Test comprehensiveness requires validation of many edge cases
func TestMarginalenCrawler_ExtractAverageRates(t *testing.T) {
	t.Parallel()

	avgHTML := crawlertest.LoadGoldenFile(t, "testdata/marginalen_avg_rates.html")
	logger := zap.NewNop()
	crawler := NewMarginalenCrawler(nil, logger)

	tests := []struct {
		name    string
		html    string
		wantErr bool
		wantMin int
	}{
		{
			name:    "valid average rates HTML",
			html:    avgHTML,
			wantErr: false,
			wantMin: 10,
		},
		{
			name:    "empty HTML",
			html:    "",
			wantErr: true,
			wantMin: 0,
		},
		{
			name:    "HTML without table",
			html:    "<html><body><p>No table here</p></body></html>",
			wantErr: true,
			wantMin: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fixedTestTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
			results, err := crawler.extractAverageRates(tt.html, fixedTestTime)

			if (err != nil) != tt.wantErr {
				t.Errorf("wantErr=%v, got err=%v", tt.wantErr, err)
			}

			if len(results) < tt.wantMin {
				t.Errorf("expected at least %d results, got %d", tt.wantMin, len(results))
			}

			// Verify results structure
			crawlertest.AssertBankName(t, results, marginalenBankName)

			for i, r := range results {
				if r.Type != model.TypeAverageRate {
					t.Errorf("result[%d]: expected type=%s, got=%s", i, model.TypeAverageRate, r.Type)
				}

				if r.NominalRate <= 0 {
					t.Errorf("result[%d]: expected positive rate, got=%f", i, r.NominalRate)
				}

				// Marginalen publishes rates for: 3 Mån, 6 Mån, 1 år, 2 år, 3 år
				validTerms := []model.Term{
					model.Term3months,
					model.Term6months,
					model.Term1year,
					model.Term2years,
					model.Term3years,
				}
				found := false
				for _, vt := range validTerms {
					if r.Term == vt {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("result[%d]: unexpected term=%s", i, r.Term)
				}
			}
		})
	}
}

func TestMarginalenCrawler_ParseMarginalenRate(t *testing.T) {
	t.Parallel()

	crawler := &MarginalenCrawler{logger: zap.NewNop()}

	tests := []struct {
		name     string
		input    string
		wantRate float64
		wantErr  bool
	}{
		{name: "rate with comma and percent", input: "5,92 %", wantRate: 5.92, wantErr: false},
		{name: "rate with comma no percent", input: "6,35", wantRate: 6.35, wantErr: false},
		{name: "rate with period and percent", input: "4.41 %", wantRate: 4.41, wantErr: false},
		{name: "rate with period no percent", input: "10.50", wantRate: 10.50, wantErr: false},
		{name: "integer rate", input: "6 %", wantRate: 6.0, wantErr: false},
		{name: "missing dash", input: "-", wantRate: 0, wantErr: true},
		{name: "empty string", input: "", wantRate: 0, wantErr: true},
		{name: "invalid format", input: "abc", wantRate: 0, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rate, err := crawler.parseMarginalenRate(tt.input)

			if (err != nil) != tt.wantErr {
				t.Errorf("wantErr=%v, got err=%v", tt.wantErr, err)
			}

			if !tt.wantErr && rate != tt.wantRate {
				t.Errorf("expected rate=%.2f, got=%.2f", tt.wantRate, rate)
			}
		})
	}
}

func TestMarginalenCrawler_ParseMarginalenPeriod(t *testing.T) {
	t.Parallel()

	crawler := &MarginalenCrawler{logger: zap.NewNop()}

	tests := []struct {
		name      string
		input     string
		wantYear  int
		wantMonth int
		wantErr   bool
	}{
		{name: "valid period December 2024", input: "202412", wantYear: 2024, wantMonth: 12, wantErr: false},
		{name: "valid period January 2025", input: "202501", wantYear: 2025, wantMonth: 1, wantErr: false},
		{name: "valid period November 2025", input: "202511", wantYear: 2025, wantMonth: 11, wantErr: false},
		{name: "invalid month 13", input: "202513", wantYear: 0, wantMonth: 0, wantErr: true},
		{name: "invalid month 00", input: "202500", wantYear: 0, wantMonth: 0, wantErr: true},
		{name: "invalid format YYYY-MM", input: "2025-11", wantYear: 0, wantMonth: 0, wantErr: true},
		{name: "invalid format short", input: "20251", wantYear: 0, wantMonth: 0, wantErr: true},
		{name: "empty string", input: "", wantYear: 0, wantMonth: 0, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			date, err := crawler.parseMarginalenPeriod(tt.input)

			if (err != nil) != tt.wantErr {
				t.Errorf("wantErr=%v, got err=%v", tt.wantErr, err)
			}

			if !tt.wantErr {
				if date.Year() != tt.wantYear {
					t.Errorf("expected year=%d, got=%d", tt.wantYear, date.Year())
				}
				if int(date.Month()) != tt.wantMonth {
					t.Errorf("expected month=%d, got=%d", tt.wantMonth, int(date.Month()))
				}
			}
		})
	}
}

func TestMarginalenCrawler_ExtractTermsFromHeader(t *testing.T) {
	t.Parallel()

	crawler := &MarginalenCrawler{logger: zap.NewNop()}

	tests := []struct {
		name      string
		headers   []string
		wantTerms []model.Term
		wantErr   bool
	}{
		{
			name:      "valid header row",
			headers:   []string{"3 Mån", "6 Mån", "1 år", "2 år", "3 år"},
			wantTerms: []model.Term{model.Term3months, model.Term6months, model.Term1year, model.Term2years, model.Term3years},
			wantErr:   false,
		},
		{
			name:      "partial valid headers",
			headers:   []string{"3 Mån", "invalid", "1 år"},
			wantTerms: []model.Term{model.Term3months, model.Term1year},
			wantErr:   false,
		},
		{
			name:      "empty headers",
			headers:   []string{},
			wantTerms: nil,
			wantErr:   true,
		},
		{
			name:      "all invalid headers",
			headers:   []string{"invalid", "also invalid"},
			wantTerms: nil,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			terms, err := crawler.extractTermsFromHeader(tt.headers)

			if (err != nil) != tt.wantErr {
				t.Errorf("wantErr=%v, got err=%v", tt.wantErr, err)
			}

			if !tt.wantErr {
				if len(terms) != len(tt.wantTerms) {
					t.Errorf("expected %d terms, got %d", len(tt.wantTerms), len(terms))
				}

				for i, wantTerm := range tt.wantTerms {
					if i < len(terms) && terms[i] != wantTerm {
						t.Errorf("term[%d]: expected=%s, got=%s", i, wantTerm, terms[i])
					}
				}
			}
		})
	}
}
