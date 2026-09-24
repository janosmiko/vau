package app

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/janosmiko/vau/internal/model"
)

// updateSizeProgress handles window resize and background-progress messages.
func (m *Model) updateSizeProgress(msg tea.Msg) (tea.Model, tea.Cmd, bool) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil, true

	case progressTickMsg:
		m.progress.current = msg.current
		if msg.total > 0 {
			m.progress.total = msg.total
		}
		return m, nil, true

	case progressDoneMsg:
		m.progress.active = false
		if m.progress.cancel != nil {
			m.progress.cancel()
		}
		m.progress.cancel = nil
		if msg.err != nil {
			if errors.Is(msg.err, context.Canceled) {
				m.status = fmt.Sprintf("Cancelled %s: %d items processed", msg.operation, msg.count)
			} else {
				m.errMsg = fmt.Sprintf("%s failed after %d items: %v", msg.operation, msg.count, msg.err)
			}
		} else {
			m.status = msg.status
			m.errMsg = ""
		}
		return m, m.refresh(), true
	}
	return m, nil, false
}

// updateCoreResult handles the top-level explorer async result messages.
func (m *Model) updateCoreResult(msg tea.Msg) (tea.Model, tea.Cmd, bool) {
	switch msg := msg.(type) {
	case listResultMsg:
		result, cmd := m.handleListResult(msg)
		return result, cmd, true

	case secretResultMsg:
		result, cmd := m.handleSecretResult(msg)
		return result, cmd, true

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
		return m, nil, true

	case jumpCompletionsMsg:
		if msg.input == m.jumpLastInput {
			m.jumpCompletions = msg.completions
			m.jumpCompIdx = -1
		}
		return m, nil, true
	}
	return m, nil, false
}

// updateWorkspaceMsg handles policy, auth-method, role, entity, and group async messages.
func (m *Model) updateWorkspaceMsg(msg tea.Msg) (tea.Model, tea.Cmd, bool) {
	switch msg := msg.(type) {
	case policyListMsg:
		result, cmd := m.handlePolicyListResult(msg)
		return result, cmd, true
	case policyContentMsg:
		return m.handlePolicyContentResult(msg), nil, true
	case policyPreviewMsg:
		m.policyPreview = msg.content
		return m, nil, true
	case authMethodsMsg:
		result, cmd := m.handleAuthMethodsResult(msg)
		return result, cmd, true
	case roleListMsg:
		result, cmd := m.handleRoleListResult(msg)
		return result, cmd, true
	case roleDataMsg:
		return m.handleRoleDataResult(msg), nil, true
	case rolePreviewMsg:
		m.rolePreview = msg.roles
		return m, nil, true
	case roleDataPreviewMsg:
		m.roleDataPreview = msg.data
		return m, nil, true

	case entityListMsg:
		result, cmd := m.handleEntityListResult(msg)
		return result, cmd, true
	case entityDataMsg:
		return m.handleEntityDataResult(msg), nil, true
	case entityDataPreviewMsg:
		m.entityDataPreview = msg.data
		return m, nil, true

	case groupListMsg:
		result, cmd := m.handleGroupListResult(msg)
		return result, cmd, true
	case groupDataMsg:
		return m.handleGroupDataResult(msg), nil, true
	case groupDataPreviewMsg:
		m.groupDataPreview = msg.data
		return m, nil, true
	}
	return m, nil, false
}

// updateEditMsg handles editor-launch and token-related async messages.
func (m *Model) updateEditMsg(msg tea.Msg) (tea.Model, tea.Cmd, bool) {
	switch msg := msg.(type) {
	case policyContentForEditMsg:
		return m, m.editPolicyInEditor(msg.name, msg.content), true
	case roleDataForEditMsg:
		return m, m.editRoleInEditor(msg.authPath, msg.name, msg.data), true

	case policyEditorResultMsg:
		result, cmd := m.handlePolicyEditorResult(msg)
		return result, cmd, true
	case policyDeletedMsg:
		result, cmd := m.handlePolicyDeletedResult(msg)
		return result, cmd, true
	case roleEditorResultMsg:
		result, cmd := m.handleRoleEditorResult(msg)
		return result, cmd, true
	case roleDeletedMsg:
		result, cmd := m.handleRoleDeletedResult(msg)
		return result, cmd, true

	case tokenListMsg:
		result, cmd := m.handleTokenListResult(msg)
		return result, cmd, true
	case tokenDataMsg:
		return m.handleTokenDataResult(msg), nil, true
	case tokenDataPreviewMsg:
		m.tokenDataPreview = msg.data
		return m, nil, true
	case tokenCreatedMsg:
		return m.handleTokenCreatedResult(msg), nil, true
	case tokenRevokedMsg:
		result, cmd := m.handleTokenRevokedResult(msg)
		return result, cmd, true
	}
	return m, nil, false
}

// updateStatusErrorMsg handles plain status/error/confirm/version messages.
func (m *Model) updateStatusErrorMsg(msg tea.Msg) (tea.Model, tea.Cmd, bool) {
	switch msg := msg.(type) {
	case statusMsg:
		m.status = string(msg)
		m.errMsg = ""
		if m.mode == model.ModeExplorer {
			return m, m.refresh(), true
		}
		return m, nil, true

	case errorMsg:
		m.errMsg = string(msg)
		m.status = ""
		return m, nil, true

	case confirmCreateMsg:
		path := string(msg)
		m.confirmMsg = fmt.Sprintf("Secret %q already exists. Overwrite?", path)
		m.prevConfirmMode = model.ModeExplorer
		m.confirmAction = func() tea.Cmd {
			return m.createEmptySecretAndOpen(path)
		}
		m.enterConfirmMode()
		return m, nil, true

	case confirmCreateEditorMsg:
		path := string(msg)
		m.confirmMsg = fmt.Sprintf("Secret %q already exists. Overwrite?", path)
		m.prevConfirmMode = model.ModeExplorer
		m.confirmAction = func() tea.Cmd {
			return m.openEditorForNewSecret(path)
		}
		m.enterConfirmMode()
		return m, nil, true

	case versionHistoryMsg:
		if msg.err != nil {
			m.errMsg = msg.err.Error()
			return m, nil, true
		}
		m.versionHistory = msg.versions
		m.versionCursor = 0
		m.mode = model.ModeVersionHistory
		return m, nil, true

	case versionDetailMsg:
		if msg.err != nil {
			m.errMsg = msg.err.Error()
			return m, nil, true
		}
		m.secret = msg.secret
		m.secretCursor = 0
		m.revealed = make(map[string]bool)
		m.secretBase64 = make(map[string]bool)
		m.secretAllRevealed = false
		m.secretJSONView = false
		m.mode = model.ModeSecret
		return m, nil, true
	}
	return m, nil, false
}

// updateYankUndoRedoMsg handles yank results and undo/redo status messages.
func (m *Model) updateYankUndoRedoMsg(msg tea.Msg) (tea.Model, tea.Cmd, bool) {
	switch msg := msg.(type) {
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
		return m, nil, true

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
			return m, m.refresh(), true
		}
		return m, nil, true

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
			return m, m.refresh(), true
		}
		return m, nil, true
	}
	return m, nil, false
}

// updateSecretEditMsg handles new-secret and editor-result messages.
func (m *Model) updateSecretEditMsg(msg tea.Msg) (tea.Model, tea.Cmd, bool) {
	switch msg := msg.(type) {
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
		return m, tea.Batch(textinput.Blink, m.refresh()), true

	case newSecretEditorMsg:
		path := string(msg)
		return m, m.openEditorForNewSecret(path), true

	case editorResultMsg:
		// New secret creation via editor
		if msg.newSecretPath != "" {
			path := msg.newSecretPath
			client := m.client
			return m, func() tea.Msg {
				if err := client.Write(path, msg.data); err != nil {
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
			}, true
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

			return m, m.saveSecretFromEditor(secretPath, snapData, snapKeys), true
		}
		return m, nil, true
	}
	return m, nil, false
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	for _, handler := range []func(tea.Msg) (tea.Model, tea.Cmd, bool){
		m.updateSizeProgress,
		m.updateCoreResult,
		m.updateWorkspaceMsg,
		m.updateEditMsg,
		m.updateStatusErrorMsg,
		m.updateYankUndoRedoMsg,
		m.updateSecretEditMsg,
	} {
		if result, cmd, handled := handler(msg); handled {
			return result, cmd
		}
	}

	switch msg := msg.(type) {
	case tea.MouseMsg:
		return m.handleMouse(msg)
	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

// --- Key handling ---

// handleWorkspaceRoutedKey routes to the policy/role/entity/group/token mode handlers.
func (m *Model) handleWorkspaceRoutedKey(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	switch m.mode {
	case model.ModePolicyList:
		result, cmd := m.handlePolicyListKey(msg)
		return result, cmd, true
	case model.ModePolicyView:
		result, cmd := m.handlePolicyViewKey(msg)
		return result, cmd, true
	case model.ModeAuthMethods:
		result, cmd := m.handleAuthMethodsKey(msg)
		return result, cmd, true
	case model.ModeRoleList:
		result, cmd := m.handleRoleListKey(msg)
		return result, cmd, true
	case model.ModeRoleView:
		result, cmd := m.handleRoleViewKey(msg)
		return result, cmd, true
	case model.ModeEntityList:
		result, cmd := m.handleEntityListKey(msg)
		return result, cmd, true
	case model.ModeEntityView:
		return m.handleEntityViewKey(msg), nil, true
	case model.ModeGroupList:
		result, cmd := m.handleGroupListKey(msg)
		return result, cmd, true
	case model.ModeGroupView:
		return m.handleGroupViewKey(msg), nil, true
	case model.ModeTokenList:
		result, cmd := m.handleTokenListKey(msg)
		return result, cmd, true
	case model.ModeTokenView:
		return m.handleTokenViewKey(msg), nil, true
	case model.ModeTokenCreated:
		// Any key dismisses the token created overlay
		m.mode = model.ModeTokenList
		m.tokenCreatedData = nil
		return m, nil, true
	}
	return m, nil, false
}

// handleOverlayRoutedKey routes to the input/search/filter/help/confirm/bookmark overlay handlers.
func (m *Model) handleOverlayRoutedKey(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	switch m.mode {
	case model.ModeInput:
		result, cmd := m.handleInputKey(msg)
		return result, cmd, true
	case model.ModeSearch:
		result, cmd := m.handleSearchKey(msg)
		return result, cmd, true
	case model.ModeFilter:
		result, cmd := m.handleFilterKey(msg)
		return result, cmd, true
	case model.ModeJumpPath:
		result, cmd := m.handleJumpPathKey(msg)
		return result, cmd, true
	case model.ModeVersionHistory:
		result, cmd := m.handleVersionHistoryKey(msg)
		return result, cmd, true
	case model.ModeDockerConfig:
		result, cmd := m.handleDockerConfigKey(msg)
		return result, cmd, true
	case model.ModeHelp:
		result, cmd := m.handleHelpKey(msg)
		return result, cmd, true
	case model.ModeSecretEdit:
		result, cmd := m.handleSecretEditKey(msg)
		return result, cmd, true
	case model.ModeConfirm:
		result, cmd := m.handleConfirmKey(msg)
		return result, cmd, true
	case model.ModeBookmark:
		result, cmd := m.handleBookmarkOverlayKey(msg)
		return result, cmd, true
	case model.ModeThemePicker:
		return m.handleThemePickerKey(msg), nil, true
	}
	return m, nil, false
}

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// During active progress, only allow cancellation
	if m.progress.active {
		key := msg.String()
		if key == "ctrl+c" || key == "esc" {
			if m.progress.cancel != nil {
				m.progress.cancel()
			}
			m.status = fmt.Sprintf("Cancelling %s...", m.progress.operation)
			return m, nil
		}
		// Swallow all other keys during progress
		return m, nil
	}

	if result, cmd, handled := m.handleWorkspaceRoutedKey(msg); handled {
		return result, cmd
	}
	if result, cmd, handled := m.handleOverlayRoutedKey(msg); handled {
		return result, cmd
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
	// Swallow mouse events during progress
	if m.progress.active {
		return m, nil
	}
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
	// only handle scroll wheel as a no-op. Keyboard drives them.
	return m, nil
}
