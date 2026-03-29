package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/dustin/go-humanize"
	"github.com/gregoirelafitte/packman/pkg/types"
)

type sortColumn int

const (
	sortByName sortColumn = iota
	sortByVersion
	sortBySize
	sortByDeps
	sortByUsage
	sortByHealth
	sortColumnCount
)

func (s sortColumn) String() string {
	switch s {
	case sortByName:
		return "Name"
	case sortByVersion:
		return "Version"
	case sortBySize:
		return "Size"
	case sortByDeps:
		return "Deps"
	case sortByUsage:
		return "Usage"
	case sortByHealth:
		return "Health"
	default:
		return ""
	}
}

// columnWidths defines the width for each column.
var columnWidths = []int{28, 12, 12, 8, 12, 10}

func renderTableHeader(width int) string {
	headers := []string{"Name", "Version", "Size", "Deps", "Usage", "Health"}

	var cols []string
	for i, h := range headers {
		cols = append(cols, padRight(headerCellStyle.Render(h), columnWidths[i]))
	}

	row := strings.Join(cols, " ")
	return headerRowStyle.Render(row)
}

func truncate(s string, maxWidth int) string {
	if len(s) <= maxWidth {
		return s
	}
	if maxWidth <= 1 {
		return s[:maxWidth]
	}
	return s[:maxWidth-1] + "…"
}

func renderTableRow(r types.AnalysisResult, selected bool, width int) string {
	name := truncate(r.Dependency.Name, columnWidths[0])
	if r.Dependency.IsDev {
		name = cellDim.Render(name)
	}

	version := r.Cost.Version
	if version == "" {
		version = r.Dependency.Version
	}
	version = truncate(version, columnWidths[1])

	// Size cell with color
	var sizeStr string
	if r.Cost.InstallSize > 0 {
		sizeStr = humanize.IBytes(uint64(r.Cost.InstallSize))
		switch {
		case r.Cost.InstallSize < 100*1024:
			sizeStr = cellGreen.Render(sizeStr)
		case r.Cost.InstallSize < 1024*1024:
			sizeStr = cellYellow.Render(sizeStr)
		default:
			sizeStr = cellRed.Render(sizeStr)
		}
	} else {
		sizeStr = cellDim.Render("—")
	}

	// Deps cell
	depsStr := fmt.Sprintf("%d", r.Cost.TransitiveDeps)
	if r.Cost.TransitiveDeps > 20 {
		depsStr = cellRed.Render(depsStr)
	} else if r.Cost.TransitiveDeps > 5 {
		depsStr = cellYellow.Render(depsStr)
	} else {
		depsStr = cellGreen.Render(depsStr)
	}

	// Usage cell — show level + import count
	var usageStr string
	count := r.Usage.ImportCount
	switch r.Usage.Level {
	case types.UsageUnused:
		usageStr = cellRed.Bold(true).Render("UNUSED")
	case types.UsageTooling:
		usageStr = cellBlue.Render("Tooling")
	case types.UsageLow:
		usageStr = cellYellow.Render(fmt.Sprintf("Low(%d)", count))
	case types.UsageNormal:
		usageStr = cellGreen.Render(fmt.Sprintf("Mid(%d)", count))
	case types.UsageHeavy:
		usageStr = cellGreen.Bold(true).Render(fmt.Sprintf("High(%d)", count))
	}

	// Health cell
	var healthStr string
	switch {
	case r.HealthScore >= 0.7:
		healthStr = healthGood
	case r.HealthScore >= 0.4:
		healthStr = healthWarn
	default:
		healthStr = healthBad
	}

	cols := []string{
		padRight(name, columnWidths[0]),
		padRight(version, columnWidths[1]),
		padRight(sizeStr, columnWidths[2]),
		padRight(depsStr, columnWidths[3]),
		padRight(usageStr, columnWidths[4]),
		padRight(healthStr, columnWidths[5]),
	}

	row := strings.Join(cols, " ")

	if selected {
		row = selectedStyle.Render(row)
	}

	return row
}

func sortResults(results []types.AnalysisResult, col sortColumn, ascending bool) {
	sort.SliceStable(results, func(i, j int) bool {
		var less bool
		switch col {
		case sortByName:
			less = strings.ToLower(results[i].Dependency.Name) < strings.ToLower(results[j].Dependency.Name)
		case sortByVersion:
			less = results[i].Cost.Version < results[j].Cost.Version
		case sortBySize:
			less = results[i].Cost.InstallSize < results[j].Cost.InstallSize
		case sortByDeps:
			less = results[i].Cost.TransitiveDeps < results[j].Cost.TransitiveDeps
		case sortByUsage:
			less = results[i].Usage.ImportCount < results[j].Usage.ImportCount
		case sortByHealth:
			less = results[i].HealthScore < results[j].HealthScore
		}
		if !ascending {
			less = !less
		}
		return less
	})
}

func filterResults(results []types.AnalysisResult, query string) []types.AnalysisResult {
	if query == "" {
		return results
	}
	query = strings.ToLower(query)
	var filtered []types.AnalysisResult
	for _, r := range results {
		if strings.Contains(strings.ToLower(r.Dependency.Name), query) {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

func padRight(s string, width int) string {
	// Account for ANSI escape codes in visible width calculation
	visible := stripANSI(s)
	padding := width - len(visible)
	if padding <= 0 {
		return s
	}
	return s + strings.Repeat(" ", padding)
}

func stripANSI(s string) string {
	var result []byte
	inEscape := false
	for i := 0; i < len(s); i++ {
		if s[i] == '\x1b' {
			inEscape = true
			continue
		}
		if inEscape {
			if (s[i] >= 'a' && s[i] <= 'z') || (s[i] >= 'A' && s[i] <= 'Z') {
				inEscape = false
			}
			continue
		}
		result = append(result, s[i])
	}
	return string(result)
}
