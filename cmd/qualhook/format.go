package main

// formatCmd represents the format command
var formatCmd = createQualityCommand(
	"format",
	"Run the configured formatting command",
	`Run the configured formatting command for the current project.

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
	`  # Format all files in the current project
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
)

func init() {
	rootCmd.AddCommand(formatCmd)
}