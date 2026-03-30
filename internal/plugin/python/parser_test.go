package python

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseRequirementsTxt(t *testing.T) {
	deps, err := parseRequirementsTxt("../../../testdata/python/requirements.txt", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := map[string]string{
		"flask":         ">=2.0.0",
		"requests":      "==2.31.0",
		"SQLAlchemy":    ">=2.0,<3.0",
		"celery":        ">=5.3",
		"python-dotenv": ">=1.0.0",
		"gunicorn":      ">=21.2.0",
		"pydantic":      ">=2.0",
		"redis":         ">=5.0",
	}

	if len(deps) != len(expected) {
		t.Errorf("got %d deps, want %d", len(deps), len(expected))
		for _, d := range deps {
			t.Logf("  dep: %s %s", d.Name, d.Version)
		}
	}

	found := map[string]bool{}
	for _, dep := range deps {
		found[dep.Name] = true
		if want, ok := expected[dep.Name]; ok {
			if dep.Version != want {
				t.Errorf("dep %s: got version %q, want %q", dep.Name, dep.Version, want)
			}
		}
		if dep.IsDev {
			t.Errorf("dep %s should not be dev", dep.Name)
		}
		if dep.Source != "requirements.txt" {
			t.Errorf("dep %s: got source %q, want %q", dep.Name, dep.Source, "requirements.txt")
		}
	}

	for name := range expected {
		if !found[name] {
			t.Errorf("missing dependency: %s", name)
		}
	}
}

func TestParseRequirementsTxtDev(t *testing.T) {
	deps, err := parseRequirementsTxt("../../../testdata/python/requirements-dev.txt", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(deps) != 4 {
		t.Errorf("got %d deps, want 4", len(deps))
	}

	for _, dep := range deps {
		if !dep.IsDev {
			t.Errorf("dep %s should be dev", dep.Name)
		}
	}
}

func TestParsePyprojectTomlPEP621(t *testing.T) {
	dir := t.TempDir()
	content := `
[project]
name = "myapp"
version = "1.0.0"
dependencies = [
    "fastapi>=0.100.0",
    "uvicorn[standard]>=0.23.0",
    "sqlalchemy>=2.0",
]

[project.optional-dependencies]
dev = [
    "pytest>=7.0",
    "black>=23.0",
]
`
	os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte(content), 0o644)

	deps, err := parsePyprojectToml(filepath.Join(dir, "pyproject.toml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(deps) < 3 {
		t.Errorf("got %d deps, want at least 3", len(deps))
		for _, d := range deps {
			t.Logf("  dep: %s %s dev=%v", d.Name, d.Version, d.IsDev)
		}
	}

	foundFastapi := false
	for _, dep := range deps {
		if dep.Name == "fastapi" {
			foundFastapi = true
			if dep.IsDev {
				t.Error("fastapi should not be dev")
			}
		}
	}
	if !foundFastapi {
		t.Error("missing dependency: fastapi")
	}
}

func TestParsePyprojectTomlPoetry(t *testing.T) {
	dir := t.TempDir()
	content := `
[tool.poetry]
name = "myapp"
version = "1.0.0"

[tool.poetry.dependencies]
python = "^3.11"
django = "^4.2"
celery = {version = "^5.3", extras = ["redis"]}

[tool.poetry.dev-dependencies]
pytest = "^7.0"
`
	os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte(content), 0o644)

	deps, err := parsePyprojectToml(filepath.Join(dir, "pyproject.toml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	foundDjango := false
	foundPython := false
	for _, dep := range deps {
		if dep.Name == "django" {
			foundDjango = true
			if dep.IsDev {
				t.Error("django should not be dev")
			}
		}
		if dep.Name == "python" {
			foundPython = true
		}
	}
	if !foundDjango {
		t.Error("missing dependency: django")
	}
	if foundPython {
		t.Error("python itself should be excluded")
	}
}

func TestParsePipfile(t *testing.T) {
	dir := t.TempDir()
	content := `
[packages]
flask = ">=2.0"
requests = "*"
sqlalchemy = {version = ">=2.0"}

[dev-packages]
pytest = ">=7.0"
black = "*"
`
	os.WriteFile(filepath.Join(dir, "Pipfile"), []byte(content), 0o644)

	deps, err := parsePipfile(filepath.Join(dir, "Pipfile"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(deps) < 4 {
		t.Errorf("got %d deps, want at least 4", len(deps))
	}

	devCount := 0
	for _, dep := range deps {
		if dep.IsDev {
			devCount++
		}
	}
	if devCount != 2 {
		t.Errorf("got %d dev deps, want 2", devCount)
	}
}
