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

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "qualhook",
	Short: "Quality checks for Claude Code",
	Long: `Qualhook is a configurable command-line utility that serves as Claude Code 
hooks to enforce code quality for LLM coding agents.

It acts as an intelligent wrapper around project-specific commands (format, lint, 
typecheck, test), filtering their output to provide only relevant error 
information to the LLM.`,
	Version: Version,
}

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
		knownCommands := []string{"format", "lint", "typecheck", "test", "help", "completion"}
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
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
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

func init() {
	// Global flags
	rootCmd.PersistentFlags().BoolVar(&debugFlag, "debug", false, "Enable debug output")
	rootCmd.PersistentFlags().StringVar(&configPath, "config", "", "Path to configuration file")

	// Disable the default completion command
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	
	// Allow interspersed args for custom commands
	rootCmd.Flags().SetInterspersed(false)
	
}

