package main

// testCmd represents the test command
var testCmd = createQualityCommandWithUsage(
	"test",
	" [test-files-or-patterns...]",
	"Run the configured test command",
	`Run the configured test command for the current project.

This command executes the test tool configured in .qualhook.json
and filters its output to provide only relevant error information.

The test command will:
  • Execute your project's test runner (jest, go test, cargo test, pytest, etc.)
  • Filter test output to show only failures and errors
  • Provide clear failure messages with file locations
  • Include relevant stack traces without noise

TEST OUTPUT FILTERING:
  Qualhook intelligently processes test output to:
  • Show failed test names and assertions
  • Include error messages and diffs
  • Display file locations for quick navigation
  • Remove verbose setup/teardown logs

Exit codes:
  0 - All tests passed
  1 - Configuration or execution error
  2 - Test failures detected (for Claude Code integration)`,
	`  # Run all tests
  qualhook test

  # Run specific test files
  qualhook test src/__tests__/api.test.js

  # Run tests matching a pattern
  qualhook test "**/user*.test.js"

  # Run tests in watch mode (if supported)
  qualhook test --watch

  # Common test runners configured:
  # JavaScript/TypeScript: jest, vitest, mocha
  # Go: go test ./...
  # Rust: cargo test
  # Python: pytest, unittest

  # Example filtered output:
  # FAIL src/utils.test.js
  #   calculateTotal
  #     ✕ should return sum of items (5ms)
  #       Expected: 150
  #       Received: 140
  #       at src/utils.test.js:15:23`,
)

func init() {
	rootCmd.AddCommand(testCmd)
}