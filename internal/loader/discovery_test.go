package loader

import (
	"testing"
)

func TestParseDownloadsPage_FindsNDC(t *testing.T) {
	html := `<html><body>
		<a href="https://www.accessdata.fda.gov/cder/ndctext.zip">NDC Directory</a>
		<a href="https://www.fda.gov/media/89850/download">Drugs@FDA</a>
		<a href="https://example.com/other.zip">Other Data</a>
	</body></html>`

	datasets := parseDownloadsPage(html)

	if len(datasets) < 2 {
		t.Fatalf("expected at least 2 datasets, got %d", len(datasets))
	}

	foundNDC := false
	foundDrugsFDA := false
	for _, ds := range datasets {
		if ds.Name == "ndc_directory" {
			foundNDC = true
		}
		if ds.Name == "drugsfda" {
			foundDrugsFDA = true
		}
	}

	if !foundNDC {
		t.Error("expected to find ndc_directory dataset")
	}
	if !foundDrugsFDA {
		t.Error("expected to find drugsfda dataset")
	}
}

func TestParseDownloadsPage_EmptyHTML(t *testing.T) {
	datasets := parseDownloadsPage("")
	if len(datasets) != 0 {
		t.Errorf("expected 0 datasets from empty HTML, got %d", len(datasets))
	}
}

func TestParseDownloadsPage_NoDuplicates(t *testing.T) {
	html := `<html>
		<a href="https://www.accessdata.fda.gov/cder/ndctext.zip">Link 1</a>
		<a href="https://www.accessdata.fda.gov/cder/ndctext.zip">Link 2</a>
	</html>`

	datasets := parseDownloadsPage(html)

	ndcCount := 0
	for _, ds := range datasets {
		if ds.Name == "ndc_directory" {
			ndcCount++
		}
	}
	if ndcCount != 1 {
		t.Errorf("expected 1 ndc_directory entry, got %d", ndcCount)
	}
}

func TestInferDatasetName(t *testing.T) {
	tests := []struct {
		url      string
		expected string
	}{
		{"https://www.accessdata.fda.gov/cder/ndctext.zip", "ndc_directory"},
		{"https://www.fda.gov/media/89850/download", "drugsfda"},
		{"https://example.com/drugsfda_data.zip", "drugsfda"},
		{"https://example.com/random.zip", "random"},
		{"https://example.com/no-extension", ""},
	}

	for _, tc := range tests {
		t.Run(tc.url, func(t *testing.T) {
			name := inferDatasetName(tc.url)
			if name != tc.expected {
				t.Errorf("inferDatasetName(%q) = %q, want %q", tc.url, name, tc.expected)
			}
		})
	}
}
