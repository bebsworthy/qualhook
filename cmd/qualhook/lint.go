package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/qualhook/qualhook/internal/config"
)

// lintCmd represents the lint command
var lintCmd = &cobra.Command{
	Use:   "lint [files...]",
	Short: "Run the configured linting command",
	Long: `Run the configured linting command for the current project.

This command executes the linting tool configured in .qualhook.json
and filters its output to provide only relevant error information.

The lint command will:
  • Execute your project's linter (ESLint, golangci-lint, clippy, etc.)
  • Filter verbose output to show only actual errors and warnings
  • Group errors by file for better readability
  • Provide actionable error messages for the LLM

FILTERING BEHAVIOR:
  Qualhook intelligently filters linter output to:
  • Include error locations (file:line:column)
  • Show error messages and rule names
  • Remove redundant information
  • Limit output size for LLM consumption

Exit codes:
  0 - No linting issues found
  1 - Configuration or execution error
  2 - Linting issues detected (for Claude Code integration)`,
	Example: `  # Lint all files in the current project
  qualhook lint

  # Lint specific files
  qualhook lint src/app.js src/components/

  # Lint with specific severity (if supported by linter)
  qualhook lint --error-only

  # Lint in debug mode to see full output
  qualhook --debug lint

  # Common linters configured:
  # JavaScript/TypeScript: eslint
  # Go: golangci-lint run
  # Rust: cargo clippy
  # Python: pylint, flake8, ruff

  # Example filtered output:
  # src/main.js:10:5: error: 'unused' is defined but never used [no-unused-vars]
  # src/utils.js:25:1: warning: Missing semicolon [semi]`,
	RunE: runLintCommand,
}

func init() {
	rootCmd.AddCommand(lintCmd)
}

func runLintCommand(cmd *cobra.Command, args []string) error {
	// Load configuration
	loader := config.NewLoader()
	if configPath != "" {
		cfg, err := loader.LoadFromPath(configPath)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		return executeCommand(cfg, "lint", args)
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

	return executeCommand(cfg, "lint", args)
}