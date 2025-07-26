# Test Fixtures

This directory contains test fixtures used throughout the qualhook test suite. Fixtures provide realistic test data and help maintain cleaner, more maintainable tests.

## Directory Structure

### `/configs/`
Sample configuration files for testing config loading and validation:
- `basic.qualhook.json` - Simple configuration with basic lint and test commands
- `complex.qualhook.json` - Advanced configuration with multiple hooks, environment variables, and settings
- `golang.qualhook.json` - Go-specific configuration with golangci-lint, go test, and go vet
- `python.qualhook.json` - Python-specific configuration with flake8, mypy, pytest, and black
- `monorepo.qualhook.json` - Configuration for monorepo projects with workspace-specific commands
- `minimal.qualhook.json` - Minimal valid configuration
- `invalid.qualhook.json` - Invalid configuration for testing error handling

### `/projects/`
Sample project structures for testing project detection and config resolution:
- `/golang/` - Go project with go.mod, main.go, tests, and .qualhook.json
- `/nodejs/` - Node.js project with package.json, index.js, tests, and .qualhook.json
- `/python/` - Python project with pyproject.toml, main.py, tests, and .qualhook.json
- `/monorepo/` - Monorepo with multiple packages and root .qualhook.json
- `/rust/` - Rust project with Cargo.toml and src/main.rs

### `/outputs/`
Expected command outputs for testing output filtering and error detection:
- `error_output.txt` - Sample error output with various error formats
- `complex_logs.txt` - Complex build logs with mixed output
- `large_error_output.txt` - Large error output for performance testing
- `line_numbers.txt` - Output with line number formatting
- `multiple_errors.txt` - Output containing multiple error types

## Usage

### Loading Fixtures in Tests

Use the `testutil` package to load fixtures:

```go
import "github.com/bebsworthy/qualhook/internal/testutil"

// Get path to a fixture
configPath := testutil.ConfigFixture(t, "basic")

// Load fixture content
content := testutil.LoadFixtureString(t, "outputs/error_output.txt")

// Copy fixture to temp directory for modification
tempDir := testutil.CreateTempFixture(t, "projects/golang")

// Get project fixture path
projectPath := testutil.ProjectFixture(t, "nodejs")
```

### Adding New Fixtures

When adding new fixtures:
1. Place files in the appropriate subdirectory
2. Use realistic, representative data
3. Include comments explaining the fixture's purpose
4. Update this README with the new fixture

### Best Practices

1. **Use fixtures instead of hardcoding** - Load test data from fixtures rather than embedding it in test code
2. **Version with tests** - When updating fixtures, ensure related tests are updated
3. **Keep fixtures focused** - Each fixture should test specific functionality
4. **Document edge cases** - If a fixture tests an edge case, document it clearly