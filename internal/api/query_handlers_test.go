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

// Reproduces Code/ndc-loader#4: a request for limit=999 made the handler echo
// "limit":999 in the response while the store silently caps results at 50. The
// envelope must report the cap actually applied, and the store must never be
// asked for more than the cap.
func TestQueryHandler_Search_OverLimitReportsAppliedCap(t *testing.T) {
	var gotLimit int
	mock := &mockQueryProvider{
		searchFn: func(_ context.Context, _ string, limit, _ int) ([]store.SearchResult, int, error) {
			gotLimit = limit
			return []store.SearchResult{{ProductNDC: "0002-1433", Relevance: 0.9}}, 412, nil
		},
	}
	router := setupQueryTestRouter(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/ndc/search?q=aspirin&limit=999", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if gotLimit != 50 {
		t.Errorf("store should receive the applied cap 50, got %d", gotLimit)
	}
	var resp map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp["limit"].(float64) != 50 {
		t.Errorf("response should report the applied cap 50, got %v", resp["limit"])
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

// Issue #8: a drug whose product NDC is stored in 5-3 form ("12345-678") must
// return its packages when queried via the 10-digit unhyphenated NDC. Before the
// fix, ListPackages only tried the 4-4 product variant and returned an empty list.
func TestQueryHandler_Packages_TenDigit_FiveThreeLabeler_ReturnsPackages(t *testing.T) {
	mock := &mockQueryProvider{
		packagesFn: func(_ context.Context, productNDC string) ([]store.PackageResult, error) {
			// Only the 5-3 product variant matches this drug.
			if productNDC == "12345-678" {
				return []store.PackageResult{{NDC: "12345-678-90", Description: strPtr("30 TABLET")}}, nil
			}
			return nil, nil
		},
	}
	router := setupQueryTestRouter(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/ndc/1234567890/packages", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp struct {
		ProductNDC string                `json:"product_ndc"`
		Packages   []store.PackageResult `json:"packages"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Packages) != 1 {
		t.Fatalf("expected 1 package for 5-3 labeler via 10-digit NDC, got %d: %s", len(resp.Packages), rec.Body.String())
	}
	if resp.ProductNDC != "12345-678" {
		t.Errorf("expected product_ndc 12345-678 (the matched variant), got %s", resp.ProductNDC)
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
