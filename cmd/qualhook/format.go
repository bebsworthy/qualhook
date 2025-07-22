package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/qualhook/qualhook/internal/config"
)

// formatCmd represents the format command
var formatCmd = &cobra.Command{
	Use:   "format",
	Short: "Run the configured formatting command",
	Long: `Run the configured formatting command for the current project.

This command executes the formatting tool configured in .qualhook.json
and filters its output to provide only relevant error information.

Exit codes:
  0 - No formatting issues found
  1 - Configuration or execution error
  2 - Formatting issues detected (for Claude Code integration)`,
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