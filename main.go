package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/janosmiko/vau/internal/app"
	"github.com/janosmiko/vau/internal/config"
	"github.com/janosmiko/vau/internal/ui"
	"github.com/janosmiko/vau/internal/vault"
	"github.com/janosmiko/vau/internal/version"
)

const helpText = `vau - A yazi-inspired terminal UI for browsing and editing HashiCorp Vault KV secrets.

Usage:
  vau [flags]

Flags:
  -h, --help       Show this help message
  -v, --version    Print version information

Environment Variables:
  VAULT_ADDR         Vault server address (default: http://127.0.0.1:8200)
  VAULT_TOKEN        Vault authentication token (required)
  VAULT_MOUNT_PATH   KV mount path to open initially (default: secret)
  EDITOR             External editor for JSON editing (falls back to VISUAL, then vim)
  VISUAL             Fallback editor if EDITOR is not set

Config File:
  ~/.config/vau/config.yaml

  Configure editor, colorscheme, keybindings, bookmarks, and custom theme
  colors. Press '?' inside the app for a full keybinding reference.
`

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--version", "-v":
			fmt.Println(version.Full())
			os.Exit(0)
		case "--help", "-h":
			fmt.Print(helpText)
			os.Exit(0)
		}
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Determine colorscheme: config file > default.
	colorscheme := cfg.Colorscheme
	if colorscheme == "" {
		colorscheme = ui.DefaultThemeName()
	}
	cfg.Colorscheme = colorscheme

	ui.ApplyTheme(colorscheme, cfg.Theme)

	client, err := vault.NewClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error connecting to Vault: %v\n", err)
		os.Exit(1)
	}

	m := app.NewModel(client, cfg, version.Short())
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	m.SetProgram(p)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
