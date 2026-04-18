package api

import (
	"context"
	"testing"

	"github.com/calebdunn/ndc-loader/internal/store"
)

func TestCheckPostgres_NilPool(t *testing.T) {
	check := checkPostgres(context.Background(), nil)

	if check.Name != "postgres" {
		t.Errorf("expected name postgres, got %s", check.Name)
	}
	if check.Status != "disconnected" {
		t.Errorf("expected disconnected, got %s", check.Status)
	}
	if check.Error == nil {
		t.Error("expected error message for nil pool")
	}
	if check.LatencyMs != nil {
		t.Error("expected nil latency for nil pool")
	}
}

func TestCheckPostgres_NilPool_ErrorMessage(t *testing.T) {
	check := checkPostgres(context.Background(), nil)

	if *check.Error != "database pool not configured" {
		t.Errorf("expected 'database pool not configured', got %s", *check.Error)
	}
}

func TestEnrichProduct_NilProduct(t *testing.T) {
	// Should not panic.
	enrichProduct(nil)
}

func TestEnrichProduct_NilPharmClasses(t *testing.T) {
	p := &store.ProductResult{ProductNDC: "test"}
	enrichProduct(p)

	if p.PharmClassesStructured != nil {
		t.Error("expected nil pharm_classes_structured for nil pharm_classes")
	}
}

func TestEnrichProduct_EmptyPharmClasses(t *testing.T) {
	empty := ""
	p := &store.ProductResult{ProductNDC: "test", PharmClasses: &empty}
	enrichProduct(p)

	if p.PharmClassesStructured != nil {
		t.Error("expected nil pharm_classes_structured for empty string")
	}
}

func TestEnrichProduct_WithPharmClasses(t *testing.T) {
	classes := "Biguanide [EPC]"
	p := &store.ProductResult{ProductNDC: "test", PharmClasses: &classes}
	enrichProduct(p)

	if p.PharmClassesStructured == nil {
		t.Error("expected pharm_classes_structured to be set")
	}
}
