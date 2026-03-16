package ui

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/janosmiko/vau/internal/model"
)

// Inner panel style for the data table inside the secret modal.
var innerPanelStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.RoundedBorder()).
	BorderForeground(colorBorder).
	Padding(0, 1)

// RenderSecretOverlay renders a centered popup overlay for secret editing.
// editingColumn: -1 = not editing, 0 = editing key column, 1 = editing value column.
func RenderSecretOverlay(
	secret *model.Secret,
	selectedIdx int,
	revealedKeys map[string]bool,
	allRevealed bool,
	jsonView bool,
	editingKey string,
	editingView string,
	editingColumn int,
	base64Keys map[string]bool,
	screenWidth, screenHeight int,
) string {
	if secret == nil {
		return overlayBoxStyle.Render(ErrorStyle.Render("No secret loaded"))
	}

	// Popup dimensions: 75% of screen
	boxW := screenWidth * 75 / 100
	boxH := screenHeight * 75 / 100
	if boxW < 50 {
		boxW = 50
	}
	if boxH < 10 {
		boxH = 10
	}

	outerPadH := 4 // outer border (2) + outer padding (2)
	outerPadW := 6 // outer border (2) + outer padding (2*2)
	innerPadH := 2 // inner border (2) + inner padding (0)
	innerPadW := 4 // inner border (2) + inner padding (1*2)
	titleH := 1    // title line
	helpH := 1     // help line
	gapH := 2      // blank line above inner panel + blank line below

	panelContentH := boxH - outerPadH - innerPadH - titleH - helpH - gapH
	if panelContentH < 3 {
		panelContentH = 3
	}
	panelContentW := boxW - outerPadW - innerPadW
	if panelContentW < 20 {
		panelContentW = 20
	}
	panelW := boxW - outerPadW

	// Title
	title := TitleStyle.Render(secret.Path)

	// Data content
	var dataContent string
	if jsonView {
		dataContent = renderSecretPopupJSON(secret, panelContentH)
	} else {
		dataContent = renderSecretTable(secret, selectedIdx, revealedKeys, base64Keys, editingKey, editingView, editingColumn, panelContentW, panelContentH)
	}

	// Inner bordered panel with fixed height
	innerPanel := innerPanelStyle.
		Width(panelW).
		Height(panelContentH).
		Render(dataContent)

	// Help line
	var helpLine string
	if editingColumn >= 0 {
		helpLine = HelpKeyStyle.Render("Tab") + HelpDescStyle.Render(" switch col") + "  " +
			HelpKeyStyle.Render("Enter") + HelpDescStyle.Render(" save") + "  " +
			HelpKeyStyle.Render("Esc") + HelpDescStyle.Render(" cancel")
	} else {
		helpLine = HelpKeyStyle.Render("jk") + HelpDescStyle.Render(" nav") + "  " +
			HelpKeyStyle.Render("v") + HelpDescStyle.Render(" toggle") + "  " +
			HelpKeyStyle.Render("V") + HelpDescStyle.Render(" json") + "  " +
			HelpKeyStyle.Render("b") + HelpDescStyle.Render(" b64") + "  " +
			HelpKeyStyle.Render("y") + HelpDescStyle.Render(" copy") + "  " +
			HelpKeyStyle.Render("p") + HelpDescStyle.Render(" paste") + "  " +
			HelpKeyStyle.Render("e") + HelpDescStyle.Render(" edit") + "  " +
			HelpKeyStyle.Render("a") + HelpDescStyle.Render(" add") + "  " +
			HelpKeyStyle.Render("H") + HelpDescStyle.Render(" history") + "  " +
			HelpKeyStyle.Render("D") + HelpDescStyle.Render(" del") + "  " +
			HelpKeyStyle.Render("^d/^u") + HelpDescStyle.Render(" scroll") + "  " +
			HelpKeyStyle.Render("esc") + HelpDescStyle.Render(" close")
	}

	body := title + "\n" + innerPanel + "\n" + helpLine

	return overlayBoxStyle.
		Width(boxW).
		Render(body)
}

// renderSecretTable renders the key-value table content (shared by overlay).
// editingColumn: -1 = not editing, 0 = editing key column, 1 = editing value column.
func renderSecretTable(
	secret *model.Secret,
	selectedIdx int,
	revealedKeys map[string]bool,
	base64Keys map[string]bool,
	editingKey string,
	editingView string,
	editingColumn int,
	width, height int,
) string {
	keyColW := 0
	for _, k := range secret.Keys {
		if len(k) > keyColW {
			keyColW = len(k)
		}
	}
	if keyColW < 10 {
		keyColW = 10
	}
	if keyColW > width/3 {
		keyColW = width / 3
	}

	valColW := width - keyColW - 10
	if valColW < 8 {
		valColW = 8
	}

	var lines []string

	// Header: pad plain text before styling
	keyPadded := fmt.Sprintf("%-*s", keyColW, "Key")
	headerLine := "  " + TableHeaderStyle.Render(keyPadded) + "  │  " + TableHeaderStyle.Render("Value")
	separator := "  " + strings.Repeat("─", keyColW) + "──┼──" + strings.Repeat("─", valColW)
	lines = append(lines, headerLine)
	lines = append(lines, separator)

	tableHeight := height - 2
	if tableHeight < 1 {
		tableHeight = 1
	}
	start := 0
	if selectedIdx >= tableHeight {
		start = selectedIdx - tableHeight + 1
	}
	end := start + tableHeight
	if end > len(secret.Keys) {
		end = len(secret.Keys)
	}

	for i := start; i < end; i++ {
		k := secret.Keys[i]
		v := secret.Data[k]

		var line string
		if i == selectedIdx && editingColumn >= 0 && (k == editingKey || editingColumn == 0) {
			// Inline edit mode: show text input in the active column
			if editingColumn == 0 {
				// Editing key column
				valDisplay := getValueDisplayWithBase64(v, revealedKeys[k], base64Keys[k], valColW)
				line = "▸ " + editingView + fmt.Sprintf("%-*s", max(0, keyColW-lipgloss.Width(editingView)), "") + "  │  " + valDisplay
			} else {
				// Editing value column
				rawLine := fmt.Sprintf("▸ %-*s  │  ", keyColW, truncate(k, keyColW))
				line = rawLine + editingView
			}
		} else if i == selectedIdx {
			// Selected row: build raw text then apply SelectedStyle
			valDisplay := getValueDisplayWithBase64(v, revealedKeys[k], base64Keys[k], valColW)
			rawLine := fmt.Sprintf("▸ %-*s  │  %-*s", keyColW, truncate(k, keyColW), valColW, valDisplay)
			line = SelectedStyle.Render(rawLine)
		} else {
			// Normal row: pad plain text before styling
			kPadded := fmt.Sprintf("%-*s", keyColW, truncate(k, keyColW))
			keyStr := TableKeyStyle.Render(kPadded)

			var valStr string
			if base64Keys[k] {
				decoded := tryBase64Decode(v)
				valStr = Base64ValueStyle.Render("[b64] " + truncate(decoded, valColW-6))
			} else if revealedKeys[k] {
				valStr = TableValueStyle.Render(truncate(v, valColW))
			} else {
				valStr = HiddenValueStyle.Render("********")
			}

			line = "  " + keyStr + "  │  " + valStr
		}
		lines = append(lines, line)
	}

	if len(secret.Keys) == 0 {
		lines = append(lines, HelpDescStyle.Render("  (empty — press 'a' to add a key)"))
	}

	return strings.Join(lines, "\n")
}

func renderSecretPopupJSON(secret *model.Secret, height int) string {
	var lines []string
	lines = append(lines, HelpDescStyle.Render("{"))
	limit := height - 2
	if limit > len(secret.Keys) {
		limit = len(secret.Keys)
	}
	for i := 0; i < limit; i++ {
		k := secret.Keys[i]
		v := secret.Data[k]
		comma := ","
		if i == len(secret.Keys)-1 {
			comma = ""
		}
		line := fmt.Sprintf("  %s: %s%s",
			TableKeyStyle.Render(fmt.Sprintf("%q", k)),
			TableValueStyle.Render(fmt.Sprintf("%q", v)),
			comma,
		)
		lines = append(lines, line)
	}
	lines = append(lines, HelpDescStyle.Render("}"))
	return strings.Join(lines, "\n")
}

func getValueDisplay(val string, revealed bool, maxW int) string {
	if revealed {
		return truncate(val, maxW)
	}
	return "********"
}

func getValueDisplayWithBase64(val string, revealed bool, isBase64 bool, maxW int) string {
	if isBase64 {
		decoded := tryBase64Decode(val)
		return "[b64] " + truncate(decoded, maxW-6)
	}
	return getValueDisplay(val, revealed, maxW)
}

func tryBase64Decode(s string) string {
	var raw []byte
	if decoded, err := base64.StdEncoding.DecodeString(s); err == nil {
		raw = decoded
	} else if decoded, err := base64.URLEncoding.DecodeString(s); err == nil {
		raw = decoded
	} else if decoded, err := base64.RawStdEncoding.DecodeString(s); err == nil {
		raw = decoded
	} else {
		return s
	}
	// Sanitize: replace control chars and newlines with visible representations
	return sanitizeForDisplay(string(raw))
}

func sanitizeForDisplay(s string) string {
	var out []rune
	for _, r := range s {
		if r == '\n' {
			out = append(out, '↵')
		} else if r == '\r' {
			// skip
		} else if r == '\t' {
			out = append(out, ' ', ' ')
		} else if r < 32 || r == 127 {
			out = append(out, '·')
		} else {
			out = append(out, r)
		}
	}
	return string(out)
}

func truncate(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	if len(s) > maxLen {
		if maxLen > 3 {
			return s[:maxLen-3] + "..."
		}
		return s[:maxLen]
	}
	return s
}

// RenderVersionHistoryOverlay renders a centered popup for secret version history.
func RenderVersionHistoryOverlay(versions []model.SecretVersion, selectedIdx int, path string, screenWidth, screenHeight int) string {
	boxW := screenWidth * 60 / 100
	boxH := screenHeight * 60 / 100
	if boxW < 50 {
		boxW = 50
	}
	if boxH < 10 {
		boxH = 10
	}

	title := TitleStyle.Render("Version History: " + path)

	innerW := boxW - 10

	var lines []string
	headerLine := fmt.Sprintf("  %-8s  %-24s  %s", TableHeaderStyle.Render("Version"), TableHeaderStyle.Render("Created"), TableHeaderStyle.Render("Status"))
	lines = append(lines, headerLine)
	lines = append(lines, "  "+strings.Repeat("─", innerW))

	tableH := boxH - 8
	if tableH < 1 {
		tableH = 1
	}
	start := 0
	if selectedIdx >= tableH {
		start = selectedIdx - tableH + 1
	}
	end := start + tableH
	if end > len(versions) {
		end = len(versions)
	}

	for i := start; i < end; i++ {
		v := versions[i]
		status := "current"
		if v.Destroyed {
			status = "destroyed"
		} else if v.DeletionTime != "" && v.DeletionTime != "0001-01-01T00:00:00Z" {
			status = "deleted"
		}
		// Truncate created time to reasonable length
		created := v.CreatedTime
		if len(created) > 19 {
			created = created[:19]
		}

		var line string
		if i == selectedIdx {
			rawLine := fmt.Sprintf("▸ v%-7s  %-24s  %s", v.Version, created, status)
			line = SelectedStyle.Render(rawLine)
		} else {
			line = fmt.Sprintf("  v%-7s  %-24s  %s",
				TableKeyStyle.Render(v.Version),
				HelpDescStyle.Render(created),
				TableValueStyle.Render(status),
			)
		}
		lines = append(lines, line)
	}

	if len(versions) == 0 {
		lines = append(lines, HelpDescStyle.Render("  (no versions found)"))
	}

	panelContent := strings.Join(lines, "\n")
	panelW := boxW - 6
	innerPanel := innerPanelStyle.Width(panelW).Render(panelContent)

	helpLine := HelpKeyStyle.Render("jk") + HelpDescStyle.Render(" nav") + "  " +
		HelpKeyStyle.Render("Enter") + HelpDescStyle.Render(" view version") + "  " +
		HelpKeyStyle.Render("esc") + HelpDescStyle.Render(" back")

	body := title + "\n" + innerPanel + "\n" + helpLine

	return overlayBoxStyle.Width(boxW).Render(body)
}
