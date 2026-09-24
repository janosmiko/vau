package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janosmiko/vau/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubEditor writes a script under a directory whose name has a space. The
// script records each argument it receives on its own line.
func stubEditor(t *testing.T) (script, out string) {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "my editor")
	require.NoError(t, os.MkdirAll(dir, 0o750))
	out = filepath.Join(dir, "args.txt")
	script = filepath.Join(dir, "rec.sh")
	body := "#!/bin/sh\nfor a in \"$@\"; do printf '%s\\n' \"$a\"; done > '" + out + "'\n"
	require.NoError(t, os.WriteFile(script, []byte(body), 0o700)) //nolint:gosec // test script must be executable
	return script, out
}

func runEditor(t *testing.T, editor string, trailingArgs ...string) {
	t.Helper()
	require.NoError(t, editorExecCommand(editor, trailingArgs...).Run())
}

func recordedArgs(t *testing.T, out string) []string {
	t.Helper()
	data, err := os.ReadFile(out) //nolint:gosec // path is from t.TempDir
	require.NoError(t, err)
	return strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")
}

func TestEditorExecCommand_PassesEditorArgumentsBeforeFile(t *testing.T) {
	script, out := stubEditor(t)
	runEditor(t, "'"+script+"' --wait", "/tmp/foo.json")
	assert.Equal(t, []string{"--wait", "/tmp/foo.json"}, recordedArgs(t, out))
}

func TestEditorExecCommand_KeepsQuotedEmptyArgument(t *testing.T) {
	script, out := stubEditor(t)
	runEditor(t, "'"+script+"' -a \"\"", "f.json")
	assert.Equal(t, []string{"-a", "", "f.json"}, recordedArgs(t, out))
}

func TestEditorExecCommand_FileIsNotParsedByShell(t *testing.T) {
	script, out := stubEditor(t)
	file := "/tmp/a b; touch pwned.json"
	runEditor(t, "'"+script+"'", file)
	assert.Equal(t, []string{file}, recordedArgs(t, out))
}

// editorCommand must validate only the executable, not the whole configured
// string, so a value like "sh -c true" is accepted.
func TestEditorCommand_LooksUpFirstWordOnly(t *testing.T) {
	m := &Model{config: &config.Config{Editor: "sh -c true"}}
	editor, err := m.editorCommand()
	require.NoError(t, err)
	assert.Equal(t, "sh -c true", editor)
}

func TestEditorCommand_AcceptsQuotedPathWithSpaces(t *testing.T) {
	script, _ := stubEditor(t)
	m := &Model{config: &config.Config{Editor: "'" + script + "' --wait"}}
	_, err := m.editorCommand()
	require.NoError(t, err)
}

func TestEditorCommand_RejectsMissingBinary(t *testing.T) {
	m := &Model{config: &config.Config{Editor: "no-such-editor-xyz --wait"}}
	_, err := m.editorCommand()
	require.Error(t, err)
}
