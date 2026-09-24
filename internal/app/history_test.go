package app

import (
	"testing"

	"github.com/janosmiko/vau/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestExecuteUndo_CutPaste_WritesBeforeDeleting covers a failed restore-write
// during cut-paste undo: the secret at the moved-to path must survive, since
// it is the only remaining copy of the data.
func TestExecuteUndo_CutPaste_WritesBeforeDeleting(t *testing.T) {
	c, fv := newFakeVault(t)
	m := newTestModel()
	m.client = c

	fv.put("moved", map[string]string{"k": "v"})
	fv.failWrite("original")

	action := model.UndoAction{
		Type:    model.UndoCutPaste,
		Path:    "moved",
		OldPath: "original",
		Data:    map[string]string{"k": "v"},
		Keys:    []string{"k"},
	}

	msg := m.executeUndo(action)()

	_, isErr := msg.(errorMsg)
	require.True(t, isErr, "a failed restore-write must surface as an error")

	_, stillThere := fv.get("moved")
	assert.True(t, stillThere, "the moved secret must not be deleted before the restore-write succeeds")
	_, restored := fv.get("original")
	assert.False(t, restored, "the restore-write itself failed, so the original path must stay empty")
}

func TestCutPaste_UndoThenRedo_MovesSecretAgain(t *testing.T) {
	c, fv := newFakeVault(t)
	m := newTestModel()
	m.client = c
	fv.put("moved", map[string]string{"k": "v"})

	undone := m.executeUndo(model.UndoAction{
		Type:    model.UndoCutPaste,
		Path:    "moved",
		OldPath: "original",
		Data:    map[string]string{"k": "v"},
		Keys:    []string{"k"},
	})()
	redo, ok := undone.(redoableStatusMsg)
	require.True(t, ok, "undo failed: %v", undone)

	_ = m.executeRedo(redo.redo)()

	_, atDst := fv.get("moved")
	assert.True(t, atDst, "redo must write the secret back to the paste destination")
	_, atSrc := fv.get("original")
	assert.False(t, atSrc, "redo must delete the cut source again")
}
