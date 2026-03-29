package nodejs

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/gregoirelafitte/packman/pkg/types"
)

// packageJSON represents the relevant fields of a package.json file.
type packageJSON struct {
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
}

// packageLockJSON represents relevant fields of a package-lock.json.
type packageLockJSON struct {
	LockfileVersion int                        `json:"lockfileVersion"`
	Packages        map[string]lockPackageInfo  `json:"packages"`
	Dependencies    map[string]lockDepInfoV1    `json:"dependencies"`
}

type lockPackageInfo struct {
	Version      string            `json:"version"`
	Resolved     string            `json:"resolved"`
	Dependencies map[string]string `json:"dependencies"`
	Dev          bool              `json:"dev"`
}

type lockDepInfoV1 struct {
	Version      string            `json:"version"`
	Resolved     string            `json:"resolved"`
	Dependencies map[string]string `json:"requires"`
	Dev          bool              `json:"dev"`
}

// parseDependencies reads package.json and optional lockfile to produce dependencies.
func parseDependencies(projectRoot string) ([]types.Dependency, error) {
	pkgPath := filepath.Join(projectRoot, "package.json")
	data, err := os.ReadFile(pkgPath)
	if err != nil {
		return nil, err
	}

	var pkg packageJSON
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil, err
	}

	// Try to load lockfile for resolved versions
	resolved := loadResolvedVersions(projectRoot)

	var deps []types.Dependency

	for name, version := range pkg.Dependencies {
		d := types.Dependency{
			Name:    name,
			Version: version,
			IsDev:   false,
			Source:  "package.json",
		}
		if v, ok := resolved[name]; ok {
			d.ResolvedVersion = v
		}
		deps = append(deps, d)
	}

	for name, version := range pkg.DevDependencies {
		d := types.Dependency{
			Name:    name,
			Version: version,
			IsDev:   true,
			Source:  "package.json",
		}
		if v, ok := resolved[name]; ok {
			d.ResolvedVersion = v
		}
		deps = append(deps, d)
	}

	return deps, nil
}

// loadResolvedVersions attempts to read resolved versions from a lockfile.
func loadResolvedVersions(projectRoot string) map[string]string {
	resolved := map[string]string{}

	// Try package-lock.json first
	lockPath := filepath.Join(projectRoot, "package-lock.json")
	data, err := os.ReadFile(lockPath)
	if err == nil {
		var lock packageLockJSON
		if json.Unmarshal(data, &lock) == nil {
			// Lockfile v2/v3 uses "packages" with "node_modules/" prefix
			for key, info := range lock.Packages {
				if key == "" {
					continue // root package
				}
				// Extract package name from "node_modules/<name>" or "node_modules/@scope/name"
				name := extractPackageName(key)
				if name != "" {
					resolved[name] = info.Version
				}
			}
			// Lockfile v1 uses "dependencies"
			if len(resolved) == 0 {
				for name, info := range lock.Dependencies {
					resolved[name] = info.Version
				}
			}
		}
	}

	return resolved
}

// extractPackageName extracts the package name from a lockfile key like "node_modules/@scope/pkg".
func extractPackageName(key string) string {
	const prefix = "node_modules/"
	// Find the last occurrence of "node_modules/" to handle nested deps
	idx := len(key) - 1
	for idx >= 0 {
		start := lastIndex(key[:idx+1], prefix)
		if start == -1 {
			break
		}
		return key[start+len(prefix):]
	}

	if len(key) > len(prefix) && key[:len(prefix)] == prefix {
		return key[len(prefix):]
	}
	return ""
}

func lastIndex(s, substr string) int {
	for i := len(s) - len(substr); i >= 0; i-- {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
