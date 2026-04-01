package app

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/janosmiko/vau/internal/model"
	"github.com/janosmiko/vau/internal/ui"
	"github.com/janosmiko/vau/internal/vault"
)

// --- Workspace message types ---

type policyListMsg struct {
	policies []string
	err      error
}

type policyContentMsg struct {
	name    string
	content string
	err     error
}

type authMethodsMsg struct {
	methods []model.Entry
	err     error
}

type roleListMsg struct {
	authPath string
	roles    []model.Entry
	err      error
}

type roleDataMsg struct {
	authPath string
	name     string
	data     map[string]any
	err      error
}

// Preview messages (for right-pane preview loading).
type policyPreviewMsg struct {
	name    string
	content string
}

type rolePreviewMsg struct {
	authPath string
	roles    []model.Entry
}

type roleDataPreviewMsg struct {
	name string
	data map[string]any
}

type entityListMsg struct {
	entities []model.Entry
	err      error
}

type entityDataMsg struct {
	name string
	data map[string]any
	err  error
}

type entityDataPreviewMsg struct {
	name string
	data map[string]any
}

type groupListMsg struct {
	groups []model.Entry
	err    error
}

type groupDataMsg struct {
	name string
	data map[string]any
	err  error
}

type groupDataPreviewMsg struct {
	name string
	data map[string]any
}

// Token messages
type tokenListMsg struct {
	accessors []model.Entry
	err       error
}

type tokenDataMsg struct {
	accessor string
	data     map[string]any
	err      error
}

type tokenDataPreviewMsg struct {
	accessor string
	data     map[string]any
}

type tokenCreatedMsg struct {
	data map[string]any
	err  error
}

type tokenRevokedMsg struct {
	accessor string
	err      error
}

// Intermediate messages for edit-from-list flow
type policyContentForEditMsg struct {
	name    string
	content string
}

type roleDataForEditMsg struct {
	authPath string
	name     string
	data     map[string]any
}

// CRUD result messages
type policyEditorResultMsg struct {
	name    string
	content string
	isNew   bool
	err     error
}

type policyDeletedMsg struct {
	name string
	err  error
}

type roleEditorResultMsg struct {
	authPath string
	name     string
	data     map[string]any
	isNew    bool
	err      error
}

type roleDeletedMsg struct {
	authPath string
	name     string
	err      error
}

// handleCommonKeys handles keys shared across all workspace list views
// (tabs, help, quit). Returns true if the key was handled.
func (m *Model) handleCommonKeys(key string) (tea.Model, bool) {
	switch {
	case matchKey(key, m.keys.NewTab):
		if len(m.tabs) < 9 {
			m.saveCurrentTab()
			newTab := m.tabs[m.activeTab]
			newTab.selected = make(map[int]bool)
			for k, v := range m.tabs[m.activeTab].selected {
				newTab.selected[k] = v
			}
			newTab.cursorMemory = make(map[string]int)
			for k, v := range m.tabs[m.activeTab].cursorMemory {
				newTab.cursorMemory[k] = v
			}
			m.tabs = append(m.tabs, newTab)
			m.activeTab = len(m.tabs) - 1
			m.loadTab(m.activeTab)
			m.status = fmt.Sprintf("Tab %d created", m.activeTab+1)
		}
		return m, true
	case matchKey(key, m.keys.NextTab):
		if len(m.tabs) > 1 {
			m.saveCurrentTab()
			m.loadTab((m.activeTab + 1) % len(m.tabs))
		}
		return m, true
	case matchKey(key, m.keys.PrevTab):
		if len(m.tabs) > 1 {
			m.saveCurrentTab()
			idx := m.activeTab - 1
			if idx < 0 {
				idx = len(m.tabs) - 1
			}
			m.loadTab(idx)
		}
		return m, true
	case matchKey(key, m.keys.CloseTab):
		if len(m.tabs) > 1 {
			m.tabs = append(m.tabs[:m.activeTab], m.tabs[m.activeTab+1:]...)
			if m.activeTab >= len(m.tabs) {
				m.activeTab = len(m.tabs) - 1
			}
			m.loadTab(m.activeTab)
		}
		return m, true
	case matchKey(key, m.keys.Help):
		m.mode = model.ModeHelp
		return m, true
	}
	return m, false
}

// --- Workspace switching ---

// enterAccessCategory switches into an access category view from the root level.
func (m *Model) enterAccessCategory(name string) (tea.Model, tea.Cmd) {
	switch name {
	case accessPolicies:
		m.mode = model.ModePolicyList
		return m, m.loadPolicies()
	case accessAuthMethods:
		m.mode = model.ModeAuthMethods
		return m, m.loadAuthMethods()
	case accessEntities:
		m.mode = model.ModeEntityList
		return m, m.loadEntities()
	case accessGroups:
		m.mode = model.ModeGroupList
		return m, m.loadGroups()
	case accessTokens:
		m.mode = model.ModeTokenList
		return m, m.loadTokenAccessors()
	case accessLeases:
		m.status = "Leases: not yet implemented"
		return m, nil
	}
	return m, nil
}

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

func (m *Model) handlePolicyContentResult(msg policyContentMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.errMsg = msg.err.Error()
		return m, nil
	}
	m.policyName = msg.name
	m.policyContent = msg.content
	m.policyScroll = 0
	m.mode = model.ModePolicyView
	m.errMsg = ""
	return m, nil
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

func (m *Model) handleRoleDataResult(msg roleDataMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.errMsg = msg.err.Error()
		return m, nil
	}
	m.roleName = msg.name
	m.roleData = msg.data
	m.roleScroll = 0
	m.mode = model.ModeRoleView
	m.errMsg = ""
	return m, nil
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

func (m *Model) handleRoleListKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	if result, handled := m.handleCommonKeys(key); handled {
		return result, nil
	}

	switch {
	case matchKey(key, m.keys.Quit):
		return m, tea.Quit

	case matchKey(key, m.keys.Up):
		if m.roleCursor < len(m.roles)-1 {
			m.roleCursor++
			m.roleDataPreview = nil
			return m, m.loadRoleDataPreview(m.roleAuthPath, m.roles[m.roleCursor].Name)
		}

	case matchKey(key, m.keys.Down):
		if m.roleCursor > 0 {
			m.roleCursor--
			m.roleDataPreview = nil
			return m, m.loadRoleDataPreview(m.roleAuthPath, m.roles[m.roleCursor].Name)
		}

	case matchKey(key, m.keys.HalfDown):
		if n := len(m.roles); n > 0 {
			m.roleCursor = min(m.roleCursor+10, n-1)
			m.roleDataPreview = nil
			return m, m.loadRoleDataPreview(m.roleAuthPath, m.roles[m.roleCursor].Name)
		}
	case matchKey(key, m.keys.HalfUp):
		if len(m.roles) > 0 {
			m.roleCursor = max(m.roleCursor-10, 0)
			m.roleDataPreview = nil
			return m, m.loadRoleDataPreview(m.roleAuthPath, m.roles[m.roleCursor].Name)
		}
	case matchKey(key, m.keys.Top):
		if len(m.roles) > 0 {
			m.roleCursor = 0
			m.roleDataPreview = nil
			return m, m.loadRoleDataPreview(m.roleAuthPath, m.roles[m.roleCursor].Name)
		}
	case matchKey(key, m.keys.Bottom):
		if n := len(m.roles); n > 0 {
			m.roleCursor = n - 1
			m.roleDataPreview = nil
			return m, m.loadRoleDataPreview(m.roleAuthPath, m.roles[m.roleCursor].Name)
		}

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

// --- Entity loaders ---

func (m *Model) loadEntities() tea.Cmd {
	return func() tea.Msg {
		entities, err := m.client.ListEntities()
		return entityListMsg{entities: entities, err: err}
	}
}

func (m *Model) loadEntityData(name string) tea.Cmd {
	return func() tea.Msg {
		data, err := m.client.GetEntity(name)
		if err == nil && data != nil {
			m.client.EnrichEntityAliases(data)
		}
		return entityDataMsg{name: name, data: data, err: err}
	}
}

func (m *Model) loadEntityDataPreview(name string) tea.Cmd {
	return func() tea.Msg {
		data, _ := m.client.GetEntity(name)
		if data != nil {
			m.client.EnrichEntityAliases(data)
		}
		return entityDataPreviewMsg{name: name, data: data}
	}
}

// --- Group loaders ---

func (m *Model) loadGroups() tea.Cmd {
	return func() tea.Msg {
		groups, err := m.client.ListGroups()
		return groupListMsg{groups: groups, err: err}
	}
}

func (m *Model) loadGroupData(name string) tea.Cmd {
	return func() tea.Msg {
		data, err := m.client.GetGroup(name)
		return groupDataMsg{name: name, data: data, err: err}
	}
}

func (m *Model) loadGroupDataPreview(name string) tea.Cmd {
	return func() tea.Msg {
		data, _ := m.client.GetGroup(name)
		return groupDataPreviewMsg{name: name, data: data}
	}
}

// --- Entity message handlers ---

func (m *Model) handleEntityListResult(msg entityListMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.errMsg = msg.err.Error()
		return m, nil
	}
	m.entities = msg.entities
	m.entityCursor = 0
	m.entityDataPreview = nil
	m.errMsg = ""
	if len(m.entities) > 0 {
		return m, m.loadEntityDataPreview(m.entities[0].Name)
	}
	return m, nil
}

func (m *Model) handleEntityDataResult(msg entityDataMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.errMsg = msg.err.Error()
		return m, nil
	}
	m.entityName = msg.name
	m.entityData = msg.data
	m.entityScroll = 0
	m.mode = model.ModeEntityView
	m.errMsg = ""
	return m, nil
}

// --- Group message handlers ---

func (m *Model) handleGroupListResult(msg groupListMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.errMsg = msg.err.Error()
		return m, nil
	}
	m.groups = msg.groups
	m.groupCursor = 0
	m.groupDataPreview = nil
	m.errMsg = ""
	if len(m.groups) > 0 {
		return m, m.loadGroupDataPreview(m.groups[0].Name)
	}
	return m, nil
}

func (m *Model) handleGroupDataResult(msg groupDataMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.errMsg = msg.err.Error()
		return m, nil
	}
	m.groupName = msg.name
	m.groupData = msg.data
	m.groupScroll = 0
	m.mode = model.ModeGroupView
	m.errMsg = ""
	return m, nil
}

// --- Entity key handlers ---

func (m *Model) handleEntityListKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	if result, handled := m.handleCommonKeys(key); handled {
		return result, nil
	}

	switch {
	case matchKey(key, m.keys.Quit):
		return m, tea.Quit
	case matchKey(key, m.keys.Up):
		if m.entityCursor < len(m.entities)-1 {
			m.entityCursor++
			m.entityDataPreview = nil // show loading indicator
			return m, m.loadEntityDataPreview(m.entities[m.entityCursor].Name)
		}
	case matchKey(key, m.keys.Down):
		if m.entityCursor > 0 {
			m.entityCursor--
			m.entityDataPreview = nil
			return m, m.loadEntityDataPreview(m.entities[m.entityCursor].Name)
		}
	case matchKey(key, m.keys.HalfDown):
		if n := len(m.entities); n > 0 {
			m.entityCursor = min(m.entityCursor+10, n-1)
			m.entityDataPreview = nil
			return m, m.loadEntityDataPreview(m.entities[m.entityCursor].Name)
		}
	case matchKey(key, m.keys.HalfUp):
		if len(m.entities) > 0 {
			m.entityCursor = max(m.entityCursor-10, 0)
			m.entityDataPreview = nil
			return m, m.loadEntityDataPreview(m.entities[m.entityCursor].Name)
		}
	case matchKey(key, m.keys.Top):
		if len(m.entities) > 0 {
			m.entityCursor = 0
			m.entityDataPreview = nil
			return m, m.loadEntityDataPreview(m.entities[m.entityCursor].Name)
		}
	case matchKey(key, m.keys.Bottom):
		if n := len(m.entities); n > 0 {
			m.entityCursor = n - 1
			m.entityDataPreview = nil
			return m, m.loadEntityDataPreview(m.entities[m.entityCursor].Name)
		}
	case matchKey(key, m.keys.Right) || matchKey(key, m.keys.Open):
		if len(m.entities) > 0 && m.entityCursor < len(m.entities) {
			return m, m.loadEntityData(m.entities[m.entityCursor].Name)
		}
	case matchKey(key, m.keys.Left) || key == "esc":
		m.mode = model.ModeExplorer
		m.atMountLevel = true
		return m, nil
	}
	return m, nil
}

func (m *Model) handleEntityViewKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	switch {
	case matchKey(key, m.keys.Up):
		m.entityScroll++
	case matchKey(key, m.keys.Down):
		m.entityScroll = max(m.entityScroll-1, 0)
	case matchKey(key, m.keys.HalfDown):
		m.entityScroll += 10
	case matchKey(key, m.keys.HalfUp):
		m.entityScroll = max(m.entityScroll-10, 0)
	case matchKey(key, m.keys.FullDown):
		m.entityScroll += 20
	case matchKey(key, m.keys.FullUp):
		m.entityScroll = max(m.entityScroll-20, 0)
	case matchKey(key, m.keys.Left) || key == "esc" || matchKey(key, m.keys.Quit):
		m.mode = model.ModeEntityList
		m.entityData = nil
		m.entityName = ""
		m.entityScroll = 0
	}
	return m, nil
}

// --- Group key handlers ---

func (m *Model) handleGroupListKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	if result, handled := m.handleCommonKeys(key); handled {
		return result, nil
	}

	switch {
	case matchKey(key, m.keys.Quit):
		return m, tea.Quit
	case matchKey(key, m.keys.Up):
		if m.groupCursor < len(m.groups)-1 {
			m.groupCursor++
			m.groupDataPreview = nil
			return m, m.loadGroupDataPreview(m.groups[m.groupCursor].Name)
		}
	case matchKey(key, m.keys.Down):
		if m.groupCursor > 0 {
			m.groupCursor--
			m.groupDataPreview = nil
			return m, m.loadGroupDataPreview(m.groups[m.groupCursor].Name)
		}
	case matchKey(key, m.keys.HalfDown):
		if n := len(m.groups); n > 0 {
			m.groupCursor = min(m.groupCursor+10, n-1)
			m.groupDataPreview = nil
			return m, m.loadGroupDataPreview(m.groups[m.groupCursor].Name)
		}
	case matchKey(key, m.keys.HalfUp):
		if len(m.groups) > 0 {
			m.groupCursor = max(m.groupCursor-10, 0)
			m.groupDataPreview = nil
			return m, m.loadGroupDataPreview(m.groups[m.groupCursor].Name)
		}
	case matchKey(key, m.keys.Top):
		if len(m.groups) > 0 {
			m.groupCursor = 0
			m.groupDataPreview = nil
			return m, m.loadGroupDataPreview(m.groups[m.groupCursor].Name)
		}
	case matchKey(key, m.keys.Bottom):
		if n := len(m.groups); n > 0 {
			m.groupCursor = n - 1
			m.groupDataPreview = nil
			return m, m.loadGroupDataPreview(m.groups[m.groupCursor].Name)
		}
	case matchKey(key, m.keys.Right) || matchKey(key, m.keys.Open):
		if len(m.groups) > 0 && m.groupCursor < len(m.groups) {
			return m, m.loadGroupData(m.groups[m.groupCursor].Name)
		}
	case matchKey(key, m.keys.Left) || key == "esc":
		m.mode = model.ModeExplorer
		m.atMountLevel = true
		return m, nil
	}
	return m, nil
}

func (m *Model) handleGroupViewKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	switch {
	case matchKey(key, m.keys.Up):
		m.groupScroll++
	case matchKey(key, m.keys.Down):
		m.groupScroll = max(m.groupScroll-1, 0)
	case matchKey(key, m.keys.HalfDown):
		m.groupScroll += 10
	case matchKey(key, m.keys.HalfUp):
		m.groupScroll = max(m.groupScroll-10, 0)
	case matchKey(key, m.keys.FullDown):
		m.groupScroll += 20
	case matchKey(key, m.keys.FullUp):
		m.groupScroll = max(m.groupScroll-20, 0)
	case matchKey(key, m.keys.Left) || key == "esc" || matchKey(key, m.keys.Quit):
		m.mode = model.ModeGroupList
		m.groupData = nil
		m.groupName = ""
		m.groupScroll = 0
	}
	return m, nil
}

// --- Entity/Group render helpers ---

func (m *Model) renderEntityListView() string {
	explorerHeight := m.height - 2
	parentEntries := m.mountEntries()
	parentIdx := -1
	for i, e := range parentEntries {
		if e.Name == accessEntities {
			parentIdx = i
			break
		}
	}
	tabLabels := m.workspaceTabLabels()
	return ui.RenderEntityList(
		parentEntries, parentIdx,
		m.entities, m.entityCursor, m.entityDataPreview,
		tabLabels, m.activeTab,
		m.version, m.width, explorerHeight,
	)
}

func (m *Model) renderGroupListView() string {
	explorerHeight := m.height - 2
	parentEntries := m.mountEntries()
	parentIdx := -1
	for i, e := range parentEntries {
		if e.Name == accessGroups {
			parentIdx = i
			break
		}
	}
	tabLabels := m.workspaceTabLabels()
	return ui.RenderGroupList(
		parentEntries, parentIdx,
		m.groups, m.groupCursor, m.groupDataPreview,
		tabLabels, m.activeTab,
		m.version, m.width, explorerHeight,
	)
}

// --- Policy CRUD ---

func (m *Model) createPolicyWithEditor(name string) tea.Cmd {
	template := "# Policy: " + name + "\n# See: https://developer.hashicorp.com/vault/docs/concepts/policies\n\npath \"secret/*\" {\n  capabilities = [\"read\", \"list\"]\n}\n"
	return m.openEditorWithContent(template, ".hcl", func(content string) tea.Msg {
		err := m.client.PutPolicy(name, content)
		return policyEditorResultMsg{name: name, content: content, isNew: true, err: err}
	})
}

// editPolicyFromList fetches the policy content, then opens editor.
func (m *Model) editPolicyFromList(name string) tea.Cmd {
	return func() tea.Msg {
		content, err := m.client.GetPolicy(name)
		if err != nil {
			return errorMsg(fmt.Sprintf("reading policy %q: %v", name, err))
		}
		// Return a message that triggers the editor open
		return policyContentForEditMsg{name: name, content: content}
	}
}

// editRoleFromList fetches the role data, then opens editor.
func (m *Model) editRoleFromList(authPath, name string) tea.Cmd {
	return func() tea.Msg {
		data, err := m.client.GetRole(authPath, name)
		if err != nil {
			return errorMsg(fmt.Sprintf("reading role %q: %v", name, err))
		}
		return roleDataForEditMsg{authPath: authPath, name: name, data: data}
	}
}

func (m *Model) editPolicyInEditor(name, content string) tea.Cmd {
	return m.openEditorWithContent(content, ".hcl", func(newContent string) tea.Msg {
		if newContent == content {
			return statusMsg("Policy unchanged")
		}
		err := m.client.PutPolicy(name, newContent)
		return policyEditorResultMsg{name: name, content: newContent, isNew: false, err: err}
	})
}

func (m *Model) deletePolicy(name string) tea.Cmd {
	return func() tea.Msg {
		err := m.client.DeletePolicy(name)
		return policyDeletedMsg{name: name, err: err}
	}
}

// --- Role CRUD ---

// roleTemplate returns a prefilled JSON template based on the auth method type.
func roleTemplate(authPath string) string {
	authType := vault.AuthMethodType(authPath)
	switch authType {
	case "token":
		return `{
  "allowed_policies": ["default"],
  "disallowed_policies": [],
  "orphan": false,
  "renewable": true,
  "token_period": "0",
  "token_explicit_max_ttl": "0"
}
`
	case "approle":
		return `{
  "token_policies": ["default"],
  "token_ttl": "1h",
  "token_max_ttl": "4h",
  "secret_id_ttl": "0",
  "secret_id_num_uses": 0,
  "token_num_uses": 0,
  "bind_secret_id": true
}
`
	case "userpass":
		return `{
  "password": "",
  "token_policies": ["default"],
  "token_ttl": "1h",
  "token_max_ttl": "4h"
}
`
	case "ldap":
		return `{
  "policies": ["default"]
}
`
	case "jwt", "oidc":
		return `{
  "role_type": "` + authType + `",
  "bound_audiences": [],
  "user_claim": "sub",
  "token_policies": ["default"],
  "token_ttl": "1h",
  "token_max_ttl": "4h"
}
`
	case "cert":
		return `{
  "certificate": "",
  "token_policies": ["default"],
  "token_ttl": "1h"
}
`
	case "kubernetes":
		return `{
  "bound_service_account_names": ["*"],
  "bound_service_account_namespaces": ["default"],
  "token_policies": ["default"],
  "token_ttl": "1h",
  "token_max_ttl": "4h"
}
`
	default:
		return "{\n  \n}\n"
	}
}

func (m *Model) createRoleWithEditor(authPath, name string) tea.Cmd {
	template := roleTemplate(authPath)
	return m.openEditorWithContent(template, ".json", func(content string) tea.Msg {
		var data map[string]any
		if err := json.Unmarshal([]byte(content), &data); err != nil {
			return roleEditorResultMsg{authPath: authPath, name: name, err: fmt.Errorf("invalid JSON: %w", err)}
		}
		err := m.client.WriteRole(authPath, name, data)
		return roleEditorResultMsg{authPath: authPath, name: name, data: data, isNew: true, err: err}
	})
}

func (m *Model) editRoleInEditor(authPath, name string, data map[string]any) tea.Cmd {
	content, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return func() tea.Msg {
			return roleEditorResultMsg{authPath: authPath, name: name, err: fmt.Errorf("marshal: %w", err)}
		}
	}
	original := string(content)
	return m.openEditorWithContent(original, ".json", func(newContent string) tea.Msg {
		if newContent == original {
			return statusMsg("Role unchanged")
		}
		var newData map[string]any
		if err := json.Unmarshal([]byte(newContent), &newData); err != nil {
			return roleEditorResultMsg{authPath: authPath, name: name, err: fmt.Errorf("invalid JSON: %w", err)}
		}
		err := m.client.WriteRole(authPath, name, newData)
		return roleEditorResultMsg{authPath: authPath, name: name, data: newData, isNew: false, err: err}
	})
}

func (m *Model) deleteRole(authPath, name string) tea.Cmd {
	return func() tea.Msg {
		err := m.client.DeleteRole(authPath, name)
		return roleDeletedMsg{authPath: authPath, name: name, err: err}
	}
}

// --- Shared editor helper ---

// openEditorWithContent creates a temp file with the given content and extension,
// opens it in the user's editor, and calls the callback with the edited content.
func (m *Model) openEditorWithContent(content, extension string, callback func(string) tea.Msg) tea.Cmd {
	editor, err := m.editorCommand()
	if err != nil {
		return func() tea.Msg { return errorMsg("editor: " + err.Error()) }
	}

	tmpDir := os.TempDir()
	if xdg := os.Getenv("XDG_RUNTIME_DIR"); xdg != "" {
		tmpDir = xdg
	}

	tmpFile, err := os.CreateTemp(tmpDir, ".vau-edit-*"+extension)
	if err != nil {
		return func() tea.Msg { return errorMsg("temp file: " + err.Error()) }
	}

	if _, err := tmpFile.WriteString(content); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return func() tea.Msg { return errorMsg("write temp: " + err.Error()) }
	}
	tmpFile.Close()

	c := exec.Command(editor, tmpFile.Name()) //nolint:gosec // editor is user-configured
	return tea.ExecProcess(c, func(err error) tea.Msg {
		defer os.Remove(tmpFile.Name())
		if err != nil {
			return errorMsg("editor failed: " + err.Error())
		}
		edited, err := os.ReadFile(tmpFile.Name())
		if err != nil {
			return errorMsg("read edited: " + err.Error())
		}
		return callback(string(edited))
	})
}

// --- CRUD message handlers ---

func (m *Model) handlePolicyEditorResult(msg policyEditorResultMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.errMsg = msg.err.Error()
		m.mode = model.ModePolicyList
		return m, nil
	}
	if msg.isNew {
		m.status = fmt.Sprintf("Created policy: %s", msg.name)
	} else {
		m.status = fmt.Sprintf("Updated policy: %s", msg.name)
		m.policyContent = msg.content
		m.policyScroll = 0
	}
	m.mode = model.ModePolicyList
	return m, m.loadPolicies()
}

func (m *Model) handlePolicyDeletedResult(msg policyDeletedMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.errMsg = msg.err.Error()
		return m, nil
	}
	m.status = fmt.Sprintf("Deleted policy: %s", msg.name)
	m.mode = model.ModePolicyList
	return m, m.loadPolicies()
}

func (m *Model) handleRoleEditorResult(msg roleEditorResultMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.errMsg = msg.err.Error()
		m.mode = model.ModeRoleList
		return m, nil
	}
	if msg.isNew {
		m.status = fmt.Sprintf("Created role: %s", msg.name)
	} else {
		m.status = fmt.Sprintf("Updated role: %s", msg.name)
		m.roleData = msg.data
		m.roleScroll = 0
	}
	m.mode = model.ModeRoleList
	return m, m.loadRoles(msg.authPath)
}

func (m *Model) handleRoleDeletedResult(msg roleDeletedMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.errMsg = msg.err.Error()
		return m, nil
	}
	m.status = fmt.Sprintf("Deleted role: %s", msg.name)
	m.mode = model.ModeRoleList
	return m, m.loadRoles(msg.authPath)
}

// --- Token loaders ---

func (m *Model) loadTokenAccessors() tea.Cmd {
	return func() tea.Msg {
		accessors, err := m.client.ListTokenAccessors()
		return tokenListMsg{accessors: accessors, err: err}
	}
}

func (m *Model) loadTokenData(accessor string) tea.Cmd {
	return func() tea.Msg {
		data, err := m.client.LookupAccessor(accessor)
		return tokenDataMsg{accessor: accessor, data: data, err: err}
	}
}

func (m *Model) loadTokenDataPreview(accessor string) tea.Cmd {
	return func() tea.Msg {
		data, _ := m.client.LookupAccessor(accessor)
		return tokenDataPreviewMsg{accessor: accessor, data: data}
	}
}

func (m *Model) createTokenFromRole(role string) tea.Cmd {
	return func() tea.Msg {
		data, err := m.client.CreateTokenWithRole(role, nil)
		return tokenCreatedMsg{data: data, err: err}
	}
}

func (m *Model) revokeToken(accessor string) tea.Cmd {
	return func() tea.Msg {
		err := m.client.RevokeAccessor(accessor)
		return tokenRevokedMsg{accessor: accessor, err: err}
	}
}

// --- Token message handlers ---

func (m *Model) handleTokenListResult(msg tokenListMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.errMsg = msg.err.Error()
		return m, nil
	}
	m.tokenAccessors = msg.accessors
	m.tokenCursor = 0
	m.tokenDataPreview = nil
	m.errMsg = ""
	if len(m.tokenAccessors) > 0 {
		return m, m.loadTokenDataPreview(m.tokenAccessors[0].Name)
	}
	return m, nil
}

func (m *Model) handleTokenDataResult(msg tokenDataMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.errMsg = msg.err.Error()
		return m, nil
	}
	m.tokenData = msg.data
	m.tokenScroll = 0
	m.mode = model.ModeTokenView
	m.errMsg = ""
	return m, nil
}

func (m *Model) handleTokenCreatedResult(msg tokenCreatedMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.errMsg = msg.err.Error()
		return m, nil
	}
	m.tokenCreatedData = msg.data
	m.mode = model.ModeTokenCreated
	m.errMsg = ""
	return m, nil
}

func (m *Model) handleTokenRevokedResult(msg tokenRevokedMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.errMsg = msg.err.Error()
		return m, nil
	}
	m.status = fmt.Sprintf("Revoked token: %s", msg.accessor)
	return m, m.loadTokenAccessors()
}

// --- Token key handlers ---

func (m *Model) handleTokenListKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	if result, handled := m.handleCommonKeys(key); handled {
		return result, nil
	}

	switch {
	case matchKey(key, m.keys.Quit):
		return m, tea.Quit

	case matchKey(key, m.keys.Up):
		if m.tokenCursor < len(m.tokenAccessors)-1 {
			m.tokenCursor++
			m.tokenDataPreview = nil
			return m, m.loadTokenDataPreview(m.tokenAccessors[m.tokenCursor].Name)
		}

	case matchKey(key, m.keys.Down):
		if m.tokenCursor > 0 {
			m.tokenCursor--
			m.tokenDataPreview = nil
			return m, m.loadTokenDataPreview(m.tokenAccessors[m.tokenCursor].Name)
		}

	case matchKey(key, m.keys.HalfDown):
		if n := len(m.tokenAccessors); n > 0 {
			m.tokenCursor = min(m.tokenCursor+10, n-1)
			m.tokenDataPreview = nil
			return m, m.loadTokenDataPreview(m.tokenAccessors[m.tokenCursor].Name)
		}
	case matchKey(key, m.keys.HalfUp):
		if len(m.tokenAccessors) > 0 {
			m.tokenCursor = max(m.tokenCursor-10, 0)
			m.tokenDataPreview = nil
			return m, m.loadTokenDataPreview(m.tokenAccessors[m.tokenCursor].Name)
		}
	case matchKey(key, m.keys.Top):
		if len(m.tokenAccessors) > 0 {
			m.tokenCursor = 0
			m.tokenDataPreview = nil
			return m, m.loadTokenDataPreview(m.tokenAccessors[m.tokenCursor].Name)
		}
	case matchKey(key, m.keys.Bottom):
		if n := len(m.tokenAccessors); n > 0 {
			m.tokenCursor = n - 1
			m.tokenDataPreview = nil
			return m, m.loadTokenDataPreview(m.tokenAccessors[m.tokenCursor].Name)
		}

	case matchKey(key, m.keys.Right) || matchKey(key, m.keys.Open):
		if len(m.tokenAccessors) > 0 && m.tokenCursor < len(m.tokenAccessors) {
			return m, m.loadTokenData(m.tokenAccessors[m.tokenCursor].Name)
		}

	case matchKey(key, m.keys.NewSecret):
		// Create custom token via editor
		template := `{
  "policies": ["default"],
  "ttl": "1h",
  "renewable": true,
  "num_uses": 0
}
`
		return m, m.openEditorWithContent(template, ".json", func(content string) tea.Msg {
			var params map[string]any
			if err := json.Unmarshal([]byte(content), &params); err != nil {
				return errorMsg("invalid JSON: " + err.Error())
			}
			data, err := m.client.CreateToken(params)
			return tokenCreatedMsg{data: data, err: err}
		})

	case matchKey(key, m.keys.Delete):
		if len(m.tokenAccessors) > 0 && m.tokenCursor < len(m.tokenAccessors) {
			accessor := m.tokenAccessors[m.tokenCursor].Name
			m.confirmMsg = fmt.Sprintf("Revoke token %s?", accessor[:min(len(accessor), 12)]+"...")
			m.prevConfirmMode = model.ModeTokenList
			m.confirmAction = func() tea.Cmd {
				return m.revokeToken(accessor)
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

func (m *Model) handleTokenViewKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	switch {
	case matchKey(key, m.keys.Up):
		m.tokenScroll++
	case matchKey(key, m.keys.Down):
		m.tokenScroll = max(m.tokenScroll-1, 0)
	case matchKey(key, m.keys.HalfDown):
		m.tokenScroll += 10
	case matchKey(key, m.keys.HalfUp):
		m.tokenScroll = max(m.tokenScroll-10, 0)
	case matchKey(key, m.keys.FullDown):
		m.tokenScroll += 20
	case matchKey(key, m.keys.FullUp):
		m.tokenScroll = max(m.tokenScroll-20, 0)
	case matchKey(key, m.keys.Left) || key == "esc" || matchKey(key, m.keys.Quit):
		m.mode = model.ModeTokenList
		m.tokenData = nil
		m.tokenScroll = 0
	}
	return m, nil
}

// --- Token render helper ---

func (m *Model) renderTokenListView() string {
	explorerHeight := m.height - 2
	parentEntries := m.mountEntries()
	parentIdx := -1
	for i, e := range parentEntries {
		if e.Name == accessTokens {
			parentIdx = i
			break
		}
	}
	tabLabels := m.workspaceTabLabels()
	return ui.RenderTokenList(
		parentEntries, parentIdx,
		m.tokenAccessors, m.tokenCursor, m.tokenDataPreview,
		tabLabels, m.activeTab,
		m.version, m.width, explorerHeight,
	)
}
