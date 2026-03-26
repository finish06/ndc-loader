package internal

import (
	"testing"
)

func TestSetupLogger_JSON(t *testing.T) {
	logger := SetupLogger("info", "json")
	if logger == nil {
		t.Fatal("expected non-nil logger")
	}
}

func TestSetupLogger_Text(t *testing.T) {
	logger := SetupLogger("debug", "text")
	if logger == nil {
		t.Fatal("expected non-nil logger")
	}
}

func TestSetupLogger_Levels(t *testing.T) {
	levels := []string{"debug", "info", "warn", "error", "unknown"}
	for _, level := range levels {
		logger := SetupLogger(level, "json")
		if logger == nil {
			t.Fatalf("expected non-nil logger for level %s", level)
		}
	}
}
