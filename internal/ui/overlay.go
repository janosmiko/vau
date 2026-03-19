package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/janosmiko/vau/internal/config"
)

var (
	overlayBoxStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(colorPrimary).
			Padding(1, 2)

	confirmBoxStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(colorWarn).
			Padding(1, 2)
)

// PlaceOverlay renders content centered on top of a background string.
// It replaces the lines in the background with the overlay content.
func PlaceOverlay(bg string, overlay string, bgWidth, bgHeight int) string {
	bgLines := strings.Split(bg, "\n")
	// Pad background to full height
	for len(bgLines) < bgHeight {
		bgLines = append(bgLines, "")
	}

	// Get overlay dimensions
	olLines := strings.Split(overlay, "\n")
	olWidth := lipgloss.Width(overlay)
	olHeight := len(olLines)

	// Center position
	startRow := (bgHeight - olHeight) / 2
	startCol := (bgWidth - olWidth) / 2
	if startRow < 0 {
		startRow = 0
	}
	if startCol < 0 {
		startCol = 0
	}

	for i, olLine := range olLines {
		row := startRow + i
		if row >= len(bgLines) {
			break
		}
		bgLine := bgLines[row]
		bgVisualWidth := lipgloss.Width(bgLine)

		// Build the new line: bg prefix + overlay + bg suffix
		prefix := ""
		if startCol > 0 {
			// Take the visual prefix from background
			prefix = takeVisualWidth(bgLine, startCol)
		}

		suffix := ""
		endCol := startCol + lipgloss.Width(olLine)
		if endCol < bgVisualWidth {
			suffix = skipVisualWidth(bgLine, endCol)
		}

		bgLines[row] = prefix + olLine + suffix
	}

	return strings.Join(bgLines[:bgHeight], "\n")
}

// RenderInputOverlay renders a centered input prompt overlay.
func RenderInputOverlay(label string, inputView string, width int) string {
	boxWidth := width / 2
	if boxWidth < 40 {
		boxWidth = 40
	}
	if boxWidth > width-4 {
		boxWidth = width - 4
	}

	content := InputLabelStyle.Render(label) + "\n\n" + inputView

	return overlayBoxStyle.Width(boxWidth).Render(content)
}

// RenderConfirmOverlay renders a centered confirmation dialog overlay.
// It displays the message and a text input where the user must type "DELETE" to confirm.
func RenderConfirmOverlay(message string, inputView string, width int) string {
	boxWidth := width / 2
	if boxWidth < 40 {
		boxWidth = 40
	}
	if boxWidth > width-4 {
		boxWidth = width - 4
	}

	content := message + "\n\n" +
		HelpDescStyle.Render("Type DELETE to confirm:") + "\n\n" +
		inputView + "\n\n" +
		HelpKeyStyle.Render("Enter") + HelpDescStyle.Render(" confirm") + "    " +
		HelpKeyStyle.Render("Esc") + HelpDescStyle.Render(" cancel")

	return confirmBoxStyle.Width(boxWidth).Align(lipgloss.Center).Render(content)
}

// RenderJumpPathOverlay renders a centered jump-to-path prompt with completions.
func RenderJumpPathOverlay(inputView string, completions []string, selectedIdx int, width, height int) string {
	boxWidth := width / 2
	if boxWidth < 50 {
		boxWidth = 50
	}
	if boxWidth > width-4 {
		boxWidth = width - 4
	}

	// Max width for entries (border 2 + padding 4 + prefix 2)
	maxEntryW := boxWidth - 8
	if maxEntryW < 10 {
		maxEntryW = 10
	}

	content := InputLabelStyle.Render("Jump to path:") + "\n\n" + inputView

	// Show completions if available
	if len(completions) > 0 {
		content += "\n\n"
		// Cap visible completions to fit terminal (label + input + gaps + help + border/padding ~ 8 lines)
		maxShow := height - 8
		if maxShow > 10 {
			maxShow = 10
		}
		if maxShow < 1 {
			maxShow = 1
		}
		if len(completions) < maxShow {
			maxShow = len(completions)
		}
		for i := 0; i < maxShow; i++ {
			entry := completions[i]
			// Truncate long paths
			displayEntry := entry
			if len(displayEntry) > maxEntryW {
				displayEntry = displayEntry[:maxEntryW-3] + "..."
			}
			if i == selectedIdx {
				content += SelectedStyle.Render(" "+displayEntry+" ") + "\n"
			} else {
				if strings.HasSuffix(entry, "/") {
					content += DirStyle.Render("  "+displayEntry) + "\n"
				} else {
					content += FileStyle.Render("  "+displayEntry) + "\n"
				}
			}
		}
		if len(completions) > maxShow {
			content += HelpDescStyle.Render(fmt.Sprintf("  ... and %d more", len(completions)-maxShow))
		}
	}

	// Help line
	content += "\n" +
		HelpKeyStyle.Render("Tab") + HelpDescStyle.Render(" complete  ") +
		HelpKeyStyle.Render("Enter") + HelpDescStyle.Render(" go  ") +
		HelpKeyStyle.Render("Esc") + HelpDescStyle.Render(" cancel")

	return overlayBoxStyle.Width(boxWidth).Render(content)
}

// RenderSearchOverlay renders a centered search/filter overlay.
func RenderSearchOverlay(label string, inputView string, width int) string {
	boxWidth := width / 2
	if boxWidth < 40 {
		boxWidth = 40
	}
	if boxWidth > width-4 {
		boxWidth = width - 4
	}

	content := InputLabelStyle.Render(label) + "\n\n" + inputView

	return overlayBoxStyle.Width(boxWidth).Render(content)
}

// RenderBookmarkOverlay renders a centered bookmark overlay with filter and cursor navigation.
func RenderBookmarkOverlay(allBookmarks []config.Bookmark, filter string, searching bool, cursor int, width, height int) string {
	boxWidth := width / 3
	if boxWidth < 40 {
		boxWidth = 40
	}
	if boxWidth > width-4 {
		boxWidth = width - 4
	}

	maxEntryW := boxWidth - 8
	if maxEntryW < 10 {
		maxEntryW = 10
	}

	content := InputLabelStyle.Render("Marks")

	// Show filter input only when searching or filter is active
	if searching {
		content += "\n\n" + HelpDescStyle.Render("/ ") + HelpKeyStyle.Render(filter+"_")
	} else if filter != "" {
		content += "\n\n" + HelpDescStyle.Render("/ ") + HelpKeyStyle.Render(filter)
	}

	// Filter bookmarks
	var filtered []config.Bookmark
	if filter == "" {
		filtered = allBookmarks
	} else {
		lf := strings.ToLower(filter)
		for _, bm := range allBookmarks {
			if strings.Contains(strings.ToLower(bm.Name), lf) {
				filtered = append(filtered, bm)
			}
		}
	}

	if len(allBookmarks) == 0 {
		content += "\n\n" + HelpDescStyle.Render("  No bookmarks yet")
		content += "\n" + HelpDescStyle.Render("  Press m + [a-z,0-9] to set a mark")
	} else if len(filtered) == 0 {
		content += "\n\n" + HelpDescStyle.Render("  No matching bookmarks")
	} else {
		content += "\n"
		maxShow := height - 10
		if maxShow < 3 {
			maxShow = 3
		}
		if maxShow > len(filtered) {
			maxShow = len(filtered)
		}

		// Scroll window around cursor
		start := 0
		if cursor >= maxShow {
			start = cursor - maxShow + 1
		}
		end := start + maxShow
		if end > len(filtered) {
			end = len(filtered)
		}

		for i := start; i < end; i++ {
			bm := filtered[i]
			name := bm.Name
			if len(name) > maxEntryW-4 {
				name = "..." + name[len(name)-maxEntryW+7:]
			}

			if i == cursor {
				// Selected: plain text with highlight background
				var prefix string
				if bm.Slot != "" {
					prefix = bm.Slot + " "
				} else {
					prefix = "  "
				}
				content += "\n" + SelectedStyle.Render(" "+prefix+name+" ")
			} else {
				// Non-selected: slot key in bold green
				var prefix string
				if bm.Slot != "" {
					prefix = HelpKeyStyle.Render(bm.Slot) + " "
				} else {
					prefix = "  "
				}
				content += "\n " + prefix + HelpDescStyle.Render(name)
			}
		}
		if len(filtered) > maxShow {
			content += "\n" + HelpDescStyle.Render(fmt.Sprintf("  ... %d total", len(filtered)))
		}
	}

	content += "\n\n" +
		HelpKeyStyle.Render("a-z/0-9") + HelpDescStyle.Render(" quick jump  ") +
		HelpKeyStyle.Render("enter") + HelpDescStyle.Render(" jump  ") +
		HelpKeyStyle.Render("/") + HelpDescStyle.Render(" filter  ") +
		HelpKeyStyle.Render("D") + HelpDescStyle.Render(" del  ") +
		HelpKeyStyle.Render("ctrl+x") + HelpDescStyle.Render(" del all  ") +
		HelpKeyStyle.Render("esc") + HelpDescStyle.Render(" close")

	return overlayBoxStyle.Width(boxWidth).Render(content)
}

// RenderThemePickerOverlay renders a centered colorscheme picker overlay
// with entries grouped by dark/light themes.
func RenderThemePickerOverlay(entries []ThemeEntry, cursor int, activeTheme string, width, height int) string {
	boxWidth := width / 2
	if boxWidth < 40 {
		boxWidth = 40
	}
	if boxWidth > width-4 {
		boxWidth = width - 4
	}

	maxEntryW := boxWidth - 8
	if maxEntryW < 10 {
		maxEntryW = 10
	}

	content := InputLabelStyle.Render("Colorscheme")

	if len(entries) == 0 {
		content += "\n\n" + HelpDescStyle.Render("  No themes available")
	} else {
		content += "\n"
		maxShow := height - 10
		if maxShow < 3 {
			maxShow = 3
		}
		if maxShow > len(entries) {
			maxShow = len(entries)
		}

		// Scroll window around cursor
		start := 0
		if cursor >= maxShow {
			start = cursor - maxShow + 1
		}
		end := start + maxShow
		if end > len(entries) {
			end = len(entries)
		}

		for i := start; i < end; i++ {
			entry := entries[i]

			if entry.IsHeader {
				// Render group header
				header := lipgloss.NewStyle().
					Foreground(colorPrimary).
					Bold(true).
					Render(entry.Name)
				content += "\n\n " + header
				continue
			}

			name := entry.Name
			if len(name) > maxEntryW {
				name = name[:maxEntryW]
			}

			marker := "  "
			if entry.Name == activeTheme {
				marker = "* "
			}

			if i == cursor {
				content += "\n" + SelectedStyle.Render(" "+marker+name+" ")
			} else {
				content += "\n" + HelpDescStyle.Render(" "+marker) + FileStyle.Render(name)
			}
		}
	}

	content += "\n\n" +
		HelpKeyStyle.Render("enter") + HelpDescStyle.Render(" select  ") +
		HelpKeyStyle.Render("j/k") + HelpDescStyle.Render(" navigate  ") +
		HelpKeyStyle.Render("esc") + HelpDescStyle.Render(" cancel")

	return overlayBoxStyle.Width(boxWidth).Render(content)
}

// takeVisualWidth returns the prefix of s up to n visual columns,
// preserving ANSI sequences.
func takeVisualWidth(s string, n int) string {
	if n <= 0 {
		return ""
	}
	col := 0
	var result []rune
	inEsc := false
	for _, r := range s {
		if r == '\033' {
			inEsc = true
			result = append(result, r)
			continue
		}
		if inEsc {
			result = append(result, r)
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '~' {
				inEsc = false
			}
			continue
		}
		if col >= n {
			break
		}
		result = append(result, r)
		col++
	}
	// Pad with spaces if bg is shorter than needed
	for col < n {
		result = append(result, ' ')
		col++
	}
	return string(result)
}

// skipVisualWidth returns the suffix of s after n visual columns,
// preserving ANSI sequences that appear after.
func skipVisualWidth(s string, n int) string {
	col := 0
	var result []rune
	inEsc := false
	skipping := true
	for _, r := range s {
		if r == '\033' {
			inEsc = true
			if !skipping {
				result = append(result, r)
			}
			continue
		}
		if inEsc {
			if !skipping {
				result = append(result, r)
			}
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '~' {
				inEsc = false
			}
			continue
		}
		if skipping {
			col++
			if col >= n {
				skipping = false
			}
			continue
		}
		result = append(result, r)
	}
	return string(result)
}
