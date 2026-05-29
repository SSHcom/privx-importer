package field

import (
	"errors"
	"fmt"
	"strings"

	"github.com/SSHcom/inventory_import_privx_dev/internal/util"
)

var (
	ErrCompositeParseUnsupported = errors.New("Parser is not supported for composite fields")
)

// CompositeField implements a nested field composed of named subfields.
type CompositeField struct {
	path   string
	fields Fields
}

// Name returns the logical field name.
func (f CompositeField) Name() string {
	return f.path
}

// Kind returns the configured field kind.
func (f CompositeField) Kind() FieldKind {
	return FieldKindComposite
}

// Value returns the nested field values.
func (f CompositeField) Value() any {
	return f.fields
}

// Fields returns value in correct type, for those who know the field type already.
func (f CompositeField) Fields() Fields {
	return f.fields
}

func (f CompositeField) Encode() any {
	return f.fields.Encode()
}

// Text returns the nested field value as canonical text.
func (f CompositeField) Text() string {
	if len(f.fields) == 0 {
		return ""
	}

	parts := make([]string, 0, len(f.fields))

	for name, field := range f.fields {
		parts = append(parts, fmt.Sprintf("%s=%s", name, field.Text()))
	}

	return strings.Join(parts, ", ")
}

// String returns a human-readable representation of the nested fields.
func (f CompositeField) String() string {
	if len(f.fields) == 0 {
		return ""
	}

	lines := make([]string, 0, len(f.fields))

	for _, key := range f.fields.Keys() {
		field := f.fields[key]

		lines = append(
			lines,
			fmt.Sprintf(
				"%s: %s",
				util.Leaf(field.Name()),
				field.String(),
			),
		)
	}

	return strings.Join(lines, "\n")
}

// Equals reports whether the other field has an equal nested value.
func (f CompositeField) Equals(other Field) bool {
	o, ok := other.(CompositeField)
	if !ok {
		return false
	}

	if len(f.fields) != len(o.fields) {
		return false
	}

	return f.fields.EqualsSubset(o.fields) &&
		o.fields.EqualsSubset(f.fields)
}

// EqualsSubset reports whether the other composite field is a subset of this field.
func (f CompositeField) EqualsSubset(other Field) bool {
	o, ok := other.(CompositeField)
	if !ok {
		return false
	}

	return f.fields.EqualsSubset(o.fields)
}

// IsEmpty reports whether the composite field has no nested fields.
func (f CompositeField) IsEmpty() bool {
	return len(f.fields) == 0
}

// Set returns a new composite field with the given nested fields.
func (f CompositeField) Set(value any) (Field, error) {
	v, ok := value.(Fields)
	if !ok {
		return nil, fmt.Errorf(
			"%w: field=%s expected=field.Fields actual=%T",
			ErrFieldValueTypeMismatch,
			f.path,
			value,
		)
	}

	return CompositeField{
		path:   f.path,
		fields: cloneFields(v),
	}, nil
}

// Merge returns a new composite field merged with the given nested fields.
func (f CompositeField) Merge(value any) (Field, error) {
	v, ok := value.(Fields)
	if !ok {
		return nil, fmt.Errorf(
			"%w: field=%s expected=field.Fields actual=%T",
			ErrFieldValueTypeMismatch,
			f.path,
			value,
		)
	}

	merged := cloneFields(f.fields)

	for name, incoming := range v {
		current, exists := merged[name]
		if !exists {
			merged[name] = incoming
			continue
		}

		next, err := current.Merge(incoming.Value())
		if err != nil {
			return nil, err
		}

		merged[name] = next
	}

	return CompositeField{
		path:   f.path,
		fields: merged,
	}, nil
}

// Parse returns an error because composite fields require typed nested values.
//
// Composite fields are not formed by parsing, but by collecting already parsed fields together.
func (f CompositeField) Parse(value string) (Field, error) {
	return nil, fmt.Errorf(
		"%w: field=%s value=%q",
		ErrCompositeParseUnsupported,
		f.path,
		value,
	)
}

func cloneFields(input Fields) Fields {
	out := make(Fields, len(input))

	for name, field := range input {
		out[name] = field
	}

	return out
}

// Walk visits this composite field and its child fields.
func (f CompositeField) Walk(fn WalkFunc) error {
	if err := fn(f); err != nil {
		return err
	}

	return f.fields.Walk(fn)
}

// Transform transforms this composite field and its child fields.
func (f CompositeField) Transform(
	fn TransformFunc,
) (Field, error) {
	children, err := f.fields.Transform(fn)
	if err != nil {
		return nil, err
	}

	replacement := CompositeField{
		path:   f.path,
		fields: children,
	}

	return fn(replacement)
}

func (f *CompositeField) AddChild(child Field) {
	f.fields[child.Name()] = child
}

// Overlay returns this field with values from other overlaid on top.
// Second return parameter is the list of field paths that changed, including subfields.
func (f CompositeField) Overlay(other Field, fn IncludeFunc) (Field, []string, error) {
	if !fn(f) {
		return f, nil, nil
	}
	o, ok := other.(CompositeField)
	if !ok {
		return nil, nil, fmt.Errorf(
			"%w: composite field %q",
			ErrOverlayTypeMismatch,
			f.path,
		)
	}

	fields, changed, err := f.fields.Overlay(o.fields, fn)
	if err != nil {
		return nil, nil, err
	}

	if len(changed) == 0 {
		return f, nil, nil
	}

	return CompositeField{
		path:   f.path,
		fields: fields,
	}, changed, nil
}
