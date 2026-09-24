package app

import (
	"testing"

	"github.com/janosmiko/vau/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEditorExecCommand_SingleWord(t *testing.T) {
	cmd := editorExecCommand("vim", "/tmp/foo.json")
	assert.Equal(t, []string{"vim", "/tmp/foo.json"}, cmd.Args)
}

func TestEditorExecCommand_WithArguments(t *testing.T) {
	cmd := editorExecCommand("code --wait", "/tmp/foo.json")
	assert.Equal(t, []string{"code", "--wait", "/tmp/foo.json"}, cmd.Args)
}

func TestEditorExecCommand_ExtraArgsBeforeTrailingArgs(t *testing.T) {
	cmd := editorExecCommand("nvim -u NONE", "a.json", "b.json")
	assert.Equal(t, []string{"nvim", "-u", "NONE", "a.json", "b.json"}, cmd.Args)
}

// editorCommand must validate only the executable, not the whole configured
// string, so a value like "sh -c true" is accepted.
func TestEditorCommand_LooksUpFirstWordOnly(t *testing.T) {
	m := &Model{config: &config.Config{Editor: "sh -c true"}}
	editor, err := m.editorCommand()
	require.NoError(t, err)
	assert.Equal(t, "sh -c true", editor)
}
