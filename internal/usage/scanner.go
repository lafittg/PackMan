package usage

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/gregoirelafitte/packman/pkg/types"
)

// ImportFinder is a function that finds imports in a source file.
// It returns a map of package name -> list of import locations found in the file.
type ImportFinder func(filePath string, source []byte, knownPackages map[string]bool) (map[string][]types.ImportLocation, error)

// ScanProject walks source files and uses the provided ImportFinder to detect imports.
func ScanProject(projectRoot string, deps []types.Dependency, globs []string, excludeDirs []string, finder ImportFinder) ([]types.UsageInfo, error) {
	knownPackages := make(map[string]bool, len(deps))
	for _, d := range deps {
		knownPackages[d.Name] = true
	}

	// Collect all source files
	files, err := collectFiles(projectRoot, globs, excludeDirs)
	if err != nil {
		return nil, err
	}

	// Track imports per package
	packageImports := map[string][]types.ImportLocation{}

	for _, file := range files {
		source, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		found, err := finder(file, source, knownPackages)
		if err != nil {
			continue
		}

		for pkg, locations := range found {
			packageImports[pkg] = append(packageImports[pkg], locations...)
		}
	}

	// Build UsageInfo for each dependency
	results := make([]types.UsageInfo, 0, len(deps))
	for _, dep := range deps {
		locations := packageImports[dep.Name]
		importCount := countUniqueFiles(locations)
		usageCount := len(locations)

		results = append(results, types.UsageInfo{
			PackageName:     dep.Name,
			ImportCount:     importCount,
			UsageCount:      usageCount,
			Level:           types.UsageLevelFromCount(importCount),
			ImportLocations: locations,
		})
	}

	return results, nil
}

// collectFiles walks the project and returns files matching the given globs.
func collectFiles(root string, globs []string, excludeDirs []string) ([]string, error) {
	excludeSet := make(map[string]bool, len(excludeDirs))
	for _, d := range excludeDirs {
		excludeSet[d] = true
	}

	var files []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // skip errors
		}

		if d.IsDir() {
			name := d.Name()
			if excludeSet[name] || strings.HasPrefix(name, ".") {
				return filepath.SkipDir
			}
			return nil
		}

		// Check if file matches any glob
		for _, pattern := range globs {
			matched, _ := filepath.Match(pattern, d.Name())
			if matched {
				files = append(files, path)
				break
			}
		}

		return nil
	})

	return files, err
}

func countUniqueFiles(locations []types.ImportLocation) int {
	seen := map[string]bool{}
	for _, loc := range locations {
		seen[loc.FilePath] = true
	}
	return len(seen)
}
