package app

import (
	"testing"

	"github.com/janosmiko/vau/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPasteSecrets_SameDirDuplicate_UndoTargetsActualCopy(t *testing.T) {
	c, fv := newFakeVault(t)
	fv.put("name", map[string]string{"k": "v"})

	m := newTestModel()
	m.client = c
	m.path = nil
	m.yankedSecrets = []*model.Secret{{Path: "name", Data: map[string]string{"k": "v"}, Keys: []string{"k"}}}
	m.yankPaths = []string{"name"}
	m.yankIsCut = false
	m.yankIsDir = false

	cmd := m.pasteSecrets()
	require.NotNil(t, cmd)

	msg := cmd()
	result, ok := msg.(undoableStatusMsg)
	require.True(t, ok, "expected undoableStatusMsg, got %T: %+v", msg, msg)

	assert.Equal(t, "name_1", result.undo.Path, "undo should target the destination the paste loop actually chose")

	_, origExists := fv.get("name")
	assert.True(t, origExists, "original secret should still exist right after paste")
	_, copyExists := fv.get("name_1")
	assert.True(t, copyExists, "pasted copy should exist at the suffixed name")

	undoMsg := m.executeUndo(result.undo)()
	_, isErr := undoMsg.(errorMsg)
	assert.False(t, isErr, "undo returned error: %v", undoMsg)

	_, origExists = fv.get("name")
	assert.True(t, origExists, "undo must not delete the original secret")
	_, copyExists = fv.get("name_1")
	assert.False(t, copyExists, "undo must delete the pasted copy")
}

func TestPasteSecrets_CutOverExistingName_UndoTargetsActualCopy(t *testing.T) {
	c, fv := newFakeVault(t)
	fv.put("src/name", map[string]string{"k": "v1"})
	fv.put("name", map[string]string{"k": "existing"})

	m := newTestModel()
	m.client = c
	m.path = nil
	m.yankedSecrets = []*model.Secret{{Path: "src/name", Data: map[string]string{"k": "v1"}, Keys: []string{"k"}}}
	m.yankPaths = []string{"src/name"}
	m.yankIsCut = true
	m.yankIsDir = false

	cmd := m.pasteSecrets()
	require.NotNil(t, cmd)

	msg := cmd()
	result, ok := msg.(undoableStatusMsg)
	require.True(t, ok, "expected undoableStatusMsg, got %T: %+v", msg, msg)

	assert.Equal(t, "name_1", result.undo.Path)

	undoMsg := m.executeUndo(result.undo)()
	_, isErr := undoMsg.(errorMsg)
	assert.False(t, isErr, "undo returned error: %v", undoMsg)

	_, unrelatedExists := fv.get("name")
	assert.True(t, unrelatedExists, "undo must not delete the unrelated pre-existing secret at name")
	_, copyExists := fv.get("name_1")
	assert.False(t, copyExists, "undo must delete the moved copy")
}
