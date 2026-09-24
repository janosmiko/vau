package app

import (
	"fmt"
	"strings"

	"github.com/janosmiko/vau/internal/model"
	"github.com/janosmiko/vau/internal/ui"
)

// renderBackground renders the view content behind any overlay, based on bgMode.
func (m *Model) renderBackground(bgMode model.ViewMode, tabLabels []string) string {
	switch bgMode {
	case model.ModePolicyList:
		return m.renderPolicyListView()
	case model.ModeAuthMethods:
		return m.renderAuthMethodListView()
	case model.ModeRoleList:
		return m.renderRoleListView()
	case model.ModeEntityList:
		return m.renderEntityListView()
	case model.ModeGroupList:
		return m.renderGroupListView()
	case model.ModeTokenList:
		return m.renderTokenListView()
	case model.ModeExplorer:
		explorerHeight := m.height - 2
		if m.atMountLevel {
			return ui.RenderExplorer(
				nil, // no parent at mount level
				m.mountEntries(),
				m.previewEntries, m.previewSecret, m.previewMode,
				-1, m.mountCursor, nil, nil, "mounts",
				tabLabels, m.activeTab,
				"",
				m.version,
				m.width, explorerHeight,
			)
		}
		return ui.RenderExplorer(
			m.leftPaneEntries(),
			m.visibleEntries(),
			m.previewEntries, m.previewSecret, m.previewMode,
			m.leftPaneSelectedIdx(),
			m.visibleCursor(),
			m.selectedVisible(),
			m.path, m.client.Mount(),
			tabLabels, m.activeTab,
			m.highlightQuery(),
			m.version,
			m.width, explorerHeight,
		)
	}
	return ""
}

// renderStatusBar renders the progress/error/status line above the help bar.
func (m *Model) renderStatusBar() string {
	if m.progress.active {
		var progressText string
		if m.progress.total > 0 {
			progressText = fmt.Sprintf("%s %d/%d secrets... [Ctrl+C to cancel]",
				m.progress.operation, m.progress.current, m.progress.total)
		} else {
			progressText = fmt.Sprintf("%s... (%d done) [Ctrl+C to cancel]",
				m.progress.operation, m.progress.current)
		}
		return ui.StatusStyle.Render(progressText)
	}
	if m.errMsg != "" {
		return ui.ErrorStyle.Render("Error: " + m.errMsg)
	}
	if m.status != "" {
		return ui.StatusStyle.Render(m.status)
	}
	return ""
}

// renderCenteredOverlay renders the overlay for m.mode (if any) on top of base.
func (m *Model) renderCenteredOverlay(base string) string {
	overlay := m.buildOverlay()
	if overlay == "" {
		return base
	}
	return ui.PlaceOverlay(base, overlay, m.width, m.height)
}

// buildOverlay returns the overlay content for m.mode, or "" if m.mode has no overlay.
func (m *Model) buildOverlay() string {
	switch m.mode {
	case model.ModeSecret, model.ModeSecretEdit:
		editingKey := ""
		editingView := ""
		editingColumn := -1
		if m.mode == model.ModeSecretEdit {
			editingKey = m.secretEditKey
			editingView = m.textInput.View()
			editingColumn = m.secretEditColumn
		}
		return ui.RenderSecretOverlay(
			m.secret, m.secretCursor, m.revealed,
			m.secretAllRevealed, m.secretJSONView,
			editingKey, editingView, editingColumn,
			m.secretBase64,
			m.width, m.height,
		)
	case model.ModeConfirm:
		return ui.RenderConfirmOverlay(m.confirmMsg, m.confirmInput.View(), m.width)
	case model.ModeInput:
		return ui.RenderInputOverlay(m.inputLabel, m.textInput.View(), m.width)
	case model.ModeSearch:
		return ui.RenderSearchOverlay("Search: ", m.searchInput.View(), m.width)
	case model.ModeFilter:
		return ui.RenderSearchOverlay("Filter: ", m.searchInput.View(), m.width)
	case model.ModeVersionHistory:
		return ui.RenderVersionHistoryOverlay(m.versionHistory, m.versionCursor, m.versionPath, m.width, m.height)
	case model.ModeDockerConfig:
		return ui.RenderDockerConfigOverlay(m.dockerTitle, m.dockerFields, m.dockerCursor, m.dockerRevealed, m.width, m.height)
	case model.ModeHelp:
		return ui.RenderHelpScreen(m.width, m.height, m.helpScroll, m.helpFilter, m.helpSearching, &m.searchInput)
	case model.ModeJumpPath:
		return ui.RenderJumpPathOverlay(m.textInput.View(), m.jumpCompletions, m.jumpCompIdx, m.width, m.height)
	case model.ModeBookmark:
		return ui.RenderBookmarkOverlay(m.bookmarks, m.bookmarkFilter, m.bookmarkSearching, m.bookmarkCursor, m.width, m.height)
	case model.ModeThemePicker:
		return ui.RenderThemePickerOverlay(m.themeEntries, m.themeCursor, m.activeColorscheme, m.width, m.height)
	case model.ModePolicyView:
		return ui.RenderPolicyViewOverlay(m.policyName, m.policyContent, m.policyScroll, m.width, m.height)
	case model.ModeRoleView:
		return ui.RenderRoleViewOverlay(m.roleName, m.roleData, m.roleScroll, m.width, m.height)
	case model.ModeEntityView:
		return ui.RenderEntityViewOverlay(m.entityName, m.entityData, m.entityScroll, m.width, m.height)
	case model.ModeGroupView:
		return ui.RenderGroupViewOverlay(m.groupName, m.groupData, m.groupScroll, m.width, m.height)
	case model.ModeTokenView:
		return ui.RenderRoleViewOverlay(m.tokenAccessors[m.tokenCursor].Name, m.tokenData, m.tokenScroll, m.width, m.height)
	case model.ModeTokenCreated:
		return ui.RenderTokenCreatedOverlay(m.tokenCreatedData, m.width, m.height)
	}
	return ""
}

func (m *Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	// Build tab labels (show only the last path segment for brevity)
	var tabLabels []string
	if len(m.tabs) > 1 {
		m.saveCurrentTab()
		for _, t := range m.tabs {
			tabLabels = append(tabLabels, tabLabel(t))
		}
	}

	// Render the background view based on underlying mode
	// (overlays like confirm/input/search appear on top)
	content := m.renderBackground(m.backgroundMode(), tabLabels)

	helpBar := ui.HelpBar(m.mode, m.width)
	base := content + "\n" + m.renderStatusBar() + "\n" + helpBar

	return m.renderCenteredOverlay(base)
}

// highlightQuery returns the current search/filter term for highlighting.
func (m *Model) highlightQuery() string {
	switch m.mode {
	case model.ModeSearch:
		return m.searchInput.Value()
	case model.ModeFilter:
		return m.searchInput.Value()
	default:
		return m.filterQuery
	}
}

// visibleEntries returns the filtered entry list (or full list if no filter).
func (m *Model) visibleEntries() []model.Entry {
	if m.filterQuery == "" {
		return m.entries
	}
	result := make([]model.Entry, 0, len(m.filteredIdx))
	for _, idx := range m.filteredIdx {
		if idx < len(m.entries) {
			result = append(result, m.entries[idx])
		}
	}
	return result
}

func (m *Model) visibleCursor() int {
	return m.cursor
}

func (m *Model) selectedEntry() *model.Entry {
	vis := m.visibleEntries()
	if m.cursor < 0 || m.cursor >= len(vis) {
		return nil
	}
	e := vis[m.cursor]
	return &e
}

func (m *Model) applyFilter() {
	if m.filterQuery == "" {
		m.filteredIdx = nil
		return
	}
	q := strings.ToLower(m.filterQuery)
	m.filteredIdx = nil
	for i, e := range m.entries {
		if strings.Contains(strings.ToLower(e.Name), q) {
			m.filteredIdx = append(m.filteredIdx, i)
		}
	}
}

func (m *Model) clearFilter() {
	m.filterQuery = ""
	m.filteredIdx = nil
}

// backgroundMode returns the underlying view mode (explorer or secret)
// that should render behind any overlay (confirm/input/search).
func (m *Model) backgroundMode() model.ViewMode {
	switch m.mode {
	case model.ModeConfirm:
		return m.prevConfirmMode
	case model.ModeInput:
		return m.prevInputMode
	case model.ModeSecret, model.ModeSecretEdit, model.ModeVersionHistory, model.ModeDockerConfig, model.ModeSearch, model.ModeFilter, model.ModeHelp, model.ModeJumpPath, model.ModeBookmark, model.ModeThemePicker:
		return model.ModeExplorer
	case model.ModePolicyView:
		return model.ModePolicyList
	case model.ModeRoleView:
		return model.ModeRoleList
	case model.ModeEntityView:
		return model.ModeEntityList
	case model.ModeGroupView:
		return model.ModeGroupList
	case model.ModeTokenView, model.ModeTokenCreated:
		return model.ModeTokenList
	default:
		return m.mode
	}
}

// Left pane: at root level show mounts + access categories, otherwise show parent entries
func (m *Model) leftPaneEntries() []model.Entry {
	if len(m.path) == 0 {
		return m.mountEntries()
	}
	return m.parentList
}

func (m *Model) leftPaneSelectedIdx() int {
	if len(m.path) == 0 {
		// Highlight current mount in the combined root list
		mountName := m.client.Mount() + "/"
		for i, e := range m.mountEntries() {
			if e.Name == mountName {
				return i
			}
		}
		return 0
	}
	parentDir := m.parentPath()
	if pos, ok := m.cursorMemory[parentDir]; ok {
		return pos
	}
	return 0
}

// Access category names shown at root level alongside KV mounts.
const (
	accessPolicies    = "[Policies]"
	accessAuthMethods = "[Auth Methods]"
	accessEntities    = "[Entities]"
	accessGroups      = "[Groups]"
	accessLeases      = "[Leases]"
	accessTokens      = "[Tokens]"
)

func (m *Model) mountEntries() []model.Entry {
	entries := make([]model.Entry, 0, len(m.mounts)+5)
	for _, mt := range m.mounts {
		entries = append(entries, model.Entry{Name: mt + "/", IsDir: true})
	}
	// Access categories
	entries = append(entries,
		model.Entry{Name: accessPolicies, IsDir: true},
		model.Entry{Name: accessAuthMethods, IsDir: true},
		model.Entry{Name: accessEntities, IsDir: true},
		model.Entry{Name: accessGroups, IsDir: true},
		model.Entry{Name: accessLeases, IsDir: true},
		model.Entry{Name: accessTokens, IsDir: true},
	)
	return entries
}

// isAccessCategory returns true if the entry name is a virtual access category.
func isAccessCategory(name string) bool {
	switch name {
	case accessPolicies, accessAuthMethods, accessEntities, accessGroups, accessLeases, accessTokens:
		return true
	}
	return false
}
