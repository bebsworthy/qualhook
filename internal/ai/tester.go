// Package ai provides AI-powered configuration generation for qualhook.
package ai

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/bebsworthy/qualhook/internal/executor"
	"github.com/bebsworthy/qualhook/pkg/config"
)

// testRunner implements the TestRunner interface for validating commands
type testRunner struct {
	executor commandExecutor
}

// NewTestRunner creates a new test runner instance
func NewTestRunner(exec commandExecutor) TestRunner {
	return &testRunner{
		executor: exec,
	}
}

// TestCommands tests a set of commands with user approval
func (t *testRunner) TestCommands(ctx context.Context, commands map[string]*config.CommandConfig) (map[string]TestResult, error) {
	results := make(map[string]TestResult)

	// Display all commands to be tested
	fmt.Println("\n📋 Commands to test:")
	for name, cmd := range commands {
		cmdStr := cmd.Command
		if len(cmd.Args) > 0 {
			cmdStr += " " + strings.Join(cmd.Args, " ")
		}
		fmt.Printf("  • %s: %s\n", name, cmdStr)
	}

	// Ask for user confirmation to proceed
	proceed := false
	prompt := &survey.Confirm{
		Message: "Would you like to test these commands?",
		Default: true,
	}
	if err := survey.AskOne(prompt, &proceed); err != nil {
		return nil, fmt.Errorf("failed to get user confirmation: %w", err)
	}

	if !proceed {
		// User chose not to test, return empty results
		return results, nil
	}

	// Test each command
	for name, cmd := range commands {
		select {
		case <-ctx.Done():
			return results, ctx.Err()
		default:
			result, err := t.TestCommand(ctx, name, cmd)
			if err != nil {
				return results, fmt.Errorf("failed to test command %s: %w", name, err)
			}
			results[name] = *result
		}
	}

	return results, nil
}

// TestCommand tests a single command
func (t *testRunner) TestCommand(ctx context.Context, name string, cmd *config.CommandConfig) (*TestResult, error) {
	// Display command to be tested
	cmdStr := cmd.Command
	if len(cmd.Args) > 0 {
		cmdStr += " " + strings.Join(cmd.Args, " ")
	}

	fmt.Printf("\n🧪 Testing '%s' command: %s\n", name, cmdStr)

	// Ask for user confirmation before running
	runCommand := false
	confirmPrompt := &survey.Confirm{
		Message: "Run this command?",
		Default: true,
	}
	if err := survey.AskOne(confirmPrompt, &runCommand); err != nil {
		return nil, fmt.Errorf("failed to get user confirmation: %w", err)
	}

	if !runCommand {
		// User chose to skip this command
		return &TestResult{
			Success:      false,
			Output:       "Command skipped by user",
			Error:        nil,
			Modified:     false,
			FinalCommand: cmd,
		}, nil
	}

	// Execute the command
	result := t.executeCommand(cmd)

	// Show command output
	if result.Output != "" {
		fmt.Println("\n📄 Command output:")
		// Limit output display to prevent flooding the terminal
		lines := strings.Split(result.Output, "\n")
		maxLines := 20
		if len(lines) > maxLines {
			for i := 0; i < maxLines; i++ {
				fmt.Println(lines[i])
			}
			fmt.Printf("... (%d more lines)\n", len(lines)-maxLines)
		} else {
			fmt.Print(result.Output)
		}
	}

	// Check if command succeeded
	if result.Success {
		fmt.Printf("✅ Command executed successfully (exit code: 0)\n")
		return result, nil
	}

	// Command failed, show error
	fmt.Printf("❌ Command failed (exit code: %d)\n", result.FinalCommand.ExitCodes[0])
	if result.Error != nil {
		fmt.Printf("   Error: %v\n", result.Error)
	}

	// Ask if user wants to modify the command
	modifyCommand := false
	modifyPrompt := &survey.Confirm{
		Message: "Would you like to modify this command?",
		Default: true,
	}
	if err := survey.AskOne(modifyPrompt, &modifyCommand); err != nil {
		return result, nil
	}

	if !modifyCommand {
		return result, nil
	}

	// Allow user to modify the command
	modifiedResult, err := t.modifyAndRetestCommand(cmd)
	if err != nil {
		return result, fmt.Errorf("failed to modify command: %w", err)
	}

	return modifiedResult, nil
}

// executeCommand executes a command and returns the result
func (t *testRunner) executeCommand(cmd *config.CommandConfig) *TestResult {
	// Set up execution options
	options := executor.ExecOptions{
		Timeout:    30 * time.Second, // Default timeout for test runs
		InheritEnv: true,
	}

	// Execute the command
	execResult, err := t.executor.Execute(cmd.Command, cmd.Args, options)

	// Determine success based on exit code
	success := err == nil && execResult != nil && execResult.ExitCode == 0

	// Build output string
	var output strings.Builder
	if execResult != nil {
		if execResult.Stdout != "" {
			output.WriteString(execResult.Stdout)
		}
		if execResult.Stderr != "" {
			if output.Len() > 0 {
				output.WriteString("\n")
			}
			output.WriteString(execResult.Stderr)
		}
	}

	// Create test result
	result := &TestResult{
		Success:      success,
		Output:       output.String(),
		Error:        err,
		Modified:     false,
		FinalCommand: cmd,
	}

	// If command failed, update the final command with the actual exit code
	if execResult != nil && execResult.ExitCode != 0 {
		finalCmd := *cmd
		finalCmd.ExitCodes = []int{execResult.ExitCode}
		result.FinalCommand = &finalCmd
	}

	return result
}

// modifyAndRetestCommand allows the user to modify a command and retest it
func (t *testRunner) modifyAndRetestCommand(originalCmd *config.CommandConfig) (*TestResult, error) {
	// Create a copy of the command to modify
	modifiedCmd := *originalCmd

	// Ask for new command
	newCommand := modifiedCmd.Command
	commandPrompt := &survey.Input{
		Message: "Enter new command:",
		Default: newCommand,
	}
	if err := survey.AskOne(commandPrompt, &newCommand); err != nil {
		return nil, err
	}
	modifiedCmd.Command = newCommand

	// Ask for new arguments
	currentArgs := strings.Join(modifiedCmd.Args, " ")
	newArgs := currentArgs
	argsPrompt := &survey.Input{
		Message: "Enter arguments (space-separated):",
		Default: currentArgs,
	}
	if err := survey.AskOne(argsPrompt, &newArgs); err != nil {
		return nil, err
	}

	// Parse arguments
	if newArgs == "" {
		modifiedCmd.Args = []string{}
	} else {
		modifiedCmd.Args = strings.Fields(newArgs)
	}

	// Ask if user wants to test the modified command
	testModified := false
	testPrompt := &survey.Confirm{
		Message: "Test the modified command?",
		Default: true,
	}
	if err := survey.AskOne(testPrompt, &testModified); err != nil {
		return nil, err
	}

	if !testModified {
		// Return the modified command without testing
		return &TestResult{
			Success:      false,
			Output:       "Modified command not tested",
			Error:        nil,
			Modified:     true,
			FinalCommand: &modifiedCmd,
		}, nil
	}

	// Test the modified command
	fmt.Printf("\n🔄 Retesting with modified command...\n")
	result := t.executeCommand(&modifiedCmd)
	result.Modified = true

	// Show result
	if result.Success {
		fmt.Printf("✅ Modified command executed successfully!\n")
	} else {
		fmt.Printf("❌ Modified command still failed (exit code: %d)\n", result.FinalCommand.ExitCodes[0])

		// Ask if user wants to try again
		tryAgain := false
		tryAgainPrompt := &survey.Confirm{
			Message: "Would you like to modify again?",
			Default: false,
		}
		if err := survey.AskOne(tryAgainPrompt, &tryAgain); err == nil && tryAgain {
			return t.modifyAndRetestCommand(&modifiedCmd)
		}
	}

	return result, nil
}
