package loader

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseTabDelimited_ProductSample(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "product_sample.txt")
	result, err := ParseTabDelimited(path, '\t', true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Headers) != 20 {
		t.Errorf("expected 20 headers, got %d: %v", len(result.Headers), result.Headers)
	}
	if result.Headers[0] != "PRODUCTID" {
		t.Errorf("expected first header PRODUCTID, got %s", result.Headers[0])
	}
	if len(result.Rows) != 3 {
		t.Errorf("expected 3 rows, got %d", len(result.Rows))
	}
	if result.Skipped != 0 {
		t.Errorf("expected 0 skipped, got %d", result.Skipped)
	}

	// Check first row values.
	if result.Rows[0][0] != "0002-1433_b1c3d4e5" {
		t.Errorf("expected product ID 0002-1433_b1c3d4e5, got %s", result.Rows[0][0])
	}
	if result.Rows[0][3] != "Metformin Hydrochloride" {
		t.Errorf("expected proprietary name Metformin Hydrochloride, got %s", result.Rows[0][3])
	}
}

func TestParseTabDelimited_PackageSample(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "package_sample.txt")
	result, err := ParseTabDelimited(path, '\t', true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Headers) != 8 {
		t.Errorf("expected 8 headers, got %d", len(result.Headers))
	}
	if len(result.Rows) != 4 {
		t.Errorf("expected 4 rows, got %d", len(result.Rows))
	}
}

func TestParseTabDelimited_ApplicationsSample(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "applications_sample.txt")
	result, err := ParseTabDelimited(path, '\t', true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Headers) != 4 {
		t.Errorf("expected 4 headers, got %d", len(result.Headers))
	}
	if len(result.Rows) != 3 {
		t.Errorf("expected 3 rows, got %d", len(result.Rows))
	}
}

func TestParseTabDelimited_FileNotFound(t *testing.T) {
	_, err := ParseTabDelimited("/nonexistent/file.txt", '\t', true)
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestParseTabDelimited_NoHeader(t *testing.T) {
	// Create a temp file without a header.
	tmp := t.TempDir()
	path := filepath.Join(tmp, "noheader.txt")
	content := "val1\tval2\tval3\nval4\tval5\tval6\n"
	os.WriteFile(path, []byte(content), 0o644)

	result, err := ParseTabDelimited(path, '\t', false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Headers) != 0 {
		t.Errorf("expected 0 headers, got %d", len(result.Headers))
	}
	if len(result.Rows) != 2 {
		t.Errorf("expected 2 rows, got %d", len(result.Rows))
	}
}

func TestMapColumns_ProductMapping(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "product_sample.txt")
	parsed, err := ParseTabDelimited(path, '\t', true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	targetColumns := []string{
		"product_id", "product_ndc", "product_type", "proprietary_name",
		"proprietary_name_suffix", "nonproprietary_name", "dosage_form", "route",
		"labeler_name", "substance_name", "strength", "strength_unit",
		"pharm_classes", "dea_schedule", "marketing_category", "application_number",
		"marketing_start", "marketing_end", "ndc_exclude", "listing_certified",
	}

	mapping := headerMappings["products"]
	rows, err := MapColumns(parsed, targetColumns, mapping)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(rows))
	}

	// Check first mapped row.
	row := rows[0]

	if row[0] != "0002-1433_b1c3d4e5" {
		t.Errorf("expected product_id 0002-1433_b1c3d4e5, got %v", row[0])
	}
	if row[1] != "0002-1433" {
		t.Errorf("expected product_ndc 0002-1433, got %v", row[1])
	}
	if row[3] != "Metformin Hydrochloride" {
		t.Errorf("expected proprietary_name Metformin Hydrochloride, got %v", row[3])
	}
	if row[15] != "ANDA076543" {
		t.Errorf("expected application_number ANDA076543, got %v", row[15])
	}

	// Check date coercion (marketing_start is a date column, index 16).
	if row[16] == nil {
		t.Error("expected marketing_start to be non-nil")
	}

	// Check boolean coercion (ndc_exclude is a bool column, index 18).
	if row[18] != false {
		t.Errorf("expected ndc_exclude false, got %v", row[18])
	}
}

func TestParseDate(t *testing.T) {
	tests := []struct {
		input string
		isNil bool
	}{
		{"20260301", false},
		{"2026-03-01", false},
		{"03/01/2026", false},
		{"invalid", true},
		{"", true}, // empty won't reach parseDate due to nil check in MapColumns
	}

	for _, tc := range tests {
		result := parseDate(tc.input)
		if tc.isNil && result != nil {
			t.Errorf("parseDate(%q) = %v, expected nil", tc.input, result)
		}
		if !tc.isNil && result == nil {
			t.Errorf("parseDate(%q) = nil, expected non-nil", tc.input)
		}
	}
}

func TestParseBool(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"Y", true},
		{"N", false},
		{"Yes", true},
		{"No", false},
		{"TRUE", true},
		{"FALSE", false},
		{"1", true},
		{"0", false},
		{"unknown", false},
	}

	for _, tc := range tests {
		result := parseBool(tc.input)
		if result != tc.want {
			t.Errorf("parseBool(%q) = %v, want %v", tc.input, result, tc.want)
		}
	}
}

func TestMapColumns_NoHeaders(t *testing.T) {
	parsed := &ParseResult{
		Rows: [][]string{{"a", "b"}},
	}
	_, err := MapColumns(parsed, []string{"col1", "col2"}, nil)
	if err == nil {
		t.Fatal("expected error for missing headers")
	}
}

func TestMapColumns_NoMatchingColumns(t *testing.T) {
	parsed := &ParseResult{
		Headers: []string{"foo", "bar"},
		Rows:    [][]string{{"a", "b"}},
	}
	_, err := MapColumns(parsed, []string{"baz", "qux"}, nil)
	if err == nil {
		t.Fatal("expected error for no matching columns")
	}
}
