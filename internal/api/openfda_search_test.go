package api

import (
	"testing"
)

func TestParseOpenFDASearch_FullText(t *testing.T) {
	clauses, err := ParseOpenFDASearch("metformin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(clauses) != 1 {
		t.Fatalf("expected 1 clause, got %d", len(clauses))
	}
	if clauses[0].Field != "" {
		t.Errorf("expected empty field for full-text, got %s", clauses[0].Field)
	}
	if clauses[0].Value != "metformin" {
		t.Errorf("expected value 'metformin', got %s", clauses[0].Value)
	}
}

func TestParseOpenFDASearch_FieldSpecific(t *testing.T) {
	clauses, err := ParseOpenFDASearch("brand_name:metformin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(clauses) != 1 {
		t.Fatalf("expected 1 clause, got %d", len(clauses))
	}
	if clauses[0].Field != "brand_name" {
		t.Errorf("expected field 'brand_name', got %s", clauses[0].Field)
	}
	if clauses[0].Value != "metformin" {
		t.Errorf("expected value 'metformin', got %s", clauses[0].Value)
	}
	if clauses[0].Exact {
		t.Error("expected non-exact match")
	}
}

func TestParseOpenFDASearch_ExactPhrase(t *testing.T) {
	clauses, err := ParseOpenFDASearch(`brand_name:"Metformin Hydrochloride"`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if clauses[0].Value != "Metformin Hydrochloride" {
		t.Errorf("expected 'Metformin Hydrochloride', got %s", clauses[0].Value)
	}
	if !clauses[0].Exact {
		t.Error("expected exact match for quoted value")
	}
}

func TestParseOpenFDASearch_ANDCombination(t *testing.T) {
	clauses, err := ParseOpenFDASearch("brand_name:metformin+generic_name:hydrochloride")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(clauses) != 2 {
		t.Fatalf("expected 2 clauses, got %d", len(clauses))
	}
	if clauses[0].Field != "brand_name" {
		t.Errorf("expected first field 'brand_name', got %s", clauses[0].Field)
	}
	if clauses[1].Field != "generic_name" {
		t.Errorf("expected second field 'generic_name', got %s", clauses[1].Field)
	}
}

func TestParseOpenFDASearch_Empty(t *testing.T) {
	_, err := ParseOpenFDASearch("")
	if err == nil {
		t.Fatal("expected error for empty search")
	}
}

func TestParseOpenFDASearch_ProductNDC(t *testing.T) {
	clauses, err := ParseOpenFDASearch(`product_ndc:"0002-1433"`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if clauses[0].Field != "product_ndc" {
		t.Errorf("expected field 'product_ndc', got %s", clauses[0].Field)
	}
	if clauses[0].Value != "0002-1433" {
		t.Errorf("expected value '0002-1433', got %s", clauses[0].Value)
	}
}

func TestMapFieldToColumn(t *testing.T) {
	tests := []struct {
		field    string
		expected string
	}{
		{"brand_name", "proprietary_name"},
		{"generic_name", "nonproprietary_name"},
		{"product_ndc", "product_ndc"},
		{"labeler_name", "labeler_name"},
		{"manufacturer_name", "labeler_name"},
		{"application_number", "application_number"},
		{"unknown_field", ""},
	}

	for _, tc := range tests {
		result := MapFieldToColumn(tc.field)
		if result != tc.expected {
			t.Errorf("MapFieldToColumn(%q) = %q, want %q", tc.field, result, tc.expected)
		}
	}
}

func TestBuildSearchQuery_FullText(t *testing.T) {
	clauses := []SearchClause{{Field: "", Value: "metformin"}}
	where, args := BuildSearchQuery(clauses)

	if where == "" {
		t.Fatal("expected non-empty WHERE clause")
	}
	if len(args) != 1 {
		t.Fatalf("expected 1 arg, got %d", len(args))
	}
	if args[0] != "metformin:*" {
		t.Errorf("expected 'metformin:*', got %v", args[0])
	}
}

func TestBuildSearchQuery_FieldExact(t *testing.T) {
	clauses := []SearchClause{{Field: "product_ndc", Value: "0002-1433"}}
	where, args := BuildSearchQuery(clauses)

	if where != "product_ndc = $1" {
		t.Errorf("expected 'product_ndc = $1', got %s", where)
	}
	if args[0] != "0002-1433" {
		t.Errorf("expected '0002-1433', got %v", args[0])
	}
}

func TestBuildSearchQuery_FieldILIKE(t *testing.T) {
	clauses := []SearchClause{{Field: "brand_name", Value: "metformin"}}
	where, args := BuildSearchQuery(clauses)

	if where != "proprietary_name ILIKE $1" {
		t.Errorf("expected 'proprietary_name ILIKE $1', got %s", where)
	}
	if args[0] != "%metformin%" {
		t.Errorf("expected '%%metformin%%', got %v", args[0])
	}
}

func TestBuildSearchQuery_AND(t *testing.T) {
	clauses := []SearchClause{
		{Field: "brand_name", Value: "metformin"},
		{Field: "generic_name", Value: "hydrochloride"},
	}
	where, args := BuildSearchQuery(clauses)

	if len(args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(args))
	}
	// Should contain AND.
	if where != "proprietary_name ILIKE $1 AND nonproprietary_name ILIKE $2" {
		t.Errorf("unexpected WHERE: %s", where)
	}
}
