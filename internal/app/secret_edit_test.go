package app

import (
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/janosmiko/vau/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newSecretEditModel returns a model with a fake Vault holding one secret,
// ready to drive the inline edit key handlers in internal/app/app_secret_keys.go.
const secretEditTestPath = "app/db"

func newSecretEditModel(t *testing.T, data map[string]string, keys []string) (*Model, *fakeVault) {
	t.Helper()
	c, fv := newFakeVault(t)
	fv.put(secretEditTestPath, data)

	m := newTestModel()
	m.client = c
	m.mode = model.ModeSecret
	m.textInput = textinput.New()
	m.secret = &model.Secret{Path: secretEditTestPath, Data: copyMap(data), Keys: copySlice(keys)}
	m.secretCursor = 0
	return m, fv
}

// runCmd executes a tea.Cmd and returns its message, failing the test if the
// command is nil or returns an errorMsg.
func runCmd(t *testing.T, cmd tea.Cmd) tea.Msg {
	t.Helper()
	require.NotNil(t, cmd, "expected a non-nil command")
	msg := cmd()
	if em, ok := msg.(errorMsg); ok {
		t.Fatalf("command returned error: %s", string(em))
	}
	return msg
}

func TestInlineEditValue_UndoRestoresOriginal(t *testing.T) {
	m, fv := newSecretEditModel(t, map[string]string{"user": "orig"}, []string{"user"})

	_, cmd := m.handleSecretKey(keyMsg("e"))
	require.NotNil(t, cmd)
	assert.Equal(t, model.ModeSecretEdit, m.mode)

	m.textInput.SetValue("changed")

	_, cmd = m.handleSecretEditKey(keyMsg("enter"))
	msg := runCmd(t, cmd)
	undoMsg, ok := msg.(undoableStatusMsg)
	require.True(t, ok, "expected undoableStatusMsg, got %T", msg)

	got, _ := fv.get(secretEditTestPath)
	assert.Equal(t, "changed", got["user"], "write should have applied the new value")

	undoCmd := m.executeUndo(undoMsg.undo)
	runCmd(t, undoCmd)

	got, _ = fv.get(secretEditTestPath)
	assert.Equal(t, "orig", got["user"], "undo must restore the pre-edit value")
}

func TestInlineRenameKey_UndoRestoresOriginal(t *testing.T) {
	m, fv := newSecretEditModel(t, map[string]string{"user": "orig"}, []string{"user"})

	_, cmd := m.handleSecretKey(keyMsg("e"))
	require.NotNil(t, cmd)
	_, _ = m.handleSecretEditKey(keyMsg("tab"))

	m.textInput.SetValue("username")

	_, cmd = m.handleSecretEditKey(keyMsg("enter"))
	msg := runCmd(t, cmd)
	undoMsg, ok := msg.(undoableStatusMsg)
	require.True(t, ok, "expected undoableStatusMsg, got %T", msg)

	got, _ := fv.get(secretEditTestPath)
	assert.Equal(t, "orig", got["username"], "renamed key should carry the original value")
	_, hasOld := got["user"]
	assert.False(t, hasOld, "old key should be gone after rename")

	undoCmd := m.executeUndo(undoMsg.undo)
	runCmd(t, undoCmd)

	got, _ = fv.get(secretEditTestPath)
	assert.Equal(t, "orig", got["user"], "undo must restore the original key")
	_, hasNew := got["username"]
	assert.False(t, hasNew, "renamed key must not survive undo")
}

func TestInlineEdit_EscRevertsAfterTabRename(t *testing.T) {
	m, _ := newSecretEditModel(t, map[string]string{"user": "orig"}, []string{"user"})

	_, _ = m.handleSecretKey(keyMsg("e"))
	_, _ = m.handleSecretEditKey(keyMsg("tab")) // value column -> key column
	m.textInput.SetValue("renamed")
	_, _ = m.handleSecretEditKey(keyMsg("tab")) // commits the rename, key column -> value column

	_, cmd := m.handleSecretEditKey(keyMsg("esc"))
	assert.Nil(t, cmd)

	assert.Equal(t, []string{"user"}, m.secret.Keys)
	assert.Equal(t, "orig", m.secret.Data["user"])
	_, hasRenamed := m.secret.Data["renamed"]
	assert.False(t, hasRenamed, "esc must discard the uncommitted key rename")
}

func TestInlineEdit_EscRevertsNewRowAfterTab(t *testing.T) {
	m, _ := newSecretEditModel(t, map[string]string{"user": "orig"}, []string{"user"})

	_, cmd := m.handleSecretKey(keyMsg("a"))
	require.NotNil(t, cmd)
	require.Equal(t, []string{"user", ""}, m.secret.Keys)

	m.textInput.SetValue("newkey")
	_, _ = m.handleSecretEditKey(keyMsg("tab"))

	_, cmd = m.handleSecretEditKey(keyMsg("esc"))
	assert.Nil(t, cmd)

	assert.Equal(t, []string{"user"}, m.secret.Keys)
	_, hasNewKey := m.secret.Data["newkey"]
	assert.False(t, hasNewKey, "esc must remove a new row typed via tab, not just an untouched one")
}
