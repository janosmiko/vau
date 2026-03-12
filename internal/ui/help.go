package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
	"github.com/janosmiko/vau/internal/model"
)

type helpBinding struct {
	key  string
	desc string
}

// HelpBar returns the help text for the current mode.
func HelpBar(mode model.ViewMode, width int) string {
	var bindings []helpBinding

	switch mode {
	case model.ModeExplorer:
		bindings = []helpBinding{
			{"hjkl", "nav"},
			{"Enter", "open"},
			{"/", "search"},
			{"f", "filter"},
			{"a", "new"},
			{"e", "edit"},
			{"r", "rename"},
			{"D", "del"},
			{"y/p/x", "yank/paste/cut"},
			{"u", "undo"},
			{"b/B", "bookmark"},
			{"T", "theme"},
			{"?", "help"},
			{"q", "quit"},
		}
	case model.ModeSecret, model.ModeSecretEdit, model.ModeVersionHistory, model.ModeHelp, model.ModeThemePicker:
		// Help is embedded in the overlay, just show minimal bar
		bindings = []helpBinding{}
	case model.ModeConfirm:
		bindings = []helpBinding{
			{"Enter", "confirm"},
			{"Esc", "cancel"},
		}
	case model.ModeInput:
		bindings = []helpBinding{
			{"Enter", "confirm"},
			{"Esc", "cancel"},
		}
	case model.ModeSearch:
		bindings = []helpBinding{
			{"Enter", "jump to match"},
			{"Esc", "cancel"},
		}
	case model.ModeFilter:
		bindings = []helpBinding{
			{"Enter", "apply filter"},
			{"Esc", "clear filter"},
		}
	case model.ModeJumpPath:
		bindings = []helpBinding{
			{"Tab", "complete"},
			{"Enter", "go"},
			{"Esc", "cancel"},
		}
	case model.ModeBookmark:
		bindings = []helpBinding{
			{"enter", "jump"},
			{"ctrl+d", "delete"},
			{"esc", "close"},
		}
	}

	sep := "  |  "
	usable := width - 2 // padding
	var parts []string
	used := 0
	for _, b := range bindings {
		entry := b.key + " " + b.desc
		entryW := len(entry)
		extra := 0
		if len(parts) > 0 {
			extra = len(sep)
		}
		if used+extra+entryW > usable {
			break
		}
		parts = append(parts,
			HelpKeyStyle.Render(b.key)+" "+HelpDescStyle.Render(b.desc))
		used += extra + entryW
	}

	line := strings.Join(parts, HelpDescStyle.Render(sep))
	return StatusStyle.Width(width).Render(line)
}

// helpSection groups keybindings under a section header.
type helpSection struct {
	title    string
	bindings []helpBinding
}

// helpSections returns all help sections with their keybindings.
func helpSections() []helpSection {
	return []helpSection{
		{
			title: "Explorer",
			bindings: []helpBinding{
				{"h / Left", "Navigate to parent directory"},
				{"j / Down", "Move cursor down"},
				{"k / Up", "Move cursor up"},
				{"l / Right", "Enter directory (dirs only)"},
				{"Enter", "Enter directory or open secret popup"},
				{"g", "Go to top of list"},
				{"G", "Go to bottom of list"},
				{"v", "Toggle secret values in preview"},
				{"V", "Toggle JSON view in preview"},
				{"Space", "Toggle selection (bulk ops)"},
				{"/", "Search (jump to first match)"},
				{"J", "Jump to Vault path (with tab completion)"},
				{"f", "Filter (hide non-matching)"},
				{"e", "Edit secret in external editor"},
				{"a", "Create new secret (inline editor)"},
				{"A", "Create new secret (external editor)"},
				{"r", "Rename / move secret"},
				{"y", "Yank (copy) secret data"},
				{"p", "Paste yanked secret"},
				{"x", "Cut (yank for move)"},
				{"D", "Delete (with confirmation)"},
				{"u", "Undo last action"},
				{"Ctrl+R", "Redo previous undo"},
				{"Ctrl+D", "Half-page scroll down"},
				{"Ctrl+U", "Half-page scroll up"},
				{"Ctrl+F", "Full-page scroll down"},
				{"Ctrl+B", "Full-page scroll up"},
				{"R", "Refresh listing"},
				{"t", "Create new tab (clone current)"},
				{"[", "Switch to previous tab"},
				{"]", "Switch to next tab"},
				{"Ctrl+C", "Close tab (quit if last)"},
				{"b", "Show bookmarks"},
				{"B", "Save bookmark"},
				{"T", "Change colorscheme"},
				{"?", "Show this help screen"},
				{"q", "Quit"},
			},
		},
		{
			title: "Mount Selection",
			bindings: []helpBinding{
				{"j / k", "Navigate mounts"},
				{"l / Enter", "Select mount"},
			},
		},
		{
			title: "Secret Popup",
			bindings: []helpBinding{
				{"j / k", "Navigate keys"},
				{"v / Tab", "Toggle value visibility"},
				{"V", "Toggle JSON view"},
				{"b", "Toggle base64 decode"},
				{"y", "Copy value to clipboard"},
				{"p", "Paste clipboard as value"},
				{"e", "Edit value (inline / ext. editor in JSON view)"},
				{"a", "Add new key-value pair (inline)"},
				{"H", "View version history"},
				{"D", "Delete selected key"},
				{"Ctrl+D", "Half-page scroll down"},
				{"Ctrl+U", "Half-page scroll up"},
				{"Ctrl+F", "Full-page scroll down"},
				{"Ctrl+B", "Full-page scroll up"},
				{"Esc / q / h", "Close popup"},
			},
		},
		{
			title: "Inline Edit (Secret Popup)",
			bindings: []helpBinding{
				{"Tab / Shift+Tab", "Switch between key and value columns"},
				{"Enter", "Save changes"},
				{"Esc", "Cancel edit"},
			},
		},
		{
			title: "Search (/)",
			bindings: []helpBinding{
				{"(type)", "Jump cursor to first match"},
				{"Enter", "Confirm position"},
				{"Esc", "Cancel (restore cursor)"},
			},
		},
		{
			title: "Filter (f)",
			bindings: []helpBinding{
				{"(type)", "Live filter entries"},
				{"Enter", "Confirm (keep filter)"},
				{"Esc", "Cancel (clear filter)"},
			},
		},
		{
			title: "Jump to Path (J)",
			bindings: []helpBinding{
				{"(type)", "Enter a Vault path"},
				{"Tab", "Autocomplete path segment"},
				{"Shift+Tab", "Cycle completions backward"},
				{"Enter", "Navigate to path"},
				{"Esc", "Cancel"},
			},
		},
		{
			title: "Bookmarks (b)",
			bindings: []helpBinding{
				{"j / k", "Navigate"},
				{"Enter / l", "Jump to bookmark"},
				{"/", "Filter bookmarks"},
				{"d", "Delete bookmark"},
				{"D", "Delete all bookmarks"},
				{"Esc", "Close (clear filter first)"},
			},
		},
		{
			title: "Help (?)",
			bindings: []helpBinding{
				{"j / k", "Scroll up / down"},
				{"Ctrl+D / Ctrl+U", "Half-page scroll down / up"},
				{"Ctrl+F / Ctrl+B", "Full-page scroll down / up"},
				{"g / G", "Go to top / bottom"},
				{"/", "Search keybindings"},
				{"Esc / ? / q", "Close help"},
			},
		},
	}
}

// buildHelpLines builds the formatted help lines, optionally filtering by a query string.
func buildHelpLines(filter string) []string {
	sections := helpSections()
	lowerFilter := strings.ToLower(filter)

	var lines []string
	for si, section := range sections {
		var sectionLines []string
		keyW := 20
		for _, b := range section.bindings {
			if filter != "" {
				lowerKey := strings.ToLower(b.key)
				lowerDesc := strings.ToLower(b.desc)
				if !strings.Contains(lowerKey, lowerFilter) && !strings.Contains(lowerDesc, lowerFilter) {
					continue
				}
			}
			keyPart := HelpKeyStyle.Render(fmt.Sprintf("%-*s", keyW, b.key))
			descPart := HelpDescStyle.Render(b.desc)
			sectionLines = append(sectionLines, "    "+keyPart+"  "+descPart)
		}

		// Only include sections that have matching bindings
		if len(sectionLines) == 0 {
			continue
		}

		if len(lines) > 0 || si > 0 {
			if len(lines) > 0 {
				lines = append(lines, "")
			}
		}
		header := lipgloss.NewStyle().
			Foreground(colorPrimary).
			Bold(true).
			Underline(true).
			Render(section.title)
		lines = append(lines, "  "+header)
		lines = append(lines, sectionLines...)
	}

	if filter != "" && len(lines) == 0 {
		lines = append(lines, HelpDescStyle.Render("  No matching keybindings"))
	}

	return lines
}

// HelpContentLineCount returns the total number of content lines for the help screen
// with the given filter. Used by the app model to calculate max scroll.
func HelpContentLineCount(filter string) int {
	return len(buildHelpLines(filter))
}

// RenderHelpScreen renders a full help overlay with all keybindings.
// It supports scrolling via the scroll parameter and filtering via the filter parameter.
func RenderHelpScreen(screenWidth, screenHeight, scroll int, filter string, searching bool, searchInput *textinput.Model) string {
	boxW := screenWidth * 70 / 100
	boxH := screenHeight * 80 / 100
	if boxW < 50 {
		boxW = 50
	}
	if boxH < 20 {
		boxH = 20
	}

	contentW := boxW - 6 // account for border + padding

	title := TitleStyle.Render("Keybindings")

	lines := buildHelpLines(filter)
	totalLines := len(lines)

	// Calculate visible area
	maxLines := max(boxH-6, 5) // title, borders, padding, help line

	// Clamp scroll
	maxScroll := max(totalLines-maxLines, 0)
	if scroll > maxScroll {
		scroll = maxScroll
	}
	if scroll < 0 {
		scroll = 0
	}

	// Determine scroll indicators
	hasAbove := scroll > 0
	hasBelow := scroll+maxLines < totalLines

	// Reserve lines for scroll indicators if needed
	visibleLines := maxLines
	if hasAbove {
		visibleLines--
	}
	if hasBelow {
		visibleLines--
	}

	// Slice visible portion
	end := min(scroll+visibleLines, totalLines)
	visible := lines[scroll:end]

	// Build final lines with indicators
	var displayLines []string
	if hasAbove {
		displayLines = append(displayLines, HelpDescStyle.Render("  ↑ more above"))
	}
	displayLines = append(displayLines, visible...)
	if hasBelow {
		displayLines = append(displayLines, HelpDescStyle.Render("  ↓ more below"))
	}

	content := strings.Join(displayLines, "\n")
	innerPanel := innerPanelStyle.
		Width(contentW).
		Render(content)

	// Build help/status line
	var helpLine string
	if searching {
		helpLine = HelpDescStyle.Render("search: ") + searchInput.View()
	} else if filter != "" {
		helpLine = HelpDescStyle.Render("filter: ") +
			HelpKeyStyle.Render(filter) +
			HelpDescStyle.Render("  ") +
			HelpKeyStyle.Render("/") + HelpDescStyle.Render(" edit  ") +
			HelpKeyStyle.Render("Esc") + HelpDescStyle.Render(" close")
	} else {
		helpLine = HelpKeyStyle.Render("j/k") + HelpDescStyle.Render(" scroll  ") +
			HelpKeyStyle.Render("^d/^u") + HelpDescStyle.Render(" half-page  ") +
			HelpKeyStyle.Render("/") + HelpDescStyle.Render(" search  ") +
			HelpKeyStyle.Render("Esc") + HelpDescStyle.Render(" / ") +
			HelpKeyStyle.Render("?") + HelpDescStyle.Render(" / ") +
			HelpKeyStyle.Render("q") + HelpDescStyle.Render(" close")
	}

	body := title + "\n" + innerPanel + "\n" + helpLine

	return overlayBoxStyle.
		Width(boxW).
		Render(body)
}
