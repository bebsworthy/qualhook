package testutil

import (
	"testing"
	"time"

	"github.com/bebsworthy/qualhook/internal/executor"
)

// CommandBuilder provides a fluent interface for building test commands.
type CommandBuilder struct {
	command TestCommand
	options executor.ExecOptions
}

// NewCommandBuilder creates a new CommandBuilder with default values.
func NewCommandBuilder() *CommandBuilder {
	return &CommandBuilder{
		command: TestCommand{
			Name:    "test-command",
			Command: SafeCommands.Echo,
			Args:    []string{"test"},
			Timeout: 5 * time.Second,
		},
		options: executor.ExecOptions{
			InheritEnv: true,
			Timeout:    5 * time.Second,
		},
	}
}

// WithName sets the command name for identification.
func (b *CommandBuilder) WithName(name string) *CommandBuilder {
	b.command.Name = name
	return b
}

// WithCommand sets the command to execute.
func (b *CommandBuilder) WithCommand(command string) *CommandBuilder {
	b.command.Command = command
	return b
}

// WithArgs sets the command arguments.
func (b *CommandBuilder) WithArgs(args ...string) *CommandBuilder {
	b.command.Args = args
	return b
}

// WithEnv adds environment variables.
func (b *CommandBuilder) WithEnv(env ...string) *CommandBuilder {
	b.command.Env = append(b.command.Env, env...)
	return b
}

// WithDir sets the working directory.
func (b *CommandBuilder) WithDir(dir string) *CommandBuilder {
	b.command.Dir = dir
	b.options.WorkingDir = dir
	return b
}

// WithTimeout sets the command timeout.
func (b *CommandBuilder) WithTimeout(timeout time.Duration) *CommandBuilder {
	b.command.Timeout = timeout
	b.options.Timeout = timeout
	return b
}

// Echo creates an echo command with the given message.
func (b *CommandBuilder) Echo(message string) *CommandBuilder {
	b.command.Name = "echo"
	b.command.Command = SafeCommands.Echo
	b.command.Args = []string{message}
	return b
}

// Sleep creates a sleep command with the given duration.
func (b *CommandBuilder) Sleep(duration string) *CommandBuilder {
	b.command.Name = "sleep"
	b.command.Command = SafeCommands.Sleep
	b.command.Args = []string{duration}
	return b
}

// Failing creates a command that will fail (exit code 1).
func (b *CommandBuilder) Failing() *CommandBuilder {
	b.command.Name = "false" //nolint:goconst
	b.command.Command = SafeCommands.False
	b.command.Args = CommandArgs(SafeCommands.False, "false")
	return b
}

// Successful creates a command that will succeed (exit code 0).
func (b *CommandBuilder) Successful() *CommandBuilder {
	b.command.Name = "true"
	b.command.Command = SafeCommands.True
	b.command.Args = CommandArgs(SafeCommands.True)
	return b
}

// Script creates a command that executes a shell script.
func (b *CommandBuilder) Script(script string) *CommandBuilder {
	b.command.Name = "script"
	b.command.Command = "sh"
	b.command.Args = []string{"-c", script}
	return b
}

// Build returns the constructed TestCommand.
func (b *CommandBuilder) Build() TestCommand {
	return b.command
}

// BuildOptions returns the constructed ExecOptions.
func (b *CommandBuilder) BuildOptions() executor.ExecOptions {
	// Sync environment from command to options
	b.options.Environment = b.command.Env
	return b.options
}

// Execute runs the command and returns the result.
// This is a convenience method for immediate execution in tests.
func (b *CommandBuilder) Execute(t testing.TB) (stdout, stderr string, exitCode int) {
	return RunCommand(t, b.command)
}
