package app

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/janosmiko/vau/internal/config"
	"github.com/janosmiko/vau/internal/model"
	"github.com/stretchr/testify/assert"
)

// TestHandleBookmarkOverlayKey_Delete_MatchesBySlot covers the case where two
// slots point at the same mount and path. Deleting the selected slot must not
// remove the other slot that happens to share the same location.
func TestHandleBookmarkOverlayKey_Delete_MatchesBySlot(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())

	m := &Model{
		mode: model.ModeBookmark,
		bookmarks: []config.Bookmark{
			{Name: "kv/same", Mount: "kv", Path: "same", Slot: "a"},
			{Name: "kv/same", Mount: "kv", Path: "same", Slot: "b"},
		},
		bookmarkCursor: 1,
	}

	m.handleBookmarkOverlayKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'D'}})

	assert.Len(t, m.bookmarks, 1)
	assert.Equal(t, "a", m.bookmarks[0].Slot)
}
