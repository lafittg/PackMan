package golang

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseGoMod(t *testing.T) {
	deps, err := parseGoMod("../../../testdata/golang/go.mod")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(deps) != 6 {
		t.Errorf("got %d deps, want 6", len(deps))
		for _, d := range deps {
			t.Logf("  dep: %s %s indirect=%v", d.Name, d.Version, d.IsDev)
		}
	}

	expected := map[string]struct {
		version  string
		indirect bool
	}{
		"github.com/labstack/echo/v4": {"v4.11.4", false},
		"github.com/lib/pq":          {"v1.10.9", false},
		"go.uber.org/zap":            {"v1.27.0", false},
		"github.com/spf13/viper":     {"v1.18.2", false},
		"github.com/stretchr/testify": {"v1.9.0", true},
		"golang.org/x/crypto":        {"v0.21.0", true},
	}

	for _, dep := range deps {
		exp, ok := expected[dep.Name]
		if !ok {
			t.Errorf("unexpected dep: %s", dep.Name)
			continue
		}
		if dep.Version != exp.version {
			t.Errorf("dep %s: got version %q, want %q", dep.Name, dep.Version, exp.version)
		}
		if dep.IsDev != exp.indirect {
			t.Errorf("dep %s: got indirect=%v, want %v", dep.Name, dep.IsDev, exp.indirect)
		}
		if dep.Source != "go.mod" {
			t.Errorf("dep %s: got source %q, want %q", dep.Name, dep.Source, "go.mod")
		}
	}
}

func TestParseGoModReplace(t *testing.T) {
	dir := t.TempDir()
	content := `module example.com/myapp

go 1.24

require (
	github.com/old/module v1.0.0
	github.com/another/pkg v2.1.0
)

replace github.com/old/module => github.com/new/module v1.1.0
`
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte(content), 0o644)

	deps, err := parseGoMod(filepath.Join(dir, "go.mod"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := false
	for _, dep := range deps {
		if dep.Name == "github.com/new/module" {
			found = true
		}
		if dep.Name == "github.com/old/module" {
			t.Error("old module should have been replaced")
		}
	}
	if !found {
		t.Error("replacement module not found")
	}
}

func TestIsStdlib(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"fmt", true},
		{"net/http", true},
		{"encoding/json", true},
		{"github.com/foo/bar", false},
		{"golang.org/x/crypto", false},
		{"go.uber.org/zap", false},
	}

	for _, tt := range tests {
		got := isStdlib(tt.path)
		if got != tt.want {
			t.Errorf("isStdlib(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}
