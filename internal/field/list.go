package field

import (
	"fmt"
	"strings"
)

// ListField implements a list field.
type ListField struct {
	path string
	list []string
}

func (f ListField) Name() string {
	return f.path
}

func (f ListField) Kind() FieldKind {
	return FieldKindList
}

func (f ListField) Equals(other Field) bool {
	o, ok := other.(ListField)
	if !ok {
		return false
	}

	return stringSlicesEqualUnordered(f.list, o.list)
}

func (f ListField) EqualsSubset(other Field) bool {
	o, ok := other.(ListField)
	if !ok {
		return false
	}

	// build lookup for current values
	set := make(map[string]int)
	for _, v := range f.list {
		set[v]++
	}

	// check that all other values exist
	for _, v := range o.list {
		if set[v] == 0 {
			return false
		}
		set[v]--
	}

	return true
}

func (f ListField) Value() any {
	return f.list
}

func (f ListField) Encode() any {
	return f.list
}

// Set returns a new list field with the given value.
//
// Accepted input types:
//   - []string
//   - []any, where every element must be a string (typical JSON unmarshal result)
func (f ListField) Set(value any) (Field, error) {
	var out []string

	switch v := value.(type) {
	case []string:
		out = append([]string(nil), v...)

	case []any:
		out = make([]string, 0, len(v))

		for _, item := range v {
			s, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf(
					"%w: ListField %q Set expected []string but got %T",
					ErrFieldValueTypeMismatch,
					f.path,
					item,
				)
			}

			out = append(out, s)
		}

	case nil:
		out = make([]string, 0)

	default:
		return nil, fmt.Errorf(
			"%w: ListField %q Set expected []string but got %T",
			ErrFieldValueTypeMismatch,
			f.path,
			value,
		)
	}

	return ListField{
		path: f.path,
		list: out,
	}, nil
}

func (f ListField) Merge(value any) (Field, error) {
	var incoming []string

	switch v := value.(type) {
	case []string:
		incoming = append([]string(nil), v...)
	case string:
		if v != "" {
			incoming = []string{v}
		} else {
			incoming = []string{}
		}
	case ListField:
		incoming = append([]string(nil), v.list...)
	default:
		return nil, fmt.Errorf(
			"%w: ListField %q Merge expected=[]string|string and actual was %T",
			ErrFieldValueTypeMismatch,
			f.path,
			value,
		)
	}

	if len(incoming) == 0 {
		return f, nil
	}

	merged := append([]string(nil), f.list...)
	seen := make(map[string]struct{}, len(merged))

	for _, item := range merged {
		seen[item] = struct{}{}
	}

	for _, item := range incoming {
		if _, exists := seen[item]; exists {
			continue
		}

		merged = append(merged, item)
		seen[item] = struct{}{}
	}

	return ListField{
		path: f.path,
		list: merged,
	}, nil
}

func (f ListField) Text() string {
	return strings.Join(f.list, ",")
}

// Parse converts a string into a ListField.
// Comma-separated values are trimmed, empty items are ignored.
func (f ListField) Parse(value string) (Field, error) {
	parts := strings.Split(value, ",")

	items := make([]string, 0, len(parts))

	for _, part := range parts {
		item := strings.TrimSpace(part)
		if item == "" {
			continue
		}

		items = append(items, item)
	}

	return ListField{
		path: f.path,
		list: items,
	}, nil
}

func (f ListField) String() string {
	return strings.Join(f.list, ", ")
}

func stringSlicesEqualUnordered(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	counts := make(map[string]int)

	for _, v := range a {
		counts[v]++
	}

	for _, v := range b {
		counts[v]--
		if counts[v] < 0 {
			return false
		}
	}

	return true
}

// IsEmpty reports whether the list field has no values.
func (f ListField) IsEmpty() bool {
	return len(f.list) == 0
}

// Walk visits this field.
func (f ListField) Walk(fn WalkFunc) error {
	return fn(f)
}

// Transform transforms this field.
func (f ListField) Transform(fn TransformFunc) (Field, error) {
	return fn(f)
}

// Overlay returns this field with values from other overlaid on top.
// Second return parameter is the list of field paths that changed, including subfields.
func (f ListField) Overlay(other Field, fn IncludeFunc) (Field, []string, error) {
	if !fn(f) {
		return f, nil, nil
	}

	o, ok := other.(ListField)
	if !ok {
		return nil, nil, fmt.Errorf(
			"%w: list field %q",
			ErrOverlayTypeMismatch,
			f.path,
		)
	}

	if f.Equals(o) {
		return f, nil, nil
	}

	return o, []string{f.path}, nil
}
