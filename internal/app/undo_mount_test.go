package app

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/janosmiko/vau/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUndoRedoStayOnOriginalMount(t *testing.T) {
	c, fv := newFakeVault(t)
	fv.put("x", map[string]string{"k": "secret"})
	fv.put("other/x", map[string]string{"k": "other"})

	m := newTestModel()
	m.client = c
	m.mode = model.ModeExplorer

	_, _ = m.Update(m.deleteEntry(model.Entry{Name: "x"})())
	require.Len(t, m.undoStack, 1)
	m.client = c.WithMount("other")

	_, undo := m.Update(keyMsg("u"))
	require.NotNil(t, undo)
	_, _ = m.Update(undo())

	restored, ok := fv.get("x")
	assert.True(t, ok, "undo must recreate the secret on the mount it was deleted from")
	assert.Equal(t, "secret", restored["k"])
	other, _ := fv.get("other/x")
	assert.Equal(t, "other", other["k"], "undo must not write into the mount switched to later")

	_, redo := m.Update(tea.KeyMsg{Type: tea.KeyCtrlR})
	require.NotNil(t, redo)
	_ = redo()

	_, ok = fv.get("x")
	assert.False(t, ok, "redo must delete on the original mount")
	_, ok = fv.get("other/x")
	assert.True(t, ok, "redo must not delete from the mount switched to later")
}

func TestUndoReloadFromOtherMountLeavesOpenSecret(t *testing.T) {
	m := newTestModel()
	m.mode = model.ModeSecret
	m.secret = &model.Secret{Path: "x", Data: map[string]string{"k": "mine"}, Keys: []string{"k"}}

	_, _ = m.Update(redoableStatusMsg{
		redo:         model.UndoAction{Mount: "other", Path: "x"},
		reloadSecret: &model.Secret{Path: "x", Data: map[string]string{"k": "theirs"}, Keys: []string{"k"}},
	})
	assert.Equal(t, "mine", m.secret.Data["k"])
}
