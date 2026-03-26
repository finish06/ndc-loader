package loader

import (
	"archive/zip"
	"fmt"
	"strings"
	"io"
	"log/slog"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// Downloader handles downloading and extracting FDA dataset ZIP files.
type Downloader struct {
	downloadDir      string
	maxRetryAttempts int
	httpClient       *http.Client
}

// NewDownloader creates a new Downloader.
func NewDownloader(downloadDir string, maxRetry int) *Downloader {
	return &Downloader{
		downloadDir:      downloadDir,
		maxRetryAttempts: maxRetry,
		httpClient: &http.Client{
			Timeout: 5 * time.Minute,
		},
	}
}

// Download fetches a file from url and saves it to the download directory.
// Retries with exponential backoff on failure.
func (d *Downloader) Download(url, filename string) (string, error) {
	if err := os.MkdirAll(d.downloadDir, 0o755); err != nil {
		return "", fmt.Errorf("creating download dir: %w", err)
	}

	destPath := filepath.Join(d.downloadDir, filename)

	var lastErr error
	for attempt := 0; attempt <= d.maxRetryAttempts; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(math.Pow(2, float64(attempt-1))) * time.Second
			slog.Warn("retrying download", "url", url, "attempt", attempt, "backoff", backoff)
			time.Sleep(backoff)
		}

		if err := d.downloadFile(url, destPath); err != nil {
			lastErr = err
			slog.Error("download attempt failed", "url", url, "attempt", attempt, "error", err)
			continue
		}

		slog.Info("download complete", "url", url, "path", destPath)
		return destPath, nil
	}

	return "", fmt.Errorf("download failed after %d attempts: %w", d.maxRetryAttempts+1, lastErr)
}

func (d *Downloader) downloadFile(url, dest string) error {
	resp, err := d.httpClient.Get(url)
	if err != nil {
		return fmt.Errorf("HTTP GET %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP GET %s returned status %d", url, resp.StatusCode)
	}

	f, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("creating file %s: %w", dest, err)
	}
	defer f.Close()

	if _, err := io.Copy(f, resp.Body); err != nil {
		os.Remove(dest)
		return fmt.Errorf("writing to %s: %w", dest, err)
	}

	return nil
}

// Extract unzips a ZIP file into the download directory and returns the extraction path.
func (d *Downloader) Extract(zipPath, datasetName string) (string, error) {
	extractDir := filepath.Join(d.downloadDir, datasetName)
	if err := os.MkdirAll(extractDir, 0o755); err != nil {
		return "", fmt.Errorf("creating extract dir: %w", err)
	}

	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return "", fmt.Errorf("opening zip %s: %w", zipPath, err)
	}
	defer r.Close()

	for _, f := range r.File {
		destPath := filepath.Join(extractDir, f.Name)

		// Prevent zip slip attack.
		if !isSubPath(extractDir, destPath) {
			return "", fmt.Errorf("zip slip detected: %s", f.Name)
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(destPath, 0o755)
			continue
		}

		if err := extractFile(f, destPath); err != nil {
			return "", fmt.Errorf("extracting %s: %w", f.Name, err)
		}
	}

	slog.Info("extraction complete", "zip", zipPath, "dir", extractDir, "files", len(r.File))
	return extractDir, nil
}

func extractFile(f *zip.File, dest string) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}

	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, rc)
	return err
}

func isSubPath(parent, child string) bool {
	rel, err := filepath.Rel(parent, child)
	if err != nil {
		return false
	}
	if filepath.IsAbs(rel) {
		return false
	}
	return !strings.HasPrefix(rel, "..")
}

// Cleanup removes downloaded and extracted files for a dataset.
func (d *Downloader) Cleanup(datasetName string) {
	extractDir := filepath.Join(d.downloadDir, datasetName)
	os.RemoveAll(extractDir)

	// Remove ZIP files matching the dataset name pattern.
	matches, _ := filepath.Glob(filepath.Join(d.downloadDir, datasetName+"*"))
	for _, m := range matches {
		os.Remove(m)
	}
}
