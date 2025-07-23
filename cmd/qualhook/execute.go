package main

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/qualhook/qualhook/internal/debug"
	"github.com/qualhook/qualhook/internal/executor"
	"github.com/qualhook/qualhook/internal/filter"
	"github.com/qualhook/qualhook/internal/hook"
	"github.com/qualhook/qualhook/internal/reporter"
	"github.com/qualhook/qualhook/internal/watcher"
	"github.com/qualhook/qualhook/pkg/config"
)

// osExit is a variable to allow mocking os.Exit in tests
var osExit = os.Exit

// executeCommand executes a configured command and processes its output
func executeCommand(cfg *config.Config, commandName string, extraArgs []string) error {
	start := time.Now()
	debug.LogSection("Execute Command")
	debug.Log("Command: %s", commandName)
	debug.Log("Extra Args: %v", extraArgs)
	
	// Check if command exists in configuration
	cmdConfig, exists := cfg.Commands[commandName]
	if !exists {
		return fmt.Errorf("command %q not found in configuration", commandName)
	}

	// Parse hook input if available
	hookInput := parseHookInput()
	
	// Extract edited files if available
	editedFiles := extractEditedFiles(hookInput)

	// Determine execution mode
	var results []executor.ComponentExecResult
	
	if len(editedFiles) > 0 {
		r, err := executeFileAwareCommand(cfg, commandName, extraArgs, editedFiles)
		if err != nil {
			return err
		}
		results = r
	} else {
		r, err := executeSingleCommand(cmdConfig, commandName, extraArgs)
		if err != nil {
			return err
		}
		results = r
	}
	
	// Report and output results
	reportAndOutputResults(results, start)
	
	return nil
}

// parseHookInput parses Claude Code hook input from environment
func parseHookInput() *hook.HookInput {
	input := os.Getenv("CLAUDE_HOOK_INPUT")
	if input == "" {
		return nil
	}
	
	debug.Log("Claude Code hook input detected")
	parser := hook.NewParser()
	hookInput, err := parser.Parse(bytes.NewReader([]byte(input)))
	if err != nil {
		debug.LogError(err, "parsing hook input")
		return nil
	}
	
	debug.Log("Session ID: %s", hookInput.SessionID)
	debug.Log("Hook Event: %s", hookInput.HookEventName)
	if hookInput.ToolUse != nil {
		debug.Log("Tool: %s", hookInput.ToolUse.Name)
	}
	
	return hookInput
}

// extractEditedFiles extracts edited files from hook input
func extractEditedFiles(hookInput *hook.HookInput) []string {
	if hookInput == nil {
		return nil
	}
	
	parser := hook.NewParser()
	files, err := parser.ExtractEditedFiles(hookInput)
	if err != nil {
		debug.LogError(err, "extracting edited files")
		return nil
	}
	
	return files
}

// executeFileAwareCommand executes command for edited files
func executeFileAwareCommand(cfg *config.Config, commandName string, extraArgs []string, editedFiles []string) ([]executor.ComponentExecResult, error) {
	debug.LogSection("File-Aware Execution")
	debug.Log("Edited files: %v", editedFiles)
	
	// Map files to components
	mapper := watcher.NewFileMapper(cfg)
	groups, err := mapper.MapFilesToComponents(editedFiles)
	if err != nil {
		debug.LogError(err, "mapping files to components")
		return nil, err
	}
	debug.Log("Mapped to %d component groups", len(groups))
	
	var results []executor.ComponentExecResult
	for _, group := range groups {
		result, err := executeComponentCommand(&group, commandName, extraArgs)
		if err != nil {
			results = append(results, executor.ComponentExecResult{
				Path:            group.Path,
				Command:         commandName,
				CommandConfig:   nil,
				ExecutionError:  err,
			})
			continue
		}
		if result != nil {
			results = append(results, *result)
		}
	}
	
	return results, nil
}

// executeComponentCommand executes command for a single component
func executeComponentCommand(group *watcher.ComponentGroup, commandName string, extraArgs []string) (*executor.ComponentExecResult, error) {
	debug.Log("Executing for component: %s", group.Path)
	
	// Get command config for this component
	var cmdConfig *config.CommandConfig
	if group.Config != nil {
		cmdConfig = group.Config[commandName]
	}
	if cmdConfig == nil {
		debug.Log("No command config for %s in component %s", commandName, group.Path)
		return nil, nil
	}
	
	// Build the command arguments
	args := make([]string, 0, len(cmdConfig.Args)+len(extraArgs))
	args = append(args, cmdConfig.Args...)
	args = append(args, extraArgs...)
	
	// Execute command
	result, err := executeWithOptions(cmdConfig, args, group.Path)
	if err != nil {
		return nil, err
	}
	
	// Apply output filtering
	filteredOutput := applyOutputFilter(cmdConfig, result)
	
	return &executor.ComponentExecResult{
		Path:           group.Path,
		Command:        commandName,
		CommandConfig:  cmdConfig,
		ExecResult:     result,
		FilteredOutput: filteredOutput,
	}, nil
}

// executeSingleCommand executes a single command
func executeSingleCommand(cmdConfig *config.CommandConfig, commandName string, extraArgs []string) ([]executor.ComponentExecResult, error) {
	debug.LogSection("Single Command Execution")
	
	// Build the command arguments
	args := make([]string, 0, len(cmdConfig.Args)+len(extraArgs))
	args = append(args, cmdConfig.Args...)
	args = append(args, extraArgs...)

	// Get working directory
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get working directory: %w", err)
	}

	debug.LogCommand(cmdConfig.Command, args, cwd)
	
	// Execute command
	execStart := time.Now()
	result, err := executeWithOptions(cmdConfig, args, cwd)
	debug.LogTiming("command execution", time.Since(execStart))
	
	if err != nil {
		return nil, fmt.Errorf("command execution failed: %w", err)
	}
	
	// Apply output filtering
	filteredOutput := applyOutputFilter(cmdConfig, result)
	
	return []executor.ComponentExecResult{
		{
			Path:           "",
			Command:        commandName,
			CommandConfig:  cmdConfig,
			ExecResult:     result,
			FilteredOutput: filteredOutput,
		},
	}, nil
}

// executeWithOptions executes command with configured options
func executeWithOptions(cmdConfig *config.CommandConfig, args []string, workingDir string) (*executor.ExecResult, error) {
	cmdExecutor := executor.NewCommandExecutor(2 * time.Minute)
	execOptions := executor.ExecOptions{
		WorkingDir: workingDir,
		InheritEnv: true,
	}
	if cmdConfig.Timeout > 0 {
		execOptions.Timeout = time.Duration(cmdConfig.Timeout) * time.Millisecond
	}
	
	return cmdExecutor.Execute(cmdConfig.Command, args, execOptions)
}

// applyOutputFilter applies output filtering to execution result
func applyOutputFilter(cmdConfig *config.CommandConfig, result *executor.ExecResult) *filter.FilteredOutput {
	if cmdConfig.OutputFilter == nil {
		return nil
	}
	
	debug.LogSection("Output Filtering")
	outputFilter := filter.NewSimpleOutputFilter()
	
	// Combine stdout and stderr for filtering
	combinedOutput := result.Stdout
	if result.Stderr != "" {
		if combinedOutput != "" {
			combinedOutput += "\n"
		}
		combinedOutput += result.Stderr
	}
	
	filterStart := time.Now()
	filteredOutput := outputFilter.FilterWithRules(combinedOutput, &filter.FilterRules{
		ErrorPatterns:   cmdConfig.OutputFilter.ErrorPatterns,
		ContextPatterns: cmdConfig.OutputFilter.IncludePatterns,
		MaxLines:        cmdConfig.OutputFilter.MaxOutput,
		ContextLines:    cmdConfig.OutputFilter.ContextLines,
	})
	debug.LogTiming("output filtering", time.Since(filterStart))
	debug.LogFilterProcess(
		strings.Count(combinedOutput, "\n")+1,
		len(filteredOutput.Lines),
		len(filteredOutput.Lines),
	)
	
	return filteredOutput
}

// reportAndOutputResults reports execution results and outputs to stdout/stderr
func reportAndOutputResults(results []executor.ComponentExecResult, start time.Time) {
	debug.LogSection("Error Reporting")
	errorReporter := reporter.NewErrorReporter()
	report := errorReporter.Report(results)
	
	debug.Log("Exit code: %d", report.ExitCode)
	debug.LogTiming("total execution", time.Since(start))
	
	// Output results
	if report.Stdout != "" {
		_, _ = fmt.Fprintln(os.Stdout, report.Stdout) //nolint:errcheck // Best effort output to stdout
	}
	if report.Stderr != "" {
		_, _ = fmt.Fprintln(os.Stderr, report.Stderr) //nolint:errcheck // Best effort output to stderr
	}
	
	if report.ExitCode != 0 {
		osExit(report.ExitCode)
	}
}