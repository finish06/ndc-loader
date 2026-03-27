package loader

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"sync"

	"github.com/google/uuid"

	"github.com/calebdunn/ndc-loader/internal/model"
	"github.com/calebdunn/ndc-loader/internal/store"
)

// Orchestrator coordinates the full data load lifecycle:
// discovery -> download -> extract -> parse -> load -> checkpoint.
type Orchestrator struct {
	logger          *slog.Logger
	downloader      *Downloader
	dataLoader      BulkLoader
	checkpointStore CheckpointManager
	datasetsCfg     *model.DatasetsConfig
	mu              sync.Mutex
	activeLoadID    string
}

// NewOrchestrator creates a new Orchestrator.
func NewOrchestrator(
	logger *slog.Logger,
	downloader *Downloader,
	dataLoader BulkLoader,
	checkpointStore CheckpointManager,
	datasetsCfg *model.DatasetsConfig,
) *Orchestrator {
	return &Orchestrator{
		logger:          logger,
		downloader:      downloader,
		dataLoader:      dataLoader,
		checkpointStore: checkpointStore,
		datasetsCfg:     datasetsCfg,
	}
}

// headerMappings maps target DB column names to source file header names
// where they differ. Keys are "table.column" patterns.
var headerMappings = map[string]map[string]string{
	"products": {
		"product_id":              "PRODUCTID",
		"product_ndc":             "PRODUCTNDC",
		"product_type":            "PRODUCTTYPENAME",
		"proprietary_name":        "PROPRIETARYNAME",
		"proprietary_name_suffix": "PROPRIETARYNAMESUFFIX",
		"nonproprietary_name":     "NONPROPRIETARYNAME",
		"dosage_form":             "DOSAGEFORMNAME",
		"route":                   "ROUTENAME",
		"labeler_name":            "LABELERNAME",
		"substance_name":          "SUBSTANCENAME",
		"strength":                "ACTIVE_NUMERATOR_STRENGTH",
		"strength_unit":           "ACTIVE_INGRED_UNIT",
		"pharm_classes":           "PHARM_CLASSES",
		"dea_schedule":            "DEASCHEDULE",
		"marketing_category":      "MARKETINGCATEGORYNAME",
		"application_number":      "APPLICATIONNUMBER",
		"marketing_start":         "STARTMARKETINGDATE",
		"marketing_end":           "ENDMARKETINGDATE",
		"ndc_exclude":             "NDC_EXCLUDE_FLAG",
		"listing_certified":       "LISTING_RECORD_CERTIFIED_THROUGH",
	},
	"packages": {
		"product_id":       "PRODUCTID",
		"product_ndc":      "PRODUCTNDC",
		"ndc_package_code": "NDCPACKAGECODE",
		"description":      "PACKAGEDESCRIPTION",
		"marketing_start":  "STARTMARKETINGDATE",
		"marketing_end":    "ENDMARKETINGDATE",
		"ndc_exclude":      "NDC_EXCLUDE_FLAG",
		"sample_package":   "SAMPLE_PACKAGE",
	},
	"applications": {
		"appl_no":           "ApplNo",
		"appl_type":         "ApplType",
		"appl_public_notes": "ApplPublicNotes",
		"sponsor_name":      "SponsorName",
	},
	"drugsfda_products": {
		"appl_no":            "ApplNo",
		"product_no":         "ProductNo",
		"form":               "Form",
		"strength":           "Strength",
		"reference_drug":     "ReferenceDrug",
		"drug_name":          "DrugName",
		"active_ingredient":  "ActiveIngredient",
		"reference_standard": "ReferenceStandard",
	},
	"submissions": {
		"appl_no":                   "ApplNo",
		"submission_class_code_id":  "SubmissionClassCodeID",
		"submission_type":           "SubmissionType",
		"submission_no":             "SubmissionNo",
		"submission_status":         "SubmissionStatus",
		"submission_status_date":    "SubmissionStatusDate",
		"submissions_public_notes":  "SubmissionsPublicNotes",
		"review_priority":           "ReviewPriority",
	},
	"marketing_status": {
		"marketing_status_id": "MarketingStatusID",
		"appl_no":             "ApplNo",
		"product_no":          "ProductNo",
	},
	"te_codes": {
		"appl_no":             "ApplNo",
		"product_no":          "ProductNo",
		"marketing_status_id": "MarketingStatusID",
		"te_code":             "TECode",
	},
}

// RunLoad executes a full data load for all enabled datasets.
// If resumeLoadID is provided, it resumes a previous load from where it left off.
func (o *Orchestrator) RunLoad(ctx context.Context, datasetNames []string, force bool, resumeLoadID string) (string, error) {
	o.mu.Lock()
	if o.activeLoadID != "" {
		activeID := o.activeLoadID
		o.mu.Unlock()
		return "", fmt.Errorf("load already in progress: %s", activeID)
	}

	loadID := resumeLoadID
	if loadID == "" {
		loadID = uuid.New().String()
	}
	o.activeLoadID = loadID
	o.mu.Unlock()

	defer func() {
		o.mu.Lock()
		o.activeLoadID = ""
		o.mu.Unlock()
	}()

	o.logger.Info("starting data load", "load_id", loadID)

	datasets := EnabledDatasets(o.datasetsCfg)
	if len(datasetNames) > 0 {
		datasets = filterDatasets(datasets, datasetNames)
	}

	// Get previously loaded tables if resuming.
	var loadedTables map[string]bool
	if resumeLoadID != "" {
		var err error
		loadedTables, err = o.checkpointStore.GetLoadedTables(ctx, loadID)
		if err != nil {
			return loadID, fmt.Errorf("getting loaded tables for resume: %w", err)
		}
		o.logger.Info("resuming load", "load_id", loadID, "already_loaded", len(loadedTables))
	}

	for _, ds := range datasets {
		if err := o.loadDataset(ctx, loadID, ds, force, loadedTables); err != nil {
			o.logger.Error("dataset load failed", "dataset", ds.Name, "error", err)
			// Continue to next dataset — don't fail the entire load.
		}
	}

	o.logger.Info("data load complete", "load_id", loadID)
	return loadID, nil
}

func (o *Orchestrator) loadDataset(ctx context.Context, loadID string, ds model.DatasetConfig, force bool, loadedTables map[string]bool) error {
	o.logger.Info("loading dataset", "dataset", ds.Name, "url", ds.SourceURL)

	// Download.
	zipFilename := ds.Name + ".zip"
	zipPath, err := o.downloader.Download(ds.SourceURL, zipFilename)
	if err != nil {
		return fmt.Errorf("downloading %s: %w", ds.Name, err)
	}

	// Extract.
	extractDir, err := o.downloader.Extract(zipPath, ds.Name)
	if err != nil {
		return fmt.Errorf("extracting %s: %w", ds.Name, err)
	}
	defer o.downloader.Cleanup(ds.Name)

	// Process each file in the dataset.
	for _, fc := range ds.Files {
		tableName := fc.Table

		// Skip if already loaded (resume mode).
		if loadedTables != nil && loadedTables[tableName] {
			o.logger.Info("skipping already loaded table", "table", tableName)
			continue
		}

		// Create checkpoint.
		cp := &model.LoadCheckpoint{
			LoadID:    loadID,
			Dataset:   ds.Name,
			TableName: tableName,
			Status:    model.LoadStatusPending,
		}
		if err := o.checkpointStore.CreateCheckpoint(ctx, cp); err != nil {
			o.logger.Error("failed to create checkpoint", "table", tableName, "error", err)
		}

		if err := o.loadFile(ctx, loadID, extractDir, fc, ds.Name, force); err != nil {
			o.logger.Error("failed to load file",
				"dataset", ds.Name, "file", fc.Filename, "table", tableName, "error", err)
			if cpErr := o.checkpointStore.SetError(ctx, loadID, tableName, err.Error()); cpErr != nil {
				o.logger.Error("failed to set checkpoint error", "table", tableName, "error", cpErr)
			}
			// Continue to next file.
			continue
		}
	}

	return nil
}

func (o *Orchestrator) loadFile(ctx context.Context, loadID, extractDir string, fc model.FileConfig, datasetName string, force bool) error {
	tableName := fc.Table
	filePath := filepath.Join(extractDir, fc.Filename)

	// Update checkpoint: loading.
	if err := o.checkpointStore.UpdateStatus(ctx, loadID, tableName, model.LoadStatusLoading); err != nil {
		o.logger.Warn("failed to update checkpoint status", "table", tableName, "error", err)
	}

	// Parse the file.
	delimiter := '\t'
	if fc.Delimiter != "" && fc.Delimiter != "\\t" {
		delimiter = rune(fc.Delimiter[0])
	}

	parsed, err := ParseTabDelimited(filePath, delimiter, fc.HasHeader)
	if err != nil {
		return fmt.Errorf("parsing %s: %w", fc.Filename, err)
	}

	// Get target columns.
	columns, err := store.GetTableColumns(tableName)
	if err != nil {
		return fmt.Errorf("getting columns for %s: %w", tableName, err)
	}

	// Map columns.
	mapping := headerMappings[tableName]
	if mapping == nil {
		mapping = make(map[string]string)
	}

	rows, err := MapColumns(parsed, columns, mapping)
	if err != nil {
		return fmt.Errorf("mapping columns for %s: %w", tableName, err)
	}

	// Row count safety check.
	if !force {
		prevCount, err := o.checkpointStore.GetPreviousRowCount(ctx, tableName)
		if err != nil {
			o.logger.Warn("could not get previous row count", "table", tableName, "error", err)
		} else if prevCount > 0 {
			if err := o.dataLoader.CheckRowCountSafety(prevCount, len(rows)); err != nil {
				return fmt.Errorf("row count safety check failed for %s: %w", tableName, err)
			}
			if err := o.checkpointStore.SetPreviousRowCount(ctx, loadID, tableName, prevCount); err != nil {
				o.logger.Warn("failed to set previous row count", "table", tableName, "error", err)
			}
		}
	}

	// Bulk load.
	result, err := o.dataLoader.BulkLoad(ctx, tableName, columns, rows)
	if err != nil {
		return fmt.Errorf("bulk loading %s: %w", tableName, err)
	}

	// Update checkpoint: loaded.
	if err := o.checkpointStore.UpdateStatus(ctx, loadID, tableName, model.LoadStatusLoaded); err != nil {
		o.logger.Warn("failed to update checkpoint status to loaded", "table", tableName, "error", err)
	}
	if err := o.checkpointStore.SetRowCount(ctx, loadID, tableName, result.RowCount); err != nil {
		o.logger.Warn("failed to set row count", "table", tableName, "error", err)
	}

	o.logger.Info("table loaded successfully",
		"dataset", datasetName,
		"table", tableName,
		"rows", result.RowCount,
		"skipped", parsed.Skipped,
	)

	return nil
}

// GetActiveLoadID returns the currently active load ID, if any.
func (o *Orchestrator) GetActiveLoadID() string {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.activeLoadID
}

func filterDatasets(datasets []model.DatasetConfig, names []string) []model.DatasetConfig {
	nameSet := make(map[string]bool)
	for _, n := range names {
		nameSet[n] = true
	}
	var filtered []model.DatasetConfig
	for _, ds := range datasets {
		if nameSet[ds.Name] {
			filtered = append(filtered, ds)
		}
	}
	return filtered
}
