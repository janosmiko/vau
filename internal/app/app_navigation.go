package app

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/janosmiko/vau/internal/model"
)

// --- Navigation ---

func (m *Model) currentPath() string {
	return strings.Join(m.path, "")
}

func (m *Model) parentPath() string {
	if len(m.path) < 2 {
		return ""
	}
	return strings.Join(m.path[:len(m.path)-1], "")
}

func (m *Model) navigateUp() tea.Cmd {
	m.clearFilter()
	m.selected = make(map[int]bool)

	if len(m.path) == 0 {
		// At mount root: go to mount selection
		m.atMountLevel = true
		m.previewEntries = nil
		m.previewSecret = nil
		return m.loadMountPreview()
	}
	m.path = m.path[:len(m.path)-1]
	// Restore cursor position for the directory we're returning to
	dirKey := strings.Join(m.path, "")
	if pos, ok := m.cursorMemory[dirKey]; ok {
		m.cursor = pos
	} else {
		m.cursor = 0
	}
	return m.refresh()
}

func (m *Model) navigateIn() tea.Cmd {
	entry := m.selectedEntry()
	if entry == nil {
		return nil
	}

	m.clearFilter()
	m.selected = make(map[int]bool)

	if entry.IsDir {
		// Remember cursor position for the current directory
		dirKey := strings.Join(m.path, "")
		if m.cursorMemory == nil {
			m.cursorMemory = make(map[string]int)
		}
		m.cursorMemory[dirKey] = m.cursor
		m.path = append(m.path, entry.Name)
		// Restore cursor for the directory we're entering, or start at 0
		newDirKey := strings.Join(m.path, "")
		if pos, ok := m.cursorMemory[newDirKey]; ok {
			m.cursor = pos
		} else {
			m.cursor = 0
		}
		return m.refresh()
	}
	// It's a secret — open secret view
	path := m.currentPath() + entry.Name
	return m.readSecret(path)
}

func (m *Model) refresh() tea.Cmd {
	return tea.Batch(
		m.listDir(m.currentPath()),
		m.listParent(),
	)
}

// --- Vault commands ---

func (m *Model) loadMounts() tea.Cmd {
	return func() tea.Msg {
		mounts, err := m.client.ListMounts()
		return mountsResultMsg{mounts: mounts, err: err}
	}
}

func (m *Model) loadMountPreview() tea.Cmd {
	allEntries := m.mountEntries()
	if m.mountCursor >= len(allEntries) {
		return nil
	}

	selected := allEntries[m.mountCursor]

	// Access category previews
	if isAccessCategory(selected.Name) {
		return m.loadAccessCategoryPreview(selected.Name)
	}

	// KV mount preview
	mount := strings.TrimSuffix(selected.Name, "/")
	return func() tea.Msg {
		entries, err := m.client.ListWithMount(mount, "")
		if err != nil {
			return listResultMsg{path: "@@mount_preview@@", entries: nil, err: err}
		}
		return listResultMsg{path: "@@mount_preview@@", entries: entries}
	}
}

// loadAccessCategoryPreview loads a preview for access categories at root level.
func (m *Model) loadAccessCategoryPreview(name string) tea.Cmd {
	switch name {
	case accessPolicies:
		return func() tea.Msg {
			policies, _ := m.client.ListPolicies()
			entries := make([]model.Entry, len(policies))
			for i, p := range policies {
				entries[i] = model.Entry{Name: p}
			}
			return listResultMsg{path: "@@mount_preview@@", entries: entries}
		}
	case accessAuthMethods:
		return func() tea.Msg {
			methods, _ := m.client.ListAuthMethods()
			return listResultMsg{path: "@@mount_preview@@", entries: methods}
		}
	case accessEntities:
		return func() tea.Msg {
			entries, _ := m.client.ListEntities()
			return listResultMsg{path: "@@mount_preview@@", entries: entries}
		}
	case accessGroups:
		return func() tea.Msg {
			entries, _ := m.client.ListGroups()
			return listResultMsg{path: "@@mount_preview@@", entries: entries}
		}
	case accessLeases:
		return func() tea.Msg {
			// Leases require prefix — show empty for now
			return listResultMsg{path: "@@mount_preview@@", entries: nil}
		}
	case accessTokens:
		return func() tea.Msg {
			entries, _ := m.client.ListTokenAccessors()
			return listResultMsg{path: "@@mount_preview@@", entries: entries}
		}
	}
	return nil
}

func (m *Model) listDir(path string) tea.Cmd {
	return func() tea.Msg {
		entries, err := m.client.List(path)
		return listResultMsg{path: path, entries: entries, err: err}
	}
}

func (m *Model) listParent() tea.Cmd {
	pp := m.parentPath()
	return func() tea.Msg {
		entries, err := m.client.List(pp)
		if err != nil {
			return nil
		}
		return listResultMsg{path: pp, entries: entries}
	}
}

func (m *Model) loadPreview() tea.Cmd {
	entry := m.selectedEntry()
	if entry == nil {
		m.previewMode = model.PreviewHidden
		m.previewEntries = nil
		m.previewSecret = nil
		return nil
	}

	if entry.IsDir {
		m.previewMode = model.PreviewHidden
		m.previewSecret = nil
		path := m.currentPath() + entry.Name
		return func() tea.Msg {
			entries, err := m.client.List(path)
			if err != nil {
				return errorMsg(err.Error())
			}
			return listResultMsg{path: path, entries: entries}
		}
	}

	// It's a secret — load preview.
	// Only reset visibility if the cursor moved to a different secret.
	path := m.currentPath() + entry.Name
	if m.previewSecret == nil || m.previewSecret.Path != path {
		m.previewMode = model.PreviewHidden
	}
	m.previewEntries = nil
	return func() tea.Msg {
		secret, err := m.client.Read(path)
		return secretResultMsg{path: path, secret: secret, err: err}
	}
}

func (m *Model) readSecret(path string) tea.Cmd {
	return func() tea.Msg {
		secret, err := m.client.Read(path)
		if err != nil {
			return errorMsg(err.Error())
		}
		return secretResultMsg{path: path, secret: secret, openPopup: true}
	}
}

// --- Result handlers ---

func (m *Model) handleListResult(msg listResultMsg) (tea.Model, tea.Cmd) {
	if msg.path == "@@mount_preview@@" {
		// Preview result for mount selection
		if msg.err != nil {
			m.previewEntries = nil
		} else {
			m.previewEntries = msg.entries
		}
		m.previewSecret = nil
		return m, nil
	}

	if msg.err != nil {
		m.errMsg = msg.err.Error()
		return m, nil
	}

	currentPath := m.currentPath()
	parentPath := m.parentPath()

	if msg.path == currentPath {
		// If the directory is now empty and we're not at mount root,
		// auto-navigate up to the lowest still-existing ancestor.
		if len(msg.entries) == 0 && len(m.path) > 0 {
			m.path = m.path[:len(m.path)-1]
			return m, tea.Batch(m.listDir(m.currentPath()), m.listParent())
		}
		m.entries = msg.entries
		// Re-apply filter if active
		if m.filterQuery != "" {
			m.applyFilter()
		}
		vis := m.visibleEntries()
		if m.cursor >= len(vis) {
			m.cursor = max(0, len(vis)-1)
		}
		return m, m.loadPreview()
	}

	if msg.path == parentPath {
		m.parentList = msg.entries
		if len(m.path) > 0 {
			// Update parent's cursor memory to point at the current directory entry
			currentDir := m.path[len(m.path)-1]
			for i, e := range m.parentList {
				if e.Name == currentDir {
					if m.cursorMemory == nil {
						m.cursorMemory = make(map[string]int)
					}
					m.cursorMemory[m.parentPath()] = i
					break
				}
			}
		}
		return m, nil
	}

	// Must be a preview result
	m.previewEntries = msg.entries
	m.previewSecret = nil
	return m, nil
}

func (m *Model) handleSecretResult(msg secretResultMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.errMsg = msg.err.Error()
		return m, nil
	}

	if msg.openEditor {
		// Open secret directly in external editor
		m.secret = msg.secret
		m.secretJSONView = true
		return m, m.editSecretInEditor()
	}

	if msg.openPopup {
		// Open secret in popup overlay
		m.mode = model.ModeSecret
		m.secret = msg.secret
		m.secretCursor = 0
		m.revealed = make(map[string]bool)
		m.secretBase64 = make(map[string]bool)
		m.secretAllRevealed = false
		m.secretJSONView = false
		return m, nil
	}

	// Preview result
	m.previewSecret = msg.secret
	m.previewEntries = nil
	return m, nil
}
