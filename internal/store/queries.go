package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// QueryStore handles read queries for NDC and drug data.
type QueryStore struct {
	db *pgxpool.Pool
}

// NewQueryStore creates a new QueryStore.
func NewQueryStore(db *pgxpool.Pool) *QueryStore {
	return &QueryStore{db: db}
}

// ProductResult is the API response for a single product lookup.
type ProductResult struct {
	ProductNDC             string          `json:"product_ndc"`
	BrandName              *string         `json:"brand_name"`
	GenericName            *string         `json:"generic_name"`
	DosageForm             *string         `json:"dosage_form"`
	Route                  *string         `json:"route"`
	Manufacturer           *string         `json:"manufacturer"`
	ActiveIngredient       *string         `json:"active_ingredients"`
	Strength               *string         `json:"strength"`
	StrengthUnit           *string         `json:"strength_unit"`
	PharmClasses           *string         `json:"pharm_classes"`
	PharmClassesStructured interface{}     `json:"pharm_classes_structured,omitempty"`
	DEASchedule            *string         `json:"dea_schedule"`
	MarketingCategory      *string         `json:"marketing_category"`
	ApplicationNumber      *string         `json:"application_number"`
	Packages               []PackageResult `json:"packages"`
	MatchedPackage         *string         `json:"matched_package"`
}

// PackageResult is a single package in the API response.
type PackageResult struct {
	NDC         string  `json:"ndc"`
	Description *string `json:"description"`
	Sample      bool    `json:"sample"`
}

// SearchResult is a single result in a search response.
type SearchResult struct {
	ProductNDC   string  `json:"product_ndc"`
	BrandName    *string `json:"brand_name"`
	GenericName  *string `json:"generic_name"`
	DosageForm   *string `json:"dosage_form"`
	Manufacturer *string `json:"manufacturer"`
	Relevance    float64 `json:"relevance"`
}

// StatsResult is the response for the stats endpoint.
type StatsResult struct {
	Products     int      `json:"products"`
	Packages     int      `json:"packages"`
	Applications int      `json:"applications"`
	LastLoaded   *string  `json:"last_loaded"`
	LoadDuration *float64 `json:"load_duration_seconds"`
}

// LookupByProductNDC finds a product by its 2-segment product NDC.
// Tries each variant for unhyphenated input.
func (q *QueryStore) LookupByProductNDC(ctx context.Context, variants []string) (*ProductResult, error) {
	for _, ndc := range variants {
		var p ProductResult
		err := q.db.QueryRow(ctx, `
			SELECT product_ndc, proprietary_name, nonproprietary_name,
			       dosage_form, route, labeler_name, substance_name,
			       strength, strength_unit, pharm_classes, dea_schedule,
			       marketing_category, application_number
			FROM products
			WHERE product_ndc = $1
			LIMIT 1
		`, ndc).Scan(
			&p.ProductNDC, &p.BrandName, &p.GenericName,
			&p.DosageForm, &p.Route, &p.Manufacturer, &p.ActiveIngredient,
			&p.Strength, &p.StrengthUnit, &p.PharmClasses, &p.DEASchedule,
			&p.MarketingCategory, &p.ApplicationNumber,
		)
		if err != nil {
			continue // Try next variant.
		}

		// Load packages.
		packages, err := q.getPackages(ctx, p.ProductNDC)
		if err == nil {
			p.Packages = packages
		}
		return &p, nil
	}

	return nil, fmt.Errorf("product not found")
}

// LookupByPackageNDC finds a product by its 3-segment package NDC.
func (q *QueryStore) LookupByPackageNDC(ctx context.Context, variants []string) (*ProductResult, string, error) {
	for _, ndc := range variants {
		var productNDC string
		err := q.db.QueryRow(ctx, `
			SELECT product_ndc FROM packages WHERE ndc_package_code = $1 LIMIT 1
		`, ndc).Scan(&productNDC)
		if err != nil {
			continue
		}

		product, err := q.LookupByProductNDC(ctx, []string{productNDC})
		if err != nil {
			return nil, "", err
		}
		return product, ndc, nil
	}

	return nil, "", fmt.Errorf("package not found")
}

// SearchProducts performs full-text search across product names.
func (q *QueryStore) SearchProducts(ctx context.Context, query string, limit, offset int) ([]SearchResult, int, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	// Use plainto_tsquery for simple terms, prefix matching via :* suffix.
	tsQuery := query + ":*"

	var total int
	err := q.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM products
		WHERE search_vector @@ to_tsquery('english', $1)
	`, tsQuery).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("counting search results: %w", err)
	}

	rows, err := q.db.Query(ctx, `
		SELECT product_ndc, proprietary_name, nonproprietary_name,
		       dosage_form, labeler_name,
		       ts_rank(search_vector, to_tsquery('english', $1)) AS relevance
		FROM products
		WHERE search_vector @@ to_tsquery('english', $1)
		ORDER BY relevance DESC
		LIMIT $2 OFFSET $3
	`, tsQuery, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("searching products: %w", err)
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var r SearchResult
		if err := rows.Scan(
			&r.ProductNDC, &r.BrandName, &r.GenericName,
			&r.DosageForm, &r.Manufacturer, &r.Relevance,
		); err != nil {
			return nil, 0, fmt.Errorf("scanning search result: %w", err)
		}
		results = append(results, r)
	}

	return results, total, rows.Err()
}

// GetPackagesByProductNDC returns all packages for a product NDC.
func (q *QueryStore) GetPackagesByProductNDC(ctx context.Context, productNDC string) ([]PackageResult, error) {
	return q.getPackages(ctx, productNDC)
}

func (q *QueryStore) getPackages(ctx context.Context, productNDC string) ([]PackageResult, error) {
	rows, err := q.db.Query(ctx, `
		SELECT ndc_package_code, description, sample_package
		FROM packages
		WHERE product_ndc = $1
		ORDER BY ndc_package_code
	`, productNDC)
	if err != nil {
		return nil, fmt.Errorf("querying packages: %w", err)
	}
	defer rows.Close()

	var packages []PackageResult
	for rows.Next() {
		var p PackageResult
		if err := rows.Scan(&p.NDC, &p.Description, &p.Sample); err != nil {
			return nil, fmt.Errorf("scanning package: %w", err)
		}
		packages = append(packages, p)
	}

	return packages, rows.Err()
}

// OpenFDASearch performs a search using a pre-built WHERE clause and returns full product
// details with packages, suitable for transforming into the openFDA response format.
func (q *QueryStore) OpenFDASearch(ctx context.Context, whereClause string, args []interface{}, limit, skip int) ([]ProductResult, int, error) {
	if limit <= 0 || limit > 1000 {
		limit = 1
	}
	if skip < 0 {
		skip = 0
	}

	// Count total matches.
	countSQL := fmt.Sprintf("SELECT COUNT(*) FROM products WHERE %s", whereClause)
	var total int
	if err := q.db.QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("counting openfda results: %w", err)
	}

	// Fetch products.
	selectSQL := fmt.Sprintf(`
		SELECT product_id, product_ndc, proprietary_name, nonproprietary_name,
		       dosage_form, route, labeler_name, substance_name,
		       strength, strength_unit, pharm_classes, dea_schedule,
		       marketing_category, application_number
		FROM products
		WHERE %s
		ORDER BY product_ndc
		LIMIT %d OFFSET %d
	`, whereClause, limit, skip)

	rows, err := q.db.Query(ctx, selectSQL, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("openfda search query: %w", err)
	}
	defer rows.Close()

	var products []ProductResult
	var productNDCs []string
	for rows.Next() {
		var p ProductResult
		var productID string
		if err := rows.Scan(
			&productID, &p.ProductNDC, &p.BrandName, &p.GenericName,
			&p.DosageForm, &p.Route, &p.Manufacturer, &p.ActiveIngredient,
			&p.Strength, &p.StrengthUnit, &p.PharmClasses, &p.DEASchedule,
			&p.MarketingCategory, &p.ApplicationNumber,
		); err != nil {
			return nil, 0, fmt.Errorf("scanning openfda result: %w", err)
		}

		products = append(products, p)
		productNDCs = append(productNDCs, p.ProductNDC)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	// Batch-load packages for all products in a single query (avoids N+1).
	if len(productNDCs) > 0 {
		packageMap, err := q.getPackagesBatch(ctx, productNDCs)
		if err == nil {
			for i := range products {
				products[i].Packages = packageMap[products[i].ProductNDC]
			}
		}
	}

	return products, total, nil
}

// getPackagesBatch loads packages for multiple product NDCs in a single query.
func (q *QueryStore) getPackagesBatch(ctx context.Context, productNDCs []string) (map[string][]PackageResult, error) {
	rows, err := q.db.Query(ctx, `
		SELECT product_ndc, ndc_package_code, description, sample_package
		FROM packages
		WHERE product_ndc = ANY($1)
		ORDER BY product_ndc, ndc_package_code
	`, productNDCs)
	if err != nil {
		return nil, fmt.Errorf("batch querying packages: %w", err)
	}
	defer rows.Close()

	result := make(map[string][]PackageResult)
	for rows.Next() {
		var productNDC string
		var p PackageResult
		if err := rows.Scan(&productNDC, &p.NDC, &p.Description, &p.Sample); err != nil {
			return nil, fmt.Errorf("scanning batch package: %w", err)
		}
		result[productNDC] = append(result[productNDC], p)
	}

	return result, rows.Err()
}

// GetStats returns dataset statistics.
func (q *QueryStore) GetStats(ctx context.Context) (*StatsResult, error) {
	var s StatsResult

	_ = q.db.QueryRow(ctx, "SELECT COUNT(*) FROM products").Scan(&s.Products)
	_ = q.db.QueryRow(ctx, "SELECT COUNT(*) FROM packages").Scan(&s.Packages)
	_ = q.db.QueryRow(ctx, "SELECT COUNT(*) FROM applications").Scan(&s.Applications)

	var lastLoaded *string
	_ = q.db.QueryRow(ctx, `
		SELECT TO_CHAR(completed_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"')
		FROM load_checkpoints
		WHERE status = 'loaded' AND completed_at IS NOT NULL
		ORDER BY completed_at DESC LIMIT 1
	`).Scan(&lastLoaded)
	s.LastLoaded = lastLoaded

	return &s, nil
}
