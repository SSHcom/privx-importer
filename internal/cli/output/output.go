// Package cli.output implements the [output] interface for CLI environment.
//
// This package uses lipgloss for some visual enhancements on TTY.
package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/SSHcom/inventory_import_privx_dev/internal/config"
	"github.com/SSHcom/inventory_import_privx_dev/internal/domain"
	"github.com/SSHcom/inventory_import_privx_dev/internal/output"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-isatty"
)

// toVerbosity converts CLI flags to verbosity value.
func ToVerbosity(v int) output.Verbosity {
	switch {
	case v <= 0:
		return output.VerbosityNormal
	case v == 1:
		return output.VerbosityInfo
	case v == 2:
		return output.VerbosityDebug
	default:
		return output.VerbosityTrace
	}
}

// Lipgloss colors.
const (
	colorBase    = "0"
	colorError   = "1"
	colorWarning = "3"
	colorInfo    = "4"
)

// Output handles console output formatting.
type Output struct {
	out       io.Writer
	err       io.Writer
	verbosity output.Verbosity
	isTTY     bool
	styles    map[output.StyleID]lipgloss.Style
}

// New creates a new Output instance.
func New(out io.Writer, err io.Writer, verbosity output.Verbosity) *Output {
	if out == nil {
		out = os.Stdout
	}

	if err == nil {
		err = os.Stderr
	}

	return &Output{
		out:       out,
		err:       err,
		verbosity: verbosity,
		isTTY:     detectTTY(out),
		styles:    defaultStyles(),
	}
}

// HasVerbosity returns true when output should be printed at given verbosity level.
func (o *Output) HasVerbosity(min output.Verbosity) bool {
	return o.verbosity >= min
}

// Printf prints a formatted message when verbosity is at least min.
func (o *Output) Printf(min output.Verbosity, format string, args ...any) {
	if o.verbosity < min {
		return
	}

	fmt.Fprintf(o.out, format, args...)
}

// Debugf prints debug output.
func (o *Output) Debugf(format string, args ...any) {
	o.Printf(output.VerbosityDebug, format, args...)
}

// Tracef prints trace output.
func (o *Output) Tracef(format string, args ...any) {
	o.Printf(output.VerbosityTrace, format, args...)
}

// DumpTrace prints trace-level dump.
func (o *Output) DumpTrace(title string, value any) {
	o.DumpStruct(output.VerbosityTrace, title, value, config.MaskedMap)
}

// Trace prints a trace message when verbosity is at least VerbosityTrace.
func (o *Output) Trace(msg string) {
	if o.verbosity < output.VerbosityTrace {
		return
	}

	fmt.Fprintf(o.out, "TRACE: %s\n", msg)
}

// style returns a lipgloss style with colors disabled for non-TTY output.
func (o *Output) style() lipgloss.Style {
	style := lipgloss.NewStyle()

	if !o.isTTY {
		return style
	}

	return style
}

func detectTTY(w io.Writer) bool {
	file, ok := w.(*os.File)
	if !ok {
		return false
	}

	return isatty.IsTerminal(file.Fd())
}

// DebugJSON prints a JSON-formatted value when verbosity is at least min.
func (o *Output) DebugJSON(min output.Verbosity, title string, v any) error {
	if o.verbosity < min {
		return nil
	}

	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}

	o.Section(title, string(data))
	return nil
}

// Diagnostic will handle a diagnostic message by printing it out.
//
// This method exists so it could also store the message for logging purposes.
func (o *Output) Diagnostic(d domain.Diagnostic) {
	o.printDiagnostic(d)
}

// printDiagnosticColored prints a single diagnostic with severity colors.
func (o *Output) printDiagnostic(d domain.Diagnostic) {
	line := d.String()

	if !o.isTTY {
		fmt.Fprintln(o.out, line)
		return
	}

	style := o.style()

	switch d.Severity {
	case domain.SeverityError:
		style = style.Foreground(lipgloss.Color(colorError))
	case domain.SeverityWarning:
		style = style.Foreground(lipgloss.Color(colorWarning))
	default:
		style = style.Foreground(lipgloss.Color(colorBase))
	}

	fmt.Fprintln(o.out, style.Render(line))

	if o.HasVerbosity(output.VerbosityDebug) {
		fmt.Fprintf(o.out,
			"  source=%q line=%d field=%q code=%q severity=%q\n",
			d.Source,
			d.Line,
			d.Field,
			d.Code,
			d.Severity,
		)
	}
}

// Style applies a semantic style to text when output is a TTY.
func (o *Output) Style(text string, styleID output.StyleID) string {
	if !o.isTTY {
		return text
	}

	style, ok := o.styles[styleID]
	if !ok {
		return text
	}

	return style.Render(text)
}

// defaultStyles returns the default semantic output styles.
func defaultStyles() map[output.StyleID]lipgloss.Style {
	return map[output.StyleID]lipgloss.Style{
		output.StyleBase: lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorBase)),

		output.StyleError: lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorError)).
			Bold(true),

		output.StyleWarning: lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorWarning)).
			Bold(true),

		output.StyleInfo: lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorInfo)),

		output.StyleImportantField: lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorError)),

		output.StyleHeader: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(colorBase)),
	}
}
