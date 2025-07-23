package config

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/bebsworthy/qualhook/pkg/config"
)

func TestValidator_Validate(t *testing.T) {
	validator := NewValidator()
	// Don't check command existence in tests
	validator.CheckCommands = false

	tests := []struct {
		name    string
		config  *config.Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: &config.Config{
				Version: "1.0",
				Commands: map[string]*config.CommandConfig{
					"lint": {
						Command: "npm",
						Args:    []string{"run", "lint"},
						ErrorDetection: &config.ErrorDetection{
							ExitCodes: []int{1},
							Patterns: []*config.RegexPattern{
								{Pattern: "error:", Flags: "i"},
							},
						},
						OutputFilter: &config.FilterConfig{
							ErrorPatterns: []*config.RegexPattern{
								{Pattern: "\\d+:\\d+\\s+error", Flags: ""},
							},
							MaxOutput: 100,
						},
						Timeout: 5000,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "timeout too short",
			config: &config.Config{
				Version: "1.0",
				Commands: map[string]*config.CommandConfig{
					"test": {
						Command: "npm",
						Args:    []string{"test"},
						Timeout: 50, // Too short
						OutputFilter: &config.FilterConfig{
							ErrorPatterns: []*config.RegexPattern{
								{Pattern: "fail", Flags: "i"},
							},
						},
					},
				},
			},
			wantErr: true,
			errMsg:  "timeout 50ms is too short",
		},
		{
			name: "timeout too long",
			config: &config.Config{
				Version: "1.0",
				Commands: map[string]*config.CommandConfig{
					"test": {
						Command: "npm",
						Args:    []string{"test"},
						Timeout: 3700000, // More than 1 hour
						OutputFilter: &config.FilterConfig{
							ErrorPatterns: []*config.RegexPattern{
								{Pattern: "fail", Flags: "i"},
							},
						},
					},
				},
			},
			wantErr: true,
			errMsg:  "exceeds maximum allowed",
		},
		{
			name: "dangerous regex pattern",
			config: &config.Config{
				Version: "1.0",
				Commands: map[string]*config.CommandConfig{
					"lint": {
						Command: "eslint",
						ErrorDetection: &config.ErrorDetection{
							Patterns: []*config.RegexPattern{
								{Pattern: "(.*)*", Flags: ""}, // Catastrophic backtracking
							},
						},
						OutputFilter: &config.FilterConfig{
							ErrorPatterns: []*config.RegexPattern{
								{Pattern: "error", Flags: "i"},
							},
						},
					},
				},
			},
			wantErr: true,
			errMsg:  "catastrophic backtracking",
		},
		{
			name: "invalid path pattern",
			config: &config.Config{
				Version: "1.0",
				Commands: map[string]*config.CommandConfig{
					"build": {
						Command: "make",
						OutputFilter: &config.FilterConfig{
							ErrorPatterns: []*config.RegexPattern{
								{Pattern: "error", Flags: "i"},
							},
						},
					},
				},
				Paths: []*config.PathConfig{
					{
						Path: "../outside", // Parent directory reference
						Commands: map[string]*config.CommandConfig{
							"build": {
								Command: "make",
								Args:    []string{"all"},
								OutputFilter: &config.FilterConfig{
									ErrorPatterns: []*config.RegexPattern{
										{Pattern: "error", Flags: "i"},
									},
								},
							},
						},
					},
				},
			},
			wantErr: true,
			errMsg:  "directory traversal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("Expected error containing %q, got %q", tt.errMsg, err.Error())
			}
		})
	}
}

func TestValidator_ValidateCommand(t *testing.T) {
	validator := NewValidator()
	validator.CheckCommands = false

	tests := []struct {
		name    string
		command *config.CommandConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid command",
			command: &config.CommandConfig{
				Command: "npm",
				Args:    []string{"run", "test"},
				OutputFilter: &config.FilterConfig{
					ErrorPatterns: []*config.RegexPattern{
						{Pattern: "FAIL", Flags: ""},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "command with allowed list",
			command: &config.CommandConfig{
				Command: "rm", // Dangerous command
				Args:    []string{"-rf", "/"},
				OutputFilter: &config.FilterConfig{
					ErrorPatterns: []*config.RegexPattern{
						{Pattern: "error", Flags: "i"},
					},
				},
			},
			wantErr: true,
			errMsg:  "dangerous rm command",
		},
	}

	// Test with allowed commands list
	validatorWithAllowList := NewValidator()
	validatorWithAllowList.CheckCommands = false
	validatorWithAllowList.AllowedCommands = []string{"npm", "go", "cargo", "python"}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := validator
			if tt.errMsg == "not in allowed list" {
				v = validatorWithAllowList
			}

			err := v.ValidateCommand(tt.command)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCommand() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("Expected error containing %q, got %q", tt.errMsg, err.Error())
			}
		})
	}
}

func TestValidator_CheckDangerousRegex(t *testing.T) {
	validator := NewValidator()

	tests := []struct {
		pattern     string
		wantErr     bool
		description string
	}{
		{"^error:", false, "simple pattern"},
		{"\\d+:\\d+", false, "line:column pattern"},
		{"(.*)*", true, "catastrophic backtracking"},
		{"(a+)+", true, "nested quantifiers"},
		{"(.+)*", true, "greedy nested quantifiers"},
		{"(\\s*)*", true, "whitespace catastrophic backtracking"},
		{strings.Repeat("a", 600), true, "pattern too long"},
		{"(" + strings.Repeat("(", 15) + ")", true, "too many capture groups"},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			err := validator.checkDangerousRegex(tt.pattern)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkDangerousRegex(%q) error = %v, wantErr %v", tt.pattern, err, tt.wantErr)
			}
		})
	}
}

func TestValidator_IsTooGenericPattern(t *testing.T) {
	validator := NewValidator()

	tests := []struct {
		pattern    string
		tooGeneric bool
	}{
		{".*", true},
		{".+", true},
		{"^.*$", true},
		{"^.+$", true},
		{"\\w+", true},
		{"\\s+", true},
		{"error", false},
		{"^\\s*at\\s+", false},
		{"\\d+:\\d+", false},
		{"ERROR|WARN|FAIL", false},
	}

	for _, tt := range tests {
		t.Run(tt.pattern, func(t *testing.T) {
			re, err := regexp.Compile(tt.pattern)
			if err != nil {
				t.Fatalf("Failed to compile pattern %q: %v", tt.pattern, err)
			}

			result := validator.isTooGenericPattern(re, tt.pattern)
			if result != tt.tooGeneric {
				t.Errorf("isTooGenericPattern(%q) = %v, want %v", tt.pattern, result, tt.tooGeneric)
			}
		})
	}
}

func TestValidator_ValidatePathPattern(t *testing.T) {
	validator := NewValidator()

	tests := []struct {
		pattern string
		wantErr bool
		errMsg  string
	}{
		{"frontend/**", false, ""},
		{"src/**/*.js", false, ""},
		{"packages/*/src", false, ""},
		{"", true, "cannot be empty"},
		{"/absolute/path", true, "absolute paths are not allowed"},
		{"../parent", true, "directory traversal"},
		{"path\x00null", true, "null byte"},
		{"[invalid", true, "invalid glob pattern"},
	}

	for _, tt := range tests {
		t.Run(tt.pattern, func(t *testing.T) {
			err := validator.validatePathPattern(tt.pattern)
			if (err != nil) != tt.wantErr {
				t.Errorf("validatePathPattern(%q) error = %v, wantErr %v", tt.pattern, err, tt.wantErr)
			}
			if err != nil && tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("Expected error containing %q, got %q", tt.errMsg, err.Error())
			}
		})
	}
}

func TestValidator_SuggestFixes(t *testing.T) {
	validator := NewValidator()

	tests := []struct {
		errMsg      string
		wantSuggest []string
	}{
		{
			errMsg: "command \"npm\" not found in PATH",
			wantSuggest: []string{
				"Make sure the command is installed",
				"Install Node.js",
			},
		},
		{
			errMsg: "invalid regex pattern",
			wantSuggest: []string{
				"Check your regex pattern syntax",
				"Test your pattern at",
			},
		},
		{
			errMsg: "timeout 50ms is too short",
			wantSuggest: []string{
				"Use a timeout between 100ms and 3600000ms",
			},
		},
		{
			errMsg: "invalid path pattern",
			wantSuggest: []string{
				"Use relative paths only",
				"Use ** for recursive matching",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.errMsg, func(t *testing.T) {
			suggestions := validator.SuggestFixes(fmt.Errorf("%s", tt.errMsg))
			
			for _, want := range tt.wantSuggest {
				found := false
				for _, got := range suggestions {
					if strings.Contains(got, want) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected suggestion containing %q, got %v", want, suggestions)
				}
			}
		})
	}
}

func TestValidator_IsCommandAllowed(t *testing.T) {
	validator := &Validator{
		AllowedCommands: []string{"npm", "go", "python", "cargo"},
	}

	tests := []struct {
		command string
		allowed bool
	}{
		{"npm", true},
		{"go", true},
		{"/usr/local/bin/npm", true}, // Should match base name
		{"rm", false},
		{"curl", false},
		{"./custom-script", false},
	}

	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			result := validator.isCommandAllowed(tt.command)
			if result != tt.allowed {
				t.Errorf("isCommandAllowed(%q) = %v, want %v", tt.command, result, tt.allowed)
			}
		})
	}
}