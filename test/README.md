# Qualhook Test Architecture

This document provides comprehensive guidance on testing in the Qualhook project, including test organization, patterns, utilities, and best practices.

## Table of Contents

1. [Test Categories](#test-categories)
2. [Test Organization](#test-organization)
3. [Writing Tests](#writing-tests)
4. [Test Utilities](#test-utilities)
5. [Test Data Management](#test-data-management)
6. [Performance Guidelines](#performance-guidelines)
7. [Best Practices](#best-practices)
8. [CI/CD Integration](#cicd-integration)

## Test Categories

Qualhook uses build tags to categorize tests into three main types:

### Unit Tests (`//go:build unit`)

Unit tests focus on testing individual components in isolation with minimal dependencies.

- **Purpose**: Test individual functions, methods, and small components
- **Scope**: Single package or module
- **Dependencies**: Minimal, use interfaces and test utilities
- **Execution Time**: Fast (< 100ms per test)
- **Location**: Same package as the code being tested

**Example:**
```go
//go:build unit

package executor

import (
    "testing"
    "time"
)

func TestNewCommandExecutor(t *testing.T) {
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
            name:            "with zero timeout uses default",
            timeout:         0,
            expectedTimeout: 2 * time.Minute,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            executor := NewCommandExecutor(tt.timeout)
            if executor.defaultTimeout != tt.expectedTimeout {
                t.Errorf("expected timeout %v, got %v", 
                    tt.expectedTimeout, executor.defaultTimeout)
            }
        })
    }
}
```

### Integration Tests (`//go:build integration`)

Integration tests verify that multiple components work correctly together.

- **Purpose**: Test interactions between components
- **Scope**: Multiple packages or external dependencies
- **Dependencies**: Real implementations, file system, processes
- **Execution Time**: Moderate (< 5s per test)
- **Location**: `*_integration_test.go` files or dedicated test packages

**Example:**
```go
//go:build integration

package qualhook_test

import (
    "testing"
    "github.com/bebsworthy/qualhook/internal/config"
    "github.com/bebsworthy/qualhook/internal/executor"
    "github.com/bebsworthy/qualhook/internal/testutil"
)

func TestCommandExecutionWithConfig(t *testing.T) {
    // Create test configuration
    cfg := testutil.NewConfigBuilder().
        WithSimpleCommand("lint", "echo", "linting").
        WithSimpleCommand("test", "echo", "testing").
        Build()

    // Test that executor properly uses config
    exec := executor.New(executor.WithTimeout(5 * time.Second))
    
    result, err := exec.Execute(cfg.Commands["lint"])
    if err != nil {
        t.Fatalf("execution failed: %v", err)
    }
    
    if !strings.Contains(result.Stdout, "linting") {
        t.Errorf("expected output to contain 'linting', got %s", result.Stdout)
    }
}
```

### End-to-End Tests (`//go:build e2e`)

E2E tests verify complete workflows from the user's perspective.

- **Purpose**: Test complete user scenarios
- **Scope**: Full application behavior
- **Dependencies**: Complete system, external tools
- **Execution Time**: Slower (may take several seconds)
- **Location**: `e2e_test.go` files, typically in cmd package

**Example:**
```go
//go:build e2e

package main_test

import (
    "os"
    "os/exec"
    "path/filepath"
    "testing"
    "github.com/bebsworthy/qualhook/internal/testutil"
)

func TestQualhookFullWorkflow(t *testing.T) {
    // Skip if required tools not available
    testutil.SkipIfShort(t)
    
    // Create temporary project
    tmpDir := t.TempDir()
    configPath := filepath.Join(tmpDir, ".qualhook.json")
    
    // Initialize configuration
    cmd := exec.Command("qualhook", "init")
    cmd.Dir = tmpDir
    if err := cmd.Run(); err != nil {
        t.Fatalf("init failed: %v", err)
    }
    
    // Run lint command
    cmd = exec.Command("qualhook", "lint")
    cmd.Dir = tmpDir
    output, err := cmd.CombinedOutput()
    if err != nil {
        t.Fatalf("lint failed: %v\nOutput: %s", err, output)
    }
    
    // Verify output
    if !strings.Contains(string(output), "Linting completed") {
        t.Errorf("unexpected output: %s", output)
    }
}
```

## Test Organization

### Directory Structure

```
qualhook/
├── cmd/qualhook/
│   ├── *_test.go          # Unit tests for commands
│   ├── *_integration_test.go  # Integration tests
│   └── e2e_test.go        # End-to-end tests
├── internal/
│   ├── */
│   │   ├── *_test.go      # Unit tests
│   │   └── *_integration_test.go
│   └── testutil/          # Shared test utilities
├── pkg/
│   └── */
│       └── *_test.go      # Public API tests
└── test/
    └── fixtures/          # Test data and fixtures
        ├── golang/
        ├── nodejs/
        ├── python/
        └── rust/
```

### Running Tests by Category

```bash
# Run all tests
go test ./...

# Run only unit tests
go test -tags=unit ./...

# Run only integration tests
go test -tags=integration ./...

# Run only e2e tests
go test -tags=e2e ./...

# Run with coverage
go test -tags=unit -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## Writing Tests

### Table-Driven Tests

The preferred pattern for writing tests in Qualhook is table-driven tests:

```go
func TestFeature(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
        wantErr  bool
    }{
        {
            name:     "valid input",
            input:    "test",
            expected: "TEST",
            wantErr:  false,
        },
        {
            name:     "empty input",
            input:    "",
            expected: "",
            wantErr:  true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := Feature(tt.input)
            
            if (err != nil) != tt.wantErr {
                t.Errorf("Feature() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            
            if result != tt.expected {
                t.Errorf("Feature() = %v, want %v", result, tt.expected)
            }
        })
    }
}
```

### Error Testing

When testing error conditions:

```go
func TestErrorHandling(t *testing.T) {
    tests := []struct {
        name        string
        input       string
        wantErr     bool
        errContains string
    }{
        {
            name:        "invalid command",
            input:       "invalid",
            wantErr:     true,
            errContains: "command not found",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            _, err := Execute(tt.input)
            
            if (err != nil) != tt.wantErr {
                t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            
            if tt.wantErr && tt.errContains != "" {
                if !strings.Contains(err.Error(), tt.errContains) {
                    t.Errorf("error %v should contain %q", err, tt.errContains)
                }
            }
        })
    }
}
```

## Test Utilities

The `internal/testutil` package provides comprehensive utilities for testing:

### ConfigBuilder

Create test configurations easily:

```go
// Simple configuration
cfg := testutil.DefaultTestConfig()

// Custom configuration
cfg := testutil.NewConfigBuilder().
    WithCommand("lint", "eslint", []string{"."}, nil).
    WithSimpleCommand("test", "jest").
    WithPath("src/", testutil.NewConfigBuilder().
        WithSimpleCommand("format", "prettier", "--write", ".").
        Build()).
    Build()
```

### Output Capture

Capture stdout and stderr during tests:

```go
// Capture both stdout and stderr
stdout, stderr, err := testutil.CaptureOutput(func() {
    fmt.Println("stdout message")
    fmt.Fprintln(os.Stderr, "stderr message")
})

// Capture only stdout
stdout := testutil.CaptureStdout(t, func() {
    fmt.Println("message")
})

// Use TestWriter for concurrent-safe output
writer := testutil.NewTestWriter(t)
fmt.Fprintln(writer, "test output")
```

### Command Helpers

Cross-platform command utilities:

```go
// Get a safe test command
cmd := testutil.SafeTestCommand("Hello, World!")

// Run command and capture output
stdout, stderr, exitCode := testutil.RunCommand(t, cmd)

// Platform-specific testing
if runtime.GOOS == "windows" {
    testutil.SkipIfNotWindows(t)
}

// Get platform-safe commands
echoCmd := testutil.SafeCommands["echo"]
```

### Test Helpers

```go
// Skip tests conditionally
testutil.SkipIfShort(t)  // Skip if -short flag is set
testutil.SkipIfCI(t)     // Skip in CI environment
testutil.SkipIfNotWindows(t)  // Skip on non-Windows

// Temporary directories with cleanup
tmpDir := testutil.TempDir(t, "test-*")

// Assert file contents
testutil.AssertFileContains(t, filepath, "expected content")

// Wait for condition
testutil.WaitFor(t, 5*time.Second, func() bool {
    return serviceReady()
})
```

## Test Data Management

### Fixtures

Test fixtures are located in `test/fixtures/` organized by project type:

```go
func TestWithFixture(t *testing.T) {
    // Use existing fixtures
    fixturePath := "test/fixtures/golang/main.go"
    
    // Copy fixture to temp directory
    tmpDir := t.TempDir()
    testutil.CopyFile(t, fixturePath, filepath.Join(tmpDir, "main.go"))
    
    // Run test with fixture
    result := ProcessGoFile(filepath.Join(tmpDir, "main.go"))
    // ... assertions
}
```

### Creating Test Data

```go
func setupTestProject(t *testing.T) string {
    tmpDir := t.TempDir()
    
    // Create project structure
    testutil.WriteFile(t, filepath.Join(tmpDir, "go.mod"), `
module testproject

go 1.21
`)
    
    testutil.WriteFile(t, filepath.Join(tmpDir, "main.go"), `
package main

func main() {
    println("Hello, World!")
}
`)
    
    return tmpDir
}
```

## Performance Guidelines

### Benchmarking

Write benchmarks for performance-critical code:

```go
func BenchmarkExecutor(b *testing.B) {
    executor := NewCommandExecutor(5 * time.Second)
    cmd := testutil.SafeTestCommand("test")
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, _ = executor.Execute(cmd)
    }
}

// Table-driven benchmarks
func BenchmarkFilter(b *testing.B) {
    tests := []struct {
        name  string
        input string
    }{
        {"small", generateInput(100)},
        {"medium", generateInput(1000)},
        {"large", generateInput(10000)},
    }
    
    for _, tt := range tests {
        b.Run(tt.name, func(b *testing.B) {
            b.ResetTimer()
            for i := 0; i < b.N; i++ {
                _ = Filter(tt.input)
            }
        })
    }
}
```

### Performance Testing Best Practices

1. **Reset Timer**: Use `b.ResetTimer()` after setup
2. **Avoid Allocations**: Minimize allocations in hot paths
3. **Parallel Benchmarks**: Use `b.RunParallel()` for concurrent code
4. **Memory Benchmarks**: Report allocations with `b.ReportAllocs()`

```go
func BenchmarkParallel(b *testing.B) {
    b.ReportAllocs()
    b.RunParallel(func(pb *testing.PB) {
        for pb.Next() {
            // Benchmark code
        }
    })
}
```

## Best Practices

### 1. Test Naming

- Use descriptive test names: `TestExecutor_Execute_WithTimeout`
- For table tests, use clear scenario names
- Prefix benchmark names with `Benchmark`

### 2. Test Independence

- Each test should be independent
- Use `t.TempDir()` for temporary files
- Clean up resources with `t.Cleanup()`

```go
func TestWithCleanup(t *testing.T) {
    resource := acquireResource()
    t.Cleanup(func() {
        resource.Close()
    })
    
    // Test code
}
```

### 3. Assertions

- Use clear error messages
- Include actual vs expected values
- Use subtests for better organization

```go
if got != want {
    t.Errorf("ProcessFile() = %v, want %v", got, want)
}
```

### 4. Test Coverage

- Aim for >80% coverage for unit tests
- Focus on critical paths
- Test error conditions
- Don't test generated code

### 5. Parallel Testing

Mark tests that can run in parallel:

```go
func TestParallel(t *testing.T) {
    t.Parallel()
    
    // Test code
}
```

### 6. Testing Interfaces

Test through interfaces when possible:

```go
type Executor interface {
    Execute(cmd Command) (Result, error)
}

func TestExecutorInterface(t *testing.T, exec Executor) {
    // Test any implementation
}
```

## CI/CD Integration

### GitHub Actions Configuration

Tests are automatically run in CI with proper categorization:

```yaml
- name: Unit Tests
  run: go test -tags=unit -v -coverprofile=coverage.out ./...

- name: Integration Tests
  run: go test -tags=integration -v ./...

- name: E2E Tests
  run: go test -tags=e2e -v ./...
```

### Pre-commit Hooks

Configure pre-commit hooks to run tests:

```json
{
  "commands": {
    "pre-commit": {
      "run": "go test -tags=unit -short ./...",
      "description": "Run unit tests before commit"
    }
  }
}
```

## Common Patterns and Examples

### Testing File Operations

```go
func TestFileOperation(t *testing.T) {
    tmpDir := t.TempDir()
    testFile := filepath.Join(tmpDir, "test.txt")
    
    // Write test file
    if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
        t.Fatalf("failed to create test file: %v", err)
    }
    
    // Test operation
    result, err := ProcessFile(testFile)
    if err != nil {
        t.Fatalf("ProcessFile failed: %v", err)
    }
    
    // Verify result
    if result != "expected" {
        t.Errorf("got %q, want %q", result, "expected")
    }
}
```

### Testing Command Execution

```go
func TestCommandExecution(t *testing.T) {
    tests := []struct {
        name     string
        command  string
        args     []string
        wantExit int
    }{
        {
            name:     "successful command",
            command:  testutil.SafeCommands["echo"],
            args:     []string{"hello"},
            wantExit: 0,
        },
        {
            name:     "failing command",
            command:  testutil.SafeCommands["false"],
            args:     []string{},
            wantExit: 1,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            cmd := exec.Command(tt.command, tt.args...)
            stdout, stderr, exitCode := testutil.RunCommand(t, cmd)
            
            if exitCode != tt.wantExit {
                t.Errorf("exit code = %d, want %d\nstdout: %s\nstderr: %s",
                    exitCode, tt.wantExit, stdout, stderr)
            }
        })
    }
}
```

### Testing Concurrent Code

```go
func TestConcurrentOperation(t *testing.T) {
    op := NewConcurrentOperation()
    
    var wg sync.WaitGroup
    errors := make(chan error, 10)
    
    // Run concurrent operations
    for i := 0; i < 10; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()
            if err := op.Process(id); err != nil {
                errors <- err
            }
        }(i)
    }
    
    wg.Wait()
    close(errors)
    
    // Check for errors
    for err := range errors {
        t.Errorf("concurrent operation failed: %v", err)
    }
}
```

## Troubleshooting

### Common Issues

1. **Tests timing out**: Increase timeout or use `testutil.WaitFor()`
2. **Flaky tests**: Check for race conditions, use `-race` flag
3. **Platform-specific failures**: Use platform detection helpers
4. **Missing fixtures**: Ensure fixtures are included in version control

### Debugging Tests

```go
// Enable verbose logging
func TestWithDebug(t *testing.T) {
    if testing.Verbose() {
        t.Logf("Debug: starting test with input %v", input)
    }
    
    // Use testutil.TestWriter for captured output
    writer := testutil.NewTestWriter(t)
    fmt.Fprintln(writer, "debug output")
}
```

## Contributing

When adding new tests:

1. Choose the appropriate test category
2. Follow the established patterns
3. Add necessary test utilities to `testutil`
4. Update fixtures if needed
5. Ensure tests pass on all platforms
6. Maintain test coverage above 80%

For questions or improvements to the test architecture, please open an issue or pull request.