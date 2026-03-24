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

// ---------------------------------------------------------------------------
// configPath
// ---------------------------------------------------------------------------

func TestConfigPathReturnsYAML(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/tmp/test-cfg")
	path, err := configPath()
	require.NoError(t, err)
	assert.Equal(t, "/tmp/test-cfg/vau/config.yaml", path)
	assert.True(t, filepath.Ext(path) == ".yaml")
}

func TestConfigPathDefaultsWithoutXDG(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")
	path, err := configPath()
	require.NoError(t, err)

	home, _ := os.UserHomeDir()
	assert.Equal(t, filepath.Join(home, ".config", "vau", "config.yaml"), path)
}

// ---------------------------------------------------------------------------
// mustConfigBase
// ---------------------------------------------------------------------------

func TestMustConfigBaseRespectsXDG(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/custom/config")
	assert.Equal(t, "/custom/config", mustConfigBase())
}

func TestMustConfigBaseDefaultsWithoutXDG(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")
	home, _ := os.UserHomeDir()
	assert.Equal(t, filepath.Join(home, ".config"), mustConfigBase())
}

// ---------------------------------------------------------------------------
// Load
// ---------------------------------------------------------------------------

func TestLoadValidConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configDirPath := filepath.Join(tmpDir, "vau")
	require.NoError(t, os.MkdirAll(configDirPath, 0o755))

	configContent := []byte(`editor: nvim
colorscheme: tokyonight
theme:
  primary: "#aabbcc"
bookmarks:
  - name: b1
    mount: secret
    path: a/b
keybindings:
  quit: Q
`)
	require.NoError(t, os.WriteFile(filepath.Join(configDirPath, "config.yaml"), configContent, 0o644))

	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	cfg, err := Load()
	require.NoError(t, err)
	assert.Equal(t, "nvim", cfg.Editor)
	assert.Equal(t, "tokyonight", cfg.Colorscheme)
	assert.Equal(t, "#aabbcc", cfg.Theme.Primary)
	require.Len(t, cfg.Bookmarks, 1)
	assert.Equal(t, "b1", cfg.Bookmarks[0].Name)
	assert.Equal(t, "Q", cfg.Keybindings["quit"])
}

func TestLoadNonExistentFileReturnsDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	cfg, err := Load()
	require.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Empty(t, cfg.Editor)
	assert.Empty(t, cfg.Colorscheme)
	assert.Nil(t, cfg.Bookmarks)
	assert.Nil(t, cfg.Keybindings)
}

func TestLoadInvalidYAMLReturnsError(t *testing.T) {
	tmpDir := t.TempDir()
	configDirPath := filepath.Join(tmpDir, "vau")
	require.NoError(t, os.MkdirAll(configDirPath, 0o755))

	badContent := []byte("editor: [invalid yaml\n  - broken")
	require.NoError(t, os.WriteFile(filepath.Join(configDirPath, "config.yaml"), badContent, 0o644))

	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	cfg, err := Load()
	assert.Error(t, err)
	assert.Nil(t, cfg)
}

func TestLoadEmptyFileReturnsDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	configDirPath := filepath.Join(tmpDir, "vau")
	require.NoError(t, os.MkdirAll(configDirPath, 0o755))

	require.NoError(t, os.WriteFile(filepath.Join(configDirPath, "config.yaml"), []byte(""), 0o644))

	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	cfg, err := Load()
	require.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Empty(t, cfg.Editor)
}

func TestLoadReadPermissionError(t *testing.T) {
	tmpDir := t.TempDir()
	configDirPath := filepath.Join(tmpDir, "vau")
	require.NoError(t, os.MkdirAll(configDirPath, 0o755))

	configFile := filepath.Join(configDirPath, "config.yaml")
	require.NoError(t, os.WriteFile(configFile, []byte("editor: vim"), 0o600))
	// Remove read permissions.
	require.NoError(t, os.Chmod(configFile, 0o000))
	t.Cleanup(func() { _ = os.Chmod(configFile, 0o600) })

	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	cfg, err := Load()
	assert.Error(t, err)
	assert.Nil(t, cfg)
}

// ---------------------------------------------------------------------------
// LoadBookmarks edge cases
// ---------------------------------------------------------------------------

func TestLoadBookmarksCorruptYAML(t *testing.T) {
	tmpDir := t.TempDir()
	stateVauDir := filepath.Join(tmpDir, "state", "vau")
	require.NoError(t, os.MkdirAll(stateVauDir, 0o755))

	badContent := []byte("- name: [broken\n  invalid yaml here")
	require.NoError(t, os.WriteFile(filepath.Join(stateVauDir, "bookmarks.yaml"), badContent, 0o644))

	t.Setenv("XDG_STATE_HOME", filepath.Join(tmpDir, "state"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, "config"))

	bookmarks, err := LoadBookmarks()
	assert.Error(t, err)
	assert.Nil(t, bookmarks)
}

func TestLoadBookmarksEmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	stateVauDir := filepath.Join(tmpDir, "state", "vau")
	require.NoError(t, os.MkdirAll(stateVauDir, 0o755))

	require.NoError(t, os.WriteFile(filepath.Join(stateVauDir, "bookmarks.yaml"), []byte(""), 0o644))

	t.Setenv("XDG_STATE_HOME", filepath.Join(tmpDir, "state"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, "config"))

	bookmarks, err := LoadBookmarks()
	require.NoError(t, err)
	assert.Nil(t, bookmarks)
}

func TestLoadBookmarksEmptyArray(t *testing.T) {
	tmpDir := t.TempDir()
	stateVauDir := filepath.Join(tmpDir, "state", "vau")
	require.NoError(t, os.MkdirAll(stateVauDir, 0o755))

	require.NoError(t, os.WriteFile(filepath.Join(stateVauDir, "bookmarks.yaml"), []byte("[]"), 0o644))

	t.Setenv("XDG_STATE_HOME", filepath.Join(tmpDir, "state"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, "config"))

	bookmarks, err := LoadBookmarks()
	require.NoError(t, err)
	assert.Empty(t, bookmarks)
}

func TestLoadBookmarksReadPermissionError(t *testing.T) {
	tmpDir := t.TempDir()
	stateVauDir := filepath.Join(tmpDir, "state", "vau")
	require.NoError(t, os.MkdirAll(stateVauDir, 0o755))

	bmFile := filepath.Join(stateVauDir, "bookmarks.yaml")
	require.NoError(t, os.WriteFile(bmFile, []byte("- name: x\n  mount: s\n  path: p\n"), 0o600))
	require.NoError(t, os.Chmod(bmFile, 0o000))
	t.Cleanup(func() { _ = os.Chmod(bmFile, 0o600) })

	t.Setenv("XDG_STATE_HOME", filepath.Join(tmpDir, "state"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, "config"))

	bookmarks, err := LoadBookmarks()
	assert.Error(t, err)
	assert.Nil(t, bookmarks)
}

// ---------------------------------------------------------------------------
// SaveBookmarks edge cases
// ---------------------------------------------------------------------------

func TestSaveBookmarksNil(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", filepath.Join(tmpDir, "state"))

	require.NoError(t, SaveBookmarks(nil))

	// Verify the saved file can be loaded back.
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, "config"))
	loaded, err := LoadBookmarks()
	require.NoError(t, err)
	assert.Empty(t, loaded)
}

func TestSaveBookmarksEmptySlice(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", filepath.Join(tmpDir, "state"))

	require.NoError(t, SaveBookmarks([]Bookmark{}))

	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, "config"))
	loaded, err := LoadBookmarks()
	require.NoError(t, err)
	assert.Empty(t, loaded)
}

func TestSaveBookmarksWithSlotField(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", filepath.Join(tmpDir, "state"))

	original := []Bookmark{
		{Name: "slotted", Mount: "kv", Path: "a/b", Slot: "a"},
		{Name: "unslotted", Mount: "secret", Path: "c/d"},
	}
	require.NoError(t, SaveBookmarks(original))

	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, "config"))
	loaded, err := LoadBookmarks()
	require.NoError(t, err)
	require.Len(t, loaded, 2)
	assert.Equal(t, "a", loaded[0].Slot)
	assert.Empty(t, loaded[1].Slot)
}

func TestSaveBookmarksCreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", filepath.Join(tmpDir, "state"))

	// state/vau/ does not exist yet -- SaveBookmarks should create it.
	require.NoError(t, SaveBookmarks([]Bookmark{
		{Name: "test", Mount: "m", Path: "p"},
	}))

	// Verify the directory was created.
	info, err := os.Stat(filepath.Join(tmpDir, "state", "vau"))
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestSaveBookmarksOverwritesExisting(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", filepath.Join(tmpDir, "state"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, "config"))

	first := []Bookmark{{Name: "first", Mount: "s", Path: "a"}}
	require.NoError(t, SaveBookmarks(first))

	second := []Bookmark{{Name: "second", Mount: "kv", Path: "b"}}
	require.NoError(t, SaveBookmarks(second))

	loaded, err := LoadBookmarks()
	require.NoError(t, err)
	require.Len(t, loaded, 1)
	assert.Equal(t, "second", loaded[0].Name)
}

// ---------------------------------------------------------------------------
// Original round-trip test
// ---------------------------------------------------------------------------

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
