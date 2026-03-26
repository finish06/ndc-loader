package api

import (
	"context"
	"time"

	"github.com/calebdunn/ndc-loader/internal/model"
)

// mockCheckpointQuerier is a test double for CheckpointQuerier.
type mockCheckpointQuerier struct {
	checkpoints map[string][]model.LoadCheckpoint
	err         error
}

func newMockCheckpointQuerier() *mockCheckpointQuerier {
	return &mockCheckpointQuerier{
		checkpoints: make(map[string][]model.LoadCheckpoint),
	}
}

func (m *mockCheckpointQuerier) GetCheckpoints(_ context.Context, loadID string) ([]model.LoadCheckpoint, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.checkpoints[loadID], nil
}

// mockLastLoadInfoProvider is a test double for LastLoadInfoProvider.
type mockLastLoadInfoProvider struct {
	lastLoad *time.Time
	ageHours float64
	err      error
}

func (m *mockLastLoadInfoProvider) GetLastLoadInfo(_ context.Context) (*time.Time, float64, error) {
	return m.lastLoad, m.ageHours, m.err
}

// mockCheckpointStoreProvider combines both interfaces for NewRouter.
type mockCheckpointStoreProvider struct {
	*mockCheckpointQuerier
	*mockLastLoadInfoProvider
}

// Compile-time interface check.
var _ CheckpointStoreProvider = (*mockCheckpointStoreProvider)(nil)
