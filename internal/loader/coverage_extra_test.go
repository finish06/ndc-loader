package loader

import (
	"os"
	"path/filepath"
	"testing"
)

// --- sanitizeUTF8 coverage ---

func TestSanitizeUTF8_ValidInput(t *testing.T) {
	input := []byte("hello world")
	result := sanitizeUTF8(input)
	if string(result) != "hello world" {
		t.Errorf("expected 'hello world', got %s", string(result))
	}
}

func TestSanitizeUTF8_InvalidBytes(t *testing.T) {
	// 0x92 is Windows-1252 right single quote, invalid in UTF-8.
	input := []byte("it\x92s a test")
	result := sanitizeUTF8(input)

	// Should replace invalid byte with U+FFFD.
	if string(result) != "it\uFFFDs a test" {
		t.Errorf("expected replacement character, got %q", string(result))
	}
}

func TestSanitizeUTF8_MultipleInvalidBytes(t *testing.T) {
	input := []byte("\xbf\x92\xff")
	result := sanitizeUTF8(input)

	// Each invalid byte should become a replacement character.
	expected := "\uFFFD\uFFFD\uFFFD"
	if string(result) != expected {
		t.Errorf("expected %q, got %q", expected, string(result))
	}
}

func TestSanitizeUTF8_EmptyInput(t *testing.T) {
	result := sanitizeUTF8([]byte{})
	if len(result) != 0 {
		t.Errorf("expected empty result, got %q", string(result))
	}
}

func TestSanitizeUTF8_MixedValidInvalid(t *testing.T) {
	// Valid UTF-8 multibyte char (e.g., euro sign) mixed with invalid byte.
	input := []byte("price: \xe2\x82\xac100\x92")
	result := sanitizeUTF8(input)

	// Euro sign should be preserved, 0x92 should be replaced.
	expected := "price: \u20ac100\uFFFD"
	if string(result) != expected {
		t.Errorf("expected %q, got %q", expected, string(result))
	}
}

// --- ParseTabDelimited: invalid UTF-8 path ---

func TestParseTabDelimited_InvalidUTF8(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "bad_utf8.txt")

	// Create file with invalid UTF-8 bytes.
	content := []byte("col1\tcol2\nval\x92ue1\tval2\n")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	result, err := ParseTabDelimited(path, '\t', true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Headers) != 2 {
		t.Errorf("expected 2 headers, got %d", len(result.Headers))
	}
	if len(result.Rows) != 1 {
		t.Errorf("expected 1 row, got %d", len(result.Rows))
	}
}

// --- ParseTabDelimited: empty file (no header) ---

func TestParseTabDelimited_EmptyFile(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "empty.txt")
	if err := os.WriteFile(path, []byte(""), 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	result, err := ParseTabDelimited(path, '\t', false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Rows) != 0 {
		t.Errorf("expected 0 rows, got %d", len(result.Rows))
	}
}

// --- ParseTabDelimited: header-only file ---

func TestParseTabDelimited_HeaderOnly(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "header_only.txt")
	content := "col1\tcol2\tcol3\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	result, err := ParseTabDelimited(path, '\t', true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Headers) != 3 {
		t.Errorf("expected 3 headers, got %d", len(result.Headers))
	}
	if len(result.Rows) != 0 {
		t.Errorf("expected 0 rows, got %d", len(result.Rows))
	}
}

// --- ParseTabDelimited: comma delimiter ---

func TestParseTabDelimited_CommaDelimiter(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "csv.txt")
	content := "col1,col2\nval1,val2\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	result, err := ParseTabDelimited(path, ',', true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Headers) != 2 {
		t.Errorf("expected 2 headers, got %d", len(result.Headers))
	}
	if len(result.Rows) != 1 {
		t.Errorf("expected 1 row, got %d", len(result.Rows))
	}
}

// --- NewScheduler: valid schedule with orchestrator ---

func TestNewScheduler_NilOrchestrator(t *testing.T) {
	// The cron function references orchestrator, but we're just testing that
	// NewScheduler succeeds with a valid schedule. The cron func won't be called.
	s, err := NewScheduler(nil, "0 3 * * *", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s == nil {
		t.Fatal("expected non-nil scheduler")
	}
	// Start and stop should not panic even with nil logger/orchestrator.
}

// --- Downloader: downloadFile error paths ---

func TestDownloader_Download_CreateDirError(t *testing.T) {
	// Use a path that can't be created.
	d := NewDownloader("/dev/null/impossible", 0)
	_, err := d.Download("http://example.com/file.zip", "test.zip")
	if err == nil {
		t.Fatal("expected error for impossible directory")
	}
}

// --- isSubPath: edge cases ---

func TestIsSubPath_SamePath(t *testing.T) {
	if !isSubPath("/tmp/dir", "/tmp/dir") {
		t.Error("expected true for same path")
	}
}

func TestIsSubPath_RelativeChild(t *testing.T) {
	if !isSubPath("/tmp/dir", "/tmp/dir/sub/./file.txt") {
		t.Error("expected true for relative child path")
	}
}
