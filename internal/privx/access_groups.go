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

const accessGroupListLimit = 100

var (
	// ErrAccessGroupNameEmpty means that PrivX returned an access group without a name.
	ErrAccessGroupNameEmpty = errors.New("privx access group name empty")

	// ErrAccessGroupIDEmpty means that PrivX returned an access group without an ID.
	ErrAccessGroupIDEmpty = errors.New("privx access group id empty")

	// ErrAccessGroupNameDuplicate means that multiple access groups have the same name.
	ErrAccessGroupNameDuplicate = errors.New("privx access group name duplicate")
)

// AccessGroup contains the PrivX access group fields needed by the importer.
type AccessGroup struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// accessGroupsResponse is the paged PrivX access groups response.
type accessGroupsResponse struct {
	Count int           `json:"count"`
	Items []AccessGroup `json:"items"`
}

// ListAccessGroups returns all PrivX access groups.
func (c *Client) ListAccessGroups(ctx context.Context) ([]AccessGroup, error) {
	var result []AccessGroup

	for offset := 0; ; offset += accessGroupListLimit {
		query := url.Values{}
		query.Set("offset", fmt.Sprintf("%d", offset))
		query.Set("limit", fmt.Sprintf("%d", accessGroupListLimit))
		query.Set("sortkey", "name")
		query.Set("sortdir", "ASC")

		path := pathAccessGroups + "?" + query.Encode()

		var resp accessGroupsResponse
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

// AccessGroups returns access group IDs indexed by access group name or ID.
func (c *Client) AccessGroups(ctx context.Context) (translators.Dictionary, error) {
	groups, err := c.ListAccessGroups(ctx)
	if err != nil {
		return translators.Dictionary{}, err
	}

	result := translators.Dictionary{
		ByName: make(map[string]translators.DictItem, len(groups)),
		ByID:   make(map[string]translators.DictItem, len(groups)),
	}

	for _, group := range groups {
		name := strings.TrimSpace(group.Name)
		id := strings.TrimSpace(group.ID)

		if name == "" {
			return result, ErrAccessGroupNameEmpty
		}
		if id == "" {
			return result, fmt.Errorf("%w: %s", ErrAccessGroupIDEmpty, name)
		}
		if _, exists := result.ByName[name]; exists {
			return result, fmt.Errorf("%w: %s", ErrAccessGroupNameDuplicate, name)
		}

		group := translators.DictItem{
			Name: name,
			ID:   id,
		}
		result.ByName[name] = group
		result.ByID[id] = group
	}

	return result, nil
}
