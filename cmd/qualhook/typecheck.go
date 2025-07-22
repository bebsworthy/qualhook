package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/qualhook/qualhook/internal/config"
)

// typecheckCmd represents the typecheck command
var typecheckCmd = &cobra.Command{
	Use:   "typecheck",
	Short: "Run the configured type checking command",
	Long: `Run the configured type checking command for the current project.

This command executes the type checking tool configured in .qualhook.json
and filters its output to provide only relevant error information.

Exit codes:
  0 - No type errors found
  1 - Configuration or execution error
  2 - Type errors detected (for Claude Code integration)`,
	RunE: runTypecheckCommand,
}

func init() {
	rootCmd.AddCommand(typecheckCmd)
}

func runTypecheckCommand(cmd *cobra.Command, args []string) error {
	// Load configuration
	loader := config.NewLoader()
	if configPath != "" {
		cfg, err := loader.LoadFromPath(configPath)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		return executeCommand(cfg, "typecheck", args)
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

	return executeCommand(cfg, "typecheck", args)
}