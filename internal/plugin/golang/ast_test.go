package golang

import (
	"testing"
)

func TestFindImportsGo(t *testing.T) {
	source := []byte(`package main

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	_ "github.com/lib/pq"
	"go.uber.org/zap"
)

func main() {
	fmt.Println("hello")
}
`)

	known := map[string]bool{
		"github.com/labstack/echo/v4": true,
		"github.com/lib/pq":          true,
		"go.uber.org/zap":            true,
		"github.com/spf13/viper":     true,
	}

	result, err := findImportsGo("main.go", source, known)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should detect echo (both import paths resolve to the same module)
	if locs, ok := result["github.com/labstack/echo/v4"]; !ok {
		t.Error("missing import detection for echo/v4")
	} else if len(locs) != 2 {
		t.Errorf("echo/v4: got %d locations, want 2 (base + middleware)", len(locs))
	}

	// Should detect pq
	if _, ok := result["github.com/lib/pq"]; !ok {
		t.Error("missing import detection for lib/pq")
	}

	// Should detect zap
	if _, ok := result["go.uber.org/zap"]; !ok {
		t.Error("missing import detection for zap")
	}

	// Should NOT detect viper (not imported)
	if _, ok := result["github.com/spf13/viper"]; ok {
		t.Error("should not detect viper (not imported)")
	}

	// Should NOT detect stdlib
	if _, ok := result["fmt"]; ok {
		t.Error("should not detect stdlib: fmt")
	}
	if _, ok := result["net/http"]; ok {
		t.Error("should not detect stdlib: net/http")
	}
}

func TestFindImportsSingleLine(t *testing.T) {
	source := []byte(`package main

import "github.com/labstack/echo/v4"
import myalias "go.uber.org/zap"

func main() {}
`)

	known := map[string]bool{
		"github.com/labstack/echo/v4": true,
		"go.uber.org/zap":            true,
	}

	result, err := findImportsGo("main.go", source, known)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, ok := result["github.com/labstack/echo/v4"]; !ok {
		t.Error("missing single-line import: echo/v4")
	}

	if _, ok := result["go.uber.org/zap"]; !ok {
		t.Error("missing aliased import: zap")
	}
}

func TestResolveModulePath(t *testing.T) {
	known := map[string]bool{
		"github.com/labstack/echo/v4": true,
		"github.com/lib/pq":          true,
		"go.uber.org/zap":            true,
	}

	tests := []struct {
		importPath string
		want       string
	}{
		{"github.com/labstack/echo/v4", "github.com/labstack/echo/v4"},
		{"github.com/labstack/echo/v4/middleware", "github.com/labstack/echo/v4"},
		{"github.com/lib/pq", "github.com/lib/pq"},
		{"go.uber.org/zap", "go.uber.org/zap"},
		{"go.uber.org/zap/zapcore", "go.uber.org/zap"},
		{"fmt", ""},          // stdlib
		{"net/http", ""},     // stdlib
		{"unknown/pkg", ""},  // not in known
	}

	for _, tt := range tests {
		got := resolveModulePath(tt.importPath, known)
		if got != tt.want {
			t.Errorf("resolveModulePath(%q) = %q, want %q", tt.importPath, got, tt.want)
		}
	}
}
