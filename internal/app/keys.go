package app

// KeyMap holds all configurable keybindings. Each field is a list of key
// strings that trigger the action (first match wins). Users override
// individual actions via the "keybindings" map in config.yaml; an override
// replaces the entire default list for that action with a single key.
type KeyMap struct {
	// Explorer navigation
	Up       []string
	Down     []string
	Left     []string
	Right    []string
	Top      []string
	Bottom   []string
	HalfDown []string
	HalfUp   []string
	FullDown []string
	FullUp   []string
	Open     []string

	// Actions
	Search          []string
	Filter          []string
	JumpPath        []string
	NewSecret       []string
	NewSecretEditor []string
	Edit            []string
	Rename          []string
	Delete          []string
	Yank            []string
	Paste           []string
	Cut             []string
	Undo            []string
	Redo            []string
	Refresh         []string
	ToggleValues    []string
	ToggleJSON      []string
	Select          []string
	Help            []string
	Quit            []string

	// Tabs
	NewTab   []string
	NextTab  []string
	PrevTab  []string
	CloseTab []string

	// Bookmarks
	BookmarkSave []string
	BookmarkShow []string

	// Theme
	ThemePicker []string
}

// DefaultKeyMap returns a KeyMap populated with the default keybindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up:       []string{"j", "down", "ctrl+n"},
		Down:     []string{"k", "up", "ctrl+p"},
		Left:     []string{"h", "left"},
		Right:    []string{"l", "right"},
		Top:      []string{"g"},
		Bottom:   []string{"G"},
		HalfDown: []string{"ctrl+d"},
		HalfUp:   []string{"ctrl+u"},
		FullDown: []string{"ctrl+f"},
		FullUp:   []string{"ctrl+b"},
		Open:     []string{"enter"},

		Search:          []string{"/"},
		Filter:          []string{"f"},
		JumpPath:        []string{"J"},
		NewSecret:       []string{"a"},
		NewSecretEditor: []string{"A"},
		Edit:            []string{"e"},
		Rename:          []string{"r"},
		Delete:          []string{"D"},
		Yank:            []string{"y"},
		Paste:           []string{"p"},
		Cut:             []string{"x"},
		Undo:            []string{"u"},
		Redo:            []string{"ctrl+r"},
		Refresh:         []string{"R"},
		ToggleValues:    []string{"v"},
		ToggleJSON:      []string{"V"},
		Select:          []string{" "},
		Help:            []string{"?"},
		Quit:            []string{"q"},

		NewTab:   []string{"t"},
		NextTab:  []string{"]"},
		PrevTab:  []string{"["},
		CloseTab: []string{"ctrl+c"},

		BookmarkSave: []string{"B"},
		BookmarkShow: []string{"b"},

		ThemePicker: []string{"T"},
	}
}

// actionMap returns a mapping from config action names to pointers into the
// KeyMap fields. This is the single source of truth for which config keys
// map to which struct fields.
func (km *KeyMap) actionMap() map[string]*[]string {
	return map[string]*[]string{
		"navigate_up":       &km.Up,
		"navigate_down":     &km.Down,
		"navigate_left":     &km.Left,
		"navigate_right":    &km.Right,
		"top":               &km.Top,
		"bottom":            &km.Bottom,
		"half_down":         &km.HalfDown,
		"half_up":           &km.HalfUp,
		"full_down":         &km.FullDown,
		"full_up":           &km.FullUp,
		"open":              &km.Open,
		"search":            &km.Search,
		"filter":            &km.Filter,
		"jump_path":         &km.JumpPath,
		"new_secret":        &km.NewSecret,
		"new_secret_editor": &km.NewSecretEditor,
		"edit":              &km.Edit,
		"rename":            &km.Rename,
		"delete":            &km.Delete,
		"yank":              &km.Yank,
		"paste":             &km.Paste,
		"cut":               &km.Cut,
		"undo":              &km.Undo,
		"redo":              &km.Redo,
		"refresh":           &km.Refresh,
		"toggle_values":     &km.ToggleValues,
		"toggle_json":       &km.ToggleJSON,
		"select":            &km.Select,
		"help":              &km.Help,
		"quit":              &km.Quit,
		"new_tab":           &km.NewTab,
		"next_tab":          &km.NextTab,
		"prev_tab":          &km.PrevTab,
		"close_tab":         &km.CloseTab,
		"bookmark_save":     &km.BookmarkSave,
		"bookmark_show":     &km.BookmarkShow,
		"theme_picker":      &km.ThemePicker,
	}
}

// ApplyOverrides replaces default bindings with user-configured overrides.
// Each override value is a single key string that replaces the entire default
// list for that action. Unknown action names are silently ignored.
func (km *KeyMap) ApplyOverrides(overrides map[string]string) {
	if len(overrides) == 0 {
		return
	}
	am := km.actionMap()
	for action, key := range overrides {
		target, ok := am[action]
		if !ok {
			continue // unknown action — ignore gracefully
		}
		*target = []string{key}
	}
}

// matchKey reports whether key matches any of the configured bindings.
func matchKey(key string, bindings []string) bool {
	for _, b := range bindings {
		if key == b {
			return true
		}
	}
	return false
}
