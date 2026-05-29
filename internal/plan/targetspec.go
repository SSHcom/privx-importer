package plan

import (
	"github.com/SSHcom/inventory_import_privx_dev/internal/domain"
	"github.com/SSHcom/inventory_import_privx_dev/internal/domain/host"
	"github.com/SSHcom/inventory_import_privx_dev/internal/field"
	"github.com/SSHcom/inventory_import_privx_dev/internal/output"
	"github.com/SSHcom/inventory_import_privx_dev/internal/transform"
)

// TargetSpec is the internal target specification before comparing
// it against the current state in PrivX.
type TargetSpec struct {
	Keys host.HostKey
	Key  string

	Fields field.Fields
}

// IsDisabled returns true if the host is marked as disabled.
func (t TargetSpec) IsDisabled() bool {
	disabled, ok := t.Fields["disabled"]
	if !ok {
		return false
	}
	return disabled.Value() != "FALSE"
}

// BuildTargetSpec converts validated records into TargetSpec host models.
//
// Multiple validated records can contribute to the same host. Records are
// matched by host key identifiers, and field values are merged into the
// resulting target specification.
func BuildTargetSpec(out output.Output, validated transform.ValidatedResult) map[string]TargetSpec {
	hosts := make(map[string]*TargetSpec)

	for _, record := range validated {
		spec, mapKey, exists := findTargetSpecByHostKey(hosts, record.Keys)

		if !exists {
			// This is the first time we encounter this key, so create target record and use source keys.
			spec = &TargetSpec{
				Keys:   record.Keys,
				Key:    record.Key,
				Fields: make(map[string]field.Field),
			}
			hosts[record.Key] = spec
		} else {
			// This host already has a target record, so create new key with best of both of them.
			mergedKey, err := spec.Keys.Merge(record.Keys)
			if !mergedKey.Equal(spec.Keys) {
				out.Printf(output.VerbosityDebug, "Key for %s upgraded to %s", spec.Key, mergedKey)
			}
			if err != nil {
				// The new source row matches somehow with the existing one, but has conflicting values.
				out.Diagnostic(domain.Diagnostic{
					Severity: domain.SeverityError,
					Code:     "host_key_conflict",
					Message:  err.Error(),
				})
				continue
			}
			spec.Keys = mergedKey

			newMapKey := mergedKey.BestKey()
			if newMapKey != mapKey {
				// The new, improved complex key "hashes" to a different string value than before.
				delete(hosts, mapKey)
				hosts[newMapKey] = spec
			}
		}

		err := mergeValidatedRecordIntoTargetSpec(spec, record)
		if err != nil {
			out.Diagnostic(domain.Diagnostic{
				Severity: domain.SeverityWarning,
				Message:  err.Error(),
			})
		}
	}

	result := make(map[string]TargetSpec, len(hosts))
	for key, spec := range hosts {
		result[key] = *spec
	}

	return result
}

// findTargetSpecByHostKey returns the first target spec matching the given host key.
func findTargetSpecByHostKey(hosts map[string]*TargetSpec, key host.HostKey) (*TargetSpec, string, bool) {
	for mapKey, spec := range hosts {
		if spec.Keys.Match(key) {
			return spec, mapKey, true
		}
	}

	return nil, "", false
}

func mergeValidatedRecordIntoTargetSpec(spec *TargetSpec, record transform.ValidatedRecord) error {

	for _, field := range record.Fields {
		err := mergeFieldIntoTargetSpec(spec, field)
		if err != nil {
			return err
		}
	}
	return nil
}

// mergeFieldIntoTargetSpec merges one validated field into a target spec.
func mergeFieldIntoTargetSpec(spec *TargetSpec, incoming field.Field) error {
	name := incoming.Name()

	existing, ok := spec.Fields[name]
	if !ok {
		// Field did not exist, use new value directly.
		spec.Fields[name] = incoming
		return nil
	}

	// Use field specific method to merge values.
	merged, err := existing.Merge(incoming.Value())
	if err != nil {
		return err
	}

	spec.Fields[name] = merged
	return nil
}
