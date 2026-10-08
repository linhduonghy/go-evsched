package db

import (
	"context"
	"evsched/config"
	"log/slog"

	"github.com/exaring/otelpgx"
	"github.com/jackc/pgx/v5/pgxpool"
)

type QueryTracer struct {
	log *slog.Logger
}

func NewPostgres(_ context.Context, cfg config.DBConfig) (*pgxpool.Pool, error) {

	poolConfig, err := pgxpool.ParseConfig(cfg.URL())
	if err != nil {
		return nil, err
	}

	poolConfig.MaxConns = cfg.MaxOpenConns
	poolConfig.MinConns = cfg.MinIdleConns
	poolConfig.ConnConfig.Tracer = otelpgx.NewTracer()

	pool, err := pgxpool.NewWithConfig(
		context.Background(),
		poolConfig,
	)
	if err != nil {
		return nil, err
	}

	if err = pool.Ping(context.Background()); err != nil {
		pool.Close()
		return nil, err
	}

	return pool, nil
}
