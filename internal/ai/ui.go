// Package ai provides AI-powered configuration generation for qualhook
package ai

import (
	"fmt"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/bebsworthy/qualhook/pkg/config"
)

const (
	claudeTool = "claude"
	geminiTool = "gemini"
)

// InteractiveUI provides UI helpers for AI-assisted configuration
type InteractiveUI struct{}

// NewInteractiveUI creates a new interactive UI helper
func NewInteractiveUI() *InteractiveUI {
	return &InteractiveUI{}
}

// SelectTool prompts the user to select an AI tool from available options
func (ui *InteractiveUI) SelectTool(availableTools []Tool) (string, error) {
	if len(availableTools) == 0 {
		return "", fmt.Errorf("no AI tools available")
	}

	// If only one tool is available, use it automatically
	if len(availableTools) == 1 {
		fmt.Printf("Using %s for AI assistance.\n", availableTools[0].Name)
		return availableTools[0].Name, nil
	}

	// Build options list with tool information
	options := make([]string, len(availableTools))
	for i, tool := range availableTools {
		if tool.Version != "" {
			options[i] = fmt.Sprintf("%s (%s)", tool.Name, tool.Version)
		} else {
			options[i] = tool.Name
		}
	}

	// Prompt for tool selection
	var selected string
	prompt := &survey.Select{
		Message: "Select an AI tool for configuration assistance:",
		Options: options,
	}

	if err := survey.AskOne(prompt, &selected); err != nil {
		return "", err
	}

	// Extract tool name from selection
	for _, tool := range availableTools {
		if strings.HasPrefix(selected, tool.Name) {
			return tool.Name, nil
		}
	}

	return "", fmt.Errorf("invalid tool selection")
}

// ReviewConfiguration displays a configuration summary for review
func (ui *InteractiveUI) ReviewConfiguration(cfg *config.Config) error {
	fmt.Println("\n=== Generated Configuration Summary ===")
	
	// Display project type if set
	if cfg.ProjectType != "" {
		fmt.Printf("Project Type: %s\n", cfg.ProjectType)
	}

	// Display standard commands
	ui.displayCommands(cfg)
	
	// Display monorepo paths if configured
	if len(cfg.Paths) > 0 {
		ui.displayMonorepoConfig(cfg.Paths)
	}

	fmt.Println("\n=====================================")
	return nil
}

// displayCommands shows standard and custom commands
func (ui *InteractiveUI) displayCommands(cfg *config.Config) {
	standardCommands := []string{"format", "lint", "typecheck", "test"}
	
	fmt.Println("\nCommands:")
	// Display standard commands
	for _, cmdType := range standardCommands {
		if cmd, exists := cfg.Commands[cmdType]; exists {
			ui.printCommand("  ", cmdType, cmd)
		} else {
			fmt.Printf("  %s: <not configured>\n", cmdType)
		}
	}
	
	// Display custom commands
	customCommands := make([]string, 0)
	for name := range cfg.Commands {
		if !isStandardCommand(name, standardCommands) {
			customCommands = append(customCommands, name)
		}
	}
	
	if len(customCommands) > 0 {
		fmt.Println("\nCustom Commands:")
		for _, name := range customCommands {
			ui.printCommand("  ", name, cfg.Commands[name])
		}
	}
}

// displayMonorepoConfig shows monorepo-specific configuration
func (ui *InteractiveUI) displayMonorepoConfig(paths []*config.PathConfig) {
	fmt.Println("\nMonorepo Configuration:")
	for _, path := range paths {
		fmt.Printf("  Path: %s\n", path.Path)
		if len(path.Commands) > 0 {
			fmt.Println("    Overrides:")
			for cmdName, cmd := range path.Commands {
				ui.printCommand("      ", cmdName, cmd)
			}
		}
	}
}

// printCommand formats and prints a command
func (ui *InteractiveUI) printCommand(indent, name string, cmd *config.CommandConfig) {
	fmt.Printf("%s%s: %s", indent, name, cmd.Command)
	if len(cmd.Args) > 0 {
		fmt.Printf(" %s", strings.Join(cmd.Args, " "))
	}
	fmt.Println()
}

// isStandardCommand checks if a command is in the standard list
func isStandardCommand(name string, standardCommands []string) bool {
	for _, std := range standardCommands {
		if name == std {
			return true
		}
	}
	return false
}

// ConfirmTestRun asks for user approval before running a test command
func (ui *InteractiveUI) ConfirmTestRun(commandName string, command string, args []string) (bool, error) {
	fullCommand := command
	if len(args) > 0 {
		fullCommand = fmt.Sprintf("%s %s", command, strings.Join(args, " "))
	}

	fmt.Printf("\nTest command for '%s':\n", commandName)
	fmt.Printf("  %s\n", fullCommand)

	confirm := false
	prompt := &survey.Confirm{
		Message: "Run this command to test it?",
		Default: true,
	}

	if err := survey.AskOne(prompt, &confirm); err != nil {
		return false, err
	}

	return confirm, nil
}

// DisplayCommandComparison shows differences between configurations
func (ui *InteractiveUI) DisplayCommandComparison(original, suggested *config.CommandConfig, commandName string) error {
	fmt.Printf("\n=== Command Comparison for '%s' ===\n", commandName)

	// Show original command if it exists
	if original != nil {
		fmt.Println("Original:")
		fmt.Printf("  Command: %s", original.Command)
		if len(original.Args) > 0 {
			fmt.Printf(" %s", strings.Join(original.Args, " "))
		}
		fmt.Println()
		if len(original.ErrorPatterns) > 0 {
			fmt.Printf("  Error Patterns: %d configured\n", len(original.ErrorPatterns))
		}
		if len(original.ExitCodes) > 0 {
			fmt.Printf("  Exit Codes: %v\n", original.ExitCodes)
		}
	} else {
		fmt.Println("Original: <not configured>")
	}

	// Show suggested command
	fmt.Println("\nSuggested:")
	fmt.Printf("  Command: %s", suggested.Command)
	if len(suggested.Args) > 0 {
		fmt.Printf(" %s", strings.Join(suggested.Args, " "))
	}
	fmt.Println()
	if len(suggested.ErrorPatterns) > 0 {
		fmt.Printf("  Error Patterns: %d configured\n", len(suggested.ErrorPatterns))
	}
	if len(suggested.ExitCodes) > 0 {
		fmt.Printf("  Exit Codes: %v\n", suggested.ExitCodes)
	}

	fmt.Println("===================================")
	return nil
}

// PromptForCommandModification allows the user to modify a command after test failure
func (ui *InteractiveUI) PromptForCommandModification(commandName string, current *config.CommandConfig) (*config.CommandConfig, error) {
	fmt.Printf("\nThe %s command failed during testing.\n", commandName)

	// Ask if user wants to modify
	modify := false
	modifyPrompt := &survey.Confirm{
		Message: "Would you like to modify the command?",
		Default: true,
	}
	if err := survey.AskOne(modifyPrompt, &modify); err != nil {
		return nil, err
	}

	if !modify {
		return current, nil
	}

	// Create a copy of the current command
	modified := &config.CommandConfig{
		Command:       current.Command,
		Args:          append([]string{}, current.Args...),
		ErrorPatterns: current.ErrorPatterns,
		ExitCodes:     current.ExitCodes,
		Prompt:        current.Prompt,
	}

	// Modify command
	newCommand := modified.Command
	commandPrompt := &survey.Input{
		Message: "Command:",
		Default: modified.Command,
	}
	if err := survey.AskOne(commandPrompt, &newCommand); err != nil {
		return nil, err
	}
	modified.Command = newCommand

	// Modify arguments
	currentArgs := strings.Join(modified.Args, " ")
	newArgs := currentArgs
	argsPrompt := &survey.Input{
		Message: "Arguments:",
		Default: currentArgs,
	}
	if err := survey.AskOne(argsPrompt, &newArgs); err != nil {
		return nil, err
	}
	if newArgs == "" {
		modified.Args = []string{}
	} else {
		modified.Args = strings.Fields(newArgs)
	}

	return modified, nil
}

// ConfirmConfiguration asks for final approval before saving configuration
func (ui *InteractiveUI) ConfirmConfiguration() (bool, error) {
	confirm := false
	prompt := &survey.Confirm{
		Message: "Save this configuration?",
		Default: true,
	}

	if err := survey.AskOne(prompt, &confirm); err != nil {
		return false, err
	}

	return confirm, nil
}

// SelectCommandsToReview allows selecting specific commands to review/modify
func (ui *InteractiveUI) SelectCommandsToReview(commands map[string]*config.CommandConfig) ([]string, error) {
	// Build options list
	options := make([]string, 0, len(commands))
	for name := range commands {
		options = append(options, name)
	}

	// Sort standard commands first
	standardCommands := []string{"format", "lint", "typecheck", "test"}
	sortedOptions := []string{}
	for _, std := range standardCommands {
		if _, exists := commands[std]; exists {
			sortedOptions = append(sortedOptions, std)
		}
	}
	// Add custom commands
	for _, opt := range options {
		isStandard := false
		for _, std := range standardCommands {
			if opt == std {
				isStandard = true
				break
			}
		}
		if !isStandard {
			sortedOptions = append(sortedOptions, opt)
		}
	}

	selected := []string{}
	prompt := &survey.MultiSelect{
		Message: "Select commands to review/modify:",
		Options: sortedOptions,
	}

	if err := survey.AskOne(prompt, &selected); err != nil {
		return nil, err
	}

	return selected, nil
}

// ShowAIProgress displays a message while AI is processing
func (ui *InteractiveUI) ShowAIProgress(message string) {
	fmt.Printf("\n%s\n", message)
	fmt.Println("Press ESC to cancel...")
}

// ShowAIError displays an AI-related error with helpful context
func (ui *InteractiveUI) ShowAIError(err error, toolName string) {
	fmt.Printf("\n⚠️  AI assistance error with %s: %v\n", toolName, err)
	
	// Provide specific guidance based on error type
	switch {
	case strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "not available"):
		fmt.Println("\nTo install the AI tool:")
		ui.ShowInstallInstructions(toolName)
	case strings.Contains(err.Error(), "timeout"):
		fmt.Println("\nThe AI tool is taking longer than expected. You can:")
		fmt.Println("  - Wait longer for the response")
		fmt.Println("  - Cancel and try again")
		fmt.Println("  - Continue with manual configuration")
	case strings.Contains(err.Error(), "canceled"):
		fmt.Println("\nAI assistance canceled. Continuing with manual configuration...")
	}
}

// ShowInstallInstructions displays platform-specific installation instructions
func (ui *InteractiveUI) ShowInstallInstructions(toolName string) {
	switch toolName {
	case claudeTool:
		fmt.Println("  macOS/Linux: curl -fsSL https://cli.claude.ai/install.sh | sh")
		fmt.Println("  Windows: Visit https://cli.claude.ai for installation instructions")
	case geminiTool:
		fmt.Println("  All platforms: pip install google-generativeai")
		fmt.Println("  Or visit: https://ai.google.dev/gemini-api/docs/quickstart")
	default:
		fmt.Printf("  Please refer to the %s documentation for installation instructions.\n", toolName)
	}
}