package watcher

import (
	"reflect"
	"sort"
	"testing"

	"github.com/qualhook/qualhook/pkg/config"
)

func TestFileMapper_MapFilesToComponents(t *testing.T) {
	// Create a test configuration
	testConfig := &config.Config{
		Version: "1.0",
		Commands: map[string]*config.CommandConfig{
			"lint": {
				Command: "npm",
				Args:    []string{"run", "lint"},
			},
			"test": {
				Command: "npm",
				Args:    []string{"test"},
			},
		},
		Paths: []*config.PathConfig{
			{
				Path: "frontend/**",
				Commands: map[string]*config.CommandConfig{
					"lint": {
						Command: "npm",
						Args:    []string{"run", "lint", "--prefix", "frontend"},
					},
				},
			},
			{
				Path: "backend/**",
				Commands: map[string]*config.CommandConfig{
					"lint": {
						Command: "go",
						Args:    []string{"vet", "./..."},
					},
					"test": {
						Command: "go",
						Args:    []string{"test", "./..."},
					},
				},
			},
			{
				Path: "frontend/components/**",
				Commands: map[string]*config.CommandConfig{
					"lint": {
						Command: "npm",
						Args:    []string{"run", "lint:components", "--prefix", "frontend"},
					},
				},
			},
		},
	}

	mapper := NewFileMapper(testConfig)

	tests := []struct {
		name     string
		files    []string
		expected []ComponentGroup
	}{
		{
			name:  "empty files",
			files: []string{},
			expected: nil,
		},
		{
			name:  "single frontend file",
			files: []string{"frontend/src/app.js"},
			expected: []ComponentGroup{
				{
					Path:  "frontend/**",
					Files: []string{"frontend/src/app.js"},
					Config: map[string]*config.CommandConfig{
						"lint": {
							Command: "npm",
							Args:    []string{"run", "lint", "--prefix", "frontend"},
						},
						"test": {
							Command: "npm",
							Args:    []string{"test"},
						},
					},
				},
			},
		},
		{
			name:  "frontend component file (more specific match)",
			files: []string{"frontend/components/button.js"},
			expected: []ComponentGroup{
				{
					Path:  "frontend/components/**",
					Files: []string{"frontend/components/button.js"},
					Config: map[string]*config.CommandConfig{
						"lint": {
							Command: "npm",
							Args:    []string{"run", "lint:components", "--prefix", "frontend"},
						},
						"test": {
							Command: "npm",
							Args:    []string{"test"},
						},
					},
				},
			},
		},
		{
			name:  "multiple files in same component",
			files: []string{"backend/main.go", "backend/handler/api.go"},
			expected: []ComponentGroup{
				{
					Path:  "backend/**",
					Files: []string{"backend/main.go", "backend/handler/api.go"},
					Config: map[string]*config.CommandConfig{
						"lint": {
							Command: "go",
							Args:    []string{"vet", "./..."},
						},
						"test": {
							Command: "go",
							Args:    []string{"test", "./..."},
						},
					},
				},
			},
		},
		{
			name: "files across multiple components",
			files: []string{
				"frontend/src/app.js",
				"backend/main.go",
				"README.md",
			},
			expected: []ComponentGroup{
				{
					Path:  ".",
					Files: []string{"README.md"},
					Config: map[string]*config.CommandConfig{
						"lint": {
							Command: "npm",
							Args:    []string{"run", "lint"},
						},
						"test": {
							Command: "npm",
							Args:    []string{"test"},
						},
					},
				},
				{
					Path:  "backend/**",
					Files: []string{"backend/main.go"},
					Config: map[string]*config.CommandConfig{
						"lint": {
							Command: "go",
							Args:    []string{"vet", "./..."},
						},
						"test": {
							Command: "go",
							Args:    []string{"test", "./..."},
						},
					},
				},
				{
					Path:  "frontend/**",
					Files: []string{"frontend/src/app.js"},
					Config: map[string]*config.CommandConfig{
						"lint": {
							Command: "npm",
							Args:    []string{"run", "lint", "--prefix", "frontend"},
						},
						"test": {
							Command: "npm",
							Args:    []string{"test"},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			groups, err := mapper.MapFilesToComponents(tt.files)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !compareComponentGroups(groups, tt.expected) {
				t.Errorf("MapFilesToComponents() = %v, want %v", groups, tt.expected)
			}
		})
	}
}

func TestFileMapper_matchesPath(t *testing.T) {
	mapper := &FileMapper{}

	tests := []struct {
		name        string
		filePath    string
		pattern     string
		wantMatch   bool
		wantSpecMin int // minimum expected specificity
	}{
		{
			name:        "exact match",
			filePath:    "frontend/src/app.js",
			pattern:     "frontend/src/app.js",
			wantMatch:   true,
			wantSpecMin: 20,
		},
		{
			name:        "wildcard match",
			filePath:    "frontend/src/app.js",
			pattern:     "frontend/**",
			wantMatch:   true,
			wantSpecMin: 0,
		},
		{
			name:        "nested wildcard match",
			filePath:    "frontend/components/button/index.js",
			pattern:     "frontend/components/**",
			wantMatch:   true,
			wantSpecMin: 10,
		},
		{
			name:        "no match",
			filePath:    "backend/main.go",
			pattern:     "frontend/**",
			wantMatch:   false,
			wantSpecMin: -1,
		},
		{
			name:        "single star match",
			filePath:    "src/test.js",
			pattern:     "src/*.js",
			wantMatch:   true,
			wantSpecMin: 5,
		},
		{
			name:        "question mark match",
			filePath:    "src/a.js",
			pattern:     "src/?.js",
			wantMatch:   true,
			wantSpecMin: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			match, spec := mapper.matchesPath(tt.filePath, tt.pattern)
			if match != tt.wantMatch {
				t.Errorf("matchesPath() match = %v, want %v", match, tt.wantMatch)
			}
			if match && spec < tt.wantSpecMin {
				t.Errorf("matchesPath() specificity = %v, want >= %v", spec, tt.wantSpecMin)
			}
		})
	}
}

func TestCalculateSpecificity(t *testing.T) {
	tests := []struct {
		pattern  string
		pattern2 string
		wantCmp  string // "<", ">", or "="
	}{
		{
			pattern:  "frontend/components/**",
			pattern2: "frontend/**",
			wantCmp:  ">", // more specific
		},
		{
			pattern:  "src/components/button.js",
			pattern2: "src/components/*.js",
			wantCmp:  ">", // exact path is more specific
		},
		{
			pattern:  "a/b/c/**",
			pattern2: "a/**",
			wantCmp:  ">", // deeper path is more specific
		},
		{
			pattern:  "src/*.js",
			pattern2: "src/*.ts",
			wantCmp:  "=", // same specificity
		},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+" vs "+tt.pattern2, func(t *testing.T) {
			spec1 := calculateSpecificity(tt.pattern)
			spec2 := calculateSpecificity(tt.pattern2)

			var got string
			if spec1 > spec2 {
				got = ">"
			} else if spec1 < spec2 {
				got = "<"
			} else {
				got = "="
			}

			if got != tt.wantCmp {
				t.Errorf("calculateSpecificity(%q)=%d vs calculateSpecificity(%q)=%d, got %s, want %s",
					tt.pattern, spec1, tt.pattern2, spec2, got, tt.wantCmp)
			}
		})
	}
}

func TestFileMapper_mergeConfigs(t *testing.T) {
	// Test configuration with extends
	testConfig := &config.Config{
		Version: "1.0",
		Commands: map[string]*config.CommandConfig{
			"format": {
				Command: "prettier",
				Args:    []string{"--write"},
			},
			"lint": {
				Command: "eslint",
			},
		},
		Paths: []*config.PathConfig{
			{
				Path: "base/**",
				Commands: map[string]*config.CommandConfig{
					"lint": {
						Command: "base-lint",
					},
					"test": {
						Command: "jest",
					},
				},
			},
			{
				Path:    "frontend/**",
				Extends: "base/**",
				Commands: map[string]*config.CommandConfig{
					"lint": {
						Command: "frontend-lint",
					},
				},
			},
		},
	}

	mapper := NewFileMapper(testConfig)

	// Test merging with extends
	pathConfig := testConfig.Paths[1] // frontend/**
	merged := mapper.mergeConfigs(pathConfig)

	// Should have format from root, test from base, and lint from frontend
	if merged["format"].Command != "prettier" {
		t.Errorf("format command = %s, want prettier", merged["format"].Command)
	}
	if merged["test"].Command != "jest" {
		t.Errorf("test command = %s, want jest", merged["test"].Command)
	}
	if merged["lint"].Command != "frontend-lint" {
		t.Errorf("lint command = %s, want frontend-lint", merged["lint"].Command)
	}
}

// Helper function to compare component groups
func compareComponentGroups(a, b []ComponentGroup) bool {
	if len(a) != len(b) {
		return false
	}

	// Sort both slices by path for consistent comparison
	sort.Slice(a, func(i, j int) bool { return a[i].Path < a[j].Path })
	sort.Slice(b, func(i, j int) bool { return b[i].Path < b[j].Path })

	for i := range a {
		if a[i].Path != b[i].Path {
			return false
		}

		// Sort files for comparison
		sort.Strings(a[i].Files)
		sort.Strings(b[i].Files)
		if !reflect.DeepEqual(a[i].Files, b[i].Files) {
			return false
		}

		// Compare configs
		if len(a[i].Config) != len(b[i].Config) {
			return false
		}

		for cmd, cfg := range a[i].Config {
			if bCfg, ok := b[i].Config[cmd]; !ok {
				return false
			} else if cfg.Command != bCfg.Command {
				return false
			} else if !reflect.DeepEqual(cfg.Args, bCfg.Args) {
				return false
			}
		}
	}

	return true
}