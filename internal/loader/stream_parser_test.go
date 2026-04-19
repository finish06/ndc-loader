package loader

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestStreamParser_BasicParse(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "data.txt")
	content := "col_a\tcol_b\tcol_c\nval1\tval2\tval3\nval4\tval5\tval6\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	sp := NewStreamParser(path, '\t', true)

	targetColumns := []string{"col_a", "col_b", "col_c"}
	headerMapping := map[string]string{}

	var rows [][]interface{}
	result, err := sp.Parse(targetColumns, headerMapping, func(row []interface{}) error {
		rowCopy := make([]interface{}, len(row))
		copy(rowCopy, row)
		rows = append(rows, rowCopy)
		return nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.RowCount != 2 {
		t.Errorf("expected 2 rows, got %d", result.RowCount)
	}
	if result.Skipped != 0 {
		t.Errorf("expected 0 skipped, got %d", result.Skipped)
	}
	if len(result.Headers) != 3 {
		t.Errorf("expected 3 headers, got %d", len(result.Headers))
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows collected, got %d", len(rows))
	}
	if rows[0][0] != "val1" {
		t.Errorf("expected val1, got %v", rows[0][0])
	}
	if rows[1][2] != "val6" {
		t.Errorf("expected val6, got %v", rows[1][2])
	}
}

func TestStreamParser_WithHeaderMapping(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "data.txt")
	content := "PRODUCTNDC\tPROPRIETARYNAME\n0002-1433\tMetformin\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	sp := NewStreamParser(path, '\t', true)

	targetColumns := []string{"product_ndc", "proprietary_name"}
	headerMapping := map[string]string{
		"product_ndc":      "PRODUCTNDC",
		"proprietary_name": "PROPRIETARYNAME",
	}

	var rows [][]interface{}
	result, err := sp.Parse(targetColumns, headerMapping, func(row []interface{}) error {
		rowCopy := make([]interface{}, len(row))
		copy(rowCopy, row)
		rows = append(rows, rowCopy)
		return nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.RowCount != 1 {
		t.Errorf("expected 1 row, got %d", result.RowCount)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if rows[0][0] != "0002-1433" {
		t.Errorf("expected product_ndc 0002-1433, got %v", rows[0][0])
	}
	if rows[0][1] != "Metformin" {
		t.Errorf("expected proprietary_name Metformin, got %v", rows[0][1])
	}
}

func TestStreamParser_EmptyValues(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "data.txt")
	content := "col_a\tcol_b\nval1\t\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	sp := NewStreamParser(path, '\t', true)

	var rows [][]interface{}
	result, err := sp.Parse([]string{"col_a", "col_b"}, map[string]string{}, func(row []interface{}) error {
		rowCopy := make([]interface{}, len(row))
		copy(rowCopy, row)
		rows = append(rows, rowCopy)
		return nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.RowCount != 1 {
		t.Errorf("expected 1 row, got %d", result.RowCount)
	}
	if rows[0][0] != "val1" {
		t.Errorf("expected val1, got %v", rows[0][0])
	}
	// Empty value should be nil.
	if rows[0][1] != nil {
		t.Errorf("expected nil for empty value, got %v", rows[0][1])
	}
}

func TestStreamParser_FileNotFound(t *testing.T) {
	sp := NewStreamParser("/nonexistent/file.txt", '\t', true)
	_, err := sp.Parse([]string{"col"}, map[string]string{}, func(_ []interface{}) error { return nil })
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestStreamParser_NoMatchingColumns(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "data.txt")
	content := "col_a\tcol_b\nval1\tval2\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	sp := NewStreamParser(path, '\t', true)
	_, err := sp.Parse([]string{"nonexistent_col"}, map[string]string{}, func(_ []interface{}) error { return nil })
	if err == nil {
		t.Fatal("expected error for no matching columns")
	}
}

func TestStreamParser_RowFnError(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "data.txt")
	content := "col_a\nval1\nval2\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	sp := NewStreamParser(path, '\t', true)
	callCount := 0
	_, err := sp.Parse([]string{"col_a"}, map[string]string{}, func(_ []interface{}) error {
		callCount++
		if callCount == 2 {
			return fmt.Errorf("batch full")
		}
		return nil
	})
	if err == nil {
		t.Fatal("expected error from rowFn")
	}
}

func TestStreamParser_DateCoercion(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "data.txt")
	content := "marketing_start\tndc_exclude\n20260301\tY\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	sp := NewStreamParser(path, '\t', true)

	var rows [][]interface{}
	_, err := sp.Parse([]string{"marketing_start", "ndc_exclude"}, map[string]string{}, func(row []interface{}) error {
		rowCopy := make([]interface{}, len(row))
		copy(rowCopy, row)
		rows = append(rows, rowCopy)
		return nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if rows[0][0] == nil {
		t.Error("expected non-nil date for marketing_start")
	}
	if rows[0][1] != true {
		t.Errorf("expected true for ndc_exclude=Y, got %v", rows[0][1])
	}
}

func TestStreamParser_WithProductSample(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "product_sample.txt")
	sp := NewStreamParser(path, '\t', true)

	targetColumns := []string{
		"product_id", "product_ndc", "product_type", "proprietary_name",
	}
	headerMapping := headerMappings["products"]

	var count int
	result, err := sp.Parse(targetColumns, headerMapping, func(row []interface{}) error {
		count++
		return nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.RowCount != 3 {
		t.Errorf("expected 3 rows, got %d", result.RowCount)
	}
	if count != 3 {
		t.Errorf("expected rowFn called 3 times, got %d", count)
	}
}

func TestStreamParser_InvalidUTF8(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "data.txt")
	// Include invalid UTF-8 byte (0x92 = Windows-1252 right single quote).
	content := []byte("col_a\nval\x92ue\n")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	sp := NewStreamParser(path, '\t', true)

	var rows [][]interface{}
	result, err := sp.Parse([]string{"col_a"}, map[string]string{}, func(row []interface{}) error {
		rowCopy := make([]interface{}, len(row))
		copy(rowCopy, row)
		rows = append(rows, rowCopy)
		return nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.RowCount != 1 {
		t.Errorf("expected 1 row, got %d", result.RowCount)
	}
}
