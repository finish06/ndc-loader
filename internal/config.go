package internal

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/calebdunn/ndc-loader/internal/model"
)

// LoadConfig reads configuration from environment variables with defaults.
func LoadConfig() (*model.Config, error) {
	cfg := &model.Config{
		DatabaseURL:     envOrDefault("DATABASE_URL", "postgres://ndc:ndc@localhost:5432/ndc?sslmode=disable"),
		ListenAddr:      envOrDefault("LISTEN_ADDR", ":8081"),
		LoadSchedule:    envOrDefault("LOAD_SCHEDULE", "0 3 * * *"),
		FDADownloadsURL: envOrDefault("FDA_DOWNLOADS_URL", "https://open.fda.gov/data/downloads/"),
		DownloadDir:     envOrDefault("DOWNLOAD_DIR", "/tmp/fda-data"),
		LogLevel:        envOrDefault("LOG_LEVEL", "info"),
		LogFormat:       envOrDefault("LOG_FORMAT", "json"),
		DatasetsFile:    envOrDefault("DATASETS_FILE", "datasets.yaml"),
		LandingURL:      os.Getenv("LANDING_URL"),
	}

	maxRetry, err := strconv.Atoi(envOrDefault("MAX_RETRY_ATTEMPTS", "3"))
	if err != nil {
		return nil, fmt.Errorf("invalid MAX_RETRY_ATTEMPTS: %w", err)
	}
	cfg.MaxRetryAttempts = maxRetry

	threshold, err := strconv.ParseFloat(envOrDefault("ROW_COUNT_DROP_THRESHOLD", "0.20"), 64)
	if err != nil {
		return nil, fmt.Errorf("invalid ROW_COUNT_DROP_THRESHOLD: %w", err)
	}
	cfg.RowCountDropThreshold = threshold

	apiKeys := os.Getenv("API_KEYS")
	if apiKeys == "" {
		return nil, fmt.Errorf("API_KEYS environment variable is required")
	}
	for _, key := range strings.Split(apiKeys, ",") {
		trimmed := strings.TrimSpace(key)
		if trimmed != "" {
			cfg.APIKeys = append(cfg.APIKeys, trimmed)
		}
	}

	if len(cfg.APIKeys) == 0 {
		return nil, fmt.Errorf("API_KEYS must contain at least one non-empty key")
	}

	return cfg, nil
}

func envOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
