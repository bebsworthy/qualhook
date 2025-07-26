// Package testutil provides common test utilities and helpers for the qualhook test suite.
//
// The package includes three main components:
//
// ConfigBuilder: A fluent interface for building test configurations
//   - Create configurations with NewConfigBuilder()
//   - Add commands with WithCommand() or WithSimpleCommand()
//   - Add path-specific configs with WithPath()
//   - Use DefaultTestConfig() for a basic test configuration
//
// OutputCapture: Utilities for capturing stdout and stderr
//   - Use CaptureOutput() to capture both stdout and stderr
//   - Use CaptureStdout() or CaptureStderr() for specific streams
//   - TestWriter provides a thread-safe io.Writer for tests
//
// CommandHelpers: Safe commands and utilities for cross-platform testing
//   - SafeCommands provides platform-agnostic command names
//   - SafeTestCommand() returns a simple echo command
//   - FailingTestCommand() and SuccessfulTestCommand() for exit code testing
//   - RunCommand() executes commands with proper error handling
//   - Platform detection and skip helpers
//
// Example usage:
//
//	// Create a test configuration
//	cfg := testutil.NewConfigBuilder().
//		WithSimpleCommand("lint", "eslint", ".").
//		WithSimpleCommand("test", "jest").
//		Build()
//
//	// Capture command output
//	stdout, stderr, err := testutil.CaptureOutput(func() {
//		// Run code that prints to stdout/stderr
//	})
//
//	// Run a safe test command
//	cmd := testutil.SafeTestCommand("Hello, World!")
//	stdout, stderr, exitCode := testutil.RunCommand(t, cmd)
//
package testutil