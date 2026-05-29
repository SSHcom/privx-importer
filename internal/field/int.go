package field

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// IntField implements an integer field.
// Note that there is no off channel indication whether value is set -- 0 is interpreted as "not set" when necessary.
type IntField struct {
	FieldName  string
	FieldValue int
}

// Name returns the logical field name.
func (f IntField) Name() string { return f.FieldName }

// Kind returns the configured field kind.
func (f IntField) Kind() FieldKind { return FieldKindInt }

// Value returns the typed field value.
func (f IntField) Value() any { return f.FieldValue }

// Text returns the field value as canonical text.
func (f IntField) Text() string { return strconv.Itoa(f.FieldValue) }

func (f IntField) Encode() any {
	return f.FieldValue
}

// String returns a printable form of value.
func (f IntField) String() string { return f.Text() }

// IsEmpty reports whether the field value can be considered empty.
func (f IntField) IsEmpty() bool { return f.FieldValue == 0 }

// Equals reports whether the other field has an equal value.
func (f IntField) Equals(other Field) bool {
	o, ok := other.(IntField)
	return ok && f.FieldValue == o.FieldValue
}

// EqualsSubset reports whether the other field is a subset of this field.
func (f IntField) EqualsSubset(other Field) bool {
	return f.Equals(other)
}

// Parse returns a new field with the parameter parsed as the value.
func (f IntField) Parse(value string) (Field, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return f, nil
	}

	v, err := strconv.Atoi(value)
	if err != nil {
		return nil, fmt.Errorf(
			"%w: field=%s expected=int actual=%q",
			ErrFieldValueTypeMismatch,
			f.FieldName,
			value,
		)
	}

	return IntField{
		FieldName:  f.FieldName,
		FieldValue: v,
	}, nil
}

// Merge returns a new field merged with the given value.
func (f IntField) Merge(value any) (Field, error) {
	switch v := value.(type) {
	case IntField:
		value = v.FieldValue

	case *IntField:
		value = v.FieldValue

	case float64:
		value = int(v)
	}

	i, ok := value.(int)
	if !ok {
		return nil, fmt.Errorf(
			"%w: field=%s expected=int actual=%T",
			ErrFieldValueTypeMismatch,
			f.FieldName,
			value,
		)
	}

	if f.FieldValue == 0 {
		return IntField{
			FieldName:  f.FieldName,
			FieldValue: i,
		}, nil
	}

	if i == 0 {
		return f, nil
	}

	if f.FieldValue != i {
		return nil, fmt.Errorf(
			"%w: field=%s current=%d incoming=%d",
			ErrScalarMergeConflict,
			f.FieldName,
			f.FieldValue,
			i,
		)
	}

	return f, nil
}

// Set returns a new field with the given value.
func (f IntField) Set(value any) (Field, error) {
	var newValue int
	switch v := value.(type) {
	case int:
		newValue = v

	case int64:
		newValue = int(v)

	case float64:
		if v != math.Trunc(v) {
			return nil, fmt.Errorf(
				"%w: field=%s expected=int actual=%v",
				ErrFieldValueTypeMismatch,
				f.FieldName,
				v,
			)
		}

		newValue = int(v)

	case string:
		return f.Parse(v)

	case IntField:
		newValue = v.FieldValue

	case *IntField:
		newValue = v.FieldValue

	default:
		return nil, fmt.Errorf(
			"%w: field=%s expected=int actual=%T",
			ErrFieldValueTypeMismatch,
			f.FieldName,
			value,
		)
	}

	return IntField{
		FieldName:  f.FieldName,
		FieldValue: newValue,
	}, nil
}

// Walk visits this field.
func (f IntField) Walk(fn WalkFunc) error {
	return fn(f)
}

// Transform transforms this field.
func (f IntField) Transform(fn TransformFunc) (Field, error) {
	return fn(f)
}

// Overlay returns this field with values from other overlaid on top.
// Second return parameter is the list of field paths that changed, including subfields.
func (f IntField) Overlay(other Field, fn IncludeFunc) (Field, []string, error) {
	if !fn(f) {
		return f, nil, nil
	}

	o, ok := other.(IntField)
	if !ok {
		return nil, nil, fmt.Errorf(
			"%w: int field %q",
			ErrOverlayTypeMismatch,
			f.FieldName,
		)
	}

	if f.FieldValue == o.FieldValue {
		return f, nil, nil
	}

	return o, []string{f.FieldName}, nil
}
