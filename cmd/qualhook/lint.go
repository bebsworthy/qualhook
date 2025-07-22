package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/qualhook/qualhook/internal/config"
)

// lintCmd represents the lint command
var lintCmd = &cobra.Command{
	Use:   "lint",
	Short: "Run the configured linting command",
	Long: `Run the configured linting command for the current project.

This command executes the linting tool configured in .qualhook.json
and filters its output to provide only relevant error information.

Exit codes:
  0 - No linting issues found
  1 - Configuration or execution error
  2 - Linting issues detected (for Claude Code integration)`,
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