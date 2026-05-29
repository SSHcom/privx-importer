package workflow

import (
	"context"
	"fmt"
	"slices"

	"github.com/SSHcom/inventory_import_privx_dev/internal/config"
	"github.com/SSHcom/inventory_import_privx_dev/internal/domain"
	"github.com/SSHcom/inventory_import_privx_dev/internal/field"
	"github.com/SSHcom/inventory_import_privx_dev/internal/output"
	"github.com/SSHcom/inventory_import_privx_dev/internal/plan"
	"github.com/SSHcom/inventory_import_privx_dev/internal/privx"
	"github.com/SSHcom/inventory_import_privx_dev/internal/reader"
	runt "github.com/SSHcom/inventory_import_privx_dev/internal/runtime"
	"github.com/SSHcom/inventory_import_privx_dev/internal/transform"
	"github.com/SSHcom/inventory_import_privx_dev/internal/translators"
)

var (
	ErrUnsupportedPlanAction = "Invalid plan type"
)

// PreparedPlan contains the plan as far as we can before checking PrivX state.
type LocalPlan struct {
	ReadResult  reader.ReadResult
	Validated   transform.ValidatedResult
	TargetSpecs map[string]plan.TargetSpec
}

// PreparedPlan contains the plan to update PrivX and also the intermediate stages leading to that.
type PreparedPlan struct {
	CurrentHosts map[string]plan.TargetSpec
	HostPlans    map[string]plan.HostPlan
}

// PrepareLocalPlans reads input files, normalizes and validates them and forms a desired target state.
func PrepareLocalPlans(ctx context.Context, inputPaths []string, rt *runt.Runtime) (*LocalPlan, error) {
	prepared := &LocalPlan{}

	inputs, err := reader.BuildInputs(inputPaths)
	if err != nil {
		return prepared, err
	}

	mr := reader.NewMultiReader(map[reader.Format]reader.Reader{
		reader.FormatCSV: &reader.CSVReader{},
	})

	prepared.ReadResult, err = mr.ReadAll(ctx, rt.Output, *rt.Config, inputs)
	if err != nil {
		return prepared, err
	}

	managedFields := rt.Config.Catalog.ManagedFieldNames(*rt.Config)
	normalisedResults := transform.NormalizeReadResult(rt.Output, rt.Config.Catalog, rt.Translators, prepared.ReadResult, managedFields)

	prepared.Validated = transform.Validate(rt.Output, normalisedResults, *rt.Config)

	prepared.TargetSpecs = plan.BuildTargetSpec(rt.Output, prepared.Validated)

	return prepared, nil
}

// PrepareHostPlans reads input files, fetches current state info and builds a change plan from them.
func PrepareHostPlans(ctx context.Context, targetSpecs map[string]plan.TargetSpec, rt *runt.Runtime) (*PreparedPlan, error) {
	prepared := &PreparedPlan{}

	currentHosts, err := rt.PrivX.ListHosts(
		ctx,
		rt.Config.Catalog,
		rt.Output,
	)
	if err != nil {
		return prepared, err
	}
	prepared.CurrentHosts = currentHosts

	prepared.HostPlans = plan.BuildHostPlans(*rt.Config, rt.Output, currentHosts, targetSpecs)

	return prepared, nil
}

// ApplyPlans applies planned host changes to PrivX.
//
// Host-level apply errors are collected as diagnostics and processing continues
// with the next host.
func ApplyPlans(
	ctx context.Context,
	cfg config.Config,
	out output.Output,
	client *privx.Client,
	prepared *PreparedPlan,
) {
	var (
		created = 0
		updated = 0
		deleted = 0
		failed  = 0
	)
	for _, hostPlan := range prepared.HostPlans {
		if err := applyHostPlan(ctx, cfg, out, client, hostPlan); err != nil {
			out.Diagnostic(domain.Diagnostic{
				Source:   "privx",
				Severity: domain.SeverityError,
				Code:     "apply_host_failed",
				Message:  err.Error(),
			})
			failed = failed + 1
		} else {
			switch hostPlan.Action {
			case plan.ActionCreate:
				created = created + 1
			case plan.ActionUpdate:
				updated = updated + 1
			case plan.ActionDelete:
				deleted = deleted + 1
			}
		}
	}
	out.Printf(output.VerbosityNormal, "Host updates:\n  %5d created\n  %5d updated\n  %5d deleted\n", created, updated, deleted)
	if failed > 0 {
		out.Printf(output.VerbosityNormal, "  %5d update attempts failed\n", failed)
	}
	out.Printf(output.VerbosityNormal, "\n")
}

func applyHostPlan(
	ctx context.Context,
	cfg config.Config,
	out output.Output,
	client *privx.Client,
	hostPlan plan.HostPlan,
) error {
	switch hostPlan.Action {
	case plan.ActionCreate:
		id, err := client.CreateHost(ctx, out, hostPlan.Fields)
		if err == nil {
			out.Diagnostic(domain.Diagnostic{
				Severity: domain.SeverityInfo,
				Message:  fmt.Sprintf("Host %s created with id %s\n", hostPlan.Keys.Human(), id),
			})
		}
		return err

	case plan.ActionUpdate:
		if slices.Contains(hostPlan.Changed, "disabled") {
			// disabled can not be updated with normal update, needs a separate call.
			value, ok := hostPlan.Fields["disabled"].Value().(string)
			if !ok {
				return fmt.Errorf("%w: disabled type", field.ErrFieldValueTypeMismatch)
			}
			disabled := !(value == "FALSE")
			out.Printf(output.VerbosityInfo, "setting %s disabled state to %s\n", hostPlan.Keys.Human(), value)
			err := client.SetHostDisabled(ctx, hostPlan.Fields.Text("id"), disabled)
			if err != nil {
				return err
			}
		}
		err := client.UpdateHost(ctx, out, hostPlan.Fields)
		if err == nil {
			out.Diagnostic(domain.Diagnostic{
				Severity: domain.SeverityInfo,
				Message:  fmt.Sprintf("Host %s updated\n", hostPlan.Keys.Human()),
			})
		}
		return err

	case plan.ActionDelete:
		switch cfg.DeleteMode {
		case config.DeleteModeDisable:
			err := client.SetHostDisabled(ctx, hostPlan.Keys.ID, true)
			if err == nil {
				out.Diagnostic(domain.Diagnostic{
					Severity: domain.SeverityInfo,
					Message:  fmt.Sprintf("Host %s disabled\n", hostPlan.Keys.Human()),
				})
			}
			return err
		case config.DeleteModeDelete:
			err := client.DeleteHost(ctx, hostPlan.Keys.ID)
			if err != nil {
				out.Diagnostic(domain.Diagnostic{
					Severity: domain.SeverityInfo,
					Message:  fmt.Sprintf("Should have deleted %s but %s happened\n", hostPlan.Keys.Human(), err.Error()),
				})
			}
		default:
			out.Diagnostic(domain.Diagnostic{
				Severity: domain.SeverityError,
				Message:  fmt.Sprintf("Should have deleted %s but did not understand %s\n", hostPlan.Keys.Human(), cfg.DeleteMode),
			})
		}
		return nil

	default:
		return fmt.Errorf("%s: %s", ErrUnsupportedPlanAction, hostPlan.Action)
	}
}

// Prepare prepares the work environment by fetching necessary catalogs.
func Prepare(
	ctx context.Context,
	rt *runt.Runtime,
) error {
	rt.Translators = transform.TranslatorMap{}

	// Access groups to UUID
	groups, err := rt.PrivX.AccessGroups(ctx)
	if err != nil {
		return err
	}
	rt.Translators[config.TranslatorAccessGroup] = translators.LookupTranslator{
		Dictionary: groups,
	}

	// Roles to UUID
	roles, err := rt.PrivX.Roles(ctx)
	if err != nil {
		return err
	}
	rt.Translators[config.TranslatorRole] = translators.LookupTranslator{
		Dictionary: roles,
	}

	// AllowedValuesTranslator can be used with different fields by initialising it with different dictionary.
	rt.Translators[config.TranslatorService] = translators.AllowedValuesTranslator{
		Allowed: rt.Config.AllowedValues[config.ConfigKeyAllowedServices],
	}

	rt.Output.DumpStruct(output.VerbosityTrace, "Available translators:", rt.Translators, config.MaskedMap)
	return nil
}
