package runtime

import (
	"context"

	"github.com/SSHcom/inventory_import_privx_dev/internal/config"
	"github.com/SSHcom/inventory_import_privx_dev/internal/output"
	"github.com/SSHcom/inventory_import_privx_dev/internal/privx"
	"github.com/SSHcom/inventory_import_privx_dev/internal/transform"
	"github.com/SSHcom/inventory_import_privx_dev/internal/translators"
)

// Runtime handles all common parameters for commands.
type Runtime struct {
	Config       *config.Config
	Output       output.Output
	PrivX        *privx.Client
	AccessGroups translators.Dictionary
	Roles        translators.Dictionary
	Translators  transform.TranslatorMap
	// ManagedFields is a map of field canonical names,
	// stating what fields are not ignored when comparing to existing state.
	ManagedFields map[string]struct{}
}

// PrivXClient returns the PrivX client, creating it if necessary.
func (rt *Runtime) PrivXClient(ctx context.Context) (*privx.Client, error) {
	if rt.PrivX != nil {
		return rt.PrivX, nil
	}

	client, err := privx.New(ctx, *rt.Config, rt.Output)
	if err != nil {
		return nil, err
	}

	rt.PrivX = client
	return client, nil
}
