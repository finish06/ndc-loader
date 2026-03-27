package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/calebdunn/ndc-loader/internal/store"
)

func setupQueryTestRouter(mock *mockQueryProvider) http.Handler {
	r := chi.NewRouter()
	h := NewQueryHandler(slog.Default(), mock)
	r.Get("/api/ndc/search", h.SearchNDC)
	r.Get("/api/ndc/stats", h.GetStats)
	r.Get("/api/ndc/{ndc}/packages", h.ListPackages)
	r.Get("/api/ndc/{ndc}", h.LookupNDC)
	return r
}

func TestQueryHandler_LookupProduct_Found(t *testing.T) {
	mock := &mockQueryProvider{
		lookupProductFn: func(_ context.Context, _ []string) (*store.ProductResult, error) {
			return &store.ProductResult{ProductNDC: "0002-1433", BrandName: strPtr("Metformin")}, nil
		},
	}
	router := setupQueryTestRouter(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/ndc/0002-1433", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var result store.ProductResult
	_ = json.Unmarshal(rec.Body.Bytes(), &result)
	if result.ProductNDC != "0002-1433" {
		t.Errorf("expected product_ndc 0002-1433, got %s", result.ProductNDC)
	}
}

func TestQueryHandler_LookupProduct_NotFound(t *testing.T) {
	mock := &mockQueryProvider{}
	router := setupQueryTestRouter(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/ndc/9999-9999", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestQueryHandler_LookupProduct_InvalidNDC(t *testing.T) {
	mock := &mockQueryProvider{}
	router := setupQueryTestRouter(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/ndc/abc", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestQueryHandler_LookupPackage(t *testing.T) {
	mock := &mockQueryProvider{
		lookupPackageFn: func(_ context.Context, _ []string) (*store.ProductResult, string, error) {
			return &store.ProductResult{ProductNDC: "0002-1433"}, "0002-1433-61", nil
		},
	}
	router := setupQueryTestRouter(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/ndc/0002-1433-61", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestQueryHandler_Search_Success(t *testing.T) {
	mock := &mockQueryProvider{
		searchFn: func(_ context.Context, _ string, _, _ int) ([]store.SearchResult, int, error) {
			return []store.SearchResult{{ProductNDC: "0002-1433", Relevance: 0.9}}, 1, nil
		},
	}
	router := setupQueryTestRouter(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/ndc/search?q=metformin&limit=10", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var resp map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp["total"].(float64) != 1 {
		t.Errorf("expected total 1, got %v", resp["total"])
	}
}

func TestQueryHandler_Search_MissingQuery(t *testing.T) {
	mock := &mockQueryProvider{}
	router := setupQueryTestRouter(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/ndc/search", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestQueryHandler_Search_Error(t *testing.T) {
	mock := &mockQueryProvider{
		searchFn: func(_ context.Context, _ string, _, _ int) ([]store.SearchResult, int, error) {
			return nil, 0, fmt.Errorf("db error")
		},
	}
	router := setupQueryTestRouter(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/ndc/search?q=test", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rec.Code)
	}
}

func TestQueryHandler_Packages(t *testing.T) {
	mock := &mockQueryProvider{
		packagesFn: func(_ context.Context, _ string) ([]store.PackageResult, error) {
			return []store.PackageResult{{NDC: "0002-1433-61", Description: strPtr("100 TABLET")}}, nil
		},
	}
	router := setupQueryTestRouter(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/ndc/0002-1433/packages", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestQueryHandler_Packages_InvalidNDC(t *testing.T) {
	mock := &mockQueryProvider{}
	router := setupQueryTestRouter(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/ndc/abc/packages", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestQueryHandler_Stats(t *testing.T) {
	mock := &mockQueryProvider{
		statsFn: func(_ context.Context) (*store.StatsResult, error) {
			return &store.StatsResult{Products: 100, Packages: 200, Applications: 50}, nil
		},
	}
	router := setupQueryTestRouter(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/ndc/stats", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var result store.StatsResult
	_ = json.Unmarshal(rec.Body.Bytes(), &result)
	if result.Products != 100 {
		t.Errorf("expected 100 products, got %d", result.Products)
	}
}

func TestQueryHandler_Stats_Error(t *testing.T) {
	mock := &mockQueryProvider{
		statsFn: func(_ context.Context) (*store.StatsResult, error) {
			return nil, fmt.Errorf("db error")
		},
	}
	router := setupQueryTestRouter(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/ndc/stats", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rec.Code)
	}
}
