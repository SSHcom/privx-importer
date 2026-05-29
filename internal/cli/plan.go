package cli

import (
	"context"

	"github.com/SSHcom/inventory_import_privx_dev/internal/config"
	"github.com/SSHcom/inventory_import_privx_dev/internal/field"
	"github.com/SSHcom/inventory_import_privx_dev/internal/output"
	"github.com/SSHcom/inventory_import_privx_dev/internal/plan"
	runt "github.com/SSHcom/inventory_import_privx_dev/internal/runtime"
	"github.com/SSHcom/inventory_import_privx_dev/internal/util"
	"github.com/SSHcom/inventory_import_privx_dev/internal/workflow"
	"github.com/spf13/cobra"
)

func newPlanCmd(rt *runt.Runtime) *cobra.Command {
	opts := PlanOptions{}

	cmd := &cobra.Command{
		Use:   "plan <input-file> [input-file...]",
		Short: "Calculate planned PrivX changes",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Inputs = append([]string(nil), args...)
			cmd.SilenceUsage = true
			return runPlan(cmd.Context(), rt, opts)
		},
	}

	cmd.Flags().StringVar(
		&opts.Config,
		"config",
		"mapping.yaml",
		"Path to configuration file (optional).",
	)

	return cmd
}

func runPlan(ctx context.Context, rt *runt.Runtime, opts PlanOptions) error {

	rt.Output.Printf(output.VerbosityInfo, "inputs: %v\n", opts.Inputs)

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
	PrintPreparedPlan(rt.Output, rt.Config.Scope, prepared)

	return nil
}

// PrintLocalPlan outputs all stages of planning.
func PrintLocalPlan(o output.Output, prepared *workflow.LocalPlan) {
	o.DumpStruct(output.VerbosityDebug, "Raw input data", prepared.ReadResult, config.MaskedMap)

	for name, record := range prepared.Validated {
		o.DumpFields(output.VerbosityDebug, "Validated input data for "+name, record.Fields)
	}

	for name, record := range prepared.TargetSpecs {
		o.DumpFields(output.VerbosityDebug, "Raw target specification data for "+name, record.Fields)
	}
}

// PrintPreparedPlan outputs all stages of planning.
func PrintPreparedPlan(o output.Output, scope field.Fields, prepared *workflow.PreparedPlan) {
	for name, record := range prepared.CurrentHosts {
		if record.Fields.EqualsSubset(scope) {
			o.DumpFields(output.VerbosityDebug, "Raw current state data for "+name, record.Fields)
		}
	}
	for name, record := range prepared.HostPlans {
		o.DumpFields(output.VerbosityDebug, "Planned data for "+name, record.Fields)
	}
	PrintHostPlans(o, prepared.HostPlans)
}

func PrintHostPlans(o output.Output, plans map[string]plan.HostPlan) {
	o.Header("Planned changes")

	count := map[plan.Action]int{
		plan.ActionCreate: 0,
		plan.ActionUpdate: 0,
		plan.ActionDelete: 0,
	}

	for _, key := range util.SortedKeys(plans) {
		count[plans[key].Action] = count[plans[key].Action] + 1
		printHostPlan(o, plans[key])
	}

	o.Header("Summary")
	for action, n := range count {
		o.Printf(output.VerbosityNormal, "hosts to %s: %d\n", action.Label(), n)
	}
}

func printHostPlan(o output.Output, p plan.HostPlan) {
	o.Printf(output.VerbosityNormal, "%s   %s\n", p.Keys, p.Action.Label())
	changed := p.Fields.WithKeys(p.Changed)
	if len(changed) > 0 {
		o.DumpFields(output.VerbosityNormal, "Changed fields:\n", changed)
	}
	unchanged := p.Fields.WithoutKeys(p.Changed)
	if len(unchanged) > 0 {
		o.DumpFields(output.VerbosityInfo, "Unchanged fields:\n", unchanged)
	}
	o.Printf(output.VerbosityNormal, "\n")
}
