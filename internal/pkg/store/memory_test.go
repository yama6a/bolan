// #nosec G404 // not used in security context, no strong randomness needed
package store

import (
	"math/rand"
	"reflect"
	"testing"
	"time"

	"github.com/yama6a/bolan-compare/internal/pkg/model"
	"go.uber.org/zap"
)

// generateInterestSet generates random interest sets for testing.
func generateInterestSet(interestType model.Type, term model.Term, rate float32) model.InterestSet {
	baseTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)

	// Random bank names
	banks := []model.Bank{"SEB", "Danske Bank", "Swedbank", "Handelsbanken", "Nordea", "Länsförsäkringar"}

	set := model.InterestSet{
		Bank:          banks[rand.Intn(len(banks))],
		Type:          interestType,
		Term:          term,
		NominalRate:   rate,
		ChangedOn:     nil,
		LastCrawledAt: baseTime.Add(time.Duration(rand.Intn(24)) * time.Hour),
	}

	// Add type-specific fields
	switch interestType { //nolint:exhaustive
	case model.TypeAverageRate:
		set.AverageReferenceMonth = &model.AvgMonth{
			Month: time.Month(rand.Intn(12) + 1),
			Year:  uint(2024),
		}
	case model.TypeRatioDiscounted:
		set.RatioDiscountBoundaries = &model.RatioDiscountBoundary{
			MinRatio: rand.Float32() * 0.5,
			MaxRatio: 0.5 + rand.Float32()*0.5,
		}
	case model.TypeUnionDiscounted:
		set.UnionDiscount = true
	}

	return set
}

func TestMemoryStore_GetInterestSets(t *testing.T) {
	t.Parallel()

	// Pre-generate test data - only set critical attributes
	listRate := generateInterestSet(model.TypeListRate, model.Term1year, 3.5)
	avgRateJan := generateInterestSet(model.TypeAverageRate, model.Term2years, 4.2)
	avgRateJan.AverageReferenceMonth = &model.AvgMonth{Month: time.January, Year: 2024}

	tests := []struct {
		name         string
		existingData []model.InterestSet
		wantData     []model.InterestSet
	}{
		{
			name:         "empty store",
			existingData: []model.InterestSet{},
			wantData:     []model.InterestSet{},
		},
		{
			name:         "single interest set",
			existingData: []model.InterestSet{listRate},
			wantData:     []model.InterestSet{listRate},
		},
		{
			name:         "multiple interest sets",
			existingData: []model.InterestSet{listRate, avgRateJan},
			wantData:     []model.InterestSet{listRate, avgRateJan},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := &MemoryStore{
				logger: zap.NewNop(),
				data:   tt.existingData,
			}
			got, err := s.GetInterestSets()
			if err != nil {
				t.Errorf("GetInterestSets() error = %v", err)
			}
			if !reflect.DeepEqual(got, tt.wantData) {
				t.Errorf("GetInterestSets() got = %v, wantData %v", got, tt.wantData)
			}
		})
	}
}

func TestMemoryStore_UpsertInterestSet(t *testing.T) {
	t.Parallel()

	// Pre-generate test data - only set critical attributes, let others be randomized
	listRate := generateInterestSet(model.TypeListRate, model.Term1year, 3.5)
	listRateUpdated := generateInterestSet(model.TypeListRate, model.Term1year, 4.0)
	// Ensure they have the same bank for matching
	listRateUpdated.Bank = listRate.Bank

	avgRateJan := generateInterestSet(model.TypeAverageRate, model.Term2years, 4.2)
	avgRateJan.AverageReferenceMonth = &model.AvgMonth{Month: time.January, Year: 2024}
	avgRateJanUpdated := generateInterestSet(model.TypeAverageRate, model.Term2years, 4.5)
	avgRateJanUpdated.AverageReferenceMonth = &model.AvgMonth{Month: time.January, Year: 2024}
	// Ensure they have the same bank for matching
	avgRateJanUpdated.Bank = avgRateJan.Bank

	avgRateFeb := generateInterestSet(model.TypeAverageRate, model.Term2years, 4.8)
	avgRateFeb.AverageReferenceMonth = &model.AvgMonth{Month: time.February, Year: 2024}
	// Ensure they have the same bank for matching
	avgRateFeb.Bank = avgRateJan.Bank

	ratioDiscounted := generateInterestSet(model.TypeRatioDiscounted, model.Term3years, 2.8)
	unionDiscounted := generateInterestSet(model.TypeUnionDiscounted, model.Term5years, 2.1)

	withChangedOn := generateInterestSet(model.TypeListRate, model.Term10years, 4.8)
	changedTime := time.Date(2024, 1, 10, 9, 0, 0, 0, time.UTC)
	withChangedOn.ChangedOn = &changedTime

	tests := []struct {
		name         string
		existingData []model.InterestSet
		arg          model.InterestSet
		wantData     []model.InterestSet
	}{
		{
			name:         "add to empty store",
			existingData: []model.InterestSet{},
			arg:          listRate,
			wantData:     []model.InterestSet{listRate},
		},
		{
			name:         "add to existing data",
			existingData: []model.InterestSet{listRate},
			arg:          avgRateJan,
			wantData:     []model.InterestSet{listRate, avgRateJan},
		},
		{
			name:         "add ratio discounted rate",
			existingData: []model.InterestSet{},
			arg:          ratioDiscounted,
			wantData:     []model.InterestSet{ratioDiscounted},
		},
		{
			name:         "add union discounted rate",
			existingData: []model.InterestSet{},
			arg:          unionDiscounted,
			wantData:     []model.InterestSet{unionDiscounted},
		},
		{
			name:         "add with changed on date",
			existingData: []model.InterestSet{},
			arg:          withChangedOn,
			wantData:     []model.InterestSet{withChangedOn},
		},
		{
			name:         "update existing entry (replaces duplicate)",
			existingData: []model.InterestSet{listRate},
			arg:          listRateUpdated,
			wantData:     []model.InterestSet{listRateUpdated},
		},
		{
			name:         "update average rate with same reference month",
			existingData: []model.InterestSet{avgRateJan},
			arg:          avgRateJanUpdated,
			wantData:     []model.InterestSet{avgRateJanUpdated},
		},
		{
			name:         "add average rate with different reference month",
			existingData: []model.InterestSet{avgRateJan},
			arg:          avgRateFeb,
			wantData:     []model.InterestSet{avgRateJan, avgRateFeb},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := &MemoryStore{
				logger: zap.NewNop(),
				data:   tt.existingData,
			}
			if err := s.UpsertInterestSet(tt.arg); err != nil {
				t.Errorf("UpsertInterestSet() error = %v", err)
			}
			if !reflect.DeepEqual(s.data, tt.wantData) {
				t.Errorf("UpsertInterestSet() got = %v, wantData %v", s.data, tt.wantData)
			}
		})
	}
}

func TestMemoryStore_UpsertInterestSet_NonMatchingCases(t *testing.T) {
	t.Parallel()

	// Base entry that we'll compare against
	baseEntry := model.InterestSet{
		Bank:        "Nordea",
		Type:        model.TypeListRate,
		Term:        model.Term1year,
		NominalRate: 3.5,
	}

	tests := []struct {
		name      string
		existing  model.InterestSet
		newEntry  model.InterestSet
		wantCount int // expected number of entries after upsert
	}{
		{
			name:     "different bank adds new entry",
			existing: baseEntry,
			newEntry: model.InterestSet{
				Bank:        "SEB", // Different bank
				Type:        model.TypeListRate,
				Term:        model.Term1year,
				NominalRate: 3.6,
			},
			wantCount: 2,
		},
		{
			name:     "different type adds new entry",
			existing: baseEntry,
			newEntry: model.InterestSet{
				Bank:        "Nordea",
				Type:        model.TypeRatioDiscounted, // Different type
				Term:        model.Term1year,
				NominalRate: 3.2,
			},
			wantCount: 2,
		},
		{
			name:     "different term adds new entry",
			existing: baseEntry,
			newEntry: model.InterestSet{
				Bank:        "Nordea",
				Type:        model.TypeListRate,
				Term:        model.Term5years, // Different term
				NominalRate: 4.0,
			},
			wantCount: 2,
		},
		{
			name:     "same bank/type/term updates existing entry",
			existing: baseEntry,
			newEntry: model.InterestSet{
				Bank:        "Nordea",
				Type:        model.TypeListRate,
				Term:        model.Term1year,
				NominalRate: 4.5, // Only rate different
			},
			wantCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			s := &MemoryStore{
				logger: zap.NewNop(),
				data:   []model.InterestSet{tt.existing},
			}

			if err := s.UpsertInterestSet(tt.newEntry); err != nil {
				t.Fatalf("UpsertInterestSet() error = %v", err)
			}

			if len(s.data) != tt.wantCount {
				t.Errorf("UpsertInterestSet() resulted in %d entries, want %d", len(s.data), tt.wantCount)
			}
		})
	}
}

func TestMemoryStore_UpsertInterestSet_AverageRateEdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("average rate with nil reference month does not match", func(t *testing.T) {
		t.Parallel()

		existing := model.InterestSet{
			Bank:                  "Nordea",
			Type:                  model.TypeAverageRate,
			Term:                  model.Term1year,
			NominalRate:           3.5,
			AverageReferenceMonth: nil, // nil reference month
		}

		newEntry := model.InterestSet{
			Bank:                  "Nordea",
			Type:                  model.TypeAverageRate,
			Term:                  model.Term1year,
			NominalRate:           3.8,
			AverageReferenceMonth: &model.AvgMonth{Month: time.January, Year: 2024},
		}

		s := &MemoryStore{
			logger: zap.NewNop(),
			data:   []model.InterestSet{existing},
		}

		if err := s.UpsertInterestSet(newEntry); err != nil {
			t.Fatalf("UpsertInterestSet() error = %v", err)
		}

		// Should add new entry since existing has nil AverageReferenceMonth
		if len(s.data) != 2 {
			t.Errorf("Expected 2 entries (nil ref month should not match), got %d", len(s.data))
		}
	})

	t.Run("average rate same month different year adds new entry", func(t *testing.T) {
		t.Parallel()

		existing := model.InterestSet{
			Bank:                  "Nordea",
			Type:                  model.TypeAverageRate,
			Term:                  model.Term1year,
			NominalRate:           3.5,
			AverageReferenceMonth: &model.AvgMonth{Month: time.January, Year: 2024},
		}

		newEntry := model.InterestSet{
			Bank:                  "Nordea",
			Type:                  model.TypeAverageRate,
			Term:                  model.Term1year,
			NominalRate:           3.8,
			AverageReferenceMonth: &model.AvgMonth{Month: time.January, Year: 2025}, // Different year
		}

		s := &MemoryStore{
			logger: zap.NewNop(),
			data:   []model.InterestSet{existing},
		}

		if err := s.UpsertInterestSet(newEntry); err != nil {
			t.Fatalf("UpsertInterestSet() error = %v", err)
		}

		if len(s.data) != 2 {
			t.Errorf("Expected 2 entries (different year should not match), got %d", len(s.data))
		}
	})

	t.Run("both average rates with nil reference month do not match", func(t *testing.T) {
		t.Parallel()

		existing := model.InterestSet{
			Bank:                  "Nordea",
			Type:                  model.TypeAverageRate,
			Term:                  model.Term1year,
			NominalRate:           3.5,
			AverageReferenceMonth: nil,
		}

		newEntry := model.InterestSet{
			Bank:                  "Nordea",
			Type:                  model.TypeAverageRate,
			Term:                  model.Term1year,
			NominalRate:           3.8,
			AverageReferenceMonth: nil, // Also nil
		}

		s := &MemoryStore{
			logger: zap.NewNop(),
			data:   []model.InterestSet{existing},
		}

		if err := s.UpsertInterestSet(newEntry); err != nil {
			t.Fatalf("UpsertInterestSet() error = %v", err)
		}

		// According to alreadyExists logic, both nil returns false, so new entry is added
		if len(s.data) != 2 {
			t.Errorf("Expected 2 entries (both nil should not match), got %d", len(s.data))
		}
	})
}

func TestMemoryStore_UpsertInterestSet_MultipleUpdates(t *testing.T) {
	t.Parallel()

	s := NewMemoryStore(nil, zap.NewNop())

	// First insert
	entry1 := model.InterestSet{
		Bank:        "Nordea",
		Type:        model.TypeListRate,
		Term:        model.Term1year,
		NominalRate: 3.0,
	}
	if err := s.UpsertInterestSet(entry1); err != nil {
		t.Fatalf("First UpsertInterestSet() error = %v", err)
	}

	// Second update (same key)
	entry2 := model.InterestSet{
		Bank:        "Nordea",
		Type:        model.TypeListRate,
		Term:        model.Term1year,
		NominalRate: 3.5,
	}
	if err := s.UpsertInterestSet(entry2); err != nil {
		t.Fatalf("Second UpsertInterestSet() error = %v", err)
	}

	// Third update (same key)
	entry3 := model.InterestSet{
		Bank:        "Nordea",
		Type:        model.TypeListRate,
		Term:        model.Term1year,
		NominalRate: 4.0,
	}
	if err := s.UpsertInterestSet(entry3); err != nil {
		t.Fatalf("Third UpsertInterestSet() error = %v", err)
	}

	// Should still have only 1 entry
	if len(s.data) != 1 {
		t.Errorf("Expected 1 entry after multiple updates, got %d", len(s.data))
	}

	// Should have the last rate
	if s.data[0].NominalRate != 4.0 {
		t.Errorf("Expected rate 4.0 after updates, got %f", s.data[0].NominalRate)
	}
}

func TestMemoryStore_UpsertInterestSet_PreservesOtherEntries(t *testing.T) {
	t.Parallel()

	// Create store with multiple entries
	entries := []model.InterestSet{
		{Bank: "Nordea", Type: model.TypeListRate, Term: model.Term1year, NominalRate: 3.0},
		{Bank: "SEB", Type: model.TypeListRate, Term: model.Term1year, NominalRate: 3.1},
		{Bank: "Swedbank", Type: model.TypeListRate, Term: model.Term1year, NominalRate: 3.2},
	}

	s := &MemoryStore{
		logger: zap.NewNop(),
		data:   append([]model.InterestSet{}, entries...), // Copy slice
	}

	// Update the middle entry
	updatedEntry := model.InterestSet{
		Bank:        "SEB",
		Type:        model.TypeListRate,
		Term:        model.Term1year,
		NominalRate: 3.5, // Updated rate
	}

	if err := s.UpsertInterestSet(updatedEntry); err != nil {
		t.Fatalf("UpsertInterestSet() error = %v", err)
	}

	// Should still have 3 entries
	if len(s.data) != 3 {
		t.Fatalf("Expected 3 entries, got %d", len(s.data))
	}

	// Verify each entry
	if s.data[0].Bank != "Nordea" || s.data[0].NominalRate != 3.0 {
		t.Errorf("First entry modified unexpectedly: %+v", s.data[0])
	}
	if s.data[1].Bank != "SEB" || s.data[1].NominalRate != 3.5 {
		t.Errorf("Second entry not updated correctly: %+v", s.data[1])
	}
	if s.data[2].Bank != "Swedbank" || s.data[2].NominalRate != 3.2 {
		t.Errorf("Third entry modified unexpectedly: %+v", s.data[2])
	}
}
