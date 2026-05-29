package domain

import "fmt"

type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
	SeverityInfo    Severity = "info"
)

type Diagnostic struct {
	Source   string
	Line     int
	Field    string
	Severity Severity
	Code     string
	Message  string
}

// NewErrorDiagnostic creates a diagnostic from error.
func NewErrorDiagnostic(source, field, code string, err error) Diagnostic {
	return Diagnostic{
		Source:   source,
		Field:    field,
		Severity: SeverityError,
		Code:     code,
		Message:  err.Error(),
	}
}

// String returns a human-readable representation of the diagnostic.
func (d Diagnostic) String() string {
	if d.Line > 0 && d.Field != "" {
		return fmt.Sprintf("[%s] %s:%d %s (%s): %s",
			d.Severity,
			d.Source,
			d.Line,
			d.Field,
			d.Code,
			d.Message,
		)
	}

	if d.Line > 0 {
		return fmt.Sprintf("[%s] %s:%d (%s): %s",
			d.Severity,
			d.Source,
			d.Line,
			d.Code,
			d.Message,
		)
	}

	if d.Field != "" {
		return fmt.Sprintf("[%s] %s %s (%s): %s",
			d.Severity,
			d.Source,
			d.Field,
			d.Code,
			d.Message,
		)
	}

	return fmt.Sprintf("[%s] %s (%s): %s",
		d.Severity,
		d.Source,
		d.Code,
		d.Message,
	)
}
