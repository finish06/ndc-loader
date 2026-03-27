package loader

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"unicode/utf8"
)

// StreamParser reads a tab-delimited file and yields rows one at a time via a callback.
// This avoids loading the entire file + all parsed rows into memory simultaneously.
type StreamParser struct {
	filePath  string
	delimiter rune
	hasHeader bool
}

// NewStreamParser creates a StreamParser.
func NewStreamParser(filePath string, delimiter rune, hasHeader bool) *StreamParser {
	return &StreamParser{filePath: filePath, delimiter: delimiter, hasHeader: hasHeader}
}

// StreamResult holds the outcome of a streaming parse operation.
type StreamResult struct {
	Headers  []string
	RowCount int
	Skipped  int
}

// Parse reads the file and calls rowFn for each parsed + mapped row.
// rowFn receives a slice of interface{} values ready for database insertion.
// This streams through the file without holding all rows in memory.
func (sp *StreamParser) Parse(
	targetColumns []string,
	headerMapping map[string]string,
	rowFn func(row []interface{}) error,
) (*StreamResult, error) {
	rawData, err := os.ReadFile(sp.filePath)
	if err != nil {
		return nil, fmt.Errorf("reading file %s: %w", sp.filePath, err)
	}

	// Sanitize invalid UTF-8 if needed.
	if !utf8.Valid(rawData) {
		rawData = sanitizeUTF8(rawData)
		slog.Warn("sanitized invalid UTF-8 bytes", "file", sp.filePath)
	}

	reader := csv.NewReader(bytes.NewReader(rawData))
	reader.Comma = sp.delimiter
	reader.LazyQuotes = true
	reader.FieldsPerRecord = -1

	result := &StreamResult{}

	// Read headers.
	if sp.hasHeader {
		headers, err := reader.Read()
		if err != nil {
			return nil, fmt.Errorf("reading header: %w", err)
		}
		for i, h := range headers {
			headers[i] = strings.TrimSpace(h)
		}
		result.Headers = headers
	}

	// Build column mapping.
	type colMap struct {
		targetIdx  int
		sourceIdx  int
		targetName string
	}

	sourceIdx := make(map[string]int)
	for i, h := range result.Headers {
		sourceIdx[strings.ToLower(strings.TrimSpace(h))] = i
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
		return nil, fmt.Errorf("no column mappings found")
	}

	// Stream rows — map and emit one at a time.
	lineNum := 2
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			slog.Warn("skipping malformed row", "file", sp.filePath, "line", lineNum, "error", err)
			result.Skipped++
			lineNum++
			continue
		}

		// Map this row.
		mapped := make([]interface{}, len(targetColumns))
		for _, m := range mappings {
			if m.sourceIdx < len(record) {
				val := strings.TrimSpace(record[m.sourceIdx])
				if val == "" {
					mapped[m.targetIdx] = nil
				} else {
					mapped[m.targetIdx] = coerceValue(val, m.targetName)
				}
			}
		}

		if err := rowFn(mapped); err != nil {
			return nil, fmt.Errorf("processing row %d: %w", lineNum, err)
		}

		result.RowCount++
		lineNum++
	}

	slog.Info("streamed file",
		"file", sp.filePath,
		"rows", result.RowCount,
		"skipped", result.Skipped,
	)

	return result, nil
}
