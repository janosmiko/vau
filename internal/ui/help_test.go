package ui

import (
	"testing"

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

func TestHelpBarSecretModeEmpty(t *testing.T) {
	// Secret mode should render but have minimal/empty bindings.
	bar := HelpBar(model.ModeSecret, 200)
	// It should at least return a string (the status bar rendering).
	assert.NotEmpty(t, bar)
}

func TestHelpBarNarrowWidth(t *testing.T) {
	// With a very narrow width, some bindings should be truncated.
	bar := HelpBar(model.ModeExplorer, 30)
	assert.NotEmpty(t, bar)
	// It should have fewer bindings than the wide version.
	wideBar := HelpBar(model.ModeExplorer, 200)
	assert.LessOrEqual(t, len(bar), len(wideBar))
}
