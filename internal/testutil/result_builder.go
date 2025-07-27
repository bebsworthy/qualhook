package testutil

import (
	"errors"
	"fmt"
	"testing"

	"github.com/bebsworthy/qualhook/internal/executor"
)

// ResultBuilder provides a fluent interface for building expected test results.
type ResultBuilder struct {
	result executor.ExecResult
}

// NewResultBuilder creates a new ResultBuilder with default values.
func NewResultBuilder() *ResultBuilder {
	return &ResultBuilder{
		result: executor.ExecResult{
			ExitCode: 0,
			TimedOut: false,
		},
	}
}

// WithStdout sets the expected stdout output.
func (b *ResultBuilder) WithStdout(stdout string) *ResultBuilder {
	b.result.Stdout = stdout
	return b
}

// WithStderr sets the expected stderr output.
func (b *ResultBuilder) WithStderr(stderr string) *ResultBuilder {
	b.result.Stderr = stderr
	return b
}

// WithExitCode sets the expected exit code.
func (b *ResultBuilder) WithExitCode(exitCode int) *ResultBuilder {
	b.result.ExitCode = exitCode
	return b
}

// WithError sets an error for the result.
func (b *ResultBuilder) WithError(err error) *ResultBuilder {
	b.result.Error = err
	return b
}

// WithErrorMessage creates an error with the given message.
func (b *ResultBuilder) WithErrorMessage(message string) *ResultBuilder {
	b.result.Error = errors.New(message)
	return b
}

// TimedOut marks the result as having timed out.
func (b *ResultBuilder) TimedOut() *ResultBuilder {
	b.result.TimedOut = true
	b.result.ExitCode = -1
	if b.result.Error == nil {
		b.result.Error = errors.New("command timed out")
	}
	return b
}

// Success creates a successful result with optional stdout.
func (b *ResultBuilder) Success(stdout ...string) *ResultBuilder {
	b.result.ExitCode = 0
	b.result.TimedOut = false
	b.result.Error = nil
	if len(stdout) > 0 {
		b.result.Stdout = stdout[0]
	}
	return b
}

// Failure creates a failed result with exit code 1.
func (b *ResultBuilder) Failure(stderr ...string) *ResultBuilder {
	b.result.ExitCode = 1
	b.result.TimedOut = false
	if len(stderr) > 0 {
		b.result.Stderr = stderr[0]
	}
	return b
}

// FailureWithCode creates a failed result with a specific exit code.
func (b *ResultBuilder) FailureWithCode(exitCode int, stderr ...string) *ResultBuilder {
	b.result.ExitCode = exitCode
	b.result.TimedOut = false
	if len(stderr) > 0 {
		b.result.Stderr = stderr[0]
	}
	return b
}

// Build returns the constructed ExecResult.
func (b *ResultBuilder) Build() executor.ExecResult {
	return b.result
}

// AssertEqual checks if the actual result matches the expected result.
// Returns nil if they match, or an error describing the differences.
func (b *ResultBuilder) AssertEqual(actual executor.ExecResult) error {
	expected := b.result

	var diffs []string

	if expected.Stdout != actual.Stdout {
		diffs = append(diffs, fmt.Sprintf("stdout: expected %q, got %q", expected.Stdout, actual.Stdout))
	}

	if expected.Stderr != actual.Stderr {
		diffs = append(diffs, fmt.Sprintf("stderr: expected %q, got %q", expected.Stderr, actual.Stderr))
	}

	if expected.ExitCode != actual.ExitCode {
		diffs = append(diffs, fmt.Sprintf("exit code: expected %d, got %d", expected.ExitCode, actual.ExitCode))
	}

	if expected.TimedOut != actual.TimedOut {
		diffs = append(diffs, fmt.Sprintf("timed out: expected %v, got %v", expected.TimedOut, actual.TimedOut))
	}

	// For errors, we only check if both are nil or both are non-nil
	expectedHasError := expected.Error != nil
	actualHasError := actual.Error != nil
	if expectedHasError != actualHasError {
		if expectedHasError {
			diffs = append(diffs, fmt.Sprintf("error: expected error %q, got nil", expected.Error))
		} else {
			diffs = append(diffs, fmt.Sprintf("error: expected nil, got %q", actual.Error))
		}
	}

	if len(diffs) > 0 {
		return fmt.Errorf("result mismatch:\n  %s", diffs[0])
	}

	return nil
}

// MustEqual is like AssertEqual but calls t.Fatalf if the results don't match.
func (b *ResultBuilder) MustEqual(t testing.TB, actual executor.ExecResult) {
	t.Helper()
	if err := b.AssertEqual(actual); err != nil {
		t.Fatalf("Result assertion failed: %v", err)
	}
}
