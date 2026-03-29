package nodejs

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/gregoirelafitte/packman/pkg/types"
)

// Import patterns for JavaScript/TypeScript
var (
	// ES module: import X from 'package' | import { X } from 'package' | import * as X from 'package' | import 'package'
	esImportRe = regexp.MustCompile(`(?m)^\s*import\s+(?:(?:type\s+)?(?:\{[^}]*\}|\*\s+as\s+\w+|\w+(?:\s*,\s*(?:\{[^}]*\}|\*\s+as\s+\w+))?)\s+from\s+)?['"]([^'"]+)['"]`)

	// CommonJS: const X = require('package') | require('package')
	requireRe = regexp.MustCompile(`(?m)require\s*\(\s*['"]([^'"]+)['"]\s*\)`)

	// Dynamic import: import('package')
	dynamicImportRe = regexp.MustCompile(`(?m)import\s*\(\s*['"]([^'"]+)['"]\s*\)`)
)

// findImportsJS scans a JavaScript/TypeScript file for import statements.
func findImportsJS(filePath string, source []byte, knownPackages map[string]bool) (map[string][]types.ImportLocation, error) {
	result := map[string][]types.ImportLocation{}
	lines := strings.Split(string(source), "\n")

	patterns := []*regexp.Regexp{esImportRe, requireRe, dynamicImportRe}

	for lineNum, line := range lines {
		for _, re := range patterns {
			matches := re.FindAllStringSubmatch(line, -1)
			for _, match := range matches {
				if len(match) < 2 {
					continue
				}
				specifier := match[1]
				pkgName := resolvePackageName(specifier)

				if !knownPackages[pkgName] {
					continue
				}

				result[pkgName] = append(result[pkgName], types.ImportLocation{
					FilePath: filePath,
					Line:     lineNum + 1,
					Column:   strings.Index(line, match[0]) + 1,
				})
			}
		}
	}

	return result, nil
}

// resolvePackageName extracts the npm package name from an import specifier.
// Handles scoped packages (@scope/pkg) and deep imports (lodash/get -> lodash).
func resolvePackageName(specifier string) string {
	// Skip relative imports
	if strings.HasPrefix(specifier, ".") || strings.HasPrefix(specifier, "/") {
		return ""
	}

	// Scoped package: @scope/pkg/deep -> @scope/pkg
	if strings.HasPrefix(specifier, "@") {
		parts := strings.SplitN(specifier, "/", 3)
		if len(parts) >= 2 {
			return parts[0] + "/" + parts[1]
		}
		return specifier
	}

	// Regular package: lodash/get -> lodash
	parts := strings.SplitN(specifier, "/", 2)
	return parts[0]
}

// configFilePatterns lists glob patterns for config files that reference packages.
var configFilePatterns = []string{
	"*.config.js", "*.config.ts", "*.config.mjs", "*.config.cjs",
	".babelrc", ".eslintrc", ".eslintrc.js", ".eslintrc.json",
	".prettierrc", ".prettierrc.js",
	".postcssrc", ".postcssrc.js",
}

// packageRefRe matches package names in config files: strings like 'package-name'
// or "package-name", or bare identifiers used as plugin names.
var packageRefRe = regexp.MustCompile(`['"](@[\w-]+/[\w.-]+|[\w][\w.-]*)['"]`)

// scanConfigFiles scans config files at the project root for package name references.
// Returns a set of package names found in config files.
func scanConfigFiles(projectRoot string, deps []types.Dependency) map[string]bool {
	knownPackages := make(map[string]bool, len(deps))
	for _, d := range deps {
		knownPackages[d.Name] = true
	}

	found := make(map[string]bool)

	// Collect config files matching our patterns
	for _, pattern := range configFilePatterns {
		matches, err := filepath.Glob(filepath.Join(projectRoot, pattern))
		if err != nil {
			continue
		}
		for _, file := range matches {
			source, err := os.ReadFile(file)
			if err != nil {
				continue
			}
			content := string(source)

			// Find all string references that match known package names
			allMatches := packageRefRe.FindAllStringSubmatch(content, -1)
			for _, m := range allMatches {
				if len(m) < 2 {
					continue
				}
				name := m[1]
				if knownPackages[name] {
					found[name] = true
				}
			}

			// Also check for require/import in config files (they're valid JS/TS)
			lines := strings.Split(content, "\n")
			patterns := []*regexp.Regexp{esImportRe, requireRe, dynamicImportRe}
			for _, line := range lines {
				for _, re := range patterns {
					lineMatches := re.FindAllStringSubmatch(line, -1)
					for _, match := range lineMatches {
						if len(match) < 2 {
							continue
						}
						pkgName := resolvePackageName(match[1])
						if knownPackages[pkgName] {
							found[pkgName] = true
						}
					}
				}
			}
		}
	}

	return found
}
