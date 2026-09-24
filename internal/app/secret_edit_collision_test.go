package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInlineRenameKey_CollisionRejected(t *testing.T) {
	m, fv := newSecretEditModel(t, map[string]string{
		"user": "alice",
		"pass": "secret123",
	}, []string{"user", "pass"})

	_, cmd := m.handleSecretKey(keyMsg("e"))
	require.NotNil(t, cmd)
	_, _ = m.handleSecretEditKey(keyMsg("tab"))

	m.textInput.SetValue("pass")

	_, cmd = m.handleSecretEditKey(keyMsg("enter"))

	if cmd != nil {
		msg := cmd()
		_, isUndoable := msg.(undoableStatusMsg)
		assert.False(t, isUndoable, "collision must not produce a successful write")
	}

	assert.NotEmpty(t, m.errMsg, "collision should surface an error message")

	got, _ := fv.get(secretEditTestPath)
	assert.Equal(t, "secret123", got["pass"], "existing key's value must survive a colliding rename attempt")
	assert.Contains(t, got, "user", "original key should not be dropped by a rejected rename")
}
