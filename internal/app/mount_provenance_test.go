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
