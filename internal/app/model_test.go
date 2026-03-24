package app

import (
	"reflect"
	"testing"
	"unsafe"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/janosmiko/vau/internal/config"
	"github.com/janosmiko/vau/internal/model"
	"github.com/janosmiko/vau/internal/ui"
	"github.com/janosmiko/vau/internal/vault"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestClient creates a minimal vault.Client with only the mount field set.
// This is safe for pure-function tests that only call Mount() / SetMount().
//
//nolint:gosec,unparam // unsafe is intentional for test-only access to unexported field; mount always "secret" in tests
func newTestClient(mount string) *vault.Client {
	c := new(vault.Client)
	v := reflect.ValueOf(c).Elem()
	f := v.FieldByName("mount")
	// Use unsafe to set the unexported field.
	reflect.NewAt(f.Type(), unsafe.Pointer(f.UnsafeAddr())).Elem().SetString(mount) //nolint:gosec
	return c
}

// newTestModel creates a minimal Model suitable for pure-function testing.
// The vault client only supports Mount()/SetMount(); do not call methods that
// hit the Vault API.
func newTestModel() *Model {
	si := textinput.New()
	return &Model{
		client:       newTestClient("secret"),
		config:       &config.Config{},
		keys:         DefaultKeyMap(),
		selected:     make(map[int]bool),
		cursorMemory: make(map[string]int),
		searchInput:  si,
		tabs: []TabState{{
			selected:     make(map[int]bool),
			cursorMemory: make(map[string]int),
			mount:        "secret",
		}},
	}
}

// ---------------------------------------------------------------------------
// Tab state management: saveCurrentTab / loadTab
// ---------------------------------------------------------------------------

func TestSaveAndLoadTab(t *testing.T) {
	m := newTestModel()

	// Populate model with meaningful state.
	m.path = []string{"dir1/", "dir2/"}
	m.entries = []model.Entry{
		{Name: "a", IsDir: false},
		{Name: "b/", IsDir: true},
	}
	m.cursor = 1
	m.parentList = []model.Entry{{Name: "dir1/", IsDir: true}}
	m.cursorMemory = map[string]int{"dir1/": 3}
	m.previewEntries = []model.Entry{{Name: "child", IsDir: false}}
	m.previewMode = model.PreviewValues
	m.selected = map[int]bool{0: true}
	m.filterQuery = "foo"
	m.filteredIdx = []int{0}
	m.atMountLevel = false
	m.mountCursor = 2

	// Save state into tab 0.
	m.saveCurrentTab()

	// Mutate all fields to different values.
	m.path = []string{"other/"}
	m.entries = nil
	m.cursor = 0
	m.parentList = nil
	m.cursorMemory = nil
	m.previewEntries = nil
	m.previewMode = model.PreviewHidden
	m.selected = make(map[int]bool)
	m.filterQuery = ""
	m.filteredIdx = nil
	m.atMountLevel = true
	m.mountCursor = 0

	// Restore from tab 0.
	m.loadTab(0)

	assert.Equal(t, []string{"dir1/", "dir2/"}, m.path)
	assert.Equal(t, 2, len(m.entries))
	assert.Equal(t, 1, m.cursor)
	assert.Equal(t, 1, len(m.parentList))
	assert.Equal(t, map[string]int{"dir1/": 3}, m.cursorMemory)
	assert.Equal(t, 1, len(m.previewEntries))
	assert.Equal(t, model.PreviewValues, m.previewMode)
	assert.True(t, m.selected[0])
	assert.Equal(t, "foo", m.filterQuery)
	assert.Equal(t, []int{0}, m.filteredIdx)
	assert.False(t, m.atMountLevel)
	assert.Equal(t, 2, m.mountCursor)
}

func TestSaveAndLoadTab_DeepCopiesSlices(t *testing.T) {
	m := newTestModel()
	m.entries = []model.Entry{{Name: "orig", IsDir: false}}
	m.saveCurrentTab()

	// Mutate the model's entries after saving.
	m.entries[0].Name = "mutated"

	// The saved tab should still have the original value.
	assert.Equal(t, "orig", m.tabs[0].entries[0].Name)
}

func TestSaveAndLoadTab_DeepCopiesMaps(t *testing.T) {
	m := newTestModel()
	m.cursorMemory = map[string]int{"a": 1}
	m.selected = map[int]bool{0: true}
	m.saveCurrentTab()

	// Mutate after save.
	m.cursorMemory["a"] = 99
	m.selected[1] = true

	// Saved tab should be unchanged.
	assert.Equal(t, 1, m.tabs[0].cursorMemory["a"])
	assert.False(t, m.tabs[0].selected[1])
}

func TestLoadTab_ResetsMode(t *testing.T) {
	m := newTestModel()
	m.saveCurrentTab()

	m.mode = model.ModeSecret
	m.secret = &model.Secret{Path: "test"}
	m.loadTab(0)

	assert.Equal(t, model.ModeExplorer, m.mode)
	assert.Nil(t, m.secret)
}

// ---------------------------------------------------------------------------
// highlightQuery
// ---------------------------------------------------------------------------

func TestHighlightQuery(t *testing.T) {
	tests := []struct {
		name     string
		mode     model.ViewMode
		inputVal string
		filterQ  string
		want     string
	}{
		{
			name:     "ModeSearch returns searchInput value",
			mode:     model.ModeSearch,
			inputVal: "hello",
			filterQ:  "stale",
			want:     "hello",
		},
		{
			name:     "ModeFilter returns searchInput value",
			mode:     model.ModeFilter,
			inputVal: "world",
			filterQ:  "stale",
			want:     "world",
		},
		{
			name:     "ModeExplorer returns filterQuery",
			mode:     model.ModeExplorer,
			inputVal: "ignored",
			filterQ:  "active",
			want:     "active",
		},
		{
			name:     "default mode returns filterQuery",
			mode:     model.ModeSecret,
			inputVal: "ignored",
			filterQ:  "some-filter",
			want:     "some-filter",
		},
		{
			name:     "ModeSearch with empty input",
			mode:     model.ModeSearch,
			inputVal: "",
			filterQ:  "leftover",
			want:     "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := newTestModel()
			m.mode = tc.mode
			m.searchInput.SetValue(tc.inputVal)
			m.filterQuery = tc.filterQ
			assert.Equal(t, tc.want, m.highlightQuery())
		})
	}
}

// ---------------------------------------------------------------------------
// visibleEntries
// ---------------------------------------------------------------------------

func TestVisibleEntries_NoFilter(t *testing.T) {
	m := newTestModel()
	m.entries = []model.Entry{
		{Name: "a", IsDir: false},
		{Name: "b/", IsDir: true},
	}
	vis := m.visibleEntries()
	assert.Equal(t, m.entries, vis)
}

func TestVisibleEntries_WithFilter(t *testing.T) {
	m := newTestModel()
	m.entries = []model.Entry{
		{Name: "alpha", IsDir: false},
		{Name: "beta", IsDir: false},
		{Name: "gamma", IsDir: false},
	}
	m.filterQuery = "a"
	m.filteredIdx = []int{0, 2} // alpha, gamma

	vis := m.visibleEntries()
	require.Len(t, vis, 2)
	assert.Equal(t, "alpha", vis[0].Name)
	assert.Equal(t, "gamma", vis[1].Name)
}

func TestVisibleEntries_FilteredIdxOutOfBounds(t *testing.T) {
	m := newTestModel()
	m.entries = []model.Entry{{Name: "only", IsDir: false}}
	m.filterQuery = "x"
	m.filteredIdx = []int{0, 5} // 5 is out of bounds

	vis := m.visibleEntries()
	assert.Len(t, vis, 1) // only index 0 is valid
}

func TestVisibleEntries_EmptyEntries(t *testing.T) {
	m := newTestModel()
	m.entries = nil

	vis := m.visibleEntries()
	assert.Empty(t, vis)
}

// ---------------------------------------------------------------------------
// visibleCursor
// ---------------------------------------------------------------------------

func TestVisibleCursor(t *testing.T) {
	m := newTestModel()
	m.cursor = 5
	assert.Equal(t, 5, m.visibleCursor())
}

// ---------------------------------------------------------------------------
// selectedEntry
// ---------------------------------------------------------------------------

func TestSelectedEntry_Valid(t *testing.T) {
	m := newTestModel()
	m.entries = []model.Entry{
		{Name: "a", IsDir: false},
		{Name: "b/", IsDir: true},
	}
	m.cursor = 1

	e := m.selectedEntry()
	require.NotNil(t, e)
	assert.Equal(t, "b/", e.Name)
	assert.True(t, e.IsDir)
}

func TestSelectedEntry_EmptyEntries(t *testing.T) {
	m := newTestModel()
	m.entries = nil
	m.cursor = 0
	assert.Nil(t, m.selectedEntry())
}

func TestSelectedEntry_CursorNegative(t *testing.T) {
	m := newTestModel()
	m.entries = []model.Entry{{Name: "a"}}
	m.cursor = -1
	assert.Nil(t, m.selectedEntry())
}

func TestSelectedEntry_CursorOutOfBounds(t *testing.T) {
	m := newTestModel()
	m.entries = []model.Entry{{Name: "a"}}
	m.cursor = 10
	assert.Nil(t, m.selectedEntry())
}

func TestSelectedEntry_WithFilter(t *testing.T) {
	m := newTestModel()
	m.entries = []model.Entry{
		{Name: "alpha", IsDir: false},
		{Name: "beta", IsDir: false},
		{Name: "gamma", IsDir: false},
	}
	m.filterQuery = "a"
	m.filteredIdx = []int{0, 2}
	m.cursor = 1 // visible index 1 = gamma

	e := m.selectedEntry()
	require.NotNil(t, e)
	assert.Equal(t, "gamma", e.Name)
}

// ---------------------------------------------------------------------------
// applyFilter / clearFilter
// ---------------------------------------------------------------------------

func TestApplyFilter(t *testing.T) {
	m := newTestModel()
	m.entries = []model.Entry{
		{Name: "alpha", IsDir: false},
		{Name: "beta", IsDir: false},
		{Name: "GAMMA", IsDir: true},
		{Name: "delta", IsDir: false},
	}

	m.filterQuery = "a"
	m.applyFilter()

	// All contain "a" (case-insensitive): alpha(0), beta(1), GAMMA(2), delta(3)
	assert.Equal(t, []int{0, 1, 2, 3}, m.filteredIdx)
}

func TestApplyFilter_NoMatch(t *testing.T) {
	m := newTestModel()
	m.entries = []model.Entry{
		{Name: "alpha", IsDir: false},
		{Name: "beta", IsDir: false},
	}

	m.filterQuery = "zzz"
	m.applyFilter()

	assert.Empty(t, m.filteredIdx)
}

func TestApplyFilter_EmptyQuery(t *testing.T) {
	m := newTestModel()
	m.entries = []model.Entry{{Name: "a"}}
	m.filteredIdx = []int{0}

	m.filterQuery = ""
	m.applyFilter()

	assert.Nil(t, m.filteredIdx)
}

func TestApplyFilter_CaseInsensitive(t *testing.T) {
	m := newTestModel()
	m.entries = []model.Entry{
		{Name: "Hello", IsDir: false},
		{Name: "HELLO", IsDir: false},
		{Name: "world", IsDir: false},
	}

	m.filterQuery = "hello"
	m.applyFilter()

	assert.Equal(t, []int{0, 1}, m.filteredIdx)
}

func TestClearFilter(t *testing.T) {
	m := newTestModel()
	m.filterQuery = "something"
	m.filteredIdx = []int{0, 1, 2}

	m.clearFilter()

	assert.Equal(t, "", m.filterQuery)
	assert.Nil(t, m.filteredIdx)
}

// ---------------------------------------------------------------------------
// backgroundMode
// ---------------------------------------------------------------------------

func TestBackgroundMode(t *testing.T) {
	tests := []struct {
		name            string
		mode            model.ViewMode
		prevConfirmMode model.ViewMode
		want            model.ViewMode
	}{
		{
			name: "ModeExplorer returns itself",
			mode: model.ModeExplorer,
			want: model.ModeExplorer,
		},
		{
			name:            "ModeConfirm returns prevConfirmMode",
			mode:            model.ModeConfirm,
			prevConfirmMode: model.ModeSecret,
			want:            model.ModeSecret,
		},
		{
			name:            "ModeConfirm with Explorer prev",
			mode:            model.ModeConfirm,
			prevConfirmMode: model.ModeExplorer,
			want:            model.ModeExplorer,
		},
		{
			name: "ModeInput returns Explorer",
			mode: model.ModeInput,
			want: model.ModeExplorer,
		},
		{
			name: "ModeSecret returns Explorer",
			mode: model.ModeSecret,
			want: model.ModeExplorer,
		},
		{
			name: "ModeSecretEdit returns Explorer",
			mode: model.ModeSecretEdit,
			want: model.ModeExplorer,
		},
		{
			name: "ModeSearch returns Explorer",
			mode: model.ModeSearch,
			want: model.ModeExplorer,
		},
		{
			name: "ModeFilter returns Explorer",
			mode: model.ModeFilter,
			want: model.ModeExplorer,
		},
		{
			name: "ModeVersionHistory returns Explorer",
			mode: model.ModeVersionHistory,
			want: model.ModeExplorer,
		},
		{
			name: "ModeHelp returns Explorer",
			mode: model.ModeHelp,
			want: model.ModeExplorer,
		},
		{
			name: "ModeJumpPath returns Explorer",
			mode: model.ModeJumpPath,
			want: model.ModeExplorer,
		},
		{
			name: "ModeBookmark returns Explorer",
			mode: model.ModeBookmark,
			want: model.ModeExplorer,
		},
		{
			name: "ModeThemePicker returns Explorer",
			mode: model.ModeThemePicker,
			want: model.ModeExplorer,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := newTestModel()
			m.mode = tc.mode
			m.prevConfirmMode = tc.prevConfirmMode
			assert.Equal(t, tc.want, m.backgroundMode())
		})
	}
}

// ---------------------------------------------------------------------------
// filteredBookmarks
// ---------------------------------------------------------------------------

func TestFilteredBookmarks_NoFilter(t *testing.T) {
	m := newTestModel()
	m.bookmarks = []config.Bookmark{
		{Name: "db-prod", Mount: "secret", Path: "db/prod"},
		{Name: "api-key", Mount: "secret", Path: "api/key"},
	}
	m.bookmarkFilter = ""

	result := m.filteredBookmarks()
	assert.Equal(t, m.bookmarks, result)
}

func TestFilteredBookmarks_WithMatch(t *testing.T) {
	m := newTestModel()
	m.bookmarks = []config.Bookmark{
		{Name: "db-prod", Mount: "secret", Path: "db/prod"},
		{Name: "api-key", Mount: "secret", Path: "api/key"},
		{Name: "db-staging", Mount: "secret", Path: "db/staging"},
	}
	m.bookmarkFilter = "db"

	result := m.filteredBookmarks()
	require.Len(t, result, 2)
	assert.Equal(t, "db-prod", result[0].Name)
	assert.Equal(t, "db-staging", result[1].Name)
}

func TestFilteredBookmarks_CaseInsensitive(t *testing.T) {
	m := newTestModel()
	m.bookmarks = []config.Bookmark{
		{Name: "DB-Prod", Mount: "secret", Path: "db/prod"},
		{Name: "api-key", Mount: "secret", Path: "api/key"},
	}
	m.bookmarkFilter = "db"

	result := m.filteredBookmarks()
	require.Len(t, result, 1)
	assert.Equal(t, "DB-Prod", result[0].Name)
}

func TestFilteredBookmarks_NoMatch(t *testing.T) {
	m := newTestModel()
	m.bookmarks = []config.Bookmark{
		{Name: "db-prod", Mount: "secret", Path: "db/prod"},
	}
	m.bookmarkFilter = "zzz"

	result := m.filteredBookmarks()
	assert.Empty(t, result)
}

func TestFilteredBookmarks_EmptyBookmarks(t *testing.T) {
	m := newTestModel()
	m.bookmarks = nil
	m.bookmarkFilter = "anything"

	result := m.filteredBookmarks()
	assert.Empty(t, result)
}

// ---------------------------------------------------------------------------
// actualIdx
// ---------------------------------------------------------------------------

func TestActualIdx_NoFilter(t *testing.T) {
	m := newTestModel()
	m.filterQuery = ""

	assert.Equal(t, 0, m.actualIdx(0))
	assert.Equal(t, 5, m.actualIdx(5))
}

func TestActualIdx_WithFilter(t *testing.T) {
	m := newTestModel()
	m.filterQuery = "x"
	m.filteredIdx = []int{2, 5, 8}

	assert.Equal(t, 2, m.actualIdx(0))
	assert.Equal(t, 5, m.actualIdx(1))
	assert.Equal(t, 8, m.actualIdx(2))
}

func TestActualIdx_OutOfBounds(t *testing.T) {
	m := newTestModel()
	m.filterQuery = "x"
	m.filteredIdx = []int{2, 5}

	// When visibleIdx exceeds filteredIdx length, falls back to visibleIdx.
	assert.Equal(t, 10, m.actualIdx(10))
}

// ---------------------------------------------------------------------------
// selectedEntries
// ---------------------------------------------------------------------------

func TestSelectedEntries(t *testing.T) {
	m := newTestModel()
	m.entries = []model.Entry{
		{Name: "a", IsDir: false},
		{Name: "b/", IsDir: true},
		{Name: "c", IsDir: false},
	}
	m.selected = map[int]bool{0: true, 2: true}

	entries := m.selectedEntries()
	assert.Len(t, entries, 2)

	names := make(map[string]bool)
	for _, e := range entries {
		names[e.Name] = true
	}
	assert.True(t, names["a"])
	assert.True(t, names["c"])
}

func TestSelectedEntries_OutOfBoundsIgnored(t *testing.T) {
	m := newTestModel()
	m.entries = []model.Entry{{Name: "a"}}
	m.selected = map[int]bool{0: true, 99: true}

	entries := m.selectedEntries()
	assert.Len(t, entries, 1)
	assert.Equal(t, "a", entries[0].Name)
}

func TestSelectedEntries_NoneSelected(t *testing.T) {
	m := newTestModel()
	m.entries = []model.Entry{{Name: "a"}}
	m.selected = make(map[int]bool)

	entries := m.selectedEntries()
	assert.Empty(t, entries)
}

// ---------------------------------------------------------------------------
// selectedVisible
// ---------------------------------------------------------------------------

func TestSelectedVisible_NoFilter(t *testing.T) {
	m := newTestModel()
	m.filterQuery = ""
	m.selected = map[int]bool{0: true, 3: true}

	vis := m.selectedVisible()
	// Without filter, returns the same map.
	assert.Equal(t, m.selected, vis)
}

func TestSelectedVisible_WithFilter(t *testing.T) {
	m := newTestModel()
	m.filterQuery = "x"
	m.filteredIdx = []int{2, 5, 8}
	m.selected = map[int]bool{5: true, 8: true}

	vis := m.selectedVisible()
	// visible index 1 -> actual 5 (selected), visible index 2 -> actual 8 (selected)
	assert.True(t, vis[1])
	assert.True(t, vis[2])
	assert.False(t, vis[0])
	assert.Len(t, vis, 2)
}

func TestSelectedVisible_FilterNoSelection(t *testing.T) {
	m := newTestModel()
	m.filterQuery = "x"
	m.filteredIdx = []int{0, 1}
	m.selected = make(map[int]bool)

	vis := m.selectedVisible()
	assert.Empty(t, vis)
}

// ---------------------------------------------------------------------------
// mountEntries
// ---------------------------------------------------------------------------

func TestMountEntries(t *testing.T) {
	m := newTestModel()
	m.mounts = []string{"secret", "kv", "pki"}

	entries := m.mountEntries()
	require.Len(t, entries, 3)

	assert.Equal(t, "secret/", entries[0].Name)
	assert.True(t, entries[0].IsDir)
	assert.Equal(t, "kv/", entries[1].Name)
	assert.True(t, entries[1].IsDir)
	assert.Equal(t, "pki/", entries[2].Name)
	assert.True(t, entries[2].IsDir)
}

func TestMountEntries_Empty(t *testing.T) {
	m := newTestModel()
	m.mounts = nil

	entries := m.mountEntries()
	assert.Empty(t, entries)
}

// ---------------------------------------------------------------------------
// Layout math: headerLineCount, explorerColHeight, explorerScrollOffset,
// explorerColumnBounds, explorerHalfPage, secretPopupHalfPage, helpVisibleLines
// ---------------------------------------------------------------------------

func TestHeaderLineCount(t *testing.T) {
	t.Run("single tab", func(t *testing.T) {
		m := newTestModel()
		// Default: 1 tab
		assert.Equal(t, 1, m.headerLineCount())
	})

	t.Run("multiple tabs", func(t *testing.T) {
		m := newTestModel()
		m.tabs = append(m.tabs, TabState{
			selected:     make(map[int]bool),
			cursorMemory: make(map[string]int),
			mount:        "kv",
		})
		assert.Equal(t, 2, m.headerLineCount())
	})
}

func TestExplorerColHeight(t *testing.T) {
	tests := []struct {
		name   string
		height int
		nTabs  int
		want   int
	}{
		{name: "normal height single tab", height: 40, nTabs: 1, want: 40 - 5 - 1},
		{name: "normal height multi tab", height: 40, nTabs: 2, want: 40 - 5 - 2},
		{name: "very small height", height: 4, nTabs: 1, want: 1},
		{name: "zero height", height: 0, nTabs: 1, want: 1},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := newTestModel()
			m.height = tc.height
			for i := 1; i < tc.nTabs; i++ {
				m.tabs = append(m.tabs, TabState{
					selected:     make(map[int]bool),
					cursorMemory: make(map[string]int),
					mount:        "kv",
				})
			}
			assert.Equal(t, tc.want, m.explorerColHeight())
		})
	}
}

func TestExplorerScrollOffset(t *testing.T) {
	m := newTestModel()
	m.height = 20

	colH := m.explorerColHeight()

	t.Run("cursor within first page", func(t *testing.T) {
		m.cursor = 0
		assert.Equal(t, 0, m.explorerScrollOffset())
	})

	t.Run("cursor at boundary", func(t *testing.T) {
		m.cursor = colH - 1
		assert.Equal(t, 0, m.explorerScrollOffset())
	})

	t.Run("cursor past first page", func(t *testing.T) {
		m.cursor = colH + 5
		assert.Equal(t, 6, m.explorerScrollOffset())
	})
}

func TestExplorerColumnBounds(t *testing.T) {
	tests := []struct {
		name         string
		width        int
		wantLeftEnd  int
		wantMidEnd   int
		leftPositive bool
		midGtLeft    bool
	}{
		{
			name:         "typical terminal width",
			width:        120,
			leftPositive: true,
			midGtLeft:    true,
		},
		{
			name:         "narrow terminal",
			width:        40,
			leftPositive: true,
			midGtLeft:    true,
		},
		{
			name:         "very narrow terminal (minimums apply)",
			width:        20,
			leftPositive: true,
			midGtLeft:    true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := newTestModel()
			m.width = tc.width
			leftEnd, midEnd := m.explorerColumnBounds()

			if tc.leftPositive {
				assert.Greater(t, leftEnd, 0)
			}
			if tc.midGtLeft {
				assert.Greater(t, midEnd, leftEnd)
			}
		})
	}

	t.Run("specific calculation for known width", func(t *testing.T) {
		m := newTestModel()
		m.width = 106 // usable = 100
		leftEnd, midEnd := m.explorerColumnBounds()
		// leftW = 100*12/100 = 12, midW = 100*51/100 = 51
		// leftEnd = 12 + 2 = 14, midEnd = 14 + 51 + 2 = 67
		assert.Equal(t, 14, leftEnd)
		assert.Equal(t, 67, midEnd)
	})
}

func TestExplorerHalfPage(t *testing.T) {
	tests := []struct {
		name   string
		height int
		want   int
	}{
		{name: "normal height", height: 40, want: 19},
		{name: "small height", height: 4, want: 1},
		{name: "very small", height: 1, want: 1},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := newTestModel()
			m.height = tc.height
			assert.Equal(t, tc.want, m.explorerHalfPage())
		})
	}
}

func TestSecretPopupHalfPage(t *testing.T) {
	t.Run("normal height", func(t *testing.T) {
		m := newTestModel()
		m.height = 40
		half := m.secretPopupHalfPage()
		assert.Greater(t, half, 0)
	})

	t.Run("small height", func(t *testing.T) {
		m := newTestModel()
		m.height = 5
		half := m.secretPopupHalfPage()
		assert.GreaterOrEqual(t, half, 1)
	})
}

func TestHelpVisibleLines(t *testing.T) {
	t.Run("normal height", func(t *testing.T) {
		m := newTestModel()
		m.height = 50
		lines := m.helpVisibleLines()
		assert.Greater(t, lines, 5)
	})

	t.Run("small height uses minimum", func(t *testing.T) {
		m := newTestModel()
		m.height = 5
		lines := m.helpVisibleLines()
		assert.GreaterOrEqual(t, lines, 5)
	})
}

// ---------------------------------------------------------------------------
// currentPath / parentPath
// ---------------------------------------------------------------------------

func TestCurrentPath(t *testing.T) {
	tests := []struct {
		name string
		path []string
		want string
	}{
		{name: "root", path: nil, want: ""},
		{name: "one level", path: []string{"dir/"}, want: "dir/"},
		{name: "two levels", path: []string{"a/", "b/"}, want: "a/b/"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := newTestModel()
			m.path = tc.path
			assert.Equal(t, tc.want, m.currentPath())
		})
	}
}

func TestParentPath(t *testing.T) {
	tests := []struct {
		name string
		path []string
		want string
	}{
		{name: "root", path: nil, want: ""},
		{name: "one level", path: []string{"dir/"}, want: ""},
		{name: "two levels", path: []string{"a/", "b/"}, want: "a/"},
		{name: "three levels", path: []string{"a/", "b/", "c/"}, want: "a/b/"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := newTestModel()
			m.path = tc.path
			assert.Equal(t, tc.want, m.parentPath())
		})
	}
}

// ---------------------------------------------------------------------------
// handleMarkSave (validation paths only -- no vault client calls)
// ---------------------------------------------------------------------------

func TestHandleMarkSave_EscCancels(t *testing.T) {
	m := newTestModel()
	m.status = "Set mark: [a-z, 0-9]"
	m.lastMarkAttempt = "a"

	_, _ = m.handleMarkSave("esc")

	assert.Equal(t, "", m.status)
	assert.Equal(t, "", m.lastMarkAttempt)
}

func TestHandleMarkSave_InvalidKey(t *testing.T) {
	tests := []struct {
		name string
		key  string
	}{
		{name: "uppercase", key: "A"},
		{name: "special char", key: "!"},
		{name: "multi-char", key: "ab"},
		{name: "space", key: " "},
		{name: "ctrl sequence", key: "ctrl+a"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := newTestModel()
			_, _ = m.handleMarkSave(tc.key)
			assert.Equal(t, "Invalid mark key (use a-z, 0-9)", m.status)
			assert.Equal(t, "", m.lastMarkAttempt)
		})
	}
}

func TestHandleMarkSave_DuplicateSlotSameLocation(t *testing.T) {
	m := newTestModel()
	m.path = nil // root
	m.bookmarks = []config.Bookmark{
		{Name: "secret/", Mount: "secret", Path: "", Slot: "a"},
	}

	_, _ = m.handleMarkSave("a")

	assert.Equal(t, "Mark 'a' already set to this location", m.status)
	assert.Equal(t, "", m.lastMarkAttempt)
}

func TestHandleMarkSave_DuplicateSlotDifferentLocation_FirstAttempt(t *testing.T) {
	m := newTestModel()
	m.path = []string{"new/"}
	m.bookmarks = []config.Bookmark{
		{Name: "secret/old/", Mount: "secret", Path: "old/", Slot: "a"},
	}
	m.lastMarkAttempt = ""

	_, _ = m.handleMarkSave("a")

	// First attempt should warn and set lastMarkAttempt.
	assert.Contains(t, m.status, "already set")
	assert.Contains(t, m.status, "press m+a again")
	assert.Equal(t, "a", m.lastMarkAttempt)
}

// ---------------------------------------------------------------------------
// leftPaneEntries
// ---------------------------------------------------------------------------

func TestLeftPaneEntries_AtRoot(t *testing.T) {
	m := newTestModel()
	m.path = nil
	m.mounts = []string{"secret", "kv"}
	m.parentList = []model.Entry{{Name: "should-not-appear"}}

	entries := m.leftPaneEntries()
	require.Len(t, entries, 2)
	assert.Equal(t, "secret/", entries[0].Name)
	assert.True(t, entries[0].IsDir)
	assert.Equal(t, "kv/", entries[1].Name)
}

func TestLeftPaneEntries_NonRoot(t *testing.T) {
	m := newTestModel()
	m.path = []string{"dir/"}
	m.parentList = []model.Entry{{Name: "parent-item"}}

	entries := m.leftPaneEntries()
	require.Len(t, entries, 1)
	assert.Equal(t, "parent-item", entries[0].Name)
}

// ---------------------------------------------------------------------------
// Copy helpers: copyMap, copySlice, copyMapStringInt, copyMapIntBool
// ---------------------------------------------------------------------------

func TestCopyMap(t *testing.T) {
	src := map[string]string{"a": "1", "b": "2"}
	dst := copyMap(src)

	assert.Equal(t, src, dst)

	// Mutation of src should not affect dst.
	src["a"] = "changed"
	assert.Equal(t, "1", dst["a"])
}

func TestCopyMap_Nil(t *testing.T) {
	var src map[string]string
	dst := copyMap(src)
	assert.NotNil(t, dst)
	assert.Empty(t, dst)
}

func TestCopySlice(t *testing.T) {
	src := []string{"a", "b", "c"}
	dst := copySlice(src)

	assert.Equal(t, src, dst)

	// Mutation of src should not affect dst.
	src[0] = "changed"
	assert.Equal(t, "a", dst[0])
}

func TestCopySlice_Nil(t *testing.T) {
	var src []string
	dst := copySlice(src)
	assert.NotNil(t, dst)
	assert.Empty(t, dst)
}

func TestCopyMapStringInt(t *testing.T) {
	t.Run("non-nil", func(t *testing.T) {
		src := map[string]int{"x": 1, "y": 2}
		dst := copyMapStringInt(src)
		assert.Equal(t, src, dst)

		src["x"] = 99
		assert.Equal(t, 1, dst["x"])
	})

	t.Run("nil returns nil", func(t *testing.T) {
		assert.Nil(t, copyMapStringInt(nil))
	})
}

func TestCopyMapIntBool(t *testing.T) {
	t.Run("non-nil", func(t *testing.T) {
		src := map[int]bool{1: true, 2: false}
		dst := copyMapIntBool(src)
		assert.Equal(t, src, dst)

		src[1] = false
		assert.True(t, dst[1])
	})

	t.Run("nil returns nil", func(t *testing.T) {
		assert.Nil(t, copyMapIntBool(nil))
	})
}

// ---------------------------------------------------------------------------
// removeLastEmptyKey
// ---------------------------------------------------------------------------

func TestRemoveLastEmptyKey(t *testing.T) {
	m := newTestModel()
	m.secret = &model.Secret{
		Path: "test",
		Keys: []string{"a", "", "b", ""},
		Data: map[string]string{"a": "1", "": "", "b": "2"},
	}

	result := m.removeLastEmptyKey()
	assert.Equal(t, []string{"a", "", "b"}, result)
}

func TestRemoveLastEmptyKey_NoEmptyKey(t *testing.T) {
	m := newTestModel()
	m.secret = &model.Secret{
		Path: "test",
		Keys: []string{"a", "b"},
		Data: map[string]string{"a": "1", "b": "2"},
	}

	result := m.removeLastEmptyKey()
	assert.Equal(t, []string{"a", "b"}, result)
}

func TestRemoveLastEmptyKey_SingleEmpty(t *testing.T) {
	m := newTestModel()
	m.secret = &model.Secret{
		Path: "test",
		Keys: []string{""},
		Data: map[string]string{"": ""},
	}

	result := m.removeLastEmptyKey()
	assert.Empty(t, result)
}

// ---------------------------------------------------------------------------
// applyFilter + visibleEntries integration
// ---------------------------------------------------------------------------

func TestApplyFilter_ThenVisibleEntries(t *testing.T) {
	m := newTestModel()
	m.entries = []model.Entry{
		{Name: "secret-prod", IsDir: false},
		{Name: "config/", IsDir: true},
		{Name: "secret-staging", IsDir: false},
		{Name: "notes", IsDir: false},
	}

	m.filterQuery = "secret"
	m.applyFilter()

	vis := m.visibleEntries()
	require.Len(t, vis, 2)
	assert.Equal(t, "secret-prod", vis[0].Name)
	assert.Equal(t, "secret-staging", vis[1].Name)
}

// ---------------------------------------------------------------------------
// Tab state: save/load preserves mount across tabs
// ---------------------------------------------------------------------------

func TestSaveAndLoadTab_MountField(t *testing.T) {
	m := newTestModel()
	m.saveCurrentTab()

	// Verify mount was saved from client.
	assert.Equal(t, "secret", m.tabs[0].mount)
}

// ---------------------------------------------------------------------------
// Multiple tabs: load different tab restores state
// ---------------------------------------------------------------------------

func TestMultipleTabsSwitching(t *testing.T) {
	m := newTestModel()

	// Setup tab 0.
	m.path = []string{"tab0-dir/"}
	m.entries = []model.Entry{{Name: "tab0-entry"}}
	m.cursor = 0
	m.saveCurrentTab()

	// Create tab 1.
	m.tabs = append(m.tabs, TabState{
		path:         []string{"tab1-dir/"},
		entries:      []model.Entry{{Name: "tab1-entry-a"}, {Name: "tab1-entry-b"}},
		cursor:       1,
		selected:     make(map[int]bool),
		cursorMemory: make(map[string]int),
		mount:        "secret",
	})

	// Switch to tab 1.
	m.loadTab(1)
	assert.Equal(t, 1, m.activeTab)
	assert.Equal(t, []string{"tab1-dir/"}, m.path)
	assert.Equal(t, 1, m.cursor)
	assert.Len(t, m.entries, 2)

	// Switch back to tab 0.
	m.loadTab(0)
	assert.Equal(t, 0, m.activeTab)
	assert.Equal(t, []string{"tab0-dir/"}, m.path)
	assert.Equal(t, 0, m.cursor)
	assert.Len(t, m.entries, 1)
}

// ---------------------------------------------------------------------------
// Edge cases for explorerColumnBounds with minimum values
// ---------------------------------------------------------------------------

func TestExplorerColumnBounds_MinimumWidths(t *testing.T) {
	m := newTestModel()
	// Width so small that computed leftW and midW would be < 10.
	m.width = 10
	leftEnd, midEnd := m.explorerColumnBounds()

	// Both should use minimums: leftW=10, midW=10
	// leftEnd = 10+2 = 12, midEnd = 12+10+2 = 24
	assert.Equal(t, 12, leftEnd)
	assert.Equal(t, 24, midEnd)
}

// ---------------------------------------------------------------------------
// visibleEntries with empty filter and empty entries
// ---------------------------------------------------------------------------

func TestVisibleEntries_EmptyFilterNonEmptyEntries(t *testing.T) {
	m := newTestModel()
	m.entries = []model.Entry{{Name: "a"}, {Name: "b"}}
	m.filterQuery = ""
	m.filteredIdx = nil

	vis := m.visibleEntries()
	assert.Len(t, vis, 2)
}

// ---------------------------------------------------------------------------
// selectedEntry returns a copy (pointer to copy)
// ---------------------------------------------------------------------------

func TestSelectedEntry_ReturnsCopy(t *testing.T) {
	m := newTestModel()
	m.entries = []model.Entry{{Name: "original", IsDir: false}}
	m.cursor = 0

	e := m.selectedEntry()
	require.NotNil(t, e)
	e.Name = "modified"

	// Original should be unchanged.
	assert.Equal(t, "original", m.entries[0].Name)
}

// ---------------------------------------------------------------------------
// actualIdx: filter empty
// ---------------------------------------------------------------------------

func TestActualIdx_EmptyFilteredIdx(t *testing.T) {
	m := newTestModel()
	m.filterQuery = "active"
	m.filteredIdx = []int{} // empty but filter is active

	// Falls back to visibleIdx since visibleIdx >= len(filteredIdx).
	assert.Equal(t, 3, m.actualIdx(3))
}

// ---------------------------------------------------------------------------
// pushUndo / pushRedo
// ---------------------------------------------------------------------------

func TestPushUndo(t *testing.T) {
	m := newTestModel()
	m.redoStack = []model.UndoAction{{Description: "old-redo"}}

	action := model.UndoAction{Description: "edit", Path: "secret/foo"}
	m.pushUndo(action)

	require.Len(t, m.undoStack, 1)
	assert.Equal(t, "edit", m.undoStack[0].Description)
	// pushUndo clears redo stack.
	assert.Empty(t, m.redoStack)
}

func TestPushUndo_Accumulates(t *testing.T) {
	m := newTestModel()
	m.pushUndo(model.UndoAction{Description: "first"})
	m.pushUndo(model.UndoAction{Description: "second"})

	require.Len(t, m.undoStack, 2)
	assert.Equal(t, "first", m.undoStack[0].Description)
	assert.Equal(t, "second", m.undoStack[1].Description)
}

func TestPushRedo(t *testing.T) {
	m := newTestModel()

	action := model.UndoAction{Description: "redo-action"}
	m.pushRedo(action)

	require.Len(t, m.redoStack, 1)
	assert.Equal(t, "redo-action", m.redoStack[0].Description)
}

func TestPushRedo_DoesNotClearUndo(t *testing.T) {
	m := newTestModel()
	m.undoStack = []model.UndoAction{{Description: "existing-undo"}}

	m.pushRedo(model.UndoAction{Description: "redo"})

	// undo stack should be untouched.
	require.Len(t, m.undoStack, 1)
	assert.Equal(t, "existing-undo", m.undoStack[0].Description)
}

// ---------------------------------------------------------------------------
// enterConfirmMode
// ---------------------------------------------------------------------------

func TestEnterConfirmMode(t *testing.T) {
	m := newTestModel()
	ci := textinput.New()
	ci.Placeholder = "DELETE"
	m.confirmInput = ci
	m.confirmInput.SetValue("leftover")

	m.enterConfirmMode()

	assert.Equal(t, model.ModeConfirm, m.mode)
	assert.Equal(t, "", m.confirmInput.Value())
}

// ---------------------------------------------------------------------------
// jumpToMatch
// ---------------------------------------------------------------------------

func TestJumpToMatch_FindsFirst(t *testing.T) {
	m := newTestModel()
	m.entries = []model.Entry{
		{Name: "alpha"},
		{Name: "beta"},
		{Name: "gamma"},
	}
	m.priorCursor = 2

	m.jumpToMatch("bet")
	assert.Equal(t, 1, m.cursor)
}

func TestJumpToMatch_CaseInsensitive(t *testing.T) {
	m := newTestModel()
	m.entries = []model.Entry{
		{Name: "Alpha"},
		{Name: "BETA"},
	}

	m.jumpToMatch("beta")
	assert.Equal(t, 1, m.cursor)
}

func TestJumpToMatch_EmptyQuery(t *testing.T) {
	m := newTestModel()
	m.entries = []model.Entry{{Name: "a"}, {Name: "b"}}
	m.priorCursor = 1
	m.cursor = 0

	m.jumpToMatch("")
	// Should restore priorCursor.
	assert.Equal(t, 1, m.cursor)
}

func TestJumpToMatch_NoMatch(t *testing.T) {
	m := newTestModel()
	m.entries = []model.Entry{{Name: "alpha"}, {Name: "beta"}}
	m.cursor = 1

	m.jumpToMatch("zzz")
	// Cursor should remain unchanged when no match found.
	assert.Equal(t, 1, m.cursor)
}

func TestJumpToMatch_UsesVisibleEntries(t *testing.T) {
	m := newTestModel()
	m.entries = []model.Entry{
		{Name: "alpha"},
		{Name: "beta"},
		{Name: "gamma"},
	}
	m.filterQuery = "a"
	m.filteredIdx = []int{0, 2} // alpha, gamma

	m.jumpToMatch("gam")
	// gamma is at visible index 1.
	assert.Equal(t, 1, m.cursor)
}

// ---------------------------------------------------------------------------
// themeMoveCursor
// ---------------------------------------------------------------------------

func TestThemeMoveCursor_SkipsHeaders(t *testing.T) {
	m := newTestModel()
	m.themeEntries = []ui.ThemeEntry{
		{Name: "Dark Themes", IsHeader: true},
		{Name: "monokai", IsHeader: false},
		{Name: "dracula", IsHeader: false},
		{Name: "Light Themes", IsHeader: true},
		{Name: "solarized-light", IsHeader: false},
	}
	m.themeCursor = 1 // monokai

	m.themeMoveCursor(1)
	assert.Equal(t, 2, m.themeCursor) // dracula

	m.themeMoveCursor(1)
	assert.Equal(t, 4, m.themeCursor) // solarized-light (skipped header at 3)
}

func TestThemeMoveCursor_StopsAtBoundary(t *testing.T) {
	m := newTestModel()
	m.themeEntries = []ui.ThemeEntry{
		{Name: "only", IsHeader: false},
	}
	m.themeCursor = 0

	m.themeMoveCursor(-1)
	assert.Equal(t, 0, m.themeCursor) // can't go further

	m.themeMoveCursor(1)
	assert.Equal(t, 0, m.themeCursor) // can't go further
}

func TestThemeMoveCursor_BackwardsSkipsHeaders(t *testing.T) {
	m := newTestModel()
	m.themeEntries = []ui.ThemeEntry{
		{Name: "first", IsHeader: false},
		{Name: "Header", IsHeader: true},
		{Name: "second", IsHeader: false},
	}
	m.themeCursor = 2 // second

	m.themeMoveCursor(-1)
	assert.Equal(t, 0, m.themeCursor) // first (skipped header at 1)
}

// ---------------------------------------------------------------------------
// applyCurrentEditInput
// ---------------------------------------------------------------------------

func TestApplyCurrentEditInput_EditKey(t *testing.T) {
	m := newTestModel()
	m.secret = &model.Secret{
		Path: "test",
		Keys: []string{"old_key"},
		Data: map[string]string{"old_key": "value1"},
	}
	m.secretEditKey = "old_key"
	m.secretEditColumn = 0 // editing key column
	m.textInput.SetValue("new_key")

	m.applyCurrentEditInput()

	assert.Equal(t, "new_key", m.secretEditKey)
	assert.Equal(t, []string{"new_key"}, m.secret.Keys)
	assert.Equal(t, "value1", m.secret.Data["new_key"])
	_, oldExists := m.secret.Data["old_key"]
	assert.False(t, oldExists)
}

func TestApplyCurrentEditInput_EditValue(t *testing.T) {
	m := newTestModel()
	m.secret = &model.Secret{
		Path: "test",
		Keys: []string{"key"},
		Data: map[string]string{"key": "old_value"},
	}
	m.secretEditKey = "key"
	m.secretEditColumn = 1 // editing value column
	m.textInput.SetValue("new_value")

	m.applyCurrentEditInput()

	assert.Equal(t, "new_value", m.secret.Data["key"])
	// Key name should be unchanged.
	assert.Equal(t, []string{"key"}, m.secret.Keys)
}

func TestApplyCurrentEditInput_KeyUnchanged(t *testing.T) {
	m := newTestModel()
	m.secret = &model.Secret{
		Path: "test",
		Keys: []string{"same_key"},
		Data: map[string]string{"same_key": "val"},
	}
	m.secretEditKey = "same_key"
	m.secretEditColumn = 0
	m.textInput.SetValue("same_key") // no change

	m.applyCurrentEditInput()

	// Nothing should change.
	assert.Equal(t, "same_key", m.secretEditKey)
	assert.Equal(t, []string{"same_key"}, m.secret.Keys)
}

// ---------------------------------------------------------------------------
// leftPaneSelectedIdx
// ---------------------------------------------------------------------------

func TestLeftPaneSelectedIdx_AtRoot(t *testing.T) {
	m := newTestModel()
	m.path = nil
	m.mounts = []string{"kv", "secret", "pki"}

	// The function checks m.client.Mount() against mounts.
	idx := m.leftPaneSelectedIdx()
	// "secret" is at index 1.
	assert.Equal(t, 1, idx)
}

func TestLeftPaneSelectedIdx_AtRoot_NoMatch(t *testing.T) {
	m := newTestModel()
	m.path = nil
	m.mounts = []string{"kv", "pki"}

	// "secret" is not in mounts, so it falls back to 0.
	idx := m.leftPaneSelectedIdx()
	assert.Equal(t, 0, idx)
}

func TestLeftPaneSelectedIdx_NonRoot_WithMemory(t *testing.T) {
	m := newTestModel()
	m.path = []string{"a/", "b/"}
	m.cursorMemory = map[string]int{"a/": 3}

	idx := m.leftPaneSelectedIdx()
	assert.Equal(t, 3, idx)
}

func TestLeftPaneSelectedIdx_NonRoot_NoMemory(t *testing.T) {
	m := newTestModel()
	m.path = []string{"a/", "b/"}
	m.cursorMemory = make(map[string]int)

	idx := m.leftPaneSelectedIdx()
	assert.Equal(t, 0, idx)
}

// ---------------------------------------------------------------------------
// helpVisibleLines edge cases
// ---------------------------------------------------------------------------

func TestHelpVisibleLines_ExactMinimum(t *testing.T) {
	m := newTestModel()
	// When boxH=20, maxLines=20-6=14
	// height * 80/100 = 20 => height = 25
	m.height = 25
	lines := m.helpVisibleLines()
	assert.Equal(t, 14, lines)
}

// ---------------------------------------------------------------------------
// secretPopupHalfPage edge cases
// ---------------------------------------------------------------------------

func TestSecretPopupHalfPage_VeryTallTerminal(t *testing.T) {
	m := newTestModel()
	m.height = 200
	half := m.secretPopupHalfPage()
	// Should be a reasonable positive value
	assert.Greater(t, half, 10)
}

// ---------------------------------------------------------------------------
// explorerColHeight edge cases
// ---------------------------------------------------------------------------

func TestExplorerColHeight_NegativeResult(t *testing.T) {
	m := newTestModel()
	// height - 5 - 1 = -2, clamps to 1
	m.height = 4
	assert.Equal(t, 1, m.explorerColHeight())
}

// ---------------------------------------------------------------------------
// explorerHalfPage edge cases
// ---------------------------------------------------------------------------

func TestExplorerHalfPage_ZeroHeight(t *testing.T) {
	m := newTestModel()
	m.height = 0
	// (0 - 2) / 2 = -1, clamped to 1
	assert.Equal(t, 1, m.explorerHalfPage())
}

// ---------------------------------------------------------------------------
// Integration: applyFilter + selectedVisible + actualIdx
// ---------------------------------------------------------------------------

func TestFilterAndSelection_Integration(t *testing.T) {
	m := newTestModel()
	m.entries = []model.Entry{
		{Name: "alpha"},   // 0
		{Name: "beta"},    // 1
		{Name: "gamma"},   // 2
		{Name: "delta"},   // 3
		{Name: "epsilon"}, // 4
	}

	// Filter to entries containing "a": alpha(0), beta(1), gamma(2), delta(3)
	m.filterQuery = "a"
	m.applyFilter()

	// Select actual indices 1 (beta) and 3 (delta).
	m.selected = map[int]bool{1: true, 3: true}

	// Verify visible selection mapping.
	vis := m.selectedVisible()
	// beta is filteredIdx[1]=1, delta is filteredIdx[3]=3
	assert.True(t, vis[1])  // visible index 1 -> actual 1
	assert.True(t, vis[3])  // visible index 3 -> actual 3
	assert.False(t, vis[0]) // alpha not selected
	assert.False(t, vis[2]) // gamma not selected

	// Verify actualIdx mapping.
	assert.Equal(t, 0, m.actualIdx(0)) // visible 0 -> alpha (actual 0)
	assert.Equal(t, 1, m.actualIdx(1)) // visible 1 -> beta (actual 1)
	assert.Equal(t, 2, m.actualIdx(2)) // visible 2 -> gamma (actual 2)
	assert.Equal(t, 3, m.actualIdx(3)) // visible 3 -> delta (actual 3)
}

// ---------------------------------------------------------------------------
// Integration: full tab workflow with filter state
// ---------------------------------------------------------------------------

func TestTabWorkflow_FilterStatePersists(t *testing.T) {
	m := newTestModel()
	m.entries = []model.Entry{
		{Name: "alpha"},
		{Name: "beta"},
		{Name: "gamma"},
	}
	m.filterQuery = "al"
	m.applyFilter()
	m.cursor = 0

	// Save tab 0 state.
	m.saveCurrentTab()

	// Modify state (simulating tab switch).
	m.entries = []model.Entry{{Name: "other"}}
	m.filterQuery = ""
	m.filteredIdx = nil
	m.cursor = 0

	// Restore tab 0.
	m.loadTab(0)

	assert.Equal(t, "al", m.filterQuery)
	assert.Equal(t, []int{0}, m.filteredIdx)

	vis := m.visibleEntries()
	require.Len(t, vis, 1)
	assert.Equal(t, "alpha", vis[0].Name)
}
