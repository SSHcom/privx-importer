package field

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/SSHcom/inventory_import_privx_dev/internal/util"
)

var (
	ErrFieldNameChanged    = errors.New("transform function changed field name")
	ErrOverlayTypeMismatch = errors.New("overlay field type mismatch")
)

// Fields is a map of Field types, indexed by the canonical field name.
type Fields map[string]Field

// Text returns Text value of field, if it exists, empty string otherwise.
func (fs Fields) Text(name string) string {
	field, ok := fs[name]
	if !ok || field == nil {
		return ""
	}

	return field.Text()
}

// String returns a printable form of all fields.
func (fs Fields) String() string {
	var b strings.Builder

	for _, key := range util.SortedKeys(fs) {
		fmt.Fprintf(&b, "  %s: %s\n", key, fs[key])
	}

	return b.String()
}

// Encode returns a map representation that can be marshaled into a JSON.
//
// Empty fields are ignored and field names will contain only the leaf.
func (fs Fields) Encode() map[string]any {
	raw := make(map[string]any)

	for key, f := range fs {
		if f.IsEmpty() {
			continue
		}

		raw[util.Leaf(key)] = f.Encode()
	}

	return raw
}

// WithKeys returns a new field map containing only the given keys.
func (fs Fields) WithKeys(keys []string) Fields {
	keySet := make(map[string]struct{}, len(keys))

	for _, key := range keys {
		keySet[key] = struct{}{}
	}

	result := make(Fields)

	for key, field := range fs {
		if _, ok := keySet[key]; ok {
			result[key] = field
		}
	}

	return result
}

// WithoutKeys returns a new field map excluding the given keys.
func (fs Fields) WithoutKeys(keys []string) Fields {
	keySet := make(map[string]struct{}, len(keys))

	for _, key := range keys {
		keySet[key] = struct{}{}
	}

	result := make(Fields)

	for key, field := range fs {
		if _, ok := keySet[key]; ok {
			continue
		}

		result[key] = field
	}

	return result
}

// EqualsSubset returns true if all fields in other match the receiver
// using subset comparison.
func (fs Fields) EqualsSubset(other Fields) bool {
	for name, otherField := range other {
		currentField, ok := fs[name]
		if !ok {
			return false
		}

		if !currentField.EqualsSubset(otherField) {
			return false
		}
	}

	return true
}

// Equals returns true if both field sets contain exactly same fields and values.
func (fs Fields) Equals(other Fields) bool {
	return fs.EqualsSubset(other) &&
		other.EqualsSubset(fs)
}

// Keys returns the field names as an ordered list.
func (fs Fields) Keys() []string {
	keys := make([]string, 0, len(fs))
	for key := range fs {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}

// KeyList returns the field names as a comma-separated string.
func (fs Fields) KeyList() string {
	return strings.Join(fs.Keys(), ",")
}

// GetByLeaf returns a child field by leaf name or canonical path suffix.
func (fs Fields) GetByLeaf(leaf string) (Field, bool) {
	for name, f := range fs {
		if name == leaf || strings.HasSuffix(name, "."+leaf) {
			return f, true
		}
	}

	return nil, false
}

// Join field paths
func JoinPath(path string, leaf string) string {
	if path == "" {
		return leaf
	}
	return path + "." + leaf
}

// Join fields paths where first one is indexed, so format is "path[0].leaf"
func JoinPathWithIndex(path string, index int, leaf string) string {
	if path == "" {
		return leaf
	}
	return fmt.Sprintf("%s[%d].%s", path, index, leaf)
}

// WalkFunc is called for each field visited by WalkFields.
type WalkFunc func(f Field) error

// walkFields walks through fields recursively.
// The walking will exit if the function returns error at any point.
func (fs Fields) Walk(fn WalkFunc) error {
	for _, f := range fs {

		if err := f.Walk(fn); err != nil {
			return err
		}
	}

	return nil
}

// TransformFunc returns a replacement for the visited field.
type TransformFunc func(f Field) (Field, error)

// Transform walks through fields recursively and returns transformed fields.
func (fields Fields) Transform(
	fn TransformFunc,
) (Fields, error) {
	new := Fields{}
	for name, f := range fields {
		transformed, err := f.Transform(fn)
		if err != nil {
			// In case of error, the original is returned.
			return fields, err
		}

		if transformed.Name() != name {
			return fields, fmt.Errorf(
				"%w: %s -> %s",
				ErrFieldNameChanged,
				name,
				transformed.Name(),
			)
		}

		new.Add(transformed)
	}

	return new, nil
}

// Add inserts or replaces a field using its canonical path as the key.
func (fs *Fields) Add(field Field) {
	(*fs)[field.Name()] = field
}

// Remove deletes a field by its canonical path.
func (fs *Fields) Remove(name string) {
	delete(*fs, name)
}

// Overlay returns a new field set where fields from other override receiver fields.
// It also returns the names of fields whose values changed.
// Only the fields present in the other are handled.
func (fs Fields) Overlay(other Fields, fn IncludeFunc) (Fields, []string, error) {
	next := make(Fields, len(fs)+len(other))
	changed := make([]string, 0, len(other))

	maps.Copy(next, fs)

	for _, name := range other.Keys() {
		otherField := other[name]

		currentField, exists := fs[name]
		if !exists {
			next[name] = otherField
			changed = append(changed, name)
			continue
		}

		overlaidField, fieldChanged, err := currentField.Overlay(otherField, fn)
		if err != nil {
			return nil, nil, err
		}

		next[name] = overlaidField
		changed = append(changed, fieldChanged...)
	}

	return next, changed, nil
}
