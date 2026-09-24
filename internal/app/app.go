package app

import (
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
		mount   string
		path    string
		entries []model.Entry
		err     error
	}
	secretResultMsg struct {
		mount      string
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
	statusMsg        string
	errorMsg         string
	confirmCreateMsg struct { // path to create after overwrite confirmation
		path  string
		mount string
	}
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
		mount    string
		path     string
		versions []model.SecretVersion
		err      error
	}
	versionDetailMsg struct {
		mount   string
		path    string
		version int
		secret  *model.Secret
		err     error
	}
	editorResultMsg struct {
		data          map[string]string
		newSecretPath string // non-empty when creating a new secret via editor
	}
	newSecretEditorMsg struct { // path for new secret to open in editor
		path  string
		mount string
	}
	confirmCreateEditorMsg struct { // path to create after overwrite confirmation (editor flow)
		path  string
		mount string
	}
	newSecretInlineMsg struct {
		secret *model.Secret
		mount  string
	}
	yankResultMsg struct {
		seq     int
		mount   string
		secrets []*model.Secret
		paths   []string
		isCut   bool
		isDir   bool
	}
	jumpCompletionsMsg struct {
		input       string
		completions []string
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
	mount          string         // which mount this tab is using
	viewMode       model.ViewMode // preserved for workspace views

	// Workspace state (per-tab)
	policies          []string
	policyCursor      int
	policyPreview     string
	authMethods       []model.Entry
	authCursor        int
	rolePreview       []model.Entry
	roles             []model.Entry
	roleCursor        int
	roleAuthPath      string
	roleDataPreview   map[string]any
	entities          []model.Entry
	entityCursor      int
	entityDataPreview map[string]any
	groups            []model.Entry
	groupCursor       int
	groupDataPreview  map[string]any
	tokenAccessors    []model.Entry
	tokenCursor       int
	tokenDataPreview  map[string]any
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
	mode               model.ViewMode
	secret             *model.Secret
	secretCursor       int
	revealed           map[string]bool
	secretBase64       map[string]bool // tracks base64 decode toggle per key
	secretAllRevealed  bool
	secretJSONView     bool
	dockerFields       []model.DockerConfigField
	dockerCursor       int
	dockerRevealed     bool
	dockerTitle        string
	secretEditKey      string            // key being inline-edited
	secretEditColumn   int               // 0=key, 1=value column being edited
	secretEditOrigKey  string            // original key name before key-column edit (for rename)
	secretEditSnapData map[string]string // secret.Data snapshot taken when inline editing started
	secretEditSnapKeys []string          // secret.Keys snapshot taken when inline editing started

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
	prevInputMode   model.ViewMode // mode to return to after input
	confirmInput    textinput.Model

	// Input prompt
	inputAction model.InputAction
	inputLabel  string
	inputBuffer string // for multi-step inputs (e.g., new key then value)
	textInput   textinput.Model

	// Clipboard: stores yanked secret data for single or bulk operations
	yankedSecrets []*model.Secret // yanked secrets (one or more)
	yankPaths     []string        // paths of yanked items (for directory operations)
	yankMount     string          // mount path at yank time (for cross-mount detection)
	yankIsCut     bool            // true if yanked via cut (x) — paste will delete source
	yankIsDir     bool            // true if yanked items include directories
	yankSeq       int             // drops yank results that finish after a newer yank

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
	copySecret        *model.Secret // secret to copy when format key arrives

	// Theme picker
	themeEntries      []ui.ThemeEntry // grouped theme list with headers
	themeCursor       int
	activeColorscheme string // currently applied colorscheme name

	// Progress tracking for long-running operations
	program  *tea.Program  // reference to the tea.Program for sending progress updates
	progress progressState // tracks active operation progress

	// Policy and access category state
	policies      []string // cached policy names
	policyCursor  int
	policyName    string // name of the policy being viewed
	policyContent string // HCL content of the viewed policy
	policyScroll  int
	policyPreview string // HCL preview for selected policy in list

	authMethods     []model.Entry // cached auth method list
	authCursor      int
	roles           []model.Entry // cached role list for selected auth method
	roleCursor      int
	roleName        string
	roleAuthPath    string         // which auth method's roles we're viewing
	roleData        map[string]any // role config data for viewed role
	roleScroll      int
	rolePreview     []model.Entry  // role names preview for selected auth method
	roleDataPreview map[string]any // role data preview for selected role

	entities          []model.Entry // cached entity list
	entityCursor      int
	entityName        string
	entityData        map[string]any // entity data for viewed entity
	entityScroll      int
	entityDataPreview map[string]any // entity data preview for selected entity

	groups           []model.Entry // cached group list
	groupCursor      int
	groupName        string
	groupData        map[string]any // group data for viewed group
	groupScroll      int
	groupDataPreview map[string]any // group data preview for selected group

	tokenAccessors   []model.Entry // cached token accessor list
	tokenCursor      int
	tokenData        map[string]any // metadata for viewed token
	tokenScroll      int
	tokenDataPreview map[string]any // preview for selected accessor
	tokenCreatedData map[string]any // newly created token data (one-time view)
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
			bookmarks = append(bookmarks, config.Bookmark{Name: name, Mount: cb.Mount, Path: cb.Path, Slot: cb.Slot})
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
	t.viewMode = m.mode

	// Workspace state
	t.policies = append([]string(nil), m.policies...)
	t.policyCursor = m.policyCursor
	t.policyPreview = m.policyPreview
	t.authMethods = append([]model.Entry(nil), m.authMethods...)
	t.authCursor = m.authCursor
	t.rolePreview = append([]model.Entry(nil), m.rolePreview...)
	t.roles = append([]model.Entry(nil), m.roles...)
	t.roleCursor = m.roleCursor
	t.roleAuthPath = m.roleAuthPath
	t.roleDataPreview = m.roleDataPreview
	t.entities = append([]model.Entry(nil), m.entities...)
	t.entityCursor = m.entityCursor
	t.entityDataPreview = m.entityDataPreview
	t.groups = append([]model.Entry(nil), m.groups...)
	t.groupCursor = m.groupCursor
	t.groupDataPreview = m.groupDataPreview
	t.tokenAccessors = append([]model.Entry(nil), m.tokenAccessors...)
	t.tokenCursor = m.tokenCursor
	t.tokenDataPreview = m.tokenDataPreview
}

// loadTab restores Model fields from the given tab index.
func (m *Model) loadTab(idx int) {
	t := m.tabs[idx]
	m.activeTab = idx

	// Restore mode — for overlay modes (secret popup, confirm, etc.) fall back to explorer;
	// for workspace list modes (policies, auth methods, etc.) preserve them.
	switch t.viewMode {
	case model.ModeExplorer, model.ModePolicyList, model.ModeAuthMethods,
		model.ModeRoleList, model.ModeEntityList, model.ModeGroupList,
		model.ModeTokenList:
		m.mode = t.viewMode
	default:
		m.mode = model.ModeExplorer
	}
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
	m.client = m.client.WithMount(t.mount)

	// Workspace state
	m.policies = append([]string(nil), t.policies...)
	m.policyCursor = t.policyCursor
	m.policyPreview = t.policyPreview
	m.authMethods = append([]model.Entry(nil), t.authMethods...)
	m.authCursor = t.authCursor
	m.rolePreview = append([]model.Entry(nil), t.rolePreview...)
	m.roles = append([]model.Entry(nil), t.roles...)
	m.roleCursor = t.roleCursor
	m.roleAuthPath = t.roleAuthPath
	m.roleDataPreview = t.roleDataPreview
	m.entities = append([]model.Entry(nil), t.entities...)
	m.entityCursor = t.entityCursor
	m.entityDataPreview = t.entityDataPreview
	m.groups = append([]model.Entry(nil), t.groups...)
	m.groupCursor = t.groupCursor
	m.groupDataPreview = t.groupDataPreview
	m.tokenAccessors = append([]model.Entry(nil), t.tokenAccessors...)
	m.tokenCursor = t.tokenCursor
	m.tokenDataPreview = t.tokenDataPreview
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(m.listDir(""), m.loadMounts())
}
