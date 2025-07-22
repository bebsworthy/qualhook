package reporter

import (
	"errors"
	"strings"
	"testing"

	"github.com/qualhook/qualhook/internal/executor"
	"github.com/qualhook/qualhook/internal/filter"
	"github.com/qualhook/qualhook/pkg/config"
)

func TestNewErrorReporter(t *testing.T) {
	reporter := NewErrorReporter()
	if reporter == nil {
		t.Fatal("NewErrorReporter returned nil")
	}
	if reporter.defaultPrompt != "Fix the following errors:" {
		t.Errorf("unexpected default prompt: %q", reporter.defaultPrompt)
	}
}

func TestReport_NoErrors(t *testing.T) {
	reporter := NewErrorReporter()
	
	results := []ComponentResult{
		{
			Command: "lint",
			ExecResult: &executor.ExecResult{
				ExitCode: 0,
				Stdout:   "All checks passed",
			},
			FilteredOutput: &filter.FilteredOutput{
				HasErrors: false,
			},
			Config: &config.CommandConfig{
				ErrorDetection: &config.ErrorDetection{
					ExitCodes: []int{1},
				},
			},
		},
	}
	
	report := reporter.Report(results)
	
	if report.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", report.ExitCode)
	}
	if report.Stderr != "" {
		t.Errorf("expected empty stderr, got %q", report.Stderr)
	}
	if !strings.Contains(report.Stdout, "successfully") {
		t.Errorf("expected success message, got %q", report.Stdout)
	}
}

func TestReport_WithErrors(t *testing.T) {
	reporter := NewErrorReporter()
	
	results := []ComponentResult{
		{
			Command: "lint",
			ExecResult: &executor.ExecResult{
				ExitCode: 1,
				Stderr:   "file.js:10:5: error: Missing semicolon",
			},
			FilteredOutput: &filter.FilteredOutput{
				Lines:     []string{"file.js:10:5: error: Missing semicolon"},
				HasErrors: true,
			},
			Config: &config.CommandConfig{
				ErrorDetection: &config.ErrorDetection{
					ExitCodes: []int{1},
				},
				Prompt: "Fix the linting errors below:",
			},
		},
	}
	
	report := reporter.Report(results)
	
	if report.ExitCode != 2 {
		t.Errorf("expected exit code 2 for LLM errors, got %d", report.ExitCode)
	}
	if report.Stdout != "" {
		t.Errorf("expected empty stdout, got %q", report.Stdout)
	}
	if !strings.Contains(report.Stderr, "Fix the linting errors below:") {
		t.Errorf("expected prompt in stderr, got %q", report.Stderr)
	}
	if !strings.Contains(report.Stderr, "Missing semicolon") {
		t.Errorf("expected error message in stderr, got %q", report.Stderr)
	}
}

func TestReport_ExecutionError(t *testing.T) {
	reporter := NewErrorReporter()
	
	execErr := &executor.ExecError{
		Type:    executor.ErrorTypeCommandNotFound,
		Command: "eslint",
		Err:     errors.New("command not found"),
	}
	
	results := []ComponentResult{
		{
			Command:        "lint",
			ExecutionError: execErr,
		},
	}
	
	report := reporter.Report(results)
	
	if report.ExitCode != 1 {
		t.Errorf("expected exit code 1 for execution errors, got %d", report.ExitCode)
	}
	if !strings.Contains(report.Stderr, "[QUALHOOK ERROR]") {
		t.Errorf("expected error prefix in stderr, got %q", report.Stderr)
	}
	if !strings.Contains(report.Stderr, "Command not found") {
		t.Errorf("expected error message in stderr, got %q", report.Stderr)
	}
	if !strings.Contains(report.Stderr, "eslint") {
		t.Errorf("expected command name in stderr, got %q", report.Stderr)
	}
}

func TestReport_MonorepoMultipleComponents(t *testing.T) {
	reporter := NewErrorReporter()
	
	results := []ComponentResult{
		{
			Path:    "frontend",
			Command: "lint",
			ExecResult: &executor.ExecResult{
				ExitCode: 1,
				Stderr:   "frontend/app.js:5: error",
			},
			FilteredOutput: &filter.FilteredOutput{
				Lines:     []string{"frontend/app.js:5: error"},
				HasErrors: true,
			},
			Config: &config.CommandConfig{
				ErrorDetection: &config.ErrorDetection{
					ExitCodes: []int{1},
				},
			},
		},
		{
			Path:    "backend",
			Command: "lint",
			ExecResult: &executor.ExecResult{
				ExitCode: 1,
				Stderr:   "backend/server.go:10: error",
			},
			FilteredOutput: &filter.FilteredOutput{
				Lines:     []string{"backend/server.go:10: error"},
				HasErrors: true,
			},
			Config: &config.CommandConfig{
				ErrorDetection: &config.ErrorDetection{
					ExitCodes: []int{1},
				},
			},
		},
	}
	
	report := reporter.Report(results)
	
	if report.ExitCode != 2 {
		t.Errorf("expected exit code 2, got %d", report.ExitCode)
	}
	if !strings.Contains(report.Stderr, "frontend") {
		t.Errorf("expected frontend path in output, got %q", report.Stderr)
	}
	if !strings.Contains(report.Stderr, "backend") {
		t.Errorf("expected backend path in output, got %q", report.Stderr)
	}
	if !strings.Contains(report.Stderr, "---") {
		t.Errorf("expected separator between components, got %q", report.Stderr)
	}
}

func TestReport_TruncatedOutput(t *testing.T) {
	reporter := NewErrorReporter()
	
	results := []ComponentResult{
		{
			Command: "test",
			ExecResult: &executor.ExecResult{
				ExitCode: 1,
			},
			FilteredOutput: &filter.FilteredOutput{
				Lines:      []string{"Test failed: assertion error"},
				HasErrors:  true,
				Truncated:  true,
				TotalLines: 500,
			},
			Config: &config.CommandConfig{
				ErrorDetection: &config.ErrorDetection{
					ExitCodes: []int{1},
				},
			},
		},
	}
	
	report := reporter.Report(results)
	
	if !strings.Contains(report.Stderr, "Output truncated") {
		t.Errorf("expected truncation message, got %q", report.Stderr)
	}
	if !strings.Contains(report.Stderr, "500 total lines") {
		t.Errorf("expected total line count, got %q", report.Stderr)
	}
}

func TestHasErrors(t *testing.T) {
	reporter := NewErrorReporter()
	
	tests := []struct {
		name     string
		result   ComponentResult
		expected bool
	}{
		{
			name: "exit code match",
			result: ComponentResult{
				ExecResult: &executor.ExecResult{ExitCode: 1},
				Config: &config.CommandConfig{
					ErrorDetection: &config.ErrorDetection{
						ExitCodes: []int{1, 2},
					},
				},
			},
			expected: true,
		},
		{
			name: "exit code no match",
			result: ComponentResult{
				ExecResult: &executor.ExecResult{ExitCode: 0},
				Config: &config.CommandConfig{
					ErrorDetection: &config.ErrorDetection{
						ExitCodes: []int{1, 2},
					},
				},
			},
			expected: false,
		},
		{
			name: "filtered output has errors",
			result: ComponentResult{
				ExecResult:     &executor.ExecResult{ExitCode: 0},
				FilteredOutput: &filter.FilteredOutput{HasErrors: true},
			},
			expected: true,
		},
		{
			name: "no config non-zero exit",
			result: ComponentResult{
				ExecResult: &executor.ExecResult{ExitCode: 1},
			},
			expected: true,
		},
		{
			name: "no config zero exit",
			result: ComponentResult{
				ExecResult: &executor.ExecResult{ExitCode: 0},
			},
			expected: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := reporter.hasErrors(tt.result)
			if got != tt.expected {
				t.Errorf("hasErrors() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGetPrompt(t *testing.T) {
	reporter := NewErrorReporter()
	
	tests := []struct {
		name       string
		command    string
		components []ComponentResult
		expected   string
	}{
		{
			name:    "custom prompt",
			command: "lint",
			components: []ComponentResult{
				{Config: &config.CommandConfig{Prompt: "Custom prompt:"}},
			},
			expected: "Custom prompt:",
		},
		{
			name:       "format default",
			command:    "format",
			components: []ComponentResult{{}},
			expected:   "Fix the formatting issues below:",
		},
		{
			name:       "lint default",
			command:    "lint",
			components: []ComponentResult{{}},
			expected:   "Fix the linting errors below:",
		},
		{
			name:       "typecheck default",
			command:    "typecheck",
			components: []ComponentResult{{}},
			expected:   "Fix the type errors below:",
		},
		{
			name:       "test default",
			command:    "test",
			components: []ComponentResult{{}},
			expected:   "Fix the failing tests below:",
		},
		{
			name:       "unknown command",
			command:    "custom",
			components: []ComponentResult{{}},
			expected:   "Fix the following errors:",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := reporter.getPrompt(tt.command, tt.components)
			if got != tt.expected {
				t.Errorf("getPrompt() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestReportSingleError(t *testing.T) {
	reporter := NewErrorReporter()
	
	report := reporter.ReportSingleError(
		"Configuration Error",
		"Invalid JSON syntax",
		"Check line 10 of qualhook.json",
		"Ensure proper comma placement",
	)
	
	if report.ExitCode != 1 {
		t.Errorf("expected exit code 1, got %d", report.ExitCode)
	}
	if !strings.Contains(report.Stderr, "[QUALHOOK ERROR]") {
		t.Errorf("expected error prefix, got %q", report.Stderr)
	}
	if !strings.Contains(report.Stderr, "Configuration Error") {
		t.Errorf("expected error type, got %q", report.Stderr)
	}
	if !strings.Contains(report.Stderr, "Invalid JSON syntax") {
		t.Errorf("expected error message, got %q", report.Stderr)
	}
	if !strings.Contains(report.Stderr, "Check line 10") {
		t.Errorf("expected first detail, got %q", report.Stderr)
	}
	if !strings.Contains(report.Stderr, "Ensure proper comma") {
		t.Errorf("expected second detail, got %q", report.Stderr)
	}
	if !strings.Contains(report.Stderr, "Debug with:") {
		t.Errorf("expected debug hint, got %q", report.Stderr)
	}
}

func TestFormatExecutionError(t *testing.T) {
	reporter := NewErrorReporter()
	
	tests := []struct {
		name     string
		result   ComponentResult
		execErr  *executor.ExecError
		contains []string
	}{
		{
			name: "command not found",
			result: ComponentResult{
				Path:    "frontend",
				Command: "npm run lint",
			},
			execErr: &executor.ExecError{
				Type:    executor.ErrorTypeCommandNotFound,
				Command: "npm",
			},
			contains: []string{
				"Component: frontend",
				"Command: npm run lint",
				"Error: Command not found",
				"not installed or not in PATH",
			},
		},
		{
			name: "permission denied",
			result: ComponentResult{
				Command: "./script.sh",
			},
			execErr: &executor.ExecError{
				Type: executor.ErrorTypePermissionDenied,
			},
			contains: []string{
				"Error: Permission denied",
				"Check file permissions",
			},
		},
		{
			name: "timeout",
			result: ComponentResult{
				Command: "test",
			},
			execErr: &executor.ExecError{
				Type: executor.ErrorTypeTimeout,
			},
			contains: []string{
				"Error: Command timed out",
				"Increase timeout in configuration",
			},
		},
		{
			name: "working directory",
			result: ComponentResult{
				Command: "build",
			},
			execErr: &executor.ExecError{
				Type:    executor.ErrorTypeWorkingDirectory,
				Details: "/path/to/dir does not exist",
			},
			contains: []string{
				"Error: Working directory error",
				"/path/to/dir does not exist",
				"Ensure the working directory exists",
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := reporter.formatExecutionError(tt.result, tt.execErr)
			for _, expected := range tt.contains {
				if !strings.Contains(msg, expected) {
					t.Errorf("expected %q in output, got:\n%s", expected, msg)
				}
			}
		})
	}
}

func TestGroupByCommand(t *testing.T) {
	reporter := NewErrorReporter()
	
	components := []ComponentResult{
		{Command: "lint", Path: "frontend"},
		{Command: "test", Path: "backend"},
		{Command: "lint", Path: "backend"},
		{Command: "format", Path: "shared"},
	}
	
	groups := reporter.groupByCommand(components)
	
	if len(groups) != 3 {
		t.Errorf("expected 3 groups, got %d", len(groups))
	}
	
	if len(groups["lint"]) != 2 {
		t.Errorf("expected 2 lint components, got %d", len(groups["lint"]))
	}
	
	if len(groups["test"]) != 1 {
		t.Errorf("expected 1 test component, got %d", len(groups["test"]))
	}
	
	if len(groups["format"]) != 1 {
		t.Errorf("expected 1 format component, got %d", len(groups["format"]))
	}
}

func TestReport_FallbackToRawOutput(t *testing.T) {
	reporter := NewErrorReporter()
	
	tests := []struct {
		name     string
		result   ComponentResult
		expected string
	}{
		{
			name: "stderr output",
			result: ComponentResult{
				Command: "lint",
				ExecResult: &executor.ExecResult{
					ExitCode: 1,
					Stderr:   "Error on line 10",
				},
				Config: &config.CommandConfig{
					ErrorDetection: &config.ErrorDetection{
						ExitCodes: []int{1},
					},
				},
			},
			expected: "Error on line 10",
		},
		{
			name: "stdout output when no stderr",
			result: ComponentResult{
				Command: "test",
				ExecResult: &executor.ExecResult{
					ExitCode: 1,
					Stdout:   "Test failed: assertion error",
				},
				Config: &config.CommandConfig{
					ErrorDetection: &config.ErrorDetection{
						ExitCodes: []int{1},
					},
				},
			},
			expected: "Test failed: assertion error",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := reporter.Report([]ComponentResult{tt.result})
			if !strings.Contains(report.Stderr, tt.expected) {
				t.Errorf("expected %q in stderr, got %q", tt.expected, report.Stderr)
			}
		})
	}
}

func TestReport_MixedResults(t *testing.T) {
	reporter := NewErrorReporter()
	
	results := []ComponentResult{
		{
			Command: "lint",
			ExecResult: &executor.ExecResult{
				ExitCode: 0,
			},
			Config: &config.CommandConfig{
				ErrorDetection: &config.ErrorDetection{
					ExitCodes: []int{1},
				},
			},
		},
		{
			Command: "test",
			ExecResult: &executor.ExecResult{
				ExitCode: 1,
				Stderr:   "Test failed",
			},
			FilteredOutput: &filter.FilteredOutput{
				Lines:     []string{"Test failed: assertion error"},
				HasErrors: true,
			},
			Config: &config.CommandConfig{
				ErrorDetection: &config.ErrorDetection{
					ExitCodes: []int{1},
				},
			},
		},
	}
	
	report := reporter.Report(results)
	
	// Should report errors despite one success
	if report.ExitCode != 2 {
		t.Errorf("expected exit code 2, got %d", report.ExitCode)
	}
	if !strings.Contains(report.Stderr, "Test failed") {
		t.Errorf("expected test error in output, got %q", report.Stderr)
	}
	// Should not include the successful lint command
	if strings.Contains(report.Stderr, "lint") {
		t.Errorf("should not include successful command in error output, got %q", report.Stderr)
	}
}