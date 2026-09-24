package app

import (
	"testing"

	"github.com/janosmiko/vau/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Update can mutate selection/path between renameEntry returning its cmd
// and Bubble Tea running that cmd on another goroutine.
func TestRenameEntry_CapturesStateBeforeCmdRuns(t *testing.T) {
	c, fv := newFakeVault(t)
	fv.put("a", map[string]string{"k": "v"})

	m := newTestModel()
	m.client = c
	m.path = nil
	m.entries = []model.Entry{{Name: "a", IsDir: false}}
	m.cursor = 0

	cmd := m.renameEntry("b")
	require.NotNil(t, cmd)

	// Simulate a concurrent Update changing selection/path before cmd runs.
	m.path = []string{"other/"}
	m.cursor = 5
	m.entries = nil

	msg := cmd()
	_, isErr := msg.(errorMsg)
	require.False(t, isErr, "renameEntry cmd returned error: %v", msg)

	_, oldExists := fv.get("a")
	assert.False(t, oldExists, "original entry should have been renamed")
	_, newExists := fv.get("b")
	assert.True(t, newExists, "renamed entry should exist under the new name")
}
