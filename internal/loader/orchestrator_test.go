package loader

import (
	"testing"
)

func TestFilterDatasets(t *testing.T) {
	datasets := []DatasetConfigForTest{
		{Name: "ndc_directory"},
		{Name: "drugsfda"},
		{Name: "other"},
	}

	// Convert to model.DatasetConfig for the test.
	var modelDatasets []modelDatasetConfig
	for _, d := range datasets {
		modelDatasets = append(modelDatasets, modelDatasetConfig{Name: d.Name, Enabled: true})
	}

	filtered := filterDatasetsHelper(modelDatasets, []string{"ndc_directory", "drugsfda"})
	if len(filtered) != 2 {
		t.Errorf("expected 2 filtered datasets, got %d", len(filtered))
	}

	filtered = filterDatasetsHelper(modelDatasets, []string{"ndc_directory"})
	if len(filtered) != 1 {
		t.Errorf("expected 1 filtered dataset, got %d", len(filtered))
	}

	filtered = filterDatasetsHelper(modelDatasets, []string{"nonexistent"})
	if len(filtered) != 0 {
		t.Errorf("expected 0 filtered datasets, got %d", len(filtered))
	}

	filtered = filterDatasetsHelper(modelDatasets, nil)
	if len(filtered) != 0 {
		t.Errorf("expected 0 filtered datasets for nil names, got %d", len(filtered))
	}
}

type DatasetConfigForTest struct {
	Name string
}

type modelDatasetConfig = struct {
	Name    string
	Enabled bool
}

func filterDatasetsHelper(datasets []modelDatasetConfig, names []string) []modelDatasetConfig {
	nameSet := make(map[string]bool)
	for _, n := range names {
		nameSet[n] = true
	}
	var filtered []modelDatasetConfig
	for _, ds := range datasets {
		if nameSet[ds.Name] {
			filtered = append(filtered, ds)
		}
	}
	return filtered
}

func TestHeaderMappings_AllTablesExist(t *testing.T) {
	expectedTables := []string{
		"products", "packages", "applications", "drugsfda_products",
		"submissions", "marketing_status", "active_ingredients", "te_codes",
	}

	for _, table := range expectedTables {
		mapping, ok := headerMappings[table]
		if !ok {
			t.Errorf("missing header mapping for table %s", table)
			continue
		}
		if len(mapping) == 0 {
			t.Errorf("empty header mapping for table %s", table)
		}
	}
}

func TestHeaderMappings_ProductsHasAllColumns(t *testing.T) {
	mapping := headerMappings["products"]

	expectedKeys := []string{
		"product_id", "product_ndc", "product_type", "proprietary_name",
		"nonproprietary_name", "dosage_form", "route", "labeler_name",
		"substance_name", "strength", "strength_unit", "pharm_classes",
		"dea_schedule", "marketing_category", "application_number",
		"marketing_start", "marketing_end", "ndc_exclude", "listing_certified",
	}

	for _, key := range expectedKeys {
		if _, ok := mapping[key]; !ok {
			t.Errorf("missing mapping for products column %q", key)
		}
	}
}

func TestHeaderMappings_PackagesHasAllColumns(t *testing.T) {
	mapping := headerMappings["packages"]

	expectedKeys := []string{
		"product_id", "product_ndc", "ndc_package_code", "description",
		"marketing_start", "marketing_end", "ndc_exclude", "sample_package",
	}

	for _, key := range expectedKeys {
		if _, ok := mapping[key]; !ok {
			t.Errorf("missing mapping for packages column %q", key)
		}
	}
}
