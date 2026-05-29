package reader

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	clioutput "github.com/SSHcom/inventory_import_privx_dev/internal/cli/output"
	"github.com/SSHcom/inventory_import_privx_dev/internal/config"
	"github.com/SSHcom/inventory_import_privx_dev/internal/output"
)

func TestCSVReaderRead_HappyCase(t *testing.T) {
	t.Parallel()

	path := writeTempCSV(t, "happy.csv", ""+
		"name,address,comment\n"+
		"host-01,10.0.0.1,Primary host\n"+
		"host-02,10.0.0.2,Secondary host\n")

	r := NewCSVReader()

	var outBuf bytes.Buffer
	var errBuf bytes.Buffer
	out := clioutput.New(&outBuf, &errBuf, output.VerbosityTrace)

	result, err := r.Read(context.Background(), out, config.Config{MaxRecords: 100}, Input{
		Path:   path,
		Source: "test-source",
		Kind:   "csv",
	})
	if err != nil {
		t.Fatalf("Read returned error: %v", err)
	}

	if result.Source != SourceID("test-source") {
		t.Fatalf("unexpected source: got %q want %q", result.Source, "test-source")
	}

	if result.Kind != SourceKind("csv") {
		t.Fatalf("unexpected kind: got %q want %q", result.Kind, "csv")
	}

	if result.Lines != 2 {
		t.Fatalf("unexpected line count: got %d want %d", result.Lines, 2)
	}

	if len(result.Records) != 2 {
		t.Fatalf("unexpected record count: got %d want %d", len(result.Records), 2)
	}

	first := result.Records[0]
	if first.Line != 2 {
		t.Fatalf("unexpected first record line: got %d want %d", first.Line, 2)
	}

	if first.Fields["name"] != "host-01" {
		t.Fatalf("unexpected field value for name: got %q want %q", first.Fields["name"], "host-01")
	}

	if first.Fields["address"] != "10.0.0.1" {
		t.Fatalf("unexpected field value for address: got %q want %q", first.Fields["address"], "10.0.0.1")
	}

	if first.Fields["comment"] != "Primary host" {
		t.Fatalf("unexpected field value for comment: got %q want %q", first.Fields["comment"], "Primary host")
	}
}

func TestCSVReaderRead_DuplicateHeader(t *testing.T) {
	t.Parallel()

	path := writeTempCSV(t, "duplicate_header.csv", ""+
		"name,address,name\n"+
		"host-01,10.0.0.1,duplicate\n")

	r := NewCSVReader()
	var outBuf bytes.Buffer
	var errBuf bytes.Buffer
	out := clioutput.New(&outBuf, &errBuf, output.VerbosityTrace)

	_, err := r.Read(context.Background(), out, config.Config{}, Input{
		Path:   path,
		Source: "test-source",
		Kind:   "csv",
	})
	if err == nil {
		t.Fatal("Read returned nil error, want error")
	}

	if !errors.Is(err, ErrCSVHeaderInvalid) {
		t.Fatalf("unexpected error: got %v, want error wrapping %v", err, ErrCSVHeaderInvalid)
	}
}

func TestCSVReaderRead_EmptyHeader(t *testing.T) {
	t.Parallel()

	path := writeTempCSV(t, "empty_header.csv", ""+
		"name,,comment\n"+
		"host-01,10.0.0.1,example\n")

	r := NewCSVReader()
	var outBuf bytes.Buffer
	var errBuf bytes.Buffer
	out := clioutput.New(&outBuf, &errBuf, output.VerbosityTrace)

	_, err := r.Read(context.Background(), out, config.Config{}, Input{
		Path:   path,
		Source: "test-source",
		Kind:   "csv",
	})
	if err == nil {
		t.Fatal("Read returned nil error, want error")
	}

	if !errors.Is(err, ErrCSVHeaderInvalid) {
		t.Fatalf("unexpected error: got %v, want error wrapping %v", err, ErrCSVHeaderInvalid)
	}
}

func TestCSVReaderRead_EmptyFieldValue(t *testing.T) {
	t.Parallel()

	path := writeTempCSV(t, "empty_value.csv", ""+
		"name,address,comment\n"+
		"host-01,10.0.0.1,\n")

	r := NewCSVReader()
	var outBuf bytes.Buffer
	var errBuf bytes.Buffer
	out := clioutput.New(&outBuf, &errBuf, output.VerbosityTrace)

	result, err := r.Read(context.Background(), out, config.Config{MaxRecords: 100}, Input{
		Path:   path,
		Source: "test-source",
		Kind:   "csv",
	})
	if err != nil {
		t.Fatalf("Read returned error: %v", err)
	}

	if result.Lines != 1 {
		t.Fatalf("unexpected line count: got %d want %d", result.Lines, 1)
	}

	if len(result.Records) != 1 {
		t.Fatalf("unexpected record count: got %d want %d", len(result.Records), 1)
	}

	got := result.Records[0].Fields["comment"]
	if got != "" {
		t.Fatalf("unexpected empty field value: got %q want empty string", got)
	}
}

func TestCSVReaderRead_TwoDifferentCSVFiles(t *testing.T) {
	t.Parallel()

	pathA := writeTempCSV(t, "first.csv", ""+
		"name,address\n"+
		"host-01,10.0.0.1\n")

	pathB := writeTempCSV(t, "second.csv", ""+
		"name,address\n"+
		"host-02,10.0.0.2\n")

	r := NewCSVReader()
	ctx := context.Background()
	var outBuf bytes.Buffer
	var errBuf bytes.Buffer
	out := clioutput.New(&outBuf, &errBuf, output.VerbosityTrace)

	resultA, err := r.Read(ctx, out, config.Config{MaxRecords: 100}, Input{
		Path:   pathA,
		Source: "source-a",
		Kind:   "csv",
	})
	if err != nil {
		t.Fatalf("first Read returned error: %v", err)
	}

	resultB, err := r.Read(ctx, out, config.Config{MaxRecords: 100}, Input{
		Path:   pathB,
		Source: "source-b",
		Kind:   "csv",
	})
	if err != nil {
		t.Fatalf("second Read returned error: %v", err)
	}

	if len(resultA.Records) != 1 {
		t.Fatalf("unexpected first result record count: got %d want %d", len(resultA.Records), 1)
	}

	if len(resultB.Records) != 1 {
		t.Fatalf("unexpected second result record count: got %d want %d", len(resultB.Records), 1)
	}

	if resultA.Records[0].Fields["name"] != "host-01" {
		t.Fatalf("unexpected first file record name: got %q want %q", resultA.Records[0].Fields["name"], "host-01")
	}

	if resultB.Records[0].Fields["name"] != "host-02" {
		t.Fatalf("unexpected second file record name: got %q want %q", resultB.Records[0].Fields["name"], "host-02")
	}

	if resultA.Source != SourceID("source-a") {
		t.Fatalf("unexpected first result source: got %q want %q", resultA.Source, "source-a")
	}

	if resultB.Source != SourceID("source-b") {
		t.Fatalf("unexpected second result source: got %q want %q", resultB.Source, "source-b")
	}
}

func writeTempCSV(t *testing.T, name string, content string) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, name)

	err := os.WriteFile(path, []byte(content), 0o600)
	if err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	return path
}
