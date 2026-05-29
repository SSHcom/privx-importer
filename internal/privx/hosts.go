package privx

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/SSHcom/inventory_import_privx_dev/internal/config"
	"github.com/SSHcom/inventory_import_privx_dev/internal/domain"
	"github.com/SSHcom/inventory_import_privx_dev/internal/domain/host"
	"github.com/SSHcom/inventory_import_privx_dev/internal/field"
	"github.com/SSHcom/inventory_import_privx_dev/internal/output"
	"github.com/SSHcom/inventory_import_privx_dev/internal/plan"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

var (
	// ErrResolveKeyEmpty is returned when no lookup fields are given.
	ErrResolveKeyEmpty = errors.New("host resolve: lookup key empty")
	// ErrPrivXFindHostBadStatus is returned when host search returns a non-success status.
	ErrPrivXFindHostBadStatus = errors.New("privx find host: bad http status")
	// ErrPrivXGetHostBadStatus is returned when get host returns a non-success status.
	ErrPrivXGetHostBadStatus = errors.New("privx get host: bad http status")
	// ErrHostIDEmpty is returned when host ID is empty.
	ErrHostIDEmpty = errors.New("host id is empty")
	// ErrHostKeyMissing is returned if key can not be extracted from API response JSON:
	ErrHostKeyMissing = errors.New("host key missing")
	// ErrPrivXSetHostDisabledFailed is returned when enabling or disabling a host fails.
	ErrPrivXSetHostDisabledFailed = errors.New("privx set host disabled failed")

	ErrPrivXDeleteHostFailed = errors.New("privx delete host failed")
)

type hostAPI struct {
	ID         string   `json:"id"`
	CommonName string   `json:"common_name"`
	Addresses  []string `json:"addresses"`
}

type hostSearchResponse struct {
	// TODO: Verify whether v43 uses "items", "hosts", or another collection field.
	Items []hostAPI `json:"items"`
}

// Resolve returns host.ResolveResult, including UUID, using some human readable identification.
func (c *Client) Resolve(ctx context.Context, key host.LookupKey) (host.ResolveResult, error) {
	if isLookupKeyEmpty(key) {
		return host.ResolveResult{}, ErrResolveKeyEmpty
	}

	if key.ID != "" {
		m, found, err := c.getHostMatch(ctx, key.ID)
		if err != nil {
			return host.ResolveResult{}, err
		}
		if !found {
			return host.ResolveResult{}, nil
		}
		return host.ResolveResult{
			Matches: []host.Match{m},
		}, nil
	}

	req := buildHostSearch(key)

	var resp hostSearchResponse
	if err := c.doJSON(ctx, http.MethodPost, "/hosts/search", req, &resp); err != nil {
		return host.ResolveResult{}, err
	}

	return host.ResolveResult{
		Matches: mapHostMatches(resp.Items),
	}, nil
}

func (c *Client) getHostMatch(ctx context.Context, hostID string) (host.Match, bool, error) {
	var apiHost hostAPI
	path := "/hosts/" + url.PathEscape(hostID)

	err := c.doJSON(ctx, http.MethodGet, path, nil, &apiHost)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return host.Match{}, false, nil
		}
		return host.Match{}, false, err
	}

	return mapHostMatch(apiHost), true, nil
}

func mapHostMatches(items []hostAPI) []host.Match {
	out := make([]host.Match, 0, len(items))
	for _, item := range items {
		out = append(out, mapHostMatch(item))
	}
	return out
}

func mapHostMatch(item hostAPI) host.Match {
	return host.Match{
		ID:         item.ID,
		CommonName: item.CommonName,
		Addresses:  append([]string(nil), item.Addresses...),
	}
}

// HostFilter contains field values used to filter hosts to retrieve.
//
// An example is to use a tag to identify which hosts are under our automatic control.
type HostFilter struct {
	Tags []string
}

// HostDTO contains only those fields from API which are of interest to us.
type HostDTO struct {
	ID         string   `json:"id"`
	CommonName string   `json:"common_name"`
	Addresses  []string `json:"addresses"`
	Tags       []string `json:"tags"`
}

// HostKey returns the host key derived from the PrivX host DTO.
func (h HostDTO) HostKey() host.HostKey {
	key := host.HostKey{
		ID:         h.ID,
		CommonName: h.CommonName,
	}

	if len(h.Addresses) > 0 {
		key.IPAddress = h.Addresses[0]
	}

	return key
}

// ListHosts returns all hosts.
func (c *Client) ListHosts(
	ctx context.Context,
	catalog config.FieldCatalog,
	out output.Output,
) (map[string]plan.TargetSpec, error) {
	const pageSize = 200

	search := HostSearch{}

	all := make(map[string]plan.TargetSpec)
	offset := 0

	for {
		var resp genericRawJSONResponse

		path := fmt.Sprintf(pathHostsSearch+"?offset=%d&limit=%d", offset, pageSize)

		if err := c.doJSON(ctx, http.MethodPost, path, search, &resp); err != nil {
			return nil, fmt.Errorf("list hosts: %w", err)
		}

		if len(resp.Items) == 0 {
			break
		}

		c.debugger.Debugf("%d hosts received from PrivX\n", resp.Count)

		for _, raw := range resp.Items {
			fields, err := catalog.DecodeFields(out, raw)
			if err != nil {
				c.debugger.DumpTrace("Item from PrivX failed to decode:", raw)
				return nil, err
			}

			keys := host.HostKeyFromFields(fields)
			key := keys.BestKey()

			if key == "" {
				return nil, ErrHostKeyMissing
			}

			all[key] = plan.TargetSpec{
				Keys:   keys,
				Key:    key,
				Fields: fields,
			}
		}

		if len(all) >= resp.Count {
			break
		}

		offset += len(resp.Items)
	}

	return all, nil
}

// GetHost reads host data from PrivX into a field list.
func (c *Client) GetHost(
	ctx context.Context,
	hostID openapi_types.UUID,
	catalog config.FieldCatalog,
	out output.Output,
) (field.Fields, error) {
	path := fmt.Sprintf(pathHosts+"/%s", url.PathEscape(hostID.String()))

	var raw map[string]any
	if err := c.doJSON(ctx, "GET", path, nil, &raw); err != nil {
		return nil, err
	}

	fields, err := catalog.DecodeFields(out, raw)
	if err != nil {
		return nil, err
	}

	return fields, nil
}

// CreateHost creates a new host in PrivX and returns the created host ID.
func (c *Client) CreateHost(
	ctx context.Context,
	out output.Output,
	fields field.Fields,
) (string, error) {
	var resp struct {
		ID string `json:"id"`
	}

	jsonCompatible := fields.Encode()
	if err := c.doJSON(
		ctx,
		http.MethodPost,
		pathHosts,
		jsonCompatible,
		&resp,
	); err != nil {
		out.DumpStruct(output.VerbosityDebug, "JSON compatible payload", jsonCompatible, config.MaskedMap)
		return "", err
	}

	return resp.ID, nil
}

// UpdateHost updates an existing host in PrivX.
func (c *Client) UpdateHost(
	ctx context.Context,
	out output.Output,
	fields field.Fields,
) error {

	id, ok := fields["id"]
	if !ok {
		out.Diagnostic(domain.Diagnostic{
			Severity: domain.SeverityError,
			Message:  fmt.Sprintf("Can't update host %s, ID is not known", host.HostKeyFromFields(fields).BestKey()),
		})
		return ErrHostKeyMissing
	}
	path := fmt.Sprintf(
		pathHosts+"/%s",
		url.PathEscape(id.Text()),
	)

	var resp Host

	jsonCompatible := fields.Encode()
	if err := c.doJSON(
		ctx,
		"PUT",
		path,
		jsonCompatible,
		&resp,
	); err != nil || out.HasVerbosity(output.VerbosityTrace) {
		out.DumpStruct(output.VerbosityDebug, "JSON compatible payload", jsonCompatible, config.MaskedMap)
		return err
	}

	return nil
}

// SetHostDisabled enables or disables an existing PrivX host.
func (c *Client) SetHostDisabled(
	ctx context.Context,
	hostID string,
	disabled bool,
) error {
	if hostID == "" {
		return ErrHostIDEmpty
	}

	path := fmt.Sprintf(
		pathHosts+"/%s/disabled",
		url.PathEscape(hostID),
	)

	req := struct {
		Disabled bool `json:"disabled"`
	}{
		Disabled: disabled,
	}

	if err := c.doJSON(ctx, http.MethodPut, path, req, nil); err != nil {
		return fmt.Errorf("%w: %w", ErrPrivXSetHostDisabledFailed, err)
	}

	return nil
}

// DeleteHost deletes an existing PrivX host.
func (c *Client) DeleteHost(ctx context.Context, hostID string) error {
	if hostID == "" {
		return ErrHostIDEmpty
	}

	path := fmt.Sprintf(
		pathHosts+"/%s",
		url.PathEscape(hostID),
	)

	if err := c.doJSON(ctx, http.MethodDelete, path, nil, nil); err != nil {
		return fmt.Errorf("%w: %w", ErrPrivXDeleteHostFailed, err)
	}

	return nil
}

// Build search criteria with priority ExternalID > Hostname > Address.
func buildHostSearch(key host.LookupKey) HostSearch {
	var s HostSearch

	switch {
	case key.ID != "":
		s.ExternalId = &key.ID

	case key.CommonName != "":
		s.CommonName = &[]string{key.CommonName}

	case key.Address != "":
		s.Address = &[]string{key.Address}
	}

	return s
}

func isLookupKeyEmpty(key host.LookupKey) bool {
	return key.ID == "" &&
		key.CommonName == "" &&
		key.Address == ""
}
