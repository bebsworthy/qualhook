// +build integration

package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bebsworthy/qualhook/internal/debug"
	"github.com/bebsworthy/qualhook/internal/hook"
	"github.com/bebsworthy/qualhook/pkg/config"
)

// TestIntegration_FormatCommand tests the format command end-to-end
func TestIntegration_FormatCommand(t *testing.T) {
	// Create a test project directory
	tempDir := t.TempDir()
	
	// Create a simple configuration
	cfg := &config.Config{
		Version: "1.0",
		Commands: map[string]*config.CommandConfig{
			"format": {
				Command: "echo",
				Args:    []string{"Formatting completed"},
				ErrorDetection: &config.ErrorDetection{
					ExitCodes: []int{1},
				},
				OutputFilter: &config.FilterConfig{
					ErrorPatterns: []*config.RegexPattern{
						{Pattern: "error"},
					},
				},
			},
		},
	}
	
	// Write configuration
	configPath := filepath.Join(tempDir, ".qualhook.json")
	configData, err := config.SaveConfig(cfg)
	if err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}
	if err := os.WriteFile(configPath, configData, 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}
	
	// Change to test directory
	oldDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldDir)
	
	// Execute format command
	rootCmd := newRootCmd()
	rootCmd.SetArgs([]string{"format"})
	
	// Capture output
	var stdout, stderr bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)
	
	// Also redirect the global output writers used by executeCommand
	oldOutputWriter := outputWriter
	oldErrorWriter := errorWriter
	outputWriter = &stdout
	errorWriter = &stderr
	defer func() {
		outputWriter = oldOutputWriter
		errorWriter = oldErrorWriter
	}()
	
	err = rootCmd.Execute()
	if err != nil {
		t.Errorf("Command failed: %v", err)
	}
	
	// Verify output - the error reporter returns success message when no errors
	stdoutStr := stdout.String()
	if !strings.Contains(stdoutStr, "All quality checks passed successfully") {
		t.Errorf("Expected output not found in stdout: %q", stdoutStr)
		t.Logf("Stderr: %q", stderr.String())
	}
}

// TestIntegration_LintWithErrors tests the lint command with errors
func TestIntegration_LintWithErrors(t *testing.T) {
	tempDir := t.TempDir()
	
	// Create a script that exits with error
	scriptPath := filepath.Join(tempDir, "lint.sh")
	scriptContent := `#!/bin/bash
echo "file.js:10:5: error: Missing semicolon" >&2
exit 1`
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
		t.Fatalf("Failed to write script: %v", err)
	}
	
	// Create configuration
	cfg := &config.Config{
		Version: "1.0",
		Commands: map[string]*config.CommandConfig{
			"lint": {
				Command: "bash",
				Args:    []string{scriptPath},
				ErrorDetection: &config.ErrorDetection{
					ExitCodes: []int{1},
				},
				OutputFilter: &config.FilterConfig{
					ErrorPatterns: []*config.RegexPattern{
						{Pattern: "error:", Flags: "i"},
					},
					ContextLines: 0,
					MaxOutput:    10,
				},
				Prompt: "Fix the linting errors below:",
			},
		},
	}
	
	// Write configuration
	configPath := filepath.Join(tempDir, ".qualhook.json")
	configData, err := config.SaveConfig(cfg)
	if err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}
	if err := os.WriteFile(configPath, configData, 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}
	
	// Change to test directory
	oldDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldDir)
	
	// Execute lint command
	rootCmd := newRootCmd()
	rootCmd.SetArgs([]string{"lint"})
	
	// Capture output
	var stdout, stderr bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)
	
	// Also redirect the global output writers used by executeCommand
	oldOutputWriter := outputWriter
	oldErrorWriter := errorWriter
	outputWriter = &stdout
	errorWriter = &stderr
	defer func() {
		outputWriter = oldOutputWriter
		errorWriter = oldErrorWriter
	}()
	
	// Replace os.Exit temporarily to capture exit code
	oldExit := osExit
	exitCode := 0
	osExit = func(code int) {
		exitCode = code
	}
	defer func() { osExit = oldExit }()
	
	// Execute the command
	err = rootCmd.Execute()
	
	// Verify exit code 2 for errors
	if exitCode != 2 {
		t.Errorf("Expected exit code 2, got %d", exitCode)
	}
	
	// Verify output contains the error (error reporter outputs to stderr)
	stdoutStr := stdout.String()
	stderrStr := stderr.String()
	combinedOutput := stdoutStr + stderrStr
	
	if !strings.Contains(combinedOutput, "Fix the linting errors below:") {
		t.Errorf("Expected prompt not found. Stdout: %s, Stderr: %s", stdoutStr, stderrStr)
	}
	if !strings.Contains(combinedOutput, "Missing semicolon") {
		t.Errorf("Expected error not found. Stdout: %s, Stderr: %s", stdoutStr, stderrStr)
	}
}

// TestIntegration_MonorepoFileAware tests file-aware execution in a monorepo
func TestIntegration_MonorepoFileAware(t *testing.T) {
	tempDir := t.TempDir()
	
	// Create monorepo structure
	frontendDir := filepath.Join(tempDir, "frontend")
	backendDir := filepath.Join(tempDir, "backend")
	os.MkdirAll(frontendDir, 0755)
	os.MkdirAll(backendDir, 0755)
	
	// Create test files
	frontendFile := filepath.Join(frontendDir, "app.js")
	backendFile := filepath.Join(backendDir, "server.go")
	os.WriteFile(frontendFile, []byte("console.log('frontend')"), 0644)
	os.WriteFile(backendFile, []byte("package main"), 0644)
	
	// Create configuration
	cfg := &config.Config{
		Version: "1.0",
		Commands: map[string]*config.CommandConfig{
			"lint": {
				Command: "echo",
				Args:    []string{"No linting configured for root"},
				ErrorDetection: &config.ErrorDetection{
					ExitCodes: []int{1},
				},
				OutputFilter: &config.FilterConfig{
					ErrorPatterns: []*config.RegexPattern{
						{Pattern: "error"},
					},
				},
			},
		},
		Paths: []*config.PathConfig{
			{
				Path: "frontend/**",
				Commands: map[string]*config.CommandConfig{
					"lint": {
						Command: "echo",
						Args:    []string{"Linting frontend"},
						ErrorDetection: &config.ErrorDetection{
							ExitCodes: []int{1},
						},
						OutputFilter: &config.FilterConfig{
							ErrorPatterns: []*config.RegexPattern{
								{Pattern: "error"},
							},
						},
					},
				},
			},
			{
				Path: "backend/**",
				Commands: map[string]*config.CommandConfig{
					"lint": {
						Command: "echo",
						Args:    []string{"Linting backend"},
						ErrorDetection: &config.ErrorDetection{
							ExitCodes: []int{1},
						},
						OutputFilter: &config.FilterConfig{
							ErrorPatterns: []*config.RegexPattern{
								{Pattern: "error"},
							},
						},
					},
				},
			},
		},
	}
	
	// Write configuration
	configPath := filepath.Join(tempDir, ".qualhook.json")
	configData, err := config.SaveConfig(cfg)
	if err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}
	if err := os.WriteFile(configPath, configData, 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}
	
	// Create hook input with edited files
	inputData, err := json.Marshal(map[string]interface{}{
		"command":      "str_replace",
		"path":         frontendFile,
		"old_str":      "console.log('frontend')",
		"new_str":      "console.log('updated frontend')",
	})
	if err != nil {
		t.Fatalf("Failed to marshal input data: %v", err)
	}

	hookInput := &hook.HookInput{
		SessionID:      "test-session",
		TranscriptPath: "/tmp/transcript",
		CWD:            tempDir,
		HookEventName:  "post_command",
		ToolUse: &hook.ToolUse{
			Name:  "str_replace_editor",
			Input: json.RawMessage(inputData),
		},
	}
	
	// Write hook input
	hookInputPath := filepath.Join(tempDir, "hook_input.json")
	hookInputData, _ := json.Marshal(hookInput)
	os.WriteFile(hookInputPath, hookInputData, 0644)
	
	// Change to test directory
	oldDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldDir)
	
	// Set environment variable for hook input
	os.Setenv("CLAUDE_CODE_HOOK_INPUT_FILE", hookInputPath)
	defer os.Unsetenv("CLAUDE_CODE_HOOK_INPUT_FILE")
	
	// Execute lint command
	rootCmd := newRootCmd()
	rootCmd.SetArgs([]string{"lint"})
	
	// Capture output
	var stdout, stderr bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)
	
	// Also redirect the global output writers used by executeCommand
	oldOutputWriter := outputWriter
	oldErrorWriter := errorWriter
	outputWriter = &stdout
	errorWriter = &stderr
	defer func() {
		outputWriter = oldOutputWriter
		errorWriter = oldErrorWriter
	}()
	
	err = rootCmd.Execute()
	if err != nil {
		t.Errorf("Command failed: %v", err)
	}
	
	// Verify command succeeded (file-aware execution happened)
	stdoutStr := stdout.String()
	if !strings.Contains(stdoutStr, "All quality checks passed successfully") {
		t.Errorf("Expected success message not found: %s", stdoutStr)
	}
	// Note: We can't verify which component was linted because the error reporter
	// replaces the actual command output with the standardized message
}

// TestIntegration_CustomCommand tests custom command execution
func TestIntegration_CustomCommand(t *testing.T) {
	tempDir := t.TempDir()
	
	// Create configuration with custom command
	cfg := &config.Config{
		Version: "1.0",
		Commands: map[string]*config.CommandConfig{
			"custom-check": {
				Command: "echo",
				Args:    []string{"Running custom check"},
				ErrorDetection: &config.ErrorDetection{
					ExitCodes: []int{1},
				},
				OutputFilter: &config.FilterConfig{
					ErrorPatterns: []*config.RegexPattern{
						{Pattern: "error"},
					},
				},
			},
		},
	}
	
	// Write configuration
	configPath := filepath.Join(tempDir, ".qualhook.json")
	configData, err := config.SaveConfig(cfg)
	if err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}
	if err := os.WriteFile(configPath, configData, 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}
	
	// Change to test directory
	oldDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldDir)
	
	// Execute custom command using tryCustomCommand (custom commands aren't cobra subcommands)
	// Capture stdout
	oldStdout := os.Stdout
	oldOutputWriter := outputWriter
	r, w, _ := os.Pipe()
	os.Stdout = w
	outputWriter = w
	
	// tryCustomCommand will load config from current directory
	// since we've already changed to tempDir
	
	err = tryCustomCommand("custom-check", []string{})
	
	w.Close()
	os.Stdout = oldStdout
	outputWriter = oldOutputWriter
	
	// Read captured output
	var buf bytes.Buffer
	io.Copy(&buf, r)
	
	if err != nil {
		t.Errorf("Command failed: %v", err)
	}
	
	// Verify output - error reporter replaces actual output
	if !strings.Contains(buf.String(), "All quality checks passed successfully") {
		t.Errorf("Expected output not found: %s", buf.String())
	}
}

// TestIntegration_ConfigValidation tests config validation command
func TestIntegration_ConfigValidation(t *testing.T) {
	tempDir := t.TempDir()
	
	// Create invalid configuration
	invalidConfig := `{
		"version": "1.0",
		"commands": {
			"lint": {
				"outputFilter": {
					"errorPatterns": []
				}
			}
		}
	}`
	
	// Write configuration
	configPath := filepath.Join(tempDir, ".qualhook.json")
	if err := os.WriteFile(configPath, []byte(invalidConfig), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}
	
	// Change to test directory
	oldDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldDir)
	
	// Execute config validate command
	rootCmd := newRootCmd()
	rootCmd.SetArgs([]string{"config", "--validate"})
	
	// Capture output
	var stdout, stderr bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)
	
	// Expect validation to fail
	err := rootCmd.Execute()
	if err == nil {
		t.Error("Expected validation to fail for invalid config")
	}
	
	// Verify error message
	output := stdout.String() + stderr.String()
	if !strings.Contains(output, "command is required") {
		t.Errorf("Expected validation error not found: %s", output)
	}
}

// TestIntegration_DebugMode tests debug mode execution
func TestIntegration_DebugMode(t *testing.T) {
	tempDir := t.TempDir()
	
	// Create configuration
	cfg := &config.Config{
		Version: "1.0",
		Commands: map[string]*config.CommandConfig{
			"test": {
				Command: "echo",
				Args:    []string{"Running tests"},
				ErrorDetection: &config.ErrorDetection{
					ExitCodes: []int{1},
				},
				OutputFilter: &config.FilterConfig{
					ErrorPatterns: []*config.RegexPattern{
						{Pattern: "fail"},
					},
				},
			},
		},
	}
	
	// Write configuration
	configPath := filepath.Join(tempDir, ".qualhook.json")
	configData, err := config.SaveConfig(cfg)
	if err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}
	if err := os.WriteFile(configPath, configData, 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}
	
	// Change to test directory
	oldDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldDir)
	
	// Set up debug output capture
	var debugBuf bytes.Buffer
	debug.SetWriter(&debugBuf)
	debug.Enable() // Enable debug mode for this test
	defer debug.SetWriter(os.Stderr) // Reset after test
	
	// Execute test command with debug flag
	rootCmd := newRootCmd()
	rootCmd.SetArgs([]string{"--debug", "test"})
	
	// Capture output
	var stdout, stderr bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)
	
	// Also redirect the global output writers used by executeCommand
	oldOutputWriter := outputWriter
	oldErrorWriter := errorWriter
	outputWriter = &stdout
	errorWriter = &stderr
	defer func() {
		outputWriter = oldOutputWriter
		errorWriter = oldErrorWriter
	}()
	
	err = rootCmd.Execute()
	if err != nil {
		t.Errorf("Command failed: %v", err)
	}
	
	// Get debug output
	debugOutput := debugBuf.String()
	
	// Verify debug output
	if !strings.Contains(debugOutput, "[DEBUG") {
		t.Errorf("Expected debug output not found. Debug output: %s", debugOutput)
	}
	if !strings.Contains(debugOutput, "Command Execution") {
		t.Errorf("Expected debug section not found. Debug output: %s", debugOutput)
	}
}

// Helper function to capture command output
func captureOutput(f func() error) (string, string, error) {
	oldStdout := os.Stdout
	oldStderr := os.Stderr
	
	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()
	
	os.Stdout = wOut
	os.Stderr = wErr
	
	err := f()
	
	wOut.Close()
	wErr.Close()
	
	stdout, _ := io.ReadAll(rOut)
	stderr, _ := io.ReadAll(rErr)
	
	os.Stdout = oldStdout
	os.Stderr = oldStderr
	
	return string(stdout), string(stderr), err
}