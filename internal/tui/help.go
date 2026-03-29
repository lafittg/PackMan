package tui

import (
	"strings"
)

func renderHelp(width int) string {
	bindings := []struct {
		key  string
		desc string
	}{
		{"↑/k", "Move up"},
		{"↓/j", "Move down"},
		{"Enter", "View package detail"},
		{"Esc", "Close detail / cancel filter"},
		{"/", "Filter packages by name"},
		{"s", "Cycle sort column"},
		{"S", "Reverse sort order"},
		{"Tab", "Switch ecosystem tab"},
		{"PgUp/Ctrl+U", "Page up"},
		{"PgDn/Ctrl+D", "Page down"},
		{"?", "Toggle help"},
		{"q/Ctrl+C", "Quit"},
	}

	var b strings.Builder
	b.WriteString(detailTitle.Render("  Keyboard Shortcuts") + "\n\n")

	for _, bind := range bindings {
		key := helpKeyStyle.Width(16).Render(bind.key)
		desc := helpDescStyle.Render(bind.desc)
		b.WriteString("  " + key + desc + "\n")
	}

	maxWidth := width - 6
	if maxWidth < 40 {
		maxWidth = 40
	}

	return detailBorder.Width(maxWidth).Render(b.String())
}
