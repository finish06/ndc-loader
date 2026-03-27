package api

import (
	"fmt"
	"regexp"
	"strings"
)

// SearchClause represents a single parsed search term from the openFDA search syntax.
type SearchClause struct {
	Field string // openFDA field name (empty for full-text)
	Value string // search value (quotes stripped)
	Exact bool   // true if value was quoted (exact phrase match)
}

// openFDA field name -> PostgreSQL column mapping.
var openFDAFieldMap = map[string]string{
	"brand_name":         "proprietary_name",
	"generic_name":       "nonproprietary_name",
	"product_ndc":        "product_ndc",
	"labeler_name":       "labeler_name",
	"manufacturer_name":  "labeler_name",
	"application_number": "application_number",
	"pharm_class":        "pharm_classes",
	"dosage_form":        "dosage_form",
	"route":              "route",
	"product_type":       "product_type",
	"marketing_category": "marketing_category",
	"dea_schedule":       "dea_schedule",
}

// fieldClauseRe matches `field_name:value` or `field_name:"quoted value"`.
var fieldClauseRe = regexp.MustCompile(`^([a-z_]+):(.+)$`)

// ParseOpenFDASearch parses an openFDA-style search string into structured clauses.
//
// Supported syntax:
//   - "metformin" → full-text search
//   - "brand_name:metformin" → field-specific search
//   - `brand_name:"Metformin Hydrochloride"` → exact phrase match
//   - "brand_name:metformin+generic_name:hydrochloride" → AND combination
func ParseOpenFDASearch(search string) ([]SearchClause, error) {
	search = strings.TrimSpace(search)
	if search == "" {
		return nil, fmt.Errorf("empty search query")
	}

	// Split on "+" for AND combinations.
	parts := strings.Split(search, "+")
	clauses := make([]SearchClause, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		clause, err := parseClause(part)
		if err != nil {
			return nil, err
		}
		clauses = append(clauses, clause)
	}

	if len(clauses) == 0 {
		return nil, fmt.Errorf("no valid search terms")
	}

	return clauses, nil
}

func parseClause(s string) (SearchClause, error) {
	m := fieldClauseRe.FindStringSubmatch(s)
	if m == nil {
		// No field prefix — full-text search.
		return SearchClause{
			Field: "",
			Value: stripQuotes(s),
			Exact: isQuoted(s),
		}, nil
	}

	field := m[1]
	value := m[2]

	return SearchClause{
		Field: field,
		Value: stripQuotes(value),
		Exact: isQuoted(value),
	}, nil
}

func stripQuotes(s string) string {
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}

func isQuoted(s string) bool {
	return len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"'
}

// MapFieldToColumn maps an openFDA field name to the corresponding PostgreSQL column.
// Returns empty string if the field is unknown (triggers full-text fallback).
func MapFieldToColumn(field string) string {
	if col, ok := openFDAFieldMap[field]; ok {
		return col
	}
	return ""
}

// BuildSearchQuery converts parsed search clauses into a SQL WHERE clause and arguments.
// Returns the WHERE clause (without "WHERE") and positional arguments.
func BuildSearchQuery(clauses []SearchClause) (whereClause string, args []interface{}) {
	var conditions []string
	argIdx := 1

	for _, c := range clauses {
		if c.Field == "" {
			// Full-text search via tsvector.
			conditions = append(conditions, fmt.Sprintf(
				"search_vector @@ to_tsquery('english', $%d)", argIdx,
			))
			args = append(args, c.Value+":*")
			argIdx++
			continue
		}

		col := MapFieldToColumn(c.Field)
		if col == "" {
			// Unknown field — fall back to full-text.
			conditions = append(conditions, fmt.Sprintf(
				"search_vector @@ to_tsquery('english', $%d)", argIdx,
			))
			args = append(args, c.Value+":*")
			argIdx++
			continue
		}

		switch {
		case c.Field == "product_ndc" || c.Field == "application_number":
			// Exact match fields.
			conditions = append(conditions, fmt.Sprintf("%s = $%d", col, argIdx))
			args = append(args, c.Value)
		case c.Exact:
			// Exact phrase — case-insensitive.
			conditions = append(conditions, fmt.Sprintf("%s ILIKE $%d", col, argIdx))
			args = append(args, c.Value)
		default:
			// Partial match — case-insensitive with wildcards.
			conditions = append(conditions, fmt.Sprintf("%s ILIKE $%d", col, argIdx))
			args = append(args, "%"+c.Value+"%")
		}
		argIdx++
	}

	if len(conditions) == 0 {
		return "TRUE", nil
	}

	return strings.Join(conditions, " AND "), args
}
