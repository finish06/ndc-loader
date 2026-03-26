package loader

import (
	"log/slog"
	"testing"
)

func TestNewScheduler_ValidSchedule(t *testing.T) {
	logger := slog.Default()
	s, err := NewScheduler(logger, "0 3 * * *", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s == nil {
		t.Fatal("expected non-nil scheduler")
	}
}

func TestNewScheduler_InvalidSchedule(t *testing.T) {
	logger := slog.Default()
	_, err := NewScheduler(logger, "invalid cron", nil)
	if err == nil {
		t.Fatal("expected error for invalid cron schedule")
	}
}

func TestScheduler_StartStop(t *testing.T) {
	logger := slog.Default()
	s, err := NewScheduler(logger, "0 3 * * *", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should not panic.
	s.Start()
	s.Stop()
}
