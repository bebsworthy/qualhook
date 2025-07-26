//go:build unit

package executor

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"testing"
)

func TestExecError_Error(t *testing.T) {
	tests := []struct {
		name     string
		execErr  *ExecError
		expected string
	}{
		{
			name: "command not found",
			execErr: &ExecError{
				Type:    ErrorTypeCommandNotFound,
				Command: "nonexistent",
				Args:    []string{},
			},
			expected: "command not found: nonexistent",
		},
		{
			name: "permission denied with args",
			execErr: &ExecError{
				Type:    ErrorTypePermissionDenied,
				Command: "sudo",
				Args:    []string{"rm", "-rf", "/"},
			},
			expected: "permission denied: sudo rm -rf /",
		},
		{
			name: "timeout",
			execErr: &ExecError{
				Type:    ErrorTypeTimeout,
				Command: "sleep",
				Args:    []string{"100"},
			},
			expected: "command timed out: sleep 100",
		},
		{
			name: "working directory error",
			execErr: &ExecError{
				Type:    ErrorTypeWorkingDirectory,
				Command: "ls",
				Details: "no such directory",
			},
			expected: "working directory error: no such directory",
		},
		{
			name: "execution error",
			execErr: &ExecError{
				Type:    ErrorTypeExecution,
				Command: "false",
				Err:     errors.New("exit status 1"),
			},
			expected: "execution error for false: exit status 1",
		},
		{
			name: "unknown error",
			execErr: &ExecError{
				Type:    ErrorTypeUnknown,
				Command: "mystery",
				Err:     errors.New("something went wrong"),
			},
			expected: "unknown error for mystery: something went wrong",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.execErr.Error()
			if got != tt.expected {
				t.Errorf("Error() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestExecError_Is(t *testing.T) {
	tests := []struct {
		name     string
		execErr  *ExecError
		target   error
		expected bool
	}{
		{
			name: "matches command not found",
			execErr: &ExecError{
				Type: ErrorTypeCommandNotFound,
			},
			target:   ErrCommandNotFound,
			expected: true,
		},
		{
			name: "matches permission denied",
			execErr: &ExecError{
				Type: ErrorTypePermissionDenied,
			},
			target:   ErrPermissionDenied,
			expected: true,
		},
		{
			name: "matches timeout",
			execErr: &ExecError{
				Type: ErrorTypeTimeout,
			},
			target:   ErrTimeout,
			expected: true,
		},
		{
			name: "matches working directory",
			execErr: &ExecError{
				Type: ErrorTypeWorkingDirectory,
			},
			target:   ErrInvalidWorkingDirectory,
			expected: true,
		},
		{
			name: "does not match different error",
			execErr: &ExecError{
				Type: ErrorTypeCommandNotFound,
			},
			target:   ErrPermissionDenied,
			expected: false,
		},
		{
			name: "does not match random error",
			execErr: &ExecError{
				Type: ErrorTypeTimeout,
			},
			target:   errors.New("random error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := errors.Is(tt.execErr, tt.target)
			if got != tt.expected {
				t.Errorf("Is(%v) = %v, want %v", tt.target, got, tt.expected)
			}
		})
	}
}

func TestExecError_Unwrap(t *testing.T) {
	originalErr := errors.New("original error")
	execErr := &ExecError{
		Type: ErrorTypeUnknown,
		Err:  originalErr,
	}

	unwrapped := execErr.Unwrap()
	if unwrapped != originalErr {
		t.Errorf("Unwrap() = %v, want %v", unwrapped, originalErr)
	}
}

func TestClassifyError(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		expectedType ErrorType
	}{
		{
			name:         "nil error",
			err:          nil,
			expectedType: ErrorTypeUnknown,
		},
		{
			name:         "context deadline exceeded",
			err:          context.DeadlineExceeded,
			expectedType: ErrorTypeTimeout,
		},
		{
			name: "exec.Error - file not found",
			err: &exec.Error{
				Name: "nonexistent",
				Err:  errors.New("executable file not found in $PATH"),
			},
			expectedType: ErrorTypeCommandNotFound,
		},
		{
			name: "exec.Error - permission denied",
			err: &exec.Error{
				Name: "restricted",
				Err:  errors.New("permission denied"),
			},
			expectedType: ErrorTypePermissionDenied,
		},
		{
			name:         "exit error",
			err:          &exec.ExitError{},
			expectedType: ErrorTypeExecution,
		},
		{
			name:         "string contains permission denied",
			err:          errors.New("failed: permission denied"),
			expectedType: ErrorTypePermissionDenied,
		},
		{
			name:         "string contains not found",
			err:          errors.New("command not found"),
			expectedType: ErrorTypeCommandNotFound,
		},
		{
			name:         "string contains timeout",
			err:          errors.New("operation timeout"),
			expectedType: ErrorTypeTimeout,
		},
		{
			name:         "string contains working directory",
			err:          errors.New("chdir failed"),
			expectedType: ErrorTypeWorkingDirectory,
		},
		{
			name:         "unknown error",
			err:          errors.New("something else"),
			expectedType: ErrorTypeExecution,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ClassifyError(tt.err, "test-cmd", []string{"arg1", "arg2"})

			if tt.err == nil {
				if result != nil {
					t.Errorf("expected nil for nil error, got %v", result)
				}
				return
			}

			if result == nil {
				t.Fatal("expected non-nil result")
			}

			if result.Type != tt.expectedType {
				t.Errorf("Type = %v, want %v", result.Type, tt.expectedType)
			}

			if result.Command != "test-cmd" {
				t.Errorf("Command = %q, want %q", result.Command, "test-cmd")
			}

			if len(result.Args) != 2 || result.Args[0] != "arg1" || result.Args[1] != "arg2" {
				t.Errorf("Args = %v, want [arg1 arg2]", result.Args)
			}
		})
	}
}

func TestHandleTimeoutCleanup(t *testing.T) {
	tests := []struct {
		name    string
		cmd     *exec.Cmd
		wantErr bool
	}{
		{
			name:    "nil command",
			cmd:     nil,
			wantErr: false,
		},
		{
			name:    "command with nil process",
			cmd:     &exec.Cmd{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := HandleTimeoutCleanup(tt.cmd)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleTimeoutCleanup() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestClassifyError_RealExecErrors tests with real exec errors
func TestClassifyError_RealExecErrors(t *testing.T) {
	// Try to execute a non-existent command
	cmd := exec.Command("this-command-definitely-does-not-exist-12345")
	err := cmd.Run()

	if err != nil {
		result := ClassifyError(err, cmd.Path, cmd.Args)
		if result.Type != ErrorTypeCommandNotFound {
			t.Errorf("expected command not found error, got %v", result.Type)
		}
	}
}

// TestErrorMessages verifies error message formatting
func TestErrorMessages(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		message string
	}{
		{
			name:    "ErrCommandNotFound",
			err:     ErrCommandNotFound,
			message: "command not found",
		},
		{
			name:    "ErrPermissionDenied",
			err:     ErrPermissionDenied,
			message: "permission denied",
		},
		{
			name:    "ErrTimeout",
			err:     ErrTimeout,
			message: "command timed out",
		},
		{
			name:    "ErrInvalidWorkingDirectory",
			err:     ErrInvalidWorkingDirectory,
			message: "invalid working directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !strings.Contains(tt.err.Error(), tt.message) {
				t.Errorf("error message %q does not contain %q", tt.err.Error(), tt.message)
			}
		})
	}
}
