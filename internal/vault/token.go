package vault

import (
	"fmt"
	"sort"

	"github.com/janosmiko/vau/internal/model"
)

// ListTokenAccessors returns all token accessor IDs as entries.
func (c *Client) ListTokenAccessors() ([]model.Entry, error) {
	secret, err := c.raw.Logical().List("auth/token/accessors")
	if err != nil {
		return nil, fmt.Errorf("listing token accessors: %w", err)
	}

	if secret == nil || secret.Data == nil {
		return []model.Entry{}, nil
	}

	keysRaw, ok := secret.Data["keys"]
	if !ok {
		return []model.Entry{}, nil
	}

	keysList, ok := keysRaw.([]any)
	if !ok {
		return nil, fmt.Errorf("unexpected keys type for token accessors")
	}

	entries := make([]model.Entry, 0, len(keysList))
	for _, k := range keysList {
		entries = append(entries, model.Entry{
			Name: fmt.Sprintf("%v", k),
		})
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name < entries[j].Name
	})

	return entries, nil
}

// LookupAccessor returns metadata for a token by its accessor ID.
func (c *Client) LookupAccessor(accessor string) (map[string]any, error) {
	secret, err := c.raw.Logical().Write("auth/token/lookup-accessor", map[string]any{
		"accessor": accessor,
	})
	if err != nil {
		return nil, fmt.Errorf("looking up accessor %s: %w", accessor, err)
	}

	if secret == nil || secret.Data == nil {
		return nil, fmt.Errorf("no data for accessor %s", accessor)
	}

	return secret.Data, nil
}

// CreateToken creates a new token with the given parameters.
// Returns the full response including auth.client_token (only visible once).
func (c *Client) CreateToken(params map[string]any) (map[string]any, error) {
	secret, err := c.raw.Logical().Write("auth/token/create", params)
	if err != nil {
		return nil, fmt.Errorf("creating token: %w", err)
	}

	if secret == nil || secret.Auth == nil {
		return nil, fmt.Errorf("no auth data in token create response")
	}

	result := map[string]any{
		"client_token":   secret.Auth.ClientToken,
		"accessor":       secret.Auth.Accessor,
		"policies":       secret.Auth.Policies,
		"lease_duration": secret.Auth.LeaseDuration,
		"renewable":      secret.Auth.Renewable,
		"orphan":         secret.Auth.Orphan,
	}
	return result, nil
}

// CreateTokenWithRole creates a new token using a token role.
func (c *Client) CreateTokenWithRole(role string, params map[string]any) (map[string]any, error) {
	secret, err := c.raw.Logical().Write("auth/token/create/"+role, params)
	if err != nil {
		return nil, fmt.Errorf("creating token with role %q: %w", role, err)
	}

	if secret == nil || secret.Auth == nil {
		return nil, fmt.Errorf("no auth data in token create response")
	}

	result := map[string]any{
		"client_token":   secret.Auth.ClientToken,
		"accessor":       secret.Auth.Accessor,
		"policies":       secret.Auth.Policies,
		"lease_duration": secret.Auth.LeaseDuration,
		"renewable":      secret.Auth.Renewable,
		"orphan":         secret.Auth.Orphan,
	}
	return result, nil
}

// RevokeAccessor revokes a token by its accessor ID.
func (c *Client) RevokeAccessor(accessor string) error {
	_, err := c.raw.Logical().Write("auth/token/revoke-accessor", map[string]any{
		"accessor": accessor,
	})
	if err != nil {
		return fmt.Errorf("revoking accessor %s: %w", accessor, err)
	}
	return nil
}
