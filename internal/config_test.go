package internal

import (
	"os"
	"testing"
)

func TestLoadConfig_Defaults(t *testing.T) {
	// Set required API_KEYS and clear DATABASE_URL so default is used.
	t.Setenv("API_KEYS", "test-key-1,test-key-2")
	t.Setenv("DATABASE_URL", "")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.DatabaseURL != "postgres://ndc:ndc@localhost:5432/ndc?sslmode=disable" {
		t.Errorf("unexpected DatabaseURL: %s", cfg.DatabaseURL)
	}
	if cfg.ListenAddr != ":8081" {
		t.Errorf("unexpected ListenAddr: %s", cfg.ListenAddr)
	}
	if cfg.LoadSchedule != "0 3 * * *" {
		t.Errorf("unexpected LoadSchedule: %s", cfg.LoadSchedule)
	}
	if cfg.MaxRetryAttempts != 3 {
		t.Errorf("unexpected MaxRetryAttempts: %d", cfg.MaxRetryAttempts)
	}
	if cfg.RowCountDropThreshold != 0.20 {
		t.Errorf("unexpected RowCountDropThreshold: %f", cfg.RowCountDropThreshold)
	}
	if len(cfg.APIKeys) != 2 {
		t.Errorf("unexpected API key count: %d", len(cfg.APIKeys))
	}
	if cfg.APIKeys[0] != "test-key-1" {
		t.Errorf("unexpected first API key: %s", cfg.APIKeys[0])
	}
}

func TestLoadConfig_CustomValues(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://custom:custom@db:5432/custom")
	t.Setenv("LISTEN_ADDR", ":9090")
	t.Setenv("MAX_RETRY_ATTEMPTS", "5")
	t.Setenv("ROW_COUNT_DROP_THRESHOLD", "0.30")
	t.Setenv("API_KEYS", "key1")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.DatabaseURL != "postgres://custom:custom@db:5432/custom" {
		t.Errorf("unexpected DatabaseURL: %s", cfg.DatabaseURL)
	}
	if cfg.ListenAddr != ":9090" {
		t.Errorf("unexpected ListenAddr: %s", cfg.ListenAddr)
	}
	if cfg.MaxRetryAttempts != 5 {
		t.Errorf("unexpected MaxRetryAttempts: %d", cfg.MaxRetryAttempts)
	}
	if cfg.RowCountDropThreshold != 0.30 {
		t.Errorf("unexpected RowCountDropThreshold: %f", cfg.RowCountDropThreshold)
	}
}

func TestLoadConfig_MissingAPIKeys(t *testing.T) {
	os.Unsetenv("API_KEYS")

	_, err := LoadConfig()
	if err == nil {
		t.Fatal("expected error for missing API_KEYS")
	}
}

func TestLoadConfig_EmptyAPIKeys(t *testing.T) {
	t.Setenv("API_KEYS", ",,")

	_, err := LoadConfig()
	if err == nil {
		t.Fatal("expected error for empty API_KEYS")
	}
}

func TestLoadConfig_InvalidMaxRetry(t *testing.T) {
	t.Setenv("API_KEYS", "key1")
	t.Setenv("MAX_RETRY_ATTEMPTS", "not-a-number")

	_, err := LoadConfig()
	if err == nil {
		t.Fatal("expected error for invalid MAX_RETRY_ATTEMPTS")
	}
}

func TestLoadConfig_InvalidThreshold(t *testing.T) {
	t.Setenv("API_KEYS", "key1")
	t.Setenv("ROW_COUNT_DROP_THRESHOLD", "not-a-float")

	_, err := LoadConfig()
	if err == nil {
		t.Fatal("expected error for invalid ROW_COUNT_DROP_THRESHOLD")
	}
}
