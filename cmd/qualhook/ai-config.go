// Package main provides the ai-config command for qualhook
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/bebsworthy/qualhook/internal/ai"
	"github.com/bebsworthy/qualhook/internal/executor"
	pkgconfig "github.com/bebsworthy/qualhook/pkg/config"
	"github.com/spf13/cobra"
	"github.com/AlecAivazis/survey/v2"
)

var (
	aiTool       string
	aiTimeout    time.Duration
	noTest       bool
	aiForceFlag  bool
)

// aiConfigCmd represents the ai-config command
var aiConfigCmd = &cobra.Command{
	Use:   "ai-config",
	Short: "Generate configuration using AI assistance",
	Long: `Automatically generate .qualhook.json by analyzing your project with Claude or Gemini.

This command uses AI tools to:
- Analyze your project structure and dependencies
- Detect the appropriate build system and package manager
- Generate suitable commands for formatting, linting, type checking, and testing
- Identify common error patterns for each command
- Handle monorepo configurations with workspace-specific settings

The AI analyzes your project files directly and generates a complete configuration
that you can review, test, and save.

Examples:
  # Generate configuration with AI tool selection
  qualhook ai-config

  # Use specific AI tool
  qualhook ai-config --tool claude
  qualhook ai-config --tool gemini

  # Generate without testing commands
  qualhook ai-config --no-test

  # Set timeout for AI analysis
  qualhook ai-config --timeout 2m

  # Force overwrite existing configuration
  qualhook ai-config --force

REQUIREMENTS:
  You need either Claude CLI or Gemini CLI installed:
  
  Claude CLI:
    npm install -g @anthropic-ai/claude-cli
    
  Gemini CLI:
    pip install google-generativeai-cli
    
For more information about AI CLI tools, see:
- Claude CLI: https://github.com/anthropics/claude-cli
- Gemini CLI: https://pypi.org/project/google-generativeai-cli/`,
	RunE: runAIConfig,
}

func init() {
	aiConfigCmd.Flags().StringVar(&aiTool, "tool", "", "AI tool to use (claude or gemini)")
	aiConfigCmd.Flags().DurationVar(&aiTimeout, "timeout", 5*time.Minute, "Timeout for AI analysis")
	aiConfigCmd.Flags().BoolVar(&noTest, "no-test", false, "Skip testing generated commands")
	aiConfigCmd.Flags().BoolVar(&aiForceFlag, "force", false, "Force overwrite existing configuration")
}

func runAIConfig(cmd *cobra.Command, args []string) error {
	fmt.Println("🤖 Generating qualhook configuration with AI assistance...")

	// Get working directory
	workingDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	// Check for existing configuration
	configPath := filepath.Join(workingDir, ".qualhook.json")
	if _, err := os.Stat(configPath); err == nil && !aiForceFlag {
		// Configuration exists, ask user what to do
		action, err := promptForExistingConfig()
		if err != nil {
			return err
		}
		
		switch action {
		case "cancel":
			fmt.Println("Configuration generation canceled.")
			return nil
		case "overwrite":
			// Continue with generation
		case "merge":
			return fmt.Errorf("configuration merging is not yet implemented - use --force to overwrite")
		}
	}

	// Create AI assistant
	executor := executor.NewCommandExecutor(2 * time.Minute)
	assistant := ai.NewAssistant(executor)

	// Set up AI options
	options := ai.AIOptions{
		Tool:         aiTool,
		WorkingDir:   workingDir,
		Interactive:  true,
		TestCommands: !noTest,
		Timeout:      aiTimeout,
	}

	// Generate configuration with AI
	ctx := context.Background()
	cfg, err := assistant.GenerateConfig(ctx, options)
	if err != nil {
		return handleAIError(err)
	}

	// Show configuration summary
	fmt.Println("\n📋 Generated Configuration Summary:")
	displayConfigSummary(cfg)

	// Ask for user confirmation before saving
	if !confirmSaveConfiguration() {
		fmt.Println("Configuration not saved. You can run 'qualhook ai-config' again to regenerate.")
		return nil
	}

	// Save configuration
	if err := saveConfiguration(cfg, configPath); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	fmt.Printf("\n✅ Configuration saved to %s\n", configPath)
	fmt.Println("\nYou can now use qualhook commands:")
	fmt.Println("  qualhook format    # Format your code")
	fmt.Println("  qualhook lint      # Run linting")
	fmt.Println("  qualhook typecheck # Check types")
	fmt.Println("  qualhook test      # Run tests")
	
	// Show custom commands if any
	if len(cfg.Commands) > 4 {
		fmt.Println("\nCustom commands detected:")
		standardCommands := map[string]bool{
			"format": true, "lint": true, "typecheck": true, "test": true,
		}
		for name := range cfg.Commands {
			if !standardCommands[name] {
				fmt.Printf("  qualhook %s\n", name)
			}
		}
	}

	return nil
}

// promptForExistingConfig asks the user what to do with existing configuration
func promptForExistingConfig() (string, error) {
	fmt.Println("\n⚠️  A .qualhook.json file already exists.")
	
	options := []string{
		"Overwrite with new AI-generated configuration",
		"Merge with existing configuration (coming soon)",
		"Cancel and keep existing configuration",
	}
	
	var choice string
	prompt := &survey.Select{
		Message: "What would you like to do?",
		Options: options,
	}
	
	if err := survey.AskOne(prompt, &choice); err != nil {
		return "", fmt.Errorf("failed to get user choice: %w", err)
	}
	
	switch choice {
	case options[0]:
		return "overwrite", nil
	case options[1]:
		return "merge", nil
	default:
		return "cancel", nil
	}
}

// confirmSaveConfiguration asks the user to confirm saving the configuration
func confirmSaveConfiguration() bool {
	confirm := false
	prompt := &survey.Confirm{
		Message: "Save this configuration to .qualhook.json?",
		Default: true,
	}
	
	// If survey fails, default to not saving for safety
	if err := survey.AskOne(prompt, &confirm); err != nil {
		fmt.Printf("Failed to get confirmation: %v\n", err)
		return false
	}
	
	return confirm
}

// displayConfigSummary shows a summary of the generated configuration
func displayConfigSummary(cfg *pkgconfig.Config) {
	if cfg.ProjectType != "" {
		fmt.Printf("   Project Type: %s\n", cfg.ProjectType)
	}
	
	fmt.Printf("   Commands Configured: %d\n", len(cfg.Commands))
	
	// List configured commands
	if len(cfg.Commands) > 0 {
		fmt.Println("   Commands:")
		for name, cmd := range cfg.Commands {
			fmt.Printf("     • %s: %s %v\n", name, cmd.Command, cmd.Args)
		}
	}
	
	// Show monorepo information if applicable
	if len(cfg.Paths) > 0 {
		fmt.Printf("   Monorepo Paths: %d\n", len(cfg.Paths))
		fmt.Println("   Workspace-specific configurations detected:")
		for _, pathConfig := range cfg.Paths {
			fmt.Printf("     • %s: %d overrides\n", pathConfig.Path, len(pathConfig.Commands))
		}
	}
}

// saveConfiguration saves the configuration to the specified path
func saveConfiguration(cfg *pkgconfig.Config, configPath string) error {
	// Serialize the configuration
	data, err := pkgconfig.SaveConfig(cfg)
	if err != nil {
		return fmt.Errorf("failed to serialize configuration: %w", err)
	}
	
	// Write to file
	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write configuration file: %w", err)
	}
	
	return nil
}

// handleAIError handles AI-specific errors with helpful messages
func handleAIError(err error) error {
	// Check if it's an ErrorWithRecovery (which embeds AIError)
	if recoveryErr, ok := err.(*ai.ErrorWithRecovery); ok {
		fmt.Fprintf(os.Stderr, "\n❌ AI Configuration Error: %s\n", recoveryErr.Message)
		
		// Show recovery suggestions if available
		suggestions := recoveryErr.RecoverySuggestions
		if len(suggestions) > 0 {
			fmt.Fprintf(os.Stderr, "\n💡 Suggestions:\n")
			for _, suggestion := range suggestions {
				fmt.Fprintf(os.Stderr, "   • %s\n", suggestion)
			}
		}
		
		// Suggest fallback to manual configuration
		fmt.Fprintf(os.Stderr, "\n📝 You can always configure manually using:\n")
		fmt.Fprintf(os.Stderr, "   qualhook config\n")
		
		return fmt.Errorf("AI configuration failed")
	}
	
	// Check if it's a regular AI error
	if aiErr, ok := err.(*ai.AIError); ok {
		fmt.Fprintf(os.Stderr, "\n❌ AI Configuration Error: %s\n", aiErr.Message)
		
		// Suggest fallback to manual configuration
		fmt.Fprintf(os.Stderr, "\n📝 You can always configure manually using:\n")
		fmt.Fprintf(os.Stderr, "   qualhook config\n")
		
		return fmt.Errorf("AI configuration failed")
	}
	
	// Generic error handling
	return fmt.Errorf("failed to generate configuration: %w", err)
}