package python

import (
	"os"
	"path/filepath"
	"strings"
)

// PackageCategory describes how a package is consumed.
type PackageCategory int

const (
	CategorySource  PackageCategory = iota // imported in source code
	CategoryTooling                        // CLI / build tool / linter / test runner
)

// knownToolingPackages are Python packages that are never imported in source code
// but are used as CLI tools, build tools, linters, test runners, etc.
var knownToolingPackages = map[string]bool{
	// Linters & formatters
	"black":        true,
	"ruff":         true,
	"flake8":       true,
	"pylint":       true,
	"mypy":         true,
	"pyright":      true,
	"autopep8":     true,
	"yapf":         true,
	"isort":        true,
	"pyflakes":     true,
	"pycodestyle":  true,
	"pydocstyle":   true,
	"bandit":       true,

	// Testing tools (runners, not libraries)
	"pytest":          true,
	"pytest-cov":      true,
	"pytest-xdist":    true,
	"pytest-asyncio":  true,
	"pytest-mock":     true,
	"pytest-django":   true,
	"pytest-flask":    true,
	"pytest-env":      true,
	"pytest-timeout":  true,
	"pytest-randomly": true,
	"tox":             true,
	"nox":             true,
	"coverage":        true,

	// Build & packaging
	"setuptools":       true,
	"wheel":            true,
	"pip":              true,
	"build":            true,
	"twine":            true,
	"flit":             true,
	"poetry":           true,
	"poetry-core":      true,
	"hatchling":        true,
	"hatch":            true,
	"pdm":              true,
	"maturin":          true,
	"setuptools-scm":   true,
	"pip-tools":        true,
	"pipenv":           true,

	// Type stubs
	"types-requests":      true,
	"types-setuptools":    true,
	"types-PyYAML":        true,
	"types-toml":          true,
	"types-six":           true,
	"types-redis":         true,
	"types-Pillow":        true,
	"types-protobuf":      true,
	"types-python-dateutil": true,

	// Pre-commit & git hooks
	"pre-commit": true,

	// Documentation
	"sphinx":          true,
	"mkdocs":          true,
	"mkdocs-material": true,
	"pdoc":            true,
	"pdoc3":           true,

	// Environment
	"virtualenv":       true,
	"python-dotenv":    true,
	"environs":         true,

	// Database CLI tools
	"alembic":    true,
	"django":     false, // django is imported, not just CLI
	"ipython":    true,
	"jupyter":    true,
	"notebook":   true,
	"ipykernel":  true,

	// Debug & profiling
	"debugpy":     true,
	"py-spy":      true,
	"pyinstrument": true,

	// Runtime servers (CLI-invoked, not imported)
	"gunicorn":     true,
	"uvicorn":      true,
	"hypercorn":    true,
	"daphne":       true,
	"waitress":     true,

	// Misc tooling
	"bumpversion":     true,
	"bump2version":    true,
	"invoke":          true,
	"fabric":          true,
}

// classifyPackage determines the category of a Python package.
func classifyPackage(name string) PackageCategory {
	lower := strings.ToLower(name)

	// types-* packages are type stubs
	if strings.HasPrefix(lower, "types-") {
		return CategoryTooling
	}

	// *-stubs packages are type stubs
	if strings.HasSuffix(lower, "-stubs") {
		return CategoryTooling
	}

	// Check known tooling list (case-insensitive)
	normalized := strings.ToLower(strings.ReplaceAll(name, "_", "-"))
	if knownToolingPackages[normalized] {
		return CategoryTooling
	}
	// Also try original name
	if knownToolingPackages[name] {
		return CategoryTooling
	}

	return CategorySource
}

// scanConfigFiles checks Python config files for package references.
func scanConfigFiles(projectRoot string, candidates map[string]bool) map[string]bool {
	found := make(map[string]bool)

	// Build normalized lookup
	normalizedCandidates := make(map[string]string, len(candidates))
	for name := range candidates {
		normalizedCandidates[strings.ToLower(strings.ReplaceAll(name, "-", "_"))] = name
		normalizedCandidates[strings.ToLower(strings.ReplaceAll(name, "_", "-"))] = name
		normalizedCandidates[strings.ToLower(name)] = name
	}

	configFiles := []string{
		"pyproject.toml", "setup.cfg", "tox.ini", ".flake8",
		"mypy.ini", ".mypy.ini", ".pylintrc", ".coveragerc",
		"pytest.ini", "conftest.py", "Makefile",
	}

	for _, name := range configFiles {
		path := filepath.Join(projectRoot, name)
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		content := strings.ToLower(string(data))

		for normalized, origName := range normalizedCandidates {
			if strings.Contains(content, normalized) {
				found[origName] = true
			}
		}
	}

	return found
}
