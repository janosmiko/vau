package app

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/janosmiko/vau/internal/config"
	"github.com/janosmiko/vau/internal/model"
	"github.com/janosmiko/vau/internal/ui"
	"github.com/janosmiko/vau/internal/vault"
)

// Messages
type (
	listResultMsg struct {
		path    string
		entries []model.Entry
		err     error
	}
	secretResultMsg struct {
		path       string
		secret     *model.Secret
		err        error
		openPopup  bool // true = open in popup, false = preview
		openEditor bool // true = open in external editor directly
	}
	mountsResultMsg struct {
		mounts []string
		err    error
	}
	statusMsg         string
	errorMsg          string
	confirmCreateMsg  string // path to create after overwrite confirmation
	undoableStatusMsg struct {
		status       string
		undo         model.UndoAction
		reloadSecret *model.Secret // optional: updated secret to apply to m.secret
	}
	redoableStatusMsg struct {
		status       string
		redo         model.UndoAction
		reloadSecret *model.Secret // optional: updated secret to apply to m.secret
	}
	versionHistoryMsg struct {
		path     string
		versions []model.SecretVersion
		err      error
	}
	versionDetailMsg struct {
		version int
		secret  *model.Secret
		err     error
	}
	editorResultMsg struct {
		data          map[string]string
		newSecretPath string // non-empty when creating a new secret via editor
	}
	newSecretEditorMsg     string // path for new secret to open in editor
	confirmCreateEditorMsg string // path to create after overwrite confirmation (editor flow)
	newSecretInlineMsg     struct {
		secret *model.Secret
	}
	yankResultMsg struct {
		secrets []*model.Secret
		paths   []string
		isCut   bool
		isDir   bool
	}
)

// TabState holds per-tab navigation state.
type TabState struct {
	path           []string
	entries        []model.Entry
	cursor         int
	parentList     []model.Entry
	cursorMemory   map[string]int
	previewEntries []model.Entry
	previewSecret  *model.Secret
	previewMode    model.PreviewMode
	selected       map[int]bool
	filterQuery    string
	filteredIdx    []int
	atMountLevel   bool
	mountCursor    int
	mount          string // which mount this tab is using
}

// Model is the main application model.
type Model struct {
	client  *vault.Client
	config  *config.Config
	keys    KeyMap
	version string

	// Tabs
	tabs      []TabState
	activeTab int

	// Dimensions
	width  int
	height int

	// Mount-level state
	atMountLevel bool
	mounts       []string
	mountCursor  int

	// Explorer state
	path         []string // current path segments (each ends with / for dirs)
	entries      []model.Entry
	cursor       int
	parentList   []model.Entry
	cursorMemory map[string]int // remembered cursor position per directory path

	// Preview state
	previewEntries []model.Entry
	previewSecret  *model.Secret
	previewMode    model.PreviewMode

	// Secret view state
	mode              model.ViewMode
	secret            *model.Secret
	secretCursor      int
	revealed          map[string]bool
	secretBase64      map[string]bool // tracks base64 decode toggle per key
	secretAllRevealed bool
	secretJSONView    bool
	secretEditKey     string // key being inline-edited
	secretEditColumn  int    // 0=key, 1=value column being edited
	secretEditOrigKey string // original key name before key-column edit (for rename)

	// Bulk selection
	selected map[int]bool // selected entry indices (in unfiltered entries list)

	// Version history
	versionHistory []model.SecretVersion
	versionCursor  int
	versionPath    string // path of the secret whose history we're viewing

	// Confirm dialog
	confirmMsg      string
	confirmAction   func() tea.Cmd
	prevConfirmMode model.ViewMode // mode to return to after confirm
	confirmInput    textinput.Model

	// Input prompt
	inputAction model.InputAction
	inputLabel  string
	inputBuffer string // for multi-step inputs (e.g., new key then value)
	textInput   textinput.Model

	// Clipboard: stores yanked secret data for single or bulk operations
	yankedSecrets []*model.Secret // yanked secrets (one or more)
	yankPaths     []string        // paths of yanked items (for directory operations)
	yankIsCut     bool            // true if yanked via cut (x) — paste will delete source
	yankIsDir     bool            // true if yanked items include directories

	// Search (s = jump-to) and Filter (/ = hide non-matching)
	searchInput textinput.Model
	priorCursor int    // cursor before search started
	filterQuery string // active filter text (empty = no filter)
	filteredIdx []int  // indices into m.entries matching filter

	// Status/error messages
	status string
	errMsg string

	// Help screen state
	helpScroll    int
	helpFilter    string
	helpSearching bool

	// Jump-to-path state (S)
	jumpCompletions []string // available completions for current input
	jumpCompIdx     int      // index into jumpCompletions (-1 = no selection)
	jumpLastInput   string   // last input value used to load completions

	// Undo/redo stacks
	undoStack []model.UndoAction
	redoStack []model.UndoAction

	// Bookmarks
	bookmarks         []config.Bookmark
	bookmarkFilter    string
	bookmarkCursor    int
	bookmarkSearching bool // true when typing in the filter input

	// Mark pending state (vim-style two-key sequences)
	markPending     bool   // true after pressing m, waiting for slot key
	lastMarkAttempt string // tracks last "m+slot" attempt for overwrite confirmation

	// Copy format pending (Y + j/y/d for json/yaml/dotenv)
	copyFormatPending bool

	// Theme picker
	themeEntries      []ui.ThemeEntry // grouped theme list with headers
	themeCursor       int
	activeColorscheme string // currently applied colorscheme name
}

// NewModel creates a new application model.
func NewModel(client *vault.Client, cfg *config.Config, version string) *Model {
	ti := textinput.New()
	si := textinput.New()
	si.Placeholder = "search..."
	ci := textinput.New()
	ci.Placeholder = "DELETE"

	// Load bookmarks: start with saved bookmarks, then merge config bookmarks (dedup by mount+path).
	bookmarks, _ := config.LoadBookmarks()
	for _, cb := range cfg.Bookmarks {
		found := false
		for _, bm := range bookmarks {
			if bm.Mount == cb.Mount && bm.Path == cb.Path {
				found = true
				break
			}
		}
		if !found {
			name := cb.Name
			if name == "" {
				name = cb.Mount + "/" + cb.Path
			}
			bookmarks = append(bookmarks, config.Bookmark{Name: name, Mount: cb.Mount, Path: cb.Path})
		}
	}

	keys := DefaultKeyMap()
	keys.ApplyOverrides(cfg.Keybindings)

	colorscheme := cfg.Colorscheme
	if colorscheme == "" {
		colorscheme = ui.DefaultThemeName()
	}

	return &Model{
		client:       client,
		config:       cfg,
		keys:         keys,
		version:      version,
		revealed:     make(map[string]bool),
		secretBase64: make(map[string]bool),
		selected:     make(map[int]bool),
		textInput:    ti,
		searchInput:  si,
		confirmInput: ci,
		tabs: []TabState{{
			selected: make(map[int]bool),
			mount:    client.Mount(),
		}},
		activeTab:         0,
		bookmarks:         bookmarks,
		themeEntries:      ui.GroupedThemeEntries(),
		activeColorscheme: colorscheme,
	}
}

// saveCurrentTab persists the active Model fields into the current TabState.
// Slices and maps are deep copied to prevent shared-reference mutations across tabs.
func (m *Model) saveCurrentTab() {
	t := &m.tabs[m.activeTab]
	t.path = append([]string(nil), m.path...)
	t.entries = append([]model.Entry(nil), m.entries...)
	t.cursor = m.cursor
	t.parentList = append([]model.Entry(nil), m.parentList...)
	t.cursorMemory = copyMapStringInt(m.cursorMemory)
	t.previewEntries = append([]model.Entry(nil), m.previewEntries...)
	t.previewSecret = m.previewSecret
	t.previewMode = m.previewMode
	t.selected = copyMapIntBool(m.selected)
	t.filterQuery = m.filterQuery
	t.filteredIdx = append([]int(nil), m.filteredIdx...)
	t.atMountLevel = m.atMountLevel
	t.mountCursor = m.mountCursor
	t.mount = m.client.Mount()
}

// loadTab restores Model fields from the given tab index.
func (m *Model) loadTab(idx int) {
	t := m.tabs[idx]
	m.activeTab = idx

	// Close any open popup/overlay
	m.mode = model.ModeExplorer
	m.secret = nil
	m.textInput.Blur()
	m.searchInput.Blur()

	// Deep copy slices and maps to prevent cross-tab mutation.
	m.path = append([]string(nil), t.path...)
	m.entries = append([]model.Entry(nil), t.entries...)
	m.cursor = t.cursor
	m.parentList = append([]model.Entry(nil), t.parentList...)
	m.cursorMemory = copyMapStringInt(t.cursorMemory)
	m.previewEntries = append([]model.Entry(nil), t.previewEntries...)
	m.previewSecret = t.previewSecret
	m.previewMode = t.previewMode
	m.selected = copyMapIntBool(t.selected)
	m.filterQuery = t.filterQuery
	m.filteredIdx = append([]int(nil), t.filteredIdx...)
	m.atMountLevel = t.atMountLevel
	m.mountCursor = t.mountCursor
	m.client.SetMount(t.mount)
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(m.listDir(""), m.loadMounts())
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case listResultMsg:
		return m.handleListResult(msg)

	case secretResultMsg:
		return m.handleSecretResult(msg)

	case mountsResultMsg:
		if msg.err != nil {
			m.errMsg = msg.err.Error()
		} else {
			m.mounts = msg.mounts
			// Set mountCursor to current mount
			for i, mt := range m.mounts {
				if mt == m.client.Mount() {
					m.mountCursor = i
					break
				}
			}
		}
		return m, nil

	case statusMsg:
		m.status = string(msg)
		m.errMsg = ""
		if m.mode == model.ModeExplorer {
			return m, m.refresh()
		}
		return m, nil

	case yankResultMsg:
		m.yankedSecrets = msg.secrets
		m.yankPaths = msg.paths
		m.yankIsCut = msg.isCut
		m.yankIsDir = msg.isDir
		m.errMsg = ""
		if len(msg.secrets) == 1 && !msg.isDir {
			if msg.isCut {
				m.status = "Cut (yanked for move): " + msg.secrets[0].Path
			} else {
				m.status = "Yanked: " + msg.secrets[0].Path
			}
		} else if msg.isDir && len(msg.paths) > 0 {
			if msg.isCut {
				m.status = fmt.Sprintf("Cut %d items for move", len(msg.paths))
			} else {
				m.status = fmt.Sprintf("Yanked %d items", len(msg.paths))
			}
		} else {
			count := len(msg.secrets)
			if msg.isCut {
				m.status = fmt.Sprintf("Cut %d secrets for move", count)
			} else {
				m.status = fmt.Sprintf("Yanked %d secrets", count)
			}
		}
		return m, nil

	case errorMsg:
		m.errMsg = string(msg)
		m.status = ""
		return m, nil

	case confirmCreateMsg:
		path := string(msg)
		m.confirmMsg = fmt.Sprintf("Secret %q already exists. Overwrite?", path)
		m.prevConfirmMode = model.ModeExplorer
		m.confirmAction = func() tea.Cmd {
			return m.createEmptySecretAndOpen(path)
		}
		m.enterConfirmMode()
		return m, nil

	case confirmCreateEditorMsg:
		path := string(msg)
		m.confirmMsg = fmt.Sprintf("Secret %q already exists. Overwrite?", path)
		m.prevConfirmMode = model.ModeExplorer
		m.confirmAction = func() tea.Cmd {
			return m.openEditorForNewSecret(path)
		}
		m.enterConfirmMode()
		return m, nil

	case undoableStatusMsg:
		m.status = msg.status
		m.errMsg = ""
		m.pushUndo(msg.undo)
		if msg.reloadSecret != nil && m.secret != nil && m.secret.Path == msg.reloadSecret.Path {
			m.secret.Data = msg.reloadSecret.Data
			m.secret.Keys = msg.reloadSecret.Keys
			if m.secretCursor >= len(m.secret.Keys) && m.secretCursor > 0 {
				m.secretCursor = len(m.secret.Keys) - 1
			}
		}
		if m.mode == model.ModeExplorer {
			return m, m.refresh()
		}
		return m, nil

	case redoableStatusMsg:
		m.status = msg.status
		m.errMsg = ""
		m.pushRedo(msg.redo)
		if msg.reloadSecret != nil && m.secret != nil && m.secret.Path == msg.reloadSecret.Path {
			m.secret.Data = msg.reloadSecret.Data
			m.secret.Keys = msg.reloadSecret.Keys
			if m.secretCursor >= len(m.secret.Keys) && m.secretCursor > 0 {
				m.secretCursor = len(m.secret.Keys) - 1
			}
		}
		if m.mode == model.ModeExplorer {
			return m, m.refresh()
		}
		return m, nil

	case versionHistoryMsg:
		if msg.err != nil {
			m.errMsg = msg.err.Error()
			return m, nil
		}
		m.versionHistory = msg.versions
		m.versionCursor = 0
		m.mode = model.ModeVersionHistory
		return m, nil

	case versionDetailMsg:
		if msg.err != nil {
			m.errMsg = msg.err.Error()
			return m, nil
		}
		m.secret = msg.secret
		m.secretCursor = 0
		m.revealed = make(map[string]bool)
		m.secretBase64 = make(map[string]bool)
		m.secretAllRevealed = false
		m.secretJSONView = false
		m.mode = model.ModeSecret
		return m, nil

	case newSecretInlineMsg:
		// Open the new secret in popup with inline editing ready
		m.mode = model.ModeSecretEdit
		m.secret = msg.secret
		m.secretCursor = 0
		m.revealed = make(map[string]bool)
		m.secretBase64 = make(map[string]bool)
		m.secretAllRevealed = false
		m.secretJSONView = false
		m.secretEditKey = ""
		m.secretEditOrigKey = ""
		m.secretEditColumn = 0
		m.textInput.SetValue("")
		m.textInput.Focus()
		m.textInput.CursorEnd()
		m.status = "Created: " + msg.secret.Path
		return m, tea.Batch(textinput.Blink, m.refresh())

	case newSecretEditorMsg:
		path := string(msg)
		return m, m.openEditorForNewSecret(path)

	case editorResultMsg:
		// New secret creation via editor
		if msg.newSecretPath != "" {
			path := msg.newSecretPath
			return m, func() tea.Msg {
				if err := m.client.Write(path, msg.data); err != nil {
					return errorMsg(err.Error())
				}
				return undoableStatusMsg{
					status: "Created: " + path,
					undo: model.UndoAction{
						Type:        model.UndoCreateSecret,
						Description: "create " + path,
						Path:        path,
					},
				}
			}
		}
		// Editing existing secret
		if m.secret != nil {
			snapData := copyMap(m.secret.Data)
			snapKeys := copySlice(m.secret.Keys)
			secretPath := m.secret.Path

			// Build ordered keys: preserve existing order for surviving keys, then append new keys sorted
			newKeys := make([]string, 0, len(msg.data))
			for _, k := range m.secret.Keys {
				if _, ok := msg.data[k]; ok {
					newKeys = append(newKeys, k)
				}
			}
			var addedKeys []string
			existing := make(map[string]bool, len(newKeys))
			for _, k := range newKeys {
				existing[k] = true
			}
			for k := range msg.data {
				if !existing[k] {
					addedKeys = append(addedKeys, k)
				}
			}
			sort.Strings(addedKeys)
			newKeys = append(newKeys, addedKeys...)

			m.secret.Data = msg.data
			m.secret.Keys = newKeys
			if m.secretCursor >= len(newKeys) && m.secretCursor > 0 {
				m.secretCursor = len(newKeys) - 1
			}

			return m, m.saveSecretFromEditor(secretPath, snapData, snapKeys)
		}

	case tea.MouseMsg:
		return m.handleMouse(msg)

	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m *Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	// Render the background view based on underlying mode
	// (overlays like confirm/input/search appear on top)
	var content string

	// Build tab labels (show only the last path segment for brevity)
	var tabLabels []string
	if len(m.tabs) > 1 {
		m.saveCurrentTab()
		for _, t := range m.tabs {
			var label string
			if len(t.path) > 0 {
				label = t.path[len(t.path)-1]
			} else {
				label = t.mount + "/"
			}
			tabLabels = append(tabLabels, label)
		}
	}

	bgMode := m.backgroundMode()
	switch bgMode {
	case model.ModeExplorer:
		explorerHeight := m.height - 2
		if m.atMountLevel {
			content = ui.RenderExplorer(
				nil, // no parent at mount level
				m.mountEntries(),
				m.previewEntries, m.previewSecret, m.previewMode,
				-1, m.mountCursor, nil, nil, "mounts",
				tabLabels, m.activeTab,
				"",
				m.version,
				m.width, explorerHeight,
			)
		} else {
			content = ui.RenderExplorer(
				m.leftPaneEntries(),
				m.visibleEntries(),
				m.previewEntries, m.previewSecret, m.previewMode,
				m.leftPaneSelectedIdx(),
				m.visibleCursor(),
				m.selectedVisible(),
				m.path, m.client.Mount(),
				tabLabels, m.activeTab,
				m.highlightQuery(),
				m.version,
				m.width, explorerHeight,
			)
		}
	}

	// Status/error bar
	var statusBar string
	if m.errMsg != "" {
		statusBar = ui.ErrorStyle.Render("Error: " + m.errMsg)
	} else if m.status != "" {
		statusBar = ui.StatusStyle.Render(m.status)
	}

	helpBar := ui.HelpBar(m.mode, m.width)
	base := content + "\n" + statusBar + "\n" + helpBar

	// Centered overlays
	switch m.mode {
	case model.ModeSecret, model.ModeSecretEdit:
		editingKey := ""
		editingView := ""
		editingColumn := -1
		if m.mode == model.ModeSecretEdit {
			editingKey = m.secretEditKey
			editingView = m.textInput.View()
			editingColumn = m.secretEditColumn
		}
		overlay := ui.RenderSecretOverlay(
			m.secret, m.secretCursor, m.revealed,
			m.secretAllRevealed, m.secretJSONView,
			editingKey, editingView, editingColumn,
			m.secretBase64,
			m.width, m.height,
		)
		base = ui.PlaceOverlay(base, overlay, m.width, m.height)
	case model.ModeConfirm:
		overlay := ui.RenderConfirmOverlay(m.confirmMsg, m.confirmInput.View(), m.width)
		base = ui.PlaceOverlay(base, overlay, m.width, m.height)
	case model.ModeInput:
		overlay := ui.RenderInputOverlay(m.inputLabel, m.textInput.View(), m.width)
		base = ui.PlaceOverlay(base, overlay, m.width, m.height)
	case model.ModeSearch:
		overlay := ui.RenderSearchOverlay("Search: ", m.searchInput.View(), m.width)
		base = ui.PlaceOverlay(base, overlay, m.width, m.height)
	case model.ModeFilter:
		overlay := ui.RenderSearchOverlay("Filter: ", m.searchInput.View(), m.width)
		base = ui.PlaceOverlay(base, overlay, m.width, m.height)
	case model.ModeVersionHistory:
		overlay := ui.RenderVersionHistoryOverlay(m.versionHistory, m.versionCursor, m.versionPath, m.width, m.height)
		base = ui.PlaceOverlay(base, overlay, m.width, m.height)
	case model.ModeHelp:
		overlay := ui.RenderHelpScreen(m.width, m.height, m.helpScroll, m.helpFilter, m.helpSearching, &m.searchInput)
		base = ui.PlaceOverlay(base, overlay, m.width, m.height)
	case model.ModeJumpPath:
		overlay := ui.RenderJumpPathOverlay(m.textInput.View(), m.jumpCompletions, m.jumpCompIdx, m.width, m.height)
		base = ui.PlaceOverlay(base, overlay, m.width, m.height)
	case model.ModeBookmark:
		overlay := ui.RenderBookmarkOverlay(m.bookmarks, m.bookmarkFilter, m.bookmarkSearching, m.bookmarkCursor, m.width, m.height)
		base = ui.PlaceOverlay(base, overlay, m.width, m.height)
	case model.ModeThemePicker:
		overlay := ui.RenderThemePickerOverlay(m.themeEntries, m.themeCursor, m.activeColorscheme, m.width, m.height)
		base = ui.PlaceOverlay(base, overlay, m.width, m.height)
	}

	return base
}

// highlightQuery returns the current search/filter term for highlighting.
func (m *Model) highlightQuery() string {
	switch m.mode {
	case model.ModeSearch:
		return m.searchInput.Value()
	case model.ModeFilter:
		return m.searchInput.Value()
	default:
		return m.filterQuery
	}
}

// visibleEntries returns the filtered entry list (or full list if no filter).
func (m *Model) visibleEntries() []model.Entry {
	if m.filterQuery == "" {
		return m.entries
	}
	result := make([]model.Entry, 0, len(m.filteredIdx))
	for _, idx := range m.filteredIdx {
		if idx < len(m.entries) {
			result = append(result, m.entries[idx])
		}
	}
	return result
}

func (m *Model) visibleCursor() int {
	return m.cursor
}

func (m *Model) selectedEntry() *model.Entry {
	vis := m.visibleEntries()
	if m.cursor < 0 || m.cursor >= len(vis) {
		return nil
	}
	e := vis[m.cursor]
	return &e
}

func (m *Model) applyFilter() {
	if m.filterQuery == "" {
		m.filteredIdx = nil
		return
	}
	q := strings.ToLower(m.filterQuery)
	m.filteredIdx = nil
	for i, e := range m.entries {
		if strings.Contains(strings.ToLower(e.Name), q) {
			m.filteredIdx = append(m.filteredIdx, i)
		}
	}
}

func (m *Model) clearFilter() {
	m.filterQuery = ""
	m.filteredIdx = nil
}

// backgroundMode returns the underlying view mode (explorer or secret)
// that should render behind any overlay (confirm/input/search).
func (m *Model) backgroundMode() model.ViewMode {
	switch m.mode {
	case model.ModeConfirm:
		return m.prevConfirmMode
	case model.ModeInput:
		// Input overlays always sit on top of explorer
		return model.ModeExplorer
	case model.ModeSecret, model.ModeSecretEdit, model.ModeVersionHistory, model.ModeSearch, model.ModeFilter, model.ModeHelp, model.ModeJumpPath, model.ModeBookmark, model.ModeThemePicker:
		return model.ModeExplorer
	default:
		return m.mode
	}
}

// Left pane: at root level show mounts, otherwise show parent entries
func (m *Model) leftPaneEntries() []model.Entry {
	if len(m.path) == 0 {
		// At mount root: show mounts in left pane
		entries := make([]model.Entry, 0, len(m.mounts))
		for _, mt := range m.mounts {
			entries = append(entries, model.Entry{Name: mt + "/", IsDir: true})
		}
		return entries
	}
	return m.parentList
}

func (m *Model) leftPaneSelectedIdx() int {
	if len(m.path) == 0 {
		// Highlight current mount
		for i, mt := range m.mounts {
			if mt == m.client.Mount() {
				return i
			}
		}
		return 0
	}
	parentDir := m.parentPath()
	if pos, ok := m.cursorMemory[parentDir]; ok {
		return pos
	}
	return 0
}

func (m *Model) mountEntries() []model.Entry {
	entries := make([]model.Entry, 0, len(m.mounts))
	for _, mt := range m.mounts {
		entries = append(entries, model.Entry{Name: mt + "/", IsDir: true})
	}
	return entries
}

// --- Key handling ---

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Input mode: delegate to textinput
	if m.mode == model.ModeInput {
		return m.handleInputKey(msg)
	}

	// Search mode (jump-to)
	if m.mode == model.ModeSearch {
		return m.handleSearchKey(msg)
	}

	// Filter mode
	if m.mode == model.ModeFilter {
		return m.handleFilterKey(msg)
	}

	// Jump-to-path mode
	if m.mode == model.ModeJumpPath {
		return m.handleJumpPathKey(msg)
	}

	// Version history mode
	if m.mode == model.ModeVersionHistory {
		return m.handleVersionHistoryKey(msg)
	}

	// Help screen
	if m.mode == model.ModeHelp {
		return m.handleHelpKey(msg)
	}

	// Inline secret edit mode
	if m.mode == model.ModeSecretEdit {
		return m.handleSecretEditKey(msg)
	}

	// Confirm mode
	if m.mode == model.ModeConfirm {
		return m.handleConfirmKey(msg)
	}

	// Bookmark overlay
	if m.mode == model.ModeBookmark {
		return m.handleBookmarkOverlayKey(msg)
	}

	// Theme picker overlay
	if m.mode == model.ModeThemePicker {
		return m.handleThemePickerKey(msg)
	}

	switch m.mode {
	case model.ModeExplorer:
		if m.atMountLevel {
			return m.handleMountKey(msg)
		}
		return m.handleExplorerKey(msg)
	case model.ModeSecret:
		return m.handleSecretKey(msg)
	}
	return m, nil
}

// handleMouse processes mouse events for all view modes.
func (m *Model) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	// Ignore motion-only events (no button pressed).
	if msg.Button == tea.MouseButtonNone {
		return m, nil
	}

	switch m.mode {
	case model.ModeExplorer:
		return m.handleExplorerMouse(msg)
	case model.ModeSecret:
		return m.handleSecretMouse(msg)
	case model.ModeHelp:
		return m.handleHelpMouse(msg)
	case model.ModeVersionHistory:
		return m.handleVersionHistoryMouse(msg)
	case model.ModeBookmark:
		return m.handleBookmarkMouse(msg)
	case model.ModeThemePicker:
		return m.handleThemePickerMouse(msg)
	}

	// For other overlay modes (input, search, filter, confirm, etc.)
	// only handle scroll wheel as a no-op; let keyboard drive them.
	return m, nil
}

// explorerColumnBounds calculates the x-coordinate boundaries for the three
// explorer columns. Returns leftEnd, midEnd (right column extends to the edge).
func (m *Model) explorerColumnBounds() (leftEnd, midEnd int) {
	usable := m.width - 6 // 3 columns x 2 border chars
	leftW := usable * 12 / 100
	midW := usable * 51 / 100
	if leftW < 10 {
		leftW = 10
	}
	if midW < 10 {
		midW = 10
	}
	// Left column occupies x: 0 .. leftW+1 (content + 2 border chars)
	leftEnd = leftW + 2
	// Mid column occupies x: leftEnd .. leftEnd+midW+1
	midEnd = leftEnd + midW + 2
	return
}

// explorerColHeight returns the number of visible entry rows in the explorer.
func (m *Model) headerLineCount() int {
	if len(m.tabs) > 1 {
		return 2 // tab bar + breadcrumb
	}
	return 1 // breadcrumb only
}

func (m *Model) explorerColHeight() int {
	// Must match RenderExplorer: height(m.height-2) - 3 - headerLines
	h := m.height - 5 - m.headerLineCount()
	if h < 1 {
		h = 1
	}
	return h
}

// explorerScrollOffset returns the scroll offset for the current entry list,
// matching the logic in renderEntryList.
func (m *Model) explorerScrollOffset() int {
	colHeight := m.explorerColHeight()
	start := 0
	if m.cursor >= colHeight {
		start = m.cursor - colHeight + 1
	}
	return start
}

func (m *Model) handleExplorerMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	// Scroll wheel: move cursor up/down
	if msg.Button == tea.MouseButtonWheelUp {
		if m.atMountLevel {
			if m.mountCursor > 0 {
				m.mountCursor--
				return m, m.loadMountPreview()
			}
			return m, nil
		}
		if m.cursor > 0 {
			m.cursor--
			return m, m.loadPreview()
		}
		return m, nil
	}
	if msg.Button == tea.MouseButtonWheelDown {
		if m.atMountLevel {
			if m.mountCursor < len(m.mounts)-1 {
				m.mountCursor++
				return m, m.loadMountPreview()
			}
			return m, nil
		}
		vis := m.visibleEntries()
		if m.cursor < len(vis)-1 {
			m.cursor++
			return m, m.loadPreview()
		}
		return m, nil
	}

	// Left click: select entry or navigate
	if msg.Button == tea.MouseButtonLeft && msg.Action == tea.MouseActionPress {
		// Check for tab bar click (row 0)
		if msg.Y == 1 && len(m.tabs) > 1 {
			return m.handleTabBarClick(msg.X)
		}

		if m.atMountLevel {
			return m.handleMountLevelClick(msg)
		}

		leftEnd, midEnd := m.explorerColumnBounds()
		colHeight := m.explorerColHeight()
		// Entry rows start after header lines + top border
		entryRow := msg.Y - m.headerLineCount() - 1
		if entryRow < 0 || entryRow >= colHeight {
			return m, nil
		}

		if msg.X < leftEnd {
			// Left column click: navigate up
			return m, m.navigateUp()
		} else if msg.X < midEnd {
			// Mid column click: move cursor to clicked entry
			scrollOffset := m.explorerScrollOffset()
			targetIdx := scrollOffset + entryRow
			vis := m.visibleEntries()
			if targetIdx >= 0 && targetIdx < len(vis) {
				m.cursor = targetIdx
				return m, m.loadPreview()
			}
		} else {
			// Right column click: navigate into the selected entry if it's a directory
			entry := m.selectedEntry()
			if entry != nil && entry.IsDir {
				return m, m.navigateIn()
			}
		}
	}

	// Double click on middle column: open entry (like Enter)
	if msg.Button == tea.MouseButtonLeft && msg.Action == tea.MouseActionMotion {
		// bubbletea doesn't distinguish double-click from drag,
		// so we skip this to avoid accidental navigation.
		return m, nil
	}

	return m, nil
}

// handleTabBarClick determines which tab was clicked based on x position
// and switches to it.
func (m *Model) handleTabBarClick(x int) (tea.Model, tea.Cmd) {
	if len(m.tabs) <= 1 {
		return m, nil
	}

	// Mirror the exact logic from RenderTabBar to compute tab positions.
	m.saveCurrentTab()

	var tabLabels []string
	for _, t := range m.tabs {
		var label string
		if len(t.path) > 0 {
			label = t.path[len(t.path)-1]
		} else {
			label = t.mount + "/"
		}
		tabLabels = append(tabLabels, label)
	}

	maxBarW := m.width - 2
	maxPathLen := maxBarW / len(tabLabels)
	if maxPathLen < 8 {
		maxPathLen = 8
	}

	// Measure each tab's visual width (label + padding of 1 on each side = +2)
	type tabInfo struct {
		width int
	}
	tabs := make([]tabInfo, len(tabLabels))
	for i, path := range tabLabels {
		if len(path) > maxPathLen {
			path = "…" + path[len(path)-maxPathLen+1:]
		}
		label := fmt.Sprintf("%d %s", i+1, path)
		// Padding(0,1) adds 1 cell each side
		tabs[i] = tabInfo{width: len([]rune(label)) + 2}
	}

	sepW := 3 // " │ " rendered width

	// Determine visible window (same algorithm as RenderTabBar)
	totalW := 0
	for i, t := range tabs {
		totalW += t.width
		if i < len(tabs)-1 {
			totalW += sepW
		}
	}

	left := 0
	right := len(tabs) - 1
	if totalW > maxBarW {
		// Window around active tab
		left = m.activeTab
		right = m.activeTab
		usedW := tabs[m.activeTab].width
		for {
			expanded := false
			if left > 0 {
				needed := sepW + tabs[left-1].width
				if usedW+needed <= maxBarW {
					left--
					usedW += needed
					expanded = true
				}
			}
			if right < len(tabs)-1 {
				needed := sepW + tabs[right+1].width
				if usedW+needed <= maxBarW {
					right++
					usedW += needed
					expanded = true
				}
			}
			if !expanded {
				break
			}
		}
	}

	// Walk the visible tabs and match click position
	pos := 1 // leading space
	if left > 0 {
		// "◂" indicator + separator
		indicatorW := 3 // "◂" with padding(0,1) = 1+2
		pos += indicatorW + sepW
	}
	for i := left; i <= right; i++ {
		tabW := tabs[i].width
		if x >= pos && x < pos+tabW {
			if i != m.activeTab {
				m.loadTab(i)
				return m, m.refresh()
			}
			return m, nil
		}
		if i < right {
			pos += tabW + sepW
		}
	}

	return m, nil
}

// handleMountLevelClick handles clicks when at the mount selection level.
func (m *Model) handleMountLevelClick(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	_, midEnd := m.explorerColumnBounds()
	colHeight := m.explorerColHeight()
	entryRow := msg.Y - m.headerLineCount() - 1
	if entryRow < 0 || entryRow >= colHeight {
		return m, nil
	}

	if msg.X < midEnd {
		// Click on a mount in the mid column: select it
		if entryRow >= 0 && entryRow < len(m.mounts) {
			m.mountCursor = entryRow
			return m, m.loadMountPreview()
		}
	} else {
		// Click on right column: enter selected mount
		if len(m.mounts) > 0 && m.mountCursor < len(m.mounts) {
			m.client.SetMount(m.mounts[m.mountCursor])
			m.atMountLevel = false
			m.path = nil
			m.cursor = 0
			return m, m.refresh()
		}
	}
	return m, nil
}

func (m *Model) handleSecretMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	// Scroll wheel: move secret cursor
	if msg.Button == tea.MouseButtonWheelUp {
		if m.secretCursor > 0 {
			m.secretCursor--
		}
		return m, nil
	}
	if msg.Button == tea.MouseButtonWheelDown {
		if m.secret != nil && m.secretCursor < len(m.secret.Keys)-1 {
			m.secretCursor++
		}
		return m, nil
	}

	// Left click outside the popup: close it
	if msg.Button == tea.MouseButtonLeft && msg.Action == tea.MouseActionPress {
		// Popup dimensions: 75% of screen, centered
		boxW := m.width * 75 / 100
		boxH := m.height * 75 / 100
		if boxW < 50 {
			boxW = 50
		}
		if boxH < 10 {
			boxH = 10
		}
		startCol := (m.width - boxW) / 2
		startRow := (m.height - boxH) / 2
		endCol := startCol + boxW
		endRow := startRow + boxH

		if msg.X < startCol || msg.X >= endCol || msg.Y < startRow || msg.Y >= endRow {
			// Click outside popup: close
			m.mode = model.ModeExplorer
			m.secret = nil
			m.secretCursor = 0
			m.revealed = make(map[string]bool)
			m.secretBase64 = make(map[string]bool)
			m.secretAllRevealed = false
			m.secretJSONView = false
			return m, m.refresh()
		}
	}

	return m, nil
}

func (m *Model) handleHelpMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	maxScroll := ui.HelpContentLineCount(m.helpFilter) - m.helpVisibleLines()
	if maxScroll < 0 {
		maxScroll = 0
	}

	if msg.Button == tea.MouseButtonWheelUp {
		if m.helpScroll > 0 {
			m.helpScroll--
		}
		return m, nil
	}
	if msg.Button == tea.MouseButtonWheelDown {
		if m.helpScroll < maxScroll {
			m.helpScroll++
		}
		return m, nil
	}

	// Left click outside the help overlay: close it
	if msg.Button == tea.MouseButtonLeft && msg.Action == tea.MouseActionPress {
		boxW := m.width * 70 / 100
		boxH := m.height * 80 / 100
		if boxW < 50 {
			boxW = 50
		}
		if boxH < 20 {
			boxH = 20
		}
		startCol := (m.width - boxW) / 2
		startRow := (m.height - boxH) / 2
		endCol := startCol + boxW
		endRow := startRow + boxH

		if msg.X < startCol || msg.X >= endCol || msg.Y < startRow || msg.Y >= endRow {
			m.mode = model.ModeExplorer
			return m, nil
		}
	}

	return m, nil
}

func (m *Model) handleVersionHistoryMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if msg.Button == tea.MouseButtonWheelUp {
		if m.versionCursor > 0 {
			m.versionCursor--
		}
		return m, nil
	}
	if msg.Button == tea.MouseButtonWheelDown {
		if m.versionCursor < len(m.versionHistory)-1 {
			m.versionCursor++
		}
		return m, nil
	}
	return m, nil
}

func (m *Model) handleMountKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	switch {
	case matchKey(key, m.keys.Quit):
		return m, tea.Quit

	case matchKey(key, m.keys.CloseTab):
		if len(m.tabs) <= 1 {
			return m, tea.Quit
		}
		m.tabs = append(m.tabs[:m.activeTab], m.tabs[m.activeTab+1:]...)
		if m.activeTab >= len(m.tabs) {
			m.activeTab = len(m.tabs) - 1
		}
		m.loadTab(m.activeTab)
		return m, nil

	case matchKey(key, m.keys.Up):
		if m.mountCursor < len(m.mounts)-1 {
			m.mountCursor++
			return m, m.loadMountPreview()
		}

	case matchKey(key, m.keys.Down):
		if m.mountCursor > 0 {
			m.mountCursor--
			return m, m.loadMountPreview()
		}

	case matchKey(key, m.keys.Right) || matchKey(key, m.keys.Open):
		if len(m.mounts) > 0 && m.mountCursor < len(m.mounts) {
			m.client.SetMount(m.mounts[m.mountCursor])
			m.atMountLevel = false
			m.path = nil
			m.cursor = 0
			return m, m.refresh()
		}
	}
	return m, nil
}

func (m *Model) handleExplorerKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Handle mark pending states (two-key sequences like m+a, '+a)
	if m.markPending {
		m.markPending = false
		return m.handleMarkSave(key)
	}
	switch {
	case matchKey(key, m.keys.Quit):
		return m, tea.Quit

	case matchKey(key, m.keys.CloseTab):
		if len(m.tabs) <= 1 {
			return m, tea.Quit
		}
		// Close current tab
		m.tabs = append(m.tabs[:m.activeTab], m.tabs[m.activeTab+1:]...)
		if m.activeTab >= len(m.tabs) {
			m.activeTab = len(m.tabs) - 1
		}
		m.loadTab(m.activeTab)
		return m, nil

	case matchKey(key, m.keys.NewTab):
		if len(m.tabs) < 9 {
			m.saveCurrentTab()
			newTab := m.tabs[m.activeTab] // struct copy
			// Deep copy maps for the new tab
			newTab.selected = make(map[int]bool)
			for k, v := range m.tabs[m.activeTab].selected {
				newTab.selected[k] = v
			}
			newTab.cursorMemory = make(map[string]int)
			for k, v := range m.tabs[m.activeTab].cursorMemory {
				newTab.cursorMemory[k] = v
			}
			m.tabs = append(m.tabs, newTab)
			m.activeTab = len(m.tabs) - 1
			m.loadTab(m.activeTab)
			m.status = fmt.Sprintf("Tab %d created", m.activeTab+1)
		}
		return m, nil

	case matchKey(key, m.keys.NextTab):
		if len(m.tabs) > 1 {
			m.saveCurrentTab()
			m.loadTab((m.activeTab + 1) % len(m.tabs))
			return m, m.refresh()
		}
		return m, nil

	case matchKey(key, m.keys.PrevTab):
		if len(m.tabs) > 1 {
			m.saveCurrentTab()
			m.loadTab((m.activeTab - 1 + len(m.tabs)) % len(m.tabs))
			return m, m.refresh()
		}
		return m, nil

	case matchKey(key, m.keys.Up):
		vis := m.visibleEntries()
		if m.cursor < len(vis)-1 {
			m.cursor++
			return m, m.loadPreview()
		}

	case matchKey(key, m.keys.Down):
		if m.cursor > 0 {
			m.cursor--
			return m, m.loadPreview()
		}

	case matchKey(key, m.keys.Left):
		return m, m.navigateUp()

	case matchKey(key, m.keys.Right):
		// Navigate into directories only
		entry := m.selectedEntry()
		if entry != nil && entry.IsDir {
			return m, m.navigateIn()
		}

	case matchKey(key, m.keys.Open):
		// Enter directory or open secret popup
		return m, m.navigateIn()

	case matchKey(key, m.keys.Top):
		m.cursor = 0
		return m, m.loadPreview()

	case matchKey(key, m.keys.Bottom):
		vis := m.visibleEntries()
		if len(vis) > 0 {
			m.cursor = len(vis) - 1
			return m, m.loadPreview()
		}

	case matchKey(key, m.keys.Select):
		// Toggle selection on current entry
		vis := m.visibleEntries()
		if m.cursor >= 0 && m.cursor < len(vis) {
			actualIdx := m.actualIdx(m.cursor)
			if m.selected[actualIdx] {
				delete(m.selected, actualIdx)
			} else {
				m.selected[actualIdx] = true
			}
			// Move cursor down
			if m.cursor < len(vis)-1 {
				m.cursor++
			}
			return m, m.loadPreview()
		}

	case matchKey(key, m.keys.Refresh):
		return m, m.refresh()

	case matchKey(key, m.keys.NewSecret):
		return m, m.startInput(model.InputNewSecret, "New secret name:", "")

	case matchKey(key, m.keys.NewSecretEditor):
		// New secret via external editor
		return m, m.startInput(model.InputNewSecretEditor, "New secret name (editor):", "")

	case matchKey(key, m.keys.Rename):
		entry := m.selectedEntry()
		if entry != nil {
			displayName := strings.TrimSuffix(entry.Name, "/")
			return m, m.startInput(model.InputRename, "Rename to:", displayName)
		}

	case matchKey(key, m.keys.Yank):
		if len(m.selected) > 0 {
			entries := m.selectedEntries()
			m.status = fmt.Sprintf("Yanking %d items...", len(entries))
			m.selected = make(map[int]bool)
			return m, m.bulkYankSecrets(entries, false)
		}
		entry := m.selectedEntry()
		if entry != nil && !entry.IsDir {
			path := m.currentPath() + entry.Name
			m.yankIsCut = false
			m.yankIsDir = false
			m.yankPaths = []string{path}
			m.status = "Yanking " + entry.Name + "..."
			return m, m.yankSecret(path)
		}

	case matchKey(key, m.keys.Paste):
		if m.yankIsDir && m.yankIsCut {
			cmd := m.pasteSecrets()
			m.yankIsCut = false
			m.yankIsDir = false
			return m, cmd
		}
		if m.yankIsDir && !m.yankIsCut {
			m.errMsg = "Cannot yank-paste directories, use cut (x) instead"
			return m, nil
		}
		if len(m.yankedSecrets) > 0 {
			cmd := m.pasteSecrets()
			m.yankIsCut = false
			return m, cmd
		}
		m.errMsg = "Nothing yanked"

	case matchKey(key, m.keys.ToggleValues):
		// Toggle secret value visibility in preview
		if m.previewSecret != nil {
			if m.previewMode == model.PreviewValues {
				m.previewMode = model.PreviewHidden
			} else {
				m.previewMode = model.PreviewValues
			}
		}

	case matchKey(key, m.keys.ToggleJSON):
		// Toggle JSON preview
		if m.previewSecret != nil {
			if m.previewMode == model.PreviewJSON {
				m.previewMode = model.PreviewHidden
			} else {
				m.previewMode = model.PreviewJSON
			}
		}

	case matchKey(key, m.keys.HalfDown):
		// Half-page scroll down
		vis := m.visibleEntries()
		if len(vis) > 0 {
			halfPage := m.explorerHalfPage()
			m.cursor += halfPage
			if m.cursor >= len(vis) {
				m.cursor = len(vis) - 1
			}
			return m, m.loadPreview()
		}

	case matchKey(key, m.keys.HalfUp):
		// Half-page scroll up
		if len(m.visibleEntries()) > 0 {
			halfPage := m.explorerHalfPage()
			m.cursor -= halfPage
			if m.cursor < 0 {
				m.cursor = 0
			}
			return m, m.loadPreview()
		}

	case matchKey(key, m.keys.FullDown):
		// Full-page scroll down
		vis := m.visibleEntries()
		if len(vis) > 0 {
			fullPage := m.explorerHalfPage() * 2
			m.cursor += fullPage
			if m.cursor >= len(vis) {
				m.cursor = len(vis) - 1
			}
			return m, m.loadPreview()
		}

	case matchKey(key, m.keys.FullUp):
		// Full-page scroll up
		if len(m.visibleEntries()) > 0 {
			fullPage := m.explorerHalfPage() * 2
			m.cursor -= fullPage
			if m.cursor < 0 {
				m.cursor = 0
			}
			return m, m.loadPreview()
		}

	case matchKey(key, m.keys.Delete):
		if len(m.selected) > 0 {
			// Bulk delete
			count := len(m.selected)
			entries := m.selectedEntries()
			m.confirmMsg = fmt.Sprintf("Delete %d selected items?", count)
			m.prevConfirmMode = model.ModeExplorer
			m.confirmAction = func() tea.Cmd {
				return m.bulkDelete(entries)
			}
			m.enterConfirmMode()
			return m, nil
		}
		entry := m.selectedEntry()
		if entry != nil {
			name := entry.Name
			entryCopy := *entry
			if entry.IsDir {
				m.confirmMsg = fmt.Sprintf("Recursively delete %q and all contents?", name)
			} else {
				m.confirmMsg = fmt.Sprintf("Delete %q?", name)
			}
			m.prevConfirmMode = model.ModeExplorer
			m.confirmAction = func() tea.Cmd {
				return m.deleteEntry(entryCopy)
			}
			m.enterConfirmMode()
			return m, nil
		}

	case matchKey(key, m.keys.Cut):
		if len(m.selected) > 0 {
			entries := m.selectedEntries()
			m.status = fmt.Sprintf("Cutting %d items...", len(entries))
			m.selected = make(map[int]bool)
			return m, m.bulkYankSecrets(entries, true)
		}
		entry := m.selectedEntry()
		if entry != nil {
			if entry.IsDir {
				path := m.currentPath() + strings.TrimSuffix(entry.Name, "/")
				m.yankIsCut = true
				m.yankIsDir = true
				m.yankPaths = []string{path}
				m.yankedSecrets = nil
				m.status = "Cut (directory): " + entry.Name
				return m, nil
			}
			path := m.currentPath() + entry.Name
			m.yankIsCut = true
			m.yankIsDir = false
			m.yankPaths = []string{path}
			m.status = "Cutting " + entry.Name + "..."
			return m, m.yankSecret(path)
		}

	case matchKey(key, m.keys.Undo):
		// Undo
		if len(m.undoStack) == 0 {
			m.errMsg = "Nothing to undo"
			return m, nil
		}
		action := m.undoStack[len(m.undoStack)-1]
		m.undoStack = m.undoStack[:len(m.undoStack)-1]
		return m, m.executeUndo(action)

	case matchKey(key, m.keys.Redo):
		// Redo
		if len(m.redoStack) == 0 {
			m.errMsg = "Nothing to redo"
			return m, nil
		}
		action := m.redoStack[len(m.redoStack)-1]
		m.redoStack = m.redoStack[:len(m.redoStack)-1]
		return m, m.executeRedo(action)

	case matchKey(key, m.keys.Search):
		// Jump-to search: overlay, cursor jumps to first match
		m.clearFilter()
		m.mode = model.ModeSearch
		m.priorCursor = m.cursor
		m.searchInput.SetValue("")
		m.searchInput.Focus()
		return m, textinput.Blink

	case matchKey(key, m.keys.Filter):
		// Filter: overlay, hides non-matching entries
		m.clearFilter()
		m.mode = model.ModeFilter
		m.priorCursor = m.cursor
		m.searchInput.SetValue("")
		m.searchInput.Focus()
		return m, textinput.Blink

	case matchKey(key, m.keys.Help):
		m.mode = model.ModeHelp
		m.helpScroll = 0
		m.helpFilter = ""
		m.helpSearching = false
		return m, nil

	case matchKey(key, m.keys.JumpPath):
		m.mode = model.ModeJumpPath
		m.textInput.SetValue(m.currentPath())
		m.textInput.Focus()
		m.textInput.CursorEnd()
		m.jumpCompletions = nil
		m.jumpCompIdx = -1
		m.jumpLastInput = ""
		m.loadJumpCompletions()
		return m, textinput.Blink

	case matchKey(key, m.keys.Edit):
		// Edit secret in external editor from explorer
		entry := m.selectedEntry()
		if entry != nil && !entry.IsDir {
			path := m.currentPath() + entry.Name
			return m, func() tea.Msg {
				secret, err := m.client.Read(path)
				if err != nil {
					return errorMsg(err.Error())
				}
				return secretResultMsg{path: path, secret: secret, openEditor: true}
			}
		}

	case matchKey(key, m.keys.BookmarkSave):
		m.markPending = true
		m.status = "Set mark: [a-z, 0-9]"
		return m, nil

	case matchKey(key, m.keys.BookmarkShow):
		m.mode = model.ModeBookmark
		m.bookmarkFilter = ""
		m.bookmarkCursor = 0
		return m, nil

	case matchKey(key, m.keys.ThemePicker):
		// Open theme picker overlay
		m.mode = model.ModeThemePicker
		// Set cursor to currently active theme (skip headers)
		m.themeCursor = -1
		for i, entry := range m.themeEntries {
			if !entry.IsHeader && entry.Name == m.activeColorscheme {
				m.themeCursor = i
				break
			}
		}
		// Fallback: first non-header entry
		if m.themeCursor < 0 {
			for i, entry := range m.themeEntries {
				if !entry.IsHeader {
					m.themeCursor = i
					break
				}
			}
		}
		return m, nil
	}
	return m, nil
}

// handleBookmarkOverlayKey handles keys in the bookmark overlay mode.
func (m *Model) handleBookmarkOverlayKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	filtered := m.filteredBookmarks()

	// When searching (typing in filter input), handle text input keys
	if m.bookmarkSearching {
		switch key {
		case "esc":
			m.bookmarkSearching = false
			m.bookmarkFilter = ""
			m.bookmarkCursor = 0
			return m, nil
		case "enter":
			m.bookmarkSearching = false
			return m, nil
		case "backspace":
			if len(m.bookmarkFilter) > 0 {
				m.bookmarkFilter = m.bookmarkFilter[:len(m.bookmarkFilter)-1]
				m.bookmarkCursor = 0
			}
			return m, nil
		default:
			if len(key) == 1 && key[0] >= 32 && key[0] < 127 {
				m.bookmarkFilter += key
				m.bookmarkCursor = 0
			}
			return m, nil
		}
	}

	// Normal bookmark overlay navigation
	switch key {
	case "esc":
		if m.bookmarkFilter != "" {
			m.bookmarkFilter = ""
			m.bookmarkCursor = 0
			return m, nil
		}
		m.mode = model.ModeExplorer
		return m, nil

	case "/":
		m.bookmarkSearching = true
		return m, nil

	case "enter", "l":
		if len(filtered) > 0 && m.bookmarkCursor < len(filtered) {
			bm := filtered[m.bookmarkCursor]
			m.mode = model.ModeExplorer
			return m, m.jumpToBookmark(bm)
		}
		return m, nil

	case "D":
		if len(filtered) > 0 && m.bookmarkCursor < len(filtered) {
			bm := filtered[m.bookmarkCursor]
			for i, b := range m.bookmarks {
				if b.Mount == bm.Mount && b.Path == bm.Path {
					m.bookmarks = append(m.bookmarks[:i], m.bookmarks[i+1:]...)
					break
				}
			}
			_ = config.SaveBookmarks(m.bookmarks)
			newFiltered := m.filteredBookmarks()
			if m.bookmarkCursor >= len(newFiltered) && m.bookmarkCursor > 0 {
				m.bookmarkCursor = len(newFiltered) - 1
			}
			if len(m.bookmarks) == 0 {
				m.mode = model.ModeExplorer
				m.status = "All marks deleted"
			}
		}
		return m, nil

	case "ctrl+x":
		m.bookmarks = nil
		_ = config.SaveBookmarks(m.bookmarks)
		m.mode = model.ModeExplorer
		m.status = "All marks deleted"
		return m, nil

	case "j", "down", "ctrl+n":
		if m.bookmarkCursor < len(filtered)-1 {
			m.bookmarkCursor++
		}
		return m, nil

	case "k", "up", "ctrl+p":
		if m.bookmarkCursor > 0 {
			m.bookmarkCursor--
		}
		return m, nil

	default:
		// Quick jump: pressing a slot key (a-z, 0-9) jumps directly to that mark
		if len(key) == 1 && ((key[0] >= 'a' && key[0] <= 'z') || (key[0] >= '0' && key[0] <= '9')) {
			for _, bm := range m.bookmarks {
				if bm.Slot == key {
					m.mode = model.ModeExplorer
					return m, m.jumpToBookmark(bm)
				}
			}
			m.status = fmt.Sprintf("Mark '%s' not set", key)
			m.mode = model.ModeExplorer
			return m, nil
		}
	}
	return m, nil
}

// filteredBookmarks returns bookmarks matching the current filter.
func (m *Model) filteredBookmarks() []config.Bookmark {
	if m.bookmarkFilter == "" {
		return m.bookmarks
	}
	lf := strings.ToLower(m.bookmarkFilter)
	var result []config.Bookmark
	for _, bm := range m.bookmarks {
		if strings.Contains(strings.ToLower(bm.Name), lf) {
			result = append(result, bm)
		}
	}
	return result
}

// handleBookmarkMouse handles mouse events in the bookmark overlay.
func (m *Model) handleBookmarkMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	filtered := m.filteredBookmarks()
	if msg.Button == tea.MouseButtonWheelUp {
		if m.bookmarkCursor > 0 {
			m.bookmarkCursor--
		}
		return m, nil
	}
	if msg.Button == tea.MouseButtonWheelDown {
		if m.bookmarkCursor < len(filtered)-1 {
			m.bookmarkCursor++
		}
		return m, nil
	}
	return m, nil
}

// handleMarkSave processes the second key after pressing the mark key (m).
// Valid keys (a-z, 0-9) save the current location to that slot.
// If the slot is already occupied, shows a warning and requires pressing m+slot again to overwrite.
func (m *Model) handleMarkSave(key string) (tea.Model, tea.Cmd) {
	if key == "esc" {
		m.status = ""
		m.lastMarkAttempt = ""
		return m, nil
	}
	// Only accept single alphanumeric characters as slot names
	if len(key) != 1 || (key[0] < 'a' || key[0] > 'z') && (key[0] < '0' || key[0] > '9') {
		m.status = "Invalid mark key (use a-z, 0-9)"
		m.lastMarkAttempt = ""
		return m, nil
	}
	slot := key
	mount := m.client.Mount()
	path := m.currentPath()
	name := mount + "/" + path

	// Check if slot is already occupied
	for i, bm := range m.bookmarks {
		if bm.Slot == slot {
			if bm.Mount == mount && bm.Path == path {
				m.status = "Mark '" + slot + "' already set to this location"
				m.lastMarkAttempt = ""
				return m, nil
			}
			// Slot exists with different location — check if this is a confirmed overwrite
			if m.lastMarkAttempt == slot {
				// Second press — overwrite
				m.bookmarks[i].Mount = mount
				m.bookmarks[i].Path = path
				m.bookmarks[i].Name = name
				if err := config.SaveBookmarks(m.bookmarks); err != nil {
					m.errMsg = "Failed to save mark: " + err.Error()
					m.lastMarkAttempt = ""
					return m, nil
				}
				m.status = fmt.Sprintf("Mark '%s' updated: %s", slot, name)
				m.lastMarkAttempt = ""
				return m, nil
			}
			// First press — warn and remember attempt
			m.lastMarkAttempt = slot
			m.status = fmt.Sprintf("Mark '%s' already set (%s) — press m+%s again to overwrite", slot, bm.Name, slot)
			return m, nil
		}
	}

	// Slot is free — save directly
	m.lastMarkAttempt = ""
	m.bookmarks = append(m.bookmarks, config.Bookmark{
		Name:  name,
		Mount: mount,
		Path:  path,
		Slot:  slot,
	})
	if err := config.SaveBookmarks(m.bookmarks); err != nil {
		m.errMsg = "Failed to save mark: " + err.Error()
		return m, nil
	}
	m.status = fmt.Sprintf("Mark '%s' set: %s", slot, name)
	return m, nil
}

// handleThemePickerKey handles keyboard input in the theme picker overlay.
func (m *Model) handleThemePickerKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	switch key {
	case "esc", "q":
		// Revert to the previously active theme and close.
		ui.ApplyTheme(m.activeColorscheme, m.config.Theme)
		m.mode = model.ModeExplorer
		return m, nil

	case "enter", "l":
		// Confirm selection (runtime only, not persisted) and close.
		if m.themeCursor >= 0 && m.themeCursor < len(m.themeEntries) && !m.themeEntries[m.themeCursor].IsHeader {
			name := m.themeEntries[m.themeCursor].Name
			ui.ApplyTheme(name, m.config.Theme)
			m.activeColorscheme = name
			m.status = "Colorscheme: " + name + " (set colorscheme in config to persist)"
		}
		m.mode = model.ModeExplorer
		return m, nil

	case "j", "down", "ctrl+n":
		m.themeMoveCursor(1)
		return m, nil

	case "k", "up", "ctrl+p":
		m.themeMoveCursor(-1)
		return m, nil
	}
	return m, nil
}

// themeMoveCursor moves the theme picker cursor by delta, skipping headers.
func (m *Model) themeMoveCursor(delta int) {
	next := m.themeCursor
	for {
		next += delta
		if next < 0 || next >= len(m.themeEntries) {
			return // boundary reached, don't move
		}
		if !m.themeEntries[next].IsHeader {
			m.themeCursor = next
			ui.ApplyTheme(m.themeEntries[m.themeCursor].Name, m.config.Theme)
			return
		}
	}
}

// handleThemePickerMouse handles mouse events in the theme picker overlay.
func (m *Model) handleThemePickerMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if msg.Button == tea.MouseButtonWheelUp {
		m.themeMoveCursor(-1)
		return m, nil
	}
	if msg.Button == tea.MouseButtonWheelDown {
		m.themeMoveCursor(1)
		return m, nil
	}
	return m, nil
}

// jumpToBookmark navigates to a bookmark's mount and path.
func (m *Model) jumpToBookmark(bm config.Bookmark) tea.Cmd {
	m.clearFilter()
	m.selected = make(map[int]bool)
	m.cursorMemory = nil

	// Switch mount if needed
	if bm.Mount != m.client.Mount() {
		m.client.SetMount(bm.Mount)
	}
	m.atMountLevel = false

	// Parse path into segments
	if bm.Path == "" {
		m.path = nil
	} else {
		cleaned := strings.TrimSuffix(bm.Path, "/")
		if cleaned == "" {
			m.path = nil
		} else {
			parts := strings.Split(cleaned, "/")
			m.path = make([]string, len(parts))
			for i, p := range parts {
				m.path[i] = p + "/"
			}
		}
	}
	m.cursor = 0
	m.mode = model.ModeExplorer
	m.status = fmt.Sprintf("Jumped to bookmark: %s/%s", bm.Mount, bm.Path)
	return m.refresh()
}

func (m *Model) handleSecretKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Handle copy format pending (Y + j/y/d)
	if m.copyFormatPending {
		m.copyFormatPending = false
		return m.handleCopyFormat(key)
	}

	switch key {
	case "q", "esc", "h":
		m.mode = model.ModeExplorer
		m.secret = nil
		m.secretCursor = 0
		m.revealed = make(map[string]bool)
		m.secretBase64 = make(map[string]bool)
		m.secretAllRevealed = false
		m.secretJSONView = false
		return m, m.refresh()

	case "j", "down", "ctrl+n":
		if m.secret != nil && m.secretCursor < len(m.secret.Keys)-1 {
			m.secretCursor++
		}

	case "k", "up", "ctrl+p":
		if m.secretCursor > 0 {
			m.secretCursor--
		}

	case "v", "tab":
		// Toggle reveal ALL values
		if m.secret != nil {
			m.secretJSONView = false
			m.secretAllRevealed = !m.secretAllRevealed
			for _, k := range m.secret.Keys {
				m.revealed[k] = m.secretAllRevealed
			}
		}

	case "V":
		// Toggle JSON view
		if m.secret != nil {
			m.secretJSONView = !m.secretJSONView
		}

	case "y":
		if m.secret == nil {
			break
		}
		if m.secretJSONView {
			// Copy entire secret as JSON
			return m, m.copySecretAsJSON()
		}
		// Copy selected value
		if len(m.secret.Keys) > 0 {
			key := m.secret.Keys[m.secretCursor]
			val := m.secret.Data[key]
			return m, copyToSystemClipboard(val, key)
		}

	case "Y":
		// Copy entire secret — choose format
		if m.secret != nil {
			m.copyFormatPending = true
			m.status = "Copy as: (j)son  (y)aml  (d)otenv"
			return m, nil
		}

	case "p":
		if m.secret != nil && len(m.secret.Keys) > 0 {
			val, err := readFromSystemClipboard()
			if err != nil {
				m.errMsg = "clipboard: " + err.Error()
				return m, nil
			}
			key := m.secret.Keys[m.secretCursor]
			return m, m.editValue(key, val)
		}

	case "e":
		if m.secret != nil {
			if m.secretJSONView {
				return m, m.editSecretInEditor()
			}
			if len(m.secret.Keys) > 0 {
				key := m.secret.Keys[m.secretCursor]
				m.secretEditKey = key
				m.secretEditOrigKey = key
				m.secretEditColumn = 1 // edit value column
				m.mode = model.ModeSecretEdit
				m.textInput.SetValue(m.secret.Data[key])
				m.textInput.Focus()
				m.textInput.CursorEnd()
				return m, textinput.Blink
			}
		}

	case "b":
		if m.secret != nil && len(m.secret.Keys) > 0 {
			key := m.secret.Keys[m.secretCursor]
			val := m.secret.Data[key]
			if m.secretBase64[key] {
				// Toggle off
				m.secretBase64[key] = false
			} else {
				// Try to decode
				_, err := base64.StdEncoding.DecodeString(val)
				if err != nil {
					_, err2 := base64.URLEncoding.DecodeString(val)
					_, err3 := base64.RawStdEncoding.DecodeString(val)
					if err2 != nil && err3 != nil {
						m.errMsg = "Not valid base64"
						break
					}
				}
				m.secretBase64[key] = true
			}
		}

	case "H":
		if m.secret != nil {
			if m.client.IsKV1() {
				m.errMsg = "Version history not available for KV v1"
				return m, nil
			}
			m.versionPath = m.secret.Path
			return m, m.loadVersionHistory(m.secret.Path)
		}

	case "a":
		// Inline add: append empty row and start editing the key column
		if m.secret != nil {
			m.secret.Keys = append(m.secret.Keys, "")
			m.secret.Data[""] = ""
			m.secretCursor = len(m.secret.Keys) - 1
			m.secretEditKey = ""
			m.secretEditOrigKey = ""
			m.secretEditColumn = 0 // start editing key column
			m.mode = model.ModeSecretEdit
			m.textInput.SetValue("")
			m.textInput.Focus()
			m.textInput.CursorEnd()
			return m, textinput.Blink
		}

	case "ctrl+d":
		// Half-page scroll down in secret popup
		if m.secret != nil && len(m.secret.Keys) > 0 {
			halfPage := m.secretPopupHalfPage()
			m.secretCursor += halfPage
			if m.secretCursor >= len(m.secret.Keys) {
				m.secretCursor = len(m.secret.Keys) - 1
			}
		}

	case "ctrl+u":
		// Half-page scroll up in secret popup
		if m.secret != nil && len(m.secret.Keys) > 0 {
			halfPage := m.secretPopupHalfPage()
			m.secretCursor -= halfPage
			if m.secretCursor < 0 {
				m.secretCursor = 0
			}
		}

	case "ctrl+f":
		// Full-page scroll down in secret popup
		if m.secret != nil && len(m.secret.Keys) > 0 {
			fullPage := m.secretPopupHalfPage() * 2
			m.secretCursor += fullPage
			if m.secretCursor >= len(m.secret.Keys) {
				m.secretCursor = len(m.secret.Keys) - 1
			}
		}

	case "ctrl+b":
		// Full-page scroll up in secret popup
		if m.secret != nil && len(m.secret.Keys) > 0 {
			fullPage := m.secretPopupHalfPage() * 2
			m.secretCursor -= fullPage
			if m.secretCursor < 0 {
				m.secretCursor = 0
			}
		}

	case "D":
		if m.secret != nil && len(m.secret.Keys) > 0 {
			key := m.secret.Keys[m.secretCursor]
			m.confirmMsg = fmt.Sprintf("Delete key %q?", key)
			m.prevConfirmMode = model.ModeSecret
			m.confirmAction = func() tea.Cmd {
				return m.deleteKey(key)
			}
			m.enterConfirmMode()
			return m, nil
		}
	}
	return m, nil
}

func (m *Model) handleSecretEditKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// Cancel: revert any uncommitted changes
		m.textInput.Blur()
		// If we were adding a new key (origKey is empty) and haven't saved yet,
		// remove the placeholder empty-key row
		if m.secretEditOrigKey == "" && m.secretEditKey == "" {
			// Remove the empty placeholder row we added
			delete(m.secret.Data, "")
			// Just remove last empty key (the one we added)
			m.secret.Keys = m.removeLastEmptyKey()
			if m.secretCursor >= len(m.secret.Keys) && m.secretCursor > 0 {
				m.secretCursor = len(m.secret.Keys) - 1
			}
		}
		m.mode = model.ModeSecret
		m.secretEditKey = ""
		m.secretEditOrigKey = ""
		m.secretEditColumn = 0
		return m, nil

	case "tab":
		// Save current column input and switch to the other column
		return m.switchEditColumn()

	case "shift+tab":
		// Same as tab — toggle between columns
		return m.switchEditColumn()

	case "enter":
		// Save the current input and commit all changes
		m.applyCurrentEditInput()
		m.textInput.Blur()

		origKey := m.secretEditOrigKey
		currentKey := m.secretEditKey
		val := m.secret.Data[currentKey]

		m.mode = model.ModeSecret
		m.secretEditKey = ""
		m.secretEditOrigKey = ""
		m.secretEditColumn = 0

		if origKey == "" && currentKey == "" {
			// New key was added but name left empty — remove it
			delete(m.secret.Data, "")
			m.secret.Keys = m.removeLastEmptyKey()
			if m.secretCursor >= len(m.secret.Keys) && m.secretCursor > 0 {
				m.secretCursor = len(m.secret.Keys) - 1
			}
			return m, nil
		}

		if origKey == "" {
			// Adding a new key — use addKeyValue
			// Remove the placeholder first
			delete(m.secret.Data, currentKey)
			newKeys := make([]string, 0, len(m.secret.Keys))
			for _, k := range m.secret.Keys {
				if k != currentKey {
					newKeys = append(newKeys, k)
				}
			}
			m.secret.Keys = newKeys
			if m.secretCursor >= len(m.secret.Keys) && m.secretCursor > 0 {
				m.secretCursor = len(m.secret.Keys) - 1
			}
			return m, m.addKeyValue(currentKey, val)
		}

		if origKey != currentKey {
			// Key was renamed — need to rename and possibly update value
			return m, m.renameKey(origKey, currentKey, val)
		}

		// Only value changed
		return m, m.editValue(currentKey, val)
	}
	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

// switchEditColumn saves the current column's textinput and switches to the other column.
func (m *Model) switchEditColumn() (tea.Model, tea.Cmd) {
	m.applyCurrentEditInput()

	// Toggle column
	if m.secretEditColumn == 0 {
		m.secretEditColumn = 1
		// Load value into textinput
		m.textInput.SetValue(m.secret.Data[m.secretEditKey])
	} else {
		m.secretEditColumn = 0
		// Load key into textinput
		m.textInput.SetValue(m.secretEditKey)
	}
	m.textInput.Focus()
	m.textInput.CursorEnd()
	return m, textinput.Blink
}

// applyCurrentEditInput writes the textinput value back to the appropriate field.
func (m *Model) applyCurrentEditInput() {
	if m.secretEditColumn == 0 {
		// Editing key column — rename in-place
		newKey := m.textInput.Value()
		oldKey := m.secretEditKey
		if newKey != oldKey {
			// Update the key in the Keys slice (preserve order)
			for i, k := range m.secret.Keys {
				if k == oldKey {
					m.secret.Keys[i] = newKey
					break
				}
			}
			// Move the data
			val := m.secret.Data[oldKey]
			delete(m.secret.Data, oldKey)
			m.secret.Data[newKey] = val
			m.secretEditKey = newKey
		}
	} else {
		// Editing value column
		m.secret.Data[m.secretEditKey] = m.textInput.Value()
	}
}

// removeLastEmptyKey removes the last occurrence of an empty key from secret.Keys.
func (m *Model) removeLastEmptyKey() []string {
	lastEmpty := -1
	for i, k := range m.secret.Keys {
		if k == "" {
			lastEmpty = i
		}
	}
	if lastEmpty >= 0 {
		return append(m.secret.Keys[:lastEmpty], m.secret.Keys[lastEmpty+1:]...)
	}
	return m.secret.Keys
}

// renameKey handles renaming a key in the secret (delete old, add new with value).
func (m *Model) renameKey(oldKey, newKey, val string) tea.Cmd {
	snapData := copyMap(m.secret.Data)
	snapKeys := copySlice(m.secret.Keys)
	secretPath := m.secret.Path

	// Mutate model state synchronously (safe — called from Update goroutine).
	delete(m.secret.Data, oldKey)
	m.secret.Data[newKey] = val
	for i, k := range m.secret.Keys {
		if k == oldKey {
			m.secret.Keys[i] = newKey
			break
		}
	}

	// Copy updated state for the async Vault write.
	writeData := copyMap(m.secret.Data)

	return func() tea.Msg {
		if err := m.client.Write(secretPath, writeData); err != nil {
			return errorMsg(err.Error())
		}
		return undoableStatusMsg{
			status: fmt.Sprintf("Renamed key: %s → %s", oldKey, newKey),
			undo: model.UndoAction{
				Type:        model.UndoEditSecret,
				Description: fmt.Sprintf("rename key %s → %s", oldKey, newKey),
				Path:        secretPath,
				Data:        snapData,
				Keys:        snapKeys,
			},
		}
	}
}

func (m *Model) handleHelpKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// When searching, delegate to the text input
	if m.helpSearching {
		switch msg.String() {
		case "enter":
			m.helpSearching = false
			m.helpFilter = m.searchInput.Value()
			m.helpScroll = 0
			return m, nil
		case "esc":
			m.helpSearching = false
			m.helpFilter = ""
			m.searchInput.SetValue("")
			m.helpScroll = 0
			return m, nil
		default:
			var cmd tea.Cmd
			m.searchInput, cmd = m.searchInput.Update(msg)
			// Live filter while typing
			m.helpFilter = m.searchInput.Value()
			m.helpScroll = 0
			return m, cmd
		}
	}

	maxScroll := ui.HelpContentLineCount(m.helpFilter) - m.helpVisibleLines()
	if maxScroll < 0 {
		maxScroll = 0
	}

	switch msg.String() {
	case "esc", "?", "q":
		m.mode = model.ModeExplorer
	case "j", "down", "ctrl+n":
		if m.helpScroll < maxScroll {
			m.helpScroll++
		}
	case "k", "up", "ctrl+p":
		if m.helpScroll > 0 {
			m.helpScroll--
		}
	case "ctrl+d":
		halfPage := m.helpVisibleLines() / 2
		if halfPage < 1 {
			halfPage = 1
		}
		m.helpScroll += halfPage
		if m.helpScroll > maxScroll {
			m.helpScroll = maxScroll
		}
	case "ctrl+u":
		halfPage := m.helpVisibleLines() / 2
		if halfPage < 1 {
			halfPage = 1
		}
		m.helpScroll -= halfPage
		if m.helpScroll < 0 {
			m.helpScroll = 0
		}
	case "ctrl+f":
		fullPage := m.helpVisibleLines()
		if fullPage < 1 {
			fullPage = 1
		}
		m.helpScroll += fullPage
		if m.helpScroll > maxScroll {
			m.helpScroll = maxScroll
		}
	case "ctrl+b":
		fullPage := m.helpVisibleLines()
		if fullPage < 1 {
			fullPage = 1
		}
		m.helpScroll -= fullPage
		if m.helpScroll < 0 {
			m.helpScroll = 0
		}
	case "g":
		m.helpScroll = 0
	case "G":
		m.helpScroll = maxScroll
	case "/":
		m.helpSearching = true
		m.searchInput.SetValue(m.helpFilter)
		m.searchInput.Focus()
		return m, textinput.Blink
	}
	return m, nil
}

// helpVisibleLines returns the number of content lines visible in the help overlay.
func (m *Model) helpVisibleLines() int {
	boxH := m.height * 80 / 100
	if boxH < 20 {
		boxH = 20
	}
	// title + borders/padding + help line + possible scroll indicators
	maxLines := boxH - 6
	if maxLines < 5 {
		maxLines = 5
	}
	return maxLines
}

// explorerHalfPage returns the half-page scroll amount for explorer mode.
func (m *Model) explorerHalfPage() int {
	// The explorer visible height is roughly height - 2 (status + help bars)
	// minus some overhead for borders/padding. Use a simple estimate.
	half := (m.height - 2) / 2
	if half < 1 {
		half = 1
	}
	return half
}

// secretPopupHalfPage returns the half-page scroll amount for the secret popup.
func (m *Model) secretPopupHalfPage() int {
	boxH := m.height * 75 / 100
	if boxH < 10 {
		boxH = 10
	}
	outerPadH := 4 // outer border (2) + outer padding (2)
	innerPadH := 2 // inner border (2) + inner padding (0)
	titleH := 1
	helpH := 1
	gapH := 2

	panelContentH := boxH - outerPadH - innerPadH - titleH - helpH - gapH
	if panelContentH < 3 {
		panelContentH = 3
	}
	tableHeight := panelContentH - 2
	if tableHeight < 1 {
		tableHeight = 1
	}
	half := tableHeight / 2
	if half < 1 {
		half = 1
	}
	return half
}

func (m *Model) copySecretAsJSON() tea.Cmd {
	data, err := json.Marshal(m.secret.Data)
	if err != nil {
		return func() tea.Msg { return errorMsg("json: " + err.Error()) }
	}
	return copyToSystemClipboard(string(data), "secret JSON")
}

// formatSecretAsYAML renders secret key-value pairs as YAML text.
func formatSecretAsYAML(secret *model.Secret) string {
	var buf strings.Builder
	for _, k := range secret.Keys {
		v := secret.Data[k]
		// Quote values that contain special YAML characters or are empty
		if v == "" || strings.ContainsAny(v, ":#{}[]&*!|>'\",\n") || v == "true" || v == "false" || v == "null" {
			buf.WriteString(k + ": " + strconv.Quote(v) + "\n")
		} else {
			buf.WriteString(k + ": " + v + "\n")
		}
	}
	return buf.String()
}

func (m *Model) copySecretAsYAML() tea.Cmd {
	if m.secret == nil {
		return nil
	}
	return copyToSystemClipboard(formatSecretAsYAML(m.secret), "secret YAML")
}

// formatSecretAsDotenv renders secret key-value pairs as dotenv text.
func formatSecretAsDotenv(secret *model.Secret) string {
	var buf strings.Builder
	for _, k := range secret.Keys {
		v := secret.Data[k]
		// Use double quotes for values containing special characters
		if strings.ContainsAny(v, " \t\n\"'\\$`!#") || v == "" {
			escaped := strings.ReplaceAll(v, "\\", "\\\\")
			escaped = strings.ReplaceAll(escaped, "\"", "\\\"")
			escaped = strings.ReplaceAll(escaped, "\n", "\\n")
			buf.WriteString(k + "=\"" + escaped + "\"\n")
		} else {
			buf.WriteString(k + "=" + v + "\n")
		}
	}
	return buf.String()
}

func (m *Model) copySecretAsDotenv() tea.Cmd {
	if m.secret == nil {
		return nil
	}
	return copyToSystemClipboard(formatSecretAsDotenv(m.secret), "secret dotenv")
}

// handleCopyFormat processes the second key after pressing Y in the secret popup.
func (m *Model) handleCopyFormat(key string) (tea.Model, tea.Cmd) {
	if m.secret == nil {
		m.status = ""
		return m, nil
	}
	switch key {
	case "j":
		m.status = ""
		return m, m.copySecretAsJSON()
	case "y":
		m.status = ""
		return m, m.copySecretAsYAML()
	case "d":
		m.status = ""
		return m, m.copySecretAsDotenv()
	default:
		m.status = ""
		return m, nil
	}
}

func (m *Model) enterConfirmMode() {
	m.confirmInput.SetValue("")
	m.confirmInput.Focus()
	m.mode = model.ModeConfirm
}

func (m *Model) handleConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		if m.confirmInput.Value() == "DELETE" {
			action := m.confirmAction
			m.mode = m.prevConfirmMode
			m.confirmAction = nil
			m.confirmMsg = ""
			m.confirmInput.Blur()
			if action != nil {
				return m, action()
			}
		} else {
			m.errMsg = "Type DELETE to confirm"
		}
		return m, nil
	case "esc":
		m.mode = m.prevConfirmMode
		m.confirmAction = nil
		m.confirmMsg = ""
		m.confirmInput.Blur()
		return m, nil
	}
	// Forward all other keypresses to the text input
	var cmd tea.Cmd
	m.confirmInput, cmd = m.confirmInput.Update(msg)
	return m, cmd
}

// --- Undo/redo helpers ---

func (m *Model) pushUndo(action model.UndoAction) {
	m.undoStack = append(m.undoStack, action)
	m.redoStack = nil // new action clears redo
}

func (m *Model) pushRedo(action model.UndoAction) {
	m.redoStack = append(m.redoStack, action)
}

func copyMap(src map[string]string) map[string]string {
	c := make(map[string]string, len(src))
	for k, v := range src {
		c[k] = v
	}
	return c
}

func copySlice(src []string) []string {
	c := make([]string, len(src))
	copy(c, src)
	return c
}

func copyMapStringInt(src map[string]int) map[string]int {
	if src == nil {
		return nil
	}
	c := make(map[string]int, len(src))
	for k, v := range src {
		c[k] = v
	}
	return c
}

func copyMapIntBool(src map[int]bool) map[int]bool {
	if src == nil {
		return nil
	}
	c := make(map[int]bool, len(src))
	for k, v := range src {
		c[k] = v
	}
	return c
}

// --- Input handling (using textinput) ---

func (m *Model) startInput(action model.InputAction, label, value string) tea.Cmd {
	m.mode = model.ModeInput
	m.inputAction = action
	m.inputLabel = label
	m.textInput.SetValue(value)
	m.textInput.Focus()
	m.textInput.CursorEnd()
	return textinput.Blink
}

func (m *Model) handleInputKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.textInput.Blur()
		if m.secret != nil {
			m.mode = model.ModeSecret
		} else {
			m.mode = model.ModeExplorer
		}
		m.inputAction = model.InputNone
		m.inputBuffer = ""
		return m, nil

	case "enter":
		return m.handleInputSubmit()
	}

	// Delegate all other keys to the textinput component
	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m *Model) handleInputSubmit() (tea.Model, tea.Cmd) {
	value := m.textInput.Value()
	m.textInput.Blur()

	switch m.inputAction {
	case model.InputRename:
		m.mode = model.ModeExplorer
		m.inputAction = model.InputNone
		if value == "" {
			return m, nil
		}
		return m, m.renameEntry(value)

	case model.InputNewSecret:
		m.mode = model.ModeExplorer
		m.inputAction = model.InputNone
		if value == "" {
			return m, nil
		}
		path := m.currentPath() + value
		return m, m.createSecretWithCheck(path)

	case model.InputNewSecretEditor:
		m.mode = model.ModeExplorer
		m.inputAction = model.InputNone
		if value == "" {
			return m, nil
		}
		path := m.currentPath() + value
		return m, m.createSecretWithEditor(path)

	case model.InputNewKey:
		// Save key name, now ask for value
		m.inputBuffer = value
		m.inputAction = model.InputNewValue
		m.inputLabel = "Value for " + value + ":"
		m.textInput.SetValue("")
		m.textInput.Focus()
		return m, textinput.Blink

	case model.InputNewValue:
		key := m.inputBuffer
		val := value
		m.mode = model.ModeSecret
		m.inputAction = model.InputNone
		m.inputBuffer = ""
		if key == "" {
			return m, nil
		}
		return m, m.addKeyValue(key, val)

	case model.InputEditValue:
		m.mode = model.ModeSecret
		m.inputAction = model.InputNone
		key := m.secret.Keys[m.secretCursor]
		return m, m.editValue(key, value)
	}

	m.mode = model.ModeExplorer
	m.inputAction = model.InputNone
	return m, nil
}

// --- Search handling (s = jump-to) ---

func (m *Model) handleSearchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// Cancel: restore cursor
		m.searchInput.Blur()
		m.mode = model.ModeExplorer
		m.cursor = m.priorCursor
		return m, m.loadPreview()

	case "enter":
		// Confirm: stay at jumped-to cursor
		m.searchInput.Blur()
		m.mode = model.ModeExplorer
		return m, m.loadPreview()
	}

	// Delegate to textinput, then jump cursor to first match
	var cmd tea.Cmd
	m.searchInput, cmd = m.searchInput.Update(msg)
	m.jumpToMatch(m.searchInput.Value())
	return m, cmd
}

// jumpToMatch moves the cursor to the first entry matching query.
func (m *Model) jumpToMatch(query string) {
	if query == "" {
		m.cursor = m.priorCursor
		return
	}
	q := strings.ToLower(query)
	vis := m.visibleEntries()
	for i, e := range vis {
		if strings.Contains(strings.ToLower(e.Name), q) {
			m.cursor = i
			return
		}
	}
}

// --- Filter handling (/ = filter entries) ---

func (m *Model) handleFilterKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// Cancel: clear filter, restore cursor
		m.searchInput.Blur()
		m.mode = model.ModeExplorer
		m.clearFilter()
		m.cursor = m.priorCursor
		return m, m.loadPreview()

	case "enter":
		// Confirm: keep filter active
		m.searchInput.Blur()
		m.mode = model.ModeExplorer
		return m, m.loadPreview()
	}

	// Delegate to textinput, live-filter
	var cmd tea.Cmd
	m.searchInput, cmd = m.searchInput.Update(msg)
	m.filterQuery = m.searchInput.Value()
	m.applyFilter()
	vis := m.visibleEntries()
	if m.cursor >= len(vis) {
		m.cursor = max(0, len(vis)-1)
	}
	return m, cmd
}

// --- Jump-to-path handling (S) ---

func (m *Model) handleJumpPathKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.textInput.Blur()
		m.mode = model.ModeExplorer
		m.jumpCompletions = nil
		m.jumpCompIdx = -1
		return m, nil

	case "enter":
		input := m.textInput.Value()
		if input == "" {
			m.textInput.Blur()
			m.jumpCompletions = nil
			m.jumpCompIdx = -1
			m.mode = model.ModeExplorer
			return m, nil
		}
		// If a completion is selected, accept it
		if m.jumpCompIdx >= 0 && m.jumpCompIdx < len(m.jumpCompletions) {
			selected := m.jumpCompletions[m.jumpCompIdx]
			m.textInput.SetValue(selected)
			m.textInput.CursorEnd()
			m.jumpCompIdx = -1
			// If it's a directory, reload completions for that dir
			if strings.HasSuffix(selected, "/") {
				m.jumpCompletions = nil
				m.loadJumpCompletions()
				return m, nil
			}
			// If it's a secret, navigate to it
			m.textInput.Blur()
			m.jumpCompletions = nil
			return m, m.jumpToPath(selected)
		}
		// No completion selected, navigate directly
		m.textInput.Blur()
		m.jumpCompletions = nil
		m.jumpCompIdx = -1
		return m, m.jumpToPath(input)

	case "tab", "ctrl+n":
		// Cycle highlight forward through completions (don't fill input)
		if len(m.jumpCompletions) > 0 {
			m.jumpCompIdx++
			if m.jumpCompIdx >= len(m.jumpCompletions) {
				m.jumpCompIdx = 0
			}
		}
		return m, nil

	case "shift+tab", "ctrl+p":
		// Cycle highlight backward through completions (don't fill input)
		if len(m.jumpCompletions) > 0 {
			m.jumpCompIdx--
			if m.jumpCompIdx < 0 {
				m.jumpCompIdx = len(m.jumpCompletions) - 1
			}
		}
		return m, nil
	}

	// All other keys: delegate to textinput and reload completions
	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	m.jumpCompletions = nil
	m.jumpCompIdx = -1
	m.loadJumpCompletions()
	return m, cmd
}

// loadJumpCompletions lists the parent directory of the current input
// and filters entries matching the typed prefix.
func (m *Model) loadJumpCompletions() {
	input := m.textInput.Value()
	m.jumpLastInput = input

	// Split into parent dir and prefix at the last slash
	lastSlash := strings.LastIndex(input, "/")
	var parentPath, prefix string
	if lastSlash >= 0 {
		parentPath = input[:lastSlash+1]
		prefix = input[lastSlash+1:]
	} else {
		parentPath = ""
		prefix = input
	}

	entries, err := m.client.List(parentPath)
	if err != nil || entries == nil {
		m.jumpCompletions = nil
		return
	}

	lowerPrefix := strings.ToLower(prefix)
	var completions []string
	for _, e := range entries {
		if strings.HasPrefix(strings.ToLower(e.Name), lowerPrefix) {
			completions = append(completions, parentPath+e.Name)
		}
	}
	m.jumpCompletions = completions
	m.jumpCompIdx = -1
}

// jumpToPath navigates the explorer to the given Vault path.
func (m *Model) jumpToPath(input string) tea.Cmd {
	m.clearFilter()
	m.selected = make(map[int]bool)
	m.cursorMemory = nil // reset history on direct jump

	if strings.HasSuffix(input, "/") {
		// Directory navigation: set path segments and refresh
		cleaned := strings.TrimSuffix(input, "/")
		if cleaned == "" {
			m.path = nil
		} else {
			parts := strings.Split(cleaned, "/")
			// Reconstruct path segments with trailing slashes (how the app stores them)
			m.path = make([]string, len(parts))
			for i, p := range parts {
				m.path[i] = p + "/"
			}
		}
		m.cursor = 0
		m.mode = model.ModeExplorer
		return m.refresh()
	}

	// Path points to a secret (no trailing slash)
	// Navigate to parent directory and try to open the secret
	lastSlash := strings.LastIndex(input, "/")
	var dirPath string
	var secretName string
	if lastSlash >= 0 {
		dirPath = input[:lastSlash+1]
		secretName = input[lastSlash+1:]
	} else {
		dirPath = ""
		secretName = input
	}

	// Set path segments for the directory
	cleaned := strings.TrimSuffix(dirPath, "/")
	if cleaned == "" {
		m.path = nil
	} else {
		parts := strings.Split(cleaned, "/")
		m.path = make([]string, len(parts))
		for i, p := range parts {
			m.path[i] = p + "/"
		}
	}
	m.cursor = 0
	m.mode = model.ModeExplorer

	// Refresh the directory listing, then try to read the secret
	secretPath := m.currentPath() + secretName
	return tea.Batch(
		m.refresh(),
		m.readSecret(secretPath),
	)
}

// --- Navigation ---

func (m *Model) currentPath() string {
	return strings.Join(m.path, "")
}

func (m *Model) parentPath() string {
	if len(m.path) < 2 {
		return ""
	}
	return strings.Join(m.path[:len(m.path)-1], "")
}

func (m *Model) navigateUp() tea.Cmd {
	m.clearFilter()
	m.selected = make(map[int]bool)

	if len(m.path) == 0 {
		// At mount root: go to mount selection
		m.atMountLevel = true
		m.previewEntries = nil
		m.previewSecret = nil
		return m.loadMountPreview()
	}
	m.path = m.path[:len(m.path)-1]
	// Restore cursor position for the directory we're returning to
	dirKey := strings.Join(m.path, "")
	if pos, ok := m.cursorMemory[dirKey]; ok {
		m.cursor = pos
	} else {
		m.cursor = 0
	}
	return m.refresh()
}

func (m *Model) navigateIn() tea.Cmd {
	entry := m.selectedEntry()
	if entry == nil {
		return nil
	}

	m.clearFilter()
	m.selected = make(map[int]bool)

	if entry.IsDir {
		// Remember cursor position for the current directory
		dirKey := strings.Join(m.path, "")
		if m.cursorMemory == nil {
			m.cursorMemory = make(map[string]int)
		}
		m.cursorMemory[dirKey] = m.cursor
		m.path = append(m.path, entry.Name)
		// Restore cursor for the directory we're entering, or start at 0
		newDirKey := strings.Join(m.path, "")
		if pos, ok := m.cursorMemory[newDirKey]; ok {
			m.cursor = pos
		} else {
			m.cursor = 0
		}
		return m.refresh()
	}
	// It's a secret — open secret view
	path := m.currentPath() + entry.Name
	return m.readSecret(path)
}

func (m *Model) refresh() tea.Cmd {
	return tea.Batch(
		m.listDir(m.currentPath()),
		m.listParent(),
	)
}

// --- Vault commands ---

func (m *Model) loadMounts() tea.Cmd {
	return func() tea.Msg {
		mounts, err := m.client.ListMounts()
		return mountsResultMsg{mounts: mounts, err: err}
	}
}

func (m *Model) loadMountPreview() tea.Cmd {
	if len(m.mounts) == 0 || m.mountCursor >= len(m.mounts) {
		return nil
	}
	mount := m.mounts[m.mountCursor]
	return func() tea.Msg {
		entries, err := m.client.ListWithMount(mount, "")
		if err != nil {
			return listResultMsg{path: "@@mount_preview@@", entries: nil, err: err}
		}
		return listResultMsg{path: "@@mount_preview@@", entries: entries}
	}
}

func (m *Model) listDir(path string) tea.Cmd {
	return func() tea.Msg {
		entries, err := m.client.List(path)
		return listResultMsg{path: path, entries: entries, err: err}
	}
}

func (m *Model) listParent() tea.Cmd {
	return func() tea.Msg {
		pp := m.parentPath()
		entries, err := m.client.List(pp)
		if err != nil {
			return nil
		}
		return listResultMsg{path: pp, entries: entries}
	}
}

func (m *Model) loadPreview() tea.Cmd {
	entry := m.selectedEntry()
	if entry == nil {
		m.previewMode = model.PreviewHidden
		m.previewEntries = nil
		m.previewSecret = nil
		return nil
	}

	if entry.IsDir {
		m.previewMode = model.PreviewHidden
		m.previewSecret = nil
		path := m.currentPath() + entry.Name
		return func() tea.Msg {
			entries, err := m.client.List(path)
			if err != nil {
				return errorMsg(err.Error())
			}
			return listResultMsg{path: path, entries: entries}
		}
	}

	// It's a secret — load preview.
	// Only reset visibility if the cursor moved to a different secret.
	path := m.currentPath() + entry.Name
	if m.previewSecret == nil || m.previewSecret.Path != path {
		m.previewMode = model.PreviewHidden
	}
	m.previewEntries = nil
	return func() tea.Msg {
		secret, err := m.client.Read(path)
		return secretResultMsg{path: path, secret: secret, err: err}
	}
}

func (m *Model) readSecret(path string) tea.Cmd {
	return func() tea.Msg {
		secret, err := m.client.Read(path)
		if err != nil {
			return errorMsg(err.Error())
		}
		return secretResultMsg{path: path, secret: secret, openPopup: true}
	}
}

func (m *Model) yankSecret(path string) tea.Cmd {
	isCut := m.yankIsCut
	return func() tea.Msg {
		secret, err := m.client.Read(path)
		if err != nil {
			return errorMsg(fmt.Sprintf("yank failed: %v", err))
		}
		return yankResultMsg{secrets: []*model.Secret{secret}, paths: []string{path}, isCut: isCut}
	}
}

func (m *Model) bulkYankSecrets(entries []model.Entry, isCut bool) tea.Cmd {
	basePath := m.currentPath()
	return func() tea.Msg {
		var secrets []*model.Secret
		var paths []string
		hasDir := false
		for _, entry := range entries {
			path := basePath + strings.TrimSuffix(entry.Name, "/")
			paths = append(paths, path)
			if entry.IsDir {
				hasDir = true
				continue
			}
			secret, err := m.client.Read(basePath + entry.Name)
			if err != nil {
				return errorMsg(fmt.Sprintf("yank failed for %s: %v", entry.Name, err))
			}
			secrets = append(secrets, secret)
		}
		return yankResultMsg{secrets: secrets, paths: paths, isCut: isCut, isDir: hasDir}
	}
}

func (m *Model) deleteEntry(entry model.Entry) tea.Cmd {
	return func() tea.Msg {
		path := m.currentPath() + strings.TrimSuffix(entry.Name, "/")
		if entry.IsDir {
			count, err := m.client.DeleteRecursive(path)
			if err != nil {
				return errorMsg(err.Error())
			}
			// Directory deletes cannot be undone (no stored data for all children).
			return statusMsg(fmt.Sprintf("Deleted %d secrets in %s (cannot undo)", count, entry.Name))
		}
		// Read secret data before deleting (for undo)
		secret, _ := m.client.Read(path)
		if err := m.client.Delete(path); err != nil {
			return errorMsg(err.Error())
		}
		var data map[string]string
		var keys []string
		if secret != nil {
			data = copyMap(secret.Data)
			keys = copySlice(secret.Keys)
		}
		return undoableStatusMsg{
			status: "Deleted: " + entry.Name,
			undo: model.UndoAction{
				Type:        model.UndoDeleteSecret,
				Description: "delete " + entry.Name,
				Path:        path,
				Data:        data,
				Keys:        keys,
			},
		}
	}
}

func (m *Model) renameEntry(newName string) tea.Cmd {
	return func() tea.Msg {
		entry := m.selectedEntry()
		if entry == nil {
			return errorMsg("no entry selected")
		}
		if entry.IsDir {
			src := m.currentPath() + strings.TrimSuffix(entry.Name, "/")
			dst := m.currentPath() + strings.TrimSuffix(newName, "/")
			count, err := m.client.MoveRecursive(src, dst)
			if err != nil {
				return errorMsg(err.Error())
			}
			return undoableStatusMsg{
				status: fmt.Sprintf("Renamed directory %s → %s (%d secrets)", entry.Name, newName, count),
				undo: model.UndoAction{
					Type:        model.UndoRenameSecret,
					Description: fmt.Sprintf("rename dir %s → %s", entry.Name, newName),
					Path:        dst,
					OldPath:     src,
				},
			}
		}
		src := m.currentPath() + entry.Name
		dst := m.currentPath() + newName
		if err := m.client.Move(src, dst); err != nil {
			return errorMsg(err.Error())
		}
		return undoableStatusMsg{
			status: fmt.Sprintf("Renamed %s → %s", entry.Name, newName),
			undo: model.UndoAction{
				Type:        model.UndoRenameSecret,
				Description: fmt.Sprintf("rename %s → %s", entry.Name, newName),
				Path:        dst,
				OldPath:     src,
			},
		}
	}
}

func (m *Model) createSecretWithCheck(path string) tea.Cmd {
	return func() tea.Msg {
		// Check if secret already exists
		if _, err := m.client.Read(path); err == nil {
			return confirmCreateMsg(path)
		}
		// Create an empty secret and open the popup for inline editing
		return m.createEmptySecretAndOpen(path)()
	}
}

// createEmptySecretAndOpen writes an empty secret to Vault and returns a
// secretResultMsg that opens the popup with inline editing ready.
func (m *Model) createEmptySecretAndOpen(path string) tea.Cmd {
	return func() tea.Msg {
		emptyData := map[string]string{}
		if err := m.client.Write(path, emptyData); err != nil {
			return errorMsg(fmt.Sprintf("Failed to create secret: %v", err))
		}
		secret := &model.Secret{
			Path: path,
			Data: map[string]string{"": ""},
			Keys: []string{""},
		}
		return newSecretInlineMsg{secret: secret}
	}
}

// createSecretWithEditor checks if a secret exists, and either prompts for
// overwrite or directly opens the external editor for a new secret.
func (m *Model) createSecretWithEditor(path string) tea.Cmd {
	return func() tea.Msg {
		if _, err := m.client.Read(path); err == nil {
			return confirmCreateEditorMsg(path)
		}
		return newSecretEditorMsg(path)
	}
}

func (m *Model) pasteSecrets() tea.Cmd {
	isCut := m.yankIsCut
	isDir := m.yankIsDir
	yankPaths := make([]string, len(m.yankPaths))
	copy(yankPaths, m.yankPaths)
	yankedSecrets := make([]*model.Secret, len(m.yankedSecrets))
	copy(yankedSecrets, m.yankedSecrets)
	basePath := m.currentPath()

	return func() tea.Msg {
		// Directory cut+paste: use MoveRecursive for each directory
		if isDir && isCut && len(yankPaths) > 0 {
			totalMoved := 0
			for _, yp := range yankPaths {
				parts := strings.Split(strings.TrimSuffix(yp, "/"), "/")
				name := parts[len(parts)-1]
				dst := basePath + name
				count, err := m.client.MoveRecursive(yp, dst)
				if err != nil {
					return errorMsg(fmt.Sprintf("moved %d items, then error: %v", totalMoved, err))
				}
				totalMoved += count
			}
			return statusMsg(fmt.Sprintf("Moved %d items", totalMoved))
		}

		if len(yankedSecrets) == 0 {
			return errorMsg("nothing yanked")
		}

		pastedCount := 0
		for _, yanked := range yankedSecrets {
			parts := strings.Split(strings.TrimSuffix(yanked.Path, "/"), "/")
			name := parts[len(parts)-1]
			dst := basePath + name

			// If destination exists, append _1, _2, etc.
			if _, err := m.client.Read(dst); err == nil {
				for i := 1; ; i++ {
					candidate := basePath + fmt.Sprintf("%s_%d", name, i)
					if _, err := m.client.Read(candidate); err != nil {
						dst = candidate
						break
					}
				}
			}

			if err := m.client.Write(dst, yanked.Data); err != nil {
				return errorMsg(fmt.Sprintf("pasted %d items, then error: %v", pastedCount, err))
			}

			if isCut {
				if err := m.client.Delete(yanked.Path); err != nil {
					return errorMsg(fmt.Sprintf("pasted to %s but failed to delete source: %v", dst, err))
				}
			}
			pastedCount++
		}

		if pastedCount == 1 {
			yanked := yankedSecrets[0]
			parts := strings.Split(strings.TrimSuffix(yanked.Path, "/"), "/")
			name := parts[len(parts)-1]
			dst := basePath + name
			if isCut {
				return undoableStatusMsg{
					status: fmt.Sprintf("Moved %s → %s", yanked.Path, dst),
					undo: model.UndoAction{
						Type:        model.UndoCutPaste,
						Description: fmt.Sprintf("move %s → %s", yanked.Path, dst),
						Path:        dst,
						OldPath:     yanked.Path,
						Data:        copyMap(yanked.Data),
						Keys:        copySlice(yanked.Keys),
					},
				}
			}
			return undoableStatusMsg{
				status: fmt.Sprintf("Pasted %s → %s", yanked.Path, dst),
				undo: model.UndoAction{
					Type:        model.UndoPasteSecret,
					Description: "paste to " + dst,
					Path:        dst,
					Data:        copyMap(yanked.Data),
					Keys:        copySlice(yanked.Keys),
				},
			}
		}

		if isCut {
			return statusMsg(fmt.Sprintf("Moved %d secrets", pastedCount))
		}
		return statusMsg(fmt.Sprintf("Pasted %d secrets", pastedCount))
	}
}

func (m *Model) deleteKey(key string) tea.Cmd {
	// Snapshot before change
	snapData := copyMap(m.secret.Data)
	snapKeys := copySlice(m.secret.Keys)
	secretPath := m.secret.Path

	// Mutate model state synchronously (safe — called from Update goroutine).
	delete(m.secret.Data, key)
	newKeys := make([]string, 0, len(m.secret.Keys))
	for _, k := range m.secret.Keys {
		if k != key {
			newKeys = append(newKeys, k)
		}
	}
	m.secret.Keys = newKeys

	if m.secretCursor >= len(m.secret.Keys) && m.secretCursor > 0 {
		m.secretCursor--
	}

	// Copy updated state for the async Vault write.
	writeData := copyMap(m.secret.Data)

	return func() tea.Msg {
		if err := m.client.Write(secretPath, writeData); err != nil {
			return errorMsg(err.Error())
		}
		return undoableStatusMsg{
			status: "Deleted key: " + key,
			undo: model.UndoAction{
				Type:        model.UndoEditSecret,
				Description: "delete key " + key,
				Path:        secretPath,
				Data:        snapData,
				Keys:        snapKeys,
			},
		}
	}
}

// editorCommand returns the editor to use, checking config, then $EDITOR,
// then $VISUAL, and falling back to "vim". It validates that the editor
// binary exists in PATH before returning.
func (m *Model) editorCommand() (string, error) {
	editor := "vim"
	if m.config != nil && m.config.Editor != "" {
		editor = m.config.Editor
	} else if e := os.Getenv("EDITOR"); e != "" {
		editor = e
	} else if e := os.Getenv("VISUAL"); e != "" {
		editor = e
	}
	if _, err := exec.LookPath(editor); err != nil {
		return "", fmt.Errorf("editor %q not found in PATH", editor)
	}
	return editor, nil
}

func (m *Model) openEditorForNewSecret(path string) tea.Cmd {
	data := []byte("{\n  \n}")

	tmpDir := os.Getenv("XDG_RUNTIME_DIR")
	if tmpDir == "" {
		tmpDir = os.TempDir()
	}
	tmpFile, err := os.CreateTemp(tmpDir, ".vau-edit-*.json")
	if err != nil {
		return func() tea.Msg { return errorMsg("tmpfile: " + err.Error()) }
	}
	if _, err := tmpFile.Write(data); err != nil {
		os.Remove(tmpFile.Name())
		return func() tea.Msg { return errorMsg("write: " + err.Error()) }
	}
	tmpFile.Close()
	os.Chmod(tmpFile.Name(), 0o600)

	editor, err := m.editorCommand()
	if err != nil {
		os.Remove(tmpFile.Name())
		return func() tea.Msg { return errorMsg(err.Error()) }
	}

	c := exec.Command(editor, tmpFile.Name())
	return tea.ExecProcess(c, func(err error) tea.Msg {
		defer os.Remove(tmpFile.Name())
		if err != nil {
			return errorMsg("editor: " + err.Error())
		}
		edited, err := os.ReadFile(tmpFile.Name())
		if err != nil {
			return errorMsg("read: " + err.Error())
		}
		var newData map[string]string
		if err := json.Unmarshal(edited, &newData); err != nil {
			return errorMsg("invalid JSON: " + err.Error())
		}
		return editorResultMsg{data: newData, newSecretPath: path}
	})
}

func (m *Model) addKeyValue(key, val string) tea.Cmd {
	snapData := copyMap(m.secret.Data)
	snapKeys := copySlice(m.secret.Keys)
	secretPath := m.secret.Path

	// Mutate model state synchronously (safe — called from Update goroutine).
	m.secret.Data[key] = val
	m.secret.Keys = append(m.secret.Keys, key)

	// Copy updated state for the async Vault write.
	writeData := copyMap(m.secret.Data)

	return func() tea.Msg {
		if err := m.client.Write(secretPath, writeData); err != nil {
			return errorMsg(err.Error())
		}
		return undoableStatusMsg{
			status: "Added key: " + key,
			undo: model.UndoAction{
				Type:        model.UndoEditSecret,
				Description: "add key " + key,
				Path:        secretPath,
				Data:        snapData,
				Keys:        snapKeys,
			},
		}
	}
}

func (m *Model) editValue(key, val string) tea.Cmd {
	snapData := copyMap(m.secret.Data)
	snapKeys := copySlice(m.secret.Keys)
	secretPath := m.secret.Path

	// Mutate model state synchronously (safe — called from Update goroutine).
	m.secret.Data[key] = val

	// Copy updated state for the async Vault write.
	writeData := copyMap(m.secret.Data)

	return func() tea.Msg {
		if err := m.client.Write(secretPath, writeData); err != nil {
			return errorMsg(err.Error())
		}
		return undoableStatusMsg{
			status: "Updated: " + key,
			undo: model.UndoAction{
				Type:        model.UndoEditSecret,
				Description: "edit " + key,
				Path:        secretPath,
				Data:        snapData,
				Keys:        snapKeys,
			},
		}
	}
}

// editSecretInEditor opens the secret data in an external editor as pretty-printed JSON.
func (m *Model) editSecretInEditor() tea.Cmd {
	data, err := json.MarshalIndent(m.secret.Data, "", "  ")
	if err != nil {
		return func() tea.Msg { return errorMsg("json: " + err.Error()) }
	}

	tmpDir := os.Getenv("XDG_RUNTIME_DIR")
	if tmpDir == "" {
		tmpDir = os.TempDir()
	}
	tmpFile, err := os.CreateTemp(tmpDir, ".vau-edit-*.json")
	if err != nil {
		return func() tea.Msg { return errorMsg("tmpfile: " + err.Error()) }
	}
	if _, err := tmpFile.Write(data); err != nil {
		os.Remove(tmpFile.Name())
		return func() tea.Msg { return errorMsg("write: " + err.Error()) }
	}
	tmpFile.Close()
	os.Chmod(tmpFile.Name(), 0o600)

	editor, err := m.editorCommand()
	if err != nil {
		os.Remove(tmpFile.Name())
		return func() tea.Msg { return errorMsg(err.Error()) }
	}

	c := exec.Command(editor, tmpFile.Name())
	return tea.ExecProcess(c, func(err error) tea.Msg {
		defer os.Remove(tmpFile.Name())
		if err != nil {
			return errorMsg("editor: " + err.Error())
		}

		edited, err := os.ReadFile(tmpFile.Name())
		if err != nil {
			return errorMsg("read: " + err.Error())
		}

		var newData map[string]string
		if err := json.Unmarshal(edited, &newData); err != nil {
			return errorMsg("invalid JSON: " + err.Error())
		}

		return editorResultMsg{data: newData}
	})
}

// saveSecretFromEditor writes the editor-modified secret to Vault and returns an undoable status.
func (m *Model) saveSecretFromEditor(secretPath string, snapData map[string]string, snapKeys []string) tea.Cmd {
	// Copy updated state for the async Vault write (avoid reading m.secret in goroutine).
	writeData := copyMap(m.secret.Data)

	return func() tea.Msg {
		if err := m.client.Write(secretPath, writeData); err != nil {
			return errorMsg(err.Error())
		}
		return undoableStatusMsg{
			status: "Updated secret via editor",
			undo: model.UndoAction{
				Type:        model.UndoEditSecret,
				Description: "editor edit",
				Path:        secretPath,
				Data:        snapData,
				Keys:        snapKeys,
			},
		}
	}
}

// --- Bulk selection helpers ---

func (m *Model) actualIdx(visibleIdx int) int {
	if m.filterQuery == "" {
		return visibleIdx
	}
	if visibleIdx < len(m.filteredIdx) {
		return m.filteredIdx[visibleIdx]
	}
	return visibleIdx
}

func (m *Model) selectedEntries() []model.Entry {
	var entries []model.Entry
	for idx := range m.selected {
		if idx < len(m.entries) {
			entries = append(entries, m.entries[idx])
		}
	}
	return entries
}

func (m *Model) selectedVisible() map[int]bool {
	if m.filterQuery == "" {
		return m.selected
	}
	vis := make(map[int]bool)
	for vi, ai := range m.filteredIdx {
		if m.selected[ai] {
			vis[vi] = true
		}
	}
	return vis
}

func (m *Model) bulkDelete(entries []model.Entry) tea.Cmd {
	return func() tea.Msg {
		deleted := 0
		for _, entry := range entries {
			path := m.currentPath() + strings.TrimSuffix(entry.Name, "/")
			if entry.IsDir {
				n, err := m.client.DeleteRecursive(path)
				if err != nil {
					return errorMsg(fmt.Sprintf("deleted %d, then error: %v", deleted, err))
				}
				deleted += n
			} else {
				if err := m.client.Delete(path); err != nil {
					return errorMsg(fmt.Sprintf("deleted %d, then error: %v", deleted, err))
				}
				deleted++
			}
		}
		m.selected = make(map[int]bool)
		return statusMsg(fmt.Sprintf("Deleted %d items", deleted))
	}
}

// --- Version history ---

func (m *Model) loadVersionHistory(path string) tea.Cmd {
	return func() tea.Msg {
		versions, err := m.client.ReadVersionMetadata(path)
		return versionHistoryMsg{path: path, versions: versions, err: err}
	}
}

func (m *Model) handleVersionHistoryKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc", "h":
		m.mode = model.ModeSecret
		m.versionHistory = nil
		return m, nil
	case "j", "down", "ctrl+n":
		if m.versionCursor < len(m.versionHistory)-1 {
			m.versionCursor++
		}
	case "k", "up", "ctrl+p":
		if m.versionCursor > 0 {
			m.versionCursor--
		}
	case "enter", "l":
		if len(m.versionHistory) > 0 && m.versionCursor < len(m.versionHistory) {
			sv := m.versionHistory[m.versionCursor]
			ver, _ := strconv.Atoi(sv.Version)
			path := m.versionPath
			return m, func() tea.Msg {
				secret, err := m.client.ReadVersion(path, ver)
				return versionDetailMsg{version: ver, secret: secret, err: err}
			}
		}
	}
	return m, nil
}

// --- Undo/redo execution ---

func (m *Model) executeUndo(action model.UndoAction) tea.Cmd {
	return func() tea.Msg {
		var reverse model.UndoAction
		var reloadSecret *model.Secret

		switch action.Type {
		case model.UndoCreateSecret:
			// Undo create = delete. Read current data for redo.
			secret, _ := m.client.Read(action.Path)
			if err := m.client.Delete(action.Path); err != nil {
				return errorMsg("undo: " + err.Error())
			}
			data := action.Data
			keys := action.Keys
			if secret != nil {
				data = copyMap(secret.Data)
				keys = copySlice(secret.Keys)
			}
			reverse = model.UndoAction{Type: model.UndoDeleteSecret, Description: action.Description, Path: action.Path, Data: data, Keys: keys}

		case model.UndoDeleteSecret:
			// Undo delete/cut = recreate
			if err := m.client.Write(action.Path, action.Data); err != nil {
				return errorMsg("undo: " + err.Error())
			}
			reverse = model.UndoAction{Type: model.UndoCreateSecret, Description: action.Description, Path: action.Path, Data: copyMap(action.Data), Keys: copySlice(action.Keys)}

		case model.UndoRenameSecret:
			// Undo rename: move from Path (current) back to OldPath (original).
			// Use MoveRecursive to handle both files and directories.
			if _, err := m.client.MoveRecursive(action.Path, action.OldPath); err != nil {
				return errorMsg("undo: " + err.Error())
			}
			reverse = model.UndoAction{Type: model.UndoRenameSecret, Description: action.Description, Path: action.OldPath, OldPath: action.Path}

		case model.UndoPasteSecret:
			// Undo paste = delete the pasted secret. Read data first for redo.
			secret, _ := m.client.Read(action.Path)
			if err := m.client.Delete(action.Path); err != nil {
				return errorMsg("undo: " + err.Error())
			}
			data := action.Data
			keys := action.Keys
			if secret != nil {
				data = copyMap(secret.Data)
				keys = copySlice(secret.Keys)
			}
			reverse = model.UndoAction{Type: model.UndoPasteSecret, Description: action.Description, Path: action.Path, Data: data, Keys: keys}

		case model.UndoCutPaste:
			// Undo cut+paste: if Data is nil it was a directory move, use MoveRecursive.
			if action.Data == nil {
				if _, err := m.client.MoveRecursive(action.Path, action.OldPath); err != nil {
					return errorMsg("undo: " + err.Error())
				}
				reverse = model.UndoAction{
					Type:        model.UndoCutPaste,
					Description: action.Description,
					Path:        action.OldPath,
					OldPath:     action.Path,
				}
			} else {
				if err := m.client.Delete(action.Path); err != nil {
					return errorMsg("undo: " + err.Error())
				}
				if err := m.client.Write(action.OldPath, action.Data); err != nil {
					return errorMsg("undo: " + err.Error())
				}
				reverse = model.UndoAction{
					Type:        model.UndoCutPaste,
					Description: action.Description,
					Path:        action.OldPath,
					OldPath:     action.Path,
					Data:        copyMap(action.Data),
					Keys:        copySlice(action.Keys),
				}
			}

		case model.UndoEditSecret:
			// Undo edit = restore snapshot. Read current state for redo.
			current, _ := m.client.Read(action.Path)
			if err := m.client.Write(action.Path, action.Data); err != nil {
				return errorMsg("undo: " + err.Error())
			}
			redoData := action.Data
			redoKeys := action.Keys
			if current != nil {
				redoData = copyMap(current.Data)
				redoKeys = copySlice(current.Keys)
			}
			reverse = model.UndoAction{Type: model.UndoEditSecret, Description: action.Description, Path: action.Path, Data: redoData, Keys: redoKeys}
			reloadSecret = &model.Secret{Path: action.Path, Data: copyMap(action.Data), Keys: copySlice(action.Keys)}

		default:
			return errorMsg("unknown undo action")
		}

		return redoableStatusMsg{status: "Undo: " + action.Description, redo: reverse, reloadSecret: reloadSecret}
	}
}

func (m *Model) executeRedo(action model.UndoAction) tea.Cmd {
	return func() tea.Msg {
		var reverse model.UndoAction
		var reloadSecret *model.Secret

		switch action.Type {
		case model.UndoCreateSecret:
			// Redo of undo-delete = delete again
			secret, _ := m.client.Read(action.Path)
			if err := m.client.Delete(action.Path); err != nil {
				return errorMsg("redo: " + err.Error())
			}
			data := action.Data
			keys := action.Keys
			if secret != nil {
				data = copyMap(secret.Data)
				keys = copySlice(secret.Keys)
			}
			reverse = model.UndoAction{Type: model.UndoDeleteSecret, Description: action.Description, Path: action.Path, Data: data, Keys: keys}

		case model.UndoDeleteSecret:
			// Redo of undo-create = recreate
			if err := m.client.Write(action.Path, action.Data); err != nil {
				return errorMsg("redo: " + err.Error())
			}
			reverse = model.UndoAction{Type: model.UndoCreateSecret, Description: action.Description, Path: action.Path, Data: copyMap(action.Data), Keys: copySlice(action.Keys)}

		case model.UndoRenameSecret:
			// Redo rename: use MoveRecursive to handle both files and directories.
			if _, err := m.client.MoveRecursive(action.Path, action.OldPath); err != nil {
				return errorMsg("redo: " + err.Error())
			}
			reverse = model.UndoAction{Type: model.UndoRenameSecret, Description: action.Description, Path: action.OldPath, OldPath: action.Path}

		case model.UndoPasteSecret:
			// Redo paste = recreate the pasted secret
			if err := m.client.Write(action.Path, action.Data); err != nil {
				return errorMsg("redo: " + err.Error())
			}
			reverse = model.UndoAction{Type: model.UndoPasteSecret, Description: action.Description, Path: action.Path, Data: copyMap(action.Data), Keys: copySlice(action.Keys)}

		case model.UndoCutPaste:
			// Redo cut+paste: if Data is nil it was a directory move, use MoveRecursive.
			if action.Data == nil {
				if _, err := m.client.MoveRecursive(action.Path, action.OldPath); err != nil {
					return errorMsg("redo: " + err.Error())
				}
				reverse = model.UndoAction{
					Type:        model.UndoCutPaste,
					Description: action.Description,
					Path:        action.OldPath,
					OldPath:     action.Path,
				}
			} else {
				if err := m.client.Write(action.Path, action.Data); err != nil {
					return errorMsg("redo: " + err.Error())
				}
				if err := m.client.Delete(action.OldPath); err != nil {
					return errorMsg("redo: " + err.Error())
				}
				reverse = model.UndoAction{
					Type:        model.UndoCutPaste,
					Description: action.Description,
					Path:        action.OldPath,
					OldPath:     action.Path,
					Data:        copyMap(action.Data),
					Keys:        copySlice(action.Keys),
				}
			}

		case model.UndoEditSecret:
			// Redo edit = apply the redo snapshot
			current, _ := m.client.Read(action.Path)
			if err := m.client.Write(action.Path, action.Data); err != nil {
				return errorMsg("redo: " + err.Error())
			}
			undoData := action.Data
			undoKeys := action.Keys
			if current != nil {
				undoData = copyMap(current.Data)
				undoKeys = copySlice(current.Keys)
			}
			reverse = model.UndoAction{Type: model.UndoEditSecret, Description: action.Description, Path: action.Path, Data: undoData, Keys: undoKeys}
			reloadSecret = &model.Secret{Path: action.Path, Data: copyMap(action.Data), Keys: copySlice(action.Keys)}

		default:
			return errorMsg("unknown redo action")
		}

		return undoableStatusMsg{status: "Redo: " + action.Description, undo: reverse, reloadSecret: reloadSecret}
	}
}

// --- Result handlers ---

func (m *Model) handleListResult(msg listResultMsg) (tea.Model, tea.Cmd) {
	if msg.path == "@@mount_preview@@" {
		// Preview result for mount selection
		if msg.err != nil {
			m.previewEntries = nil
		} else {
			m.previewEntries = msg.entries
		}
		m.previewSecret = nil
		return m, nil
	}

	if msg.err != nil {
		m.errMsg = msg.err.Error()
		return m, nil
	}

	currentPath := m.currentPath()
	parentPath := m.parentPath()

	if msg.path == currentPath {
		// If the directory is now empty and we're not at mount root,
		// auto-navigate up to the lowest still-existing ancestor.
		if len(msg.entries) == 0 && len(m.path) > 0 {
			m.path = m.path[:len(m.path)-1]
			return m, tea.Batch(m.listDir(m.currentPath()), m.listParent())
		}
		m.entries = msg.entries
		// Re-apply filter if active
		if m.filterQuery != "" {
			m.applyFilter()
		}
		vis := m.visibleEntries()
		if m.cursor >= len(vis) {
			m.cursor = max(0, len(vis)-1)
		}
		return m, m.loadPreview()
	}

	if msg.path == parentPath {
		m.parentList = msg.entries
		if len(m.path) > 0 {
			// Update parent's cursor memory to point at the current directory entry
			currentDir := m.path[len(m.path)-1]
			for i, e := range m.parentList {
				if e.Name == currentDir {
					if m.cursorMemory == nil {
						m.cursorMemory = make(map[string]int)
					}
					m.cursorMemory[m.parentPath()] = i
					break
				}
			}
		}
		return m, nil
	}

	// Must be a preview result
	m.previewEntries = msg.entries
	m.previewSecret = nil
	return m, nil
}

func (m *Model) handleSecretResult(msg secretResultMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.errMsg = msg.err.Error()
		return m, nil
	}

	if msg.openEditor {
		// Open secret directly in external editor
		m.secret = msg.secret
		m.secretJSONView = true
		return m, m.editSecretInEditor()
	}

	if msg.openPopup {
		// Open secret in popup overlay
		m.mode = model.ModeSecret
		m.secret = msg.secret
		m.secretCursor = 0
		m.revealed = make(map[string]bool)
		m.secretBase64 = make(map[string]bool)
		m.secretAllRevealed = false
		m.secretJSONView = false
		return m, nil
	}

	// Preview result
	m.previewSecret = msg.secret
	m.previewEntries = nil
	return m, nil
}
