package loader

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/robfig/cron/v3"
)

// Scheduler manages cron-based data load scheduling.
type Scheduler struct {
	logger       *slog.Logger
	cron         *cron.Cron
	orchestrator *Orchestrator
}

// NewScheduler creates a new Scheduler with the given cron schedule.
func NewScheduler(logger *slog.Logger, schedule string, orchestrator *Orchestrator) (*Scheduler, error) {
	c := cron.New()

	s := &Scheduler{
		logger:       logger,
		cron:         c,
		orchestrator: orchestrator,
	}

	_, err := c.AddFunc(schedule, func() {
		logger.Info("scheduled load triggered")
		ctx := context.Background()
		loadID, err := orchestrator.RunLoad(ctx, nil, false, "")
		if err != nil {
			logger.Error("scheduled load failed", "error", err)
			return
		}
		logger.Info("scheduled load complete", "load_id", loadID)
	})
	if err != nil {
		return nil, fmt.Errorf("invalid cron schedule %q: %w", schedule, err)
	}

	return s, nil
}

// Start begins the cron scheduler.
func (s *Scheduler) Start() {
	s.logger.Info("starting scheduler")
	s.cron.Start()
}

// Stop halts the cron scheduler.
func (s *Scheduler) Stop() {
	s.logger.Info("stopping scheduler")
	s.cron.Stop()
}
