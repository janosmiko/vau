package ui

import (
	"strings"
	"testing"

	"github.com/janosmiko/vau/internal/model"
	"github.com/stretchr/testify/assert"
)

// ---------------------------------------------------------------------------
// DimText
// ---------------------------------------------------------------------------

func TestDimText(t *testing.T) {
	result := DimText("(empty)", 80)
	assert.Contains(t, result, "(empty)")
}

func TestDimTextEmptyString(t *testing.T) {
	result := DimText("", 80)
	assert.NotNil(t, result)
}

// ---------------------------------------------------------------------------
// highlightName
// ---------------------------------------------------------------------------

func TestHighlightNameNoQuery(t *testing.T) {
	result := highlightName("myfile.txt", "", FileStyle, HighlightStyle)
	assert.Contains(t, result, "myfile.txt")
}

func TestHighlightNameMatchFound(t *testing.T) {
	result := highlightName("myfile.txt", "file", FileStyle, HighlightStyle)
	assert.Contains(t, result, "file")
	assert.Contains(t, result, "my")
	assert.Contains(t, result, ".txt")
}

func TestHighlightNameNoMatch(t *testing.T) {
	result := highlightName("myfile.txt", "zzz", FileStyle, HighlightStyle)
	assert.Contains(t, result, "myfile.txt")
}

func TestHighlightNameCaseInsensitive(t *testing.T) {
	result := highlightName("MyFile.TXT", "myfile", FileStyle, HighlightStyle)
	assert.Contains(t, result, "MyFile")
}

func TestHighlightNameMatchAtStart(t *testing.T) {
	result := highlightName("secret-key", "secret", DirStyle, HighlightStyle)
	assert.Contains(t, result, "secret")
	assert.Contains(t, result, "-key")
}

func TestHighlightNameMatchAtEnd(t *testing.T) {
	result := highlightName("my-secret", "secret", FileStyle, HighlightStyle)
	assert.Contains(t, result, "my-")
	assert.Contains(t, result, "secret")
}

func TestHighlightNameEntireStringMatches(t *testing.T) {
	result := highlightName("abc", "abc", FileStyle, HighlightStyle)
	assert.Contains(t, result, "abc")
}

// ---------------------------------------------------------------------------
// renderEntryList
// ---------------------------------------------------------------------------

func TestRenderEntryListEmpty(t *testing.T) {
	result := renderEntryList(nil, 0, nil, 40, 20, true, "")
	assert.Contains(t, result, "(empty)")
}

func TestRenderEntryListDirsAndFiles(t *testing.T) {
	entries := []model.Entry{
		{Name: "subdir/", IsDir: true},
		{Name: "file.txt", IsDir: false},
	}
	result := renderEntryList(entries, 0, nil, 40, 20, true, "")
	assert.Contains(t, result, "subdir/")
	assert.Contains(t, result, "file.txt")
}

func TestRenderEntryListSelectedIdx(t *testing.T) {
	entries := []model.Entry{
		{Name: "alpha", IsDir: false},
		{Name: "beta", IsDir: false},
	}
	result := renderEntryList(entries, 1, nil, 40, 20, true, "")
	assert.Contains(t, result, "alpha")
	assert.Contains(t, result, "beta")
}

func TestRenderEntryListInactiveColumn(t *testing.T) {
	entries := []model.Entry{
		{Name: "item", IsDir: false},
	}
	result := renderEntryList(entries, 0, nil, 40, 20, false, "")
	assert.Contains(t, result, "item")
}

func TestRenderEntryListWithHighlight(t *testing.T) {
	entries := []model.Entry{
		{Name: "secret-prod", IsDir: false},
		{Name: "secret-dev", IsDir: false},
		{Name: "other", IsDir: false},
	}
	result := renderEntryList(entries, 0, nil, 40, 20, true, "secret")
	assert.Contains(t, result, "secret")
	assert.Contains(t, result, "other")
}

func TestRenderEntryListWithSelection(t *testing.T) {
	entries := []model.Entry{
		{Name: "file1", IsDir: false},
		{Name: "file2", IsDir: false},
		{Name: "file3", IsDir: false},
	}
	selected := map[int]bool{0: true, 2: true}
	result := renderEntryList(entries, 1, selected, 40, 20, true, "")
	// Selected items should show bullet marker
	assert.Contains(t, result, "\u25CF") // bullet
	assert.Contains(t, result, "file1")
	assert.Contains(t, result, "file2")
	assert.Contains(t, result, "file3")
}

func TestRenderEntryListScrolling(t *testing.T) {
	entries := make([]model.Entry, 50)
	for i := range entries {
		entries[i] = model.Entry{Name: "item-" + string(rune('a'+i%26)), IsDir: false}
	}
	// Select item 45 with height 10 -- triggers scroll
	result := renderEntryList(entries, 45, nil, 40, 10, true, "")
	assert.NotEmpty(t, result)
}

func TestRenderEntryListLongNames(t *testing.T) {
	entries := []model.Entry{
		{Name: "a-very-long-entry-name-that-should-be-truncated-in-display", IsDir: false},
	}
	result := renderEntryList(entries, 0, nil, 30, 20, true, "")
	assert.Contains(t, result, "...")
}

func TestRenderEntryListDirSelected(t *testing.T) {
	entries := []model.Entry{
		{Name: "dir/", IsDir: true},
		{Name: "file", IsDir: false},
	}
	selected := map[int]bool{0: true}
	result := renderEntryList(entries, 1, selected, 40, 20, true, "")
	assert.Contains(t, result, "dir/")
}

// ---------------------------------------------------------------------------
// renderPreview
// ---------------------------------------------------------------------------

func TestRenderPreviewEmpty(t *testing.T) {
	result := renderPreview(nil, nil, model.PreviewHidden, 40, 20)
	assert.Contains(t, result, "(empty)")
}

func TestRenderPreviewWithEntries(t *testing.T) {
	entries := []model.Entry{
		{Name: "subdir/", IsDir: true},
		{Name: "config.yaml", IsDir: false},
	}
	result := renderPreview(entries, nil, model.PreviewHidden, 40, 20)
	assert.Contains(t, result, "subdir/")
	assert.Contains(t, result, "config.yaml")
}

func TestRenderPreviewWithMoreEntries(t *testing.T) {
	entries := make([]model.Entry, 30)
	for i := range entries {
		entries[i] = model.Entry{Name: "entry-" + string(rune('a'+i%26)), IsDir: false}
	}
	result := renderPreview(entries, nil, model.PreviewHidden, 40, 10)
	assert.Contains(t, result, "more")
}

func TestRenderPreviewLongEntryNames(t *testing.T) {
	entries := []model.Entry{
		{Name: "a-very-long-entry-name-that-exceeds-the-column-width-limit", IsDir: false},
	}
	result := renderPreview(entries, nil, model.PreviewHidden, 30, 20)
	assert.Contains(t, result, "...")
}

func TestRenderPreviewSecretHidden(t *testing.T) {
	secret := &model.Secret{
		Keys: []string{"username", "password"},
		Data: map[string]string{"username": "admin", "password": "s3cret"},
	}
	result := renderPreview(nil, secret, model.PreviewHidden, 60, 20)
	assert.Contains(t, result, "username")
	assert.Contains(t, result, "********")
	assert.NotContains(t, result, "admin")
}

func TestRenderPreviewSecretValues(t *testing.T) {
	secret := &model.Secret{
		Keys: []string{"username"},
		Data: map[string]string{"username": "admin"},
	}
	result := renderPreview(nil, secret, model.PreviewValues, 60, 20)
	assert.Contains(t, result, "username")
	assert.Contains(t, result, "admin")
}

func TestRenderPreviewSecretJSON(t *testing.T) {
	secret := &model.Secret{
		Keys: []string{"key1"},
		Data: map[string]string{"key1": "val1"},
	}
	result := renderPreview(nil, secret, model.PreviewJSON, 60, 20)
	assert.Contains(t, result, "{")
	assert.Contains(t, result, "}")
	assert.Contains(t, result, "key1")
}

// ---------------------------------------------------------------------------
// renderSecretPreview
// ---------------------------------------------------------------------------

func TestRenderSecretPreviewHidden(t *testing.T) {
	secret := &model.Secret{
		Keys: []string{"a", "b"},
		Data: map[string]string{"a": "val-a", "b": "val-b"},
	}
	result := renderSecretPreview(secret, model.PreviewHidden, 60, 20)
	assert.Contains(t, result, "Key")
	assert.Contains(t, result, "Value")
	assert.Contains(t, result, "a")
	assert.Contains(t, result, "b")
	assert.Contains(t, result, "********")
	assert.NotContains(t, result, "val-a")
}

func TestRenderSecretPreviewValues(t *testing.T) {
	secret := &model.Secret{
		Keys: []string{"host"},
		Data: map[string]string{"host": "localhost"},
	}
	result := renderSecretPreview(secret, model.PreviewValues, 60, 20)
	assert.Contains(t, result, "host")
	assert.Contains(t, result, "localhost")
}

func TestRenderSecretPreviewJSON(t *testing.T) {
	secret := &model.Secret{
		Keys: []string{"k"},
		Data: map[string]string{"k": "v"},
	}
	result := renderSecretPreview(secret, model.PreviewJSON, 60, 20)
	assert.Contains(t, result, "{")
	assert.Contains(t, result, "}")
}

func TestRenderSecretPreviewSmallWidth(t *testing.T) {
	secret := &model.Secret{
		Keys: []string{"key"},
		Data: map[string]string{"key": "value"},
	}
	result := renderSecretPreview(secret, model.PreviewHidden, 8, 10)
	assert.NotEmpty(t, result)
}

func TestRenderSecretPreviewHeightLimit(t *testing.T) {
	keys := make([]string, 20)
	data := make(map[string]string)
	for i := range 20 {
		k := string(rune('a' + i))
		keys[i] = k
		data[k] = "val"
	}
	secret := &model.Secret{Keys: keys, Data: data}
	result := renderSecretPreview(secret, model.PreviewValues, 60, 5)
	assert.NotEmpty(t, result)
	// Not all 20 keys should be rendered
	lines := strings.Split(result, "\n")
	assert.LessOrEqual(t, len(lines), 7) // header + sep + 5 items max
}

// ---------------------------------------------------------------------------
// renderSecretJSON (explorer.go version)
// ---------------------------------------------------------------------------

func TestRenderSecretJSONExplorer(t *testing.T) {
	secret := &model.Secret{
		Keys: []string{"name", "ver"},
		Data: map[string]string{"name": "app", "ver": "2.0"},
	}
	result := renderSecretJSON(secret, 20)
	assert.Contains(t, result, "{")
	assert.Contains(t, result, "}")
	assert.Contains(t, result, "name")
	assert.Contains(t, result, "app")
}

func TestRenderSecretJSONExplorerHeightLimit(t *testing.T) {
	keys := make([]string, 20)
	data := make(map[string]string)
	for i := range 20 {
		k := "key" + string(rune('a'+i))
		keys[i] = k
		data[k] = "val"
	}
	secret := &model.Secret{Keys: keys, Data: data}
	result := renderSecretJSON(secret, 5)
	assert.Contains(t, result, "{")
	assert.Contains(t, result, "}")
	lines := strings.Split(result, "\n")
	assert.LessOrEqual(t, len(lines), 5) // { + 3 entries + }
}

func TestRenderSecretJSONExplorerCommas(t *testing.T) {
	secret := &model.Secret{
		Keys: []string{"a", "b"},
		Data: map[string]string{"a": "1", "b": "2"},
	}
	result := renderSecretJSON(secret, 20)
	lines := strings.Split(result, "\n")
	// First data line should have comma, last should not
	var dataLines []string
	for _, line := range lines {
		if strings.Contains(line, ":") {
			dataLines = append(dataLines, line)
		}
	}
	if len(dataLines) >= 2 {
		assert.Contains(t, dataLines[0], ",")
		assert.NotContains(t, dataLines[len(dataLines)-1], ",")
	}
}

// ---------------------------------------------------------------------------
// RenderTabBar
// ---------------------------------------------------------------------------

func TestRenderTabBarSingleTab(t *testing.T) {
	result := RenderTabBar([]string{"secret/"}, 0, 100)
	assert.Contains(t, result, "1 secret/")
}

func TestRenderTabBarMultipleTabs(t *testing.T) {
	labels := []string{"secret/", "kv/prod", "kv/staging"}
	result := RenderTabBar(labels, 1, 100)
	assert.Contains(t, result, "1 secret/")
	assert.Contains(t, result, "2 kv/prod")
	assert.Contains(t, result, "3 kv/staging")
}

func TestRenderTabBarActiveHighlighted(t *testing.T) {
	labels := []string{"tab-a", "tab-b"}
	resultA := RenderTabBar(labels, 0, 100)
	resultB := RenderTabBar(labels, 1, 100)
	// Both should contain both labels
	assert.Contains(t, resultA, "tab-a")
	assert.Contains(t, resultA, "tab-b")
	assert.Contains(t, resultB, "tab-a")
	assert.Contains(t, resultB, "tab-b")
}

func TestRenderTabBarNarrowWidthTruncates(t *testing.T) {
	labels := make([]string, 10)
	for i := range labels {
		labels[i] = "a-very-long-tab-path-that-is-way-too-long"
	}
	result := RenderTabBar(labels, 5, 60)
	assert.NotEmpty(t, result)
	// Should truncate with ellipsis character
	assert.Contains(t, result, "\u2026") // ...
}

func TestRenderTabBarWindowsAroundActive(t *testing.T) {
	labels := make([]string, 20)
	for i := range labels {
		labels[i] = "path-" + string(rune('a'+i%26))
	}
	// Active tab near the end with limited width
	result := RenderTabBar(labels, 18, 80)
	assert.NotEmpty(t, result)
	// Should show arrow indicators when not all tabs fit
	assert.Contains(t, result, "\u25C2") // left arrow
}

func TestRenderTabBarAllFitNoArrows(t *testing.T) {
	labels := []string{"a", "b"}
	result := RenderTabBar(labels, 0, 200)
	// Should not contain arrow indicators
	assert.NotContains(t, result, "\u25C2")
	assert.NotContains(t, result, "\u25B8")
}

// ---------------------------------------------------------------------------
// RenderExplorer
// ---------------------------------------------------------------------------

func TestRenderExplorerBasic(t *testing.T) {
	parent := []model.Entry{{Name: "root/", IsDir: true}}
	current := []model.Entry{
		{Name: "subdir/", IsDir: true},
		{Name: "secret-a", IsDir: false},
	}
	preview := []model.Entry{{Name: "child/", IsDir: true}}

	result := RenderExplorer(
		parent, current, preview,
		nil, model.PreviewHidden,
		0, 0, nil,
		[]string{"root/"},
		"secret",
		nil, 0,
		"",
		"v0.1.0",
		100, 40,
	)

	assert.Contains(t, result, "secret/")
	assert.Contains(t, result, "root/")
	assert.Contains(t, result, "vau v0.1.0")
}

func TestRenderExplorerWithTabs(t *testing.T) {
	result := RenderExplorer(
		nil, nil, nil,
		nil, model.PreviewHidden,
		0, 0, nil,
		nil,
		"kv",
		[]string{"kv/prod", "kv/staging"}, 0,
		"",
		"v0.2.0",
		100, 40,
	)

	assert.Contains(t, result, "1 kv/prod")
	assert.Contains(t, result, "2 kv/staging")
}

func TestRenderExplorerWithSecretPreview(t *testing.T) {
	secret := &model.Secret{
		Keys: []string{"key1"},
		Data: map[string]string{"key1": "val1"},
	}

	result := RenderExplorer(
		nil,
		[]model.Entry{{Name: "mysecret", IsDir: false}},
		nil,
		secret, model.PreviewValues,
		0, 0, nil,
		nil,
		"secret",
		nil, 0,
		"",
		"v0.3.0",
		100, 40,
	)

	assert.Contains(t, result, "key1")
	assert.Contains(t, result, "val1")
}

func TestRenderExplorerSmallDimensions(t *testing.T) {
	assert.NotPanics(t, func() {
		RenderExplorer(
			nil, nil, nil,
			nil, model.PreviewHidden,
			0, 0, nil,
			nil,
			"kv",
			nil, 0,
			"",
			"v1",
			30, 10,
		)
	})
}

func TestRenderExplorerWithHighlightQuery(t *testing.T) {
	current := []model.Entry{
		{Name: "secret-prod", IsDir: false},
		{Name: "secret-dev", IsDir: false},
		{Name: "other-entry", IsDir: false},
	}

	result := RenderExplorer(
		nil, current, nil,
		nil, model.PreviewHidden,
		0, 0, nil,
		nil,
		"secret",
		nil, 0,
		"secret",
		"v1",
		100, 40,
	)

	assert.Contains(t, result, "secret-prod")
	assert.Contains(t, result, "secret-dev")
	assert.Contains(t, result, "other-entry")
}

func TestRenderExplorerNarrowPaddingClamped(t *testing.T) {
	// Very long mount + path causes breadcrumb to exceed width, clamping padding to 1
	result := RenderExplorer(
		nil, nil, nil,
		nil, model.PreviewHidden,
		0, 0, nil,
		[]string{"very/long/path/segment/here/"},
		"a-long-mount-name",
		nil, 0,
		"",
		"v999.999.999",
		40, 30,
	)
	assert.NotEmpty(t, result)
}

func TestRenderExplorerEmptyPath(t *testing.T) {
	result := RenderExplorer(
		nil, nil, nil,
		nil, model.PreviewHidden,
		0, 0, nil,
		nil,
		"secret",
		nil, 0,
		"",
		"v1",
		100, 40,
	)
	assert.Contains(t, result, "secret/")
}

func TestRenderExplorerJSONPreview(t *testing.T) {
	secret := &model.Secret{
		Keys: []string{"k"},
		Data: map[string]string{"k": "v"},
	}
	result := RenderExplorer(
		nil,
		[]model.Entry{{Name: "s", IsDir: false}},
		nil,
		secret, model.PreviewJSON,
		0, 0, nil,
		nil,
		"secret",
		nil, 0,
		"",
		"v1",
		100, 40,
	)
	assert.Contains(t, result, "{")
}

// ---------------------------------------------------------------------------
// renderEntryList (additional branches)
// ---------------------------------------------------------------------------

func TestRenderEntryListSelectedDirBulk(t *testing.T) {
	// Cover the branch: selected[i] && e.IsDir (non-cursor row)
	entries := []model.Entry{
		{Name: "dir1/", IsDir: true},
		{Name: "dir2/", IsDir: true},
		{Name: "file1", IsDir: false},
	}
	selected := map[int]bool{0: true, 2: true}
	result := renderEntryList(entries, 1, selected, 40, 20, true, "")
	assert.Contains(t, result, "dir1/")
	assert.Contains(t, result, "file1")
}

func TestRenderEntryListHighlightOnSelectedRow(t *testing.T) {
	entries := []model.Entry{
		{Name: "secret-key", IsDir: false},
	}
	result := renderEntryList(entries, 0, nil, 50, 20, true, "secret")
	assert.Contains(t, result, "secret")
}

func TestRenderEntryListNarrowWidth(t *testing.T) {
	entries := []model.Entry{
		{Name: "x", IsDir: false},
	}
	// Width 6 means maxW = 6 - 2 - 2 = 2, clamped to 4
	result := renderEntryList(entries, 0, nil, 6, 20, true, "")
	assert.Contains(t, result, "x")
}

// ---------------------------------------------------------------------------
// renderSecretPreview (separator width clamped to 4)
// ---------------------------------------------------------------------------

func TestRenderSecretPreviewMinSepWidth(t *testing.T) {
	secret := &model.Secret{
		Keys: []string{"k"},
		Data: map[string]string{"k": "v"},
	}
	// Width 4 means sepW = 4-2 = 2, clamped to 4
	result := renderSecretPreview(secret, model.PreviewHidden, 4, 10)
	assert.NotEmpty(t, result)
}

// ---------------------------------------------------------------------------
// RenderTabBar (right arrow indicator)
// ---------------------------------------------------------------------------

func TestRenderTabBarRightArrow(t *testing.T) {
	labels := make([]string, 20)
	for i := range labels {
		labels[i] = "tab-" + string(rune('a'+i%26))
	}
	// Active tab at start -- right arrow should appear, no left arrow
	result := RenderTabBar(labels, 0, 60)
	assert.Contains(t, result, "\u25B8") // right arrow
}

// ---------------------------------------------------------------------------
// renderEntryList (pad <= 0 branch, short name does not need padding)
// ---------------------------------------------------------------------------

func TestRenderEntryListShortNameNoPadding(t *testing.T) {
	// Name fills the entire available width so pad <= 0
	entries := []model.Entry{
		{Name: "abcdefghijklmnopqrstuvwxyz0123456789", IsDir: false},
	}
	result := renderEntryList(entries, 0, nil, 40, 20, true, "")
	assert.NotEmpty(t, result)
}

// ---------------------------------------------------------------------------
// RenderExplorer (very small width triggers all column clamps)
// ---------------------------------------------------------------------------

func TestRenderExplorerColumnWidthClamps(t *testing.T) {
	// With width=20, usable=14, leftW=1<10 clamped, midW=7<10 clamped, rightW<10 clamped
	assert.NotPanics(t, func() {
		RenderExplorer(
			nil, nil, nil,
			nil, model.PreviewHidden,
			0, 0, nil,
			nil,
			"kv",
			nil, 0,
			"",
			"v1",
			20, 20,
		)
	})
}
