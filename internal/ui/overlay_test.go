package ui

import (
	"strings"
	"testing"

	"github.com/janosmiko/vau/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestTakeVisualWidth(t *testing.T) {
	tests := []struct {
		name string
		s    string
		n    int
		want string
	}{
		{
			name: "ASCII prefix",
			s:    "hello world",
			n:    5,
			want: "hello",
		},
		{
			name: "full string when n equals length",
			s:    "abc",
			n:    3,
			want: "abc",
		},
		{
			name: "pads when string shorter than n",
			s:    "hi",
			n:    5,
			want: "hi   ",
		},
		{
			name: "empty string pads",
			s:    "",
			n:    3,
			want: "   ",
		},
		{
			name: "n equals zero",
			s:    "hello",
			n:    0,
			want: "",
		},
		{
			name: "n is negative",
			s:    "hello",
			n:    -1,
			want: "",
		},
		{
			name: "single character",
			s:    "x",
			n:    1,
			want: "x",
		},
		{
			name: "n larger than string",
			s:    "ab",
			n:    10,
			want: "ab        ",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := takeVisualWidth(tc.s, tc.n)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestTakeVisualWidthWithANSI(t *testing.T) {
	// ANSI sequence should be preserved but not counted as visual width.
	s := "\033[31mhello\033[0m"
	got := takeVisualWidth(s, 3)
	// Should contain the opening ANSI escape and first 3 chars.
	assert.Contains(t, got, "\033[31m")
	assert.Contains(t, got, "hel")
}

func TestSkipVisualWidth(t *testing.T) {
	tests := []struct {
		name string
		s    string
		n    int
		want string
	}{
		{
			name: "skip prefix",
			s:    "hello world",
			n:    6,
			want: "world",
		},
		{
			name: "skip past end",
			s:    "abc",
			n:    10,
			want: "",
		},
		{
			name: "skip zero skips first char",
			s:    "hello",
			n:    0,
			want: "ello",
		},
		{
			name: "empty string",
			s:    "",
			n:    5,
			want: "",
		},
		{
			name: "skip exact length",
			s:    "abc",
			n:    3,
			want: "",
		},
		{
			name: "skip one from single char",
			s:    "x",
			n:    1,
			want: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := skipVisualWidth(tc.s, tc.n)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestRenderBookmarkOverlayEmpty(t *testing.T) {
	result := RenderBookmarkOverlay(nil, "", false, 0, 80, 40)
	assert.Contains(t, result, "No bookmarks yet")
	assert.Contains(t, result, "Marks")
}

func TestRenderBookmarkOverlayWithBookmarks(t *testing.T) {
	bookmarks := []config.Bookmark{
		{Name: "prod-secrets", Mount: "secret", Path: "prod/app"},
		{Name: "dev-db", Mount: "kv", Path: "dev/database"},
	}
	result := RenderBookmarkOverlay(bookmarks, "", false, 0, 80, 40)
	assert.Contains(t, result, "prod-secrets")
	assert.Contains(t, result, "dev-db")
	assert.Contains(t, result, "Marks")
	assert.NotContains(t, result, "No bookmarks yet")
}

func TestRenderBookmarkOverlayWithFilter(t *testing.T) {
	bookmarks := []config.Bookmark{
		{Name: "prod-secrets", Mount: "secret", Path: "prod/app"},
		{Name: "dev-db", Mount: "kv", Path: "dev/database"},
	}

	// Filter active (not searching mode).
	result := RenderBookmarkOverlay(bookmarks, "prod", false, 0, 80, 40)
	assert.Contains(t, result, "prod-secrets")
	assert.Contains(t, result, "prod") // The filter text should appear.

	// Searching mode shows cursor.
	resultSearching := RenderBookmarkOverlay(bookmarks, "dev", true, 0, 80, 40)
	assert.Contains(t, resultSearching, "dev")
}

func TestRenderBookmarkOverlayFilterNoMatch(t *testing.T) {
	bookmarks := []config.Bookmark{
		{Name: "prod-secrets", Mount: "secret", Path: "prod/app"},
	}
	result := RenderBookmarkOverlay(bookmarks, "zzz", false, 0, 80, 40)
	assert.Contains(t, result, "No matching bookmarks")
}

func TestRenderBookmarkOverlayWithSlots(t *testing.T) {
	bookmarks := []config.Bookmark{
		{Name: "secret/prod/db/", Mount: "secret", Path: "prod/db/", Slot: "a"},
		{Name: "secret/staging/", Mount: "secret", Path: "staging/", Slot: "1"},
		{Name: "kv/legacy", Mount: "kv", Path: "legacy"},
	}
	result := RenderBookmarkOverlay(bookmarks, "", false, 0, 80, 40)
	// Slotted bookmarks should show the slot key.
	assert.Contains(t, result, "a")
	assert.Contains(t, result, "1")
	assert.Contains(t, result, "secret/prod/db/")
	assert.Contains(t, result, "secret/staging/")
	assert.Contains(t, result, "kv/legacy")
}

func TestRenderThemePickerOverlayWithEntries(t *testing.T) {
	entries := []ThemeEntry{
		{Name: "Dark Themes", IsHeader: true},
		{Name: "tokyonight"},
		{Name: "nord"},
		{Name: "Light Themes", IsHeader: true},
		{Name: "gruvbox-light"},
	}
	result := RenderThemePickerOverlay(entries, 1, "tokyonight", 80, 40)
	assert.Contains(t, result, "Colorscheme")
	assert.Contains(t, result, "tokyonight")
	assert.Contains(t, result, "nord")
	assert.Contains(t, result, "gruvbox-light")
	// Active theme marker.
	assert.Contains(t, result, "*")
}

func TestRenderThemePickerOverlayEmpty(t *testing.T) {
	result := RenderThemePickerOverlay(nil, 0, "", 80, 40)
	assert.Contains(t, result, "No themes available")
}

func TestRenderThemePickerOverlayActiveThemeMarker(t *testing.T) {
	entries := []ThemeEntry{
		{Name: "Dark Themes", IsHeader: true},
		{Name: "tokyonight"},
		{Name: "dracula"},
	}

	// When activeTheme matches an entry, the marker should appear.
	result := RenderThemePickerOverlay(entries, 1, "dracula", 80, 40)
	// Split into lines to check the dracula line has the marker.
	lines := strings.Split(result, "\n")
	found := false
	for _, line := range lines {
		if strings.Contains(line, "dracula") && strings.Contains(line, "*") {
			found = true
			break
		}
	}
	assert.True(t, found, "dracula entry should have active marker")

	// tokyonight should not have the marker.
	for _, line := range lines {
		if strings.Contains(line, "tokyonight") {
			assert.NotContains(t, line, "*",
				"tokyonight should not have active marker when dracula is active")
		}
	}
}

// ---------------------------------------------------------------------------
// RenderSearchOverlay
// ---------------------------------------------------------------------------

func TestRenderSearchOverlay(t *testing.T) {
	result := RenderSearchOverlay("Search:", "my-query", 100)
	assert.Contains(t, result, "Search:")
	assert.Contains(t, result, "my-query")
}

func TestRenderSearchOverlayEmptyInput(t *testing.T) {
	result := RenderSearchOverlay("Filter entries:", "", 100)
	assert.Contains(t, result, "Filter entries:")
}

func TestRenderSearchOverlayNarrowWidth(t *testing.T) {
	// Width < 40 should be clamped to minimum 40
	result := RenderSearchOverlay("Search:", "test", 30)
	assert.NotEmpty(t, result)
	assert.Contains(t, result, "Search:")
	assert.Contains(t, result, "test")
}

func TestRenderSearchOverlayWideWidth(t *testing.T) {
	// Width large enough that box can render properly
	result := RenderSearchOverlay("Find:", "pattern", 100)
	assert.NotEmpty(t, result)
	assert.Contains(t, result, "Find:")
}

// ---------------------------------------------------------------------------
// RenderInputOverlay
// ---------------------------------------------------------------------------

func TestRenderInputOverlay(t *testing.T) {
	result := RenderInputOverlay("Enter name:", "my-input", 100)
	assert.Contains(t, result, "Enter name:")
	assert.Contains(t, result, "my-input")
}

func TestRenderInputOverlayMinWidth(t *testing.T) {
	result := RenderInputOverlay("Label:", "val", 20)
	assert.NotEmpty(t, result)
	assert.Contains(t, result, "Label:")
}

// ---------------------------------------------------------------------------
// RenderConfirmOverlay
// ---------------------------------------------------------------------------

func TestRenderConfirmOverlay(t *testing.T) {
	result := RenderConfirmOverlay("Are you sure?", "DEL", 100)
	assert.Contains(t, result, "Are you sure?")
	assert.Contains(t, result, "DELETE")
	assert.Contains(t, result, "DEL")
	assert.Contains(t, result, "confirm")
	assert.Contains(t, result, "cancel")
}

// ---------------------------------------------------------------------------
// PlaceOverlay
// ---------------------------------------------------------------------------

func TestPlaceOverlayBasic(t *testing.T) {
	bg := "aaaaaaaaaa\naaaaaaaaaa\naaaaaaaaaa\naaaaaaaaaa\naaaaaaaaaa"
	overlay := "XX\nXX"
	result := PlaceOverlay(bg, overlay, 10, 5)
	assert.Contains(t, result, "XX")
	// Background lines should still be present
	assert.Contains(t, result, "a")
}

func TestPlaceOverlayEmptyBackground(t *testing.T) {
	overlay := "hi"
	result := PlaceOverlay("", overlay, 10, 3)
	assert.Contains(t, result, "hi")
}

// ---------------------------------------------------------------------------
// skipVisualWidth (additional edge cases)
// ---------------------------------------------------------------------------

func TestSkipVisualWidthWithANSI(t *testing.T) {
	// ANSI sequences after the skip point should be preserved.
	s := "abc\033[31mdef\033[0m"
	got := skipVisualWidth(s, 3)
	// After skipping 3 visible chars ("abc"), we should get the ANSI + "def" + reset.
	assert.Contains(t, got, "def")
}

func TestSkipVisualWidthNegative(t *testing.T) {
	// Negative n: skipping starts from -1, so col increments to 0
	// which triggers skipping=false at col >= n=-1 immediately.
	// The first char check: col=0, col++ makes col=1, 1 >= -1 => skip.
	// Actually n < 0 means col starts at 0 which is >= n, so skipping stops
	// after the first rune is consumed. Let's just verify no panic.
	got := skipVisualWidth("hello", -1)
	assert.NotEmpty(t, got)
}

func TestSkipVisualWidthANSIInSkippedRegion(t *testing.T) {
	// ANSI sequences within the skipped region should be discarded.
	s := "\033[31mabc\033[0mdef"
	got := skipVisualWidth(s, 3)
	// Skipping 3 visual chars skips "abc" (with its ANSI wrappers).
	// Remaining should be "def".
	assert.Contains(t, got, "def")
	assert.NotContains(t, got, "\033[31m")
}

// ---------------------------------------------------------------------------
// RenderJumpPathOverlay
// ---------------------------------------------------------------------------

func TestRenderJumpPathOverlay(t *testing.T) {
	completions := []string{"secret/prod/", "secret/staging/", "secret/dev/"}
	result := RenderJumpPathOverlay("secret/", completions, 0, 100, 40)
	assert.Contains(t, result, "Jump to path")
	assert.Contains(t, result, "secret/prod/")
	assert.Contains(t, result, "secret/staging/")
	assert.Contains(t, result, "secret/dev/")
}

func TestRenderJumpPathOverlayNoCompletions(t *testing.T) {
	result := RenderJumpPathOverlay("secret/xyz", nil, 0, 100, 40)
	assert.Contains(t, result, "Jump to path")
	assert.NotContains(t, result, "more")
}

func TestRenderJumpPathOverlayManyCompletions(t *testing.T) {
	// More completions than can fit should show "... and N more"
	completions := make([]string, 30)
	for i := range 30 {
		completions[i] = "path/" + string(rune('a'+i%26))
	}
	result := RenderJumpPathOverlay("path/", completions, 0, 100, 20)
	assert.Contains(t, result, "more")
}

// ---------------------------------------------------------------------------
// PlaceOverlay (additional edge cases for uncovered branches)
// ---------------------------------------------------------------------------

func TestPlaceOverlayOverlayLargerThanBg(t *testing.T) {
	bg := "ab\ncd"
	overlay := "XXXXXXXXXX\nXXXXXXXXXX\nXXXXXXXXXX\nXXXXXXXXXX\nXXXXXXXXXX"
	result := PlaceOverlay(bg, overlay, 2, 2)
	assert.NotEmpty(t, result)
	assert.Contains(t, result, "X")
}

func TestPlaceOverlayZeroDimensions(t *testing.T) {
	// With bgHeight=0, the function slices bgLines[:0] and returns "".
	assert.NotPanics(t, func() {
		PlaceOverlay("", "hi", 0, 0)
	})
}

func TestPlaceOverlayBgShorterThanHeight(t *testing.T) {
	// Background has fewer lines than bgHeight -- should be padded
	bg := "line1"
	overlay := "XX"
	result := PlaceOverlay(bg, overlay, 10, 5)
	lines := strings.Split(result, "\n")
	assert.Equal(t, 5, len(lines))
}

func TestPlaceOverlayOverlayAtStartCol(t *testing.T) {
	// When overlay is wider than bg, startCol becomes 0 (or negative clamped to 0)
	bg := "ab\ncd\nef"
	overlay := "XY"
	result := PlaceOverlay(bg, overlay, 2, 3)
	assert.Contains(t, result, "XY")
}

func TestPlaceOverlaySuffixPreserved(t *testing.T) {
	// Background is wider than overlay -- suffix should be preserved
	bg := "0123456789\n0123456789\n0123456789"
	overlay := "XX"
	result := PlaceOverlay(bg, overlay, 10, 3)
	// The overlay is centered so bg chars after the overlay should still exist
	assert.Contains(t, result, "XX")
	// Some of the original background digits should remain
	assert.Contains(t, result, "0")
}

func TestPlaceOverlayEmptyOverlay(t *testing.T) {
	bg := "aaaa\nbbbb\ncccc"
	result := PlaceOverlay(bg, "", 4, 3)
	assert.Contains(t, result, "aaaa")
	assert.Contains(t, result, "bbbb")
}

// ---------------------------------------------------------------------------
// RenderConfirmOverlay (additional widths)
// ---------------------------------------------------------------------------

func TestRenderConfirmOverlayNarrowWidth(t *testing.T) {
	result := RenderConfirmOverlay("Delete this?", "DEL", 30)
	assert.Contains(t, result, "Delete this?")
	assert.Contains(t, result, "DELETE")
}

func TestRenderConfirmOverlayWideWidth(t *testing.T) {
	result := RenderConfirmOverlay("Are you sure?", "", 200)
	assert.Contains(t, result, "Are you sure?")
	assert.Contains(t, result, "confirm")
	assert.Contains(t, result, "cancel")
}

func TestRenderConfirmOverlayWidthClampToMax(t *testing.T) {
	// When width is small, boxWidth = width/2 could be < 40, clamped to 40.
	// Then if 40 > width-4, it gets clamped again.
	result := RenderConfirmOverlay("Sure?", "DEL", 38)
	assert.Contains(t, result, "Sure?")
}

// ---------------------------------------------------------------------------
// RenderInputOverlay (additional widths)
// ---------------------------------------------------------------------------

func TestRenderInputOverlayWideWidth(t *testing.T) {
	result := RenderInputOverlay("Enter:", "val", 200)
	assert.Contains(t, result, "Enter:")
	assert.Contains(t, result, "val")
}

func TestRenderInputOverlayWidthClamped(t *testing.T) {
	// width/2 = 18 < 40, so boxWidth = 40, then 40 > 36-4=32, so clamped to 32
	result := RenderInputOverlay("Name:", "x", 36)
	assert.Contains(t, result, "Name:")
}

// ---------------------------------------------------------------------------
// RenderThemePickerOverlay (additional coverage)
// ---------------------------------------------------------------------------

func TestRenderThemePickerOverlayScrolling(t *testing.T) {
	entries := make([]ThemeEntry, 0)
	entries = append(entries, ThemeEntry{Name: "Dark Themes", IsHeader: true})
	for i := 0; i < 30; i++ {
		entries = append(entries, ThemeEntry{Name: "theme-" + string(rune('a'+i%26))})
	}
	// Cursor near bottom, small height forces scrolling
	result := RenderThemePickerOverlay(entries, 25, "theme-a", 80, 20)
	assert.Contains(t, result, "Colorscheme")
}

func TestRenderThemePickerOverlayCursorOnHeader(t *testing.T) {
	entries := []ThemeEntry{
		{Name: "Dark Themes", IsHeader: true},
		{Name: "tokyonight"},
		{Name: "Light Themes", IsHeader: true},
		{Name: "gruvbox-light"},
	}
	// Cursor on index 0 which is a header
	result := RenderThemePickerOverlay(entries, 0, "tokyonight", 80, 40)
	assert.Contains(t, result, "Dark Themes")
	assert.Contains(t, result, "tokyonight")
}

func TestRenderThemePickerOverlayLongThemeName(t *testing.T) {
	entries := []ThemeEntry{
		{Name: "a-very-long-theme-name-that-exceeds-normal-column-width-limits"},
	}
	result := RenderThemePickerOverlay(entries, 0, "", 60, 40)
	assert.Contains(t, result, "Colorscheme")
}

func TestRenderThemePickerOverlayNoActiveTheme(t *testing.T) {
	entries := []ThemeEntry{
		{Name: "Dark Themes", IsHeader: true},
		{Name: "nord"},
	}
	result := RenderThemePickerOverlay(entries, 1, "", 80, 40)
	// No asterisk marker should appear on nord since activeTheme is empty
	lines := strings.Split(result, "\n")
	for _, line := range lines {
		if strings.Contains(line, "nord") {
			assert.NotContains(t, line, "*")
		}
	}
}

func TestRenderThemePickerOverlaySmallHeight(t *testing.T) {
	entries := []ThemeEntry{
		{Name: "Dark Themes", IsHeader: true},
		{Name: "tokyonight"},
		{Name: "nord"},
		{Name: "dracula"},
	}
	// Height 12 minus 10 overhead = maxShow of 2, so not all entries visible
	result := RenderThemePickerOverlay(entries, 0, "", 80, 12)
	assert.Contains(t, result, "Colorscheme")
}

// ---------------------------------------------------------------------------
// RenderBookmarkOverlay (additional coverage for scroll window)
// ---------------------------------------------------------------------------

func TestRenderBookmarkOverlayScrollWindow(t *testing.T) {
	bookmarks := make([]config.Bookmark, 30)
	for i := range bookmarks {
		bookmarks[i] = config.Bookmark{
			Name:  "bm-" + string(rune('a'+i%26)),
			Mount: "secret",
			Path:  "path/" + string(rune('a'+i%26)),
		}
	}
	// Cursor at end, small height
	result := RenderBookmarkOverlay(bookmarks, "", false, 28, 80, 20)
	assert.Contains(t, result, "Marks")
	// Should show "total" count indicator
	assert.Contains(t, result, "total")
}

func TestRenderBookmarkOverlaySearchingMode(t *testing.T) {
	bookmarks := []config.Bookmark{
		{Name: "alpha", Mount: "kv", Path: "alpha"},
	}
	result := RenderBookmarkOverlay(bookmarks, "al", true, 0, 80, 40)
	// Searching mode shows cursor underscore
	assert.Contains(t, result, "al_")
}

func TestRenderBookmarkOverlayNameTruncation(t *testing.T) {
	bookmarks := []config.Bookmark{
		{
			Name:  "a-very-long-bookmark-name-that-should-be-truncated-in-display",
			Mount: "secret",
			Path:  "long/path",
		},
	}
	result := RenderBookmarkOverlay(bookmarks, "", false, 0, 60, 40)
	assert.Contains(t, result, "...")
}

// ---------------------------------------------------------------------------
// takeVisualWidth / skipVisualWidth (additional ANSI edge cases)
// ---------------------------------------------------------------------------

func TestTakeVisualWidthMultipleANSI(t *testing.T) {
	// Multiple ANSI sequences interleaved with text
	s := "\033[1m\033[31mhello\033[0m world"
	got := takeVisualWidth(s, 7)
	assert.Contains(t, got, "hello")
	// Should contain start of " w" from "world" (positions 6-7)
	assert.Contains(t, got, " w")
}

func TestTakeVisualWidthOnlyANSI(t *testing.T) {
	// String is all ANSI with no visible content
	s := "\033[31m\033[0m"
	got := takeVisualWidth(s, 3)
	// Should pad with spaces since there are no visible chars
	assert.Len(t, got, len("\033[31m\033[0m")+3)
}

func TestSkipVisualWidthMultipleANSI(t *testing.T) {
	s := "\033[1mABC\033[31mDEF\033[0m"
	got := skipVisualWidth(s, 3)
	// After skipping 3 visible chars "ABC", should get "DEF" with its ANSI
	assert.Contains(t, got, "DEF")
}

func TestTakeVisualWidthTildeTerminator(t *testing.T) {
	// Some ANSI sequences end with ~ (e.g., cursor position reports)
	s := "\033[1~hello"
	got := takeVisualWidth(s, 3)
	assert.Contains(t, got, "hel")
}

// ---------------------------------------------------------------------------
// RenderJumpPathOverlay (additional coverage)
// ---------------------------------------------------------------------------

func TestRenderJumpPathOverlaySelectedCompletion(t *testing.T) {
	completions := []string{"path/alpha/", "path/beta", "path/gamma/"}
	result := RenderJumpPathOverlay("path/", completions, 1, 100, 40)
	assert.Contains(t, result, "path/alpha/")
	assert.Contains(t, result, "path/beta")
	assert.Contains(t, result, "path/gamma/")
}

func TestRenderJumpPathOverlayLongPaths(t *testing.T) {
	completions := []string{
		"a/very/long/path/that/should/be/truncated/in/the/display/somehow/yeah",
	}
	result := RenderJumpPathOverlay("a/", completions, 0, 80, 40)
	assert.Contains(t, result, "...")
}

func TestRenderJumpPathOverlaySmallHeight(t *testing.T) {
	completions := []string{"a/", "b/", "c/"}
	// Height = 8 means maxShow = height-8 = 0, clamped to 1
	result := RenderJumpPathOverlay("", completions, 0, 100, 8)
	assert.Contains(t, result, "Jump to path")
}

func TestRenderJumpPathOverlayNarrowWidth(t *testing.T) {
	completions := []string{"abc/"}
	// Width < 50 triggers clamping
	result := RenderJumpPathOverlay("a", completions, 0, 40, 40)
	assert.Contains(t, result, "Jump to path")
}

func TestRenderJumpPathOverlayBoxWidthExceedsAvailable(t *testing.T) {
	// width=52 => boxWidth=26 < 50, so clamped to 50. Then 50 > 52-4=48, so clamped to 48.
	completions := []string{"a/"}
	result := RenderJumpPathOverlay("x", completions, 0, 52, 40)
	assert.Contains(t, result, "Jump to path")
}

func TestRenderJumpPathOverlayFileCompletion(t *testing.T) {
	// Completion without trailing slash is rendered as a file
	completions := []string{"secret-file"}
	result := RenderJumpPathOverlay("sec", completions, -1, 100, 40)
	assert.Contains(t, result, "secret-file")
}

// ---------------------------------------------------------------------------
// RenderBookmarkOverlay (narrow width clamping)
// ---------------------------------------------------------------------------

func TestRenderBookmarkOverlayNarrowWidth(t *testing.T) {
	// width=38 => boxWidth=12 < 40, clamped to 40. Then 40 > 38-4=34, clamped to 34.
	bookmarks := []config.Bookmark{
		{Name: "bm", Mount: "kv", Path: "bm"},
	}
	result := RenderBookmarkOverlay(bookmarks, "", false, 0, 38, 40)
	assert.Contains(t, result, "bm")
}

func TestRenderBookmarkOverlayNoSlotNonCursor(t *testing.T) {
	// Non-cursor bookmark without a slot uses "  " prefix
	bookmarks := []config.Bookmark{
		{Name: "first", Mount: "kv", Path: "first"},
		{Name: "second", Mount: "kv", Path: "second"},
	}
	result := RenderBookmarkOverlay(bookmarks, "", false, 0, 80, 40)
	assert.Contains(t, result, "first")
	assert.Contains(t, result, "second")
}

func TestRenderBookmarkOverlaySlotOnCursor(t *testing.T) {
	bookmarks := []config.Bookmark{
		{Name: "prod-db", Mount: "secret", Path: "prod/db", Slot: "p"},
	}
	result := RenderBookmarkOverlay(bookmarks, "", false, 0, 80, 40)
	assert.Contains(t, result, "p")
	assert.Contains(t, result, "prod-db")
}

// ---------------------------------------------------------------------------
// RenderThemePickerOverlay (narrow width clamping)
// ---------------------------------------------------------------------------

func TestRenderThemePickerOverlayNarrowWidth(t *testing.T) {
	// width=38 => boxWidth=19 < 40, clamped to 40. Then 40 > 38-4=34, clamped to 34.
	entries := []ThemeEntry{{Name: "nord"}}
	result := RenderThemePickerOverlay(entries, 0, "", 38, 40)
	assert.Contains(t, result, "nord")
}

func TestRenderThemePickerOverlayCursorNotOnEntry(t *testing.T) {
	// Cursor beyond range -- should not panic
	entries := []ThemeEntry{
		{Name: "Dark Themes", IsHeader: true},
		{Name: "tokyonight"},
	}
	assert.NotPanics(t, func() {
		RenderThemePickerOverlay(entries, 99, "", 80, 40)
	})
}

func TestRenderThemePickerOverlayNameTruncation(t *testing.T) {
	// Theme name longer than maxEntryW (boxWidth-8)
	longName := strings.Repeat("x", 100)
	entries := []ThemeEntry{{Name: longName}}
	result := RenderThemePickerOverlay(entries, 0, "", 50, 40)
	assert.Contains(t, result, "Colorscheme")
}

// ---------------------------------------------------------------------------
// RenderBookmarkOverlay (small height triggers maxShow < 3 clamp)
// ---------------------------------------------------------------------------

func TestRenderBookmarkOverlaySmallHeight(t *testing.T) {
	bookmarks := []config.Bookmark{
		{Name: "bm1", Mount: "kv", Path: "bm1"},
		{Name: "bm2", Mount: "kv", Path: "bm2"},
		{Name: "bm3", Mount: "kv", Path: "bm3"},
	}
	// height=10, maxShow=10-10=0, clamped to 3
	result := RenderBookmarkOverlay(bookmarks, "", false, 0, 80, 10)
	assert.Contains(t, result, "bm1")
}

func TestRenderBookmarkOverlayNoSlotNoFilter(t *testing.T) {
	// Bookmark without slot, filter empty, not selected
	bookmarks := []config.Bookmark{
		{Name: "a", Mount: "kv", Path: "a"},
		{Name: "b", Mount: "kv", Path: "b"},
	}
	result := RenderBookmarkOverlay(bookmarks, "", false, 1, 80, 40)
	assert.Contains(t, result, "a")
	assert.Contains(t, result, "b")
}
