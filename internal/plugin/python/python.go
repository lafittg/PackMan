package python

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

// Plugin implements the plugin.Plugin interface for Python projects.
type Plugin struct{}

func (p *Plugin) Name() string {
	return "python"
}

func (p *Plugin) Detect(projectRoot string) (bool, error) {
	markers := []string{
		"requirements.txt",
		"pyproject.toml",
		"setup.py",
		"setup.cfg",
		"Pipfile",
	}
	for _, m := range markers {
		if _, err := os.Stat(filepath.Join(projectRoot, m)); err == nil {
			return true, nil
		}
	}
	return false, nil
}

func (p *Plugin) ParseDependencies(projectRoot string) ([]types.Dependency, error) {
	return parseDependencies(projectRoot)
}

func (p *Plugin) FetchCostData(deps []types.Dependency) ([]types.CostInfo, error) {
	client := registry.NewClient(10, 24*time.Hour)
	return fetchCostData(client, deps)
}

func (p *Plugin) AnalyzeUsage(projectRoot string, deps []types.Dependency) ([]types.UsageInfo, error) {
	results, err := usage.ScanProject(projectRoot, deps, p.SourceGlobs(), p.ExcludeDirs(), findImportsPython)
	if err != nil {
		return nil, err
	}

	// Post-process: reclassify packages that appear unused but are tooling/types/config
	unusedNames := make(map[string]bool)

	for i := range results {
		if results[i].Level != types.UsageUnused {
			continue
		}

		name := results[i].PackageName
		if classifyPackage(name) == CategoryTooling {
			results[i].Level = types.UsageTooling
		} else {
			unusedNames[name] = true
		}
	}

	// Check config files for references to remaining unused packages
	if len(unusedNames) > 0 {
		configRefs := scanConfigFiles(projectRoot, unusedNames)
		for i := range results {
			if results[i].Level == types.UsageUnused && configRefs[results[i].PackageName] {
				results[i].Level = types.UsageTooling
			}
		}
	}

	return results, nil
}

func (p *Plugin) SourceGlobs() []string {
	return []string{"*.py"}
}

func (p *Plugin) ExcludeDirs() []string {
	return []string{
		"venv", ".venv", "env", ".env",
		"__pycache__", ".tox", ".nox", ".mypy_cache",
		"node_modules", "dist", "build", ".eggs",
		"site-packages", ".pytest_cache", "htmlcov",
	}
}
