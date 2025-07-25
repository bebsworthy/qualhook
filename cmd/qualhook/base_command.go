package main

import (
	"fmt"
	"os"

	"github.com/bebsworthy/qualhook/internal/config"
	"github.com/spf13/cobra"
)

// createQualityCommand creates a standard qualhook command with common execution logic
func createQualityCommand(name, short, long, example string) *cobra.Command {
	return createQualityCommandWithUsage(name, " [files...]", short, long, example)
}

// createQualityCommandWithUsage creates a standard qualhook command with custom usage suffix
func createQualityCommandWithUsage(name, usageSuffix, short, long, example string) *cobra.Command {
	cmd := &cobra.Command{
		Use:     name + usageSuffix,
		Short:   short,
		Long:    long,
		Example: example,
		RunE:    createRunFunc(name),
	}
	return cmd
}

// createRunFunc creates the RunE function for a command with the given name
func createRunFunc(commandName string) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		// Load configuration
		loader := config.NewLoader()
		if configPath != "" {
			cfg, err := loader.LoadFromPath(configPath)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}
			return executeCommand(cfg, commandName, args)
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

		return executeCommand(cfg, commandName, args)
	}
}
