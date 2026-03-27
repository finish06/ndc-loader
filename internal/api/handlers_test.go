package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthHandler(t *testing.T) {
	handler := healthHandler(nil)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse body: %v", err)
	}

	if body["status"] != "ok" {
		t.Errorf("expected status 'ok', got %v", body["status"])
	}
	if body["db"] != "connected" {
		t.Errorf("expected db 'connected', got %v", body["db"])
	}
}

func TestNewRouter_HealthNoAuth(t *testing.T) {
	router := NewRouter(nil, []string{"secret-key"}, nil, nil, nil)

	// Health endpoint should work without API key.
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for /health without auth, got %d", rec.Code)
	}
}

func TestNewRouter_AdminRequiresAuth(t *testing.T) {
	router := NewRouter(nil, []string{"secret-key"}, nil, nil, nil)

	// Admin endpoint should require API key.
	req := httptest.NewRequest(http.MethodPost, "/api/admin/load", bytes.NewReader([]byte("{}")))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for /api/admin/load without auth, got %d", rec.Code)
	}
}

func TestNewRouter_AdminWithAuth(t *testing.T) {
	// Create a minimal router without orchestrator — it will panic on nil deref,
	// but we can test that auth passes by checking we get past 401.
	router := NewRouter(nil, []string{"secret-key"}, nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/load/test-load-id", nil)
	req.Header.Set("X-API-Key", "secret-key")
	rec := httptest.NewRecorder()

	// This will hit the handler which needs checkpointStore — it will panic.
	// We use recover to check that auth passed.
	func() {
		defer func() {
			if r := recover(); r != nil {
				// Expected — handler panics because checkpointStore is nil.
				// But we got past auth, which is what we're testing.
			}
		}()
		router.ServeHTTP(rec, req)
	}()

	// If we got a 401, auth failed. Anything else means auth passed.
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
