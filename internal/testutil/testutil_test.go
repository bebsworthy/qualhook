//go:build unit

package testutil

import (
	"os"
	"strings"
	"testing"
)

func TestConfigBuilder(t *testing.T) {
	t.Run("basic config creation", func(t *testing.T) {
		cfg := NewConfigBuilder().
			WithVersion("1.0").
			WithSimpleCommand("lint", "eslint", ".", "--fix").
			Build()

		if cfg.Version != "1.0" {
			t.Errorf("Expected version 1.0, got %s", cfg.Version)
		}

		if len(cfg.Commands) != 1 {
			t.Errorf("Expected 1 command, got %d", len(cfg.Commands))
		}

		lintCmd := cfg.Commands["lint"]
		if lintCmd == nil {
			t.Fatal("Expected lint command to exist")
		}

		if lintCmd.Command != "eslint" {
			t.Errorf("Expected command eslint, got %s", lintCmd.Command)
		}

		if len(lintCmd.Args) != 2 || lintCmd.Args[0] != "." || lintCmd.Args[1] != "--fix" {
			t.Errorf("Expected args [. --fix], got %v", lintCmd.Args)
		}
	})

	t.Run("default test config", func(t *testing.T) {
		cfg := DefaultTestConfig()

		if cfg.Version != "1.0" {
			t.Errorf("Expected version 1.0, got %s", cfg.Version)
		}

		expectedCommands := []string{"lint", "format", "test", "typecheck"}
		if len(cfg.Commands) != len(expectedCommands) {
			t.Errorf("Expected %d commands, got %d", len(expectedCommands), len(cfg.Commands))
		}

		for _, cmd := range expectedCommands {
			if _, exists := cfg.Commands[cmd]; !exists {
				t.Errorf("Expected command %s to exist", cmd)
			}
		}
	})

	t.Run("write to file", func(t *testing.T) {
		tempDir := t.TempDir()
		configPath := tempDir + "/.qualhook.json"

		builder := NewConfigBuilder().
			WithVersion("1.0").
			WithSimpleCommand("test", "npm", "test")

		err := builder.WriteToFile(configPath)
		if err != nil {
			t.Fatalf("Failed to write config: %v", err)
		}

		// Verify file exists
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			t.Error("Config file was not created")
		}
	})
}

func TestOutputCapture(t *testing.T) {
	t.Run("capture stdout", func(t *testing.T) {
		stdout, err := CaptureStdout(func() {
			os.Stdout.WriteString("Hello, stdout!")
		})

		if err != nil {
			t.Fatalf("Failed to capture stdout: %v", err)
		}

		if !strings.Contains(stdout, "Hello, stdout!") {
			t.Errorf("Expected stdout to contain 'Hello, stdout!', got: %s", stdout)
		}
	})

	t.Run("capture stderr", func(t *testing.T) {
		stderr, err := CaptureStderr(func() {
			os.Stderr.WriteString("Hello, stderr!")
		})

		if err != nil {
			t.Fatalf("Failed to capture stderr: %v", err)
		}

		if !strings.Contains(stderr, "Hello, stderr!") {
			t.Errorf("Expected stderr to contain 'Hello, stderr!', got: %s", stderr)
		}
	})

	t.Run("capture both", func(t *testing.T) {
		stdout, stderr, err := CaptureOutput(func() {
			os.Stdout.WriteString("stdout message")
			os.Stderr.WriteString("stderr message")
		})

		if err != nil {
			t.Fatalf("Failed to capture output: %v", err)
		}

		if !strings.Contains(stdout, "stdout message") {
			t.Errorf("Expected stdout to contain 'stdout message', got: %s", stdout)
		}

		if !strings.Contains(stderr, "stderr message") {
			t.Errorf("Expected stderr to contain 'stderr message', got: %s", stderr)
		}
	})
}

func TestCommandHelpers(t *testing.T) {
	t.Run("safe test command", func(t *testing.T) {
		cmd := SafeTestCommand("Hello, World!")
		stdout, stderr, exitCode := RunCommand(t, cmd)

		if exitCode != 0 {
			t.Errorf("Expected exit code 0, got %d", exitCode)
		}

		if !strings.Contains(stdout, "Hello, World!") {
			t.Errorf("Expected stdout to contain 'Hello, World!', got: %s", stdout)
		}

		if stderr != "" {
			t.Errorf("Expected no stderr, got: %s", stderr)
		}
	})

	t.Run("failing test command", func(t *testing.T) {
		cmd := FailingTestCommand()
		_, _, exitCode := RunCommand(t, cmd)

		if exitCode == 0 {
			t.Error("Expected non-zero exit code")
		}
	})

	t.Run("successful test command", func(t *testing.T) {
		cmd := SuccessfulTestCommand()
		_, _, exitCode := RunCommand(t, cmd)

		if exitCode != 0 {
			t.Errorf("Expected exit code 0, got %d", exitCode)
		}
	})
}

func TestTestWriter(t *testing.T) {
	tw := NewTestWriter()

	// Write some data
	n, err := tw.Write([]byte("Hello"))
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if n != 5 {
		t.Errorf("Expected to write 5 bytes, wrote %d", n)
	}

	// Write more data
	tw.Write([]byte(", World!"))

	// Check content
	if tw.String() != "Hello, World!" {
		t.Errorf("Expected 'Hello, World!', got '%s'", tw.String())
	}

	// Reset and verify
	tw.Reset()
	if tw.String() != "" {
		t.Errorf("Expected empty string after reset, got '%s'", tw.String())
	}
}
