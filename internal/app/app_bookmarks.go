package app

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/janosmiko/vau/internal/config"
	"github.com/janosmiko/vau/internal/model"
	"github.com/janosmiko/vau/internal/ui"
)

// handleBookmarkSearchInputKey handles text-input keys while the bookmark
// filter is being typed.
func (m *Model) handleBookmarkSearchInputKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "esc":
		m.bookmarkSearching = false
		m.bookmarkFilter = ""
		m.bookmarkCursor = 0
	case "enter":
		m.bookmarkSearching = false
	case "backspace":
		if len(m.bookmarkFilter) > 0 {
			m.bookmarkFilter = m.bookmarkFilter[:len(m.bookmarkFilter)-1]
			m.bookmarkCursor = 0
		}
	default:
		if len(key) == 1 && key[0] >= 32 && key[0] < 127 {
			m.bookmarkFilter += key
			m.bookmarkCursor = 0
		}
	}
	return m, nil
}

// handleBookmarkOverlayKey handles keys in the bookmark overlay mode.
func (m *Model) handleBookmarkOverlayKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	filtered := m.filteredBookmarks()

	// When searching (typing in filter input), handle text input keys
	if m.bookmarkSearching {
		return m.handleBookmarkSearchInputKey(key)
	}

	// Normal bookmark overlay navigation
	switch key {
	case "esc":
		if m.bookmarkFilter != "" {
			m.bookmarkFilter = ""
			m.bookmarkCursor = 0
			return m, nil
		}
		m.mode = model.ModeExplorer
		return m, nil

	case "/":
		m.bookmarkSearching = true
		return m, nil

	case "enter", "l":
		if len(filtered) > 0 && m.bookmarkCursor < len(filtered) {
			bm := filtered[m.bookmarkCursor]
			m.mode = model.ModeExplorer
			return m, m.jumpToBookmark(bm)
		}
		return m, nil

	case "D":
		if len(filtered) > 0 && m.bookmarkCursor < len(filtered) {
			bm := filtered[m.bookmarkCursor]
			for i, b := range m.bookmarks {
				if b.Mount == bm.Mount && b.Path == bm.Path {
					m.bookmarks = append(m.bookmarks[:i], m.bookmarks[i+1:]...)
					break
				}
			}
			_ = config.SaveBookmarks(m.bookmarks)
			newFiltered := m.filteredBookmarks()
			if m.bookmarkCursor >= len(newFiltered) && m.bookmarkCursor > 0 {
				m.bookmarkCursor = len(newFiltered) - 1
			}
			if len(m.bookmarks) == 0 {
				m.mode = model.ModeExplorer
				m.status = "All marks deleted"
			}
		}
		return m, nil

	case "ctrl+x":
		m.bookmarks = nil
		_ = config.SaveBookmarks(m.bookmarks)
		m.mode = model.ModeExplorer
		m.status = "All marks deleted"
		return m, nil

	case "j", "down", "ctrl+n":
		if m.bookmarkCursor < len(filtered)-1 {
			m.bookmarkCursor++
		}
		return m, nil

	case "k", "up", "ctrl+p":
		if m.bookmarkCursor > 0 {
			m.bookmarkCursor--
		}
		return m, nil

	default:
		// Quick jump: pressing a slot key (a-z, 0-9) jumps directly to that mark
		if len(key) == 1 && ((key[0] >= 'a' && key[0] <= 'z') || (key[0] >= '0' && key[0] <= '9')) {
			for _, bm := range m.bookmarks {
				if bm.Slot == key {
					m.mode = model.ModeExplorer
					return m, m.jumpToBookmark(bm)
				}
			}
			m.status = fmt.Sprintf("Mark '%s' not set", key)
			m.mode = model.ModeExplorer
			return m, nil
		}
	}
	return m, nil
}

// filteredBookmarks returns bookmarks matching the current filter.
func (m *Model) filteredBookmarks() []config.Bookmark {
	if m.bookmarkFilter == "" {
		return m.bookmarks
	}
	lf := strings.ToLower(m.bookmarkFilter)
	var result []config.Bookmark
	for _, bm := range m.bookmarks {
		if strings.Contains(strings.ToLower(bm.Name), lf) {
			result = append(result, bm)
		}
	}
	return result
}

// handleBookmarkMouse handles mouse events in the bookmark overlay.
func (m *Model) handleBookmarkMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	filtered := m.filteredBookmarks()
	if msg.Button == tea.MouseButtonWheelUp {
		if m.bookmarkCursor > 0 {
			m.bookmarkCursor--
		}
		return m, nil
	}
	if msg.Button == tea.MouseButtonWheelDown {
		if m.bookmarkCursor < len(filtered)-1 {
			m.bookmarkCursor++
		}
		return m, nil
	}
	return m, nil
}

// handleMarkSave processes the second key after pressing the mark key (m).
// Valid keys (a-z, 0-9) save the current location to that slot.
// If the slot is already occupied, shows a warning and requires pressing m+slot again to overwrite.
func (m *Model) handleMarkSave(key string) (tea.Model, tea.Cmd) {
	if key == "esc" {
		m.status = ""
		m.lastMarkAttempt = ""
		return m, nil
	}
	// Only accept single alphanumeric characters as slot names
	if len(key) != 1 || (key[0] < 'a' || key[0] > 'z') && (key[0] < '0' || key[0] > '9') {
		m.status = "Invalid mark key (use a-z, 0-9)"
		m.lastMarkAttempt = ""
		return m, nil
	}
	slot := key
	mount := m.client.Mount()
	path := m.currentPath()
	name := mount + "/" + path

	// Check if slot is already occupied
	for i, bm := range m.bookmarks {
		if bm.Slot == slot {
			if bm.Mount == mount && bm.Path == path {
				m.status = "Mark '" + slot + "' already set to this location"
				m.lastMarkAttempt = ""
				return m, nil
			}
			// Slot exists with different location — check if this is a confirmed overwrite
			if m.lastMarkAttempt == slot {
				// Second press — overwrite
				m.bookmarks[i].Mount = mount
				m.bookmarks[i].Path = path
				m.bookmarks[i].Name = name
				if err := config.SaveBookmarks(m.bookmarks); err != nil {
					m.errMsg = "Failed to save mark: " + err.Error()
					m.lastMarkAttempt = ""
					return m, nil
				}
				m.status = fmt.Sprintf("Mark '%s' updated: %s", slot, name)
				m.lastMarkAttempt = ""
				return m, nil
			}
			// First press — warn and remember attempt
			m.lastMarkAttempt = slot
			m.status = fmt.Sprintf("Mark '%s' already set (%s) — press m+%s again to overwrite", slot, bm.Name, slot)
			return m, nil
		}
	}

	// Slot is free — save directly
	m.lastMarkAttempt = ""
	m.bookmarks = append(m.bookmarks, config.Bookmark{
		Name:  name,
		Mount: mount,
		Path:  path,
		Slot:  slot,
	})
	if err := config.SaveBookmarks(m.bookmarks); err != nil {
		m.errMsg = "Failed to save mark: " + err.Error()
		return m, nil
	}
	m.status = fmt.Sprintf("Mark '%s' set: %s", slot, name)
	return m, nil
}

// handleThemePickerKey handles keyboard input in the theme picker overlay.
func (m *Model) handleThemePickerKey(msg tea.KeyMsg) tea.Model {
	key := msg.String()

	switch key {
	case "esc", "q":
		// Revert to the previously active theme and close.
		ui.ApplyTheme(m.activeColorscheme, m.config.Theme)
		m.mode = model.ModeExplorer

	case "enter", "l":
		// Confirm selection (runtime only, not persisted) and close.
		if m.themeCursor >= 0 && m.themeCursor < len(m.themeEntries) && !m.themeEntries[m.themeCursor].IsHeader {
			name := m.themeEntries[m.themeCursor].Name
			ui.ApplyTheme(name, m.config.Theme)
			m.activeColorscheme = name
			m.status = "Colorscheme: " + name + " (set colorscheme in config to persist)"
		}
		m.mode = model.ModeExplorer

	case "j", "down", "ctrl+n":
		m.themeMoveCursor(1)

	case "k", "up", "ctrl+p":
		m.themeMoveCursor(-1)
	}
	return m
}

// themeMoveCursor moves the theme picker cursor by delta, skipping headers.
func (m *Model) themeMoveCursor(delta int) {
	next := m.themeCursor
	for {
		next += delta
		if next < 0 || next >= len(m.themeEntries) {
			return // boundary reached, don't move
		}
		if !m.themeEntries[next].IsHeader {
			m.themeCursor = next
			ui.ApplyTheme(m.themeEntries[m.themeCursor].Name, m.config.Theme)
			return
		}
	}
}

// handleThemePickerMouse handles mouse events in the theme picker overlay.
func (m *Model) handleThemePickerMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if msg.Button == tea.MouseButtonWheelUp {
		m.themeMoveCursor(-1)
		return m, nil
	}
	if msg.Button == tea.MouseButtonWheelDown {
		m.themeMoveCursor(1)
		return m, nil
	}
	return m, nil
}

// jumpToBookmark navigates to a bookmark's mount and path.
func (m *Model) jumpToBookmark(bm config.Bookmark) tea.Cmd {
	m.clearFilter()
	m.selected = make(map[int]bool)
	m.cursorMemory = nil

	// Switch mount if needed
	if bm.Mount != m.client.Mount() {
		m.client.SetMount(bm.Mount)
	}
	m.atMountLevel = false

	// Parse path into segments
	if bm.Path == "" {
		m.path = nil
	} else {
		cleaned := strings.TrimSuffix(bm.Path, "/")
		if cleaned == "" {
			m.path = nil
		} else {
			parts := strings.Split(cleaned, "/")
			m.path = make([]string, len(parts))
			for i, p := range parts {
				m.path[i] = p + "/"
			}
		}
	}
	m.cursor = 0
	m.mode = model.ModeExplorer
	m.status = fmt.Sprintf("Jumped to bookmark: %s/%s", bm.Mount, bm.Path)
	return m.refresh()
}
