package store

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/calebdunn/ndc-loader/migrations"
)

// NewDB creates a new connection pool to PostgreSQL.
func NewDB(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parsing database URL: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("creating connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("pinging database: %w", err)
	}

	slog.Info("connected to database")
	return pool, nil
}

// RunMigrations applies all SQL migration files in order.
func RunMigrations(ctx context.Context, db *pgxpool.Pool) error {
	entries, err := migrations.FS.ReadDir(".")
	if err != nil {
		return fmt.Errorf("reading migrations directory: %w", err)
	}

	var filenames []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			filenames = append(filenames, e.Name())
		}
	}
	sort.Strings(filenames)

	for _, name := range filenames {
		data, err := migrations.FS.ReadFile(name)
		if err != nil {
			return fmt.Errorf("reading migration %s: %w", name, err)
		}

		slog.Info("applying migration", "file", name)
		if _, err := db.Exec(ctx, string(data)); err != nil {
			return fmt.Errorf("executing migration %s: %w", name, err)
		}
	}

	slog.Info("all migrations applied", "count", len(filenames))
	return nil
}
