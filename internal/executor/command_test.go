//go:build unit

package executor

import (
	"bytes"
	"errors"
	"fmt"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestNewCommandExecutor(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name            string
		timeout         time.Duration
		expectedTimeout time.Duration
	}{
		{
			name:            "with valid timeout",
			timeout:         5 * time.Second,
			expectedTimeout: 5 * time.Second,
		},
		{
			name:            "with zero timeout",
			timeout:         0,
			expectedTimeout: 2 * time.Minute,
		},
		{
			name:            "with negative timeout",
			timeout:         -1 * time.Second,
			expectedTimeout: 2 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			executor := NewCommandExecutor(tt.timeout)
			if executor.defaultTimeout != tt.expectedTimeout {
				t.Errorf("expected timeout %v, got %v", tt.expectedTimeout, executor.defaultTimeout)
			}
		})
	}
}

func TestExecute_CommonScenarios(t *testing.T) {
	t.Parallel()
	executor := NewCommandExecutor(10 * time.Second)
	tests := commonTestScenarios(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cmd, args := tt.setupCmd()
			result, err := executor.Execute(cmd, args, tt.opts)

			// Check error expectations
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Check exec error in result
			if tt.wantExecErr {
				if result.Error == nil {
					t.Fatal("expected error in result")
				}
				var execErr *ExecError
				if !errors.As(result.Error, &execErr) {
					t.Fatalf("expected ExecError, got %T", result.Error)
				}
				if execErr.Type != tt.wantErrType {
					t.Errorf("expected error type %v, got %v", tt.wantErrType, execErr.Type)
				}
				return
			}

			// Check result expectations
			if tt.wantTimedOut && !result.TimedOut {
				t.Error("expected command to timeout")
			}
			if tt.wantTimedOut && result.ExitCode == 0 {
				t.Error("expected non-zero exit code for timeout")
			}
			if !tt.wantTimedOut && tt.wantExitCode != result.ExitCode {
				t.Errorf("expected exit code %d, got %d", tt.wantExitCode, result.ExitCode)
			}
			if tt.wantInStdout != "" && !strings.Contains(result.Stdout, tt.wantInStdout) {
				t.Errorf("expected stdout to contain %q, got %q", tt.wantInStdout, result.Stdout)
			}
		})
	}
}

func TestExecute_CommandSpecific(t *testing.T) {
	t.Parallel()
	executor := NewCommandExecutor(10 * time.Second)

	tests := []struct {
		name         string
		setup        func() (cmd string, args []string, opts ExecOptions, cleanup func())
		wantErr      bool
		wantExitCode int
		wantInErr    string
	}{
		{
			name: "working directory",
			setup: func() (string, []string, ExecOptions, func()) {
				tmpDir, cleanup := createTempDir(t)
				cmd, args := pc.pwd()
				return cmd, args, ExecOptions{WorkingDir: tmpDir}, cleanup
			},
			wantExitCode: 0,
		},
		{
			name: "invalid working directory",
			setup: func() (string, []string, ExecOptions, func()) {
				cmd := echoCommand
				if runtime.GOOS == osWindows {
					cmd = cmdCommand
				}
				return cmd, []string{},
					ExecOptions{WorkingDir: "/this/does/not/exist/12345"}, nil
			},
			wantErr:   true,
			wantInErr: "invalid working directory",
		},
		{
			name: "empty command",
			setup: func() (string, []string, ExecOptions, func()) {
				return "", []string{}, ExecOptions{}, nil
			},
			wantErr:   true,
			wantInErr: "command cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cmd, args, opts, cleanup := tt.setup()
			if cleanup != nil {
				defer cleanup()
			}

			result, err := executor.Execute(cmd, args, opts)

			// Check error expectations
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error but got none")
				}
				if tt.wantInErr != "" && !strings.Contains(err.Error(), tt.wantInErr) {
					t.Errorf("expected error containing %q, got %v", tt.wantInErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			assertExecResult(t, result, tt.wantExitCode, "", "")
		})
	}
}

func TestExecuteWithStreaming(t *testing.T) {
	t.Parallel()
	executor := NewCommandExecutor(10 * time.Second)

	var stdoutBuf, stderrBuf bytes.Buffer

	// Use echo command for stdout test
	cmd, args := pc.echo("stdout message")

	result, err := executor.ExecuteWithStreaming(cmd, args, ExecOptions{}, &stdoutBuf, &stderrBuf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check that output was captured in result
	if !strings.Contains(result.Stdout, "stdout message") {
		t.Errorf("expected stdout in result, got %q", result.Stdout)
	}

	// Check that output was streamed to stdout buffer
	if !strings.Contains(stdoutBuf.String(), "stdout message") {
		t.Errorf("expected stdout in buffer, got %q", stdoutBuf.String())
	}
}

func TestPrepareEnvironment(t *testing.T) {
	// Skip this test as prepareEnvironment is now using security sanitization
	t.Skip("prepareEnvironment now uses security sanitization - testing through integration tests")
	/*

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
	*/
}

func TestStreamingWriter(t *testing.T) {
	t.Parallel()
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
