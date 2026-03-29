package types

import "time"

// Dependency represents a declared dependency from a manifest file.
type Dependency struct {
	Name            string
	Version         string // declared version constraint
	ResolvedVersion string // actual resolved version from lockfile
	IsDev           bool   // devDependency / test dependency
	Source          string // e.g. "package.json", "requirements.txt"
}

// CostInfo holds the "cost" metrics for a single package.
type CostInfo struct {
	PackageName     string
	Version         string
	PublishSize     int64         // tarball/wheel size in bytes
	InstallSize     int64         // unpacked install size in bytes
	TransitiveDeps  int           // total number of transitive dependencies
	DirectDeps      int           // number of direct sub-dependencies
	EstInstallTime  time.Duration // estimated install time
	LastPublish     time.Time     // when the version was published
	WeeklyDownloads int64         // popularity signal
	DepTree         []string      // names of transitive deps (flat list)
}

// UsageLevel classifies how heavily a package is used.
type UsageLevel int

const (
	UsageUnused  UsageLevel = iota // 0 imports found
	UsageTooling                   // tooling / types / config-only package
	UsageLow                       // 1-2 import sites
	UsageNormal                    // 3-10 import sites
	UsageHeavy                     // 11+ import sites
)

// String returns a human-readable label for the usage level.
func (u UsageLevel) String() string {
	switch u {
	case UsageUnused:
		return "UNUSED"
	case UsageTooling:
		return "Tooling"
	case UsageLow:
		return "Low"
	case UsageNormal:
		return "Normal"
	case UsageHeavy:
		return "Heavy"
	default:
		return "Unknown"
	}
}

// UsageLevelFromCount derives a UsageLevel from an import count.
func UsageLevelFromCount(count int) UsageLevel {
	switch {
	case count == 0:
		return UsageUnused
	case count <= 2:
		return UsageLow
	case count <= 10:
		return UsageNormal
	default:
		return UsageHeavy
	}
}

// UsageInfo describes how a dependency is used in the codebase.
type UsageInfo struct {
	PackageName     string
	ImportCount     int              // number of files that import this package
	UsageCount      int              // total usage sites across files
	Level           UsageLevel
	ImportLocations []ImportLocation // where it is imported
}

// ImportLocation pinpoints a single import statement.
type ImportLocation struct {
	FilePath string
	Line     int
	Column   int
}

// AnalysisResult is the final combined result for one package.
type AnalysisResult struct {
	Dependency  Dependency
	Cost        CostInfo
	Usage       UsageInfo
	HealthScore float64 // 0.0 (bad) to 1.0 (good)
}

// ProjectReport is the top-level report for the entire project.
type ProjectReport struct {
	ProjectRoot  string
	Ecosystem    string
	AnalyzedAt   time.Time
	Dependencies []AnalysisResult
	Summary      SummaryStats
}

// SummaryStats provides aggregate statistics.
type SummaryStats struct {
	TotalDeps           int
	TotalTransitiveDeps int
	TotalInstallSize    int64
	UnusedCount         int
	LowUsageCount       int
	DevDepsCount        int
	EstTotalInstallTime time.Duration
}
