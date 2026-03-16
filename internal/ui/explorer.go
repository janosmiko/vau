package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/janosmiko/vau/internal/model"
)

// RenderTabBar renders the tab bar showing tab labels with the active tab highlighted.
// It ensures the active tab is always visible by windowing tabs around it.
func RenderTabBar(tabLabels []string, activeTab, width int) string {
	activeStyle := lipgloss.NewStyle().
		Foreground(colorSelected).
		Background(colorPrimary).
		Bold(true).
		Padding(0, 1)
	inactiveStyle := lipgloss.NewStyle().
		Foreground(colorDim).
		Padding(0, 1)
	separatorStyle := lipgloss.NewStyle().
		Foreground(colorBorder)
	sep := separatorStyle.Render(" │ ")
	sepW := lipgloss.Width(sep)

	maxBarW := width - 2 // leading space + margin

	// Truncate long paths to keep tabs compact
	maxPathLen := maxBarW / len(tabLabels)
	if maxPathLen < 8 {
		maxPathLen = 8
	}

	// Build rendered tabs and measure widths
	type renderedTab struct {
		text  string
		width int
	}
	tabs := make([]renderedTab, len(tabLabels))
	for i, path := range tabLabels {
		if len(path) > maxPathLen {
			path = "…" + path[len(path)-maxPathLen+1:]
		}
		label := fmt.Sprintf("%d %s", i+1, path)
		var text string
		if i == activeTab {
			text = activeStyle.Render(label)
		} else {
			text = inactiveStyle.Render(label)
		}
		tabs[i] = renderedTab{text: text, width: lipgloss.Width(text)}
	}

	// Try to show all tabs
	totalW := 0
	for i, t := range tabs {
		totalW += t.width
		if i < len(tabs)-1 {
			totalW += sepW
		}
	}

	if totalW <= maxBarW {
		// Everything fits
		var parts []string
		for i, t := range tabs {
			parts = append(parts, t.text)
			if i < len(tabs)-1 {
				parts = append(parts, sep)
			}
		}
		return " " + strings.Join(parts, "")
	}

	// Window around active tab: expand outward from active tab
	left := activeTab
	right := activeTab
	usedW := tabs[activeTab].width

	for {
		expanded := false
		// Try expanding left
		if left > 0 {
			needed := sepW + tabs[left-1].width
			if usedW+needed <= maxBarW {
				left--
				usedW += needed
				expanded = true
			}
		}
		// Try expanding right
		if right < len(tabs)-1 {
			needed := sepW + tabs[right+1].width
			if usedW+needed <= maxBarW {
				right++
				usedW += needed
				expanded = true
			}
		}
		if !expanded {
			break
		}
	}

	var parts []string
	if left > 0 {
		parts = append(parts, inactiveStyle.Render("◂"))
		parts = append(parts, sep)
	}
	for i := left; i <= right; i++ {
		parts = append(parts, tabs[i].text)
		if i < right {
			parts = append(parts, sep)
		}
	}
	if right < len(tabs)-1 {
		parts = append(parts, sep)
		parts = append(parts, inactiveStyle.Render("▸"))
	}

	return " " + strings.Join(parts, "")
}

// RenderExplorer renders the 3-column explorer view.
func RenderExplorer(
	parentEntries []model.Entry,
	currentEntries []model.Entry,
	previewEntries []model.Entry,
	previewSecret *model.Secret,
	previewMode model.PreviewMode,
	parentIdx int,
	currentIdx int,
	selected map[int]bool,
	path []string,
	mount string,
	tabLabels []string, activeTab int,
	highlightQuery string,
	version string,
	width, height int,
) string {
	// Breadcrumb
	breadcrumb := mount + "/"
	if len(path) > 0 {
		breadcrumb += strings.Join(path, "")
	}

	// Version tag right-aligned on the breadcrumb line.
	versionTag := "vau " + version
	breadcrumbRendered := BreadcrumbStyle.Render(" " + breadcrumb)
	versionRendered := StatusStyle.Render(versionTag)
	breadcrumbWidth := lipgloss.Width(breadcrumbRendered)
	versionWidth := lipgloss.Width(versionRendered)
	padding := width - breadcrumbWidth - versionWidth
	if padding < 1 {
		padding = 1
	}
	breadcrumbLine := breadcrumbRendered + strings.Repeat(" ", padding) + versionRendered

	// Build header: breadcrumb first, tab bar below (only when 2+ tabs)
	headerLines := 1
	var header string
	if len(tabLabels) > 1 {
		header = breadcrumbLine + "\n" + RenderTabBar(tabLabels, activeTab, width)
		headerLines = 2
	} else {
		header = breadcrumbLine
	}

	// Calculate column widths (subtract borders and padding)
	usable := width - 6 // 3 columns × 2 border chars
	leftW := usable * 12 / 100
	midW := usable * 51 / 100
	rightW := usable - leftW - midW
	if leftW < 10 {
		leftW = 10
	}
	if midW < 10 {
		midW = 10
	}
	if rightW < 10 {
		rightW = 10
	}

	colHeight := height - 3 - headerLines // header lines + borders(2) + padding(1)

	leftCol := renderEntryList(parentEntries, parentIdx, nil, leftW, colHeight, false, "")
	midCol := renderEntryList(currentEntries, currentIdx, selected, midW, colHeight, true, highlightQuery)
	rightCol := renderPreview(previewEntries, previewSecret, previewMode, rightW, colHeight)

	left := ColumnStyle.Width(leftW).Height(colHeight).MaxHeight(colHeight + 2).Render(leftCol)
	mid := ActiveColumnStyle.Width(midW).Height(colHeight).MaxHeight(colHeight + 2).Render(midCol)
	right := ColumnStyle.Width(rightW).Height(colHeight).MaxHeight(colHeight + 2).Render(rightCol)

	columns := lipgloss.JoinHorizontal(lipgloss.Top, left, mid, right)

	return header + "\n" + columns
}

func renderEntryList(entries []model.Entry, selectedIdx int, selected map[int]bool, width, height int, isActive bool, highlight string) string {
	if len(entries) == 0 {
		return DimText("(empty)", width)
	}

	var lines []string

	// Calculate scroll offset for long lists
	start := 0
	if selectedIdx >= height {
		start = selectedIdx - height + 1
	}
	end := start + height
	if end > len(entries) {
		end = len(entries)
	}

	for i := start; i < end; i++ {
		e := entries[i]
		name := e.Name
		marker := "  "
		if selected != nil && selected[i] {
			marker = "● "
		}
		maxW := width - 2 - len(marker)
		if maxW < 4 {
			maxW = 4
		}
		if len(name) > maxW {
			name = name[:maxW-3] + "..."
		}

		var line string
		if i == selectedIdx && isActive {
			line = SelectedStyle.Render(marker) + highlightName(name, highlight, SelectedStyle, SelectedHighlightStyle)
			// Pad to full width
			rendered := marker + name
			pad := maxW + len(marker) - lipgloss.Width(rendered)
			if pad > 0 {
				line += SelectedStyle.Render(strings.Repeat(" ", pad))
			}
		} else if i == selectedIdx && !isActive {
			line = ParentSelectedStyle.Width(maxW + len(marker)).Render(marker + name)
		} else {
			var style lipgloss.Style
			if selected != nil && selected[i] {
				if e.IsDir {
					style = DirStyle
				} else {
					style = FileStyle
				}
			} else if e.IsDir {
				style = DirStyle
			} else {
				style = FileStyle
			}
			line = style.Render(marker) + highlightName(name, highlight, style, HighlightStyle)
		}
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

// highlightName renders a name with the matching portion highlighted.
// If query is empty or not found, renders the full name with the base style.
func highlightName(name, query string, baseStyle, hlStyle lipgloss.Style) string {
	if query == "" {
		return baseStyle.Render(name)
	}
	lower := strings.ToLower(name)
	q := strings.ToLower(query)
	idx := strings.Index(lower, q)
	if idx < 0 {
		return baseStyle.Render(name)
	}
	before := name[:idx]
	match := name[idx : idx+len(query)]
	after := name[idx+len(query):]
	return baseStyle.Render(before) + hlStyle.Render(match) + baseStyle.Render(after)
}

func renderPreview(entries []model.Entry, secret *model.Secret, previewMode model.PreviewMode, width, height int) string {
	if secret != nil {
		return renderSecretPreview(secret, previewMode, width, height)
	}

	if len(entries) == 0 {
		return DimText("(empty)", width)
	}

	var lines []string
	// Reserve 1 line for "more" indicator if needed
	limit := height
	hasMore := len(entries) > height
	if hasMore {
		limit = height - 1
	}
	if limit > len(entries) {
		limit = len(entries)
	}
	for i := 0; i < limit; i++ {
		e := entries[i]
		name := e.Name
		if len(name) > width-1 {
			name = name[:width-4] + "..."
		}
		if e.IsDir {
			lines = append(lines, DirStyle.Render(name))
		} else {
			lines = append(lines, FileStyle.Render(name))
		}
	}
	if hasMore {
		lines = append(lines, HelpDescStyle.Render(fmt.Sprintf("... +%d more", len(entries)-limit)))
	}

	return strings.Join(lines, "\n")
}

func renderSecretPreview(secret *model.Secret, previewMode model.PreviewMode, width, height int) string {
	if previewMode == model.PreviewJSON {
		return renderSecretJSON(secret, height)
	}

	sepW := width - 2
	if sepW < 4 {
		sepW = 4
	}

	var lines []string
	lines = append(lines, TableHeaderStyle.Render("Key")+"  "+TableHeaderStyle.Render("Value"))
	lines = append(lines, strings.Repeat("─", sepW))

	limit := height - 2
	if limit > len(secret.Keys) {
		limit = len(secret.Keys)
	}
	for i := 0; i < limit; i++ {
		k := secret.Keys[i]
		keyStr := TableKeyStyle.Render(k)
		var valStr string
		if previewMode == model.PreviewValues {
			v := secret.Data[k]
			valStr = TableValueStyle.Render(truncate(v, width-len(k)-4))
		} else {
			valStr = HiddenValueStyle.Render("********")
		}
		lines = append(lines, keyStr+"  "+valStr)
	}

	return strings.Join(lines, "\n")
}

func renderSecretJSON(secret *model.Secret, height int) string {
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

// DimText renders dimmed placeholder text.
func DimText(text string, _ int) string {
	return HelpDescStyle.Render(text)
}
