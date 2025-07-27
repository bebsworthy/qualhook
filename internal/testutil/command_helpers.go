package testutil

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

// SafeCommands provides safe commands for testing across different platforms.
var SafeCommands = struct {
	Echo            string
	True            string
	False           string
	Sleep           string
	Cat             string
	Touch           string
	Remove          string
	MakeDir         string
	ListDir         string
	PrintWorkingDir string
}{
	Echo:            "echo",
	True:            getTrueCommand(),
	False:           getFalseCommand(),
	Sleep:           "sleep",
	Cat:             getCatCommand(),
	Touch:           "touch",
	Remove:          getRemoveCommand(),
	MakeDir:         "mkdir",
	ListDir:         "ls",
	PrintWorkingDir: "pwd",
}

// TestCommand represents a test command configuration.
type TestCommand struct {
	Name    string
	Command string
	Args    []string
	Env     []string
	Dir     string
	Timeout time.Duration
}

// SafeTestCommand returns a safe echo command for testing.
func SafeTestCommand(message string) TestCommand {
	return TestCommand{
		Name:    "echo",
		Command: SafeCommands.Echo,
		Args:    []string{message},
		Timeout: 5 * time.Second,
	}
}

// FailingTestCommand returns a command that will exit with code 1.
func FailingTestCommand() TestCommand {
	return TestCommand{
		Name:    "false",
		Command: SafeCommands.False,
		Args:    []string{},
		Timeout: 5 * time.Second,
	}
}

// SuccessfulTestCommand returns a command that will exit with code 0.
func SuccessfulTestCommand() TestCommand {
	return TestCommand{
		Name:    "true",
		Command: SafeCommands.True,
		Args:    []string{},
		Timeout: 5 * time.Second,
	}
}

// SleepCommand returns a sleep command for testing timeouts.
func SleepCommand(duration string) TestCommand {
	return TestCommand{
		Name:    "sleep",
		Command: SafeCommands.Sleep,
		Args:    []string{duration},
		Timeout: 10 * time.Second,
	}
}

// RunCommand executes a test command and returns the output.
func RunCommand(t testing.TB, tc TestCommand) (stdout, stderr string, exitCode int) {
	t.Helper()

	ctx := context.Background()
	if tc.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, tc.Timeout)
		defer cancel()
	}

	cmd := exec.CommandContext(ctx, tc.Command, tc.Args...)
	if tc.Dir != "" {
		cmd.Dir = tc.Dir
	}
	if len(tc.Env) > 0 {
		cmd.Env = append(os.Environ(), tc.Env...)
	}

	outBuf := &TestWriter{}
	errBuf := &TestWriter{}
	cmd.Stdout = outBuf
	cmd.Stderr = errBuf

	err := cmd.Run()
	stdout = outBuf.String()
	stderr = errBuf.String()

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
	}

	return stdout, stderr, exitCode
}

// RequireCommand skips the test if the command is not available.
func RequireCommand(t testing.TB, command string) {
	t.Helper()

	_, err := exec.LookPath(command)
	if err != nil {
		t.Skipf("Command %q not found in PATH", command)
	}
}

// TempScript creates a temporary executable script for testing.
func TempScript(t testing.TB, content string) string {
	t.Helper()

	tmpFile, err := os.CreateTemp(t.TempDir(), "test-script-*.sh")
	if err != nil {
		t.Fatalf("Failed to create temp script: %v", err)
	}
	defer func() { _ = tmpFile.Close() }() //nolint:errcheck

	if runtime.GOOS != windowsOS {
		// Add shebang for Unix-like systems
		content = "#!/bin/sh\n" + content
	}

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("Failed to write script content: %v", err)
	}

	// Make executable on Unix-like systems
	if runtime.GOOS != windowsOS {
		if err := os.Chmod(tmpFile.Name(), 0755); err != nil { //nolint:gosec // G302: Script needs to be executable for testing
			t.Fatalf("Failed to make script executable: %v", err)
		}
	}

	return tmpFile.Name()
}

// Platform-specific command helpers

const (
	cmdExe    = "cmd"
	windowsOS = "windows"
)

func getTrueCommand() string {
	if runtime.GOOS == windowsOS {
		return cmdExe
	}
	return "true"
}

func getFalseCommand() string {
	if runtime.GOOS == windowsOS {
		return cmdExe
	}
	return "false"
}

func getCatCommand() string {
	if runtime.GOOS == windowsOS {
		return "type"
	}
	return "cat"
}

func getRemoveCommand() string {
	if runtime.GOOS == windowsOS {
		return "del"
	}
	return "rm"
}

// CommandArgs returns platform-specific command arguments.
func CommandArgs(command string, args ...string) []string {
	if runtime.GOOS == windowsOS {
		if command == "cmd" {
			// For true command on Windows
			if len(args) == 0 {
				return []string{"/c", "exit 0"}
			}
			// For false command on Windows
			if args[0] == "false" {
				return []string{"/c", "exit 1"}
			}
			return append([]string{"/c"}, args...)
		}
	}
	return args
}

// IsWindows returns true if running on Windows.
func IsWindows() bool {
	return runtime.GOOS == windowsOS
}

// SkipOnWindows skips the test if running on Windows.
func SkipOnWindows(t testing.TB, reason string) {
	t.Helper()
	if IsWindows() {
		t.Skip("Skipping on Windows: " + reason)
	}
}

// SkipOnCI skips the test if running in a CI environment.
func SkipOnCI(t testing.TB, reason string) {
	t.Helper()
	if os.Getenv("CI") != "" {
		t.Skip("Skipping in CI: " + reason)
	}
}

// TestEnvironment provides an isolated environment for test execution.
type TestEnvironment struct {
	tempDir         string
	originalEnv     []string
	originalDir     string
	allowedCommands map[string]bool
	cleanupFuncs    []func()
	mu              sync.Mutex
	t               testing.TB
}

// SafeCommandEnvironment creates a new isolated test environment with command whitelisting.
// The environment provides:
// - A temporary directory for all file operations
// - Command whitelisting to prevent dangerous operations
// - Automatic cleanup of resources
// - Isolated environment variables
func SafeCommandEnvironment(t testing.TB) *TestEnvironment {
	t.Helper()

	// Save original state
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}

	// Create temp directory
	tempDir := t.TempDir()

	// Default allowed commands - only safe ones
	allowedCommands := map[string]bool{
		"echo":    true,
		"true":    true,
		"false":   true,
		"sh":      true,
		"bash":    true,
		"cat":     true,
		"touch":   true,
		"mkdir":   true,
		"ls":      true,
		"pwd":     true,
		"sleep":   true,
		"test":    true,
		"[":       true, // for shell test command
		"cmd":     true, // Windows cmd
		"cmd.exe": true,
		"type":    true, // Windows cat equivalent
		"dir":     true, // Windows ls equivalent
	}

	env := &TestEnvironment{
		tempDir:         tempDir,
		originalEnv:     os.Environ(),
		originalDir:     origDir,
		allowedCommands: allowedCommands,
		cleanupFuncs:    []func(){},
		t:               t,
	}

	// Register cleanup
	t.Cleanup(func() {
		env.Cleanup()
	})

	return env
}

// TempDir returns the temporary directory for this test environment.
func (e *TestEnvironment) TempDir() string {
	return e.tempDir
}

// AddCleanup registers a cleanup function to be called when the environment is cleaned up.
func (e *TestEnvironment) AddCleanup(fn func()) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.cleanupFuncs = append(e.cleanupFuncs, fn)
}

// AllowCommand adds a command to the whitelist.
func (e *TestEnvironment) AllowCommand(command string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.allowedCommands[command] = true
}

// IsCommandAllowed checks if a command is whitelisted for execution.
func (e *TestEnvironment) IsCommandAllowed(command string) bool {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Extract base command name
	baseCmd := filepath.Base(command)
	// Remove .exe extension on Windows
	if runtime.GOOS == windowsOS {
		baseCmd = strings.TrimSuffix(baseCmd, ".exe")
	}

	return e.allowedCommands[baseCmd]
}

// ValidateCommand checks if a command is safe to execute in the test environment.
// Returns an error if the command is not whitelisted or potentially dangerous.
func (e *TestEnvironment) ValidateCommand(command string, args []string) error {
	if command == "" {
		return fmt.Errorf("command cannot be empty")
	}

	if !e.IsCommandAllowed(command) {
		return fmt.Errorf("command %q is not whitelisted for test execution", command)
	}

	// Check for dangerous arguments
	dangerousPatterns := []string{
		"..", // Path traversal
		"/etc",
		"/sys",
		"/proc",
		"/dev",
		"C:\\Windows",
		"C:\\System32",
		"~", // Home directory expansion
		"$", // Variable expansion
		">", // Redirection
		"<",
		"|", // Piping
		"&", // Background execution
		";", // Command chaining
		"`", // Command substitution
	}

	allArgs := strings.Join(args, " ")
	for _, pattern := range dangerousPatterns {
		if strings.Contains(command, pattern) || strings.Contains(allArgs, pattern) {
			return fmt.Errorf("potentially dangerous pattern %q found in command or arguments", pattern)
		}
	}

	return nil
}

// RunSafeCommand executes a command with safety checks in the isolated environment.
func (e *TestEnvironment) RunSafeCommand(tc TestCommand) (stdout, stderr string, exitCode int) {
	e.t.Helper()

	// Validate command
	if err := e.ValidateCommand(tc.Command, tc.Args); err != nil {
		e.t.Fatalf("Command validation failed: %v", err)
	}

	// Set working directory to temp dir if not specified
	if tc.Dir == "" {
		tc.Dir = e.tempDir
	}

	// Ensure directory is within temp directory
	if !strings.HasPrefix(filepath.Clean(tc.Dir), e.tempDir) {
		e.t.Fatalf("Working directory %q must be within temp directory %q", tc.Dir, e.tempDir)
	}

	return RunCommand(e.t, tc)
}

// CreateTempFile creates a temporary file with the given content in the test environment.
func (e *TestEnvironment) CreateTempFile(name, content string) string {
	e.t.Helper()

	filePath := filepath.Join(e.tempDir, name)
	dir := filepath.Dir(filePath)

	// Create directory if needed
	if err := os.MkdirAll(dir, 0750); err != nil {
		e.t.Fatalf("Failed to create directory %q: %v", dir, err)
	}

	// Write file
	if err := os.WriteFile(filePath, []byte(content), 0600); err != nil {
		e.t.Fatalf("Failed to write file %q: %v", filePath, err)
	}

	return filePath
}

// CreateTempDir creates a temporary directory within the test environment.
func (e *TestEnvironment) CreateTempDir(name string) string {
	e.t.Helper()

	dirPath := filepath.Join(e.tempDir, name)
	if err := os.MkdirAll(dirPath, 0750); err != nil {
		e.t.Fatalf("Failed to create directory %q: %v", dirPath, err)
	}

	return dirPath
}

// Cleanup cleans up all resources associated with the test environment.
// This is automatically called via t.Cleanup() but can be called manually if needed.
func (e *TestEnvironment) Cleanup() {
	e.mu.Lock()
	cleanupFuncs := e.cleanupFuncs
	e.mu.Unlock()

	// Run cleanup functions in reverse order
	for i := len(cleanupFuncs) - 1; i >= 0; i-- {
		if fn := cleanupFuncs[i]; fn != nil {
			fn()
		}
	}

	// Restore original directory
	if e.originalDir != "" {
		_ = os.Chdir(e.originalDir) //nolint:errcheck
	}
}

// WithIsolatedEnvironment runs a test function in an isolated environment.
// This is a convenience function that creates a SafeCommandEnvironment and passes it to the test function.
func WithIsolatedEnvironment(t testing.TB, fn func(env *TestEnvironment)) {
	t.Helper()
	env := SafeCommandEnvironment(t)
	fn(env)
}

// TestContext creates a context with a timeout suitable for tests.
// The context is automatically canceled when the test completes.
func TestContext(t testing.TB) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	return ctx
}
