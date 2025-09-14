package store

import (
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/yama6a/bolan-compare/internal/pkg/model"
	"go.uber.org/zap"
)

type MemoryStore struct {
	pool   *pgxpool.Pool
	logger *zap.Logger
	data   []model.InterestSet
}

func NewMemoryStore(pool *pgxpool.Pool, logger *zap.Logger) *MemoryStore {
	return &MemoryStore{
		pool:   pool,
		logger: logger,
		data:   []model.InterestSet{},
	}
}

func (s *MemoryStore) UpsertInterestSet(set model.InterestSet) error {
	s.logger.Debug("adding InterestSet", zap.Any("interestSet", set))
	s.data = append(s.data, set)

	// todo: warn-log when overwriting in PG Database, because bank tries to alter history and store somewhere else?
	return nil
}

func (s *MemoryStore) GetInterestSets() []model.InterestSet {
	return s.data
}
