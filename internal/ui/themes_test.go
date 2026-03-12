package ui

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultThemeName(t *testing.T) {
	assert.Equal(t, "tokyonight", DefaultThemeName())
}

func TestGetThemeValid(t *testing.T) {
	theme, ok := GetTheme("tokyonight")
	require.True(t, ok, "tokyonight should be a valid theme")

	// Verify all fields are populated.
	v := reflect.ValueOf(theme)
	typ := v.Type()
	for i := 0; i < v.NumField(); i++ {
		fieldVal := v.Field(i).String()
		fieldName := typ.Field(i).Name
		assert.NotEmpty(t, fieldVal, "theme field %s should not be empty", fieldName)
	}
}

func TestGetThemeInvalid(t *testing.T) {
	_, ok := GetTheme("nonexistent-theme")
	assert.False(t, ok)
}

func TestGetThemeDefaultExists(t *testing.T) {
	_, ok := GetTheme(DefaultThemeName())
	assert.True(t, ok, "default theme %q should exist in builtinThemes", DefaultThemeName())
}

func TestAllBuiltinThemesHaveAllFields(t *testing.T) {
	for name := range builtinThemes {
		theme, ok := GetTheme(name)
		require.True(t, ok, "theme %q should exist", name)

		v := reflect.ValueOf(theme)
		typ := v.Type()
		for i := 0; i < v.NumField(); i++ {
			fieldVal := v.Field(i).String()
			fieldName := typ.Field(i).Name
			assert.NotEmpty(t, fieldVal, "theme %q field %s should not be empty", name, fieldName)
		}
	}
}

func TestGroupedThemeEntries(t *testing.T) {
	entries := GroupedThemeEntries()
	require.NotEmpty(t, entries)

	// Collect headers and theme names.
	var headers []string
	themeNames := make(map[string]bool)
	headerIndices := make(map[string]int)

	for i, entry := range entries {
		if entry.IsHeader {
			headers = append(headers, entry.Name)
			headerIndices[entry.Name] = i
		} else {
			themeNames[entry.Name] = true
		}
	}

	// Must have both group headers.
	assert.Contains(t, headers, "Dark Themes")
	assert.Contains(t, headers, "Light Themes")

	// Dark Themes header should come before Light Themes header.
	assert.Less(t, headerIndices["Dark Themes"], headerIndices["Light Themes"],
		"Dark Themes header should appear before Light Themes header")

	// All builtin themes should be present.
	for name := range builtinThemes {
		assert.True(t, themeNames[name], "theme %q should be in GroupedThemeEntries", name)
	}

	// Number of non-header entries should match builtinThemes count.
	assert.Equal(t, len(builtinThemes), len(themeNames))
}

func TestGroupedThemeEntriesHeadersBeforeItems(t *testing.T) {
	entries := GroupedThemeEntries()

	// First entry must be a header.
	require.True(t, entries[0].IsHeader, "first entry should be a header")

	// After "Dark Themes" header, items should appear until next header.
	// After "Light Themes" header, light items should appear.
	darkHeaderIdx := -1
	lightHeaderIdx := -1
	for i, entry := range entries {
		if entry.IsHeader && entry.Name == "Dark Themes" {
			darkHeaderIdx = i
		}
		if entry.IsHeader && entry.Name == "Light Themes" {
			lightHeaderIdx = i
		}
	}

	require.NotEqual(t, -1, darkHeaderIdx)
	require.NotEqual(t, -1, lightHeaderIdx)

	// Items between dark header and light header should be dark themes.
	for i := darkHeaderIdx + 1; i < lightHeaderIdx; i++ {
		assert.False(t, entries[i].IsHeader)
		assert.False(t, lightThemes[entries[i].Name],
			"theme %q between Dark Themes header and Light Themes header should not be a light theme", entries[i].Name)
	}

	// Items after light header should be light themes.
	for i := lightHeaderIdx + 1; i < len(entries); i++ {
		assert.False(t, entries[i].IsHeader)
		assert.True(t, lightThemes[entries[i].Name],
			"theme %q after Light Themes header should be a light theme", entries[i].Name)
	}
}

func TestLightThemesCorrectlyClassified(t *testing.T) {
	expectedLight := []string{
		"tokyonight-light",
		"kanagawa-lotus",
		"gruvbox-light",
		"catppuccin-latte",
		"bluloco-light",
	}

	for _, name := range expectedLight {
		assert.True(t, lightThemes[name], "%q should be classified as a light theme", name)
		_, exists := GetTheme(name)
		assert.True(t, exists, "light theme %q should exist in builtinThemes", name)
	}

	// Verify that no other themes are marked as light.
	for name := range builtinThemes {
		isExpectedLight := false
		for _, ln := range expectedLight {
			if name == ln {
				isExpectedLight = true
				break
			}
		}
		if isExpectedLight {
			assert.True(t, lightThemes[name])
		} else {
			assert.False(t, lightThemes[name], "%q should not be classified as a light theme", name)
		}
	}
}

func TestGroupedThemeEntriesSorted(t *testing.T) {
	entries := GroupedThemeEntries()

	// Collect dark and light theme names in order.
	var darkNames, lightNames []string
	inLight := false
	for _, entry := range entries {
		if entry.IsHeader {
			if entry.Name == "Light Themes" {
				inLight = true
			}
			continue
		}
		if inLight {
			lightNames = append(lightNames, entry.Name)
		} else {
			darkNames = append(darkNames, entry.Name)
		}
	}

	// Both groups should be sorted alphabetically.
	assert.IsNonDecreasing(t, darkNames, "dark themes should be sorted")
	assert.IsNonDecreasing(t, lightNames, "light themes should be sorted")
}
