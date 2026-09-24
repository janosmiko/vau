package app

import (
	"testing"

	"github.com/janosmiko/vau/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEditReadFromPreviousMountDoesNotOpenEditor(t *testing.T) {
	c, fv := newFakeVault(t)
	fv.put("x", map[string]string{"k": "secret"})

	m := newTestModel()
	m.client = c
	m.mode = model.ModeExplorer
	m.entries = []model.Entry{{Name: "x"}}

	_, read := m.Update(keyMsg("e"))
	require.NotNil(t, read)
	m.client = c.WithMount("other")

	_, next := m.Update(read())
	assert.Nil(t, next, "a result from the previous mount must not open the editor")
	assert.Nil(t, m.secret)

	_, save := m.Update(editorResultMsg{data: map[string]string{"k": "edited"}})
	if save != nil {
		_ = save()
	}
	_, written := fv.get("other/x")
	assert.False(t, written, "data read on secret/ must never be saved to other/")
}

func TestPreviewFromPreviousMountIsDropped(t *testing.T) {
	c, fv := newFakeVault(t)
	fv.put("x", map[string]string{"k": "secret"})

	m := newTestModel()
	m.client = c
	m.entries = []model.Entry{{Name: "x"}}

	preview := m.loadPreview()
	popup := m.readSecret("x")
	m.client = c.WithMount("other")

	_, _ = m.Update(preview())
	assert.Nil(t, m.previewSecret)

	_, _ = m.Update(popup())
	assert.Equal(t, model.ModeExplorer, m.mode)
	assert.Nil(t, m.secret)
}

func TestListFromPreviousMountIsDropped(t *testing.T) {
	c, fv := newFakeVault(t)
	fv.put("dir/x", map[string]string{"k": "secret"})

	m := newTestModel()
	m.client = c
	m.path = []string{"dir/"}

	list := m.listDir("dir/")
	m.client = c.WithMount("other")
	m.entries = []model.Entry{{Name: "mine"}}

	_, _ = m.Update(list())
	assert.Equal(t, []model.Entry{{Name: "mine"}}, m.entries)
}

func TestVersionResultsFromPreviousMountAreDropped(t *testing.T) {
	m := newTestModel()
	m.mode = model.ModeSecret

	_, _ = m.Update(versionHistoryMsg{mount: "other", path: "x", versions: []model.SecretVersion{{Version: "1"}}})
	assert.Equal(t, model.ModeSecret, m.mode)
	assert.Nil(t, m.versionHistory)

	_, _ = m.Update(versionDetailMsg{mount: "other", version: 1, secret: &model.Secret{Path: "x"}})
	assert.Nil(t, m.secret)
}

func TestVersionResultsForAnotherPathAreDropped(t *testing.T) {
	m := newTestModel()
	m.mode = model.ModeSecret
	m.versionPath = "b"
	mount := m.client.Mount()

	_, _ = m.Update(versionHistoryMsg{mount: mount, path: "a", versions: []model.SecretVersion{{Version: "7"}}})
	assert.Equal(t, model.ModeSecret, m.mode)
	assert.Nil(t, m.versionHistory, "a destroy on b must never use the version list of a")

	_, _ = m.Update(versionDetailMsg{mount: mount, path: "a", version: 7, secret: &model.Secret{Path: "a"}})
	assert.Nil(t, m.secret)
}

func TestStaleYankResultIsDropped(t *testing.T) {
	c, fv := newFakeVault(t)
	fv.put("a", map[string]string{"k": "1"})
	fv.put("b", map[string]string{"k": "2"})

	m := newTestModel()
	m.client = c
	yankA := m.yankSecret("a")
	yankB := m.yankSecret("b")

	_, _ = m.Update(yankB())
	_, _ = m.Update(yankA())

	require.Len(t, m.yankedSecrets, 1)
	assert.Equal(t, "b", m.yankedSecrets[0].Path)
}

func TestPendingYankOnNewMountDoesNotUnlockPaste(t *testing.T) {
	c, fv := newFakeVault(t)
	fv.put("a", map[string]string{"k": "1"})

	m := newTestModel()
	m.client = c
	m.mode = model.ModeExplorer
	_, _ = m.Update(m.yankSecret("a")())
	require.Equal(t, c.Mount(), m.yankMount)

	m.client = c.WithMount("other")
	m.entries = []model.Entry{{Name: "b"}}
	_, pending := m.handleExplorerKey(keyMsg("y"))
	require.NotNil(t, pending)

	_, paste := m.handleExplorerKey(keyMsg("p"))
	assert.Nil(t, paste)
	assert.Equal(t, "Cannot paste across different mounts", m.errMsg)
}

func TestPendingCutNeverDeletesPreviouslyYankedSecret(t *testing.T) {
	c, fv := newFakeVault(t)
	fv.put("dir/old", map[string]string{"k": "1"})
	fv.put("dir/new", map[string]string{"k": "2"})

	m := newTestModel()
	m.client = c
	m.mode = model.ModeExplorer
	m.path = []string{"dir/"}
	m.entries = []model.Entry{{Name: "old"}, {Name: "new"}}

	_, yank := m.handleExplorerKey(keyMsg("y"))
	_, _ = m.Update(yank())

	m.cursor = 1
	_, pending := m.handleExplorerKey(keyMsg("x"))
	require.NotNil(t, pending)

	m.path = []string{"other/"}
	_, paste := m.handleExplorerKey(keyMsg("p"))
	if paste != nil {
		_ = paste()
	}
	_, exists := fv.get("dir/old")
	assert.True(t, exists, "a copy-yanked secret must never be deleted by a pending cut")
}

func TestNewSecretResultsFromPreviousMountAreDropped(t *testing.T) {
	m := newTestModel()
	m.mode = model.ModeExplorer
	old := m.client.Mount()
	m.client = m.client.WithMount("other")

	_, cmd := m.Update(newSecretEditorMsg{path: "x", mount: old})
	assert.Nil(t, cmd, "the editor must not open for a path checked on another mount")

	_, _ = m.Update(confirmCreateMsg{path: "x", mount: old})
	assert.Empty(t, m.confirmMsg)

	_, _ = m.Update(confirmCreateEditorMsg{path: "x", mount: old})
	assert.Empty(t, m.confirmMsg)

	_, _ = m.Update(newSecretInlineMsg{secret: &model.Secret{Path: "x"}, mount: old})
	assert.Equal(t, model.ModeExplorer, m.mode)
	assert.Nil(t, m.secret)
}
