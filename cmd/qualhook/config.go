// Package main provides the config command for qualhook
package main

import (
	"fmt"
	"os"

	"github.com/bebsworthy/qualhook/internal/config"
	"github.com/bebsworthy/qualhook/internal/wizard"
	pkgconfig "github.com/bebsworthy/qualhook/pkg/config"
	"github.com/spf13/cobra"
)

var (
	validateFlag bool
	outputPath   string
	forceFlag    bool
)

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configure qualhook for your project",
	Long: `Configure qualhook for your project through an interactive wizard.

The config command helps you set up qualhook by:
- Detecting your project type automatically
- Suggesting appropriate default configurations
- Allowing customization of commands and patterns
- Validating your configuration

Examples:
  # Run interactive configuration wizard
  qualhook config

  # Validate existing configuration
  qualhook config --validate

  # Create configuration in specific location
  qualhook config --output /path/to/.qualhook.json

  # Force overwrite existing configuration
  qualhook config --force`,
	RunE: runConfig,
}

func init() {
	configCmd.Flags().BoolVar(&validateFlag, "validate", false, "Validate existing configuration")
	configCmd.Flags().StringVar(&outputPath, "output", "", "Output path for configuration file")
	configCmd.Flags().BoolVar(&forceFlag, "force", false, "Force overwrite existing configuration")

	rootCmd.AddCommand(configCmd)
}

func runConfig(cmd *cobra.Command, args []string) error {
	if validateFlag {
		return runValidateConfig()
	}

	return runConfigWizard()
}

// runValidateConfig validates the current configuration
func runValidateConfig() error {
	fmt.Println("Validating qualhook configuration...")

	// Load configuration
	loader := config.NewLoader()
	var cfg *pkgconfig.Config
	var err error

	if configPath != "" {
		cfg, err = loader.LoadFromPath(configPath)
	} else {
		cfg, err = loader.Load()
	}

	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Validate configuration
	validator := config.NewValidator()
	if err := validator.Validate(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "\n❌ Configuration validation failed:\n")
		fmt.Fprintf(os.Stderr, "   %v\n", err)

		// Suggest fixes
		suggestions := validator.SuggestFixes(err)
		if len(suggestions) > 0 {
			fmt.Fprintf(os.Stderr, "\n💡 Suggestions:\n")
			for _, suggestion := range suggestions {
				fmt.Fprintf(os.Stderr, "   • %s\n", suggestion)
			}
		}

		return fmt.Errorf("configuration is invalid")
	}

	fmt.Println("\n✅ Configuration is valid!")

	// Display configuration summary
	fmt.Printf("\n📋 Configuration Summary:\n")
	fmt.Printf("   Version: %s\n", cfg.Version)
	if cfg.ProjectType != "" {
		fmt.Printf("   Project Type: %s\n", cfg.ProjectType)
	}
	fmt.Printf("   Commands: %d configured\n", len(cfg.Commands))
	if len(cfg.Paths) > 0 {
		fmt.Printf("   Monorepo Paths: %d configured\n", len(cfg.Paths))
	}

	// List configured commands
	if len(cfg.Commands) > 0 {
		fmt.Printf("\n   Configured Commands:\n")
		for name := range cfg.Commands {
			fmt.Printf("   • %s\n", name)
		}
	}

	return nil
}

// runConfigWizard runs the interactive configuration wizard
func runConfigWizard() error {
	w, err := wizard.NewConfigWizard()
	if err != nil {
		return fmt.Errorf("failed to create wizard: %w", err)
	}

	return w.Run(outputPath, forceFlag)
}
