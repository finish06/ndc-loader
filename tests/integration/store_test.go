//go:build integration

package integration

import (
	"context"
	"os"
	"testing"

	"github.com/calebdunn/ndc-loader/internal/store"
	"github.com/calebdunn/ndc-loader/internal/model"
)

func getTestDB(t *testing.T) *store.TestDB {
	t.Helper()
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://ndc:ndc@localhost:5432/ndc?sslmode=disable"
	}

	ctx := context.Background()
	db, err := store.NewDB(ctx, dbURL)
	if err != nil {
		t.Skipf("skipping integration test: cannot connect to database: %v", err)
	}

	if err := store.RunMigrations(ctx, db); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	return &store.TestDB{Pool: db}
}

func TestMigrations_CreatesTables(t *testing.T) {
	tdb := getTestDB(t)
	defer tdb.Pool.Close()

	ctx := context.Background()
	tables := []string{
		"products", "packages", "applications", "drugsfda_products",
		"submissions", "marketing_status", "active_ingredients",
		"te_codes", "load_checkpoints",
	}

	for _, table := range tables {
		var exists bool
		err := tdb.Pool.QueryRow(ctx,
			"SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = $1)",
			table,
		).Scan(&exists)
		if err != nil {
			t.Fatalf("error checking table %s: %v", table, err)
		}
		if !exists {
			t.Errorf("expected table %s to exist", table)
		}
	}
}

func TestCheckpointStore_CRUD(t *testing.T) {
	tdb := getTestDB(t)
	defer tdb.Pool.Close()

	ctx := context.Background()
	cs := store.NewCheckpointStore(tdb.Pool)

	// Clean up any previous test data.
	tdb.Pool.Exec(ctx, "DELETE FROM load_checkpoints WHERE load_id = 'test-load-123'")

	// Create.
	cp := &model.LoadCheckpoint{
		LoadID:    "test-load-123",
		Dataset:   "ndc_directory",
		TableName: "products",
		Status:    model.LoadStatusPending,
	}
	err := cs.CreateCheckpoint(ctx, cp)
	if err != nil {
		t.Fatalf("failed to create checkpoint: %v", err)
	}

	// Read.
	checkpoints, err := cs.GetCheckpoints(ctx, "test-load-123")
	if err != nil {
		t.Fatalf("failed to get checkpoints: %v", err)
	}
	if len(checkpoints) != 1 {
		t.Fatalf("expected 1 checkpoint, got %d", len(checkpoints))
	}
	if checkpoints[0].Status != model.LoadStatusPending {
		t.Errorf("expected pending status, got %s", checkpoints[0].Status)
	}

	// Update status.
	err = cs.UpdateStatus(ctx, "test-load-123", "products", model.LoadStatusLoading)
	if err != nil {
		t.Fatalf("failed to update status: %v", err)
	}

	checkpoints, _ = cs.GetCheckpoints(ctx, "test-load-123")
	if checkpoints[0].Status != model.LoadStatusLoading {
		t.Errorf("expected loading status, got %s", checkpoints[0].Status)
	}

	// Set row count.
	err = cs.SetRowCount(ctx, "test-load-123", "products", 1000)
	if err != nil {
		t.Fatalf("failed to set row count: %v", err)
	}

	// Mark as loaded.
	err = cs.UpdateStatus(ctx, "test-load-123", "products", model.LoadStatusLoaded)
	if err != nil {
		t.Fatalf("failed to update to loaded: %v", err)
	}

	// Get loaded tables.
	loaded, err := cs.GetLoadedTables(ctx, "test-load-123")
	if err != nil {
		t.Fatalf("failed to get loaded tables: %v", err)
	}
	if !loaded["products"] {
		t.Error("expected products to be in loaded tables")
	}

	// Get previous row count.
	prev, err := cs.GetPreviousRowCount(ctx, "products")
	if err != nil {
		t.Fatalf("failed to get previous row count: %v", err)
	}
	if prev != 1000 {
		t.Errorf("expected previous row count 1000, got %d", prev)
	}

	// Check active load — this one should be loaded (not active).
	_, active, err := cs.HasActiveLoad(ctx)
	if err != nil {
		t.Fatalf("failed to check active load: %v", err)
	}
	if active {
		t.Error("expected no active load after all tables loaded")
	}

	// Cleanup.
	tdb.Pool.Exec(ctx, "DELETE FROM load_checkpoints WHERE load_id = 'test-load-123'")
}

func TestBulkLoad_AtomicSwap(t *testing.T) {
	tdb := getTestDB(t)
	defer tdb.Pool.Close()

	ctx := context.Background()
	dl := store.NewDataLoader(tdb.Pool, 0.20)

	// Prepare test data — load into applications table (simplest schema).
	columns := []string{"appl_no", "appl_type", "sponsor_name"}
	rows := [][]interface{}{
		{"TEST001", "NDA", "Test Sponsor 1"},
		{"TEST002", "ANDA", "Test Sponsor 2"},
		{"TEST003", "NDA", "Test Sponsor 3"},
	}

	result, err := dl.BulkLoad(ctx, "applications", columns, rows)
	if err != nil {
		t.Fatalf("bulk load failed: %v", err)
	}
	if result.RowCount != 3 {
		t.Errorf("expected 3 rows loaded, got %d", result.RowCount)
	}

	// Verify data is in the live table.
	count, err := dl.GetCurrentRowCount(ctx, "applications")
	if err != nil {
		t.Fatalf("failed to get row count: %v", err)
	}
	if count < 3 {
		t.Errorf("expected at least 3 rows, got %d", count)
	}

	// Second load should replace data atomically.
	rows2 := [][]interface{}{
		{"TEST004", "NDA", "New Sponsor"},
	}
	result2, err := dl.BulkLoad(ctx, "applications", columns, rows2)
	if err != nil {
		t.Fatalf("second bulk load failed: %v", err)
	}
	if result2.RowCount != 1 {
		t.Errorf("expected 1 row loaded, got %d", result2.RowCount)
	}

	// Verify old data is gone.
	count, _ = dl.GetCurrentRowCount(ctx, "applications")
	if count != 1 {
		t.Errorf("expected 1 row after swap, got %d", count)
	}

	// Cleanup.
	tdb.Pool.Exec(ctx, "DELETE FROM applications")
}
