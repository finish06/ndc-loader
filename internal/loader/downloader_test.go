package loader

import (
	"archive/zip"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestDownloader_Download_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("test file content"))
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	d := NewDownloader(tmpDir, 3)

	path, err := d.Download(server.URL+"/test.zip", "test.zip")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read downloaded file: %v", err)
	}
	if string(content) != "test file content" {
		t.Errorf("unexpected file content: %s", string(content))
	}
}

func TestDownloader_Download_RetryOnFailure(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Write([]byte("success after retries"))
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	d := NewDownloader(tmpDir, 3)

	path, err := d.Download(server.URL+"/test.zip", "test.zip")
	if err != nil {
		t.Fatalf("unexpected error after retries: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read: %v", err)
	}
	if string(content) != "success after retries" {
		t.Errorf("unexpected content: %s", string(content))
	}
	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

func TestDownloader_Download_AllRetriesFail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	d := NewDownloader(tmpDir, 2)

	_, err := d.Download(server.URL+"/test.zip", "test.zip")
	if err == nil {
		t.Fatal("expected error after all retries fail")
	}
}

func TestDownloader_Extract_ValidZip(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test ZIP file.
	zipPath := filepath.Join(tmpDir, "test.zip")
	createTestZip(t, zipPath, map[string]string{
		"product.txt": "header1\theader2\nval1\tval2\n",
		"package.txt": "h1\th2\nv1\tv2\n",
	})

	d := NewDownloader(tmpDir, 1)
	extractDir, err := d.Extract(zipPath, "test_dataset")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify extracted files exist.
	for _, name := range []string{"product.txt", "package.txt"} {
		if _, err := os.Stat(filepath.Join(extractDir, name)); os.IsNotExist(err) {
			t.Errorf("expected extracted file %s to exist", name)
		}
	}
}

func TestDownloader_Extract_InvalidZip(t *testing.T) {
	tmpDir := t.TempDir()
	badZip := filepath.Join(tmpDir, "bad.zip")
	os.WriteFile(badZip, []byte("not a zip file"), 0o644)

	d := NewDownloader(tmpDir, 1)
	_, err := d.Extract(badZip, "test")
	if err == nil {
		t.Fatal("expected error for invalid ZIP")
	}
}

func TestIsSubPath(t *testing.T) {
	tests := []struct {
		parent string
		child  string
		want   bool
	}{
		{"/tmp/extract", "/tmp/extract/file.txt", true},
		{"/tmp/extract", "/tmp/extract/sub/file.txt", true},
		{"/tmp/extract", "/tmp/other/file.txt", false},
		{"/tmp/extract", "/tmp/extract/../other/file.txt", false},
	}

	for _, tc := range tests {
		got := isSubPath(tc.parent, tc.child)
		if got != tc.want {
			t.Errorf("isSubPath(%q, %q) = %v, want %v", tc.parent, tc.child, got, tc.want)
		}
	}
}

func createTestZip(t *testing.T, path string, files map[string]string) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	w := zip.NewWriter(f)
	for name, content := range files {
		fw, err := w.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		fw.Write([]byte(content))
	}
	w.Close()
}
