package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadJumpCompletionsIsAsync(t *testing.T) {
	c, fv := newFakeVault(t)
	fv.put("app1/foo", map[string]string{"k": "v"})

	m := newTestModel()
	m.client = c
	m.textInput.SetValue("app1/")

	cmd := m.loadJumpCompletions()
	require.NotNil(t, cmd)
	assert.Nil(t, m.jumpCompletions)

	msg := cmd()
	result, ok := msg.(jumpCompletionsMsg)
	require.True(t, ok)
	assert.Equal(t, "app1/", result.input)
	assert.Contains(t, result.completions, "app1/foo")
}

func TestJumpCompletionsMsgDroppedWhenStale(t *testing.T) {
	m := newTestModel()
	m.jumpLastInput = "app2/"
	m.jumpCompletions = []string{"app2/keep"}

	_, _, handled := m.updateCoreResult(jumpCompletionsMsg{input: "app1/", completions: []string{"app1/stale"}})

	require.True(t, handled)
	assert.Equal(t, []string{"app2/keep"}, m.jumpCompletions)
}

func TestJumpCompletionsMsgAppliedWhenCurrent(t *testing.T) {
	m := newTestModel()
	m.jumpLastInput = "app1/"

	_, _, handled := m.updateCoreResult(jumpCompletionsMsg{input: "app1/", completions: []string{"app1/foo"}})

	require.True(t, handled)
	assert.Equal(t, []string{"app1/foo"}, m.jumpCompletions)
}
