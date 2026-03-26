package loader

import (
	"context"

	"github.com/calebdunn/ndc-loader/internal/model"
	"github.com/calebdunn/ndc-loader/internal/store"
)

// CheckpointManager abstracts checkpoint storage operations used by the orchestrator.
type CheckpointManager interface {
	CreateCheckpoint(ctx context.Context, cp *model.LoadCheckpoint) error
	UpdateStatus(ctx context.Context, loadID, tableName string, status model.LoadStatus) error
	SetRowCount(ctx context.Context, loadID, tableName string, count int) error
	SetPreviousRowCount(ctx context.Context, loadID, tableName string, count int) error
	SetError(ctx context.Context, loadID, tableName, errMsg string) error
	GetCheckpoints(ctx context.Context, loadID string) ([]model.LoadCheckpoint, error)
	GetLoadedTables(ctx context.Context, loadID string) (map[string]bool, error)
	HasActiveLoad(ctx context.Context) (string, bool, error)
	GetPreviousRowCount(ctx context.Context, tableName string) (int, error)
}

// BulkLoader abstracts bulk data loading operations used by the orchestrator.
type BulkLoader interface {
	BulkLoad(ctx context.Context, tableName string, columns []string, rows [][]interface{}) (*store.LoadResult, error)
	CheckRowCountSafety(previousCount, newCount int) error
}
