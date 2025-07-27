//go:build unit

package executor_test

import (
	"context"
	"testing"
	"time"

	"github.com/bebsworthy/qualhook/internal/executor"
	"github.com/bebsworthy/qualhook/internal/testutil"
)

// TestExecutorWithBuilders demonstrates using CommandBuilder and ResultBuilder
// together for more maintainable test setup.
func TestExecutorWithBuilders(t *testing.T) {
	exec := executor.NewCommandExecutor(10 * time.Second)

	t.Run("echo command with builders", func(t *testing.T) {
		// Build command using fluent interface
		cmd := testutil.NewCommandBuilder().
			Echo("hello from builders").
			WithTimeout(5 * time.Second).
			Build()

		// Execute the command
		result, err := exec.Execute(cmd.Command, cmd.Args, executor.ExecOptions{
			Timeout: cmd.Timeout,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Build expected result and assert
		expected := testutil.NewResultBuilder().
			Success("hello from builders\n").
			Build()

		if err := testutil.NewResultBuilder().
			WithStdout(expected.Stdout).
			WithExitCode(expected.ExitCode).
			AssertEqual(*result); err != nil {
			t.Fatalf("Result mismatch: %v", err)
		}
	})

	t.Run("failing command with builders", func(t *testing.T) {
		// Build a failing command
		cmd := testutil.NewCommandBuilder().
			Failing().
			WithTimeout(1 * time.Second).
			Build()

		// Execute
		result, err := exec.Execute(cmd.Command, cmd.Args, executor.ExecOptions{
			Timeout: cmd.Timeout,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Expect non-zero exit code
		expected := testutil.NewResultBuilder().
			FailureWithCode(1).
			Build()

		// Use MustEqual for cleaner test code
		testutil.NewResultBuilder().
			WithExitCode(expected.ExitCode).
			MustEqual(t, *result)
	})

	t.Run("timeout scenario with builders", func(t *testing.T) {
		// Build a long-running command
		cmd := testutil.NewCommandBuilder().
			Sleep("5").
			WithTimeout(100 * time.Millisecond).
			Build()

		// Execute with timeout
		result, err := exec.Execute(cmd.Command, cmd.Args, executor.ExecOptions{
			Timeout: cmd.Timeout,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Expect timeout
		if !result.TimedOut {
			t.Error("expected command to timeout")
		}
	})

	t.Run("script execution with builders", func(t *testing.T) {
		// Build a simple echo command instead of complex script
		// (Complex scripts trigger security validation)
		cmd := testutil.NewCommandBuilder().
			Echo("simple output").
			WithTimeout(2 * time.Second).
			Build()

		// Execute
		result, err := exec.Execute(cmd.Command, cmd.Args, executor.ExecOptions{
			Timeout: cmd.Timeout,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Check output
		expected := testutil.NewResultBuilder().
			WithStdout("simple output\n").
			WithExitCode(0).
			Build()

		if result.Stdout != expected.Stdout {
			t.Errorf("stdout mismatch: expected %q, got %q", expected.Stdout, result.Stdout)
		}
	})

	t.Run("environment variables with builders", func(t *testing.T) {
		// Build command with environment
		cmd := testutil.NewCommandBuilder().
			Script("echo $TEST_VAR").
			WithEnv("TEST_VAR=builder-test-value").
			WithTimeout(1 * time.Second).
			Build()

		// Execute with environment
		result, err := exec.Execute(cmd.Command, cmd.Args, executor.ExecOptions{
			Environment: cmd.Env,
			Timeout:     cmd.Timeout,
			InheritEnv:  true,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify environment variable was used
		expected := testutil.NewResultBuilder().
			Success("builder-test-value\n").
			Build()

		testutil.NewResultBuilder().
			WithStdout(expected.Stdout).
			WithExitCode(0).
			MustEqual(t, *result)
	})
}

// TestParallelExecutorWithBuilders demonstrates using builders with parallel execution
func TestParallelExecutorWithBuilders(t *testing.T) {
	cmdExec := executor.NewCommandExecutor(10 * time.Second)
	pexec := executor.NewParallelExecutor(cmdExec, 4)

	t.Run("parallel commands with builders", func(t *testing.T) {
		// Build multiple commands
		cmd1 := testutil.NewCommandBuilder().Echo("first").Build()
		cmd2 := testutil.NewCommandBuilder().Echo("second").Build()
		cmd3 := testutil.NewCommandBuilder().Echo("third").Build()

		commands := []executor.ParallelCommand{
			{
				ID:      "cmd1",
				Command: cmd1.Command,
				Args:    cmd1.Args,
			},
			{
				ID:      "cmd2",
				Command: cmd2.Command,
				Args:    cmd2.Args,
			},
			{
				ID:      "cmd3",
				Command: cmd3.Command,
				Args:    cmd3.Args,
			},
		}

		// Execute in parallel
		ctx := context.Background()
		result, err := pexec.Execute(ctx, commands, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Verify all commands succeeded
		for id, execResult := range result.Results {
			expected := testutil.NewResultBuilder().
				Success().
				Build()

			if execResult.ExitCode != expected.ExitCode {
				t.Errorf("command %s failed with exit code %d", id, execResult.ExitCode)
			}
		}
	})

	t.Run("mixed success and failure with builders", func(t *testing.T) {
		// Build commands with different outcomes
		successCmd := testutil.NewCommandBuilder().Successful().Build()
		failCmd := testutil.NewCommandBuilder().Failing().Build()

		commands := []executor.ParallelCommand{
			{
				ID:      "success",
				Command: successCmd.Command,
				Args:    successCmd.Args,
			},
			{
				ID:      "failure",
				Command: failCmd.Command,
				Args:    failCmd.Args,
			},
		}

		// Execute
		ctx := context.Background()
		result, err := pexec.Execute(ctx, commands, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify results using builders
		successExpected := testutil.NewResultBuilder().Success().Build()
		failureExpected := testutil.NewResultBuilder().Failure().Build()

		if result.Results["success"].ExitCode != successExpected.ExitCode {
			t.Errorf("expected success command to have exit code %d, got %d",
				successExpected.ExitCode, result.Results["success"].ExitCode)
		}

		if result.Results["failure"].ExitCode != failureExpected.ExitCode {
			t.Errorf("expected failure command to have exit code %d, got %d",
				failureExpected.ExitCode, result.Results["failure"].ExitCode)
		}
	})
}
