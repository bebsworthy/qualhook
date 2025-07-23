package executor

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewParallelExecutor(t *testing.T) {
	cmdExecutor := NewCommandExecutor(10 * time.Second)

	tests := []struct {
		name            string
		maxParallel     int
		expectedParallel int
	}{
		{
			name:            "positive value",
			maxParallel:     8,
			expectedParallel: 8,
		},
		{
			name:            "zero value defaults to 4",
			maxParallel:     0,
			expectedParallel: 4,
		},
		{
			name:            "negative value defaults to 4",
			maxParallel:     -1,
			expectedParallel: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pe := NewParallelExecutor(cmdExecutor, tt.maxParallel)
			if pe.maxParallel != tt.expectedParallel {
				t.Errorf("expected maxParallel %d, got %d", tt.expectedParallel, pe.maxParallel)
			}
		})
	}
}

func TestParallelExecute_Basic(t *testing.T) {
	cmdExecutor := NewCommandExecutor(10 * time.Second)
	pe := NewParallelExecutor(cmdExecutor, 4)

	// Create test commands
	var commands []ParallelCommand
	for i := 0; i < 3; i++ {
		var cmd string
		var args []string
		if runtime.GOOS == osWindows {
			cmd = cmdCommand
			args = []string{cmdArgC, echoCommand, fmt.Sprintf("test%d", i)}
		} else {
			cmd = echoCommand
			args = []string{fmt.Sprintf("test%d", i)}
		}
		
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
	cmdExecutor := NewCommandExecutor(10 * time.Second)
	pe := NewParallelExecutor(cmdExecutor, 2)

	// Create test commands
	var commands []ParallelCommand
	for i := 0; i < 4; i++ {
		var cmd string
		var args []string
		if runtime.GOOS == osWindows {
			cmd = cmdCommand
			args = []string{cmdArgC, echoCommand, "test"}
		} else {
			cmd = echoCommand
			args = []string{"test"}
		}
		
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
	cmdExecutor := NewCommandExecutor(10 * time.Second)
	pe := NewParallelExecutor(cmdExecutor, 4)

	// Mix of successful and failing commands
	commands := []ParallelCommand{
		{
			ID:      "success",
			Command: getEchoCommand(),
			Args:    getEchoArgs("success"),
			Options: ExecOptions{},
		},
		{
			ID:      "fail",
			Command: getExitCommand(),
			Args:    getExitArgs(1),
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
	cmdExecutor := NewCommandExecutor(10 * time.Second)
	pe := NewParallelExecutor(cmdExecutor, 2)

	// Create slow commands
	var commands []ParallelCommand
	for i := 0; i < 4; i++ {
		commands = append(commands, ParallelCommand{
			ID:      fmt.Sprintf("cmd-%d", i),
			Command: getSleepCommand(),
			Args:    getSleepArgs(2), // 2 second sleep
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
	pe := NewTestParallelExecutor(4)

	// Create commands with different outputs
	commands := []ParallelCommand{
		{
			ID:      "stdout-only",
			Command: getEchoCommand(),
			Args:    getEchoArgs("stdout message"),
			Options: ExecOptions{},
		},
		{
			ID:      "stderr-only",
			Command: getStderrCommand(),
			Args:    getStderrArgs("stderr message"),
			Options: ExecOptions{},
		},
		{
			ID:      "both",
			Command: getBothOutputCommand(),
			Args:    getBothOutputArgs("out", "err"),
			Options: ExecOptions{},
		},
		{
			ID:      "failure",
			Command: getExitCommand(),
			Args:    getExitArgs(1),
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

	// Check aggregated stderr
	if len(result.CombinedStderr) < 1 {
		t.Errorf("expected at least 1 stderr entry, got %d", len(result.CombinedStderr))
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

	if !strings.Contains(summary, "Failed commands (1/4)") {
		t.Errorf("unexpected failure summary format: %s", summary)
	}
}

func TestGetFailureSummary(t *testing.T) {
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

// Helper functions for cross-platform commands
func getEchoCommand() string {
	if runtime.GOOS == osWindows {
		return cmdCommand
	}
	return echoCommand
}

func getEchoArgs(message string) []string {
	if runtime.GOOS == osWindows {
		return []string{cmdArgC, echoCommand, message}
	}
	return []string{message}
}

func getExitCommand() string {
	if runtime.GOOS == osWindows {
		return "cmd"
	}
	return "sh"
}

func getExitArgs(code int) []string {
	if runtime.GOOS == osWindows {
		return []string{"/c", "exit", fmt.Sprintf("%d", code)}
	}
	return []string{"-c", fmt.Sprintf("exit %d", code)}
}

func getSleepCommand() string {
	if runtime.GOOS == osWindows {
		return "cmd"
	}
	return "sleep"
}

func getSleepArgs(seconds int) []string {
	if runtime.GOOS == osWindows {
		return []string{"/c", "timeout", "/t", fmt.Sprintf("%d", seconds), "/nobreak"}
	}
	return []string{fmt.Sprintf("%d", seconds)}
}

func getStderrCommand() string {
	if runtime.GOOS == osWindows {
		return "cmd"
	}
	return "sh"
}

func getStderrArgs(message string) []string {
	if runtime.GOOS == osWindows {
		return []string{"/c", "echo", message, "1>&2"}
	}
	return []string{"-c", fmt.Sprintf("echo '%s' >&2", message)}
}

func getBothOutputCommand() string {
	if runtime.GOOS == osWindows {
		return "cmd"
	}
	return "sh"
}

func getBothOutputArgs(stdout, stderr string) []string {
	if runtime.GOOS == osWindows {
		return []string{"/c", fmt.Sprintf("echo %s && echo %s 1>&2", stdout, stderr)}
	}
	return []string{"-c", fmt.Sprintf("echo '%s' && echo '%s' >&2", stdout, stderr)}
}