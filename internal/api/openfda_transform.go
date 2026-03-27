package api

import (
	"strings"

	"github.com/calebdunn/ndc-loader/internal/store"
)

// TransformToOpenFDA converts an internal ProductResult to the openFDA response format.
func TransformToOpenFDA(p *store.ProductResult) OpenFDAProduct {
	product := OpenFDAProduct{
		ProductNDC:         p.ProductNDC,
		GenericName:        deref(p.GenericName),
		LabelerName:        deref(p.Manufacturer),
		BrandName:          deref(p.BrandName),
		Finished:           true,
		MarketingCategory:  deref(p.MarketingCategory),
		DosageForm:         deref(p.DosageForm),
		SPLID:              "",
		ProductType:        "HUMAN PRESCRIPTION DRUG",
		MarketingStartDate: "",
		ProductID:          "",
		ApplicationNumber:  deref(p.ApplicationNumber),
		BrandNameBase:      deref(p.BrandName),
	}

	product.ActiveIngredients = parseActiveIngredients(
		deref(p.ActiveIngredient),
		deref(p.Strength),
		deref(p.StrengthUnit),
	)
	product.Route = splitSemicolon(deref(p.Route))
	product.PharmClass = splitSemicolon(deref(p.PharmClasses))
	product.Packaging = transformPackages(p.Packages)
	product.OpenFDA = OpenFDANested{
		ManufacturerName:   wrapArray(deref(p.Manufacturer)),
		RXCUI:              []string{},
		SPLSetID:           []string{},
		IsOriginalPackager: []bool{},
		UPC:                []string{},
		UNII:               []string{},
	}

	return product
}

// parseActiveIngredients splits semicolon-delimited substance names and strengths
// into an array of {name, strength} objects matching the openFDA format.
func parseActiveIngredients(substanceName, strength, strengthUnit string) []OpenFDAActiveIngredient {
	if substanceName == "" {
		return []OpenFDAActiveIngredient{}
	}

	names := splitSemicolon(substanceName)
	strengths := splitSemicolon(strength)
	units := splitSemicolon(strengthUnit)

	ingredients := make([]OpenFDAActiveIngredient, len(names))
	for i, name := range names {
		ing := OpenFDAActiveIngredient{Name: strings.TrimSpace(name)}

		// Build "strength unit" string (e.g., "500 mg/1").
		var s, u string
		if i < len(strengths) {
			s = strings.TrimSpace(strengths[i])
		}
		if i < len(units) {
			u = strings.TrimSpace(units[i])
		}
		if s != "" && u != "" {
			ing.Strength = s + " " + u
		} else if s != "" {
			ing.Strength = s
		}

		ingredients[i] = ing
	}

	return ingredients
}

// transformPackages converts internal PackageResult slice to openFDA Packaging format.
func transformPackages(packages []store.PackageResult) []OpenFDAPackaging {
	if len(packages) == 0 {
		return []OpenFDAPackaging{}
	}

	result := make([]OpenFDAPackaging, len(packages))
	for i, p := range packages {
		result[i] = OpenFDAPackaging{
			PackageNDC:         p.NDC,
			Description:        deref(p.Description),
			MarketingStartDate: "", // We don't store per-package marketing_start in query results
			Sample:             p.Sample,
		}
	}
	return result
}

// splitSemicolon splits a semicolon-delimited string into a trimmed string slice.
// Returns an empty slice (not nil) if input is empty.
func splitSemicolon(s string) []string {
	if s == "" {
		return []string{}
	}

	// FDA uses both "; " and ";" as delimiters.
	parts := strings.Split(s, ";")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	if len(result) == 0 {
		return []string{}
	}
	return result
}

// wrapArray wraps a single string in a string slice. Returns empty slice if empty.
func wrapArray(s string) []string {
	if s == "" {
		return []string{}
	}
	return []string{s}
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
