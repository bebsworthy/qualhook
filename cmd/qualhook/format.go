package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/qualhook/qualhook/internal/config"
)

// formatCmd represents the format command
var formatCmd = &cobra.Command{
	Use:   "format [files...]",
	Short: "Run the configured formatting command",
	Long: `Run the configured formatting command for the current project.

This command executes the formatting tool configured in .qualhook.json
and filters its output to provide only relevant error information.

The format command will:
  • Execute your project's formatter (prettier, gofmt, rustfmt, etc.)
  • Filter output to show only actual formatting issues
  • Return appropriate exit codes for Claude Code integration
  • Support monorepo configurations with path-specific formatters

Exit codes:
  0 - No formatting issues found
  1 - Configuration or execution error
  2 - Formatting issues detected (for Claude Code integration)`,
	Example: `  # Format all files in the current project
  qualhook format

  # Format specific files
  qualhook format src/main.js src/utils.js

  # Format with custom config
  qualhook --config ./frontend/.qualhook.json format

  # Format in a monorepo (auto-detects based on current directory)
  cd frontend && qualhook format

  # Common formatters configured:
  # JavaScript/TypeScript: prettier --write
  # Go: gofmt -w
  # Rust: cargo fmt
  # Python: black`,
	RunE: runFormatCommand,
}

func init() {
	rootCmd.AddCommand(formatCmd)
}

func runFormatCommand(cmd *cobra.Command, args []string) error {
	// Load configuration
	loader := config.NewLoader()
	if configPath != "" {
		cfg, err := loader.LoadFromPath(configPath)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		return executeCommand(cfg, "format", args)
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

	return executeCommand(cfg, "format", args)
}