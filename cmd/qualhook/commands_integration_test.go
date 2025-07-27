//go:build integration

package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bebsworthy/qualhook/internal/debug"
	"github.com/bebsworthy/qualhook/internal/hook"
	"github.com/bebsworthy/qualhook/internal/testutil"
	"github.com/bebsworthy/qualhook/pkg/config"
)

// TestCommandExecution tests standard command execution scenarios
func TestCommandExecution(t *testing.T) {
	tests := []struct {
		name            string
		command         string
		args            []string
		config          *config.Config
		wantExitCode    int
		wantStdout      string
		wantStderr      string
		skipStdoutCheck bool // For cases where error reporter modifies output
	}{
		{
			name:    "format success",
			command: "format",
			config: testutil.NewConfigBuilder().
				WithSimpleCommand("format", "echo", "Formatting complete").
				Build(),
			wantExitCode:    0,
			wantStdout:      "All quality checks passed successfully",
			skipStdoutCheck: false,
		},
		{
			name:    "lint success",
			command: "lint",
			config: testutil.NewConfigBuilder().
				WithSimpleCommand("lint", "echo", "No linting issues found").
				Build(),
			wantExitCode:    0,
			wantStdout:      "All quality checks passed successfully",
			skipStdoutCheck: false,
		},
		{
			name:    "typecheck success",
			command: "typecheck",
			config: testutil.NewConfigBuilder().
				WithSimpleCommand("typecheck", "echo", "Type checking passed").
				Build(),
			wantExitCode:    0,
			wantStdout:      "All quality checks passed successfully",
			skipStdoutCheck: false,
		},
		{
			name:    "test success",
			command: "test",
			config: testutil.NewConfigBuilder().
				WithSimpleCommand("test", "echo", "All tests passed").
				Build(),
			wantExitCode:    0,
			wantStdout:      "All quality checks passed successfully",
			skipStdoutCheck: false,
		},
		{
			name:    "format with errors",
			command: "format",
			config: testutil.NewConfigBuilder().
				WithCommand("format", &config.CommandConfig{
					Command:   "echo",
					Args:      []string{"Error: File not formatted: main.js"},
					ExitCodes: []int{0}, // echo returns 0, but we'll match on pattern
					ErrorPatterns: []*config.RegexPattern{
						{Pattern: "Error:", Flags: ""},
					},
					MaxOutput: 100,
					Prompt:    "Fix the formatting issues below:",
				}).Build(),
			wantExitCode: 2,
			wantStderr:   "Fix the formatting issues below:",
		},
		{
			name:    "lint with errors",
			command: "lint",
			config: testutil.NewConfigBuilder().
				WithCommand("lint", &config.CommandConfig{
					Command:   "echo",
					Args:      []string{"error: lint failed at line 10"},
					ExitCodes: []int{0},
					ErrorPatterns: []*config.RegexPattern{
						{Pattern: "error:", Flags: ""},
					},
					MaxOutput: 100,
					Prompt:    "Fix the linting errors below:",
				}).Build(),
			wantExitCode: 2,
			wantStderr:   "Fix the linting errors below:",
		},
		{
			name:    "custom command success",
			command: "custom-check",
			config: testutil.NewConfigBuilder().
				WithSimpleCommand("custom-check", "echo", "Running custom check").
				Build(),
			wantExitCode:    0,
			wantStdout:      "All quality checks passed successfully",
			skipStdoutCheck: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test directory with config
			tempDir := t.TempDir()
			_, err := testutil.CreateTestConfigFile(tempDir, tt.config)
			if err != nil {
				t.Fatalf("Failed to create test config: %v", err)
			}

			// Change to test directory
			oldDir, _ := os.Getwd()
			os.Chdir(tempDir)
			defer os.Chdir(oldDir)

			// Capture exit code
			oldExit := osExit
			exitCode := -1
			osExit = func(code int) {
				exitCode = code
			}
			defer func() { osExit = oldExit }()

			// Capture output
			stdout := testutil.NewTestWriter()
			stderr := testutil.NewTestWriter()

			oldOutputWriter := outputWriter
			oldErrorWriter := errorWriter
			outputWriter = stdout
			errorWriter = stderr
			defer func() {
				outputWriter = oldOutputWriter
				errorWriter = oldErrorWriter
			}()

			// Execute command
			if tt.command == "custom-check" {
				// Custom commands aren't cobra subcommands
				err = tryCustomCommand(tt.command, tt.args)
			} else {
				cmd := newRootCmd()
				cmd.SetArgs(append([]string{tt.command}, tt.args...))
				cmd.SetOut(stdout)
				cmd.SetErr(stderr)
				err = cmd.Execute()
			}

			// Check results
			if exitCode == -1 {
				exitCode = 0 // No exit was called, assume success
			}

			if exitCode != tt.wantExitCode {
				t.Errorf("Exit code = %d, want %d", exitCode, tt.wantExitCode)
			}

			stdoutStr := stdout.String()
			stderrStr := stderr.String()

			if !tt.skipStdoutCheck && tt.wantStdout != "" && !strings.Contains(stdoutStr, tt.wantStdout) {
				t.Errorf("Stdout does not contain %q, got: %q", tt.wantStdout, stdoutStr)
			}

			if tt.wantStderr != "" && !strings.Contains(stderrStr, tt.wantStderr) {
				t.Errorf("Stderr does not contain %q, got: %q", tt.wantStderr, stderrStr)
			}
		})
	}
}

// TestMonorepoFileAwareExecution tests file-aware execution in a monorepo
func TestMonorepoFileAwareExecution(t *testing.T) {
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

	// Create monorepo configuration
	cfg := testutil.NewConfigBuilder().
		WithSimpleCommand("lint", "echo", "No linting configured for root").
		WithPathCommand("frontend/**", map[string]*config.CommandConfig{
			"lint": testutil.SafeCommandConfig("Linting frontend"),
		}).
		WithPathCommand("backend/**", map[string]*config.CommandConfig{
			"lint": testutil.SafeCommandConfig("Linting backend"),
		}).
		Build()

	// Write configuration
	_, err := testutil.CreateTestConfigFile(tempDir, cfg)
	if err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	tests := []struct {
		name         string
		workDir      string
		hookInput    *hook.HookInput
		wantExitCode int
		wantOutput   string
	}{
		{
			name:    "frontend file edit triggers frontend lint",
			workDir: tempDir,
			hookInput: &hook.HookInput{
				SessionID:      "test-session",
				TranscriptPath: "/tmp/transcript",
				CWD:            tempDir,
				HookEventName:  "post_command",
				ToolUse: &hook.ToolUse{
					Name: "str_replace_editor",
					Input: json.RawMessage(`{
						"command": "str_replace",
						"path": "` + frontendFile + `",
						"old_str": "console.log('frontend')",
						"new_str": "console.log('updated frontend')"
					}`),
				},
			},
			wantExitCode: 0,
			wantOutput:   "All quality checks passed successfully",
		},
		{
			name:    "backend file edit triggers backend lint",
			workDir: tempDir,
			hookInput: &hook.HookInput{
				SessionID:      "test-session",
				TranscriptPath: "/tmp/transcript",
				CWD:            tempDir,
				HookEventName:  "post_command",
				ToolUse: &hook.ToolUse{
					Name: "str_replace_editor",
					Input: json.RawMessage(`{
						"command": "str_replace",
						"path": "` + backendFile + `",
						"old_str": "package main",
						"new_str": "package main\n\nimport \"fmt\""
					}`),
				},
			},
			wantExitCode: 0,
			wantOutput:   "All quality checks passed successfully",
		},
		{
			name:         "no hook input runs root command",
			workDir:      tempDir,
			hookInput:    nil,
			wantExitCode: 0,
			wantOutput:   "All quality checks passed successfully",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Change to work directory
			oldDir, _ := os.Getwd()
			os.Chdir(tt.workDir)
			defer os.Chdir(oldDir)

			// Set up hook input if provided
			if tt.hookInput != nil {
				hookInputPath := filepath.Join(tempDir, "hook_input.json")
				hookInputData, _ := json.Marshal(tt.hookInput)
				os.WriteFile(hookInputPath, hookInputData, 0644)
				os.Setenv("CLAUDE_CODE_HOOK_INPUT_FILE", hookInputPath)
				defer os.Unsetenv("CLAUDE_CODE_HOOK_INPUT_FILE")
			}

			// Capture output
			stdout := testutil.NewTestWriter()
			stderr := testutil.NewTestWriter()

			oldOutputWriter := outputWriter
			oldErrorWriter := errorWriter
			outputWriter = stdout
			errorWriter = stderr
			defer func() {
				outputWriter = oldOutputWriter
				errorWriter = oldErrorWriter
			}()

			// Execute lint command
			cmd := newRootCmd()
			cmd.SetArgs([]string{"lint"})
			cmd.SetOut(stdout)
			cmd.SetErr(stderr)

			err := cmd.Execute()
			if err != nil {
				t.Errorf("Command failed: %v", err)
			}

			// Verify output
			output := stdout.String() + stderr.String()
			if !strings.Contains(output, tt.wantOutput) {
				t.Errorf("Expected output %q not found, got: %q", tt.wantOutput, output)
			}
		})
	}
}

// TestConfigValidation tests config validation command
func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name       string
		configJSON string
		wantError  bool
		errorMsg   string
	}{
		{
			name: "valid config",
			configJSON: `{
				"version": "1.0",
				"commands": {
					"lint": {
						"command": "echo",
						"args": ["linting"],
						"exitCodes": [1],
						"errorPatterns": [{"pattern": "error", "flags": "i"}]
					}
				}
			}`,
			wantError: false,
		},
		{
			name: "missing command field",
			configJSON: `{
				"version": "1.0",
				"commands": {
					"lint": {
						"errorPatterns": []
					}
				}
			}`,
			wantError: true,
			errorMsg:  "command is required",
		},
		{
			name: "invalid version",
			configJSON: `{
				"version": "0.5",
				"commands": {}
			}`,
			wantError: true,
			errorMsg:  "version", // Just check that version is mentioned in the error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			configPath := filepath.Join(tempDir, ".qualhook.json")
			if err := os.WriteFile(configPath, []byte(tt.configJSON), 0644); err != nil {
				t.Fatalf("Failed to write config: %v", err)
			}

			// Change to test directory
			oldDir, _ := os.Getwd()
			os.Chdir(tempDir)
			defer os.Chdir(oldDir)

			// Execute config validate command
			cmd := newRootCmd()
			cmd.SetArgs([]string{"config", "--validate"})

			var stdout, stderr bytes.Buffer
			cmd.SetOut(&stdout)
			cmd.SetErr(&stderr)

			err := cmd.Execute()

			if tt.wantError {
				if err == nil {
					t.Error("Expected validation to fail but it succeeded")
				}
				output := stdout.String() + stderr.String()
				if tt.errorMsg != "" && !strings.Contains(output, tt.errorMsg) {
					t.Errorf("Expected error message %q not found in output: %s", tt.errorMsg, output)
				}
			} else {
				if err != nil {
					t.Errorf("Expected validation to succeed but got error: %v", err)
				}
			}
		})
	}
}

// TestDebugMode tests debug mode execution
func TestDebugMode(t *testing.T) {
	tempDir := t.TempDir()

	cfg := testutil.NewConfigBuilder().
		WithSimpleCommand("test", "echo", "Running tests").
		Build()

	_, err := testutil.CreateTestConfigFile(tempDir, cfg)
	if err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	// Change to test directory
	oldDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldDir)

	// Set up debug output capture
	var debugBuf bytes.Buffer
	debug.SetWriter(&debugBuf)
	debug.Enable()
	defer debug.SetWriter(os.Stderr)

	// Capture regular output
	stdout := testutil.NewTestWriter()
	stderr := testutil.NewTestWriter()

	oldOutputWriter := outputWriter
	oldErrorWriter := errorWriter
	outputWriter = stdout
	errorWriter = stderr
	defer func() {
		outputWriter = oldOutputWriter
		errorWriter = oldErrorWriter
	}()

	// Execute test command with debug flag
	cmd := newRootCmd()
	cmd.SetArgs([]string{"--debug", "test"})
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)

	err = cmd.Execute()
	if err != nil {
		t.Errorf("Command failed: %v", err)
	}

	// Verify debug output
	debugOutput := debugBuf.String()
	if !strings.Contains(debugOutput, "[DEBUG") {
		t.Errorf("Expected debug output not found. Debug output: %s", debugOutput)
	}
	if !strings.Contains(debugOutput, "Command Execution") {
		t.Errorf("Expected debug section not found. Debug output: %s", debugOutput)
	}

	// Verify regular output still works
	if !strings.Contains(stdout.String(), "All quality checks passed successfully") {
		t.Errorf("Expected success message not found: %s", stdout.String())
	}
}

// TestFailingCommands tests various failure scenarios
func TestFailingCommands(t *testing.T) {
	testutil.SkipOnWindows(t, "script execution tests")

	tempDir := t.TempDir()

	// Create a script that exits with code 1
	exitScript := filepath.Join(tempDir, "exit1.sh")
	if err := os.WriteFile(exitScript, []byte("#!/bin/bash\nexit 1"), 0755); err != nil {
		t.Fatalf("Failed to write exit script: %v", err)
	}

	// Create a script that outputs error pattern
	errorScript := filepath.Join(tempDir, "error.sh")
	if err := os.WriteFile(errorScript, []byte("#!/bin/bash\necho 'FAIL: test failed' >&2\nexit 0"), 0755); err != nil {
		t.Fatalf("Failed to write error script: %v", err)
	}

	tests := []struct {
		name         string
		config       *config.Config
		command      string
		wantExitCode int
		wantError    string
	}{
		{
			name: "command exits with error code",
			config: testutil.NewConfigBuilder().
				WithCommand("lint", &config.CommandConfig{
					Command:   "bash",
					Args:      []string{exitScript},
					ExitCodes: []int{1}, // Exit code 1 indicates errors
					MaxOutput: 100,
				}).Build(),
			command:      "lint",
			wantExitCode: 2,
		},
		{
			name: "command output matches error pattern",
			config: testutil.NewConfigBuilder().
				WithCommand("test", &config.CommandConfig{
					Command:   "bash",
					Args:      []string{errorScript},
					ExitCodes: []int{0},
					ErrorPatterns: []*config.RegexPattern{
						{Pattern: "FAIL:", Flags: ""},
					},
					MaxOutput: 100,
					Prompt:    "Tests failed:",
				}).Build(),
			command:      "test",
			wantExitCode: 2,
			wantError:    "Tests failed:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create config in the parent tempDir where scripts are
			_, err := testutil.CreateTestConfigFile(tempDir, tt.config)
			if err != nil {
				t.Fatalf("Failed to create config: %v", err)
			}

			// Change to test directory
			oldDir, _ := os.Getwd()
			os.Chdir(tempDir)
			defer os.Chdir(oldDir)

			// Capture exit code
			oldExit := osExit
			exitCode := -1
			osExit = func(code int) {
				exitCode = code
			}
			defer func() { osExit = oldExit }()

			// Capture output
			stdout := testutil.NewTestWriter()
			stderr := testutil.NewTestWriter()
			oldOutputWriter := outputWriter
			oldErrorWriter := errorWriter
			outputWriter = stdout
			errorWriter = stderr
			defer func() {
				outputWriter = oldOutputWriter
				errorWriter = oldErrorWriter
			}()

			// Execute command
			cmd := newRootCmd()
			cmd.SetArgs([]string{tt.command})
			cmd.Execute()

			// Verify exit code
			if exitCode != tt.wantExitCode {
				t.Errorf("Exit code = %d, want %d", exitCode, tt.wantExitCode)
				t.Logf("stdout: %q", stdout.String())
				t.Logf("stderr: %q", stderr.String())
			}

			// Verify error output
			if tt.wantError != "" {
				stderrStr := stderr.String()
				if !strings.Contains(stderrStr, tt.wantError) {
					t.Errorf("Stderr does not contain %q, got: %q", tt.wantError, stderrStr)
				}
			}
		})
	}
}

// TestScriptExecution tests execution of actual scripts
func TestScriptExecution(t *testing.T) {
	testutil.SkipOnWindows(t, "script execution tests")

	tempDir := t.TempDir()

	// Create a simple test script that echoes success
	scriptPath := filepath.Join(tempDir, "lint.sh")
	scriptContent := `#!/bin/bash
echo "Linting complete"
exit 0`

	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
		t.Fatalf("Failed to write script: %v", err)
	}

	// Create an error script
	errorScriptPath := filepath.Join(tempDir, "lint-error.sh")
	errorScriptContent := `#!/bin/bash
echo "Error: linting failed" >&2
exit 1`

	if err := os.WriteFile(errorScriptPath, []byte(errorScriptContent), 0755); err != nil {
		t.Fatalf("Failed to write error script: %v", err)
	}

	tests := []struct {
		name         string
		scriptPath   string
		wantExitCode int
		wantOutput   string
	}{
		{
			name:         "successful script",
			scriptPath:   scriptPath,
			wantExitCode: 0,
			wantOutput:   "All quality checks passed successfully",
		},
		{
			name:         "failing script",
			scriptPath:   errorScriptPath,
			wantExitCode: 2,
			wantOutput:   "Error: linting failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create config that uses the script
			cfg := testutil.NewConfigBuilder().
				WithCommand("lint", &config.CommandConfig{
					Command:   "bash",
					Args:      []string{tt.scriptPath},
					ExitCodes: []int{1}, // bash exits with 1 on error
					ErrorPatterns: []*config.RegexPattern{
						{Pattern: "Error:", Flags: ""},
					},
					MaxOutput: 100,
					Prompt:    "Fix linting errors:",
				}).Build()

			_, err := testutil.CreateTestConfigFile(tempDir, cfg)
			if err != nil {
				t.Fatalf("Failed to create config: %v", err)
			}

			// Change to test directory
			oldDir, _ := os.Getwd()
			os.Chdir(tempDir)
			defer os.Chdir(oldDir)

			// Capture exit code
			oldExit := osExit
			exitCode := -1
			osExit = func(code int) {
				exitCode = code
			}
			defer func() { osExit = oldExit }()

			// Capture output
			stdout := testutil.NewTestWriter()
			stderr := testutil.NewTestWriter()

			oldOutputWriter := outputWriter
			oldErrorWriter := errorWriter
			outputWriter = stdout
			errorWriter = stderr
			defer func() {
				outputWriter = oldOutputWriter
				errorWriter = oldErrorWriter
			}()

			// Execute command
			cmd := newRootCmd()
			cmd.SetArgs([]string{"lint"})
			cmd.Execute()

			// Check results
			if exitCode == -1 {
				exitCode = 0
			}

			if exitCode != tt.wantExitCode {
				t.Errorf("Exit code = %d, want %d", exitCode, tt.wantExitCode)
			}

			output := stdout.String() + stderr.String()
			if !strings.Contains(output, tt.wantOutput) {
				t.Errorf("Output does not contain %q, got: %q", tt.wantOutput, output)
			}
		})
	}
}
