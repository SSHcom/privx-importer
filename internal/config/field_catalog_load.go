package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/SSHcom/inventory_import_privx_dev/internal/field"
	"github.com/SSHcom/inventory_import_privx_dev/internal/output"
	"github.com/SSHcom/inventory_import_privx_dev/internal/util"
	toml "github.com/pelletier/go-toml/v2"
)

var (
	// ErrReadFieldCatalog is returned when the catalog file cannot be read.
	ErrReadFieldCatalog = fmt.Errorf("read field catalog failed")

	// ErrDecodeFieldCatalog is returned when TOML decoding fails.
	ErrDecodeFieldCatalog = fmt.Errorf("decode field catalog failed")

	// ErrDuplicateFieldDef is returned when the catalog contains the same field name more than once.
	ErrDuplicateFieldDef = errors.New("duplicate field definition")

	ErrUnsupportedConfigFormat = errors.New("unsupported config format")
)

type fieldCatalogFile struct {
	Fields []FieldDef `toml:"field"`
}

// LoadFieldCatalog reads field definitions from a TOML file.
func LoadFieldCatalog(out output.Output, path string) (FieldCatalog, error) {
	catalog := NewFieldCatalog()
	data, err := os.ReadFile(path)
	if err != nil {
		return catalog, fmt.Errorf("%w: %v", ErrReadFieldCatalog, err)
	}

	var file fieldCatalogFile

	fileExt := strings.ToLower(filepath.Ext(path))
	switch fileExt {
	case ".json":
		// Currently unused -- JSON might be easier to proof than TOML.
		if err := json.Unmarshal(data, &file); err != nil {
			return catalog, fmt.Errorf("%w: %v", ErrDecodeFieldCatalog, err)
		}

	case ".toml":
		if err := toml.Unmarshal(data, &file); err != nil {
			return catalog, fmt.Errorf("%w: %v", ErrDecodeFieldCatalog, err)
		}

	default:
		return FieldCatalog{}, fmt.Errorf("%w: %v", ErrUnsupportedConfigFormat, fileExt)
	}

	normalizeFieldDefPaths(file.Fields)

	for _, f := range file.Fields {

		if !f.Validate(out) {
			out.Printf(output.VerbosityInfo, "Field definition %s skipped due to errors\n", f.Name)
			continue
		}

		if _, exists := catalog.Fields[f.Name]; exists {
			out.Printf(output.VerbosityInfo, "Field definition %s is duplicate\n", f.Name)
			continue
		}

		catalog.Fields[f.Name] = f
	}

	catalog.buildIndex()
	catalog.structurals = catalog.structuralDefs()

	return catalog, nil
}

// normalizeFieldDefPaths makes all field definition names canonical paths.
func normalizeFieldDefPaths(fields []FieldDef) {
	for i := range fields {
		normalizeFieldDefPath(&fields[i], "")
	}
}

// normalizeFieldDefPath makes one field definition and its children use canonical paths.
func normalizeFieldDefPath(def *FieldDef, parentPath string) {
	if len(def.InputKeys) == 0 {
		// Inputkeys is optional in the TOML file with this.
		def.InputKeys = append(def.InputKeys, def.Name)
	}
	def.Name = util.JoinPath(parentPath, def.Name)

	for i := range def.Children {
		normalizeFieldDefPath(&def.Children[i], def.Name)
	}
}

// buildIndex prepares lookup map for input keys and canonical field paths.
func (fc *FieldCatalog) buildIndex() {
	fc.InputKeys = make(map[string]string)
	fc.flatFields = make(map[string]FieldDef)

	// This loop goes through all top level fields and the called function dives into subfields.
	for _, def := range fc.Fields {
		fc.indexField(def, []string{})
	}
}

// indexField recursively indexes input keys and canonical field paths.
func (fc *FieldCatalog) indexField(
	def FieldDef,
	// paths is a list of all possible path combinations above this, for example ["principal.id", "principals.id"]
	paths []string,
) {
	fieldPath := def.Name

	fc.flatFields[fieldPath] = def

	if len(def.Children) == 0 {
		// This is a leaf that has no sublevels.
		for _, key := range def.InputKeys {
			if len(paths) == 0 {
				// Field is on main level.
				fc.InputKeys[key] = def.Name
			} else {
				// Field is on a sub level.
				for _, path := range paths {
					fc.InputKeys[path+"."+key] = fieldPath
				}
			}
		}

		return
	}

	// We are dealing with a complex field with subfields

	paths = formSubPaths(paths, def.InputKeys)

	for _, child := range def.Children {
		fc.indexField(child, paths)
	}
}

// formSubPaths combines the earlier path possibilities with the different possibilities for this leaf
func formSubPaths(paths []string, leafs []string) []string {
	if len(paths) == 0 {
		// This is first level of subfield, so possible paths are simply the alternatives for the master field.
		return leafs
	}

	// There are combinations of alternatives
	newPaths := make([]string, 0, len(paths)*len(leafs))
	for _, path := range paths {
		for _, leaf := range leafs {
			newPaths = append(newPaths, path+"."+leaf)
		}
	}
	return newPaths
}

// structuralDefs returns those field definitions that have child fields,
// ordered deepest first.
func (fc FieldCatalog) structuralDefs() []FieldDef {
	var defs []FieldDef

	for _, def := range fc.Fields {
		def.Walk(func(def FieldDef) {
			if len(def.Children) > 0 {
				defs = append(defs, def)
			}
		})
	}

	// Order according to depth in the structure.
	sort.SliceStable(defs, func(i, j int) bool {
		return defs[i].Depth() > defs[j].Depth()
	})

	return defs
}

// Validate returns true if  the field definition is consistent.
func (f FieldDef) Validate(out output.Output) bool {
	result := true

	if f.Translator != "" {
		_, ok := TranslatorIDSet[f.Translator]
		if !ok {
			out.Printf(output.VerbosityNormal, "Field definition %s has nonexistent translator %s\n", f.Name, f.Translator)
			result = false
		}
	}

	if !field.IsKnownFieldKind(f.Kind) {
		out.Printf(output.VerbosityNormal, "Field definition %s has unknown kind %s\n", f.Name, f.Kind)
		result = false
	}

	if err := f.Management.Validate(); err != nil {
		out.Printf(output.VerbosityNormal, "Field definition %s has invalid Management %s\n", f.Name, err.Error())
		result = false
	}

	for _, child := range f.Children {
		if !child.Validate(out) {
			result = false
		}
	}
	return result
}
