package store

import "github.com/jackc/pgx/v5/pgxpool"

// TestDB wraps a database pool for use in integration tests.
type TestDB struct {
	Pool *pgxpool.Pool
}
