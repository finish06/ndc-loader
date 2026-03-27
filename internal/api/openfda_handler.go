package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"
)

// OpenFDAHandler handles the openFDA-compatible API endpoint.
type OpenFDAHandler struct {
	logger     *slog.Logger
	queryStore QueryProvider
}

// NewOpenFDAHandler creates a new OpenFDAHandler.
func NewOpenFDAHandler(logger *slog.Logger, queryStore QueryProvider) *OpenFDAHandler {
	return &OpenFDAHandler{
		logger:     logger,
		queryStore: queryStore,
	}
}

// HandleNDCJSON handles GET /api/openfda/ndc.json — mirrors the openFDA /drug/ndc.json endpoint.
func (h *OpenFDAHandler) HandleNDCJSON(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("search")
	if search == "" {
		h.writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Missing required parameter: search")
		return
	}

	limit := 1 // openFDA default is 1
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
			if limit > 1000 {
				limit = 1000
			}
		}
	}

	skip := 0
	if s := r.URL.Query().Get("skip"); s != "" {
		if parsed, err := strconv.Atoi(s); err == nil && parsed >= 0 {
			skip = parsed
		}
	}

	// Parse search syntax.
	clauses, err := ParseOpenFDASearch(search)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	// Build SQL query from clauses.
	whereClause, args := BuildSearchQuery(clauses)

	// Execute query.
	products, total, err := h.queryStore.OpenFDASearch(r.Context(), whereClause, args, limit, skip)
	if err != nil {
		h.logger.Error("openfda search failed", "search", search, "error", err)
		h.writeError(w, http.StatusInternalServerError, "SERVER_ERROR", "Search failed")
		return
	}

	if len(products) == 0 {
		h.writeError(w, http.StatusNotFound, "NOT_FOUND", "No matches found!")
		return
	}

	// Transform to openFDA format.
	results := make([]OpenFDAProduct, len(products))
	for i, p := range products {
		results[i] = TransformToOpenFDA(&p)
	}

	// Build response.
	resp := OpenFDAResponse{
		Meta: OpenFDAMeta{
			Disclaimer:  openFDADisclaimer,
			Terms:       openFDATerms,
			License:     openFDALicense,
			LastUpdated: time.Now().Format("2006-01-02"),
			Results: OpenFDAPagination{
				Skip:  skip,
				Limit: limit,
				Total: total,
			},
		},
		Results: results,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *OpenFDAHandler) writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(OpenFDAError{
		Error: OpenFDAErrorDetail{
			Code:    code,
			Message: message,
		},
	})
}
