package config

import (
	"reflect"
	"strings"
	"testing"
)

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config with commands",
			config: &Config{
				Version: "1.0",
				Commands: map[string]*CommandConfig{
					"lint": {
						Command: "npm",
						Args:    []string{"run", "lint"},
						ErrorDetection: &ErrorDetection{
							ExitCodes: []int{1},
						},
						OutputFilter: &FilterConfig{
							ErrorPatterns: []*RegexPattern{
								{Pattern: "error"},
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "missing version",
			config: &Config{
				Commands: map[string]*CommandConfig{
					"lint": {Command: "npm"},
				},
			},
			wantErr: true,
			errMsg:  "version is required",
		},
		{
			name: "no commands or paths",
			config: &Config{
				Version: "1.0",
			},
			wantErr: true,
			errMsg:  "at least one command or path configuration is required",
		},
		{
			name: "invalid command config",
			config: &Config{
				Version: "1.0",
				Commands: map[string]*CommandConfig{
					"lint": {},
				},
			},
			wantErr: true,
			errMsg:  "command \"lint\": command is required",
		},
		{
			name: "valid config with paths",
			config: &Config{
				Version: "1.0",
				Paths: []*PathConfig{
					{
						Path: "frontend/**",
						Commands: map[string]*CommandConfig{
							"lint": {Command: "npm"},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid path config",
			config: &Config{
				Version: "1.0",
				Paths: []*PathConfig{
					{
						Commands: map[string]*CommandConfig{
							"lint": {Command: "npm"},
						},
					},
				},
			},
			wantErr: true,
			errMsg:  "path config 0: path is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
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
			name: "invalid error detection",
			config: &CommandConfig{
				Command: "npm",
				ErrorDetection: &ErrorDetection{
					Patterns: []*RegexPattern{
						{Pattern: "[invalid"},
					},
				},
			},
			wantErr: true,
			errMsg:  "error detection",
		},
		{
			name: "invalid output filter",
			config: &CommandConfig{
				Command: "npm",
				OutputFilter: &FilterConfig{
					ErrorPatterns: []*RegexPattern{},
				},
			},
			wantErr: true,
			errMsg:  "output filter",
		},
		{
			name: "valid with all fields",
			config: &CommandConfig{
				Command: "npm",
				Args:    []string{"run", "lint"},
				ErrorDetection: &ErrorDetection{
					ExitCodes: []int{1},
					Patterns: []*RegexPattern{
						{Pattern: "error"},
					},
				},
				OutputFilter: &FilterConfig{
					ErrorPatterns: []*RegexPattern{
						{Pattern: "error", Flags: "i"},
					},
					ContextLines: 2,
					MaxOutput:    100,
				},
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

func TestErrorDetection_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *ErrorDetection
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid with exit codes",
			config: &ErrorDetection{
				ExitCodes: []int{1, 2},
			},
			wantErr: false,
		},
		{
			name: "valid with patterns",
			config: &ErrorDetection{
				Patterns: []*RegexPattern{
					{Pattern: "error"},
					{Pattern: "fail", Flags: "i"},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid pattern",
			config: &ErrorDetection{
				Patterns: []*RegexPattern{
					{Pattern: "[invalid"},
				},
			},
			wantErr: true,
			errMsg:  "pattern 0: invalid regex pattern",
		},
		{
			name:    "empty is valid",
			config:  &ErrorDetection{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("ErrorDetection.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("ErrorDetection.Validate() error = %v, want error containing %q", err, tt.errMsg)
			}
		})
	}
}

func TestFilterConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *FilterConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid filter config",
			config: &FilterConfig{
				ErrorPatterns: []*RegexPattern{
					{Pattern: "error"},
				},
				ContextLines: 2,
				MaxOutput:    100,
			},
			wantErr: false,
		},
		{
			name: "no error patterns",
			config: &FilterConfig{
				ErrorPatterns: []*RegexPattern{},
			},
			wantErr: true,
			errMsg:  "at least one error pattern is required",
		},
		{
			name: "invalid error pattern",
			config: &FilterConfig{
				ErrorPatterns: []*RegexPattern{
					{Pattern: "[invalid"},
				},
			},
			wantErr: true,
			errMsg:  "error pattern 0",
		},
		{
			name: "invalid include pattern",
			config: &FilterConfig{
				ErrorPatterns: []*RegexPattern{
					{Pattern: "error"},
				},
				IncludePatterns: []*RegexPattern{
					{Pattern: "[invalid"},
				},
			},
			wantErr: true,
			errMsg:  "include pattern 0",
		},
		{
			name: "negative context lines",
			config: &FilterConfig{
				ErrorPatterns: []*RegexPattern{
					{Pattern: "error"},
				},
				ContextLines: -1,
			},
			wantErr: true,
			errMsg:  "context lines must be non-negative",
		},
		{
			name: "negative max output",
			config: &FilterConfig{
				ErrorPatterns: []*RegexPattern{
					{Pattern: "error"},
				},
				MaxOutput: -1,
			},
			wantErr: true,
			errMsg:  "max output must be non-negative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("FilterConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("FilterConfig.Validate() error = %v, want error containing %q", err, tt.errMsg)
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
						"errorDetection": {
							"exitCodes": [1]
						},
						"outputFilter": {
							"errorPatterns": [
								{"pattern": "error", "flags": "i"}
							],
							"maxOutput": 100
						}
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
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &Config{
				Version: "1.0",
				Commands: map[string]*CommandConfig{
					"lint": {
						Command: "npm",
						Args:    []string{"run", "lint"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid config",
			config: &Config{
				Commands: map[string]*CommandConfig{
					"lint": {Command: "npm"},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := SaveConfig(tt.config)
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
				if loaded.Version != tt.config.Version {
					t.Errorf("Loaded config version = %v, want %v", loaded.Version, tt.config.Version)
				}
			}
		})
	}
}

func TestCommandConfig_Clone(t *testing.T) {
	original := &CommandConfig{
		Command: "npm",
		Args:    []string{"run", "lint"},
		ErrorDetection: &ErrorDetection{
			ExitCodes: []int{1, 2},
			Patterns: []*RegexPattern{
				{Pattern: "error", Flags: "i"},
			},
		},
		OutputFilter: &FilterConfig{
			ErrorPatterns: []*RegexPattern{
				{Pattern: "error"},
			},
			ContextLines: 2,
			MaxOutput:    100,
		},
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

	// Verify deep copy - modifying clone should not affect original
	clone.Args[0] = "test"
	if original.Args[0] == "test" {
		t.Error("Args not deep copied")
	}

	clone.ErrorDetection.ExitCodes[0] = 99
	if original.ErrorDetection.ExitCodes[0] == 99 {
		t.Error("ErrorDetection not deep copied")
	}

	// Test nil clone
	var nilCmd *CommandConfig
	if nilCmd.Clone() != nil {
		t.Error("Clone of nil should return nil")
	}
}

func TestErrorDetection_Clone(t *testing.T) {
	original := &ErrorDetection{
		ExitCodes: []int{1, 2, 3},
		Patterns: []*RegexPattern{
			{Pattern: "error", Flags: "i"},
			{Pattern: "fail"},
		},
	}

	clone := original.Clone()

	// Verify deep copy
	if !reflect.DeepEqual(clone.ExitCodes, original.ExitCodes) {
		t.Error("ExitCodes not cloned correctly")
	}
	if len(clone.Patterns) != len(original.Patterns) {
		t.Error("Patterns not cloned correctly")
	}

	// Modify clone and verify original is unchanged
	clone.ExitCodes[0] = 99
	if original.ExitCodes[0] == 99 {
		t.Error("ExitCodes not deep copied")
	}

	clone.Patterns[0].Pattern = testModifiedValue
	if original.Patterns[0].Pattern == "modified" {
		t.Error("Patterns not deep copied")
	}

	// Test nil clone
	var nilED *ErrorDetection
	if nilED.Clone() != nil {
		t.Error("Clone of nil should return nil")
	}
}

func TestFilterConfig_Clone(t *testing.T) {
	original := &FilterConfig{
		ErrorPatterns: []*RegexPattern{
			{Pattern: "error", Flags: "i"},
			{Pattern: "fail"},
		},
		IncludePatterns: []*RegexPattern{
			{Pattern: "include"},
		},
		ContextLines: 5,
		MaxOutput:    200,
	}

	clone := original.Clone()

	// Verify all fields are copied
	if clone.ContextLines != original.ContextLines {
		t.Error("ContextLines not cloned correctly")
	}
	if clone.MaxOutput != original.MaxOutput {
		t.Error("MaxOutput not cloned correctly")
	}
	if len(clone.ErrorPatterns) != len(original.ErrorPatterns) {
		t.Error("ErrorPatterns not cloned correctly")
	}
	if len(clone.IncludePatterns) != len(original.IncludePatterns) {
		t.Error("IncludePatterns not cloned correctly")
	}

	// Verify deep copy
	clone.ErrorPatterns[0].Pattern = "modified"
	if original.ErrorPatterns[0].Pattern == "modified" {
		t.Error("ErrorPatterns not deep copied")
	}

	// Test nil clone
	var nilFC *FilterConfig
	if nilFC.Clone() != nil {
		t.Error("Clone of nil should return nil")
	}
}

func TestConfig_ComplexValidation(t *testing.T) {
	// Test a complex monorepo configuration
	config := &Config{
		Version:     "1.0",
		ProjectType: "monorepo",
		Commands: map[string]*CommandConfig{
			"format": {
				Command: "prettier",
				Args:    []string{"--check", "."},
				ErrorDetection: &ErrorDetection{
					ExitCodes: []int{1},
				},
				OutputFilter: &FilterConfig{
					ErrorPatterns: []*RegexPattern{
						{Pattern: "\\[error\\]", Flags: "i"},
					},
					MaxOutput: 50,
				},
				Prompt: "Fix the formatting issues below:",
			},
		},
		Paths: []*PathConfig{
			{
				Path:    "frontend/**",
				Extends: "base",
				Commands: map[string]*CommandConfig{
					"lint": {
						Command: "npm",
						Args:    []string{"run", "lint", "--prefix", "frontend"},
						ErrorDetection: &ErrorDetection{
							ExitCodes: []int{1},
							Patterns: []*RegexPattern{
								{Pattern: "\\d+ errors?"},
							},
						},
						OutputFilter: &FilterConfig{
							ErrorPatterns: []*RegexPattern{
								{Pattern: "error", Flags: "i"},
								{Pattern: "^\\s*\\d+:\\d+", Flags: "m"},
							},
							ContextLines: 2,
							MaxOutput:    100,
						},
					},
				},
			},
			{
				Path: "backend/**",
				Commands: map[string]*CommandConfig{
					"lint": {
						Command: "go",
						Args:    []string{"vet", "./..."},
						ErrorDetection: &ErrorDetection{
							ExitCodes: []int{1},
						},
						OutputFilter: &FilterConfig{
							ErrorPatterns: []*RegexPattern{
								{Pattern: ".*"},
							},
						},
					},
				},
			},
		},
	}

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