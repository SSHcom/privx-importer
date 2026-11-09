package transform

import (
	"errors"
	"fmt"
	"strings"

	"github.com/SSHcom/inventory_import_privx_dev/internal/config"
	"github.com/SSHcom/inventory_import_privx_dev/internal/domain"
	"github.com/SSHcom/inventory_import_privx_dev/internal/domain/host"
	"github.com/SSHcom/inventory_import_privx_dev/internal/field"
	"github.com/SSHcom/inventory_import_privx_dev/internal/output"
	"github.com/SSHcom/inventory_import_privx_dev/internal/reader"
)

const (
	normalizeDuplicateFieldCode = "normalize_duplicate_field"
)

var (
	ErrHostKeyMissing = errors.New("host key missing")
)

// normalizeFieldNameReplacer will replace special separator characters with underscore.
// Period is an exception, it is used as subfield separator.
var normalizeFieldNameReplacer = strings.NewReplacer(
	" ", "_",
	"-", "_",
	"/", "_",
	"\\", "_",
)

// NormalizedRecord is a row of normalized data.
//
// Data keys are in normalized form.
// Values have no trailing or leading whitespace.
type NormalizedRecord struct {
	Line   int
	Fields field.Fields
}

// HostKey returns the host key derived from normalized fields.
func (r NormalizedRecord) HostKey() (host.HostKey, error) {
	key := host.HostKey{
		CommonName: r.Fields.Text("common_name"),
		ID:         r.Fields.Text("id"),
		IPAddress:  r.Fields.Text("addresses"),
	}

	if key.IsEmpty() {
		return key, ErrHostKeyMissing
	}

	return key, nil
}

// NormalizedResult is the normalized data from a data source.
type NormalizedResult struct {
	Source reader.SourceID
	Kind   reader.SourceKind
	Lines  int

	Records []NormalizedRecord
}
type ManyNormalizedResults []NormalizedResult

// NormalizeReadResult applies generic normalization to all sources in a ReadResult.
//
// - Trailing and leading blanks will be trimmed
// - field titles will be downcase
// - no empty titles
func NormalizeReadResult(
	out output.Output,
	catalog config.FieldCatalog,
	translators TranslatorMap,
	input reader.ReadResult,
	managedFields map[string]struct{},
) ManyNormalizedResults {
	output := make([]NormalizedResult, 0, len(input.Sources))

	for _, src := range input.Sources {
		output = append(output, NormalizeSource(out, catalog, translators, src, managedFields))
	}

	return output
}

// NormalizeSource applies generic normalization to a SourceResult.
//
// The function performs only source-agnostic cleanup, such as trimming
// whitespace and normalizing field names. It does not apply entity-specific
// rules. Non-fatal issues are appended to SourceResult.Diagnostics.
func NormalizeSource(
	out output.Output,
	catalog config.FieldCatalog,
	translators TranslatorMap,
	input reader.SourceResult,
	managedFields map[string]struct{},
) NormalizedResult {
	output := NormalizedResult{
		Source:  input.Source,
		Kind:    input.Kind,
		Lines:   input.Lines,
		Records: make([]NormalizedRecord, 0, len(input.Records)),
	}

	for _, record := range input.Records {
		output.Records = append(output.Records, normalizeRecord(out, catalog, translators, input.Source, record, managedFields))
	}

	return output
}

// normalizeRecord normalizes one raw record.
// Keys will be only the ones found in catalog.
func normalizeRecord(
	out output.Output,
	catalog config.FieldCatalog,
	translators TranslatorMap,
	source reader.SourceID,
	record reader.RawRecord,
	managedFields map[string]struct{},
) NormalizedRecord {
	normalized := NormalizedRecord{
		Line:   record.Line,
		Fields: make(field.Fields, len(record.Fields)),
	}

	for rawKey, rawValue := range record.Fields {

		// Is the field defined in catalog?
		key := normalizeFieldName(rawKey)
		canonicalKey, ok := catalog.ResolveInputKey(key)
		if !ok {
			out.Diagnostic(domain.Diagnostic{
				Source:   string(source),
				Line:     record.Line,
				Field:    key,
				Severity: domain.SeverityWarning,
				Message: fmt.Sprintf(
					"Column name %q is not found in field catalog",
					key),
			},
			)
			continue
		}

		// Is it among the managed fields?
		_, ok = managedFields[canonicalKey]
		if !ok {
			out.Diagnostic(domain.Diagnostic{
				Source:   string(source),
				Line:     record.Line,
				Field:    canonicalKey,
				Severity: domain.SeverityWarning,
				Message: fmt.Sprintf(
					"field %q not managed; ignoring",
					canonicalKey,
				),
			})
			continue
		}

		// Is it a duplicate?
		if _, exists := normalized.Fields[canonicalKey]; exists {
			out.Diagnostic(domain.Diagnostic{
				Source:   string(source),
				Line:     record.Line,
				Field:    canonicalKey,
				Severity: domain.SeverityWarning,
				Code:     normalizeDuplicateFieldCode,
				Message: fmt.Sprintf(
					"multiple field names normalize to %q; ignoring duplicate %q",
					canonicalKey,
					rawKey,
				),
			})
			continue
		}

		// Can we create a field for it?
		fieldDef, ok := catalog.FieldDef(canonicalKey)
		if !ok {
			out.Diagnostic(domain.Diagnostic{
				Source:   string(source),
				Field:    canonicalKey,
				Severity: domain.SeverityError,
				Code:     validateFieldCreateFailCode,
				Message:  fmt.Sprintf("No field catalog entry for %q", canonicalKey),
			})
			continue
		}
		field, err := field.NewField(canonicalKey, field.FieldKind(fieldDef.Kind))
		if err != nil {
			out.Diagnostic(domain.Diagnostic{
				Source:   string(source),
				Line:     record.Line,
				Field:    canonicalKey,
				Severity: domain.SeverityError,
				Code:     validateFieldCreateFailCode,
				Message:  err.Error(),
			})
			continue
		}

		value := normalizeFieldValue(rawValue)

		// Use default value if field has an empty value of "".
		if value == "" && fieldDef.Default != "" {
			out.Printf(output.VerbosityTrace, "Use default %q for field %q\n", fieldDef.Default, canonicalKey)
			value = fieldDef.Default
		}

		// Can the value be parsed into this field?
		field, err = field.Parse(value)
		if err != nil {
			out.Diagnostic(domain.Diagnostic{
				Source:   string(source),
				Line:     record.Line,
				Field:    canonicalKey,
				Severity: domain.SeverityError,
				Code:     validateFieldCreateFailCode,
				Message:  err.Error(),
			})
			continue
		}
		normalized.Fields[canonicalKey] = field
	}

	applyMissingDefaults(out, catalog, source, &normalized)

	// In case of error, the original is returned anyway
	fields, err := applyTranslations(out, catalog, translators, normalized.Fields)
	if err != nil {
		out.Diagnostic(domain.Diagnostic{
			Source:   string(source),
			Line:     record.Line,
			Severity: domain.SeverityWarning,
			Message:  err.Error(),
		})
	}

	// Run folding into complex fields
	err = catalog.RestructureFields(out, &fields)
	if err != nil {
		out.Diagnostic(domain.Diagnostic{
			Severity: domain.SeverityError,
			Message:  fmt.Errorf("%w: Folding of read values into hierarchy failed", err).Error(),
		})
	}

	normalized.Fields = fields

	return normalized
}

// normalizeFieldName converts a raw field name to a canonical generic form.
func normalizeFieldName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ToLower(name)
	name = normalizeFieldNameReplacer.Replace(name)

	for strings.Contains(name, "__") {
		name = strings.ReplaceAll(name, "__", "_")
	}

	name = strings.Trim(name, "_")
	return name
}

// normalizeFieldValue applies generic cleanup to a raw field value.
func normalizeFieldValue(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")
	return value
}

// applyMissingDefaults adds catalog default values for fields missing from the record.
func applyMissingDefaults(
	out output.Output,
	catalog config.FieldCatalog,
	source reader.SourceID,
	record *NormalizedRecord,
) {
	for canonicalKey, fieldDef := range catalog.Fields {
		if fieldDef.Default == "" {
			continue
		}

		if _, exists := record.Fields[canonicalKey]; exists {
			continue
		}

		f, err := field.NewField(canonicalKey, field.FieldKind(fieldDef.Kind))
		if err != nil {
			out.Diagnostic(domain.Diagnostic{
				Source:   string(source),
				Line:     record.Line,
				Field:    canonicalKey,
				Severity: domain.SeverityError,
				Code:     validateFieldCreateFailCode,
				Message:  err.Error(),
			})
			continue
		}

		f, err = f.Parse(fieldDef.Default)
		if err != nil {
			out.Diagnostic(domain.Diagnostic{
				Source:   string(source),
				Line:     record.Line,
				Field:    canonicalKey,
				Severity: domain.SeverityError,
				Code:     validateFieldCreateFailCode,
				Message:  err.Error(),
			})
			continue
		}

		out.Printf(output.VerbosityTrace, "Use default %q for missing field %q\n", fieldDef.Default, canonicalKey)

		record.Fields[canonicalKey] = f
	}
}

var (
	// ErrUnknownTranslator indicates that the field catalog refers to an unknown translator.
	ErrUnknownTranslator = errors.New("unknown field translator")

	// ErrTranslationFailed indicates that a field value could not be translated.
	ErrTranslationFailed = errors.New("field translation failed")
)

// applyTranslations applies configured translations to field values.
func applyTranslations(
	out output.Output,
	catalog config.FieldCatalog,
	translators TranslatorMap,
	fields field.Fields,
) (field.Fields, error) {

	return fields.Transform(
		func(f field.Field) (field.Field, error) {
			// FieldDef covers nested fields too; catalog.Fields has only top-level ones.
			def, ok := catalog.FieldDef(f.Name())
			if !ok || def.Translator == "" {
				return f, nil
			}

			out.Printf(output.VerbosityTrace, " translator %s used on %s/n", def.Translator, f.Name())
			translator, ok := translators[def.Translator]
			if !ok {
				available := translators.Keys()
				return f, fmt.Errorf(
					"%w: field %q uses translator %q but available ones are %s",
					ErrUnknownTranslator,
					f.Name(),
					def.Translator,
					strings.Join(available, ","),
				)
			}

			translated, err := translator.ToMachine(out, f)
			if err != nil {
				return nil, fmt.Errorf(
					"field %q uses translator %q: %w",
					f.Name(),
					def.Translator,
					err,
				)
			}

			return translated, nil
		},
	)
}
