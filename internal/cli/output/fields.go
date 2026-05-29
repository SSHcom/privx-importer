package output

import (
	"fmt"
	"strings"

	"github.com/SSHcom/inventory_import_privx_dev/internal/field"
)

// Fields formats generic fields for console output.
func Fields(fields field.Fields) string {
	lines := make([]string, 0, len(fields))

	for name, f := range fields {
		lines = append(lines, fmt.Sprintf("  %s: %s", name, f.String()))
	}

	return strings.Join(lines, "\n")
}

// PrintFields prints generic fields directly.
func (o *Output) PrintFields(fields field.Fields) {
	fmt.Fprintln(o.out, Fields(fields))
}
