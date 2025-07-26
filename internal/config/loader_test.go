//go:build unit

package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bebsworthy/qualhook/internal/testutil"
	"github.com/bebsworthy/qualhook/pkg/config"
)

func TestLoader_Load(t *testing.T) {
	// Create a temporary directory for test files
	tempDir := t.TempDir()

	// Create a test configuration using ConfigBuilder
	testConfig := testutil.NewConfigBuilder().
		WithSimpleCommand("lint", "npm", "run", "lint").
		Build()

	// Write test configuration to file
	configPath := filepath.Join(tempDir, ConfigFileName)
	builder := testutil.NewConfigBuilder().
		WithSimpleCommand("lint", "npm", "run", "lint")
	if err := builder.WriteToFile(configPath); err != nil {
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

	// Create a test configuration using ConfigBuilder
	testConfig := testutil.NewConfigBuilder().
		WithSimpleCommand("format", "prettier", "--write", ".").
		Build()

	// Write test configuration to custom path
	configPath := filepath.Join(tempDir, "custom-config.json")
	builder := testutil.NewConfigBuilder().
		WithSimpleCommand("format", "prettier", "--write", ".")
	if err := builder.WriteToFile(configPath); err != nil {
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

	// Create a monorepo configuration using ConfigBuilder
	monorepoConfig := testutil.NewConfigBuilder().
		WithSimpleCommand("lint", "npm", "run", "lint").
		WithPathCommand("frontend/**", map[string]*config.CommandConfig{
			"lint": {
				Command:       "npm",
				Args:          []string{"run", "lint", "--prefix", "frontend"},
				ExitCodes:     []int{1},
				ErrorPatterns: []*config.RegexPattern{{Pattern: "eslint", Flags: "i"}},
				MaxOutput:     100,
			},
		}).
		WithPathCommand("backend/**", map[string]*config.CommandConfig{
			"lint": {
				Command:       "go",
				Args:          []string{"vet", "./..."},
				ExitCodes:     []int{1},
				ErrorPatterns: []*config.RegexPattern{{Pattern: "vet:", Flags: ""}},
				MaxOutput:     100,
			},
		}).
		Build()

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
		relPath      string
		pattern      string
		wantMatch    bool
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

	// Valid config using ConfigBuilder
	validPath := filepath.Join(tempDir, "valid.json")
	validBuilder := testutil.NewConfigBuilder().
		WithSimpleCommand("lint", "npm", "run", "lint")
	if err := validBuilder.WriteToFile(validPath); err != nil {
		t.Fatalf("Failed to write valid config: %v", err)
	}

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

// TestLoader_LoadFixtures tests loading configuration from fixture files
func TestLoader_LoadFixtures(t *testing.T) {
	tests := []struct {
		name         string
		fixtureName  string
		expectValid  bool
		checkCommand string // Command to verify if loaded
		expectError  string // Expected error message for invalid configs
	}{
		{
			name:         "basic config",
			fixtureName:  "basic",
			expectValid:  true,
			checkCommand: "lint",
		},
		{
			name:         "complex config",
			fixtureName:  "complex",
			expectValid:  true,
			checkCommand: "typecheck",
		},
		{
			name:         "golang config",
			fixtureName:  "golang",
			expectValid:  true,
			checkCommand: "vet",
		},
		{
			name:         "python config",
			fixtureName:  "python",
			expectValid:  true,
			checkCommand: "typecheck",
		},
		{
			name:         "monorepo config",
			fixtureName:  "monorepo",
			expectValid:  true,
			checkCommand: "lint:frontend",
		},
		{
			name:         "minimal config",
			fixtureName:  "minimal",
			expectValid:  true,
			checkCommand: "check",
		},
		{
			name:         "invalid config",
			fixtureName:  "invalid",
			expectValid:  false,
			expectError:  "version is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory and copy fixture with correct name
			tempDir := t.TempDir()
			configContent := testutil.LoadFixture(t, "configs/"+tt.fixtureName+".qualhook.json")
			configPath := filepath.Join(tempDir, ConfigFileName)
			if err := os.WriteFile(configPath, configContent, 0644); err != nil {
				t.Fatalf("Failed to write config: %v", err)
			}
			
			// Create loader pointing to temp directory
			loader := &Loader{
				SearchPaths: []string{tempDir},
			}
			
			// Try to load the config
			cfg, err := loader.Load()
			
			if tt.expectValid {
				if err != nil {
					t.Fatalf("Failed to load valid config: %v", err)
				}
				
				// Check that expected command exists
				if tt.checkCommand != "" {
					if _, exists := cfg.Commands[tt.checkCommand]; !exists {
						t.Errorf("Expected command %q not found in config", tt.checkCommand)
					}
				}
			} else {
				if err == nil {
					t.Fatal("Expected error loading invalid config, got nil")
				}
				
				// Verify error message contains expected text
				if tt.expectError != "" && !strings.Contains(err.Error(), tt.expectError) {
					t.Errorf("Expected error containing %q, got %q", tt.expectError, err.Error())
				}
			}
		})
	}
}

// TestLoader_ProjectFixtures tests loading configs from project fixture directories
func TestLoader_ProjectFixtures(t *testing.T) {
	projects := []struct {
		name         string
		projectType  string
		expectConfig bool
		checkCommand string
	}{
		{
			name:         "golang project",
			projectType:  "golang",
			expectConfig: true,
			checkCommand: "lint",
		},
		{
			name:         "nodejs project",
			projectType:  "nodejs",
			expectConfig: true,
			checkCommand: "test",
		},
		{
			name:         "python project",
			projectType:  "python",
			expectConfig: true,
			checkCommand: "lint",
		},
		{
			name:         "monorepo project",
			projectType:  "monorepo",
			expectConfig: true,
			checkCommand: "lint:all",
		},
	}

	for _, tt := range projects {
		t.Run(tt.name, func(t *testing.T) {
			// Get project fixture path
			projectPath := testutil.ProjectFixture(t, tt.projectType)
			
			// Create loader for project directory
			loader := &Loader{
				SearchPaths: []string{projectPath},
			}
			
			// Load config
			cfg, err := loader.Load()
			
			if tt.expectConfig {
				if err != nil {
					t.Fatalf("Failed to load config from project fixture: %v", err)
				}
				
				// Verify expected command exists
				if _, exists := cfg.Commands[tt.checkCommand]; !exists {
					t.Errorf("Expected command %q not found in project config", tt.checkCommand)
				}
			} else {
				if err == nil {
					t.Error("Expected no config, but found one")
				}
			}
		})
	}
}
