package config

import (
	"testing"

	"github.com/bebsworthy/qualhook/pkg/config"
)

func TestNewDefaultConfigs(t *testing.T) {
	dc, err := NewDefaultConfigs()
	if err != nil {
		t.Fatalf("Failed to create default configs: %v", err)
	}

	// Check that all expected project types are loaded
	expectedTypes := []ProjectType{
		ProjectTypeNodeJS,
		ProjectTypeGo,
		ProjectTypePython,
		ProjectTypeRust,
	}

	for _, pt := range expectedTypes {
		if _, ok := dc.configs[pt]; !ok {
			t.Errorf("Expected project type %s to be loaded", pt)
		}
	}
}

func TestDefaultConfigs_GetConfig(t *testing.T) {
	dc, err := NewDefaultConfigs()
	if err != nil {
		t.Fatalf("Failed to create default configs: %v", err)
	}

	tests := []struct {
		projectType ProjectType
		wantErr     bool
	}{
		{ProjectTypeNodeJS, false},
		{ProjectTypeGo, false},
		{ProjectTypePython, false},
		{ProjectTypeRust, false},
		{ProjectTypeUnknown, true},
		{ProjectType("invalid"), true},
	}

	for _, tt := range tests {
		t.Run(string(tt.projectType), func(t *testing.T) {
			cfg, err := dc.GetConfig(tt.projectType)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetConfig() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && cfg == nil {
				t.Error("Expected non-nil config")
			}

			if !tt.wantErr {
				// Verify basic structure
				if cfg.Version == "" {
					t.Error("Config version should not be empty")
				}
				if len(cfg.Commands) == 0 {
					t.Error("Config should have commands")
				}
			}
		})
	}
}

func TestDefaultConfigs_NodeJSConfig(t *testing.T) {
	dc, err := NewDefaultConfigs()
	if err != nil {
		t.Fatalf("Failed to create default configs: %v", err)
	}

	cfg, err := dc.GetConfig(ProjectTypeNodeJS)
	if err != nil {
		t.Fatalf("Failed to get Node.js config: %v", err)
	}

	// Check expected commands
	expectedCommands := []string{"format", "lint", "typecheck", "test"}
	for _, cmd := range expectedCommands {
		if _, ok := cfg.Commands[cmd]; !ok {
			t.Errorf("Expected Node.js config to have %s command", cmd)
		}
	}

	// Check lint command specifics
	lintCmd := cfg.Commands["lint"]
	if lintCmd.Command != "npm" {
		t.Errorf("Expected lint command to be 'npm', got %s", lintCmd.Command)
	}

	// Check error patterns
	if lintCmd.ErrorPatterns == nil || len(lintCmd.ErrorPatterns) == 0 {
		t.Error("Lint command should have error patterns")
	}
}

func TestDefaultConfigs_GoConfig(t *testing.T) {
	dc, err := NewDefaultConfigs()
	if err != nil {
		t.Fatalf("Failed to create default configs: %v", err)
	}

	cfg, err := dc.GetConfig(ProjectTypeGo)
	if err != nil {
		t.Fatalf("Failed to get Go config: %v", err)
	}

	// Check expected commands
	expectedCommands := []string{"format", "lint", "typecheck", "test", "vet"}
	for _, cmd := range expectedCommands {
		if _, ok := cfg.Commands[cmd]; !ok {
			t.Errorf("Expected Go config to have %s command", cmd)
		}
	}

	// Check format command
	formatCmd := cfg.Commands["format"]
	if formatCmd.Command != "go" {
		t.Errorf("Expected format command to be 'go', got %s", formatCmd.Command)
	}

	// Check lint command uses golangci-lint
	lintCmd := cfg.Commands["lint"]
	if lintCmd.Command != "golangci-lint" {
		t.Errorf("Expected lint command to be 'golangci-lint', got %s", lintCmd.Command)
	}
}

func TestDefaultConfigs_GetCommonErrorPatterns(t *testing.T) {
	dc, err := NewDefaultConfigs()
	if err != nil {
		t.Fatalf("Failed to create default configs: %v", err)
	}

	tests := []struct {
		projectType ProjectType
		minPatterns int
	}{
		{ProjectTypeNodeJS, 5},
		{ProjectTypeGo, 5},
		{ProjectTypePython, 5},
		{ProjectTypeRust, 5},
	}

	for _, tt := range tests {
		t.Run(string(tt.projectType), func(t *testing.T) {
			patterns, err := dc.GetCommonErrorPatterns(tt.projectType)
			if err != nil {
				t.Fatalf("Failed to get error patterns: %v", err)
			}

			if len(patterns) < tt.minPatterns {
				t.Errorf("Expected at least %d patterns, got %d", tt.minPatterns, len(patterns))
			}

			// Verify patterns are valid
			for i, p := range patterns {
				if err := p.Validate(); err != nil {
					t.Errorf("Pattern %d is invalid: %v", i, err)
				}
			}
		})
	}
}

func TestDefaultConfigs_MergeWithDefaults(t *testing.T) {
	dc, err := NewDefaultConfigs()
	if err != nil {
		t.Fatalf("Failed to create default configs: %v", err)
	}

	// Create a user config that overrides some values
	userConfig := &config.Config{
		Version: "2.0",
		Commands: map[string]*config.CommandConfig{
			"lint": {
				Command: "custom-linter",
				Args:    []string{"--strict"},
				ErrorPatterns: []*config.RegexPattern{
					{Pattern: "CUSTOM_ERROR", Flags: ""},
				},
				MaxOutput: 50,
			},
			"custom": {
				Command: "my-tool",
				Args:    []string{"check"},
				ErrorPatterns: []*config.RegexPattern{
					{Pattern: "problem", Flags: "i"},
				},
			},
		},
	}

	merged, err := dc.MergeWithDefaults(userConfig, ProjectTypeNodeJS)
	if err != nil {
		t.Fatalf("Failed to merge configs: %v", err)
	}

	// Check that version was overridden
	if merged.Version != "2.0" {
		t.Errorf("Expected version 2.0, got %s", merged.Version)
	}

	// Check that lint command was overridden
	lintCmd := merged.Commands["lint"]
	if lintCmd.Command != "custom-linter" {
		t.Errorf("Expected lint command to be 'custom-linter', got %s", lintCmd.Command)
	}

	// Check that custom command was added
	if _, ok := merged.Commands["custom"]; !ok {
		t.Error("Expected custom command to be added")
	}

	// Check that default commands are still present
	if _, ok := merged.Commands["format"]; !ok {
		t.Error("Expected format command from defaults to be present")
	}
}

func TestDetectProjectType(t *testing.T) {
	tests := []struct {
		markers     []string
		expected    ProjectType
		description string
	}{
		{
			markers:     []string{"package.json", "node_modules"},
			expected:    ProjectTypeNodeJS,
			description: "Node.js with package.json",
		},
		{
			markers:     []string{"yarn.lock", "src"},
			expected:    ProjectTypeNodeJS,
			description: "Node.js with yarn.lock",
		},
		{
			markers:     []string{"go.mod", "main.go"},
			expected:    ProjectTypeGo,
			description: "Go with go.mod",
		},
		{
			markers:     []string{"requirements.txt", "setup.py"},
			expected:    ProjectTypePython,
			description: "Python with requirements.txt",
		},
		{
			markers:     []string{"Cargo.toml", "src"},
			expected:    ProjectTypeRust,
			description: "Rust with Cargo.toml",
		},
		{
			markers:     []string{"README.md", ".gitignore"},
			expected:    ProjectTypeUnknown,
			description: "Unknown project type",
		},
		{
			markers:     []string{},
			expected:    ProjectTypeUnknown,
			description: "No markers",
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			result := DetectProjectType(tt.markers)
			if result != tt.expected {
				t.Errorf("DetectProjectType(%v) = %v, want %v", tt.markers, result, tt.expected)
			}
		})
	}
}

func TestDefaultConfigs_ExportTemplate(t *testing.T) {
	dc, err := NewDefaultConfigs()
	if err != nil {
		t.Fatalf("Failed to create default configs: %v", err)
	}

	projectTypes := []ProjectType{
		ProjectTypeNodeJS,
		ProjectTypeGo,
		ProjectTypePython,
		ProjectTypeRust,
	}

	for _, pt := range projectTypes {
		t.Run(string(pt), func(t *testing.T) {
			data, err := dc.ExportTemplate(pt)
			if err != nil {
				t.Fatalf("Failed to export template: %v", err)
			}

			// Verify it's valid JSON by loading it back
			cfg, err := config.LoadConfig(data)
			if err != nil {
				t.Errorf("Exported template is not valid JSON: %v", err)
			}

			if cfg.ProjectType != string(pt) {
				t.Errorf("Expected project type %s, got %s", pt, cfg.ProjectType)
			}
		})
	}
}

func TestDefaultConfigs_CloneIsolation(t *testing.T) {
	dc, err := NewDefaultConfigs()
	if err != nil {
		t.Fatalf("Failed to create default configs: %v", err)
	}

	// Get two copies of the same config
	cfg1, err := dc.GetConfig(ProjectTypeNodeJS)
	if err != nil {
		t.Fatalf("Failed to get config 1: %v", err)
	}

	cfg2, err := dc.GetConfig(ProjectTypeNodeJS)
	if err != nil {
		t.Fatalf("Failed to get config 2: %v", err)
	}

	// Modify cfg1
	cfg1.Version = testModifiedValue
	cfg1.Commands["lint"].Command = "modified-linter"

	// Verify cfg2 is not affected
	if cfg2.Version == testModifiedValue {
		t.Error("Config 2 should not be modified when config 1 is changed")
	}

	if cfg2.Commands["lint"].Command == "modified-linter" {
		t.Error("Config 2 commands should not be modified when config 1 is changed")
	}

	// Verify original is not affected
	cfg3, err := dc.GetConfig(ProjectTypeNodeJS)
	if err != nil {
		t.Fatalf("Failed to get config 3: %v", err)
	}

	if cfg3.Version == testModifiedValue {
		t.Error("Original config should not be modified")
	}
}
