package app

import (
	"testing"

	"github.com/janosmiko/vau/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestHandleListResultDropsStalePreview(t *testing.T) {
	m := newTestModel()
	m.entries = []model.Entry{{Name: "a/", IsDir: true}, {Name: "b/", IsDir: true}}
	m.cursor = 1 // selected entry is "b/"
	m.previewEntries = []model.Entry{{Name: "old"}}

	stale := listResultMsg{path: "a/", entries: []model.Entry{{Name: "stale"}}}
	_, _ = m.handleListResult(stale)

	assert.Equal(t, []model.Entry{{Name: "old"}}, m.previewEntries)
}

func TestHandleListResultAppliesCurrentPreview(t *testing.T) {
	m := newTestModel()
	m.entries = []model.Entry{{Name: "a/", IsDir: true}, {Name: "b/", IsDir: true}}
	m.cursor = 1 // selected entry is "b/"

	current := listResultMsg{path: "b/", entries: []model.Entry{{Name: "fresh"}}}
	_, _ = m.handleListResult(current)

	assert.Equal(t, []model.Entry{{Name: "fresh"}}, m.previewEntries)
}

func TestHandleSecretResultDropsStalePreview(t *testing.T) {
	m := newTestModel()
	m.entries = []model.Entry{{Name: "a"}, {Name: "b"}}
	m.cursor = 1 // selected entry is "b"
	m.previewSecret = &model.Secret{Path: "old"}

	stale := secretResultMsg{path: "a", secret: &model.Secret{Path: "a"}}
	_, _ = m.handleSecretResult(stale)

	assert.Equal(t, "old", m.previewSecret.Path)
}

func TestHandleSecretResultAppliesCurrentPreview(t *testing.T) {
	m := newTestModel()
	m.entries = []model.Entry{{Name: "a"}, {Name: "b"}}
	m.cursor = 1 // selected entry is "b"

	current := secretResultMsg{path: "b", secret: &model.Secret{Path: "b"}}
	_, _ = m.handleSecretResult(current)

	assert.Equal(t, "b", m.previewSecret.Path)
}
