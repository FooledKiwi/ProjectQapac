package storage

import (
	"context"

	"github.com/FooledKiwi/ProjectQapac/api/qapac-api/internal/migrations"
	"github.com/jackc/pgx/v5/pgxpool"
)

// RunMigrations applies all pending SQL migrations and verifies the schema.
// It delegates to the migrations package, which tracks applied versions in the
// schema_migrations table to guarantee idempotence across multiple startups.
func RunMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	if err := migrations.Run(ctx, pool); err != nil {
		return err
	}

	return migrations.CheckSchema(ctx, pool)
}
