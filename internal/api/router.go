package api

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	httpSwagger "github.com/swaggo/http-swagger/v2"

	"github.com/calebdunn/ndc-loader/internal/loader"
)

// CheckpointStoreProvider combines CheckpointQuerier and LastLoadInfoProvider
// for use by both admin handlers and the health endpoint.
type CheckpointStoreProvider interface {
	CheckpointQuerier
	LastLoadInfoProvider
}

// NewRouter creates the Chi router with all endpoints and middleware.
func NewRouter(
	logger *slog.Logger,
	apiKeys []string,
	orchestrator *loader.Orchestrator,
	checkpointStore CheckpointStoreProvider,
	queryStore QueryProvider,
	db *pgxpool.Pool,
	landingURL string,
) http.Handler {
	r := chi.NewRouter()

	// Global middleware.
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Compress(5))

	// Operations endpoints (no auth required).
	r.Get("/", rootRedirectHandler(landingURL))
	r.Get("/health", newHealthHandler(db, checkpointStore))
	r.Get("/version", versionHandler())
	r.Handle("/metrics", promhttp.Handler())
	r.Get("/swagger/*", httpSwagger.WrapHandler)

	// All other routes require API key.
	r.Group(func(r chi.Router) {
		r.Use(APIKeyAuth(apiKeys))

		// Admin endpoints.
		adminHandler := NewAdminHandler(logger, orchestrator, checkpointStore)
		r.Post("/api/admin/load", adminHandler.TriggerLoad)
		r.Get("/api/admin/load/{loadID}", adminHandler.GetLoadStatus)

		// Query endpoints.
		if queryStore != nil {
			queryHandler := NewQueryHandler(logger, queryStore)
			r.Get("/api/ndc/search", queryHandler.SearchNDC)
			r.Get("/api/ndc/stats", queryHandler.GetStats)
			r.Get("/api/ndc/{ndc}/packages", queryHandler.ListPackages)
			r.Get("/api/ndc/{ndc}", queryHandler.LookupNDC)

			// openFDA-compatible endpoint (drop-in replacement for drug-cash).
			openFDAHandler := NewOpenFDAHandler(logger, queryStore)
			r.Get("/api/openfda/ndc.json", openFDAHandler.HandleNDCJSON)
		}
	})

	return r
}
