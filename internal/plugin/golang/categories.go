package golang

import "strings"

// PackageCategory describes how a package is consumed.
type PackageCategory int

const (
	CategorySource  PackageCategory = iota // imported in source code
	CategoryTooling                        // build/code-gen/lint tool
)

// knownToolingModules are Go modules that are used as CLI tools or code generators,
// not typically imported in application source code.
var knownToolingModules = map[string]bool{
	// Code generators
	"github.com/golang/mock":                  true,
	"go.uber.org/mock":                        true,
	"github.com/vektra/mockery":               true,
	"github.com/maxbrunsfeld/counterfeiter":   true,
	"github.com/deepmap/oapi-codegen":         true,
	"github.com/oapi-codegen/oapi-codegen":    true,
	"google.golang.org/protobuf":              true,
	"google.golang.org/grpc/cmd/protoc-gen-go-grpc": true,
	"github.com/bufbuild/buf":                 true,
	"github.com/sqlc-dev/sqlc":                true,

	// Linters & static analysis
	"golang.org/x/lint":                       true,
	"github.com/golangci/golangci-lint":       true,
	"honnef.co/go/tools":                      true,
	"github.com/mgechev/revive":               true,
	"github.com/securego/gosec":               true,

	// Build & task runners
	"github.com/magefile/mage":                true,
	"github.com/go-task/task":                 true,

	// Database migration tools
	"github.com/golang-migrate/migrate":       true,
	"github.com/pressly/goose":                true,
	"github.com/jackc/tern":                   true,
	"ariga.io/atlas":                          true,

	// Documentation
	"github.com/swaggo/swag":                  true,

	// Embedding & asset tools
	"github.com/go-bindata/go-bindata":        true,
	"github.com/markbates/pkger":              true,

	// Wire (dependency injection code gen)
	"github.com/google/wire":                  true,

	// Stringer, enumer, etc.
	"golang.org/x/tools":                      true,
	"github.com/dmarkham/enumer":              true,
}

// knownToolingPrefixes are module path prefixes that indicate tooling.
var knownToolingPrefixes = []string{
	"github.com/golangci/",
	"github.com/go-delve/",
}

// classifyPackage determines the category of a Go module.
func classifyPackage(modulePath string) PackageCategory {
	// Check exact match
	if knownToolingModules[modulePath] {
		return CategoryTooling
	}

	// Check known prefixes
	for _, prefix := range knownToolingPrefixes {
		if strings.HasPrefix(modulePath, prefix) {
			return CategoryTooling
		}
	}

	// Modules with /cmd/ in the path are often CLI tools
	// But be conservative — only flag if the whole module path ends with a cmd-like pattern
	parts := strings.Split(modulePath, "/")
	if len(parts) > 0 {
		last := parts[len(parts)-1]
		// Versioned modules: github.com/foo/bar/v4 → check bar
		if strings.HasPrefix(last, "v") && len(last) <= 3 && len(parts) > 1 {
			// skip, this is just a version suffix
		}
	}

	return CategorySource
}
