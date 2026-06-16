//go:build integration

package integration

import (
	"context"
	"testing"

	"github.com/calebdunn/ndc-loader/internal/store"
)

// seedExcludeFixtures inserts a matched pair of products (one normal, one
// flagged ndc_exclude=TRUE) plus packages, so the query layer can be exercised
// for the "excluded NDCs leak into results" bug (issue #10).
//
// All rows share the proprietary_name "ZZZEXCLUDEFIX" so a single search term
// hits both, and live under NDCs prefixed 90000- so cleanup is isolated.
func seedExcludeFixtures(t *testing.T, tdb *store.TestDB) {
	t.Helper()
	ctx := context.Background()

	cleanupExcludeFixtures(t, tdb)

	// Normal, patient-facing product.
	_, err := tdb.Pool.Exec(ctx, `
		INSERT INTO products (product_id, product_ndc, proprietary_name, ndc_exclude, search_vector)
		VALUES ('EXC-NORMAL', '90000-001', 'ZZZEXCLUDEFIX', FALSE,
		        to_tsvector('english', 'ZZZEXCLUDEFIX'))
	`)
	if err != nil {
		t.Fatalf("seed normal product: %v", err)
	}

	// Excluded product (bulk ingredient / compounding component).
	_, err = tdb.Pool.Exec(ctx, `
		INSERT INTO products (product_id, product_ndc, proprietary_name, ndc_exclude, search_vector)
		VALUES ('EXC-BULK', '90000-002', 'ZZZEXCLUDEFIX', TRUE,
		        to_tsvector('english', 'ZZZEXCLUDEFIX'))
	`)
	if err != nil {
		t.Fatalf("seed excluded product: %v", err)
	}

	// Packages: one normal and one excluded, both under the normal product.
	_, err = tdb.Pool.Exec(ctx, `
		INSERT INTO packages (product_id, product_ndc, ndc_package_code, ndc_exclude)
		VALUES ('EXC-NORMAL', '90000-001', '90000-001-01', FALSE),
		       ('EXC-NORMAL', '90000-001', '90000-001-99', TRUE)
	`)
	if err != nil {
		t.Fatalf("seed packages: %v", err)
	}

	// Package belonging to the excluded product.
	_, err = tdb.Pool.Exec(ctx, `
		INSERT INTO packages (product_id, product_ndc, ndc_package_code, ndc_exclude)
		VALUES ('EXC-BULK', '90000-002', '90000-002-01', FALSE)
	`)
	if err != nil {
		t.Fatalf("seed excluded product package: %v", err)
	}
}

func cleanupExcludeFixtures(t *testing.T, tdb *store.TestDB) {
	t.Helper()
	ctx := context.Background()
	tdb.Pool.Exec(ctx, "DELETE FROM packages WHERE product_ndc LIKE '90000-%'")
	tdb.Pool.Exec(ctx, "DELETE FROM products WHERE product_ndc LIKE '90000-%'")
}

// TestSearchProducts_ExcludedProductHiddenByDefault reproduces issue #10:
// a full-text search must not surface ndc_exclude=TRUE products by default.
func TestSearchProducts_ExcludedProductHiddenByDefault(t *testing.T) {
	tdb := getTestDB(t)
	defer tdb.Pool.Close()
	seedExcludeFixtures(t, tdb)
	defer cleanupExcludeFixtures(t, tdb)

	ctx := context.Background()
	q := store.NewQueryStore(tdb.Pool)

	results, total, err := q.SearchProducts(ctx, "ZZZEXCLUDEFIX", 50, 0, false)
	if err != nil {
		t.Fatalf("SearchProducts: %v", err)
	}
	if total != 1 {
		t.Errorf("expected total=1 (excluded product hidden), got %d", total)
	}
	for _, r := range results {
		if r.ProductNDC == "90000-002" {
			t.Errorf("excluded product 90000-002 leaked into search results")
		}
	}

	// Opt-in: include_excluded=true surfaces both products.
	_, totalAll, err := q.SearchProducts(ctx, "ZZZEXCLUDEFIX", 50, 0, true)
	if err != nil {
		t.Fatalf("SearchProducts(includeExcluded): %v", err)
	}
	if totalAll != 2 {
		t.Errorf("expected total=2 with includeExcluded, got %d", totalAll)
	}
}

// TestLookupByProductNDC_ExcludedProductNotFoundByDefault reproduces issue #10:
// looking up an excluded product NDC must report not-found by default.
func TestLookupByProductNDC_ExcludedProductNotFoundByDefault(t *testing.T) {
	tdb := getTestDB(t)
	defer tdb.Pool.Close()
	seedExcludeFixtures(t, tdb)
	defer cleanupExcludeFixtures(t, tdb)

	ctx := context.Background()
	q := store.NewQueryStore(tdb.Pool)

	if _, err := q.LookupByProductNDC(ctx, []string{"90000-002"}, false); err == nil {
		t.Errorf("expected not-found for excluded product 90000-002, got a result")
	}

	// The normal product must still resolve.
	if _, err := q.LookupByProductNDC(ctx, []string{"90000-001"}, false); err != nil {
		t.Errorf("expected normal product 90000-001 to resolve, got: %v", err)
	}

	// Opt-in: include_excluded=true resolves the excluded product.
	if _, err := q.LookupByProductNDC(ctx, []string{"90000-002"}, true); err != nil {
		t.Errorf("expected excluded product 90000-002 to resolve with includeExcluded, got: %v", err)
	}
}

// TestLookupByPackageNDC_ExcludedPackageNotFoundByDefault reproduces issue #10:
// looking up an excluded package NDC must report not-found by default.
func TestLookupByPackageNDC_ExcludedPackageNotFoundByDefault(t *testing.T) {
	tdb := getTestDB(t)
	defer tdb.Pool.Close()
	seedExcludeFixtures(t, tdb)
	defer cleanupExcludeFixtures(t, tdb)

	ctx := context.Background()
	q := store.NewQueryStore(tdb.Pool)

	if _, _, err := q.LookupByPackageNDC(ctx, []string{"90000-001-99"}, false); err == nil {
		t.Errorf("expected not-found for excluded package 90000-001-99, got a result")
	}

	// Opt-in: include_excluded=true resolves the excluded package.
	if _, _, err := q.LookupByPackageNDC(ctx, []string{"90000-001-99"}, true); err != nil {
		t.Errorf("expected excluded package 90000-001-99 to resolve with includeExcluded, got: %v", err)
	}
}

// TestGetPackagesByProductNDC_ExcludedPackageHiddenByDefault reproduces issue #10:
// excluded packages must not be returned by default.
func TestGetPackagesByProductNDC_ExcludedPackageHiddenByDefault(t *testing.T) {
	tdb := getTestDB(t)
	defer tdb.Pool.Close()
	seedExcludeFixtures(t, tdb)
	defer cleanupExcludeFixtures(t, tdb)

	ctx := context.Background()
	q := store.NewQueryStore(tdb.Pool)

	pkgs, err := q.GetPackagesByProductNDC(ctx, "90000-001", false)
	if err != nil {
		t.Fatalf("GetPackagesByProductNDC: %v", err)
	}
	for _, p := range pkgs {
		if p.NDC == "90000-001-99" {
			t.Errorf("excluded package 90000-001-99 leaked into results")
		}
	}

	// Opt-in: include_excluded=true returns the excluded package.
	allPkgs, err := q.GetPackagesByProductNDC(ctx, "90000-001", true)
	if err != nil {
		t.Fatalf("GetPackagesByProductNDC(includeExcluded): %v", err)
	}
	if len(allPkgs) != 2 {
		t.Errorf("expected 2 packages with includeExcluded, got %d", len(allPkgs))
	}
}

// TestOpenFDASearch_ExcludedProductHiddenByDefault reproduces issue #10:
// the openFDA-compatible search must not surface excluded products by default.
func TestOpenFDASearch_ExcludedProductHiddenByDefault(t *testing.T) {
	tdb := getTestDB(t)
	defer tdb.Pool.Close()
	seedExcludeFixtures(t, tdb)
	defer cleanupExcludeFixtures(t, tdb)

	ctx := context.Background()
	q := store.NewQueryStore(tdb.Pool)

	products, total, err := q.OpenFDASearch(ctx, "proprietary_name = $1",
		[]interface{}{"ZZZEXCLUDEFIX"}, 100, 0, false)
	if err != nil {
		t.Fatalf("OpenFDASearch: %v", err)
	}
	if total != 1 {
		t.Errorf("expected total=1 (excluded product hidden), got %d", total)
	}
	for _, p := range products {
		if p.ProductNDC == "90000-002" {
			t.Errorf("excluded product 90000-002 leaked into openFDA results")
		}
	}

	// Opt-in: include_excluded=true surfaces both products.
	_, totalAll, err := q.OpenFDASearch(ctx, "proprietary_name = $1",
		[]interface{}{"ZZZEXCLUDEFIX"}, 100, 0, true)
	if err != nil {
		t.Fatalf("OpenFDASearch(includeExcluded): %v", err)
	}
	if totalAll != 2 {
		t.Errorf("expected total=2 with includeExcluded, got %d", totalAll)
	}
}
