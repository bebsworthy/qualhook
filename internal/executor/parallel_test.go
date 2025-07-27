//go:build unit

package executor

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewParallelExecutor(t *testing.T) {
	t.Parallel()
	cmdExecutor := NewCommandExecutor(10 * time.Second)

	tests := []struct {
		name             string
		maxParallel      int
		expectedParallel int
	}{
		{
			name:             "positive value",
			maxParallel:      8,
			expectedParallel: 8,
		},
		{
			name:             "zero value defaults to 4",
			maxParallel:      0,
			expectedParallel: 4,
		},
		{
			name:             "negative value defaults to 4",
			maxParallel:      -1,
			expectedParallel: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			pe := NewParallelExecutor(cmdExecutor, tt.maxParallel)
			if pe.maxParallel != tt.expectedParallel {
				t.Errorf("expected maxParallel %d, got %d", tt.expectedParallel, pe.maxParallel)
			}
		})
	}
}

func TestParallelExecute_Basic(t *testing.T) {
	t.Parallel()
	cmdExecutor := NewCommandExecutor(10 * time.Second)
	pe := NewParallelExecutor(cmdExecutor, 4)

	// Create test commands
	var commands []ParallelCommand
	for i := 0; i < 3; i++ {
		cmd, args := pc.echo(fmt.Sprintf("test%d", i))
		commands = append(commands, ParallelCommand{
			ID:      fmt.Sprintf("cmd-%d", i),
			Command: cmd,
			Args:    args,
			Options: ExecOptions{},
		})
	}

	ctx := context.Background()
	result, err := pe.Execute(ctx, commands, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify results
	if len(result.Results) != 3 {
		t.Errorf("expected 3 results, got %d", len(result.Results))
	}

	if len(result.Order) != 3 {
		t.Errorf("expected 3 items in order, got %d", len(result.Order))
	}

	if result.SuccessCount != 3 {
		t.Errorf("expected 3 successes, got %d", result.SuccessCount)
	}

	if result.FailureCount != 0 {
		t.Errorf("expected 0 failures, got %d", result.FailureCount)
	}

	if result.HasFailures {
		t.Error("expected no failures")
	}

	// Check individual results
	for i := 0; i < 3; i++ {
		id := fmt.Sprintf("cmd-%d", i)
		execResult, ok := result.Results[id]
		if !ok {
			t.Errorf("missing result for %s", id)
			continue
		}

		if execResult.ExitCode != 0 {
			t.Errorf("%s: expected exit code 0, got %d", id, execResult.ExitCode)
		}

		expectedOutput := fmt.Sprintf("test%d", i)
		if !strings.Contains(execResult.Stdout, expectedOutput) {
			t.Errorf("%s: expected output to contain %q, got %q", id, expectedOutput, execResult.Stdout)
		}
	}
}

func TestParallelExecute_WithProgress(t *testing.T) {
	t.Parallel()
	cmdExecutor := NewCommandExecutor(10 * time.Second)
	pe := NewParallelExecutor(cmdExecutor, 2)

	// Create test commands
	var commands []ParallelCommand
	for i := 0; i < 4; i++ {
		cmd, args := pc.echo("test")
		commands = append(commands, ParallelCommand{
			ID:      fmt.Sprintf("cmd-%d", i),
			Command: cmd,
			Args:    args,
			Options: ExecOptions{},
		})
	}

	// Track progress
	var progressCount int32
	progressIDs := make(map[string]bool)
	var mu sync.Mutex

	progress := func(completed, total int, currentID string) {
		atomic.AddInt32(&progressCount, 1)

		mu.Lock()
		progressIDs[currentID] = true
		mu.Unlock()

		if completed > total {
			t.Errorf("completed %d > total %d", completed, total)
		}
	}

	ctx := context.Background()
	result, err := pe.Execute(ctx, commands, progress)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify progress was called
	finalCount := atomic.LoadInt32(&progressCount)
	if finalCount != 4 {
		t.Errorf("expected 4 progress calls, got %d", finalCount)
	}

	mu.Lock()
	progressIDCount := len(progressIDs)
	mu.Unlock()

	if progressIDCount != 4 {
		t.Errorf("expected 4 unique IDs in progress, got %d", progressIDCount)
	}

	// Verify all commands succeeded
	if result.SuccessCount != 4 {
		t.Errorf("expected 4 successes, got %d", result.SuccessCount)
	}
}

func TestParallelExecute_WithFailures(t *testing.T) {
	t.Parallel()
	cmdExecutor := NewCommandExecutor(10 * time.Second)
	pe := NewParallelExecutor(cmdExecutor, 4)

	// Mix of successful and failing commands
	success := func() (string, []string) { return pc.echo("success") }
	fail := func() (string, []string) { return pc.exit(1) }

	sCmd, sArgs := success()
	fCmd, fArgs := fail()

	commands := []ParallelCommand{
		{
			ID:      "success",
			Command: sCmd,
			Args:    sArgs,
			Options: ExecOptions{},
		},
		{
			ID:      "fail",
			Command: fCmd,
			Args:    fArgs,
			Options: ExecOptions{},
		},
		{
			ID:      "not-found",
			Command: "this-command-does-not-exist-12345",
			Args:    []string{},
			Options: ExecOptions{},
		},
	}

	ctx := context.Background()
	result, err := pe.Execute(ctx, commands, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify mixed results
	if result.SuccessCount != 1 {
		t.Errorf("expected 1 success, got %d", result.SuccessCount)
	}

	if result.FailureCount != 2 {
		t.Errorf("expected 2 failures, got %d", result.FailureCount)
	}

	if !result.HasFailures {
		t.Error("expected HasFailures to be true")
	}

	// Check specific results
	successResult := result.Results["success"]
	if successResult.ExitCode != 0 {
		t.Error("expected success command to have exit code 0")
	}

	failResult := result.Results["fail"]
	if failResult.ExitCode != 1 {
		t.Error("expected fail command to have exit code 1")
	}

	notFoundResult := result.Results["not-found"]
	if notFoundResult.Error == nil {
		t.Error("expected not-found command to have error")
	}
}

func TestParallelExecute_ContextCancellation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping slow test in short mode")
	}
	t.Parallel()
	cmdExecutor := NewCommandExecutor(10 * time.Second)
	pe := NewParallelExecutor(cmdExecutor, 2)

	// Create slow commands
	var commands []ParallelCommand
	for i := 0; i < 4; i++ {
		cmd, args := pc.sleep(2) // 2 second sleep
		commands = append(commands, ParallelCommand{
			ID:      fmt.Sprintf("cmd-%d", i),
			Command: cmd,
			Args:    args,
			Options: ExecOptions{},
		})
	}

	// Create context with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	result, err := pe.Execute(ctx, commands, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Some commands should have been canceled
	cancelledCount := 0
	for _, execResult := range result.Results {
		if execResult.Error == context.DeadlineExceeded {
			cancelledCount++
		}
	}

	if cancelledCount == 0 {
		t.Error("expected at least some commands to be canceled")
	}
}

func TestParallelExecute_Empty(t *testing.T) {
	t.Parallel()
	cmdExecutor := NewCommandExecutor(10 * time.Second)
	pe := NewParallelExecutor(cmdExecutor, 4)

	ctx := context.Background()
	result, err := pe.Execute(ctx, []ParallelCommand{}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Results) != 0 {
		t.Errorf("expected empty results, got %d", len(result.Results))
	}

	if result.SuccessCount != 0 || result.FailureCount != 0 {
		t.Error("expected zero counts for empty execution")
	}
}

func TestExecuteWithAggregation(t *testing.T) {
	t.Parallel()
	cmdExecutor := NewCommandExecutor(10 * time.Second)
	pe := NewParallelExecutor(cmdExecutor, 4)

	// Create commands with different outputs
	cmd1, args1 := pc.echo("message 1")
	cmd2, args2 := pc.echo("message 2")
	cmdF, argsF := pc.exit(1)

	commands := []ParallelCommand{
		{
			ID:      "success-1",
			Command: cmd1,
			Args:    args1,
			Options: ExecOptions{},
		},
		{
			ID:      "success-2",
			Command: cmd2,
			Args:    args2,
			Options: ExecOptions{},
		},
		{
			ID:      "failure",
			Command: cmdF,
			Args:    argsF,
			Options: ExecOptions{},
		},
	}

	ctx := context.Background()
	result, err := pe.ExecuteWithAggregation(ctx, commands, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check aggregated stdout
	if len(result.CombinedStdout) < 2 {
		t.Errorf("expected at least 2 stdout entries, got %d", len(result.CombinedStdout))
	}

	// Check failed commands
	if len(result.FailedCommands) != 1 {
		t.Errorf("expected 1 failed command, got %d", len(result.FailedCommands))
	}

	if result.FailedCommands[0] != "failure" {
		t.Errorf("expected 'failure' in failed commands, got %v", result.FailedCommands)
	}

	// Check failure summary
	summary := result.GetFailureSummary()
	if summary == "" {
		t.Error("expected non-empty failure summary")
	}

	if !strings.Contains(summary, "Failed commands (1/3)") {
		t.Errorf("unexpected failure summary format: %s", summary)
	}
}

func TestGetFailureSummary(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		result   *AggregatedResult
		expected string
	}{
		{
			name: "no failures",
			result: &AggregatedResult{
				ParallelResult: &ParallelResult{
					HasFailures: false,
				},
			},
			expected: "",
		},
		{
			name: "timeout failure",
			result: &AggregatedResult{
				ParallelResult: &ParallelResult{
					HasFailures:  true,
					FailureCount: 1,
					Order:        []string{"cmd1"},
					Results: map[string]*ExecResult{
						"cmd1": {TimedOut: true},
					},
				},
				FailedCommands: []string{"cmd1"},
			},
			expected: "timed out",
		},
		{
			name: "error failure",
			result: &AggregatedResult{
				ParallelResult: &ParallelResult{
					HasFailures:  true,
					FailureCount: 1,
					Order:        []string{"cmd1"},
					Results: map[string]*ExecResult{
						"cmd1": {Error: fmt.Errorf("command failed")},
					},
				},
				FailedCommands: []string{"cmd1"},
			},
			expected: "command failed",
		},
		{
			name: "exit code failure",
			result: &AggregatedResult{
				ParallelResult: &ParallelResult{
					HasFailures:  true,
					FailureCount: 1,
					Order:        []string{"cmd1"},
					Results: map[string]*ExecResult{
						"cmd1": {ExitCode: 127},
					},
				},
				FailedCommands: []string{"cmd1"},
			},
			expected: "exit code 127",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			summary := tt.result.GetFailureSummary()
			if tt.expected == "" {
				if summary != "" {
					t.Errorf("expected empty summary, got %q", summary)
				}
			} else {
				if !strings.Contains(summary, tt.expected) {
					t.Errorf("expected summary to contain %q, got %q", tt.expected, summary)
				}
			}
		})
	}
}

func TestParallelExecutorPool(t *testing.T) {
	t.Parallel()
	cmdExecutor := NewCommandExecutor(10 * time.Second)

	tests := []struct {
		name         string
		size         int
		expectedSize int
	}{
		{
			name:         "positive size",
			size:         3,
			expectedSize: 3,
		},
		{
			name:         "zero size defaults to 1",
			size:         0,
			expectedSize: 1,
		},
		{
			name:         "negative size defaults to 1",
			size:         -1,
			expectedSize: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			pool := NewParallelExecutorPool(tt.size, cmdExecutor, 4)
			if len(pool.executors) != tt.expectedSize {
				t.Errorf("expected pool size %d, got %d", tt.expectedSize, len(pool.executors))
			}

			// Test round-robin behavior
			executors := make([]*ParallelExecutor, tt.expectedSize*2)
			for i := 0; i < len(executors); i++ {
				executors[i] = pool.Get()
			}

			// Check that we cycle through all executors
			for i := 0; i < tt.expectedSize; i++ {
				if executors[i] != executors[i+tt.expectedSize] {
					t.Error("expected round-robin behavior")
				}
			}
		})
	}
}

// Note: Platform-specific command helpers have been moved to test_helpers.go

// TestParallelExecute_ErrorHandlingPreservesOutput verifies that error information
// is properly preserved when command execution fails
func TestParallelExecute_ErrorHandlingPreservesOutput(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping slow test in short mode")
	}
	t.Parallel()
	cmdExecutor := NewCommandExecutor(10 * time.Second)
	pe := NewParallelExecutor(cmdExecutor, 2)

	// Create various test commands
	exit42Cmd, exit42Args := pc.exit(42)
	stderrCmd, stderrArgs := pc.stderr("Error message before failure")
	sleepCmd, sleepArgs := pc.sleep(5)

	tests := []struct {
		name          string
		command       ParallelCommand
		expectError   bool
		checkStderr   bool
		checkExitCode bool
	}{
		{
			name: "command validation failure preserves error details",
			command: ParallelCommand{
				ID:      "validation-fail",
				Command: "test-command",
				Args:    []string{";", "dangerous", "arg"},
				Options: ExecOptions{},
			},
			expectError:   true,
			checkStderr:   true,
			checkExitCode: true,
		},
		{
			name: "command with non-zero exit code",
			command: ParallelCommand{
				ID:      "exit-error",
				Command: exit42Cmd,
				Args:    exit42Args,
				Options: ExecOptions{},
			},
			expectError:   false, // Exit commands typically don't return Go errors
			checkExitCode: true,
		},
		{
			name: "command with stderr output before failure",
			command: ParallelCommand{
				ID:      "stderr-then-fail",
				Command: stderrCmd,
				Args:    stderrArgs,
				Options: ExecOptions{},
			},
			expectError: false,
			checkStderr: true,
		},
		{
			name: "command with timeout shows clear timeout message",
			command: ParallelCommand{
				ID:      "timeout-test",
				Command: sleepCmd,
				Args:    sleepArgs, // Sleep for 5 seconds
				Options: ExecOptions{
					Timeout: 100 * time.Millisecond, // But timeout after 100ms
				},
			},
			expectError:   true,
			checkStderr:   true,
			checkExitCode: true,
		},
		{
			name: "command with format specifiers in args handled safely",
			command: ParallelCommand{
				ID:      "format-specifier-test",
				Command: "test-command",
				Args:    []string{"%s", "%d", "%;rm -rf /", "%v"},
				Options: ExecOptions{},
			},
			expectError:   true,
			checkStderr:   true,
			checkExitCode: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			result, err := pe.Execute(ctx, []ParallelCommand{tt.command}, nil)

			if err != nil {
				t.Fatalf("unexpected Execute error: %v", err)
			}

			execResult, ok := result.Results[tt.command.ID]
			if !ok {
				t.Fatal("result not found for command")
			}

			// Verify error information is preserved
			if tt.expectError && execResult.Error == nil {
				t.Error("expected error to be preserved, but got nil")
			}

			// Verify stderr contains useful information
			if tt.checkStderr && execResult.Stderr == "" {
				t.Error("expected stderr to contain error information, but it was empty")
			}

			// Verify non-zero exit code for errors
			if tt.checkExitCode && execResult.ExitCode == 0 {
				t.Errorf("expected non-zero exit code for error, got %d", execResult.ExitCode)
			}

			// For validation failures, verify the command details are in error output
			if tt.command.ID == "validation-fail" {
				if execResult.Stderr != "" && !strings.Contains(execResult.Stderr, "test-command") {
					t.Errorf("expected stderr to contain command name 'test-command', got: %s",
						execResult.Stderr)
				}
			}

			// For timeout tests, verify timeout message and flag
			if tt.command.ID == "timeout-test" {
				if !execResult.TimedOut {
					t.Error("expected TimedOut flag to be true for timeout test")
				}
				if execResult.Stderr != "" && !strings.Contains(execResult.Stderr, "timed out") {
					t.Errorf("expected stderr to contain 'timed out', got: %s", execResult.Stderr)
				}
				if execResult.Stderr != "" && !strings.Contains(execResult.Stderr, "100ms") {
					t.Errorf("expected stderr to contain timeout duration '100ms', got: %s", execResult.Stderr)
				}
			}

			// For format specifier test, verify args are properly escaped
			if tt.command.ID == "format-specifier-test" {
				// The error message should contain the literal format specifiers, not interpreted
				if execResult.Stderr != "" {
					if !strings.Contains(execResult.Stderr, "%s") || !strings.Contains(execResult.Stderr, "%d") {
						t.Errorf("expected stderr to contain literal format specifiers, got: %s", execResult.Stderr)
					}
					// Verify the dangerous args are shown correctly
					if !strings.Contains(execResult.Stderr, "%s %d %;rm -rf / %v") {
						t.Errorf("expected stderr to show all args safely, got: %s", execResult.Stderr)
					}
				}
			}

			// Log the results for debugging
			t.Logf("Command: %s %v", tt.command.Command, tt.command.Args)
			t.Logf("ExitCode: %d", execResult.ExitCode)
			t.Logf("Er_ror: %v", execResult.Error)
			t.Logf("Stderr: %s", execResult.Stderr)
			t.Logf("Stdout: %s", execResult.Stdout)
		})
	}
}

// TestParallelExecute_PreservesPartialOutput verifies that partial output
// is preserved even when commands fail
func TestParallelExecute_PreservesPartialOutput(t *testing.T) {
	// This test would require a mock executor that can simulate
	// partial output before failure. For now, we document the expected behavior.
	t.Skip("Test requires mock executor implementation")
}
