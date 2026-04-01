package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/janosmiko/vau/internal/model"
)

// RenderPolicyList renders a three-column policy browser matching the explorer layout.
func RenderPolicyList(
	parentEntries []model.Entry, parentIdx int,
	policies []string, cursor int, preview string,
	tabLabels []string, activeTab int,
	version string, width, height int,
) string {
	lay := computeThreeColumnLayout("Policies/", version, tabLabels, activeTab, width, height)

	leftCol := renderEntryList(parentEntries, parentIdx, nil, lay.leftW, lay.colHeight, false, "")

	policyEntries := make([]model.Entry, len(policies))
	for i, p := range policies {
		policyEntries[i] = model.Entry{Name: p}
	}
	midCol := renderEntryList(policyEntries, cursor, nil, lay.midW, lay.colHeight, true, "")
	rightCol := renderHCLPreview(preview, lay.rightW, lay.colHeight)

	left := ColumnStyle.Width(lay.leftW).Height(lay.colHeight).MaxHeight(lay.colHeight + 2).Render(leftCol)
	mid := ActiveColumnStyle.Width(lay.midW).Height(lay.colHeight).MaxHeight(lay.colHeight + 2).Render(midCol)
	right := ColumnStyle.Width(lay.rightW).Height(lay.colHeight).MaxHeight(lay.colHeight + 2).Render(rightCol)

	columns := lipgloss.JoinHorizontal(lipgloss.Top, left, mid, right)
	return lay.header + "\n" + columns
}

func renderHCLPreview(preview string, width, height int) string {
	if preview == "" {
		return DimText("(no preview)", width)
	}

	sepW := max(width-2, 4)
	var lines []string
	lines = append(lines, TableHeaderStyle.Render("HCL Policy"))
	lines = append(lines, strings.Repeat("─", sepW))

	allLines := strings.Split(preview, "\n")
	limit := min(height-2, len(allLines)) // -2 for header + separator

	for i := range limit {
		line := allLines[i]
		if len(line) > width-2 {
			line = line[:width-5] + "..."
		}
		lines = append(lines, HelpDescStyle.Render(line))
	}
	if len(allLines) > limit {
		lines = append(lines, HelpDescStyle.Render("..."))
	}
	return strings.Join(lines, "\n")
}

// RenderPolicyViewOverlay renders a centered overlay popup displaying policy HCL.
func RenderPolicyViewOverlay(name string, hcl string, scroll int, width, height int) string {
	boxW := max(width*75/100, 50)
	boxH := max(height*75/100, 10)

	title := TitleStyle.Render("Policy: " + name)

	panelContentH := max(boxH-7, 3)
	panelContentW := max(boxW-10, 20)
	panelW := boxW - 6

	allLines := strings.Split(hcl, "\n")
	totalLines := len(allLines)

	scroll = max(scroll, 0)
	scroll = min(scroll, totalLines-1)
	end := min(scroll+panelContentH, totalLines)

	gutterW := max(len(fmt.Sprintf("%d", end)), 3)
	contentW := max(panelContentW-gutterW-3, 10)

	var lines []string
	for i := scroll; i < end; i++ {
		lineNum := HelpDescStyle.Render(fmt.Sprintf("%*d", gutterW, i+1))
		sep := HelpDescStyle.Render(" | ")
		content := allLines[i]
		if len(content) > contentW {
			content = content[:contentW-3] + "..."
		}
		lines = append(lines, lineNum+sep+HelpDescStyle.Render(content))
	}

	panelContent := strings.Join(lines, "\n")
	innerPanel := innerPanelStyle.
		Width(panelW).
		Height(panelContentH).
		Render(panelContent)

	body := title + "\n" + innerPanel
	return overlayBoxStyle.Width(boxW).Render(body)
}
