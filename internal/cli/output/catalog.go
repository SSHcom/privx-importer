package output

import (
	"sort"
	"strings"

	"github.com/SSHcom/inventory_import_privx_dev/internal/config"
	interout "github.com/SSHcom/inventory_import_privx_dev/internal/output"
)

// PrintFieldCatalog prints field catalog in a readable tree format.
func PrintFieldCatalog(out interout.Output, min interout.Verbosity, catalog config.FieldCatalog) {
	out.Header("Field catalog\n")

	for _, def := range sortedFieldDefs(catalog.Fields) {
		printFieldDef(out, min, def, 0)
	}
}

// PrintFieldCatalog prints field catalog indexes in a readable format.
func PrintFieldCatalogIndex(out interout.Output, min interout.Verbosity, catalog config.FieldCatalog) {
	out.Header("Field catalog index\n")

	// Keys to alphabetical order
	keys := make([]string, 0, len(catalog.InputKeys))
	for key := range catalog.InputKeys {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, inputKey := range keys {
		canonical := catalog.InputKeys[inputKey]

		out.Printf(
			min,
			"%-30s → %s\n",
			inputKey,
			canonical,
		)
	}
}

// printFieldDef prints one field definition and its children recursively.
func printFieldDef(out interout.Output, min interout.Verbosity, def config.FieldDef, depth int) {
	indent := strings.Repeat("  ", depth)

	out.Printf(min, "%s- %s (%s)\n", indent, def.Name, def.Kind)

	if len(def.InputKeys) > 0 {
		out.Printf(min, "%s  input keys: %s\n", indent, strings.Join(def.InputKeys, ", "))
	}

	if def.Default != "" {
		out.Printf(min, "%s  default: %s\n", indent, def.Default)
	}

	if def.Translator != "" {
		out.Printf(
			min,
			"%s  %s: %s\n",
			indent,
			out.Style("translator", interout.StyleImportantField),
			def.Translator,
		)
	}

	if len(def.Validators) > 0 {
		out.Printf(
			min,
			"%s  %s: %s\n",
			indent,
			out.Style("validators", interout.StyleImportantField),
			joinValidatorIDs(def.Validators),
		)
	}

	if def.Management != "" {
		out.Printf(min, "%s  managed: %s\n", indent, out.Style(string(def.Management), interout.StyleImportantField))
	}

	for _, child := range def.Children {
		printFieldDef(out, min, child, depth+1)
	}
}

// sortedFieldDefs returns field definitions sorted by canonical field name.
func sortedFieldDefs(fields map[string]config.FieldDef) []config.FieldDef {
	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	result := make([]config.FieldDef, 0, len(keys))
	for _, key := range keys {
		result = append(result, fields[key])
	}

	return result
}

// joinValidatorIDs joins validator identifiers for output.
func joinValidatorIDs(ids []config.ValidatorID) string {
	values := make([]string, 0, len(ids))
	for _, id := range ids {
		if id == "" {
			continue
		}
		values = append(values, string(id))
	}

	return strings.Join(values, ", ")
}
