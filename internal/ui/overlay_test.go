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
