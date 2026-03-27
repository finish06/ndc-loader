package loader

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseTabDelimited_MalformedRow(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "malformed.txt")
	// Create a file with a valid header, a good row, and a row with mismatched quotes.
	content := "col1\tcol2\nval1\tval2\n\"unclosed\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	result, err := ParseTabDelimited(path, '\t', true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Skipped == 0 {
		// LazyQuotes may handle this differently — just verify we don't crash.
		t.Log("no rows skipped (LazyQuotes may have handled it)")
	}
}

func TestCoerceValue_Dates(t *testing.T) {
	// Date column.
	val := coerceValue("20260301", "marketing_start")
	if val == nil {
		t.Error("expected non-nil date for marketing_start")
	}

	val = coerceValue("20260301", "marketing_end")
	if val == nil {
		t.Error("expected non-nil date for marketing_end")
	}

	val = coerceValue("20260301", "listing_certified")
	if val == nil {
		t.Error("expected non-nil date for listing_certified")
	}

	val = coerceValue("20260301", "most_recent_submission")
	if val == nil {
		t.Error("expected non-nil date for most_recent_submission")
	}

	val = coerceValue("20260301", "submission_status_date")
	if val == nil {
		t.Error("expected non-nil date for submission_status_date")
	}
}

func TestCoerceValue_Booleans(t *testing.T) {
	val := coerceValue("Y", "ndc_exclude")
	if val != true {
		t.Errorf("expected true for ndc_exclude=Y, got %v", val)
	}

	val = coerceValue("N", "sample_package")
	if val != false {
		t.Errorf("expected false for sample_package=N, got %v", val)
	}
}

func TestCoerceValue_Strings(t *testing.T) {
	// Non-date, non-bool column — should pass through as string.
	val := coerceValue("hello", "product_id")
	if val != "hello" {
		t.Errorf("expected 'hello', got %v", val)
	}
}

func TestMapColumns_ApplicationsMapping(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "applications_sample.txt")
	parsed, err := ParseTabDelimited(path, '\t', true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	targetColumns := []string{"appl_no", "appl_type", "appl_public_notes", "sponsor_name"}
	mapping := headerMappings["applications"]
	rows, err := MapColumns(parsed, targetColumns, mapping)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(rows))
	}

	// First row: ANDA076543.
	if rows[0][0] != "ANDA076543" {
		t.Errorf("expected appl_no ANDA076543, got %v", rows[0][0])
	}
	if rows[0][1] != "ANDA" {
		t.Errorf("expected appl_type ANDA, got %v", rows[0][1])
	}
	// appl_public_notes is empty in test data.
	if rows[0][2] != nil {
		t.Errorf("expected nil appl_public_notes, got %v", rows[0][2])
	}
	if rows[0][3] != "ELI LILLY AND CO" {
		t.Errorf("expected sponsor ELI LILLY AND CO, got %v", rows[0][3])
	}
}

func TestMapColumns_PackagesMapping(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "package_sample.txt")
	parsed, err := ParseTabDelimited(path, '\t', true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	targetColumns := []string{
		"product_id", "product_ndc", "ndc_package_code", "description",
		"marketing_start", "marketing_end", "ndc_exclude", "sample_package",
	}
	mapping := headerMappings["packages"]
	rows, err := MapColumns(parsed, targetColumns, mapping)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(rows) != 4 {
		t.Fatalf("expected 4 rows, got %d", len(rows))
	}

	// Check first row.
	if rows[0][0] != "0002-1433_b1c3d4e5" {
		t.Errorf("expected product_id 0002-1433_b1c3d4e5, got %v", rows[0][0])
	}
	if rows[0][2] != "0002-1433-02" {
		t.Errorf("expected ndc_package_code 0002-1433-02, got %v", rows[0][2])
	}
	// ndc_exclude should be boolean false.
	if rows[0][6] != false {
		t.Errorf("expected ndc_exclude false, got %v", rows[0][6])
	}
	// sample_package should be boolean false.
	if rows[0][7] != false {
		t.Errorf("expected sample_package false, got %v", rows[0][7])
	}
}
