package app

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/janosmiko/vau/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A mount switch in Update must not redirect a cmd that was already returned.
func TestCmdKeepsMountAfterMountSwitch(t *testing.T) {
	c, fv := newFakeVault(t)
	fv.put("a", map[string]string{"k": "secret"})
	fv.put("other/a", map[string]string{"k": "other"})

	m := newTestModel()
	m.client = c
	m.entries = []model.Entry{{Name: "a"}}

	preview := m.loadPreview()
	read := m.readSecret("a")
	require.NotNil(t, preview)

	m.client = m.client.WithMount("other")

	for name, cmd := range map[string]func() any{
		"loadPreview": func() any { return preview() },
		"readSecret":  func() any { return read() },
	} {
		msg, ok := cmd().(secretResultMsg)
		require.True(t, ok, name)
		require.NoError(t, msg.err, name)
		assert.Equal(t, "secret", msg.secret.Data["k"], name)
	}
}

// Bubble Tea runs cmds on another goroutine while Update keeps going. Run under -race.
func TestCmdRunsWhileMountSwitches(t *testing.T) {
	c, fv := newFakeVault(t)
	fv.put("a", map[string]string{"k": "secret"})

	m := newTestModel()
	m.client = c
	m.entries = []model.Entry{{Name: "a"}}

	cmd := m.loadPreview()
	done := make(chan tea.Msg)
	go func() { done <- cmd() }()
	m.client = m.client.WithMount("other")

	msg, ok := (<-done).(secretResultMsg)
	require.True(t, ok)
	require.NoError(t, msg.err)
	assert.Equal(t, "secret", msg.secret.Data["k"])
}
