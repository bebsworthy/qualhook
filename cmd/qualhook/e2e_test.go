package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bebsworthy/qualhook/pkg/config"
)

// TestE2EFormatCommand tests the format command end-to-end
func TestE2EFormatCommand(t *testing.T) {
	tempDir := t.TempDir()

	// Create config that will produce formatting errors
	cfg := &config.Config{
		Version: "1.0",
		Commands: map[string]*config.CommandConfig{
			"format": {
				Command: "echo",
				Args:    []string{"Error: File not formatted: main.js"},
				ExitCodes: []int{0}, // echo returns 0, but we'll match on pattern
				ErrorPatterns: []*config.RegexPattern{
					{Pattern: "Error:", Flags: ""},
				},
				MaxOutput: 100,
				Prompt: "Fix the formatting issues below:",
			},
		},
	}

	configData, _ := json.Marshal(cfg)
	configFile := filepath.Join(tempDir, ".qualhook.json")
	os.WriteFile(configFile, configData, 0644)

	oldDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldDir)

	// Replace os.Exit temporarily
	oldExit := osExit
	exitCode := -1
	osExit = func(code int) {
		exitCode = code
	}
	defer func() { osExit = oldExit }()

	// Capture stderr using errorWriter
	oldStderr := os.Stderr
	oldErrorWriter := errorWriter
	r, w, _ := os.Pipe()
	os.Stderr = w
	errorWriter = w

	cmd := newRootCmd()
	cmd.SetArgs([]string{"format"})
	cmd.Execute()

	w.Close()
	os.Stderr = oldStderr
	errorWriter = oldErrorWriter

	// Read captured output
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Verify exit code 2 for errors
	if exitCode != 2 {
		t.Errorf("Expected exit code 2, got %d", exitCode)
	}

	// Verify error output contains prompt and error
	if !strings.Contains(output, "Fix the formatting issues below:") {
		t.Errorf("Expected prompt in output, got: %q", output)
	}
	if !strings.Contains(output, "Error: File not formatted: main.js") {
		t.Errorf("Expected error message in output, got: %q", output)
	}
}

// TestE2ESuccessfulCommand tests successful command execution
func TestE2ESuccessfulCommand(t *testing.T) {
	tempDir := t.TempDir()

	cfg := &config.Config{
		Version: "1.0",
		Commands: map[string]*config.CommandConfig{
			"lint": {
				Command: "echo",
				Args:    []string{"No linting issues found"},
				ExitCodes: []int{1},
				ErrorPatterns: []*config.RegexPattern{
					{Pattern: "error", Flags: "i"},
				},
				MaxOutput: 100,
			},
		},
	}

	configData, _ := json.Marshal(cfg)
	configFile := filepath.Join(tempDir, ".qualhook.json")
	os.WriteFile(configFile, configData, 0644)

	oldDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldDir)

	// Capture stdout using outputWriter
	oldStdout := os.Stdout
	oldOutputWriter := outputWriter
	r, w, _ := os.Pipe()
	os.Stdout = w
	outputWriter = w

	cmd := newRootCmd()
	cmd.SetArgs([]string{"lint"})
	err := cmd.Execute()

	w.Close()
	os.Stdout = oldStdout
	outputWriter = oldOutputWriter

	if err != nil {
		t.Errorf("Command failed: %v", err)
	}

	// Read captured output
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Should get success message
	if !strings.Contains(output, "All quality checks passed successfully") {
		t.Errorf("Expected success message, got: %q", output)
	}
}

// TestE2EMonorepoExecution tests monorepo path-based configuration
func TestE2EMonorepoExecution(t *testing.T) {
	tempDir := t.TempDir()

	// Create monorepo structure
	frontendDir := filepath.Join(tempDir, "frontend")
	backendDir := filepath.Join(tempDir, "backend")
	os.MkdirAll(frontendDir, 0755)
	os.MkdirAll(backendDir, 0755)

	// Create monorepo config
	cfg := &config.Config{
		Version: "1.0",
		Commands: map[string]*config.CommandConfig{
			"lint": {
				Command: "echo",
				Args:    []string{"Root lint"},
				ExitCodes: []int{1},
				ErrorPatterns: []*config.RegexPattern{
					{Pattern: "error", Flags: "i"},
				},
				MaxOutput: 100,
			},
		},
		Paths: []*config.PathConfig{
			{
				Path: "frontend/**",
				Commands: map[string]*config.CommandConfig{
					"lint": {
						Command: "echo",
						Args:    []string{"Frontend lint"},
						ExitCodes: []int{1},
						ErrorPatterns: []*config.RegexPattern{
							{Pattern: "error", Flags: "i"},
						},
						MaxOutput: 100,
					},
				},
			},
			{
				Path: "backend/**",
				Commands: map[string]*config.CommandConfig{
					"lint": {
						Command: "echo",
						Args:    []string{"Backend lint"},
						ExitCodes: []int{1},
						ErrorPatterns: []*config.RegexPattern{
							{Pattern: "error", Flags: "i"},
						},
						MaxOutput: 100,
					},
				},
			},
		},
	}

	configData, _ := json.Marshal(cfg)
	configFile := filepath.Join(tempDir, ".qualhook.json")
	os.WriteFile(configFile, configData, 0644)

	// Test from frontend directory
	t.Run("frontend path", func(t *testing.T) {
		oldDir, _ := os.Getwd()
		os.Chdir(frontendDir)
		defer os.Chdir(oldDir)

		// Capture stdout using outputWriter
		oldStdout := os.Stdout
		oldOutputWriter := outputWriter
		r, w, _ := os.Pipe()
		os.Stdout = w
		outputWriter = w

		cmd := newRootCmd()
		cmd.SetArgs([]string{"--config", configFile, "lint"})
		cmd.Execute()

		w.Close()
		os.Stdout = oldStdout
		outputWriter = oldOutputWriter

		var buf bytes.Buffer
		buf.ReadFrom(r)
		output := buf.String()

		// Should execute frontend-specific command
		if !strings.Contains(output, "All quality checks passed successfully") {
			t.Errorf("Expected success for frontend lint, got: %q", output)
		}
	})
}

// TestE2ECustomCommand tests custom command execution
func TestE2ECustomCommand(t *testing.T) {
	tempDir := t.TempDir()

	cfg := &config.Config{
		Version: "1.0",
		Commands: map[string]*config.CommandConfig{
			"custom-check": {
				Command: "echo",
				Args:    []string{"Running custom check"},
				ExitCodes: []int{1},
				ErrorPatterns: []*config.RegexPattern{
					{Pattern: "fail", Flags: "i"},
				},
				MaxOutput: 100,
			},
		},
	}

	configData, _ := json.Marshal(cfg)
	configFile := filepath.Join(tempDir, ".qualhook.json")
	os.WriteFile(configFile, configData, 0644)

	oldDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldDir)

	// Capture stdout using outputWriter
	oldStdout := os.Stdout
	oldOutputWriter := outputWriter
	r, w, _ := os.Pipe()
	os.Stdout = w
	outputWriter = w

	// Set global config path for tryCustomCommand
	oldConfigPath := configPath
	configPath = configFile
	defer func() { configPath = oldConfigPath }()

	// Execute custom command directly (not through cobra subcommand)
	err := tryCustomCommand("custom-check", []string{})

	w.Close()
	os.Stdout = oldStdout
	outputWriter = oldOutputWriter

	if err != nil {
		t.Errorf("Custom command failed: %v", err)
	}

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if !strings.Contains(output, "All quality checks passed successfully") {
		t.Errorf("Expected success message for custom command, got: %q", output)
	}
}
