package security

import (
	"strings"
	"testing"
	"time"
)

func TestValidateCommand(t *testing.T) {
	v := NewSecurityValidator()

	tests := []struct {
		name    string
		command string
		args    []string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid command",
			command: "npm",
			args:    []string{"run", "test"},
			wantErr: false,
		},
		{
			name:    "empty command",
			command: "",
			args:    []string{},
			wantErr: true,
			errMsg:  "command cannot be empty",
		},
		{
			name:    "command with shell injection",
			command: "npm; rm -rf /",
			args:    []string{},
			wantErr: true,
			errMsg:  "potential shell injection",
		},
		{
			name:    "args with shell injection",
			command: "npm",
			args:    []string{"run", "test; echo hacked"},
			wantErr: true,
			errMsg:  "potential shell injection",
		},
		{
			name:    "command with pipe",
			command: "cat",
			args:    []string{"file.txt", "|", "grep", "secret"},
			wantErr: true,
			errMsg:  "potential shell injection",
		},
		{
			name:    "command with backticks",
			command: "echo",
			args:    []string{"`whoami`"},
			wantErr: true,
			errMsg:  "potential shell injection",
		},
		{
			name:    "command with dollar sign execution",
			command: "echo",
			args:    []string{"$(whoami)"},
			wantErr: true,
			errMsg:  "potential shell injection",
		},
		{
			name:    "dangerous rm command",
			command: "rm",
			args:    []string{"-rf", "/"},
			wantErr: true,
			errMsg:  "dangerous rm command",
		},
		{
			name:    "safe rm command",
			command: "rm",
			args:    []string{"temp.txt"},
			wantErr: false,
		},
		{
			name:    "encoded shell metacharacters",
			command: "echo",
			args:    []string{"test%3Bwhoami"},
			wantErr: true,
			errMsg:  "encoded shell metacharacters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateCommand(tt.command, tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCommand() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("ValidateCommand() error = %v, want error containing %q", err, tt.errMsg)
			}
		})
	}
}

func TestValidateCommandWithWhitelist(t *testing.T) {
	v := NewSecurityValidator()
	v.SetAllowedCommands([]string{"npm", "go", "python"})

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
			name:    "allowed command with path",
			command: "/usr/bin/npm",
			wantErr: false,
		},
		{
			name:    "disallowed command",
			command: "curl",
			wantErr: true,
		},
		{
			name:    "disallowed command with path",
			command: "/usr/bin/curl",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateCommand(tt.command, []string{})
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCommand() with whitelist error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidatePath(t *testing.T) {
	v := NewSecurityValidator()

	tests := []struct {
		name    string
		path    string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid relative path",
			path:    "src/main.go",
			wantErr: false,
		},
		{
			name:    "valid nested path",
			path:    "./internal/config/loader.go",
			wantErr: false,
		},
		{
			name:    "empty path",
			path:    "",
			wantErr: true,
			errMsg:  "path cannot be empty",
		},
		{
			name:    "path with null byte",
			path:    "file\x00.txt",
			wantErr: true,
			errMsg:  "null byte",
		},
		{
			name:    "path with directory traversal",
			path:    "../../../etc/passwd",
			wantErr: true,
			errMsg:  "directory traversal",
		},
		{
			name:    "path with encoded traversal",
			path:    "..%2F..%2Fetc%2Fpasswd",
			wantErr: true,
			errMsg:  "directory traversal",
		},
		{
			name:    "absolute path to system directory",
			path:    "/etc/passwd",
			wantErr: true,
			errMsg:  "forbidden",
		},
		{
			name:    "windows system path",
			path:    "C:\\Windows\\System32\\config",
			wantErr: true,
			errMsg:  "forbidden",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidatePath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePath() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("ValidatePath() error = %v, want error containing %q", err, tt.errMsg)
			}
		})
	}
}

func TestValidateRegexPattern(t *testing.T) {
	v := NewSecurityValidator()

	tests := []struct {
		name    string
		pattern string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid simple pattern",
			pattern: "error:\\s*(.+)",
			wantErr: false,
		},
		{
			name:    "valid complex pattern",
			pattern: "^\\s*at\\s+(.+?)\\s*\\((.+?):(\\d+):(\\d+)\\)",
			wantErr: false,
		},
		{
			name:    "ReDoS vulnerable pattern - nested quantifiers",
			pattern: "(.*)*",
			wantErr: true,
			errMsg:  "catastrophic backtracking",
		},
		{
			name:    "ReDoS vulnerable pattern - alternation",
			pattern: "(.+)+",
			wantErr: true,
			errMsg:  "catastrophic backtracking",
		},
		{
			name:    "ReDoS vulnerable pattern - complex",
			pattern: "(a+)+b",
			wantErr: true,
			errMsg:  "catastrophic backtracking",
		},
		{
			name:    "too many alternations",
			pattern: "a|b|c|d|e|f|g|h|i|j|k|l|m|n",
			wantErr: true,
			errMsg:  "too many alternations",
		},
		{
			name:    "too many capturing groups",
			pattern: "(a)(b)(c)(d)(e)(f)(g)(h)(i)(j)(k)(l)",
			wantErr: true,
			errMsg:  "too many capturing groups",
		},
		{
			name:    "invalid regex syntax",
			pattern: "[abc",
			wantErr: true,
			errMsg:  "invalid regex",
		},
		{
			name:    "pattern too long",
			pattern: strings.Repeat("a", 501),
			wantErr: true,
			errMsg:  "too long",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateRegexPattern(tt.pattern)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateRegexPattern() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("ValidateRegexPattern() error = %v, want error containing %q", err, tt.errMsg)
			}
		})
	}
}

func TestValidateTimeout(t *testing.T) {
	v := NewSecurityValidator()

	tests := []struct {
		name    string
		timeout time.Duration
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid timeout",
			timeout: 30 * time.Second,
			wantErr: false,
		},
		{
			name:    "zero timeout",
			timeout: 0,
			wantErr: false,
		},
		{
			name:    "negative timeout",
			timeout: -1 * time.Second,
			wantErr: true,
			errMsg:  "negative",
		},
		{
			name:    "too short timeout",
			timeout: 50 * time.Millisecond,
			wantErr: true,
			errMsg:  "too short",
		},
		{
			name:    "timeout exceeds maximum",
			timeout: 2 * time.Hour,
			wantErr: true,
			errMsg:  "exceeds maximum",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateTimeout(tt.timeout)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTimeout() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("ValidateTimeout() error = %v, want error containing %q", err, tt.errMsg)
			}
		})
	}
}

func TestValidateResourceLimits(t *testing.T) {
	v := NewSecurityValidator()

	tests := []struct {
		name        string
		outputSize  int64
		memoryLimit int64
		wantErr     bool
		errMsg      string
	}{
		{
			name:        "valid limits",
			outputSize:  1024 * 1024,       // 1MB
			memoryLimit: 256 * 1024 * 1024, // 256MB
			wantErr:     false,
		},
		{
			name:        "output size exceeds limit",
			outputSize:  20 * 1024 * 1024, // 20MB
			memoryLimit: 0,
			wantErr:     true,
			errMsg:      "output size",
		},
		{
			name:        "memory limit exceeds maximum",
			outputSize:  1024,
			memoryLimit: 2 * 1024 * 1024 * 1024, // 2GB
			wantErr:     true,
			errMsg:      "memory limit",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateResourceLimits(tt.outputSize, tt.memoryLimit)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateResourceLimits() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("ValidateResourceLimits() error = %v, want error containing %q", err, tt.errMsg)
			}
		})
	}
}

func TestCheckReDoSPattern(t *testing.T) {
	v := NewSecurityValidator()

	dangerousPatterns := []string{
		"(a*)*",
		"(a+)+",
		"(a*)+",
		"(a+)*",
		"(.*)*",
		"(.+)+",
		"(.*)+",
		"(.+)*",
		"([a-z]+)*",
		"([a-z]*)+",
		"(\\d+)+",
		"(\\w*)*",
		"((a))+",
		"(a|a)*",
		"(a|ab)*",
	}

	for _, pattern := range dangerousPatterns {
		t.Run(pattern, func(t *testing.T) {
			err := v.checkReDoSPattern(pattern)
			if err == nil {
				t.Errorf("checkReDoSPattern(%q) expected error, got nil", pattern)
			}
		})
	}

	safePatterns := []string{
		"a*",
		"a+",
		"(a)",
		"a|b",
		"[a-z]+",
		"\\d+",
		"\\w*",
		"^test$",
		"error: (.+)",
	}

	for _, pattern := range safePatterns {
		t.Run(pattern, func(t *testing.T) {
			err := v.checkReDoSPattern(pattern)
			if err != nil {
				t.Errorf("checkReDoSPattern(%q) unexpected error: %v", pattern, err)
			}
		})
	}
}
