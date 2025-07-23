// Package main is the entry point for the qualhook CLI tool.
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/qualhook/qualhook/internal/config"
	"github.com/qualhook/qualhook/internal/debug"
	pkgconfig "github.com/qualhook/qualhook/pkg/config"
)

// Version is set at build time via ldflags
var Version = "dev"

// Global flags
var (
	debugFlag  bool
	configPath string
)

// newRootCmd creates and returns the root command
func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "qualhook",
		Short: "Quality checks for Claude Code",
		Long: `Qualhook is a configurable command-line utility that serves as Claude Code 
hooks to enforce code quality for LLM coding agents.

It acts as an intelligent wrapper around project-specific commands (format, lint, 
typecheck, test), filtering their output to provide only relevant error 
information to the LLM.

GETTING STARTED:
  1. Configure qualhook for your project:
     $ qualhook config

  2. Run quality checks:
     $ qualhook format    # Run code formatter
     $ qualhook lint      # Run linter
     $ qualhook typecheck # Run type checker
     $ qualhook test      # Run tests

COMMON USAGE PATTERNS:
  • Monorepo with multiple projects:
    Configure path-specific commands in .qualhook.json
    
  • Custom commands:
    Add any command to your configuration and run it directly:
    $ qualhook my-custom-check

  • Claude Code integration:
    Use exit code 2 for errors that should be fed back to the LLM

EXAMPLES:
  # Configure a new project interactively
  $ qualhook config

  # Run linting and see only relevant errors
  $ qualhook lint

  # Validate your configuration
  $ qualhook config --validate

  # Use a specific config file
  $ qualhook --config ./custom-config.json lint

  # Enable debug output for troubleshooting
  $ qualhook --debug format

  # Import a configuration template
  $ qualhook template import nodejs-eslint

For more information, see: https://github.com/qualhook/qualhook`,
		Version: Version,
		Example: `  # Initial setup
  qualhook config

  # Daily usage
  qualhook format
  qualhook lint
  qualhook test

  # CI/CD integration
  qualhook lint || exit 2`,
	}
	
	// Global flags
	cmd.PersistentFlags().BoolVar(&debugFlag, "debug", false, "Enable debug output")
	cmd.PersistentFlags().StringVar(&configPath, "config", "", "Path to configuration file")

	// Disable the default completion command
	cmd.CompletionOptions.DisableDefaultCmd = true
	
	// Allow interspersed args for custom commands
	cmd.Flags().SetInterspersed(false)
	
	// Add subcommands
	cmd.AddCommand(formatCmd)
	cmd.AddCommand(lintCmd)
	cmd.AddCommand(typecheckCmd)
	cmd.AddCommand(testCmd)
	cmd.AddCommand(configCmd)
	cmd.AddCommand(templateCmd)
	
	return cmd
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = newRootCmd()

func main() {
	// Parse global flags early to enable debug logging
	for i := 1; i < len(os.Args); i++ {
		if os.Args[i] == "--debug" {
			debug.Enable()
			break
		}
	}
	
	// Check if the first argument might be a custom command
	if len(os.Args) > 1 && !strings.HasPrefix(os.Args[1], "-") {
		// Check if it's a known command
		cmdName := os.Args[1]
		knownCommands := []string{"format", "lint", "typecheck", "test", "config", "template", "help", "completion", "man"}
		isKnown := false
		for _, known := range knownCommands {
			if cmdName == known {
				isKnown = true
				break
			}
		}
		
		// If not a known command, try to execute as custom command
		if !isKnown {
			// Extract global flags first
			parseGlobalFlags()
			
			if err := tryCustomCommand(cmdName, extractNonFlagArgs(os.Args[2:])); err == nil {
				return
			}
		}
	}
	
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// parseGlobalFlags extracts global flags from command line
func parseGlobalFlags() {
	for i := 1; i < len(os.Args); i++ {
		switch os.Args[i] {
		case "--debug":
			debugFlag = true
		case "--config":
			if i+1 < len(os.Args) && !strings.HasPrefix(os.Args[i+1], "-") {
				configPath = os.Args[i+1]
				i++
			}
		}
	}
}

// extractNonFlagArgs returns only non-flag arguments
func extractNonFlagArgs(args []string) []string {
	var result []string
	for i := 0; i < len(args); i++ {
		if args[i] == "--config" && i+1 < len(args) {
			i++ // Skip the config value
			continue
		}
		if !strings.HasPrefix(args[i], "-") {
			result = append(result, args[i])
		}
	}
	return result
}

// tryCustomCommand attempts to execute a custom command from configuration
func tryCustomCommand(cmdName string, args []string) error {
	loader := config.NewLoader()
	var cfg *pkgconfig.Config
	var err error

	if configPath != "" {
		cfg, err = loader.LoadFromPath(configPath)
	} else {
		cwd, errWd := os.Getwd()
		if errWd != nil {
			return fmt.Errorf("failed to get working directory: %w", errWd)
		}
		cfg, err = loader.LoadForMonorepo(cwd)
	}

	if err != nil {
		return err
	}

	// Check if this is a configured command
	if _, exists := cfg.Commands[cmdName]; exists {
		return executeCommand(cfg, cmdName, args)
	}

	return fmt.Errorf("unknown command %q", cmdName)
}