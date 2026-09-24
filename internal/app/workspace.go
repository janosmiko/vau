package app

import (
	"fmt"
	"maps"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/janosmiko/vau/internal/model"
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
			maps.Copy(newTab.selected, m.tabs[m.activeTab].selected)
			newTab.cursorMemory = make(map[string]int)
			maps.Copy(newTab.cursorMemory, m.tabs[m.activeTab].cursorMemory)
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
