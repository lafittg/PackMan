package tui

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	colorGreen  = lipgloss.Color("#04B575")
	colorYellow = lipgloss.Color("#FBBF24")
	colorRed    = lipgloss.Color("#EF4444")
	colorBlue   = lipgloss.Color("#60A5FA")
	colorDim    = lipgloss.Color("#6B7280")
	colorWhite  = lipgloss.Color("#F9FAFB")
	colorBg     = lipgloss.Color("#1F2937")
	colorBorder = lipgloss.Color("#374151")

	// Header styles
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorWhite).
			Background(lipgloss.Color("#7C3AED")).
			Padding(0, 1)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(colorDim).
			Italic(true)

	// Summary bar
	summaryStyle = lipgloss.NewStyle().
			Foreground(colorWhite).
			Padding(0, 1)

	summaryValueStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorBlue)

	summaryWarnStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorRed)

	// Table styles
	headerCellStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorWhite)

	headerRowStyle = lipgloss.NewStyle().
			BorderBottom(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(colorBorder)

	selectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorWhite).
			Background(lipgloss.Color("#374151"))

	// Cell color styles
	cellGreen  = lipgloss.NewStyle().Foreground(colorGreen)
	cellYellow = lipgloss.NewStyle().Foreground(colorYellow)
	cellRed    = lipgloss.NewStyle().Foreground(colorRed)
	cellBlue   = lipgloss.NewStyle().Foreground(colorBlue)
	cellDim   = lipgloss.NewStyle().Foreground(colorDim)

	// Health indicators
	healthGood = lipgloss.NewStyle().Foreground(colorGreen).Bold(true).Render("● Good")
	healthWarn = lipgloss.NewStyle().Foreground(colorYellow).Bold(true).Render("● Warn")
	healthBad  = lipgloss.NewStyle().Foreground(colorRed).Bold(true).Render("● Bad ")

	// Detail panel
	detailBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7C3AED")).
			Padding(1, 2)

	detailTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7C3AED"))

	detailLabel = lipgloss.NewStyle().
			Foreground(colorDim).
			Width(20)

	detailValue = lipgloss.NewStyle().
			Foreground(colorWhite)

	// Help
	helpKeyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7C3AED")).
			Bold(true)

	helpDescStyle = lipgloss.NewStyle().
			Foreground(colorDim)

	// Footer
	footerStyle = lipgloss.NewStyle().
			Foreground(colorDim).
			BorderTop(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(colorBorder).
			Padding(0, 1)

	// Filter input
	filterPromptStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#7C3AED")).
				Bold(true)

	filterInputStyle = lipgloss.NewStyle().
				Foreground(colorWhite)

	// Spinner
	spinnerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7C3AED"))
)
