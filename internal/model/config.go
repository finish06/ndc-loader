package model

import "time"

// Config holds application configuration loaded from environment variables.
type Config struct {
	DatabaseURL           string
	ListenAddr            string
	LoadSchedule          string
	FDADownloadsURL       string
	DownloadDir           string
	MaxRetryAttempts      int
	RowCountDropThreshold float64
	APIKeys               []string
	LogLevel              string
	LogFormat             string
	DatasetsFile          string
}

// DatasetConfig represents a configured FDA dataset to download and ingest.
type DatasetConfig struct {
	Name      string       `yaml:"name"`
	Enabled   bool         `yaml:"enabled"`
	SourceURL string       `yaml:"source_url"`
	Format    string       `yaml:"format"`
	Files     []FileConfig `yaml:"files"`
}

// FileConfig represents a single file within a dataset ZIP.
type FileConfig struct {
	Filename  string `yaml:"filename"`
	Table     string `yaml:"table"`
	Delimiter string `yaml:"delimiter"`
	HasHeader bool   `yaml:"has_header"`
}

// DatasetsConfig is the top-level structure for datasets.yaml.
type DatasetsConfig struct {
	Datasets []DatasetConfig `yaml:"datasets"`
}

// LoadCheckpoint tracks the progress of a single table load within a load execution.
type LoadCheckpoint struct {
	ID               int
	LoadID           string
	Dataset          string
	TableName        string
	Status           LoadStatus
	RowCount         *int
	PreviousRowCount *int
	ErrorMessage     *string
	StartedAt        *time.Time
	CompletedAt      *time.Time
	CreatedAt        time.Time
}

// LoadStatus represents the state of a table load.
type LoadStatus string

const (
	LoadStatusPending     LoadStatus = "pending"
	LoadStatusDownloading LoadStatus = "downloading"
	LoadStatusDownloaded  LoadStatus = "downloaded"
	LoadStatusLoading     LoadStatus = "loading"
	LoadStatusLoaded      LoadStatus = "loaded"
	LoadStatusFailed      LoadStatus = "failed"
)
