// Package executor provides command execution functionality for qualhook.
package executor

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// ExecOptions defines options for command execution
type ExecOptions struct {
	// Working directory for the command
	WorkingDir string
	// Environment variables (in KEY=VALUE format)
	Environment []string
	// Timeout for command execution
	Timeout time.Duration
	// Whether to inherit parent process environment
	InheritEnv bool
}

// ExecResult contains the result of command execution
type ExecResult struct {
	// Standard output from the command
	Stdout string
	// Standard error from the command
	Stderr string
	// Exit code of the command
	ExitCode int
	// Whether the command timed out
	TimedOut bool
	// Error if command failed to start
	Error error
}

// CommandExecutor executes external commands safely
type CommandExecutor struct {
	// Default timeout for commands if not specified
	defaultTimeout time.Duration
}

// NewCommandExecutor creates a new command executor
func NewCommandExecutor(defaultTimeout time.Duration) *CommandExecutor {
	if defaultTimeout <= 0 {
		defaultTimeout = 2 * time.Minute
	}
	return &CommandExecutor{
		defaultTimeout: defaultTimeout,
	}
}

// Execute runs a command with the given options
func (e *CommandExecutor) Execute(command string, args []string, options ExecOptions) (*ExecResult, error) {
	// Validate command
	if command == "" {
		return nil, fmt.Errorf("command cannot be empty")
	}

	// Set default timeout if not specified
	timeout := options.Timeout
	if timeout <= 0 {
		timeout = e.defaultTimeout
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Create command
	cmd := exec.CommandContext(ctx, command, args...)

	// Set working directory
	if options.WorkingDir != "" {
		absPath, err := filepath.Abs(options.WorkingDir)
		if err != nil {
			return nil, fmt.Errorf("invalid working directory: %w", err)
		}
		// Check if directory exists
		if _, err := os.Stat(absPath); err != nil {
			if os.IsNotExist(err) {
				return nil, fmt.Errorf("invalid working directory: %s does not exist", absPath)
			}
			return nil, fmt.Errorf("invalid working directory: %w", err)
		}
		cmd.Dir = absPath
	}

	// Set environment
	env := e.prepareEnvironment(options)
	if len(env) > 0 {
		cmd.Env = env
	}

	// Capture output
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	// Start the command
	err := cmd.Start()
	if err != nil {
		// Classify the error
		execErr := ClassifyError(err, command, args)
		return &ExecResult{
			ExitCode: -1,
			Error:    execErr,
		}, nil
	}

	// Wait for command to complete
	waitErr := cmd.Wait()

	// Check if context was cancelled (timeout)
	timedOut := false
	if ctx.Err() == context.DeadlineExceeded {
		timedOut = true
		// Ensure process is cleaned up after timeout
		_ = HandleTimeoutCleanup(cmd)
	}

	// Get exit code
	exitCode := 0
	if waitErr != nil {
		if exitErr, ok := waitErr.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			// Command failed to run properly
			return &ExecResult{
				Stdout:   stdoutBuf.String(),
				Stderr:   stderrBuf.String(),
				ExitCode: -1,
				TimedOut: timedOut,
				Error:    waitErr,
			}, nil
		}
	}

	return &ExecResult{
		Stdout:   stdoutBuf.String(),
		Stderr:   stderrBuf.String(),
		ExitCode: exitCode,
		TimedOut: timedOut,
	}, nil
}

// ExecuteWithStreaming runs a command and streams output to the provided writers
func (e *CommandExecutor) ExecuteWithStreaming(command string, args []string, options ExecOptions, stdoutWriter, stderrWriter io.Writer) (*ExecResult, error) {
	// Validate command
	if command == "" {
		return nil, fmt.Errorf("command cannot be empty")
	}

	// Set default timeout if not specified
	timeout := options.Timeout
	if timeout <= 0 {
		timeout = e.defaultTimeout
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Create command
	cmd := exec.CommandContext(ctx, command, args...)

	// Set working directory
	if options.WorkingDir != "" {
		absPath, err := filepath.Abs(options.WorkingDir)
		if err != nil {
			return nil, fmt.Errorf("invalid working directory: %w", err)
		}
		// Check if directory exists
		if _, err := os.Stat(absPath); err != nil {
			if os.IsNotExist(err) {
				return nil, fmt.Errorf("invalid working directory: %s does not exist", absPath)
			}
			return nil, fmt.Errorf("invalid working directory: %w", err)
		}
		cmd.Dir = absPath
	}

	// Set environment
	env := e.prepareEnvironment(options)
	if len(env) > 0 {
		cmd.Env = env
	}

	// Create buffers to capture output while also streaming
	var stdoutBuf, stderrBuf bytes.Buffer

	// Create multi-writers to both stream and capture
	if stdoutWriter != nil {
		cmd.Stdout = io.MultiWriter(stdoutWriter, &stdoutBuf)
	} else {
		cmd.Stdout = &stdoutBuf
	}

	if stderrWriter != nil {
		cmd.Stderr = io.MultiWriter(stderrWriter, &stderrBuf)
	} else {
		cmd.Stderr = &stderrBuf
	}

	// Start the command
	err := cmd.Start()
	if err != nil {
		// Classify the error
		execErr := ClassifyError(err, command, args)
		return &ExecResult{
			ExitCode: -1,
			Error:    execErr,
		}, nil
	}

	// Wait for command to complete
	waitErr := cmd.Wait()

	// Check if context was cancelled (timeout)
	timedOut := false
	if ctx.Err() == context.DeadlineExceeded {
		timedOut = true
		// Ensure process is cleaned up after timeout
		_ = HandleTimeoutCleanup(cmd)
	}

	// Get exit code
	exitCode := 0
	if waitErr != nil {
		if exitErr, ok := waitErr.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			// Command failed to run properly
			return &ExecResult{
				Stdout:   stdoutBuf.String(),
				Stderr:   stderrBuf.String(),
				ExitCode: -1,
				TimedOut: timedOut,
				Error:    waitErr,
			}, nil
		}
	}

	return &ExecResult{
		Stdout:   stdoutBuf.String(),
		Stderr:   stderrBuf.String(),
		ExitCode: exitCode,
		TimedOut: timedOut,
	}, nil
}

// prepareEnvironment prepares the environment variables for the command
func (e *CommandExecutor) prepareEnvironment(options ExecOptions) []string {
	var env []string

	// Start with parent environment if requested
	if options.InheritEnv {
		env = os.Environ()
	}

	// Create a map for efficient lookup and override
	envMap := make(map[string]string)
	for _, e := range env {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}

	// Add/override with provided environment variables
	for _, e := range options.Environment {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}

	// Convert back to slice
	env = make([]string, 0, len(envMap))
	for k, v := range envMap {
		env = append(env, k+"="+v)
	}

	return env
}

// StreamingWriter is a thread-safe writer that can be used for streaming output
type StreamingWriter struct {
	mu     sync.Mutex
	writer io.Writer
}

// NewStreamingWriter creates a new streaming writer
func NewStreamingWriter(w io.Writer) *StreamingWriter {
	return &StreamingWriter{writer: w}
}

// Write implements io.Writer interface
func (sw *StreamingWriter) Write(p []byte) (n int, err error) {
	sw.mu.Lock()
	defer sw.mu.Unlock()
	return sw.writer.Write(p)
}