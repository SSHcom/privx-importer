package plan

import (
	"errors"
	"fmt"

	"github.com/SSHcom/inventory_import_privx_dev/internal/config"
	"github.com/SSHcom/inventory_import_privx_dev/internal/domain"
	"github.com/SSHcom/inventory_import_privx_dev/internal/domain/host"
	"github.com/SSHcom/inventory_import_privx_dev/internal/field"
	"github.com/SSHcom/inventory_import_privx_dev/internal/output"
	"github.com/SSHcom/inventory_import_privx_dev/internal/util"
)

var (
	// ErrPlanFieldNotSupportedByAPIHost is returned when the plan contains
	// a generic field that does not exist in the API Host definition.
	ErrPlanFieldNotSupportedByAPIHost = errors.New(
		"plan contains a field that is not supported by API Host",
	)
)

// Action tells the planned host-level operation.
type Action string

const (
	ActionCreate Action = "create"
	ActionUpdate Action = "update"
	ActionDelete Action = "delete"
)

func (a Action) Label() string {
	switch a {
	case ActionCreate:
		return "add"
	case ActionUpdate:
		return "modify"
	case ActionDelete:
		return "delete"
	default:
		return "unknown"
	}
}

// HostPlan is the planned operation for one host.
type HostPlan struct {
	Action Action
	Keys   host.HostKey
	Key    string
	Fields field.Fields
	// Changed is a list of keys of changed fields
	Changed []string
}

// BuildHostPlan creates detailed plan for existing or new hosts.
// Nil return means no changes are required.
func BuildHostPlan(
	cfg config.Config,
	out output.Output,
	current *TargetSpec,
	spec TargetSpec,
) *HostPlan {
	_ = cfg

	if current == nil {
		// No corresponding existing host --> create.
		return &HostPlan{
			Action: ActionCreate,
			Keys:   spec.Keys,
			Key:    spec.Key,
			Fields: spec.Fields,
			// All fields are new == changed
			Changed: util.Keys(spec.Fields),
		}
	}

	managedFields := cfg.Catalog.ManagedFieldNames(cfg)
	// Overlay will not touch fields that are not present in spec.
	nextFields, changed, _ := current.Fields.Overlay(
		spec.Fields,
		func(f field.Field) bool {
			_, ok := managedFields[f.Name()]
			return ok
		},
	)
	if len(changed) == 0 {
		out.Printf(output.VerbosityInfo, "Host %s is unchanged\n", current.Key)
		return nil
	}

	return &HostPlan{
		Action:  ActionUpdate,
		Keys:    spec.Keys,
		Key:     spec.Key,
		Fields:  nextFields,
		Changed: changed,
	}
}

// BuildDeleteHostPlan creates action plan for hosts that should be removed.
func BuildDeleteHostPlan(current *TargetSpec) HostPlan {
	p := HostPlan{
		Action: ActionDelete,
		Keys:   current.Keys,
		Key:    current.Key,
	}

	return p
}

// BuildHostPlans goes through all hosts, both in input and in PrivX, and makes a plan for each.
// Only hosts within scope are handled.
func BuildHostPlans(
	cfg config.Config,
	out output.Output,
	// currentHosts are the existing PrivX hosts.
	currentHosts map[string]TargetSpec,
	// targetSpecs are the host descriptions gathered from inputs.
	targetSpecs map[string]TargetSpec,
) map[string]HostPlan {
	plans := make(map[string]HostPlan, len(currentHosts)+len(targetSpecs))
	matchedCurrent := make(map[string]struct{}, len(currentHosts))

	// Go through all hosts we have collected from input files.
	for targetKey, spec := range targetSpecs {
		currentKey, current, found := findCurrentHostByKey(currentHosts, spec.Keys)
		if found {

			// Let's remember what hosts we have in input
			matchedCurrent[currentKey] = struct{}{}

			if !current.Fields.EqualsSubset(cfg.Scope) {
				// We are not allowed to touch this host.
				out.Diagnostic(domain.Diagnostic{
					Source:   "plan",
					Field:    "scope",
					Severity: domain.SeverityWarning,
					Code:     "target_host_out_of_scope",
					Message:  fmt.Sprintf("target host out of scope: %s", current.Keys.Human()),
				})
				continue
			}

			plan := BuildHostPlan(cfg, out, &current, spec)
			if plan == nil {
				// There was nothing to change.
				continue
			}

			// Let's check if policy allows modification
			if cfg.UpdateMode != config.UpdateModeModify && cfg.UpdateMode != config.UpdateModeSync {
				out.Diagnostic(domain.Diagnostic{
					Severity: domain.SeverityInfo,
					Code:     "Update mode does not allow modification",
					Message:  fmt.Sprintf("Update mode does not allow modification hosts: %s", current.Keys.Human()),
				})
				continue
			}
			plans[targetKey] = *plan
			continue
		}

		// Host was not found in current, so it is a new host.
		plans[targetKey] = *BuildHostPlan(cfg, out, nil, spec)
	}

	// Check the existing hosts for ones that were not mentioned in input.
	var (
		total      = 0
		outOfScope = 0
	)
	for currentKey, current := range currentHosts {
		total = total + 1
		if _, ok := matchedCurrent[currentKey]; ok {
			out.Printf(output.VerbosityTrace, "Host %s was processed in the input list\n", current.Keys.Human())
			continue
		}
		if !current.Fields.EqualsSubset(cfg.Scope) {
			// Host is not in input, and not in scope.
			// It is simply not in scope, no need for any action or diagnostic.
			out.Printf(output.VerbosityTrace, "Host %s is out of scope\n", current.Keys.Human())
			outOfScope = outOfScope + 1
			continue
		}
		// If the host is disabled, and our delete config is to disable, then no reason to change.
		if current.IsDisabled() {
			out.Printf(output.VerbosityInfo, "Host %s is already disabled\n", current.Keys.Human())
			continue
		}
		// Let's check if policy allows deletion
		if cfg.UpdateMode != config.UpdateModeSync {
			out.Diagnostic(domain.Diagnostic{
				Severity: domain.SeverityInfo,
				Code:     "Update mode does not allow deletion",
				Message:  fmt.Sprintf("Update mode does not allow deletion of hosts: %s", current.Keys.Human()),
			})
			continue
		}
		out.Printf(output.VerbosityTrace, "Host %s should be deleted\n", current.Keys.Human())
		plans[currentKey] = BuildDeleteHostPlan(&current)
	}
	out.Printf(output.VerbosityNormal, "Hosts in PrivX:\n  %5d total\n  %5d in scope\n\n", total, total-outOfScope)
	return plans
}

// findCurrentHostByKey returns the current PrivX host matching the given target key.
func findCurrentHostByKey(
	currentHosts map[string]TargetSpec,
	keys host.HostKey,
) (string, TargetSpec, bool) {
	for currentKey, current := range currentHosts {
		if current.Keys.Match(keys) {
			return currentKey, current, true
		}
	}

	return "", TargetSpec{}, false
}
