//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/calebdunn/ndc-loader/internal/api"
	"github.com/calebdunn/ndc-loader/internal/store"
)

func setupQueryRouter(t *testing.T) http.Handler {
	t.Helper()
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set")
	}

	ctx := context.Background()
	db, err := store.NewDB(ctx, dbURL)
	if err != nil {
		t.Skipf("cannot connect to database: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	if err := store.RunMigrations(ctx, db); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	logger := slog.Default()
	checkpointStore := store.NewCheckpointStore(db)
	queryStore := store.NewQueryStore(db)

	return api.NewRouter(logger, []string{"test-key"}, nil, checkpointStore, queryStore, db)
}

func TestQueryAPI_LookupByProductNDC(t *testing.T) {
	router := setupQueryRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/ndc/0002-1433", nil)
	req.Header.Set("X-API-Key", "test-key")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var result store.ProductResult
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if result.ProductNDC != "0002-1433" {
		t.Errorf("expected product_ndc 0002-1433, got %s", result.ProductNDC)
	}
	if len(result.Packages) == 0 {
		t.Error("expected at least one package")
	}
	t.Logf("Product: %s, Brand: %v, Packages: %d", result.ProductNDC, result.BrandName, len(result.Packages))
}

func TestQueryAPI_LookupByPackageNDC(t *testing.T) {
	router := setupQueryRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/ndc/0002-1433-61", nil)
	req.Header.Set("X-API-Key", "test-key")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var result store.ProductResult
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if result.MatchedPackage == nil {
		t.Error("expected matched_package to be set for 3-segment lookup")
	}
	t.Logf("Matched package: %v", result.MatchedPackage)
}

func TestQueryAPI_LookupNotFound(t *testing.T) {
	router := setupQueryRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/ndc/9999-9999", nil)
	req.Header.Set("X-API-Key", "test-key")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestQueryAPI_LookupInvalidFormat(t *testing.T) {
	router := setupQueryRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/ndc/abc", nil)
	req.Header.Set("X-API-Key", "test-key")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestQueryAPI_Search(t *testing.T) {
	router := setupQueryRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/ndc/search?q=metformin&limit=10", nil)
	req.Header.Set("X-API-Key", "test-key")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var result map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	total := int(result["total"].(float64))
	if total == 0 {
		t.Error("expected search results for 'metformin'")
	}
	t.Logf("Search 'metformin': %d total results", total)

	results := result["results"].([]interface{})
	if len(results) == 0 {
		t.Error("expected non-empty results array")
	}
	if len(results) > 10 {
		t.Errorf("expected max 10 results (limit=10), got %d", len(results))
	}
}

func TestQueryAPI_SearchPrefixMatch(t *testing.T) {
	router := setupQueryRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/ndc/search?q=metfor", nil)
	req.Header.Set("X-API-Key", "test-key")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var result map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &result)
	total := int(result["total"].(float64))
	if total == 0 {
		t.Error("expected prefix search 'metfor' to match 'metformin'")
	}
	t.Logf("Prefix search 'metfor': %d results", total)
}

func TestQueryAPI_SearchMissingQuery(t *testing.T) {
	router := setupQueryRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/ndc/search", nil)
	req.Header.Set("X-API-Key", "test-key")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestQueryAPI_Packages(t *testing.T) {
	router := setupQueryRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/ndc/0002-1433/packages", nil)
	req.Header.Set("X-API-Key", "test-key")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var result map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &result)
	packages := result["packages"].([]interface{})
	if len(packages) == 0 {
		t.Error("expected at least one package")
	}
	t.Logf("Packages for 0002-1433: %d", len(packages))
}

func TestQueryAPI_Stats(t *testing.T) {
	router := setupQueryRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/ndc/stats", nil)
	req.Header.Set("X-API-Key", "test-key")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var result store.StatsResult
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if result.Products == 0 {
		t.Error("expected non-zero product count")
	}
	t.Logf("Stats: products=%d, packages=%d, applications=%d", result.Products, result.Packages, result.Applications)
}

func TestQueryAPI_RequiresAuth(t *testing.T) {
	router := setupQueryRouter(t)

	// No API key.
	req := httptest.NewRequest(http.MethodGet, "/api/ndc/0002-1433", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 without API key, got %d", rec.Code)
	}
}
