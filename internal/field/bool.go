package field

import (
	"fmt"
	"strconv"
	"strings"
)

// BoolField implements a bool field.
type BoolField struct {
	FieldName  string
	FieldValue *bool
}

func (f BoolField) Kind() FieldKind {
	return FieldKindBool
}

func (f BoolField) Name() string {
	return f.FieldName
}

func (f BoolField) Equals(other Field) bool {
	o, ok := other.(BoolField)
	return ok && *f.FieldValue == *o.FieldValue
}

func (f BoolField) EqualsSubset(other Field) bool {
	o, ok := other.(BoolField)
	if !ok {
		return false
	}

	return *f.FieldValue == *o.FieldValue
}

func (f BoolField) Value() any {
	return *f.FieldValue
}

func (f BoolField) Encode() any {
	return *f.FieldValue
}

func (f BoolField) Set(value any) (Field, error) {
	if value == nil || value == "" {
		return BoolField{
			FieldName:  f.FieldName,
			FieldValue: nil,
		}, nil
	}

	v, ok := value.(bool)
	if !ok {
		return nil, fmt.Errorf(
			"%w: BoolField Set field=%s expected=bool actual=%T",
			ErrFieldValueTypeMismatch,
			f.FieldName,
			value,
		)
	}

	return BoolField{
		FieldName:  f.FieldName,
		FieldValue: &v,
	}, nil
}

func (f BoolField) Merge(value any) (Field, error) {

	switch v := value.(type) {
	case BoolField:
		value = *v.FieldValue

	case *BoolField:
		value = *v.FieldValue
	}

	v, ok := value.(bool)
	if !ok {
		return nil, fmt.Errorf(
			"%w: field=%s BoolField merge expected=bool actual=%T",
			ErrFieldValueTypeMismatch,
			f.FieldName,
			value,
		)
	}

	if *f.FieldValue != v {
		return nil, fmt.Errorf(
			"%w: field=%s current type is %t, merged type is %t",
			ErrScalarMergeConflict,
			f.FieldName,
			*f.FieldValue,
			v,
		)
	}

	return f, nil
}

func (f BoolField) Parse(value string) (Field, error) {
	v, err := parseBool(value)
	if err != nil {
		return nil, fmt.Errorf(
			"%w: field=%s kind=%s value=%q",
			err,
			f.FieldName,
			f.Kind(),
			value,
		)
	}

	return f.Set(v)
}

func (f BoolField) Text() string {
	return strconv.FormatBool(*f.FieldValue)
}

func (f BoolField) String() string {
	if f.FieldValue == nil {
		return "<nil>"
	}
	if *f.FieldValue {
		return "true"
	}

	return "false"
}

func parseBool(value string) (bool, error) {
	trimmed := strings.ToLower(value)

	switch trimmed {
	case "true", "1", "yes", "y", "on":
		return true, nil
	case "", "false", "0", "no", "n", "off":
		return false, nil
	default:
		return false, ErrFieldParseFailed
	}
}

// IsEmpty reports whether the boolean field has no meaningful value.
func (f BoolField) IsEmpty() bool {
	return f.FieldValue == nil
}

// Walk visits this field.
func (f BoolField) Walk(fn WalkFunc) error {
	return fn(f)
}

// Transform transforms this field.
func (f BoolField) Transform(fn TransformFunc) (Field, error) {
	return fn(f)
}

// Overlay returns this field with values from other overlaid on top.
// Second return parameter is the list of field paths that changed, including subfields.
func (f BoolField) Overlay(other Field, fn IncludeFunc) (Field, []string, error) {
	if !fn(f) {
		return f, nil, nil
	}
	o, ok := other.(BoolField)
	if !ok {
		return nil, nil, fmt.Errorf(
			"%w: bool field %q",
			ErrOverlayTypeMismatch,
			f.FieldName,
		)
	}

	if f.Equals(o) {
		return f, nil, nil
	}

	return o, []string{f.FieldName}, nil
}
