package app

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/janosmiko/vau/internal/model"
	"github.com/janosmiko/vau/internal/ui"
)

// explorerColumnBounds calculates the x-coordinate boundaries for the three
// explorer columns. Returns leftEnd, midEnd (right column extends to the edge).
func (m *Model) explorerColumnBounds() (leftEnd, midEnd int) {
	usable := m.width - 6 // 3 columns x 2 border chars
	leftW := usable * 12 / 100
	midW := usable * 51 / 100
	if leftW < 10 {
		leftW = 10
	}
	if midW < 10 {
		midW = 10
	}
	// Left column occupies x: 0 .. leftW+1 (content + 2 border chars)
	leftEnd = leftW + 2
	// Mid column occupies x: leftEnd .. leftEnd+midW+1
	midEnd = leftEnd + midW + 2
	return
}

// explorerColHeight returns the number of visible entry rows in the explorer.
func (m *Model) headerLineCount() int {
	if len(m.tabs) > 1 {
		return 2 // tab bar + breadcrumb
	}
	return 1 // breadcrumb only
}

func (m *Model) explorerColHeight() int {
	// Must match RenderExplorer: height(m.height-2) - 2 - headerLines
	h := max(m.height-4-m.headerLineCount(), 1)
	return h
}

// explorerScrollOffset returns the scroll offset for the current entry list,
// matching the logic in renderEntryList.
func (m *Model) explorerScrollOffset() int {
	colHeight := m.explorerColHeight()
	start := 0
	if m.cursor >= colHeight {
		start = m.cursor - colHeight + 1
	}
	return start
}

func (m *Model) handleExplorerMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	// Scroll wheel: move cursor up/down
	if msg.Button == tea.MouseButtonWheelUp {
		if m.atMountLevel {
			if m.mountCursor > 0 {
				m.mountCursor--
				return m, m.loadMountPreview()
			}
			return m, nil
		}
		if m.cursor > 0 {
			m.cursor--
			return m, m.loadPreview()
		}
		return m, nil
	}
	if msg.Button == tea.MouseButtonWheelDown {
		if m.atMountLevel {
			if m.mountCursor < len(m.mountEntries())-1 {
				m.mountCursor++
				return m, m.loadMountPreview()
			}
			return m, nil
		}
		vis := m.visibleEntries()
		if m.cursor < len(vis)-1 {
			m.cursor++
			return m, m.loadPreview()
		}
		return m, nil
	}

	// Left click: select entry or navigate
	if msg.Button == tea.MouseButtonLeft && msg.Action == tea.MouseActionPress {
		// Check for tab bar click (row 0)
		if msg.Y == 1 && len(m.tabs) > 1 {
			return m.handleTabBarClick(msg.X)
		}

		if m.atMountLevel {
			return m.handleMountLevelClick(msg)
		}

		leftEnd, midEnd := m.explorerColumnBounds()
		colHeight := m.explorerColHeight()
		// Entry rows start after header lines + top border
		entryRow := msg.Y - m.headerLineCount() - 1
		if entryRow < 0 || entryRow >= colHeight {
			return m, nil
		}

		if msg.X < leftEnd {
			// Left column click: navigate up
			return m, m.navigateUp()
		} else if msg.X < midEnd {
			// Mid column click: move cursor to clicked entry
			scrollOffset := m.explorerScrollOffset()
			targetIdx := scrollOffset + entryRow
			vis := m.visibleEntries()
			if targetIdx >= 0 && targetIdx < len(vis) {
				m.cursor = targetIdx
				return m, m.loadPreview()
			}
		} else {
			// Right column click: navigate into the selected entry if it's a directory
			entry := m.selectedEntry()
			if entry != nil && entry.IsDir {
				return m, m.navigateIn()
			}
		}
	}

	// Double click on middle column: open entry (like Enter)
	if msg.Button == tea.MouseButtonLeft && msg.Action == tea.MouseActionMotion {
		// bubbletea doesn't distinguish double-click from drag,
		// so we skip this to avoid accidental navigation.
		return m, nil
	}

	return m, nil
}

// handleTabBarClick determines which tab was clicked based on x position
// and switches to it.
func (m *Model) handleTabBarClick(x int) (tea.Model, tea.Cmd) {
	if len(m.tabs) <= 1 {
		return m, nil
	}

	// Mirror the exact logic from RenderTabBar to compute tab positions.
	m.saveCurrentTab()

	var tabLabels []string
	for _, t := range m.tabs {
		var label string
		if len(t.path) > 0 {
			label = t.path[len(t.path)-1]
		} else {
			label = t.mount + "/"
		}
		tabLabels = append(tabLabels, label)
	}

	maxBarW := m.width - 2
	maxPathLen := max(maxBarW/len(tabLabels), 8)

	// Measure each tab's visual width (label + padding of 1 on each side = +2)
	type tabInfo struct {
		width int
	}
	tabs := make([]tabInfo, len(tabLabels))
	for i, path := range tabLabels {
		if len(path) > maxPathLen {
			path = "…" + path[len(path)-maxPathLen+1:]
		}
		label := fmt.Sprintf("%d %s", i+1, path)
		// Padding(0,1) adds 1 cell each side
		tabs[i] = tabInfo{width: len([]rune(label)) + 2}
	}

	sepW := 3 // " │ " rendered width

	// Determine visible window (same algorithm as RenderTabBar)
	totalW := 0
	for i, t := range tabs {
		totalW += t.width
		if i < len(tabs)-1 {
			totalW += sepW
		}
	}

	left := 0
	right := len(tabs) - 1
	if totalW > maxBarW {
		// Window around active tab
		left = m.activeTab
		right = m.activeTab
		usedW := tabs[m.activeTab].width
		for {
			expanded := false
			if left > 0 {
				needed := sepW + tabs[left-1].width
				if usedW+needed <= maxBarW {
					left--
					usedW += needed
					expanded = true
				}
			}
			if right < len(tabs)-1 {
				needed := sepW + tabs[right+1].width
				if usedW+needed <= maxBarW {
					right++
					usedW += needed
					expanded = true
				}
			}
			if !expanded {
				break
			}
		}
	}

	// Walk the visible tabs and match click position
	pos := 1 // leading space
	if left > 0 {
		// "◂" indicator + separator
		indicatorW := 3 // "◂" with padding(0,1) = 1+2
		pos += indicatorW + sepW
	}
	for i := left; i <= right; i++ {
		tabW := tabs[i].width
		if x >= pos && x < pos+tabW {
			if i != m.activeTab {
				m.loadTab(i)
				return m, m.refresh()
			}
			return m, nil
		}
		if i < right {
			pos += tabW + sepW
		}
	}

	return m, nil
}

// handleMountLevelClick handles clicks when at the mount selection level.
func (m *Model) handleMountLevelClick(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	_, midEnd := m.explorerColumnBounds()
	colHeight := m.explorerColHeight()
	entryRow := msg.Y - m.headerLineCount() - 1
	if entryRow < 0 || entryRow >= colHeight {
		return m, nil
	}

	allEntries := m.mountEntries()
	if msg.X < midEnd {
		// Click on a mount or access category in the mid column: select it
		if entryRow >= 0 && entryRow < len(allEntries) {
			m.mountCursor = entryRow
			return m, m.loadMountPreview()
		}
	} else {
		// Click on right column: enter the selected mount or access category
		if m.mountCursor < len(allEntries) {
			selected := allEntries[m.mountCursor]
			if isAccessCategory(selected.Name) {
				return m.enterAccessCategory(selected.Name)
			}
			m.client = m.client.WithMount(strings.TrimSuffix(selected.Name, "/"))
			m.atMountLevel = false
			m.path = nil
			m.cursor = 0
			return m, m.refresh()
		}
	}
	return m, nil
}

func (m *Model) handleSecretMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	// Scroll wheel: move secret cursor
	if msg.Button == tea.MouseButtonWheelUp {
		if m.secretCursor > 0 {
			m.secretCursor--
		}
		return m, nil
	}
	if msg.Button == tea.MouseButtonWheelDown {
		if m.secret != nil && m.secretCursor < len(m.secret.Keys)-1 {
			m.secretCursor++
		}
		return m, nil
	}

	// Left click outside the popup: close it
	if msg.Button == tea.MouseButtonLeft && msg.Action == tea.MouseActionPress {
		// Popup dimensions: 75% of screen, centered
		boxW := m.width * 75 / 100
		boxH := m.height * 75 / 100
		if boxW < 50 {
			boxW = 50
		}
		if boxH < 10 {
			boxH = 10
		}
		startCol := (m.width - boxW) / 2
		startRow := (m.height - boxH) / 2
		endCol := startCol + boxW
		endRow := startRow + boxH

		if msg.X < startCol || msg.X >= endCol || msg.Y < startRow || msg.Y >= endRow {
			// Click outside popup: close
			m.mode = model.ModeExplorer
			m.secret = nil
			m.secretCursor = 0
			m.revealed = make(map[string]bool)
			m.secretBase64 = make(map[string]bool)
			m.secretAllRevealed = false
			m.secretJSONView = false
			return m, m.refresh()
		}
	}

	return m, nil
}

func (m *Model) handleHelpMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	maxScroll := max(ui.HelpContentLineCount(m.helpFilter)-m.helpVisibleLines(), 0)

	if msg.Button == tea.MouseButtonWheelUp {
		if m.helpScroll > 0 {
			m.helpScroll--
		}
		return m, nil
	}
	if msg.Button == tea.MouseButtonWheelDown {
		if m.helpScroll < maxScroll {
			m.helpScroll++
		}
		return m, nil
	}

	// Left click outside the help overlay: close it
	if msg.Button == tea.MouseButtonLeft && msg.Action == tea.MouseActionPress {
		boxW := m.width * 70 / 100
		boxH := m.height * 80 / 100
		if boxW < 50 {
			boxW = 50
		}
		if boxH < 20 {
			boxH = 20
		}
		startCol := (m.width - boxW) / 2
		startRow := (m.height - boxH) / 2
		endCol := startCol + boxW
		endRow := startRow + boxH

		if msg.X < startCol || msg.X >= endCol || msg.Y < startRow || msg.Y >= endRow {
			m.mode = model.ModeExplorer
			return m, nil
		}
	}

	return m, nil
}

func (m *Model) handleVersionHistoryMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if msg.Button == tea.MouseButtonWheelUp {
		if m.versionCursor > 0 {
			m.versionCursor--
		}
		return m, nil
	}
	if msg.Button == tea.MouseButtonWheelDown {
		if m.versionCursor < len(m.versionHistory)-1 {
			m.versionCursor++
		}
		return m, nil
	}
	return m, nil
}
