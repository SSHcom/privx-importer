package host

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/SSHcom/inventory_import_privx_dev/internal/field"
)

// ScalarUpdateMode tells how a scalar field should be handled
// when the target host already exists.
type ScalarUpdateMode string

const (
	ScalarUpdateModeUnset   ScalarUpdateMode = ""
	ScalarUpdateModeReplace ScalarUpdateMode = "replace"
	ScalarUpdateModeIgnore  ScalarUpdateMode = "ignore"
	ScalarUpdateModeClear   ScalarUpdateMode = "clear"
)

// ListUpdateMode tells how a list field should be handled
// when the target host already exists.
type ListUpdateMode string

const (
	ListUpdateModeUnset     ListUpdateMode = ""
	ListUpdateModeReplace   ListUpdateMode = "replace"
	ListUpdateModeIgnore    ListUpdateMode = "ignore"
	ListUpdateModeAddToList ListUpdateMode = "add-to-list"
	ListUpdateModeMerge     ListUpdateMode = "merge"
	ListUpdateModeClear     ListUpdateMode = "clear"
)

var (
	ErrHostKeyConflict = errors.New("host key conflict")
)

// HostKey uniquely identifies a host during import.
// Priority / interpretation can be decided later in matching logic.
type HostKey struct {
	CommonName string
	ID         string
	IPAddress  string
}

// Match reports whether two host keys refer to the same host.
// Any shared non-empty identifier is considered a match.
func (k HostKey) Match(other HostKey) bool {
	if k.ID != "" && other.ID != "" &&
		k.ID == other.ID {
		return true
	}

	if k.CommonName != "" && other.CommonName != "" &&
		k.CommonName == other.CommonName {
		return true
	}

	if k.IPAddress != "" && other.IPAddress != "" &&
		k.IPAddress == other.IPAddress {
		return true
	}

	return false
}

// Equal returns true if the compared key is identical in all parts.
func (k HostKey) Equal(other HostKey) bool {
	return k.ID == other.ID && k.CommonName == other.CommonName && k.IPAddress == other.IPAddress
}

// Merge combines two matching host keys into one richer key.
// Conflicting non-empty values for the same identifier return an error.
func (k HostKey) Merge(other HostKey) (HostKey, error) {
	out := k

	if out.ID == "" {
		out.ID = other.ID
	} else if other.ID != "" && out.ID != other.ID {
		return HostKey{}, fmt.Errorf(
			"%w: field=id current=%q incoming=%q",
			ErrHostKeyConflict,
			out.ID,
			other.ID,
		)
	}

	if out.CommonName == "" {
		out.CommonName = other.CommonName
	} else if other.CommonName != "" && out.CommonName != other.CommonName {
		return HostKey{}, fmt.Errorf(
			"%w: field=hostname current=%q incoming=%q",
			ErrHostKeyConflict,
			out.CommonName,
			other.CommonName,
		)
	}

	if out.IPAddress == "" {
		out.IPAddress = other.IPAddress
	} else if other.IPAddress != "" && out.IPAddress != other.IPAddress {
		return HostKey{}, fmt.Errorf(
			"%w: field=ip_address current=%q incoming=%q",
			ErrHostKeyConflict,
			out.IPAddress,
			other.IPAddress,
		)
	}

	return out, nil
}

// BestKey returns the strongest available canonical key.
// Priority order: ID > CommonName > IPAddress.
func (k HostKey) BestKey() string {
	switch {
	case k.ID != "":
		return "id:" + k.ID

	case k.CommonName != "":
		return "hostname:" + k.CommonName

	case k.IPAddress != "":
		return "ip_address:" + k.IPAddress

	default:
		return ""
	}
}

// BestKey returns the most human readable key.
func (k HostKey) Human() string {
	switch {

	case k.CommonName != "":
		return k.CommonName

	case k.IPAddress != "":
		return k.IPAddress

	case k.ID != "":
		return k.ID

	default:
		return ""
	}
}

func (k HostKey) String() string {
	parts := make([]string, 0, 5)

	if k.ID != "" {
		parts = append(parts, "id="+k.ID)
	}
	if k.CommonName != "" {
		parts = append(parts, "hostname="+k.CommonName)
	}
	if k.IPAddress != "" {
		parts = append(parts, "ip_address="+k.IPAddress)
	}

	if len(parts) == 0 {
		return "<empty>"
	}

	return strings.Join(parts, ", ")
}

func (k HostKey) IsEmpty() bool {
	return k.CommonName == "" && k.ID == "" && k.IPAddress == ""
}

// HostKeyFromFields returns the best-effort host key derived from generic fields.
func HostKeyFromFields(f field.Fields) HostKey {
	return HostKey{
		ID:         f.Text("id"),
		CommonName: f.Text("common_name"),
		IPAddress:  firstAddress(f),
	}
}

func firstAddress(f field.Fields) string {
	addressesField, ok := f["addresses"]
	if !ok {
		return ""
	}

	addresses, ok := addressesField.Value().([]string)
	if !ok || len(addresses) == 0 {
		return ""
	}

	return addresses[0]
}

// Disabled describes disabled state and optional reason.
// Nil means "not specified".
type Disabled struct {
	Value  bool
	Reason string
}

// SessionRecordingOptions describes session recording configuration.
// Nil means "not specified".
type SessionRecordingOptions struct {
	Enabled        bool
	RecordTerminal bool
	RecordCommands bool
}

// TargetServiceSpec describes one target service item.
type TargetServiceSpec struct {
	Name string
	Port int
}

// TargetPrincipalSpec describes one target principal item.
type TargetPrincipalSpec struct {
	Name string
}

// LookupKey gives alternative ways to identify a host for search.
type LookupKey struct {
	ID         string // PrivX UUID
	CommonName string
	Address    string
}

type Match struct {
	ID         string
	CommonName string
	Addresses  []string
}

type ResolveResult struct {
	Matches []Match
}

func (r ResolveResult) None() bool   { return len(r.Matches) == 0 }
func (r ResolveResult) Unique() bool { return len(r.Matches) == 1 }
func (r ResolveResult) Many() bool   { return len(r.Matches) > 1 }

type Resolver interface {
	Resolve(ctx context.Context, key LookupKey) (ResolveResult, error)
}
