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

func loadBluestepGoldenFile(t *testing.T, filename string) string {
	t.Helper()

	data, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("failed to load golden file %s: %v", filename, err)
	}

	return string(data)
}

func TestNewBluestepCrawler(t *testing.T) {
	t.Parallel()

	mockClient := &httpmock.ClientMock{}
	logger := zap.NewNop()

	crawler := NewBluestepCrawler(mockClient, logger)

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

func TestBluestepCrawler_InterfaceCompliance(t *testing.T) {
	t.Parallel()

	var _ SiteCrawler = &BluestepCrawler{}
}

func TestBluestepCrawler_Crawl(t *testing.T) {
	t.Parallel()

	listHTML := loadBluestepGoldenFile(t, "testdata/bluestep_list_rates.html")
	avgHTML := loadBluestepGoldenFile(t, "testdata/bluestep_avg_rates.html")
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
				if url == bluestepListRatesURL {
					return listHTML, nil
				}
				return avgHTML, nil
			},
			wantListRates:    3, // 3 mån, 3 år, 5 år
			wantAvgRates:     true,
			wantTotalMinimum: 15, // 3 list + at least 12 average (4 terms * ~12 months)
		},
		{
			name: "fetch error for list rates still returns average rates",
			mockFetch: func(url string, _ map[string]string) (string, error) {
				if url == bluestepListRatesURL {
					return "", errors.New("network error")
				}
				return avgHTML, nil
			},
			wantListRates:    0,
			wantAvgRates:     true,
			wantTotalMinimum: 10,
		},
		{
			name: "fetch error for both returns no results",
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

			results := runBluestepCrawl(t, tt.mockFetch, logger)
			verifyBluestepCrawlResults(t, results, tt.wantListRates, tt.wantAvgRates, tt.wantTotalMinimum)
		})
	}
}

func runBluestepCrawl(t *testing.T, mockFetch func(string, map[string]string) (string, error), logger *zap.Logger) []model.InterestSet {
	t.Helper()

	mockClient := &httpmock.ClientMock{FetchFunc: mockFetch}
	crawler := NewBluestepCrawler(mockClient, logger)
	resultChan := make(chan model.InterestSet, 200)

	crawler.Crawl(resultChan)
	close(resultChan)

	results := make([]model.InterestSet, 0, 200)
	for set := range resultChan {
		results = append(results, set)
	}

	return results
}

func verifyBluestepCrawlResults(t *testing.T, results []model.InterestSet, wantListRates int, wantAvgRates bool, wantTotalMinimum int) {
	t.Helper()

	listRateCount, avgRateCount := countBluestepRatesByType(results)

	if listRateCount != wantListRates {
		t.Errorf("list rate count = %d, want %d", listRateCount, wantListRates)
	}

	if wantAvgRates && avgRateCount == 0 {
		t.Error("expected average rates but got none")
	}

	if len(results) < wantTotalMinimum {
		t.Errorf("total results = %d, want at least %d", len(results), wantTotalMinimum)
	}

	for _, r := range results {
		if r.Bank != bluestepBankName {
			t.Errorf("expected bank %s, got %s", bluestepBankName, r.Bank)
		}
	}
}

// countBluestepRatesByType counts list rates and average rates from results.
func countBluestepRatesByType(results []model.InterestSet) (listCount, avgCount int) {
	for _, r := range results {
		switch r.Type {
		case model.TypeListRate:
			listCount++
		case model.TypeAverageRate:
			avgCount++
		case model.TypeRatioDiscounted, model.TypeUnionDiscounted:
			// Bluestep doesn't use discounted rates
		}
	}

	return listCount, avgCount
}

func TestBluestepCrawler_extractListRates(t *testing.T) {
	t.Parallel()

	goldenHTML := loadBluestepGoldenFile(t, "testdata/bluestep_list_rates.html")
	logger := zap.NewNop()
	crawler := NewBluestepCrawler(nil, logger)
	crawlTime := time.Now().UTC()

	rates, err := crawler.extractListRates(goldenHTML, crawlTime)
	if err != nil {
		t.Fatalf("extractListRates failed: %v", err)
	}

	// Expected: 3 terms (3 mån, 3 år, 5 år)
	if len(rates) != 3 {
		t.Errorf("expected 3 list rates, got %d", len(rates))
	}

	// Verify terms are correct
	expectedTerms := map[model.Term]bool{
		model.Term3months: false,
		model.Term3years:  false,
		model.Term5years:  false,
	}

	for _, r := range rates {
		if r.Type != model.TypeListRate {
			t.Errorf("expected TypeListRate, got %v", r.Type)
		}

		if r.Bank != bluestepBankName {
			t.Errorf("expected bank %s, got %s", bluestepBankName, r.Bank)
		}

		if r.NominalRate <= 0 {
			t.Errorf("expected positive rate, got %f for term %s", r.NominalRate, r.Term)
		}

		if _, ok := expectedTerms[r.Term]; ok {
			expectedTerms[r.Term] = true
		} else {
			t.Errorf("unexpected term: %s", r.Term)
		}
	}

	for term, found := range expectedTerms {
		if !found {
			t.Errorf("expected term %s not found", term)
		}
	}
}

func TestBluestepCrawler_extractAverageRates(t *testing.T) {
	t.Parallel()

	goldenHTML := loadBluestepGoldenFile(t, "testdata/bluestep_avg_rates.html")
	logger := zap.NewNop()
	crawler := NewBluestepCrawler(nil, logger)
	crawlTime := time.Now().UTC()

	rates, err := crawler.extractAverageRates(goldenHTML, crawlTime)
	if err != nil {
		t.Fatalf("extractAverageRates failed: %v", err)
	}

	// Expect multiple average rates (4 terms * ~12 months)
	if len(rates) < 10 {
		t.Errorf("expected at least 10 average rates, got %d", len(rates))
	}

	// Verify all rates are of correct type
	for _, r := range rates {
		if r.Type != model.TypeAverageRate {
			t.Errorf("expected TypeAverageRate, got %v", r.Type)
		}

		if r.Bank != bluestepBankName {
			t.Errorf("expected bank %s, got %s", bluestepBankName, r.Bank)
		}

		if r.AverageReferenceMonth == nil {
			t.Error("expected AverageReferenceMonth to be set")
		}

		if r.NominalRate <= 0 {
			t.Errorf("expected positive rate, got %f", r.NominalRate)
		}
	}
}

func TestBluestepCrawler_parseBluestepTerm(t *testing.T) {
	t.Parallel()

	logger := zap.NewNop()
	crawler := NewBluestepCrawler(nil, logger)

	tests := []struct {
		input    string
		expected model.Term
		wantErr  bool
	}{
		{"Rörlig 3 månader", model.Term3months, false},
		{"R&ouml;rlig 3 m&aring;nader", model.Term3months, false},
		{"Fast 3 år", model.Term3years, false},
		{"Fast 3 &aring;r", model.Term3years, false},
		{"Fast 5 år", model.Term5years, false},
		{"Fast 5 &aring;r", model.Term5years, false},
		{"1 år", model.Term1year, false},
		{"invalid", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()

			result, err := crawler.parseBluestepTerm(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestBluestepCrawler_parseBluestepRate(t *testing.T) {
	t.Parallel()

	logger := zap.NewNop()
	crawler := NewBluestepCrawler(nil, logger)

	tests := []struct {
		input    string
		expected float32
		wantErr  bool
	}{
		{"4,45%", 4.45, false},
		{"5.68%", 5.68, false},
		{"6,33%", 6.33, false},
		{"4.60%", 4.60, false},
		{"4,68%", 4.68, false},
		{"5,68", 5.68, false}, // without %
		{"invalid", 0, true},
		{"", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()

			result, err := crawler.parseBluestepRate(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			// Compare with tolerance for float comparison
			if diff := result - tt.expected; diff > 0.001 || diff < -0.001 {
				t.Errorf("expected %f, got %f", tt.expected, result)
			}
		})
	}
}

func TestBluestepCrawler_parseBluestepMonth(t *testing.T) {
	t.Parallel()

	logger := zap.NewNop()
	crawler := NewBluestepCrawler(nil, logger)

	tests := []struct {
		input         string
		expectedMonth time.Month
		expectedYear  uint
		wantErr       bool
	}{
		{"2025 11", time.November, 2025, false},
		{"2025 01", time.January, 2025, false},
		{"2024 12", time.December, 2024, false},
		{"2025  9", time.September, 2025, false},
		{"invalid", 0, 0, true},
		{"2025", 0, 0, true},
		{"2025 13", 0, 0, true}, // invalid month
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()

			result, err := crawler.parseBluestepMonth(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if result.Month != tt.expectedMonth {
				t.Errorf("expected month %v, got %v", tt.expectedMonth, result.Month)
			}

			if result.Year != tt.expectedYear {
				t.Errorf("expected year %d, got %d", tt.expectedYear, result.Year)
			}
		})
	}
}
