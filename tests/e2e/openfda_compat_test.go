//go:build e2e

package e2e

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/calebdunn/ndc-loader/internal/api"
)

func TestOpenFDACompat_SearchMetformin(t *testing.T) {
	router := setupQueryRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/openfda/ndc.json?search=brand_name:metformin&limit=5", nil)
	req.Header.Set("X-API-Key", "test-key")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp api.OpenFDAResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	// Verify meta structure.
	if resp.Meta.Disclaimer == "" {
		t.Error("expected non-empty disclaimer")
	}
	if resp.Meta.Terms != "https://open.fda.gov/terms/" {
		t.Errorf("expected openFDA terms URL, got %s", resp.Meta.Terms)
	}
	if resp.Meta.Results.Limit != 5 {
		t.Errorf("expected limit 5, got %d", resp.Meta.Results.Limit)
	}
	if resp.Meta.Results.Total == 0 {
		t.Error("expected non-zero total")
	}
	t.Logf("Total results for brand_name:metformin: %d", resp.Meta.Results.Total)

	// Verify result structure.
	if len(resp.Results) == 0 {
		t.Fatal("expected at least 1 result")
	}
	if len(resp.Results) > 5 {
		t.Errorf("expected max 5 results, got %d", len(resp.Results))
	}

	r := resp.Results[0]
	if r.ProductNDC == "" {
		t.Error("expected non-empty product_ndc")
	}
	if len(r.ActiveIngredients) == 0 {
		t.Error("expected non-empty active_ingredients array")
	}
	if r.ActiveIngredients[0].Name == "" {
		t.Error("expected non-empty ingredient name")
	}
	if len(r.Route) == 0 {
		t.Error("expected non-empty route array")
	}
	if len(r.OpenFDA.ManufacturerName) == 0 {
		t.Error("expected non-empty openfda.manufacturer_name")
	}
	t.Logf("First result: %s (%s)", r.ProductNDC, r.BrandName)
}

func TestOpenFDACompat_FullTextSearch(t *testing.T) {
	router := setupQueryRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/openfda/ndc.json?search=metformin&limit=3", nil)
	req.Header.Set("X-API-Key", "test-key")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp api.OpenFDAResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)

	if resp.Meta.Results.Total == 0 {
		t.Error("expected results for full-text 'metformin'")
	}
}

func TestOpenFDACompat_ProductNDCLookup(t *testing.T) {
	router := setupQueryRouter(t)

	req := httptest.NewRequest(http.MethodGet, `/api/openfda/ndc.json?search=product_ndc:"0002-1433"&limit=1`, nil)
	req.Header.Set("X-API-Key", "test-key")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp api.OpenFDAResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)

	if len(resp.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(resp.Results))
	}
	if resp.Results[0].ProductNDC != "0002-1433" {
		t.Errorf("expected product_ndc 0002-1433, got %s", resp.Results[0].ProductNDC)
	}
	if len(resp.Results[0].Packaging) == 0 {
		t.Error("expected packages in result")
	}
}

func TestOpenFDACompat_NotFound(t *testing.T) {
	router := setupQueryRouter(t)

	req := httptest.NewRequest(http.MethodGet, `/api/openfda/ndc.json?search=product_ndc:"9999-9999"&limit=1`, nil)
	req.Header.Set("X-API-Key", "test-key")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}

	var errResp api.OpenFDAError
	_ = json.Unmarshal(rec.Body.Bytes(), &errResp)

	if errResp.Error.Code != "NOT_FOUND" {
		t.Errorf("expected error code NOT_FOUND, got %s", errResp.Error.Code)
	}
	if errResp.Error.Message != "No matches found!" {
		t.Errorf("expected 'No matches found!', got %s", errResp.Error.Message)
	}
}

func TestOpenFDACompat_MissingSearch(t *testing.T) {
	router := setupQueryRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/openfda/ndc.json", nil)
	req.Header.Set("X-API-Key", "test-key")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing search, got %d", rec.Code)
	}
}

func TestOpenFDACompat_DefaultLimit(t *testing.T) {
	router := setupQueryRouter(t)

	// No limit param — should default to 1 (matching openFDA).
	req := httptest.NewRequest(http.MethodGet, "/api/openfda/ndc.json?search=metformin", nil)
	req.Header.Set("X-API-Key", "test-key")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp api.OpenFDAResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)

	if resp.Meta.Results.Limit != 1 {
		t.Errorf("expected default limit 1, got %d", resp.Meta.Results.Limit)
	}
	if len(resp.Results) != 1 {
		t.Errorf("expected 1 result with default limit, got %d", len(resp.Results))
	}
}

func TestOpenFDACompat_Pagination(t *testing.T) {
	router := setupQueryRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/openfda/ndc.json?search=metformin&limit=2&skip=2", nil)
	req.Header.Set("X-API-Key", "test-key")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp api.OpenFDAResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)

	if resp.Meta.Results.Skip != 2 {
		t.Errorf("expected skip 2, got %d", resp.Meta.Results.Skip)
	}
	if resp.Meta.Results.Limit != 2 {
		t.Errorf("expected limit 2, got %d", resp.Meta.Results.Limit)
	}
}

func TestOpenFDACompat_RequiresAuth(t *testing.T) {
	router := setupQueryRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/openfda/ndc.json?search=metformin", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

// TestOpenFDACompat_FormatParity compares our response structure against the real openFDA API.
func TestOpenFDACompat_FormatParity(t *testing.T) {
	router := setupQueryRouter(t)

	// Fetch from real openFDA API.
	realResp, err := http.Get("https://api.fda.gov/drug/ndc.json?search=brand_name:metformin&limit=1")
	if err != nil {
		t.Skipf("cannot reach openFDA API: %v", err)
	}
	defer realResp.Body.Close()

	if realResp.StatusCode != http.StatusOK {
		t.Skipf("openFDA API returned %d", realResp.StatusCode)
	}

	realBody, _ := io.ReadAll(realResp.Body)
	var realData map[string]interface{}
	if err := json.Unmarshal(realBody, &realData); err != nil {
		t.Fatalf("failed to parse real openFDA response: %v", err)
	}

	// Fetch from our API.
	req := httptest.NewRequest(http.MethodGet, "/api/openfda/ndc.json?search=brand_name:metformin&limit=1", nil)
	req.Header.Set("X-API-Key", "test-key")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	var ourData map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &ourData); err != nil {
		t.Fatalf("failed to parse our response: %v", err)
	}

	// Compare top-level keys.
	realMeta, _ := realData["meta"].(map[string]interface{})
	ourMeta, _ := ourData["meta"].(map[string]interface{})

	for _, key := range []string{"disclaimer", "terms", "license", "last_updated", "results"} {
		if _, ok := ourMeta[key]; !ok {
			t.Errorf("missing meta.%s in our response", key)
		}
		if _, ok := realMeta[key]; !ok {
			t.Logf("note: real openFDA also missing meta.%s", key)
		}
	}

	// Compare result field names.
	realResults, _ := realData["results"].([]interface{})
	ourResults, _ := ourData["results"].([]interface{})

	if len(realResults) == 0 || len(ourResults) == 0 {
		t.Skip("need at least 1 result from both APIs")
	}

	realProduct, _ := realResults[0].(map[string]interface{})
	ourProduct, _ := ourResults[0].(map[string]interface{})

	requiredFields := []string{
		"product_ndc", "generic_name", "labeler_name", "brand_name",
		"active_ingredients", "finished", "packaging", "openfda",
		"marketing_category", "dosage_form", "product_type",
		"route", "application_number", "pharm_class",
	}

	for _, field := range requiredFields {
		if _, ok := ourProduct[field]; !ok {
			t.Errorf("missing field %q in our response", field)
		}
		if _, ok := realProduct[field]; !ok {
			t.Logf("note: field %q not in real openFDA response either", field)
		}
	}

	// Verify array types match.
	arrayFields := []string{"active_ingredients", "packaging", "route", "pharm_class"}
	for _, field := range arrayFields {
		if _, ok := ourProduct[field].([]interface{}); !ok {
			t.Errorf("field %q should be an array in our response", field)
		}
	}

	t.Log("Format parity check passed — all required fields present with correct types")

	// Save comparison for debugging if needed.
	if os.Getenv("SAVE_PARITY_CHECK") != "" {
		os.WriteFile("/tmp/openfda_real.json", realBody, 0o644)
		os.WriteFile("/tmp/openfda_ours.json", rec.Body.Bytes(), 0o644)
		t.Log("Saved responses to /tmp/openfda_real.json and /tmp/openfda_ours.json")
	}
}
