package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/calebdunn/ndc-loader/internal/store"
)

// QueryProvider abstracts query operations for unit testing.
type QueryProvider interface {
	LookupByProductNDC(ctx context.Context, variants []string) (*store.ProductResult, error)
	LookupByPackageNDC(ctx context.Context, variants []string) (*store.ProductResult, string, error)
	SearchProducts(ctx context.Context, query string, limit, offset int) ([]store.SearchResult, int, error)
	GetPackagesByProductNDC(ctx context.Context, productNDC string) ([]store.PackageResult, error)
	GetStats(ctx context.Context) (*store.StatsResult, error)
	OpenFDASearch(ctx context.Context, whereClause string, args []interface{}, limit, skip int) ([]store.ProductResult, int, error)
}

// QueryHandler handles NDC query API endpoints.
type QueryHandler struct {
	logger     *slog.Logger
	queryStore QueryProvider
}

// NewQueryHandler creates a new QueryHandler.
func NewQueryHandler(logger *slog.Logger, queryStore QueryProvider) *QueryHandler {
	return &QueryHandler{
		logger:     logger,
		queryStore: queryStore,
	}
}

// LookupNDC handles GET /api/ndc/{ndc}.
func (h *QueryHandler) LookupNDC(w http.ResponseWriter, r *http.Request) {
	ndcInput := chi.URLParam(r, "ndc")

	parsed, err := ParseNDC(ndcInput)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error":   "invalid_ndc",
			"message": err.Error(),
		})
		return
	}

	var product *store.ProductResult

	if parsed.Type == NDCTypePackage {
		// 3-segment or 10-digit: look up by package NDC.
		packageVariants := NDCSearchVariants(ndcInput)
		var matchedNDC string
		product, matchedNDC, err = h.queryStore.LookupByPackageNDC(r.Context(), packageVariants)
		if err == nil {
			product.MatchedPackage = &matchedNDC
		}
	} else {
		// 2-segment: look up by product NDC.
		productVariants := NDCSearchVariants(ndcInput)
		product, err = h.queryStore.LookupByProductNDC(r.Context(), productVariants)
	}

	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error":   "not_found",
			"message": "NDC not found in directory",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(product)
}

// SearchNDC handles GET /api/ndc/search.
func (h *QueryHandler) SearchNDC(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error":   "missing_query",
			"message": "query parameter 'q' is required",
		})
		return
	}

	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	offset := 0
	if o := r.URL.Query().Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	results, total, err := h.queryStore.SearchProducts(r.Context(), query, limit, offset)
	if err != nil {
		h.logger.Error("search failed", "query", query, "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "search_failed"})
		return
	}

	if results == nil {
		results = []store.SearchResult{}
	}

	resp := map[string]interface{}{
		"query":   query,
		"total":   total,
		"limit":   limit,
		"offset":  offset,
		"results": results,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// ListPackages handles GET /api/ndc/{ndc}/packages.
func (h *QueryHandler) ListPackages(w http.ResponseWriter, r *http.Request) {
	ndcInput := chi.URLParam(r, "ndc")

	parsed, err := ParseNDC(ndcInput)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error":   "invalid_ndc",
			"message": err.Error(),
		})
		return
	}

	productVariants := NDCSearchVariants(parsed.ProductNDC)
	var packages []store.PackageResult
	for _, v := range productVariants {
		packages, err = h.queryStore.GetPackagesByProductNDC(r.Context(), v)
		if err == nil && len(packages) > 0 {
			break
		}
	}

	if packages == nil {
		packages = []store.PackageResult{}
	}

	resp := map[string]interface{}{
		"product_ndc": parsed.ProductNDC,
		"packages":    packages,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// GetStats handles GET /api/ndc/stats.
func (h *QueryHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.queryStore.GetStats(r.Context())
	if err != nil {
		h.logger.Error("stats query failed", "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "stats_failed"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(stats)
}
