package loader

import (
	"encoding/csv"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
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
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("opening file %s: %w", filePath, err)
	}
	defer f.Close()

	reader := csv.NewReader(f)
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

// MapColumns maps parsed rows to the target table columns using header names.
// Source headers are matched case-insensitively to target column names.
// Unknown source columns are ignored. Missing target columns get nil values.
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
		targetIdx int
		sourceIdx int
	}
	var mappings []colMap

	for targetIdx, targetCol := range targetColumns {
		// Check if there's a custom header mapping.
		sourceHeader := targetCol
		if mapped, ok := headerMapping[targetCol]; ok {
			sourceHeader = mapped
		}

		srcIdx, found := sourceIdx[strings.ToLower(sourceHeader)]
		if !found {
			// Target column not in source — will be nil.
			continue
		}
		mappings = append(mappings, colMap{targetIdx: targetIdx, sourceIdx: srcIdx})
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
					mapped[m.targetIdx] = val
				}
			}
		}
		result = append(result, mapped)
	}

	return result, nil
}
