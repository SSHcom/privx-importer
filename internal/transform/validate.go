package transform

import (
	"errors"
	"fmt"

	"github.com/SSHcom/inventory_import_privx_dev/internal/config"
	"github.com/SSHcom/inventory_import_privx_dev/internal/domain"
	"github.com/SSHcom/inventory_import_privx_dev/internal/domain/host"
	"github.com/SSHcom/inventory_import_privx_dev/internal/field"
	"github.com/SSHcom/inventory_import_privx_dev/internal/output"
	"github.com/SSHcom/inventory_import_privx_dev/internal/validators"
)

const (
	validateUnknownFieldCode      = "validate_unknown_field"
	validateMissingIdentifierCode = "validate_missing_identifier"
	validateConflictingScalarCode = "validate_conflicting_scalar"
	validateFieldCreateFailCode   = "field creation failed"
)

var (
	ErrValidationFailed = errors.New("validation failed")
	ErrNoValidatorFound = errors.New("specified validator not found")
)

// ValidatedRecord is a row of normalized and validated data.
type ValidatedRecord struct {
	Keys host.HostKey
	Key  string

	// Fields is indexed by field internal name, which is also contained in the Field value.
	Fields field.Fields
}

// ValidatedResult is the validated data from all data sources.
type ValidatedResult map[string]ValidatedRecord

// Validate validates normalized host input records before target-state construction.
func Validate(
	out output.Output,
	input ManyNormalizedResults,
	cfg config.Config,
) ValidatedResult {

	// First merge rows with common key
	result := mergeRows(out, input)

	for _, record := range result {
		EnsureScope(out, &record, cfg)
	}

	// Run the configured validation functions
	for key, record := range result {
		if err := runValidators(record, cfg, out); err != nil {
			out.Printf(output.VerbosityNormal, "Host %s skipped due to validation errors\n", key)
			out.DumpStruct(output.VerbosityDebug, "Failed record:", record, config.MaskedMap)
			delete(result, key)
		}
	}

	return result
}

// mergeRows joins records with same keys
func mergeRows(
	out output.Output,
	input ManyNormalizedResults,
) ValidatedResult {
	result := make(ValidatedResult)
	for _, src := range input {
		for _, row := range src.Records {
			keys := host.HostKeyFromFields(row.Fields)
			key := keys.BestKey()
			if key == "" {
				// The row did not have enough information to form a key.
				out.Diagnostic(domain.Diagnostic{
					Source:   string(src.Source),
					Line:     row.Line,
					Severity: domain.SeverityError,
					Code:     validateMissingIdentifierCode,
				})
				out.DumpStruct(output.VerbosityDebug, "Fields", row.Fields, config.MaskedMap)
				continue
			}

			spec, ok := result[key]
			if !ok {
				// Record with this key not yet exists so create an empty one.
				// We let the merge-validate function fill it.
				spec = ValidatedRecord{
					Keys:   keys,
					Key:    key,
					Fields: make(field.Fields, len(spec.Fields)),
				}
				result[key] = spec
			}

			result[key] = mergeFields(out, spec, row)
		}
	}
	return result
}

// mergeFields joins the fields of the new record into the existing one.
func mergeFields(
	out output.Output,
	current ValidatedRecord,
	new NormalizedRecord,
) ValidatedRecord {
	// First validate each field separately
	for key, field := range new.Fields {

		_, ok := current.Fields[key]
		if !ok {
			// Current does not have this field at all yet.
			current.Fields.Add(field)
			continue
		} else {
			merged, err := current.Fields[key].Merge(field)
			if err != nil {
				out.Diagnostic(domain.Diagnostic{

					Field:    field.Name(),
					Severity: domain.SeverityError,
					Code:     err.Error(),
					Message: fmt.Sprintf(
						"can not merge value %q into field %q with current value of %q",
						field.Text(),
						key,
						current.Fields[key].Text(),
					),
				})
				continue
			}
			current.Fields.Add(merged)
		}
	}

	return current
}

// EnsureScope ensures that target spec satisfies scope.
// If not, it adds missing values and creates a warning diagnostic.
func EnsureScope(
	out output.Output,
	spec *ValidatedRecord,
	cfg config.Config,
) {
	if len(cfg.Scope) == 0 || spec.Fields.EqualsSubset(cfg.Scope) {
		return
	}

	// Merge scope into spec
	for name, scopeField := range cfg.Scope {
		if existing, ok := spec.Fields[name]; ok {
			merged, err := existing.Merge(scopeField)
			if err != nil {
				out.Diagnostic(domain.Diagnostic{
					Source:   "validate",
					Field:    scopeField.Name(),
					Severity: domain.SeverityError,
					Code:     "scope_merge_failed",
					Message:  err.Error(),
				})
				continue
			}
			spec.Fields[name] = merged
		} else {
			spec.Fields[name] = scopeField
		}
	}

	out.Diagnostic(
		domain.Diagnostic{
			Source:   "plan",
			Severity: domain.SeverityWarning,
			Code:     "target_scope_adjusted",
			Message:  fmt.Sprintf("target host %s data was augmented with scope limiting values for %s", spec.Keys.Human(), cfg.Scope.KeyList()),
		})
}

// runValidators runs configured validators for fields in a validated record.
func runValidators(
	record ValidatedRecord,
	cfg config.Config,
	out output.Output,
) error {

	err := record.Fields.Walk(func(f field.Field) error {

		def, ok := cfg.Catalog.FieldDef(f.Name())
		if !ok {
			out.Diagnostic(domain.Diagnostic{
				Field:    f.Name(),
				Severity: domain.SeverityError,
				Code:     "field_definition_not_found",
				Message:  fmt.Sprintf("%s: field definition not found for %q", config.ErrNoFieldDef.Error(), f.Name()),
			})
			return config.ErrNoFieldDef
		}

		// Run all validators defined for this field
		for _, validatorName := range def.Validators {
			out.Printf(output.VerbosityTrace, "running validator %s for %s\n", validatorName, f.Name())
			validator, ok := validators.Validators[validatorName]
			if !ok {
				out.Diagnostic(domain.Diagnostic{
					Field:    f.Name(),
					Severity: domain.SeverityError,
					Code:     "unknown_validator",
					Message:  fmt.Sprintf("%s: unknown validator %q for field %q", ErrNoValidatorFound.Error(), validatorName, f.Name()),
				})
				return ErrNoValidatorFound
			}

			diagnostic := validator(f)
			if diagnostic == nil {
				continue
			}

			if diagnostic.Field == "" {
				diagnostic.Field = f.Name()
			}

			out.Diagnostic(*diagnostic)
			return ErrValidationFailed
		}
		return nil
	})

	return err
}
