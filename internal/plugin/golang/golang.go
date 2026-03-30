package golang

import (
	"os"
	"path/filepath"
	"time"

	"github.com/gregoirelafitte/packman/internal/plugin"
	"github.com/gregoirelafitte/packman/internal/registry"
	"github.com/gregoirelafitte/packman/internal/usage"
	"github.com/gregoirelafitte/packman/pkg/types"
)

func init() {
	plugin.Register(&Plugin{})
}

// Plugin implements the plugin.Plugin interface for Go projects.
type Plugin struct{}

func (p *Plugin) Name() string {
	return "go"
}

func (p *Plugin) Detect(projectRoot string) (bool, error) {
	_, err := os.Stat(filepath.Join(projectRoot, "go.mod"))
	return err == nil, nil
}

func (p *Plugin) ParseDependencies(projectRoot string) ([]types.Dependency, error) {
	return parseDependencies(projectRoot)
}

func (p *Plugin) FetchCostData(deps []types.Dependency) ([]types.CostInfo, error) {
	client := registry.NewClient(10, 24*time.Hour)
	return fetchCostData(client, deps)
}

func (p *Plugin) AnalyzeUsage(projectRoot string, deps []types.Dependency) ([]types.UsageInfo, error) {
	results, err := usage.ScanProject(projectRoot, deps, p.SourceGlobs(), p.ExcludeDirs(), findImportsGo)
	if err != nil {
		return nil, err
	}

	// Post-process: reclassify packages that appear unused but are tooling
	for i := range results {
		if results[i].Level != types.UsageUnused {
			continue
		}
		if classifyPackage(results[i].PackageName) == CategoryTooling {
			results[i].Level = types.UsageTooling
		}
	}

	// Check if remaining unused packages are indirect dependencies required by direct ones
	unusedNames := make(map[string]bool)
	for i := range results {
		if results[i].Level == types.UsageUnused {
			unusedNames[results[i].PackageName] = true
		}
	}
	if len(unusedNames) > 0 {
		indirect := findIndirectDeps(projectRoot)
		for i := range results {
			if results[i].Level == types.UsageUnused && indirect[results[i].PackageName] {
				results[i].Level = types.UsageTooling
			}
		}
	}

	return results, nil
}

func (p *Plugin) SourceGlobs() []string {
	return []string{"*.go"}
}

func (p *Plugin) ExcludeDirs() []string {
	return []string{
		"vendor", "testdata", "node_modules",
		".git", "_build", "dist",
	}
}
