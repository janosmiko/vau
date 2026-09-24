package app

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/janosmiko/vau/internal/model"
	"github.com/janosmiko/vau/internal/vault"
)

func (m *Model) yankSecret(path string) tea.Cmd {
	isCut := m.yankIsCut
	client := m.client
	return func() tea.Msg {
		secret, err := client.Read(path)
		if err != nil {
			return errorMsg(fmt.Sprintf("yank failed: %v", err))
		}
		return yankResultMsg{secrets: []*model.Secret{secret}, paths: []string{path}, isCut: isCut}
	}
}

func (m *Model) bulkYankSecrets(entries []model.Entry, isCut bool) tea.Cmd {
	basePath := m.currentPath()
	client := m.client
	return func() tea.Msg {
		var secrets []*model.Secret
		var paths []string
		hasDir := false
		for _, entry := range entries {
			path := basePath + strings.TrimSuffix(entry.Name, "/")
			paths = append(paths, path)
			if entry.IsDir {
				hasDir = true
				continue
			}
			secret, err := client.Read(basePath + entry.Name)
			if err != nil {
				return errorMsg(fmt.Sprintf("yank failed for %s: %v", entry.Name, err))
			}
			secrets = append(secrets, secret)
		}
		return yankResultMsg{secrets: secrets, paths: paths, isCut: isCut, isDir: hasDir}
	}
}

func (m *Model) deleteEntry(entry model.Entry) tea.Cmd {
	path := m.currentPath() + strings.TrimSuffix(entry.Name, "/")
	client := m.client

	if entry.IsDir {
		operation := "Deleting"

		ctx, cancel := context.WithCancel(context.Background()) //nolint:gosec // cancel stored in m.progress.cancel
		m.progress = progressState{
			active:    true,
			cancel:    cancel,
			operation: operation,
			current:   0,
			total:     0, // unknown until counted inside the cmd below
		}

		var reporter *progressReporter
		if m.program != nil {
			reporter = newProgressReporter(m.program)
		}

		return func() tea.Msg {
			if err := ctx.Err(); err != nil {
				return progressDoneMsg{operation: operation, err: err}
			}
			total, _ := client.CountRecursive(path)
			if reporter != nil {
				reporter.report(0, total)
			}

			counter := 0
			var cb vault.ProgressCallback
			if reporter != nil {
				cb = func(current, _ int) {
					reporter.report(current, total)
				}
			}
			count, err := client.DeleteRecursiveCtx(ctx, path, cb, &counter)
			if err != nil {
				return progressDoneMsg{operation: operation, count: count, err: err}
			}
			return progressDoneMsg{
				operation: operation,
				count:     count,
				status:    fmt.Sprintf("Deleted %d secrets in %s (cannot undo)", count, entry.Name),
			}
		}
	}

	return func() tea.Msg {
		// Read secret data before deleting (for undo)
		secret, _ := client.Read(path)
		if err := client.Delete(path); err != nil {
			return errorMsg(err.Error())
		}
		var data map[string]string
		var keys []string
		if secret != nil {
			data = copyMap(secret.Data)
			keys = copySlice(secret.Keys)
		}
		return undoableStatusMsg{
			status: "Deleted: " + entry.Name,
			undo: model.UndoAction{
				Type:        model.UndoDeleteSecret,
				Description: "delete " + entry.Name,
				Path:        path,
				Data:        data,
				Keys:        keys,
				Mount:       client.Mount(),
			},
		}
	}
}

func (m *Model) renameEntry(newName string) tea.Cmd {
	entry := m.selectedEntry()
	base := m.currentPath()
	client := m.client
	return func() tea.Msg {
		if entry == nil {
			return errorMsg("no entry selected")
		}
		if entry.IsDir {
			src := base + strings.TrimSuffix(entry.Name, "/")
			dst := base + strings.TrimSuffix(newName, "/")
			count, err := client.MoveRecursive(src, dst)
			if err != nil {
				return errorMsg(err.Error())
			}
			return undoableStatusMsg{
				status: fmt.Sprintf("Renamed directory %s → %s (%d secrets)", entry.Name, newName, count),
				undo: model.UndoAction{
					Type:        model.UndoRenameSecret,
					Description: fmt.Sprintf("rename dir %s → %s", entry.Name, newName),
					Path:        dst,
					OldPath:     src,
					Mount:       client.Mount(),
				},
			}
		}
		src := base + entry.Name
		dst := base + newName
		if err := client.Move(src, dst); err != nil {
			return errorMsg(err.Error())
		}
		return undoableStatusMsg{
			status: fmt.Sprintf("Renamed %s → %s", entry.Name, newName),
			undo: model.UndoAction{
				Type:        model.UndoRenameSecret,
				Description: fmt.Sprintf("rename %s → %s", entry.Name, newName),
				Path:        dst,
				OldPath:     src,
				Mount:       client.Mount(),
			},
		}
	}
}

func (m *Model) createSecretWithCheck(path string) tea.Cmd {
	client := m.client
	create := m.createEmptySecretAndOpen(path)
	return func() tea.Msg {
		// Check if secret already exists
		if _, err := client.Read(path); err == nil {
			return confirmCreateMsg(path)
		}
		// Create an empty secret and open the popup for inline editing
		return create()
	}
}

// createEmptySecretAndOpen writes an empty secret to Vault and returns a
// secretResultMsg that opens the popup with inline editing ready.
func (m *Model) createEmptySecretAndOpen(path string) tea.Cmd {
	client := m.client
	return func() tea.Msg {
		emptyData := map[string]string{}
		if err := client.Write(path, emptyData); err != nil {
			return errorMsg(fmt.Sprintf("Failed to create secret: %v", err))
		}
		secret := &model.Secret{
			Path: path,
			Data: map[string]string{"": ""},
			Keys: []string{""},
		}
		return newSecretInlineMsg{secret: secret}
	}
}

// createSecretWithEditor checks if a secret exists, and either prompts for
// overwrite or directly opens the external editor for a new secret.
func (m *Model) createSecretWithEditor(path string) tea.Cmd {
	client := m.client
	return func() tea.Msg {
		if _, err := client.Read(path); err == nil {
			return confirmCreateEditorMsg(path)
		}
		return newSecretEditorMsg(path)
	}
}

// countYankTotal counts the items under yankPaths (best-effort) for progress
// display, doubling for a cut since it involves a copy phase and a delete phase.
func countYankTotal(client *vault.Client, yankPaths []string, isCut bool) int {
	total := 0
	for _, yp := range yankPaths {
		if n, err := client.CountRecursive(yp); err == nil {
			total += n
		}
	}
	if isCut {
		total *= 2
	}
	return total
}

func (m *Model) pasteSecrets() tea.Cmd {
	isCut := m.yankIsCut
	isDir := m.yankIsDir
	yankPaths := make([]string, len(m.yankPaths))
	copy(yankPaths, m.yankPaths)
	yankedSecrets := make([]*model.Secret, len(m.yankedSecrets))
	copy(yankedSecrets, m.yankedSecrets)
	basePath := m.currentPath()
	client := m.client

	// Directory operations use progress tracking.
	if isDir && len(yankPaths) > 0 {
		operation := "Copying"
		if isCut {
			operation = "Moving"
		}

		ctx, cancel := context.WithCancel(context.Background()) //nolint:gosec // cancel stored in m.progress.cancel
		m.progress = progressState{
			active:    true,
			cancel:    cancel,
			operation: operation,
			current:   0,
			total:     0, // unknown until counted inside the cmd below
		}

		var reporter *progressReporter
		if m.program != nil {
			reporter = newProgressReporter(m.program)
		}

		return func() tea.Msg {
			if err := ctx.Err(); err != nil {
				return progressDoneMsg{operation: operation, err: err}
			}

			total := countYankTotal(client, yankPaths, isCut)
			if reporter != nil {
				reporter.report(0, total)
			}

			totalCount := 0
			counter := 0
			for _, yp := range yankPaths {
				parts := strings.Split(strings.TrimSuffix(yp, "/"), "/")
				name := parts[len(parts)-1]
				dst := basePath + name

				// If destination directory already exists, append _1, _2, etc.
				if entries, err := client.List(dst); err == nil && entries != nil {
					for i := 1; ; i++ {
						candidate := basePath + fmt.Sprintf("%s_%d", name, i)
						if entries, err := client.List(candidate); err != nil || entries == nil {
							dst = candidate
							break
						}
					}
				}

				var cb vault.ProgressCallback
				if reporter != nil {
					cb = func(current, _ int) {
						reporter.report(current, total)
					}
				}

				if isCut {
					count, err := client.MoveRecursiveCtx(ctx, yp, dst, cb)
					if err != nil {
						return progressDoneMsg{operation: operation, count: totalCount + count, err: err}
					}
					totalCount += count
				} else {
					count, err := client.CopyRecursiveWithHistoryCtx(ctx, yp, dst, cb, &counter)
					if err != nil {
						return progressDoneMsg{operation: operation, count: totalCount + count, err: err}
					}
					totalCount += count
				}
			}
			if isCut {
				return progressDoneMsg{operation: operation, count: totalCount, status: fmt.Sprintf("Moved %d items", totalCount)}
			}
			return progressDoneMsg{operation: operation, count: totalCount, status: fmt.Sprintf("Copied %d items (with history)", totalCount)}
		}
	}

	return func() tea.Msg {
		if len(yankedSecrets) == 0 {
			return errorMsg("nothing yanked")
		}

		pastedCount := 0
		lastDst := ""
		for _, yanked := range yankedSecrets {
			parts := strings.Split(strings.TrimSuffix(yanked.Path, "/"), "/")
			name := parts[len(parts)-1]
			dst := basePath + name

			// If destination exists, append _1, _2, etc.
			if _, err := client.Read(dst); err == nil {
				for i := 1; ; i++ {
					candidate := basePath + fmt.Sprintf("%s_%d", name, i)
					if _, err := client.Read(candidate); err != nil {
						dst = candidate
						break
					}
				}
			}

			// Use CopyWithHistory to preserve version history.
			if err := client.CopyWithHistory(yanked.Path, dst); err != nil {
				return errorMsg(fmt.Sprintf("pasted %d items, then error: %v", pastedCount, err))
			}

			if isCut {
				if err := client.Delete(yanked.Path); err != nil {
					return errorMsg(fmt.Sprintf("pasted to %s but failed to delete source: %v", dst, err))
				}
			}
			pastedCount++
			lastDst = dst
		}

		if pastedCount == 1 {
			yanked := yankedSecrets[0]
			dst := lastDst
			if isCut {
				return undoableStatusMsg{
					status: fmt.Sprintf("Moved %s → %s", yanked.Path, dst),
					undo: model.UndoAction{
						Type:        model.UndoCutPaste,
						Description: fmt.Sprintf("move %s → %s", yanked.Path, dst),
						Path:        dst,
						OldPath:     yanked.Path,
						Data:        copyMap(yanked.Data),
						Keys:        copySlice(yanked.Keys),
						Mount:       client.Mount(),
					},
				}
			}
			return undoableStatusMsg{
				status: fmt.Sprintf("Pasted %s → %s", yanked.Path, dst),
				undo: model.UndoAction{
					Type:        model.UndoPasteSecret,
					Description: "paste to " + dst,
					Path:        dst,
					Data:        copyMap(yanked.Data),
					Keys:        copySlice(yanked.Keys),
					Mount:       client.Mount(),
				},
			}
		}

		if isCut {
			return statusMsg(fmt.Sprintf("Moved %d secrets", pastedCount))
		}
		return statusMsg(fmt.Sprintf("Pasted %d secrets", pastedCount))
	}
}

func (m *Model) deleteKey(key string) tea.Cmd {
	// Snapshot before change
	snapData := copyMap(m.secret.Data)
	snapKeys := copySlice(m.secret.Keys)
	secretPath := m.secret.Path

	// Mutate model state synchronously (safe — called from Update goroutine).
	delete(m.secret.Data, key)
	newKeys := make([]string, 0, len(m.secret.Keys))
	for _, k := range m.secret.Keys {
		if k != key {
			newKeys = append(newKeys, k)
		}
	}
	m.secret.Keys = newKeys

	if m.secretCursor >= len(m.secret.Keys) && m.secretCursor > 0 {
		m.secretCursor--
	}

	// Copy updated state for the async Vault write.
	writeData := copyMap(m.secret.Data)
	client := m.client

	return func() tea.Msg {
		if err := client.Write(secretPath, writeData); err != nil {
			return errorMsg(err.Error())
		}
		return undoableStatusMsg{
			status: "Deleted key: " + key,
			undo: model.UndoAction{
				Type:        model.UndoEditSecret,
				Description: "delete key " + key,
				Path:        secretPath,
				Data:        snapData,
				Keys:        snapKeys,
				Mount:       client.Mount(),
			},
		}
	}
}

// editorCommand returns the editor to use, checking config, then $EDITOR,
// then $VISUAL, and falling back to "vim". It validates that the editor
// binary exists in PATH before returning.
func (m *Model) editorCommand() (string, error) {
	editor := "vim"
	if m.config != nil && m.config.Editor != "" {
		editor = m.config.Editor
	} else if e := os.Getenv("EDITOR"); e != "" {
		editor = e
	} else if e := os.Getenv("VISUAL"); e != "" {
		editor = e
	}
	parts := strings.Fields(editor)
	if len(parts) == 0 {
		return "", fmt.Errorf("editor is empty")
	}
	// A quoted value cannot be split on spaces, so the shell reports a missing binary instead.
	if !strings.ContainsAny(editor, `"'\`) {
		if _, err := exec.LookPath(parts[0]); err != nil {
			return "", fmt.Errorf("editor %q not found in PATH", parts[0])
		}
	}
	return editor, nil
}

// editorExecCommand runs the editor value through sh like git does, so quotes
// and arguments in it work. trailingArgs go in as "$@" and are never parsed.
func editorExecCommand(editor string, trailingArgs ...string) *exec.Cmd {
	args := append([]string{"-c", editor + ` "$@"`, editor}, trailingArgs...)
	return exec.Command("sh", args...) //nolint:gosec // editor is user-configured
}

func (m *Model) openEditorForNewSecret(path string) tea.Cmd {
	data := []byte("{\n  \n}")

	tmpDir := os.Getenv("XDG_RUNTIME_DIR")
	if tmpDir == "" {
		tmpDir = os.TempDir()
	}
	tmpFile, err := os.CreateTemp(tmpDir, ".vau-edit-*.json")
	if err != nil {
		return func() tea.Msg { return errorMsg("tmpfile: " + err.Error()) }
	}
	if _, err := tmpFile.Write(data); err != nil {
		os.Remove(tmpFile.Name())
		return func() tea.Msg { return errorMsg("write: " + err.Error()) }
	}
	tmpFile.Close()
	os.Chmod(tmpFile.Name(), 0o600)

	editor, err := m.editorCommand()
	if err != nil {
		os.Remove(tmpFile.Name())
		return func() tea.Msg { return errorMsg(err.Error()) }
	}

	c := editorExecCommand(editor, tmpFile.Name())
	return tea.ExecProcess(c, func(err error) tea.Msg {
		defer os.Remove(tmpFile.Name())
		if err != nil {
			return errorMsg("editor: " + err.Error())
		}
		edited, err := os.ReadFile(tmpFile.Name())
		if err != nil {
			return errorMsg("read: " + err.Error())
		}
		var newData map[string]string
		if err := json.Unmarshal(edited, &newData); err != nil {
			return errorMsg("invalid JSON: " + err.Error())
		}
		return editorResultMsg{data: newData, newSecretPath: path}
	})
}

func (m *Model) addKeyValue(key, val string) tea.Cmd {
	snapData := copyMap(m.secret.Data)
	snapKeys := copySlice(m.secret.Keys)
	secretPath := m.secret.Path

	// Mutate model state synchronously (safe — called from Update goroutine).
	m.secret.Data[key] = val
	m.secret.Keys = append(m.secret.Keys, key)

	// Copy updated state for the async Vault write.
	writeData := copyMap(m.secret.Data)
	client := m.client

	return func() tea.Msg {
		if err := client.Write(secretPath, writeData); err != nil {
			return errorMsg(err.Error())
		}
		return undoableStatusMsg{
			status: "Added key: " + key,
			undo: model.UndoAction{
				Type:        model.UndoEditSecret,
				Description: "add key " + key,
				Path:        secretPath,
				Data:        snapData,
				Keys:        snapKeys,
				Mount:       client.Mount(),
			},
		}
	}
}

func (m *Model) editValue(key, val string) tea.Cmd {
	snapData := copyMap(m.secret.Data)
	snapKeys := copySlice(m.secret.Keys)
	secretPath := m.secret.Path

	// Mutate model state synchronously (safe — called from Update goroutine).
	m.secret.Data[key] = val

	// Copy updated state for the async Vault write.
	writeData := copyMap(m.secret.Data)
	client := m.client

	return func() tea.Msg {
		if err := client.Write(secretPath, writeData); err != nil {
			return errorMsg(err.Error())
		}
		return undoableStatusMsg{
			status: "Updated: " + key,
			undo: model.UndoAction{
				Type:        model.UndoEditSecret,
				Description: "edit " + key,
				Path:        secretPath,
				Data:        snapData,
				Keys:        snapKeys,
				Mount:       client.Mount(),
			},
		}
	}
}

// editValueWithSnapshot is like editValue, but takes snapData/snapKeys from
// the caller: applyCurrentEditInput already mutated m.secret by this point.
func (m *Model) editValueWithSnapshot(key, val string, snapData map[string]string, snapKeys []string) tea.Cmd {
	secretPath := m.secret.Path

	m.secret.Data[key] = val

	writeData := copyMap(m.secret.Data)
	client := m.client

	return func() tea.Msg {
		if err := client.Write(secretPath, writeData); err != nil {
			return errorMsg(err.Error())
		}
		return undoableStatusMsg{
			status: "Updated: " + key,
			undo: model.UndoAction{
				Type:        model.UndoEditSecret,
				Description: "edit " + key,
				Path:        secretPath,
				Data:        snapData,
				Keys:        snapKeys,
				Mount:       client.Mount(),
			},
		}
	}
}

// editSecretInEditor opens the secret data in an external editor as pretty-printed JSON.
func (m *Model) editSecretInEditor() tea.Cmd {
	data, err := json.MarshalIndent(m.secret.Data, "", "  ")
	if err != nil {
		return func() tea.Msg { return errorMsg("json: " + err.Error()) }
	}

	tmpDir := os.Getenv("XDG_RUNTIME_DIR")
	if tmpDir == "" {
		tmpDir = os.TempDir()
	}
	tmpFile, err := os.CreateTemp(tmpDir, ".vau-edit-*.json")
	if err != nil {
		return func() tea.Msg { return errorMsg("tmpfile: " + err.Error()) }
	}
	if _, err := tmpFile.Write(data); err != nil {
		os.Remove(tmpFile.Name())
		return func() tea.Msg { return errorMsg("write: " + err.Error()) }
	}
	tmpFile.Close()
	os.Chmod(tmpFile.Name(), 0o600)

	editor, err := m.editorCommand()
	if err != nil {
		os.Remove(tmpFile.Name())
		return func() tea.Msg { return errorMsg(err.Error()) }
	}

	c := editorExecCommand(editor, tmpFile.Name())
	return tea.ExecProcess(c, func(err error) tea.Msg {
		defer os.Remove(tmpFile.Name())
		if err != nil {
			return errorMsg("editor: " + err.Error())
		}

		edited, err := os.ReadFile(tmpFile.Name())
		if err != nil {
			return errorMsg("read: " + err.Error())
		}

		var newData map[string]string
		if err := json.Unmarshal(edited, &newData); err != nil {
			return errorMsg("invalid JSON: " + err.Error())
		}

		return editorResultMsg{data: newData}
	})
}

// saveSecretFromEditor writes the editor-modified secret to Vault and returns an undoable status.
func (m *Model) saveSecretFromEditor(secretPath string, snapData map[string]string, snapKeys []string) tea.Cmd {
	// Copy updated state for the async Vault write (avoid reading m.secret in goroutine).
	writeData := copyMap(m.secret.Data)
	client := m.client

	return func() tea.Msg {
		if err := client.Write(secretPath, writeData); err != nil {
			return errorMsg(err.Error())
		}
		return undoableStatusMsg{
			status: "Updated secret via editor",
			undo: model.UndoAction{
				Type:        model.UndoEditSecret,
				Description: "editor edit",
				Path:        secretPath,
				Data:        snapData,
				Keys:        snapKeys,
				Mount:       client.Mount(),
			},
		}
	}
}

// --- Bulk selection helpers ---

func (m *Model) actualIdx(visibleIdx int) int {
	if m.filterQuery == "" {
		return visibleIdx
	}
	if visibleIdx < len(m.filteredIdx) {
		return m.filteredIdx[visibleIdx]
	}
	return visibleIdx
}

func (m *Model) selectedEntries() []model.Entry {
	var entries []model.Entry
	for idx := range m.selected {
		if idx < len(m.entries) {
			entries = append(entries, m.entries[idx])
		}
	}
	return entries
}

func (m *Model) selectedVisible() map[int]bool {
	if m.filterQuery == "" {
		return m.selected
	}
	vis := make(map[int]bool)
	for vi, ai := range m.filteredIdx {
		if m.selected[ai] {
			vis[vi] = true
		}
	}
	return vis
}

func (m *Model) bulkDelete(entries []model.Entry) tea.Cmd {
	operation := "Deleting"
	basePath := m.currentPath()
	client := m.client

	ctx, cancel := context.WithCancel(context.Background()) //nolint:gosec // cancel stored in m.progress.cancel
	m.progress = progressState{
		active:    true,
		cancel:    cancel,
		operation: operation,
		current:   0,
		total:     0, // unknown until counted inside the cmd below
	}
	m.selected = make(map[int]bool)

	var reporter *progressReporter
	if m.program != nil {
		reporter = newProgressReporter(m.program)
	}

	return func() tea.Msg {
		if err := ctx.Err(); err != nil {
			return progressDoneMsg{operation: operation, err: err}
		}

		// Count total items for progress.
		total := 0
		for _, entry := range entries {
			if entry.IsDir {
				path := basePath + strings.TrimSuffix(entry.Name, "/")
				n, _ := client.CountRecursive(path)
				total += n
			} else {
				total++
			}
		}
		if reporter != nil {
			reporter.report(0, total)
		}

		deleted := 0
		counter := 0
		for _, entry := range entries {
			if err := ctx.Err(); err != nil {
				return progressDoneMsg{operation: operation, count: deleted, err: err}
			}
			path := basePath + strings.TrimSuffix(entry.Name, "/")
			if entry.IsDir {
				var cb vault.ProgressCallback
				if reporter != nil {
					cb = func(current, _ int) {
						reporter.report(current, total)
					}
				}
				n, err := client.DeleteRecursiveCtx(ctx, path, cb, &counter)
				if err != nil {
					return progressDoneMsg{operation: operation, count: deleted + n, err: err}
				}
				deleted += n
			} else {
				if err := client.Delete(path); err != nil {
					return progressDoneMsg{operation: operation, count: deleted, err: fmt.Errorf("deleting %s: %w", path, err)}
				}
				deleted++
				counter++
				if reporter != nil {
					reporter.report(counter, total)
				}
			}
		}
		return progressDoneMsg{operation: operation, count: deleted, status: fmt.Sprintf("Deleted %d items", deleted)}
	}
}
