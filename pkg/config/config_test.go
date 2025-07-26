//go:build unit

package config

import (
	"reflect"
	"strings"
	"testing"
)

// testConfigBuilder is a local helper to build test configs without import cycles
type testConfigBuilder struct {
	config *Config
}

func newTestConfigBuilder() *testConfigBuilder {
	return &testConfigBuilder{
		config: &Config{
			Version:  "1.0",
			Commands: make(map[string]*CommandConfig),
		},
	}
}

func (b *testConfigBuilder) withVersion(version string) *testConfigBuilder {
	b.config.Version = version
	return b
}

func (b *testConfigBuilder) withCommand(name string, cmd *CommandConfig) *testConfigBuilder {
	if b.config.Commands == nil {
		b.config.Commands = make(map[string]*CommandConfig)
	}
	b.config.Commands[name] = cmd
	return b
}

func (b *testConfigBuilder) withSimpleCommand(name, command string, args ...string) *testConfigBuilder {
	return b.withCommand(name, &CommandConfig{
		Command:       command,
		Args:          args,
		ExitCodes:     []int{1},
		ErrorPatterns: []*RegexPattern{{Pattern: "error", Flags: "i"}},
		MaxOutput:     100,
	})
}

func (b *testConfigBuilder) withPath(pathConfig *PathConfig) *testConfigBuilder {
	b.config.Paths = append(b.config.Paths, pathConfig)
	return b
}

func (b *testConfigBuilder) build() *Config {
	return b.config
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name      string
		buildFunc func() *Config
		wantErr   bool
		errMsg    string
	}{
		{
			name: "valid config with commands",
			buildFunc: func() *Config {
				return newTestConfigBuilder().
					withCommand("lint", &CommandConfig{
						Command:       "npm",
						Args:          []string{"run", "lint"},
						ExitCodes:     []int{1},
						ErrorPatterns: []*RegexPattern{{Pattern: "error"}},
					}).
					build()
			},
			wantErr: false,
		},
		{
			name: "missing version",
			buildFunc: func() *Config {
				return &Config{
					Commands: map[string]*CommandConfig{
						"lint": {Command: "npm"},
					},
				}
			},
			wantErr: true,
			errMsg:  "version is required",
		},
		{
			name: "no commands or paths",
			buildFunc: func() *Config {
				return newTestConfigBuilder().
					withVersion("1.0").
					build()
			},
			wantErr: true,
			errMsg:  "at least one command or path configuration is required",
		},
		{
			name: "invalid command config",
			buildFunc: func() *Config {
				return newTestConfigBuilder().
					withCommand("lint", &CommandConfig{}).
					build()
			},
			wantErr: true,
			errMsg:  "command \"lint\": command is required",
		},
		{
			name: "valid config with paths",
			buildFunc: func() *Config {
				return newTestConfigBuilder().
					withPath(&PathConfig{
						Path:     "frontend/**",
						Commands: map[string]*CommandConfig{"lint": {Command: "npm"}},
					}).
					build()
			},
			wantErr: false,
		},
		{
			name: "invalid path config",
			buildFunc: func() *Config {
				return newTestConfigBuilder().
					withPath(&PathConfig{
						Commands: map[string]*CommandConfig{"lint": {Command: "npm"}},
					}).
					build()
			},
			wantErr: true,
			errMsg:  "path config 0: path is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.buildFunc()
			err := cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("Config.Validate() error = %v, want error containing %q", err, tt.errMsg)
			}
		})
	}
}

func TestCommandConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *CommandConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid command config",
			config: &CommandConfig{
				Command: "npm",
				Args:    []string{"run", "lint"},
			},
			wantErr: false,
		},
		{
			name:    "missing command",
			config:  &CommandConfig{},
			wantErr: true,
			errMsg:  "command is required",
		},
		{
			name: "negative timeout",
			config: &CommandConfig{
				Command: "npm",
				Timeout: -1000,
			},
			wantErr: true,
			errMsg:  "timeout must be non-negative",
		},
		{
			name: "invalid error pattern",
			config: &CommandConfig{
				Command: "npm",
				ErrorPatterns: []*RegexPattern{
					{Pattern: "[invalid"},
				},
			},
			wantErr: true,
			errMsg:  "error pattern 0",
		},
		{
			name: "negative context lines",
			config: &CommandConfig{
				Command: "npm",
				ContextLines: -1,
			},
			wantErr: true,
			errMsg:  "context lines must be non-negative",
		},
		{
			name: "valid with all fields",
			config: &CommandConfig{
				Command: "npm",
				Args:    []string{"run", "lint"},
				ExitCodes: []int{1},
				ErrorPatterns: []*RegexPattern{
					{Pattern: "error", Flags: "i"},
				},
				ContextLines: 2,
				MaxOutput:    100,
				Prompt:  "Fix the errors:",
				Timeout: 60000,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("CommandConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("CommandConfig.Validate() error = %v, want error containing %q", err, tt.errMsg)
			}
		})
	}
}

func TestPathConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *PathConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid path config",
			config: &PathConfig{
				Path: "frontend/**",
				Commands: map[string]*CommandConfig{
					"lint": {Command: "npm"},
				},
			},
			wantErr: false,
		},
		{
			name: "missing path",
			config: &PathConfig{
				Commands: map[string]*CommandConfig{
					"lint": {Command: "npm"},
				},
			},
			wantErr: true,
			errMsg:  "path is required",
		},
		{
			name: "invalid command",
			config: &PathConfig{
				Path: "frontend/**",
				Commands: map[string]*CommandConfig{
					"lint": {},
				},
			},
			wantErr: true,
			errMsg:  "command \"lint\": command is required",
		},
		{
			name: "valid with extends",
			config: &PathConfig{
				Path:    "frontend/**",
				Extends: "base",
				Commands: map[string]*CommandConfig{
					"lint": {Command: "npm"},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("PathConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("PathConfig.Validate() error = %v, want error containing %q", err, tt.errMsg)
			}
		})
	}
}


func TestRegexPattern_Validate(t *testing.T) {
	tests := []struct {
		name    string
		pattern *RegexPattern
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid pattern",
			pattern: &RegexPattern{Pattern: "error"},
			wantErr: false,
		},
		{
			name:    "valid pattern with flags",
			pattern: &RegexPattern{Pattern: "error", Flags: "im"},
			wantErr: false,
		},
		{
			name:    "empty pattern",
			pattern: &RegexPattern{},
			wantErr: true,
			errMsg:  "pattern is required",
		},
		{
			name:    "invalid regex",
			pattern: &RegexPattern{Pattern: "[invalid"},
			wantErr: true,
			errMsg:  "invalid regex pattern",
		},
		{
			name:    "invalid flag",
			pattern: &RegexPattern{Pattern: "error", Flags: "x"},
			wantErr: true,
			errMsg:  "invalid regex",
		},
		{
			name:    "all valid flags",
			pattern: &RegexPattern{Pattern: "error", Flags: "imsU"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.pattern.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("RegexPattern.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("RegexPattern.Validate() error = %v, want error containing %q", err, tt.errMsg)
			}
		})
	}
}

func TestRegexPattern_Compile(t *testing.T) {
	tests := []struct {
		name    string
		pattern *RegexPattern
		wantErr bool
	}{
		{
			name:    "simple pattern",
			pattern: &RegexPattern{Pattern: "error"},
			wantErr: false,
		},
		{
			name:    "pattern with flags",
			pattern: &RegexPattern{Pattern: "ERROR", Flags: "i"},
			wantErr: false,
		},
		{
			name:    "invalid pattern",
			pattern: &RegexPattern{Pattern: "[invalid"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			re, err := tt.pattern.Compile()
			if (err != nil) != tt.wantErr {
				t.Errorf("RegexPattern.Compile() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && re == nil {
				t.Error("RegexPattern.Compile() returned nil regex")
			}
		})
	}
}

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			data: `{
				"version": "1.0",
				"commands": {
					"lint": {
						"command": "npm",
						"args": ["run", "lint"],
						"exitCodes": [1],
						"errorPatterns": [
							{"pattern": "error", "flags": "i"}
						],
						"maxOutput": 100
					}
				}
			}`,
			wantErr: false,
		},
		{
			name:    "invalid JSON",
			data:    `{"version": "1.0"`,
			wantErr: true,
			errMsg:  "failed to parse config",
		},
		{
			name:    "invalid config",
			data:    `{"commands": {}}`,
			wantErr: true,
			errMsg:  "invalid config: version is required",
		},
		{
			name: "config with paths",
			data: `{
				"version": "1.0",
				"projectType": "monorepo",
				"paths": [
					{
						"path": "frontend/**",
						"commands": {
							"lint": {
								"command": "npm",
								"args": ["run", "lint", "--prefix", "frontend"]
							}
						}
					}
				]
			}`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := LoadConfig([]byte(tt.data))
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("LoadConfig() error = %v, want error containing %q", err, tt.errMsg)
			}
			if !tt.wantErr && config == nil {
				t.Error("LoadConfig() returned nil config")
			}
		})
	}
}

func TestSaveConfig(t *testing.T) {
	tests := []struct {
		name      string
		buildFunc func() *Config
		wantErr   bool
	}{
		{
			name: "valid config",
			buildFunc: func() *Config {
				return newTestConfigBuilder().
					withSimpleCommand("lint", "npm", "run", "lint").
					build()
			},
			wantErr: false,
		},
		{
			name: "invalid config",
			buildFunc: func() *Config {
				return &Config{
					Commands: map[string]*CommandConfig{
						"lint": {Command: "npm"},
					},
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.buildFunc()
			data, err := SaveConfig(cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("SaveConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && len(data) == 0 {
				t.Error("SaveConfig() returned empty data")
			}

			// Verify the saved data can be loaded back
			if !tt.wantErr {
				loaded, err := LoadConfig(data)
				if err != nil {
					t.Errorf("Failed to load saved config: %v", err)
				}
				if loaded.Version != cfg.Version {
					t.Errorf("Loaded config version = %v, want %v", loaded.Version, cfg.Version)
				}
			}
		})
	}
}

func TestCommandConfig_Clone(t *testing.T) {
	original := &CommandConfig{
		Command: "npm",
		Args:    []string{"run", "lint"},
		ExitCodes: []int{1, 2},
		ErrorPatterns: []*RegexPattern{
			{Pattern: "error", Flags: "i"},
		},
		IncludePatterns: []*RegexPattern{
			{Pattern: "warning"},
		},
		ContextLines: 2,
		MaxOutput:    100,
		Prompt:  "Fix errors:",
		Timeout: 60000,
	}

	clone := original.Clone()

	// Verify all fields are copied
	if clone.Command != original.Command {
		t.Error("Command not cloned correctly")
	}
	if !reflect.DeepEqual(clone.Args, original.Args) {
		t.Error("Args not cloned correctly")
	}
	if clone.Prompt != original.Prompt {
		t.Error("Prompt not cloned correctly")
	}
	if clone.Timeout != original.Timeout {
		t.Error("Timeout not cloned correctly")
	}
	if clone.ContextLines != original.ContextLines {
		t.Error("ContextLines not cloned correctly")
	}
	if clone.MaxOutput != original.MaxOutput {
		t.Error("MaxOutput not cloned correctly")
	}

	// Verify deep copy - modifying clone should not affect original
	clone.Args[0] = "test"
	if original.Args[0] == "test" {
		t.Error("Args not deep copied")
	}

	clone.ExitCodes[0] = 99
	if original.ExitCodes[0] == 99 {
		t.Error("ExitCodes not deep copied")
	}

	clone.ErrorPatterns[0].Pattern = "modified"
	if original.ErrorPatterns[0].Pattern == "modified" {
		t.Error("ErrorPatterns not deep copied")
	}

	// Test nil clone
	var nilCmd *CommandConfig
	if nilCmd.Clone() != nil {
		t.Error("Clone of nil should return nil")
	}
}


func TestConfig_ComplexValidation(t *testing.T) {
	// Test a complex monorepo configuration
	config := newTestConfigBuilder().
		withVersion("1.0").
		withCommand("format", &CommandConfig{
			Command:       "prettier",
			Args:          []string{"--check", "."},
			ExitCodes:     []int{1},
			ErrorPatterns: []*RegexPattern{{Pattern: "\\[error\\]", Flags: "i"}},
			MaxOutput:     50,
			Prompt:        "Fix the formatting issues below:",
		}).
		withPath(&PathConfig{
			Path:    "frontend/**",
			Extends: "base",
			Commands: map[string]*CommandConfig{
				"lint": {
					Command:   "npm",
					Args:      []string{"run", "lint", "--prefix", "frontend"},
					ExitCodes: []int{1},
					ErrorPatterns: []*RegexPattern{
						{Pattern: "error", Flags: "i"},
						{Pattern: "^\\s*\\d+:\\d+", Flags: "m"},
					},
					ContextLines: 2,
					MaxOutput:    100,
				},
			},
		}).
		withPath(&PathConfig{
			Path: "backend/**",
			Commands: map[string]*CommandConfig{
				"lint": {
					Command:       "go",
					Args:          []string{"vet", "./..."},
					ExitCodes:     []int{1},
					ErrorPatterns: []*RegexPattern{{Pattern: ".*"}},
				},
			},
		}).
		build()
	
	// Set ProjectType after building
	config.ProjectType = "monorepo"

	err := config.Validate()
	if err != nil {
		t.Errorf("Complex config validation failed: %v", err)
	}

	// Test JSON roundtrip
	data, err := SaveConfig(config)
	if err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	loaded, err := LoadConfig(data)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify loaded config matches original
	if loaded.Version != config.Version {
		t.Error("Version mismatch after roundtrip")
	}
	if loaded.ProjectType != config.ProjectType {
		t.Error("ProjectType mismatch after roundtrip")
	}
	if len(loaded.Commands) != len(config.Commands) {
		t.Error("Commands count mismatch after roundtrip")
	}
	if len(loaded.Paths) != len(config.Paths) {
		t.Error("Paths count mismatch after roundtrip")
	}
}
