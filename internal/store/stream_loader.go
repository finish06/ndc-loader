package store

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// StreamLoader handles streaming bulk data loading with O(1) table swap.
type StreamLoader struct {
	db                    *pgxpool.Pool
	rowCountDropThreshold float64
}

// NewStreamLoader creates a new StreamLoader.
func NewStreamLoader(db *pgxpool.Pool, threshold float64) *StreamLoader {
	return &StreamLoader{db: db, rowCountDropThreshold: threshold}
}

// StreamLoad loads data into a table using a streaming approach:
// 1. Create staging table
// 2. Caller streams rows into staging via the returned CopyFrom channel
// 3. O(1) atomic swap: DROP live + RENAME staging
//
// Returns a RowWriter that the caller uses to send rows, and a Finish func to complete the swap.
func (l *StreamLoader) StreamLoad(ctx context.Context, tableName string, columns []string) (*RowWriter, error) {
	stagingTable := tableName + "_staging"

	// Drop any leftover staging table (outside transaction — DDL).
	_, _ = l.db.Exec(ctx, fmt.Sprintf("DROP TABLE IF EXISTS %s", pgx.Identifier{stagingTable}.Sanitize()))

	// Create staging table with same structure.
	_, err := l.db.Exec(ctx, fmt.Sprintf(
		"CREATE TABLE %s (LIKE %s INCLUDING DEFAULTS INCLUDING INDEXES)",
		pgx.Identifier{stagingTable}.Sanitize(),
		pgx.Identifier{tableName}.Sanitize(),
	))
	if err != nil {
		return nil, fmt.Errorf("creating staging table: %w", err)
	}

	return &RowWriter{
		db:           l.db,
		tableName:    tableName,
		stagingTable: stagingTable,
		columns:      columns,
		threshold:    l.rowCountDropThreshold,
		rows:         make([][]interface{}, 0, 1024),
	}, nil
}

// RowWriter collects rows and performs the bulk load + swap.
type RowWriter struct {
	db           *pgxpool.Pool
	tableName    string
	stagingTable string
	columns      []string
	threshold    float64
	rows         [][]interface{}
}

// AddRow adds a row to the batch.
func (w *RowWriter) AddRow(row []interface{}) {
	w.rows = append(w.rows, row)
}

// Finish performs the bulk COPY into staging, then atomically swaps tables.
// Returns the number of rows loaded.
func (w *RowWriter) Finish(ctx context.Context, previousRowCount int) (int, error) {
	// Row count safety check.
	if previousRowCount > 0 {
		dl := &DataLoader{rowCountDropThreshold: w.threshold}
		if err := dl.CheckRowCountSafety(previousRowCount, len(w.rows)); err != nil {
			// Clean up staging table.
			_, _ = w.db.Exec(ctx, fmt.Sprintf("DROP TABLE IF EXISTS %s", pgx.Identifier{w.stagingTable}.Sanitize()))
			return 0, err
		}
	}

	// COPY rows into staging table.
	copyCount, err := w.db.CopyFrom(
		ctx,
		pgx.Identifier{w.stagingTable},
		w.columns,
		pgx.CopyFromRows(w.rows),
	)
	if err != nil {
		_, _ = w.db.Exec(ctx, fmt.Sprintf("DROP TABLE IF EXISTS %s", pgx.Identifier{w.stagingTable}.Sanitize()))
		return 0, fmt.Errorf("copying data to staging: %w", err)
	}

	slog.Info("bulk copy complete", "table", w.tableName, "rows", copyCount)

	// Atomic swap via transaction: DROP live + RENAME staging.
	tx, err := w.db.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("beginning swap transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Drop the live table and rename staging.
	sanitizedLive := pgx.Identifier{w.tableName}.Sanitize()
	sanitizedStaging := pgx.Identifier{w.stagingTable}.Sanitize()

	if _, err := tx.Exec(ctx, fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", sanitizedLive)); err != nil {
		return 0, fmt.Errorf("dropping live table: %w", err)
	}

	if _, err := tx.Exec(ctx, fmt.Sprintf("ALTER TABLE %s RENAME TO %s",
		sanitizedStaging,
		pgx.Identifier{w.tableName}.Sanitize(),
	)); err != nil {
		return 0, fmt.Errorf("renaming staging to live: %w", err)
	}

	// Re-add the tsvector trigger for products table.
	if w.tableName == "products" {
		triggerSQL := `
			DROP TRIGGER IF EXISTS products_search_vector_trigger ON products;
			CREATE TRIGGER products_search_vector_trigger
				BEFORE INSERT OR UPDATE ON products
				FOR EACH ROW EXECUTE FUNCTION products_search_vector_update();
		`
		if _, err := tx.Exec(ctx, triggerSQL); err != nil {
			slog.Warn("failed to re-add search vector trigger", "error", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("committing swap: %w", err)
	}

	// Rebuild search vectors for products (trigger only fires on INSERT/UPDATE,
	// but our COPY bypasses triggers — need to update manually).
	if w.tableName == "products" {
		if _, err := w.db.Exec(ctx, `
			UPDATE products SET search_vector = to_tsvector('english',
				coalesce(proprietary_name, '') || ' ' ||
				coalesce(nonproprietary_name, '') || ' ' ||
				coalesce(labeler_name, '') || ' ' ||
				coalesce(substance_name, '')
			)
		`); err != nil {
			slog.Warn("failed to rebuild search vectors", "error", err)
		} else {
			slog.Info("rebuilt search vectors", "table", w.tableName)
		}
	}

	return int(copyCount), nil
}

// Abort cleans up the staging table without performing the swap.
func (w *RowWriter) Abort(ctx context.Context) {
	cleanupSQL := fmt.Sprintf("DROP TABLE IF EXISTS %s", pgx.Identifier{w.stagingTable}.Sanitize())
	_, _ = w.db.Exec(ctx, cleanupSQL)
}

// CheckRowCountSafety validates row count drop (delegated to DataLoader).
func (l *StreamLoader) CheckRowCountSafety(previousCount, newCount int) error {
	dl := &DataLoader{rowCountDropThreshold: l.rowCountDropThreshold}
	return dl.CheckRowCountSafety(previousCount, newCount)
}

// GetCurrentRowCount returns the current number of rows in a table.
func (l *StreamLoader) GetCurrentRowCount(ctx context.Context, tableName string) (int, error) {
	var count int
	err := l.db.QueryRow(ctx,
		fmt.Sprintf("SELECT COUNT(*) FROM %s", pgx.Identifier{tableName}.Sanitize()),
	).Scan(&count)
	if err != nil {
		// Table might not exist yet.
		if strings.Contains(err.Error(), "does not exist") {
			return 0, nil
		}
		return 0, err
	}
	return count, nil
}
