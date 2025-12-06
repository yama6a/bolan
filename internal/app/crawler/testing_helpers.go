package crawler

import (
	"os"
	"testing"
	"time"

	"github.com/yama6a/bolan-compare/internal/pkg/model"
)

// LoadGoldenFile loads a golden file from disk and returns its contents as a string.
// The filename should be relative to the test's working directory (typically the package directory).
func LoadGoldenFile(t *testing.T, filename string) string {
	t.Helper()
	data, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("failed to read golden file %s: %v", filename, err)
	}
	return string(data)
}

// LoadGoldenFileBytes loads a golden file from disk and returns its contents as bytes.
// Useful for binary files like XLSX, PDF, etc.
func LoadGoldenFileBytes(t *testing.T, filename string) []byte {
	t.Helper()
	data, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("failed to read golden file %s: %v", filename, err)
	}
	return data
}

// CountRatesByType counts the number of list and average rate results.
func CountRatesByType(results []model.InterestSet) (listCount, avgCount int) {
	for _, r := range results {
		switch r.Type {
		case model.TypeListRate:
			listCount++
		case model.TypeAverageRate:
			avgCount++
		case model.TypeRatioDiscounted, model.TypeUnionDiscounted:
			// Discounted rates are not counted separately in this function
		}
	}
	return listCount, avgCount
}

// AssertBankName checks that all results have the expected bank name.
func AssertBankName(t *testing.T, results []model.InterestSet, wantBank model.Bank) {
	t.Helper()
	for _, r := range results {
		if r.Bank != wantBank {
			t.Errorf("Bank = %q, want %q", r.Bank, wantBank)
		}
	}
}

// TestInvalidJSON tests that a crawler handles invalid JSON gracefully.
func TestInvalidJSON(t *testing.T, parseFunc func(string) error) {
	t.Helper()
	tests := []struct {
		name string
		json string
	}{
		{
			name: "empty string",
			json: "",
		},
		{
			name: "invalid JSON",
			json: "{invalid}",
		},
		{
			name: "null",
			json: "null",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := parseFunc(tt.json)
			if err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

// RunCrawl executes a crawler and collects all results into a slice.
// This helper eliminates the boilerplate of creating a channel, running the crawler,
// and collecting results.
func RunCrawl(t *testing.T, crawler SiteCrawler) []model.InterestSet {
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

// ListRateConfig specifies bank-specific validation requirements for list rates.
type ListRateConfig struct {
	Bank           model.Bank
	ExpectChangeOn bool // true if bank API provides ChangedOn dates
}

// AssertListRateFields validates common fields for list rate results.
func AssertListRateFields(t *testing.T, r model.InterestSet, cfg ListRateConfig, crawlTime time.Time) {
	t.Helper()

	if r.Bank != cfg.Bank {
		t.Errorf("Bank = %q, want %q", r.Bank, cfg.Bank)
	}
	if r.Type != model.TypeListRate {
		t.Errorf("Type = %q, want %q", r.Type, model.TypeListRate)
	}
	if r.NominalRate <= 0 {
		t.Errorf("NominalRate = %f, want positive value", r.NominalRate)
	}
	if cfg.ExpectChangeOn && r.ChangedOn == nil {
		t.Error("ChangedOn is nil, want non-nil")
	}
	if !cfg.ExpectChangeOn && r.ChangedOn != nil {
		t.Error("ChangedOn is not nil, want nil")
	}
	if r.LastCrawledAt != crawlTime {
		t.Errorf("LastCrawledAt = %v, want %v", r.LastCrawledAt, crawlTime)
	}
}

// AssertAverageRateFields validates common fields for average rate results.
func AssertAverageRateFields(t *testing.T, r model.InterestSet, bank model.Bank, crawlTime time.Time) {
	t.Helper()

	if r.Bank != bank {
		t.Errorf("Bank = %q, want %q", r.Bank, bank)
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
