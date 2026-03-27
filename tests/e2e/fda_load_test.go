//go:build e2e

package e2e

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/calebdunn/ndc-loader/internal/loader"
	"github.com/calebdunn/ndc-loader/internal/model"
	"github.com/calebdunn/ndc-loader/internal/store"
)

func TestFullFDALoad(t *testing.T) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set")
	}

	ctx := context.Background()
	logger := slog.Default()

	// Connect and migrate.
	db, err := store.NewDB(ctx, dbURL)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer db.Close()

	if err := store.RunMigrations(ctx, db); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	// Load datasets config.
	datasetsCfg, err := loader.LoadDatasetsConfig("../../datasets.yaml")
	if err != nil {
		t.Fatalf("failed to load datasets config: %v", err)
	}

	// Set up orchestrator.
	tmpDir := t.TempDir()
	checkpointStore := store.NewCheckpointStore(db)
	dataLoader := store.NewDataLoader(db, 0.20)
	downloader := loader.NewDownloader(tmpDir, 2)

	orch := loader.NewOrchestrator(logger, downloader, dataLoader, checkpointStore, datasetsCfg)

	// Run the full load.
	t.Log("Starting full FDA data load...")
	loadID, err := orch.RunLoad(ctx, nil, false, "")
	if err != nil {
		t.Fatalf("RunLoad failed: %v", err)
	}
	t.Logf("Load complete: %s", loadID)

	// Verify row counts.
	var productCount, packageCount int
	db.QueryRow(ctx, "SELECT COUNT(*) FROM products").Scan(&productCount)
	db.QueryRow(ctx, "SELECT COUNT(*) FROM packages").Scan(&packageCount)
	t.Logf("Products: %d, Packages: %d", productCount, packageCount)

	if productCount < 100000 {
		t.Errorf("expected >100K products, got %d", productCount)
	}
	if packageCount < 200000 {
		t.Errorf("expected >200K packages, got %d", packageCount)
	}

	// Verify Drugs@FDA tables.
	var applCount, dfProductCount, submissionCount int
	db.QueryRow(ctx, "SELECT COUNT(*) FROM applications").Scan(&applCount)
	db.QueryRow(ctx, "SELECT COUNT(*) FROM drugsfda_products").Scan(&dfProductCount)
	db.QueryRow(ctx, "SELECT COUNT(*) FROM submissions").Scan(&submissionCount)
	t.Logf("Applications: %d, DrugsFDA Products: %d, Submissions: %d", applCount, dfProductCount, submissionCount)

	if applCount < 20000 {
		t.Errorf("expected >20K applications, got %d", applCount)
	}

	// Verify join: products.application_number -> applications.appl_no
	// NDC uses format "ANDA076543" or "NDA018936" (type prefix + number).
	// Drugs@FDA uses just "076543" (number only, zero-padded to 6 digits).
	// Join by stripping the alpha prefix and zero-padding to 6 digits.
	var joinCount int
	_ = db.QueryRow(ctx, `
		SELECT COUNT(*) FROM products p
		JOIN applications a ON LPAD(regexp_replace(p.application_number, '^[A-Za-z]+', ''), 6, '0') = a.appl_no
		WHERE p.application_number IS NOT NULL
		  AND p.application_number ~ '^[A-Za-z]'
	`).Scan(&joinCount)
	t.Logf("Products with matching application (normalized join): %d", joinCount)

	if joinCount == 0 {
		t.Error("expected some products to join with applications via normalized application_number")
	}

	// Verify search_vector is populated.
	var searchCount int
	db.QueryRow(ctx, "SELECT COUNT(*) FROM products WHERE search_vector IS NOT NULL").Scan(&searchCount)
	t.Logf("Products with search_vector: %d", searchCount)

	if searchCount < productCount/2 {
		t.Errorf("expected most products to have search_vector, got %d/%d", searchCount, productCount)
	}

	// Verify full-text search works.
	var searchResultCount int
	db.QueryRow(ctx, `
		SELECT COUNT(*) FROM products
		WHERE search_vector @@ plainto_tsquery('english', 'metformin')
	`).Scan(&searchResultCount)
	t.Logf("Search 'metformin': %d results", searchResultCount)

	if searchResultCount == 0 {
		t.Error("expected search for 'metformin' to return results")
	}

	// Verify checkpoints.
	checkpoints, err := checkpointStore.GetCheckpoints(ctx, loadID)
	if err != nil {
		t.Fatalf("failed to get checkpoints: %v", err)
	}
	t.Logf("Checkpoints: %d", len(checkpoints))

	for _, cp := range checkpoints {
		t.Logf("  %s.%s: %s (rows: %v)", cp.Dataset, cp.TableName, cp.Status, cp.RowCount)
		if cp.Status != model.LoadStatusLoaded {
			t.Errorf("checkpoint %s.%s has status %s, error: %v", cp.Dataset, cp.TableName, cp.Status, cp.ErrorMessage)
		}
	}
}
