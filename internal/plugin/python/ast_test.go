package python

import (
	"testing"
)

func TestFindImportsPython(t *testing.T) {
	source := []byte(`
import requests
from flask import Flask, jsonify
from sqlalchemy import create_engine
import celery
from pydantic import BaseModel
import redis
from PIL import Image
import yaml
from sklearn.ensemble import RandomForestClassifier
import os  # stdlib, should be ignored
import json  # stdlib, should be ignored
from datetime import datetime  # stdlib, should be ignored
import numpy as np
from google.cloud import storage
`)

	known := map[string]bool{
		"requests":             true,
		"flask":                true,
		"SQLAlchemy":           true,
		"celery":               true,
		"pydantic":             true,
		"redis":                true,
		"Pillow":               true,
		"PyYAML":               true,
		"scikit-learn":         true,
		"numpy":                true,
		"google-cloud-storage": true,
	}

	result, err := findImportsPython("test.py", source, known)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{"requests", "flask", "SQLAlchemy", "celery", "pydantic", "redis", "Pillow", "PyYAML", "scikit-learn", "numpy"}
	for _, pkg := range expected {
		if _, ok := result[pkg]; !ok {
			t.Errorf("missing import detection for %s", pkg)
		}
	}

	// Stdlib should not be detected
	for _, pkg := range []string{"os", "json", "datetime"} {
		if _, ok := result[pkg]; ok {
			t.Errorf("should not detect stdlib: %s", pkg)
		}
	}
}

func TestResolvePackageName(t *testing.T) {
	known := map[string]bool{
		"Pillow":       true,
		"PyYAML":       true,
		"scikit-learn": true,
		"requests":     true,
		"flask":        true,
		"numpy":        true,
		"python-dotenv": true,
	}

	normalized := make(map[string]string, len(known))
	for pkg := range known {
		normalized[normalizePythonName(pkg)] = pkg
	}

	tests := []struct {
		module string
		want   string
	}{
		{"PIL", "Pillow"},
		{"PIL.Image", "Pillow"},
		{"yaml", "PyYAML"},
		{"sklearn", "scikit-learn"},
		{"sklearn.ensemble", "scikit-learn"},
		{"requests", "requests"},
		{"requests.auth", "requests"},
		{"flask", "flask"},
		{"numpy", "numpy"},
		{"dotenv", "python-dotenv"},
		{"os", ""},       // stdlib
		{"unknown", ""},  // not in known
	}

	for _, tt := range tests {
		got := resolvePackageName(tt.module, normalized)
		if got != tt.want {
			t.Errorf("resolvePackageName(%q) = %q, want %q", tt.module, got, tt.want)
		}
	}
}

func TestFindImportsMultipleImport(t *testing.T) {
	source := []byte(`import os, requests, json`)

	known := map[string]bool{
		"requests": true,
	}

	result, err := findImportsPython("test.py", source, known)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, ok := result["requests"]; !ok {
		t.Error("should detect requests in multi-import statement")
	}
}

func TestFindImportsWithAlias(t *testing.T) {
	source := []byte(`
import numpy as np
import pandas as pd
from matplotlib import pyplot as plt
`)

	known := map[string]bool{
		"numpy":      true,
		"pandas":     true,
		"matplotlib": true,
	}

	result, err := findImportsPython("test.py", source, known)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, pkg := range []string{"numpy", "pandas", "matplotlib"} {
		if _, ok := result[pkg]; !ok {
			t.Errorf("missing import detection for %s", pkg)
		}
	}
}
