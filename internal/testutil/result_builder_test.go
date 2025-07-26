package testutil

import (
	"errors"
	"testing"

	"github.com/bebsworthy/qualhook/internal/executor"
)

func TestResultBuilder(t *testing.T) {
	t.Run("default values", func(t *testing.T) {
		result := NewResultBuilder().Build()
		
		if result.Stdout != "" {
			t.Errorf("Expected empty stdout, got %q", result.Stdout)
		}
		if result.Stderr != "" {
			t.Errorf("Expected empty stderr, got %q", result.Stderr)
		}
		if result.ExitCode != 0 {
			t.Errorf("Expected exit code 0, got %d", result.ExitCode)
		}
		if result.TimedOut {
			t.Errorf("Expected TimedOut to be false")
		}
		if result.Error != nil {
			t.Errorf("Expected nil error, got %v", result.Error)
		}
	})

	t.Run("fluent interface", func(t *testing.T) {
		result := NewResultBuilder().
			WithStdout("output").
			WithStderr("error output").
			WithExitCode(1).
			WithError(errors.New("test error")).
			Build()

		if result.Stdout != "output" {
			t.Errorf("Expected stdout 'output', got %q", result.Stdout)
		}
		if result.Stderr != "error output" {
			t.Errorf("Expected stderr 'error output', got %q", result.Stderr)
		}
		if result.ExitCode != 1 {
			t.Errorf("Expected exit code 1, got %d", result.ExitCode)
		}
		if result.Error == nil || result.Error.Error() != "test error" {
			t.Errorf("Expected error 'test error', got %v", result.Error)
		}
	})

	t.Run("with error message", func(t *testing.T) {
		result := NewResultBuilder().WithErrorMessage("custom error").Build()
		
		if result.Error == nil || result.Error.Error() != "custom error" {
			t.Errorf("Expected error 'custom error', got %v", result.Error)
		}
	})

	t.Run("timed out", func(t *testing.T) {
		result := NewResultBuilder().TimedOut().Build()
		
		if !result.TimedOut {
			t.Errorf("Expected TimedOut to be true")
		}
		if result.ExitCode != -1 {
			t.Errorf("Expected exit code -1 for timeout, got %d", result.ExitCode)
		}
		if result.Error == nil || result.Error.Error() != "command timed out" {
			t.Errorf("Expected timeout error, got %v", result.Error)
		}
	})

	t.Run("timed out with existing error", func(t *testing.T) {
		result := NewResultBuilder().
			WithError(errors.New("existing error")).
			TimedOut().
			Build()
		
		if !result.TimedOut {
			t.Errorf("Expected TimedOut to be true")
		}
		if result.Error == nil || result.Error.Error() != "existing error" {
			t.Errorf("Expected existing error to be preserved, got %v", result.Error)
		}
	})

	t.Run("success", func(t *testing.T) {
		result := NewResultBuilder().Success().Build()
		
		if result.ExitCode != 0 {
			t.Errorf("Expected exit code 0, got %d", result.ExitCode)
		}
		if result.TimedOut {
			t.Errorf("Expected TimedOut to be false")
		}
		if result.Error != nil {
			t.Errorf("Expected nil error, got %v", result.Error)
		}
	})

	t.Run("success with stdout", func(t *testing.T) {
		result := NewResultBuilder().Success("success output").Build()
		
		if result.Stdout != "success output" {
			t.Errorf("Expected stdout 'success output', got %q", result.Stdout)
		}
		if result.ExitCode != 0 {
			t.Errorf("Expected exit code 0, got %d", result.ExitCode)
		}
	})

	t.Run("failure", func(t *testing.T) {
		result := NewResultBuilder().Failure().Build()
		
		if result.ExitCode != 1 {
			t.Errorf("Expected exit code 1, got %d", result.ExitCode)
		}
		if result.TimedOut {
			t.Errorf("Expected TimedOut to be false")
		}
	})

	t.Run("failure with stderr", func(t *testing.T) {
		result := NewResultBuilder().Failure("error message").Build()
		
		if result.Stderr != "error message" {
			t.Errorf("Expected stderr 'error message', got %q", result.Stderr)
		}
		if result.ExitCode != 1 {
			t.Errorf("Expected exit code 1, got %d", result.ExitCode)
		}
	})

	t.Run("failure with code", func(t *testing.T) {
		result := NewResultBuilder().FailureWithCode(127, "command not found").Build()
		
		if result.ExitCode != 127 {
			t.Errorf("Expected exit code 127, got %d", result.ExitCode)
		}
		if result.Stderr != "command not found" {
			t.Errorf("Expected stderr 'command not found', got %q", result.Stderr)
		}
	})

	t.Run("assert equal - matching", func(t *testing.T) {
		builder := NewResultBuilder().
			WithStdout("output").
			WithStderr("error").
			WithExitCode(1)
		
		actual := executor.ExecResult{
			Stdout:   "output",
			Stderr:   "error",
			ExitCode: 1,
			TimedOut: false,
			Error:    nil,
		}
		
		if err := builder.AssertEqual(actual); err != nil {
			t.Errorf("Expected results to match, got error: %v", err)
		}
	})

	t.Run("assert equal - stdout mismatch", func(t *testing.T) {
		builder := NewResultBuilder().WithStdout("expected")
		actual := executor.ExecResult{Stdout: "actual"}
		
		err := builder.AssertEqual(actual)
		if err == nil {
			t.Errorf("Expected error for stdout mismatch")
		}
		if err.Error() != `result mismatch:
  stdout: expected "expected", got "actual"` {
			t.Errorf("Unexpected error message: %v", err)
		}
	})

	t.Run("assert equal - exit code mismatch", func(t *testing.T) {
		builder := NewResultBuilder().WithExitCode(0)
		actual := executor.ExecResult{ExitCode: 1}
		
		err := builder.AssertEqual(actual)
		if err == nil {
			t.Errorf("Expected error for exit code mismatch")
		}
		if err.Error() != `result mismatch:
  exit code: expected 0, got 1` {
			t.Errorf("Unexpected error message: %v", err)
		}
	})

	t.Run("assert equal - error mismatch", func(t *testing.T) {
		builder := NewResultBuilder().WithError(errors.New("expected error"))
		actual := executor.ExecResult{Error: nil}
		
		err := builder.AssertEqual(actual)
		if err == nil {
			t.Errorf("Expected error for error mismatch")
		}
	})

	t.Run("must equal - matching", func(t *testing.T) {
		expected := NewResultBuilder().Success("output")
		actual := executor.ExecResult{
			Stdout:   "output",
			ExitCode: 0,
		}
		
		// This should not panic
		expected.MustEqual(t, actual)
	})

	t.Run("must equal - mismatch", func(t *testing.T) {
		expected := NewResultBuilder().Success("expected")
		actual := executor.ExecResult{
			Stdout:   "actual",
			ExitCode: 0,
		}
		
		// Create a mock test context to capture the failure
		mockT := &mockTestContext{}
		expected.MustEqual(mockT, actual)
		
		if !mockT.failed {
			t.Errorf("Expected MustEqual to call t.Fatalf on mismatch")
		}
	})
}

// mockTestContext implements testing.TB for testing
type mockTestContext struct {
	testing.TB
	failed bool
	message string
}

func (m *mockTestContext) Helper() {}

func (m *mockTestContext) Fatalf(format string, args ...interface{}) {
	m.failed = true
	m.message = format
}