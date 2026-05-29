package output

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/SSHcom/inventory_import_privx_dev/internal/output"
	"github.com/SSHcom/inventory_import_privx_dev/internal/util"
)

// DumpStruct prints a struct recursively and masks fields with certain field names.
func (o *Output) DumpStruct(
	min output.Verbosity,
	title string,
	value any,
	maskedFields map[string]struct{},
) {
	if o.verbosity < min {
		return
	}

	o.Header(title)
	o.dumpValue("", reflect.ValueOf(value), 0, maskedFields)
}

// dumpValue prints a value recursively.
func (o *Output) dumpValue(
	name string,
	v reflect.Value,
	indent int,
	maskedFields map[string]struct{},
) {
	if !v.IsValid() {
		o.printScalar(name, "<nil>", indent)
		return
	}

	for v.Kind() == reflect.Pointer {
		if v.IsNil() {
			o.printScalar(name, "<nil>", indent)
			return
		}

		v = v.Elem()
	}

	switch v.Kind() {
	case reflect.Struct:
		o.dumpStruct(name, v, indent, maskedFields)

	case reflect.Map:
		o.dumpMap(name, v, indent, maskedFields)

	case reflect.Slice, reflect.Array:
		o.dumpSlice(name, v, indent, maskedFields)

	default:
		text := fmt.Sprint(v.Interface())

		if shouldMask(name, maskedFields) {
			text = util.MaskString(text)
		}

		o.printScalar(name, text, indent)
	}
}

// dumpStruct prints a struct recursively.
func (o *Output) dumpStruct(
	name string,
	v reflect.Value,
	indent int,
	maskedFields map[string]struct{},
) {
	prefix := strings.Repeat("  ", indent)

	if name != "" {
		fmt.Fprintf(o.out, "%s%s:\n", prefix, name)
		indent++
	}

	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		fieldInfo := t.Field(i)

		if !fieldInfo.IsExported() {
			continue
		}

		fieldName := fieldInfo.Name
		fieldValue := v.Field(i)

		o.dumpValue(fieldName, fieldValue, indent, maskedFields)
	}
}

// printScalar prints a single scalar value.
func (o *Output) printScalar(name string, value string, indent int) {
	prefix := strings.Repeat("  ", indent)

	if name == "" {
		fmt.Fprintf(o.out, "%s%s\n", prefix, value)
		return
	}

	fmt.Fprintf(o.out, "%s%-20s: %s\n", prefix, name, value)
}

// dumpSlice prints a slice or array recursively.
func (o *Output) dumpSlice(
	name string,
	v reflect.Value,
	indent int,
	maskedFields map[string]struct{},
) {
	prefix := strings.Repeat("  ", indent)

	if name != "" {
		fmt.Fprintf(o.out, "%s%s:\n", prefix, name)
		indent++
	}

	if v.Kind() == reflect.Slice && v.IsNil() {
		fmt.Fprintf(o.out, "%s<nil>\n", strings.Repeat("  ", indent))
		return
	}

	for i := 0; i < v.Len(); i++ {
		itemName := fmt.Sprintf("[%d]", i)
		o.dumpValue(itemName, v.Index(i), indent, maskedFields)
	}
}

// dumpMap prints a map recursively.
func (o *Output) dumpMap(
	name string,
	v reflect.Value,
	indent int,
	maskedFields map[string]struct{},
) {
	prefix := strings.Repeat("  ", indent)

	if name != "" {
		fmt.Fprintf(o.out, "%s%s:\n", prefix, name)
		indent++
	}

	if v.IsNil() {
		fmt.Fprintf(o.out, "%s<nil>\n", strings.Repeat("  ", indent))
		return
	}

	keys := v.MapKeys()

	sort.Slice(keys, func(i, j int) bool {
		return fmt.Sprint(keys[i].Interface()) <
			fmt.Sprint(keys[j].Interface())
	})

	for _, key := range keys {
		keyName := fmt.Sprint(key.Interface())

		o.dumpValue(
			keyName,
			v.MapIndex(key),
			indent,
			maskedFields,
		)
	}
}

// shouldMask reports whether a field name should be masked.
func shouldMask(name string, maskedFields map[string]struct{}) bool {
	_, ok := maskedFields[name]
	return ok
}
