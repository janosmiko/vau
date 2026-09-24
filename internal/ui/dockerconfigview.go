package ui

import (
	"fmt"
	"strings"

	"github.com/janosmiko/vau/internal/model"
)

// Fields that stay masked until the user reveals them.
var dockerSecretFields = map[string]bool{"password": true, "auth": true, "identitytoken": true}

// RenderDockerConfigOverlay renders the flattened registry credentials of a
// dockerconfigjson value as a centered popup.
func RenderDockerConfigOverlay(title string, fields []model.DockerConfigField, cursor int, revealed bool, width, height int) string {
	boxW := max(width*75/100, 50)
	boxH := max(height*75/100, 10)
	panelContentH := max(boxH-7, 3)
	panelContentW := max(boxW-10, 20)

	regW := len("Registry")
	for _, f := range fields {
		regW = max(regW, len(f.Registry))
	}
	regW = min(regW, panelContentW/3)
	nameW := len("identitytoken")
	valW := max(panelContentW-regW-nameW-6, 8)

	lines := []string{
		"  " + TableHeaderStyle.Render(fmt.Sprintf("%-*s  %-*s  Value", regW, "Registry", nameW, "Field")),
		"  " + strings.Repeat("─", panelContentW-2),
	}

	rows := max(panelContentH-2, 1)
	start := max(cursor-rows+1, 0)
	end := min(start+rows, len(fields))
	for i := start; i < end; i++ {
		f := fields[i]
		val := "********"
		if revealed || !dockerSecretFields[f.Name] {
			val = truncate(sanitizeForDisplay(f.Value), valW)
		}
		reg := truncate(sanitizeForDisplay(f.Registry), regW)
		if i == cursor {
			lines = append(lines, SelectedStyle.Render(fmt.Sprintf("▸ %-*s  %-*s  %-*s", regW, reg, nameW, f.Name, valW, val)))
			continue
		}
		lines = append(lines, "  "+TableKeyStyle.Render(fmt.Sprintf("%-*s", regW, reg))+"  "+
			HelpDescStyle.Render(fmt.Sprintf("%-*s", nameW, f.Name))+"  "+TableValueStyle.Render(val))
	}

	innerPanel := innerPanelStyle.Width(boxW - 6).Height(panelContentH).Render(strings.Join(lines, "\n"))
	return overlayBoxStyle.Width(boxW).Render(TitleStyle.Render(sanitizeForDisplay(title)) + "\n" + innerPanel)
}
