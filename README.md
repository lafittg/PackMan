# PackMan

A CLI tool that analyzes your project's dependencies to surface their **cost** (disk size, transitive deps, install time) and **usage** (unused, underused, heavily used) through an interactive terminal dashboard.

```
┌──────────────────────────────────────────────────────────┐
│  PackMan  Node.js Analysis                    47 packages │
│  Total: 148 MB | Transitive: 234 | Unused: 3             │
├──────────────────────────────────────────────────────────┤
│  Name           Version  Size     Deps  Usage    Health   │
│> lodash         4.17.21  1.4 MB   0     Low      ● Warn  │
│  express        4.18.2   210 KB   30    Heavy    ● Good  │
│  moment         2.29.4   4.2 MB   0     UNUSED   ● Bad   │
│  axios          1.6.2    89 KB    7     Normal   ● Good  │
├──────────────────────────────────────────────────────────┤
│  ↑↓ navigate  enter detail  /filter  s sort  ? help  q  │
└──────────────────────────────────────────────────────────┘
```

## Features

- **Cost analysis** — install size, publish size, transitive dependency count, estimated install time, weekly downloads
- **Usage detection** — scans source files for imports to find unused and underused packages
- **Health scoring** — weighted score combining usage, size, and dependency footprint
- **Interactive TUI** — sortable, filterable table with package detail overlays, built with [bubbletea](https://github.com/charmbracelet/bubbletea)
- **CI mode** — exits non-zero when unused dependencies are found
- **JSON output** — machine-readable output for scripting and pipelines
- **Plugin architecture** — extensible ecosystem support (Node.js shipped, Python planned)

## Installation

### From source

Requires [Go 1.21+](https://go.dev/dl/).

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

## Supported ecosystems

| Ecosystem | Status | Detects via |
|---|---|---|
| Node.js (npm/yarn/pnpm) | Shipped | `package.json` |
| Python (pip/poetry) | Planned | `requirements.txt`, `pyproject.toml` |

## How it works

1. **Detect** — auto-detects which ecosystem(s) are present in the project
2. **Parse** — reads dependency manifests and lockfiles for declared packages
3. **Cost** — queries the package registry (npm, PyPI) for size and metadata; results are cached to `~/.cache/packman/` with a 24h TTL
4. **Usage** — walks source files and scans for import/require statements, mapping them back to declared dependencies
5. **Score** — computes a 0–100% health score per package based on usage level, install size, and transitive dependency count
6. **Display** — renders an interactive TUI or outputs JSON/CI results

## Project structure

```
packman/
├── main.go                        # Entry point, plugin registration
├── cmd/packman/root.go            # Cobra CLI commands
├── pkg/types/                     # Shared types (Dependency, CostInfo, UsageInfo)
├── internal/
│   ├── plugin/                    # Plugin interface + registry
│   │   └── nodejs/                # Node.js ecosystem plugin
│   ├── analyzer/                  # Orchestrator (ties plugins together)
│   ├── cost/                      # Health score + summary computation
│   ├── usage/                     # Source file walker + import finder
│   ├── registry/                  # HTTP client with caching + npm API
│   └── tui/                       # bubbletea interactive dashboard
└── testdata/                      # Sample projects for testing
```

## Adding a new ecosystem

PackMan uses a plugin architecture. To add support for a new ecosystem:

1. Create `internal/plugin/<name>/` with files implementing the `plugin.Plugin` interface
2. Register via `init()`: `plugin.Register(&MyPlugin{})`
3. Add a blank import in `main.go`: `_ "github.com/gregoirelafitte/packman/internal/plugin/<name>"`

The `Plugin` interface requires six methods: `Detect`, `ParseDependencies`, `FetchCostData`, `AnalyzeUsage`, `SourceGlobs`, and `ExcludeDirs`.

## Contributing

Contributions are welcome. Please open an issue first to discuss what you'd like to change.

```bash
# Run tests
go test ./...

# Build
go build -o packman .

# Test against a sample project
./packman analyze testdata/nodejs/
```

## License

[Apache 2.0](LICENSE)
