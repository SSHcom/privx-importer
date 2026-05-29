package config

import (
	"fmt"

	"github.com/SSHcom/inventory_import_privx_dev/internal/field"
	"github.com/SSHcom/inventory_import_privx_dev/internal/output"
	"github.com/SSHcom/inventory_import_privx_dev/internal/util"
)

// RestructureFields restructures flat fields according to catalog hierarchy.
func (fc FieldCatalog) RestructureFields(
	out output.Output,
	fields *field.Fields,
) error {
	out.DumpStruct(output.VerbosityTrace, "Record to fold", *fields, map[string]struct{}{})
	for _, def := range fc.structurals {
		children := collectChildren(*fields, def.Name)
		if len(children) == 0 {
			out.Printf(output.VerbosityTrace, "no children for  %s\n", def.Name)
			continue
		}

		parent, ok := (*fields)[def.Name]
		if !ok {
			// Parent field does not exist, which is probably expected, create first
			out.Printf(output.VerbosityTrace, "creating parent field %s for %d children\n", def.Name, len(children))
			var err error
			parent, err = field.NewField(def.Name, def.Kind)
			if err != nil {
				return err
			}
			fields.Add(parent)
		}

		out.Printf(output.VerbosityTrace, "folding field %s\n", def.Name)
		err := foldStructuralField(fields, parent, children)
		if err != nil {
			return err
		}
	}
	return nil
}

// collectMatchingFields searches the top level of the argument fields for any fields that are a direct descendant.
func collectChildren(fields field.Fields, parentName string) field.Fields {
	children := field.Fields{}
	for _, f := range fields {
		if isDirectChild(f, parentName) {
			children.Add(f)
		}
	}
	return children
}

// If those are found, they are removed from the top level and instead moved to be the children of the composite field.
func foldStructuralField(
	fields *field.Fields,
	parent field.Field,
	children field.Fields,
) error {
	switch composite := parent.(type) {
	case field.CompositeField:
		for name, f := range children {
			composite.AddChild(f)
			fields.Remove(name)
		}

		fields.Add(composite)

	case field.ListCompositeField:
		composite.AddItem(children)

		for name := range children {
			fields.Remove(name)
		}

		fields.Add(composite)

	default:
		return fmt.Errorf(
			"%w: trying to fold children of %q which is a %s",
			field.ErrFieldValueTypeMismatch,
			parent.Name(),
			parent.Kind(),
		)
	}

	return nil
}

func isDirectChild(child field.Field, parentName string) bool {
	return util.Trunk(child.Name()) == parentName
}
