package store

import (
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/ymakhloufi/bolan-compare/internal/pkg/model"
	"go.uber.org/zap"
)

type PostgresStore struct {
	pool   *pgxpool.Pool
	logger *zap.Logger
}

func NewPostgres(pool *pgxpool.Pool, logger *zap.Logger) *PostgresStore {
	return &PostgresStore{
		pool:   pool,
		logger: logger,
	}
}

func (s PostgresStore) UpsertInterestSet(set model.InterestSet) error {
	s.logger.Info("upserting InterestSet", zap.Any("interestSet", set))
	// todo: warn-log when overwriting, because bank tries to alter history?
	return nil
}
