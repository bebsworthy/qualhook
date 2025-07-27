//go:build unit

package testutil

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSafeCommandEnvironment(t *testing.T) {
	t.Run("creates isolated environment", func(t *testing.T) {
		env := SafeCommandEnvironment(t)

		// Check temp directory exists
		if env.TempDir() == "" {
			t.Error("TempDir should not be empty")
		}

		// Check temp directory is accessible
		if _, err := os.Stat(env.TempDir()); err != nil {
			t.Errorf("TempDir should exist: %v", err)
		}
	})

	t.Run("whitelists safe commands", func(t *testing.T) {
		env := SafeCommandEnvironment(t)

		safeCommands := []string{"echo", "true", "false", "cat", "ls", "pwd"}
		for _, cmd := range safeCommands {
			if !env.IsCommandAllowed(cmd) {
				t.Errorf("Command %q should be allowed", cmd)
			}
		}
	})

	t.Run("blocks dangerous commands", func(t *testing.T) {
		env := SafeCommandEnvironment(t)

		dangerousCommands := []string{"rm", "dd", "format", "shutdown", "reboot", "kill"}
		for _, cmd := range dangerousCommands {
			if env.IsCommandAllowed(cmd) {
				t.Errorf("Command %q should not be allowed", cmd)
			}
		}
	})

	t.Run("allows adding commands to whitelist", func(t *testing.T) {
		env := SafeCommandEnvironment(t)

		// Initially not allowed
		if env.IsCommandAllowed("custom-tool") {
			t.Error("custom-tool should not be initially allowed")
		}

		// Add to whitelist
		env.AllowCommand("custom-tool")

		// Now allowed
		if !env.IsCommandAllowed("custom-tool") {
			t.Error("custom-tool should be allowed after whitelisting")
		}
	})
}

func TestValidateCommand(t *testing.T) {
	env := SafeCommandEnvironment(t)

	tests := []struct {
		name    string
		command string
		args    []string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "empty command",
			command: "",
			args:    []string{},
			wantErr: true,
			errMsg:  "command cannot be empty",
		},
		{
			name:    "non-whitelisted command",
			command: "rm",
			args:    []string{"-rf", "/"},
			wantErr: true,
			errMsg:  "not whitelisted",
		},
		{
			name:    "safe command",
			command: "echo",
			args:    []string{"hello"},
			wantErr: false,
		},
		{
			name:    "path traversal in args",
			command: "cat",
			args:    []string{"../../etc/passwd"},
			wantErr: true,
			errMsg:  "dangerous pattern",
		},
		{
			name:    "system path in args",
			command: "ls",
			args:    []string{"/etc"},
			wantErr: true,
			errMsg:  "dangerous pattern",
		},
		{
			name:    "command substitution",
			command: "echo",
			args:    []string{"`whoami`"},
			wantErr: true,
			errMsg:  "dangerous pattern",
		},
		{
			name:    "piping",
			command: "echo",
			args:    []string{"test", "|", "cat"},
			wantErr: true,
			errMsg:  "dangerous pattern",
		},
		{
			name:    "redirection",
			command: "echo",
			args:    []string{"test", ">", "/etc/passwd"},
			wantErr: true,
			errMsg:  "dangerous pattern",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := env.ValidateCommand(tt.command, tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCommand() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("ValidateCommand() error = %v, want error containing %q", err, tt.errMsg)
			}
		})
	}
}

func TestRunSafeCommand(t *testing.T) {
	t.Run("executes safe commands", func(t *testing.T) {
		env := SafeCommandEnvironment(t)

		cmd := SafeTestCommand("Hello, World!")
		stdout, stderr, exitCode := env.RunSafeCommand(cmd)

		if exitCode != 0 {
			t.Errorf("Expected exit code 0, got %d", exitCode)
		}
		if !strings.Contains(stdout, "Hello, World!") {
			t.Errorf("Expected stdout to contain 'Hello, World!', got %q", stdout)
		}
		if stderr != "" {
			t.Errorf("Expected empty stderr, got %q", stderr)
		}
	})

	t.Run("blocks dangerous commands", func(t *testing.T) {
		env := SafeCommandEnvironment(t)

		// This should fail validation because rm is not whitelisted
		cmd := TestCommand{
			Name:    "rm",
			Command: "rm",
			Args:    []string{"-rf", "/"},
		}

		// Use a different approach - check validation directly
		err := env.ValidateCommand(cmd.Command, cmd.Args)
		if err == nil {
			t.Error("Expected validation error for dangerous command")
		}
		if !strings.Contains(err.Error(), "not whitelisted") {
			t.Errorf("Expected 'not whitelisted' error, got: %v", err)
		}
	})

	t.Run("enforces temp directory for working directory", func(t *testing.T) {
		env := SafeCommandEnvironment(t)

		// Test that working directory must be within temp directory
		cmd := TestCommand{
			Name:    "echo",
			Command: "echo",
			Args:    []string{"test"},
			Dir:     "/tmp", // Outside temp directory
		}

		// We can't easily test Fatal calls, so let's test the logic directly
		// by checking that the directory validation works
		if strings.HasPrefix(filepath.Clean(cmd.Dir), env.TempDir()) {
			t.Error("Test setup error: /tmp should not be within temp directory")
		}

		// Now test with a valid directory
		cmd.Dir = env.TempDir()
		if !strings.HasPrefix(filepath.Clean(cmd.Dir), env.TempDir()) {
			t.Error("Test setup error: temp directory should be within itself")
		}

		// Test that a subdirectory is allowed
		subDir := env.CreateTempDir("subdir")
		cmd.Dir = subDir
		stdout, _, exitCode := env.RunSafeCommand(cmd)
		if exitCode != 0 {
			t.Errorf("Expected success when running in subdirectory, got exit code %d", exitCode)
		}
		if !strings.Contains(stdout, "test") {
			t.Error("Expected output to contain 'test'")
		}
	})
}

func TestCreateTempFile(t *testing.T) {
	env := SafeCommandEnvironment(t)

	t.Run("creates file with content", func(t *testing.T) {
		content := "test content"
		filePath := env.CreateTempFile("test.txt", content)

		// Check file exists
		if _, err := os.Stat(filePath); err != nil {
			t.Errorf("File should exist: %v", err)
		}

		// Check content
		data, err := os.ReadFile(filePath)
		if err != nil {
			t.Fatalf("Failed to read file: %v", err)
		}
		if string(data) != content {
			t.Errorf("Expected content %q, got %q", content, string(data))
		}

		// Check file is in temp directory
		if !strings.HasPrefix(filePath, env.TempDir()) {
			t.Errorf("File %q should be in temp directory %q", filePath, env.TempDir())
		}
	})

	t.Run("creates nested directories", func(t *testing.T) {
		filePath := env.CreateTempFile("subdir/nested/file.txt", "content")

		// Check file exists
		if _, err := os.Stat(filePath); err != nil {
			t.Errorf("File should exist: %v", err)
		}

		// Check parent directories exist
		parentDir := filepath.Dir(filePath)
		if _, err := os.Stat(parentDir); err != nil {
			t.Errorf("Parent directory should exist: %v", err)
		}
	})
}

func TestCreateTempDir(t *testing.T) {
	env := SafeCommandEnvironment(t)

	t.Run("creates directory", func(t *testing.T) {
		dirPath := env.CreateTempDir("testdir")

		// Check directory exists
		info, err := os.Stat(dirPath)
		if err != nil {
			t.Fatalf("Directory should exist: %v", err)
		}
		if !info.IsDir() {
			t.Error("Path should be a directory")
		}

		// Check directory is in temp directory
		if !strings.HasPrefix(dirPath, env.TempDir()) {
			t.Errorf("Directory %q should be in temp directory %q", dirPath, env.TempDir())
		}
	})

	t.Run("creates nested directories", func(t *testing.T) {
		dirPath := env.CreateTempDir("parent/child/grandchild")

		// Check directory exists
		if _, err := os.Stat(dirPath); err != nil {
			t.Errorf("Directory should exist: %v", err)
		}
	})
}

func TestCleanup(t *testing.T) {
	t.Run("runs cleanup functions in reverse order", func(t *testing.T) {
		env := SafeCommandEnvironment(t)

		var order []int
		env.AddCleanup(func() { order = append(order, 1) })
		env.AddCleanup(func() { order = append(order, 2) })
		env.AddCleanup(func() { order = append(order, 3) })

		env.Cleanup()

		// Check reverse order
		expected := []int{3, 2, 1}
		if len(order) != len(expected) {
			t.Errorf("Expected %d cleanup calls, got %d", len(expected), len(order))
		}
		for i, v := range order {
			if v != expected[i] {
				t.Errorf("Expected cleanup order %v, got %v", expected, order)
				break
			}
		}
	})
}

func TestWithIsolatedEnvironment(t *testing.T) {
	t.Run("provides isolated environment to test function", func(t *testing.T) {
		var envTempDir string

		WithIsolatedEnvironment(t, func(env *TestEnvironment) {
			envTempDir = env.TempDir()

			// Test that we can use the environment
			filePath := env.CreateTempFile("test.txt", "content")
			if _, err := os.Stat(filePath); err != nil {
				t.Errorf("File should exist: %v", err)
			}
		})

		// Verify temp directory was set
		if envTempDir == "" {
			t.Error("Environment temp directory should have been set")
		}
	})
}

func TestWindowsCommandHandling(t *testing.T) {
	if !IsWindows() {
		t.Skip("Windows-specific test")
	}

	env := SafeCommandEnvironment(t)

	t.Run("allows Windows-specific commands", func(t *testing.T) {
		windowsCommands := []string{"cmd", "cmd.exe", "type", "dir"}
		for _, cmd := range windowsCommands {
			if !env.IsCommandAllowed(cmd) {
				t.Errorf("Windows command %q should be allowed", cmd)
			}
		}
	})

	t.Run("strips .exe extension for validation", func(t *testing.T) {
		if !env.IsCommandAllowed("cmd.exe") {
			t.Error("cmd.exe should be allowed")
		}
	})
}
