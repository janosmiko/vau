package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestBookmarkYAMLRoundTrip(t *testing.T) {
	original := []Bookmark{
		{Name: "prod-secrets", Mount: "secret", Path: "prod/app1"},
		{Name: "dev-db", Mount: "kv", Path: "dev/database"},
	}

	data, err := yaml.Marshal(original)
	require.NoError(t, err)

	var decoded []Bookmark
	err = yaml.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, original, decoded)
}

func TestBookmarkYAMLTags(t *testing.T) {
	yamlData := `
- name: my-bookmark
  mount: secret
  path: foo/bar
`
	var bookmarks []Bookmark
	err := yaml.Unmarshal([]byte(yamlData), &bookmarks)
	require.NoError(t, err)
	require.Len(t, bookmarks, 1)

	assert.Equal(t, "my-bookmark", bookmarks[0].Name)
	assert.Equal(t, "secret", bookmarks[0].Mount)
	assert.Equal(t, "foo/bar", bookmarks[0].Path)
}

func TestBookmarkYAMLUnmarshalEmpty(t *testing.T) {
	var bookmarks []Bookmark
	err := yaml.Unmarshal([]byte("[]"), &bookmarks)
	require.NoError(t, err)
	assert.Empty(t, bookmarks)
}

func TestConfigYAMLUnmarshalAllFields(t *testing.T) {
	yamlData := `
editor: nvim
colorscheme: tokyonight
theme:
  primary: "#ff0000"
  dir: "#00ff00"
  file: "#0000ff"
  selected: "#111111"
  border: "#222222"
  dim: "#333333"
  error: "#444444"
  warn: "#555555"
  breadcrumb: "#666666"
  table_key: "#777777"
  table_value: "#888888"
  table_header: "#999999"
  hidden_value: "#aaaaaa"
  help_key: "#bbbbbb"
  help_desc: "#cccccc"
  status_bar: "#dddddd"
  title: "#eeeeee"
  base64_value: "#ffffff"
bookmarks:
  - name: bm1
    mount: secret
    path: a/b
keybindings:
  quit: Q
  help: "?"
`
	var cfg Config
	err := yaml.Unmarshal([]byte(yamlData), &cfg)
	require.NoError(t, err)

	assert.Equal(t, "nvim", cfg.Editor)
	assert.Equal(t, "tokyonight", cfg.Colorscheme)
	assert.Equal(t, "#ff0000", cfg.Theme.Primary)
	assert.Equal(t, "#00ff00", cfg.Theme.Dir)
	assert.Equal(t, "#0000ff", cfg.Theme.File)
	assert.Equal(t, "#111111", cfg.Theme.Selected)
	assert.Equal(t, "#222222", cfg.Theme.Border)
	assert.Equal(t, "#333333", cfg.Theme.Dim)
	assert.Equal(t, "#444444", cfg.Theme.Error)
	assert.Equal(t, "#555555", cfg.Theme.Warn)
	assert.Equal(t, "#666666", cfg.Theme.Breadcrumb)
	assert.Equal(t, "#777777", cfg.Theme.TableKey)
	assert.Equal(t, "#888888", cfg.Theme.TableValue)
	assert.Equal(t, "#999999", cfg.Theme.TableHeader)
	assert.Equal(t, "#aaaaaa", cfg.Theme.HiddenValue)
	assert.Equal(t, "#bbbbbb", cfg.Theme.HelpKey)
	assert.Equal(t, "#cccccc", cfg.Theme.HelpDesc)
	assert.Equal(t, "#dddddd", cfg.Theme.StatusBar)
	assert.Equal(t, "#eeeeee", cfg.Theme.Title)
	assert.Equal(t, "#ffffff", cfg.Theme.Base64Value)

	require.Len(t, cfg.Bookmarks, 1)
	assert.Equal(t, "bm1", cfg.Bookmarks[0].Name)

	require.Len(t, cfg.Keybindings, 2)
	assert.Equal(t, "Q", cfg.Keybindings["quit"])
	assert.Equal(t, "?", cfg.Keybindings["help"])
}

func TestConfigYAMLUnmarshalDefaults(t *testing.T) {
	// Empty YAML should produce zero-value Config.
	var cfg Config
	err := yaml.Unmarshal([]byte("{}"), &cfg)
	require.NoError(t, err)

	assert.Empty(t, cfg.Editor)
	assert.Empty(t, cfg.Colorscheme)
	assert.Empty(t, cfg.Theme.Primary)
	assert.Nil(t, cfg.Bookmarks)
	assert.Nil(t, cfg.Keybindings)
}

func TestConfigYAMLUnmarshalMissingFields(t *testing.T) {
	yamlData := `editor: vim`
	var cfg Config
	err := yaml.Unmarshal([]byte(yamlData), &cfg)
	require.NoError(t, err)

	assert.Equal(t, "vim", cfg.Editor)
	assert.Empty(t, cfg.Colorscheme)
	assert.Empty(t, cfg.Theme.Primary)
	assert.Nil(t, cfg.Bookmarks)
	assert.Nil(t, cfg.Keybindings)
}

func TestThemeConfigAllFieldsPresent(t *testing.T) {
	tc := ThemeConfig{
		Primary:     "#1",
		Dir:         "#2",
		File:        "#3",
		Selected:    "#4",
		Border:      "#5",
		Dim:         "#6",
		Error:       "#7",
		Warn:        "#8",
		Breadcrumb:  "#9",
		TableKey:    "#a",
		TableValue:  "#b",
		TableHeader: "#c",
		HiddenValue: "#d",
		HelpKey:     "#e",
		HelpDesc:    "#f",
		StatusBar:   "#10",
		Title:       "#11",
		Base64Value: "#12",
	}

	// Verify no field is empty.
	assert.NotEmpty(t, tc.Primary)
	assert.NotEmpty(t, tc.Dir)
	assert.NotEmpty(t, tc.File)
	assert.NotEmpty(t, tc.Selected)
	assert.NotEmpty(t, tc.Border)
	assert.NotEmpty(t, tc.Dim)
	assert.NotEmpty(t, tc.Error)
	assert.NotEmpty(t, tc.Warn)
	assert.NotEmpty(t, tc.Breadcrumb)
	assert.NotEmpty(t, tc.TableKey)
	assert.NotEmpty(t, tc.TableValue)
	assert.NotEmpty(t, tc.TableHeader)
	assert.NotEmpty(t, tc.HiddenValue)
	assert.NotEmpty(t, tc.HelpKey)
	assert.NotEmpty(t, tc.HelpDesc)
	assert.NotEmpty(t, tc.StatusBar)
	assert.NotEmpty(t, tc.Title)
	assert.NotEmpty(t, tc.Base64Value)
}

func TestConfigDirRespectsXDG(t *testing.T) {
	t.Run("uses XDG_CONFIG_HOME when set", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", "/tmp/test-xdg-config")
		dir, err := configDir()
		require.NoError(t, err)
		assert.Equal(t, "/tmp/test-xdg-config/vau", dir)
	})

	t.Run("defaults to ~/.config when XDG_CONFIG_HOME is empty", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", "")
		dir, err := configDir()
		require.NoError(t, err)
		home, _ := os.UserHomeDir()
		assert.Equal(t, filepath.Join(home, ".config", "vau"), dir)
	})
}

func TestStateDirRespectsXDG(t *testing.T) {
	t.Run("uses XDG_STATE_HOME when set", func(t *testing.T) {
		t.Setenv("XDG_STATE_HOME", "/tmp/test-xdg-state")
		dir, err := stateDir()
		require.NoError(t, err)
		assert.Equal(t, "/tmp/test-xdg-state/vau", dir)
	})

	t.Run("defaults to ~/.local/state when XDG_STATE_HOME is empty", func(t *testing.T) {
		t.Setenv("XDG_STATE_HOME", "")
		dir, err := stateDir()
		require.NoError(t, err)
		home, _ := os.UserHomeDir()
		assert.Equal(t, filepath.Join(home, ".local", "state", "vau"), dir)
	})
}

func TestBookmarksPathUsesStateDir(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", "/tmp/test-state")
	path, err := bookmarksPath()
	require.NoError(t, err)
	assert.Equal(t, "/tmp/test-state/vau/bookmarks.yaml", path)
}

func TestBookmarksMigration(t *testing.T) {
	t.Run("migrates from old app name (v)", func(t *testing.T) {
		tmpDir := t.TempDir()
		oldConfigDir := filepath.Join(tmpDir, "config", "v")
		newStateDir := filepath.Join(tmpDir, "state", "vau")

		t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, "config"))
		t.Setenv("XDG_STATE_HOME", filepath.Join(tmpDir, "state"))

		// Write bookmarks at old "v" config location.
		require.NoError(t, os.MkdirAll(oldConfigDir, 0o755))
		legacyData := []byte("- name: legacy\n  mount: secret\n  path: old/path\n")
		require.NoError(t, os.WriteFile(filepath.Join(oldConfigDir, "bookmarks.yaml"), legacyData, 0o644))

		bookmarks, err := LoadBookmarks()
		require.NoError(t, err)
		require.Len(t, bookmarks, 1)
		assert.Equal(t, "legacy", bookmarks[0].Name)

		// New file should exist in state dir.
		_, err = os.Stat(filepath.Join(newStateDir, "bookmarks.yaml"))
		assert.NoError(t, err, "bookmarks should be written to state dir")

		// Old file should be removed.
		_, err = os.Stat(filepath.Join(oldConfigDir, "bookmarks.yaml"))
		assert.True(t, os.IsNotExist(err), "legacy file should be removed")
	})

	t.Run("migrates from vau config dir", func(t *testing.T) {
		tmpDir := t.TempDir()
		oldConfigDir := filepath.Join(tmpDir, "config", "vau")
		newStateDir := filepath.Join(tmpDir, "state", "vau")

		t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, "config"))
		t.Setenv("XDG_STATE_HOME", filepath.Join(tmpDir, "state"))

		// Write bookmarks at vau config location (pre-XDG-state migration).
		require.NoError(t, os.MkdirAll(oldConfigDir, 0o755))
		legacyData := []byte("- name: config-bm\n  mount: kv\n  path: foo/bar\n")
		require.NoError(t, os.WriteFile(filepath.Join(oldConfigDir, "bookmarks.yaml"), legacyData, 0o644))

		bookmarks, err := LoadBookmarks()
		require.NoError(t, err)
		require.Len(t, bookmarks, 1)
		assert.Equal(t, "config-bm", bookmarks[0].Name)

		_, err = os.Stat(filepath.Join(newStateDir, "bookmarks.yaml"))
		assert.NoError(t, err)

		_, err = os.Stat(filepath.Join(oldConfigDir, "bookmarks.yaml"))
		assert.True(t, os.IsNotExist(err))
	})
}

func TestLoadBookmarksNoFiles(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, "config"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(tmpDir, "state"))

	bookmarks, err := LoadBookmarks()
	assert.NoError(t, err)
	assert.Nil(t, bookmarks)
}

func TestSaveAndLoadBookmarks(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", filepath.Join(tmpDir, "state"))

	original := []Bookmark{
		{Name: "test", Mount: "secret", Path: "test/path"},
	}
	require.NoError(t, SaveBookmarks(original))

	loaded, err := LoadBookmarks()
	require.NoError(t, err)
	assert.Equal(t, original, loaded)
}

func TestConfigYAMLRoundTrip(t *testing.T) {
	original := Config{
		Editor:      "code",
		Colorscheme: "nord",
		Theme: ThemeConfig{
			Primary: "#aabbcc",
			Error:   "#ff0000",
		},
		Bookmarks: []Bookmark{
			{Name: "test", Mount: "kv", Path: "test/path"},
		},
		Keybindings: map[string]string{
			"quit": "Q",
		},
	}

	data, err := yaml.Marshal(original)
	require.NoError(t, err)

	var decoded Config
	err = yaml.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, original.Editor, decoded.Editor)
	assert.Equal(t, original.Colorscheme, decoded.Colorscheme)
	assert.Equal(t, original.Theme.Primary, decoded.Theme.Primary)
	assert.Equal(t, original.Theme.Error, decoded.Theme.Error)
	assert.Equal(t, original.Bookmarks, decoded.Bookmarks)
	assert.Equal(t, original.Keybindings, decoded.Keybindings)
}
