package ui

import (
	"testing"

	"github.com/janosmiko/vau/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestRenderAuthMethodListEmpty(t *testing.T) {
	result := RenderAuthMethodList(testParentEntries, 2, nil, 0, nil, nil, 0, "0.1.0", 100, 30)
	assert.Contains(t, result, "Auth Methods/")
	assert.Contains(t, result, "(empty)")
}

func TestRenderAuthMethodListWithMethods(t *testing.T) {
	methods := []model.Entry{
		{Name: "approle/", IsDir: true},
		{Name: "token/", IsDir: true},
	}
	result := RenderAuthMethodList(testParentEntries, 2, methods, 0, nil, nil, 0, "0.1.0", 100, 30)
	assert.Contains(t, result, "approle/")
	assert.Contains(t, result, "token/")
}

func TestRenderAuthMethodListShowsTabBar(t *testing.T) {
	methods := []model.Entry{{Name: "token/", IsDir: true}}
	tabs := []string{"secret/", "kv/"}
	result := RenderAuthMethodList(testParentEntries, 2, methods, 0, nil, tabs, 0, "0.1.0", 100, 30)
	assert.Contains(t, result, "secret/")
	assert.Contains(t, result, "kv/")
}

func TestRenderRoleListWithRoles(t *testing.T) {
	authParent := []model.Entry{{Name: "userpass/", IsDir: true}}
	roles := []model.Entry{{Name: "admin-role"}, {Name: "dev-role"}}
	result := RenderRoleList(authParent, 0, roles, 0, "userpass/", nil, nil, 0, "0.1.0", 100, 30)
	assert.Contains(t, result, "Auth Methods/userpass/")
	assert.Contains(t, result, "admin-role")
}

func TestRenderRoleListEmpty(t *testing.T) {
	authParent := []model.Entry{{Name: "userpass/", IsDir: true}}
	result := RenderRoleList(authParent, 0, nil, 0, "userpass/", nil, nil, 0, "0.1.0", 100, 30)
	assert.Contains(t, result, "(empty)")
}

func TestRenderRoleViewOverlay(t *testing.T) {
	data := map[string]any{"token_ttl": "1h", "policies": "[default]"}
	result := RenderRoleViewOverlay("dev-role", data, 0, 100, 40)
	assert.Contains(t, result, "Role: dev-role")
	assert.Contains(t, result, "token_ttl")
}

func TestRenderRoleViewOverlayEmpty(t *testing.T) {
	result := RenderRoleViewOverlay("empty", nil, 0, 100, 40)
	assert.Contains(t, result, "(no data)")
}

func TestRenderEntityListWithEntities(t *testing.T) {
	entities := []model.Entry{{Name: "user1"}, {Name: "user2"}}
	result := RenderEntityList(testParentEntries, -1, entities, 0, nil, nil, 0, "0.1.0", 100, 30)
	assert.Contains(t, result, "Entities/")
	assert.Contains(t, result, "user1")
}

func TestRenderEntityViewOverlay(t *testing.T) {
	data := map[string]any{
		"name": "admin",
		"aliases": []any{
			map[string]any{"name": "admin", "mount_type": "userpass", "mount_path": "auth/userpass/", "id": "abc"},
		},
	}
	result := RenderEntityViewOverlay("admin", data, 0, 100, 40)
	assert.Contains(t, result, "Entity: admin")
	assert.Contains(t, result, "userpass")
}

func TestRenderGroupListWithGroups(t *testing.T) {
	groups := []model.Entry{{Name: "admins"}, {Name: "devs"}}
	result := RenderGroupList(testParentEntries, -1, groups, 0, nil, nil, 0, "0.1.0", 100, 30)
	assert.Contains(t, result, "Groups/")
	assert.Contains(t, result, "admins")
}

func TestRenderGroupViewOverlay(t *testing.T) {
	data := map[string]any{"policies": "[admin]", "type": "internal"}
	result := RenderGroupViewOverlay("admins", data, 0, 100, 40)
	assert.Contains(t, result, "policies")
}

func TestRenderRoleDataPreviewNil(t *testing.T) {
	result := renderRoleDataPreview(nil, 40, 10)
	assert.Contains(t, result, "Loading...")
}

func TestRenderRoleDataPreviewEmpty(t *testing.T) {
	result := renderRoleDataPreview(map[string]any{}, 40, 10)
	assert.Contains(t, result, "(no data)")
}

func TestSortedMapKeys(t *testing.T) {
	m := map[string]any{"c": 1, "a": 2, "b": 3}
	keys := sortedMapKeys(m)
	assert.Equal(t, []string{"a", "b", "c"}, keys)
}
