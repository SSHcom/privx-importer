package cli

import (
	"context"
	"os"
	"slices"

	clioutput "github.com/SSHcom/inventory_import_privx_dev/internal/cli/output"
	"github.com/SSHcom/inventory_import_privx_dev/internal/config"
	"github.com/SSHcom/inventory_import_privx_dev/internal/output"
	"github.com/SSHcom/inventory_import_privx_dev/internal/runtime"
	"github.com/SSHcom/inventory_import_privx_dev/internal/util"
	"github.com/spf13/cobra"
)

func newShowCmd(rt *runtime.Runtime) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "show",
		Aliases: []string{"sh"},
		Short:   "Show data from PrivX",
	}

	cmd.AddCommand(newShowAccessGroupsCmd(rt))
	cmd.AddCommand(newShowEnvCmd(rt))
	cmd.AddCommand(newShowConfigCmd(rt))
	cmd.AddCommand(newShowFieldsCmd(rt))
	cmd.AddCommand(newShowRolesCmd(rt))

	return cmd
}

func newShowAccessGroupsCmd(rt *runtime.Runtime) *cobra.Command {
	return &cobra.Command{
		Use:     "accessgroups",
		Aliases: []string{"ag"},
		Short:   "Show access groups",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runShowAccessGroups(cmd.Context(), rt)
		},
	}
}

func runShowAccessGroups(ctx context.Context, rt *runtime.Runtime) error {

	client, err := rt.PrivXClient(ctx)
	if err != nil {
		return err
	}

	groups, err := client.AccessGroups(ctx)
	if err != nil {
		return err
	}

	for name, value := range groups.ByName {
		rt.Output.Printf(output.VerbosityNormal, "  %-15s: %s\n", name, value.ID)
	}
	return nil
}

func newShowEnvCmd(rt *runtime.Runtime) *cobra.Command {
	return &cobra.Command{
		Use:     "environment",
		Aliases: []string{"env"},
		Short:   "Show allowed environment variables and their values",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runShowEnv(rt)
		},
	}
}

func runShowEnv(rt *runtime.Runtime) error {
	rt.Output.Printf(output.VerbosityNormal, "Possible environment variables and their values. They can also be set in the config.toml file.\n")

	for _, key := range config.AllConfigKeys {
		value := os.Getenv(key.EnvName())
		if _, ok := config.MaskedMap[key.EnvName()]; ok {
			value = util.MaskString(value)
		}
		rt.Output.Printf(
			output.VerbosityNormal,
			"  %s: %q\n",
			key.EnvName(),
			value,
		)
	}
	return nil
}

func newShowConfigCmd(rt *runtime.Runtime) *cobra.Command {
	return &cobra.Command{
		Use:     "config",
		Aliases: []string{"cfg"},
		Short:   "Show active runtime configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runShowConfig(rt)
		},
	}
}

func runShowConfig(rt *runtime.Runtime) error {

	rt.Output.Header("General config values")
	rt.Output.Printf(output.VerbosityNormal, "%-20s: %d\n", "Max CSV records", rt.Config.MaxRecords)
	rt.Output.Printf(output.VerbosityNormal, "%-20s: %s\n", "Default to manage field", rt.Config.DefaultManageMode)
	rt.Output.Printf(output.VerbosityNormal, "%-20s: %s\n", "Delete mode", rt.Config.DeleteMode)
	rt.Output.DumpFields(output.VerbosityNormal, "Scope limiters", rt.Config.Scope)
	rt.Output.DumpStruct(
		output.VerbosityNormal,
		"PrivX API configuration",
		rt.Config.PrivX,
		config.MaskedMap,
	)
	clioutput.PrintFieldCatalog(rt.Output, output.VerbosityNormal, rt.Config.Catalog)
	clioutput.PrintFieldCatalogIndex(rt.Output, output.VerbosityTrace, rt.Config.Catalog)

	return nil
}

func newShowFieldsCmd(rt *runtime.Runtime) *cobra.Command {
	return &cobra.Command{
		Use:     "fields",
		Aliases: []string{"field"},
		Short:   "Show all available CSV fields",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runShowFields(rt)
		},
	}
}

func runShowFields(rt *runtime.Runtime) error {
	rt.Output.Printf(
		output.VerbosityNormal,
		"Paths for implemented fields, to be used as CSV column names:\n",
	)

	for _, field := range rt.Config.Catalog.CanonicalFields() {
		rt.Output.Printf(
			output.VerbosityNormal,
			"  %s\n",
			field,
		)
	}

	return nil
}

func newShowRolesCmd(rt *runtime.Runtime) *cobra.Command {
	return &cobra.Command{
		Use:     "roles",
		Aliases: []string{"role"},
		Short:   "Show roles",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runShowRoles(cmd.Context(), rt)
		},
	}
}

func runShowRoles(ctx context.Context, rt *runtime.Runtime) error {
	client, err := rt.PrivXClient(ctx)
	if err != nil {
		return err
	}

	roles, err := client.Roles(ctx)
	if err != nil {
		return err
	}

	names := make([]string, 0, len(roles.ByName))
	for name := range roles.ByName {
		names = append(names, name)
	}

	slices.Sort(names)

	for _, name := range names {
		value := roles.ByName[name]

		rt.Output.Printf(
			output.VerbosityNormal,
			"  %-20s: %s\n",
			name,
			value.ID,
		)
	}

	return nil
}
