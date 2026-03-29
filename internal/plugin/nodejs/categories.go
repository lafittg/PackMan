package nodejs

import "strings"

// PackageCategory describes how a package is consumed.
type PackageCategory int

const (
	CategorySource  PackageCategory = iota // imported in source code
	CategoryTooling                        // CLI / build tool / linter / test runner
	CategoryTypes                          // @types/* declarations
	CategoryConfig                         // referenced in config files only
)

// knownToolingPackages are packages that are never imported in source code
// but are used as CLI tools, build tools, compilers, linters, test runners, etc.
var knownToolingPackages = map[string]bool{
	// Compilers & runtimes
	"typescript": true,
	"tsx":        true,
	"ts-node":    true,

	// Bundlers & build tools
	"webpack":              true,
	"webpack-cli":          true,
	"webpack-dev-server":   true,
	"vite":                 true,
	"rollup":               true,
	"esbuild":              true,
	"turbo":                true,
	"parcel":               true,
	"@next/bundle-analyzer": true,

	// Linters & formatters
	"eslint":             true,
	"prettier":           true,
	"@biomejs/biome":     true,
	"stylelint":          true,
	"@eslint/js":         true,

	// Testing
	"jest":                       true,
	"vitest":                     true,
	"cypress":                    true,
	"playwright":                 true,
	"@testing-library/jest-dom":  true,
	"@testing-library/react":     true,
	"@testing-library/user-event": true,
	"@vitest/browser":            true,
	"@vitest/coverage-v8":        true,
	"@vitest/ui":                 true,
	"start-server-and-test":      true,
	"cypress-dotenv":             true,
	"c8":                         true,
	"nyc":                        true,

	// Storybook
	"storybook":                 true,
	"chromatic":                 true,
	"msw-storybook-addon":       true,

	// Database CLI tools
	"prisma": true,

	// CSS tooling
	"postcss":          true,
	"autoprefixer":     true,
	"tailwindcss":      true,
	"sass":             true,
	"less":             true,

	// Environment & config
	"dotenv":     true,
	"dotenv-cli": true,
	"cross-env":  true,

	// Git hooks
	"husky":       true,
	"lint-staged": true,

	// Deployment
	"vercel":    true,
	"netlify-cli": true,

	// Misc build tooling
	"concurrently":  true,
	"npm-run-all":   true,
	"rimraf":        true,
	"nodemon":       true,
}

// classifyPackage determines the category of a Node.js package.
func classifyPackage(name string) PackageCategory {
	// @types/* packages are type declarations
	if strings.HasPrefix(name, "@types/") {
		return CategoryTypes
	}

	// @storybook/* packages are tooling
	if strings.HasPrefix(name, "@storybook/") {
		return CategoryTooling
	}

	// @typescript-eslint/* packages are tooling
	if strings.HasPrefix(name, "@typescript-eslint/") {
		return CategoryTooling
	}

	// @rollup/* packages are tooling
	if strings.HasPrefix(name, "@rollup/") {
		return CategoryTooling
	}

	// @chromatic-com/* packages are tooling
	if strings.HasPrefix(name, "@chromatic-com/") {
		return CategoryTooling
	}

	// Check known tooling list
	if knownToolingPackages[name] {
		return CategoryTooling
	}

	return CategorySource
}
