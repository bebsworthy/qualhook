package executor

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestNewCommandExecutor(t *testing.T) {
	tests := []struct {
		name           string
		timeout        time.Duration
		expectedTimeout time.Duration
	}{
		{
			name:           "with valid timeout",
			timeout:        5 * time.Second,
			expectedTimeout: 5 * time.Second,
		},
		{
			name:           "with zero timeout",
			timeout:        0,
			expectedTimeout: 2 * time.Minute,
		},
		{
			name:           "with negative timeout",
			timeout:        -1 * time.Second,
			expectedTimeout: 2 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := NewCommandExecutor(tt.timeout)
			if executor.defaultTimeout != tt.expectedTimeout {
				t.Errorf("expected timeout %v, got %v", tt.expectedTimeout, executor.defaultTimeout)
			}
		})
	}
}

func TestExecute_Success(t *testing.T) {
	executor := NewCommandExecutor(10 * time.Second)

	// Use a simple command that exists on all platforms
	var cmd string
	var args []string
	if runtime.GOOS == "windows" {
		cmd = "cmd"
		args = []string{"/c", "echo", "hello world"}
	} else {
		cmd = "echo"
		args = []string{"hello world"}
	}

	result, err := executor.Execute(cmd, args, ExecOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}

	if !strings.Contains(result.Stdout, "hello world") {
		t.Errorf("expected stdout to contain 'hello world', got %q", result.Stdout)
	}

	if result.TimedOut {
		t.Error("command should not have timed out")
	}

	if result.Error != nil {
		t.Errorf("expected no error, got %v", result.Error)
	}
}

func TestExecute_CommandNotFound(t *testing.T) {
	executor := NewCommandExecutor(10 * time.Second)

	result, err := executor.Execute("this-command-does-not-exist-12345", []string{}, ExecOptions{})
	if err != nil {
		t.Fatalf("Execute should not return error for command not found: %v", err)
	}

	if result.Error == nil {
		t.Fatal("expected error in result")
	}

	var execErr *ExecError
	if !errors.As(result.Error, &execErr) {
		t.Fatalf("expected ExecError, got %T", result.Error)
	}

	if execErr.Type != ErrorTypeCommandNotFound {
		t.Errorf("expected ErrorTypeCommandNotFound, got %v", execErr.Type)
	}

	if !errors.Is(result.Error, ErrCommandNotFound) {
		t.Error("error should match ErrCommandNotFound")
	}
}

func TestExecute_Timeout(t *testing.T) {
	executor := NewCommandExecutor(10 * time.Second)

	// Use a command that sleeps
	var cmd string
	var args []string
	if runtime.GOOS == "windows" {
		cmd = "cmd"
		args = []string{"/c", "timeout", "/t", "5", "/nobreak"}
	} else {
		cmd = "sleep"
		args = []string{"5"}
	}

	result, err := executor.Execute(cmd, args, ExecOptions{
		Timeout: 100 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.TimedOut {
		t.Error("expected command to timeout")
	}

	// Exit code might vary on timeout
	if result.ExitCode == 0 {
		t.Error("expected non-zero exit code for timeout")
	}
}

func TestExecute_NonZeroExit(t *testing.T) {
	executor := NewCommandExecutor(10 * time.Second)

	// Use a command that exits with non-zero
	var cmd string
	var args []string
	if runtime.GOOS == "windows" {
		cmd = "cmd"
		args = []string{"/c", "exit", "1"}
	} else {
		cmd = "sh"
		args = []string{"-c", "exit 1"}
	}

	result, err := executor.Execute(cmd, args, ExecOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ExitCode != 1 {
		t.Errorf("expected exit code 1, got %d", result.ExitCode)
	}

	if result.Error != nil {
		t.Errorf("expected no error for non-zero exit, got %v", result.Error)
	}
}

func TestExecute_WorkingDirectory(t *testing.T) {
	executor := NewCommandExecutor(10 * time.Second)

	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("", "qualhook-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Use pwd/cd to verify working directory
	var cmd string
	var args []string
	if runtime.GOOS == "windows" {
		cmd = "cmd"
		args = []string{"/c", "cd"}
	} else {
		cmd = "pwd"
		args = []string{}
	}

	result, err := executor.Execute(cmd, args, ExecOptions{
		WorkingDir: tmpDir,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}

	// Normalize paths for comparison
	expectedPath, _ := filepath.Abs(tmpDir)
	actualPath := strings.TrimSpace(result.Stdout)
	
	if !strings.Contains(actualPath, filepath.Base(expectedPath)) {
		t.Errorf("expected working directory %q in output, got %q", expectedPath, actualPath)
	}
}

func TestExecute_InvalidWorkingDirectory(t *testing.T) {
	executor := NewCommandExecutor(10 * time.Second)

	var cmd string
	if runtime.GOOS == "windows" {
		cmd = "cmd"
	} else {
		cmd = "echo"
	}

	_, err := executor.Execute(cmd, []string{}, ExecOptions{
		WorkingDir: "/this/does/not/exist/12345",
	})
	if err == nil {
		t.Fatal("expected error for invalid working directory")
	}

	if !strings.Contains(err.Error(), "invalid working directory") {
		t.Errorf("expected 'invalid working directory' error, got %v", err)
	}
}

func TestExecute_Environment(t *testing.T) {
	executor := NewCommandExecutor(10 * time.Second)

	// Command to print environment variable
	var cmd string
	var args []string
	testVar := "QUALHOOK_TEST_VAR"
	testValue := "test-value-12345"
	
	if runtime.GOOS == "windows" {
		cmd = "cmd"
		args = []string{"/c", "echo", "%" + testVar + "%"}
	} else {
		cmd = "sh"
		args = []string{"-c", "echo $" + testVar}
	}

	result, err := executor.Execute(cmd, args, ExecOptions{
		Environment: []string{testVar + "=" + testValue},
		InheritEnv:  true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result.Stdout, testValue) {
		t.Errorf("expected output to contain %q, got %q", testValue, result.Stdout)
	}
}

func TestExecute_EmptyCommand(t *testing.T) {
	executor := NewCommandExecutor(10 * time.Second)

	_, err := executor.Execute("", []string{}, ExecOptions{})
	if err == nil {
		t.Fatal("expected error for empty command")
	}

	if !strings.Contains(err.Error(), "command cannot be empty") {
		t.Errorf("expected 'command cannot be empty' error, got %v", err)
	}
}

func TestExecuteWithStreaming(t *testing.T) {
	executor := NewCommandExecutor(10 * time.Second)

	var stdoutBuf, stderrBuf bytes.Buffer

	// Command that outputs to both stdout and stderr
	var cmd string
	var args []string
	if runtime.GOOS == "windows" {
		cmd = "cmd"
		args = []string{"/c", "echo stdout message && echo stderr message 1>&2"}
	} else {
		cmd = "sh"
		args = []string{"-c", "echo stdout message && echo stderr message >&2"}
	}

	result, err := executor.ExecuteWithStreaming(cmd, args, ExecOptions{}, &stdoutBuf, &stderrBuf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check that output was captured in result
	if !strings.Contains(result.Stdout, "stdout message") {
		t.Errorf("expected stdout in result, got %q", result.Stdout)
	}

	if !strings.Contains(result.Stderr, "stderr message") {
		t.Errorf("expected stderr in result, got %q", result.Stderr)
	}

	// Check that output was streamed to buffers
	if !strings.Contains(stdoutBuf.String(), "stdout message") {
		t.Errorf("expected stdout in buffer, got %q", stdoutBuf.String())
	}

	if !strings.Contains(stderrBuf.String(), "stderr message") {
		t.Errorf("expected stderr in buffer, got %q", stderrBuf.String())
	}
}

func TestPrepareEnvironment(t *testing.T) {
	executor := NewCommandExecutor(10 * time.Second)

	tests := []struct {
		name        string
		options     ExecOptions
		checkEnv    map[string]string
		shouldExist map[string]bool
	}{
		{
			name: "no inherit, custom vars only",
			options: ExecOptions{
				InheritEnv: false,
				Environment: []string{
					"FOO=bar",
					"BAZ=qux",
				},
			},
			checkEnv: map[string]string{
				"FOO": "bar",
				"BAZ": "qux",
			},
			shouldExist: map[string]bool{
				"PATH": false, // Should not inherit PATH
			},
		},
		{
			name: "inherit with override",
			options: ExecOptions{
				InheritEnv: true,
				Environment: []string{
					"PATH=/custom/path",
					"NEW_VAR=value",
				},
			},
			checkEnv: map[string]string{
				"PATH":    "/custom/path",
				"NEW_VAR": "value",
			},
		},
		{
			name: "malformed env vars ignored",
			options: ExecOptions{
				Environment: []string{
					"GOOD=value",
					"BAD_NO_EQUALS",
					"ALSO_GOOD=has=equals",
				},
			},
			checkEnv: map[string]string{
				"GOOD":      "value",
				"ALSO_GOOD": "has=equals",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := executor.prepareEnvironment(tt.options)
			envMap := make(map[string]string)
			
			for _, e := range env {
				parts := strings.SplitN(e, "=", 2)
				if len(parts) == 2 {
					envMap[parts[0]] = parts[1]
				}
			}

			// Check expected values
			for k, v := range tt.checkEnv {
				if envMap[k] != v {
					t.Errorf("expected %s=%s, got %s", k, v, envMap[k])
				}
			}

			// Check existence
			for k, shouldExist := range tt.shouldExist {
				_, exists := envMap[k]
				if exists != shouldExist {
					t.Errorf("expected %s existence to be %v", k, shouldExist)
				}
			}
		})
	}
}

func TestStreamingWriter(t *testing.T) {
	var buf bytes.Buffer
	writer := NewStreamingWriter(&buf)

	// Test concurrent writes
	done := make(chan bool, 3)
	for i := 0; i < 3; i++ {
		go func(n int) {
			msg := fmt.Sprintf("message %d\n", n)
			_, err := writer.Write([]byte(msg))
			if err != nil {
				t.Errorf("write error: %v", err)
			}
			done <- true
		}(i)
	}

	// Wait for all writes
	for i := 0; i < 3; i++ {
		<-done
	}

	output := buf.String()
	for i := 0; i < 3; i++ {
		expected := fmt.Sprintf("message %d\n", i)
		if !strings.Contains(output, expected) {
			t.Errorf("expected output to contain %q", expected)
		}
	}
}