package app

import (
	"encoding/base64"
	"fmt"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/janosmiko/vau/internal/model"
)

// handleSecretNavKey handles cursor movement and view-toggle keys in the secret popup.
func (m *Model) handleSecretNavKey(key string) (tea.Model, tea.Cmd, bool) {
	switch key {
	case "q", "esc", "h":
		m.mode = model.ModeExplorer
		m.secret = nil
		m.secretCursor = 0
		m.revealed = make(map[string]bool)
		m.secretBase64 = make(map[string]bool)
		m.secretAllRevealed = false
		m.secretJSONView = false
		return m, m.refresh(), true

	case "j", "down", "ctrl+n":
		if m.secret != nil && m.secretCursor < len(m.secret.Keys)-1 {
			m.secretCursor++
		}
		return m, nil, true

	case "k", "up", "ctrl+p":
		if m.secretCursor > 0 {
			m.secretCursor--
		}
		return m, nil, true

	case "v", "tab":
		// Toggle reveal ALL values
		if m.secret != nil {
			m.secretJSONView = false
			m.secretAllRevealed = !m.secretAllRevealed
			for _, k := range m.secret.Keys {
				m.revealed[k] = m.secretAllRevealed
			}
		}
		return m, nil, true

	case "V":
		// Toggle JSON view
		if m.secret != nil {
			m.secretJSONView = !m.secretJSONView
		}
		return m, nil, true

	case "ctrl+d":
		// Half-page scroll down in secret popup
		if m.secret != nil && len(m.secret.Keys) > 0 {
			halfPage := m.secretPopupHalfPage()
			m.secretCursor += halfPage
			if m.secretCursor >= len(m.secret.Keys) {
				m.secretCursor = len(m.secret.Keys) - 1
			}
		}
		return m, nil, true

	case "ctrl+u":
		// Half-page scroll up in secret popup
		if m.secret != nil && len(m.secret.Keys) > 0 {
			halfPage := m.secretPopupHalfPage()
			m.secretCursor -= halfPage
			if m.secretCursor < 0 {
				m.secretCursor = 0
			}
		}
		return m, nil, true

	case "ctrl+f":
		// Full-page scroll down in secret popup
		if m.secret != nil && len(m.secret.Keys) > 0 {
			fullPage := m.secretPopupHalfPage() * 2
			m.secretCursor += fullPage
			if m.secretCursor >= len(m.secret.Keys) {
				m.secretCursor = len(m.secret.Keys) - 1
			}
		}
		return m, nil, true

	case "ctrl+b":
		// Full-page scroll up in secret popup
		if m.secret != nil && len(m.secret.Keys) > 0 {
			fullPage := m.secretPopupHalfPage() * 2
			m.secretCursor -= fullPage
			if m.secretCursor < 0 {
				m.secretCursor = 0
			}
		}
		return m, nil, true
	}
	return m, nil, false
}

// handleSecretClipboardKey handles copy/paste keys in the secret popup.
func (m *Model) handleSecretClipboardKey(key string) (tea.Model, tea.Cmd, bool) {
	switch key {
	case "y":
		if m.secret == nil {
			return m, nil, true
		}
		if m.secretJSONView {
			// Copy entire secret as JSON
			return m, m.copySecretAsJSON(), true
		}
		// Copy selected value
		if len(m.secret.Keys) > 0 {
			key := m.secret.Keys[m.secretCursor]
			val := m.secret.Data[key]
			return m, copyToSystemClipboard(val, key), true
		}
		return m, nil, true

	case "Y":
		// Copy entire secret — choose format
		if m.secret != nil {
			m.copySecret = m.secret
			m.copyFormatPending = true
			m.status = "Copy as: (j)son  (y)aml  (d)otenv"
			return m, nil, true
		}
		return m, nil, true

	case "p":
		if m.secret != nil && len(m.secret.Keys) > 0 {
			val, err := readFromSystemClipboard()
			if err != nil {
				m.errMsg = "clipboard: " + err.Error()
				return m, nil, true
			}
			key := m.secret.Keys[m.secretCursor]
			return m, m.editValue(key, val), true
		}
		return m, nil, true
	}
	return m, nil, false
}

// handleSecretEditActionKey handles edit/add/delete keys in the secret popup.
func (m *Model) handleSecretEditActionKey(key string) (tea.Model, tea.Cmd, bool) {
	switch key {
	case "e":
		if m.secret != nil {
			if m.secretJSONView {
				return m, m.editSecretInEditor(), true
			}
			if len(m.secret.Keys) > 0 {
				key := m.secret.Keys[m.secretCursor]
				m.secretEditKey = key
				m.secretEditOrigKey = key
				m.secretEditColumn = 1 // edit value column
				m.secretEditSnapData = copyMap(m.secret.Data)
				m.secretEditSnapKeys = copySlice(m.secret.Keys)
				m.mode = model.ModeSecretEdit
				m.textInput.SetValue(m.secret.Data[key])
				m.textInput.Focus()
				m.textInput.CursorEnd()
				return m, textinput.Blink, true
			}
		}
		return m, nil, true

	case "a":
		// Inline add: append empty row and start editing the key column
		if m.secret != nil {
			m.secretEditSnapData = copyMap(m.secret.Data)
			m.secretEditSnapKeys = copySlice(m.secret.Keys)
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
			return m, textinput.Blink, true
		}
		return m, nil, true

	case "D":
		if m.secret != nil && len(m.secret.Keys) > 0 {
			key := m.secret.Keys[m.secretCursor]
			m.confirmMsg = fmt.Sprintf("Delete key %q?", key)
			m.prevConfirmMode = model.ModeSecret
			m.confirmAction = func() tea.Cmd {
				return m.deleteKey(key)
			}
			m.enterConfirmMode()
			return m, nil, true
		}
		return m, nil, true
	}
	return m, nil, false
}

// handleSecretMiscKey handles base64, dockerconfig, and version-history keys.
func (m *Model) handleSecretMiscKey(key string) (tea.Model, tea.Cmd, bool) {
	switch key {
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
						return m, nil, true
					}
				}
				m.secretBase64[key] = true
			}
		}
		return m, nil, true

	case "d":
		if m.secret != nil && len(m.secret.Keys) > 0 {
			key := m.secret.Keys[m.secretCursor]
			fields, ok := parseDockerConfig(m.secret.Data[key])
			if !ok {
				m.errMsg = "Not a dockerconfigjson value"
				return m, nil, true
			}
			m.dockerFields = fields
			m.dockerCursor = 0
			m.dockerRevealed = false
			m.dockerTitle = m.secret.Path + "/" + key
			m.mode = model.ModeDockerConfig
		}
		return m, nil, true

	case "H":
		if m.secret != nil {
			if m.client.IsKV1() {
				m.errMsg = "Version history not available for KV v1"
				return m, nil, true
			}
			m.versionPath = m.secret.Path
			return m, m.loadVersionHistory(m.secret.Path), true
		}
		return m, nil, true
	}
	return m, nil, false
}

func (m *Model) handleSecretKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Handle copy format pending (Y + j/y/d)
	if m.copyFormatPending {
		m.copyFormatPending = false
		return m.handleCopyFormat(key)
	}

	for _, handler := range []func(string) (tea.Model, tea.Cmd, bool){
		m.handleSecretNavKey,
		m.handleSecretClipboardKey,
		m.handleSecretEditActionKey,
		m.handleSecretMiscKey,
	} {
		if result, cmd, handled := handler(key); handled {
			return result, cmd
		}
	}
	return m, nil
}

func (m *Model) handleSecretEditKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// Cancel: revert any uncommitted changes
		m.textInput.Blur()
		if m.secretEditSnapData != nil {
			// Restore the state captured when editing started, undoing any
			// tab-committed rename/value that never reached Vault.
			m.secret.Data = m.secretEditSnapData
			m.secret.Keys = m.secretEditSnapKeys
			if m.secretCursor >= len(m.secret.Keys) && m.secretCursor > 0 {
				m.secretCursor = len(m.secret.Keys) - 1
			}
		} else if m.secretEditOrigKey == "" && m.secretEditKey == "" {
			// No snapshot (e.g. editing a freshly created secret) — fall back
			// to removing the placeholder empty-key row.
			delete(m.secret.Data, "")
			m.secret.Keys = m.removeLastEmptyKey()
			if m.secretCursor >= len(m.secret.Keys) && m.secretCursor > 0 {
				m.secretCursor = len(m.secret.Keys) - 1
			}
		}
		m.mode = model.ModeSecret
		m.secretEditKey = ""
		m.secretEditOrigKey = ""
		m.secretEditColumn = 0
		m.secretEditSnapData = nil
		m.secretEditSnapKeys = nil
		return m, nil

	case "tab":
		// Save current column input and switch to the other column
		return m.switchEditColumn()

	case "shift+tab":
		// Same as tab — toggle between columns
		return m.switchEditColumn()

	case "enter":
		// Save the current input and commit all changes
		if !m.applyCurrentEditInput() {
			// Rename collision: stay in edit mode so the user can fix the name.
			return m, nil
		}
		m.textInput.Blur()

		origKey := m.secretEditOrigKey
		currentKey := m.secretEditKey
		val := m.secret.Data[currentKey]
		snapData, snapKeys := m.secretEditSnapData, m.secretEditSnapKeys

		m.mode = model.ModeSecret
		m.secretEditKey = ""
		m.secretEditOrigKey = ""
		m.secretEditColumn = 0
		m.secretEditSnapData = nil
		m.secretEditSnapKeys = nil

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
			return m, m.renameKey(origKey, currentKey, val, snapData, snapKeys)
		}

		// Only value changed
		return m, m.editValueWithSnapshot(currentKey, val, snapData, snapKeys)
	}
	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

// switchEditColumn saves the current column's textinput and switches to the other column.
func (m *Model) switchEditColumn() (tea.Model, tea.Cmd) {
	if !m.applyCurrentEditInput() {
		// Rename collision: stay on the key column so the user can fix it.
		return m, nil
	}

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

// applyCurrentEditInput writes the textinput value back to the appropriate
// field. It returns false without changing anything if a key rename would
// collide with an existing key.
func (m *Model) applyCurrentEditInput() bool {
	if m.secretEditColumn == 0 {
		// Editing key column — rename in-place
		newKey := m.textInput.Value()
		oldKey := m.secretEditKey
		if newKey != oldKey {
			if _, exists := m.secret.Data[newKey]; exists {
				m.errMsg = fmt.Sprintf("Key %q already exists", newKey)
				return false
			}
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
	return true
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

// renameKey handles renaming a key in the secret (delete old, add new with
// value). snapData/snapKeys must be the secret's Data/Keys as they were
// before the inline edit started, so undo restores the pre-edit state.
func (m *Model) renameKey(oldKey, newKey, val string, snapData map[string]string, snapKeys []string) tea.Cmd {
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
	client := m.client

	return func() tea.Msg {
		if err := client.Write(secretPath, writeData); err != nil {
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
				Mount:       client.Mount(),
			},
		}
	}
}
