//go:build unit

package executor

import (
	"fmt"
	"runtime"
	"testing"
	"time"
)

// testScenario represents a common test scenario for executors
type testScenario struct {
	name           string
	setupCmd       func() (cmd string, args []string)
	opts           ExecOptions
	wantErr        bool
	wantExecErr    bool
	wantErrType    ErrorType
	wantExitCode   int
	wantInStdout   string
	wantInStderr   string
	wantTimedOut   bool
	checkErrorType func(error) bool
}

// commonTestScenarios returns test scenarios that apply to both command and parallel executors
func commonTestScenarios(t *testing.T) []testScenario {
	return []testScenario{
		{
			name: "success",
			setupCmd: func() (string, []string) {
				if runtime.GOOS == osWindows {
					return cmdCommand, []string{"/c", "echo", "hello world"}
				}
				return echoCommand, []string{"hello world"}
			},
			wantExitCode: 0,
			wantInStdout: "hello world",
		},
		{
			name: "command not found",
			setupCmd: func() (string, []string) {
				return "this-command-does-not-exist-12345", []string{}
			},
			wantExecErr: true,
			wantErrType: ErrorTypeCommandNotFound,
		},
		{
			name: "timeout",
			setupCmd: func() (string, []string) {
				if runtime.GOOS == osWindows {
					return cmdCommand, []string{"/c", "timeout", "/t", "5", "/nobreak"}
				}
				return "sleep", []string{"5"}
			},
			opts:         ExecOptions{Timeout: 100 * time.Millisecond},
			wantTimedOut: true,
		},
		{
			name: "non-zero exit",
			setupCmd: func() (string, []string) {
				if runtime.GOOS == osWindows {
					return cmdCommand, []string{"/c", "exit", "1"}
				}
				return shCommand, []string{"-c", "exit 1"}
			},
			wantExitCode: 1,
		},
		{
			name: "environment variable",
			setupCmd: func() (string, []string) {
				testVar := "QUALHOOK_TEST_VAR"
				if runtime.GOOS == osWindows {
					return cmdCommand, []string{"/c", "echo", "%" + testVar + "%"}
				}
				return shCommand, []string{"-c", "echo $" + testVar}
			},
			opts: ExecOptions{
				Environment: []string{"QUALHOOK_TEST_VAR=test-value-12345"},
				InheritEnv:  true,
			},
			wantInStdout: "test-value-12345",
		},
	}
}

// platformCommand provides platform-specific command helpers
type platformCommand struct{}

var pc = platformCommand{}

func (platformCommand) echo(message string) (string, []string) {
	if runtime.GOOS == osWindows {
		return cmdCommand, []string{cmdArgC, echoCommand, message}
	}
	return echoCommand, []string{message}
}

func (platformCommand) exit(code int) (string, []string) {
	if runtime.GOOS == osWindows {
		return cmdCommand, []string{cmdArgC, "exit", fmt.Sprintf("%d", code)}
	}
	return shCommand, []string{shArgC, fmt.Sprintf("exit %d", code)}
}

func (platformCommand) sleep(seconds int) (string, []string) {
	if runtime.GOOS == osWindows {
		return cmdCommand, []string{cmdArgC, "timeout", "/t", fmt.Sprintf("%d", seconds), "/nobreak"}
	}
	return "sleep", []string{fmt.Sprintf("%d", seconds)}
}

func (platformCommand) pwd() (string, []string) {
	if runtime.GOOS == osWindows {
		return cmdCommand, []string{"/c", "cd"}
	}
	return "pwd", []string{}
}

func (platformCommand) stderr(message string) (string, []string) {
	if runtime.GOOS == osWindows {
		return cmdCommand, []string{"/c", "echo", message, "1>&2"}
	}
	return shCommand, []string{shArgC, fmt.Sprintf("echo '%s' >&2", message)}
}

// assertExecResult verifies common execution result properties
func assertExecResult(t *testing.T, result *ExecResult, expectedExitCode int, expectedStdout, expectedStderr string) {
	t.Helper()
	
	if result.ExitCode != expectedExitCode {
		t.Errorf("expected exit code %d, got %d", expectedExitCode, result.ExitCode)
	}
	
	if expectedStdout != "" && !contains(result.Stdout, expectedStdout) {
		t.Errorf("expected stdout to contain %q, got %q", expectedStdout, result.Stdout)
	}
	
	if expectedStderr != "" && !contains(result.Stderr, expectedStderr) {
		t.Errorf("expected stderr to contain %q, got %q", expectedStderr, result.Stderr)
	}
}

// contains is a simple string contains helper to avoid importing strings in test files
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || 
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr) >= 0))
}

func findSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// createTempDir creates a temporary directory for testing
func createTempDir(t *testing.T) (string, func()) {
	t.Helper()
	
	dir := t.TempDir()
	return dir, func() {
		// Cleanup is handled by t.TempDir()
	}
}

// timeoutTestCase returns a platform-specific timeout test case
func timeoutTestCase(timeout time.Duration) testScenario {
	return testScenario{
		name:         "timeout scenario",
		setupCmd:     func() (string, []string) { return pc.sleep(5) },
		opts:         ExecOptions{Timeout: timeout},
		wantTimedOut: true,
	}
}

// errorTestCase returns a command error test case
func errorTestCase(name string, exitCode int) testScenario {
	return testScenario{
		name:         name,
		setupCmd:     func() (string, []string) { return pc.exit(exitCode) },
		wantExitCode: exitCode,
	}
}