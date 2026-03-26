package store

import (
	"context"
	"fmt"
	"time"

	"github.com/calebdunn/ndc-loader/internal/model"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CheckpointStore manages load checkpoint records.
type CheckpointStore struct {
	db *pgxpool.Pool
}

// NewCheckpointStore creates a new CheckpointStore.
func NewCheckpointStore(db *pgxpool.Pool) *CheckpointStore {
	return &CheckpointStore{db: db}
}

// CreateCheckpoint inserts a new checkpoint record.
func (s *CheckpointStore) CreateCheckpoint(ctx context.Context, cp *model.LoadCheckpoint) error {
	_, err := s.db.Exec(ctx,
		`INSERT INTO load_checkpoints (load_id, dataset, table_name, status, created_at)
		 VALUES ($1, $2, $3, $4, $5)`,
		cp.LoadID, cp.Dataset, cp.TableName, cp.Status, time.Now(),
	)
	if err != nil {
		return fmt.Errorf("creating checkpoint: %w", err)
	}
	return nil
}

// UpdateStatus updates the status of a checkpoint.
func (s *CheckpointStore) UpdateStatus(ctx context.Context, loadID, tableName string, status model.LoadStatus) error {
	now := time.Now()
	var query string
	switch status {
	case model.LoadStatusDownloading, model.LoadStatusLoading:
		query = `UPDATE load_checkpoints SET status = $1, started_at = $2 WHERE load_id = $3 AND table_name = $4`
		_, err := s.db.Exec(ctx, query, status, now, loadID, tableName)
		return err
	case model.LoadStatusLoaded, model.LoadStatusDownloaded:
		query = `UPDATE load_checkpoints SET status = $1, completed_at = $2 WHERE load_id = $3 AND table_name = $4`
		_, err := s.db.Exec(ctx, query, status, now, loadID, tableName)
		return err
	default:
		query = `UPDATE load_checkpoints SET status = $1 WHERE load_id = $2 AND table_name = $3`
		_, err := s.db.Exec(ctx, query, status, loadID, tableName)
		return err
	}
}

// SetRowCount sets the row count for a checkpoint.
func (s *CheckpointStore) SetRowCount(ctx context.Context, loadID, tableName string, count int) error {
	_, err := s.db.Exec(ctx,
		`UPDATE load_checkpoints SET row_count = $1 WHERE load_id = $2 AND table_name = $3`,
		count, loadID, tableName,
	)
	return err
}

// SetPreviousRowCount sets the previous row count for comparison.
func (s *CheckpointStore) SetPreviousRowCount(ctx context.Context, loadID, tableName string, count int) error {
	_, err := s.db.Exec(ctx,
		`UPDATE load_checkpoints SET previous_row_count = $1 WHERE load_id = $2 AND table_name = $3`,
		count, loadID, tableName,
	)
	return err
}

// SetError records an error on a checkpoint.
func (s *CheckpointStore) SetError(ctx context.Context, loadID, tableName string, errMsg string) error {
	_, err := s.db.Exec(ctx,
		`UPDATE load_checkpoints SET status = $1, error_message = $2, completed_at = $3
		 WHERE load_id = $4 AND table_name = $5`,
		model.LoadStatusFailed, errMsg, time.Now(), loadID, tableName,
	)
	return err
}

// GetCheckpoints returns all checkpoints for a given load ID.
func (s *CheckpointStore) GetCheckpoints(ctx context.Context, loadID string) ([]model.LoadCheckpoint, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, load_id, dataset, table_name, status, row_count, previous_row_count,
		        error_message, started_at, completed_at, created_at
		 FROM load_checkpoints
		 WHERE load_id = $1
		 ORDER BY id`,
		loadID,
	)
	if err != nil {
		return nil, fmt.Errorf("querying checkpoints: %w", err)
	}
	defer rows.Close()

	var checkpoints []model.LoadCheckpoint
	for rows.Next() {
		var cp model.LoadCheckpoint
		if err := rows.Scan(
			&cp.ID, &cp.LoadID, &cp.Dataset, &cp.TableName, &cp.Status,
			&cp.RowCount, &cp.PreviousRowCount, &cp.ErrorMessage,
			&cp.StartedAt, &cp.CompletedAt, &cp.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning checkpoint: %w", err)
		}
		checkpoints = append(checkpoints, cp)
	}

	return checkpoints, rows.Err()
}

// GetLoadedTables returns table names that are already loaded for a given load ID.
func (s *CheckpointStore) GetLoadedTables(ctx context.Context, loadID string) (map[string]bool, error) {
	checkpoints, err := s.GetCheckpoints(ctx, loadID)
	if err != nil {
		return nil, err
	}

	loaded := make(map[string]bool)
	for _, cp := range checkpoints {
		if cp.Status == model.LoadStatusLoaded {
			loaded[cp.TableName] = true
		}
	}
	return loaded, nil
}

// HasActiveLoad checks if there's a load currently in progress.
func (s *CheckpointStore) HasActiveLoad(ctx context.Context) (string, bool, error) {
	var loadID string
	err := s.db.QueryRow(ctx,
		`SELECT load_id FROM load_checkpoints
		 WHERE status IN ($1, $2, $3)
		 ORDER BY created_at DESC LIMIT 1`,
		model.LoadStatusPending, model.LoadStatusDownloading, model.LoadStatusLoading,
	).Scan(&loadID)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return "", false, nil
		}
		return "", false, err
	}
	return loadID, true, nil
}

// GetPreviousRowCount gets the row count from the last successful load for a table.
func (s *CheckpointStore) GetPreviousRowCount(ctx context.Context, tableName string) (int, error) {
	var count int
	err := s.db.QueryRow(ctx,
		`SELECT row_count FROM load_checkpoints
		 WHERE table_name = $1 AND status = $2 AND row_count IS NOT NULL
		 ORDER BY completed_at DESC LIMIT 1`,
		tableName, model.LoadStatusLoaded,
	).Scan(&count)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return 0, nil
		}
		return 0, err
	}
	return count, nil
}
