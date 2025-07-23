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
	var hookInput *hook.HookInput
	if input := os.Getenv("CLAUDE_HOOK_INPUT"); input != "" {
		debug.Log("Claude Code hook input detected")
		parser := hook.NewParser()
		var err error
		hookInput, err = parser.Parse(bytes.NewReader([]byte(input)))
		if err != nil {
			debug.LogError(err, "parsing hook input")
		} else {
			debug.Log("Session ID: %s", hookInput.SessionID)
			debug.Log("Hook Event: %s", hookInput.HookEventName)
			if hookInput.ToolUse != nil {
				debug.Log("Tool: %s", hookInput.ToolUse.Name)
			}
		}
	}

	// Determine execution mode
	var results []executor.ComponentExecResult
	
	// Extract edited files if available
	var editedFiles []string
	if hookInput != nil {
		parser := hook.NewParser()
		files, err := parser.ExtractEditedFiles(hookInput)
		if err != nil {
			debug.LogError(err, "extracting edited files")
		} else {
			editedFiles = files
		}
	}
	
	if len(editedFiles) > 0 {
		// File-aware execution
		debug.LogSection("File-Aware Execution")
		debug.Log("Edited files: %v", editedFiles)
		
		// Map files to components
		mapper := watcher.NewFileMapper(cfg)
		groups, err := mapper.MapFilesToComponents(editedFiles)
		if err != nil {
			debug.LogError(err, "mapping files to components")
			return err
		}
		debug.Log("Mapped to %d component groups", len(groups))
		
		// Execute for each component
		for _, group := range groups {
			debug.Log("Executing for component: %s", group.Path)
			
			// Get command config for this component
			var cmdConfig *config.CommandConfig
			if group.Config != nil {
				cmdConfig = group.Config[commandName]
			}
			if cmdConfig == nil {
				debug.Log("No command config for %s in component %s", commandName, group.Path)
				continue
			}
			
			// Build the command arguments
			args := make([]string, 0, len(cmdConfig.Args)+len(extraArgs))
			args = append(args, cmdConfig.Args...)
			args = append(args, extraArgs...)
			
			// Execute command
			cmdExecutor := executor.NewCommandExecutor(2 * time.Minute)
			execOptions := executor.ExecOptions{
				WorkingDir: group.Path,
				InheritEnv: true,
			}
			if cmdConfig.Timeout > 0 {
				execOptions.Timeout = time.Duration(cmdConfig.Timeout) * time.Millisecond
			}
			
			result, err := cmdExecutor.Execute(cmdConfig.Command, args, execOptions)
			if err != nil {
				results = append(results, executor.ComponentExecResult{
					Path:            group.Path,
					Command:         commandName,
					CommandConfig:   cmdConfig,
					ExecutionError:  err,
				})
				continue
			}
			
			// Apply output filtering
			var filteredOutput *filter.FilteredOutput
			if cmdConfig.OutputFilter != nil {
				outputFilter := filter.NewSimpleOutputFilter()
				combinedOutput := result.Stdout
				if result.Stderr != "" {
					if combinedOutput != "" {
						combinedOutput += "\n"
					}
					combinedOutput += result.Stderr
				}
				
				filteredOutput = outputFilter.FilterWithRules(combinedOutput, &filter.FilterRules{
					ErrorPatterns:   cmdConfig.OutputFilter.ErrorPatterns,
					ContextPatterns: cmdConfig.OutputFilter.IncludePatterns,
					MaxLines:        cmdConfig.OutputFilter.MaxOutput,
					ContextLines:    cmdConfig.OutputFilter.ContextLines,
				})
			}
			
			results = append(results, executor.ComponentExecResult{
				Path:           group.Path,
				Command:        commandName,
				CommandConfig:  cmdConfig,
				ExecResult:     result,
				FilteredOutput: filteredOutput,
			})
		}
	} else {
		// Single execution
		debug.LogSection("Single Command Execution")
		
		// Build the command arguments
		args := make([]string, 0, len(cmdConfig.Args)+len(extraArgs))
		args = append(args, cmdConfig.Args...)
		args = append(args, extraArgs...)

		// Get working directory
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
		}

		debug.LogCommand(cmdConfig.Command, args, cwd)
		
		// Execute command
		cmdExecutor := executor.NewCommandExecutor(2 * time.Minute)
		execOptions := executor.ExecOptions{
			WorkingDir: cwd,
			InheritEnv: true,
		}
		if cmdConfig.Timeout > 0 {
			execOptions.Timeout = time.Duration(cmdConfig.Timeout) * time.Millisecond
		}
		
		execStart := time.Now()
		result, err := cmdExecutor.Execute(cmdConfig.Command, args, execOptions)
		debug.LogTiming("command execution", time.Since(execStart))
		
		if err != nil {
			return fmt.Errorf("command execution failed: %w", err)
		}
		
		// Apply output filtering
		var filteredOutput *filter.FilteredOutput
		if cmdConfig.OutputFilter != nil {
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
			filteredOutput = outputFilter.FilterWithRules(combinedOutput, &filter.FilterRules{
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
		}
		
		results = []executor.ComponentExecResult{
			{
				Path:           "",
				Command:        commandName,
				CommandConfig:  cmdConfig,
				ExecResult:     result,
				FilteredOutput: filteredOutput,
			},
		}
	}
	
	// Report results
	debug.LogSection("Error Reporting")
	errorReporter := reporter.NewErrorReporter()
	report := errorReporter.Report(results)
	
	debug.Log("Exit code: %d", report.ExitCode)
	debug.LogTiming("total execution", time.Since(start))
	
	// Output results
	if report.Stdout != "" {
		_, _ = fmt.Fprintln(os.Stdout, report.Stdout)
	}
	if report.Stderr != "" {
		_, _ = fmt.Fprintln(os.Stderr, report.Stderr)
	}
	
	if report.ExitCode != 0 {
		osExit(report.ExitCode)
	}
	
	return nil
}