package field

import (
	"fmt"
	"strconv"
)

// StringField implements a string field.
// Note that there is no off channel indication whether value is set -- "" is interpreted as "not set" when necessary.
type StringField struct {
	FieldName  string
	FieldValue string
}

func (f StringField) Kind() FieldKind {
	return FieldKindString
}

func (f StringField) Name() string {
	return f.FieldName
}

func (f StringField) Equals(other Field) bool {
	o, ok := other.(StringField)
	return ok && f.FieldValue == o.FieldValue
}

func (f StringField) EqualsSubset(other Field) bool {
	o, ok := other.(StringField)
	if !ok {
		return false
	}

	return f.FieldValue == o.FieldValue
}

func (f StringField) Value() any {
	return f.FieldValue
}

func (f StringField) Set(value any) (Field, error) {

	// Would be better to fix this where this value comes from, not here.
	switch v := value.(type) {
	case float64:
		value = strconv.FormatFloat(v, 'f', -1, 64)
	}

	v, ok := value.(string)
	if !ok {
		return nil, fmt.Errorf(
			"%w: StringField %q Set expected a string but got %T",
			ErrFieldValueTypeMismatch,
			f.FieldName,
			value,
		)
	}

	return StringField{
		FieldName:  f.FieldName,
		FieldValue: v,
	}, nil
}

func (f StringField) Merge(value any) (Field, error) {

	switch v := value.(type) {
	case StringField:
		value = v.FieldValue

	case *StringField:
		value = v.FieldValue
	}

	v, ok := value.(string)
	if !ok {
		return nil, fmt.Errorf(
			"%w: StringField %q Merge expected a string, actual=%T",
			ErrFieldValueTypeMismatch,
			f.FieldName,
			value,
		)
	}

	if f.FieldValue == "" {
		return StringField{
			FieldName:  f.FieldName,
			FieldValue: v,
		}, nil
	}

	if v == "" {
		return f, nil
	}

	if f.FieldValue != v {
		return nil, fmt.Errorf(
			"%w: field=%s current=%q incoming=%q",
			ErrScalarMergeConflict,
			f.FieldName,
			f.FieldValue,
			v,
		)
	}

	return f, nil
}

func (f StringField) Text() string {
	return f.FieldValue
}

func (f StringField) Encode() any {
	return f.FieldValue
}

func (f StringField) Parse(value string) (Field, error) {
	return f.Set(value)
}

func (f StringField) String() string {
	return f.FieldValue
}

// IsEmpty reports whether the string field has no meaningful value.
func (f StringField) IsEmpty() bool {
	return f.FieldValue == ""
}

// Walk visits this field.
func (f StringField) Walk(fn WalkFunc) error {
	return fn(f)
}

// Transform transforms this field.
func (f StringField) Transform(fn TransformFunc) (Field, error) {
	return fn(f)
}

// Overlay returns this field with values from other overlaid on top.
// Second return parameter is the list of field paths that changed, including subfields.
func (f StringField) Overlay(other Field, fn IncludeFunc) (Field, []string, error) {
	if !fn(f) {
		return f, nil, nil
	}
	o, ok := other.(StringField)
	if !ok {
		return nil, nil, fmt.Errorf(
			"%w: string field %q",
			ErrOverlayTypeMismatch,
			f.FieldName,
		)
	}

	if f.FieldValue == o.FieldValue {
		return f, nil, nil
	}

	return o, []string{f.FieldName}, nil
}
