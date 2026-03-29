package tui

import (
	"fmt"

	"github.com/dustin/go-humanize"
	"github.com/gregoirelafitte/packman/pkg/types"
)

func renderSummary(report types.ProjectReport, width int) string {
	s := report.Summary

	parts := []string{
		summaryValueStyle.Render(fmt.Sprintf("%d", s.TotalDeps)) + summaryStyle.Render(" packages"),
	}

	if s.DevDepsCount > 0 {
		parts = append(parts,
			summaryStyle.Render("(") +
				cellDim.Render(fmt.Sprintf("%d dev", s.DevDepsCount)) +
				summaryStyle.Render(")"))
	}

	parts = append(parts,
		summaryStyle.Render(" │ ") +
			summaryValueStyle.Render(fmt.Sprintf("%d", s.TotalTransitiveDeps)) +
			summaryStyle.Render(" transitive"))

	if s.TotalInstallSize > 0 {
		parts = append(parts,
			summaryStyle.Render(" │ ") +
				summaryValueStyle.Render(humanize.IBytes(uint64(s.TotalInstallSize))) +
				summaryStyle.Render(" total"))
	}

	if s.UnusedCount > 0 {
		parts = append(parts,
			summaryStyle.Render(" │ ") +
				summaryWarnStyle.Render(fmt.Sprintf("%d unused", s.UnusedCount)))
	}

	if s.LowUsageCount > 0 {
		parts = append(parts,
			summaryStyle.Render(" │ ") +
				cellYellow.Render(fmt.Sprintf("%d low usage", s.LowUsageCount)))
	}

	result := ""
	for _, p := range parts {
		result += p
	}

	return result
}
