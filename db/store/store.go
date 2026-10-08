package store

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	dbPool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{dbPool: pool}
}

func (s *Store) DBPool() *pgxpool.Pool {
	return s.dbPool
}

func (s *Store) WithTx(
	ctx context.Context,
	fn func(tx pgx.Tx) error,
) error {

	tx, err := s.dbPool.Begin(ctx)
	if err != nil {
		return err
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if err = fn(tx); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (s *Store) CopyFromRows(ctx context.Context, tableName string, columnNames []string, dataRows [][]any) error {
	_, err := s.dbPool.CopyFrom(
		ctx,
		pgx.Identifier{tableName},
		columnNames,
		pgx.CopyFromRows(dataRows),
	)
	return err
}
