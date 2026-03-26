package loader

import (
	"archive/zip"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/calebdunn/ndc-loader/internal/model"
)

// createOrchestratorTestZip creates a ZIP file with product and package data for testing.
func createOrchestratorTestZip(t *testing.T, dir string) string {
	t.Helper()
	zipPath := filepath.Join(dir, "ndc.zip")
	f, err := os.Create(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	w := zip.NewWriter(f)

	productData := "PRODUCTID\tPRODUCTNDC\tPRODUCTTYPENAME\tPROPRIETARYNAME\tNONPROPRIETARYNAME\tDOSAGEFORMNAME\tROUTENAME\tSTARTMARKETINGDATE\tENDMARKETINGDATE\tMARKETINGCATEGORYNAME\tAPPLICATIONNUMBER\tLABELERNAME\tSUBSTANCENAME\tACTIVE_NUMERATOR_STRENGTH\tACTIVE_INGRED_UNIT\tPHARM_CLASSES\tDEASCHEDULE\tNDC_EXCLUDE_FLAG\tLISTING_RECORD_CERTIFIED_THROUGH\n" +
		"TEST001\t0002-1433\tHUMAN PRESCRIPTION DRUG\tTestDrug\tTESTSUBSTANCE\tTABLET\tORAL\t19950301\t\tANDA\tANDA076543\tTest Lab\tTESTSUBSTANCE\t500\tmg/1\tTest [EPC]\t\tN\t20260301\n"

	fw, _ := w.Create("product.txt")
	fw.Write([]byte(productData))

	packageData := "PRODUCTID\tPRODUCTNDC\tNDCPACKAGECODE\tPACKAGEDESCRIPTION\tSTARTMARKETINGDATE\tENDMARKETINGDATE\tNDC_EXCLUDE_FLAG\tSAMPLE_PACKAGE\n" +
		"TEST001\t0002-1433\t0002-1433-02\t100 TABLET in 1 BOTTLE\t19950301\t\tN\tN\n"

	fw2, _ := w.Create("package.txt")
	fw2.Write([]byte(packageData))

	w.Close()
	return zipPath
}

func testDatasetsCfg(serverURL string) *model.DatasetsConfig {
	return &model.DatasetsConfig{
		Datasets: []model.DatasetConfig{
			{
				Name:      "ndc_directory",
				Enabled:   true,
				SourceURL: serverURL + "/ndc.zip",
				Format:    "zip",
				Files: []model.FileConfig{
					{Filename: "product.txt", Table: "products", Delimiter: "\t", HasHeader: true},
					{Filename: "package.txt", Table: "packages", Delimiter: "\t", HasHeader: true},
				},
			},
		},
	}
}

func TestRunLoad_HappyPath(t *testing.T) {
	tmpDir := t.TempDir()
	zipPath := createOrchestratorTestZip(t, tmpDir)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, zipPath)
	}))
	defer server.Close()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	downloader := NewDownloader(t.TempDir(), 1)
	cpMgr := newMockCheckpointManager()
	bulkLoader := newMockBulkLoader()
	cfg := testDatasetsCfg(server.URL)

	orch := NewOrchestrator(logger, downloader, bulkLoader, cpMgr, cfg)

	ctx := context.Background()
	loadID, err := orch.RunLoad(ctx, nil, false, "")
	if err != nil {
		t.Fatalf("RunLoad failed: %v", err)
	}
	if loadID == "" {
		t.Fatal("expected non-empty load ID")
	}

	// Verify checkpoints were created for both tables.
	if len(cpMgr.createCalls) != 2 {
		t.Errorf("expected 2 checkpoint creates, got %d", len(cpMgr.createCalls))
	}

	// Verify bulk load was called for both tables.
	if len(bulkLoader.bulkLoadCalls) != 2 {
		t.Errorf("expected 2 bulk load calls, got %d", len(bulkLoader.bulkLoadCalls))
	}

	// Verify status was updated to loaded for both.
	loadedCount := 0
	for _, call := range cpMgr.updateCalls {
		if call.status == model.LoadStatusLoaded {
			loadedCount++
		}
	}
	if loadedCount != 2 {
		t.Errorf("expected 2 loaded status updates, got %d", loadedCount)
	}

	// Verify row counts were set.
	if len(cpMgr.setRowCountCalls) != 2 {
		t.Errorf("expected 2 row count sets, got %d", len(cpMgr.setRowCountCalls))
	}

	// Active load ID should be cleared after completion.
	if id := orch.GetActiveLoadID(); id != "" {
		t.Errorf("expected empty active load ID after completion, got %s", id)
	}
}

func TestRunLoad_ConcurrentLoadRejected(t *testing.T) {
	tmpDir := t.TempDir()
	zipPath := createOrchestratorTestZip(t, tmpDir)

	// Slow server to keep load running longer.
	started := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-started:
		default:
			close(started)
		}
		http.ServeFile(w, r, zipPath)
	}))
	defer server.Close()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	downloader := NewDownloader(t.TempDir(), 0)
	cpMgr := newMockCheckpointManager()
	bulkLoader := newMockBulkLoader()
	cfg := testDatasetsCfg(server.URL)

	orch := NewOrchestrator(logger, downloader, bulkLoader, cpMgr, cfg)

	ctx := context.Background()
	done := make(chan error, 1)
	go func() {
		_, err := orch.RunLoad(ctx, nil, false, "")
		done <- err
	}()

	// Wait for the goroutine to start its load.
	<-started

	// Try to start a concurrent load.
	_, err := orch.RunLoad(ctx, nil, false, "")
	if err == nil {
		t.Fatal("expected error for concurrent load")
	}
	if !strings.Contains(err.Error(), "load already in progress") {
		t.Errorf("unexpected error: %v", err)
	}

	// Wait for the first load to finish.
	<-done

	// After the first completes, a new one should succeed.
	_, err = orch.RunLoad(ctx, nil, false, "")
	if err != nil {
		t.Fatalf("load after first completed should work: %v", err)
	}
}

func TestRunLoad_ResumeFromCheckpoint(t *testing.T) {
	tmpDir := t.TempDir()
	zipPath := createOrchestratorTestZip(t, tmpDir)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, zipPath)
	}))
	defer server.Close()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	downloader := NewDownloader(t.TempDir(), 1)
	cpMgr := newMockCheckpointManager()
	bulkLoader := newMockBulkLoader()
	cfg := testDatasetsCfg(server.URL)

	// Pre-populate: products is already loaded.
	resumeID := "resume-load-123"
	cpMgr.checkpoints = append(cpMgr.checkpoints, model.LoadCheckpoint{
		LoadID:    resumeID,
		Dataset:   "ndc_directory",
		TableName: "products",
		Status:    model.LoadStatusLoaded,
	})

	orch := NewOrchestrator(logger, downloader, bulkLoader, cpMgr, cfg)

	ctx := context.Background()
	loadID, err := orch.RunLoad(ctx, nil, false, resumeID)
	if err != nil {
		t.Fatalf("RunLoad failed: %v", err)
	}
	if loadID != resumeID {
		t.Errorf("expected resume load ID %s, got %s", resumeID, loadID)
	}

	// Only packages should be loaded (products was already done).
	if len(bulkLoader.bulkLoadCalls) != 1 {
		t.Fatalf("expected 1 bulk load call (packages only), got %d", len(bulkLoader.bulkLoadCalls))
	}
	if bulkLoader.bulkLoadCalls[0].tableName != "packages" {
		t.Errorf("expected bulk load for packages, got %s", bulkLoader.bulkLoadCalls[0].tableName)
	}
}

func TestLoadDataset_DownloadFailure(t *testing.T) {
	// Server that returns 500.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	downloader := NewDownloader(t.TempDir(), 0) // 0 retries for fast test
	cpMgr := newMockCheckpointManager()
	bulkLoader := newMockBulkLoader()
	cfg := testDatasetsCfg(server.URL)

	orch := NewOrchestrator(logger, downloader, bulkLoader, cpMgr, cfg)

	ctx := context.Background()
	loadID, err := orch.RunLoad(ctx, nil, false, "")

	// RunLoad itself does not return errors from individual datasets.
	if err != nil {
		t.Fatalf("RunLoad should not fail even if datasets fail: %v", err)
	}
	if loadID == "" {
		t.Fatal("expected non-empty load ID")
	}

	// No bulk loads should have happened.
	if len(bulkLoader.bulkLoadCalls) != 0 {
		t.Errorf("expected 0 bulk load calls, got %d", len(bulkLoader.bulkLoadCalls))
	}
}

func TestLoadFile_RowCountSafetyValveTriggered(t *testing.T) {
	tmpDir := t.TempDir()
	zipPath := createOrchestratorTestZip(t, tmpDir)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, zipPath)
	}))
	defer server.Close()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	downloader := NewDownloader(t.TempDir(), 1)
	cpMgr := newMockCheckpointManager()
	bulkLoader := newMockBulkLoader()
	cfg := testDatasetsCfg(server.URL)

	// Set up previous row counts — old data had 1000 rows, new data has ~1 row.
	// This should trigger the safety valve.
	cpMgr.previousCounts["products"] = 1000
	cpMgr.previousCounts["packages"] = 1000
	bulkLoader.safetyErr = fmt.Errorf("row count dropped 99.9%% (from 1000 to 1), exceeds threshold")

	orch := NewOrchestrator(logger, downloader, bulkLoader, cpMgr, cfg)

	ctx := context.Background()
	loadID, err := orch.RunLoad(ctx, nil, false, "")
	if err != nil {
		t.Fatalf("RunLoad should not return error: %v", err)
	}
	if loadID == "" {
		t.Fatal("expected non-empty load ID")
	}

	// Both tables should have had errors set.
	if len(cpMgr.setErrorCalls) != 2 {
		t.Errorf("expected 2 error calls (safety valve for each table), got %d", len(cpMgr.setErrorCalls))
	}

	// No bulk loads should have succeeded.
	if len(bulkLoader.bulkLoadCalls) != 0 {
		t.Errorf("expected 0 bulk load calls (safety blocked them), got %d", len(bulkLoader.bulkLoadCalls))
	}
}

func TestLoadFile_ParseError(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a ZIP with an invalid file (not proper tab-delimited).
	zipPath := filepath.Join(tmpDir, "ndc.zip")
	f, err := os.Create(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	w := zip.NewWriter(f)
	// Write an empty product.txt (will parse but have no rows/headers for mapping).
	fw, _ := w.Create("product.txt")
	fw.Write([]byte(""))
	fw2, _ := w.Create("package.txt")
	fw2.Write([]byte(""))
	w.Close()
	f.Close()

	server := httptest.NewServer(http.HandlerFunc(func(wr http.ResponseWriter, r *http.Request) {
		http.ServeFile(wr, r, zipPath)
	}))
	defer server.Close()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	downloader := NewDownloader(t.TempDir(), 1)
	cpMgr := newMockCheckpointManager()
	bulkLoader := newMockBulkLoader()
	cfg := testDatasetsCfg(server.URL)

	orch := NewOrchestrator(logger, downloader, bulkLoader, cpMgr, cfg)

	ctx := context.Background()
	loadID, err := orch.RunLoad(ctx, nil, false, "")
	if err != nil {
		t.Fatalf("RunLoad should not return error: %v", err)
	}
	if loadID == "" {
		t.Fatal("expected non-empty load ID")
	}

	// Both files should have errors (empty files fail on column mapping).
	if len(cpMgr.setErrorCalls) != 2 {
		t.Errorf("expected 2 error calls for parse failures, got %d", len(cpMgr.setErrorCalls))
	}

	// No bulk loads should have been attempted.
	if len(bulkLoader.bulkLoadCalls) != 0 {
		t.Errorf("expected 0 bulk load calls, got %d", len(bulkLoader.bulkLoadCalls))
	}
}

func TestGetActiveLoadID(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	downloader := NewDownloader(t.TempDir(), 1)
	cfg := &model.DatasetsConfig{}

	orch := NewOrchestrator(logger, downloader, nil, nil, cfg)

	// No active load.
	if id := orch.GetActiveLoadID(); id != "" {
		t.Errorf("expected empty active load ID, got %s", id)
	}
}

func TestFilterDatasets_Unit(t *testing.T) {
	datasets := []model.DatasetConfig{
		{Name: "ndc_directory", Enabled: true},
		{Name: "drugsfda", Enabled: true},
		{Name: "other", Enabled: true},
	}

	// Filter to specific datasets.
	filtered := filterDatasets(datasets, []string{"ndc_directory", "drugsfda"})
	if len(filtered) != 2 {
		t.Errorf("expected 2 filtered datasets, got %d", len(filtered))
	}

	// Filter to one.
	filtered = filterDatasets(datasets, []string{"ndc_directory"})
	if len(filtered) != 1 {
		t.Errorf("expected 1 filtered dataset, got %d", len(filtered))
	}

	// Filter to nonexistent.
	filtered = filterDatasets(datasets, []string{"nonexistent"})
	if len(filtered) != 0 {
		t.Errorf("expected 0 filtered datasets, got %d", len(filtered))
	}

	// Nil filter returns empty.
	filtered = filterDatasets(datasets, nil)
	if len(filtered) != 0 {
		t.Errorf("expected 0 filtered datasets for nil names, got %d", len(filtered))
	}

	// Empty input.
	filtered = filterDatasets(nil, []string{"ndc_directory"})
	if len(filtered) != 0 {
		t.Errorf("expected 0 filtered datasets for nil datasets, got %d", len(filtered))
	}
}

func TestRunLoad_FilteredDatasets(t *testing.T) {
	tmpDir := t.TempDir()
	zipPath := createOrchestratorTestZip(t, tmpDir)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, zipPath)
	}))
	defer server.Close()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	downloader := NewDownloader(t.TempDir(), 1)
	cpMgr := newMockCheckpointManager()
	bulkLoader := newMockBulkLoader()

	cfg := &model.DatasetsConfig{
		Datasets: []model.DatasetConfig{
			{
				Name:      "ndc_directory",
				Enabled:   true,
				SourceURL: server.URL + "/ndc.zip",
				Format:    "zip",
				Files: []model.FileConfig{
					{Filename: "product.txt", Table: "products", Delimiter: "\t", HasHeader: true},
				},
			},
			{
				Name:      "drugsfda",
				Enabled:   true,
				SourceURL: server.URL + "/fda.zip",
				Format:    "zip",
				Files: []model.FileConfig{
					{Filename: "applications.txt", Table: "applications", Delimiter: "\t", HasHeader: true},
				},
			},
		},
	}

	orch := NewOrchestrator(logger, downloader, bulkLoader, cpMgr, cfg)

	// Only load ndc_directory.
	ctx := context.Background()
	_, err := orch.RunLoad(ctx, []string{"ndc_directory"}, false, "")
	if err != nil {
		t.Fatalf("RunLoad failed: %v", err)
	}

	// Only products should be loaded (from ndc_directory), not applications.
	for _, call := range bulkLoader.bulkLoadCalls {
		if call.tableName == "applications" {
			t.Error("applications should not have been loaded")
		}
	}
}

func TestRunLoad_ForceBypassesSafety(t *testing.T) {
	tmpDir := t.TempDir()
	zipPath := createOrchestratorTestZip(t, tmpDir)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, zipPath)
	}))
	defer server.Close()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	downloader := NewDownloader(t.TempDir(), 1)
	cpMgr := newMockCheckpointManager()
	bulkLoader := newMockBulkLoader()
	cfg := testDatasetsCfg(server.URL)

	// Set previous counts that would trigger safety check.
	cpMgr.previousCounts["products"] = 1000
	cpMgr.previousCounts["packages"] = 1000

	orch := NewOrchestrator(logger, downloader, bulkLoader, cpMgr, cfg)

	ctx := context.Background()
	_, err := orch.RunLoad(ctx, nil, true, "") // force=true
	if err != nil {
		t.Fatalf("RunLoad failed: %v", err)
	}

	// Safety check should NOT have been called because force=true.
	if len(bulkLoader.safetyCalls) != 0 {
		t.Errorf("expected 0 safety calls with force=true, got %d", len(bulkLoader.safetyCalls))
	}

	// Both tables should have been loaded.
	if len(bulkLoader.bulkLoadCalls) != 2 {
		t.Errorf("expected 2 bulk load calls with force, got %d", len(bulkLoader.bulkLoadCalls))
	}
}

func TestRunLoad_BulkLoadError(t *testing.T) {
	tmpDir := t.TempDir()
	zipPath := createOrchestratorTestZip(t, tmpDir)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, zipPath)
	}))
	defer server.Close()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	downloader := NewDownloader(t.TempDir(), 1)
	cpMgr := newMockCheckpointManager()
	bulkLoader := newMockBulkLoader()
	bulkLoader.bulkLoadErr = fmt.Errorf("database connection lost")
	cfg := testDatasetsCfg(server.URL)

	orch := NewOrchestrator(logger, downloader, bulkLoader, cpMgr, cfg)

	ctx := context.Background()
	_, err := orch.RunLoad(ctx, nil, true, "")
	if err != nil {
		t.Fatalf("RunLoad should not return error: %v", err)
	}

	// Errors should have been recorded for both tables.
	if len(cpMgr.setErrorCalls) != 2 {
		t.Errorf("expected 2 error calls, got %d", len(cpMgr.setErrorCalls))
	}
	for _, call := range cpMgr.setErrorCalls {
		if !strings.Contains(call.errMsg, "database connection lost") {
			t.Errorf("expected error message about connection, got %s", call.errMsg)
		}
	}
}

func TestRunLoad_ResumeGetLoadedTablesError(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	downloader := NewDownloader(t.TempDir(), 1)
	cpMgr := newMockCheckpointManager()
	cpMgr.getLoadedErr = fmt.Errorf("db error")
	bulkLoader := newMockBulkLoader()
	cfg := &model.DatasetsConfig{
		Datasets: []model.DatasetConfig{
			{Name: "ndc_directory", Enabled: true, SourceURL: "http://localhost/ndc.zip"},
		},
	}

	orch := NewOrchestrator(logger, downloader, bulkLoader, cpMgr, cfg)

	ctx := context.Background()
	_, err := orch.RunLoad(ctx, nil, false, "resume-123")
	if err == nil {
		t.Fatal("expected error when GetLoadedTables fails during resume")
	}
	if !strings.Contains(err.Error(), "getting loaded tables for resume") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestLoadFile_GetPreviousRowCountError(t *testing.T) {
	tmpDir := t.TempDir()
	zipPath := createOrchestratorTestZip(t, tmpDir)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, zipPath)
	}))
	defer server.Close()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	downloader := NewDownloader(t.TempDir(), 1)
	cpMgr := newMockCheckpointManager()
	cpMgr.getPrevCountErr = fmt.Errorf("db error getting prev count")
	bulkLoader := newMockBulkLoader()
	cfg := testDatasetsCfg(server.URL)

	orch := NewOrchestrator(logger, downloader, bulkLoader, cpMgr, cfg)

	ctx := context.Background()
	_, err := orch.RunLoad(ctx, nil, false, "")
	if err != nil {
		t.Fatalf("RunLoad should not fail: %v", err)
	}

	// Even with GetPreviousRowCount error, load should continue (logged as warning).
	if len(bulkLoader.bulkLoadCalls) != 2 {
		t.Errorf("expected 2 bulk load calls (should proceed despite prev count error), got %d", len(bulkLoader.bulkLoadCalls))
	}
}

func TestLoadDataset_CreateCheckpointError(t *testing.T) {
	tmpDir := t.TempDir()
	zipPath := createOrchestratorTestZip(t, tmpDir)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, zipPath)
	}))
	defer server.Close()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	downloader := NewDownloader(t.TempDir(), 1)
	cpMgr := newMockCheckpointManager()
	cpMgr.createErr = fmt.Errorf("db error creating checkpoint")
	bulkLoader := newMockBulkLoader()
	cfg := testDatasetsCfg(server.URL)

	orch := NewOrchestrator(logger, downloader, bulkLoader, cpMgr, cfg)

	ctx := context.Background()
	_, err := orch.RunLoad(ctx, nil, false, "")
	if err != nil {
		t.Fatalf("RunLoad should not fail: %v", err)
	}

	// Load should still proceed even if checkpoint creation fails.
	if len(bulkLoader.bulkLoadCalls) != 2 {
		t.Errorf("expected 2 bulk load calls despite checkpoint error, got %d", len(bulkLoader.bulkLoadCalls))
	}
}

func TestLoadFile_CheckpointUpdateErrors(t *testing.T) {
	tmpDir := t.TempDir()
	zipPath := createOrchestratorTestZip(t, tmpDir)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, zipPath)
	}))
	defer server.Close()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	downloader := NewDownloader(t.TempDir(), 1)
	cpMgr := newMockCheckpointManager()
	// Set errors on all the checkpoint update operations.
	cpMgr.updateStatusErr = fmt.Errorf("update status failed")
	cpMgr.setRowCountErr = fmt.Errorf("set row count failed")
	cpMgr.setPrevCountErr = fmt.Errorf("set prev count failed")
	// Also set previous counts to exercise the SetPreviousRowCount path.
	cpMgr.previousCounts["products"] = 50
	cpMgr.previousCounts["packages"] = 50
	bulkLoader := newMockBulkLoader()
	cfg := testDatasetsCfg(server.URL)

	orch := NewOrchestrator(logger, downloader, bulkLoader, cpMgr, cfg)

	ctx := context.Background()
	_, err := orch.RunLoad(ctx, nil, false, "")
	if err != nil {
		t.Fatalf("RunLoad should not fail due to checkpoint errors: %v", err)
	}

	// Load should still succeed despite checkpoint update errors.
	if len(bulkLoader.bulkLoadCalls) != 2 {
		t.Errorf("expected 2 bulk load calls, got %d", len(bulkLoader.bulkLoadCalls))
	}
}

func TestLoadFile_NonTabDelimiter(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a ZIP with a comma-delimited file.
	zipPath := filepath.Join(tmpDir, "ndc.zip")
	f, err := os.Create(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	w := zip.NewWriter(f)
	// products table with comma delimiter.
	productData := "PRODUCTID,PRODUCTNDC,PRODUCTTYPENAME,PROPRIETARYNAME,NONPROPRIETARYNAME,DOSAGEFORMNAME,ROUTENAME,STARTMARKETINGDATE,ENDMARKETINGDATE,MARKETINGCATEGORYNAME,APPLICATIONNUMBER,LABELERNAME,SUBSTANCENAME,ACTIVE_NUMERATOR_STRENGTH,ACTIVE_INGRED_UNIT,PHARM_CLASSES,DEASCHEDULE,NDC_EXCLUDE_FLAG,LISTING_RECORD_CERTIFIED_THROUGH\n" +
		"TEST001,0002-1433,HUMAN,TestDrug,TEST,TABLET,ORAL,19950301,,ANDA,ANDA076543,Lab,SUB,500,mg/1,Test,,N,20260301\n"
	fw, _ := w.Create("product.csv")
	fw.Write([]byte(productData))
	w.Close()
	f.Close()

	server := httptest.NewServer(http.HandlerFunc(func(wr http.ResponseWriter, r *http.Request) {
		http.ServeFile(wr, r, zipPath)
	}))
	defer server.Close()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	downloader := NewDownloader(t.TempDir(), 1)
	cpMgr := newMockCheckpointManager()
	bulkLoader := newMockBulkLoader()

	cfg := &model.DatasetsConfig{
		Datasets: []model.DatasetConfig{
			{
				Name:      "ndc_directory",
				Enabled:   true,
				SourceURL: server.URL + "/ndc.zip",
				Format:    "zip",
				Files: []model.FileConfig{
					{Filename: "product.csv", Table: "products", Delimiter: ",", HasHeader: true},
				},
			},
		},
	}

	orch := NewOrchestrator(logger, downloader, bulkLoader, cpMgr, cfg)
	ctx := context.Background()
	_, err = orch.RunLoad(ctx, nil, false, "")
	if err != nil {
		t.Fatalf("RunLoad failed: %v", err)
	}

	if len(bulkLoader.bulkLoadCalls) != 1 {
		t.Errorf("expected 1 bulk load call, got %d", len(bulkLoader.bulkLoadCalls))
	}
}

func TestRunLoad_NoEnabledDatasets(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	downloader := NewDownloader(t.TempDir(), 1)
	cpMgr := newMockCheckpointManager()
	bulkLoader := newMockBulkLoader()
	cfg := &model.DatasetsConfig{
		Datasets: []model.DatasetConfig{
			{Name: "ndc_directory", Enabled: false},
		},
	}

	orch := NewOrchestrator(logger, downloader, bulkLoader, cpMgr, cfg)

	ctx := context.Background()
	loadID, err := orch.RunLoad(ctx, nil, false, "")
	if err != nil {
		t.Fatalf("RunLoad failed: %v", err)
	}
	if loadID == "" {
		t.Fatal("expected non-empty load ID")
	}

	// Nothing should have been loaded.
	if len(bulkLoader.bulkLoadCalls) != 0 {
		t.Errorf("expected 0 bulk loads, got %d", len(bulkLoader.bulkLoadCalls))
	}
}
