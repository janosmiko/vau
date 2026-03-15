# :lock: vau - Vault Navigator

A yazi-inspired terminal UI for browsing and editing HashiCorp Vault KV secrets.

**vau** is a lightning-fast, keyboard-driven, yazi-inspired navigator for your Vault secrets. Supports both **KV v1** and **KV v2** engines with automatic version detection.

![Demo](./docs/imgs/demo.gif)

> **Disclaimer**: This project is largely vibe coded with AI assistance. While it works well for everyday Vault browsing and editing, please review the code and use it at your own risk. Contributions and bug reports are welcome!

> **Disclaimer**: This project is not affiliated with, endorsed by, or associated with HashiCorp in any way.

[![GitHub Sponsors](https://img.shields.io/github/sponsors/janosmiko?style=flat&logo=githubsponsors&label=Sponsors)](https://github.com/sponsors/janosmiko)
[![Buy Me a Coffee](https://img.shields.io/badge/Buy%20Me%20a%20Coffee-support-yellow?style=flat&logo=buymeacoffee)](https://buymeacoffee.com/janosmiko)

## Installation

### Homebrew (macOS / Linux)

```bash
brew install janosmiko/tap/vau
```

### Go install

```bash
go install github.com/janosmiko/vau@latest  # binary installs as "vau"
```

### Build from source

```bash
git clone https://github.com/janosmiko/vau.git
cd vau
go build -o vau .
```

### Debian / Ubuntu

Download the `.deb` package from the [GitHub Releases](https://github.com/janosmiko/vau/releases) page and install it:

```bash
# Replace <version> and <arch> with the appropriate values (e.g., 0.1.0, amd64)
curl -LO https://github.com/janosmiko/vau/releases/download/v<version>/vau_<version>_linux_<arch>.deb
sudo dpkg -i vau_<version>_linux_<arch>.deb
```

### RHEL / Fedora

Download the `.rpm` package from the [GitHub Releases](https://github.com/janosmiko/vau/releases) page and install it:

```bash
# Replace <version> and <arch> with the appropriate values (e.g., 0.1.0, amd64)
curl -LO https://github.com/janosmiko/vau/releases/download/v<version>/vau_<version>_linux_<arch>.rpm
sudo rpm -i vau_<version>_linux_<arch>.rpm
```

### Docker

```bash
docker run --rm -it \
  -e VAULT_ADDR=http://host.docker.internal:8200 \
  -e VAULT_TOKEN=your-token \
  janosmiko/vau
```

To use a config file with Docker:

```bash
docker run --rm -it \
  -e VAULT_ADDR=http://host.docker.internal:8200 \
  -e VAULT_TOKEN=your-token \
  -v ~/.config/vau:/root/.config/vau:ro \
  janosmiko/vau
```

### Download binary

Pre-built binaries for Linux and macOS (amd64/arm64) are available on the [GitHub Releases](https://github.com/janosmiko/vau/releases) page.

## Quick Start

```bash
# Set your Vault connection
export VAULT_ADDR=http://127.0.0.1:8200
export VAULT_TOKEN=your-token

# Launch
vau
```

The TUI opens with a three-column explorer showing your Vault KV mounts. Navigate with `hjkl` (vim-style), press `Enter` to open secrets, and `?` for a full keybinding reference.

## CLI Options

```bash
vau --help       # Show help and available environment variables
vau --version    # Print version information
```

## Features

- **Three-column explorer** - Parent / current / preview layout inspired by [yazi](https://github.com/sxyazi/yazi)
- **KV v1 and v2** - Auto-detects engine version per mount
- **Secret popup** - View, edit, add, and delete key-value pairs inline
- **External editor** - Edit secrets as JSON in your preferred editor (`e` in JSON view, `A` to create)
- **Clipboard** - Copy/paste values to system clipboard (`y`/`p`)
- **Yank, cut, paste** - Move and copy secrets between paths
- **Bulk operations** - Select multiple entries with `Space`, then delete
- **Search** - Jump to first match (`/`)
- **Filter** - Hide non-matching entries (`f`)
- **Jump to path** - Navigate directly to any Vault path with tab completion (`J`)
- **Undo / redo** - Reverse destructive actions (`u` / `Ctrl+R`)
- **Tabs** - Multiple tabs with independent navigation state (`t` to create, `[`/`]` to switch)
- **Bookmarks** - Save and jump to frequently used paths (`B` to save, `b` to browse)
- **Version history** - Browse secret versions for KV v2 mounts (`H`)
- **Base64 decode** - Toggle decoded view for base64-encoded values (`b` in secret popup)
- **Mouse support** - Click entries, tabs, and scroll with the mouse wheel
- **Built-in colorschemes** - 16 themes including Tokyo Night, Kanagawa, Nord, Gruvbox, Dracula, Catppuccin, Bluloco (`T` to pick at runtime)
- **Configurable** - Custom editor, colorscheme, and keybindings via `~/.config/vau/config.yaml`
- **Cursor memory** - Remembers cursor position per directory (yazi-style)
- **Destructive action safety** - Delete operations require typing `DELETE` to confirm
- **Help screen** - Scrollable, searchable keybinding reference (`?`)

## Layout

```
 +-- Parent -------+-- Current ------------+-- Preview ------------------+
 | secret/         | > app/                | config                      |
 | kv/             |   infra/              | credentials                 |
 |                 |   mykey               | db-password                 |
 |                 |                       |                             |
 +-----------------+-----------------------+-----------------------------+
```

- **Left column**: Parent directory contents (or available mounts at root level)
- **Middle column**: Current directory entries
- **Right column**: Preview of the selected item (directory children or secret key-value pairs)

## Keybindings

### Explorer Mode

| Key | Action |
|---|---|
| `h` / `Left` | Navigate to parent directory (or mount list at root) |
| `j` / `Down` / `Ctrl+N` | Move cursor down |
| `k` / `Up` / `Ctrl+P` | Move cursor up |
| `l` / `Right` | Enter directory (dirs only) |
| `Enter` | Enter directory or open secret popup |
| `g` | Go to top of list |
| `G` | Go to bottom of list |
| `v` | Toggle secret values in preview pane |
| `V` | Toggle JSON view in preview pane |
| `Space` | Toggle selection (for bulk operations) |
| `/` | Search - jump to first match |
| `f` | Filter - hide non-matching entries |
| `J` | Jump to Vault path (with tab completion) |
| `e` | Edit secret in external editor |
| `a` | Create new secret (inline editor) |
| `A` | Create new secret (external editor) |
| `r` | Rename / move secret or directory |
| `y` | Yank (copy) secret data |
| `p` | Paste yanked secret to current directory |
| `x` | Cut (yank for move - paste will delete source) |
| `D` | Delete secret/directory (recursive, with confirmation) |
| `u` | Undo last action |
| `Ctrl+R` | Redo previous undo |
| `Ctrl+D` | Half-page scroll down |
| `Ctrl+U` | Half-page scroll up |
| `Ctrl+F` | Full-page scroll down |
| `Ctrl+B` | Full-page scroll up |
| `R` | Refresh listing |
| `t` | New tab (clone current) |
| `[` / `]` | Switch to previous / next tab |
| `Ctrl+C` | Close tab (quit if last) |
| `b` | Show bookmarks |
| `B` | Save current location as bookmark |
| `T` | Change colorscheme |
| `?` | Show help screen |
| `q` | Quit |

### Mount Selection

Press `h` at the root of a mount to enter mount selection.

| Key | Action |
|---|---|
| `j` / `k` | Navigate mounts |
| `l` / `Enter` | Select mount |

### Secret Popup

Opened by pressing `Enter` on a secret (or `e` in explorer). Appears as a centered overlay.

| Key | Action |
|---|---|
| `j` / `k` | Navigate keys |
| `v` / `Tab` | Toggle value visibility (hidden by default) |
| `V` | Toggle JSON view |
| `b` | Toggle base64 decode for selected value |
| `y` | Copy selected value to system clipboard |
| `p` | Paste system clipboard as selected value |
| `e` | Edit selected value inline, or open external editor in JSON view |
| `a` | Add new key-value pair (inline) |
| `H` | View secret version history (KV v2 only) |
| `D` | Delete selected key-value pair (with confirmation) |
| `Ctrl+D` | Half-page scroll down |
| `Ctrl+U` | Half-page scroll up |
| `Ctrl+F` | Full-page scroll down |
| `Ctrl+B` | Full-page scroll up |
| `Esc` / `q` / `h` | Back to explorer |

### Inline Edit (Secret Popup)

When editing a key-value pair (`e`) or adding a new one (`a`):

| Key | Action |
|---|---|
| `Tab` / `Shift+Tab` | Switch between key and value columns |
| `Enter` | Save changes |
| `Esc` | Cancel edit |

### Bookmarks (`b`)

Opens an overlay listing all saved bookmarks.

| Key | Action |
|---|---|
| `j` / `k` | Navigate bookmarks |
| `Enter` / `l` | Jump to selected bookmark |
| `/` | Filter bookmarks |
| `d` | Delete selected bookmark |
| `D` | Delete all bookmarks |
| `Esc` | Close (clears filter first, then closes) |

### Colorscheme Picker (`T`)

Opens an overlay to switch between built-in colorschemes, grouped by dark and light. The theme previews live as you navigate.

| Key | Action |
|---|---|
| `j` / `k` | Navigate themes |
| `Enter` | Apply theme (runtime only) |
| `Esc` | Cancel (revert to previous theme) |

### Search (`/`)

Opens an overlay. The cursor jumps to the first matching entry as you type.

| Key | Action |
|---|---|
| *(type)* | Jump cursor to first match |
| `Enter` | Confirm - stay at matched position |
| `Esc` | Cancel - restore original cursor position |

### Filter (`f`)

Opens an overlay. Non-matching entries are hidden from the list as you type.

| Key | Action |
|---|---|
| *(type)* | Live filter entries |
| `Enter` | Confirm - keep filter active |
| `Esc` | Cancel - clear filter and restore list |

### Jump to Path (`J`)

Opens an overlay. Type a Vault path to navigate directly to it.

| Key | Action |
|---|---|
| *(type)* | Enter a Vault path |
| `Tab` | Autocomplete path segment |
| `Shift+Tab` | Cycle completions backward |
| `Enter` | Navigate to path |
| `Esc` | Cancel |

### Help (`?`)

Opens a scrollable overlay with all keybindings.

| Key | Action |
|---|---|
| `j` / `k` | Scroll up / down |
| `Ctrl+D` / `Ctrl+U` | Half-page scroll down / up |
| `Ctrl+F` / `Ctrl+B` | Full-page scroll down / up |
| `g` / `G` | Go to top / bottom |
| `/` | Search keybindings |
| `Esc` / `?` / `q` | Close help |

### Mouse

| Action | Effect |
|---|---|
| Click entry | Select entry in current column |
| Click tab | Switch to that tab |
| Scroll wheel | Move cursor up/down |

## Configuration

### Environment Variables

| Variable | Description | Default |
|---|---|---|
| `VAULT_ADDR` | Vault server address | `http://127.0.0.1:8200` |
| `VAULT_TOKEN` | Vault authentication token | (required) |
| `VAULT_MOUNT_PATH` | KV mount path (initial) | `secret` |
| `EDITOR` | External editor for JSON editing | (falls back to `VISUAL`, then `vim`) |
| `VISUAL` | Fallback editor if `EDITOR` is not set | `vim` |

### Config File

Place a config file at `~/.config/vau/config.yaml` to customize behavior:

```yaml
# Override the editor used for secret editing
editor: nvim

# Select a built-in colorscheme (press T in the app to browse and preview)
# Available: tokyonight, tokyonight-storm, tokyonight-light, kanagawa,
#   kanagawa-dragon, kanagawa-lotus, nord, gruvbox-dark, gruvbox-light,
#   dracula, catppuccin-mocha, catppuccin-macchiato, catppuccin-frappe,
#   catppuccin-latte, bluloco-dark, bluloco-light
colorscheme: kanagawa

# Override individual colors on top of the colorscheme (all fields optional, hex colors)
theme:
  primary: "#7aa2f7"
  dir: "#73daca"
  file: "#c0caf5"
  selected: "#1a1b26"
  border: "#3b4261"
  dim: "#565f89"
  error: "#f7768e"
  warn: "#e0af68"
  breadcrumb: "#7aa2f7"
  table_key: "#9ece6a"
  table_value: "#c0caf5"
  table_header: "#7aa2f7"
  hidden_value: "#565f89"
  help_key: "#9ece6a"
  help_desc: "#565f89"
  status_bar: "#565f89"
  title: "#7aa2f7"
  base64_value: "#bb9af7"

# Pre-configure bookmarks
bookmarks:
  - name: "Production DB creds"
    mount: "secret"
    path: "production/database/"
  - name: "Staging apps"
    mount: "secret"
    path: "staging/apps/"

# Override default keybindings (action: key)
keybindings:
  search: "s"
  filter: "/"
  jump_path: "S"
  delete: "ctrl+d"
  bookmark_save: "M"
  bookmark_show: "m"
```

The default theme is Tokyo Night. Only specify colors you want to change - unset fields keep the defaults.

### Keybinding Configuration

The `keybindings` section maps action names to key strings. Each override replaces the default binding for that action. Available action names:

| Action Name | Default Key(s) | Description |
|---|---|---|
| `navigate_up` | `j`, `down`, `ctrl+n` | Move cursor up in list |
| `navigate_down` | `k`, `up`, `ctrl+p` | Move cursor down in list |
| `navigate_left` | `h`, `left` | Go to parent directory |
| `navigate_right` | `l`, `right` | Enter directory |
| `open` | `enter` | Open entry |
| `top` | `g` | Jump to top of list |
| `bottom` | `G` | Jump to bottom of list |
| `half_down` | `ctrl+d` | Half-page scroll down |
| `half_up` | `ctrl+u` | Half-page scroll up |
| `full_down` | `ctrl+f` | Full-page scroll down |
| `full_up` | `ctrl+b` | Full-page scroll up |
| `search` | `/` | Search (jump to match) |
| `filter` | `f` | Filter entries |
| `jump_path` | `J` | Jump to Vault path |
| `new_secret` | `a` | Create new secret (inline) |
| `new_secret_editor` | `A` | Create new secret (external editor) |
| `edit` | `e` | Edit secret |
| `rename` | `r` | Rename / move |
| `delete` | `D` | Delete (with confirmation) |
| `yank` | `y` | Yank (copy) secret |
| `paste` | `p` | Paste yanked secret |
| `cut` | `x` | Cut (yank for move) |
| `undo` | `u` | Undo last action |
| `redo` | `ctrl+r` | Redo previous undo |
| `refresh` | `R` | Refresh listing |
| `toggle_values` | `v` | Toggle value visibility |
| `toggle_json` | `V` | Toggle JSON view |
| `select` | `space` | Toggle selection |
| `help` | `?` | Show help screen |
| `quit` | `q` | Quit |
| `new_tab` | `t` | Create new tab |
| `next_tab` | `]` | Switch to next tab |
| `prev_tab` | `[` | Switch to previous tab |
| `close_tab` | `ctrl+c` | Close tab (quit if last) |
| `bookmark_save` | `B` | Save current location as bookmark |
| `bookmark_show` | `b` | Show bookmarks overlay |
| `theme_picker` | `T` | Open colorscheme picker |

When overriding, you specify a single key string that replaces all default bindings for that action.

### Colorschemes

Press `T` in the explorer to browse and live-preview themes. Runtime changes are not persisted - to make a theme permanent, set `colorscheme` in your config file.

Built-in colorschemes:

| Name | Description |
|---|---|
| `tokyonight` | Tokyo Night (default) |
| `tokyonight-storm` | Tokyo Night Storm |
| `tokyonight-light` | Tokyo Night Light |
| `kanagawa` | Kanagawa |
| `kanagawa-dragon` | Kanagawa Dragon |
| `kanagawa-lotus` | Kanagawa Lotus |
| `nord` | Nord |
| `gruvbox-dark` | Gruvbox Dark |
| `gruvbox-light` | Gruvbox Light |
| `dracula` | Dracula |
| `catppuccin-mocha` | Catppuccin Mocha |
| `catppuccin-macchiato` | Catppuccin Macchiato |
| `catppuccin-frappe` | Catppuccin Frappe |
| `catppuccin-latte` | Catppuccin Latte |
| `bluloco-dark` | Bluloco Dark |
| `bluloco-light` | Bluloco Light |

You can further customize individual colors on top of any colorscheme using the `theme` section in your config.

### Bookmarks

Bookmarks are saved automatically to `~/.local/state/vau/bookmarks.yaml` (following the XDG specification). You can also pre-configure bookmarks in your config file (see example above). Pre-configured bookmarks are merged with runtime bookmarks on startup.

## Clipboard

- **macOS**: Uses `pbcopy` / `pbpaste`
- **Linux**: Uses `xclip` or `xsel` (whichever is available)

## Project Structure

```
+-- main.go                  # Entry point
+-- internal/
|   +-- app/
|   |   +-- app.go           # Main bubbletea model, Update, View, keybindings
|   |   +-- keys.go          # Keybinding definitions and overrides
|   |   +-- clipboard.go     # System clipboard (pbcopy/xclip)
|   +-- config/
|   |   +-- config.go        # Config file, bookmarks, and theme loading
|   +-- vault/
|   |   +-- client.go        # Vault KV v1/v2 API wrapper
|   +-- ui/
|   |   +-- explorer.go      # Three-column explorer renderer
|   |   +-- secretview.go    # Secret detail table renderer
|   |   +-- overlay.go       # Centered overlay dialogs
|   |   +-- styles.go        # Lipgloss styles (Tokyo Night theme)
|   |   +-- help.go          # Help screen and context-sensitive help bar
|   +-- model/
|       +-- types.go         # Shared types (Entry, Secret, ViewMode)
```

## Contributing

Contributions are welcome! Here's how to get started:

### Development Setup

1. **Prerequisites**: Go 1.24+ and a running Vault instance

2. **Clone and build**:
   ```bash
   git clone https://github.com/janosmiko/vau.git
   cd v
   go build -o vau .
   ```

3. **Run against a dev Vault**:
   ```bash
   # Start a dev Vault server (in another terminal)
   vault server -dev

   # Set environment
   export VAULT_ADDR=http://127.0.0.1:8200
   export VAULT_TOKEN=<root-token-from-dev-server>

   # Run
   ./vau
   ```

4. **Run tests**:
   ```bash
   go test ./...
   ```

### Guidelines

- Keep changes focused - one feature or fix per PR
- Follow existing code style and patterns
- Test against both KV v1 and v2 mounts when touching Vault logic
- Update keybinding help in `internal/ui/help.go` if adding new shortcuts

## Support

If you find vau useful, consider supporting its development:

- [GitHub Sponsors](https://github.com/sponsors/janosmiko)
- [Buy Me a Coffee](https://buymeacoffee.com/janosmiko)

## License

MIT
