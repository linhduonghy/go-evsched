package db

import (
	"context"
	"embed"
	"fmt"
	"hash/fnv"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivermigrate"
)

//go:embed sql/schema/job_schema.sql sql/schema/outbox_events_schema.sql
var schemaFS embed.FS

// Explicit list of schema files in the exact order you want them executed
var schemaFiles = []string{
	"sql/schema/job_schema.sql",
	"sql/schema/outbox_events_schema.sql",
}

// InitDB handles running DB schema init and River's queue init
func InitDB(ctx context.Context, logger *slog.Logger, dbPool *pgxpool.Pool) error {

	tx, err := dbPool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin migration lock transaction: %w", err)
	}
	defer tx.Rollback(context.Background())

	// Acquire Transaction-Level Advisory Lock
	if _, err = tx.Exec(
		ctx,
		"SELECT pg_advisory_xact_lock($1)",
		advisoryLockID("evsched:init-db"),
	); err != nil {
		return fmt.Errorf("failed to acquire init db advisory lock: %w", err)
	}

	// River schema init
	err = initRiverSchema(ctx, logger, dbPool)
	if err != nil {
		return fmt.Errorf("failed to init river schema: %w", err)
	}

	// app schema init
	err = initAppSchema(ctx, logger, tx)
	if err != nil {
		return fmt.Errorf("failed to init app db schema: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit migration transaction: %w", err)
	}

	return nil
}

func initAppSchema(ctx context.Context, logger *slog.Logger, tx pgx.Tx) error {
	for _, filePath := range schemaFiles {
		logger.Info("applying schema file", "file", filePath)

		content, err := schemaFS.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("read schema file %s: %w", filePath, err)
		}

		if _, err = tx.Exec(ctx, string(content)); err != nil {
			return fmt.Errorf("execute schema %s: %w", filePath, err)
		}
	}
	return nil
}

func initRiverSchema(ctx context.Context, logger *slog.Logger, pool *pgxpool.Pool) error {
	migrator, err := rivermigrate.New(riverpgxv5.New(pool), nil)
	if err != nil {
		return fmt.Errorf("failed to create river migrator: %w", err)
	}

	res, err := migrator.Migrate(ctx, rivermigrate.DirectionUp, &rivermigrate.MigrateOpts{})
	if err != nil {
		return fmt.Errorf("river migrate up failed: %w", err)
	}

	for _, v := range res.Versions {
		logger.Info(fmt.Sprintf("[river migration] applied migration: %d [up] %s", v.Version, v.Name))
	}

	return nil
}

func advisoryLockID(name string) int64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(name))
	return int64(h.Sum64())
}
