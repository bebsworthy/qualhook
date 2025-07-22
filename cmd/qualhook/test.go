package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/qualhook/qualhook/internal/config"
)

// testCmd represents the test command
var testCmd = &cobra.Command{
	Use:   "test [test-files-or-patterns...]",
	Short: "Run the configured test command",
	Long: `Run the configured test command for the current project.

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
	Example: `  # Run all tests
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
	RunE: runTestCommand,
}

func init() {
	rootCmd.AddCommand(testCmd)
}

func runTestCommand(cmd *cobra.Command, args []string) error {
	// Load configuration
	loader := config.NewLoader()
	if configPath != "" {
		cfg, err := loader.LoadFromPath(configPath)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		return executeCommand(cfg, "test", args)
	}

	// Load configuration based on current directory
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	cfg, err := loader.LoadForMonorepo(cwd)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	return executeCommand(cfg, "test", args)
}