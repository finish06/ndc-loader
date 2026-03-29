package api

import (
	"regexp"
	"strings"
)

// PharmClassification is the structured representation of FDA pharmacological classes.
type PharmClassification struct {
	EPC []string `json:"epc"` // Established Pharmacologic Class
	MoA []string `json:"moa"` // Mechanism of Action
	PE  []string `json:"pe"`  // Physiologic Effect
	CS  []string `json:"cs"`  // Chemical/Ingredient Structure
	Raw string   `json:"raw"` // Original semicolon/comma-delimited string
}

// classTypeRe matches "Some Class Name [EPC]" and captures name + type.
var classTypeRe = regexp.MustCompile(`^\s*(.+?)\s*\[([A-Za-z]+)\]\s*$`)

// ParsePharmClasses parses the FDA pharm_classes string into a structured classification.
// Input format: "Angiotensin Converting Enzyme Inhibitor [EPC], Angiotensin-converting Enzyme Inhibitors [MoA]"
func ParsePharmClasses(raw string) *PharmClassification {
	if raw == "" {
		return nil
	}

	pc := &PharmClassification{
		EPC: []string{},
		MoA: []string{},
		PE:  []string{},
		CS:  []string{},
		Raw: raw,
	}

	// Split on comma (FDA uses ", " as delimiter between classes).
	parts := strings.Split(raw, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		m := classTypeRe.FindStringSubmatch(part)
		if m == nil {
			continue
		}

		name := strings.TrimSpace(m[1])
		classType := strings.ToUpper(m[2])

		switch classType {
		case "EPC":
			pc.EPC = append(pc.EPC, name)
		case "MOA":
			pc.MoA = append(pc.MoA, name)
		case "PE":
			pc.PE = append(pc.PE, name)
		case "CS":
			pc.CS = append(pc.CS, name)
			// EXT and unknown types are captured in Raw but not categorized.
		}
	}

	return pc
}
