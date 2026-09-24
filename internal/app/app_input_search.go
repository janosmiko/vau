package app

import (
	"maps"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/janosmiko/vau/internal/model"
)

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
	maps.Copy(c, src)
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
	maps.Copy(c, src)
	return c
}

func copyMapIntBool(src map[int]bool) map[int]bool {
	if src == nil {
		return nil
	}
	c := make(map[int]bool, len(src))
	maps.Copy(c, src)
	return c
}

// --- Input handling (using textinput) ---

func (m *Model) startInput(action model.InputAction, label, value string) tea.Cmd {
	m.prevInputMode = m.mode
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
		m.mode = m.prevInputMode
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

	case model.InputNewPolicy:
		m.mode = model.ModePolicyList
		m.inputAction = model.InputNone
		if value == "" {
			return m, nil
		}
		return m, m.createPolicyWithEditor(value)

	case model.InputNewRole:
		m.mode = model.ModeRoleList
		m.inputAction = model.InputNone
		if value == "" {
			return m, nil
		}
		return m, m.createRoleWithEditor(m.roleAuthPath, value)
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
				return m, m.loadJumpCompletions()
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
	return m, tea.Batch(cmd, m.loadJumpCompletions())
}

// loadJumpCompletions returns a command that lists the parent directory of
// the current input and filters entries matching the typed prefix.
func (m *Model) loadJumpCompletions() tea.Cmd {
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

	return func() tea.Msg {
		entries, err := m.client.List(parentPath)
		if err != nil || entries == nil {
			return jumpCompletionsMsg{input: input}
		}

		lowerPrefix := strings.ToLower(prefix)
		var completions []string
		for _, e := range entries {
			if strings.HasPrefix(strings.ToLower(e.Name), lowerPrefix) {
				completions = append(completions, parentPath+e.Name)
			}
		}
		return jumpCompletionsMsg{input: input, completions: completions}
	}
}

// jumpToPath navigates the explorer to the given Vault path.
func (m *Model) jumpToPath(input string) tea.Cmd {
	m.clearFilter()
	m.selected = make(map[int]bool)
	m.cursorMemory = nil // reset history on direct jump

	if cleaned, ok := strings.CutSuffix(input, "/"); ok {
		// Directory navigation: set path segments and refresh
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
