package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/qualhook/qualhook/internal/config"
)

// typecheckCmd represents the typecheck command
var typecheckCmd = &cobra.Command{
	Use:   "typecheck [files...]",
	Short: "Run the configured type checking command",
	Long: `Run the configured type checking command for the current project.

This command executes the type checking tool configured in .qualhook.json
and filters its output to provide only relevant error information.

The typecheck command will:
  • Execute your project's type checker (tsc, mypy, flow, etc.)
  • Filter complex type error messages to their essence
  • Highlight the specific type mismatches
  • Provide clear error locations and suggestions

TYPE CHECKING TOOLS:
  Qualhook supports various type checkers:
  • TypeScript: tsc --noEmit
  • Python: mypy, pyright
  • Flow: flow check
  • Go: Built into go build/test

Exit codes:
  0 - No type errors found
  1 - Configuration or execution error
  2 - Type errors detected (for Claude Code integration)`,
	Example: `  # Type check entire project
  qualhook typecheck

  # Type check specific files
  qualhook typecheck src/models.ts src/api.ts

  # Type check with strict mode (if configured)
  qualhook typecheck --strict

  # Common type checkers configured:
  # TypeScript: tsc --noEmit
  # Python: mypy --show-error-codes
  # Flow: flow check
  # Haskell: ghc -fno-code

  # Example filtered output:
  # src/api.ts:45:12: error TS2345: Argument of type 'string' is not assignable to parameter of type 'number'
  # src/models.py:23: error: Incompatible return value type (got "str", expected "int")`,
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