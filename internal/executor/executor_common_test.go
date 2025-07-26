//go:build unit

package executor

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"testing"
	"time"
)

// TestExecutors_CommonBehavior tests behaviors that should be consistent across all executor types
func TestExecutors_CommonBehavior(t *testing.T) {
	// Test both command executor and parallel executor with single command
	testCases := []struct {
		name        string
		getExecutor func() interface{ Execute(string, []string, ExecOptions) (*ExecResult, error) }
	}{
		{
			name: "CommandExecutor",
			getExecutor: func() interface{ Execute(string, []string, ExecOptions) (*ExecResult, error) } {
				return NewCommandExecutor(10 * time.Second)
			},
		},
		{
			name: "ParallelExecutor_SingleCommand",
			getExecutor: func() interface{ Execute(string, []string, ExecOptions) (*ExecResult, error) } {
				cmdExec := NewCommandExecutor(10 * time.Second)
				pe := NewParallelExecutor(cmdExec, 4)
				return &parallelExecutorAdapter{pe}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			executor := tc.getExecutor()
			
			t.Run("timeout_behavior", func(t *testing.T) {
				testTimeoutBehavior(t, executor)
			})
			
			t.Run("error_handling", func(t *testing.T) {
				testErrorHandling(t, executor)
			})
			
			t.Run("environment_variables", func(t *testing.T) {
				testEnvironmentVariables(t, executor)
			})
		})
	}
}

// parallelExecutorAdapter adapts ParallelExecutor to the simple Execute interface
type parallelExecutorAdapter struct {
	pe *ParallelExecutor
}

func (a *parallelExecutorAdapter) Execute(cmd string, args []string, opts ExecOptions) (*ExecResult, error) {
	ctx := context.Background()
	commands := []ParallelCommand{
		{
			ID:      "single",
			Command: cmd,
			Args:    args,
			Options: opts,
		},
	}
	
	result, err := a.pe.Execute(ctx, commands, nil)
	if err != nil {
		return nil, err
	}
	
	if execResult, ok := result.Results["single"]; ok {
		return execResult, nil
	}
	
	return nil, fmt.Errorf("no result found")
}

func testTimeoutBehavior(t *testing.T, executor interface{ Execute(string, []string, ExecOptions) (*ExecResult, error) }) {
	tests := []struct {
		name         string
		timeout      time.Duration
		sleepSeconds int
		expectTimeout bool
	}{
		{
			name:         "command completes before timeout",
			timeout:      2 * time.Second,
			sleepSeconds: 0,
			expectTimeout: false,
		},
		{
			name:         "command times out",
			timeout:      100 * time.Millisecond,
			sleepSeconds: 2,
			expectTimeout: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, args := pc.sleep(tt.sleepSeconds)
			if tt.sleepSeconds == 0 {
				// Use echo instead of sleep 0 for faster test
				cmd, args = pc.echo("quick")
			}
			
			result, err := executor.Execute(cmd, args, ExecOptions{Timeout: tt.timeout})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			
			if tt.expectTimeout {
				if !result.TimedOut {
					t.Error("expected command to timeout")
				}
				if result.ExitCode == 0 {
					t.Error("expected non-zero exit code for timeout")
				}
			} else {
				if result.TimedOut {
					t.Error("expected command to complete without timeout")
				}
			}
		})
	}
}

func testErrorHandling(t *testing.T, executor interface{ Execute(string, []string, ExecOptions) (*ExecResult, error) }) {
	tests := []struct {
		name          string
		setupCmd      func() (string, []string)
		expectError   bool
		expectExitCode int
	}{
		{
			name:          "command not found",
			setupCmd:      func() (string, []string) { return "non-existent-command-12345", []string{} },
			expectError:   true,
			expectExitCode: -1, // Error before execution
		},
		{
			name:          "exit with code 1",
			setupCmd:      func() (string, []string) { return pc.exit(1) },
			expectError:   false,
			expectExitCode: 1,
		},
		{
			name:          "exit with code 42",
			setupCmd:      func() (string, []string) { return pc.exit(42) },
			expectError:   false,
			expectExitCode: 42,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, args := tt.setupCmd()
			result, err := executor.Execute(cmd, args, ExecOptions{})
			
			if err != nil {
				t.Fatalf("unexpected execution error: %v", err)
			}
			
			if tt.expectError && result.Error == nil {
				t.Error("expected error in result")
			}
			
			if !tt.expectError && result.Error != nil {
				t.Errorf("unexpected error in result: %v", result.Error)
			}
			
			if tt.expectExitCode >= 0 && result.ExitCode != tt.expectExitCode {
				t.Errorf("expected exit code %d, got %d", tt.expectExitCode, result.ExitCode)
			}
		})
	}
}

func testEnvironmentVariables(t *testing.T, executor interface{ Execute(string, []string, ExecOptions) (*ExecResult, error) }) {
	tests := []struct {
		name         string
		envVars      []string
		inheritEnv   bool
		expectedOutput string
	}{
		{
			name:         "custom environment variable",
			envVars:      []string{"TEST_VAR=custom_value"},
			inheritEnv:   true,
			expectedOutput: "custom_value",
		},
		{
			name:         "multiple environment variables",
			envVars:      []string{"TEST_VAR=value1", "OTHER_VAR=value2"},
			inheritEnv:   true,
			expectedOutput: "value1",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cmd string
			var args []string
			
			if isWindows() {
				cmd = cmdCommand
				args = []string{"/c", "echo", "%TEST_VAR%"}
			} else {
				cmd = shCommand
				args = []string{"-c", "echo $TEST_VAR"}
			}
			
			result, err := executor.Execute(cmd, args, ExecOptions{
				Environment: tt.envVars,
				InheritEnv:  tt.inheritEnv,
			})
			
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			
			if !strings.Contains(result.Stdout, tt.expectedOutput) {
				t.Errorf("expected output to contain %q, got %q", tt.expectedOutput, result.Stdout)
			}
		})
	}
}

// isWindows returns true if running on Windows
func isWindows() bool {
	return runtime.GOOS == osWindows
}