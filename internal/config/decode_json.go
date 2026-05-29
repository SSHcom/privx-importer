package config

import (
	"fmt"

	"github.com/SSHcom/inventory_import_privx_dev/internal/field"
	"github.com/SSHcom/inventory_import_privx_dev/internal/output"
	"github.com/SSHcom/inventory_import_privx_dev/internal/util"
)

// DecodeFields takes the raw JSON data from PrivX API response and interprets the configured field values.
func (fc FieldCatalog) DecodeFields(
	out output.Output,
	raw map[string]any,
) (field.Fields, error) {
	return fc.decodeFields(out, "", raw, fc.Fields)
}

// decodeFields decodes raw values that have matching field definitions.
func (fc FieldCatalog) decodeFields(
	out output.Output,
	path string,
	raw map[string]any,
	defs map[string]FieldDef,
) (field.Fields, error) {
	fields := make(field.Fields)

	for leaf, value := range raw {
		path := util.JoinPath(path, leaf)

		def, ok := defs[path]
		if !ok {
			out.Printf(
				output.VerbosityInfo,
				"Skipping unknown API field %q\n",
				path,
			)

			continue
		}

		f, err := fc.decodeField(out, path, def, value)
		if err != nil {
			return nil, err
		}

		if f.IsEmpty() {
			continue
		}

		fields[path] = f
	}

	return fields, nil
}

// decodeField decodes one raw value according to its field definition.
func (fc FieldCatalog) decodeField(
	out output.Output,
	name string,
	def FieldDef,
	value any,
) (field.Field, error) {
	if len(def.Children) == 0 {
		f, err := field.NewField(name, field.FieldKind(def.Kind))
		if err != nil {
			return nil, err
		}

		return f.Set(value)
	}

	switch field.FieldKind(def.Kind) {
	case field.FieldKindComposite:
		return fc.decodeCompositeField(out, name, def, value)

	case field.FieldKindListComposite:
		return fc.decodeListCompositeField(out, name, def, value)

	default:
		return nil, fmt.Errorf(
			"%w: structured field %q has unsupported kind %q",
			field.ErrFieldValueTypeMismatch,
			name,
			def.Kind,
		)
	}
}

// decodeCompositeField decodes a raw object into a composite field.
func (fc FieldCatalog) decodeCompositeField(
	out output.Output,
	path string,
	def FieldDef,
	value any,
) (field.Field, error) {

	if value == nil {
		f, err := field.NewField(path, field.FieldKindComposite)
		if err != nil {
			return nil, err
		}
		return f, nil
	}

	// The value from API should be a subrecord, an array of values.
	raw, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf(
			"%w: Decoding API reply, CompositeField %q expected=map[string]any actual=%T %q",
			field.ErrFieldValueTypeMismatch,
			path,
			value,
			value,
		)
	}

	children, err := fc.decodeFields(out, path, raw, childDefs(def))
	if err != nil {
		return nil, err
	}

	f, err := field.NewField(path, field.FieldKindComposite)
	if err != nil {
		return nil, err
	}
	return f.Set(children)
}

// childDefs returns child field definitions indexed by field name.
func childDefs(def FieldDef) map[string]FieldDef {
	defs := make(map[string]FieldDef, len(def.Children))

	for _, child := range def.Children {
		defs[child.Name] = child
		// fmt.Printf("child index %q\n", child.Name)
	}

	return defs
}

// decodeListCompositeField decodes a raw object list into a list-composite field.
// Be careful in this function about where index is included in path and where not.
func (fc FieldCatalog) decodeListCompositeField(
	out output.Output,
	path string,
	def FieldDef,
	value any,
) (field.Field, error) {
	rawList, ok := value.([]any)
	if !ok {
		return nil, fmt.Errorf(
			"%w: ListCompositeField %q expected=[]any actual=%T %q",
			field.ErrFieldValueTypeMismatch,
			path,
			value,
			value,
		)
	}

	items := make([]field.Fields, 0, len(rawList))

	for _, rawItem := range rawList {
		raw, ok := rawItem.(map[string]any)
		if !ok {
			return nil, fmt.Errorf(
				"%w: ListCompositeField %q item expected=map[string]any actual=%T %q",
				field.ErrFieldValueTypeMismatch,
				// TODO In error handling, we want to see which element of the array this was.
				path,
				rawItem,
				rawItem,
			)
		}

		// prefix for children is without index, because when values are compared to existing ones,
		// the fields are matched together by field name.
		// It is the job of the upper logic to decide, which elements of first array are compared to what element in the second.
		// Therefore the field names must be independent of their location in array.
		children, err := fc.decodeFields(out, path, raw, childDefs(def))
		if err != nil {
			return nil, err
		}

		if len(children) == 0 {
			continue
		}

		items = append(items, children)
	}

	f, err := field.NewField(path, field.FieldKindListComposite)
	if err != nil {
		return nil, err
	}
	f, err = f.Set((items))
	if err != nil {
		return nil, err
	}
	return f, nil
}
