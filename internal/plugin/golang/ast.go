package golang

import (
	"regexp"
	"strings"

	"github.com/gregoirelafitte/packman/pkg/types"
)

// Go import patterns
var (
	// Single import: import "github.com/foo/bar"
	singleImportRe = regexp.MustCompile(`(?m)^\s*import\s+"([^"]+)"`)

	// Single import with alias: import foo "github.com/foo/bar"
	aliasImportRe = regexp.MustCompile(`(?m)^\s*import\s+\w+\s+"([^"]+)"`)

	// Grouped import block line: "github.com/foo/bar" or alias "github.com/foo/bar"
	groupImportRe = regexp.MustCompile(`^\s*(?:\w+\s+)?"([^"]+)"`)
)

// findImportsGo scans a Go source file for import statements and matches them
// to known module dependencies.
func findImportsGo(filePath string, source []byte, knownPackages map[string]bool) (map[string][]types.ImportLocation, error) {
	result := map[string][]types.ImportLocation{}
	content := string(source)
	lines := strings.Split(content, "\n")

	inImportBlock := false

	for lineNum, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Detect import block boundaries
		if strings.HasPrefix(trimmed, "import (") {
			inImportBlock = true
			continue
		}
		if inImportBlock && trimmed == ")" {
			inImportBlock = false
			continue
		}

		var importPath string

		if inImportBlock {
			// Inside grouped import block
			if matches := groupImportRe.FindStringSubmatch(trimmed); len(matches) >= 2 {
				importPath = matches[1]
			}
		} else {
			// Single-line import (with or without alias)
			if matches := aliasImportRe.FindStringSubmatch(line); len(matches) >= 2 {
				importPath = matches[1]
			} else if matches := singleImportRe.FindStringSubmatch(line); len(matches) >= 2 {
				importPath = matches[1]
			}
		}

		if importPath == "" {
			continue
		}

		// Resolve import path to a known module
		moduleName := resolveModulePath(importPath, knownPackages)
		if moduleName == "" {
			continue
		}

		result[moduleName] = append(result[moduleName], types.ImportLocation{
			FilePath: filePath,
			Line:     lineNum + 1,
			Column:   strings.Index(line, importPath) + 1,
		})
	}

	return result, nil
}

// resolveModulePath maps a Go import path to a known module from go.mod.
// Go imports are package paths, but modules may own entire path subtrees.
// E.g., import "github.com/labstack/echo/v4/middleware" belongs to module "github.com/labstack/echo/v4".
func resolveModulePath(importPath string, knownModules map[string]bool) string {
	// Skip standard library
	if isStdlib(importPath) {
		return ""
	}

	// Try exact match first
	if knownModules[importPath] {
		return importPath
	}

	// Try progressively shorter prefixes.
	// Module paths can own sub-packages:
	//   Module: github.com/labstack/echo/v4
	//   Import: github.com/labstack/echo/v4/middleware
	path := importPath
	for {
		lastSlash := strings.LastIndex(path, "/")
		if lastSlash == -1 {
			break
		}
		path = path[:lastSlash]
		if knownModules[path] {
			return path
		}
	}

	return ""
}
