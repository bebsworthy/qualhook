// Package executor provides test helper utilities for command execution testing.
package executor

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"
)

// TestCommandExecutor is a test version of CommandExecutor without security validation
type TestCommandExecutor struct {
	defaultTimeout time.Duration
}

// NewTestCommandExecutor creates a new test command executor
func NewTestCommandExecutor(defaultTimeout time.Duration) *TestCommandExecutor {
	if defaultTimeout <= 0 {
		defaultTimeout = 2 * time.Minute
	}
	return &TestCommandExecutor{
		defaultTimeout: defaultTimeout,
	}
}

// Execute runs a command without security validation (for testing only)
func (e *TestCommandExecutor) Execute(command string, args []string, options ExecOptions) (*ExecResult, error) {
	timeout := options.Timeout
	if timeout <= 0 {
		timeout = e.defaultTimeout
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, command, args...)
	
	if options.WorkingDir != "" {
		cmd.Dir = options.WorkingDir
	}

	// Set environment
	if options.InheritEnv {
		cmd.Env = os.Environ()
	}
	if len(options.Environment) > 0 {
		if cmd.Env == nil {
			cmd.Env = []string{}
		}
		cmd.Env = append(cmd.Env, options.Environment...)
	}

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	err := cmd.Start()
	if err != nil {
		return &ExecResult{
			ExitCode: -1,
			Error:    err,
		}, nil
	}

	waitErr := cmd.Wait()
	
	timedOut := false
	if ctx.Err() == context.DeadlineExceeded {
		timedOut = true
		_ = HandleTimeoutCleanup(cmd) //nolint:errcheck // Best effort cleanup in test
	}

	exitCode := 0
	if waitErr != nil {
		if exitErr, ok := waitErr.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
	}

	return &ExecResult{
		Stdout:   stdoutBuf.String(),
		Stderr:   stderrBuf.String(),
		ExitCode: exitCode,
		TimedOut: timedOut,
		Error:    waitErr,
	}, nil
}

// ExecuteWithStreaming runs a command and streams output (test version)
func (e *TestCommandExecutor) ExecuteWithStreaming(command string, args []string, options ExecOptions, stdoutWriter, stderrWriter io.Writer) (*ExecResult, error) {
	timeout := options.Timeout
	if timeout <= 0 {
		timeout = e.defaultTimeout
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, command, args...)
	
	if options.WorkingDir != "" {
		cmd.Dir = options.WorkingDir
	}

	// Set environment
	if options.InheritEnv {
		cmd.Env = os.Environ()
	}
	if len(options.Environment) > 0 {
		if cmd.Env == nil {
			cmd.Env = []string{}
		}
		cmd.Env = append(cmd.Env, options.Environment...)
	}

	var stdoutBuf, stderrBuf bytes.Buffer
	
	// Create streaming writers
	stdout := io.MultiWriter(&stdoutBuf, stdoutWriter)
	stderr := io.MultiWriter(&stderrBuf, stderrWriter)
	
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	err := cmd.Start()
	if err != nil {
		return &ExecResult{
			ExitCode: -1,
			Error:    err,
		}, nil
	}

	waitErr := cmd.Wait()
	
	timedOut := false
	if ctx.Err() == context.DeadlineExceeded {
		timedOut = true
		_ = HandleTimeoutCleanup(cmd) //nolint:errcheck // Best effort cleanup in test
	}

	exitCode := 0
	if waitErr != nil {
		if exitErr, ok := waitErr.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
	}

	return &ExecResult{
		Stdout:   stdoutBuf.String(),
		Stderr:   stderrBuf.String(),
		ExitCode: exitCode,
		TimedOut: timedOut,
		Error:    waitErr,
	}, nil
}

// TestParallelExecutor is a test version without security validation
type TestParallelExecutor struct {
	maxParallel int
}

// NewTestParallelExecutor creates a test parallel executor
func NewTestParallelExecutor(maxParallel int) *TestParallelExecutor {
	if maxParallel <= 0 {
		maxParallel = 4
	}
	return &TestParallelExecutor{
		maxParallel: maxParallel,
	}
}

// Execute runs multiple commands in parallel (test version)
func (pe *TestParallelExecutor) Execute(ctx context.Context, commands []ParallelCommand, progress ProgressCallback) (*ParallelResult, error) {
	if len(commands) == 0 {
		return &ParallelResult{
			Results: make(map[string]*ExecResult),
			Order:   []string{},
		}, nil
	}

	startTime := time.Now()
	
	result := &ParallelResult{
		Results:      make(map[string]*ExecResult),
		Order:        make([]string, 0, len(commands)),
		TotalTime:    0,
		HasFailures:  false,
		SuccessCount: 0,
		FailureCount: 0,
	}

	// Create executor
	executor := NewTestCommandExecutor(0)
	
	// Execute commands sequentially for simplicity in tests
	completed := 0
	total := len(commands)
	
	for _, cmd := range commands {
		if progress != nil {
			progress(completed, total, cmd.ID)
		}
		
		execResult, err := executor.Execute(cmd.Command, cmd.Args, cmd.Options)
		if err != nil {
			return nil, fmt.Errorf("failed to execute command %s: %w", cmd.ID, err)
		}
		
		result.Results[cmd.ID] = execResult
		result.Order = append(result.Order, cmd.ID)
		
		if execResult.Error != nil || execResult.ExitCode != 0 {
			result.HasFailures = true
			result.FailureCount++
		} else {
			result.SuccessCount++
		}
		
		completed++
		if progress != nil {
			progress(completed, total, cmd.ID)
		}
	}
	
	result.TotalTime = time.Since(startTime)
	
	return result, nil
}

// ExecuteWithAggregation runs commands and aggregates output (test version)
func (pe *TestParallelExecutor) ExecuteWithAggregation(ctx context.Context, commands []ParallelCommand, progress ProgressCallback) (*AggregatedResult, error) {
	parallelResult, err := pe.Execute(ctx, commands, progress)
	if err != nil {
		return nil, err
	}
	
	aggregated := &AggregatedResult{
		ParallelResult: parallelResult,
		CombinedStdout: make([]string, 0),
		CombinedStderr: make([]string, 0),
		FailedCommands: make([]string, 0),
	}
	
	for _, id := range parallelResult.Order {
		result := parallelResult.Results[id]
		if result.Stdout != "" {
			aggregated.CombinedStdout = append(aggregated.CombinedStdout, result.Stdout)
		}
		if result.Stderr != "" {
			aggregated.CombinedStderr = append(aggregated.CombinedStderr, result.Stderr)
		}
		if result.ExitCode != 0 || result.Error != nil {
			aggregated.FailedCommands = append(aggregated.FailedCommands, id)
		}
	}
	
	return aggregated, nil
}