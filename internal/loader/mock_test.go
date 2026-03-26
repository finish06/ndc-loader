package loader

import (
	"context"
	"sync"

	"github.com/calebdunn/ndc-loader/internal/model"
	"github.com/calebdunn/ndc-loader/internal/store"
)

// mockCheckpointManager is a test double for CheckpointManager.
type mockCheckpointManager struct {
	mu              sync.Mutex
	checkpoints     []model.LoadCheckpoint
	previousCounts  map[string]int
	activeLoadID    string
	hasActive       bool
	hasActiveErr    error
	createErr       error
	updateStatusErr error
	setRowCountErr  error
	setPrevCountErr error
	setErrorErr     error
	getCheckErr     error
	getLoadedErr    error
	getPrevCountErr error

	// call tracking
	createCalls      []model.LoadCheckpoint
	updateCalls      []updateStatusCall
	setRowCountCalls []setRowCountCall
	setPrevCalls     []setPrevCountCall
	setErrorCalls    []setErrorCall
}

type updateStatusCall struct {
	loadID, tableName string
	status            model.LoadStatus
}

type setRowCountCall struct {
	loadID, tableName string
	count             int
}

type setPrevCountCall struct {
	loadID, tableName string
	count             int
}

type setErrorCall struct {
	loadID, tableName, errMsg string
}

func newMockCheckpointManager() *mockCheckpointManager {
	return &mockCheckpointManager{
		previousCounts: make(map[string]int),
	}
}

func (m *mockCheckpointManager) CreateCheckpoint(_ context.Context, cp *model.LoadCheckpoint) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.createCalls = append(m.createCalls, *cp)
	if m.createErr != nil {
		return m.createErr
	}
	m.checkpoints = append(m.checkpoints, *cp)
	return nil
}

func (m *mockCheckpointManager) UpdateStatus(_ context.Context, loadID, tableName string, status model.LoadStatus) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.updateCalls = append(m.updateCalls, updateStatusCall{loadID, tableName, status})
	if m.updateStatusErr != nil {
		return m.updateStatusErr
	}
	for i := range m.checkpoints {
		if m.checkpoints[i].LoadID == loadID && m.checkpoints[i].TableName == tableName {
			m.checkpoints[i].Status = status
		}
	}
	return nil
}

func (m *mockCheckpointManager) SetRowCount(_ context.Context, loadID, tableName string, count int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.setRowCountCalls = append(m.setRowCountCalls, setRowCountCall{loadID, tableName, count})
	return m.setRowCountErr
}

func (m *mockCheckpointManager) SetPreviousRowCount(_ context.Context, loadID, tableName string, count int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.setPrevCalls = append(m.setPrevCalls, setPrevCountCall{loadID, tableName, count})
	return m.setPrevCountErr
}

func (m *mockCheckpointManager) SetError(_ context.Context, loadID, tableName, errMsg string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.setErrorCalls = append(m.setErrorCalls, setErrorCall{loadID, tableName, errMsg})
	return m.setErrorErr
}

func (m *mockCheckpointManager) GetCheckpoints(_ context.Context, loadID string) ([]model.LoadCheckpoint, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.getCheckErr != nil {
		return nil, m.getCheckErr
	}
	var result []model.LoadCheckpoint
	for _, cp := range m.checkpoints {
		if cp.LoadID == loadID {
			result = append(result, cp)
		}
	}
	return result, nil
}

func (m *mockCheckpointManager) GetLoadedTables(_ context.Context, loadID string) (map[string]bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.getLoadedErr != nil {
		return nil, m.getLoadedErr
	}
	loaded := make(map[string]bool)
	for _, cp := range m.checkpoints {
		if cp.LoadID == loadID && cp.Status == model.LoadStatusLoaded {
			loaded[cp.TableName] = true
		}
	}
	return loaded, nil
}

func (m *mockCheckpointManager) HasActiveLoad(_ context.Context) (string, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.activeLoadID, m.hasActive, m.hasActiveErr
}

func (m *mockCheckpointManager) GetPreviousRowCount(_ context.Context, tableName string) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.getPrevCountErr != nil {
		return 0, m.getPrevCountErr
	}
	return m.previousCounts[tableName], nil
}

// mockBulkLoader is a test double for BulkLoader.
type mockBulkLoader struct {
	mu            sync.Mutex
	bulkLoadErr   error
	safetyErr     error
	loadResults   map[string]*store.LoadResult
	bulkLoadCalls []bulkLoadCall
	safetyCalls   []safetyCall
}

type bulkLoadCall struct {
	tableName string
	columns   []string
	rowCount  int
}

type safetyCall struct {
	previousCount, newCount int
}

func newMockBulkLoader() *mockBulkLoader {
	return &mockBulkLoader{
		loadResults: make(map[string]*store.LoadResult),
	}
}

func (m *mockBulkLoader) BulkLoad(_ context.Context, tableName string, columns []string, rows [][]interface{}) (*store.LoadResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.bulkLoadCalls = append(m.bulkLoadCalls, bulkLoadCall{tableName, columns, len(rows)})
	if m.bulkLoadErr != nil {
		return nil, m.bulkLoadErr
	}
	if result, ok := m.loadResults[tableName]; ok {
		return result, nil
	}
	return &store.LoadResult{Table: tableName, RowCount: len(rows)}, nil
}

func (m *mockBulkLoader) CheckRowCountSafety(previousCount, newCount int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.safetyCalls = append(m.safetyCalls, safetyCall{previousCount, newCount})
	return m.safetyErr
}
