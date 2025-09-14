package store

import (
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/yama6a/bolan-compare/internal/pkg/model"
	"go.uber.org/zap"
)

type MemoryStore struct {
	logger *zap.Logger
	data   []model.InterestSet
}

func NewMemoryStore(_ *pgxpool.Pool, logger *zap.Logger) *MemoryStore {
	return &MemoryStore{
		logger: logger,
		data:   []model.InterestSet{},
	}
}

func alreadyExists(a model.InterestSet, b model.InterestSet) bool {
	if a.Bank != b.Bank || a.Type != b.Type || a.Term != b.Term {
		return false
	}

	if a.Type == model.TypeAverageRate {
		if a.AverageReferenceMonth == nil || b.AverageReferenceMonth == nil {
			return false
		}

		if a.AverageReferenceMonth.Month != b.AverageReferenceMonth.Month || a.AverageReferenceMonth.Year != b.AverageReferenceMonth.Year {
			return false
		}
	}

	return true
}

func (s *MemoryStore) UpsertInterestSet(set model.InterestSet) error {
	s.logger.Debug("upserting InterestSet", zap.Any("interestSet", set))

	// Check if an entry with the same Bank, Type, and Term already exists
	for i, existing := range s.data {
		if alreadyExists(existing, set) {
			s.logger.Debug("updating existing InterestSet",
				zap.String("bank", string(set.Bank)),
				zap.String("type", string(set.Type)),
				zap.String("term", string(set.Term)),
				zap.Float32("oldRate", existing.NominalRate),
				zap.Float32("newRate", set.NominalRate))

			// Update the existing entry
			s.data[i] = set
			return nil
		}
	}

	// No existing entry found, append new one
	s.logger.Debug("adding new InterestSet",
		zap.String("bank", string(set.Bank)),
		zap.String("type", string(set.Type)),
		zap.String("term", string(set.Term)))
	s.data = append(s.data, set)

	// todo: warn-log and store somewhere else when overwriting in PG Database, because bank tries to alter history?
	return nil
}

func (s *MemoryStore) GetInterestSets() ([]model.InterestSet, error) {
	return s.data, nil
}
