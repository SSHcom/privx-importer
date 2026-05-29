package config

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/SSHcom/inventory_import_privx_dev/internal/domain"
	"github.com/SSHcom/inventory_import_privx_dev/internal/field"
	"github.com/SSHcom/inventory_import_privx_dev/internal/output"
	"github.com/SSHcom/inventory_import_privx_dev/internal/util"
)

var (
	// ErrNoFieldDef is returned if field definition is not found in catalog.
	ErrNoFieldDef        = errors.New("no field definition found")
	ErrInvalidManagement = errors.New("invalid management")
)

// FieldDef is the definition of a generic field both for file input and API.
type FieldDef struct {
	// Name is the canonical name of field, including the path for structural fields.
	Name string `toml:"name"`
	// Kind is the data type of field as defined by FieldKind.
	Kind field.FieldKind `toml:"kind"`
	// InputKeys is a list of all possible normalized column names.
	InputKeys []string `toml:"input_keys"`
	// Children are the fields of a composite type.
	Children []FieldDef `toml:"children"`
	// Default is the value used, if input is an empty string.
	Default string `toml:"default"`
	// Translator is used at end of normalisation for example to convert human readable values to UUID for API use.
	Translator TranslatorID `toml:"translator"`
	// Validators is the optional list of validations performed.
	Validators []ValidatorID `toml:"validators"`
	// Management can be used to manually set a field to be managed or ignored.
	Management Management `toml:"management"`
}

// Depth returns the nesting depth of field definition.
func (f FieldDef) Depth() int {
	return strings.Count(f.Name, ".")
}

// Walk calls fn for field definition and all children recursively.
func (f FieldDef) Walk(fn func(FieldDef)) {
	fn(f)

	for _, child := range f.Children {
		child.Walk(fn)
	}
}

// FieldCatalog defines all possible fields and their attributes.
type FieldCatalog struct {
	Fields map[string]FieldDef
	// InputKeys is a mapping from input_key to corresponding canonical name.
	InputKeys map[string]string
	// structurals contains the field definitions of all structured types, in order of decreasing level of hierarchy.
	structurals []FieldDef
	// flatFields is a mapping from canonical paths to field definitions.
	flatFields map[string]FieldDef
}

func NewFieldCatalog() FieldCatalog {
	return FieldCatalog{
		Fields:     make(map[string]FieldDef),
		InputKeys:  make(map[string]string),
		flatFields: make(map[string]FieldDef),
	}
}

// TranslatorID identifies a field translator.
type TranslatorID string

const (
	TranslatorAccessGroup TranslatorID = "access_group"
	TranslatorService     TranslatorID = "service"
	TranslatorRole        TranslatorID = "role"
)

// TranslatorIDSet contains all supported translator identifiers.
var TranslatorIDSet = map[TranslatorID]struct{}{
	TranslatorAccessGroup: {},
	TranslatorService:     {},
	TranslatorRole:        {},
}

type ValidatorID string

const (
	ValidatorServices     ValidatorID = "services"
	ValidatorDisabled     ValidatorID = "disabled"
	ValidatorSSHPublicKey ValidatorID = "ssh_key"
)

// Management can be used to manually set a field to be managed or ignored.
type Management string

const (
	ManagementManaged Management = "managed"
	ManagementIgnored Management = "ignored"
)

// Validate checks that management value is valid.
func (m Management) Validate() error {
	switch m {
	case
		"",
		ManagementManaged,
		ManagementIgnored:
		return nil

	default:
		return fmt.Errorf(
			"%w: %q",
			ErrInvalidManagement,
			m,
		)
	}
}

// DecodeStringFields takes raw string values (read from config) and interprets them using configured field definitions.
func (fc FieldCatalog) DecodeStringFields(raw map[string]string) (field.Fields, error) {
	fields := make(field.Fields)

	for name, value := range raw {
		def, ok := fc.Fields[name]
		if !ok {
			return nil, fmt.Errorf("%w: %s", ErrNoFieldDef, name)
		}

		f, err := field.NewField(name, field.FieldKind(def.Kind))
		if err != nil {
			return nil, err
		}

		// Note: uses Parse
		f, err = f.Parse(value)
		if err != nil {
			return nil, err
		}

		if f.IsEmpty() {
			continue
		}

		fields[name] = f
	}

	return fields, nil
}

// FieldDef returns the field definition for a canonical field path.
func (fc FieldCatalog) FieldDef(path string) (FieldDef, bool) {
	// The catalog does not use array indices -- all records of array share the same definitions
	path = util.StripIndexes(path)
	def, ok := fc.flatFields[path]
	return def, ok
}

// ResolveInputKey returns canonical name using index.
func (fc FieldCatalog) ResolveInputKey(input string) (string, bool) {
	name, ok := fc.InputKeys[input]
	return name, ok
}

var (
	// ErrFieldNotFound indicates that referenced field does not exist.
	ErrFieldNotFound = errors.New("field not found")
)

// FieldsFromMap converts raw map values into generic fields.
func (fc FieldCatalog) FieldsFromMap(out output.Output, values map[string]any) (field.Fields, error) {
	fields := make(field.Fields)

	for name, value := range values {
		def, ok := fc.FieldDef(name)
		if !ok {
			out.Diagnostic(domain.Diagnostic{
				Message: fmt.Sprintf("Field %q in FieldsFromMap was not found in field catalog", name),
			})
			continue
		}

		f, err := field.NewField(name, field.FieldKind(def.Kind))
		if err != nil {
			return nil, err
		}
		f, err = f.Set(value)
		if err != nil {
			out.Diagnostic(domain.Diagnostic{
				Message: fmt.Sprintf("Field %q in FieldsFromMap can not be set with value %q", name, value),
			})
			continue
		}
		if f.IsEmpty() {
			continue
		}

		fields[name] = f
	}

	return fields, nil
}

// CanonicalFields returns all canonical field paths in alphabetical order.
func (fc FieldCatalog) CanonicalFields() []string {
	seen := make(map[string]struct{}, len(fc.InputKeys))

	for _, canonical := range fc.InputKeys {
		if canonical == "" {
			continue
		}

		seen[canonical] = struct{}{}
	}

	fields := make([]string, 0, len(seen))
	for canonical := range seen {
		fields = append(fields, canonical)
	}

	slices.Sort(fields)

	return fields
}

// ManagedFieldNames returns canonical names of fields explicitly marked as managed.
func (fc FieldCatalog) ManagedFieldNames(cfg Config) map[string]struct{} {
	names := make(map[string]struct{})

	for _, root := range fc.Fields {
		root.Walk(func(def FieldDef) {
			if def.Management == ManagementManaged ||
				(cfg.DefaultManageMode == DefaulManageModeManaged && def.Management != ManagementIgnored) {
				names[def.Name] = struct{}{}
			}
		})
	}

	return names
}
