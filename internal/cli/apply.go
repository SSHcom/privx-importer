package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/SSHcom/inventory_import_privx_dev/internal/config"
	"github.com/SSHcom/inventory_import_privx_dev/internal/output"
	runt "github.com/SSHcom/inventory_import_privx_dev/internal/runtime"
	"github.com/SSHcom/inventory_import_privx_dev/internal/workflow"
	"github.com/spf13/cobra"
)

func newApplyCmd(rt *runt.Runtime) *cobra.Command {
	opts := ApplyOptions{}

	cmd := &cobra.Command{
		Use:   "apply <input-file> [input-file...]",
		Short: "Apply planned PrivX changes",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Inputs = append([]string(nil), args...)
			return runApply(cmd.Context(), rt, opts)
		},
	}

	cmd.Flags().BoolVar(
		&opts.AutoApprove,
		"auto-approve",
		false,
		"Apply changes without interactive confirmation.",
	)

	cmd.PersistentFlags().StringVar(
		&opts.DeleteMode,
		"delete-mode",
		"",
		"Delete mode: "+strings.Join(config.DeleteModeDelete.AllowedValues(), ", "),
	)

	_ = cmd.MarkFlagRequired("input")

	return cmd
}

func runApply(ctx context.Context, rt *runt.Runtime, opts ApplyOptions) error {
	rt.Output.Printf(output.VerbosityInfo, "inputs: %v\n", opts.Inputs)
	rt.Output.Printf(output.VerbosityInfo, "auto-approve: %v\n", opts.AutoApprove)
	rt.Output.Printf(output.VerbosityInfo, "delete-mode: %v\n", opts.DeleteMode)

	_, err := rt.PrivXClient(ctx)
	if err != nil {
		return err
	}

	err = workflow.Prepare(ctx, rt)
	if err != nil {
		return err
	}

	localPlan, err := workflow.PrepareLocalPlans(ctx, opts.Inputs, rt)
	if err != nil {
		return err
	}
	PrintLocalPlan(rt.Output, localPlan)

	prepared, err := workflow.PrepareHostPlans(ctx, localPlan.TargetSpecs, rt)
	if err != nil {
		return err
	}

	if len(prepared.HostPlans) == 0 {
		rt.Output.Printf(output.VerbosityNormal, "No changes\n")
		return nil
	}

	PrintPreparedPlan(rt.Output, rt.Config.Scope, prepared)

	if !opts.AutoApprove {
		ok, err := askForConfirmation()
		if err != nil {
			return err
		}
		if !ok {
			rt.Output.Printf(output.VerbosityNormal, "Apply cancelled\n")
			return nil
		}
	}

	workflow.ApplyPlans(ctx, *rt.Config, rt.Output, rt.PrivX, prepared)

	return nil
}

func askForConfirmation() (bool, error) {
	fmt.Print("Do you want to apply these changes? [y/N]: ")

	reader := bufio.NewReader(os.Stdin)

	line, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return false, err
	}

	answer := strings.TrimSpace(strings.ToLower(line))

	switch answer {
	case "y", "yes":
		return true, nil
	default:
		return false, nil
	}
}
