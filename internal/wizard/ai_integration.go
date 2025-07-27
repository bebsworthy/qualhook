// Package wizard provides interactive configuration creation for qualhook
package wizard

import (
	"context"
	"fmt"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/bebsworthy/qualhook/internal/ai"
	"github.com/bebsworthy/qualhook/internal/executor"
	pkgconfig "github.com/bebsworthy/qualhook/pkg/config"
)

// AIIntegration adds AI capabilities to the configuration wizard
type AIIntegration struct {
	assistant ai.Assistant
	ui        *ai.InteractiveUI
	executor  *executor.CommandExecutor
}

// NewAIIntegration creates a new AI integration for the wizard
func NewAIIntegration(exec *executor.CommandExecutor) *AIIntegration {
	return &AIIntegration{
		assistant: ai.NewAssistant(exec),
		ui:        ai.NewInteractiveUI(),
		executor:  exec,
	}
}

// EnhanceCommand suggests improvements for a command using AI
func (a *AIIntegration) EnhanceCommand(ctx context.Context, commandType string, current *pkgconfig.CommandConfig) (*pkgconfig.CommandConfig, error) {
	// Ask user if they want AI assistance
	if shouldSkipAI := a.askUserForAI(commandType); !shouldSkipAI {
		return current, nil
	}

	// Get AI suggestion
	suggestion, err := a.getAISuggestion(ctx, commandType, current)
	if err != nil {
		return current, nil
	}
	if suggestion == nil {
		return current, nil
	}

	// Show and confirm suggestion
	if confirmed := a.showAndConfirmSuggestion(suggestion); !confirmed {
		return current, nil
	}

	// Convert to command config
	enhanced := a.convertSuggestionToConfig(suggestion)

	// Optionally test the command
	finalCmd, err := a.testCommandIfRequested(ctx, commandType, enhanced, current)
	if err != nil {
		return nil, err
	}

	return finalCmd, nil
}

// askUserForAI asks if the user wants AI assistance
func (a *AIIntegration) askUserForAI(commandType string) bool {
	useAI := false
	prompt := &survey.Confirm{
		Message: fmt.Sprintf("Would you like AI assistance to configure the %s command?", commandType),
		Default: false,
	}
	if err := survey.AskOne(prompt, &useAI); err != nil {
		return false
	}
	return useAI
}

// getAISuggestion generates an AI suggestion for the command
func (a *AIIntegration) getAISuggestion(ctx context.Context, commandType string, _ *pkgconfig.CommandConfig) (*ai.CommandSuggestion, error) {
	// Detect available AI tools
	detector := ai.NewToolDetector(a.executor)
	tools, err := detector.DetectTools()
	if err != nil || len(tools) == 0 {
		fmt.Println("\nNo AI tools detected. Please install Claude CLI or Gemini CLI.")
		fmt.Println("See 'qualhook ai-config --help' for installation instructions.")
		return nil, nil
	}

	// Select AI tool
	selectedTool, err := a.ui.SelectTool(tools)
	if err != nil {
		return nil, fmt.Errorf("tool selection failed: %w", err)
	}

	// Create project context
	projectInfo := ai.ProjectContext{
		ProjectType: "", // Will be detected by AI
	}

	// Generate suggestion
	fmt.Printf("\nGenerating %s command suggestion using %s...\n", commandType, selectedTool)
	suggestion, err := a.assistant.SuggestCommand(ctx, commandType, projectInfo)
	if err != nil {
		fmt.Printf("AI suggestion failed: %v\n", err)
		return nil, err
	}

	return suggestion, nil
}

// showAndConfirmSuggestion displays the AI suggestion and asks for confirmation
func (a *AIIntegration) showAndConfirmSuggestion(suggestion *ai.CommandSuggestion) bool {
	fmt.Println("\nAI Suggestion:")
	fmt.Printf("Command: %s %s\n", suggestion.Command, strings.Join(suggestion.Args, " "))
	if suggestion.Explanation != "" {
		fmt.Printf("Explanation: %s\n", suggestion.Explanation)
	}

	useSuggestion := false
	confirmPrompt := &survey.Confirm{
		Message: "Use this AI-suggested command?",
		Default: true,
	}
	if err := survey.AskOne(confirmPrompt, &useSuggestion); err != nil {
		return false
	}
	return useSuggestion
}

// convertSuggestionToConfig converts AI suggestion to command config
func (a *AIIntegration) convertSuggestionToConfig(suggestion *ai.CommandSuggestion) *pkgconfig.CommandConfig {
	enhanced := &pkgconfig.CommandConfig{
		Command:   suggestion.Command,
		Args:      suggestion.Args,
		ExitCodes: suggestion.ExitCodes,
	}

	// Convert error patterns
	for _, pattern := range suggestion.ErrorPatterns {
		enhanced.ErrorPatterns = append(enhanced.ErrorPatterns, &pkgconfig.RegexPattern{
			Pattern: pattern.Pattern,
			Flags:   pattern.Flags,
		})
	}

	return enhanced
}

// testCommandIfRequested optionally tests the command
func (a *AIIntegration) testCommandIfRequested(ctx context.Context, commandType string, enhanced, current *pkgconfig.CommandConfig) (*pkgconfig.CommandConfig, error) {
	testCommand := false
	testPrompt := &survey.Confirm{
		Message: "Test this command before saving?",
		Default: true,
	}
	if err := survey.AskOne(testPrompt, &testCommand); err != nil {
		return nil, err
	}

	if !testCommand {
		return enhanced, nil
	}

	// Run the test
	tester := ai.NewTestRunner(a.executor)
	results, err := tester.TestCommands(ctx, map[string]*pkgconfig.CommandConfig{
		commandType: enhanced,
	})
	if err != nil {
		fmt.Printf("Test failed: %v\n", err)
		return current, nil
	}

	// Process test results
	if result, ok := results[commandType]; ok {
		if !result.Success {
			return a.handleFailedTest(commandType, enhanced, current, &result)
		} else if result.Modified {
			// User modified during testing
			return result.FinalCommand, nil
		}
	}

	return enhanced, nil
}

// handleFailedTest handles the case when a command test fails
func (a *AIIntegration) handleFailedTest(_ string, enhanced, current *pkgconfig.CommandConfig, result *ai.TestResult) (*pkgconfig.CommandConfig, error) {
	fmt.Printf("Command test failed: %v\n", result.Error)

	// Allow user to modify or reject
	modifyCommand := false
	modifyPrompt := &survey.Confirm{
		Message: "Would you like to modify the command?",
		Default: true,
	}
	if err := survey.AskOne(modifyPrompt, &modifyCommand); err != nil {
		return nil, err
	}

	if !modifyCommand {
		return current, nil
	}

	// Let user edit the command
	editedCmd := strings.Join(append([]string{enhanced.Command}, enhanced.Args...), " ")
	editPrompt := &survey.Input{
		Message: "Edit command:",
		Default: editedCmd,
	}
	if err := survey.AskOne(editPrompt, &editedCmd); err != nil {
		return nil, err
	}

	// Parse edited command
	parts := strings.Fields(editedCmd)
	if len(parts) > 0 {
		enhanced.Command = parts[0]
		enhanced.Args = parts[1:]
	}

	return enhanced, nil
}

// ReviewCommands presents all commands for review with AI enhancement options
func (a *AIIntegration) ReviewCommands(ctx context.Context, commands map[string]*pkgconfig.CommandConfig, customCommands map[string]*pkgconfig.CommandConfig) error {
	fmt.Println("\n=== Command Review ===")
	fmt.Println("Let's review your quality check commands:")

	// Review standard commands
	if err := a.reviewStandardCommands(ctx, commands); err != nil {
		return err
	}

	// Review custom commands
	if err := a.reviewCustomCommands(customCommands); err != nil {
		return err
	}

	// Ask about adding new custom commands
	if err := a.promptForNewCustomCommands(ctx, customCommands); err != nil {
		return err
	}

	return nil
}

// reviewStandardCommands reviews the standard command types
func (a *AIIntegration) reviewStandardCommands(ctx context.Context, commands map[string]*pkgconfig.CommandConfig) error {
	commandTypes := []string{"format", "lint", "typecheck", "test"}

	for _, cmdType := range commandTypes {
		cmd := commands[cmdType]
		a.displayCommandStatus(cmdType, cmd)

		action, err := a.promptForCommandAction(cmdType, cmd)
		if err != nil {
			return err
		}

		if err := a.handleCommandAction(ctx, cmdType, cmd, action, commands); err != nil {
			return err
		}
	}

	return nil
}

// displayCommandStatus shows the current status of a command
func (a *AIIntegration) displayCommandStatus(cmdType string, cmd *pkgconfig.CommandConfig) {
	if cmd == nil {
		fmt.Printf("\n%s: ❌ Not Configured\n", cmdType)
	} else {
		fmt.Printf("\n%s: ✓ %s %s\n", cmdType, cmd.Command, strings.Join(cmd.Args, " "))
	}
}

// promptForCommandAction asks what to do with a command
func (a *AIIntegration) promptForCommandAction(cmdType string, cmd *pkgconfig.CommandConfig) (string, error) {
	options := []string{
		"Keep as is",
		"Configure manually",
		"Use AI assistance",
		"Skip (leave unconfigured)",
	}

	if cmd == nil {
		// Remove "Keep as is" option if not configured
		options = options[1:]
	}

	var action string
	actionPrompt := &survey.Select{
		Message: fmt.Sprintf("What would you like to do with the %s command?", cmdType),
		Options: options,
	}

	if err := survey.AskOne(actionPrompt, &action); err != nil {
		return "", err
	}

	return action, nil
}

// handleCommandAction processes the user's choice for a command
func (a *AIIntegration) handleCommandAction(ctx context.Context, cmdType string, cmd *pkgconfig.CommandConfig, action string, commands map[string]*pkgconfig.CommandConfig) error {
	switch action {
	case "Configure manually":
		newCmd, err := a.configureManually(cmdType)
		if err != nil {
			return err
		}
		commands[cmdType] = newCmd

	case "Use AI assistance":
		enhanced, err := a.EnhanceCommand(ctx, cmdType, cmd)
		if err != nil {
			return err
		}
		if enhanced != nil {
			commands[cmdType] = enhanced
		}

	case "Skip (leave unconfigured)":
		delete(commands, cmdType)

		// "Keep as is" - do nothing
	}

	return nil
}

// reviewCustomCommands reviews user-defined custom commands
func (a *AIIntegration) reviewCustomCommands(customCommands map[string]*pkgconfig.CommandConfig) error {
	if len(customCommands) == 0 {
		return nil
	}

	fmt.Println("\nReviewing custom commands...")
	for name, cmd := range customCommands {
		fmt.Printf("\nCustom command '%s': %s %s\n", name, cmd.Command, strings.Join(cmd.Args, " "))

		action, err := a.promptForCustomCommandAction(name)
		if err != nil {
			return err
		}

		if err := a.handleCustomCommandAction(name, action, customCommands); err != nil {
			return err
		}
	}

	return nil
}

// promptForCustomCommandAction asks what to do with a custom command
func (a *AIIntegration) promptForCustomCommandAction(name string) (string, error) {
	var action string
	actionPrompt := &survey.Select{
		Message: fmt.Sprintf("What would you like to do with custom command '%s'?", name),
		Options: []string{
			"Keep as is",
			"Modify",
			"Remove",
		},
	}

	if err := survey.AskOne(actionPrompt, &action); err != nil {
		return "", err
	}

	return action, nil
}

// handleCustomCommandAction processes the user's choice for a custom command
func (a *AIIntegration) handleCustomCommandAction(name, action string, customCommands map[string]*pkgconfig.CommandConfig) error {
	switch action {
	case "Modify":
		newCmd, err := a.configureManually(name)
		if err != nil {
			return err
		}
		customCommands[name] = newCmd

	case "Remove":
		delete(customCommands, name)
	}

	return nil
}

// promptForNewCustomCommands asks if user wants to add new custom commands
func (a *AIIntegration) promptForNewCustomCommands(ctx context.Context, customCommands map[string]*pkgconfig.CommandConfig) error {
	for {
		addMore := false
		addPrompt := &survey.Confirm{
			Message: "Would you like to add a new custom command?",
			Default: false,
		}
		if err := survey.AskOne(addPrompt, &addMore); err != nil {
			return err
		}

		if !addMore {
			break
		}

		name, cmd, err := a.addCustomCommand(ctx)
		if err != nil {
			return err
		}
		if name != "" && cmd != nil {
			customCommands[name] = cmd
		}
	}

	return nil
}

// configureManually allows manual command configuration
func (a *AIIntegration) configureManually(cmdType string) (*pkgconfig.CommandConfig, error) {
	var command string
	cmdPrompt := &survey.Input{
		Message: fmt.Sprintf("Enter the full command for %s (e.g., 'eslint . --fix'):", cmdType),
	}
	if err := survey.AskOne(cmdPrompt, &command, survey.WithValidator(survey.Required)); err != nil {
		return nil, err
	}

	// Parse command
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return nil, fmt.Errorf("invalid command")
	}

	cmd := &pkgconfig.CommandConfig{
		Command: parts[0],
		Args:    parts[1:],
	}

	// Ask about error patterns
	addPatterns := false
	patternPrompt := &survey.Confirm{
		Message: "Would you like to configure error patterns for this command?",
		Default: false,
	}
	if err := survey.AskOne(patternPrompt, &addPatterns); err != nil {
		return nil, err
	}

	if addPatterns {
		var patterns []string
		patternInputPrompt := &survey.Input{
			Message: "Enter error patterns (regex, comma-separated):",
			Help:    "Example: error:, warning:, FAIL",
		}
		if err := survey.AskOne(patternInputPrompt, &patterns); err != nil {
			return nil, err
		}

		if patterns != nil {
			patternList := strings.Split(strings.Join(patterns, ""), ",")
			for _, p := range patternList {
				p = strings.TrimSpace(p)
				if p != "" {
					cmd.ErrorPatterns = append(cmd.ErrorPatterns, &pkgconfig.RegexPattern{
						Pattern: p,
						Flags:   "",
					})
				}
			}
		}
	}

	return cmd, nil
}

// addCustomCommand allows adding a new custom command
func (a *AIIntegration) addCustomCommand(ctx context.Context) (string, *pkgconfig.CommandConfig, error) {
	var name string
	namePrompt := &survey.Input{
		Message: "Enter the name for the custom command:",
		Help:    "This is how you'll run it: qualhook <name>",
	}
	if err := survey.AskOne(namePrompt, &name, survey.WithValidator(survey.Required)); err != nil {
		return "", nil, err
	}

	// Check if user wants AI assistance for this custom command
	useAI := false
	aiPrompt := &survey.Confirm{
		Message: "Would you like AI assistance to configure this command?",
		Default: false,
	}
	if err := survey.AskOne(aiPrompt, &useAI); err != nil {
		return "", nil, err
	}

	if useAI {
		enhanced, err := a.EnhanceCommand(ctx, name, nil)
		if err != nil {
			return "", nil, err
		}
		return name, enhanced, nil
	}

	// Manual configuration
	cmd, err := a.configureManually(name)
	if err != nil {
		return "", nil, err
	}

	return name, cmd, nil
}
