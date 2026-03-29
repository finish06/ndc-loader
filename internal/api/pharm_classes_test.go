package api

import (
	"testing"
)

func TestParsePharmClasses_Lisinopril(t *testing.T) {
	raw := "Angiotensin Converting Enzyme Inhibitor [EPC], Angiotensin-converting Enzyme Inhibitors [MoA]"
	pc := ParsePharmClasses(raw)

	if pc == nil {
		t.Fatal("expected non-nil result")
	}
	if len(pc.EPC) != 1 || pc.EPC[0] != "Angiotensin Converting Enzyme Inhibitor" {
		t.Errorf("expected EPC=[Angiotensin Converting Enzyme Inhibitor], got %v", pc.EPC)
	}
	if len(pc.MoA) != 1 || pc.MoA[0] != "Angiotensin-converting Enzyme Inhibitors" {
		t.Errorf("expected MoA=[Angiotensin-converting Enzyme Inhibitors], got %v", pc.MoA)
	}
	if len(pc.PE) != 0 {
		t.Errorf("expected empty PE, got %v", pc.PE)
	}
	if pc.Raw != raw {
		t.Errorf("expected raw to match input")
	}
}

func TestParsePharmClasses_AllTypes(t *testing.T) {
	raw := "Drug Class [EPC], Mechanism [MoA], Effect [PE], Structure [CS]"
	pc := ParsePharmClasses(raw)

	if len(pc.EPC) != 1 || pc.EPC[0] != "Drug Class" {
		t.Errorf("EPC: got %v", pc.EPC)
	}
	if len(pc.MoA) != 1 || pc.MoA[0] != "Mechanism" {
		t.Errorf("MoA: got %v", pc.MoA)
	}
	if len(pc.PE) != 1 || pc.PE[0] != "Effect" {
		t.Errorf("PE: got %v", pc.PE)
	}
	if len(pc.CS) != 1 || pc.CS[0] != "Structure" {
		t.Errorf("CS: got %v", pc.CS)
	}
}

func TestParsePharmClasses_MultipleEPC(t *testing.T) {
	raw := "ACE Inhibitor [EPC], Calcium Channel Blocker [EPC], Some Mechanism [MoA]"
	pc := ParsePharmClasses(raw)

	if len(pc.EPC) != 2 {
		t.Fatalf("expected 2 EPCs, got %d: %v", len(pc.EPC), pc.EPC)
	}
	if pc.EPC[0] != "ACE Inhibitor" {
		t.Errorf("expected first EPC 'ACE Inhibitor', got %s", pc.EPC[0])
	}
	if pc.EPC[1] != "Calcium Channel Blocker" {
		t.Errorf("expected second EPC 'Calcium Channel Blocker', got %s", pc.EPC[1])
	}
}

func TestParsePharmClasses_Empty(t *testing.T) {
	pc := ParsePharmClasses("")
	if pc != nil {
		t.Error("expected nil for empty input")
	}
}

func TestParsePharmClasses_NoClassType(t *testing.T) {
	raw := "Just some text without brackets"
	pc := ParsePharmClasses(raw)

	if len(pc.EPC) != 0 {
		t.Errorf("expected empty EPC, got %v", pc.EPC)
	}
	if pc.Raw != raw {
		t.Error("expected raw to be preserved")
	}
}

func TestParsePharmClasses_Metformin(t *testing.T) {
	raw := "Biguanide [EPC], Biguanides [CS]"
	pc := ParsePharmClasses(raw)

	if len(pc.EPC) != 1 || pc.EPC[0] != "Biguanide" {
		t.Errorf("EPC: got %v", pc.EPC)
	}
	if len(pc.CS) != 1 || pc.CS[0] != "Biguanides" {
		t.Errorf("CS: got %v", pc.CS)
	}
}
