package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthHandler_NoDB(t *testing.T) {
	handler := newHealthHandler(nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	var body HealthResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse body: %v", err)
	}

	// No DB = error status.
	if body.Status != "error" {
		t.Errorf("expected status 'error' with nil db, got %v", body.Status)
	}
	if body.Version == "" {
		t.Error("expected non-empty version")
	}
	if body.Uptime == "" {
		t.Error("expected non-empty uptime")
	}
	if len(body.Dependencies) != 1 {
		t.Fatalf("expected 1 dependency, got %d", len(body.Dependencies))
	}
	if body.Dependencies[0].Name != "postgres" {
		t.Errorf("expected postgres dependency, got %s", body.Dependencies[0].Name)
	}
	if body.Dependencies[0].Status != "disconnected" {
		t.Errorf("expected disconnected, got %s", body.Dependencies[0].Status)
	}
}

func TestVersionHandler(t *testing.T) {
	handler := versionHandler()

	req := httptest.NewRequest(http.MethodGet, "/version", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse body: %v", err)
	}

	if body["go_version"] == "" {
		t.Error("expected non-empty go_version")
	}
	if body["os"] == "" {
		t.Error("expected non-empty os")
	}
	if body["arch"] == "" {
		t.Error("expected non-empty arch")
	}
}

func TestNewRouter_HealthNoAuth(t *testing.T) {
	router := NewRouter(nil, []string{"secret-key"}, nil, nil, nil, nil)

	// Health endpoint should work without API key.
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for /health without auth, got %d", rec.Code)
	}
}

func TestNewRouter_AdminRequiresAuth(t *testing.T) {
	router := NewRouter(nil, []string{"secret-key"}, nil, nil, nil, nil)

	// Admin endpoint should require API key.
	req := httptest.NewRequest(http.MethodPost, "/api/admin/load", bytes.NewReader([]byte("{}")))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for /api/admin/load without auth, got %d", rec.Code)
	}
}

func TestNewRouter_AdminWithAuth(t *testing.T) {
	mockCS := &mockCheckpointStoreProvider{
		mockCheckpointQuerier:    newMockCheckpointQuerier(),
		mockLastLoadInfoProvider: &mockLastLoadInfoProvider{},
	}
	router := NewRouter(nil, []string{"secret-key"}, nil, mockCS, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/load/test-load-id", nil)
	req.Header.Set("X-API-Key", "secret-key")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	// Should get 404 (no checkpoints found) not 401.
	if rec.Code == http.StatusUnauthorized {
		t.Error("expected to pass auth with valid key, got 401")
	}
}

func TestTriggerLoadRequest_ParsesJSON(t *testing.T) {
	body := `{"datasets": ["ndc_directory"], "force": true}`
	req := httptest.NewRequest(http.MethodPost, "/api/admin/load", bytes.NewReader([]byte(body)))

	var tlr TriggerLoadRequest
	json.NewDecoder(req.Body).Decode(&tlr)

	if len(tlr.Datasets) != 1 {
		t.Errorf("expected 1 dataset, got %d", len(tlr.Datasets))
	}
	if tlr.Datasets[0] != "ndc_directory" {
		t.Errorf("expected ndc_directory, got %s", tlr.Datasets[0])
	}
	if !tlr.Force {
		t.Error("expected force=true")
	}
}
