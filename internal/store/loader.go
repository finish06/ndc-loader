package store

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DataLoader handles bulk data loading into PostgreSQL tables.
type DataLoader struct {
	db                    *pgxpool.Pool
	rowCountDropThreshold float64
}

// NewDataLoader creates a new DataLoader.
func NewDataLoader(db *pgxpool.Pool, threshold float64) *DataLoader {
	return &DataLoader{db: db, rowCountDropThreshold: threshold}
}

// TableColumns defines the column mapping for a table.
type TableColumns struct {
	Table   string
	Columns []string
}

// knownTables maps table names to their column definitions for COPY operations.
var knownTables = map[string][]string{
	"products": {
		"product_id", "product_ndc", "product_type", "proprietary_name",
		"proprietary_name_suffix", "nonproprietary_name", "dosage_form", "route",
		"labeler_name", "substance_name", "strength", "strength_unit",
		"pharm_classes", "dea_schedule", "marketing_category", "application_number",
		"marketing_start", "marketing_end", "ndc_exclude", "listing_certified",
	},
	"packages": {
		"product_id", "product_ndc", "ndc_package_code", "description",
		"marketing_start", "marketing_end", "ndc_exclude", "sample_package",
	},
	"applications": {
		"appl_no", "appl_type", "appl_public_notes", "sponsor_name",
	},
	"drugsfda_products": {
		"appl_no", "product_no", "form", "strength",
		"reference_drug", "drug_name", "active_ingredient", "reference_standard",
	},
	"submissions": {
		"appl_no", "submission_class_code_id", "submission_type", "submission_no",
		"submission_status", "submission_status_date", "submissions_public_notes",
		"review_priority",
	},
	"marketing_status": {
		"marketing_status_id", "appl_no", "product_no",
	},
	"te_codes": {
		"appl_no", "product_no", "marketing_status_id", "te_code",
	},
}

// GetTableColumns returns the known columns for a table.
func GetTableColumns(tableName string) ([]string, error) {
	cols, ok := knownTables[tableName]
	if !ok {
		return nil, fmt.Errorf("unknown table: %s", tableName)
	}
	return cols, nil
}

// LoadResult contains the result of a bulk load operation.
type LoadResult struct {
	Table    string
	RowCount int
}

// BulkLoad loads rows into a table using atomic swap (staging table approach).
// It creates a staging table, copies data in, then swaps it with the live table
// inside a transaction so consumers never see partial data.
func (l *DataLoader) BulkLoad(ctx context.Context, tableName string, columns []string, rows [][]interface{}) (*LoadResult, error) {
	stagingTable := tableName + "_staging"

	tx, err := l.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("beginning transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	// Create staging table with same structure as live table.
	if _, err := tx.Exec(ctx, fmt.Sprintf("DROP TABLE IF EXISTS %s", pgx.Identifier{stagingTable}.Sanitize())); err != nil {
		return nil, fmt.Errorf("dropping old staging table: %w", err)
	}
	if _, err := tx.Exec(ctx, fmt.Sprintf(
		"CREATE TABLE %s (LIKE %s INCLUDING ALL)",
		pgx.Identifier{stagingTable}.Sanitize(),
		pgx.Identifier{tableName}.Sanitize(),
	)); err != nil {
		return nil, fmt.Errorf("creating staging table: %w", err)
	}

	// Drop FK constraints on staging table (they reference live table).
	// We'll rely on the live table's constraints after swap.
	// For tables with serial PKs, reset the sequence.

	// Bulk copy into staging table.
	copyCount, err := tx.CopyFrom(
		ctx,
		pgx.Identifier{stagingTable},
		columns,
		pgx.CopyFromRows(rows),
	)
	if err != nil {
		return nil, fmt.Errorf("copying data to staging table: %w", err)
	}

	slog.Info("bulk copy complete", "table", tableName, "rows", copyCount)

	// Swap: truncate live, insert from staging, drop staging.
	if _, err := tx.Exec(ctx, fmt.Sprintf("TRUNCATE %s CASCADE", pgx.Identifier{tableName}.Sanitize())); err != nil {
		return nil, fmt.Errorf("truncating live table: %w", err)
	}

	insertCols := make([]string, len(columns))
	for i, c := range columns {
		insertCols[i] = pgx.Identifier{c}.Sanitize()
	}
	colList := strings.Join(insertCols, ", ")
	insertSQL := fmt.Sprintf(
		"INSERT INTO %s (%s) SELECT %s FROM %s",
		pgx.Identifier{tableName}.Sanitize(),
		colList, colList,
		pgx.Identifier{stagingTable}.Sanitize(),
	)
	if _, err := tx.Exec(ctx, insertSQL); err != nil {
		return nil, fmt.Errorf("inserting from staging to live: %w", err)
	}

	if _, err := tx.Exec(ctx, fmt.Sprintf("DROP TABLE %s", pgx.Identifier{stagingTable}.Sanitize())); err != nil {
		return nil, fmt.Errorf("dropping staging table: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("committing transaction: %w", err)
	}

	return &LoadResult{Table: tableName, RowCount: int(copyCount)}, nil
}

// GetCurrentRowCount returns the current number of rows in a table.
func (l *DataLoader) GetCurrentRowCount(ctx context.Context, tableName string) (int, error) {
	var count int
	err := l.db.QueryRow(ctx, fmt.Sprintf("SELECT COUNT(*) FROM %s", pgx.Identifier{tableName}.Sanitize())).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("counting rows in %s: %w", tableName, err)
	}
	return count, nil
}

// CheckRowCountSafety validates that the new row count hasn't dropped more than the threshold.
func (l *DataLoader) CheckRowCountSafety(previousCount, newCount int) error {
	if previousCount == 0 {
		return nil // No previous data, any count is fine.
	}
	drop := 1.0 - float64(newCount)/float64(previousCount)
	if drop > l.rowCountDropThreshold {
		return fmt.Errorf(
			"row count dropped %.1f%% (from %d to %d), exceeds threshold of %.0f%%",
			drop*100, previousCount, newCount, l.rowCountDropThreshold*100,
		)
	}
	return nil
}
