package python

import (
	"regexp"
	"strings"

	"github.com/gregoirelafitte/packman/pkg/types"
)

// Python import patterns
var (
	// import package | import package.submodule | import package as alias
	importRe = regexp.MustCompile(`(?m)^\s*import\s+([\w.]+(?:\s+as\s+\w+)?(?:\s*,\s*[\w.]+(?:\s+as\s+\w+)?)*)`)

	// from package import X | from package.sub import X
	fromImportRe = regexp.MustCompile(`(?m)^\s*from\s+([\w.]+)\s+import\s+`)
)

// pythonToPackageMap maps common Python import names to their PyPI package names
// when they differ. This is a well-known pain point in the Python ecosystem.
var pythonToPackageMap = map[string]string{
	"PIL":                  "Pillow",
	"cv2":                  "opencv-python",
	"sklearn":              "scikit-learn",
	"skimage":              "scikit-image",
	"yaml":                 "PyYAML",
	"bs4":                  "beautifulsoup4",
	"serial":               "pyserial",
	"usb":                  "pyusb",
	"gi":                   "PyGObject",
	"attr":                 "attrs",
	"dateutil":             "python-dateutil",
	"dotenv":               "python-dotenv",
	"jwt":                  "PyJWT",
	"jose":                 "python-jose",
	"magic":                "python-magic",
	"Crypto":               "pycryptodome",
	"lxml":                 "lxml",
	"wx":                   "wxPython",
	"googleapiclient":      "google-api-python-client",
	"google.cloud":         "google-cloud-core",
	"google.auth":          "google-auth",
	"google.oauth2":        "google-auth",
	"bson":                 "pymongo",
	"psycopg2":             "psycopg2-binary",
	"MySQLdb":              "mysqlclient",
	"mysql":                "mysql-connector-python",
	"pymysql":              "PyMySQL",
	"docx":                 "python-docx",
	"pptx":                 "python-pptx",
	"xlrd":                 "xlrd",
	"openpyxl":             "openpyxl",
	"dns":                  "dnspython",
	"git":                  "GitPython",
	"github":               "PyGithub",
	"telegram":             "python-telegram-bot",
	"discord":              "discord.py",
	"slack_sdk":            "slack-sdk",
	"websocket":            "websocket-client",
	"socks":                "PySocks",
	"nacl":                 "PyNaCl",
	"Cython":               "Cython",
	"IPython":              "ipython",
	"Bio":                  "biopython",
	"pygame":               "pygame",
	"OpenGL":               "PyOpenGL",
	"zmq":                  "pyzmq",
	"multipart":            "python-multipart",
	"decouple":             "python-decouple",
	"ruamel":               "ruamel.yaml",
}

// findImportsPython scans a Python file for import statements.
func findImportsPython(filePath string, source []byte, knownPackages map[string]bool) (map[string][]types.ImportLocation, error) {
	result := map[string][]types.ImportLocation{}
	lines := strings.Split(string(source), "\n")

	// Build a normalized lookup: lowercase PyPI name -> original name
	normalizedKnown := make(map[string]string, len(knownPackages))
	for pkg := range knownPackages {
		normalizedKnown[normalizePythonName(pkg)] = pkg
	}

	for lineNum, line := range lines {
		// Skip comments and strings
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		// Handle "import X, Y, Z" and "import X as alias"
		if matches := importRe.FindStringSubmatch(line); len(matches) >= 2 {
			modules := splitImports(matches[1])
			for _, mod := range modules {
				if pkg := resolvePackageName(mod, normalizedKnown); pkg != "" {
					result[pkg] = append(result[pkg], types.ImportLocation{
						FilePath: filePath,
						Line:     lineNum + 1,
						Column:   strings.Index(line, matches[0]) + 1,
					})
				}
			}
		}

		// Handle "from X import Y"
		if matches := fromImportRe.FindStringSubmatch(line); len(matches) >= 2 {
			mod := matches[1]
			if pkg := resolvePackageName(mod, normalizedKnown); pkg != "" {
				result[pkg] = append(result[pkg], types.ImportLocation{
					FilePath: filePath,
					Line:     lineNum + 1,
					Column:   strings.Index(line, matches[0]) + 1,
				})
			}
		}
	}

	return result, nil
}

// splitImports splits "pkg1 as alias, pkg2, pkg3 as alias2" into module names.
func splitImports(s string) []string {
	parts := strings.Split(s, ",")
	var modules []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		// Remove "as alias" suffix
		if idx := strings.Index(p, " as "); idx != -1 {
			p = p[:idx]
		}
		p = strings.TrimSpace(p)
		if p != "" {
			modules = append(modules, p)
		}
	}
	return modules
}

// resolvePackageName maps a Python import module name to its PyPI package name.
// Returns empty string if the package is not in knownPackages.
func resolvePackageName(moduleName string, normalizedKnown map[string]string) string {
	// Get the top-level module (e.g., "google.cloud.storage" -> "google")
	topLevel := moduleName
	if idx := strings.Index(moduleName, "."); idx != -1 {
		topLevel = moduleName[:idx]
	}

	// 1. Check the well-known mapping first (exact module match, then top-level)
	candidates := []string{moduleName, topLevel}
	for _, candidate := range candidates {
		if pypiName, ok := pythonToPackageMap[candidate]; ok {
			normalized := normalizePythonName(pypiName)
			if origName, ok := normalizedKnown[normalized]; ok {
				return origName
			}
		}
	}

	// 2. Also check dotted prefixes for the mapping (e.g., "google.cloud" -> "google-cloud-core")
	if strings.Contains(moduleName, ".") {
		parts := strings.Split(moduleName, ".")
		for i := len(parts); i >= 2; i-- {
			prefix := strings.Join(parts[:i], ".")
			if pypiName, ok := pythonToPackageMap[prefix]; ok {
				normalized := normalizePythonName(pypiName)
				if origName, ok := normalizedKnown[normalized]; ok {
					return origName
				}
			}
		}
	}

	// 3. Try the module name directly (many packages have import name == package name)
	for _, candidate := range candidates {
		normalized := normalizePythonName(candidate)
		if origName, ok := normalizedKnown[normalized]; ok {
			return origName
		}
		// Also try with hyphens instead of underscores
		hyphenated := strings.ReplaceAll(normalized, "_", "-")
		if origName, ok := normalizedKnown[hyphenated]; ok {
			return origName
		}
	}

	return ""
}

// normalizePythonName normalizes a package name for comparison.
// PEP 503: lowercase, replace [-_.] with -.
func normalizePythonName(name string) string {
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, "-", "_")
	name = strings.ReplaceAll(name, ".", "_")
	return name
}
