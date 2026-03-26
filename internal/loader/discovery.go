package loader

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// DiscoveredDataset represents a dataset found on the openFDA downloads page.
type DiscoveredDataset struct {
	Name string
	URL  string
}

// DiscoverDatasets fetches the openFDA downloads page and extracts available dataset URLs.
// Falls back gracefully if the page structure changes — configured source_url per dataset
// is used as the primary download source, discovery is supplementary.
func DiscoverDatasets(downloadsURL string) ([]DiscoveredDataset, error) {
	client := &http.Client{Timeout: 30 * time.Second}

	resp, err := client.Get(downloadsURL)
	if err != nil {
		slog.Warn("failed to fetch openFDA downloads page, falling back to configured URLs",
			"url", downloadsURL, "error", err)
		return nil, fmt.Errorf("fetching downloads page: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		slog.Warn("openFDA downloads page returned non-200, falling back to configured URLs",
			"url", downloadsURL, "status", resp.StatusCode)
		return nil, fmt.Errorf("downloads page returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading downloads page body: %w", err)
	}

	return parseDownloadsPage(string(body)), nil
}

// parseDownloadsPage extracts dataset links from the openFDA downloads page HTML.
// Looks for links to ZIP files and known FDA data paths.
func parseDownloadsPage(html string) []DiscoveredDataset {
	var datasets []DiscoveredDataset

	// Match href attributes pointing to FDA data downloads.
	linkRe := regexp.MustCompile(`href=["']([^"']*(?:\.zip|download)[^"']*)["']`)
	matches := linkRe.FindAllStringSubmatch(html, -1)

	seen := make(map[string]bool)
	for _, m := range matches {
		url := m[1]
		if seen[url] {
			continue
		}
		seen[url] = true

		name := inferDatasetName(url)
		if name != "" {
			datasets = append(datasets, DiscoveredDataset{Name: name, URL: url})
		}
	}

	slog.Info("discovered datasets from openFDA", "count", len(datasets))
	return datasets
}

func inferDatasetName(url string) string {
	lower := strings.ToLower(url)
	switch {
	case strings.Contains(lower, "ndctext"):
		return "ndc_directory"
	case strings.Contains(lower, "drugsfda") || strings.Contains(lower, "89850"):
		return "drugsfda"
	case strings.Contains(lower, ".zip"):
		// Generic ZIP — extract name from filename.
		parts := strings.Split(url, "/")
		if len(parts) > 0 {
			return strings.TrimSuffix(parts[len(parts)-1], ".zip")
		}
	}
	return ""
}
