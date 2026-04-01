package app

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/janosmiko/vau/internal/model"
	"github.com/stretchr/testify/assert"
)

// ---------------------------------------------------------------------------
// enterAccessCategory
// ---------------------------------------------------------------------------

func TestEnterAccessCategory(t *testing.T) {
	t.Run("policies switches to policy list", func(t *testing.T) {
		m := newTestModel()
		result, cmd := m.enterAccessCategory(accessPolicies)
		rm := result.(*Model)
		assert.Equal(t, model.ModePolicyList, rm.mode)
		assert.NotNil(t, cmd)
	})

	t.Run("auth methods switches to auth methods view", func(t *testing.T) {
		m := newTestModel()
		result, cmd := m.enterAccessCategory(accessAuthMethods)
		rm := result.(*Model)
		assert.Equal(t, model.ModeAuthMethods, rm.mode)
		assert.NotNil(t, cmd)
	})

	t.Run("entities switches to entity list", func(t *testing.T) {
		m := newTestModel()
		result, cmd := m.enterAccessCategory(accessEntities)
		rm := result.(*Model)
		assert.Equal(t, model.ModeEntityList, rm.mode)
		assert.NotNil(t, cmd)
	})

	t.Run("groups switches to group list", func(t *testing.T) {
		m := newTestModel()
		result, cmd := m.enterAccessCategory(accessGroups)
		rm := result.(*Model)
		assert.Equal(t, model.ModeGroupList, rm.mode)
		assert.NotNil(t, cmd)
	})

	t.Run("leases shows not yet implemented", func(t *testing.T) {
		m := newTestModel()
		result, cmd := m.enterAccessCategory(accessLeases)
		rm := result.(*Model)
		assert.Contains(t, rm.status, "not yet implemented")
		assert.Nil(t, cmd)
	})
}

// ---------------------------------------------------------------------------
// handlePolicyListKey
// ---------------------------------------------------------------------------

func TestHandlePolicyListKey(t *testing.T) {
	t.Run("j increments cursor", func(t *testing.T) {
		m := newTestModel()
		m.policies = []string{"admin", "default", "readonly"}
		m.policyCursor = 0

		result, cmd := m.handlePolicyListKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
		rm := result.(*Model)
		assert.Equal(t, 1, rm.policyCursor)
		assert.NotNil(t, cmd)
	})

	t.Run("j does not move past last item", func(t *testing.T) {
		m := newTestModel()
		m.policies = []string{"admin", "default"}
		m.policyCursor = 1

		result, _ := m.handlePolicyListKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
		rm := result.(*Model)
		assert.Equal(t, 1, rm.policyCursor)
	})

	t.Run("k decrements cursor", func(t *testing.T) {
		m := newTestModel()
		m.policies = []string{"admin", "default", "readonly"}
		m.policyCursor = 2

		result, cmd := m.handlePolicyListKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
		rm := result.(*Model)
		assert.Equal(t, 1, rm.policyCursor)
		assert.NotNil(t, cmd)
	})

	t.Run("enter loads selected policy", func(t *testing.T) {
		m := newTestModel()
		m.policies = []string{"default", "admin"}
		_, cmd := m.handlePolicyListKey(tea.KeyMsg{Type: tea.KeyEnter})
		assert.NotNil(t, cmd)
	})

	t.Run("esc goes back to mount level", func(t *testing.T) {
		m := newTestModel()
		m.mode = model.ModePolicyList

		result, _ := m.handlePolicyListKey(tea.KeyMsg{Type: tea.KeyEsc})
		rm := result.(*Model)
		assert.Equal(t, model.ModeExplorer, rm.mode)
		assert.True(t, rm.atMountLevel)
	})
}

// ---------------------------------------------------------------------------
// handlePolicyViewKey
// ---------------------------------------------------------------------------

func TestHandlePolicyViewKey(t *testing.T) {
	t.Run("j increments scroll", func(t *testing.T) {
		m := newTestModel()
		result, _ := m.handlePolicyViewKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
		assert.Equal(t, 1, result.(*Model).policyScroll)
	})

	t.Run("k decrements scroll", func(t *testing.T) {
		m := newTestModel()
		m.policyScroll = 5
		result, _ := m.handlePolicyViewKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
		assert.Equal(t, 4, result.(*Model).policyScroll)
	})

	t.Run("ctrl+d scrolls down 10", func(t *testing.T) {
		m := newTestModel()
		m.policyScroll = 5
		result, _ := m.handlePolicyViewKey(tea.KeyMsg{Type: tea.KeyCtrlD})
		assert.Equal(t, 15, result.(*Model).policyScroll)
	})

	t.Run("ctrl+u scrolls up clamped to 0", func(t *testing.T) {
		m := newTestModel()
		m.policyScroll = 3
		result, _ := m.handlePolicyViewKey(tea.KeyMsg{Type: tea.KeyCtrlU})
		assert.Equal(t, 0, result.(*Model).policyScroll)
	})

	t.Run("esc returns to policy list", func(t *testing.T) {
		m := newTestModel()
		m.mode = model.ModePolicyView
		m.policyContent = "some HCL"
		result, _ := m.handlePolicyViewKey(tea.KeyMsg{Type: tea.KeyEsc})
		rm := result.(*Model)
		assert.Equal(t, model.ModePolicyList, rm.mode)
		assert.Empty(t, rm.policyContent)
	})

	t.Run("g/G navigation", func(t *testing.T) {
		m := newTestModel()
		m.policyContent = "line1\nline2\nline3\nline4\nline5"

		result, _ := m.handlePolicyViewKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
		rm := result.(*Model)
		assert.Greater(t, rm.policyScroll, 0)

		result, _ = rm.handlePolicyViewKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
		rm = result.(*Model)
		assert.Equal(t, 0, rm.policyScroll)
	})
}

// ---------------------------------------------------------------------------
// handleAuthMethodsKey
// ---------------------------------------------------------------------------

func TestHandleAuthMethodsKey(t *testing.T) {
	t.Run("j increments cursor", func(t *testing.T) {
		m := newTestModel()
		m.authMethods = []model.Entry{{Name: "userpass/"}, {Name: "approle/"}}
		result, _ := m.handleAuthMethodsKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
		assert.Equal(t, 1, result.(*Model).authCursor)
	})

	t.Run("enter loads roles", func(t *testing.T) {
		m := newTestModel()
		m.authMethods = []model.Entry{{Name: "userpass/"}}
		_, cmd := m.handleAuthMethodsKey(tea.KeyMsg{Type: tea.KeyEnter})
		assert.NotNil(t, cmd)
	})

	t.Run("esc goes back to mount level", func(t *testing.T) {
		m := newTestModel()
		m.mode = model.ModeAuthMethods
		result, _ := m.handleAuthMethodsKey(tea.KeyMsg{Type: tea.KeyEsc})
		rm := result.(*Model)
		assert.Equal(t, model.ModeExplorer, rm.mode)
		assert.True(t, rm.atMountLevel)
	})
}

// ---------------------------------------------------------------------------
// handleRoleListKey
// ---------------------------------------------------------------------------

func TestHandleRoleListKey(t *testing.T) {
	t.Run("j increments cursor", func(t *testing.T) {
		m := newTestModel()
		m.roles = []model.Entry{{Name: "dev"}, {Name: "admin"}}
		m.roleAuthPath = "userpass/"
		result, _ := m.handleRoleListKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
		assert.Equal(t, 1, result.(*Model).roleCursor)
	})

	t.Run("enter loads role data", func(t *testing.T) {
		m := newTestModel()
		m.roles = []model.Entry{{Name: "dev"}}
		m.roleAuthPath = "userpass/"
		_, cmd := m.handleRoleListKey(tea.KeyMsg{Type: tea.KeyEnter})
		assert.NotNil(t, cmd)
	})

	t.Run("esc goes back to auth methods", func(t *testing.T) {
		m := newTestModel()
		m.mode = model.ModeRoleList
		result, _ := m.handleRoleListKey(tea.KeyMsg{Type: tea.KeyEsc})
		assert.Equal(t, model.ModeAuthMethods, result.(*Model).mode)
	})
}

// ---------------------------------------------------------------------------
// handleRoleViewKey
// ---------------------------------------------------------------------------

func TestHandleRoleViewKey(t *testing.T) {
	t.Run("j increments scroll", func(t *testing.T) {
		m := newTestModel()
		result, _ := m.handleRoleViewKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
		assert.Equal(t, 1, result.(*Model).roleScroll)
	})

	t.Run("ctrl+d scrolls down 10", func(t *testing.T) {
		m := newTestModel()
		m.roleScroll = 3
		result, _ := m.handleRoleViewKey(tea.KeyMsg{Type: tea.KeyCtrlD})
		assert.Equal(t, 13, result.(*Model).roleScroll)
	})

	t.Run("esc returns to role list", func(t *testing.T) {
		m := newTestModel()
		m.mode = model.ModeRoleView
		m.roleName = "dev"
		m.roleData = map[string]any{"k": "v"}
		result, _ := m.handleRoleViewKey(tea.KeyMsg{Type: tea.KeyEsc})
		rm := result.(*Model)
		assert.Equal(t, model.ModeRoleList, rm.mode)
		assert.Nil(t, rm.roleData)
	})
}

// ---------------------------------------------------------------------------
// Message handling via Update
// ---------------------------------------------------------------------------

func TestPolicyListMsgHandling(t *testing.T) {
	t.Run("sets policies", func(t *testing.T) {
		m := newTestModel()
		m.width, m.height = 120, 40
		result, cmd := m.Update(policyListMsg{policies: []string{"default", "admin"}})
		rm := result.(*Model)
		assert.Equal(t, []string{"default", "admin"}, rm.policies)
		assert.NotNil(t, cmd)
	})

	t.Run("error sets errMsg", func(t *testing.T) {
		m := newTestModel()
		m.width, m.height = 120, 40
		result, _ := m.Update(policyListMsg{err: assert.AnError})
		assert.NotEmpty(t, result.(*Model).errMsg)
	})
}

func TestPolicyContentMsgHandling(t *testing.T) {
	t.Run("sets content and switches to view", func(t *testing.T) {
		m := newTestModel()
		m.width, m.height = 120, 40
		result, _ := m.Update(policyContentMsg{name: "admin", content: "path {}"})
		rm := result.(*Model)
		assert.Equal(t, "admin", rm.policyName)
		assert.Equal(t, model.ModePolicyView, rm.mode)
	})
}

func TestAuthMethodsMsgHandling(t *testing.T) {
	t.Run("sets methods", func(t *testing.T) {
		m := newTestModel()
		m.width, m.height = 120, 40
		methods := []model.Entry{{Name: "userpass/"}}
		result, cmd := m.Update(authMethodsMsg{methods: methods})
		assert.Equal(t, methods, result.(*Model).authMethods)
		assert.NotNil(t, cmd)
	})
}

func TestRoleListMsgHandling(t *testing.T) {
	t.Run("sets roles", func(t *testing.T) {
		m := newTestModel()
		m.width, m.height = 120, 40
		roles := []model.Entry{{Name: "dev"}}
		result, _ := m.Update(roleListMsg{authPath: "userpass/", roles: roles})
		assert.Equal(t, model.ModeRoleList, result.(*Model).mode)
	})
}

func TestRoleDataMsgHandling(t *testing.T) {
	t.Run("sets role data", func(t *testing.T) {
		m := newTestModel()
		m.width, m.height = 120, 40
		data := map[string]any{"ttl": "1h"}
		result, _ := m.Update(roleDataMsg{authPath: "up/", name: "dev", data: data})
		assert.Equal(t, model.ModeRoleView, result.(*Model).mode)
	})
}

// ---------------------------------------------------------------------------
// isAccessCategory
// ---------------------------------------------------------------------------

func TestIsAccessCategory(t *testing.T) {
	assert.True(t, isAccessCategory(accessPolicies))
	assert.True(t, isAccessCategory(accessAuthMethods))
	assert.True(t, isAccessCategory(accessEntities))
	assert.True(t, isAccessCategory(accessGroups))
	assert.True(t, isAccessCategory(accessLeases))
	assert.False(t, isAccessCategory("secret/"))
	assert.False(t, isAccessCategory(""))
}
