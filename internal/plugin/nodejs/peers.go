package nodejs

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/gregoirelafitte/packman/pkg/types"
)

// findPeerDependencies checks if any of the candidate packages are listed as
// a peerDependency or direct dependency of other declared packages. This detects
// packages like react-dom (peer of next), styled-jsx, @emotion/styled, etc.
func findPeerDependencies(projectRoot string, deps []types.Dependency, candidates map[string]bool) map[string]bool {
	found := make(map[string]bool)

	nodeModules := filepath.Join(projectRoot, "node_modules")
	if _, err := os.Stat(nodeModules); err != nil {
		return found
	}

	for _, dep := range deps {
		// Skip checking candidates against themselves
		if candidates[dep.Name] {
			continue
		}

		// Read the package.json of each declared dependency
		pkgJSON := resolveModulePath(nodeModules, dep.Name)
		if pkgJSON == "" {
			continue
		}

		data, err := os.ReadFile(pkgJSON)
		if err != nil {
			continue
		}

		var pkg struct {
			Dependencies     map[string]string `json:"dependencies"`
			PeerDependencies map[string]string `json:"peerDependencies"`
		}
		if err := json.Unmarshal(data, &pkg); err != nil {
			continue
		}

		// Check if any candidate appears in this package's deps or peerDeps
		for name := range candidates {
			if _, ok := pkg.Dependencies[name]; ok {
				found[name] = true
			}
			if _, ok := pkg.PeerDependencies[name]; ok {
				found[name] = true
			}
		}

		// Early exit if all candidates resolved
		if len(found) == len(candidates) {
			break
		}
	}

	return found
}

// resolveModulePath finds the package.json for a given package in node_modules.
func resolveModulePath(nodeModules, name string) string {
	pkgJSON := filepath.Join(nodeModules, name, "package.json")
	if _, err := os.Stat(pkgJSON); err == nil {
		return pkgJSON
	}
	return ""
}
