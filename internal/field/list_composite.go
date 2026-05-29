package field

import (
	"errors"
	"fmt"
	"strings"

	"github.com/SSHcom/inventory_import_privx_dev/internal/util"
)

var (
	// ErrListCompositeParseUnsupported is returned when a list-composite field is parsed directly from text.
	ErrListCompositeParseUnsupported = errors.New("list composite field cannot be parsed from string")
)

// ListCompositeField implements a list of composite field objects.
type ListCompositeField struct {
	path  string
	items []Fields
}

// Name returns the logical field name.
func (f ListCompositeField) Name() string {
	return f.path
}

// Kind returns the configured field kind.
func (f ListCompositeField) Kind() FieldKind {
	return FieldKindListComposite
}

// Value returns the list as JSON-compatible values.
func (f ListCompositeField) Value() any {
	return f.items
}

// Items returns value in correct type, for those who know the field type already.
func (f ListCompositeField) Items() []Fields {
	return f.items
}

func (f ListCompositeField) Encode() any {

	items := make([]any, 0, len(f.items))

	for _, item := range f.items {
		if len(item) == 0 {
			continue
		}

		items = append(items, item.Encode())
	}

	return items

}

// Text returns the field value as canonical text.
func (f ListCompositeField) Text() string {
	items := make([]string, 0, len(f.items))

	for _, item := range f.items {
		items = append(items, "{"+strings.TrimSpace(item.String())+"}")
	}

	return strings.Join(items, ", ")
}

// String returns a human-readable representation of the composite list.
func (f ListCompositeField) String() string {
	if len(f.items) == 0 {
		return ""
	}

	lines := make([]string, 0, len(f.items))

	for i, item := range f.items {
		lines = append(lines, fmt.Sprintf("[%d]", i))

		for _, key := range item.Keys() {
			field := item[key]

			lines = append(
				lines,
				fmt.Sprintf(
					"  %s: %s",
					util.Leaf(field.Name()),
					field.String(),
				),
			)
		}
	}

	return strings.Join(lines, "\n")
}

// Equals reports whether the other field has equal composite items.
// Item order must also match.
func (f ListCompositeField) Equals(other Field) bool {
	o, ok := other.(ListCompositeField)
	if !ok {
		// fmt.Printf("ListCompositeField.Equals wrong type\n")
		return false
	}

	if len(f.items) != len(o.items) {
		// fmt.Printf("ListCompositeField.Equals wrong length\n")
		return false
	}

	for i, item := range f.items {
		if !item.Equals(o.items[i]) {
			// fmt.Printf("ListCompositeField.Equals item %d not equal, %q NOT EQUAL TO %q\n", i, item.String(), o.items[i].String())
			return false
		}
	}
	return true
}

// EqualsSubset reports whether the other field is a subset of this field.
func (f ListCompositeField) EqualsSubset(other Field) bool {
	o, ok := other.(ListCompositeField)
	if !ok {
		return false
	}

	return fieldsListContainsAll(
		f.items,
		o.items,
		func(current Fields, other Fields) bool {
			return current.EqualsSubset(other)
		},
	)
}

// IsEmpty reports whether the field has no composite items.
func (f ListCompositeField) IsEmpty() bool {
	return len(f.items) == 0
}

// Set returns a new list-composite field with the given value.
func (f ListCompositeField) Set(value any) (Field, error) {
	switch v := value.(type) {
	case []Fields:
		return ListCompositeField{
			path:  f.path,
			items: cloneFieldsList(v),
		}, nil

	case ListCompositeField:
		return ListCompositeField{
			path:  f.path,
			items: cloneFieldsList(v.items),
		}, nil

	default:
		return nil, fmt.Errorf(
			"%w: ListCompositeField %q Set expected=[]field.Fields actual=%T %q",
			ErrFieldValueTypeMismatch,
			f.path,
			value,
			value,
		)
	}
}

// Merge returns a new list-composite field with appended composite items.
func (f ListCompositeField) Merge(value any) (Field, error) {
	merged := cloneFieldsList(f.items)

	switch v := value.(type) {
	case Fields:
		if len(v) > 0 {
			merged = append(merged, cloneFields(v))
		}

	case []Fields:
		merged = append(merged, cloneFieldsList(v)...)

	case ListCompositeField:
		merged = append(merged, cloneFieldsList(v.items)...)

	default:
		return nil, fmt.Errorf(
			"%w: ListCompositeField %q Merge expected=field.Fields|[]field.Fields actual=%T",
			ErrFieldValueTypeMismatch,
			f.path,
			value,
		)
	}

	return ListCompositeField{
		path:  f.path,
		items: merged,
	}, nil
}

// Parse returns an error because list-composite fields require typed nested values.
func (f ListCompositeField) Parse(value string) (Field, error) {
	return nil, fmt.Errorf(
		"%w: field=%s value=%q",
		ErrListCompositeParseUnsupported,
		f.path,
		value,
	)
}

// fieldsListContainsAll checks that every item in other
// has a matching unused item in current.
func fieldsListContainsAll(
	current []Fields,
	other []Fields,
	matches func(current Fields, other Fields) bool,
) bool {
	used := make([]bool, len(current))

	for _, otherItem := range other {
		found := false

		for i, currentItem := range current {
			if used[i] {
				continue
			}

			if matches(currentItem, otherItem) {
				used[i] = true
				found = true
				break
			}
		}

		if !found {
			return false
		}
	}

	return true
}

func cloneFieldsList(input []Fields) []Fields {
	out := make([]Fields, 0, len(input))

	for _, item := range input {
		out = append(out, cloneFields(item))
	}

	return out
}

// Walk visits this list-composite field and its item fields.
func (f ListCompositeField) Walk(fn WalkFunc) error {
	if err := fn(f); err != nil {
		return err
	}

	for _, item := range f.items {
		if err := item.Walk(fn); err != nil {
			return err
		}
	}

	return nil
}

// Transform transforms this list-composite field and its item fields.
func (f ListCompositeField) Transform(fn TransformFunc) (Field, error) {
	newItems := make([]Fields, 0, len(f.items))

	for _, item := range f.items {

		transformed, err := item.Transform(fn)
		if err != nil {
			return nil, err
		}

		newItems = append(newItems, transformed)
	}

	replacement := ListCompositeField{
		path:  f.path,
		items: newItems,
	}

	return fn(replacement)
}

func (f *ListCompositeField) AddItem(item Fields) {
	f.items = append(f.items, item)
}

// Overlay returns this field with values from other overlaid on top.
// Second return parameter is the list of field paths that changed, including subfields.
func (f ListCompositeField) Overlay(other Field, fn IncludeFunc) (Field, []string, error) {
	if !fn(f) {
		return f, nil, nil
	}

	o, ok := other.(ListCompositeField)
	if !ok {
		return nil, nil, fmt.Errorf(
			"%w: list composite field %q",
			ErrOverlayTypeMismatch,
			f.path,
		)
	}

	next := make([]Fields, 0, max(len(f.items), len(o.items)))
	changed := make([]string, 0)

	for i, item := range f.items {
		if i >= len(o.items) {
			next = append(next, item)

			continue
		}

		overlayed, itemChanged, err := item.Overlay(o.items[i], fn)
		if err != nil {
			return nil, nil, err
		}

		next = append(next, overlayed)
		changed = append(changed, itemChanged...)
	}

	for i := len(f.items); i < len(o.items); i++ {
		next = append(next, o.items[i])

		if fn(o) {
			changed = append(changed, f.path)
		}
	}

	if len(changed) == 0 {
		return f, nil, nil
	}

	return ListCompositeField{
		path:  f.path,
		items: next,
	}, changed, nil
}
