package crawler

import (
	"os"
	"testing"

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
