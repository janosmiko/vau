package app

import (
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/janosmiko/vau/internal/config"
	"github.com/janosmiko/vau/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// blockBookmarksSave points XDG_STATE_HOME at a path whose parent is a
// regular file, so config.SaveBookmarks fails on os.MkdirAll.
func blockBookmarksSave(t *testing.T) {
	t.Helper()
	tmp := t.TempDir()
	blocker := filepath.Join(tmp, "blocker")
	require.NoError(t, os.WriteFile(blocker, []byte("x"), 0o600))
	t.Setenv("XDG_STATE_HOME", filepath.Join(blocker, "state"))
}

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

// TestHandleBookmarkOverlayKey_Delete_SaveFailure_KeepsBookmarks covers a
// failed persist: m.bookmarks must stay untouched and the error surfaced,
// instead of silently dropping the bookmark in memory only.
func TestHandleBookmarkOverlayKey_Delete_SaveFailure_KeepsBookmarks(t *testing.T) {
	blockBookmarksSave(t)

	m := &Model{
		mode: model.ModeBookmark,
		bookmarks: []config.Bookmark{
			{Name: "first", Mount: "kv", Path: "a", Slot: "a"},
			{Name: "second", Mount: "kv", Path: "b", Slot: "b"},
		},
		bookmarkCursor: 0,
	}

	m.handleBookmarkOverlayKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'D'}})

	assert.Len(t, m.bookmarks, 2, "unsaved deletion must not change in-memory bookmarks")
	assert.NotEmpty(t, m.errMsg, "save failure must be reported")
}

// TestHandleBookmarkOverlayKey_CtrlX_SaveFailure_KeepsBookmarks covers a
// failed persist of the clear-all action.
func TestHandleBookmarkOverlayKey_CtrlX_SaveFailure_KeepsBookmarks(t *testing.T) {
	blockBookmarksSave(t)

	m := &Model{
		mode: model.ModeBookmark,
		bookmarks: []config.Bookmark{
			{Name: "a", Mount: "kv", Path: "a"},
			{Name: "b", Mount: "kv", Path: "b"},
		},
	}

	m.handleBookmarkOverlayKey(tea.KeyMsg{Type: tea.KeyCtrlX})

	assert.Len(t, m.bookmarks, 2, "unsaved clear must not change in-memory bookmarks")
	assert.NotEqual(t, "All marks deleted", m.status)
	assert.NotEmpty(t, m.errMsg, "save failure must be reported")
}

// TestHandleMarkSave_Overwrite_SaveFailure_KeepsOriginal covers a failed
// persist of a slot overwrite (second m+slot press).
func TestHandleMarkSave_Overwrite_SaveFailure_KeepsOriginal(t *testing.T) {
	blockBookmarksSave(t)

	m := newTestModel()
	m.bookmarks = []config.Bookmark{
		{Name: "secret/old", Mount: "secret", Path: "old", Slot: "a"},
	}
	m.lastMarkAttempt = "a"

	m.handleMarkSave("a")

	require.Len(t, m.bookmarks, 1)
	assert.Equal(t, "old", m.bookmarks[0].Path, "unsaved overwrite must not change in-memory bookmark")
	assert.NotEmpty(t, m.errMsg, "save failure must be reported")
}

// TestHandleMarkSave_New_SaveFailure_DoesNotAppend covers a failed persist
// of a brand new slot.
func TestHandleMarkSave_New_SaveFailure_DoesNotAppend(t *testing.T) {
	blockBookmarksSave(t)

	m := newTestModel()

	m.handleMarkSave("a")

	assert.Empty(t, m.bookmarks, "unsaved mark must not be appended in memory")
	assert.NotEmpty(t, m.errMsg, "save failure must be reported")
}
