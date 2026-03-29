package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/dustin/go-humanize"
	"github.com/gregoirelafitte/packman/pkg/types"
)

func renderDetail(r types.AnalysisResult, width, height int) string {
	var b strings.Builder

	name := detailTitle.Render(fmt.Sprintf("  %s@%s", r.Dependency.Name, r.Cost.Version))
	b.WriteString(name + "\n\n")

	// Cost section
	b.WriteString(detailTitle.Render("  Cost Metrics") + "\n")

	rows := []struct {
		label string
		value string
	}{
		{"Install Size", formatSize(r.Cost.InstallSize)},
		{"Publish Size", formatSize(r.Cost.PublishSize)},
		{"Direct Deps", fmt.Sprintf("%d", r.Cost.DirectDeps)},
		{"Transitive Deps", fmt.Sprintf("%d", r.Cost.TransitiveDeps)},
		{"Est. Install Time", formatDuration(r.Cost.EstInstallTime)},
		{"Weekly Downloads", formatDownloads(r.Cost.WeeklyDownloads)},
	}

	for _, row := range rows {
		b.WriteString("  " + detailLabel.Render(row.label) + detailValue.Render(row.value) + "\n")
	}

	// Usage section
	b.WriteString("\n" + detailTitle.Render("  Usage Analysis") + "\n")

	var usageStr string
	switch r.Usage.Level {
	case types.UsageUnused:
		usageStr = cellRed.Render(r.Usage.Level.String())
	case types.UsageTooling:
		usageStr = cellBlue.Render(r.Usage.Level.String())
	case types.UsageLow:
		usageStr = cellYellow.Render(r.Usage.Level.String())
	default:
		usageStr = cellGreen.Render(r.Usage.Level.String())
	}

	b.WriteString("  " + detailLabel.Render("Status") + usageStr + "\n")
	b.WriteString("  " + detailLabel.Render("Files Importing") + detailValue.Render(fmt.Sprintf("%d", r.Usage.ImportCount)) + "\n")
	b.WriteString("  " + detailLabel.Render("Total Usages") + detailValue.Render(fmt.Sprintf("%d", r.Usage.UsageCount)) + "\n")

	// Import locations
	if len(r.Usage.ImportLocations) > 0 {
		b.WriteString("\n" + detailTitle.Render("  Import Locations") + "\n")
		maxLocations := 15
		for i, loc := range r.Usage.ImportLocations {
			if i >= maxLocations {
				remaining := len(r.Usage.ImportLocations) - maxLocations
				b.WriteString(cellDim.Render(fmt.Sprintf("    ... and %d more\n", remaining)))
				break
			}
			b.WriteString(cellDim.Render(fmt.Sprintf("    %s:%d\n", loc.FilePath, loc.Line)))
		}
	}

	// Transitive dependency tree
	if len(r.Cost.DepTree) > 0 {
		b.WriteString("\n" + detailTitle.Render("  Dependency Tree") + "\n")
		maxDeps := 20
		for i, dep := range r.Cost.DepTree {
			if i >= maxDeps {
				remaining := len(r.Cost.DepTree) - maxDeps
				b.WriteString(cellDim.Render(fmt.Sprintf("    ... and %d more\n", remaining)))
				break
			}
			b.WriteString(cellDim.Render(fmt.Sprintf("    ├── %s\n", dep)))
		}
	}

	// Health score
	b.WriteString("\n" + detailTitle.Render("  Health Score") + "\n")
	b.WriteString("  " + renderHealthBar(r.HealthScore) + "\n")

	content := b.String()
	maxWidth := width - 6
	if maxWidth < 40 {
		maxWidth = 40
	}

	return detailBorder.Width(maxWidth).Render(content)
}

func renderHealthBar(score float64) string {
	barWidth := 30
	filled := int(score * float64(barWidth))
	if filled > barWidth {
		filled = barWidth
	}

	var style lipgloss.Style
	switch {
	case score >= 0.7:
		style = cellGreen
	case score >= 0.4:
		style = cellYellow
	default:
		style = cellRed
	}

	bar := style.Render(strings.Repeat("█", filled)) + cellDim.Render(strings.Repeat("░", barWidth-filled))
	return fmt.Sprintf("%s %s", bar, style.Render(fmt.Sprintf("%.0f%%", score*100)))
}

func formatSize(bytes int64) string {
	if bytes == 0 {
		return "—"
	}
	return humanize.IBytes(uint64(bytes))
}

func formatDuration(d fmt.Stringer) string {
	s := d.String()
	if s == "0s" {
		return "—"
	}
	return s
}

func formatDownloads(n int64) string {
	if n == 0 {
		return "—"
	}
	return humanize.Comma(n)
}
