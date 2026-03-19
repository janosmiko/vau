package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Bookmark represents a saved Vault location (mount + path).
type Bookmark struct {
	Name  string `yaml:"name"`
	Mount string `yaml:"mount"`
	Path  string `yaml:"path"`
	Slot  string `yaml:"slot,omitempty"` // optional single-char slot (a-z, 0-9)
}

// Config holds user configuration loaded from ~/.config/vau/config.yaml.
type Config struct {
	Editor      string            `yaml:"editor"`      // override $EDITOR
	Colorscheme string            `yaml:"colorscheme"` // built-in colorscheme name
	Theme       ThemeConfig       `yaml:"theme"`
	Bookmarks   []Bookmark        `yaml:"bookmarks"`   // pre-configured bookmarks
	Keybindings map[string]string `yaml:"keybindings"` // action name -> key string overrides
}

// ThemeConfig allows customizing the color palette via hex strings (e.g. "#ff0000").
// Empty fields leave the default colors unchanged.
//
// Example config.yaml:
//
//	editor: nvim
//
//	theme:
//	  primary: "#7aa2f7"
//	  dir: "#73daca"
//	  file: "#c0caf5"
//	  selected: "#1a1b26"
//	  border: "#3b4261"
//	  dim: "#565f89"
//	  error: "#f7768e"
//	  warn: "#e0af68"
type ThemeConfig struct {
	Primary     string `yaml:"primary"`
	Dir         string `yaml:"dir"`
	File        string `yaml:"file"`
	Selected    string `yaml:"selected"`
	Border      string `yaml:"border"`
	Dim         string `yaml:"dim"`
	Error       string `yaml:"error"`
	Warn        string `yaml:"warn"`
	Breadcrumb  string `yaml:"breadcrumb"`
	TableKey    string `yaml:"table_key"`
	TableValue  string `yaml:"table_value"`
	TableHeader string `yaml:"table_header"`
	HiddenValue string `yaml:"hidden_value"`
	HelpKey     string `yaml:"help_key"`
	HelpDesc    string `yaml:"help_desc"`
	StatusBar   string `yaml:"status_bar"`
	Title       string `yaml:"title"`
	Base64Value string `yaml:"base64_value"`
}

// configDir returns the config directory path: $XDG_CONFIG_HOME/vau/ (default ~/.config/vau/).
func configDir() (string, error) {
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, "vau"), nil
}

// stateDir returns the state directory path: $XDG_STATE_HOME/vau/ (default ~/.local/state/vau/).
func stateDir() (string, error) {
	dir := os.Getenv("XDG_STATE_HOME")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		dir = filepath.Join(home, ".local", "state")
	}
	return filepath.Join(dir, "vau"), nil
}

// mustConfigBase returns the XDG config base directory ($XDG_CONFIG_HOME or ~/.config).
// Used by migration logic to check legacy paths.
func mustConfigBase() string {
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, ".config")
	}
	return dir
}

// configPath returns the default config file path: ~/.config/vau/config.yaml.
func configPath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.yaml"), nil
}

// bookmarksPath returns the bookmarks file path: $XDG_STATE_HOME/vau/bookmarks.yaml.
func bookmarksPath() (string, error) {
	dir, err := stateDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "bookmarks.yaml"), nil
}

// Load reads the config from ~/.config/vau/config.yaml.
// If the file does not exist, it returns a default (empty) Config.
func Load() (*Config, error) {
	path, err := configPath()
	if err != nil {
		return &Config{}, nil //nolint:nilerr // graceful fallback to defaults
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// LoadBookmarks reads saved bookmarks from $XDG_STATE_HOME/vau/bookmarks.yaml.
// If the file does not exist at the new location but exists at the legacy
// ~/.config/v/bookmarks.yaml path, it migrates the file automatically.
// If neither file exists, it returns an empty slice.
func LoadBookmarks() ([]Bookmark, error) {
	path, err := bookmarksPath()
	if err != nil {
		return nil, nil //nolint:nilerr // graceful fallback to empty bookmarks
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}

		// Attempt migration from legacy config locations.
		// Check both old app name ("v") and current name ("vau") config dirs.
		var oldPath string
		for _, legacyName := range []string{"vau", "v"} {
			candidate := filepath.Join(mustConfigBase(), legacyName, "bookmarks.yaml")
			if _, statErr := os.Stat(candidate); statErr == nil {
				oldPath = candidate
				break
			}
		}
		if oldPath == "" {
			return nil, nil
		}

		data, err = os.ReadFile(oldPath)
		if err != nil {
			return nil, err
		}

		// Migrate: write to new state location, verify, then remove old file.
		dir := filepath.Dir(path)
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return nil, err
		}
		if err := os.WriteFile(path, data, 0o600); err != nil {
			return nil, err
		}
		// Verify the written file matches before deleting the old one.
		written, err := os.ReadFile(path)
		if err != nil || len(written) != len(data) {
			return nil, fmt.Errorf("bookmark migration verification failed")
		}
		_ = os.Remove(oldPath)
	}

	var bookmarks []Bookmark
	if err := yaml.Unmarshal(data, &bookmarks); err != nil {
		return nil, err
	}
	return bookmarks, nil
}

// SaveBookmarks writes bookmarks to $XDG_STATE_HOME/vau/bookmarks.yaml.
func SaveBookmarks(bookmarks []Bookmark) error {
	path, err := bookmarksPath()
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}

	data, err := yaml.Marshal(bookmarks)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}
