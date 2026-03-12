package app

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultKeyMapAllFieldsNonEmpty(t *testing.T) {
	km := DefaultKeyMap()
	v := reflect.ValueOf(km)
	typ := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		name := typ.Field(i).Name
		require.IsType(t, []string{}, field.Interface(), "field %s should be []string", name)
		bindings := field.Interface().([]string)
		assert.NotEmpty(t, bindings, "DefaultKeyMap().%s should not be empty", name)
	}
}

func TestMatchKey(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		bindings []string
		want     bool
	}{
		{
			name:     "matching key",
			key:      "j",
			bindings: []string{"j", "down", "ctrl+n"},
			want:     true,
		},
		{
			name:     "matching last binding",
			key:      "ctrl+n",
			bindings: []string{"j", "down", "ctrl+n"},
			want:     true,
		},
		{
			name:     "non-matching key",
			key:      "x",
			bindings: []string{"j", "down", "ctrl+n"},
			want:     false,
		},
		{
			name:     "empty bindings",
			key:      "j",
			bindings: []string{},
			want:     false,
		},
		{
			name:     "nil bindings",
			key:      "j",
			bindings: nil,
			want:     false,
		},
		{
			name:     "empty key against non-empty bindings",
			key:      "",
			bindings: []string{"j", "k"},
			want:     false,
		},
		{
			name:     "single binding match",
			key:      "q",
			bindings: []string{"q"},
			want:     true,
		},
		{
			name:     "single binding no match",
			key:      "Q",
			bindings: []string{"q"},
			want:     false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := matchKey(tc.key, tc.bindings)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestApplyOverrides(t *testing.T) {
	t.Run("override replaces defaults", func(t *testing.T) {
		km := DefaultKeyMap()
		km.ApplyOverrides(map[string]string{
			"quit": "Q",
			"help": "F1",
		})
		assert.Equal(t, []string{"Q"}, km.Quit)
		assert.Equal(t, []string{"F1"}, km.Help)
	})

	t.Run("unknown actions are ignored", func(t *testing.T) {
		km := DefaultKeyMap()
		original := km // copy before override
		km.ApplyOverrides(map[string]string{
			"nonexistent_action": "x",
		})
		// The known fields should remain unchanged.
		assert.Equal(t, original.Quit, km.Quit)
		assert.Equal(t, original.Help, km.Help)
		assert.Equal(t, original.Up, km.Up)
	})

	t.Run("empty overrides is no-op", func(t *testing.T) {
		km := DefaultKeyMap()
		original := km
		km.ApplyOverrides(map[string]string{})
		assert.Equal(t, original, km)
	})

	t.Run("nil overrides is no-op", func(t *testing.T) {
		km := DefaultKeyMap()
		original := km
		km.ApplyOverrides(nil)
		assert.Equal(t, original, km)
	})

	t.Run("override only affects targeted action", func(t *testing.T) {
		km := DefaultKeyMap()
		originalUp := make([]string, len(km.Up))
		copy(originalUp, km.Up)

		km.ApplyOverrides(map[string]string{
			"quit": "Q",
		})
		assert.Equal(t, []string{"Q"}, km.Quit)
		assert.Equal(t, originalUp, km.Up, "non-overridden field should be unchanged")
	})
}

func TestActionMapAllExpectedActions(t *testing.T) {
	km := DefaultKeyMap()
	am := km.actionMap()

	expectedActions := []string{
		"navigate_up", "navigate_down", "navigate_left", "navigate_right",
		"top", "bottom", "half_down", "half_up", "full_down", "full_up", "open",
		"search", "filter", "jump_path",
		"new_secret", "new_secret_editor", "edit", "rename", "delete",
		"yank", "paste", "cut", "undo", "redo",
		"refresh", "toggle_values", "toggle_json", "select",
		"help", "quit",
		"new_tab", "next_tab", "prev_tab", "close_tab",
		"bookmark_save", "bookmark_show",
		"theme_picker",
	}

	for _, action := range expectedActions {
		_, ok := am[action]
		assert.True(t, ok, "actionMap should contain %q", action)
	}

	// Verify actionMap has exactly the expected number of entries.
	assert.Len(t, am, len(expectedActions), "actionMap should have exactly %d entries", len(expectedActions))
}

func TestActionMapPointersAreValid(t *testing.T) {
	km := DefaultKeyMap()
	am := km.actionMap()

	for action, ptr := range am {
		require.NotNil(t, ptr, "pointer for action %q should not be nil", action)
		assert.NotEmpty(t, *ptr, "bindings for action %q should not be empty", action)
	}
}
