package app

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/janosmiko/vau/internal/model"
	"github.com/janosmiko/vau/internal/ui"
	"github.com/janosmiko/vau/internal/vault"
)

// --- Async loaders ---

func (m *Model) loadPolicies() tea.Cmd {
	return func() tea.Msg {
		policies, err := m.client.ListPolicies()
		return policyListMsg{policies: policies, err: err}
	}
}

func (m *Model) loadPolicy(name string) tea.Cmd {
	return func() tea.Msg {
		content, err := m.client.GetPolicy(name)
		return policyContentMsg{name: name, content: content, err: err}
	}
}

func (m *Model) loadPolicyPreview(name string) tea.Cmd {
	return func() tea.Msg {
		content, _ := m.client.GetPolicy(name)
		return policyPreviewMsg{name: name, content: content}
	}
}

func (m *Model) loadAuthMethods() tea.Cmd {
	return func() tea.Msg {
		methods, err := m.client.ListAuthMethods()
		return authMethodsMsg{methods: methods, err: err}
	}
}

func (m *Model) loadRoles(authPath string) tea.Cmd {
	return func() tea.Msg {
		roles, err := m.client.ListRoles(authPath)
		return roleListMsg{authPath: authPath, roles: roles, err: err}
	}
}

func (m *Model) loadRolePreview(authPath string) tea.Cmd {
	return func() tea.Msg {
		roles, _ := m.client.ListRoles(authPath)
		return rolePreviewMsg{authPath: authPath, roles: roles}
	}
}

func (m *Model) loadRole(authPath, name string) tea.Cmd {
	return func() tea.Msg {
		data, err := m.client.GetRole(authPath, name)
		return roleDataMsg{authPath: authPath, name: name, data: data, err: err}
	}
}

func (m *Model) loadRoleDataPreview(authPath, name string) tea.Cmd {
	return func() tea.Msg {
		data, _ := m.client.GetRole(authPath, name)
		return roleDataPreviewMsg{name: name, data: data}
	}
}

// --- Message handlers (called from Update) ---

func (m *Model) handlePolicyListResult(msg policyListMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.errMsg = msg.err.Error()
		return m, nil
	}
	m.policies = msg.policies
	m.policyCursor = 0
	m.policyPreview = ""
	m.errMsg = ""
	// Load preview for first policy.
	if len(m.policies) > 0 {
		return m, m.loadPolicyPreview(m.policies[0])
	}
	return m, nil
}

func (m *Model) handlePolicyContentResult(msg policyContentMsg) tea.Model {
	if msg.err != nil {
		m.errMsg = msg.err.Error()
		return m
	}
	m.policyName = msg.name
	m.policyContent = msg.content
	m.policyScroll = 0
	m.mode = model.ModePolicyView
	m.errMsg = ""
	return m
}

func (m *Model) handleAuthMethodsResult(msg authMethodsMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.errMsg = msg.err.Error()
		return m, nil
	}
	m.authMethods = msg.methods
	m.authCursor = 0
	m.rolePreview = nil
	m.errMsg = ""
	// Load role preview for first auth method.
	if len(m.authMethods) > 0 {
		return m, m.loadRolePreview(m.authMethods[0].Name)
	}
	return m, nil
}

func (m *Model) handleRoleListResult(msg roleListMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.errMsg = msg.err.Error()
		return m, nil
	}
	m.roleAuthPath = msg.authPath
	m.roles = msg.roles
	m.roleCursor = 0
	m.roleDataPreview = nil
	m.mode = model.ModeRoleList
	m.errMsg = ""
	// Load data preview for first role.
	if len(m.roles) > 0 {
		return m, m.loadRoleDataPreview(msg.authPath, m.roles[0].Name)
	}
	return m, nil
}

func (m *Model) handleRoleDataResult(msg roleDataMsg) tea.Model {
	if msg.err != nil {
		m.errMsg = msg.err.Error()
		return m
	}
	m.roleName = msg.name
	m.roleData = msg.data
	m.roleScroll = 0
	m.mode = model.ModeRoleView
	m.errMsg = ""
	return m
}

// --- Key handlers ---

func (m *Model) handlePolicyListKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	if result, handled := m.handleCommonKeys(key); handled {
		return result, nil
	}

	switch {
	case matchKey(key, m.keys.Quit):
		return m, tea.Quit

	case matchKey(key, m.keys.Up):
		if m.policyCursor < len(m.policies)-1 {
			m.policyCursor++
			return m, m.loadPolicyPreview(m.policies[m.policyCursor])
		}

	case matchKey(key, m.keys.Down):
		if m.policyCursor > 0 {
			m.policyCursor--
			return m, m.loadPolicyPreview(m.policies[m.policyCursor])
		}

	case matchKey(key, m.keys.Top):
		m.policyCursor = 0
		if len(m.policies) > 0 {
			return m, m.loadPolicyPreview(m.policies[m.policyCursor])
		}

	case matchKey(key, m.keys.Bottom):
		m.policyCursor = len(m.policies) - 1
		if len(m.policies) > 0 {
			return m, m.loadPolicyPreview(m.policies[m.policyCursor])
		}

	case matchKey(key, m.keys.HalfDown):
		m.policyCursor = min(m.policyCursor+10, len(m.policies)-1)
		if len(m.policies) > 0 {
			return m, m.loadPolicyPreview(m.policies[m.policyCursor])
		}

	case matchKey(key, m.keys.HalfUp):
		m.policyCursor = max(m.policyCursor-10, 0)
		if len(m.policies) > 0 {
			return m, m.loadPolicyPreview(m.policies[m.policyCursor])
		}

	case matchKey(key, m.keys.Right) || matchKey(key, m.keys.Open):
		if len(m.policies) > 0 && m.policyCursor < len(m.policies) {
			return m, m.loadPolicy(m.policies[m.policyCursor])
		}

	case matchKey(key, m.keys.Edit):
		// Edit selected policy in editor directly from list
		if len(m.policies) > 0 && m.policyCursor < len(m.policies) {
			name := m.policies[m.policyCursor]
			return m, m.editPolicyFromList(name)
		}

	case matchKey(key, m.keys.NewSecret) || matchKey(key, m.keys.NewSecretEditor):
		// Create new policy — prompt for name, then open editor
		return m, m.startInput(model.InputNewPolicy, "New policy name:", "")

	case matchKey(key, m.keys.Delete):
		// Delete selected policy
		if len(m.policies) > 0 && m.policyCursor < len(m.policies) {
			name := m.policies[m.policyCursor]
			m.confirmMsg = fmt.Sprintf("Delete policy %q?", name)
			m.prevConfirmMode = model.ModePolicyList
			m.confirmAction = func() tea.Cmd {
				return m.deletePolicy(name)
			}
			m.enterConfirmMode()
			return m, nil
		}

	case matchKey(key, m.keys.Left) || key == "esc":
		m.mode = model.ModeExplorer
		m.atMountLevel = true
		return m, nil
	}
	return m, nil
}

func (m *Model) handlePolicyViewKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	switch {
	case matchKey(key, m.keys.Up):
		m.policyScroll++
	case matchKey(key, m.keys.Down):
		m.policyScroll = max(m.policyScroll-1, 0)
	case matchKey(key, m.keys.HalfDown):
		m.policyScroll += 10
	case matchKey(key, m.keys.HalfUp):
		m.policyScroll = max(m.policyScroll-10, 0)
	case matchKey(key, m.keys.FullDown):
		m.policyScroll += 20
	case matchKey(key, m.keys.FullUp):
		m.policyScroll = max(m.policyScroll-20, 0)
	case matchKey(key, m.keys.Top):
		m.policyScroll = 0
	case matchKey(key, m.keys.Bottom):
		lines := strings.Count(m.policyContent, "\n")
		m.policyScroll = max(lines-1, 0)
	case matchKey(key, m.keys.Edit):
		// Edit policy in external editor
		if m.policyName != "" {
			return m, m.editPolicyInEditor(m.policyName, m.policyContent)
		}
	case matchKey(key, m.keys.Left) || key == "esc" || matchKey(key, m.keys.Quit):
		m.mode = model.ModePolicyList
		m.policyContent = ""
		m.policyName = ""
		m.policyScroll = 0
	}
	return m, nil
}

func (m *Model) handleAuthMethodsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	if result, handled := m.handleCommonKeys(key); handled {
		return result, nil
	}

	switch {
	case matchKey(key, m.keys.Quit):
		return m, tea.Quit

	case matchKey(key, m.keys.Up):
		if m.authCursor < len(m.authMethods)-1 {
			m.authCursor++
			return m, m.loadRolePreview(m.authMethods[m.authCursor].Name)
		}

	case matchKey(key, m.keys.Down):
		if m.authCursor > 0 {
			m.authCursor--
			return m, m.loadRolePreview(m.authMethods[m.authCursor].Name)
		}

	case matchKey(key, m.keys.HalfDown):
		n := len(m.authMethods)
		if n > 0 {
			m.authCursor = min(m.authCursor+10, n-1)
			return m, m.loadRolePreview(m.authMethods[m.authCursor].Name)
		}
	case matchKey(key, m.keys.HalfUp):
		if len(m.authMethods) > 0 {
			m.authCursor = max(m.authCursor-10, 0)
			return m, m.loadRolePreview(m.authMethods[m.authCursor].Name)
		}
	case matchKey(key, m.keys.Top):
		if len(m.authMethods) > 0 {
			m.authCursor = 0
			return m, m.loadRolePreview(m.authMethods[m.authCursor].Name)
		}
	case matchKey(key, m.keys.Bottom):
		if n := len(m.authMethods); n > 0 {
			m.authCursor = n - 1
			return m, m.loadRolePreview(m.authMethods[m.authCursor].Name)
		}

	case matchKey(key, m.keys.Right) || matchKey(key, m.keys.Open):
		if len(m.authMethods) > 0 && m.authCursor < len(m.authMethods) {
			return m, m.loadRoles(m.authMethods[m.authCursor].Name)
		}

	case matchKey(key, m.keys.Left) || key == "esc":
		m.mode = model.ModeExplorer
		m.atMountLevel = true
		return m, nil
	}
	return m, nil
}

// handleRoleListNavKey handles cursor-movement keys in the role list.
// Returns handled=false when key is not a navigation key.
func (m *Model) handleRoleListNavKey(key string) (tea.Cmd, bool) {
	switch {
	case matchKey(key, m.keys.Up):
		if m.roleCursor < len(m.roles)-1 {
			m.roleCursor++
			m.roleDataPreview = nil
			return m.loadRoleDataPreview(m.roleAuthPath, m.roles[m.roleCursor].Name), true
		}
		return nil, true

	case matchKey(key, m.keys.Down):
		if m.roleCursor > 0 {
			m.roleCursor--
			m.roleDataPreview = nil
			return m.loadRoleDataPreview(m.roleAuthPath, m.roles[m.roleCursor].Name), true
		}
		return nil, true

	case matchKey(key, m.keys.HalfDown):
		if n := len(m.roles); n > 0 {
			m.roleCursor = min(m.roleCursor+10, n-1)
			m.roleDataPreview = nil
			return m.loadRoleDataPreview(m.roleAuthPath, m.roles[m.roleCursor].Name), true
		}
		return nil, true
	case matchKey(key, m.keys.HalfUp):
		if len(m.roles) > 0 {
			m.roleCursor = max(m.roleCursor-10, 0)
			m.roleDataPreview = nil
			return m.loadRoleDataPreview(m.roleAuthPath, m.roles[m.roleCursor].Name), true
		}
		return nil, true
	case matchKey(key, m.keys.Top):
		if len(m.roles) > 0 {
			m.roleCursor = 0
			m.roleDataPreview = nil
			return m.loadRoleDataPreview(m.roleAuthPath, m.roles[m.roleCursor].Name), true
		}
		return nil, true
	case matchKey(key, m.keys.Bottom):
		if n := len(m.roles); n > 0 {
			m.roleCursor = n - 1
			m.roleDataPreview = nil
			return m.loadRoleDataPreview(m.roleAuthPath, m.roles[m.roleCursor].Name), true
		}
		return nil, true
	}
	return nil, false
}

func (m *Model) handleRoleListKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	if result, handled := m.handleCommonKeys(key); handled {
		return result, nil
	}

	if matchKey(key, m.keys.Quit) {
		return m, tea.Quit
	}

	if cmd, handled := m.handleRoleListNavKey(key); handled {
		return m, cmd
	}

	switch {
	case matchKey(key, m.keys.Right) || matchKey(key, m.keys.Open):
		if len(m.roles) > 0 && m.roleCursor < len(m.roles) {
			return m, m.loadRole(m.roleAuthPath, m.roles[m.roleCursor].Name)
		}

	case matchKey(key, m.keys.Edit):
		// Edit selected role in editor directly from list
		if len(m.roles) > 0 && m.roleCursor < len(m.roles) {
			name := m.roles[m.roleCursor].Name
			return m, m.editRoleFromList(m.roleAuthPath, name)
		}

	case matchKey(key, m.keys.NewSecret):
		// Create new role — prompt for name, then open editor
		return m, m.startInput(model.InputNewRole, "New role name:", "")

	case matchKey(key, m.keys.Delete):
		// Delete selected role
		if len(m.roles) > 0 && m.roleCursor < len(m.roles) {
			name := m.roles[m.roleCursor].Name
			authPath := m.roleAuthPath
			m.confirmMsg = fmt.Sprintf("Delete role %q from %s?", name, authPath)
			m.prevConfirmMode = model.ModeRoleList
			m.confirmAction = func() tea.Cmd {
				return m.deleteRole(authPath, name)
			}
			m.enterConfirmMode()
			return m, nil
		}

	case key == "c":
		// Quick-create token from selected role (token auth only)
		if vault.AuthMethodType(m.roleAuthPath) == "token" && len(m.roles) > 0 && m.roleCursor < len(m.roles) {
			roleName := m.roles[m.roleCursor].Name
			return m, m.createTokenFromRole(roleName)
		}

	case matchKey(key, m.keys.Left) || key == "esc":
		m.mode = model.ModeAuthMethods
		m.roles = nil
		m.roleCursor = 0
	}
	return m, nil
}

func (m *Model) handleRoleViewKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	switch {
	case matchKey(key, m.keys.Up):
		m.roleScroll++
	case matchKey(key, m.keys.Down):
		m.roleScroll = max(m.roleScroll-1, 0)
	case matchKey(key, m.keys.HalfDown):
		m.roleScroll += 10
	case matchKey(key, m.keys.HalfUp):
		m.roleScroll = max(m.roleScroll-10, 0)
	case matchKey(key, m.keys.FullDown):
		m.roleScroll += 20
	case matchKey(key, m.keys.FullUp):
		m.roleScroll = max(m.roleScroll-20, 0)
	case matchKey(key, m.keys.Edit):
		// Edit role in external editor
		if m.roleName != "" && m.roleData != nil {
			return m, m.editRoleInEditor(m.roleAuthPath, m.roleName, m.roleData)
		}
	case matchKey(key, m.keys.Left) || key == "esc" || matchKey(key, m.keys.Quit):
		m.mode = model.ModeRoleList
		m.roleData = nil
		m.roleName = ""
		m.roleScroll = 0
	}
	return m, nil
}

// --- View rendering helpers ---

// tabLabel returns a display label for a tab based on its saved state.
func tabLabel(t TabState) string {
	switch t.viewMode {
	case model.ModePolicyList, model.ModePolicyView:
		return "Policies"
	case model.ModeAuthMethods:
		return "Auth Methods"
	case model.ModeRoleList, model.ModeRoleView:
		return "Roles"
	case model.ModeEntityList, model.ModeEntityView:
		return "Entities"
	case model.ModeGroupList, model.ModeGroupView:
		return "Groups"
	case model.ModeTokenList, model.ModeTokenView, model.ModeTokenCreated:
		return "Tokens"
	}
	if len(t.path) > 0 {
		return t.path[len(t.path)-1]
	}
	return t.mount + "/"
}

func (m *Model) workspaceTabLabels() []string {
	if len(m.tabs) <= 1 {
		return nil
	}
	m.saveCurrentTab()
	labels := make([]string, len(m.tabs))
	for i, t := range m.tabs {
		labels[i] = tabLabel(t)
	}
	return labels
}

func (m *Model) renderPolicyListView() string {
	explorerHeight := m.height - 2
	parentEntries := m.mountEntries()
	// Find the index of [Policies] in the parent list
	parentIdx := -1
	for i, e := range parentEntries {
		if e.Name == accessPolicies {
			parentIdx = i
			break
		}
	}
	tabLabels := m.workspaceTabLabels()
	return ui.RenderPolicyList(
		parentEntries, parentIdx,
		m.policies, m.policyCursor, m.policyPreview,
		tabLabels, m.activeTab,
		m.version, m.width, explorerHeight,
	)
}

func (m *Model) renderAuthMethodListView() string {
	explorerHeight := m.height - 2
	parentEntries := m.mountEntries()
	parentIdx := -1
	for i, e := range parentEntries {
		if e.Name == accessAuthMethods {
			parentIdx = i
			break
		}
	}
	tabLabels := m.workspaceTabLabels()
	return ui.RenderAuthMethodList(
		parentEntries, parentIdx,
		m.authMethods, m.authCursor, m.rolePreview,
		tabLabels, m.activeTab,
		m.version, m.width, explorerHeight,
	)
}

func (m *Model) renderRoleListView() string {
	explorerHeight := m.height - 2
	// Left pane: auth methods as parent
	parentEntries := m.authMethods
	parentIdx := -1
	for i, e := range parentEntries {
		if e.Name == m.roleAuthPath {
			parentIdx = i
			break
		}
	}
	tabLabels := m.workspaceTabLabels()
	return ui.RenderRoleList(
		parentEntries, parentIdx,
		m.roles, m.roleCursor, m.roleAuthPath, m.roleDataPreview,
		tabLabels, m.activeTab,
		m.version, m.width, explorerHeight,
	)
}
