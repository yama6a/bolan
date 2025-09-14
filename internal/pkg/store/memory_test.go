package store

import (
	"reflect"
	"testing"
	"time"

	"github.com/yama6a/bolan-compare/internal/pkg/model"
	"go.uber.org/zap"
)

func TestMemoryStore_GetInterestSets(t *testing.T) {
	t.Parallel()
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
			name: "single interest set",
			existingData: []model.InterestSet{
				{
					Bank:          "SEB",
					Type:          model.TypeListRate,
					Term:          model.Term1year,
					NominalRate:   3.5,
					ChangedOn:     nil,
					LastCrawledAt: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
				},
			},
			wantData: []model.InterestSet{
				{
					Bank:          "SEB",
					Type:          model.TypeListRate,
					Term:          model.Term1year,
					NominalRate:   3.5,
					ChangedOn:     nil,
					LastCrawledAt: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
				},
			},
		},
		{
			name: "multiple interest sets",
			existingData: []model.InterestSet{
				{
					Bank:          "SEB",
					Type:          model.TypeListRate,
					Term:          model.Term1year,
					NominalRate:   3.5,
					ChangedOn:     nil,
					LastCrawledAt: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
				},
				{
					Bank:          "Danske Bank",
					Type:          model.TypeAverageRate,
					Term:          model.Term2years,
					NominalRate:   4.2,
					ChangedOn:     nil,
					LastCrawledAt: time.Date(2024, 1, 15, 11, 0, 0, 0, time.UTC),
					AverageReferenceMonth: &model.AvgMonth{
						Month: time.January,
						Year:  2024,
					},
				},
			},
			wantData: []model.InterestSet{
				{
					Bank:          "SEB",
					Type:          model.TypeListRate,
					Term:          model.Term1year,
					NominalRate:   3.5,
					ChangedOn:     nil,
					LastCrawledAt: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
				},
				{
					Bank:          "Danske Bank",
					Type:          model.TypeAverageRate,
					Term:          model.Term2years,
					NominalRate:   4.2,
					ChangedOn:     nil,
					LastCrawledAt: time.Date(2024, 1, 15, 11, 0, 0, 0, time.UTC),
					AverageReferenceMonth: &model.AvgMonth{
						Month: time.January,
						Year:  2024,
					},
				},
			},
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

	// Helper to create time pointer
	changedTime := time.Date(2024, 1, 10, 9, 0, 0, 0, time.UTC)

	tests := []struct {
		name         string
		existingData []model.InterestSet
		arg          model.InterestSet
		wantData     []model.InterestSet
	}{
		{
			name:         "add to empty store",
			existingData: []model.InterestSet{},
			arg: model.InterestSet{
				Bank:          "SEB",
				Type:          model.TypeListRate,
				Term:          model.Term1year,
				NominalRate:   3.5,
				ChangedOn:     nil,
				LastCrawledAt: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
			},
			wantData: []model.InterestSet{
				{
					Bank:          "SEB",
					Type:          model.TypeListRate,
					Term:          model.Term1year,
					NominalRate:   3.5,
					ChangedOn:     nil,
					LastCrawledAt: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
				},
			},
		},
		{
			name: "add to existing data",
			existingData: []model.InterestSet{
				{
					Bank:          "SEB",
					Type:          model.TypeListRate,
					Term:          model.Term1year,
					NominalRate:   3.5,
					ChangedOn:     nil,
					LastCrawledAt: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
				},
			},
			arg: model.InterestSet{
				Bank:          "Danske Bank",
				Type:          model.TypeAverageRate,
				Term:          model.Term2years,
				NominalRate:   4.2,
				ChangedOn:     nil,
				LastCrawledAt: time.Date(2024, 1, 15, 11, 0, 0, 0, time.UTC),
				AverageReferenceMonth: &model.AvgMonth{
					Month: time.January,
					Year:  2024,
				},
			},
			wantData: []model.InterestSet{
				{
					Bank:          "SEB",
					Type:          model.TypeListRate,
					Term:          model.Term1year,
					NominalRate:   3.5,
					ChangedOn:     nil,
					LastCrawledAt: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
				},
				{
					Bank:          "Danske Bank",
					Type:          model.TypeAverageRate,
					Term:          model.Term2years,
					NominalRate:   4.2,
					ChangedOn:     nil,
					LastCrawledAt: time.Date(2024, 1, 15, 11, 0, 0, 0, time.UTC),
					AverageReferenceMonth: &model.AvgMonth{
						Month: time.January,
						Year:  2024,
					},
				},
			},
		},
		{
			name:         "add ratio discounted rate",
			existingData: []model.InterestSet{},
			arg: model.InterestSet{
				Bank:          "Swedbank",
				Type:          model.TypeRatioDiscounted,
				Term:          model.Term3years,
				NominalRate:   2.8,
				ChangedOn:     nil,
				LastCrawledAt: time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC),
				RatioDiscountBoundaries: &model.RatioDiscountBoundary{
					MinRatio: 0.5,
					MaxRatio: 0.8,
				},
			},
			wantData: []model.InterestSet{
				{
					Bank:          "Swedbank",
					Type:          model.TypeRatioDiscounted,
					Term:          model.Term3years,
					NominalRate:   2.8,
					ChangedOn:     nil,
					LastCrawledAt: time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC),
					RatioDiscountBoundaries: &model.RatioDiscountBoundary{
						MinRatio: 0.5,
						MaxRatio: 0.8,
					},
				},
			},
		},
		{
			name:         "add union discounted rate",
			existingData: []model.InterestSet{},
			arg: model.InterestSet{
				Bank:          "Handelsbanken",
				Type:          model.TypeUnionDiscounted,
				Term:          model.Term5years,
				NominalRate:   2.1,
				ChangedOn:     nil,
				LastCrawledAt: time.Date(2024, 1, 15, 13, 0, 0, 0, time.UTC),
				UnionDiscount: true,
			},
			wantData: []model.InterestSet{
				{
					Bank:          "Handelsbanken",
					Type:          model.TypeUnionDiscounted,
					Term:          model.Term5years,
					NominalRate:   2.1,
					ChangedOn:     nil,
					LastCrawledAt: time.Date(2024, 1, 15, 13, 0, 0, 0, time.UTC),
					UnionDiscount: true,
				},
			},
		},
		{
			name:         "add with changed on date",
			existingData: []model.InterestSet{},
			arg: model.InterestSet{
				Bank:          "Nordea",
				Type:          model.TypeListRate,
				Term:          model.Term10years,
				NominalRate:   4.8,
				ChangedOn:     &changedTime,
				LastCrawledAt: time.Date(2024, 1, 15, 14, 0, 0, 0, time.UTC),
			},
			wantData: []model.InterestSet{
				{
					Bank:          "Nordea",
					Type:          model.TypeListRate,
					Term:          model.Term10years,
					NominalRate:   4.8,
					ChangedOn:     &changedTime,
					LastCrawledAt: time.Date(2024, 1, 15, 14, 0, 0, 0, time.UTC),
				},
			},
		},
		{
			name: "update existing entry (replaces duplicate)",
			existingData: []model.InterestSet{
				{
					Bank:          "SEB",
					Type:          model.TypeListRate,
					Term:          model.Term1year,
					NominalRate:   3.5,
					ChangedOn:     nil,
					LastCrawledAt: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
				},
			},
			arg: model.InterestSet{
				Bank:          "SEB",
				Type:          model.TypeListRate,
				Term:          model.Term1year,
				NominalRate:   4.0, // Updated rate
				ChangedOn:     &changedTime,
				LastCrawledAt: time.Date(2024, 1, 16, 10, 0, 0, 0, time.UTC), // Updated crawl time
			},
			wantData: []model.InterestSet{
				{
					Bank:          "SEB",
					Type:          model.TypeListRate,
					Term:          model.Term1year,
					NominalRate:   4.0, // Updated rate
					ChangedOn:     &changedTime,
					LastCrawledAt: time.Date(2024, 1, 16, 10, 0, 0, 0, time.UTC), // Updated crawl time
				},
			},
		},
		{
			name: "update average rate with same reference month",
			existingData: []model.InterestSet{
				{
					Bank:          "Danske Bank",
					Type:          model.TypeAverageRate,
					Term:          model.Term2years,
					NominalRate:   4.2,
					ChangedOn:     nil,
					LastCrawledAt: time.Date(2024, 1, 15, 11, 0, 0, 0, time.UTC),
					AverageReferenceMonth: &model.AvgMonth{
						Month: time.January,
						Year:  2024,
					},
				},
			},
			arg: model.InterestSet{
				Bank:          "Danske Bank",
				Type:          model.TypeAverageRate,
				Term:          model.Term2years,
				NominalRate:   4.5, // Updated rate
				ChangedOn:     &changedTime,
				LastCrawledAt: time.Date(2024, 1, 16, 11, 0, 0, 0, time.UTC), // Updated crawl time
				AverageReferenceMonth: &model.AvgMonth{
					Month: time.January, // Same month
					Year:  2024,         // Same year
				},
			},
			wantData: []model.InterestSet{
				{
					Bank:          "Danske Bank",
					Type:          model.TypeAverageRate,
					Term:          model.Term2years,
					NominalRate:   4.5, // Updated rate
					ChangedOn:     &changedTime,
					LastCrawledAt: time.Date(2024, 1, 16, 11, 0, 0, 0, time.UTC), // Updated crawl time
					AverageReferenceMonth: &model.AvgMonth{
						Month: time.January,
						Year:  2024,
					},
				},
			},
		},
		{
			name: "add average rate with different reference month",
			existingData: []model.InterestSet{
				{
					Bank:          "Danske Bank",
					Type:          model.TypeAverageRate,
					Term:          model.Term2years,
					NominalRate:   4.2,
					ChangedOn:     nil,
					LastCrawledAt: time.Date(2024, 1, 15, 11, 0, 0, 0, time.UTC),
					AverageReferenceMonth: &model.AvgMonth{
						Month: time.January,
						Year:  2024,
					},
				},
			},
			arg: model.InterestSet{
				Bank:          "Danske Bank",
				Type:          model.TypeAverageRate,
				Term:          model.Term2years,
				NominalRate:   4.8, // Different rate
				ChangedOn:     &changedTime,
				LastCrawledAt: time.Date(2024, 1, 16, 11, 0, 0, 0, time.UTC),
				AverageReferenceMonth: &model.AvgMonth{
					Month: time.February, // Different month
					Year:  2024,
				},
			},
			wantData: []model.InterestSet{
				{
					Bank:          "Danske Bank",
					Type:          model.TypeAverageRate,
					Term:          model.Term2years,
					NominalRate:   4.2,
					ChangedOn:     nil,
					LastCrawledAt: time.Date(2024, 1, 15, 11, 0, 0, 0, time.UTC),
					AverageReferenceMonth: &model.AvgMonth{
						Month: time.January,
						Year:  2024,
					},
				},
				{
					Bank:          "Danske Bank",
					Type:          model.TypeAverageRate,
					Term:          model.Term2years,
					NominalRate:   4.8, // New entry
					ChangedOn:     &changedTime,
					LastCrawledAt: time.Date(2024, 1, 16, 11, 0, 0, 0, time.UTC),
					AverageReferenceMonth: &model.AvgMonth{
						Month: time.February, // Different month
						Year:  2024,
					},
				},
			},
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
