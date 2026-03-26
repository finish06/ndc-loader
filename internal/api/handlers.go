package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/calebdunn/ndc-loader/internal/loader"
	"github.com/calebdunn/ndc-loader/internal/store"
	"github.com/go-chi/chi/v5"
)

// AdminHandler handles admin API endpoints.
type AdminHandler struct {
	logger          *slog.Logger
	orchestrator    *loader.Orchestrator
	checkpointStore *store.CheckpointStore
}

// NewAdminHandler creates a new AdminHandler.
func NewAdminHandler(logger *slog.Logger, orchestrator *loader.Orchestrator, checkpointStore *store.CheckpointStore) *AdminHandler {
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
		json.NewEncoder(w).Encode(map[string]string{
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
	json.NewEncoder(w).Encode(resp)
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
	Dataset         string  `json:"dataset"`
	Table           string  `json:"table"`
	Status          string  `json:"status"`
	RowCount        *int    `json:"row_count,omitempty"`
	DurationSeconds *float64 `json:"duration_seconds,omitempty"`
	Error           *string `json:"error,omitempty"`
}

// GetLoadStatus handles GET /api/admin/load/{loadID}.
func (h *AdminHandler) GetLoadStatus(w http.ResponseWriter, r *http.Request) {
	loadID := chi.URLParam(r, "loadID")

	checkpoints, err := h.checkpointStore.GetCheckpoints(r.Context(), loadID)
	if err != nil {
		h.logger.Error("failed to get checkpoints", "load_id", loadID, "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "internal_error"})
		return
	}

	if len(checkpoints) == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "load_not_found"})
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
	json.NewEncoder(w).Encode(resp)
}

// healthHandler returns the health check handler.
func healthHandler(checkpointStore *store.CheckpointStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"status": "ok",
			"db":     "connected",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}
