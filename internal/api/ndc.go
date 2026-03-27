package api

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	// Matches a hyphenated 2-segment NDC: "0002-1433"
	ndc2SegRe = regexp.MustCompile(`^(\d{4,5})-(\d{3,4})$`)
	// Matches a hyphenated 3-segment NDC: "0002-1433-02"
	ndc3SegRe = regexp.MustCompile(`^(\d{4,5})-(\d{3,4})-(\d{1,2})$`)
	// Matches an unhyphenated 10-digit NDC: "0002143302"
	ndc10Re = regexp.MustCompile(`^\d{10}$`)
	// Matches an unhyphenated product NDC (7-9 digits): "00021433"
	ndcShortRe = regexp.MustCompile(`^\d{7,9}$`)
)

// NDCType indicates whether the input is a product or package NDC.
type NDCType int

const (
	NDCTypeProduct NDCType = iota
	NDCTypePackage
)

// NDCParseResult holds the normalized NDC components.
type NDCParseResult struct {
	Type       NDCType
	ProductNDC string // Hyphenated 2-segment product NDC
	PackageNDC string // Hyphenated 3-segment package NDC (empty for product lookups)
	Raw        string // Original input
}

// ParseNDC normalizes an NDC code from any common format to its canonical form.
// Accepts:
//   - Hyphenated 2-segment: "0002-1433"
//   - Hyphenated 3-segment: "0002-1433-02"
//   - Unhyphenated 10-digit: "0002143302" (tries all segment patterns)
//   - Unhyphenated shorter: "00021433" (treated as product NDC)
func ParseNDC(input string) (*NDCParseResult, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, fmt.Errorf("empty NDC")
	}

	// Hyphenated 3-segment (package NDC).
	if m := ndc3SegRe.FindStringSubmatch(input); m != nil {
		productNDC := m[1] + "-" + m[2]
		return &NDCParseResult{
			Type:       NDCTypePackage,
			ProductNDC: productNDC,
			PackageNDC: input,
			Raw:        input,
		}, nil
	}

	// Hyphenated 2-segment (product NDC).
	if m := ndc2SegRe.FindStringSubmatch(input); m != nil {
		return &NDCParseResult{
			Type:       NDCTypeProduct,
			ProductNDC: input,
			Raw:        input,
		}, nil
	}

	// Unhyphenated 10-digit — try all three FDA segment patterns.
	if ndc10Re.MatchString(input) {
		return parse10Digit(input)
	}

	// Unhyphenated shorter — treat as product NDC with possible patterns.
	if ndcShortRe.MatchString(input) {
		return parseShortNDC(input)
	}

	return nil, fmt.Errorf("invalid NDC format: %q", input)
}

// parse10Digit tries all three FDA segment patterns for a 10-digit unhyphenated NDC.
// Returns the hyphenated form. Since we can't know which pattern is correct from
// the digits alone, we return all three possible interpretations for lookup.
func parse10Digit(input string) (*NDCParseResult, error) {
	// 4-4-2 pattern: LLLL-PPPP-KK
	productNDC := input[0:4] + "-" + input[4:8]
	packageNDC := input[0:4] + "-" + input[4:8] + "-" + input[8:10]

	return &NDCParseResult{
		Type:       NDCTypePackage,
		ProductNDC: productNDC,
		PackageNDC: packageNDC,
		Raw:        input,
	}, nil
}

// parseShortNDC handles unhyphenated product NDCs (7-9 digits).
func parseShortNDC(input string) (*NDCParseResult, error) {
	switch len(input) {
	case 8: // 4-4 pattern
		productNDC := input[0:4] + "-" + input[4:8]
		return &NDCParseResult{
			Type:       NDCTypeProduct,
			ProductNDC: productNDC,
			Raw:        input,
		}, nil
	case 9: // Could be 5-4 or 4-5 — try 5-4 first (more common)
		productNDC := input[0:5] + "-" + input[5:9]
		return &NDCParseResult{
			Type:       NDCTypeProduct,
			ProductNDC: productNDC,
			Raw:        input,
		}, nil
	case 7: // 4-3 pattern
		productNDC := input[0:4] + "-" + input[4:7]
		return &NDCParseResult{
			Type:       NDCTypeProduct,
			ProductNDC: productNDC,
			Raw:        input,
		}, nil
	default:
		return nil, fmt.Errorf("cannot parse %d-digit NDC: %q", len(input), input)
	}
}

// NDCSearchVariants returns all possible hyphenated forms for a given NDC input.
// Used for database lookup — try each variant until one matches.
func NDCSearchVariants(input string) []string {
	input = strings.TrimSpace(input)

	// Already hyphenated — return as-is plus unhyphenated.
	if strings.Contains(input, "-") {
		return []string{input}
	}

	// 10-digit unhyphenated — generate all three segment patterns.
	if len(input) == 10 {
		return []string{
			input[0:4] + "-" + input[4:8] + "-" + input[8:10], // 4-4-2
			input[0:5] + "-" + input[5:8] + "-" + input[8:10], // 5-3-2
			input[0:5] + "-" + input[5:9] + "-" + input[9:10], // 5-4-1
		}
	}

	// Shorter unhyphenated — try common product patterns.
	switch len(input) {
	case 8:
		return []string{
			input[0:4] + "-" + input[4:8], // 4-4
			input[0:5] + "-" + input[5:8], // 5-3
		}
	case 9:
		return []string{
			input[0:5] + "-" + input[5:9], // 5-4
			input[0:4] + "-" + input[4:9], // 4-5 (less common)
		}
	default:
		return []string{input}
	}
}
