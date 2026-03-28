package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/calebdunn/ndc-loader/internal/loader"
	"github.com/calebdunn/ndc-loader/internal/model"
)

// CheckpointQuerier abstracts checkpoint query operations used by admin handlers.
type CheckpointQuerier interface {
	GetCheckpoints(ctx context.Context, loadID string) ([]model.LoadCheckpoint, error)
}

// LastLoadInfoProvider abstracts the health check data freshness query.
type LastLoadInfoProvider interface {
	GetLastLoadInfo(ctx context.Context) (*time.Time, float64, error)
}

// AdminHandler handles admin API endpoints.
type AdminHandler struct {
	logger          *slog.Logger
	orchestrator    *loader.Orchestrator
	checkpointStore CheckpointQuerier
}

// NewAdminHandler creates a new AdminHandler.
func NewAdminHandler(logger *slog.Logger, orchestrator *loader.Orchestrator, checkpointStore CheckpointQuerier) *AdminHandler {
	return &AdminHandler{
		logger:          logger,
		orchestrator:    orchestrator,
		checkpointStore: checkpointStore,
	}
}

// TriggerLoadRequest is the request body for POST /api/admin/load.
type TriggerLoadRequest struct {
	Datasets []string `json:"datasets"`
	Force    bool     `json:"force"`
}

// TriggerLoadResponse is the response for POST /api/admin/load.
type TriggerLoadResponse struct {
	LoadID   string   `json:"load_id"`
	Status   string   `json:"status"`
	Datasets []string `json:"datasets"`
}

// TriggerLoad handles POST /api/admin/load.
//
//	@Summary		Trigger a data load
//	@Description	Start a manual FDA data load. Returns 409 if a load is already in progress.
//	@Tags			Admin
//	@Accept			json
//	@Produce		json
//	@Param			body	body		TriggerLoadRequest	false	"Load options"
//	@Success		202		{object}	TriggerLoadResponse
//	@Failure		401		{object}	map[string]string
//	@Failure		409		{object}	map[string]string
//	@Security		ApiKeyAuth
//	@Router			/api/admin/load [post]
func (h *AdminHandler) TriggerLoad(w http.ResponseWriter, r *http.Request) {
	var req TriggerLoadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Default to all datasets if no body.
		req = TriggerLoadRequest{}
	}

	// Check for active load.
	if activeID := h.orchestrator.GetActiveLoadID(); activeID != "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error":   "load_in_progress",
			"load_id": activeID,
		})
		return
	}

	// Start load in background.
	var loadID string
	go func() {
		ctx := context.Background()
		var err error
		loadID, err = h.orchestrator.RunLoad(ctx, req.Datasets, req.Force, "")
		if err != nil {
			h.logger.Error("manual load failed", "load_id", loadID, "error", err)
		}
	}()

	// Give the goroutine a moment to set the load ID.
	time.Sleep(50 * time.Millisecond)

	resp := TriggerLoadResponse{
		LoadID:   h.orchestrator.GetActiveLoadID(),
		Status:   "started",
		Datasets: req.Datasets,
	}
	if resp.LoadID == "" {
		resp.LoadID = "pending"
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(resp)
}

// LoadStatusResponse is the response for GET /api/admin/load/{loadID}.
type LoadStatusResponse struct {
	LoadID      string             `json:"load_id"`
	Status      string             `json:"status"`
	StartedAt   *time.Time         `json:"started_at,omitempty"`
	Checkpoints []CheckpointStatus `json:"checkpoints"`
}

// CheckpointStatus is a checkpoint in the load status response.
type CheckpointStatus struct {
	Dataset         string   `json:"dataset"`
	Table           string   `json:"table"`
	Status          string   `json:"status"`
	RowCount        *int     `json:"row_count,omitempty"`
	DurationSeconds *float64 `json:"duration_seconds,omitempty"`
	Error           *string  `json:"error,omitempty"`
}

// GetLoadStatus handles GET /api/admin/load/{loadID}.
//
//	@Summary		Check load status
//	@Description	Returns checkpoint progress for a specific load operation.
//	@Tags			Admin
//	@Produce		json
//	@Param			loadID	path		string	true	"Load ID (UUID)"
//	@Success		200		{object}	LoadStatusResponse
//	@Failure		401		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Security		ApiKeyAuth
//	@Router			/api/admin/load/{loadID} [get]
func (h *AdminHandler) GetLoadStatus(w http.ResponseWriter, r *http.Request) {
	loadID := chi.URLParam(r, "loadID")

	checkpoints, err := h.checkpointStore.GetCheckpoints(r.Context(), loadID)
	if err != nil {
		h.logger.Error("failed to get checkpoints", "load_id", loadID, "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "internal_error"})
		return
	}

	if len(checkpoints) == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "load_not_found"})
		return
	}

	// Determine overall status.
	overallStatus := "complete"
	var startedAt *time.Time
	cpStatuses := make([]CheckpointStatus, 0, len(checkpoints))

	for _, cp := range checkpoints {
		if startedAt == nil && cp.StartedAt != nil {
			startedAt = cp.StartedAt
		}

		status := CheckpointStatus{
			Dataset: cp.Dataset,
			Table:   cp.TableName,
			Status:  string(cp.Status),
		}

		if cp.RowCount != nil {
			status.RowCount = cp.RowCount
		}
		if cp.ErrorMessage != nil {
			status.Error = cp.ErrorMessage
		}
		if cp.StartedAt != nil && cp.CompletedAt != nil {
			dur := cp.CompletedAt.Sub(*cp.StartedAt).Seconds()
			status.DurationSeconds = &dur
		}

		cpStatuses = append(cpStatuses, status)

		switch cp.Status {
		case "pending", "downloading", "downloaded", "loading":
			overallStatus = "in_progress"
		case "failed":
			overallStatus = "failed"
		}
	}

	resp := LoadStatusResponse{
		LoadID:      loadID,
		Status:      overallStatus,
		StartedAt:   startedAt,
		Checkpoints: cpStatuses,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// healthHandler returns the health check handler with data freshness info.
//
//	@Summary		Health check
//	@Description	Returns service health status and data freshness. Degrades to "degraded" when data is >48 hours old. No authentication required.
//	@Tags			Operations
//	@Produce		json
//	@Success		200	{object}	map[string]interface{}
//	@Router			/health [get]
func healthHandler(checkpointStore LastLoadInfoProvider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"status": "ok",
			"db":     "connected",
		}

		// Add data freshness if checkpoint store is available.
		if checkpointStore != nil {
			lastLoad, dataAgeHours, err := checkpointStore.GetLastLoadInfo(r.Context())
			if err == nil && lastLoad != nil {
				resp["last_load"] = lastLoad.Format(time.RFC3339)
				resp["data_age_hours"] = dataAgeHours

				if dataAgeHours > 48 {
					resp["status"] = "degraded"
				}
			}
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}
}
