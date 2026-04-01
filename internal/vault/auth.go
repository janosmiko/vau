package vault

import (
	"fmt"
	"sort"
	"strings"

	"github.com/janosmiko/vau/internal/model"
)

// authMethodTypes caches the mapping from mount path to auth method type.
var authMethodTypes map[string]string

// ListAuthMethods returns all enabled auth methods as entries, sorted alphabetically.
func (c *Client) ListAuthMethods() ([]model.Entry, error) {
	auths, err := c.raw.Sys().ListAuth()
	if err != nil {
		return nil, fmt.Errorf("listing auth methods: %w", err)
	}

	if auths == nil {
		return []model.Entry{}, nil
	}

	// Cache auth method types for role path resolution
	authMethodTypes = make(map[string]string, len(auths))
	entries := make([]model.Entry, 0, len(auths))
	for path, info := range auths {
		entries = append(entries, model.Entry{
			Name:  path,
			IsDir: true,
		})
		if info != nil {
			authMethodTypes[path] = info.Type
		}
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name < entries[j].Name
	})

	return entries, nil
}

// AuthMethodType returns the type of an auth method by its mount path (e.g. "token", "approle").
func AuthMethodType(authPath string) string {
	if authMethodTypes != nil {
		return authMethodTypes[authPath]
	}
	return ""
}

// roleSubpath returns the API subpath for roles based on auth method type.
// Most methods use "role", token uses "roles", userpass uses "users", ldap uses "groups".
func roleSubpath(authPath string) string {
	if authMethodTypes != nil {
		authType := authMethodTypes[authPath]
		switch authType {
		case "token":
			return "roles"
		case "userpass", "radius":
			return "users"
		case "ldap":
			return "groups"
		case "cert":
			return "certs"
		}
	}
	return "role"
}

// ListRoles returns role names for a given auth method path.
func (c *Client) ListRoles(authPath string) ([]model.Entry, error) {
	apiPath := "auth/" + strings.TrimSuffix(authPath, "/") + "/" + roleSubpath(authPath)

	secret, err := c.raw.Logical().List(apiPath)
	if err != nil {
		return nil, fmt.Errorf("listing roles for %s: %w", authPath, err)
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
		return nil, fmt.Errorf("unexpected keys type for roles at %s", authPath)
	}

	entries := make([]model.Entry, 0, len(keysList))
	for _, k := range keysList {
		entries = append(entries, model.Entry{
			Name:  fmt.Sprintf("%v", k),
			IsDir: false,
		})
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name < entries[j].Name
	})

	return entries, nil
}

// GetRole returns the configuration data for a specific role under an auth method.
func (c *Client) GetRole(authPath, name string) (map[string]any, error) {
	apiPath := "auth/" + strings.TrimSuffix(authPath, "/") + "/" + roleSubpath(authPath) + "/" + name

	secret, err := c.raw.Logical().Read(apiPath)
	if err != nil {
		return nil, fmt.Errorf("reading role %q at %s: %w", name, authPath, err)
	}

	if secret == nil || secret.Data == nil {
		return nil, fmt.Errorf("role %q not found at %s", name, authPath)
	}

	return secret.Data, nil
}

// WriteRole creates or updates a role under an auth method.
func (c *Client) WriteRole(authPath, name string, data map[string]any) error {
	apiPath := "auth/" + strings.TrimSuffix(authPath, "/") + "/" + roleSubpath(authPath) + "/" + name
	_, err := c.raw.Logical().Write(apiPath, data)
	if err != nil {
		return fmt.Errorf("writing role %q at %s: %w", name, authPath, err)
	}
	return nil
}

// DeleteRole removes a role from an auth method.
func (c *Client) DeleteRole(authPath, name string) error {
	apiPath := "auth/" + strings.TrimSuffix(authPath, "/") + "/" + roleSubpath(authPath) + "/" + name
	_, err := c.raw.Logical().Delete(apiPath)
	if err != nil {
		return fmt.Errorf("deleting role %q at %s: %w", name, authPath, err)
	}
	return nil
}

// ResolveApproleRoleIDs builds a map from role_id -> role_name for an approle mount.
// mountPath is the full path from the entity alias (e.g. "auth/approle/").
func (c *Client) ResolveApproleRoleIDs(mountPath string) (map[string]string, error) {
	// mountPath from entity alias is already fully qualified (e.g. "auth/approle/").
	// Strip the "auth/" prefix since ListRoles adds it back.
	stripped := strings.TrimPrefix(mountPath, "auth/")

	roles, err := c.ListRoles(stripped)
	if err != nil {
		return nil, err
	}

	result := make(map[string]string, len(roles))
	trimmed := strings.TrimSuffix(stripped, "/")
	for _, r := range roles {
		apiPath := "auth/" + trimmed + "/role/" + r.Name + "/role-id"
		secret, err := c.raw.Logical().Read(apiPath)
		if err != nil || secret == nil || secret.Data == nil {
			continue
		}
		if roleID, ok := secret.Data["role_id"]; ok {
			result[fmt.Sprintf("%v", roleID)] = r.Name
		}
	}
	return result, nil
}

// EnrichEntityAliases resolves alias names (like approle role_ids) to human-readable names.
// It modifies the entity data map in place, adding a "display_name" field to each alias.
func (c *Client) EnrichEntityAliases(data map[string]any) {
	aliasesRaw, ok := data["aliases"]
	if !ok {
		return
	}
	aliases, ok := aliasesRaw.([]any)
	if !ok || len(aliases) == 0 {
		return
	}

	// Collect unique approle mount paths that need resolution
	approleCache := make(map[string]map[string]string) // mountPath → role_id → role_name

	for _, aRaw := range aliases {
		aMap, ok := aRaw.(map[string]any)
		if !ok {
			continue
		}
		mountType, _ := aMap["mount_type"].(string)
		mountPath, _ := aMap["mount_path"].(string)
		if mountType != "approle" || mountPath == "" {
			continue
		}
		if _, cached := approleCache[mountPath]; !cached {
			roleMap, err := c.ResolveApproleRoleIDs(mountPath)
			if err != nil {
				approleCache[mountPath] = nil
				continue
			}
			approleCache[mountPath] = roleMap
		}
	}

	// Enrich each alias with display_name
	for _, aRaw := range aliases {
		aMap, ok := aRaw.(map[string]any)
		if !ok {
			continue
		}
		aliasName, _ := aMap["name"].(string)
		mountType, _ := aMap["mount_type"].(string)
		mountPath, _ := aMap["mount_path"].(string)

		if mountType == "approle" {
			if roleMap := approleCache[mountPath]; roleMap != nil {
				if roleName, found := roleMap[aliasName]; found {
					aMap["display_name"] = roleName
					continue
				}
			}
		}
		// For non-approle or unresolved, display_name = name
		aMap["display_name"] = aliasName
	}
}

// ListEntities returns all identity entities as entries.
func (c *Client) ListEntities() ([]model.Entry, error) {
	secret, err := c.raw.Logical().List("identity/entity/name")
	if err != nil {
		return nil, fmt.Errorf("listing entities: %w", err)
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
		return nil, fmt.Errorf("unexpected keys type for entities")
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

// ListGroups returns all identity groups as entries.
func (c *Client) ListGroups() ([]model.Entry, error) {
	secret, err := c.raw.Logical().List("identity/group/name")
	if err != nil {
		return nil, fmt.Errorf("listing groups: %w", err)
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
		return nil, fmt.Errorf("unexpected keys type for groups")
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

// GetEntity returns entity data by name.
func (c *Client) GetEntity(name string) (map[string]any, error) {
	secret, err := c.raw.Logical().Read("identity/entity/name/" + name)
	if err != nil {
		return nil, fmt.Errorf("reading entity %q: %w", name, err)
	}
	if secret == nil || secret.Data == nil {
		return nil, fmt.Errorf("entity %q not found", name)
	}
	return secret.Data, nil
}

// GetGroup returns group data by name.
func (c *Client) GetGroup(name string) (map[string]any, error) {
	secret, err := c.raw.Logical().Read("identity/group/name/" + name)
	if err != nil {
		return nil, fmt.Errorf("reading group %q: %w", name, err)
	}
	if secret == nil || secret.Data == nil {
		return nil, fmt.Errorf("group %q not found", name)
	}
	return secret.Data, nil
}
