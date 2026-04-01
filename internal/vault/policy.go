package vault

import (
	"fmt"
	"sort"
)

// ListPolicies returns all ACL policy names from Vault, sorted alphabetically.
func (c *Client) ListPolicies() ([]string, error) {
	policies, err := c.raw.Sys().ListPolicies()
	if err != nil {
		return nil, fmt.Errorf("listing policies: %w", err)
	}

	if policies == nil {
		return []string{}, nil
	}

	sort.Strings(policies)

	return policies, nil
}

// GetPolicy returns the HCL content of a named policy.
func (c *Client) GetPolicy(name string) (string, error) {
	policy, err := c.raw.Sys().GetPolicy(name)
	if err != nil {
		return "", fmt.Errorf("getting policy %q: %w", name, err)
	}

	return policy, nil
}

// PutPolicy creates or updates a policy with the given HCL rules.
func (c *Client) PutPolicy(name, rules string) error {
	if err := c.raw.Sys().PutPolicy(name, rules); err != nil {
		return fmt.Errorf("writing policy %q: %w", name, err)
	}
	return nil
}

// DeletePolicy removes a policy by name.
func (c *Client) DeletePolicy(name string) error {
	if err := c.raw.Sys().DeletePolicy(name); err != nil {
		return fmt.Errorf("deleting policy %q: %w", name, err)
	}
	return nil
}
