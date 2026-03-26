package api

import (
	"log/slog"
	"net/http"

	"github.com/calebdunn/ndc-loader/internal/loader"
	"github.com/calebdunn/ndc-loader/internal/store"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// NewRouter creates the Chi router with all endpoints and middleware.
func NewRouter(
	logger *slog.Logger,
	apiKeys []string,
	orchestrator *loader.Orchestrator,
	checkpointStore *store.CheckpointStore,
) http.Handler {
	r := chi.NewRouter()

	// Global middleware.
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Compress(5))

	// Health endpoint (no auth required).
	r.Get("/health", healthHandler(checkpointStore))

	// All other routes require API key.
	r.Group(func(r chi.Router) {
		r.Use(APIKeyAuth(apiKeys))

		// Admin endpoints.
		handler := NewAdminHandler(logger, orchestrator, checkpointStore)
		r.Post("/api/admin/load", handler.TriggerLoad)
		r.Get("/api/admin/load/{loadID}", handler.GetLoadStatus)
	})

	return r
}
