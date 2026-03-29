package nodejs

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

// Plugin implements the plugin.Plugin interface for Node.js projects.
type Plugin struct{}

func (p *Plugin) Name() string {
	return "nodejs"
}

func (p *Plugin) Detect(projectRoot string) (bool, error) {
	_, err := os.Stat(filepath.Join(projectRoot, "package.json"))
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
	results, err := usage.ScanProject(projectRoot, deps, p.SourceGlobs(), p.ExcludeDirs(), findImportsJS)
	if err != nil {
		return nil, err
	}

	// Also scan config files at project root for package references
	configResults := scanConfigFiles(projectRoot, deps)

	// Build a map of config-detected packages for quick lookup
	configDetected := make(map[string]bool, len(configResults))
	for pkg := range configResults {
		configDetected[pkg] = true
	}

	// Build set of unused package names for peer-dep check
	unusedNames := make(map[string]bool)

	// Post-process: reclassify packages that appear unused but are tooling/types/config
	for i := range results {
		if results[i].Level != types.UsageUnused {
			continue // already detected as used, keep it
		}

		name := results[i].PackageName
		cat := classifyPackage(name)

		switch cat {
		case CategoryTypes, CategoryTooling:
			results[i].Level = types.UsageTooling
		default:
			// Check if found in config files
			if configDetected[name] {
				results[i].Level = types.UsageTooling
			} else {
				unusedNames[name] = true
			}
		}
	}

	// Check if remaining unused packages are peer dependencies of other declared packages
	if len(unusedNames) > 0 {
		peerUsed := findPeerDependencies(projectRoot, deps, unusedNames)
		for i := range results {
			if results[i].Level == types.UsageUnused && peerUsed[results[i].PackageName] {
				results[i].Level = types.UsageTooling
			}
		}
	}

	return results, nil
}

func (p *Plugin) SourceGlobs() []string {
	return []string{"*.js", "*.jsx", "*.ts", "*.tsx", "*.mjs", "*.cjs"}
}

func (p *Plugin) ExcludeDirs() []string {
	return []string{"node_modules", "dist", "build", ".next", "coverage", "__tests__", "vendor"}
}
