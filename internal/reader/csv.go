package reader

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/SSHcom/inventory_import_privx_dev/internal/config"
	"github.com/SSHcom/inventory_import_privx_dev/internal/domain"
	"github.com/SSHcom/inventory_import_privx_dev/internal/output"
)

var (
	ErrCSVReadFailed    = errors.New("csv read failed")
	ErrCSVHeaderMissing = errors.New("csv header missing")
	ErrCSVHeaderInvalid = errors.New("csv header invalid")
)

type CSVReader struct {
	Comma rune
}

func NewCSVReader() *CSVReader {
	return &CSVReader{
		Comma: ',',
	}
}

func (r *CSVReader) Read(ctx context.Context, out output.Output, cfg config.Config, input Input) (SourceResult, error) {
	if err := ctx.Err(); err != nil {
		return SourceResult{}, fmt.Errorf("%w: %w", ErrContextCancelled, err)
	}

	if strings.TrimSpace(input.Path) == "" {
		return SourceResult{}, ErrInputPathEmpty
	}

	file, err := os.Open(input.Path)
	if err != nil {
		return SourceResult{}, fmt.Errorf("%w: %s: %w", ErrOpenFailed, input.Path, err)
	}
	defer file.Close()

	csvReader := csv.NewReader(file)
	if r.Comma != 0 {
		csvReader.Comma = r.Comma
	}
	// No checking of field count
	csvReader.FieldsPerRecord = -1
	csvReader.LazyQuotes = true

	header, err := csvReader.Read()
	if err != nil {
		if errors.Is(err, io.EOF) {
			return SourceResult{}, fmt.Errorf("%w: %s", ErrCSVHeaderMissing, input.Path)
		}
		return SourceResult{}, fmt.Errorf("%w: %s: %w", ErrCSVReadFailed, input.Path, err)
	}

	header, err = normalizeHeader(header)
	if err != nil {
		return SourceResult{}, fmt.Errorf("%w: %s: %w", ErrCSVHeaderInvalid, input.Path, err)
	}

	result := SourceResult{
		Source:  SourceID(input.Source),
		Kind:    SourceKind(input.Kind),
		Lines:   0,
		Records: nil,
	}

	lineNumber := 1 // Header line.

	for {
		if err := ctx.Err(); err != nil {
			return SourceResult{}, fmt.Errorf("%w: %w", ErrContextCancelled, err)
		}

		row, err := csvReader.Read()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return SourceResult{}, fmt.Errorf("%w: %s: line %d: %w", ErrCSVReadFailed, input.Path, lineNumber+1, err)
		}

		lineNumber++

		record := buildRawRecord(out, input, header, row, lineNumber)
		result.Records = append(result.Records, record)
		result.Lines++
		if result.Lines > cfg.MaxRecords {
			return SourceResult{}, fmt.Errorf("%w: %s: line %d", ErrMaxRecordsExceeded, input.Path, lineNumber+1)
		}
	}

	return result, nil
}

// normalizeHeader will ensure no empty or duplicate headers.
func normalizeHeader(header []string) ([]string, error) {
	if len(header) == 0 {
		return nil, ErrCSVHeaderMissing
	}

	normalized := make([]string, 0, len(header))
	seen := make(map[string]struct{}, len(header))

	for i, col := range header {
		name := strings.TrimSpace(col)
		if name == "" {
			return nil, fmt.Errorf("%w: column %d has empty name", ErrCSVHeaderInvalid, i)
		}

		if _, exists := seen[name]; exists {
			return nil, fmt.Errorf("%w: column %d has duplicate name %q", ErrCSVHeaderInvalid, i, name)
		}

		seen[name] = struct{}{}
		normalized = append(normalized, name)
	}

	return normalized, nil
}

// buildRawRecord constructs a RawRecord from a CSV row using header names as keys.
//
// Too long or short lines will be read, but a warning is attached.
func buildRawRecord(out output.Output, input Input, header []string, row []string, lineNumber int) RawRecord {
	record := RawRecord{
		Line:   lineNumber,
		Fields: make(map[string]string, len(header)),
	}

	if len(row) < len(header) {
		out.Diagnostic(domain.Diagnostic{
			Source:   input.Source,
			Line:     lineNumber,
			Severity: domain.SeverityWarning,
			Code:     "csv_row_short",
			Message:  fmt.Sprintf("row has %d fields, expected %d", len(row), len(header)),
		})
	}

	if len(row) > len(header) {
		out.Diagnostic(domain.Diagnostic{
			Source:   input.Source,
			Line:     lineNumber,
			Severity: domain.SeverityWarning,
			Code:     "csv_row_long",
			Message:  fmt.Sprintf("row has %d fields, expected %d", len(row), len(header)),
		})
	}

	limit := len(header)
	if len(row) < limit {
		limit = len(row)
	}

	for i := 0; i < limit; i++ {
		record.Fields[header[i]] = strings.TrimSpace(row[i])
	}

	for i := limit; i < len(header); i++ {
		record.Fields[header[i]] = ""
	}

	return record
}
