package validators

import (
	"errors"
	"fmt"
	"slices"

	"github.com/SSHcom/inventory_import_privx_dev/internal/config"
	"github.com/SSHcom/inventory_import_privx_dev/internal/domain"
	"github.com/SSHcom/inventory_import_privx_dev/internal/field"
	"golang.org/x/crypto/ssh"
)

// Validator runs a single test on a field and returns nil or a diagnostic message.
// Note that field can be a complex field.
type Validator func(field field.Field) *domain.Diagnostic

var Validators = map[config.ValidatorID]Validator{
	config.ValidatorServices:     validateService,
	config.ValidatorDisabled:     validateDisabled,
	config.ValidatorSSHPublicKey: ValidateSSHPublicKey,
}

// Validate a single service object list.
func validateService(f field.Field) *domain.Diagnostic {
	services, ok := f.(field.ListCompositeField)
	if !ok {
		return &domain.Diagnostic{
			Field:    f.Name(),
			Severity: domain.SeverityError,
			Message:  fmt.Sprintf("%q is supposed to be a list object, not %T", f.Name(), f),
		}
	}

	// Check each service in the array.
	for i, svc := range services.Items() {
		_, ok = svc.GetByLeaf("port")
		if !ok {
			return &domain.Diagnostic{
				Field:    fmt.Sprintf("%s[%d].port", f.Name(), i),
				Severity: domain.SeverityError,
				Code:     "required_field_missing",
				Message:  fmt.Sprintf("service definition %d must have a port number defined", i),
			}
		}
		_, ok = svc.GetByLeaf("address")
		if !ok {
			return &domain.Diagnostic{
				Field:    fmt.Sprintf("%s[%d].address", f.Name(), i),
				Severity: domain.SeverityError,
				Code:     "required_field_missing",
				Message:  fmt.Sprintf("service definition %d must have an address defined", i),
			}
		}
	}

	return nil
}

func validateDisabled(f field.Field) *domain.Diagnostic {
	disabled, ok := f.(field.StringField)
	if !ok {
		return &domain.Diagnostic{
			Field:    f.Name(),
			Severity: domain.SeverityError,
			Message:  fmt.Sprintf("%q is supposed to be a string, not %T", f.Name(), f),
		}
	}
	// Note: documents for v44 state "Possible values: [BY_ADMIN, BY_LICENSE, false]" but actually it is "FALSE"
	if slices.Contains([]string{"BY_ADMIN", "BY_LICENSE", "FALSE"}, disabled.String()) {
		return nil
	}
	return &domain.Diagnostic{
		Field:    f.Name(),
		Severity: domain.SeverityError,
		Code:     "enum_fault",
		Message:  fmt.Sprintf("%s can not contain %q, only BY_ADMIN, BY_LICENSE, or FALSE", f.Name(), disabled.String()),
	}
}

var ErrInvalidSSHPublicKey = errors.New("invalid ssh public key")

// ValidateSSHPublicKey validates SSH public key format.
func ValidateSSHPublicKey(f field.Field) *domain.Diagnostic {
	_, _, _, _, err := ssh.ParseAuthorizedKey([]byte(f.Text()))
	if err == nil {
		return nil
	}

	return &domain.Diagnostic{
		Field:    f.Name(),
		Severity: domain.SeverityError,
		Code:     ErrInvalidSSHPublicKey.Error(),
		Message:  err.Error(),
	}
}
