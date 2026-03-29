package plugin

import "github.com/gregoirelafitte/packman/pkg/types"

// Plugin defines the contract every ecosystem must implement.
type Plugin interface {
	// Name returns the ecosystem identifier, e.g. "nodejs", "python".
	Name() string

	// Detect checks if this ecosystem is present in the given project root.
	Detect(projectRoot string) (bool, error)

	// ParseDependencies reads dependency manifest files and returns declared dependencies.
	ParseDependencies(projectRoot string) ([]types.Dependency, error)

	// FetchCostData queries the package registry for size/metadata.
	FetchCostData(deps []types.Dependency) ([]types.CostInfo, error)

	// AnalyzeUsage walks source files and uses AST parsing to determine usage.
	AnalyzeUsage(projectRoot string, deps []types.Dependency) ([]types.UsageInfo, error)

	// SourceGlobs returns glob patterns for source files to scan.
	SourceGlobs() []string

	// ExcludeDirs returns directories to skip during scanning.
	ExcludeDirs() []string
}
