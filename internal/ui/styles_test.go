package ui

import (
	"testing"

	"github.com/janosmiko/vau/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestApplyThemeEmptyColorschemeEmptyOverrides(t *testing.T) {
	assert.NotPanics(t, func() {
		ApplyTheme("", config.ThemeConfig{})
	})
}

func TestApplyThemeValidColorscheme(t *testing.T) {
	assert.NotPanics(t, func() {
		ApplyTheme("tokyonight", config.ThemeConfig{})
	})
}

func TestApplyThemeInvalidColorscheme(t *testing.T) {
	assert.NotPanics(t, func() {
		ApplyTheme("this-theme-does-not-exist", config.ThemeConfig{})
	})
}

func TestApplyThemeAllBuiltinSchemes(t *testing.T) {
	for name := range builtinThemes {
		t.Run(name, func(t *testing.T) {
			assert.NotPanics(t, func() {
				ApplyTheme(name, config.ThemeConfig{})
			})
		})
	}
}

func TestApplyThemeWithOverrides(t *testing.T) {
	assert.NotPanics(t, func() {
		ApplyTheme("nord", config.ThemeConfig{
			Primary: "#ff0000",
			Error:   "#00ff00",
		})
	})
}

func TestApplyThemeOnlyOverridesNoColorscheme(t *testing.T) {
	assert.NotPanics(t, func() {
		ApplyTheme("", config.ThemeConfig{
			Primary: "#aabbcc",
			Dir:     "#112233",
		})
	})
}

func TestMergeThemeOverridesTakePrecedence(t *testing.T) {
	base := config.ThemeConfig{
		Primary: "#111111",
		Dir:     "#222222",
		File:    "#333333",
		Error:   "#444444",
	}
	overrides := config.ThemeConfig{
		Primary: "#aaaaaa",
		Error:   "#bbbbbb",
	}

	merged := mergeTheme(base, overrides)

	assert.Equal(t, "#aaaaaa", merged.Primary, "override should take precedence")
	assert.Equal(t, "#bbbbbb", merged.Error, "override should take precedence")
	assert.Equal(t, "#222222", merged.Dir, "non-overridden field should keep base value")
	assert.Equal(t, "#333333", merged.File, "non-overridden field should keep base value")
}

func TestMergeThemeEmptyOverridesKeepBase(t *testing.T) {
	base := config.ThemeConfig{
		Primary:     "#111111",
		Dir:         "#222222",
		File:        "#333333",
		Selected:    "#444444",
		Border:      "#555555",
		Dim:         "#666666",
		Error:       "#777777",
		Warn:        "#888888",
		Breadcrumb:  "#999999",
		TableKey:    "#aaaaaa",
		TableValue:  "#bbbbbb",
		TableHeader: "#cccccc",
		HiddenValue: "#dddddd",
		HelpKey:     "#eeeeee",
		HelpDesc:    "#ffffff",
		StatusBar:   "#101010",
		Title:       "#202020",
		Base64Value: "#303030",
	}
	overrides := config.ThemeConfig{} // all empty

	merged := mergeTheme(base, overrides)
	assert.Equal(t, base, merged, "empty overrides should return base unchanged")
}

func TestMergeThemeAllFieldsOverridden(t *testing.T) {
	base := config.ThemeConfig{
		Primary:     "#000001",
		Dir:         "#000002",
		File:        "#000003",
		Selected:    "#000004",
		Border:      "#000005",
		Dim:         "#000006",
		Error:       "#000007",
		Warn:        "#000008",
		Breadcrumb:  "#000009",
		TableKey:    "#00000a",
		TableValue:  "#00000b",
		TableHeader: "#00000c",
		HiddenValue: "#00000d",
		HelpKey:     "#00000e",
		HelpDesc:    "#00000f",
		StatusBar:   "#000010",
		Title:       "#000011",
		Base64Value: "#000012",
	}
	overrides := config.ThemeConfig{
		Primary:     "#ffff01",
		Dir:         "#ffff02",
		File:        "#ffff03",
		Selected:    "#ffff04",
		Border:      "#ffff05",
		Dim:         "#ffff06",
		Error:       "#ffff07",
		Warn:        "#ffff08",
		Breadcrumb:  "#ffff09",
		TableKey:    "#ffff0a",
		TableValue:  "#ffff0b",
		TableHeader: "#ffff0c",
		HiddenValue: "#ffff0d",
		HelpKey:     "#ffff0e",
		HelpDesc:    "#ffff0f",
		StatusBar:   "#ffff10",
		Title:       "#ffff11",
		Base64Value: "#ffff12",
	}

	merged := mergeTheme(base, overrides)
	assert.Equal(t, overrides, merged, "all overrides should take precedence")
}

func TestMergeThemeEmptyBase(t *testing.T) {
	base := config.ThemeConfig{}
	overrides := config.ThemeConfig{
		Primary: "#aaaaaa",
		Error:   "#bbbbbb",
	}

	merged := mergeTheme(base, overrides)
	assert.Equal(t, "#aaaaaa", merged.Primary)
	assert.Equal(t, "#bbbbbb", merged.Error)
	assert.Empty(t, merged.Dir, "unset fields should remain empty")
}
