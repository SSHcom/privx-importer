package privx

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/SSHcom/inventory_import_privx_dev/internal/translators"
)

const roleListLimit = 100

var (
	// ErrRoleNameEmpty means that PrivX returned a role without a name.
	ErrRoleNameEmpty = errors.New("privx role name empty")

	// ErrRoleIDEmpty means that PrivX returned a role without an ID.
	ErrRoleIDEmpty = errors.New("privx role id empty")

	// ErrRoleNameDuplicate means that multiple roles have the same name.
	ErrRoleNameDuplicate = errors.New("privx role name duplicate")
)

// Role contains the PrivX role fields needed by the importer.
type Role struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Roles contains PrivX roles indexed by name and ID.
type Roles struct {
	ByName map[string]Role
	ByID   map[string]Role
}

// rolesResponse is the paged PrivX roles response.
type rolesResponse struct {
	Count int    `json:"count"`
	Items []Role `json:"items"`
}

// ListRoles returns all PrivX roles.
func (c *Client) ListRoles(ctx context.Context) ([]Role, error) {
	var result []Role

	for offset := 0; ; offset += roleListLimit {
		query := url.Values{}
		query.Set("offset", fmt.Sprintf("%d", offset))
		query.Set("limit", fmt.Sprintf("%d", roleListLimit))
		query.Set("sortkey", "name")
		query.Set("sortdir", "ASC")

		path := pathRoles + "?" + query.Encode()

		var resp rolesResponse
		if err := c.doJSON(ctx, http.MethodGet, path, nil, &resp); err != nil {
			return nil, err
		}

		result = append(result, resp.Items...)

		if len(resp.Items) == 0 || len(result) >= resp.Count {
			break
		}
	}

	return result, nil
}

// Roles returns role IDs indexed by role name or ID.
func (c *Client) Roles(ctx context.Context) (translators.Dictionary, error) {
	roles, err := c.ListRoles(ctx)
	if err != nil {
		return translators.Dictionary{}, err
	}

	result := translators.Dictionary{
		ByName: make(map[string]translators.DictItem, len(roles)),
		ByID:   make(map[string]translators.DictItem, len(roles)),
	}

	for _, role := range roles {
		name := strings.TrimSpace(role.Name)
		id := strings.TrimSpace(role.ID)

		if name == "" {
			return result, ErrRoleNameEmpty
		}
		if id == "" {
			return result, fmt.Errorf("%w: %s", ErrRoleIDEmpty, name)
		}
		if _, exists := result.ByName[name]; exists {
			return result, fmt.Errorf("%w: %s", ErrRoleNameDuplicate, name)
		}

		role := translators.DictItem{
			Name: name,
			ID:   id,
		}
		result.ByName[name] = role
		result.ByID[id] = role
	}

	return result, nil
}
