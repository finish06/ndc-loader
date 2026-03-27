package store

import (
	"testing"
)

func TestCheckRowCountSafety_NoDropWithinThreshold(t *testing.T) {
	dl := &DataLoader{rowCountDropThreshold: 0.20}

	err := dl.CheckRowCountSafety(100, 90)
	if err != nil {
		t.Errorf("expected no error for 10%% drop, got: %v", err)
	}
}

func TestCheckRowCountSafety_DropExceedsThreshold(t *testing.T) {
	dl := &DataLoader{rowCountDropThreshold: 0.20}

	err := dl.CheckRowCountSafety(100, 70)
	if err == nil {
		t.Fatal("expected error for 30% drop")
	}
}

func TestCheckRowCountSafety_ExactThreshold(t *testing.T) {
	dl := &DataLoader{rowCountDropThreshold: 0.20}

	// 80% of 100 = exactly 20% drop.
	err := dl.CheckRowCountSafety(100, 80)
	if err != nil {
		t.Errorf("expected no error for exactly 20%% drop, got: %v", err)
	}
}

func TestCheckRowCountSafety_NoPreviousData(t *testing.T) {
	dl := &DataLoader{rowCountDropThreshold: 0.20}

	err := dl.CheckRowCountSafety(0, 100)
	if err != nil {
		t.Errorf("expected no error when no previous data, got: %v", err)
	}
}

func TestCheckRowCountSafety_RowCountIncrease(t *testing.T) {
	dl := &DataLoader{rowCountDropThreshold: 0.20}

	err := dl.CheckRowCountSafety(100, 150)
	if err != nil {
		t.Errorf("expected no error for row count increase, got: %v", err)
	}
}

func TestGetTableColumns_KnownTables(t *testing.T) {
	tables := []string{
		"products", "packages", "applications", "drugsfda_products",
		"submissions", "marketing_status", "te_codes",
	}

	for _, table := range tables {
		cols, err := GetTableColumns(table)
		if err != nil {
			t.Errorf("unexpected error for table %s: %v", table, err)
		}
		if len(cols) == 0 {
			t.Errorf("expected columns for table %s", table)
		}
	}
}

func TestGetTableColumns_UnknownTable(t *testing.T) {
	_, err := GetTableColumns("nonexistent")
	if err == nil {
		t.Fatal("expected error for unknown table")
	}
}
