# PackMan

[![Go](https://img.shields.io/badge/Go-1.24+-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)

A CLI tool that analyzes your project's dependencies to surface their **cost** (disk size, transitive deps, install time) and **usage** (unused, underused, heavily used) through an interactive terminal dashboard.

```
┌──────────────────────────────────────────────────────────┐
│  PackMan  Node.js Analysis                    47 packages │
│  Total: 148 MB | Transitive: 234 | Unused: 3             │
├──────────────────────────────────────────────────────────┤
│  Name           Version  Size     Deps  Usage    Health   │
│> lodash         4.17.21  1.4 MB   0     Low(2)   ● Warn  │
│  express        4.18.2   210 KB   30    High(42) ● Good  │
│  moment         2.29.4   4.2 MB   0     UNUSED   ● Bad   │
│  typescript     5.3.2    65 MB    0     Tooling  ● Good  │
│  axios          1.6.2    89 KB    7     Mid(8)   ● Good  │
├──────────────────────────────────────────────────────────┤
│  ↑↓ navigate  enter detail  /filter  s sort  ? help  q  │
└──────────────────────────────────────────────────────────┘
```

## Features

- **Cost analysis** — install size, publish size, transitive dependency count, estimated install time, weekly downloads
- **Usage detection** — scans source files for imports to find unused and underused packages
- **Smart classification** — recognizes tooling (`typescript`, `eslint`, `pytest`, `black`), type definitions (`@types/*`, `types-*`), config-only packages, and peer dependencies so they aren't flagged as unused
- **Health scoring** — weighted 0–100% score combining usage, size, and dependency footprint
- **Interactive TUI** — sortable, filterable table with package detail overlays, built with [Bubble Tea](https://github.com/charmbracelet/bubbletea)
- **Progress tracking** — real-time step-by-step progress during analysis
- **CI mode** — exits non-zero when unused dependencies are found
- **JSON output** — machine-readable output for scripting and pipelines
- **Multi-ecosystem** — supports Node.js and Python projects (auto-detected)
- **Plugin architecture** — extensible design for adding new ecosystems

## Supported Ecosystems

| Ecosystem | Manifest files | Registry | Import patterns |
|---|---|---|---|
| **Node.js** (npm/yarn/pnpm) | `package.json`, `package-lock.json` | [npm](https://registry.npmjs.org) | `import`/`require`/`import()` |
| **Python** (pip/Poetry/Pipenv) | `requirements.txt`, `pyproject.toml`, `setup.cfg`, `setup.py`, `Pipfile` | [PyPI](https://pypi.org) | `import`/`from ... import` |
| **Go** (modules) | `go.mod`, `go.sum` | [Go Proxy](https://proxy.golang.org) + [deps.dev](https://deps.dev) | `import "path"` |

### Python-specific features

- Parses PEP 621 (`[project.dependencies]`), Poetry (`[tool.poetry.dependencies]`), and Pipfile formats
- Maps Python import names to PyPI packages (e.g., `PIL` → `Pillow`, `yaml` → `PyYAML`, `sklearn` → `scikit-learn`)
- Recognizes 60+ known tooling packages (linters, test runners, build tools, type stubs)

### Go-specific features

- Parses `go.mod` with full support for `require`, `replace`, and `exclude` directives
- Resolves Go import paths to their owning module (e.g., `echo/v4/middleware` → `echo/v4`)
- Classifies indirect dependencies (marked `// indirect` in `go.mod`) as tooling
- Fetches module sizes from the Go module proxy and dependency graphs from [deps.dev](https://deps.dev)
- Recognizes code generators, linters, and migration tools as tooling packages

## Installation

### From source

Requires [Go 1.24+](https://go.dev/dl/).

```bash
go install github.com/gregoirelafitte/packman@latest
```

### Build locally

```bash
git clone https://github.com/gregoirelafitte/packman.git
cd packman
go build -o packman .
```

## Usage

### Interactive mode

Run inside a project directory:

```bash
packman analyze
```

Or point to a specific path:

```bash
packman analyze /path/to/project
```

### JSON output

```bash
packman analyze --json .
```

### CI mode

Fails with a non-zero exit code if unused dependencies are detected:

```bash
packman analyze --ci .
```

### TUI keybindings

| Key | Action |
|---|---|
| `↑` / `k` | Move up |
| `↓` / `j` | Move down |
| `Enter` | View package detail |
| `Esc` | Close detail / cancel filter |
| `/` | Filter by package name |
| `s` | Cycle sort column |
| `S` | Reverse sort order |
| `Tab` | Switch ecosystem (when multiple detected) |
| `?` | Toggle help |
| `q` | Quit |

## How It Works

1. **Detect** — auto-detects which ecosystem(s) are present in the project
2. **Parse** — reads dependency manifests and lockfiles for declared packages
3. **Cost** — queries the package registry (npm, PyPI) for size and metadata; results are cached to `~/.cache/packman/` with a 24h TTL
4. **Usage** — walks source files and scans for import statements, mapping them back to declared dependencies
5. **Classify** — identifies tooling, type definitions, config-only packages, and peer dependencies
6. **Score** — computes a 0–100% health score per package based on usage level, install size, and transitive dependency count
7. **Display** — renders an interactive TUI or outputs JSON/CI results

## Project Structure

```
packman/
├── main.go                        # Entry point, plugin registration
├── cmd/packman/root.go            # Cobra CLI commands
├── pkg/types/                     # Shared types (Dependency, CostInfo, UsageInfo)
├── internal/
│   ├── plugin/                    # Plugin interface + registry
│   │   ├── golang/                # Go modules ecosystem plugin
│   │   ├── nodejs/                # Node.js ecosystem plugin
│   │   └── python/                # Python ecosystem plugin
│   ├── analyzer/                  # Orchestrator (ties plugins together)
│   ├── cost/                      # Health score + summary computation
│   ├── usage/                     # Source file walker + import finder
│   ├── registry/                  # HTTP client with caching + npm/PyPI/Go Proxy APIs
│   └── tui/                       # Bubble Tea interactive dashboard
└── testdata/                      # Sample projects for testing
```

## Adding a New Ecosystem

PackMan uses a plugin architecture. To add support for a new ecosystem:

1. Create `internal/plugin/<name>/` with files implementing the `plugin.Plugin` interface
2. Register via `init()`: `plugin.Register(&MyPlugin{})`
3. Add a blank import in `main.go`: `_ "github.com/gregoirelafitte/packman/internal/plugin/<name>"`

The `Plugin` interface requires six methods:

```go
type Plugin interface {
    Name() string
    Detect(projectRoot string) (bool, error)
    ParseDependencies(projectRoot string) ([]types.Dependency, error)
    FetchCostData(deps []types.Dependency) ([]types.CostInfo, error)
    AnalyzeUsage(projectRoot string, deps []types.Dependency) ([]types.UsageInfo, error)
    SourceGlobs() []string
    ExcludeDirs() []string
}
```

## Contributing

Contributions are welcome! Please open an issue first to discuss what you'd like to change.

```bash
# Run tests
go test ./...

# Build
go build -o packman .

# Test against sample projects
./packman analyze testdata/nodejs/
./packman analyze testdata/python/
./packman analyze testdata/golang/

# Or analyze PackMan itself
./packman analyze .
```

## License

[Apache 2.0](LICENSE)
