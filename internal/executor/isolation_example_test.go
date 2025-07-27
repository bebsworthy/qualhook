//go:build unit

package executor_test

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bebsworthy/qualhook/internal/executor"
	"github.com/bebsworthy/qualhook/internal/testutil"
)

// TestExecutorWithIsolation demonstrates how to use the new test isolation features
func TestExecutorWithIsolation(t *testing.T) {
	// Example 1: Basic isolated execution
	t.Run("basic_isolated_execution", func(t *testing.T) {
		testutil.WithIsolatedEnvironment(t, func(env *testutil.TestEnvironment) {
			exec := executor.NewCommandExecutor(5 * time.Second)

			// Create a test file in the isolated environment
			testFile := env.CreateTempFile("test.txt", "Hello, World!")

			// Run a safe command on the file
			result, err := exec.Execute("cat", []string{testFile}, executor.ExecOptions{
				WorkingDir: env.TempDir(),
			})

			if err != nil {
				t.Fatalf("Execute failed: %v", err)
			}

			if !strings.Contains(result.Stdout, "Hello, World!") {
				t.Errorf("Expected output to contain 'Hello, World!', got: %s", result.Stdout)
			}
		})
	})

	// Example 2: Test with directory structure
	t.Run("test_with_directory_structure", func(t *testing.T) {
		env := testutil.SafeCommandEnvironment(t)

		// Create a project structure
		env.CreateTempDir("src")
		env.CreateTempDir("test")

		// Create some files
		env.CreateTempFile("src/main.go", "package main\n\nfunc main() {}")
		env.CreateTempFile("test/main_test.go", "package main\n\nimport \"testing\"")

		exec := executor.NewCommandExecutor(5 * time.Second)

		// List the directory structure
		result, err := exec.Execute("ls", []string{"-la"}, executor.ExecOptions{
			WorkingDir: env.TempDir(),
		})

		if err != nil {
			t.Fatalf("Execute failed: %v", err)
		}

		// Verify directories were created
		if !strings.Contains(result.Stdout, "src") {
			t.Error("Expected to see 'src' directory in output")
		}
		if !strings.Contains(result.Stdout, "test") {
			t.Error("Expected to see 'test' directory in output")
		}
	})

	// Example 3: Test with custom whitelisted command
	t.Run("custom_whitelisted_command", func(t *testing.T) {
		env := testutil.SafeCommandEnvironment(t)

		// Create a safe test script
		scriptContent := `#!/bin/sh
echo "Custom command executed"
exit 0`

		scriptPath := env.CreateTempFile("safe-script.sh", scriptContent)

		// Make it executable (if not on Windows)
		if !testutil.IsWindows() {
			exec := executor.NewCommandExecutor(5 * time.Second)
			_, _ = exec.Execute("chmod", []string{"+x", scriptPath}, executor.ExecOptions{})
		}

		// Whitelist our custom script
		env.AllowCommand(filepath.Base(scriptPath))

		// Execute using RunSafeCommand
		cmd := testutil.TestCommand{
			Name:    "safe-script",
			Command: scriptPath,
			Args:    []string{},
			Timeout: 5 * time.Second,
		}

		stdout, stderr, exitCode := env.RunSafeCommand(cmd)

		if exitCode != 0 {
			t.Errorf("Expected exit code 0, got %d. Stderr: %s", exitCode, stderr)
		}
		if !strings.Contains(stdout, "Custom command executed") {
			t.Errorf("Expected output to contain 'Custom command executed', got: %s", stdout)
		}
	})

	// Example 4: Test cleanup functionality
	t.Run("cleanup_functionality", func(t *testing.T) {
		env := testutil.SafeCommandEnvironment(t)

		// Add custom cleanup
		var cleanupCalled bool
		env.AddCleanup(func() {
			cleanupCalled = true
		})

		// Create some test resources
		env.CreateTempFile("cleanup-test.txt", "test content")

		// Manually trigger cleanup (normally done automatically)
		env.Cleanup()

		if !cleanupCalled {
			t.Error("Cleanup function was not called")
		}
	})

	// Example 5: Parallel executor with isolation
	t.Run("parallel_executor_with_isolation", func(t *testing.T) {
		env := testutil.SafeCommandEnvironment(t)

		// Create test files for parallel processing
		env.CreateTempFile("file1.txt", "Content 1")
		env.CreateTempFile("file2.txt", "Content 2")
		env.CreateTempFile("file3.txt", "Content 3")

		cmdExec := executor.NewCommandExecutor(5 * time.Second)
		parallelExec := executor.NewParallelExecutor(cmdExec, 3)

		commands := []executor.ParallelCommand{
			{
				ID:      "cmd1",
				Command: "echo",
				Args:    []string{"Processing file1"},
				Options: executor.ExecOptions{WorkingDir: env.TempDir()},
			},
			{
				ID:      "cmd2",
				Command: "echo",
				Args:    []string{"Processing file2"},
				Options: executor.ExecOptions{WorkingDir: env.TempDir()},
			},
			{
				ID:      "cmd3",
				Command: "echo",
				Args:    []string{"Processing file3"},
				Options: executor.ExecOptions{WorkingDir: env.TempDir()},
			},
		}

		ctx := testutil.TestContext(t)
		result, err := parallelExec.Execute(ctx, commands, nil)

		if err != nil {
			t.Fatalf("Parallel execute failed: %v", err)
		}

		// Verify all commands executed
		if len(result.Results) != 3 {
			t.Errorf("Expected 3 results, got %d", len(result.Results))
		}

		for id, res := range result.Results {
			if res.ExitCode != 0 {
				t.Errorf("Command %s failed with exit code %d", id, res.ExitCode)
			}
		}
	})

	// Example 6: Security validation
	t.Run("security_validation", func(t *testing.T) {
		env := testutil.SafeCommandEnvironment(t)

		// These dangerous commands should be blocked
		dangerousTests := []struct {
			name    string
			command string
			args    []string
		}{
			{"path traversal", "cat", []string{"../../etc/passwd"}},
			{"system directory", "ls", []string{"/etc"}},
			{"command substitution", "echo", []string{"`whoami`"}},
			{"piping", "echo", []string{"test", "|", "cat"}},
			{"redirection", "echo", []string{"test", ">", "/tmp/test"}},
		}

		for _, tt := range dangerousTests {
			t.Run(tt.name, func(t *testing.T) {
				err := env.ValidateCommand(tt.command, tt.args)
				if err == nil {
					t.Errorf("Expected validation error for dangerous command: %s %v", tt.command, tt.args)
				}
			})
		}
	})
}

// TestMigrationExample shows how to migrate existing tests to use isolation
func TestMigrationExample(t *testing.T) {
	// Before: Test without isolation
	t.Run("old_style_test", func(t *testing.T) {
		t.Skip("Example of old style - don't actually run")

		// This is how tests used to work - directly using executors
		// executor := NewCommandExecutor(5 * time.Second)
		// result, err := executor.Execute("rm", []string{"-rf", "/tmp/test"}, executor.ExecOptions{})
		// This could be dangerous!
	})

	// After: Test with isolation
	t.Run("new_style_test", func(t *testing.T) {
		testutil.WithIsolatedEnvironment(t, func(env *testutil.TestEnvironment) {
			// Now we work in an isolated environment
			testFile := env.CreateTempFile("test.txt", "safe to delete")

			// Use safe commands
			exec := executor.NewCommandExecutor(5 * time.Second)

			// This will only affect the isolated temp directory
			result, err := exec.Execute("rm", []string{testFile}, executor.ExecOptions{
				WorkingDir: env.TempDir(),
			})

			// Note: rm is not whitelisted by default, so this would actually fail
			// You would need to explicitly whitelist it for this specific test:
			// env.AllowCommand("rm")

			if err != nil {
				t.Logf("As expected, rm command is not whitelisted: %v", err)
			}
			_ = result
		})
	})
}
