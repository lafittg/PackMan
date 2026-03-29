package nodejs

import (
	"testing"
)

func TestResolvePackageName(t *testing.T) {
	tests := []struct {
		specifier string
		want      string
	}{
		{"express", "express"},
		{"lodash/get", "lodash"},
		{"@scope/pkg", "@scope/pkg"},
		{"@scope/pkg/deep/path", "@scope/pkg"},
		{"./local", ""},
		{"../parent", ""},
		{"/absolute", ""},
	}

	for _, tt := range tests {
		got := resolvePackageName(tt.specifier)
		if got != tt.want {
			t.Errorf("resolvePackageName(%q) = %q, want %q", tt.specifier, got, tt.want)
		}
	}
}

func TestFindImportsJS(t *testing.T) {
	known := map[string]bool{
		"express": true,
		"lodash":  true,
		"axios":   true,
		"unused":  true,
	}

	source := []byte(`
const express = require('express');
import _ from 'lodash';
import { get } from 'lodash/fp';
const axios = require("axios");
import('./dynamic-only');
import type { Config } from 'express';
`)

	result, err := findImportsJS("test.js", source, known)
	if err != nil {
		t.Fatal(err)
	}

	if len(result["express"]) != 2 { // require + import type
		t.Errorf("express: got %d imports, want 2", len(result["express"]))
	}
	if len(result["lodash"]) != 2 { // default import + deep import
		t.Errorf("lodash: got %d imports, want 2", len(result["lodash"]))
	}
	if len(result["axios"]) != 1 {
		t.Errorf("axios: got %d imports, want 1", len(result["axios"]))
	}
	if len(result["unused"]) != 0 {
		t.Errorf("unused: got %d imports, want 0", len(result["unused"]))
	}
}
