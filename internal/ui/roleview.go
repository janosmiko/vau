package ui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/janosmiko/vau/internal/model"
)

// RenderAuthMethodList renders a three-column view matching the explorer layout.
func RenderAuthMethodList(
	parentEntries []model.Entry, parentIdx int,
	methods []model.Entry, cursor int, preview []model.Entry,
	tabLabels []string, activeTab int,
	version string, width, height int,
) string {
	lay := computeThreeColumnLayout("Auth Methods/", version, tabLabels, activeTab, width, height)

	leftCol := renderEntryList(parentEntries, parentIdx, nil, lay.leftW, lay.colHeight, false, "")
	midCol := renderEntryList(methods, cursor, nil, lay.midW, lay.colHeight, true, "")
	rightCol := renderPreview(preview, nil, model.PreviewHidden, lay.rightW, lay.colHeight)

	left := ColumnStyle.Width(lay.leftW).Height(lay.colHeight).MaxHeight(lay.colHeight + 2).Render(leftCol)
	mid := ActiveColumnStyle.Width(lay.midW).Height(lay.colHeight).MaxHeight(lay.colHeight + 2).Render(midCol)
	right := ColumnStyle.Width(lay.rightW).Height(lay.colHeight).MaxHeight(lay.colHeight + 2).Render(rightCol)

	columns := lipgloss.JoinHorizontal(lipgloss.Top, left, mid, right)
	return lay.header + "\n" + columns
}

// RenderRoleList renders a three-column view matching the explorer layout.
func RenderRoleList(
	parentEntries []model.Entry, parentIdx int,
	roles []model.Entry, cursor int, authPath string,
	preview map[string]any,
	tabLabels []string, activeTab int,
	version string, width, height int,
) string {
	lay := computeThreeColumnLayout("Auth Methods/"+authPath, version, tabLabels, activeTab, width, height)

	leftCol := renderEntryList(parentEntries, parentIdx, nil, lay.leftW, lay.colHeight, false, "")
	midCol := renderEntryList(roles, cursor, nil, lay.midW, lay.colHeight, true, "")
	rightCol := renderRoleDataPreview(preview, lay.rightW, lay.colHeight)

	left := ColumnStyle.Width(lay.leftW).Height(lay.colHeight).MaxHeight(lay.colHeight + 2).Render(leftCol)
	mid := ActiveColumnStyle.Width(lay.midW).Height(lay.colHeight).MaxHeight(lay.colHeight + 2).Render(midCol)
	right := ColumnStyle.Width(lay.rightW).Height(lay.colHeight).MaxHeight(lay.colHeight + 2).Render(rightCol)

	columns := lipgloss.JoinHorizontal(lipgloss.Top, left, mid, right)
	return lay.header + "\n" + columns
}

// RenderRoleViewOverlay renders a centered popup displaying role key-value data.
func RenderRoleViewOverlay(name string, data map[string]any, scroll int, width, height int) string {
	boxW := max(width*60/100, 50)
	boxH := max(height*60/100, 10)

	title := TitleStyle.Render("Role: " + name)

	keys := sortedMapKeys(data)

	var lines []string
	for _, k := range keys {
		lines = append(lines, TableKeyStyle.Render(k)+"  "+TableValueStyle.Render(fmt.Sprintf("%v", data[k])))
	}
	if len(lines) == 0 {
		lines = append(lines, HelpDescStyle.Render("(no data)"))
	}

	contentH := max(boxH-8, 1)
	visible := lines
	if scroll > 0 && scroll < len(visible) {
		visible = visible[scroll:]
	}
	if len(visible) > contentH {
		visible = visible[:contentH]
	}

	panelContent := strings.Join(visible, "\n")
	panelW := boxW - 6
	innerPanel := innerPanelStyle.Width(panelW).Height(contentH).Render(panelContent)

	body := title + "\n" + innerPanel
	return overlayBoxStyle.Width(boxW).Height(boxH).Render(body)
}

// renderMapTablePreview renders a map as a Key/Value table matching the secret preview style.
func renderMapTablePreview(data map[string]any, width, height int) string {
	if data == nil {
		return DimText("Loading...", width)
	}
	if len(data) == 0 {
		return DimText("(no data)", width)
	}

	sepW := max(width-2, 4)

	var lines []string
	lines = append(lines, TableHeaderStyle.Render("Key")+"  "+TableHeaderStyle.Render("Value"))
	lines = append(lines, strings.Repeat("─", sepW))

	keys := sortedMapKeys(data)
	limit := min(height-2, len(keys)) // -2 for header + separator
	for i := range limit {
		k := keys[i]
		v := fmt.Sprintf("%v", data[k])
		v = truncate(v, max(width-len(k)-4, 4))
		lines = append(lines, TableKeyStyle.Render(k)+"  "+TableValueStyle.Render(v))
	}
	if len(keys) > limit {
		lines = append(lines, HelpDescStyle.Render(fmt.Sprintf("... +%d more", len(keys)-limit)))
	}
	return strings.Join(lines, "\n")
}

func renderRoleDataPreview(data map[string]any, width, height int) string {
	return renderMapTablePreview(data, width, height)
}

// resolveAliasDisplayName returns the display_name if enriched, otherwise falls back to name.
func resolveAliasDisplayName(aMap map[string]any) string {
	if dn, ok := aMap["display_name"]; ok {
		return fmt.Sprintf("%v", dn)
	}
	return fmt.Sprintf("%v", aMap["name"])
}

func sortedMapKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// renderEntityDataPreview renders entity data in table format with aliases shown first.
func renderEntityDataPreview(data map[string]any, width, height int) string {
	if data == nil {
		return DimText("Loading...", width)
	}
	if len(data) == 0 {
		return DimText("(no data)", width)
	}

	sepW := max(width-2, 4)
	var lines []string

	// Show aliases section first
	if aliasesRaw, ok := data["aliases"]; ok {
		if aliases, ok := aliasesRaw.([]any); ok && len(aliases) > 0 {
			lines = append(lines, TableHeaderStyle.Render("Aliases"))
			lines = append(lines, strings.Repeat("─", sepW))
			for _, aRaw := range aliases {
				aMap, ok := aRaw.(map[string]any)
				if !ok {
					continue
				}
				aliasName := resolveAliasDisplayName(aMap)
				mountType := ""
				if mt, ok := aMap["mount_type"]; ok {
					mountType = fmt.Sprintf("%v", mt)
				}
				detail := aliasName
				if mountType != "" {
					detail += "  (" + mountType + ")"
				}
				lines = append(lines, TableValueStyle.Render(truncate(detail, width-2)))
			}
			lines = append(lines, "")
		}
	}

	// Properties table
	lines = append(lines, TableHeaderStyle.Render("Key")+"  "+TableHeaderStyle.Render("Value"))
	lines = append(lines, strings.Repeat("─", sepW))

	keys := sortedMapKeys(data)
	for _, k := range keys {
		if k == "aliases" {
			continue
		}
		if len(lines) >= height-1 {
			lines = append(lines, HelpDescStyle.Render("..."))
			break
		}
		v := fmt.Sprintf("%v", data[k])
		v = truncate(v, max(width-len(k)-4, 4))
		lines = append(lines, TableKeyStyle.Render(k)+"  "+TableValueStyle.Render(v))
	}

	limit := min(height, len(lines))
	return strings.Join(lines[:limit], "\n")
}

// RenderEntityList renders a three-column entity browser.
func RenderEntityList(
	parentEntries []model.Entry, parentIdx int,
	entities []model.Entry, cursor int, preview map[string]any,
	tabLabels []string, activeTab int,
	version string, width, height int,
) string {
	lay := computeThreeColumnLayout("Entities/", version, tabLabels, activeTab, width, height)

	leftCol := renderEntryList(parentEntries, parentIdx, nil, lay.leftW, lay.colHeight, false, "")
	midCol := renderEntryList(entities, cursor, nil, lay.midW, lay.colHeight, true, "")
	rightCol := renderEntityDataPreview(preview, lay.rightW, lay.colHeight)

	left := ColumnStyle.Width(lay.leftW).Height(lay.colHeight).MaxHeight(lay.colHeight + 2).Render(leftCol)
	mid := ActiveColumnStyle.Width(lay.midW).Height(lay.colHeight).MaxHeight(lay.colHeight + 2).Render(midCol)
	right := ColumnStyle.Width(lay.rightW).Height(lay.colHeight).MaxHeight(lay.colHeight + 2).Render(rightCol)

	columns := lipgloss.JoinHorizontal(lipgloss.Top, left, mid, right)
	return lay.header + "\n" + columns
}

// RenderEntityViewOverlay renders a centered popup displaying entity data.
func RenderEntityViewOverlay(name string, data map[string]any, scroll int, width, height int) string {
	boxW := max(width*60/100, 50)
	boxH := max(height*60/100, 10)

	title := TitleStyle.Render("Entity: " + name)

	lines := renderEntityFields(data)

	contentH := max(boxH-8, 1)
	visible := lines
	if scroll > 0 && scroll < len(visible) {
		visible = visible[scroll:]
	}
	if len(visible) > contentH {
		visible = visible[:contentH]
	}

	panelContent := strings.Join(visible, "\n")
	panelW := boxW - 6
	innerPanel := innerPanelStyle.Width(panelW).Height(contentH).Render(panelContent)

	body := title + "\n" + innerPanel
	return overlayBoxStyle.Width(boxW).Height(boxH).Render(body)
}

func renderEntityFields(data map[string]any) []string {
	if len(data) == 0 {
		return []string{HelpDescStyle.Render("(no data)")}
	}

	var lines []string

	if aliasesRaw, ok := data["aliases"]; ok {
		if aliases, ok := aliasesRaw.([]any); ok && len(aliases) > 0 {
			lines = append(lines, TableHeaderStyle.Render("Aliases:"))
			for i, aRaw := range aliases {
				aMap, ok := aRaw.(map[string]any)
				if !ok {
					continue
				}
				aliasName := resolveAliasDisplayName(aMap)
				mountType := ""
				if mt, ok := aMap["mount_type"]; ok {
					mountType = fmt.Sprintf("%v", mt)
				}
				mountPath := ""
				if mp, ok := aMap["mount_path"]; ok {
					mountPath = fmt.Sprintf("%v", mp)
				}

				lines = append(lines, fmt.Sprintf("  %d. %s", i+1, TableValueStyle.Render(aliasName)))
				if mountType != "" {
					lines = append(lines, "     "+TableKeyStyle.Render("type")+"  "+TableValueStyle.Render(mountType))
				}
				if mountPath != "" {
					lines = append(lines, "     "+TableKeyStyle.Render("mount")+"  "+TableValueStyle.Render(mountPath))
				}
				if id, ok := aMap["id"]; ok {
					lines = append(lines, "     "+TableKeyStyle.Render("id")+"  "+TableValueStyle.Render(fmt.Sprintf("%v", id)))
				}
			}
			lines = append(lines, "")
		}
	}

	keys := sortedMapKeys(data)
	for _, k := range keys {
		if k == "aliases" {
			continue
		}
		lines = append(lines, TableKeyStyle.Render(k)+"  "+TableValueStyle.Render(fmt.Sprintf("%v", data[k])))
	}

	if len(lines) == 0 {
		lines = append(lines, HelpDescStyle.Render("(no data)"))
	}
	return lines
}

// RenderGroupList renders a three-column group browser.
func RenderGroupList(
	parentEntries []model.Entry, parentIdx int,
	groups []model.Entry, cursor int, preview map[string]any,
	tabLabels []string, activeTab int,
	version string, width, height int,
) string {
	lay := computeThreeColumnLayout("Groups/", version, tabLabels, activeTab, width, height)

	leftCol := renderEntryList(parentEntries, parentIdx, nil, lay.leftW, lay.colHeight, false, "")
	midCol := renderEntryList(groups, cursor, nil, lay.midW, lay.colHeight, true, "")
	rightCol := renderRoleDataPreview(preview, lay.rightW, lay.colHeight)

	left := ColumnStyle.Width(lay.leftW).Height(lay.colHeight).MaxHeight(lay.colHeight + 2).Render(leftCol)
	mid := ActiveColumnStyle.Width(lay.midW).Height(lay.colHeight).MaxHeight(lay.colHeight + 2).Render(midCol)
	right := ColumnStyle.Width(lay.rightW).Height(lay.colHeight).MaxHeight(lay.colHeight + 2).Render(rightCol)

	columns := lipgloss.JoinHorizontal(lipgloss.Top, left, mid, right)
	return lay.header + "\n" + columns
}

// RenderGroupViewOverlay renders a centered popup displaying group data.
func RenderGroupViewOverlay(name string, data map[string]any, scroll int, width, height int) string {
	return RenderRoleViewOverlay(name, data, scroll, width, height)
}

// RenderTokenList renders a three-column token accessor browser.
func RenderTokenList(
	parentEntries []model.Entry, parentIdx int,
	accessors []model.Entry, cursor int, preview map[string]any,
	tabLabels []string, activeTab int,
	version string, width, height int,
) string {
	lay := computeThreeColumnLayout("Tokens/", version, tabLabels, activeTab, width, height)

	leftCol := renderEntryList(parentEntries, parentIdx, nil, lay.leftW, lay.colHeight, false, "")
	midCol := renderEntryList(accessors, cursor, nil, lay.midW, lay.colHeight, true, "")
	rightCol := renderMapTablePreview(preview, lay.rightW, lay.colHeight)

	left := ColumnStyle.Width(lay.leftW).Height(lay.colHeight).MaxHeight(lay.colHeight + 2).Render(leftCol)
	mid := ActiveColumnStyle.Width(lay.midW).Height(lay.colHeight).MaxHeight(lay.colHeight + 2).Render(midCol)
	right := ColumnStyle.Width(lay.rightW).Height(lay.colHeight).MaxHeight(lay.colHeight + 2).Render(rightCol)

	columns := lipgloss.JoinHorizontal(lipgloss.Top, left, mid, right)
	return lay.header + "\n" + columns
}

// RenderTokenCreatedOverlay renders a popup showing a newly created token.
// The client_token is only visible at creation time — emphasize copying it.
func RenderTokenCreatedOverlay(data map[string]any, width, height int) string {
	boxW := max(width*70/100, 50)
	boxH := max(height*50/100, 12)

	title := TitleStyle.Render("Token Created")

	var lines []string

	// Show client_token prominently first
	if token, ok := data["client_token"]; ok {
		lines = append(lines, "")
		lines = append(lines, TableHeaderStyle.Render("  TOKEN (copy now - cannot be retrieved later):"))
		lines = append(lines, "")
		lines = append(lines, "  "+TableValueStyle.Render(fmt.Sprintf("%v", token)))
		lines = append(lines, "")
		lines = append(lines, strings.Repeat("─", max(boxW-10, 4)))
		lines = append(lines, "")
	}

	// Show other fields
	for _, k := range []string{"accessor", "policies", "lease_duration", "renewable", "orphan"} {
		if v, ok := data[k]; ok {
			lines = append(lines, "  "+TableKeyStyle.Render(k)+"  "+TableValueStyle.Render(fmt.Sprintf("%v", v)))
		}
	}

	lines = append(lines, "")
	lines = append(lines, HelpDescStyle.Render("  Press any key to dismiss"))

	contentH := max(boxH-6, 1)
	if len(lines) > contentH {
		lines = lines[:contentH]
	}

	panelContent := strings.Join(lines, "\n")
	panelW := boxW - 6
	innerPanel := innerPanelStyle.Width(panelW).Height(contentH).Render(panelContent)

	body := title + "\n" + innerPanel
	return overlayBoxStyle.Width(boxW).Height(boxH).Render(body)
}
