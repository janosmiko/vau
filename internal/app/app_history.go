package app

import (
	"fmt"
	"strconv"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/janosmiko/vau/internal/model"
	"github.com/janosmiko/vau/internal/vault"
)

// --- Version history ---

func (m *Model) loadVersionHistory(path string) tea.Cmd {
	client := m.client
	return func() tea.Msg {
		versions, err := client.ReadVersionMetadata(path)
		return versionHistoryMsg{mount: client.Mount(), path: path, versions: versions, err: err}
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
			client := m.client
			return m, func() tea.Msg {
				secret, err := client.ReadVersion(path, ver)
				return versionDetailMsg{mount: client.Mount(), path: path, version: ver, secret: secret, err: err}
			}
		}
	case "D":
		// Destroy selected version.
		if len(m.versionHistory) > 0 && m.versionCursor < len(m.versionHistory) {
			sv := m.versionHistory[m.versionCursor]
			if sv.Destroyed {
				m.status = fmt.Sprintf("Version %s is already destroyed", sv.Version)
				return m, nil
			}
			ver, _ := strconv.Atoi(sv.Version)
			path := m.versionPath
			m.confirmMsg = fmt.Sprintf("Permanently destroy version %s?", sv.Version)
			m.prevConfirmMode = model.ModeVersionHistory
			m.confirmAction = func() tea.Cmd {
				return m.destroyVersion(path, ver)
			}
			m.enterConfirmMode()
			return m, nil
		}
	case "ctrl+x":
		// Destroy all old versions, keep latest.
		if len(m.versionHistory) > 1 {
			path := m.versionPath
			m.confirmMsg = fmt.Sprintf("Destroy all old versions of %s? (keeps latest)", path)
			m.prevConfirmMode = model.ModeVersionHistory
			m.confirmAction = func() tea.Cmd {
				return m.destroyOldVersions(path)
			}
			m.enterConfirmMode()
			return m, nil
		}
		m.status = "Only one version exists"
		return m, nil
	}
	return m, nil
}

func (m *Model) destroyVersion(path string, version int) tea.Cmd {
	client := m.client
	return func() tea.Msg {
		if err := client.DestroyVersions(path, []int{version}); err != nil {
			return errorMsg(fmt.Sprintf("destroy version %d: %v", version, err))
		}
		// Reload version history.
		versions, err := client.ReadVersionMetadata(path)
		if err != nil {
			return statusMsg(fmt.Sprintf("Destroyed version %d (reload failed: %v)", version, err))
		}
		return versionHistoryMsg{mount: client.Mount(), path: path, versions: versions}
	}
}

func (m *Model) destroyOldVersions(path string) tea.Cmd {
	client := m.client
	return func() tea.Msg {
		count, err := client.DestroyOldVersions(path)
		if err != nil {
			return errorMsg(fmt.Sprintf("destroy old versions: %v", err))
		}
		// Reload version history.
		versions, err := client.ReadVersionMetadata(path)
		if err != nil {
			return statusMsg(fmt.Sprintf("Destroyed %d old versions (reload failed: %v)", count, err))
		}
		return versionHistoryMsg{mount: client.Mount(), path: path, versions: versions}
	}
}

// --- Undo/redo execution ---

func (m *Model) actionClient(action model.UndoAction) *vault.Client {
	if action.Mount == "" {
		return m.client
	}
	return m.client.WithMount(action.Mount)
}

func (m *Model) onActionMount(action model.UndoAction) bool {
	return action.Mount == "" || action.Mount == m.client.Mount()
}

func (m *Model) executeUndo(action model.UndoAction) tea.Cmd {
	client := m.actionClient(action)
	return func() tea.Msg {
		var reverse model.UndoAction
		var reloadSecret *model.Secret

		switch action.Type {
		case model.UndoCreateSecret:
			// Undo create = delete. Read current data for redo.
			secret, _ := client.Read(action.Path)
			if err := client.Delete(action.Path); err != nil {
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
			if err := client.Write(action.Path, action.Data); err != nil {
				return errorMsg("undo: " + err.Error())
			}
			reverse = model.UndoAction{Type: model.UndoCreateSecret, Description: action.Description, Path: action.Path, Data: copyMap(action.Data), Keys: copySlice(action.Keys)}

		case model.UndoRenameSecret:
			// Undo rename: move from Path (current) back to OldPath (original).
			// Use MoveRecursive to handle both files and directories.
			if _, err := client.MoveRecursive(action.Path, action.OldPath); err != nil {
				return errorMsg("undo: " + err.Error())
			}
			reverse = model.UndoAction{Type: model.UndoRenameSecret, Description: action.Description, Path: action.OldPath, OldPath: action.Path}

		case model.UndoPasteSecret:
			// Undo paste = delete the pasted secret. Read data first for redo.
			secret, _ := client.Read(action.Path)
			if err := client.Delete(action.Path); err != nil {
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
				if _, err := client.MoveRecursive(action.Path, action.OldPath); err != nil {
					return errorMsg("undo: " + err.Error())
				}
				reverse = model.UndoAction{
					Type:        model.UndoCutPaste,
					Description: action.Description,
					Path:        action.OldPath,
					OldPath:     action.Path,
				}
			} else {
				if err := client.Write(action.OldPath, action.Data); err != nil {
					return errorMsg("undo: " + err.Error())
				}
				if err := client.Delete(action.Path); err != nil {
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
			current, _ := client.Read(action.Path)
			if err := client.Write(action.Path, action.Data); err != nil {
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

		reverse.Mount = client.Mount()
		return redoableStatusMsg{status: "Undo: " + action.Description, redo: reverse, reloadSecret: reloadSecret}
	}
}

func (m *Model) executeRedo(action model.UndoAction) tea.Cmd {
	client := m.actionClient(action)
	return func() tea.Msg {
		var reverse model.UndoAction
		var reloadSecret *model.Secret

		switch action.Type {
		case model.UndoCreateSecret:
			// Redo of undo-delete = delete again
			secret, _ := client.Read(action.Path)
			if err := client.Delete(action.Path); err != nil {
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
			if err := client.Write(action.Path, action.Data); err != nil {
				return errorMsg("redo: " + err.Error())
			}
			reverse = model.UndoAction{Type: model.UndoCreateSecret, Description: action.Description, Path: action.Path, Data: copyMap(action.Data), Keys: copySlice(action.Keys)}

		case model.UndoRenameSecret:
			// Redo rename: use MoveRecursive to handle both files and directories.
			if _, err := client.MoveRecursive(action.Path, action.OldPath); err != nil {
				return errorMsg("redo: " + err.Error())
			}
			reverse = model.UndoAction{Type: model.UndoRenameSecret, Description: action.Description, Path: action.OldPath, OldPath: action.Path}

		case model.UndoPasteSecret:
			// Redo paste = recreate the pasted secret
			if err := client.Write(action.Path, action.Data); err != nil {
				return errorMsg("redo: " + err.Error())
			}
			reverse = model.UndoAction{Type: model.UndoPasteSecret, Description: action.Description, Path: action.Path, Data: copyMap(action.Data), Keys: copySlice(action.Keys)}

		case model.UndoCutPaste:
			// Redo cut+paste: if Data is nil it was a directory move, use MoveRecursive.
			if action.Data == nil {
				if _, err := client.MoveRecursive(action.Path, action.OldPath); err != nil {
					return errorMsg("redo: " + err.Error())
				}
				reverse = model.UndoAction{
					Type:        model.UndoCutPaste,
					Description: action.Description,
					Path:        action.OldPath,
					OldPath:     action.Path,
				}
			} else {
				// After undo, Path is the cut source and OldPath the paste destination.
				if err := client.Write(action.OldPath, action.Data); err != nil {
					return errorMsg("redo: " + err.Error())
				}
				if err := client.Delete(action.Path); err != nil {
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
			current, _ := client.Read(action.Path)
			if err := client.Write(action.Path, action.Data); err != nil {
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

		reverse.Mount = client.Mount()
		return undoableStatusMsg{status: "Redo: " + action.Description, undo: reverse, reloadSecret: reloadSecret}
	}
}
