package app

import (
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/janosmiko/vau/internal/model"
	"github.com/stretchr/testify/assert"
)

func keyMsg(key string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
}

// When a user binds one key to two actions, the action that comes first in
// the original switch order must win.
func TestExplorerKeyPrecedence_RemappedBindings(t *testing.T) {
	newExplorer := func(overrides map[string]string) *Model {
		m := newTestModel()
		m.mode = model.ModeExplorer
		m.entries = []model.Entry{{Name: "dir/", IsDir: true}}
		m.previewSecret = &model.Secret{}
		m.confirmInput = textinput.New()
		m.keys.ApplyOverrides(overrides)
		return m
	}

	t.Run("yank beats the Y copy prompt", func(t *testing.T) {
		m := newExplorer(map[string]string{"yank": "Y"})
		m.handleExplorerKey(keyMsg("Y"))
		assert.True(t, m.yankIsDir)
		assert.False(t, m.copyFormatPending)
	})

	t.Run("delete beats edit", func(t *testing.T) {
		m := newExplorer(map[string]string{"edit": "D"})
		m.handleExplorerKey(keyMsg("D"))
		assert.NotEmpty(t, m.confirmMsg)
	})

	t.Run("toggle values beats half page down", func(t *testing.T) {
		m := newExplorer(map[string]string{"half_down": "v"})
		m.handleExplorerKey(keyMsg("v"))
		assert.Equal(t, model.PreviewValues, m.previewMode)
	})
}

func TestRoleListKeyPrecedence_QuitBeatsNav(t *testing.T) {
	m := newTestModel()
	m.roles = []model.Entry{{Name: "a"}, {Name: "b"}}
	m.roleCursor = 1
	m.keys.ApplyOverrides(map[string]string{"quit": "g"})

	_, cmd := m.handleRoleListKey(keyMsg("g"))

	assert.Equal(t, 1, m.roleCursor)
	if assert.NotNil(t, cmd) {
		assert.IsType(t, tea.QuitMsg{}, cmd())
	}
}
