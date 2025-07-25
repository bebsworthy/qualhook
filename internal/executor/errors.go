// Package executor provides command execution functionality for qualhook.
package executor

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// Error types for command execution
var (
	// ErrCommandNotFound indicates the command was not found in PATH
	ErrCommandNotFound = errors.New("command not found")

	// ErrPermissionDenied indicates the command cannot be executed due to permissions
	ErrPermissionDenied = errors.New("permission denied")

	// ErrTimeout indicates the command timed out
	ErrTimeout = errors.New("command timed out")

	// ErrInvalidWorkingDirectory indicates the working directory is invalid
	ErrInvalidWorkingDirectory = errors.New("invalid working directory")
)

// ErrorType represents the type of execution error
type ErrorType int

const (
	// ErrorTypeUnknown indicates an unknown error
	ErrorTypeUnknown ErrorType = iota
	// ErrorTypeCommandNotFound indicates the command was not found
	ErrorTypeCommandNotFound
	// ErrorTypePermissionDenied indicates permission was denied
	ErrorTypePermissionDenied
	// ErrorTypeTimeout indicates the command timed out
	ErrorTypeTimeout
	// ErrorTypeWorkingDirectory indicates working directory error
	ErrorTypeWorkingDirectory
	// ErrorTypeExecution indicates general execution error
	ErrorTypeExecution
)

// ExecError represents a detailed execution error
type ExecError struct {
	Type    ErrorType
	Command string
	Args    []string
	Err     error
	Details string
}

// Error implements the error interface
func (e *ExecError) Error() string {
	cmd := e.Command
	if len(e.Args) > 0 {
		cmd = fmt.Sprintf("%s %s", e.Command, strings.Join(e.Args, " "))
	}

	switch e.Type {
	case ErrorTypeCommandNotFound:
		return fmt.Sprintf("command not found: %s", e.Command)
	case ErrorTypePermissionDenied:
		return fmt.Sprintf("permission denied: %s", cmd)
	case ErrorTypeTimeout:
		return fmt.Sprintf("command timed out: %s", cmd)
	case ErrorTypeWorkingDirectory:
		return fmt.Sprintf("working directory error: %s", e.Details)
	case ErrorTypeExecution:
		return fmt.Sprintf("execution error for %s: %v", cmd, e.Err)
	default:
		return fmt.Sprintf("unknown error for %s: %v", cmd, e.Err)
	}
}

// Unwrap returns the underlying error
func (e *ExecError) Unwrap() error {
	return e.Err
}

// Is implements errors.Is support
func (e *ExecError) Is(target error) bool {
	switch target {
	case ErrCommandNotFound:
		return e.Type == ErrorTypeCommandNotFound
	case ErrPermissionDenied:
		return e.Type == ErrorTypePermissionDenied
	case ErrTimeout:
		return e.Type == ErrorTypeTimeout
	case ErrInvalidWorkingDirectory:
		return e.Type == ErrorTypeWorkingDirectory
	}
	return false
}

// ClassifyError analyzes an error and returns a typed ExecError
func ClassifyError(err error, command string, args []string) *ExecError {
	if err == nil {
		return nil
	}

	execErr := &ExecError{
		Type:    ErrorTypeUnknown,
		Command: command,
		Args:    args,
		Err:     err,
	}

	// Check for timeout
	if errors.Is(err, context.DeadlineExceeded) {
		execErr.Type = ErrorTypeTimeout
		return execErr
	}

	// Check for exec.Error which indicates command not found or permission issues
	if errType := classifyExecError(err); errType != ErrorTypeUnknown {
		execErr.Type = errType
		return execErr
	}

	// Check for exit error (command ran but returned non-zero)
	var exitError *exec.ExitError
	if errors.As(err, &exitError) {
		// This is a normal non-zero exit, not an execution error
		execErr.Type = ErrorTypeExecution
		return execErr
	}

	// Check error message for common patterns
	execErr.Type = classifyByErrorMessage(err.Error())
	if execErr.Type == ErrorTypeWorkingDirectory {
		execErr.Details = err.Error()
	}

	return execErr
}

// HandleTimeoutCleanup performs cleanup after a timeout occurs
func HandleTimeoutCleanup(cmd *exec.Cmd) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}

	// Try to kill the process
	if err := cmd.Process.Kill(); err != nil {
		// Process might have already exited
		if !strings.Contains(err.Error(), "process already finished") {
			return fmt.Errorf("failed to kill timed out process: %w", err)
		}
	}

	// Wait for the process to actually exit
	// This prevents zombie processes
	_, _ = cmd.Process.Wait() //nolint:errcheck // Best effort wait to prevent zombies

	return nil
}

// classifyExecError classifies exec.Error types
func classifyExecError(err error) ErrorType {
	var execError *exec.Error
	if !errors.As(err, &execError) {
		return ErrorTypeUnknown
	}

	errStr := strings.ToLower(execError.Error())

	if strings.Contains(errStr, "executable file not found") ||
		strings.Contains(errStr, "command not found") ||
		strings.Contains(errStr, "no such file or directory") {
		return ErrorTypeCommandNotFound
	}

	if strings.Contains(errStr, "permission denied") ||
		strings.Contains(errStr, "operation not permitted") {
		return ErrorTypePermissionDenied
	}

	return ErrorTypeUnknown
}

// classifyByErrorMessage classifies errors by their message content
func classifyByErrorMessage(errorMessage string) ErrorType {
	errStr := strings.ToLower(errorMessage)

	switch {
	case strings.Contains(errStr, "permission denied"):
		return ErrorTypePermissionDenied
	case strings.Contains(errStr, "not found"):
		return ErrorTypeCommandNotFound
	case strings.Contains(errStr, "timeout") || strings.Contains(errStr, "deadline exceeded"):
		return ErrorTypeTimeout
	case strings.Contains(errStr, "working directory") || strings.Contains(errStr, "chdir"):
		return ErrorTypeWorkingDirectory
	default:
		return ErrorTypeExecution
	}
}
