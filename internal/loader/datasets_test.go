package loader

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDatasetsConfig_Valid(t *testing.T) {
	path := filepath.Join("..", "..", "datasets.yaml")
	cfg, err := LoadDatasetsConfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cfg.Datasets) != 2 {
		t.Fatalf("expected 2 datasets, got %d", len(cfg.Datasets))
	}

	ndc := cfg.Datasets[0]
	if ndc.Name != "ndc_directory" {
		t.Errorf("expected first dataset name ndc_directory, got %s", ndc.Name)
	}
	if !ndc.Enabled {
		t.Error("expected ndc_directory to be enabled")
	}
	if len(ndc.Files) != 2 {
		t.Errorf("expected 2 files for ndc_directory, got %d", len(ndc.Files))
	}

	drugsfda := cfg.Datasets[1]
	if drugsfda.Name != "drugsfda" {
		t.Errorf("expected second dataset name drugsfda, got %s", drugsfda.Name)
	}
	if len(drugsfda.Files) != 5 {
		t.Errorf("expected 5 files for drugsfda, got %d", len(drugsfda.Files))
	}
}

func TestLoadDatasetsConfig_FileNotFound(t *testing.T) {
	_, err := LoadDatasetsConfig("/nonexistent/datasets.yaml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLoadDatasetsConfig_InvalidYAML(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "bad.yaml")
	os.WriteFile(path, []byte(":::invalid:::yaml"), 0o644)

	_, err := LoadDatasetsConfig(path)
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestLoadDatasetsConfig_MissingName(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "noname.yaml")
	content := `datasets:
  - enabled: true
    source_url: http://example.com/data.zip
    files:
      - filename: data.txt
        table: data
`
	os.WriteFile(path, []byte(content), 0o644)

	_, err := LoadDatasetsConfig(path)
	if err == nil {
		t.Fatal("expected error for missing dataset name")
	}
}

func TestLoadDatasetsConfig_MissingSourceURL(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "nourl.yaml")
	content := `datasets:
  - name: test
    enabled: true
    files:
      - filename: data.txt
        table: data
`
	os.WriteFile(path, []byte(content), 0o644)

	_, err := LoadDatasetsConfig(path)
	if err == nil {
		t.Fatal("expected error for missing source_url")
	}
}

func TestLoadDatasetsConfig_MissingFilename(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "nofilename.yaml")
	content := `datasets:
  - name: test
    enabled: true
    source_url: http://example.com/data.zip
    files:
      - table: data
`
	os.WriteFile(path, []byte(content), 0o644)

	_, err := LoadDatasetsConfig(path)
	if err == nil {
		t.Fatal("expected error for missing filename")
	}
}

func TestLoadDatasetsConfig_MissingTable(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "notable.yaml")
	content := `datasets:
  - name: test
    enabled: true
    source_url: http://example.com/data.zip
    files:
      - filename: data.txt
`
	os.WriteFile(path, []byte(content), 0o644)

	_, err := LoadDatasetsConfig(path)
	if err == nil {
		t.Fatal("expected error for missing table")
	}
}

func TestEnabledDatasets(t *testing.T) {
	path := filepath.Join("..", "..", "datasets.yaml")
	cfg, err := LoadDatasetsConfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	enabled := EnabledDatasets(cfg)
	if len(enabled) != 2 {
		t.Errorf("expected 2 enabled datasets, got %d", len(enabled))
	}

	// Disable one.
	cfg.Datasets[1].Enabled = false
	enabled = EnabledDatasets(cfg)
	if len(enabled) != 1 {
		t.Errorf("expected 1 enabled dataset, got %d", len(enabled))
	}
}
