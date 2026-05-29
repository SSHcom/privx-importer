// Package field implements a system for holding typed information.
//
// It facilitates the input, comparison, processing and output of these structures.
// Basic structure of the importer consists of reading from files and API the structures,
// comparing them with business rules and then pushing the information back through the API.
package field

import (
	"errors"
	"fmt"
)

var (
	ErrFieldValueTypeMismatch = errors.New("field value type mismatch")
	ErrScalarMergeConflict    = errors.New("scalar merge conflict")
	ErrFieldParseFailed       = errors.New("field parse failed")
	ErrUnknownFieldKind       = errors.New("unknown field type")
	ErrInvalidListValue       = errors.New("invalid list value")
)

// Field represents a typed generic field value with merge semantics.
type Field interface {
	// Name returns the logical field name as a full path.
	// There are no array indices in the path.
	Name() string
	// Kind returns the configured field kind.
	Kind() FieldKind
	// Value returns the typed field value.
	Value() any
	// Text returns the field value as canonical text.
	Text() string
	// String returns a printable form of value.
	String() string

	// Equals reports whether the other field of similar type has an equal value.
	Equals(other Field) bool
	// EqualsSubset reports whether the other field is a subset of this field.
	EqualsSubset(other Field) bool
	// IsEmpty reports whether the field value can be considered "empty" or "unset".
	IsEmpty() bool

	// Set returns a new field with the given value.
	Set(value any) (Field, error)
	// Merge returns a new field merged with the given value.
	Merge(value any) (Field, error)
	// Parse returns a new field with the parameter parsed as the value.
	Parse(value string) (Field, error)
	// Overlay returns this field with values from other overlaid on top.
	// Second return parameter is the list of field paths that changed, including subfields.
	Overlay(other Field, fn IncludeFunc) (Field, []string, error)

	// Encode returns a representation that can be marshaled into a JSON.
	Encode() any

	// Walk traverses the field recursively and calls the function.
	Walk(fn WalkFunc) error
	// Transform traverses the field recursively and calls the function, potentially modifying as it goes.
	Transform(fn TransformFunc) (Field, error)
}

// Function to decide if a field is included in an operation.
type IncludeFunc func(f Field) bool

type FieldKind string

const (
	FieldKindString        FieldKind = "string"
	FieldKindList          FieldKind = "list"
	FieldKindBool          FieldKind = "bool"
	FieldKindInt           FieldKind = "int"
	FieldKindComposite     FieldKind = "composite"
	FieldKindListComposite FieldKind = "list_composite"
)

// IsKnownFieldKind reports whether kind is supported.
func IsKnownFieldKind(kind FieldKind) bool {
	switch kind {
	case FieldKindString, FieldKindList, FieldKindBool, FieldKindComposite, FieldKindListComposite, FieldKindInt:
		return true
	default:
		return false
	}
}

// A map of fields that should not be printed out in cleartext.
var SensitiveMap = map[string]struct{}{
	"ssh_host_public_keys.key": {},
	"host_certificate_raw":     {},
}

// Returns true if the field should not be printed out.
func IsSensitive(name string) bool {
	_, ok := SensitiveMap[name]
	return ok
}

// NewField creates an empty field instance for the given name and kind.
//
// Note that this is the only place where internal fields of the field implementations are handled from outside.
func NewField(name string, kind FieldKind) (Field, error) {
	switch kind {
	case FieldKindString:
		return StringField{
			FieldName:  name,
			FieldValue: "",
		}, nil

	case FieldKindBool:
		value := false
		return BoolField{
			FieldName:  name,
			FieldValue: &value,
		}, nil

	case FieldKindInt:
		return IntField{
			FieldName: name,
		}, nil

	case FieldKindList:
		return ListField{
			path: name,
			list: []string{},
		}, nil

	case FieldKindComposite:
		return CompositeField{
			path:   name,
			fields: Fields{},
		}, nil

	case FieldKindListComposite:
		return ListCompositeField{
			path:  name,
			items: []Fields{},
		}, nil

	default:
		return nil, fmt.Errorf(
			"%w: field=%s kind=%s",
			ErrUnknownFieldKind,
			name,
			kind,
		)
	}
}
