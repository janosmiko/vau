package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/janosmiko/vau/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestHelpContentLineCountNoFilter(t *testing.T) {
	count := HelpContentLineCount("")
	assert.Greater(t, count, 0, "help content with no filter should have lines")
}

func TestHelpContentLineCountNonexistentFilter(t *testing.T) {
	count := HelpContentLineCount("nonexistent_binding_xyz")
	assert.Equal(t, 1, count, "non-matching filter should return 1 line (the 'No matching' message)")
}

func TestHelpContentLineCountWithMatchingFilter(t *testing.T) {
	// "quit" should match at least one binding.
	count := HelpContentLineCount("quit")
	assert.Greater(t, count, 0, "filter 'quit' should match at least one binding")
	assert.Less(t, count, HelpContentLineCount(""),
		"filtered count should be less than unfiltered count")
}

func TestHelpBarExplorer(t *testing.T) {
	bar := HelpBar(model.ModeExplorer, 200)
	assert.Contains(t, bar, "nav")
	assert.Contains(t, bar, "open")
	assert.Contains(t, bar, "search")
	assert.Contains(t, bar, "quit")
	assert.Contains(t, bar, "help")
}

func TestHelpBarConfirm(t *testing.T) {
	bar := HelpBar(model.ModeConfirm, 200)
	assert.Contains(t, bar, "confirm")
	assert.Contains(t, bar, "cancel")
}

func TestHelpBarInput(t *testing.T) {
	bar := HelpBar(model.ModeInput, 200)
	assert.Contains(t, bar, "confirm")
	assert.Contains(t, bar, "cancel")
}

func TestHelpBarSearch(t *testing.T) {
	bar := HelpBar(model.ModeSearch, 200)
	assert.Contains(t, bar, "jump to match")
	assert.Contains(t, bar, "cancel")
}

func TestHelpBarFilter(t *testing.T) {
	bar := HelpBar(model.ModeFilter, 200)
	assert.Contains(t, bar, "apply filter")
	assert.Contains(t, bar, "clear filter")
}

func TestHelpBarJumpPath(t *testing.T) {
	bar := HelpBar(model.ModeJumpPath, 200)
	assert.Contains(t, bar, "complete")
	assert.Contains(t, bar, "cancel")
}

func TestHelpBarBookmark(t *testing.T) {
	bar := HelpBar(model.ModeBookmark, 200)
	assert.Contains(t, bar, "jump")
	assert.Contains(t, bar, "close")
}

func TestHelpBarSecret(t *testing.T) {
	bar := HelpBar(model.ModeSecret, 200)
	assert.Contains(t, bar, "nav")
	assert.Contains(t, bar, "toggle")
	assert.Contains(t, bar, "json")
	assert.Contains(t, bar, "copy")
	assert.Contains(t, bar, "copy as")
	assert.Contains(t, bar, "edit")
	assert.Contains(t, bar, "close")
}

func TestHelpBarSecretEdit(t *testing.T) {
	bar := HelpBar(model.ModeSecretEdit, 200)
	assert.Contains(t, bar, "switch col")
	assert.Contains(t, bar, "save")
	assert.Contains(t, bar, "cancel")
}

func TestHelpBarVersionHistory(t *testing.T) {
	bar := HelpBar(model.ModeVersionHistory, 200)
	assert.Contains(t, bar, "nav")
	assert.Contains(t, bar, "view version")
	assert.Contains(t, bar, "back")
}

func TestHelpBarThemePicker(t *testing.T) {
	bar := HelpBar(model.ModeThemePicker, 200)
	assert.Contains(t, bar, "nav")
	assert.Contains(t, bar, "select")
	assert.Contains(t, bar, "cancel")
}

func TestHelpBarHelp(t *testing.T) {
	bar := HelpBar(model.ModeHelp, 200)
	// Help mode should return empty bindings (has its own built-in help).
	assert.NotEmpty(t, bar) // Still renders the status bar container.
}

func TestHelpBarNarrowWidth(t *testing.T) {
	// With a very narrow width, some bindings should be truncated.
	bar := HelpBar(model.ModeExplorer, 30)
	assert.NotEmpty(t, bar)
	// It should have fewer bindings than the wide version.
	wideBar := HelpBar(model.ModeExplorer, 200)
	assert.LessOrEqual(t, len(bar), len(wideBar))
}

// ---------------------------------------------------------------------------
// buildHelpLines
// ---------------------------------------------------------------------------

func TestBuildHelpLinesNoFilter(t *testing.T) {
	lines := buildHelpLines("")
	assert.NotEmpty(t, lines)
	// Should contain section headers from multiple sections
	found := 0
	for _, line := range lines {
		if strings.Contains(line, "Explorer") || strings.Contains(line, "Secret Popup") {
			found++
		}
	}
	assert.GreaterOrEqual(t, found, 2, "should contain multiple section headers")
}

func TestBuildHelpLinesWithFilter(t *testing.T) {
	lines := buildHelpLines("quit")
	assert.NotEmpty(t, lines)
	hasQuit := false
	for _, line := range lines {
		if strings.Contains(strings.ToLower(line), "quit") {
			hasQuit = true
		}
	}
	assert.True(t, hasQuit, "should contain quit binding")
}

func TestBuildHelpLinesFilterCaseInsensitive(t *testing.T) {
	linesUpper := buildHelpLines("ENTER")
	linesLower := buildHelpLines("enter")
	assert.Equal(t, len(linesUpper), len(linesLower),
		"filter should be case insensitive")
}

func TestBuildHelpLinesFilterMatchesKey(t *testing.T) {
	lines := buildHelpLines("hjkl")
	assert.NotEmpty(t, lines)
}

func TestBuildHelpLinesFilterMatchesDesc(t *testing.T) {
	lines := buildHelpLines("Navigate to parent")
	assert.NotEmpty(t, lines)
}

func TestBuildHelpLinesNoMatchReturnsMessage(t *testing.T) {
	lines := buildHelpLines("xyznonexistent123")
	assert.Len(t, lines, 1)
	assert.Contains(t, lines[0], "No matching keybindings")
}

func TestBuildHelpLinesFilterReducesOutput(t *testing.T) {
	all := buildHelpLines("")
	filtered := buildHelpLines("quit")
	assert.Less(t, len(filtered), len(all),
		"filtered output should have fewer lines than unfiltered")
}

func TestBuildHelpLinesEmptySectionsOmitted(t *testing.T) {
	// "colorscheme" matches "Change colorscheme" in Explorer, but not in
	// Mount Selection so that section header should be absent.
	lines := buildHelpLines("colorscheme")
	hasMountSelection := false
	for _, line := range lines {
		if strings.Contains(line, "Mount Selection") {
			hasMountSelection = true
		}
	}
	assert.False(t, hasMountSelection,
		"Mount Selection section should be omitted when filtering for colorscheme")
}

// ---------------------------------------------------------------------------
// helpSections
// ---------------------------------------------------------------------------

func TestHelpSectionsNotEmpty(t *testing.T) {
	sections := helpSections()
	assert.NotEmpty(t, sections)
	for _, s := range sections {
		assert.NotEmpty(t, s.title, "each section should have a title")
		assert.NotEmpty(t, s.bindings, "each section should have bindings")
	}
}

func TestHelpSectionsContainExpectedSections(t *testing.T) {
	sections := helpSections()
	titles := make(map[string]bool)
	for _, s := range sections {
		titles[s.title] = true
	}
	expected := []string{
		"Explorer", "Mount Selection", "Secret Popup",
		"Search (/)", "Filter (f)", "Help (?)",
	}
	for _, title := range expected {
		assert.True(t, titles[title], "should contain section %q", title)
	}
}

// ---------------------------------------------------------------------------
// RenderHelpScreen
// ---------------------------------------------------------------------------

func TestRenderHelpScreenNoFilter(t *testing.T) {
	result := RenderHelpScreen(100, 50, 0, "", false, nil)
	assert.Contains(t, result, "Keybindings")
	assert.Contains(t, result, "Explorer")
	assert.Contains(t, result, "scroll")
	assert.Contains(t, result, "close")
}

func TestRenderHelpScreenWithFilter(t *testing.T) {
	result := RenderHelpScreen(100, 50, 0, "quit", false, nil)
	assert.Contains(t, result, "Keybindings")
	assert.Contains(t, result, "filter:")
	assert.Contains(t, result, "quit")
}

func TestRenderHelpScreenSearching(t *testing.T) {
	ti := textinput.New()
	ti.SetValue("nav")
	result := RenderHelpScreen(100, 50, 0, "", true, &ti)
	assert.Contains(t, result, "search:")
}

func TestRenderHelpScreenScrollDown(t *testing.T) {
	result := RenderHelpScreen(100, 50, 5, "", false, nil)
	assert.Contains(t, result, "Keybindings")
	assert.Contains(t, result, "more above")
}

func TestRenderHelpScreenScrollZero(t *testing.T) {
	result := RenderHelpScreen(100, 50, 0, "", false, nil)
	assert.NotContains(t, result, "more above")
}

func TestRenderHelpScreenScrollBeyondMax(t *testing.T) {
	result := RenderHelpScreen(100, 50, 99999, "", false, nil)
	assert.Contains(t, result, "Keybindings")
	assert.NotContains(t, result, "more below")
}

func TestRenderHelpScreenNegativeScroll(t *testing.T) {
	result := RenderHelpScreen(100, 50, -5, "", false, nil)
	assert.Contains(t, result, "Keybindings")
	assert.NotContains(t, result, "more above")
}

func TestRenderHelpScreenSmallDimensions(t *testing.T) {
	assert.NotPanics(t, func() {
		RenderHelpScreen(30, 15, 0, "", false, nil)
	})
}

func TestRenderHelpScreenMinimumClamp(t *testing.T) {
	result := RenderHelpScreen(10, 5, 0, "", false, nil)
	assert.Contains(t, result, "Keybindings")
}

func TestRenderHelpScreenMoreBelow(t *testing.T) {
	// Small height to ensure content overflows
	result := RenderHelpScreen(80, 25, 0, "", false, nil)
	assert.Contains(t, result, "more below")
}

func TestRenderHelpScreenFilterNoMatch(t *testing.T) {
	result := RenderHelpScreen(100, 50, 0, "zzz_nonexistent", false, nil)
	assert.Contains(t, result, "No matching keybindings")
}
