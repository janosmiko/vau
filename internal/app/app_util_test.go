package app

import (
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/janosmiko/vau/internal/config"
	"github.com/janosmiko/vau/internal/model"
	"github.com/janosmiko/vau/internal/ui"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// View
// ---------------------------------------------------------------------------

func TestView_LoadingWhenWidthZero(t *testing.T) {
	m := &Model{width: 0}
	assert.Equal(t, "Loading...", m.View())
}

// ---------------------------------------------------------------------------
// startInput
// ---------------------------------------------------------------------------

func TestStartInput(t *testing.T) {
	ti := textinput.New()
	m := &Model{
		mode:      model.ModeExplorer,
		textInput: ti,
	}
	cmd := m.startInput(model.InputRename, "New name:", "old-name")

	assert.Equal(t, model.ModeInput, m.mode)
	assert.Equal(t, model.InputRename, m.inputAction)
	assert.Equal(t, "New name:", m.inputLabel)
	assert.Equal(t, "old-name", m.textInput.Value())
	assert.NotNil(t, cmd, "startInput should return a blink cmd")
}

func TestStartInput_NewKey(t *testing.T) {
	ti := textinput.New()
	m := &Model{
		mode:      model.ModeSecret,
		textInput: ti,
	}
	cmd := m.startInput(model.InputNewKey, "Key name:", "")

	assert.Equal(t, model.ModeInput, m.mode)
	assert.Equal(t, model.InputNewKey, m.inputAction)
	assert.Equal(t, "Key name:", m.inputLabel)
	assert.Equal(t, "", m.textInput.Value())
	assert.NotNil(t, cmd)
}

// ---------------------------------------------------------------------------
// handleInputKey - escape
// ---------------------------------------------------------------------------

func TestHandleInputKey_Esc_WithSecret(t *testing.T) {
	ti := textinput.New()
	ti.Focus()
	m := &Model{
		mode:          model.ModeInput,
		prevInputMode: model.ModeSecret,
		textInput:     ti,
		inputAction:   model.InputEditValue,
		inputBuffer:   "leftover",
		secret:        &model.Secret{Path: "secret/foo"},
	}

	result, cmd := m.handleInputKey(tea.KeyMsg{Type: tea.KeyEsc})
	resultModel := result.(*Model)

	assert.Equal(t, model.ModeSecret, resultModel.mode)
	assert.Equal(t, model.InputNone, resultModel.inputAction)
	assert.Empty(t, resultModel.inputBuffer)
	assert.Nil(t, cmd)
}

func TestHandleInputKey_Esc_WithoutSecret(t *testing.T) {
	ti := textinput.New()
	ti.Focus()
	m := &Model{
		mode:        model.ModeInput,
		textInput:   ti,
		inputAction: model.InputRename,
		inputBuffer: "leftover",
		secret:      nil,
	}

	result, cmd := m.handleInputKey(tea.KeyMsg{Type: tea.KeyEsc})
	resultModel := result.(*Model)

	assert.Equal(t, model.ModeExplorer, resultModel.mode)
	assert.Equal(t, model.InputNone, resultModel.inputAction)
	assert.Empty(t, resultModel.inputBuffer)
	assert.Nil(t, cmd)
}

// ---------------------------------------------------------------------------
// handleConfirmKey
// ---------------------------------------------------------------------------

func TestHandleConfirmKey_Esc(t *testing.T) {
	ci := textinput.New()
	ci.Focus()
	m := &Model{
		mode:            model.ModeConfirm,
		prevConfirmMode: model.ModeSecret,
		confirmMsg:      "Delete?",
		confirmAction:   func() tea.Cmd { return nil },
		confirmInput:    ci,
	}

	result, cmd := m.handleConfirmKey(tea.KeyMsg{Type: tea.KeyEsc})
	resultModel := result.(*Model)

	assert.Equal(t, model.ModeSecret, resultModel.mode)
	assert.Nil(t, resultModel.confirmAction)
	assert.Empty(t, resultModel.confirmMsg)
	assert.Nil(t, cmd)
}

func TestHandleConfirmKey_Enter_WrongInput(t *testing.T) {
	ci := textinput.New()
	ci.SetValue("wrong")
	ci.Focus()
	m := &Model{
		mode:            model.ModeConfirm,
		prevConfirmMode: model.ModeExplorer,
		confirmMsg:      "Delete?",
		confirmInput:    ci,
	}

	result, cmd := m.handleConfirmKey(tea.KeyMsg{Type: tea.KeyEnter})
	resultModel := result.(*Model)

	assert.Equal(t, "Type DELETE to confirm", resultModel.errMsg)
	assert.Nil(t, cmd)
}

func TestHandleConfirmKey_Enter_CorrectInput(t *testing.T) {
	ci := textinput.New()
	ci.SetValue("DELETE")
	ci.Focus()

	actionCalled := false
	m := &Model{
		mode:            model.ModeConfirm,
		prevConfirmMode: model.ModeExplorer,
		confirmMsg:      "Delete?",
		confirmAction: func() tea.Cmd {
			actionCalled = true
			return func() tea.Msg { return statusMsg("done") }
		},
		confirmInput: ci,
	}

	result, cmd := m.handleConfirmKey(tea.KeyMsg{Type: tea.KeyEnter})
	resultModel := result.(*Model)

	assert.Equal(t, model.ModeExplorer, resultModel.mode)
	assert.Nil(t, resultModel.confirmAction)
	assert.Empty(t, resultModel.confirmMsg)
	assert.True(t, actionCalled)
	assert.NotNil(t, cmd)
}

func TestHandleConfirmKey_Enter_NilAction(t *testing.T) {
	ci := textinput.New()
	ci.SetValue("DELETE")
	ci.Focus()

	m := &Model{
		mode:            model.ModeConfirm,
		prevConfirmMode: model.ModeSecret,
		confirmMsg:      "Delete?",
		confirmAction:   nil,
		confirmInput:    ci,
	}

	result, cmd := m.handleConfirmKey(tea.KeyMsg{Type: tea.KeyEnter})
	resultModel := result.(*Model)

	assert.Equal(t, model.ModeSecret, resultModel.mode)
	assert.Nil(t, cmd)
}

// ---------------------------------------------------------------------------
// switchEditColumn
// ---------------------------------------------------------------------------

func TestSwitchEditColumn_FromKeyToValue(t *testing.T) {
	ti := textinput.New()
	ti.SetValue("current-key")

	m := &Model{
		textInput:        ti,
		secretEditKey:    "mykey",
		secretEditColumn: 0,
		secret: &model.Secret{
			Keys: []string{"mykey"},
			Data: map[string]string{"mykey": "the-value"},
		},
	}

	result, cmd := m.switchEditColumn()
	resultModel := result.(*Model)

	assert.Equal(t, 1, resultModel.secretEditColumn)
	assert.Equal(t, "the-value", resultModel.textInput.Value())
	assert.NotNil(t, cmd)
}

func TestSwitchEditColumn_FromValueToKey(t *testing.T) {
	ti := textinput.New()
	ti.SetValue("edited-value")

	m := &Model{
		textInput:        ti,
		secretEditKey:    "mykey",
		secretEditColumn: 1,
		secret: &model.Secret{
			Keys: []string{"mykey"},
			Data: map[string]string{"mykey": "original"},
		},
	}

	result, cmd := m.switchEditColumn()
	resultModel := result.(*Model)

	assert.Equal(t, 0, resultModel.secretEditColumn)
	assert.Equal(t, "mykey", resultModel.textInput.Value())
	// The old value should have been saved by applyCurrentEditInput
	assert.Equal(t, "edited-value", resultModel.secret.Data["mykey"])
	assert.NotNil(t, cmd)
}

// ---------------------------------------------------------------------------
// handleKey - mode dispatcher
// ---------------------------------------------------------------------------

func TestHandleKey_Dispatches_ByMode(t *testing.T) {
	tests := []struct {
		name     string
		mode     model.ViewMode
		key      tea.KeyMsg
		wantMode model.ViewMode
	}{
		{
			name:     "confirm mode esc returns to prevConfirmMode",
			mode:     model.ModeConfirm,
			key:      tea.KeyMsg{Type: tea.KeyEsc},
			wantMode: model.ModeExplorer,
		},
		{
			name:     "input mode esc returns to explorer (no secret)",
			mode:     model.ModeInput,
			key:      tea.KeyMsg{Type: tea.KeyEsc},
			wantMode: model.ModeExplorer,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			si := textinput.New()
			ti := textinput.New()
			ci := textinput.New()
			m := &Model{
				mode:            tc.mode,
				prevConfirmMode: model.ModeExplorer,
				searchInput:     si,
				textInput:       ti,
				confirmInput:    ci,
				keys:            DefaultKeyMap(),
			}
			result, _ := m.handleKey(tc.key)
			resultModel := result.(*Model)
			assert.Equal(t, tc.wantMode, resultModel.mode)
		})
	}
}

func TestHandleKey_UnknownMode_ReturnsModelUnchanged(t *testing.T) {
	si := textinput.New()
	ti := textinput.New()
	ci := textinput.New()
	m := &Model{
		mode:         42, // invalid mode
		searchInput:  si,
		textInput:    ti,
		confirmInput: ci,
		keys:         DefaultKeyMap(),
	}
	result, cmd := m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	assert.Equal(t, m, result)
	assert.Nil(t, cmd)
}

// ---------------------------------------------------------------------------
// handleMouse - mode dispatcher
// ---------------------------------------------------------------------------

func TestHandleMouse_NoneButton_NoOp(t *testing.T) {
	m := &Model{mode: model.ModeExplorer}
	result, cmd := m.handleMouse(tea.MouseMsg{Button: tea.MouseButtonNone})
	assert.Equal(t, m, result)
	assert.Nil(t, cmd)
}

func TestHandleMouse_OverlayMode_NoOp(t *testing.T) {
	// For modes not in the switch (like ModeInput, ModeFilter, etc.)
	overlayModes := []model.ViewMode{
		model.ModeInput,
		model.ModeSearch,
		model.ModeFilter,
		model.ModeConfirm,
		model.ModeJumpPath,
	}

	for _, mode := range overlayModes {
		m := &Model{mode: mode}
		result, cmd := m.handleMouse(tea.MouseMsg{Button: tea.MouseButtonWheelUp})
		assert.Equal(t, m, result, "mode %d should return model unchanged", mode)
		assert.Nil(t, cmd, "mode %d should return nil cmd", mode)
	}
}

// ---------------------------------------------------------------------------
// handleVersionHistoryKey
// ---------------------------------------------------------------------------

func TestHandleVersionHistoryKey_Esc(t *testing.T) {
	m := &Model{
		mode: model.ModeVersionHistory,
		versionHistory: []model.SecretVersion{
			{Version: "1"},
			{Version: "2"},
		},
		versionCursor: 1,
	}
	result, cmd := m.handleVersionHistoryKey(tea.KeyMsg{Type: tea.KeyEsc})
	resultModel := result.(*Model)

	assert.Equal(t, model.ModeSecret, resultModel.mode)
	assert.Nil(t, resultModel.versionHistory)
	assert.Nil(t, cmd)
}

func TestHandleVersionHistoryKey_Q(t *testing.T) {
	m := &Model{
		mode:           model.ModeVersionHistory,
		versionHistory: []model.SecretVersion{{Version: "1"}},
	}
	result, _ := m.handleVersionHistoryKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	resultModel := result.(*Model)

	assert.Equal(t, model.ModeSecret, resultModel.mode)
}

func TestHandleVersionHistoryKey_CursorDown(t *testing.T) {
	m := &Model{
		mode: model.ModeVersionHistory,
		versionHistory: []model.SecretVersion{
			{Version: "1"},
			{Version: "2"},
			{Version: "3"},
		},
		versionCursor: 0,
	}
	m.handleVersionHistoryKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	assert.Equal(t, 1, m.versionCursor)
}

func TestHandleVersionHistoryKey_CursorDown_AtEnd(t *testing.T) {
	m := &Model{
		mode: model.ModeVersionHistory,
		versionHistory: []model.SecretVersion{
			{Version: "1"},
			{Version: "2"},
		},
		versionCursor: 1,
	}
	m.handleVersionHistoryKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	assert.Equal(t, 1, m.versionCursor, "should not go beyond last entry")
}

func TestHandleVersionHistoryKey_CursorUp(t *testing.T) {
	m := &Model{
		mode: model.ModeVersionHistory,
		versionHistory: []model.SecretVersion{
			{Version: "1"},
			{Version: "2"},
		},
		versionCursor: 1,
	}
	m.handleVersionHistoryKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	assert.Equal(t, 0, m.versionCursor)
}

func TestHandleVersionHistoryKey_CursorUp_AtTop(t *testing.T) {
	m := &Model{
		mode: model.ModeVersionHistory,
		versionHistory: []model.SecretVersion{
			{Version: "1"},
		},
		versionCursor: 0,
	}
	m.handleVersionHistoryKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	assert.Equal(t, 0, m.versionCursor, "should not go below 0")
}

// ---------------------------------------------------------------------------
// handleVersionHistoryMouse
// ---------------------------------------------------------------------------

func TestHandleVersionHistoryMouse_ScrollUp(t *testing.T) {
	m := &Model{
		mode: model.ModeVersionHistory,
		versionHistory: []model.SecretVersion{
			{Version: "1"},
			{Version: "2"},
		},
		versionCursor: 1,
	}
	m.handleVersionHistoryMouse(tea.MouseMsg{Button: tea.MouseButtonWheelUp})
	assert.Equal(t, 0, m.versionCursor)
}

func TestHandleVersionHistoryMouse_ScrollDown(t *testing.T) {
	m := &Model{
		mode: model.ModeVersionHistory,
		versionHistory: []model.SecretVersion{
			{Version: "1"},
			{Version: "2"},
		},
		versionCursor: 0,
	}
	m.handleVersionHistoryMouse(tea.MouseMsg{Button: tea.MouseButtonWheelDown})
	assert.Equal(t, 1, m.versionCursor)
}

func TestHandleVersionHistoryMouse_ScrollDown_AtEnd(t *testing.T) {
	m := &Model{
		mode: model.ModeVersionHistory,
		versionHistory: []model.SecretVersion{
			{Version: "1"},
		},
		versionCursor: 0,
	}
	m.handleVersionHistoryMouse(tea.MouseMsg{Button: tea.MouseButtonWheelDown})
	assert.Equal(t, 0, m.versionCursor, "should not go beyond last entry")
}

func TestHandleVersionHistoryMouse_ScrollUp_AtTop(t *testing.T) {
	m := &Model{
		mode:           model.ModeVersionHistory,
		versionHistory: []model.SecretVersion{{Version: "1"}},
		versionCursor:  0,
	}
	m.handleVersionHistoryMouse(tea.MouseMsg{Button: tea.MouseButtonWheelUp})
	assert.Equal(t, 0, m.versionCursor, "should not go below 0")
}

func TestHandleVersionHistoryMouse_OtherButton(t *testing.T) {
	m := &Model{
		mode:           model.ModeVersionHistory,
		versionHistory: []model.SecretVersion{{Version: "1"}},
		versionCursor:  0,
	}
	result, cmd := m.handleVersionHistoryMouse(tea.MouseMsg{Button: tea.MouseButtonLeft})
	assert.Equal(t, m, result)
	assert.Nil(t, cmd)
}

// ---------------------------------------------------------------------------
// handleBookmarkMouse
// ---------------------------------------------------------------------------

func TestHandleBookmarkMouse_ScrollUp(t *testing.T) {
	m := &Model{
		mode: model.ModeBookmark,
		bookmarks: []config.Bookmark{
			{Name: "a", Mount: "kv", Path: "a"},
			{Name: "b", Mount: "kv", Path: "b"},
		},
		bookmarkCursor: 1,
	}
	m.handleBookmarkMouse(tea.MouseMsg{Button: tea.MouseButtonWheelUp})
	assert.Equal(t, 0, m.bookmarkCursor)
}

func TestHandleBookmarkMouse_ScrollDown(t *testing.T) {
	m := &Model{
		mode: model.ModeBookmark,
		bookmarks: []config.Bookmark{
			{Name: "a", Mount: "kv", Path: "a"},
			{Name: "b", Mount: "kv", Path: "b"},
		},
		bookmarkCursor: 0,
	}
	m.handleBookmarkMouse(tea.MouseMsg{Button: tea.MouseButtonWheelDown})
	assert.Equal(t, 1, m.bookmarkCursor)
}

func TestHandleBookmarkMouse_ScrollDown_AtEnd(t *testing.T) {
	m := &Model{
		mode: model.ModeBookmark,
		bookmarks: []config.Bookmark{
			{Name: "a", Mount: "kv", Path: "a"},
		},
		bookmarkCursor: 0,
	}
	m.handleBookmarkMouse(tea.MouseMsg{Button: tea.MouseButtonWheelDown})
	assert.Equal(t, 0, m.bookmarkCursor, "should not go beyond last entry")
}

func TestHandleBookmarkMouse_ScrollUp_AtTop(t *testing.T) {
	m := &Model{
		mode:           model.ModeBookmark,
		bookmarks:      []config.Bookmark{{Name: "a", Mount: "kv", Path: "a"}},
		bookmarkCursor: 0,
	}
	m.handleBookmarkMouse(tea.MouseMsg{Button: tea.MouseButtonWheelUp})
	assert.Equal(t, 0, m.bookmarkCursor, "should not go below 0")
}

func TestHandleBookmarkMouse_OtherButton(t *testing.T) {
	m := &Model{
		mode:      model.ModeBookmark,
		bookmarks: []config.Bookmark{{Name: "a", Mount: "kv", Path: "a"}},
	}
	result, cmd := m.handleBookmarkMouse(tea.MouseMsg{Button: tea.MouseButtonLeft})
	assert.Equal(t, m, result)
	assert.Nil(t, cmd)
}

// ---------------------------------------------------------------------------
// handleThemePickerMouse
// ---------------------------------------------------------------------------

func TestHandleThemePickerMouse_ScrollDown(t *testing.T) {
	entries := []ui.ThemeEntry{
		{Name: "Dark Themes", IsHeader: true},
		{Name: "dracula", IsHeader: false},
		{Name: "monokai", IsHeader: false},
	}

	m := &Model{
		mode:              model.ModeThemePicker,
		themeEntries:      entries,
		themeCursor:       1,
		config:            &config.Config{},
		activeColorscheme: "dracula",
	}
	m.handleThemePickerMouse(tea.MouseMsg{Button: tea.MouseButtonWheelDown})
	assert.Equal(t, 2, m.themeCursor)
}

func TestHandleThemePickerMouse_ScrollUp(t *testing.T) {
	entries := []ui.ThemeEntry{
		{Name: "Dark Themes", IsHeader: true},
		{Name: "dracula", IsHeader: false},
		{Name: "monokai", IsHeader: false},
	}

	m := &Model{
		mode:              model.ModeThemePicker,
		themeEntries:      entries,
		themeCursor:       2,
		config:            &config.Config{},
		activeColorscheme: "monokai",
	}
	m.handleThemePickerMouse(tea.MouseMsg{Button: tea.MouseButtonWheelUp})
	assert.Equal(t, 1, m.themeCursor)
}

func TestHandleThemePickerMouse_OtherButton(t *testing.T) {
	m := &Model{
		mode:         model.ModeThemePicker,
		themeEntries: []ui.ThemeEntry{{Name: "a", IsHeader: false}},
	}
	result, cmd := m.handleThemePickerMouse(tea.MouseMsg{Button: tea.MouseButtonLeft})
	assert.Equal(t, m, result)
	assert.Nil(t, cmd)
}

// ---------------------------------------------------------------------------
// handleSecretMouse
// ---------------------------------------------------------------------------

func TestHandleSecretMouse_ScrollUp(t *testing.T) {
	m := &Model{
		mode: model.ModeSecret,
		secret: &model.Secret{
			Keys: []string{"a", "b", "c"},
			Data: map[string]string{"a": "1", "b": "2", "c": "3"},
		},
		secretCursor: 2,
	}
	m.handleSecretMouse(tea.MouseMsg{Button: tea.MouseButtonWheelUp})
	assert.Equal(t, 1, m.secretCursor)
}

func TestHandleSecretMouse_ScrollDown(t *testing.T) {
	m := &Model{
		mode: model.ModeSecret,
		secret: &model.Secret{
			Keys: []string{"a", "b", "c"},
			Data: map[string]string{"a": "1", "b": "2", "c": "3"},
		},
		secretCursor: 1,
	}
	m.handleSecretMouse(tea.MouseMsg{Button: tea.MouseButtonWheelDown})
	assert.Equal(t, 2, m.secretCursor)
}

func TestHandleSecretMouse_ScrollDown_AtEnd(t *testing.T) {
	m := &Model{
		mode: model.ModeSecret,
		secret: &model.Secret{
			Keys: []string{"a", "b"},
			Data: map[string]string{"a": "1", "b": "2"},
		},
		secretCursor: 1,
	}
	m.handleSecretMouse(tea.MouseMsg{Button: tea.MouseButtonWheelDown})
	assert.Equal(t, 1, m.secretCursor, "should not go beyond last key")
}

func TestHandleSecretMouse_ScrollUp_AtTop(t *testing.T) {
	m := &Model{
		mode: model.ModeSecret,
		secret: &model.Secret{
			Keys: []string{"a"},
			Data: map[string]string{"a": "1"},
		},
		secretCursor: 0,
	}
	m.handleSecretMouse(tea.MouseMsg{Button: tea.MouseButtonWheelUp})
	assert.Equal(t, 0, m.secretCursor, "should not go below 0")
}

func TestHandleSecretMouse_ScrollDown_NilSecret(t *testing.T) {
	m := &Model{
		mode:         model.ModeSecret,
		secret:       nil,
		secretCursor: 0,
	}
	result, cmd := m.handleSecretMouse(tea.MouseMsg{Button: tea.MouseButtonWheelDown})
	assert.Equal(t, 0, result.(*Model).secretCursor)
	assert.Nil(t, cmd)
}

// ---------------------------------------------------------------------------
// handleHelpMouse
// ---------------------------------------------------------------------------

func TestHandleHelpMouse_ScrollDown(t *testing.T) {
	m := &Model{
		mode:       model.ModeHelp,
		height:     100,
		width:      100,
		helpScroll: 0,
	}
	m.handleHelpMouse(tea.MouseMsg{Button: tea.MouseButtonWheelDown})
	assert.Equal(t, 1, m.helpScroll)
}

func TestHandleHelpMouse_ScrollUp(t *testing.T) {
	m := &Model{
		mode:       model.ModeHelp,
		height:     100,
		width:      100,
		helpScroll: 5,
	}
	m.handleHelpMouse(tea.MouseMsg{Button: tea.MouseButtonWheelUp})
	assert.Equal(t, 4, m.helpScroll)
}

func TestHandleHelpMouse_ScrollUp_AtTop(t *testing.T) {
	m := &Model{
		mode:       model.ModeHelp,
		height:     100,
		width:      100,
		helpScroll: 0,
	}
	m.handleHelpMouse(tea.MouseMsg{Button: tea.MouseButtonWheelUp})
	assert.Equal(t, 0, m.helpScroll)
}

func TestHandleHelpMouse_OtherButton(t *testing.T) {
	m := &Model{
		mode:       model.ModeHelp,
		height:     100,
		width:      100,
		helpScroll: 3,
	}
	result, cmd := m.handleHelpMouse(tea.MouseMsg{Button: tea.MouseButtonMiddle})
	assert.Equal(t, 3, result.(*Model).helpScroll)
	assert.Nil(t, cmd)
}

func TestHandleHelpMouse_ClickOutside_ClosesHelp(t *testing.T) {
	m := &Model{
		mode:   model.ModeHelp,
		height: 100,
		width:  100,
	}
	// Click at (0, 0) which is outside the centered help overlay
	result, cmd := m.handleHelpMouse(tea.MouseMsg{
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionPress,
		X:      0,
		Y:      0,
	})
	assert.Equal(t, model.ModeExplorer, result.(*Model).mode)
	assert.Nil(t, cmd)
}

func TestHandleHelpMouse_ClickInside_NoClose(t *testing.T) {
	m := &Model{
		mode:   model.ModeHelp,
		height: 100,
		width:  100,
	}
	// Click at center of screen which is inside the help overlay
	result, _ := m.handleHelpMouse(tea.MouseMsg{
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionPress,
		X:      50,
		Y:      50,
	})
	assert.Equal(t, model.ModeHelp, result.(*Model).mode)
}

// ---------------------------------------------------------------------------
// handleHelpKey
// ---------------------------------------------------------------------------

func TestHandleHelpKey_Esc(t *testing.T) {
	m := newTestModel()
	m.mode = model.ModeHelp
	m.height = 100

	result, _ := m.handleHelpKey(tea.KeyMsg{Type: tea.KeyEsc})
	assert.Equal(t, model.ModeExplorer, result.(*Model).mode)
}

func TestHandleHelpKey_Q(t *testing.T) {
	m := newTestModel()
	m.mode = model.ModeHelp
	m.height = 100

	result, _ := m.handleHelpKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	assert.Equal(t, model.ModeExplorer, result.(*Model).mode)
}

func TestHandleHelpKey_QuestionMark(t *testing.T) {
	m := newTestModel()
	m.mode = model.ModeHelp
	m.height = 100

	result, _ := m.handleHelpKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	assert.Equal(t, model.ModeExplorer, result.(*Model).mode)
}

func TestHandleHelpKey_ScrollDown(t *testing.T) {
	m := newTestModel()
	m.mode = model.ModeHelp
	m.height = 100
	m.helpScroll = 0

	m.handleHelpKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	assert.Equal(t, 1, m.helpScroll)
}

func TestHandleHelpKey_ScrollUp(t *testing.T) {
	m := newTestModel()
	m.mode = model.ModeHelp
	m.height = 100
	m.helpScroll = 5

	m.handleHelpKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	assert.Equal(t, 4, m.helpScroll)
}

func TestHandleHelpKey_ScrollUp_AtTop(t *testing.T) {
	m := newTestModel()
	m.mode = model.ModeHelp
	m.height = 100
	m.helpScroll = 0

	m.handleHelpKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	assert.Equal(t, 0, m.helpScroll, "should not go below 0")
}

func TestHandleHelpKey_GoToTop(t *testing.T) {
	m := newTestModel()
	m.mode = model.ModeHelp
	m.height = 100
	m.helpScroll = 10

	m.handleHelpKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	assert.Equal(t, 0, m.helpScroll)
}

func TestHandleHelpKey_GoToBottom(t *testing.T) {
	m := newTestModel()
	m.mode = model.ModeHelp
	m.height = 100
	m.helpScroll = 0

	m.handleHelpKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	assert.GreaterOrEqual(t, m.helpScroll, 0)
}

func TestHandleHelpKey_SlashStartsSearch(t *testing.T) {
	m := newTestModel()
	m.mode = model.ModeHelp
	m.height = 100

	_, cmd := m.handleHelpKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	assert.True(t, m.helpSearching)
	assert.NotNil(t, cmd, "should return blink cmd")
}

func TestHandleHelpKey_SearchMode_Enter(t *testing.T) {
	m := newTestModel()
	m.mode = model.ModeHelp
	m.height = 100
	m.helpSearching = true
	m.searchInput.SetValue("test-filter")

	m.handleHelpKey(tea.KeyMsg{Type: tea.KeyEnter})
	assert.False(t, m.helpSearching)
	assert.Equal(t, "test-filter", m.helpFilter)
	assert.Equal(t, 0, m.helpScroll)
}

func TestHandleHelpKey_SearchMode_Esc(t *testing.T) {
	m := newTestModel()
	m.mode = model.ModeHelp
	m.height = 100
	m.helpSearching = true
	m.helpFilter = "old-filter"

	m.handleHelpKey(tea.KeyMsg{Type: tea.KeyEsc})
	assert.False(t, m.helpSearching)
	assert.Empty(t, m.helpFilter)
	assert.Equal(t, 0, m.helpScroll)
}

func TestHandleHelpKey_HalfPageDown(t *testing.T) {
	m := newTestModel()
	m.mode = model.ModeHelp
	m.height = 100
	m.helpScroll = 0

	m.handleHelpKey(tea.KeyMsg{Type: tea.KeyCtrlD})
	assert.Greater(t, m.helpScroll, 0)
}

func TestHandleHelpKey_HalfPageUp(t *testing.T) {
	m := newTestModel()
	m.mode = model.ModeHelp
	m.height = 100
	m.helpScroll = 20

	m.handleHelpKey(tea.KeyMsg{Type: tea.KeyCtrlU})
	assert.Less(t, m.helpScroll, 20)
}

func TestHandleHelpKey_FullPageDown(t *testing.T) {
	m := newTestModel()
	m.mode = model.ModeHelp
	m.height = 100
	m.helpScroll = 0

	m.handleHelpKey(tea.KeyMsg{Type: tea.KeyCtrlF})
	assert.Greater(t, m.helpScroll, 0)
}

func TestHandleHelpKey_FullPageUp(t *testing.T) {
	m := newTestModel()
	m.mode = model.ModeHelp
	m.height = 100
	m.helpScroll = 50

	m.handleHelpKey(tea.KeyMsg{Type: tea.KeyCtrlB})
	assert.Less(t, m.helpScroll, 50)
}

// ---------------------------------------------------------------------------
// Update - message handling
// ---------------------------------------------------------------------------

func TestUpdate_WindowSizeMsg(t *testing.T) {
	m := newTestModel()
	msg := tea.WindowSizeMsg{Width: 120, Height: 40}

	result, cmd := m.Update(msg)
	resultModel := result.(*Model)

	assert.Equal(t, 120, resultModel.width)
	assert.Equal(t, 40, resultModel.height)
	assert.Nil(t, cmd)
}

func TestUpdate_ErrorMsg(t *testing.T) {
	m := newTestModel()
	m.status = "some status"

	result, cmd := m.Update(errorMsg("something went wrong"))
	resultModel := result.(*Model)

	assert.Equal(t, "something went wrong", resultModel.errMsg)
	assert.Empty(t, resultModel.status)
	assert.Nil(t, cmd)
}

func TestUpdate_MountsResultMsg_Success(t *testing.T) {
	m := newTestModel()
	msg := mountsResultMsg{
		mounts: []string{"kv", "secret", "aws"},
		err:    nil,
	}

	result, cmd := m.Update(msg)
	resultModel := result.(*Model)

	assert.Equal(t, []string{"kv", "secret", "aws"}, resultModel.mounts)
	assert.Equal(t, 1, resultModel.mountCursor, "should select current mount 'secret'")
	assert.Nil(t, cmd)
}

func TestUpdate_MountsResultMsg_Error(t *testing.T) {
	m := newTestModel()
	msg := mountsResultMsg{
		mounts: nil,
		err:    assert.AnError,
	}

	result, cmd := m.Update(msg)
	resultModel := result.(*Model)

	assert.Contains(t, resultModel.errMsg, "assert.AnError")
	assert.Nil(t, cmd)
}

func TestUpdate_YankResultMsg_Single(t *testing.T) {
	m := newTestModel()
	msg := yankResultMsg{
		secrets: []*model.Secret{{Path: "secret/foo"}},
		paths:   []string{"secret/foo"},
		isCut:   false,
		isDir:   false,
	}

	result, _ := m.Update(msg)
	resultModel := result.(*Model)

	assert.Equal(t, "Yanked: secret/foo", resultModel.status)
	assert.Len(t, resultModel.yankedSecrets, 1)
	assert.False(t, resultModel.yankIsCut)
}

func TestUpdate_YankResultMsg_SingleCut(t *testing.T) {
	m := newTestModel()
	msg := yankResultMsg{
		secrets: []*model.Secret{{Path: "secret/foo"}},
		paths:   []string{"secret/foo"},
		isCut:   true,
		isDir:   false,
	}

	result, _ := m.Update(msg)
	resultModel := result.(*Model)

	assert.Equal(t, "Cut (yanked for move): secret/foo", resultModel.status)
	assert.True(t, resultModel.yankIsCut)
}

func TestUpdate_YankResultMsg_DirYank(t *testing.T) {
	m := newTestModel()
	msg := yankResultMsg{
		secrets: []*model.Secret{},
		paths:   []string{"dir1/", "dir2/"},
		isCut:   false,
		isDir:   true,
	}

	result, _ := m.Update(msg)
	resultModel := result.(*Model)

	assert.Equal(t, "Yanked 2 items", resultModel.status)
}

func TestUpdate_YankResultMsg_DirCut(t *testing.T) {
	m := newTestModel()
	msg := yankResultMsg{
		secrets: []*model.Secret{},
		paths:   []string{"dir1/", "dir2/", "dir3/"},
		isCut:   true,
		isDir:   true,
	}

	result, _ := m.Update(msg)
	resultModel := result.(*Model)

	assert.Equal(t, "Cut 3 items for move", resultModel.status)
}

func TestUpdate_YankResultMsg_BulkYank(t *testing.T) {
	m := newTestModel()
	msg := yankResultMsg{
		secrets: []*model.Secret{{Path: "a"}, {Path: "b"}, {Path: "c"}},
		paths:   []string{},
		isCut:   false,
		isDir:   false,
	}

	result, _ := m.Update(msg)
	resultModel := result.(*Model)

	assert.Equal(t, "Yanked 3 secrets", resultModel.status)
}

func TestUpdate_YankResultMsg_BulkCut(t *testing.T) {
	m := newTestModel()
	msg := yankResultMsg{
		secrets: []*model.Secret{{Path: "a"}, {Path: "b"}},
		paths:   []string{},
		isCut:   true,
		isDir:   false,
	}

	result, _ := m.Update(msg)
	resultModel := result.(*Model)

	assert.Equal(t, "Cut 2 secrets for move", resultModel.status)
}

func TestUpdate_VersionHistoryMsg_Success(t *testing.T) {
	m := newTestModel()
	versions := []model.SecretVersion{
		{Version: "1", CreatedTime: "2024-01-01"},
		{Version: "2", CreatedTime: "2024-01-02"},
	}

	result, cmd := m.Update(versionHistoryMsg{versions: versions})
	resultModel := result.(*Model)

	assert.Equal(t, model.ModeVersionHistory, resultModel.mode)
	assert.Equal(t, versions, resultModel.versionHistory)
	assert.Equal(t, 0, resultModel.versionCursor)
	assert.Nil(t, cmd)
}

func TestUpdate_VersionHistoryMsg_Error(t *testing.T) {
	m := newTestModel()
	result, cmd := m.Update(versionHistoryMsg{err: assert.AnError})
	resultModel := result.(*Model)

	assert.Contains(t, resultModel.errMsg, "assert.AnError")
	assert.Nil(t, cmd)
}

func TestUpdate_VersionDetailMsg_Success(t *testing.T) {
	m := newTestModel()
	secret := &model.Secret{
		Path: "secret/foo",
		Keys: []string{"key1"},
		Data: map[string]string{"key1": "val1"},
	}

	result, cmd := m.Update(versionDetailMsg{version: 2, secret: secret})
	resultModel := result.(*Model)

	assert.Equal(t, model.ModeSecret, resultModel.mode)
	assert.Equal(t, secret, resultModel.secret)
	assert.Equal(t, 0, resultModel.secretCursor)
	assert.Empty(t, resultModel.revealed)
	assert.Empty(t, resultModel.secretBase64)
	assert.False(t, resultModel.secretAllRevealed)
	assert.False(t, resultModel.secretJSONView)
	assert.Nil(t, cmd)
}

func TestUpdate_VersionDetailMsg_Error(t *testing.T) {
	m := newTestModel()
	result, _ := m.Update(versionDetailMsg{err: assert.AnError})
	assert.Contains(t, result.(*Model).errMsg, "assert.AnError")
}

// Note: newSecretInlineMsg handler calls textinput.Focus() which internally
// requires a cursor blink ID that cannot be initialized outside of a running
// Bubble Tea program. We skip testing the full Update path for this message
// and instead verify the fields would be set correctly.

// Note: confirmCreateMsg and confirmCreateEditorMsg handlers call
// enterConfirmMode() which internally calls confirmInput.Focus().
// The Focus() method requires a cursor blink ID that cannot be initialized
// outside of a running Bubble Tea program (causes nil pointer panic).
// enterConfirmMode is already tested directly in model_test.go.

func TestUpdate_UndoableStatusMsg_InSecretMode(t *testing.T) {
	m := newTestModel()
	m.mode = model.ModeSecret
	m.secret = &model.Secret{
		Path: "secret/foo",
		Keys: []string{"key1"},
		Data: map[string]string{"key1": "old"},
	}

	reloadSecret := &model.Secret{
		Path: "secret/foo",
		Keys: []string{"key1", "key2"},
		Data: map[string]string{"key1": "new", "key2": "val2"},
	}

	msg := undoableStatusMsg{
		status:       "Updated",
		undo:         model.UndoAction{Description: "edit"},
		reloadSecret: reloadSecret,
	}

	result, cmd := m.Update(msg)
	resultModel := result.(*Model)

	assert.Equal(t, "Updated", resultModel.status)
	assert.Len(t, resultModel.undoStack, 1)
	assert.Equal(t, reloadSecret.Data, resultModel.secret.Data)
	assert.Equal(t, reloadSecret.Keys, resultModel.secret.Keys)
	assert.Nil(t, cmd, "should not refresh when not in explorer mode")
}

func TestUpdate_UndoableStatusMsg_CursorAdjustment(t *testing.T) {
	m := newTestModel()
	m.mode = model.ModeSecret
	m.secretCursor = 5
	m.secret = &model.Secret{
		Path: "secret/foo",
		Keys: []string{"a", "b", "c", "d", "e", "f"},
		Data: map[string]string{"a": "1", "b": "2", "c": "3", "d": "4", "e": "5", "f": "6"},
	}

	reloadSecret := &model.Secret{
		Path: "secret/foo",
		Keys: []string{"a", "b"}, // only 2 keys now
		Data: map[string]string{"a": "1", "b": "2"},
	}

	msg := undoableStatusMsg{
		status:       "Deleted keys",
		undo:         model.UndoAction{Description: "delete"},
		reloadSecret: reloadSecret,
	}

	result, _ := m.Update(msg)
	resultModel := result.(*Model)
	assert.Equal(t, 1, resultModel.secretCursor, "cursor should be adjusted to last key index")
}

func TestUpdate_RedoableStatusMsg(t *testing.T) {
	m := newTestModel()
	m.mode = model.ModeSecret

	msg := redoableStatusMsg{
		status: "Redid action",
		redo:   model.UndoAction{Description: "redo"},
	}

	result, cmd := m.Update(msg)
	resultModel := result.(*Model)

	assert.Equal(t, "Redid action", resultModel.status)
	assert.Len(t, resultModel.redoStack, 1)
	assert.Nil(t, cmd)
}

// ---------------------------------------------------------------------------
// handleListResult
// ---------------------------------------------------------------------------

func TestHandleListResult_MountPreview(t *testing.T) {
	m := newTestModel()
	m.previewSecret = &model.Secret{Path: "old"}
	m.previewEntries = []model.Entry{{Name: "old/"}}

	entries := []model.Entry{{Name: "subdir/", IsDir: true}}
	result, cmd := m.handleListResult(listResultMsg{
		path:    "@@mount_preview@@",
		entries: entries,
	})
	resultModel := result.(*Model)

	assert.Equal(t, entries, resultModel.previewEntries)
	assert.Nil(t, resultModel.previewSecret, "mount preview should clear previewSecret")
	assert.Nil(t, cmd)
}

func TestHandleListResult_MountPreview_Error(t *testing.T) {
	m := newTestModel()
	m.previewEntries = []model.Entry{{Name: "old/"}}

	result, cmd := m.handleListResult(listResultMsg{
		path: "@@mount_preview@@",
		err:  assert.AnError,
	})
	resultModel := result.(*Model)

	assert.Nil(t, resultModel.previewEntries)
	assert.Nil(t, resultModel.previewSecret)
	assert.Nil(t, cmd)
}

func TestHandleListResult_Error(t *testing.T) {
	m := newTestModel()
	result, cmd := m.handleListResult(listResultMsg{
		path: "some/path/",
		err:  assert.AnError,
	})
	resultModel := result.(*Model)

	assert.Contains(t, resultModel.errMsg, "assert.AnError")
	assert.Nil(t, cmd)
}

func TestHandleListResult_PreviewResult(t *testing.T) {
	m := newTestModel()
	m.path = []string{"dir/"}
	m.previewSecret = &model.Secret{Path: "old-preview"}

	entries := []model.Entry{{Name: "preview-item", IsDir: false}}
	result, cmd := m.handleListResult(listResultMsg{
		path:    "other/path/",
		entries: entries,
	})
	resultModel := result.(*Model)

	assert.Equal(t, entries, resultModel.previewEntries)
	assert.Nil(t, resultModel.previewSecret, "should clear previewSecret for preview results")
	assert.Nil(t, cmd)
}

func TestHandleListResult_ParentResult(t *testing.T) {
	m := newTestModel()
	m.path = []string{"secret/", "data/"}
	m.cursorMemory = make(map[string]int)

	parentEntries := []model.Entry{
		{Name: "other/", IsDir: true},
		{Name: "data/", IsDir: true},
	}
	result, cmd := m.handleListResult(listResultMsg{
		path:    "secret/",
		entries: parentEntries,
	})
	resultModel := result.(*Model)

	assert.Equal(t, parentEntries, resultModel.parentList)
	// cursorMemory should be set to index of "data/" (=1)
	assert.Equal(t, 1, resultModel.cursorMemory["secret/"])
	assert.Nil(t, cmd)
}

// ---------------------------------------------------------------------------
// handleSecretResult
// ---------------------------------------------------------------------------

func TestHandleSecretResult_Error(t *testing.T) {
	m := newTestModel()
	result, cmd := m.handleSecretResult(secretResultMsg{err: assert.AnError})
	resultModel := result.(*Model)

	assert.Contains(t, resultModel.errMsg, "assert.AnError")
	assert.Nil(t, cmd)
}

func TestHandleSecretResult_OpenPopup(t *testing.T) {
	m := newTestModel()
	secret := &model.Secret{
		Path: "secret/test",
		Keys: []string{"user", "pass"},
		Data: map[string]string{"user": "admin", "pass": "secret"},
	}

	result, cmd := m.handleSecretResult(secretResultMsg{
		secret:    secret,
		openPopup: true,
	})
	resultModel := result.(*Model)

	assert.Equal(t, model.ModeSecret, resultModel.mode)
	assert.Equal(t, secret, resultModel.secret)
	assert.Equal(t, 0, resultModel.secretCursor)
	assert.Empty(t, resultModel.revealed)
	assert.Empty(t, resultModel.secretBase64)
	assert.False(t, resultModel.secretAllRevealed)
	assert.False(t, resultModel.secretJSONView)
	assert.Nil(t, cmd)
}

func TestHandleSecretResult_Preview(t *testing.T) {
	m := newTestModel()
	m.mode = model.ModeExplorer

	secret := &model.Secret{
		Path: "secret/preview",
		Keys: []string{"key"},
		Data: map[string]string{"key": "val"},
	}

	result, _ := m.handleSecretResult(secretResultMsg{
		secret:    secret,
		openPopup: false,
	})
	resultModel := result.(*Model)

	// In preview mode, the secret goes to previewSecret
	assert.Equal(t, secret, resultModel.previewSecret)
	assert.Equal(t, model.ModeExplorer, resultModel.mode, "should stay in explorer for preview")
}

// ---------------------------------------------------------------------------
// editorCommand
// ---------------------------------------------------------------------------

func TestEditorCommand_FromConfig(t *testing.T) {
	m := &Model{config: &config.Config{Editor: "vim"}}
	editor, err := m.editorCommand()
	require.NoError(t, err)
	assert.Equal(t, "vim", editor)
}

func TestEditorCommand_NonexistentEditor(t *testing.T) {
	m := &Model{config: &config.Config{Editor: "nonexistent-editor-12345"}}
	_, err := m.editorCommand()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found in PATH")
}

func TestEditorCommand_FallbackToVim(t *testing.T) {
	// When config is nil editor, EDITOR and VISUAL are empty, should fall back to "vim"
	m := &Model{config: &config.Config{}}
	t.Setenv("EDITOR", "")
	t.Setenv("VISUAL", "")

	editor, err := m.editorCommand()
	// This will succeed if vim is installed (common on macOS/Linux)
	if err == nil {
		assert.Equal(t, "vim", editor)
	}
	// If vim is not installed, we get the "not found" error which is also correct behavior
}

func TestEditorCommand_FromEnvEditor(t *testing.T) {
	m := &Model{config: &config.Config{}}
	t.Setenv("EDITOR", "vi")
	t.Setenv("VISUAL", "")

	editor, err := m.editorCommand()
	// vi should exist on most systems
	if err == nil {
		assert.Equal(t, "vi", editor)
	}
}

// ---------------------------------------------------------------------------
// handleInputSubmit - testable paths
// ---------------------------------------------------------------------------

func TestHandleInputSubmit_Rename_EmptyValue(t *testing.T) {
	ti := textinput.New()
	ti.SetValue("")
	m := &Model{
		mode:        model.ModeInput,
		inputAction: model.InputRename,
		textInput:   ti,
	}

	result, cmd := m.handleInputSubmit()
	resultModel := result.(*Model)

	assert.Equal(t, model.ModeExplorer, resultModel.mode)
	assert.Equal(t, model.InputNone, resultModel.inputAction)
	assert.Nil(t, cmd)
}

func TestHandleInputSubmit_NewSecret_EmptyValue(t *testing.T) {
	ti := textinput.New()
	ti.SetValue("")
	m := &Model{
		mode:        model.ModeInput,
		inputAction: model.InputNewSecret,
		textInput:   ti,
	}

	result, cmd := m.handleInputSubmit()
	resultModel := result.(*Model)

	assert.Equal(t, model.ModeExplorer, resultModel.mode)
	assert.Equal(t, model.InputNone, resultModel.inputAction)
	assert.Nil(t, cmd)
}

func TestHandleInputSubmit_NewSecretEditor_EmptyValue(t *testing.T) {
	ti := textinput.New()
	ti.SetValue("")
	m := &Model{
		mode:        model.ModeInput,
		inputAction: model.InputNewSecretEditor,
		textInput:   ti,
	}

	result, cmd := m.handleInputSubmit()
	resultModel := result.(*Model)

	assert.Equal(t, model.ModeExplorer, resultModel.mode)
	assert.Equal(t, model.InputNone, resultModel.inputAction)
	assert.Nil(t, cmd)
}

func TestHandleInputSubmit_NewKey_TransitionsToNewValue(t *testing.T) {
	ti := textinput.New()
	ti.SetValue("api_key")
	m := &Model{
		mode:        model.ModeInput,
		inputAction: model.InputNewKey,
		textInput:   ti,
	}

	_, cmd := m.handleInputSubmit()

	assert.Equal(t, model.InputNewValue, m.inputAction)
	assert.Equal(t, "api_key", m.inputBuffer)
	assert.Contains(t, m.inputLabel, "api_key")
	assert.Equal(t, "", m.textInput.Value())
	assert.NotNil(t, cmd, "should return blink cmd for next input")
}

func TestHandleInputSubmit_NewValue_EmptyKey(t *testing.T) {
	ti := textinput.New()
	ti.SetValue("some-value")
	m := &Model{
		mode:        model.ModeInput,
		inputAction: model.InputNewValue,
		inputBuffer: "", // empty key
		textInput:   ti,
		secret:      &model.Secret{Path: "secret/foo"},
	}

	result, cmd := m.handleInputSubmit()
	resultModel := result.(*Model)

	assert.Equal(t, model.ModeSecret, resultModel.mode)
	assert.Equal(t, model.InputNone, resultModel.inputAction)
	assert.Empty(t, resultModel.inputBuffer)
	assert.Nil(t, cmd)
}

func TestHandleInputSubmit_DefaultAction(t *testing.T) {
	ti := textinput.New()
	ti.SetValue("something")
	m := &Model{
		mode:        model.ModeInput,
		inputAction: model.InputNone,
		textInput:   ti,
	}

	result, cmd := m.handleInputSubmit()
	resultModel := result.(*Model)

	assert.Equal(t, model.ModeExplorer, resultModel.mode)
	assert.Equal(t, model.InputNone, resultModel.inputAction)
	assert.Nil(t, cmd)
}

// ---------------------------------------------------------------------------
// handleSecretMouse - click outside closes popup
// ---------------------------------------------------------------------------

func TestHandleSecretMouse_ClickOutside_ClosesPopup(t *testing.T) {
	m := newTestModel()
	m.mode = model.ModeSecret
	m.width = 100
	m.height = 100
	m.secret = &model.Secret{Path: "secret/foo"}
	m.secretCursor = 2
	m.revealed = map[string]bool{"key": true}
	m.secretBase64 = map[string]bool{"key": true}
	m.secretAllRevealed = true
	m.secretJSONView = true

	// Click at (0, 0) which is outside the centered popup
	result, cmd := m.handleSecretMouse(tea.MouseMsg{
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionPress,
		X:      0,
		Y:      0,
	})
	resultModel := result.(*Model)

	assert.Equal(t, model.ModeExplorer, resultModel.mode)
	assert.Nil(t, resultModel.secret)
	assert.Equal(t, 0, resultModel.secretCursor)
	assert.Empty(t, resultModel.revealed)
	assert.Empty(t, resultModel.secretBase64)
	assert.False(t, resultModel.secretAllRevealed)
	assert.False(t, resultModel.secretJSONView)
	assert.NotNil(t, cmd, "should refresh after closing popup")
}

func TestHandleSecretMouse_ClickInside_NoClose(t *testing.T) {
	m := newTestModel()
	m.mode = model.ModeSecret
	m.width = 100
	m.height = 100
	m.secret = &model.Secret{Path: "secret/foo"}

	// Click at center of screen which is inside the popup
	result, _ := m.handleSecretMouse(tea.MouseMsg{
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionPress,
		X:      50,
		Y:      50,
	})
	assert.Equal(t, model.ModeSecret, result.(*Model).mode)
}

// ---------------------------------------------------------------------------
// handleSearchKey
// ---------------------------------------------------------------------------

func TestHandleSearchKey_Esc(t *testing.T) {
	m := newTestModel()
	m.mode = model.ModeSearch
	m.cursor = 5
	m.priorCursor = 2
	m.searchInput.Focus()

	result, _ := m.handleSearchKey(tea.KeyMsg{Type: tea.KeyEsc})
	resultModel := result.(*Model)

	assert.Equal(t, model.ModeExplorer, resultModel.mode)
	assert.Equal(t, 2, resultModel.cursor, "should restore prior cursor")
}

func TestHandleSearchKey_Enter(t *testing.T) {
	m := newTestModel()
	m.mode = model.ModeSearch
	m.cursor = 3
	m.searchInput.Focus()

	result, _ := m.handleSearchKey(tea.KeyMsg{Type: tea.KeyEnter})
	resultModel := result.(*Model)

	assert.Equal(t, model.ModeExplorer, resultModel.mode)
	assert.Equal(t, 3, resultModel.cursor, "should keep current cursor position")
}

// ---------------------------------------------------------------------------
// handleFilterKey
// ---------------------------------------------------------------------------

func TestHandleFilterKey_Esc(t *testing.T) {
	m := newTestModel()
	m.mode = model.ModeFilter
	m.filterQuery = "active-filter"
	m.filteredIdx = []int{0, 2}
	m.priorCursor = 1
	m.cursor = 0
	m.searchInput.Focus()

	result, _ := m.handleFilterKey(tea.KeyMsg{Type: tea.KeyEsc})
	resultModel := result.(*Model)

	assert.Equal(t, model.ModeExplorer, resultModel.mode)
	assert.Empty(t, resultModel.filterQuery)
	assert.Nil(t, resultModel.filteredIdx)
	assert.Equal(t, 1, resultModel.cursor, "should restore prior cursor")
}

func TestHandleFilterKey_Enter(t *testing.T) {
	m := newTestModel()
	m.mode = model.ModeFilter
	m.filterQuery = "test"
	m.cursor = 0
	m.searchInput.Focus()

	result, _ := m.handleFilterKey(tea.KeyMsg{Type: tea.KeyEnter})
	resultModel := result.(*Model)

	assert.Equal(t, model.ModeExplorer, resultModel.mode)
	assert.Equal(t, "test", resultModel.filterQuery, "should keep filter active")
}

// ---------------------------------------------------------------------------
// handleJumpPathKey
// ---------------------------------------------------------------------------

func TestHandleJumpPathKey_Esc(t *testing.T) {
	ti := textinput.New()
	ti.Focus()
	m := &Model{
		mode:            model.ModeJumpPath,
		textInput:       ti,
		jumpCompletions: []string{"a/", "b/"},
		jumpCompIdx:     1,
	}

	result, cmd := m.handleJumpPathKey(tea.KeyMsg{Type: tea.KeyEsc})
	resultModel := result.(*Model)

	assert.Equal(t, model.ModeExplorer, resultModel.mode)
	assert.Nil(t, resultModel.jumpCompletions)
	assert.Equal(t, -1, resultModel.jumpCompIdx)
	assert.Nil(t, cmd)
}

// ---------------------------------------------------------------------------
// handleMouse - dispatching to more modes
// ---------------------------------------------------------------------------

func TestHandleMouse_SecretMode_Scroll(t *testing.T) {
	m := &Model{
		mode: model.ModeSecret,
		secret: &model.Secret{
			Keys: []string{"a", "b", "c"},
			Data: map[string]string{"a": "1", "b": "2", "c": "3"},
		},
		secretCursor: 1,
	}
	result, _ := m.handleMouse(tea.MouseMsg{Button: tea.MouseButtonWheelDown})
	assert.Equal(t, 2, result.(*Model).secretCursor)
}

func TestHandleMouse_HelpMode_Scroll(t *testing.T) {
	m := &Model{
		mode:       model.ModeHelp,
		height:     100,
		width:      100,
		helpScroll: 5,
	}
	result, _ := m.handleMouse(tea.MouseMsg{Button: tea.MouseButtonWheelUp})
	assert.Equal(t, 4, result.(*Model).helpScroll)
}

func TestHandleMouse_VersionHistoryMode_Scroll(t *testing.T) {
	m := &Model{
		mode: model.ModeVersionHistory,
		versionHistory: []model.SecretVersion{
			{Version: "1"},
			{Version: "2"},
		},
		versionCursor: 0,
	}
	result, _ := m.handleMouse(tea.MouseMsg{Button: tea.MouseButtonWheelDown})
	assert.Equal(t, 1, result.(*Model).versionCursor)
}

func TestHandleMouse_BookmarkMode_Scroll(t *testing.T) {
	m := &Model{
		mode: model.ModeBookmark,
		bookmarks: []config.Bookmark{
			{Name: "a", Mount: "kv", Path: "a"},
			{Name: "b", Mount: "kv", Path: "b"},
		},
		bookmarkCursor: 0,
	}
	result, _ := m.handleMouse(tea.MouseMsg{Button: tea.MouseButtonWheelDown})
	assert.Equal(t, 1, result.(*Model).bookmarkCursor)
}

func TestHandleMouse_ThemePickerMode_Scroll(t *testing.T) {
	entries := []ui.ThemeEntry{
		{Name: "Dark Themes", IsHeader: true},
		{Name: "dracula", IsHeader: false},
		{Name: "monokai", IsHeader: false},
	}
	m := &Model{
		mode:              model.ModeThemePicker,
		themeEntries:      entries,
		themeCursor:       1,
		config:            &config.Config{},
		activeColorscheme: "dracula",
	}
	result, _ := m.handleMouse(tea.MouseMsg{Button: tea.MouseButtonWheelDown})
	assert.Equal(t, 2, result.(*Model).themeCursor)
}

// ---------------------------------------------------------------------------
// handleKey - more mode dispatches
// ---------------------------------------------------------------------------

func TestHandleKey_VersionHistory_Esc(t *testing.T) {
	m := &Model{
		mode:           model.ModeVersionHistory,
		versionHistory: []model.SecretVersion{{Version: "1"}},
		keys:           DefaultKeyMap(),
	}
	result, _ := m.handleKey(tea.KeyMsg{Type: tea.KeyEsc})
	assert.Equal(t, model.ModeSecret, result.(*Model).mode)
}

func TestHandleKey_Help_Esc(t *testing.T) {
	m := newTestModel()
	m.mode = model.ModeHelp
	m.height = 100

	result, _ := m.handleKey(tea.KeyMsg{Type: tea.KeyEsc})
	assert.Equal(t, model.ModeExplorer, result.(*Model).mode)
}

func TestHandleKey_SecretEdit_Esc(t *testing.T) {
	ti := textinput.New()
	ti.SetValue("editing")
	m := &Model{
		mode:             model.ModeSecretEdit,
		keys:             DefaultKeyMap(),
		textInput:        ti,
		secret:           &model.Secret{Path: "s", Keys: []string{"k"}, Data: map[string]string{"k": "v"}},
		secretEditKey:    "k",
		secretEditColumn: 1,
	}
	result, _ := m.handleKey(tea.KeyMsg{Type: tea.KeyEsc})
	assert.Equal(t, model.ModeSecret, result.(*Model).mode)
}

// ---------------------------------------------------------------------------
// handleSecretResult - preview path
// ---------------------------------------------------------------------------

func TestHandleSecretResult_Preview_ClearsPreviewEntries(t *testing.T) {
	m := newTestModel()
	m.previewEntries = []model.Entry{{Name: "old/"}}

	secret := &model.Secret{
		Path: "secret/preview",
		Keys: []string{"k"},
		Data: map[string]string{"k": "v"},
	}

	result, _ := m.handleSecretResult(secretResultMsg{secret: secret, openPopup: false})
	resultModel := result.(*Model)

	assert.Equal(t, secret, resultModel.previewSecret)
	assert.Nil(t, resultModel.previewEntries, "preview entries should be cleared for secret preview")
}

// ---------------------------------------------------------------------------
// handleListResult - current path updates entries
// ---------------------------------------------------------------------------

func TestHandleListResult_CurrentPath(t *testing.T) {
	m := newTestModel()
	m.path = []string{"dir/"}
	m.entries = []model.Entry{{Name: "old", IsDir: false}}
	m.cursor = 0

	newEntries := []model.Entry{
		{Name: "alpha", IsDir: false},
		{Name: "beta/", IsDir: true},
	}
	result, _ := m.handleListResult(listResultMsg{
		path:    "dir/",
		entries: newEntries,
	})
	resultModel := result.(*Model)

	assert.Equal(t, newEntries, resultModel.entries)
}

func TestHandleListResult_CurrentPath_CursorBeyondRange(t *testing.T) {
	m := newTestModel()
	m.path = []string{"dir/"}
	m.cursor = 10

	newEntries := []model.Entry{{Name: "only-one", IsDir: false}}
	result, _ := m.handleListResult(listResultMsg{
		path:    "dir/",
		entries: newEntries,
	})
	resultModel := result.(*Model)

	assert.Equal(t, 0, resultModel.cursor, "cursor should be clamped to last entry index")
}

func TestHandleListResult_CurrentPath_WithFilter(t *testing.T) {
	m := newTestModel()
	m.path = []string{"dir/"}
	m.filterQuery = "al"

	newEntries := []model.Entry{
		{Name: "alpha", IsDir: false},
		{Name: "beta", IsDir: false},
	}
	result, _ := m.handleListResult(listResultMsg{
		path:    "dir/",
		entries: newEntries,
	})
	resultModel := result.(*Model)

	assert.Equal(t, "al", resultModel.filterQuery, "filter should be preserved")
	assert.Equal(t, []int{0}, resultModel.filteredIdx, "filter should be reapplied")
}

// ---------------------------------------------------------------------------
// handleConfirmKey - other key delegates to textinput
// ---------------------------------------------------------------------------

func TestHandleConfirmKey_OtherKey_Delegates(t *testing.T) {
	ci := textinput.New()
	ci.Focus()
	m := &Model{
		mode:            model.ModeConfirm,
		prevConfirmMode: model.ModeExplorer,
		confirmInput:    ci,
	}

	// Type a character -- it should be delegated to the textinput
	result, _ := m.handleConfirmKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'D'}})
	resultModel := result.(*Model)

	assert.Equal(t, model.ModeConfirm, resultModel.mode, "should stay in confirm mode")
	assert.Equal(t, "D", resultModel.confirmInput.Value())
}

// ---------------------------------------------------------------------------
// handleInputKey - other key delegates to textinput
// ---------------------------------------------------------------------------

func TestHandleInputKey_OtherKey_Delegates(t *testing.T) {
	ti := textinput.New()
	ti.Focus()
	m := &Model{
		mode:        model.ModeInput,
		inputAction: model.InputRename,
		textInput:   ti,
	}

	result, _ := m.handleInputKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	resultModel := result.(*Model)

	assert.Equal(t, model.ModeInput, resultModel.mode, "should stay in input mode")
	assert.Equal(t, "a", resultModel.textInput.Value())
}

// ---------------------------------------------------------------------------
// Update - statusMsg in explorer mode triggers refresh
// ---------------------------------------------------------------------------

func TestUpdate_StatusMsg_InExplorerMode(t *testing.T) {
	m := newTestModel()
	m.mode = model.ModeExplorer
	m.errMsg = "old error"

	result, cmd := m.Update(statusMsg("Operation successful"))
	resultModel := result.(*Model)

	assert.Equal(t, "Operation successful", resultModel.status)
	assert.Empty(t, resultModel.errMsg)
	assert.NotNil(t, cmd, "should refresh when in explorer mode")
}

func TestUpdate_StatusMsg_InSecretMode(t *testing.T) {
	m := newTestModel()
	m.mode = model.ModeSecret

	result, cmd := m.Update(statusMsg("Operation successful"))
	resultModel := result.(*Model)

	assert.Equal(t, "Operation successful", resultModel.status)
	assert.Nil(t, cmd, "should NOT refresh when not in explorer mode")
}

// ---------------------------------------------------------------------------
// Update - undoableStatusMsg in explorer mode
// ---------------------------------------------------------------------------

func TestUpdate_UndoableStatusMsg_InExplorerMode(t *testing.T) {
	m := newTestModel()
	m.mode = model.ModeExplorer

	msg := undoableStatusMsg{
		status: "Created secret",
		undo:   model.UndoAction{Description: "create"},
	}

	result, cmd := m.Update(msg)
	resultModel := result.(*Model)

	assert.Equal(t, "Created secret", resultModel.status)
	assert.Len(t, resultModel.undoStack, 1)
	assert.NotNil(t, cmd, "should refresh when in explorer mode")
}

// ---------------------------------------------------------------------------
// Update - redoableStatusMsg in explorer mode
// ---------------------------------------------------------------------------

func TestUpdate_RedoableStatusMsg_InExplorerMode(t *testing.T) {
	m := newTestModel()
	m.mode = model.ModeExplorer

	msg := redoableStatusMsg{
		status: "Redone action",
		redo:   model.UndoAction{Description: "redo"},
	}

	result, cmd := m.Update(msg)
	resultModel := result.(*Model)

	assert.Equal(t, "Redone action", resultModel.status)
	assert.Len(t, resultModel.redoStack, 1)
	assert.NotNil(t, cmd, "should refresh when in explorer mode")
}

// ---------------------------------------------------------------------------
// Update - redoableStatusMsg with reloadSecret
// ---------------------------------------------------------------------------

func TestUpdate_RedoableStatusMsg_WithReload(t *testing.T) {
	m := newTestModel()
	m.mode = model.ModeSecret
	m.secret = &model.Secret{
		Path: "secret/foo",
		Keys: []string{"a"},
		Data: map[string]string{"a": "1"},
	}
	m.secretCursor = 0

	reloadSecret := &model.Secret{
		Path: "secret/foo",
		Keys: []string{"a", "b"},
		Data: map[string]string{"a": "new", "b": "2"},
	}

	msg := redoableStatusMsg{
		status:       "Redone edit",
		redo:         model.UndoAction{Description: "redo edit"},
		reloadSecret: reloadSecret,
	}

	result, _ := m.Update(msg)
	resultModel := result.(*Model)

	assert.Equal(t, reloadSecret.Data, resultModel.secret.Data)
	assert.Equal(t, reloadSecret.Keys, resultModel.secret.Keys)
}

func TestUpdate_RedoableStatusMsg_CursorClamp(t *testing.T) {
	m := newTestModel()
	m.mode = model.ModeSecret
	m.secretCursor = 5
	m.secret = &model.Secret{
		Path: "secret/foo",
		Keys: []string{"a", "b", "c", "d", "e", "f"},
		Data: map[string]string{"a": "1", "b": "2", "c": "3", "d": "4", "e": "5", "f": "6"},
	}

	reloadSecret := &model.Secret{
		Path: "secret/foo",
		Keys: []string{"a"},
		Data: map[string]string{"a": "1"},
	}

	msg := redoableStatusMsg{
		status:       "Redone delete",
		redo:         model.UndoAction{Description: "redo delete"},
		reloadSecret: reloadSecret,
	}

	result, _ := m.Update(msg)
	assert.Equal(t, 0, result.(*Model).secretCursor, "cursor should be clamped to last key")
}

// ---------------------------------------------------------------------------
// Update - unknown message type returns model unchanged
// ---------------------------------------------------------------------------

func TestUpdate_UnknownMsg(t *testing.T) {
	m := newTestModel()
	type customMsg struct{}

	result, cmd := m.Update(customMsg{})
	assert.Equal(t, m, result)
	assert.Nil(t, cmd)
}

// ---------------------------------------------------------------------------
// Update - mouse and key dispatch through Update
// ---------------------------------------------------------------------------

func TestUpdate_MouseMsg(t *testing.T) {
	m := newTestModel()
	m.mode = model.ModeVersionHistory
	m.versionHistory = []model.SecretVersion{{Version: "1"}, {Version: "2"}}
	m.versionCursor = 0

	result, _ := m.Update(tea.MouseMsg{Button: tea.MouseButtonWheelDown})
	assert.Equal(t, 1, result.(*Model).versionCursor)
}

func TestUpdate_KeyMsg(t *testing.T) {
	m := newTestModel()
	m.mode = model.ModeVersionHistory
	m.versionHistory = []model.SecretVersion{{Version: "1"}}

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	assert.Equal(t, model.ModeSecret, result.(*Model).mode)
}

// ---------------------------------------------------------------------------
// handleSecretEditKey - esc exits to ModeSecret
// ---------------------------------------------------------------------------

func TestHandleSecretEditKey_Esc(t *testing.T) {
	ti := textinput.New()
	ti.SetValue("editing")
	m := &Model{
		mode:              model.ModeSecretEdit,
		keys:              DefaultKeyMap(),
		textInput:         ti,
		secret:            &model.Secret{Path: "s", Keys: []string{"k"}, Data: map[string]string{"k": "v"}},
		secretEditKey:     "k",
		secretEditOrigKey: "k",
		secretEditColumn:  1,
	}
	result, _ := m.handleSecretEditKey(tea.KeyMsg{Type: tea.KeyEsc})
	resultModel := result.(*Model)

	assert.Equal(t, model.ModeSecret, resultModel.mode)
	// ESC cancels the edit -- value should NOT be saved back
	assert.Equal(t, "v", resultModel.secret.Data["k"])
	assert.Empty(t, resultModel.secretEditKey)
	assert.Empty(t, resultModel.secretEditOrigKey)
	assert.Equal(t, 0, resultModel.secretEditColumn)
}

func TestHandleSecretEditKey_Esc_NewKeyPlaceholder(t *testing.T) {
	ti := textinput.New()
	ti.SetValue("")
	m := &Model{
		mode:              model.ModeSecretEdit,
		keys:              DefaultKeyMap(),
		textInput:         ti,
		secret:            &model.Secret{Path: "s", Keys: []string{"existing", ""}, Data: map[string]string{"existing": "val", "": ""}},
		secretEditKey:     "",
		secretEditOrigKey: "",
		secretEditColumn:  0,
		secretCursor:      1,
	}
	result, _ := m.handleSecretEditKey(tea.KeyMsg{Type: tea.KeyEsc})
	resultModel := result.(*Model)

	assert.Equal(t, model.ModeSecret, resultModel.mode)
	// The empty placeholder key should be removed
	assert.Equal(t, []string{"existing"}, resultModel.secret.Keys)
	_, hasEmpty := resultModel.secret.Data[""]
	assert.False(t, hasEmpty, "empty placeholder key should be removed from Data")
	assert.Equal(t, 0, resultModel.secretCursor, "cursor should be clamped")
}

func TestHandleSecretEditKey_Tab(t *testing.T) {
	ti := textinput.New()
	ti.SetValue("key-name")
	m := &Model{
		mode:              model.ModeSecretEdit,
		keys:              DefaultKeyMap(),
		textInput:         ti,
		secret:            &model.Secret{Path: "s", Keys: []string{"key-name"}, Data: map[string]string{"key-name": "val"}},
		secretEditKey:     "key-name",
		secretEditOrigKey: "key-name",
		secretEditColumn:  0,
	}
	result, cmd := m.switchEditColumn()
	resultModel := result.(*Model)

	assert.Equal(t, 1, resultModel.secretEditColumn)
	assert.Equal(t, "val", resultModel.textInput.Value())
	assert.NotNil(t, cmd)
}

func TestHandleSecretEditKey_Enter_OnlyValueChanged(t *testing.T) {
	ti := textinput.New()
	ti.SetValue("new-value")
	m := &Model{
		mode:              model.ModeSecretEdit,
		keys:              DefaultKeyMap(),
		textInput:         ti,
		secret:            &model.Secret{Path: "s", Keys: []string{"mykey"}, Data: map[string]string{"mykey": "old"}},
		secretEditKey:     "mykey",
		secretEditOrigKey: "mykey",
		secretEditColumn:  1,
		client:            newTestClient("secret"),
	}
	result, cmd := m.handleSecretEditKey(tea.KeyMsg{Type: tea.KeyEnter})
	resultModel := result.(*Model)

	assert.Equal(t, model.ModeSecret, resultModel.mode)
	assert.Equal(t, "new-value", resultModel.secret.Data["mykey"])
	assert.NotNil(t, cmd, "should return cmd to save value")
}

func TestHandleSecretEditKey_Enter_EmptyNewKey_Removed(t *testing.T) {
	ti := textinput.New()
	ti.SetValue("")
	m := &Model{
		mode:              model.ModeSecretEdit,
		keys:              DefaultKeyMap(),
		textInput:         ti,
		secret:            &model.Secret{Path: "s", Keys: []string{"existing", ""}, Data: map[string]string{"existing": "val", "": ""}},
		secretEditKey:     "",
		secretEditOrigKey: "",
		secretEditColumn:  0,
		secretCursor:      1,
	}
	result, cmd := m.handleSecretEditKey(tea.KeyMsg{Type: tea.KeyEnter})
	resultModel := result.(*Model)

	assert.Equal(t, model.ModeSecret, resultModel.mode)
	assert.Equal(t, []string{"existing"}, resultModel.secret.Keys)
	assert.Nil(t, cmd, "should return nil cmd when new key is empty")
}

func TestHandleSecretEditKey_OtherKey_Delegates(t *testing.T) {
	ti := textinput.New()
	ti.Focus()
	ti.SetValue("")
	m := &Model{
		mode:              model.ModeSecretEdit,
		keys:              DefaultKeyMap(),
		textInput:         ti,
		secret:            &model.Secret{Path: "s", Keys: []string{"k"}, Data: map[string]string{"k": "v"}},
		secretEditKey:     "k",
		secretEditOrigKey: "k",
		secretEditColumn:  1,
	}
	m.handleSecretEditKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	assert.Equal(t, "x", m.textInput.Value())
}

// ---------------------------------------------------------------------------
// handleListResult - empty directory auto-navigates up
// ---------------------------------------------------------------------------

func TestHandleListResult_EmptyDir_NavigatesUp(t *testing.T) {
	m := newTestModel()
	m.path = []string{"dir/", "subdir/"}

	result, cmd := m.handleListResult(listResultMsg{
		path:    "dir/subdir/",
		entries: []model.Entry{}, // empty dir
	})
	resultModel := result.(*Model)

	assert.Equal(t, []string{"dir/"}, resultModel.path, "should strip last path segment")
	assert.NotNil(t, cmd, "should trigger navigation commands")
}

// ---------------------------------------------------------------------------
// handleSearchKey - typing delegates to textinput and calls jumpToMatch
// ---------------------------------------------------------------------------

func TestHandleSearchKey_Typing(t *testing.T) {
	m := newTestModel()
	m.mode = model.ModeSearch
	m.entries = []model.Entry{
		{Name: "alpha"},
		{Name: "bravo"},
		{Name: "charlie"},
	}
	m.cursor = 0
	m.searchInput.Focus()

	// Simulate typing 'b'
	m.handleSearchKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
	assert.Equal(t, 1, m.cursor, "should jump to 'bravo'")
}

// ---------------------------------------------------------------------------
// handleFilterKey - typing live-filters
// ---------------------------------------------------------------------------

func TestHandleFilterKey_Typing(t *testing.T) {
	m := newTestModel()
	m.mode = model.ModeFilter
	m.entries = []model.Entry{
		{Name: "alpha"},
		{Name: "bravo"},
		{Name: "charlie"},
	}
	m.cursor = 2
	m.searchInput.Focus()

	// Simulate typing 'a'
	m.handleFilterKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})

	assert.Equal(t, "a", m.filterQuery)
	assert.NotNil(t, m.filteredIdx)
	// Cursor should be clamped if needed
	vis := m.visibleEntries()
	assert.True(t, m.cursor < len(vis) || len(vis) == 0)
}

// ---------------------------------------------------------------------------
// handleHelpKey - down with arrow key / ctrl+n
// ---------------------------------------------------------------------------

func TestHandleHelpKey_DownArrow(t *testing.T) {
	m := newTestModel()
	m.mode = model.ModeHelp
	m.height = 100
	m.helpScroll = 0

	m.handleHelpKey(tea.KeyMsg{Type: tea.KeyDown})
	assert.Equal(t, 1, m.helpScroll)
}

func TestHandleHelpKey_UpArrow(t *testing.T) {
	m := newTestModel()
	m.mode = model.ModeHelp
	m.height = 100
	m.helpScroll = 5

	m.handleHelpKey(tea.KeyMsg{Type: tea.KeyUp})
	assert.Equal(t, 4, m.helpScroll)
}

func TestHandleHelpKey_CtrlN(t *testing.T) {
	m := newTestModel()
	m.mode = model.ModeHelp
	m.height = 100
	m.helpScroll = 0

	m.handleHelpKey(tea.KeyMsg{Type: tea.KeyCtrlN})
	assert.Equal(t, 1, m.helpScroll)
}

func TestHandleHelpKey_CtrlP(t *testing.T) {
	m := newTestModel()
	m.mode = model.ModeHelp
	m.height = 100
	m.helpScroll = 5

	m.handleHelpKey(tea.KeyMsg{Type: tea.KeyCtrlP})
	assert.Equal(t, 4, m.helpScroll)
}

// ---------------------------------------------------------------------------
// handleVersionHistoryKey - h exits (same as esc)
// ---------------------------------------------------------------------------

func TestHandleVersionHistoryKey_H(t *testing.T) {
	m := &Model{
		mode:           model.ModeVersionHistory,
		versionHistory: []model.SecretVersion{{Version: "1"}},
	}
	result, _ := m.handleVersionHistoryKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	assert.Equal(t, model.ModeSecret, result.(*Model).mode)
}

// ---------------------------------------------------------------------------
// handleVersionHistoryKey - down/up with arrow keys
// ---------------------------------------------------------------------------

func TestHandleVersionHistoryKey_Down(t *testing.T) {
	m := &Model{
		mode: model.ModeVersionHistory,
		versionHistory: []model.SecretVersion{
			{Version: "1"},
			{Version: "2"},
		},
		versionCursor: 0,
	}
	m.handleVersionHistoryKey(tea.KeyMsg{Type: tea.KeyDown})
	assert.Equal(t, 1, m.versionCursor)
}

func TestHandleVersionHistoryKey_Up(t *testing.T) {
	m := &Model{
		mode: model.ModeVersionHistory,
		versionHistory: []model.SecretVersion{
			{Version: "1"},
			{Version: "2"},
		},
		versionCursor: 1,
	}
	m.handleVersionHistoryKey(tea.KeyMsg{Type: tea.KeyUp})
	assert.Equal(t, 0, m.versionCursor)
}

// ---------------------------------------------------------------------------
// handleHelpKey - search mode typing updates live filter
// ---------------------------------------------------------------------------

func TestHandleHelpKey_SearchMode_Typing(t *testing.T) {
	m := newTestModel()
	m.mode = model.ModeHelp
	m.height = 100
	m.helpSearching = true
	m.searchInput.Focus()

	m.handleHelpKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	assert.Equal(t, "q", m.helpFilter, "typing should update live filter")
	assert.Equal(t, 0, m.helpScroll, "scroll should reset on filter change")
}

// ---------------------------------------------------------------------------
// Update - editorResultMsg with nil secret (editing non-existent)
// ---------------------------------------------------------------------------

func TestUpdate_EditorResultMsg_NilSecret(t *testing.T) {
	m := newTestModel()
	m.secret = nil

	// When secret is nil and no newSecretPath, falls through to default
	result, cmd := m.Update(editorResultMsg{
		data:          map[string]string{"k": "v"},
		newSecretPath: "",
	})
	assert.Equal(t, m, result)
	assert.Nil(t, cmd)
}

// ---------------------------------------------------------------------------
// Update - newSecretEditorMsg
// ---------------------------------------------------------------------------

func TestUpdate_NewSecretEditorMsg(t *testing.T) {
	m := newTestModel()
	// This returns a cmd that calls openEditorForNewSecret
	result, cmd := m.Update(newSecretEditorMsg("secret/new-path"))
	assert.Equal(t, m, result)
	assert.NotNil(t, cmd, "should return cmd to open editor")
}

// ---------------------------------------------------------------------------
// handleInputSubmit - InputEditValue
// ---------------------------------------------------------------------------

func TestHandleInputSubmit_EditValue(t *testing.T) {
	ti := textinput.New()
	ti.SetValue("new-value")
	m := &Model{
		mode:        model.ModeInput,
		inputAction: model.InputEditValue,
		textInput:   ti,
		secret: &model.Secret{
			Path: "secret/foo",
			Keys: []string{"key1"},
			Data: map[string]string{"key1": "old-value"},
		},
		secretCursor: 0,
		client:       newTestClient("secret"),
	}

	result, cmd := m.handleInputSubmit()
	resultModel := result.(*Model)

	assert.Equal(t, model.ModeSecret, resultModel.mode)
	assert.Equal(t, model.InputNone, resultModel.inputAction)
	assert.NotNil(t, cmd, "should return cmd to save edited value")
}

// ---------------------------------------------------------------------------
// handleInputSubmit - InputRename with non-empty value
// ---------------------------------------------------------------------------

func TestHandleInputSubmit_Rename_NonEmpty(t *testing.T) {
	ti := textinput.New()
	ti.SetValue("new-name")
	m := &Model{
		mode:        model.ModeInput,
		inputAction: model.InputRename,
		textInput:   ti,
		path:        []string{"dir/"},
		client:      newTestClient("secret"),
		entries:     []model.Entry{{Name: "old-name"}},
		cursor:      0,
	}

	result, cmd := m.handleInputSubmit()
	resultModel := result.(*Model)

	assert.Equal(t, model.ModeExplorer, resultModel.mode)
	assert.Equal(t, model.InputNone, resultModel.inputAction)
	assert.NotNil(t, cmd, "should return cmd to rename entry")
}

// ---------------------------------------------------------------------------
// handleInputSubmit - InputNewSecret with non-empty value
// ---------------------------------------------------------------------------

func TestHandleInputSubmit_NewSecret_NonEmpty(t *testing.T) {
	ti := textinput.New()
	ti.SetValue("my-secret")
	m := &Model{
		mode:        model.ModeInput,
		inputAction: model.InputNewSecret,
		textInput:   ti,
		path:        []string{"dir/"},
		client:      newTestClient("secret"),
	}

	result, cmd := m.handleInputSubmit()
	resultModel := result.(*Model)

	assert.Equal(t, model.ModeExplorer, resultModel.mode)
	assert.NotNil(t, cmd, "should return cmd to create secret")
}

// ---------------------------------------------------------------------------
// handleInputSubmit - InputNewSecretEditor with non-empty value
// ---------------------------------------------------------------------------

func TestHandleInputSubmit_NewSecretEditor_NonEmpty(t *testing.T) {
	ti := textinput.New()
	ti.SetValue("editor-secret")
	m := &Model{
		mode:        model.ModeInput,
		inputAction: model.InputNewSecretEditor,
		textInput:   ti,
		path:        []string{"dir/"},
		client:      newTestClient("secret"),
	}

	result, cmd := m.handleInputSubmit()
	resultModel := result.(*Model)

	assert.Equal(t, model.ModeExplorer, resultModel.mode)
	assert.NotNil(t, cmd, "should return cmd to create secret via editor")
}

// ---------------------------------------------------------------------------
// handleInputSubmit - InputNewValue with non-empty key
// ---------------------------------------------------------------------------

func TestHandleInputSubmit_NewValue_NonEmptyKey(t *testing.T) {
	ti := textinput.New()
	ti.SetValue("my-value")
	m := &Model{
		mode:        model.ModeInput,
		inputAction: model.InputNewValue,
		inputBuffer: "api_key",
		textInput:   ti,
		secret: &model.Secret{
			Path: "secret/foo",
			Keys: []string{"existing"},
			Data: map[string]string{"existing": "val"},
		},
		client: newTestClient("secret"),
	}

	result, cmd := m.handleInputSubmit()
	resultModel := result.(*Model)

	assert.Equal(t, model.ModeSecret, resultModel.mode)
	assert.Equal(t, model.InputNone, resultModel.inputAction)
	assert.Empty(t, resultModel.inputBuffer)
	assert.NotNil(t, cmd, "should return cmd to add key-value")
}

// ---------------------------------------------------------------------------
// handleJumpPathKey - comprehensive paths
// ---------------------------------------------------------------------------

func TestHandleJumpPathKey_Enter_EmptyInput(t *testing.T) {
	ti := textinput.New()
	ti.SetValue("")
	ti.Focus()
	m := &Model{
		mode:            model.ModeJumpPath,
		textInput:       ti,
		jumpCompletions: []string{"a/", "b/"},
		jumpCompIdx:     0,
	}

	result, cmd := m.handleJumpPathKey(tea.KeyMsg{Type: tea.KeyEnter})
	resultModel := result.(*Model)

	assert.Equal(t, model.ModeExplorer, resultModel.mode)
	assert.Nil(t, resultModel.jumpCompletions)
	assert.Equal(t, -1, resultModel.jumpCompIdx)
	assert.Nil(t, cmd)
}

func TestHandleJumpPathKey_Enter_WithSelectedCompletion_Secret(t *testing.T) {
	ti := textinput.New()
	ti.SetValue("sec")
	ti.Focus()
	m := &Model{
		mode:            model.ModeJumpPath,
		textInput:       ti,
		jumpCompletions: []string{"secret/foo", "secret/bar"},
		jumpCompIdx:     0, // selected "secret/foo" (not a dir)
		client:          newTestClient("secret"),
	}

	result, cmd := m.handleJumpPathKey(tea.KeyMsg{Type: tea.KeyEnter})
	resultModel := result.(*Model)

	assert.Equal(t, "secret/foo", resultModel.textInput.Value())
	assert.Nil(t, resultModel.jumpCompletions)
	assert.NotNil(t, cmd, "should navigate to selected secret")
}

func TestHandleJumpPathKey_Enter_NoCompletion(t *testing.T) {
	ti := textinput.New()
	ti.SetValue("some/path")
	ti.Focus()
	m := &Model{
		mode:            model.ModeJumpPath,
		textInput:       ti,
		jumpCompletions: nil,
		jumpCompIdx:     -1,
		client:          newTestClient("secret"),
	}

	result, cmd := m.handleJumpPathKey(tea.KeyMsg{Type: tea.KeyEnter})
	resultModel := result.(*Model)

	assert.Nil(t, resultModel.jumpCompletions)
	assert.Equal(t, -1, resultModel.jumpCompIdx)
	assert.NotNil(t, cmd, "should navigate directly to typed path")
	_ = resultModel
}

func TestHandleJumpPathKey_Tab_CyclesForward(t *testing.T) {
	ti := textinput.New()
	ti.SetValue("s")
	m := &Model{
		mode:            model.ModeJumpPath,
		textInput:       ti,
		jumpCompletions: []string{"secret/", "staging/", "shared/"},
		jumpCompIdx:     -1,
	}

	m.handleJumpPathKey(tea.KeyMsg{Type: tea.KeyTab})
	assert.Equal(t, 0, m.jumpCompIdx)

	m.handleJumpPathKey(tea.KeyMsg{Type: tea.KeyTab})
	assert.Equal(t, 1, m.jumpCompIdx)

	m.handleJumpPathKey(tea.KeyMsg{Type: tea.KeyTab})
	assert.Equal(t, 2, m.jumpCompIdx)

	// Wraps around
	m.handleJumpPathKey(tea.KeyMsg{Type: tea.KeyTab})
	assert.Equal(t, 0, m.jumpCompIdx)
}

func TestHandleJumpPathKey_ShiftTab_CyclesBackward(t *testing.T) {
	ti := textinput.New()
	ti.SetValue("s")
	m := &Model{
		mode:            model.ModeJumpPath,
		textInput:       ti,
		jumpCompletions: []string{"secret/", "staging/"},
		jumpCompIdx:     0,
	}

	m.handleJumpPathKey(tea.KeyMsg{Type: tea.KeyShiftTab})
	assert.Equal(t, 1, m.jumpCompIdx, "should wrap to last")

	m.handleJumpPathKey(tea.KeyMsg{Type: tea.KeyShiftTab})
	assert.Equal(t, 0, m.jumpCompIdx)
}

func TestHandleJumpPathKey_Tab_NoCompletions(t *testing.T) {
	ti := textinput.New()
	ti.SetValue("x")
	m := &Model{
		mode:            model.ModeJumpPath,
		textInput:       ti,
		jumpCompletions: []string{},
		jumpCompIdx:     -1,
	}

	result, cmd := m.handleJumpPathKey(tea.KeyMsg{Type: tea.KeyTab})
	assert.Equal(t, m, result)
	assert.Nil(t, cmd)
}

// ---------------------------------------------------------------------------
// handleSecretEditKey - enter with renamed key
// ---------------------------------------------------------------------------

func TestHandleSecretEditKey_Enter_RenamedKey(t *testing.T) {
	ti := textinput.New()
	ti.SetValue("new-name")
	m := &Model{
		mode:              model.ModeSecretEdit,
		keys:              DefaultKeyMap(),
		textInput:         ti,
		secret:            &model.Secret{Path: "s", Keys: []string{"old-name"}, Data: map[string]string{"old-name": "val"}},
		secretEditKey:     "old-name",
		secretEditOrigKey: "old-name",
		secretEditColumn:  0,
		client:            newTestClient("secret"),
	}

	result, cmd := m.handleSecretEditKey(tea.KeyMsg{Type: tea.KeyEnter})
	resultModel := result.(*Model)

	assert.Equal(t, model.ModeSecret, resultModel.mode)
	// applyCurrentEditInput renames the key
	assert.NotNil(t, cmd, "should return cmd to rename key on server")
}

func TestHandleSecretEditKey_Enter_NewKeyAdded(t *testing.T) {
	ti := textinput.New()
	ti.SetValue("brand-new")
	m := &Model{
		mode:              model.ModeSecretEdit,
		keys:              DefaultKeyMap(),
		textInput:         ti,
		secret:            &model.Secret{Path: "s", Keys: []string{"existing", "brand-new"}, Data: map[string]string{"existing": "val", "brand-new": "new-val"}},
		secretEditKey:     "brand-new",
		secretEditOrigKey: "", // was a new key
		secretEditColumn:  0,
		client:            newTestClient("secret"),
	}

	result, cmd := m.handleSecretEditKey(tea.KeyMsg{Type: tea.KeyEnter})
	resultModel := result.(*Model)

	assert.Equal(t, model.ModeSecret, resultModel.mode)
	assert.NotNil(t, cmd, "should return cmd to add new key")
}

// ---------------------------------------------------------------------------
// Update - editorResultMsg with existing secret
// ---------------------------------------------------------------------------

func TestUpdate_EditorResultMsg_ExistingSecret(t *testing.T) {
	m := newTestModel()
	m.secret = &model.Secret{
		Path: "secret/foo",
		Keys: []string{"a", "b"},
		Data: map[string]string{"a": "1", "b": "2"},
	}
	m.secretCursor = 1

	// Editor returns modified data with removed key and new key
	result, cmd := m.Update(editorResultMsg{
		data: map[string]string{"a": "updated", "c": "new"},
	})
	resultModel := result.(*Model)

	assert.Equal(t, map[string]string{"a": "updated", "c": "new"}, resultModel.secret.Data)
	// Keys should preserve existing order for surviving keys, then add new sorted
	assert.Equal(t, []string{"a", "c"}, resultModel.secret.Keys)
	assert.NotNil(t, cmd, "should return cmd to save secret")
}

func TestUpdate_EditorResultMsg_ExistingSecret_CursorClamp(t *testing.T) {
	m := newTestModel()
	m.secret = &model.Secret{
		Path: "secret/foo",
		Keys: []string{"a", "b", "c"},
		Data: map[string]string{"a": "1", "b": "2", "c": "3"},
	}
	m.secretCursor = 2

	// Editor returns only one key
	result, _ := m.Update(editorResultMsg{
		data: map[string]string{"a": "1"},
	})
	resultModel := result.(*Model)

	assert.Equal(t, 0, resultModel.secretCursor, "cursor should clamp to last key index")
}

func TestUpdate_EditorResultMsg_NewSecret(t *testing.T) {
	m := newTestModel()

	result, cmd := m.Update(editorResultMsg{
		data:          map[string]string{"key": "val"},
		newSecretPath: "secret/brand-new",
	})
	assert.Equal(t, m, result)
	assert.NotNil(t, cmd, "should return cmd to write new secret")
}

// ---------------------------------------------------------------------------
// handleKey - SecretMode dispatch (via ModeSecret)
// ---------------------------------------------------------------------------

func TestHandleKey_SecretMode(t *testing.T) {
	// "q" in secret mode should close the popup
	m := newTestModel()
	m.mode = model.ModeSecret
	m.secret = &model.Secret{Path: "s", Keys: []string{"k"}, Data: map[string]string{"k": "v"}}

	result, _ := m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	resultModel := result.(*Model)
	assert.Equal(t, model.ModeExplorer, resultModel.mode)
}

// ---------------------------------------------------------------------------
// handleKey - Search mode dispatch
// ---------------------------------------------------------------------------

func TestHandleKey_SearchMode_Esc(t *testing.T) {
	m := newTestModel()
	m.mode = model.ModeSearch
	m.priorCursor = 3
	m.searchInput.Focus()

	result, _ := m.handleKey(tea.KeyMsg{Type: tea.KeyEsc})
	resultModel := result.(*Model)

	assert.Equal(t, model.ModeExplorer, resultModel.mode)
	assert.Equal(t, 3, resultModel.cursor)
}

// ---------------------------------------------------------------------------
// handleKey - Filter mode dispatch
// ---------------------------------------------------------------------------

func TestHandleKey_FilterMode_Esc(t *testing.T) {
	m := newTestModel()
	m.mode = model.ModeFilter
	m.filterQuery = "test"
	m.priorCursor = 2
	m.searchInput.Focus()

	result, _ := m.handleKey(tea.KeyMsg{Type: tea.KeyEsc})
	resultModel := result.(*Model)

	assert.Equal(t, model.ModeExplorer, resultModel.mode)
	assert.Equal(t, 2, resultModel.cursor)
	assert.Empty(t, resultModel.filterQuery)
}

// ---------------------------------------------------------------------------
// handleKey - JumpPath mode dispatch
// ---------------------------------------------------------------------------

func TestHandleKey_JumpPathMode_Esc(t *testing.T) {
	ti := textinput.New()
	ti.Focus()
	m := &Model{
		mode:            model.ModeJumpPath,
		textInput:       ti,
		searchInput:     textinput.New(),
		confirmInput:    textinput.New(),
		keys:            DefaultKeyMap(),
		jumpCompletions: []string{"x"},
		jumpCompIdx:     0,
	}

	result, _ := m.handleKey(tea.KeyMsg{Type: tea.KeyEsc})
	resultModel := result.(*Model)

	assert.Equal(t, model.ModeExplorer, resultModel.mode)
}

// ---------------------------------------------------------------------------
// handleKey - Bookmark mode dispatch
// ---------------------------------------------------------------------------

func TestHandleKey_BookmarkMode(t *testing.T) {
	si := textinput.New()
	m := &Model{
		mode:         model.ModeBookmark,
		keys:         DefaultKeyMap(),
		searchInput:  si,
		textInput:    textinput.New(),
		confirmInput: textinput.New(),
		bookmarks:    []config.Bookmark{{Name: "test", Mount: "kv", Path: "a"}},
	}

	result, _ := m.handleKey(tea.KeyMsg{Type: tea.KeyEsc})
	resultModel := result.(*Model)
	assert.Equal(t, model.ModeExplorer, resultModel.mode)
}

// ---------------------------------------------------------------------------
// handleKey - ThemePicker mode dispatch
// ---------------------------------------------------------------------------

func TestHandleKey_ThemePickerMode(t *testing.T) {
	m := newTestModel()
	m.mode = model.ModeThemePicker
	m.themeEntries = []ui.ThemeEntry{{Name: "test", IsHeader: false}}
	m.themeCursor = 0

	result, _ := m.handleKey(tea.KeyMsg{Type: tea.KeyEsc})
	resultModel := result.(*Model)
	assert.Equal(t, model.ModeExplorer, resultModel.mode)
}

// ---------------------------------------------------------------------------
// handleBookmarkOverlayKey
// ---------------------------------------------------------------------------

func TestHandleBookmarkOverlayKey_Esc_WithFilter(t *testing.T) {
	m := &Model{
		mode:           model.ModeBookmark,
		bookmarks:      []config.Bookmark{{Name: "test", Mount: "kv", Path: "a"}},
		bookmarkFilter: "te",
		bookmarkCursor: 0,
	}
	result, _ := m.handleBookmarkOverlayKey(tea.KeyMsg{Type: tea.KeyEsc})
	resultModel := result.(*Model)

	assert.Equal(t, model.ModeBookmark, resultModel.mode, "first esc clears filter, stays in bookmark mode")
	assert.Empty(t, resultModel.bookmarkFilter)
	assert.Equal(t, 0, resultModel.bookmarkCursor)
}

func TestHandleBookmarkOverlayKey_Esc_NoFilter(t *testing.T) {
	m := &Model{
		mode:           model.ModeBookmark,
		bookmarks:      []config.Bookmark{{Name: "test", Mount: "kv", Path: "a"}},
		bookmarkFilter: "",
	}
	result, _ := m.handleBookmarkOverlayKey(tea.KeyMsg{Type: tea.KeyEsc})
	assert.Equal(t, model.ModeExplorer, result.(*Model).mode)
}

func TestHandleBookmarkOverlayKey_SlashStartsSearch(t *testing.T) {
	m := &Model{
		mode:      model.ModeBookmark,
		bookmarks: []config.Bookmark{{Name: "test", Mount: "kv", Path: "a"}},
	}
	m.handleBookmarkOverlayKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	assert.True(t, m.bookmarkSearching)
}

func TestHandleBookmarkOverlayKey_CursorDown(t *testing.T) {
	m := &Model{
		mode: model.ModeBookmark,
		bookmarks: []config.Bookmark{
			{Name: "a", Mount: "kv", Path: "a"},
			{Name: "b", Mount: "kv", Path: "b"},
		},
		bookmarkCursor: 0,
	}
	m.handleBookmarkOverlayKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	assert.Equal(t, 1, m.bookmarkCursor)
}

func TestHandleBookmarkOverlayKey_CursorUp(t *testing.T) {
	m := &Model{
		mode: model.ModeBookmark,
		bookmarks: []config.Bookmark{
			{Name: "a", Mount: "kv", Path: "a"},
			{Name: "b", Mount: "kv", Path: "b"},
		},
		bookmarkCursor: 1,
	}
	m.handleBookmarkOverlayKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	assert.Equal(t, 0, m.bookmarkCursor)
}

func TestHandleBookmarkOverlayKey_CursorDown_AtEnd(t *testing.T) {
	m := &Model{
		mode:           model.ModeBookmark,
		bookmarks:      []config.Bookmark{{Name: "a", Mount: "kv", Path: "a"}},
		bookmarkCursor: 0,
	}
	m.handleBookmarkOverlayKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	assert.Equal(t, 0, m.bookmarkCursor, "should not go beyond last entry")
}

func TestHandleBookmarkOverlayKey_CursorUp_AtTop(t *testing.T) {
	m := &Model{
		mode:           model.ModeBookmark,
		bookmarks:      []config.Bookmark{{Name: "a", Mount: "kv", Path: "a"}},
		bookmarkCursor: 0,
	}
	m.handleBookmarkOverlayKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	assert.Equal(t, 0, m.bookmarkCursor, "should not go below 0")
}

func TestHandleBookmarkOverlayKey_CtrlX_ClearsAll(t *testing.T) {
	m := &Model{
		mode: model.ModeBookmark,
		bookmarks: []config.Bookmark{
			{Name: "a", Mount: "kv", Path: "a"},
			{Name: "b", Mount: "kv", Path: "b"},
		},
	}
	result, _ := m.handleBookmarkOverlayKey(tea.KeyMsg{Type: tea.KeyCtrlX})
	resultModel := result.(*Model)

	assert.Equal(t, model.ModeExplorer, resultModel.mode)
	assert.Nil(t, resultModel.bookmarks)
	assert.Equal(t, "All marks deleted", resultModel.status)
}

func TestHandleBookmarkOverlayKey_QuickJump_NotSet(t *testing.T) {
	m := &Model{
		mode:      model.ModeBookmark,
		bookmarks: []config.Bookmark{{Name: "a", Mount: "kv", Path: "a", Slot: "a"}},
	}
	result, _ := m.handleBookmarkOverlayKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'z'}})
	resultModel := result.(*Model)

	assert.Equal(t, model.ModeExplorer, resultModel.mode)
	assert.Contains(t, resultModel.status, "Mark 'z' not set")
}

func TestHandleBookmarkOverlayKey_Delete(t *testing.T) {
	m := &Model{
		mode: model.ModeBookmark,
		bookmarks: []config.Bookmark{
			{Name: "first", Mount: "kv", Path: "a"},
			{Name: "second", Mount: "kv", Path: "b"},
		},
		bookmarkCursor: 0,
	}
	m.handleBookmarkOverlayKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'D'}})

	assert.Len(t, m.bookmarks, 1)
	assert.Equal(t, "second", m.bookmarks[0].Name)
}

func TestHandleBookmarkOverlayKey_Search_Esc(t *testing.T) {
	m := &Model{
		mode:              model.ModeBookmark,
		bookmarks:         []config.Bookmark{{Name: "a", Mount: "kv", Path: "a"}},
		bookmarkSearching: true,
		bookmarkFilter:    "old",
	}
	m.handleBookmarkOverlayKey(tea.KeyMsg{Type: tea.KeyEsc})

	assert.False(t, m.bookmarkSearching)
	assert.Empty(t, m.bookmarkFilter)
}

func TestHandleBookmarkOverlayKey_Search_Enter(t *testing.T) {
	m := &Model{
		mode:              model.ModeBookmark,
		bookmarks:         []config.Bookmark{{Name: "a", Mount: "kv", Path: "a"}},
		bookmarkSearching: true,
		bookmarkFilter:    "test",
	}
	m.handleBookmarkOverlayKey(tea.KeyMsg{Type: tea.KeyEnter})

	assert.False(t, m.bookmarkSearching)
	assert.Equal(t, "test", m.bookmarkFilter, "filter should be preserved")
}

func TestHandleBookmarkOverlayKey_Search_Backspace(t *testing.T) {
	m := &Model{
		mode:              model.ModeBookmark,
		bookmarks:         []config.Bookmark{{Name: "a", Mount: "kv", Path: "a"}},
		bookmarkSearching: true,
		bookmarkFilter:    "test",
	}
	m.handleBookmarkOverlayKey(tea.KeyMsg{Type: tea.KeyBackspace})

	assert.Equal(t, "tes", m.bookmarkFilter)
}

func TestHandleBookmarkOverlayKey_Search_Typing(t *testing.T) {
	m := &Model{
		mode:              model.ModeBookmark,
		bookmarks:         []config.Bookmark{{Name: "a", Mount: "kv", Path: "a"}},
		bookmarkSearching: true,
		bookmarkFilter:    "te",
	}
	m.handleBookmarkOverlayKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})

	assert.Equal(t, "tes", m.bookmarkFilter)
}

// ---------------------------------------------------------------------------
// handleThemePickerKey
// ---------------------------------------------------------------------------

func TestHandleThemePickerKey_Esc(t *testing.T) {
	m := newTestModel()
	m.mode = model.ModeThemePicker
	m.themeEntries = []ui.ThemeEntry{{Name: "dracula", IsHeader: false}}
	m.themeCursor = 0
	m.activeColorscheme = "dracula"

	result, _ := m.handleThemePickerKey(tea.KeyMsg{Type: tea.KeyEsc})
	assert.Equal(t, model.ModeExplorer, result.(*Model).mode)
}

func TestHandleThemePickerKey_Q(t *testing.T) {
	m := newTestModel()
	m.mode = model.ModeThemePicker
	m.themeEntries = []ui.ThemeEntry{{Name: "dracula", IsHeader: false}}
	m.themeCursor = 0
	m.activeColorscheme = "dracula"

	result, _ := m.handleThemePickerKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	assert.Equal(t, model.ModeExplorer, result.(*Model).mode)
}

func TestHandleThemePickerKey_Enter(t *testing.T) {
	m := newTestModel()
	m.mode = model.ModeThemePicker
	m.themeEntries = []ui.ThemeEntry{
		{Name: "Header", IsHeader: true},
		{Name: "dracula", IsHeader: false},
	}
	m.themeCursor = 1
	m.activeColorscheme = "monokai"

	result, _ := m.handleThemePickerKey(tea.KeyMsg{Type: tea.KeyEnter})
	resultModel := result.(*Model)

	assert.Equal(t, model.ModeExplorer, resultModel.mode)
	assert.Equal(t, "dracula", resultModel.activeColorscheme)
	assert.Contains(t, resultModel.status, "dracula")
}

func TestHandleThemePickerKey_CursorDown(t *testing.T) {
	m := newTestModel()
	m.mode = model.ModeThemePicker
	m.themeEntries = []ui.ThemeEntry{
		{Name: "dracula", IsHeader: false},
		{Name: "monokai", IsHeader: false},
	}
	m.themeCursor = 0
	m.activeColorscheme = "dracula"

	m.handleThemePickerKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	assert.Equal(t, 1, m.themeCursor)
}

func TestHandleThemePickerKey_CursorUp(t *testing.T) {
	m := newTestModel()
	m.mode = model.ModeThemePicker
	m.themeEntries = []ui.ThemeEntry{
		{Name: "dracula", IsHeader: false},
		{Name: "monokai", IsHeader: false},
	}
	m.themeCursor = 1
	m.activeColorscheme = "monokai"

	m.handleThemePickerKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	assert.Equal(t, 0, m.themeCursor)
}

func TestHandleThemePickerKey_UnknownKey(t *testing.T) {
	m := newTestModel()
	m.mode = model.ModeThemePicker
	m.themeEntries = []ui.ThemeEntry{{Name: "dracula", IsHeader: false}}
	m.themeCursor = 0

	result, cmd := m.handleThemePickerKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	assert.Equal(t, m, result)
	assert.Nil(t, cmd)
}

// ---------------------------------------------------------------------------
// handleSecretKey - navigation and toggle
// ---------------------------------------------------------------------------

func TestHandleSecretKey_CursorDown(t *testing.T) {
	m := &Model{
		mode: model.ModeSecret,
		secret: &model.Secret{
			Keys: []string{"a", "b", "c"},
			Data: map[string]string{"a": "1", "b": "2", "c": "3"},
		},
		secretCursor: 0,
		height:       40,
	}
	m.handleSecretKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	assert.Equal(t, 1, m.secretCursor)
}

func TestHandleSecretKey_CursorUp(t *testing.T) {
	m := &Model{
		mode: model.ModeSecret,
		secret: &model.Secret{
			Keys: []string{"a", "b", "c"},
			Data: map[string]string{"a": "1", "b": "2", "c": "3"},
		},
		secretCursor: 2,
		height:       40,
	}
	m.handleSecretKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	assert.Equal(t, 1, m.secretCursor)
}

func TestHandleSecretKey_ToggleReveal(t *testing.T) {
	m := &Model{
		mode: model.ModeSecret,
		secret: &model.Secret{
			Keys: []string{"a", "b"},
			Data: map[string]string{"a": "1", "b": "2"},
		},
		revealed:          make(map[string]bool),
		secretAllRevealed: false,
		height:            40,
	}
	m.handleSecretKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	assert.True(t, m.secretAllRevealed)
	assert.True(t, m.revealed["a"])
	assert.True(t, m.revealed["b"])

	// Toggle back
	m.handleSecretKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	assert.False(t, m.secretAllRevealed)
}

func TestHandleSecretKey_ToggleJSON(t *testing.T) {
	m := &Model{
		mode: model.ModeSecret,
		secret: &model.Secret{
			Keys: []string{"a"},
			Data: map[string]string{"a": "1"},
		},
		secretJSONView: false,
		height:         40,
	}
	m.handleSecretKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'V'}})
	assert.True(t, m.secretJSONView)
}

func TestHandleSecretKey_CopyFormatPending(t *testing.T) {
	m := &Model{
		mode: model.ModeSecret,
		secret: &model.Secret{
			Keys: []string{"a"},
			Data: map[string]string{"a": "1"},
		},
		height: 40,
	}
	// Press Y to enter copy format mode
	m.handleSecretKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'Y'}})
	assert.True(t, m.copyFormatPending)
	assert.Contains(t, m.status, "Copy as:")

	// Now pressing 'y' (yaml) should go through handleCopyFormat
	_, cmd := m.handleSecretKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	assert.False(t, m.copyFormatPending)
	assert.NotNil(t, cmd)
}

func TestHandleSecretKey_HalfPageDown(t *testing.T) {
	keys := make([]string, 50)
	data := make(map[string]string, 50)
	for i := range 50 {
		k := "key" + string(rune('a'+i%26))
		keys[i] = k
		data[k] = "val"
	}
	m := &Model{
		mode:         model.ModeSecret,
		secret:       &model.Secret{Keys: keys, Data: data},
		secretCursor: 0,
		height:       40,
	}
	m.handleSecretKey(tea.KeyMsg{Type: tea.KeyCtrlD})
	assert.Greater(t, m.secretCursor, 0)
}

func TestHandleSecretKey_HalfPageUp(t *testing.T) {
	keys := make([]string, 50)
	data := make(map[string]string, 50)
	for i := range 50 {
		k := "key" + string(rune('a'+i%26))
		keys[i] = k
		data[k] = "val"
	}
	m := &Model{
		mode:         model.ModeSecret,
		secret:       &model.Secret{Keys: keys, Data: data},
		secretCursor: 30,
		height:       40,
	}
	m.handleSecretKey(tea.KeyMsg{Type: tea.KeyCtrlU})
	assert.Less(t, m.secretCursor, 30)
}

func TestHandleSecretKey_FullPageDown(t *testing.T) {
	keys := make([]string, 50)
	data := make(map[string]string, 50)
	for i := range 50 {
		k := "key" + string(rune('a'+i%26))
		keys[i] = k
		data[k] = "val"
	}
	m := &Model{
		mode:         model.ModeSecret,
		secret:       &model.Secret{Keys: keys, Data: data},
		secretCursor: 0,
		height:       40,
	}
	m.handleSecretKey(tea.KeyMsg{Type: tea.KeyCtrlF})
	assert.Greater(t, m.secretCursor, 0)
}

func TestHandleSecretKey_FullPageUp(t *testing.T) {
	keys := make([]string, 50)
	data := make(map[string]string, 50)
	for i := range 50 {
		k := "key" + string(rune('a'+i%26))
		keys[i] = k
		data[k] = "val"
	}
	m := &Model{
		mode:         model.ModeSecret,
		secret:       &model.Secret{Keys: keys, Data: data},
		secretCursor: 40,
		height:       40,
	}
	m.handleSecretKey(tea.KeyMsg{Type: tea.KeyCtrlB})
	assert.Less(t, m.secretCursor, 40)
}

func TestHandleSecretKey_Base64Toggle(t *testing.T) {
	m := &Model{
		mode: model.ModeSecret,
		secret: &model.Secret{
			Keys: []string{"encoded"},
			Data: map[string]string{"encoded": "aGVsbG8="}, // base64 of "hello"
		},
		secretCursor: 0,
		secretBase64: make(map[string]bool),
		height:       40,
	}
	m.handleSecretKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
	assert.True(t, m.secretBase64["encoded"])

	// Toggle off
	m.handleSecretKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
	assert.False(t, m.secretBase64["encoded"])
}

func TestHandleSecretKey_Base64Toggle_InvalidBase64(t *testing.T) {
	m := &Model{
		mode: model.ModeSecret,
		secret: &model.Secret{
			Keys: []string{"notb64"},
			Data: map[string]string{"notb64": "not valid base64!!!"},
		},
		secretCursor: 0,
		secretBase64: make(map[string]bool),
		height:       40,
	}
	m.handleSecretKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
	assert.Equal(t, "Not valid base64", m.errMsg)
	assert.False(t, m.secretBase64["notb64"])
}

func TestHandleSecretKey_NilSecret_NoOp(t *testing.T) {
	m := &Model{
		mode:   model.ModeSecret,
		secret: nil,
		height: 40,
	}
	result, cmd := m.handleSecretKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	assert.Equal(t, m, result)
	assert.Nil(t, cmd)
}
