package app

import (
	"testing"

	"github.com/janosmiko/vau/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeleteEntryKeepsMountAfterMountSwitch(t *testing.T) {
	c, fv := newFakeVault(t)
	fv.put("a", map[string]string{"k": "secret"})
	fv.put("other/a", map[string]string{"k": "other"})

	m := newTestModel()
	m.client = c.WithMount("other")

	del := m.deleteEntry(model.Entry{Name: "a"})
	m.client = m.client.WithMount("secret")

	_, isErr := del().(errorMsg)
	require.False(t, isErr)
	_, otherExists := fv.get("other/a")
	assert.False(t, otherExists, "delete must act on the mount active when it started")
	_, secretExists := fv.get("a")
	assert.True(t, secretExists, "delete must not touch the mount switched to later")
}
