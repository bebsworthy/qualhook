# Executor Package

The `executor` package provides safe and reliable command execution functionality for the qualhook CLI tool. It includes support for subprocess execution, error handling, and parallel execution for monorepo scenarios.

## Components

### 1. Command Executor (`command.go`)
- Safe subprocess execution using Go's `exec.Command`
- Environment variable management with inheritance control
- Working directory validation and setting
- Stdout/stderr stream capture
- Timeout support using context
- Streaming output support for real-time processing

### 2. Error Handling (`errors.go`)
- Comprehensive error classification system
- Distinguishes between different error types:
  - Command not found
  - Permission denied
  - Timeout
  - Working directory issues
  - General execution errors
- Proper error wrapping and unwrapping support
- Integration with Go's errors.Is pattern

### 3. Parallel Execution (`parallel.go`)
- Concurrent command execution using goroutines
- Configurable parallelism limits
- Result aggregation across multiple commands
- Progress reporting callbacks
- Graceful handling of partial failures
- Support for monorepo scenarios

## Usage Examples

### Basic Command Execution
```go
executor := NewCommandExecutor(30 * time.Second)
result, err := executor.Execute("npm", []string{"run", "lint"}, ExecOptions{
    WorkingDir: "/path/to/project",
    Timeout: 10 * time.Second,
})
```

### Parallel Execution for Monorepos
```go
pe := NewParallelExecutor(executor, 4) // Max 4 parallel commands

commands := []ParallelCommand{
    {ID: "frontend", Command: "npm", Args: []string{"test"}, Options: ExecOptions{WorkingDir: "./frontend"}},
    {ID: "backend", Command: "go", Args: []string{"test", "./..."}, Options: ExecOptions{WorkingDir: "./backend"}},
}

result, err := pe.Execute(ctx, commands, progressCallback)
```

### Error Classification
```go
result, err := executor.Execute("unknown-command", []string{}, ExecOptions{})
if result.Error != nil {
    var execErr *ExecError
    if errors.As(result.Error, &execErr) {
        switch execErr.Type {
        case ErrorTypeCommandNotFound:
            // Handle command not found
        case ErrorTypePermissionDenied:
            // Handle permission issues
        case ErrorTypeTimeout:
            // Handle timeout
        }
    }
}
```

## Testing

The package includes comprehensive unit tests covering:
- Command execution success and failure scenarios
- Error classification and handling
- Timeout behavior
- Environment variable management
- Working directory validation
- Parallel execution with various configurations
- Progress reporting
- Cross-platform compatibility (Windows, Linux, macOS)

Run tests with:
```bash
go test ./internal/executor/... -v
```

## Security Considerations

- All command arguments are properly escaped
- Working directories are validated before use
- Environment variables can be controlled (inherit or clean)
- Timeout enforcement prevents runaway processes
- Process cleanup on timeout ensures no zombie processes