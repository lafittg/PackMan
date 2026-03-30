package python

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/gregoirelafitte/packman/pkg/types"
)

// parseDependencies reads Python dependency manifests and returns the declared dependencies.
// Supports: requirements.txt, pyproject.toml, setup.cfg, Pipfile.
func parseDependencies(projectRoot string) ([]types.Dependency, error) {
	seen := map[string]bool{}
	var deps []types.Dependency

	addDep := func(d types.Dependency) {
		key := strings.ToLower(d.Name)
		if seen[key] {
			return
		}
		seen[key] = true
		deps = append(deps, d)
	}

	// 1. pyproject.toml (PEP 621 + Poetry)
	if d, err := parsePyprojectToml(filepath.Join(projectRoot, "pyproject.toml")); err == nil {
		for _, dep := range d {
			addDep(dep)
		}
	}

	// 2. requirements.txt and variants
	for _, name := range []string{"requirements.txt", "requirements-dev.txt", "requirements_dev.txt", "requirements-test.txt"} {
		path := filepath.Join(projectRoot, name)
		isDev := name != "requirements.txt"
		if d, err := parseRequirementsTxt(path, isDev); err == nil {
			for _, dep := range d {
				addDep(dep)
			}
		}
	}

	// 3. setup.cfg
	if d, err := parseSetupCfg(filepath.Join(projectRoot, "setup.cfg")); err == nil {
		for _, dep := range d {
			addDep(dep)
		}
	}

	// 4. Pipfile
	if d, err := parsePipfile(filepath.Join(projectRoot, "Pipfile")); err == nil {
		for _, dep := range d {
			addDep(dep)
		}
	}

	if len(deps) == 0 {
		// Try setup.py as last resort
		if d, err := parseSetupPy(filepath.Join(projectRoot, "setup.py")); err == nil {
			for _, dep := range d {
				addDep(dep)
			}
		}
	}

	return deps, nil
}

// requirementRe matches lines in requirements.txt: package[extras]>=version
var requirementRe = regexp.MustCompile(`^([A-Za-z0-9][\w.-]*)(?:\[[\w,.-]+\])?\s*([<>=!~]+\s*[\w.*]+(?:\s*,\s*[<>=!~]+\s*[\w.*]+)*)?`)

// parseRequirementsTxt parses a requirements.txt file.
func parseRequirementsTxt(path string, isDev bool) ([]types.Dependency, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	source := filepath.Base(path)
	var deps []types.Dependency
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip comments, empty lines, options, and -r includes
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "-") {
			continue
		}

		// Handle inline comments
		if idx := strings.Index(line, " #"); idx != -1 {
			line = strings.TrimSpace(line[:idx])
		}

		matches := requirementRe.FindStringSubmatch(line)
		if len(matches) < 2 {
			continue
		}

		name := matches[1]
		version := ""
		if len(matches) >= 3 {
			version = strings.TrimSpace(matches[2])
		}

		deps = append(deps, types.Dependency{
			Name:    name,
			Version: version,
			IsDev:   isDev,
			Source:  source,
		})
	}

	return deps, scanner.Err()
}

// parsePyprojectToml parses dependencies from pyproject.toml.
// Supports both PEP 621 [project.dependencies] and [tool.poetry.dependencies].
func parsePyprojectToml(path string) ([]types.Dependency, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	content := string(data)
	var deps []types.Dependency

	// Parse PEP 621 style: [project] dependencies = [...]
	deps = append(deps, parsePEP621Deps(content, false)...)
	deps = append(deps, parsePEP621OptionalDeps(content)...)

	// Parse Poetry style: [tool.poetry.dependencies]
	deps = append(deps, parsePoetryDeps(content, "tool.poetry.dependencies", false)...)
	deps = append(deps, parsePoetryDeps(content, "tool.poetry.dev-dependencies", true)...)
	deps = append(deps, parsePoetryDeps(content, "tool.poetry.group.dev.dependencies", true)...)
	deps = append(deps, parsePoetryDeps(content, "tool.poetry.group.test.dependencies", true)...)

	return deps, nil
}

// pep621DepRe matches a PEP 508 dependency specifier in a list.
var pep621DepRe = regexp.MustCompile(`"([A-Za-z0-9][\w.-]*)(?:\[[\w,.-]+\])?\s*([^"]*)"`)

// parsePEP621Deps parses [project] dependencies = [...] entries.
func parsePEP621Deps(content string, isDev bool) []types.Dependency {
	section := findTOMLArray(content, "project", "dependencies")
	if section == "" {
		return nil
	}

	var deps []types.Dependency
	for _, match := range pep621DepRe.FindAllStringSubmatch(section, -1) {
		if len(match) < 2 {
			continue
		}
		deps = append(deps, types.Dependency{
			Name:    match[1],
			Version: strings.TrimSpace(match[2]),
			IsDev:   isDev,
			Source:  "pyproject.toml",
		})
	}
	return deps
}

// parsePEP621OptionalDeps parses [project.optional-dependencies].
func parsePEP621OptionalDeps(content string) []types.Dependency {
	// Find [project.optional-dependencies] section
	sectionStart := strings.Index(content, "[project.optional-dependencies]")
	if sectionStart == -1 {
		return nil
	}

	sectionContent := content[sectionStart+len("[project.optional-dependencies]"):]
	// End at next section header
	if idx := strings.Index(sectionContent, "\n["); idx != -1 {
		sectionContent = sectionContent[:idx]
	}

	var deps []types.Dependency
	for _, match := range pep621DepRe.FindAllStringSubmatch(sectionContent, -1) {
		if len(match) < 2 {
			continue
		}
		deps = append(deps, types.Dependency{
			Name:    match[1],
			Version: strings.TrimSpace(match[2]),
			IsDev:   true, // optional deps are typically dev/test
			Source:  "pyproject.toml",
		})
	}
	return deps
}

// poetryDepRe matches key = "version" or key = {version = "...", ...} in TOML.
var poetryDepRe = regexp.MustCompile(`(?m)^([A-Za-z0-9][\w.-]*)\s*=\s*(?:"([^"]*)"|\{[^}]*version\s*=\s*"([^"]*)"[^}]*\})`)

// parsePoetryDeps parses a [tool.poetry.*dependencies] section.
func parsePoetryDeps(content, section string, isDev bool) []types.Dependency {
	header := fmt.Sprintf("[%s]", section)
	idx := strings.Index(content, header)
	if idx == -1 {
		return nil
	}

	sectionContent := content[idx+len(header):]
	if end := strings.Index(sectionContent, "\n["); end != -1 {
		sectionContent = sectionContent[:end]
	}

	var deps []types.Dependency
	for _, match := range poetryDepRe.FindAllStringSubmatch(sectionContent, -1) {
		name := match[1]
		// Skip python itself
		if strings.ToLower(name) == "python" {
			continue
		}
		version := match[2]
		if version == "" {
			version = match[3]
		}
		deps = append(deps, types.Dependency{
			Name:    name,
			Version: version,
			IsDev:   isDev,
			Source:  "pyproject.toml",
		})
	}
	return deps
}

// findTOMLArray finds a TOML array value for a key within a section.
// e.g. findTOMLArray(content, "project", "dependencies") finds [project]\ndependencies = [...]
func findTOMLArray(content, section, key string) string {
	// Find the section header
	header := fmt.Sprintf("[%s]", section)
	idx := strings.Index(content, header)
	if idx == -1 {
		return ""
	}

	sectionContent := content[idx+len(header):]
	// End at next section
	if end := strings.Index(sectionContent, "\n["); end != -1 {
		sectionContent = sectionContent[:end]
	}

	// Find the key
	keyPattern := regexp.MustCompile(`(?m)^` + regexp.QuoteMeta(key) + `\s*=\s*\[`)
	loc := keyPattern.FindStringIndex(sectionContent)
	if loc == nil {
		return ""
	}

	// Extract array content (handle multiline)
	arrayStart := loc[1] - 1 // include the [
	remaining := sectionContent[arrayStart:]
	depth := 0
	for i, c := range remaining {
		if c == '[' {
			depth++
		} else if c == ']' {
			depth--
			if depth == 0 {
				return remaining[:i+1]
			}
		}
	}
	return ""
}

// parseSetupCfg parses dependencies from setup.cfg [options] install_requires.
func parseSetupCfg(path string) ([]types.Dependency, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	content := string(data)
	var deps []types.Dependency

	// Find [options] section
	optionsIdx := strings.Index(content, "[options]")
	if optionsIdx == -1 {
		return nil, nil
	}

	sectionContent := content[optionsIdx:]
	if end := strings.Index(sectionContent[1:], "\n["); end != -1 {
		sectionContent = sectionContent[:end+1]
	}

	// Find install_requires
	deps = append(deps, parseSetupCfgField(sectionContent, "install_requires", false)...)

	// Find [options.extras_require] section for dev deps
	extrasIdx := strings.Index(content, "[options.extras_require]")
	if extrasIdx != -1 {
		extrasContent := content[extrasIdx:]
		if end := strings.Index(extrasContent[1:], "\n["); end != -1 {
			extrasContent = extrasContent[:end+1]
		}
		// Parse all extras as dev dependencies
		lines := strings.Split(extrasContent, "\n")
		for _, line := range lines[1:] { // skip header
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "[") || line == "" {
				break
			}
			// Lines look like: dev = \n  package1\n  package2
			// or: test = package1\n  package2
			if idx := strings.Index(line, "="); idx != -1 {
				line = strings.TrimSpace(line[idx+1:])
			}
			if line == "" {
				continue
			}
			if d := parseOneRequirement(line, true, "setup.cfg"); d != nil {
				deps = append(deps, *d)
			}
		}
	}

	return deps, nil
}

func parseSetupCfgField(sectionContent, field string, isDev bool) []types.Dependency {
	fieldIdx := strings.Index(sectionContent, field)
	if fieldIdx == -1 {
		return nil
	}

	rest := sectionContent[fieldIdx:]
	eqIdx := strings.Index(rest, "=")
	if eqIdx == -1 {
		return nil
	}

	var deps []types.Dependency
	lines := strings.Split(rest[eqIdx+1:], "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Stop at next field (not indented continuation)
		if !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") && strings.Contains(line, "=") && !strings.HasPrefix(line, "#") {
			// Check if it looks like a new config key
			if requirementRe.FindStringSubmatch(line) == nil {
				break
			}
		}
		if d := parseOneRequirement(line, isDev, "setup.cfg"); d != nil {
			deps = append(deps, *d)
		}
	}
	return deps
}

func parseOneRequirement(line string, isDev bool, source string) *types.Dependency {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") {
		return nil
	}
	// Handle inline comments
	if idx := strings.Index(line, " #"); idx != -1 {
		line = strings.TrimSpace(line[:idx])
	}
	matches := requirementRe.FindStringSubmatch(line)
	if len(matches) < 2 {
		return nil
	}
	version := ""
	if len(matches) >= 3 {
		version = strings.TrimSpace(matches[2])
	}
	return &types.Dependency{
		Name:    matches[1],
		Version: version,
		IsDev:   isDev,
		Source:  source,
	}
}

// parsePipfile parses dependencies from a Pipfile.
func parsePipfile(path string) ([]types.Dependency, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	content := string(data)
	var deps []types.Dependency

	deps = append(deps, parsePipfileSection(content, "[packages]", false)...)
	deps = append(deps, parsePipfileSection(content, "[dev-packages]", true)...)

	return deps, nil
}

// pipfileDepRe matches Pipfile dependency lines: package = "version" or package = {version = "..."}
var pipfileDepRe = regexp.MustCompile(`(?m)^([A-Za-z0-9][\w.-]*)\s*=\s*(?:"([^"]*)"|\{[^}]*version\s*=\s*"([^"]*)"[^}]*\})`)

func parsePipfileSection(content, header string, isDev bool) []types.Dependency {
	idx := strings.Index(content, header)
	if idx == -1 {
		return nil
	}

	sectionContent := content[idx+len(header):]
	if end := strings.Index(sectionContent, "\n["); end != -1 {
		sectionContent = sectionContent[:end]
	}

	var deps []types.Dependency
	for _, match := range pipfileDepRe.FindAllStringSubmatch(sectionContent, -1) {
		name := match[1]
		version := match[2]
		if version == "" {
			version = match[3]
		}
		if version == "*" {
			version = ""
		}
		deps = append(deps, types.Dependency{
			Name:    name,
			Version: version,
			IsDev:   isDev,
			Source:  "Pipfile",
		})
	}
	return deps
}

// parseSetupPy does a best-effort regex parse of setup.py install_requires.
func parseSetupPy(path string) ([]types.Dependency, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	content := string(data)
	var deps []types.Dependency

	// Find install_requires=[...] block
	re := regexp.MustCompile(`install_requires\s*=\s*\[([\s\S]*?)\]`)
	match := re.FindStringSubmatch(content)
	if len(match) < 2 {
		return nil, nil
	}

	// Extract quoted strings
	strRe := regexp.MustCompile(`['"]([^'"]+)['"]`)
	for _, m := range strRe.FindAllStringSubmatch(match[1], -1) {
		if d := parseOneRequirement(m[1], false, "setup.py"); d != nil {
			deps = append(deps, *d)
		}
	}

	return deps, nil
}

