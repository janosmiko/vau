package app

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/janosmiko/vau/internal/model"
	"github.com/janosmiko/vau/internal/ui"
)

func (m *Model) handleHelpKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// When searching, delegate to the text input
	if m.helpSearching {
		switch msg.String() {
		case "enter":
			m.helpSearching = false
			m.helpFilter = m.searchInput.Value()
			m.helpScroll = 0
			return m, nil
		case "esc":
			m.helpSearching = false
			m.helpFilter = ""
			m.searchInput.SetValue("")
			m.helpScroll = 0
			return m, nil
		default:
			var cmd tea.Cmd
			m.searchInput, cmd = m.searchInput.Update(msg)
			// Live filter while typing
			m.helpFilter = m.searchInput.Value()
			m.helpScroll = 0
			return m, cmd
		}
	}

	maxScroll := max(ui.HelpContentLineCount(m.helpFilter)-m.helpVisibleLines(), 0)

	switch msg.String() {
	case "esc", "?", "q":
		m.mode = model.ModeExplorer
	case "j", "down", "ctrl+n":
		if m.helpScroll < maxScroll {
			m.helpScroll++
		}
	case "k", "up", "ctrl+p":
		if m.helpScroll > 0 {
			m.helpScroll--
		}
	case "ctrl+d":
		halfPage := max(m.helpVisibleLines()/2, 1)
		m.helpScroll = min(m.helpScroll+halfPage, maxScroll)
	case "ctrl+u":
		halfPage := max(m.helpVisibleLines()/2, 1)
		m.helpScroll = max(m.helpScroll-halfPage, 0)
	case "ctrl+f":
		fullPage := max(m.helpVisibleLines(), 1)
		m.helpScroll = min(m.helpScroll+fullPage, maxScroll)
	case "ctrl+b":
		fullPage := max(m.helpVisibleLines(), 1)
		m.helpScroll = max(m.helpScroll-fullPage, 0)
	case "g":
		m.helpScroll = 0
	case "G":
		m.helpScroll = maxScroll
	case "/":
		m.helpSearching = true
		m.searchInput.SetValue(m.helpFilter)
		m.searchInput.Focus()
		return m, textinput.Blink
	}
	return m, nil
}

// helpVisibleLines returns the number of content lines visible in the help overlay.
func (m *Model) helpVisibleLines() int {
	boxH := max(m.height*80/100, 20)
	// title + borders/padding + help line + possible scroll indicators
	maxLines := max(boxH-6, 5)
	return maxLines
}

// explorerHalfPage returns the half-page scroll amount for explorer mode.
func (m *Model) explorerHalfPage() int {
	// The explorer visible height is roughly height - 2 (status + help bars)
	// minus some overhead for borders/padding. Use a simple estimate.
	half := max((m.height-2)/2, 1)
	return half
}

// secretPopupHalfPage returns the half-page scroll amount for the secret popup.
func (m *Model) secretPopupHalfPage() int {
	boxH := max(m.height*75/100, 10)
	outerPadH := 4 // outer border (2) + outer padding (2)
	innerPadH := 2 // inner border (2) + inner padding (0)
	titleH := 1
	helpH := 1
	gapH := 2

	panelContentH := max(boxH-outerPadH-innerPadH-titleH-helpH-gapH, 3)
	tableHeight := max(panelContentH-2, 1)
	half := max(tableHeight/2, 1)
	return half
}

func (m *Model) copySecretAsJSON() tea.Cmd {
	data, err := json.Marshal(m.secret.Data)
	if err != nil {
		return func() tea.Msg { return errorMsg("json: " + err.Error()) }
	}
	return copyToSystemClipboard(string(data), "secret JSON")
}

// formatSecretAsYAML renders secret key-value pairs as YAML text.
func formatSecretAsYAML(secret *model.Secret) string {
	var buf strings.Builder
	for _, k := range secret.Keys {
		v := secret.Data[k]
		// Quote values that contain special YAML characters or are empty
		if v == "" || strings.ContainsAny(v, ":#{}[]&*!|>'\",\n") || v == "true" || v == "false" || v == "null" {
			buf.WriteString(k + ": " + strconv.Quote(v) + "\n")
		} else {
			buf.WriteString(k + ": " + v + "\n")
		}
	}
	return buf.String()
}

func (m *Model) copySecretAsYAML() tea.Cmd {
	if m.secret == nil {
		return nil
	}
	return copyToSystemClipboard(formatSecretAsYAML(m.secret), "secret YAML")
}

// formatSecretAsDotenv renders secret key-value pairs as dotenv text.
func formatSecretAsDotenv(secret *model.Secret) string {
	var buf strings.Builder
	for _, k := range secret.Keys {
		v := secret.Data[k]
		// Use double quotes for values containing special characters
		if strings.ContainsAny(v, " \t\n\"'\\$`!#") || v == "" {
			escaped := strings.ReplaceAll(v, "\\", "\\\\")
			escaped = strings.ReplaceAll(escaped, "\"", "\\\"")
			escaped = strings.ReplaceAll(escaped, "\n", "\\n")
			buf.WriteString(k + "=\"" + escaped + "\"\n")
		} else {
			buf.WriteString(k + "=" + v + "\n")
		}
	}
	return buf.String()
}

func (m *Model) copySecretAsDotenv() tea.Cmd {
	if m.secret == nil {
		return nil
	}
	return copyToSystemClipboard(formatSecretAsDotenv(m.secret), "secret dotenv")
}

// handleCopyFormat processes the second key after pressing Y (copy-as format).
// Works in both secret popup (m.secret) and explorer (m.previewSecret).
func (m *Model) handleCopyFormat(key string) (tea.Model, tea.Cmd) {
	s := m.copySecret
	m.copySecret = nil
	if s == nil {
		m.status = ""
		return m, nil
	}
	switch key {
	case "j":
		m.status = ""
		data, err := json.Marshal(s.Data)
		if err != nil {
			return m, func() tea.Msg { return errorMsg("json: " + err.Error()) }
		}
		return m, copyToSystemClipboard(string(data), "secret JSON")
	case "y":
		m.status = ""
		return m, copyToSystemClipboard(formatSecretAsYAML(s), "secret YAML")
	case "d":
		m.status = ""
		return m, copyToSystemClipboard(formatSecretAsDotenv(s), "secret dotenv")
	default:
		m.status = ""
		return m, nil
	}
}
