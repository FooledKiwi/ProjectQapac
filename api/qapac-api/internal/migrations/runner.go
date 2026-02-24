// Package migrations provides helpers to apply SQL migrations at startup.
//
// SQL files are embedded at compile time so the binary is self-contained.
// Migrations are tracked in the schema_migrations table, making Run idempotent:
// files already recorded there are skipped on subsequent calls.
//
// File naming convention: NNN_description.sql (lexicographic execution order).
// The 000_migrations_table.sql file must always be first so the tracking table
// exists before any other migration runs.
package migrations

import (
	"context"
	"embed"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
)

// sqlFiles embeds all *.sql files in this directory at compile time.
//
//go:embed *.sql
var sqlFiles embed.FS

// entry holds the filename and raw SQL content of a single migration file.
type entry struct {
	version string // filename used as the unique version key
	sql     string
}

// Run applies all pending migrations to the database in lexicographic order.
//
// It first ensures schema_migrations exists (idempotent), then checks which
// versions have already been recorded and skips them. Each pending migration
// runs in its own transaction; on success its version is inserted into
// schema_migrations within the same transaction.
func Run(ctx context.Context, pool *pgxpool.Pool) error {
	if err := ensureMigrationsTable(ctx, pool); err != nil {
		return fmt.Errorf("migrations: ensure tracking table: %w", err)
	}

	entries, err := loadEntries()
	if err != nil {
		return fmt.Errorf("migrations: load files: %w", err)
	}

	applied, err := appliedVersions(ctx, pool)
	if err != nil {
		return fmt.Errorf("migrations: read applied versions: %w", err)
	}

	pending := 0
	for _, e := range entries {
		if applied[e.version] {
			log.Printf("migrations: skipping %q (already applied)", e.version)
			continue
		}
		if err := applyEntry(ctx, pool, e); err != nil {
			return fmt.Errorf("migrations: apply %q: %w", e.version, err)
		}
		pending++
	}

	if pending == 0 {
		log.Println("migrations: schema is up to date")
	} else {
		log.Printf("migrations: %d migration(s) applied", pending)
	}

	return nil
}

// CheckSchema verifies that the expected business tables exist in the public
// schema. It is a lightweight sanity check – not a full structural diff.
func CheckSchema(ctx context.Context, pool *pgxpool.Pool) error {
	required := []string{
		"stops",
		"routes",
		"route_stops",
		"route_shapes",
		"stop_eta_cache",
		"route_to_stop_cache",
		"users",
		"refresh_tokens",
		"vehicles",
		"vehicle_assignments",
		"vehicle_positions",
		"trips",
		"alerts",
		"ratings",
		"favorites",
	}

	for _, table := range required {
		var exists bool
		err := pool.QueryRow(ctx,
			`SELECT EXISTS (
                SELECT 1
                FROM information_schema.tables
                WHERE table_schema = 'public'
                  AND table_name   = $1
            )`,
			table,
		).Scan(&exists)
		if err != nil {
			return fmt.Errorf("migrations: check table %q: %w", table, err)
		}
		if !exists {
			return fmt.Errorf("migrations: required table %q is missing", table)
		}
	}

	return nil
}

// ensureMigrationsTable creates schema_migrations if it does not exist.
// This is safe to call multiple times.
func ensureMigrationsTable(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `
        CREATE TABLE IF NOT EXISTS schema_migrations (
            version    VARCHAR(255) PRIMARY KEY,
            applied_at TIMESTAMP DEFAULT NOW()
        )`)
	return err
}

// appliedVersions returns the set of migration filenames already recorded in
// schema_migrations.
func appliedVersions(ctx context.Context, pool *pgxpool.Pool) (map[string]bool, error) {
	rows, err := pool.Query(ctx, `SELECT version FROM schema_migrations`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	seen := make(map[string]bool)
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		seen[v] = true
	}

	return seen, rows.Err()
}

// loadEntries reads the embedded SQL files and returns them in lexicographic
// order. embed.FS.ReadDir guarantees this ordering.
func loadEntries() ([]entry, error) {
	dirEntries, err := sqlFiles.ReadDir(".")
	if err != nil {
		return nil, fmt.Errorf("read embedded dir: %w", err)
	}

	var out []entry
	for _, de := range dirEntries {
		if de.IsDir() {
			continue
		}
		content, err := sqlFiles.ReadFile(de.Name())
		if err != nil {
			return nil, fmt.Errorf("read %q: %w", de.Name(), err)
		}
		out = append(out, entry{version: de.Name(), sql: string(content)})
	}

	return out, nil
}

// applyEntry executes a single migration and records it in schema_migrations,
// both inside a single transaction so they are atomic.
func applyEntry(ctx context.Context, pool *pgxpool.Pool, e entry) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, e.sql); err != nil {
		return fmt.Errorf("exec sql: %w", err)
	}

	if _, err := tx.Exec(ctx,
		`INSERT INTO schema_migrations (version) VALUES ($1)`,
		e.version,
	); err != nil {
		return fmt.Errorf("record version: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	log.Printf("migrations: applied %q", e.version)
	return nil
}
