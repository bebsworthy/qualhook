# Test Isolation Guide

## Overview

The test isolation features in the `testutil` package provide a safe environment for running tests with real executors. This ensures tests are secure, isolated, and won't affect the host system.

## Key Features

1. **Command Whitelisting**: Only safe commands are allowed by default
2. **Isolated File System**: All file operations occur in temporary directories
3. **Automatic Cleanup**: Resources are cleaned up automatically
4. **Security Validation**: Dangerous patterns in commands and arguments are blocked

## Quick Start

### Basic Usage

```go
func TestMyFeature(t *testing.T) {
    // Create an isolated environment
    env := testutil.SafeCommandEnvironment(t)
    
    // Create test files
    testFile := env.CreateTempFile("test.txt", "content")
    
    // Run safe commands
    cmd := testutil.SafeTestCommand("Hello")
    stdout, stderr, exitCode := env.RunSafeCommand(cmd)
}
```

### Using WithIsolatedEnvironment

```go
func TestWithIsolation(t *testing.T) {
    testutil.WithIsolatedEnvironment(t, func(env *testutil.TestEnvironment) {
        // Your test code here
        // The environment is automatically cleaned up
    })
}
```

## Whitelisted Commands

By default, these commands are allowed:
- `echo`, `true`, `false` - Basic utilities
- `sh`, `bash` - Shell interpreters (for safe scripts)
- `cat`, `touch`, `mkdir`, `ls`, `pwd` - File operations
- `sleep`, `test`, `[` - Testing utilities
- `cmd`, `cmd.exe`, `type`, `dir` - Windows equivalents

### Adding Custom Commands

```go
env := testutil.SafeCommandEnvironment(t)
env.AllowCommand("my-safe-tool")
```

## Security Features

### Blocked Patterns

The following patterns are blocked in commands and arguments:
- Path traversal: `..`
- System directories: `/etc`, `/sys`, `/proc`, `/dev`, `C:\Windows`
- Shell expansions: `~`, `$`
- Command chaining: `;`, `&`, `|`
- Redirections: `>`, `<`
- Command substitution: `` ` ``

### Working Directory Restrictions

All commands must run within the temporary directory:

```go
// This will fail - outside temp directory
cmd := testutil.TestCommand{
    Command: "ls",
    Dir: "/tmp",
}

// This works - within temp directory
cmd := testutil.TestCommand{
    Command: "ls",
    Dir: env.TempDir(),
}
```

## Migration Guide

### Before (Unsafe)

```go
func TestOldStyle(t *testing.T) {
    executor := NewCommandExecutor(5 * time.Second)
    
    // Dangerous - affects real file system
    result, err := executor.Execute("rm", []string{"-rf", "/tmp/test"}, ExecOptions{})
}
```

### After (Safe)

```go
func TestNewStyle(t *testing.T) {
    testutil.WithIsolatedEnvironment(t, func(env *testutil.TestEnvironment) {
        executor := NewCommandExecutor(5 * time.Second)
        
        // Safe - only affects isolated temp directory
        testFile := env.CreateTempFile("test.txt", "content")
        result, err := executor.Execute("cat", []string{testFile}, ExecOptions{
            WorkingDir: env.TempDir(),
        })
    })
}
```

## Best Practices

1. **Always use isolation for integration tests** that execute real commands
2. **Create all test files within the environment** using `CreateTempFile` or `CreateTempDir`
3. **Add cleanup functions** for any resources that need special handling
4. **Validate commands** before execution if accepting user input
5. **Use platform-agnostic commands** when possible (e.g., use `SafeCommands` struct)

## Advanced Usage

### Custom Cleanup

```go
env := testutil.SafeCommandEnvironment(t)

// Add custom cleanup
env.AddCleanup(func() {
    // Clean up external resources
    fmt.Println("Cleaning up...")
})
```

### Parallel Tests

```go
func TestParallel(t *testing.T) {
    t.Parallel() // Safe - each test gets its own environment
    
    testutil.WithIsolatedEnvironment(t, func(env *testutil.TestEnvironment) {
        // Test code here
    })
}
```

### Testing with Real Executors

```go
func TestRealExecutor(t *testing.T) {
    testutil.WithIsolatedEnvironment(t, func(env *testutil.TestEnvironment) {
        // Create test script
        script := env.CreateTempFile("script.sh", "#!/bin/sh\necho 'Safe script'")
        
        // Allow the script
        env.AllowCommand(filepath.Base(script))
        
        // Execute safely
        executor := NewCommandExecutor(5 * time.Second)
        result, err := executor.Execute(script, nil, ExecOptions{
            WorkingDir: env.TempDir(),
        })
    })
}
```

## Troubleshooting

### Command Not Whitelisted

If you get "command not whitelisted" errors:
1. Check if the command is in the default whitelist
2. Add it using `env.AllowCommand("command")`
3. Consider if the command is actually safe for tests

### Dangerous Pattern Detected

If you get "dangerous pattern" errors:
1. Check for path traversal (`..`)
2. Avoid system directories
3. Don't use shell expansions or redirections
4. Use full paths within the temp directory

### Working Directory Issues

Ensure all paths are within the temp directory:
```go
// Good
path := filepath.Join(env.TempDir(), "subdir", "file.txt")

// Bad
path := "/tmp/file.txt"
```