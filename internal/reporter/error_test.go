//go:build unit

package reporter

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/bebsworthy/qualhook/internal/executor"
	"github.com/bebsworthy/qualhook/internal/filter"
	"github.com/bebsworthy/qualhook/pkg/config"
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

func TestReport_Scenarios(t *testing.T) {
	reporter := NewErrorReporter()

	tests := []struct {
		name          string
		results       []executor.ComponentExecResult
		wantExitCode  int
		wantInStdout  []string
		wantInStderr  []string
		notWantStderr []string
	}{
		{
			name: "no errors",
			results: []executor.ComponentExecResult{
				{
					Path:    ".",
					Command: "lint",
					ExecResult: &executor.ExecResult{
						ExitCode: 0,
						Stdout:   "All checks passed",
					},
					CommandConfig: &config.CommandConfig{
						ExitCodes: []int{1},
					},
				},
			},
			wantExitCode: 0,
			wantInStdout: []string{"All quality checks passed"},
		},
		{
			name: "with errors",
			results: []executor.ComponentExecResult{
				{
					Path:    ".",
					Command: "lint",
					ExecResult: &executor.ExecResult{
						ExitCode: 1,
						Stderr:   "file.js:10:5: error: Missing semicolon",
					},
					FilteredOutput: &filter.FilteredOutput{
						Lines:     []string{"file.js:10:5: error: Missing semicolon"},
						HasErrors: true,
					},
					CommandConfig: &config.CommandConfig{
						ExitCodes: []int{1},
						Prompt:    "Fix the linting errors below:",
					},
				},
			},
			wantExitCode: 2,
			wantInStderr: []string{
				"Fix the linting errors below:",
				"Missing semicolon",
			},
		},
		{
			name: "execution error",
			results: []executor.ComponentExecResult{
				{
					Path:    ".",
					Command: "lint",
					ExecutionError: &executor.ExecError{
						Type:    executor.ErrorTypeCommandNotFound,
						Command: "eslint",
						Err:     errors.New("command not found"),
					},
				},
			},
			wantExitCode: 1,
			wantInStderr: []string{
				"[QUALHOOK ERROR]",
				"Command not found",
				"eslint",
			},
		},
		{
			name: "monorepo multiple components",
			results: []executor.ComponentExecResult{
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
					CommandConfig: &config.CommandConfig{
						ExitCodes: []int{1},
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
					CommandConfig: &config.CommandConfig{
						ExitCodes: []int{1},
					},
				},
			},
			wantExitCode: 2,
			wantInStderr: []string{
				"frontend",
				"backend",
				"---",
			},
		},
		{
			name: "truncated output",
			results: []executor.ComponentExecResult{
				{
					Path:    ".",
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
					CommandConfig: &config.CommandConfig{
						ExitCodes: []int{1},
					},
				},
			},
			wantExitCode: 2,
			wantInStderr: []string{
				"Output truncated",
				"500 total lines",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := reporter.Report(tt.results)

			if report.ExitCode != tt.wantExitCode {
				t.Errorf("expected exit code %d, got %d", tt.wantExitCode, report.ExitCode)
			}

			for _, want := range tt.wantInStdout {
				if !strings.Contains(report.Stdout, want) {
					t.Errorf("expected %q in stdout, got %q", want, report.Stdout)
				}
			}

			for _, want := range tt.wantInStderr {
				if !strings.Contains(report.Stderr, want) {
					t.Errorf("expected %q in stderr, got %q", want, report.Stderr)
				}
			}

			for _, notWant := range tt.notWantStderr {
				if strings.Contains(report.Stderr, notWant) {
					t.Errorf("should not have %q in stderr, got %q", notWant, report.Stderr)
				}
			}
		})
	}
}

func TestHasErrors(t *testing.T) {
	reporter := NewErrorReporter()

	tests := []struct {
		name     string
		result   executor.ComponentExecResult
		expected bool
	}{
		{
			name: "exit code match",
			result: executor.ComponentExecResult{
				ExecResult: &executor.ExecResult{ExitCode: 1},
				CommandConfig: &config.CommandConfig{
					ExitCodes: []int{1, 2},
				},
			},
			expected: true,
		},
		{
			name: "exit code no match",
			result: executor.ComponentExecResult{
				ExecResult: &executor.ExecResult{ExitCode: 0},
				CommandConfig: &config.CommandConfig{
					ExitCodes: []int{1, 2},
				},
			},
			expected: false,
		},
		{
			name: "filtered output has errors",
			result: executor.ComponentExecResult{
				ExecResult:     &executor.ExecResult{ExitCode: 0},
				FilteredOutput: &filter.FilteredOutput{HasErrors: true},
			},
			expected: true,
		},
		{
			name: "no config non-zero exit",
			result: executor.ComponentExecResult{
				ExecResult: &executor.ExecResult{ExitCode: 1},
			},
			expected: true,
		},
		{
			name: "no config zero exit",
			result: executor.ComponentExecResult{
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
		components []executor.ComponentExecResult
		expected   string
	}{
		{
			name:    "custom prompt",
			command: "lint",
			components: []executor.ComponentExecResult{
				{CommandConfig: &config.CommandConfig{Prompt: "Custom prompt:"}},
			},
			expected: "Custom prompt:",
		},
		{
			name:       "format default",
			command:    "format",
			components: []executor.ComponentExecResult{{}},
			expected:   "Fix the formatting issues below:",
		},
		{
			name:       "lint default",
			command:    "lint",
			components: []executor.ComponentExecResult{{}},
			expected:   "Fix the linting errors below:",
		},
		{
			name:       "typecheck default",
			command:    "typecheck",
			components: []executor.ComponentExecResult{{}},
			expected:   "Fix the type errors below:",
		},
		{
			name:       "test default",
			command:    "test",
			components: []executor.ComponentExecResult{{}},
			expected:   "Fix the failing tests below:",
		},
		{
			name:       "unknown command",
			command:    "custom",
			components: []executor.ComponentExecResult{{}},
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
		result   executor.ComponentExecResult
		execErr  *executor.ExecError
		contains []string
	}{
		{
			name: "command not found",
			result: executor.ComponentExecResult{
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
			result: executor.ComponentExecResult{
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
			result: executor.ComponentExecResult{
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
			result: executor.ComponentExecResult{
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

	components := []executor.ComponentExecResult{
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

func TestReport_FallbackAndMixed(t *testing.T) {
	reporter := NewErrorReporter()

	tests := []struct {
		name          string
		results       []executor.ComponentExecResult
		wantExitCode  int
		wantInStderr  []string
		notWantStderr []string
	}{
		{
			name: "fallback to stderr output",
			results: []executor.ComponentExecResult{
				{
					Command: "lint",
					ExecResult: &executor.ExecResult{
						ExitCode: 1,
						Stderr:   "Error on line 10",
					},
					CommandConfig: &config.CommandConfig{
						ExitCodes: []int{1},
					},
				},
			},
			wantExitCode: 2,
			wantInStderr: []string{"Error on line 10"},
		},
		{
			name: "fallback to stdout when no stderr",
			results: []executor.ComponentExecResult{
				{
					Command: "test",
					ExecResult: &executor.ExecResult{
						ExitCode: 1,
						Stdout:   "Test failed: assertion error",
					},
					CommandConfig: &config.CommandConfig{
						ExitCodes: []int{1},
					},
				},
			},
			wantExitCode: 2,
			wantInStderr: []string{"Test failed: assertion error"},
		},
		{
			name: "mixed results",
			results: []executor.ComponentExecResult{
				{
					Command: "lint",
					ExecResult: &executor.ExecResult{
						ExitCode: 0,
					},
					CommandConfig: &config.CommandConfig{
						ExitCodes: []int{1},
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
					CommandConfig: &config.CommandConfig{
						ExitCodes: []int{1},
					},
				},
			},
			wantExitCode:  2,
			wantInStderr:  []string{"Test failed"},
			notWantStderr: []string{"lint"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := reporter.Report(tt.results)

			if report.ExitCode != tt.wantExitCode {
				t.Errorf("expected exit code %d, got %d", tt.wantExitCode, report.ExitCode)
			}

			for _, want := range tt.wantInStderr {
				if !strings.Contains(report.Stderr, want) {
					t.Errorf("expected %q in stderr, got %q", want, report.Stderr)
				}
			}

			for _, notWant := range tt.notWantStderr {
				if strings.Contains(report.Stderr, notWant) {
					t.Errorf("should not have %q in stderr, got %q", notWant, report.Stderr)
				}
			}
		})
	}
}

func TestReport_EdgeCases(t *testing.T) {
	reporter := NewErrorReporter()

	t.Run("partial output before error", func(t *testing.T) {
		// Test that partial output is captured when a command fails mid-execution
		partialOutput := strings.Repeat("Starting test run...\n", 10)
		errorOutput := "FATAL ERROR: Segmentation fault at line 42"

		results := []executor.ComponentExecResult{
			{
				Command: "test",
				ExecResult: &executor.ExecResult{
					ExitCode: 139, // SIGSEGV exit code
					Stdout:   partialOutput,
					Stderr:   errorOutput,
				},
				FilteredOutput: &filter.FilteredOutput{
					Lines:     []string{errorOutput},
					HasErrors: true,
				},
				CommandConfig: &config.CommandConfig{
					ExitCodes: []int{1, 139},
				},
			},
		}

		report := reporter.Report(results)
		if report.ExitCode != 2 {
			t.Errorf("expected exit code 2, got %d", report.ExitCode)
		}
		if !strings.Contains(report.Stderr, errorOutput) {
			t.Errorf("expected error output in stderr, got %q", report.Stderr)
		}
	})

	t.Run("extremely large error output", func(t *testing.T) {
		// Generate 2MB of error output
		largeErrorLine := strings.Repeat("ERROR: This is a very long error message with details ", 100)
		var lines []string
		totalSize := 0
		for totalSize < 2*1024*1024 { // 2MB
			lines = append(lines, largeErrorLine)
			totalSize += len(largeErrorLine)
		}

		results := []executor.ComponentExecResult{
			{
				Command: "lint",
				ExecResult: &executor.ExecResult{
					ExitCode: 1,
					Stderr:   "Large output truncated for display", // Don't include actual large output
				},
				FilteredOutput: &filter.FilteredOutput{
					Lines:      lines[:50], // Filtered to first 50 lines
					HasErrors:  true,
					Truncated:  true,
					TotalLines: len(lines),
				},
				CommandConfig: &config.CommandConfig{
					ExitCodes: []int{1},
				},
			},
		}

		report := reporter.Report(results)
		if report.ExitCode != 2 {
			t.Errorf("expected exit code 2, got %d", report.ExitCode)
		}
		if !strings.Contains(report.Stderr, "Output truncated") {
			t.Errorf("expected truncation notice in stderr")
		}
		if !strings.Contains(report.Stderr, fmt.Sprintf("%d total lines", len(lines))) {
			t.Errorf("expected total line count in stderr")
		}
		// Verify output is reasonable size (not 2MB)
		// The filtered output (50 lines) should dominate the size
		expectedMaxSize := len(largeErrorLine) * 50 * 2 // 50 lines + some overhead
		if len(report.Stderr) > expectedMaxSize {
			t.Errorf("stderr output too large: %d bytes (expected < %d)", len(report.Stderr), expectedMaxSize)
		}
	})

	t.Run("empty results slice", func(t *testing.T) {
		report := reporter.Report([]executor.ComponentExecResult{})
		if report.ExitCode != 0 {
			t.Errorf("expected exit code 0 for empty results, got %d", report.ExitCode)
		}
		if !strings.Contains(report.Stdout, "All quality checks passed") {
			t.Errorf("expected success message in stdout, got %q", report.Stdout)
		}
	})

	t.Run("nil exec result", func(t *testing.T) {
		results := []executor.ComponentExecResult{
			{
				Command:       "test",
				ExecResult:    nil,
				CommandConfig: &config.CommandConfig{},
			},
		}

		// Should not panic and treat as success (no error)
		report := reporter.Report(results)
		if report.ExitCode != 0 {
			t.Errorf("expected exit code 0 for nil exec result, got %d", report.ExitCode)
		}
	})

	t.Run("mixed stdout and stderr in same component", func(t *testing.T) {
		results := []executor.ComponentExecResult{
			{
				Command: "test",
				ExecResult: &executor.ExecResult{
					ExitCode: 1,
					Stdout:   "Running tests...\nTest 1: PASS\nTest 2: PASS\n",
					Stderr:   "Test 3: FAIL - assertion failed",
				},
				FilteredOutput: &filter.FilteredOutput{
					Lines:     []string{"Test 3: FAIL - assertion failed"},
					HasErrors: true,
				},
				CommandConfig: &config.CommandConfig{
					ExitCodes: []int{1},
				},
			},
		}

		report := reporter.Report(results)
		if report.ExitCode != 2 {
			t.Errorf("expected exit code 2, got %d", report.ExitCode)
		}
		// Should prefer filtered output over raw stderr
		if !strings.Contains(report.Stderr, "Test 3: FAIL") {
			t.Errorf("expected filtered error in stderr, got %q", report.Stderr)
		}
	})
}

func TestReport_ConcurrentErrorReporting(t *testing.T) {
	reporter := NewErrorReporter()

	t.Run("concurrent report calls", func(t *testing.T) {
		// Test that multiple goroutines can safely call Report
		results := []executor.ComponentExecResult{
			{
				Command: "lint",
				ExecResult: &executor.ExecResult{
					ExitCode: 1,
					Stderr:   "Linting error",
				},
				CommandConfig: &config.CommandConfig{
					ExitCodes: []int{1},
				},
			},
		}

		const numGoroutines = 10
		reports := make([]*ReportResult, numGoroutines)
		done := make(chan bool, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func(idx int) {
				reports[idx] = reporter.Report(results)
				done <- true
			}(i)
		}

		// Wait for all goroutines
		for i := 0; i < numGoroutines; i++ {
			<-done
		}

		// All reports should be identical
		for i := 1; i < numGoroutines; i++ {
			if reports[i].ExitCode != reports[0].ExitCode {
				t.Errorf("report %d has different exit code: %d vs %d", i, reports[i].ExitCode, reports[0].ExitCode)
			}
			if reports[i].Stderr != reports[0].Stderr {
				t.Errorf("report %d has different stderr", i)
			}
		}
	})

	t.Run("concurrent error aggregation simulation", func(t *testing.T) {
		// Simulate errors coming from multiple parallel executors
		var results []executor.ComponentExecResult

		// Simulate 5 components with errors
		for i := 0; i < 5; i++ {
			results = append(results, executor.ComponentExecResult{
				Path:    fmt.Sprintf("component-%d", i),
				Command: "test",
				ExecResult: &executor.ExecResult{
					ExitCode: 1,
					Stderr:   fmt.Sprintf("Test failed in component %d", i),
				},
				FilteredOutput: &filter.FilteredOutput{
					Lines:     []string{fmt.Sprintf("Error at line %d in component %d", i*10, i)},
					HasErrors: true,
				},
				CommandConfig: &config.CommandConfig{
					ExitCodes: []int{1},
				},
			})
		}

		report := reporter.Report(results)
		if report.ExitCode != 2 {
			t.Errorf("expected exit code 2, got %d", report.ExitCode)
		}

		// Check all components are reported
		for i := 0; i < 5; i++ {
			componentName := fmt.Sprintf("component-%d", i)
			if !strings.Contains(report.Stderr, componentName) {
				t.Errorf("missing component %s in error report", componentName)
			}
		}
	})
}

func TestReport_ErrorAggregationFromMultipleSources(t *testing.T) {
	reporter := NewErrorReporter()

	t.Run("multiple error types from different tools", func(t *testing.T) {
		results := []executor.ComponentExecResult{
			// Linting errors
			{
				Path:    "frontend",
				Command: "lint",
				ExecResult: &executor.ExecResult{
					ExitCode: 1,
					Stderr:   "ESLint found 5 problems",
				},
				FilteredOutput: &filter.FilteredOutput{
					Lines: []string{
						"src/app.js:10:5: error: Missing semicolon",
						"src/app.js:20:1: warning: Unused variable 'x'",
					},
					HasErrors: true,
				},
				CommandConfig: &config.CommandConfig{
					ExitCodes: []int{1},
					Prompt:    "Fix the linting issues:",
				},
			},
			// Type errors
			{
				Path:    "backend",
				Command: "typecheck",
				ExecResult: &executor.ExecResult{
					ExitCode: 2,
					Stderr:   "TypeScript compilation failed",
				},
				FilteredOutput: &filter.FilteredOutput{
					Lines: []string{
						"src/server.ts:15:10: error TS2322: Type 'string' is not assignable to type 'number'",
						"src/server.ts:25:5: error TS2554: Expected 2 arguments, but got 1",
					},
					HasErrors: true,
				},
				CommandConfig: &config.CommandConfig{
					ExitCodes: []int{2},
				},
			},
			// Test failures
			{
				Path:    "shared",
				Command: "test",
				ExecResult: &executor.ExecResult{
					ExitCode: 1,
					Stdout:   "Test suite failed",
					Stderr:   "",
				},
				FilteredOutput: &filter.FilteredOutput{
					Lines: []string{
						"FAIL: TestUserAuth",
						"  Expected: true",
						"  Got: false",
					},
					HasErrors: true,
				},
				CommandConfig: &config.CommandConfig{
					ExitCodes: []int{1},
				},
			},
			// Execution error
			{
				Path:    "tools",
				Command: "custom-check",
				ExecutionError: &executor.ExecError{
					Type:    executor.ErrorTypeCommandNotFound,
					Command: "custom-check",
					Err:     errors.New("command not found"),
				},
			},
		}

		report := reporter.Report(results)

		// Should report execution error with exit code 1
		if report.ExitCode != 1 {
			t.Errorf("expected exit code 1 for execution error, got %d", report.ExitCode)
		}
		if !strings.Contains(report.Stderr, "[QUALHOOK ERROR]") {
			t.Errorf("expected error prefix for execution error")
		}
		if !strings.Contains(report.Stderr, "custom-check") {
			t.Errorf("expected command name in error")
		}
	})

	t.Run("aggregated errors without execution errors", func(t *testing.T) {
		results := []executor.ComponentExecResult{
			// Multiple lint errors from different paths
			{
				Path:    "src/frontend",
				Command: "lint",
				ExecResult: &executor.ExecResult{
					ExitCode: 1,
					Stderr:   "3 errors found",
				},
				CommandConfig: &config.CommandConfig{
					ExitCodes: []int{1},
				},
			},
			{
				Path:    "src/backend",
				Command: "lint",
				ExecResult: &executor.ExecResult{
					ExitCode: 1,
					Stderr:   "5 errors found",
				},
				CommandConfig: &config.CommandConfig{
					ExitCodes: []int{1},
				},
			},
			// Format errors
			{
				Path:    "docs",
				Command: "format",
				ExecResult: &executor.ExecResult{
					ExitCode: 1,
					Stderr:   "Formatting issues detected",
				},
				CommandConfig: &config.CommandConfig{
					ExitCodes: []int{1},
				},
			},
		}

		report := reporter.Report(results)

		if report.ExitCode != 2 {
			t.Errorf("expected exit code 2 for quality errors, got %d", report.ExitCode)
		}

		// Check that errors are grouped by command type
		if !strings.Contains(report.Stderr, "Fix the linting errors below:") {
			t.Errorf("expected lint prompt in stderr")
		}
		if !strings.Contains(report.Stderr, "Fix the formatting issues below:") {
			t.Errorf("expected format prompt in stderr")
		}

		// Check component separation
		if !strings.Contains(report.Stderr, "src/frontend") {
			t.Errorf("expected frontend component in stderr")
		}
		if !strings.Contains(report.Stderr, "src/backend") {
			t.Errorf("expected backend component in stderr")
		}
	})

	t.Run("mixed success and failure across components", func(t *testing.T) {
		results := []executor.ComponentExecResult{
			// Success
			{
				Path:    "component1",
				Command: "lint",
				ExecResult: &executor.ExecResult{
					ExitCode: 0,
					Stdout:   "All checks passed",
				},
				CommandConfig: &config.CommandConfig{
					ExitCodes: []int{1},
				},
			},
			// Failure
			{
				Path:    "component2",
				Command: "lint",
				ExecResult: &executor.ExecResult{
					ExitCode: 1,
					Stderr:   "Linting failed",
				},
				CommandConfig: &config.CommandConfig{
					ExitCodes: []int{1},
				},
			},
			// Success
			{
				Path:    "component3",
				Command: "test",
				ExecResult: &executor.ExecResult{
					ExitCode: 0,
					Stdout:   "All tests passed",
				},
				CommandConfig: &config.CommandConfig{
					ExitCodes: []int{1, 2},
				},
			},
			// Failure with filtered output
			{
				Path:    "component4",
				Command: "test",
				ExecResult: &executor.ExecResult{
					ExitCode: 2,
					Stderr:   "Test suite failed with 3 failures",
				},
				FilteredOutput: &filter.FilteredOutput{
					Lines: []string{
						"TestA: Failed - timeout",
						"TestB: Failed - assertion",
						"TestC: Failed - panic",
					},
					HasErrors:  true,
					Truncated:  true,
					TotalLines: 150,
				},
				CommandConfig: &config.CommandConfig{
					ExitCodes: []int{1, 2},
				},
			},
		}

		report := reporter.Report(results)

		if report.ExitCode != 2 {
			t.Errorf("expected exit code 2, got %d", report.ExitCode)
		}

		// Should only report failures
		if strings.Contains(report.Stderr, "component1") {
			t.Errorf("should not include successful component1 in stderr")
		}
		if strings.Contains(report.Stderr, "component3") {
			t.Errorf("should not include successful component3 in stderr")
		}

		// Should include error outputs from failed components
		// Note: component paths are only shown when multiple components of the same command fail
		if !strings.Contains(report.Stderr, "Linting failed") {
			t.Errorf("expected lint error output in stderr")
		}
		if !strings.Contains(report.Stderr, "TestA: Failed") {
			t.Errorf("expected test error output in stderr")
		}

		// Check that both command types are reported
		if !strings.Contains(report.Stderr, "Fix the linting errors below:") {
			t.Errorf("expected lint prompt in stderr")
		}
		if !strings.Contains(report.Stderr, "Fix the failing tests below:") {
			t.Errorf("expected test prompt in stderr")
		}

		// Check truncation notice for component4
		if !strings.Contains(report.Stderr, "Output truncated") {
			t.Errorf("expected truncation notice for component4")
		}
		if !strings.Contains(report.Stderr, "150 total lines") {
			t.Errorf("expected total line count for component4")
		}
	})
}
