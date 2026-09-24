package app

import (
	"fmt"
	"maps"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/janosmiko/vau/internal/model"
)

func (m *Model) handleMountKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	switch {
	case matchKey(key, m.keys.Quit):
		return m, tea.Quit

	case matchKey(key, m.keys.CloseTab):
		if len(m.tabs) <= 1 {
			return m, tea.Quit
		}
		m.tabs = append(m.tabs[:m.activeTab], m.tabs[m.activeTab+1:]...)
		if m.activeTab >= len(m.tabs) {
			m.activeTab = len(m.tabs) - 1
		}
		m.loadTab(m.activeTab)
		return m, nil

	case matchKey(key, m.keys.Up):
		allEntries := m.mountEntries()
		if m.mountCursor < len(allEntries)-1 {
			m.mountCursor++
			return m, m.loadMountPreview()
		}

	case matchKey(key, m.keys.Down):
		if m.mountCursor > 0 {
			m.mountCursor--
			return m, m.loadMountPreview()
		}

	case matchKey(key, m.keys.Right) || matchKey(key, m.keys.Open):
		allEntries := m.mountEntries()
		if m.mountCursor < len(allEntries) {
			selected := allEntries[m.mountCursor]
			if isAccessCategory(selected.Name) {
				return m.enterAccessCategory(selected.Name)
			}
			// KV mount
			mountName := strings.TrimSuffix(selected.Name, "/")
			m.client = m.client.WithMount(mountName)
			m.atMountLevel = false
			m.path = nil
			m.cursor = 0
			return m, m.refresh()
		}
	}
	return m, nil
}

// handleExplorerTabKey handles tab management keys in the explorer.
func (m *Model) handleExplorerTabKey(key string) (tea.Model, tea.Cmd, bool) {
	switch {
	case matchKey(key, m.keys.CloseTab):
		if len(m.tabs) <= 1 {
			return m, tea.Quit, true
		}
		// Close current tab
		m.tabs = append(m.tabs[:m.activeTab], m.tabs[m.activeTab+1:]...)
		if m.activeTab >= len(m.tabs) {
			m.activeTab = len(m.tabs) - 1
		}
		m.loadTab(m.activeTab)
		return m, nil, true

	case matchKey(key, m.keys.NewTab):
		if len(m.tabs) < 9 {
			m.saveCurrentTab()
			newTab := m.tabs[m.activeTab] // struct copy
			// Deep copy maps for the new tab
			newTab.selected = make(map[int]bool)
			maps.Copy(newTab.selected, m.tabs[m.activeTab].selected)
			newTab.cursorMemory = make(map[string]int)
			maps.Copy(newTab.cursorMemory, m.tabs[m.activeTab].cursorMemory)
			m.tabs = append(m.tabs, newTab)
			m.activeTab = len(m.tabs) - 1
			m.loadTab(m.activeTab)
			m.status = fmt.Sprintf("Tab %d created", m.activeTab+1)
		}
		return m, nil, true

	case matchKey(key, m.keys.NextTab):
		if len(m.tabs) > 1 {
			m.saveCurrentTab()
			m.loadTab((m.activeTab + 1) % len(m.tabs))
			return m, m.refresh(), true
		}
		return m, nil, true

	case matchKey(key, m.keys.PrevTab):
		if len(m.tabs) > 1 {
			m.saveCurrentTab()
			m.loadTab((m.activeTab - 1 + len(m.tabs)) % len(m.tabs))
			return m, m.refresh(), true
		}
		return m, nil, true
	}
	return m, nil, false
}

// handleExplorerNavKey handles cursor-movement and selection keys in the explorer.
func (m *Model) handleExplorerNavKey(key string) (tea.Model, tea.Cmd, bool) {
	switch {
	case matchKey(key, m.keys.Up):
		vis := m.visibleEntries()
		if m.cursor < len(vis)-1 {
			m.cursor++
			return m, m.loadPreview(), true
		}
		return m, nil, true

	case matchKey(key, m.keys.Down):
		if m.cursor > 0 {
			m.cursor--
			return m, m.loadPreview(), true
		}
		return m, nil, true

	case matchKey(key, m.keys.Left):
		return m, m.navigateUp(), true

	case matchKey(key, m.keys.Right):
		// Navigate into directories only
		entry := m.selectedEntry()
		if entry != nil && entry.IsDir {
			return m, m.navigateIn(), true
		}
		return m, nil, true

	case matchKey(key, m.keys.Open):
		// Enter directory or open secret popup
		return m, m.navigateIn(), true

	case matchKey(key, m.keys.Top):
		m.cursor = 0
		return m, m.loadPreview(), true

	case matchKey(key, m.keys.Bottom):
		vis := m.visibleEntries()
		if len(vis) > 0 {
			m.cursor = len(vis) - 1
			return m, m.loadPreview(), true
		}
		return m, nil, true

	case matchKey(key, m.keys.Select):
		// Toggle selection on current entry
		vis := m.visibleEntries()
		if m.cursor >= 0 && m.cursor < len(vis) {
			actualIdx := m.actualIdx(m.cursor)
			if m.selected[actualIdx] {
				delete(m.selected, actualIdx)
			} else {
				m.selected[actualIdx] = true
			}
			// Move cursor down
			if m.cursor < len(vis)-1 {
				m.cursor++
			}
			return m, m.loadPreview(), true
		}
		return m, nil, true
	}
	return m, nil, false
}

// handleExplorerPageKey handles half-page and full-page scroll keys in the explorer.
func (m *Model) handleExplorerPageKey(key string) (tea.Model, tea.Cmd, bool) {
	switch {
	case matchKey(key, m.keys.HalfDown):
		// Half-page scroll down
		vis := m.visibleEntries()
		if len(vis) > 0 {
			halfPage := m.explorerHalfPage()
			m.cursor += halfPage
			if m.cursor >= len(vis) {
				m.cursor = len(vis) - 1
			}
			return m, m.loadPreview(), true
		}
		return m, nil, true

	case matchKey(key, m.keys.HalfUp):
		// Half-page scroll up
		if len(m.visibleEntries()) > 0 {
			halfPage := m.explorerHalfPage()
			m.cursor -= halfPage
			if m.cursor < 0 {
				m.cursor = 0
			}
			return m, m.loadPreview(), true
		}
		return m, nil, true

	case matchKey(key, m.keys.FullDown):
		// Full-page scroll down
		vis := m.visibleEntries()
		if len(vis) > 0 {
			fullPage := m.explorerHalfPage() * 2
			m.cursor += fullPage
			if m.cursor >= len(vis) {
				m.cursor = len(vis) - 1
			}
			return m, m.loadPreview(), true
		}
		return m, nil, true

	case matchKey(key, m.keys.FullUp):
		// Full-page scroll up
		if len(m.visibleEntries()) > 0 {
			fullPage := m.explorerHalfPage() * 2
			m.cursor -= fullPage
			if m.cursor < 0 {
				m.cursor = 0
			}
			return m, m.loadPreview(), true
		}
		return m, nil, true
	}
	return m, nil, false
}

// handleExplorerClipboardKey handles delete/cut/undo/redo keys in the explorer.
func (m *Model) handleExplorerClipboardKey(key string) (tea.Model, tea.Cmd, bool) {
	switch {
	case matchKey(key, m.keys.Delete):
		if len(m.selected) > 0 {
			// Bulk delete
			count := len(m.selected)
			entries := m.selectedEntries()
			m.confirmMsg = fmt.Sprintf("Delete %d selected items?", count)
			m.prevConfirmMode = model.ModeExplorer
			m.confirmAction = func() tea.Cmd {
				return m.bulkDelete(entries)
			}
			m.enterConfirmMode()
			return m, nil, true
		}
		entry := m.selectedEntry()
		if entry != nil {
			name := entry.Name
			entryCopy := *entry
			if entry.IsDir {
				m.confirmMsg = fmt.Sprintf("Recursively delete %q and all contents?", name)
			} else {
				m.confirmMsg = fmt.Sprintf("Delete %q?", name)
			}
			m.prevConfirmMode = model.ModeExplorer
			m.confirmAction = func() tea.Cmd {
				return m.deleteEntry(entryCopy)
			}
			m.enterConfirmMode()
			return m, nil, true
		}
		return m, nil, true

	case matchKey(key, m.keys.Cut):
		if len(m.selected) > 0 {
			entries := m.selectedEntries()
			m.status = fmt.Sprintf("Cutting %d items...", len(entries))
			m.selected = make(map[int]bool)
			return m, m.bulkYankSecrets(entries, true), true
		}
		entry := m.selectedEntry()
		if entry != nil {
			if entry.IsDir {
				path := m.currentPath() + strings.TrimSuffix(entry.Name, "/")
				m.yankIsCut = true
				m.yankIsDir = true
				m.yankPaths = []string{path}
				m.yankMount = m.client.Mount()
				m.yankedSecrets = nil
				m.yankSeq++
				m.status = "Cut (directory): " + entry.Name
				return m, nil, true
			}
			path := m.currentPath() + entry.Name
			m.yankIsCut = true
			m.yankIsDir = false
			m.yankPaths = []string{path}
			m.status = "Cutting " + entry.Name + "..."
			return m, m.yankSecret(path), true
		}
		return m, nil, true

	case matchKey(key, m.keys.Undo):
		if len(m.undoStack) == 0 {
			m.errMsg = "Nothing to undo"
			return m, nil, true
		}
		action := m.undoStack[len(m.undoStack)-1]
		m.undoStack = m.undoStack[:len(m.undoStack)-1]
		return m, m.executeUndo(action), true

	case matchKey(key, m.keys.Redo):
		if len(m.redoStack) == 0 {
			m.errMsg = "Nothing to redo"
			return m, nil, true
		}
		action := m.redoStack[len(m.redoStack)-1]
		m.redoStack = m.redoStack[:len(m.redoStack)-1]
		return m, m.executeRedo(action), true
	}
	return m, nil, false
}

// handleExplorerModeKey handles keys that switch the explorer into another mode/overlay.
func (m *Model) handleExplorerModeKey(key string) (tea.Model, tea.Cmd, bool) {
	switch {
	case matchKey(key, m.keys.Search):
		// Jump-to search: overlay, cursor jumps to first match
		m.clearFilter()
		m.mode = model.ModeSearch
		m.priorCursor = m.cursor
		m.searchInput.SetValue("")
		m.searchInput.Focus()
		return m, textinput.Blink, true

	case matchKey(key, m.keys.Filter):
		// Filter: overlay, hides non-matching entries
		m.clearFilter()
		m.mode = model.ModeFilter
		m.priorCursor = m.cursor
		m.searchInput.SetValue("")
		m.searchInput.Focus()
		return m, textinput.Blink, true

	case matchKey(key, m.keys.Help):
		m.mode = model.ModeHelp
		m.helpScroll = 0
		m.helpFilter = ""
		m.helpSearching = false
		return m, nil, true

	case matchKey(key, m.keys.JumpPath):
		m.mode = model.ModeJumpPath
		m.textInput.SetValue(m.currentPath())
		m.textInput.Focus()
		m.textInput.CursorEnd()
		m.jumpCompletions = nil
		m.jumpCompIdx = -1
		m.jumpLastInput = ""
		return m, tea.Batch(textinput.Blink, m.loadJumpCompletions()), true

	case matchKey(key, m.keys.Edit):
		// Edit secret in external editor from explorer
		entry := m.selectedEntry()
		if entry != nil && !entry.IsDir {
			path := m.currentPath() + entry.Name
			client := m.client
			return m, func() tea.Msg {
				secret, err := client.Read(path)
				if err != nil {
					return errorMsg(err.Error())
				}
				return secretResultMsg{mount: client.Mount(), path: path, secret: secret, openEditor: true}
			}, true
		}
		return m, nil, true

	case matchKey(key, m.keys.BookmarkSave):
		m.markPending = true
		m.status = "Set mark: [a-z, 0-9] (not j/k/l)"
		return m, nil, true

	case matchKey(key, m.keys.BookmarkShow):
		m.mode = model.ModeBookmark
		m.bookmarkFilter = ""
		m.bookmarkCursor = 0
		return m, nil, true

	case matchKey(key, m.keys.ThemePicker):
		// Open theme picker overlay
		m.mode = model.ModeThemePicker
		// Set cursor to currently active theme (skip headers)
		m.themeCursor = -1
		for i, entry := range m.themeEntries {
			if !entry.IsHeader && entry.Name == m.activeColorscheme {
				m.themeCursor = i
				break
			}
		}
		// Fallback: first non-header entry
		if m.themeCursor < 0 {
			for i, entry := range m.themeEntries {
				if !entry.IsHeader {
					m.themeCursor = i
					break
				}
			}
		}
		return m, nil, true
	}
	return m, nil, false
}

// handleExplorerMiscKey handles the remaining explorer action keys.
func (m *Model) handleExplorerMiscKey(key string) (tea.Model, tea.Cmd, bool) {
	switch {
	case matchKey(key, m.keys.Refresh):
		return m, m.refresh(), true

	case matchKey(key, m.keys.NewSecret):
		return m, m.startInput(model.InputNewSecret, "New secret name:", ""), true

	case matchKey(key, m.keys.NewSecretEditor):
		// New secret via external editor
		return m, m.startInput(model.InputNewSecretEditor, "New secret name (editor):", ""), true

	case matchKey(key, m.keys.Rename):
		entry := m.selectedEntry()
		if entry != nil {
			displayName := strings.TrimSuffix(entry.Name, "/")
			return m, m.startInput(model.InputRename, "Rename to:", displayName), true
		}
		return m, nil, true

	case matchKey(key, m.keys.Yank):
		if len(m.selected) > 0 {
			entries := m.selectedEntries()
			m.status = fmt.Sprintf("Yanking %d items...", len(entries))
			m.selected = make(map[int]bool)
			return m, m.bulkYankSecrets(entries, false), true
		}
		entry := m.selectedEntry()
		if entry != nil {
			if entry.IsDir {
				path := m.currentPath() + strings.TrimSuffix(entry.Name, "/")
				m.yankIsCut = false
				m.yankIsDir = true
				m.yankPaths = []string{path}
				m.yankMount = m.client.Mount()
				m.yankedSecrets = nil
				m.yankSeq++
				m.status = "Yanked (directory): " + entry.Name
				return m, nil, true
			}
			path := m.currentPath() + entry.Name
			m.yankIsCut = false
			m.yankIsDir = false
			m.yankPaths = []string{path}
			m.status = "Yanking " + entry.Name + "..."
			return m, m.yankSecret(path), true
		}
		return m, nil, true

	case matchKey(key, m.keys.Paste):
		// Paste reads the source through the current mount, so a yank from another mount hits the wrong secret.
		if m.yankMount != "" && m.yankMount != m.client.Mount() {
			m.errMsg = "Cannot paste across different mounts"
			return m, nil, true
		}
		if m.yankIsDir {
			cmd := m.pasteSecrets()
			if m.yankIsCut {
				m.yankIsCut = false
				m.yankIsDir = false
			}
			return m, cmd, true
		}
		if len(m.yankedSecrets) > 0 {
			cmd := m.pasteSecrets()
			m.yankIsCut = false
			return m, cmd, true
		}
		m.errMsg = "Nothing yanked"
		return m, nil, true

	case matchKey(key, m.keys.ToggleValues):
		// Toggle secret value visibility in preview
		if m.previewSecret != nil {
			if m.previewMode == model.PreviewValues {
				m.previewMode = model.PreviewHidden
			} else {
				m.previewMode = model.PreviewValues
			}
		}
		return m, nil, true

	case matchKey(key, m.keys.ToggleJSON):
		// Toggle JSON preview
		if m.previewSecret != nil {
			if m.previewMode == model.PreviewJSON {
				m.previewMode = model.PreviewHidden
			} else {
				m.previewMode = model.PreviewJSON
			}
		}
		return m, nil, true

	case key == "Y":
		// Copy secret as JSON/YAML/dotenv from explorer
		entry := m.selectedEntry()
		if entry != nil && !entry.IsDir && m.previewSecret != nil {
			m.copySecret = m.previewSecret
			m.copyFormatPending = true
			m.status = "Copy as: (j)son  (y)aml  (d)otenv"
			return m, nil, true
		}
		return m, nil, true
	}
	return m, nil, false
}

func (m *Model) handleExplorerKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Handle mark pending states (two-key sequences like m+a, '+a)
	if m.markPending {
		m.markPending = false
		return m.handleMarkSave(key)
	}

	// Clear stale mark overwrite confirmation when the user does any other action
	if m.lastMarkAttempt != "" {
		m.lastMarkAttempt = ""
	}

	// Handle copy format pending (Y + j/y/d)
	if m.copyFormatPending {
		m.copyFormatPending = false
		return m.handleCopyFormat(key)
	}

	if matchKey(key, m.keys.Quit) {
		return m, tea.Quit
	}

	for _, handler := range []func(string) (tea.Model, tea.Cmd, bool){
		m.handleExplorerTabKey,
		m.handleExplorerNavKey,
		m.handleExplorerMiscKey,
		m.handleExplorerPageKey,
		m.handleExplorerClipboardKey,
		m.handleExplorerModeKey,
	} {
		if result, cmd, handled := handler(key); handled {
			return result, cmd
		}
	}
	return m, nil
}
