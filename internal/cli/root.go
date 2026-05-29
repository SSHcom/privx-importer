package cli

import (
	"context"
	"os"
	"os/signal"
	"path/filepath"
	"strings"

	"github.com/SSHcom/inventory_import_privx_dev/internal/cli/output"
	"github.com/SSHcom/inventory_import_privx_dev/internal/config"
	baseout "github.com/SSHcom/inventory_import_privx_dev/internal/output"
	"github.com/SSHcom/inventory_import_privx_dev/internal/runtime"
	"github.com/spf13/cobra"
)

func Execute() error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	return newRootCmd().ExecuteContext(ctx)
}

func newRootCmd() *cobra.Command {
	rt := &runtime.Runtime{}

	var (
		vcount     int
		configDir  string
		updateMode string
	)
	var mainConfig config.Config

	cmd := &cobra.Command{
		Use:   "privx-import",
		Short: "Import CMDB-derived data into PrivX",
		Long:  "privx-import reads CMDB export files, plans PrivX changes, and applies them.",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {

			rt.Output = output.New(os.Stdout, os.Stderr, output.ToVerbosity(vcount))

			mainConfig = config.NewConfig()

			// 4. Default values
			mainConfig.Apply(rt.Output, config.Defaults)

			// 3. Config file
			configPath, err := config.ResolveConfigDir(configDir)
			if err != nil {
				return err
			}
			rt.Output.Printf(baseout.VerbosityInfo, "config dir: %s\n", configPath)

			catalog, err := config.LoadFieldCatalog(rt.Output, filepath.Join(configPath, "host.toml"))
			if err != nil {
				return err
			}
			mainConfig.Catalog = catalog

			fileValues, fileScope, err := config.LoadConfigFile(rt.Output, configPath, mainConfig.Catalog)
			if err != nil {
				return err
			}

			mainConfig.Apply(rt.Output, fileValues)

			// Scope is a complex entity and is handled separately
			mainConfig.Scope = fileScope

			// 2. Environment variables
			mainConfig.Apply(rt.Output, config.LoadEnv(rt.Output))

			// 1. CLI options
			mainConfig.Apply(rt.Output, loadCLIConfig(cmd))

			rt.Config = &mainConfig

			rt.Output.DumpStruct(baseout.VerbosityDebug, "\nConfiguration\n", mainConfig, config.MaskedMap)

			return nil
		},
		SilenceErrors: true,
	}

	cmd.PersistentFlags().CountVarP(&vcount, "verbose", "v", "Increase verbosity (-v, -vv, -vvv)")

	cmd.PersistentFlags().StringVar(
		&configDir,
		"config-dir",
		"internal/config",
		"Configuration directory",
	)

	cmd.PersistentFlags().StringVar(
		&updateMode,
		"update-mode",
		"",
		"Update mode: "+strings.Join(config.UpdateModes.Values(), ", "),
	)
	cmd.AddCommand(newPlanCmd(rt))
	cmd.AddCommand(newApplyCmd(rt))
	cmd.AddCommand(newShowCmd(rt))

	return cmd
}

// loadCLIConfig returns configuration values explicitly set with Cobra flags.
func loadCLIConfig(cmd *cobra.Command) config.ConfigValues {
	values := config.ConfigValues{}

	if cmd.Flags().Changed("update-mode") {
		value, err := cmd.Flags().GetString("update-mode")
		if err == nil {
			values[config.ConfigKeyUpdateMode] = value
		}
	}

	if cmd.Flags().Changed("delete-mode") {
		value, err := cmd.Flags().GetString("delete-mode")
		if err == nil {
			values[config.ConfigKeyDeleteMode] = value
		}
	}

	return values
}
