package app

import (
	"testing"

	"github.com/janosmiko/vau/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeleteEntry_Dir_DoesNotCountBeforeReturningCmd(t *testing.T) {
	c, fv := newFakeVault(t)
	fv.put("dir/a", map[string]string{"k": "v"})
	fv.put("dir/b", map[string]string{"k": "v"})
	fv.put("dir/c", map[string]string{"k": "v"})

	m := newTestModel()
	m.client = c
	m.path = nil

	cmd := m.deleteEntry(model.Entry{Name: "dir/", IsDir: true})
	require.NotNil(t, cmd)

	assert.True(t, m.progress.active)
	assert.Equal(t, 0, m.progress.total, "total must stay unknown until counting happens inside the cmd")

	msg := cmd()
	done, ok := msg.(progressDoneMsg)
	require.True(t, ok, "expected progressDoneMsg, got %T: %+v", msg, msg)
	assert.NoError(t, done.err)
	assert.Equal(t, 3, done.count)
}

func TestPasteSecrets_Dir_DoesNotCountBeforeReturningCmd(t *testing.T) {
	c, fv := newFakeVault(t)
	fv.put("src/a", map[string]string{"k": "v"})
	fv.put("src/b", map[string]string{"k": "v"})

	m := newTestModel()
	m.client = c
	m.path = []string{"dst/"}
	m.yankPaths = []string{"src"}
	m.yankIsCut = false
	m.yankIsDir = true

	cmd := m.pasteSecrets()
	require.NotNil(t, cmd)

	assert.True(t, m.progress.active)
	assert.Equal(t, 0, m.progress.total, "total must stay unknown until counting happens inside the cmd")

	msg := cmd()
	done, ok := msg.(progressDoneMsg)
	require.True(t, ok, "expected progressDoneMsg, got %T: %+v", msg, msg)
	assert.NoError(t, done.err)
	assert.Equal(t, 2, done.count)

	_, aCopied := fv.get("dst/src/a")
	assert.True(t, aCopied)
}

func TestBulkDelete_DoesNotCountBeforeReturningCmd(t *testing.T) {
	c, fv := newFakeVault(t)
	fv.put("dir/a", map[string]string{"k": "v"})
	fv.put("dir/b", map[string]string{"k": "v"})
	fv.put("solo", map[string]string{"k": "v"})

	m := newTestModel()
	m.client = c
	m.path = nil

	entries := []model.Entry{
		{Name: "dir/", IsDir: true},
		{Name: "solo", IsDir: false},
	}

	cmd := m.bulkDelete(entries)
	require.NotNil(t, cmd)

	assert.True(t, m.progress.active)
	assert.Equal(t, 0, m.progress.total, "total must stay unknown until counting happens inside the cmd")

	msg := cmd()
	done, ok := msg.(progressDoneMsg)
	require.True(t, ok, "expected progressDoneMsg, got %T: %+v", msg, msg)
	assert.NoError(t, done.err)
	assert.Equal(t, 3, done.count)
}
