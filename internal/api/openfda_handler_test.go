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

func setupOpenFDATestRouter(mock *mockQueryProvider) http.Handler {
	r := chi.NewRouter()
	h := NewOpenFDAHandler(slog.Default(), mock)
	r.Get("/api/openfda/ndc.json", h.HandleNDCJSON)
	return r
}

func TestOpenFDAHandler_SearchSuccess(t *testing.T) {
	mock := &mockQueryProvider{
		openFDASearchFn: func(_ context.Context, _ string, _ []interface{}, _, _ int) ([]store.ProductResult, int, error) {
			return []store.ProductResult{
				{
					ProductNDC:       "0002-1433",
					BrandName:        strPtr("Metformin"),
					GenericName:      strPtr("METFORMIN HCL"),
					Manufacturer:     strPtr("Lilly"),
					ActiveIngredient: strPtr("METFORMIN HCL"),
					Strength:         strPtr("500"),
					StrengthUnit:     strPtr("mg/1"),
					Route:            strPtr("ORAL"),
					PharmClasses:     strPtr("Biguanide [EPC]"),
				},
			}, 100, nil
		},
	}
	router := setupOpenFDATestRouter(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/openfda/ndc.json?search=metformin&limit=5", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp OpenFDAResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)

	if resp.Meta.Results.Total != 100 {
		t.Errorf("expected total 100, got %d", resp.Meta.Results.Total)
	}
	if resp.Meta.Results.Limit != 5 {
		t.Errorf("expected limit 5, got %d", resp.Meta.Results.Limit)
	}
	if resp.Meta.Disclaimer == "" {
		t.Error("expected non-empty disclaimer")
	}
	if len(resp.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(resp.Results))
	}
	if resp.Results[0].ProductNDC != "0002-1433" {
		t.Errorf("expected product_ndc 0002-1433, got %s", resp.Results[0].ProductNDC)
	}
	if len(resp.Results[0].ActiveIngredients) == 0 {
		t.Error("expected active_ingredients array")
	}
	if len(resp.Results[0].Route) == 0 {
		t.Error("expected route array")
	}
}

func TestOpenFDAHandler_MissingSearch(t *testing.T) {
	mock := &mockQueryProvider{}
	router := setupOpenFDATestRouter(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/openfda/ndc.json", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}

	var errResp OpenFDAError
	_ = json.Unmarshal(rec.Body.Bytes(), &errResp)
	if errResp.Error.Code != "BAD_REQUEST" {
		t.Errorf("expected BAD_REQUEST, got %s", errResp.Error.Code)
	}
}

func TestOpenFDAHandler_NotFound(t *testing.T) {
	mock := &mockQueryProvider{
		openFDASearchFn: func(_ context.Context, _ string, _ []interface{}, _, _ int) ([]store.ProductResult, int, error) {
			return nil, 0, nil
		},
	}
	router := setupOpenFDATestRouter(mock)

	req := httptest.NewRequest(http.MethodGet, `/api/openfda/ndc.json?search=product_ndc:"9999-9999"`, nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}

	var errResp OpenFDAError
	_ = json.Unmarshal(rec.Body.Bytes(), &errResp)
	if errResp.Error.Code != "NOT_FOUND" {
		t.Errorf("expected NOT_FOUND, got %s", errResp.Error.Code)
	}
}

func TestOpenFDAHandler_SearchError(t *testing.T) {
	mock := &mockQueryProvider{
		openFDASearchFn: func(_ context.Context, _ string, _ []interface{}, _, _ int) ([]store.ProductResult, int, error) {
			return nil, 0, fmt.Errorf("db error")
		},
	}
	router := setupOpenFDATestRouter(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/openfda/ndc.json?search=metformin", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rec.Code)
	}
}

func TestOpenFDAHandler_DefaultLimit(t *testing.T) {
	mock := &mockQueryProvider{
		openFDASearchFn: func(_ context.Context, _ string, _ []interface{}, limit, _ int) ([]store.ProductResult, int, error) {
			if limit != 1 {
				return nil, 0, fmt.Errorf("expected default limit 1, got %d", limit)
			}
			return []store.ProductResult{{ProductNDC: "test"}}, 1, nil
		},
	}
	router := setupOpenFDATestRouter(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/openfda/ndc.json?search=test", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestOpenFDAHandler_Pagination(t *testing.T) {
	mock := &mockQueryProvider{
		openFDASearchFn: func(_ context.Context, _ string, _ []interface{}, limit, skip int) ([]store.ProductResult, int, error) {
			if skip != 10 || limit != 5 {
				return nil, 0, fmt.Errorf("expected skip=10 limit=5, got skip=%d limit=%d", skip, limit)
			}
			return []store.ProductResult{{ProductNDC: "test"}}, 100, nil
		},
	}
	router := setupOpenFDATestRouter(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/openfda/ndc.json?search=test&limit=5&skip=10", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp OpenFDAResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp.Meta.Results.Skip != 10 {
		t.Errorf("expected skip 10, got %d", resp.Meta.Results.Skip)
	}
}

func TestOpenFDAHandler_LimitCap(t *testing.T) {
	mock := &mockQueryProvider{
		openFDASearchFn: func(_ context.Context, _ string, _ []interface{}, limit, _ int) ([]store.ProductResult, int, error) {
			if limit != 1000 {
				return nil, 0, fmt.Errorf("expected capped limit 1000, got %d", limit)
			}
			return []store.ProductResult{{ProductNDC: "test"}}, 1, nil
		},
	}
	router := setupOpenFDATestRouter(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/openfda/ndc.json?search=test&limit=5000", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}
