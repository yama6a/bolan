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

// Helper function to generate random interest sets
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
