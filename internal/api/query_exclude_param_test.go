package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/calebdunn/ndc-loader/internal/store"
)

// TestSearchNDC_IncludeExcludedParamThreaded verifies the search handler hides
// excluded NDCs by default and forwards ?include_excluded=true to the store
// (issue #10).
func TestSearchNDC_IncludeExcludedParamThreaded(t *testing.T) {
	cases := []struct {
		name string
		url  string
		want bool
	}{
		{"default excludes", "/api/ndc/search?q=metformin", false},
		{"opt-in includes", "/api/ndc/search?q=metformin&include_excluded=true", true},
		{"false stays excluded", "/api/ndc/search?q=metformin&include_excluded=false", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock := &mockQueryProvider{
				searchFn: func(_ context.Context, _ string, _, _ int) ([]store.SearchResult, int, error) {
					return []store.SearchResult{}, 0, nil
				},
			}
			router := setupQueryTestRouter(mock)
			req := httptest.NewRequest(http.MethodGet, tc.url, nil)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d", rec.Code)
			}
			if mock.lastIncludeExcluded != tc.want {
				t.Errorf("includeExcluded = %v, want %v", mock.lastIncludeExcluded, tc.want)
			}
		})
	}
}

// TestLookupNDC_IncludeExcludedParamThreaded verifies the lookup handler
// forwards ?include_excluded=true to the store (issue #10).
func TestLookupNDC_IncludeExcludedParamThreaded(t *testing.T) {
	mock := &mockQueryProvider{
		lookupProductFn: func(_ context.Context, _ []string) (*store.ProductResult, error) {
			return &store.ProductResult{ProductNDC: "0002-1433"}, nil
		},
	}
	router := setupQueryTestRouter(mock)
	req := httptest.NewRequest(http.MethodGet, "/api/ndc/0002-1433?include_excluded=true", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !mock.lastIncludeExcluded {
		t.Error("expected includeExcluded=true to be threaded to LookupByProductNDC")
	}
}

// TestListPackages_IncludeExcludedParamThreaded verifies the packages handler
// forwards ?include_excluded=true to the store (issue #10).
func TestListPackages_IncludeExcludedParamThreaded(t *testing.T) {
	mock := &mockQueryProvider{
		packagesFn: func(_ context.Context, _ string) ([]store.PackageResult, error) {
			return []store.PackageResult{{NDC: "0002-1433-80"}}, nil
		},
	}
	router := setupQueryTestRouter(mock)
	req := httptest.NewRequest(http.MethodGet, "/api/ndc/0002-1433/packages?include_excluded=true", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !mock.lastIncludeExcluded {
		t.Error("expected includeExcluded=true to be threaded to GetPackagesByProductNDC")
	}
}
