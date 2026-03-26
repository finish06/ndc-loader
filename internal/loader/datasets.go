package loader

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"

	"github.com/calebdunn/ndc-loader/internal/model"
)

// LoadDatasetsConfig reads and parses the datasets.yaml configuration file.
func LoadDatasetsConfig(path string) (*model.DatasetsConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading datasets config %s: %w", path, err)
	}

	var cfg model.DatasetsConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing datasets config: %w", err)
	}

	for i, ds := range cfg.Datasets {
		if ds.Name == "" {
			return nil, fmt.Errorf("dataset at index %d has no name", i)
		}
		if ds.SourceURL == "" {
			return nil, fmt.Errorf("dataset %q has no source_url", ds.Name)
		}
		for j, f := range ds.Files {
			if f.Filename == "" {
				return nil, fmt.Errorf("dataset %q file at index %d has no filename", ds.Name, j)
			}
			if f.Table == "" {
				return nil, fmt.Errorf("dataset %q file %q has no table", ds.Name, f.Filename)
			}
		}
	}

	return &cfg, nil
}

// EnabledDatasets returns only datasets with enabled=true.
func EnabledDatasets(cfg *model.DatasetsConfig) []model.DatasetConfig {
	var enabled []model.DatasetConfig
	for _, ds := range cfg.Datasets {
		if ds.Enabled {
			enabled = append(enabled, ds)
		}
	}
	return enabled
}
