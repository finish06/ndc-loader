package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/calebdunn/ndc-loader/internal/store"
)

// Issue #7 (RED): a disconnected postgres yields status "error" in the body,
// but the handler still returned HTTP 200 — so an httpGet readiness probe (and
// any status-code-based monitor) treated the broken pod as healthy.
func TestHealthHandler_PostgresDisconnected_ReturnsNon200(t *testing.T) {
	// nil pool => checkPostgres reports "disconnected" => overall status "error".
	handler := newHealthHandler(nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	var body HealthResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode health response: %v", err)
	}
	if body.Status != "error" {
		t.Fatalf("precondition: expected body status \"error\", got %q", body.Status)
	}

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected HTTP 503 when postgres disconnected, got %d", rec.Code)
	}
}

func TestCheckPostgres_NilPool(t *testing.T) {
	check := checkPostgres(context.Background(), nil)

	if check.Name != "postgres" {
		t.Errorf("expected name postgres, got %s", check.Name)
	}
	if check.Status != "disconnected" {
		t.Errorf("expected disconnected, got %s", check.Status)
	}
	if check.Error == nil {
		t.Error("expected error message for nil pool")
	}
	if check.LatencyMs != nil {
		t.Error("expected nil latency for nil pool")
	}
}

func TestCheckPostgres_NilPool_ErrorMessage(t *testing.T) {
	check := checkPostgres(context.Background(), nil)

	if *check.Error != "database pool not configured" {
		t.Errorf("expected 'database pool not configured', got %s", *check.Error)
	}
}

func TestEnrichProduct_NilProduct(t *testing.T) {
	// Should not panic.
	enrichProduct(nil)
}

func TestEnrichProduct_NilPharmClasses(t *testing.T) {
	p := &store.ProductResult{ProductNDC: "test"}
	enrichProduct(p)

	if p.PharmClassesStructured != nil {
		t.Error("expected nil pharm_classes_structured for nil pharm_classes")
	}
}

func TestEnrichProduct_EmptyPharmClasses(t *testing.T) {
	empty := ""
	p := &store.ProductResult{ProductNDC: "test", PharmClasses: &empty}
	enrichProduct(p)

	if p.PharmClassesStructured != nil {
		t.Error("expected nil pharm_classes_structured for empty string")
	}
}

func TestEnrichProduct_WithPharmClasses(t *testing.T) {
	classes := "Biguanide [EPC]"
	p := &store.ProductResult{ProductNDC: "test", PharmClasses: &classes}
	enrichProduct(p)

	if p.PharmClassesStructured == nil {
		t.Error("expected pharm_classes_structured to be set")
	}
}
