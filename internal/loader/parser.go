package loader

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"
	"unicode/utf8"
)

// ParseResult holds the parsed rows and header mapping from a data file.
type ParseResult struct {
	Headers []string
	Rows    [][]string
	Skipped int
}

// ParseTabDelimited reads a tab-delimited file and returns parsed rows.
// It handles unexpected columns gracefully (logs warning, maps known columns).
// Malformed rows are skipped with a warning.
func ParseTabDelimited(filePath string, delimiter rune, hasHeader bool) (*ParseResult, error) {
	rawData, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("reading file %s: %w", filePath, err)
	}

	// Sanitize invalid UTF-8 sequences (FDA data sometimes contains Windows-1252 bytes).
	if !utf8.Valid(rawData) {
		rawData = sanitizeUTF8(rawData)
		slog.Warn("sanitized invalid UTF-8 bytes", "file", filePath)
	}

	reader := csv.NewReader(bytes.NewReader(rawData))
	reader.Comma = delimiter
	reader.LazyQuotes = true
	reader.FieldsPerRecord = -1 // Allow variable number of fields.

	result := &ParseResult{}

	if hasHeader {
		headers, err := reader.Read()
		if err != nil {
			return nil, fmt.Errorf("reading header: %w", err)
		}
		for i, h := range headers {
			headers[i] = strings.TrimSpace(h)
		}
		result.Headers = headers
	}

	lineNum := 1
	if hasHeader {
		lineNum = 2
	}

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			slog.Warn("skipping malformed row", "file", filePath, "line", lineNum, "error", err)
			result.Skipped++
			lineNum++
			continue
		}

		result.Rows = append(result.Rows, record)
		lineNum++
	}

	slog.Info("parsed file",
		"file", filePath,
		"rows", len(result.Rows),
		"skipped", result.Skipped,
		"headers", len(result.Headers),
	)

	return result, nil
}

// dateColumns lists target column names that should be parsed as dates.
var dateColumns = map[string]bool{
	"marketing_start":        true,
	"marketing_end":          true,
	"listing_certified":      true,
	"most_recent_submission": true,
	"submission_status_date": true,
}

// boolColumns lists target column names that should be parsed as booleans.
var boolColumns = map[string]bool{
	"ndc_exclude":    true,
	"sample_package": true,
}

// MapColumns maps parsed rows to the target table columns using header names.
// Source headers are matched case-insensitively to target column names.
// Unknown source columns are ignored. Missing target columns get nil values.
// Date and boolean columns are automatically coerced from string representations.
func MapColumns(parsed *ParseResult, targetColumns []string, headerMapping map[string]string) ([][]interface{}, error) {
	if len(parsed.Headers) == 0 {
		return nil, fmt.Errorf("no headers available for column mapping")
	}

	// Build source header index (lowercase for case-insensitive matching).
	sourceIdx := make(map[string]int)
	for i, h := range parsed.Headers {
		sourceIdx[strings.ToLower(strings.TrimSpace(h))] = i
	}

	// Build mapping: target column index -> source column index.
	type colMap struct {
		targetIdx  int
		sourceIdx  int
		targetName string
	}
	var mappings []colMap

	for targetIdx, targetCol := range targetColumns {
		sourceHeader := targetCol
		if mapped, ok := headerMapping[targetCol]; ok {
			sourceHeader = mapped
		}

		srcIdx, found := sourceIdx[strings.ToLower(sourceHeader)]
		if !found {
			continue
		}
		mappings = append(mappings, colMap{targetIdx: targetIdx, sourceIdx: srcIdx, targetName: targetCol})
	}

	if len(mappings) == 0 {
		return nil, fmt.Errorf("no column mappings found between source headers and target columns")
	}

	// Map each row.
	result := make([][]interface{}, 0, len(parsed.Rows))
	for _, row := range parsed.Rows {
		mapped := make([]interface{}, len(targetColumns))
		for _, m := range mappings {
			if m.sourceIdx < len(row) {
				val := strings.TrimSpace(row[m.sourceIdx])
				if val == "" {
					mapped[m.targetIdx] = nil
				} else {
					mapped[m.targetIdx] = coerceValue(val, m.targetName)
				}
			}
		}
		result = append(result, mapped)
	}

	return result, nil
}

// coerceValue converts a string value to the appropriate Go type
// based on the target column name.
func coerceValue(val, columnName string) interface{} {
	if dateColumns[columnName] {
		return parseDate(val)
	}
	if boolColumns[columnName] {
		return parseBool(val)
	}
	return val
}

// parseDate attempts to parse FDA date formats (YYYYMMDD or YYYY-MM-DD).
// Returns nil if parsing fails.
func parseDate(val string) interface{} {
	// Try YYYYMMDD format (most common in FDA data).
	t, err := time.Parse("20060102", val)
	if err == nil {
		return t
	}

	// Try YYYY-MM-DD.
	t, err = time.Parse("2006-01-02", val)
	if err == nil {
		return t
	}

	// Try MM/DD/YYYY.
	t, err = time.Parse("01/02/2006", val)
	if err == nil {
		return t
	}

	slog.Debug("unparseable date value, storing as nil", "value", val)
	return nil
}

// parseBool converts Y/N, Yes/No, True/False strings to bool.
func parseBool(val string) interface{} {
	switch strings.ToUpper(val) {
	case "Y", "YES", "TRUE", "1":
		return true
	case "N", "NO", "FALSE", "0":
		return false
	default:
		return false
	}
}

// sanitizeUTF8 replaces invalid UTF-8 byte sequences with the Unicode
// replacement character. FDA data sometimes contains Windows-1252 encoded
// characters (e.g., 0x92 for right single quote, 0xbf for inverted question mark).
func sanitizeUTF8(data []byte) []byte {
	var buf bytes.Buffer
	buf.Grow(len(data))
	for len(data) > 0 {
		r, size := utf8.DecodeRune(data)
		if r == utf8.RuneError && size == 1 {
			buf.WriteRune('\uFFFD')
		} else {
			buf.WriteRune(r)
		}
		data = data[size:]
	}
	return buf.Bytes()
}
