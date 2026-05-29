// Package output contains the interface definitions for general printing and output.
//
// Different execution environments (CLI, Web, API, unit testing) are expected to have their own implementation.
// Output also processes Diagnostics, which the implementation may print or process otherwise.
package output

import (
	"github.com/SSHcom/inventory_import_privx_dev/internal/domain"
	"github.com/SSHcom/inventory_import_privx_dev/internal/field"
)

type Verbosity int

const (
	VerbosityNormal Verbosity = iota
	VerbosityInfo
	VerbosityDebug
	VerbosityTrace
)

// StyleID identifies a semantic output style.
type StyleID string

const (
	StyleBase           StyleID = "base"
	StyleError          StyleID = "error"
	StyleWarning        StyleID = "warning"
	StyleInfo           StyleID = "info"
	StyleImportantField StyleID = "important_field"
	StyleHeader         StyleID = "header"
)

// DebugOutput is a minimal interface for technical logging.
type DebugOutput interface {
	Debugf(format string, args ...any)
	Tracef(format string, args ...any)
	DumpTrace(title string, value any)
	HasVerbosity(min Verbosity) bool
}

type Output interface {
	DebugOutput

	// Printf prints a formatted message when verbosity is at least min.
	Printf(min Verbosity, format string, args ...any)

	// DebugJSON prints a JSON-formatted value when verbosity is at least min.
	DebugJSON(min Verbosity, title string, v any) error

	// Diagnostic will handle a diagnostic message, either by printing it out or storing it for later examination.
	Diagnostic(d domain.Diagnostic)

	// DumpFields prints field structures recursively.
	DumpFields(min Verbosity, title string, fields field.Fields)

	// DumpStruct prints a struct recursively and masks selected field names.
	DumpStruct(
		min Verbosity,
		title string,
		value any,
		maskedFields map[string]struct{},
	)

	// Header prints a section header.
	Header(title string)

	// Style applies a semantic style to text.
	Style(text string, styleID StyleID) string
}
