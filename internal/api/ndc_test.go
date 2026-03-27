package api

import (
	"testing"
)

func TestParseNDC_Hyphenated2Segment(t *testing.T) {
	result, err := ParseNDC("0002-1433")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Type != NDCTypeProduct {
		t.Errorf("expected product type, got %d", result.Type)
	}
	if result.ProductNDC != "0002-1433" {
		t.Errorf("expected product NDC 0002-1433, got %s", result.ProductNDC)
	}
}

func TestParseNDC_Hyphenated3Segment(t *testing.T) {
	result, err := ParseNDC("0002-1433-02")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Type != NDCTypePackage {
		t.Errorf("expected package type, got %d", result.Type)
	}
	if result.ProductNDC != "0002-1433" {
		t.Errorf("expected product NDC 0002-1433, got %s", result.ProductNDC)
	}
	if result.PackageNDC != "0002-1433-02" {
		t.Errorf("expected package NDC 0002-1433-02, got %s", result.PackageNDC)
	}
}

func TestParseNDC_Unhyphenated10Digit(t *testing.T) {
	result, err := ParseNDC("0002143302")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Type != NDCTypePackage {
		t.Errorf("expected package type, got %d", result.Type)
	}
}

func TestParseNDC_Unhyphenated8Digit(t *testing.T) {
	result, err := ParseNDC("00021433")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Type != NDCTypeProduct {
		t.Errorf("expected product type, got %d", result.Type)
	}
	if result.ProductNDC != "0002-1433" {
		t.Errorf("expected product NDC 0002-1433, got %s", result.ProductNDC)
	}
}

func TestParseNDC_Unhyphenated9Digit(t *testing.T) {
	result, err := ParseNDC("123456789")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Type != NDCTypeProduct {
		t.Errorf("expected product type, got %d", result.Type)
	}
	if result.ProductNDC != "12345-6789" {
		t.Errorf("expected product NDC 12345-6789, got %s", result.ProductNDC)
	}
}

func TestParseNDC_5Segment(t *testing.T) {
	result, err := ParseNDC("12345-678")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ProductNDC != "12345-678" {
		t.Errorf("expected product NDC 12345-678, got %s", result.ProductNDC)
	}
}

func TestParseNDC_Empty(t *testing.T) {
	_, err := ParseNDC("")
	if err == nil {
		t.Fatal("expected error for empty NDC")
	}
}

func TestParseNDC_Invalid(t *testing.T) {
	invalids := []string{"abc", "12", "1234567890123", "abc-def-gh"}
	for _, input := range invalids {
		_, err := ParseNDC(input)
		if err == nil {
			t.Errorf("expected error for invalid NDC %q", input)
		}
	}
}

func TestParseNDC_Whitespace(t *testing.T) {
	result, err := ParseNDC("  0002-1433  ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ProductNDC != "0002-1433" {
		t.Errorf("expected trimmed product NDC, got %s", result.ProductNDC)
	}
}

func TestNDCSearchVariants_10Digit(t *testing.T) {
	variants := NDCSearchVariants("0002143302")
	if len(variants) != 3 {
		t.Fatalf("expected 3 variants, got %d", len(variants))
	}
	// 4-4-2
	if variants[0] != "0002-1433-02" {
		t.Errorf("expected 4-4-2 variant 0002-1433-02, got %s", variants[0])
	}
	// 5-3-2
	if variants[1] != "00021-433-02" {
		t.Errorf("expected 5-3-2 variant 00021-433-02, got %s", variants[1])
	}
	// 5-4-1
	if variants[2] != "00021-4330-2" {
		t.Errorf("expected 5-4-1 variant 00021-4330-2, got %s", variants[2])
	}
}

func TestNDCSearchVariants_Hyphenated(t *testing.T) {
	variants := NDCSearchVariants("0002-1433")
	if len(variants) != 1 {
		t.Fatalf("expected 1 variant, got %d", len(variants))
	}
	if variants[0] != "0002-1433" {
		t.Errorf("expected 0002-1433, got %s", variants[0])
	}
}

func TestNDCSearchVariants_8Digit(t *testing.T) {
	variants := NDCSearchVariants("00021433")
	if len(variants) != 2 {
		t.Fatalf("expected 2 variants, got %d", len(variants))
	}
}
