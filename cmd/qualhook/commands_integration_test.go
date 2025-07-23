package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/qualhook/qualhook/pkg/config"
)

// TestCommandsIntegration tests the actual command execution
// These tests use echo commands which bypass the error reporter
// so we get the actual command output instead of "All quality checks passed"
func TestCommandsIntegration(t *testing.T) {
	// Create a temporary directory for test
	tempDir := t.TempDir()
	
	// Create a test configuration
	cfg := &config.Config{
		Version: "1.0",
		Commands: map[string]*config.CommandConfig{
			"format": {
				Command: "echo",
				Args:    []string{"Formatting complete"},
				ErrorDetection: &config.ErrorDetection{
					ExitCodes: []int{1},
				},
				OutputFilter: &config.FilterConfig{
					ErrorPatterns: []*config.RegexPattern{
						{Pattern: "error", Flags: "i"},
					},
				},
			},
			"lint": {
				Command: "echo", 
				Args:    []string{"No linting issues found"},
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
	
	configData, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}
	
	configFile := filepath.Join(tempDir, ".qualhook.json")
	if err := os.WriteFile(configFile, configData, 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}
	
	// Change to temp directory
	oldDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldDir)
	
	// Test format command
	t.Run("format command", func(t *testing.T) {
		cmd := newRootCmd()
		cmd.SetArgs([]string{"format"})
		
		// Capture output using a pipe
		oldStdout := os.Stdout
		oldOutputWriter := outputWriter
		r, w, _ := os.Pipe()
		os.Stdout = w
		outputWriter = w
		
		err := cmd.Execute()
		
		w.Close()
		os.Stdout = oldStdout
		outputWriter = oldOutputWriter
		
		var buf bytes.Buffer
		buf.ReadFrom(r)
		output := buf.String()
		
		if err != nil {
			t.Errorf("Format command failed: %v", err)
		}
		
		if !strings.Contains(output, "All quality checks passed successfully") {
			t.Errorf("Expected success message, got: %q", output)
		}
	})
	
	// Test lint command
	t.Run("lint command", func(t *testing.T) {
		cmd := newRootCmd()
		cmd.SetArgs([]string{"lint"})
		
		// Capture output using a pipe
		oldStdout := os.Stdout
		oldOutputWriter := outputWriter
		r, w, _ := os.Pipe()
		os.Stdout = w
		outputWriter = w
		
		err := cmd.Execute()
		
		w.Close()
		os.Stdout = oldStdout
		outputWriter = oldOutputWriter
		
		var buf bytes.Buffer
		buf.ReadFrom(r)
		output := buf.String()
		
		if err != nil {
			t.Errorf("Lint command failed: %v", err)
		}
		
		if !strings.Contains(output, "All quality checks passed successfully") {
			t.Errorf("Expected success message, got: %q", output)
		}
	})
}

// TestCommandWithErrors tests error reporting
func TestCommandWithErrors(t *testing.T) {
	// Create a temporary directory for test
	tempDir := t.TempDir()
	
	// Create a test configuration that will produce errors
	cfg := &config.Config{
		Version: "1.0",
		Commands: map[string]*config.CommandConfig{
			"lint": {
				Command: "echo",
				Args:    []string{"error: lint failed at line 10"},
				ErrorDetection: &config.ErrorDetection{
					ExitCodes: []int{0}, // echo returns 0, but we'll match on pattern
					Patterns: []*config.RegexPattern{
						{Pattern: "error:", Flags: ""},
					},
				},
				OutputFilter: &config.FilterConfig{
					ErrorPatterns: []*config.RegexPattern{
						{Pattern: "error:", Flags: ""},
					},
					MaxOutput: 100,
				},
				Prompt: "Fix the linting errors below:",
			},
		},
	}
	
	configData, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}
	
	configFile := filepath.Join(tempDir, ".qualhook.json")
	if err := os.WriteFile(configFile, configData, 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}
	
	// Change to temp directory
	oldDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldDir)
	
	// Test lint command with errors
	t.Run("lint with errors", func(t *testing.T) {
		cmd := newRootCmd()
		cmd.SetArgs([]string{"lint"})
		
		// Capture stderr using errorWriter
		oldStderr := os.Stderr
		oldErrorWriter := errorWriter
		r, w, _ := os.Pipe()
		os.Stderr = w
		errorWriter = w
		
		// We expect this to exit with code 2, so we need to handle the exit
		// by replacing os.Exit temporarily
		oldExit := osExit
		exitCode := 0
		osExit = func(code int) {
			exitCode = code
		}
		defer func() { osExit = oldExit }()
		
		err := cmd.Execute()
		
		w.Close()
		os.Stderr = oldStderr
		errorWriter = oldErrorWriter
		
		var buf bytes.Buffer
		buf.ReadFrom(r)
		output := buf.String()
		
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		
		if exitCode != 2 {
			t.Errorf("Expected exit code 2, got %d", exitCode)
		}
		
		if !strings.Contains(output, "Fix the linting errors below:") {
			t.Errorf("Expected error prompt, got: %q", output)
		}
		
		if !strings.Contains(output, "error: lint failed at line 10") {
			t.Errorf("Expected error message, got: %q", output)
		}
	})
}

// Use the osExit from execute.go