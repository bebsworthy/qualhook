package testutil

import (
	"testing"
	"time"
)

const (
	testDirTmp = "/tmp"
	testCmdTrue = "true"
	testCmdFalse = "false"
)

func TestCommandBuilder(t *testing.T) {
	t.Run("default values", func(t *testing.T) {
		cmd := NewCommandBuilder().Build()
		
		if cmd.Name != "test-command" {
			t.Errorf("Expected default name 'test-command', got %q", cmd.Name)
		}
		if cmd.Command != SafeCommands.Echo {
			t.Errorf("Expected default command to be echo, got %q", cmd.Command)
		}
		if len(cmd.Args) != 1 || cmd.Args[0] != "test" {
			t.Errorf("Expected default args ['test'], got %v", cmd.Args)
		}
		if cmd.Timeout != 5*time.Second {
			t.Errorf("Expected default timeout 5s, got %v", cmd.Timeout)
		}
	})

	t.Run("fluent interface", func(t *testing.T) {
		cmd := NewCommandBuilder().
			WithName("my-test").
			WithCommand("ls").
			WithArgs("-la", "/tmp").
			WithEnv("FOO=bar", "BAZ=qux").
			WithDir("/tmp").
			WithTimeout(10 * time.Second).
			Build()

		if cmd.Name != "my-test" {
			t.Errorf("Expected name 'my-test', got %q", cmd.Name)
		}
		if cmd.Command != "ls" {
			t.Errorf("Expected command 'ls', got %q", cmd.Command)
		}
		if len(cmd.Args) != 2 || cmd.Args[0] != "-la" || cmd.Args[1] != testDirTmp {
			t.Errorf("Expected args ['-la', '/tmp'], got %v", cmd.Args)
		}
		if len(cmd.Env) != 2 || cmd.Env[0] != "FOO=bar" || cmd.Env[1] != "BAZ=qux" {
			t.Errorf("Expected env ['FOO=bar', 'BAZ=qux'], got %v", cmd.Env)
		}
		if cmd.Dir != testDirTmp {
			t.Errorf("Expected dir '/tmp', got %q", cmd.Dir)
		}
		if cmd.Timeout != 10*time.Second {
			t.Errorf("Expected timeout 10s, got %v", cmd.Timeout)
		}
	})

	t.Run("echo command", func(t *testing.T) {
		cmd := NewCommandBuilder().Echo("hello world").Build()
		
		if cmd.Name != "echo" {
			t.Errorf("Expected name 'echo', got %q", cmd.Name)
		}
		if cmd.Command != SafeCommands.Echo {
			t.Errorf("Expected echo command, got %q", cmd.Command)
		}
		if len(cmd.Args) != 1 || cmd.Args[0] != "hello world" {
			t.Errorf("Expected args ['hello world'], got %v", cmd.Args)
		}
	})

	t.Run("sleep command", func(t *testing.T) {
		cmd := NewCommandBuilder().Sleep("2").Build()
		
		if cmd.Name != "sleep" {
			t.Errorf("Expected name 'sleep', got %q", cmd.Name)
		}
		if cmd.Command != SafeCommands.Sleep {
			t.Errorf("Expected sleep command, got %q", cmd.Command)
		}
		if len(cmd.Args) != 1 || cmd.Args[0] != "2" {
			t.Errorf("Expected args ['2'], got %v", cmd.Args)
		}
	})

	t.Run("failing command", func(t *testing.T) {
		cmd := NewCommandBuilder().Failing().Build()
		
		if cmd.Name != testCmdFalse {
			t.Errorf("Expected name 'false', got %q", cmd.Name)
		}
		if cmd.Command != SafeCommands.False {
			t.Errorf("Expected false command, got %q", cmd.Command)
		}
	})

	t.Run("successful command", func(t *testing.T) {
		cmd := NewCommandBuilder().Successful().Build()
		
		if cmd.Name != testCmdTrue {
			t.Errorf("Expected name 'true', got %q", cmd.Name)
		}
		if cmd.Command != SafeCommands.True {
			t.Errorf("Expected true command, got %q", cmd.Command)
		}
	})

	t.Run("script command", func(t *testing.T) {
		cmd := NewCommandBuilder().Script("echo hello && echo world").Build()
		
		if cmd.Name != "script" {
			t.Errorf("Expected name 'script', got %q", cmd.Name)
		}
		if cmd.Command != "sh" {
			t.Errorf("Expected command 'sh', got %q", cmd.Command)
		}
		if len(cmd.Args) != 2 || cmd.Args[0] != "-c" || cmd.Args[1] != "echo hello && echo world" {
			t.Errorf("Expected args ['-c', 'echo hello && echo world'], got %v", cmd.Args)
		}
	})

	t.Run("build options", func(t *testing.T) {
		opts := NewCommandBuilder().
			WithEnv("TEST=1").
			WithDir("/tmp").
			WithTimeout(30 * time.Second).
			BuildOptions()

		if len(opts.Environment) != 1 || opts.Environment[0] != "TEST=1" {
			t.Errorf("Expected environment ['TEST=1'], got %v", opts.Environment)
		}
		if opts.WorkingDir != "/tmp" {
			t.Errorf("Expected working dir '/tmp', got %q", opts.WorkingDir)
		}
		if opts.Timeout != 30*time.Second {
			t.Errorf("Expected timeout 30s, got %v", opts.Timeout)
		}
		if !opts.InheritEnv {
			t.Errorf("Expected InheritEnv to be true")
		}
	})

	t.Run("execute integration", func(t *testing.T) {
		stdout, stderr, exitCode := NewCommandBuilder().Echo("test output").Execute(t)
		
		if exitCode != 0 {
			t.Errorf("Expected exit code 0, got %d", exitCode)
		}
		if stdout != "test output\n" {
			t.Errorf("Expected stdout 'test output\\n', got %q", stdout)
		}
		if stderr != "" {
			t.Errorf("Expected empty stderr, got %q", stderr)
		}
	})
}