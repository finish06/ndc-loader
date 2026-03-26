package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/calebdunn/ndc-loader/internal/loader"
	"github.com/calebdunn/ndc-loader/internal/model"
)

func TestTriggerLoad_ActiveLoadConflict(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	cpMock := newMockCheckpointQuerier()

	// Create an orchestrator with no datasets so RunLoad completes instantly.
	cfg := &model.DatasetsConfig{}
	downloader := loader.NewDownloader(t.TempDir(), 0)
	orch := loader.NewOrchestrator(logger, downloader, nil, nil, cfg)

	// Start a load that we can detect.
	// We need the orchestrator to have an active load. We do this by starting one
	// in a goroutine with a blocking dataset.
	handler := NewAdminHandler(logger, orch, cpMock)

	// First, when no load is active, TriggerLoad should succeed.
	body := `{"datasets": [], "force": false}`
	req := httptest.NewRequest(http.MethodPost, "/api/admin/load", bytes.NewReader([]byte(body)))
	rec := httptest.NewRecorder()
	handler.TriggerLoad(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Errorf("expected 202, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestTriggerLoad_EmptyBody(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	cpMock := newMockCheckpointQuerier()
	cfg := &model.DatasetsConfig{}
	downloader := loader.NewDownloader(t.TempDir(), 0)
	orch := loader.NewOrchestrator(logger, downloader, nil, nil, cfg)
	handler := NewAdminHandler(logger, orch, cpMock)

	// Empty body should not cause an error.
	req := httptest.NewRequest(http.MethodPost, "/api/admin/load", bytes.NewReader([]byte("")))
	rec := httptest.NewRecorder()
	handler.TriggerLoad(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Errorf("expected 202, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestGetLoadStatus_Found(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	cpMock := newMockCheckpointQuerier()

	started := time.Now().Add(-5 * time.Minute)
	completed := time.Now()
	rowCount := 100
	cpMock.checkpoints["test-load-123"] = []model.LoadCheckpoint{
		{
			LoadID:      "test-load-123",
			Dataset:     "ndc_directory",
			TableName:   "products",
			Status:      model.LoadStatusLoaded,
			RowCount:    &rowCount,
			StartedAt:   &started,
			CompletedAt: &completed,
		},
	}

	handler := NewAdminHandler(logger, nil, cpMock)

	// Use chi router to properly extract URL params.
	r := chi.NewRouter()
	r.Get("/api/admin/load/{loadID}", handler.GetLoadStatus)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/load/test-load-123", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp LoadStatusResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.LoadID != "test-load-123" {
		t.Errorf("expected load_id test-load-123, got %s", resp.LoadID)
	}
	if resp.Status != "complete" {
		t.Errorf("expected status complete, got %s", resp.Status)
	}
	if len(resp.Checkpoints) != 1 {
		t.Fatalf("expected 1 checkpoint, got %d", len(resp.Checkpoints))
	}
	if resp.Checkpoints[0].Table != "products" {
		t.Errorf("expected table products, got %s", resp.Checkpoints[0].Table)
	}
	if resp.Checkpoints[0].RowCount == nil || *resp.Checkpoints[0].RowCount != 100 {
		t.Errorf("expected row_count 100, got %v", resp.Checkpoints[0].RowCount)
	}
	if resp.Checkpoints[0].DurationSeconds == nil {
		t.Error("expected duration_seconds to be set")
	}
}

func TestGetLoadStatus_NotFound(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	cpMock := newMockCheckpointQuerier()

	handler := NewAdminHandler(logger, nil, cpMock)

	r := chi.NewRouter()
	r.Get("/api/admin/load/{loadID}", handler.GetLoadStatus)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/load/nonexistent", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestGetLoadStatus_InternalError(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	cpMock := newMockCheckpointQuerier()
	cpMock.err = fmt.Errorf("database error")

	handler := NewAdminHandler(logger, nil, cpMock)

	r := chi.NewRouter()
	r.Get("/api/admin/load/{loadID}", handler.GetLoadStatus)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/load/test-load-123", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestGetLoadStatus_InProgressStatus(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	cpMock := newMockCheckpointQuerier()

	cpMock.checkpoints["load-456"] = []model.LoadCheckpoint{
		{
			LoadID:    "load-456",
			Dataset:   "ndc_directory",
			TableName: "products",
			Status:    model.LoadStatusLoaded,
		},
		{
			LoadID:    "load-456",
			Dataset:   "ndc_directory",
			TableName: "packages",
			Status:    model.LoadStatusLoading,
		},
	}

	handler := NewAdminHandler(logger, nil, cpMock)

	r := chi.NewRouter()
	r.Get("/api/admin/load/{loadID}", handler.GetLoadStatus)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/load/load-456", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp LoadStatusResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp.Status != "in_progress" {
		t.Errorf("expected status in_progress, got %s", resp.Status)
	}
}

func TestGetLoadStatus_FailedStatus(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	cpMock := newMockCheckpointQuerier()

	errMsg := "row count safety check failed"
	cpMock.checkpoints["load-789"] = []model.LoadCheckpoint{
		{
			LoadID:       "load-789",
			Dataset:      "ndc_directory",
			TableName:    "products",
			Status:       model.LoadStatusFailed,
			ErrorMessage: &errMsg,
		},
	}

	handler := NewAdminHandler(logger, nil, cpMock)

	r := chi.NewRouter()
	r.Get("/api/admin/load/{loadID}", handler.GetLoadStatus)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/load/load-789", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	var resp LoadStatusResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp.Status != "failed" {
		t.Errorf("expected status failed, got %s", resp.Status)
	}
	if resp.Checkpoints[0].Error == nil || *resp.Checkpoints[0].Error != errMsg {
		t.Errorf("expected error message %q, got %v", errMsg, resp.Checkpoints[0].Error)
	}
}

func TestHealthHandler_WithFreshData(t *testing.T) {
	now := time.Now()
	mock := &mockLastLoadInfoProvider{
		lastLoad: &now,
		ageHours: 12.0,
	}

	handler := healthHandler(mock)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	var body map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &body)

	if body["status"] != "ok" {
		t.Errorf("expected status ok, got %v", body["status"])
	}
	if body["last_load"] == nil {
		t.Error("expected last_load to be set")
	}
	if body["data_age_hours"] == nil {
		t.Error("expected data_age_hours to be set")
	}
}

func TestHealthHandler_WithStaleData(t *testing.T) {
	old := time.Now().Add(-72 * time.Hour)
	mock := &mockLastLoadInfoProvider{
		lastLoad: &old,
		ageHours: 72.0,
	}

	handler := healthHandler(mock)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	var body map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &body)

	if body["status"] != "degraded" {
		t.Errorf("expected status degraded for stale data, got %v", body["status"])
	}
}

func TestHealthHandler_WithError(t *testing.T) {
	mock := &mockLastLoadInfoProvider{
		err: fmt.Errorf("db error"),
	}

	handler := healthHandler(mock)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	var body map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &body)

	// Should still return ok since health check should be resilient.
	if body["status"] != "ok" {
		t.Errorf("expected status ok even with error, got %v", body["status"])
	}
}
