package api

import (
	"encoding/json"
	"testing"

	"github.com/calebdunn/ndc-loader/internal/store"
)

func strPtr(s string) *string { return &s }

func TestTransformToOpenFDA_BasicProduct(t *testing.T) {
	p := &store.ProductResult{
		ProductNDC:        "0002-1433",
		BrandName:         strPtr("Metformin Hydrochloride"),
		GenericName:       strPtr("METFORMIN HYDROCHLORIDE"),
		DosageForm:        strPtr("TABLET"),
		Route:             strPtr("ORAL"),
		Manufacturer:      strPtr("Eli Lilly and Company"),
		ActiveIngredient:  strPtr("METFORMIN HYDROCHLORIDE"),
		Strength:          strPtr("500"),
		StrengthUnit:      strPtr("mg/1"),
		PharmClasses:      strPtr("Biguanide [EPC]; Biguanides [CS]"),
		MarketingCategory: strPtr("ANDA"),
		ApplicationNumber: strPtr("ANDA076543"),
		Packages: []store.PackageResult{
			{NDC: "0002-1433-61", Description: strPtr("100 TABLET in 1 BOTTLE"), Sample: false},
		},
	}

	result := TransformToOpenFDA(p)

	if result.ProductNDC != "0002-1433" {
		t.Errorf("expected product_ndc 0002-1433, got %s", result.ProductNDC)
	}
	if result.BrandName != "Metformin Hydrochloride" {
		t.Errorf("expected brand_name Metformin Hydrochloride, got %s", result.BrandName)
	}
	if result.GenericName != "METFORMIN HYDROCHLORIDE" {
		t.Errorf("expected generic_name METFORMIN HYDROCHLORIDE, got %s", result.GenericName)
	}
	if !result.Finished {
		t.Error("expected finished=true")
	}

	// Active ingredients.
	if len(result.ActiveIngredients) != 1 {
		t.Fatalf("expected 1 active ingredient, got %d", len(result.ActiveIngredients))
	}
	if result.ActiveIngredients[0].Name != "METFORMIN HYDROCHLORIDE" {
		t.Errorf("expected ingredient name METFORMIN HYDROCHLORIDE, got %s", result.ActiveIngredients[0].Name)
	}
	if result.ActiveIngredients[0].Strength != "500 mg/1" {
		t.Errorf("expected strength '500 mg/1', got %s", result.ActiveIngredients[0].Strength)
	}

	// Route as array.
	if len(result.Route) != 1 || result.Route[0] != "ORAL" {
		t.Errorf("expected route [ORAL], got %v", result.Route)
	}

	// PharmClass as array.
	if len(result.PharmClass) != 2 {
		t.Fatalf("expected 2 pharm_class entries, got %d", len(result.PharmClass))
	}
	if result.PharmClass[0] != "Biguanide [EPC]" {
		t.Errorf("expected first pharm_class 'Biguanide [EPC]', got %s", result.PharmClass[0])
	}

	// Packaging.
	if len(result.Packaging) != 1 {
		t.Fatalf("expected 1 package, got %d", len(result.Packaging))
	}
	if result.Packaging[0].PackageNDC != "0002-1433-61" {
		t.Errorf("expected package_ndc 0002-1433-61, got %s", result.Packaging[0].PackageNDC)
	}

	// OpenFDA nested.
	if len(result.OpenFDA.ManufacturerName) != 1 {
		t.Errorf("expected 1 manufacturer_name, got %d", len(result.OpenFDA.ManufacturerName))
	}
	if len(result.OpenFDA.RXCUI) != 0 {
		t.Errorf("expected empty rxcui, got %v", result.OpenFDA.RXCUI)
	}
}

func TestTransformToOpenFDA_MultipleIngredients(t *testing.T) {
	p := &store.ProductResult{
		ProductNDC:       "1234-5678",
		ActiveIngredient: strPtr("DRUG A; DRUG B; DRUG C"),
		Strength:         strPtr("100; 200; 300"),
		StrengthUnit:     strPtr("mg/1; mg/1; mg/1"),
	}

	result := TransformToOpenFDA(p)

	if len(result.ActiveIngredients) != 3 {
		t.Fatalf("expected 3 active ingredients, got %d", len(result.ActiveIngredients))
	}
	if result.ActiveIngredients[0].Name != "DRUG A" {
		t.Errorf("expected DRUG A, got %s", result.ActiveIngredients[0].Name)
	}
	if result.ActiveIngredients[1].Strength != "200 mg/1" {
		t.Errorf("expected '200 mg/1', got %s", result.ActiveIngredients[1].Strength)
	}
}

func TestTransformToOpenFDA_NilFields(t *testing.T) {
	p := &store.ProductResult{
		ProductNDC: "0000-0000",
	}

	result := TransformToOpenFDA(p)

	if result.GenericName != "" {
		t.Errorf("expected empty generic_name, got %s", result.GenericName)
	}
	if len(result.ActiveIngredients) != 0 {
		t.Errorf("expected empty active_ingredients, got %d", len(result.ActiveIngredients))
	}
	if len(result.Route) != 0 {
		t.Errorf("expected empty route, got %v", result.Route)
	}
	if len(result.Packaging) != 0 {
		t.Errorf("expected empty packaging, got %d", len(result.Packaging))
	}
}

func TestTransformToOpenFDA_JSONFieldNames(t *testing.T) {
	p := &store.ProductResult{
		ProductNDC:       "0002-1433",
		BrandName:        strPtr("Test"),
		ActiveIngredient: strPtr("DRUG"),
		Strength:         strPtr("500"),
		StrengthUnit:     strPtr("mg/1"),
	}

	result := TransformToOpenFDA(p)
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}

	// Verify key openFDA field names are present.
	jsonStr := string(data)
	requiredFields := []string{
		"product_ndc", "generic_name", "labeler_name", "brand_name",
		"active_ingredients", "finished", "packaging", "openfda",
		"marketing_category", "dosage_form", "spl_id", "product_type",
		"route", "marketing_start_date", "product_id", "application_number",
		"brand_name_base", "pharm_class",
	}
	for _, field := range requiredFields {
		if !contains(jsonStr, "\""+field+"\"") {
			t.Errorf("missing JSON field %q in output", field)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func TestSplitSemicolon(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"", 0},
		{"ORAL", 1},
		{"ORAL; TOPICAL", 2},
		{"A;B;C", 3},
		{"A; B; C", 3},
		{"; ; ", 0},
	}

	for _, tc := range tests {
		result := splitSemicolon(tc.input)
		if len(result) != tc.expected {
			t.Errorf("splitSemicolon(%q) = %d items, want %d", tc.input, len(result), tc.expected)
		}
	}
}

func TestParseActiveIngredients(t *testing.T) {
	result := parseActiveIngredients("DRUG A; DRUG B", "100; 200", "mg/1; mg/1")
	if len(result) != 2 {
		t.Fatalf("expected 2 ingredients, got %d", len(result))
	}
	if result[0].Strength != "100 mg/1" {
		t.Errorf("expected '100 mg/1', got %s", result[0].Strength)
	}

	// Empty input.
	result = parseActiveIngredients("", "", "")
	if len(result) != 0 {
		t.Errorf("expected 0 ingredients for empty input, got %d", len(result))
	}
}
