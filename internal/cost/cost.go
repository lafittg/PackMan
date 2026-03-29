package cost

import (
	"github.com/gregoirelafitte/packman/pkg/types"
)

// ComputeHealthScore calculates a 0.0-1.0 health score based on cost and usage.
func ComputeHealthScore(cost types.CostInfo, usage types.UsageInfo) float64 {
	var score float64

	// Usage weight (50%)
	switch usage.Level {
	case types.UsageUnused:
		score += 0.0
	case types.UsageTooling:
		score += 0.35 // tooling packages are needed even if not imported
	case types.UsageLow:
		score += 0.15
	case types.UsageNormal:
		score += 0.40
	case types.UsageHeavy:
		score += 0.50
	}

	// Size weight (25%) - smaller is better
	switch {
	case cost.InstallSize == 0:
		score += 0.20 // no data, assume ok
	case cost.InstallSize < 100*1024: // < 100KB
		score += 0.25
	case cost.InstallSize < 1024*1024: // < 1MB
		score += 0.15
	case cost.InstallSize < 5*1024*1024: // < 5MB
		score += 0.08
	default:
		score += 0.02
	}

	// Transitive deps weight (25%) - fewer is better
	switch {
	case cost.TransitiveDeps == 0:
		score += 0.25
	case cost.TransitiveDeps <= 5:
		score += 0.20
	case cost.TransitiveDeps <= 20:
		score += 0.12
	case cost.TransitiveDeps <= 50:
		score += 0.05
	default:
		score += 0.01
	}

	if score > 1.0 {
		score = 1.0
	}
	return score
}

// ComputeSummary aggregates statistics from analysis results.
func ComputeSummary(results []types.AnalysisResult) types.SummaryStats {
	var s types.SummaryStats
	s.TotalDeps = len(results)

	seen := map[string]bool{}
	for _, r := range results {
		s.TotalInstallSize += r.Cost.InstallSize
		s.EstTotalInstallTime += r.Cost.EstInstallTime

		if r.Dependency.IsDev {
			s.DevDepsCount++
		}

		switch r.Usage.Level {
		case types.UsageUnused:
			s.UnusedCount++
		case types.UsageLow:
			s.LowUsageCount++
		case types.UsageTooling:
			// tooling packages are not counted as unused
		}

		for _, dep := range r.Cost.DepTree {
			if !seen[dep] {
				seen[dep] = true
				s.TotalTransitiveDeps++
			}
		}
	}

	return s
}
