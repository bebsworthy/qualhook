package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/qualhook/qualhook/internal/config"
)

// testCmd represents the test command
var testCmd = &cobra.Command{
	Use:   "test",
	Short: "Run the configured test command",
	Long: `Run the configured test command for the current project.

This command executes the test tool configured in .qualhook.json
and filters its output to provide only relevant error information.

Exit codes:
  0 - All tests passed
  1 - Configuration or execution error
  2 - Test failures detected (for Claude Code integration)`,
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