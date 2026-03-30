package golang

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/gregoirelafitte/packman/pkg/types"
)

// parseDependencies reads go.mod and go.sum to extract declared dependencies.
func parseDependencies(projectRoot string) ([]types.Dependency, error) {
	goModPath := filepath.Join(projectRoot, "go.mod")
	deps, err := parseGoMod(goModPath)
	if err != nil {
		return nil, err
	}

	// Enrich with resolved versions from go.sum if available
	goSumPath := filepath.Join(projectRoot, "go.sum")
	if sumVersions, err := parseGoSum(goSumPath); err == nil {
		for i := range deps {
			if resolved, ok := sumVersions[deps[i].Name]; ok {
				deps[i].ResolvedVersion = resolved
			}
		}
	}

	return deps, nil
}

// goModRequireRe matches require lines in go.mod:
//
//	github.com/foo/bar v1.2.3
//	github.com/foo/bar v1.2.3 // indirect
var goModRequireRe = regexp.MustCompile(`^\s*([^\s]+)\s+(v[^\s]+)(?:\s+//\s+indirect)?`)

// parseGoMod parses a go.mod file and returns all required modules.
func parseGoMod(path string) ([]types.Dependency, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var deps []types.Dependency
	scanner := bufio.NewScanner(f)
	inRequireBlock := false
	inReplaceBlock := false
	inExcludeBlock := false

	// Track replacements to update module paths
	replacements := map[string]string{}

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}

		// Track block boundaries
		if strings.HasPrefix(line, "require (") || strings.HasPrefix(line, "require(") {
			inRequireBlock = true
			continue
		}
		if strings.HasPrefix(line, "replace (") || strings.HasPrefix(line, "replace(") {
			inReplaceBlock = true
			continue
		}
		if strings.HasPrefix(line, "exclude (") || strings.HasPrefix(line, "exclude(") {
			inExcludeBlock = true
			continue
		}
		if line == ")" {
			inRequireBlock = false
			inReplaceBlock = false
			inExcludeBlock = false
			continue
		}

		// Parse replacement directives (single-line)
		if strings.HasPrefix(line, "replace ") && !inReplaceBlock {
			parseReplacement(line[len("replace "):], replacements)
			continue
		}

		// Parse replacement directives (block)
		if inReplaceBlock {
			parseReplacement(line, replacements)
			continue
		}

		if inExcludeBlock {
			continue
		}

		// Parse single-line require
		if strings.HasPrefix(line, "require ") && !inRequireBlock {
			line = strings.TrimPrefix(line, "require ")
		} else if !inRequireBlock {
			continue
		}

		matches := goModRequireRe.FindStringSubmatch(line)
		if len(matches) < 3 {
			continue
		}

		modulePath := matches[1]
		version := matches[2]
		isIndirect := strings.Contains(line, "// indirect")

		// Skip the standard library and the module itself
		if isStdlib(modulePath) {
			continue
		}

		deps = append(deps, types.Dependency{
			Name:    modulePath,
			Version: version,
			IsDev:   isIndirect,
			Source:  "go.mod",
		})
	}

	// Apply replacements
	for i := range deps {
		if replacement, ok := replacements[deps[i].Name]; ok {
			deps[i].Name = replacement
		}
	}

	return deps, scanner.Err()
}

// parseReplacement parses a replace directive line.
// Formats: "old => new version" or "old version => new version"
func parseReplacement(line string, replacements map[string]string) {
	parts := strings.SplitN(line, "=>", 2)
	if len(parts) != 2 {
		return
	}
	oldParts := strings.Fields(strings.TrimSpace(parts[0]))
	newParts := strings.Fields(strings.TrimSpace(parts[1]))

	if len(oldParts) < 1 || len(newParts) < 1 {
		return
	}

	oldModule := oldParts[0]
	newModule := newParts[0]

	// Only track if the new module is a remote module (not a local path)
	if !strings.HasPrefix(newModule, ".") && !strings.HasPrefix(newModule, "/") {
		replacements[oldModule] = newModule
	}
}

// parseGoSum parses go.sum and returns the latest version for each module.
func parseGoSum(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	versions := map[string]string{}
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		modulePath := fields[0]
		version := fields[1]

		// go.sum has entries like "module v1.2.3" and "module v1.2.3/go.mod"
		// Strip the /go.mod suffix for clean versions
		version = strings.TrimSuffix(version, "/go.mod")

		// Keep the latest version seen
		versions[modulePath] = version
	}

	return versions, scanner.Err()
}

// isStdlib checks if a module path looks like a Go standard library package.
func isStdlib(path string) bool {
	// Standard library packages don't contain dots in the first path element
	firstSlash := strings.Index(path, "/")
	firstElement := path
	if firstSlash != -1 {
		firstElement = path[:firstSlash]
	}
	return !strings.Contains(firstElement, ".")
}

// findIndirectDeps reads go.mod and returns the set of indirect dependency module paths.
func findIndirectDeps(projectRoot string) map[string]bool {
	goModPath := filepath.Join(projectRoot, "go.mod")
	f, err := os.Open(goModPath)
	if err != nil {
		return nil
	}
	defer f.Close()

	indirect := map[string]bool{}
	scanner := bufio.NewScanner(f)
	inRequireBlock := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if strings.HasPrefix(line, "require (") || strings.HasPrefix(line, "require(") {
			inRequireBlock = true
			continue
		}
		if line == ")" {
			inRequireBlock = false
			continue
		}

		if !inRequireBlock && !strings.HasPrefix(line, "require ") {
			continue
		}

		if strings.Contains(line, "// indirect") {
			matches := goModRequireRe.FindStringSubmatch(line)
			if len(matches) >= 2 {
				indirect[matches[1]] = true
			}
		}
	}

	return indirect
}
