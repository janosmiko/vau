package ui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/janosmiko/vau/internal/config"
)

var (
	// Colors
	colorPrimary   = lipgloss.Color("#7aa2f7")
	colorSecondary = lipgloss.Color("#9ece6a")
	colorDir       = lipgloss.Color("#7aa2f7")
	colorFile      = lipgloss.Color("#c0caf5")
	colorSelected  = lipgloss.Color("#1a1b26")
	colorSelBg     = lipgloss.Color("#7aa2f7")
	colorBorder    = lipgloss.Color("#3b4261")
	colorDim       = lipgloss.Color("#565f89")
	colorError     = lipgloss.Color("#f7768e")
	colorWarn      = lipgloss.Color("#e0af68")
	colorHidden    = lipgloss.Color("#565f89")
	colorHighlight = lipgloss.Color("#e0af68") // search match highlight

	// Column styles
	ColumnStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			Padding(0, 1)

	ActiveColumnStyle = lipgloss.NewStyle().
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(colorPrimary).
				Padding(0, 1)

	// Entry styles
	DirStyle = lipgloss.NewStyle().
			Foreground(colorDir).
			Bold(true)

	FileStyle = lipgloss.NewStyle().
			Foreground(colorFile)

	SelectedStyle = lipgloss.NewStyle().
			Foreground(colorSelected).
			Background(colorSelBg).
			Bold(true)

	// Parent column selected entry (subtle background highlight)
	ParentSelectedStyle = lipgloss.NewStyle().
				Foreground(colorFile).
				Background(colorBorder).
				Bold(true)

	// Breadcrumb
	BreadcrumbStyle = lipgloss.NewStyle().
			Foreground(colorPrimary).
			Bold(true).
			Padding(0, 1)

	// Search highlight
	HighlightStyle = lipgloss.NewStyle().
			Foreground(colorHighlight).
			Bold(true).
			Underline(true)

	// Search highlight on selected (cursor) row
	SelectedHighlightStyle = lipgloss.NewStyle().
				Foreground(colorError).
				Background(colorSelBg).
				Bold(true).
				Underline(true)

	// Status bar
	StatusStyle = lipgloss.NewStyle().
			Foreground(colorDim).
			Padding(0, 1)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(colorError).
			Bold(true).
			Padding(0, 1)

	// Help bar
	HelpKeyStyle = lipgloss.NewStyle().
			Foreground(colorSecondary).
			Bold(true)

	HelpDescStyle = lipgloss.NewStyle().
			Foreground(colorDim)

	// Secret view
	TableHeaderStyle = lipgloss.NewStyle().
				Foreground(colorPrimary).
				Bold(true).
				Underline(true)

	TableKeyStyle = lipgloss.NewStyle().
			Foreground(colorSecondary)

	TableValueStyle = lipgloss.NewStyle().
			Foreground(colorFile)

	HiddenValueStyle = lipgloss.NewStyle().
				Foreground(colorHidden)

	Base64ValueStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#bb9af7"))

	// Confirm dialog
	ConfirmStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(colorWarn).
			Padding(1, 2).
			Align(lipgloss.Center)

	// Input prompt
	InputLabelStyle = lipgloss.NewStyle().
			Foreground(colorPrimary).
			Bold(true)

	InputStyle = lipgloss.NewStyle().
			Foreground(colorFile)

	// Title
	TitleStyle = lipgloss.NewStyle().
			Foreground(colorPrimary).
			Bold(true).
			Padding(0, 1)
)

// ApplyTheme applies a built-in colorscheme as the base palette, then
// overlays any non-empty fields from the overrides ThemeConfig on top.
// If colorscheme is empty, only the overrides are applied (preserving
// the current defaults). This must be called before the TUI starts
// rendering, and may be called again at runtime to switch themes.
func ApplyTheme(colorscheme string, overrides config.ThemeConfig) {
	// Start from built-in theme if specified.
	theme := overrides
	if colorscheme != "" {
		if base, ok := GetTheme(colorscheme); ok {
			theme = mergeTheme(base, overrides)
		}
	}

	if theme.Primary != "" {
		colorPrimary = lipgloss.Color(theme.Primary)
	}
	if theme.Dir != "" {
		colorDir = lipgloss.Color(theme.Dir)
	}
	if theme.File != "" {
		colorFile = lipgloss.Color(theme.File)
	}
	if theme.Selected != "" {
		colorSelected = lipgloss.Color(theme.Selected)
	}
	if theme.Border != "" {
		colorBorder = lipgloss.Color(theme.Border)
	}
	if theme.Dim != "" {
		colorDim = lipgloss.Color(theme.Dim)
	}
	if theme.Error != "" {
		colorError = lipgloss.Color(theme.Error)
	}
	if theme.Warn != "" {
		colorWarn = lipgloss.Color(theme.Warn)
	}
	if theme.HiddenValue != "" {
		colorHidden = lipgloss.Color(theme.HiddenValue)
	}

	// Override individual style colors when a more specific theme field is set.
	breadcrumbColor := colorPrimary
	if theme.Breadcrumb != "" {
		breadcrumbColor = lipgloss.Color(theme.Breadcrumb)
	}
	tableKeyColor := colorSecondary
	if theme.TableKey != "" {
		tableKeyColor = lipgloss.Color(theme.TableKey)
	}
	tableValueColor := colorFile
	if theme.TableValue != "" {
		tableValueColor = lipgloss.Color(theme.TableValue)
	}
	tableHeaderColor := colorPrimary
	if theme.TableHeader != "" {
		tableHeaderColor = lipgloss.Color(theme.TableHeader)
	}
	helpKeyColor := colorSecondary
	if theme.HelpKey != "" {
		helpKeyColor = lipgloss.Color(theme.HelpKey)
	}
	helpDescColor := colorDim
	if theme.HelpDesc != "" {
		helpDescColor = lipgloss.Color(theme.HelpDesc)
	}
	statusBarColor := colorDim
	if theme.StatusBar != "" {
		statusBarColor = lipgloss.Color(theme.StatusBar)
	}
	titleColor := colorPrimary
	if theme.Title != "" {
		titleColor = lipgloss.Color(theme.Title)
	}
	base64Color := lipgloss.Color("#bb9af7")
	if theme.Base64Value != "" {
		base64Color = lipgloss.Color(theme.Base64Value)
	}

	// Rebuild all styles with the (potentially updated) colors.
	ColumnStyle = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(colorBorder).
		Padding(0, 1)

	ActiveColumnStyle = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(colorPrimary).
		Padding(0, 1)

	DirStyle = lipgloss.NewStyle().
		Foreground(colorDir).
		Bold(true)

	FileStyle = lipgloss.NewStyle().
		Foreground(colorFile)

	SelectedStyle = lipgloss.NewStyle().
		Foreground(colorSelected).
		Background(colorPrimary).
		Bold(true)

	ParentSelectedStyle = lipgloss.NewStyle().
		Foreground(colorFile).
		Background(colorBorder).
		Bold(true)

	BreadcrumbStyle = lipgloss.NewStyle().
		Foreground(breadcrumbColor).
		Bold(true).
		Padding(0, 1)

	HighlightStyle = lipgloss.NewStyle().
		Foreground(colorHighlight).
		Bold(true).
		Underline(true)

	SelectedHighlightStyle = lipgloss.NewStyle().
		Foreground(colorError).
		Background(colorPrimary).
		Bold(true).
		Underline(true)

	StatusStyle = lipgloss.NewStyle().
		Foreground(statusBarColor).
		Padding(0, 1)

	ErrorStyle = lipgloss.NewStyle().
		Foreground(colorError).
		Bold(true).
		Padding(0, 1)

	HelpKeyStyle = lipgloss.NewStyle().
		Foreground(helpKeyColor).
		Bold(true)

	HelpDescStyle = lipgloss.NewStyle().
		Foreground(helpDescColor)

	TableHeaderStyle = lipgloss.NewStyle().
		Foreground(tableHeaderColor).
		Bold(true).
		Underline(true)

	TableKeyStyle = lipgloss.NewStyle().
		Foreground(tableKeyColor)

	TableValueStyle = lipgloss.NewStyle().
		Foreground(tableValueColor)

	HiddenValueStyle = lipgloss.NewStyle().
		Foreground(colorHidden)

	Base64ValueStyle = lipgloss.NewStyle().
		Foreground(base64Color)

	ConfirmStyle = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(colorWarn).
		Padding(1, 2).
		Align(lipgloss.Center)

	InputLabelStyle = lipgloss.NewStyle().
		Foreground(colorPrimary).
		Bold(true)

	InputStyle = lipgloss.NewStyle().
		Foreground(colorFile)

	TitleStyle = lipgloss.NewStyle().
		Foreground(titleColor).
		Bold(true).
		Padding(0, 1)

	// Rebuild overlay styles that depend on theme colors.
	overlayBoxStyle = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(colorPrimary).
		Padding(1, 2)

	confirmBoxStyle = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(colorWarn).
		Padding(1, 2)
}

// mergeTheme returns a ThemeConfig where base provides defaults and
// any non-empty field in overrides takes precedence.
func mergeTheme(base, overrides config.ThemeConfig) config.ThemeConfig {
	merged := base
	if overrides.Primary != "" {
		merged.Primary = overrides.Primary
	}
	if overrides.Dir != "" {
		merged.Dir = overrides.Dir
	}
	if overrides.File != "" {
		merged.File = overrides.File
	}
	if overrides.Selected != "" {
		merged.Selected = overrides.Selected
	}
	if overrides.Border != "" {
		merged.Border = overrides.Border
	}
	if overrides.Dim != "" {
		merged.Dim = overrides.Dim
	}
	if overrides.Error != "" {
		merged.Error = overrides.Error
	}
	if overrides.Warn != "" {
		merged.Warn = overrides.Warn
	}
	if overrides.Breadcrumb != "" {
		merged.Breadcrumb = overrides.Breadcrumb
	}
	if overrides.TableKey != "" {
		merged.TableKey = overrides.TableKey
	}
	if overrides.TableValue != "" {
		merged.TableValue = overrides.TableValue
	}
	if overrides.TableHeader != "" {
		merged.TableHeader = overrides.TableHeader
	}
	if overrides.HiddenValue != "" {
		merged.HiddenValue = overrides.HiddenValue
	}
	if overrides.HelpKey != "" {
		merged.HelpKey = overrides.HelpKey
	}
	if overrides.HelpDesc != "" {
		merged.HelpDesc = overrides.HelpDesc
	}
	if overrides.StatusBar != "" {
		merged.StatusBar = overrides.StatusBar
	}
	if overrides.Title != "" {
		merged.Title = overrides.Title
	}
	if overrides.Base64Value != "" {
		merged.Base64Value = overrides.Base64Value
	}
	return merged
}
