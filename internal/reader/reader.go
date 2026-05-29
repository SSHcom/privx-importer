// Package reader implements the file reading capability.
//
// The end result of the reading phase is a list of records of field names and values.
package reader

import (
	"context"
	"errors"

	"github.com/SSHcom/inventory_import_privx_dev/internal/config"
	"github.com/SSHcom/inventory_import_privx_dev/internal/output"
)

var (
	ErrInputPathEmpty   = errors.New("input path is empty")
	ErrOpenFailed       = errors.New("open failed")
	ErrContextCancelled = errors.New("read cancelled")
	// ErrMaxRecordsExceeded indicates that an input reader exceeded the configured record limit.
	ErrMaxRecordsExceeded = errors.New("max records exceeded")
)

// SourceID identifies the data source, for example a file name.
type SourceID string

// SourceKind identifies the type of data source, for example file.
type SourceKind string

type RawRecord struct {
	Line   int
	Fields map[string]string
}

// SourceResult is the raw input data from a data source.
type SourceResult struct {
	Source SourceID
	Kind   SourceKind
	Lines  int

	Records []RawRecord
}

// ReadResult contains the raw input data from all the data sources, probably files.
type ReadResult struct {
	Sources []SourceResult
}

type Reader interface {
	// Read reads the given input sources and returns a ReadResult.
	//
	// A non-nil error is returned only for fatal failures (e.g. input cannot be read at all).
	// Non-fatal issues (such as per-record errors) are reported as Diagnostics into the output.
	Read(ctx context.Context, out output.Output, cfg config.Config, input Input) (SourceResult, error)
}
