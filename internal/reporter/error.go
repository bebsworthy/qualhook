// Package reporter provides error reporting and formatting functionality for qualhook.
package reporter

import (
	"fmt"
	"strings"

	"github.com/bebsworthy/qualhook/internal/executor"
)


// ErrorReporter formats and reports errors for LLM consumption
type ErrorReporter struct {
	// Default prompt to use if not specified in config
	defaultPrompt string
}

// NewErrorReporter creates a new error reporter
func NewErrorReporter() *ErrorReporter {
	return &ErrorReporter{
		defaultPrompt: "Fix the following errors:",
	}
}

// Report aggregates results from multiple components and generates a report
func (r *ErrorReporter) Report(results []executor.ComponentExecResult) *ReportResult {
	// Check for any execution errors first
	if execErr := r.checkExecutionErrors(results); execErr != nil {
		return execErr
	}

	// Collect all components with errors
	var errorComponents []executor.ComponentExecResult
	for _, result := range results {
		if r.hasErrors(result) {
			errorComponents = append(errorComponents, result)
		}
	}

	// If no errors, return success
	if len(errorComponents) == 0 {
		return &ReportResult{
			ExitCode: 0,
			Stdout:   "All quality checks passed successfully.",
		}
	}

	// Format errors for LLM
	stderr := r.formatErrors(errorComponents)

	return &ReportResult{
		ExitCode: 2, // Exit code 2 for Claude Code hook integration
		Stderr:   stderr,
	}
}

// ReportResult contains the final report output
type ReportResult struct {
	// Exit code (0 for success, 2 for errors that LLM should fix, 1 for other errors)
	ExitCode int
	// Standard error output (for errors)
	Stderr string
	// Standard output (for success or informational messages)
	Stdout string
}

// checkExecutionErrors checks for critical execution errors that prevent normal operation
func (r *ErrorReporter) checkExecutionErrors(results []executor.ComponentExecResult) *ReportResult {
	var criticalErrors []string

	for _, result := range results {
		if result.ExecutionError != nil {
			// Handle specific error types
			var execErr *executor.ExecError
			if err, ok := result.ExecutionError.(*executor.ExecError); ok {
				execErr = err
			} else {
				execErr = executor.ClassifyError(result.ExecutionError, result.Command, nil)
			}

			msg := r.formatExecutionError(result, execErr)
			criticalErrors = append(criticalErrors, msg)
		}
	}

	if len(criticalErrors) > 0 {
		return &ReportResult{
			ExitCode: 1, // Exit code 1 for configuration/execution errors
			Stderr:   fmt.Sprintf("[QUALHOOK ERROR] Execution Error\n\n%s", strings.Join(criticalErrors, "\n\n")),
		}
	}

	return nil
}

// formatExecutionError formats an execution error for output
func (r *ErrorReporter) formatExecutionError(result executor.ComponentExecResult, execErr *executor.ExecError) string {
	var msg strings.Builder

	if result.Path != "" {
		msg.WriteString(fmt.Sprintf("Component: %s\n", result.Path))
	}
	msg.WriteString(fmt.Sprintf("Command: %s\n", result.Command))

	switch execErr.Type {
	case executor.ErrorTypeCommandNotFound:
		msg.WriteString("Error: Command not found\n")
		msg.WriteString(fmt.Sprintf("Details: The command '%s' is not installed or not in PATH\n", execErr.Command))
		msg.WriteString("Fix: Ensure the required tool is installed and accessible")
	case executor.ErrorTypePermissionDenied:
		msg.WriteString("Error: Permission denied\n")
		msg.WriteString("Details: Insufficient permissions to execute the command\n")
		msg.WriteString("Fix: Check file permissions and user privileges")
	case executor.ErrorTypeTimeout:
		msg.WriteString("Error: Command timed out\n")
		msg.WriteString("Details: The command exceeded the configured timeout\n")
		msg.WriteString("Fix: Increase timeout in configuration or optimize the command")
	case executor.ErrorTypeWorkingDirectory:
		msg.WriteString("Error: Working directory error\n")
		msg.WriteString(fmt.Sprintf("Details: %s\n", execErr.Details))
		msg.WriteString("Fix: Ensure the working directory exists and is accessible")
	default:
		msg.WriteString(fmt.Sprintf("Error: %v\n", execErr.Err))
	}

	return msg.String()
}

// hasErrors checks if a component result contains errors
func (r *ErrorReporter) hasErrors(result executor.ComponentExecResult) bool {
	// Check exit code
	if result.CommandConfig != nil && result.CommandConfig.ErrorDetection != nil {
		for _, code := range result.CommandConfig.ErrorDetection.ExitCodes {
			if result.ExecResult.ExitCode == code {
				return true
			}
		}
	}

	// Check filtered output
	if result.FilteredOutput != nil && result.FilteredOutput.HasErrors {
		return true
	}

	// If no error detection configured, assume non-zero exit code is an error
	if result.CommandConfig == nil || result.CommandConfig.ErrorDetection == nil {
		return result.ExecResult.ExitCode != 0
	}

	return false
}

// formatErrors formats error output for LLM consumption
func (r *ErrorReporter) formatErrors(errorComponents []executor.ComponentExecResult) string {
	var output strings.Builder

	// Group by command type for better organization
	commandGroups := r.groupByCommand(errorComponents)

	for command, components := range commandGroups {
		// Get prompt for this command
		prompt := r.getPrompt(command, components)
		output.WriteString(prompt)
		output.WriteString("\n\n")

		// Format errors for each component
		for i, component := range components {
			if i > 0 {
				output.WriteString("\n---\n\n")
			}

			if len(components) > 1 && component.Path != "" {
				output.WriteString(fmt.Sprintf("## %s\n\n", component.Path))
			}

			// Add filtered output
			if component.FilteredOutput != nil && len(component.FilteredOutput.Lines) > 0 {
				for _, line := range component.FilteredOutput.Lines {
					output.WriteString(line)
					output.WriteString("\n")
				}

				if component.FilteredOutput.Truncated {
					output.WriteString(fmt.Sprintf("\n[Output truncated - %d total lines]\n", 
						component.FilteredOutput.TotalLines))
				}
			} else if component.ExecResult != nil {
				// Fallback to raw output if no filtering applied
				if component.ExecResult.Stderr != "" {
					output.WriteString(component.ExecResult.Stderr)
					if !strings.HasSuffix(component.ExecResult.Stderr, "\n") {
						output.WriteString("\n")
					}
				} else if component.ExecResult.Stdout != "" {
					// Some tools output errors to stdout
					output.WriteString(component.ExecResult.Stdout)
					if !strings.HasSuffix(component.ExecResult.Stdout, "\n") {
						output.WriteString("\n")
					}
				}
			}
		}

		output.WriteString("\n")
	}

	return strings.TrimSpace(output.String())
}

// groupByCommand groups components by their command type
func (r *ErrorReporter) groupByCommand(components []executor.ComponentExecResult) map[string][]executor.ComponentExecResult {
	groups := make(map[string][]executor.ComponentExecResult)
	
	for _, component := range components {
		groups[component.Command] = append(groups[component.Command], component)
	}
	
	return groups
}

// getPrompt returns the appropriate prompt for a command
func (r *ErrorReporter) getPrompt(command string, components []executor.ComponentExecResult) string {
	// Check if any component has a custom prompt
	for _, component := range components {
		if component.CommandConfig != nil && component.CommandConfig.Prompt != "" {
			return component.CommandConfig.Prompt
		}
	}

	// Use command-specific defaults
	switch command {
	case "format":
		return "Fix the formatting issues below:"
	case "lint":
		return "Fix the linting errors below:"
	case "typecheck":
		return "Fix the type errors below:"
	case "test":
		return "Fix the failing tests below:"
	default:
		return r.defaultPrompt
	}
}

// ReportSingleError creates a report for a single error message
func (r *ErrorReporter) ReportSingleError(errorType string, message string, details ...string) *ReportResult {
	var stderr strings.Builder
	
	stderr.WriteString(fmt.Sprintf("[QUALHOOK ERROR] %s: %s\n", errorType, message))
	
	if len(details) > 0 {
		stderr.WriteString("\nDetails:\n")
		for _, detail := range details {
			stderr.WriteString(fmt.Sprintf("- %s\n", detail))
		}
	}
	
	stderr.WriteString("\nDebug with: qualhook --debug <command>")
	
	return &ReportResult{
		ExitCode: 1,
		Stderr:   stderr.String(),
	}
}