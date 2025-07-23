// Package executor provides file-aware command execution functionality.
package executor

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/bebsworthy/qualhook/internal/filter"
	"github.com/bebsworthy/qualhook/internal/hook"
	"github.com/bebsworthy/qualhook/internal/watcher"
	"github.com/bebsworthy/qualhook/pkg/config"
)

// ComponentExecResult represents the result of executing a command for a component
type ComponentExecResult struct {
	// Component path (for monorepo support)
	Path string
	// Command that was executed
	Command string
	// Files that triggered this execution
	Files []string
	// Execution result
	ExecResult *ExecResult
	// Filtered output
	FilteredOutput *filter.FilteredOutput
	// Command configuration used
	CommandConfig *config.CommandConfig
	// Any execution error (distinct from command errors)
	ExecutionError error
}

// FileAwareExecutor executes commands based on edited files
type FileAwareExecutor struct {
	commandExecutor  *CommandExecutor
	parallelExecutor *ParallelExecutor
	mapper           *watcher.FileMapper
	hookParser       *hook.Parser
	debugMode        bool
}

// NewFileAwareExecutor creates a new file-aware executor
func NewFileAwareExecutor(cfg *config.Config, debugMode bool) *FileAwareExecutor {
	defaultTimeout := 2 * time.Minute
	commandExecutor := NewCommandExecutor(defaultTimeout)
	
	return &FileAwareExecutor{
		commandExecutor:  commandExecutor,
		parallelExecutor: NewParallelExecutor(commandExecutor, 4), // Default max concurrent
		mapper:           watcher.NewFileMapper(cfg),
		hookParser:       hook.NewParser(),
		debugMode:        debugMode,
	}
}

// ExecuteForEditedFiles executes the appropriate commands based on edited files
func (e *FileAwareExecutor) ExecuteForEditedFiles(hookInput *hook.HookInput, commandName string, extraArgs []string) ([]ComponentExecResult, error) {
	// Extract edited files from hook input
	editedFiles, err := e.hookParser.ExtractEditedFiles(hookInput)
	if err != nil {
		return nil, fmt.Errorf("failed to extract edited files: %w", err)
	}

	if e.debugMode {
		fmt.Printf("[DEBUG] Edited files: %v\n", editedFiles)
	}

	// If no files were edited, run command on root
	if len(editedFiles) == 0 {
		return e.executeForRootComponent(commandName, extraArgs)
	}

	// Map files to components
	componentGroups, err := e.mapper.MapFilesToComponents(editedFiles)
	if err != nil {
		return nil, fmt.Errorf("failed to map files to components: %w", err)
	}

	e.debugLogComponentGroups(componentGroups)

	// Execute commands for each component
	results, err := e.executeForComponents(componentGroups, commandName, extraArgs)
	if err != nil {
		return nil, err
	}

	// Report execution summary
	e.debugLogExecutionSummary(results, commandName)

	return results, nil
}

// executeForComponent executes a command for a specific component
func (e *FileAwareExecutor) executeForComponent(componentPath string, files []string, cmdConfig *config.CommandConfig, commandName string, extraArgs []string) (ComponentExecResult, error) {
	result := ComponentExecResult{
		Path:    componentPath,
		Command: commandName,
		Files:   files,
	}

	// If no command config provided, return empty result
	if cmdConfig == nil {
		result.ExecutionError = fmt.Errorf("no command configuration provided for component %s", componentPath)
		return result, result.ExecutionError
	}

	result.CommandConfig = cmdConfig

	// Build the command arguments
	args := make([]string, 0, len(cmdConfig.Args)+len(extraArgs))
	args = append(args, cmdConfig.Args...)
	args = append(args, extraArgs...)

	// Set working directory for the command
	workingDir := ""
	if componentPath != "." {
		// For path components like "frontend/**", we don't change directory
		// The commands should be configured with appropriate paths
		// For example: npm run lint --prefix frontend
		workingDir = ""
	}

	// Execute the command
	execOptions := ExecOptions{
		WorkingDir: workingDir,
		InheritEnv: true,
		Timeout:    time.Duration(cmdConfig.Timeout) * time.Millisecond,
	}

	if execOptions.Timeout == 0 {
		execOptions.Timeout = 2 * time.Minute
	}

	execResult, err := e.commandExecutor.Execute(cmdConfig.Command, args, execOptions)
	if err != nil {
		result.ExecutionError = fmt.Errorf("failed to execute command: %w", err)
		return result, result.ExecutionError
	}
	result.ExecResult = execResult

	// Create output filter
	outputFilter, err := filter.NewOutputFilter(cmdConfig.OutputFilter)
	if err != nil {
		result.ExecutionError = fmt.Errorf("failed to create output filter: %w", err)
		return result, result.ExecutionError
	}

	// Filter the output
	filteredOutput := outputFilter.FilterBoth(execResult.Stdout, execResult.Stderr)
	result.FilteredOutput = filteredOutput

	return result, nil
}

// ExecuteForAllComponents executes a command for all configured components
func (e *FileAwareExecutor) ExecuteForAllComponents(commandName string, extraArgs []string) ([]ComponentExecResult, error) {
	// Get all components
	allComponents := e.mapper.ListAllComponents()
	
	var allFiles []string
	for _, component := range allComponents {
		// Create a dummy file for each component to trigger execution
		if component == "." {
			allFiles = append(allFiles, "dummy.txt")
		} else {
			allFiles = append(allFiles, filepath.Join(strings.TrimSuffix(component, "/**"), "dummy.txt"))
		}
	}

	// Map to component groups
	componentGroups, err := e.mapper.MapFilesToComponents(allFiles)
	if err != nil {
		return nil, fmt.Errorf("failed to map components: %w", err)
	}

	// Execute for each component
	var results []ComponentExecResult

	for _, group := range componentGroups {
		cmdConfig, exists := group.Config[commandName]
		if !exists {
			continue
		}

		result, err := e.executeForComponent(group.Path, group.Files, cmdConfig, commandName, extraArgs)
		if err != nil {
			return nil, err
		}

		results = append(results, result)
	}

	return results, nil
}

// getStatusText returns a status text for debug output
func getStatusText(output *filter.FilteredOutput) string {
	if output == nil || !output.HasErrors {
		return "✓ passed"
	}
	return "✗ failed"
}

// executeForRootComponent executes command on root when no files were edited
func (e *FileAwareExecutor) executeForRootComponent(commandName string, extraArgs []string) ([]ComponentExecResult, error) {
	// Get root configuration
	groups, err := e.mapper.MapFilesToComponents([]string{"dummy.txt"})
	if err != nil {
		return nil, fmt.Errorf("failed to get root configuration: %w", err)
	}
	
	if len(groups) > 0 && groups[0].Path == "." {
		cmdConfig, exists := groups[0].Config[commandName]
		if !exists {
			// No command configured
			return []ComponentExecResult{}, nil
		}
		
		result, err := e.executeForComponent(".", nil, cmdConfig, commandName, extraArgs)
		if err != nil {
			return nil, err
		}
		return []ComponentExecResult{result}, nil
	}
	
	// Fallback
	return []ComponentExecResult{}, nil
}

// debugLogComponentGroups logs component groups in debug mode
func (e *FileAwareExecutor) debugLogComponentGroups(componentGroups []watcher.ComponentGroup) {
	if !e.debugMode {
		return
	}
	
	fmt.Printf("[DEBUG] Component groups:\n")
	for _, group := range componentGroups {
		fmt.Printf("  - %s: %v\n", group.Path, group.Files)
	}
}

// executeForComponents executes commands for all component groups
func (e *FileAwareExecutor) executeForComponents(componentGroups []watcher.ComponentGroup, commandName string, extraArgs []string) ([]ComponentExecResult, error) {
	var results []ComponentExecResult

	for _, group := range componentGroups {
		// Check if this component has the requested command
		cmdConfig, exists := group.Config[commandName]
		if !exists {
			if e.debugMode {
				fmt.Printf("[DEBUG] Skipping component %s - command %q not configured\n", group.Path, commandName)
			}
			continue
		}

		// Execute the command for this component
		result, err := e.executeForComponent(group.Path, group.Files, cmdConfig, commandName, extraArgs)
		if err != nil {
			return nil, fmt.Errorf("failed to execute command for component %s: %w", group.Path, err)
		}

		results = append(results, result)

		// If any component has errors, we stop here (fail fast)
		if result.ExecResult != nil && result.ExecResult.ExitCode != 0 {
			break
		}
	}

	return results, nil
}

// debugLogExecutionSummary logs execution summary in debug mode
func (e *FileAwareExecutor) debugLogExecutionSummary(results []ComponentExecResult, commandName string) {
	if !e.debugMode || len(results) == 0 {
		return
	}
	
	fmt.Printf("\n[DEBUG] Execution summary:\n")
	for _, result := range results {
		fmt.Printf("  - Component %s: %s command %s (files: %v)\n", 
			result.Path, 
			commandName,
			getStatusText(result.FilteredOutput),
			result.Files)
	}
}