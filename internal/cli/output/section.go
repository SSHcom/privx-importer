package output

import (
	"fmt"

	"github.com/SSHcom/inventory_import_privx_dev/internal/output"
	"github.com/charmbracelet/lipgloss"
)

// Section prints a clearly separated output section.
func (o *Output) Section(title string, body string) {
	if o.isTTY {
		o.printStyledSection(title, body)
		return
	}

	fmt.Fprintf(o.out, "\n== %s ==\n%s\n", title, body)
}

// DebugSection prints a section only when debug output is enabled.
func (o *Output) DebugSection(title string, body string) {
	if !o.HasVerbosity(output.VerbosityDebug) {
		return
	}

	o.Section(title, body)
}

func (o *Output) printStyledSection(title string, body string) {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("63"))

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")).
		Padding(0, 1).
		MarginTop(1)

	content := titleStyle.Render(title) + "\n\n" + body
	fmt.Fprintln(o.out, boxStyle.Render(content))
}

// Header prints a section header.
func (o *Output) Header(title string) {
	o.Printf(output.VerbosityNormal, "\n%s\n", o.Style(title, output.StyleHeader))
}
