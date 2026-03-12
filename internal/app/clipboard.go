package app

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func copyToSystemClipboard(val, key string) tea.Cmd {
	return func() tea.Msg {
		var cmd *exec.Cmd
		switch runtime.GOOS {
		case "darwin":
			cmd = exec.Command("pbcopy")
		case "linux":
			// Try xclip first, then xsel
			if _, err := exec.LookPath("xclip"); err == nil {
				cmd = exec.Command("xclip", "-selection", "clipboard")
			} else if _, err := exec.LookPath("xsel"); err == nil {
				cmd = exec.Command("xsel", "--clipboard", "--input")
			} else {
				return statusMsg(fmt.Sprintf("Copied %q (no clipboard tool found)", key))
			}
		default:
			return statusMsg(fmt.Sprintf("Copied %q (clipboard not supported on %s)", key, runtime.GOOS))
		}

		cmd.Stdin = strings.NewReader(val)
		if err := cmd.Run(); err != nil {
			return errorMsg(fmt.Sprintf("clipboard: %v", err))
		}
		return statusMsg(fmt.Sprintf("Copied value of %q to clipboard", key))
	}
}

func readFromSystemClipboard() (string, error) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbpaste")
	case "linux":
		if _, err := exec.LookPath("xclip"); err == nil {
			cmd = exec.Command("xclip", "-selection", "clipboard", "-o")
		} else if _, err := exec.LookPath("xsel"); err == nil {
			cmd = exec.Command("xsel", "--clipboard", "--output")
		} else {
			return "", fmt.Errorf("no clipboard tool found")
		}
	default:
		return "", fmt.Errorf("clipboard not supported on %s", runtime.GOOS)
	}

	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}
