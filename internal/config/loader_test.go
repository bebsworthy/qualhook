package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bebsworthy/qualhook/pkg/config"
)

func TestLoader_Load(t *testing.T) {
	// Create a temporary directory for test files
	tempDir := t.TempDir()

	// Create a test configuration
	testConfig := &config.Config{
		Version: "1.0",
		Commands: map[string]*config.CommandConfig{
			"lint": {
				Command: "npm",
				Args:    []string{"run", "lint"},
				ErrorDetection: &config.ErrorDetection{
					ExitCodes: []int{1},
				},
				OutputFilter: &config.FilterConfig{
					ErrorPatterns: []*config.RegexPattern{
						{Pattern: "error", Flags: "i"},
					},
					MaxOutput: 100,
				},
			},
		},
	}

	// Write test configuration to file
	configPath := filepath.Join(tempDir, ConfigFileName)
	data, err := json.MarshalIndent(testConfig, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal test config: %v", err)
	}
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Test loading from search path
	loader := &Loader{
		SearchPaths: []string{tempDir},
	}

	cfg, err := loader.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if cfg.Version != testConfig.Version {
		t.Errorf("Expected version %s, got %s", testConfig.Version, cfg.Version)
	}

	if len(cfg.Commands) != len(testConfig.Commands) {
		t.Errorf("Expected %d commands, got %d", len(testConfig.Commands), len(cfg.Commands))
	}
}

func TestLoader_LoadFromEnv(t *testing.T) {
	// Create a temporary directory for test files
	tempDir := t.TempDir()

	// Create a test configuration
	testConfig := &config.Config{
		Version: "1.0",
		Commands: map[string]*config.CommandConfig{
			"format": {
				Command: "prettier",
				Args:    []string{"--write", "."},
				ErrorDetection: &config.ErrorDetection{
					ExitCodes: []int{1},
				},
				OutputFilter: &config.FilterConfig{
					ErrorPatterns: []*config.RegexPattern{
						{Pattern: "error", Flags: "i"},
					},
				},
			},
		},
	}

	// Write test configuration to custom path
	configPath := filepath.Join(tempDir, "custom-config.json")
	data, err := json.MarshalIndent(testConfig, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal test config: %v", err)
	}
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Set environment variable
	oldEnv := os.Getenv(ConfigEnvVar)
	os.Setenv(ConfigEnvVar, configPath)
	defer os.Setenv(ConfigEnvVar, oldEnv)

	loader := NewLoader()
	cfg, err := loader.Load()
	if err != nil {
		t.Fatalf("Failed to load config from env: %v", err)
	}

	if cfg.Version != testConfig.Version {
		t.Errorf("Expected version %s, got %s", testConfig.Version, cfg.Version)
	}
}

func TestLoader_LoadForMonorepo(t *testing.T) {
	// Create a temporary directory structure
	tempDir := t.TempDir()
	frontendDir := filepath.Join(tempDir, "frontend")
	backendDir := filepath.Join(tempDir, "backend")
	os.MkdirAll(frontendDir, 0755)
	os.MkdirAll(backendDir, 0755)

	// Create a monorepo configuration
	monorepoConfig := &config.Config{
		Version: "1.0",
		Commands: map[string]*config.CommandConfig{
			"lint": {
				Command: "npm",
				Args:    []string{"run", "lint"},
				ErrorDetection: &config.ErrorDetection{
					ExitCodes: []int{1},
				},
				OutputFilter: &config.FilterConfig{
					ErrorPatterns: []*config.RegexPattern{
						{Pattern: "error", Flags: "i"},
					},
				},
			},
		},
		Paths: []*config.PathConfig{
			{
				Path: "frontend/**",
				Commands: map[string]*config.CommandConfig{
					"lint": {
						Command: "npm",
						Args:    []string{"run", "lint", "--prefix", "frontend"},
						ErrorDetection: &config.ErrorDetection{
							ExitCodes: []int{1},
						},
						OutputFilter: &config.FilterConfig{
							ErrorPatterns: []*config.RegexPattern{
								{Pattern: "eslint", Flags: "i"},
							},
						},
					},
				},
			},
			{
				Path: "backend/**",
				Commands: map[string]*config.CommandConfig{
					"lint": {
						Command: "go",
						Args:    []string{"vet", "./..."},
						ErrorDetection: &config.ErrorDetection{
							ExitCodes: []int{1},
						},
						OutputFilter: &config.FilterConfig{
							ErrorPatterns: []*config.RegexPattern{
								{Pattern: "vet:", Flags: ""},
							},
						},
					},
				},
			},
		},
	}

	// Write configuration to root
	configPath := filepath.Join(tempDir, ConfigFileName)
	data, err := json.MarshalIndent(monorepoConfig, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	loader := &Loader{
		SearchPaths: []string{tempDir},
	}

	// Test loading for frontend directory
	cfg, err := loader.LoadForMonorepo(frontendDir)
	if err != nil {
		t.Fatalf("Failed to load monorepo config: %v", err)
	}

	lintCmd := cfg.Commands["lint"]
	if lintCmd == nil {
		t.Fatal("Expected lint command to exist")
	}

	// Should have frontend-specific args
	expectedArgs := []string{"run", "lint", "--prefix", "frontend"}
	if len(lintCmd.Args) != len(expectedArgs) {
		t.Errorf("Expected %d args, got %d", len(expectedArgs), len(lintCmd.Args))
	}
	for i, arg := range expectedArgs {
		if i < len(lintCmd.Args) && lintCmd.Args[i] != arg {
			t.Errorf("Expected arg[%d] to be %s, got %s", i, arg, lintCmd.Args[i])
		}
	}

	// Test loading for backend directory
	cfg, err = loader.LoadForMonorepo(backendDir)
	if err != nil {
		t.Fatalf("Failed to load backend config: %v", err)
	}

	lintCmd = cfg.Commands["lint"]
	if lintCmd.Command != "go" {
		t.Errorf("Expected command to be 'go', got %s", lintCmd.Command)
	}
}

func TestLoader_LoadErrors(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr string
	}{
		{
			name:    "invalid JSON",
			content: "{ invalid json",
			wantErr: "invalid character",
		},
		{
			name: "missing version",
			content: `{
				"commands": {
					"lint": {
						"command": "npm",
						"args": ["run", "lint"]
					}
				}
			}`,
			wantErr: "version is required",
		},
		{
			name: "invalid command",
			content: `{
				"version": "1.0",
				"commands": {
					"lint": {
						"args": ["run", "lint"]
					}
				}
			}`,
			wantErr: "command is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			configPath := filepath.Join(tempDir, ConfigFileName)
			
			if err := os.WriteFile(configPath, []byte(tt.content), 0644); err != nil {
				t.Fatalf("Failed to write test config: %v", err)
			}

			loader := &Loader{
				SearchPaths: []string{tempDir},
			}

			_, err := loader.Load()
			if err == nil {
				t.Fatal("Expected error, got nil")
			}

			// Check that error contains expected message
			if !containsError(err.Error(), tt.wantErr) {
				t.Errorf("Expected error containing %q, got %q", tt.wantErr, err.Error())
			}
		})
	}
}

func TestLoader_NoConfigFound(t *testing.T) {
	tempDir := t.TempDir()
	loader := &Loader{
		SearchPaths: []string{tempDir},
	}

	_, err := loader.Load()
	if err == nil {
		t.Fatal("Expected error when no config found")
	}

	if !containsError(err.Error(), "no configuration file found") {
		t.Errorf("Expected 'no configuration file found' error, got: %v", err)
	}
}

func TestMatchesPath(t *testing.T) {
	tests := []struct {
		relPath     string
		pattern     string
		wantMatch   bool
		wantMatchLen int
	}{
		// Exact matches
		{"frontend", "frontend", true, 8},
		{"backend", "backend", true, 7},
		
		// Directory matches
		{"frontend", "frontend/", true, 9},
		{"frontend/src", "frontend/", true, 9},
		
		// Recursive matches
		{"frontend", "frontend/**", true, 8},
		{"frontend/src", "frontend/**", true, 8},
		{"frontend/src/components", "frontend/**", true, 8},
		
		// Non-matches
		{"backend", "frontend", false, 0},
		{"src", "frontend/", false, 0},
		{"", "frontend", false, 0},
	}

	for _, tt := range tests {
		t.Run(tt.relPath+"_"+tt.pattern, func(t *testing.T) {
			gotMatch, gotLen := matchesPath(tt.relPath, tt.pattern)
			if gotMatch != tt.wantMatch {
				t.Errorf("matchesPath(%q, %q) match = %v, want %v", tt.relPath, tt.pattern, gotMatch, tt.wantMatch)
			}
			if gotMatch && gotLen != tt.wantMatchLen {
				t.Errorf("matchesPath(%q, %q) matchLen = %v, want %v", tt.relPath, tt.pattern, gotLen, tt.wantMatchLen)
			}
		})
	}
}

func TestValidateConfigFile(t *testing.T) {
	tempDir := t.TempDir()

	// Valid config
	validConfig := &config.Config{
		Version: "1.0",
		Commands: map[string]*config.CommandConfig{
			"lint": {
				Command: "npm",
				Args:    []string{"run", "lint"},
				ErrorDetection: &config.ErrorDetection{
					ExitCodes: []int{1},
				},
				OutputFilter: &config.FilterConfig{
					ErrorPatterns: []*config.RegexPattern{
						{Pattern: "error", Flags: "i"},
					},
				},
			},
		},
	}

	validPath := filepath.Join(tempDir, "valid.json")
	data, _ := json.MarshalIndent(validConfig, "", "  ")
	os.WriteFile(validPath, data, 0644)

	if err := ValidateConfigFile(validPath); err != nil {
		t.Errorf("Expected valid config to pass validation: %v", err)
	}

	// Invalid config
	invalidPath := filepath.Join(tempDir, "invalid.json")
	os.WriteFile(invalidPath, []byte(`{"version": ""}`), 0644)

	if err := ValidateConfigFile(invalidPath); err == nil {
		t.Error("Expected invalid config to fail validation")
	}
}

// Helper function to check if error message contains substring
func containsError(errMsg, want string) bool {
	return strings.Contains(errMsg, want)
}