package translators

import (
	"fmt"

	"github.com/SSHcom/inventory_import_privx_dev/internal/field"
	"github.com/SSHcom/inventory_import_privx_dev/internal/output"
	"github.com/SSHcom/inventory_import_privx_dev/internal/transform"
	"github.com/SSHcom/inventory_import_privx_dev/internal/util"
)

// DictItem contains a machine readable and human readable forms of an identifier.
type DictItem struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Dictionary struct {
	ByName map[string]DictItem
	ByID   map[string]DictItem
}

// LookupTranslator resolves access group names into UUID values.
type LookupTranslator struct {
	Dictionary Dictionary
}

// ToMachine resolves a human readable field into machine readable.
func (r LookupTranslator) ToMachine(
	out output.Output,
	f field.Field,
) (field.Field, error) {
	value := f.Text()

	id, ok := r.Dictionary.ByName[value]
	if ok {
		return f.Set(string(id.ID))
	}

	if _, ok := r.Dictionary.ByID[value]; ok {
		// Was already an ID
		return f, nil
	}

	return f, fmt.Errorf(
		"%w: Human readable term %q did not correspond to any machine or human readable form",
		transform.ErrNoTranslation,
		value,
	)

}

// ToHuman resolves machine readable into human readable.
func (r LookupTranslator) ToHuman(
	out output.Output,
	f field.Field,
) (field.Field, error) {
	value := f.Text()

	group, ok := r.Dictionary.ByID[value]
	if ok {
		return f.Set(group.Name)
	}

	if _, ok := r.Dictionary.ByName[value]; ok {
		// Was already a group name
		return f, nil
	}

	return f, fmt.Errorf(
		"%w: Machine readable form %q did not correspond to any machine or human readable term",
		transform.ErrNoTranslation,
		value,
	)
}

// AllowedValuesTranslator can be used with different fields by initialising it with different dictionary.
type AllowedValuesTranslator struct {
	Allowed []string
}

// ToMachine normalizes a value into canonical allowed value.
func (r AllowedValuesTranslator) ToMachine(
	out output.Output,
	f field.Field,
) (field.Field, error) {
	value := f.Text()

	normalized, err := util.NormalizeToList(value, r.Allowed)
	if err != nil {
		return f, err
	}
	if normalized != value {
		out.Printf(output.VerbosityTrace, "%q normalized to %q\n", value, normalized)
	}
	return f.Set(normalized)
}

// ToHuman normalizes a value into canonical allowed value.
func (r AllowedValuesTranslator) ToHuman(
	out output.Output,
	f field.Field,
) (field.Field, error) {
	return r.ToMachine(out, f)
}
