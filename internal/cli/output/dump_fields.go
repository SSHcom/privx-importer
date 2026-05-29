package output

import (
	"fmt"
	"strings"

	"github.com/SSHcom/inventory_import_privx_dev/internal/field"
	"github.com/SSHcom/inventory_import_privx_dev/internal/output"
	"github.com/SSHcom/inventory_import_privx_dev/internal/util"
)

// DumpFields prints fields recursively when verbosity is at least min.
func (o *Output) DumpFields(
	min output.Verbosity,
	title string,
	fields field.Fields,
) {
	if o.verbosity < min {
		return
	}

	o.Header("\n" + title)

	o.printFields(fields, 0)
}

// printFields prints fields recursively using the given indentation level.
func (o *Output) printFields(fs field.Fields, indent int) {
	for _, key := range fs.Keys() {
		o.printField(fs[key], indent)
	}
}

// printField prints one field recursively.
func (o *Output) printField(f field.Field, indent int) {
	prefix := strings.Repeat("  ", indent)

	switch f.Kind() {
	case field.FieldKindComposite:
		composite, ok := f.(field.CompositeField)
		if !ok {
			fmt.Fprintf(o.out, "%s%s: <invalid composite field>\n", prefix, f.Name())
			return
		}
		fmt.Fprintf(o.out, "%s%s:\n", prefix, util.Leaf(f.Name()))

		o.printFields(composite.Fields(), indent+1)

	case field.FieldKindListComposite:
		list, ok := f.(field.ListCompositeField)
		if !ok {
			fmt.Fprintf(o.out, "%s%s: <invalid list-composite field>\n", prefix, f.Name())
			return
		}
		fmt.Fprintf(o.out, "%s%s:\n", prefix, util.Leaf(f.Name()))

		for i, item := range list.Items() {
			fmt.Fprintf(o.out, "%s[%d]\n", prefix, i)
			o.printFields(item, indent+2)
		}

	default:
		value := f.String()
		if field.IsSensitive(f.Name()) {
			value = util.MaskString(value)
		}
		fmt.Fprintf(
			o.out,
			"%s%-20s: %s\n",
			prefix,
			util.Leaf(f.Name()),
			value,
		)
	}
}
