package app

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/janosmiko/vau/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestWheelDownReachesAccessCategories(t *testing.T) {
	m := newTestModel()
	m.atMountLevel = true
	m.mounts = []string{"secret"}
	m.mountCursor = 0

	total := len(m.mountEntries())
	for range total - 1 {
		_, _ = m.handleExplorerMouse(tea.MouseMsg{Button: tea.MouseButtonWheelDown})
	}

	assert.Equal(t, total-1, m.mountCursor)
}

func TestMountLevelClickEntersAccessCategory(t *testing.T) {
	m := newTestModel()
	m.atMountLevel = true
	m.mounts = []string{"secret"}
	m.width = 100
	m.height = 40
	m.mountCursor = 1 // mountEntries()[1] is the first access category

	_, midEnd := m.explorerColumnBounds()
	result, _ := m.handleMountLevelClick(tea.MouseMsg{
		X:      midEnd + 1,
		Y:      m.headerLineCount() + 1,
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionPress,
	})

	assert.Equal(t, model.ModePolicyList, result.(*Model).mode)
}
