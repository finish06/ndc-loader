package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

var serverStartTime = time.Now()

// DependencyCheck is the result of a single dependency health check.
type DependencyCheck struct {
	Name      string   `json:"name"`
	Status    string   `json:"status"`
	LatencyMs *float64 `json:"latency_ms"`
	Error     *string  `json:"error,omitempty"`
}

// HealthResponse is the full health check response.
type HealthResponse struct {
	Status       string            `json:"status"`
	Version      string            `json:"version"`
	Uptime       string            `json:"uptime"`
	StartTime    string            `json:"start_time"`
	DataAgeHours *float64          `json:"data_age_hours"`
	LastLoad     *string           `json:"last_load"`
	Dependencies []DependencyCheck `json:"dependencies"`
}

// newHealthHandler returns the comprehensive health endpoint handler.
//
//	@Summary		Health check
//	@Description	Returns service health status, uptime, dependency checks (postgres), and data freshness. No authentication required.
//	@Tags			Operations
//	@Produce		json
//	@Success		200	{object}	HealthResponse
//	@Router			/health [get]
func newHealthHandler(db *pgxpool.Pool, checkpointStore LastLoadInfoProvider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		status := "ok"
		uptime := time.Since(serverStartTime)

		// Check postgres.
		pgCheck := checkPostgres(r.Context(), db)

		if pgCheck.Status == "disconnected" {
			status = "error"
		}

		// Check data freshness.
		var dataAgeHours *float64
		var lastLoad *string

		if checkpointStore != nil {
			lastLoadTime, ageHours, err := checkpointStore.GetLastLoadInfo(r.Context())
			if err == nil && lastLoadTime != nil {
				dataAgeHours = &ageHours
				formatted := lastLoadTime.Format(time.RFC3339)
				lastLoad = &formatted

				if ageHours > 48 && status == "ok" {
					status = "degraded"
				}
			} else if status == "ok" {
				// No data loaded yet.
				status = "degraded"
			}
		}

		resp := HealthResponse{
			Status:       status,
			Version:      BuildInfo.Version,
			Uptime:       uptime.Round(time.Second).String(),
			StartTime:    serverStartTime.UTC().Format(time.RFC3339),
			DataAgeHours: dataAgeHours,
			LastLoad:     lastLoad,
			Dependencies: []DependencyCheck{pgCheck},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}
}

func checkPostgres(ctx context.Context, db *pgxpool.Pool) DependencyCheck {
	if db == nil {
		errMsg := "database pool not configured"
		return DependencyCheck{
			Name:   "postgres",
			Status: "disconnected",
			Error:  &errMsg,
		}
	}

	start := time.Now()
	err := db.Ping(ctx)
	latency := float64(time.Since(start).Microseconds()) / 1000.0

	if err != nil {
		errMsg := err.Error()
		return DependencyCheck{
			Name:      "postgres",
			Status:    "disconnected",
			LatencyMs: nil,
			Error:     &errMsg,
		}
	}

	return DependencyCheck{
		Name:      "postgres",
		Status:    "connected",
		LatencyMs: &latency,
	}
}
