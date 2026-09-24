package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A path change between Update returning the command and the command
// running must not change which parent gets listed.
func TestListParentCapturesPathBeforeRace(t *testing.T) {
	c, fv := newFakeVault(t)
	fv.put("app1/foo", map[string]string{"k": "v"})
	fv.put("app2/bar", map[string]string{"k": "v"})

	m := newTestModel()
	m.client = c
	m.path = []string{"app1/", "sub1/"}

	cmd := m.listParent()

	// Simulate Update changing m.path before the command goroutine runs.
	m.path = []string{"app2/", "sub2/"}

	msg := cmd()
	lr, ok := msg.(listResultMsg)
	require.True(t, ok)
	assert.Equal(t, "app1/", lr.path)
}
