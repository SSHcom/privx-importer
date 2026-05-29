package transform

import (
	"errors"
	"slices"

	"github.com/SSHcom/inventory_import_privx_dev/internal/config"
	"github.com/SSHcom/inventory_import_privx_dev/internal/field"
	"github.com/SSHcom/inventory_import_privx_dev/internal/output"
)

// Translator translates field values between human and machine forms.
type Translator interface {
	ToMachine(out output.Output, value field.Field) (field.Field, error)
	ToHuman(out output.Output, value field.Field) (field.Field, error)
}

type TranslatorMap map[config.TranslatorID]Translator

var (
	ErrNoTranslation = errors.New("no translation found")
)

// Keys returns a list of the available translations.
func (tm *TranslatorMap) Keys() []string {
	keys := make([]string, 0, len(*tm))
	for key := range *tm {
		keys = append(keys, string(key))
	}
	slices.Sort(keys)
	return keys
}
