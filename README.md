# Quality Hook

Quality Hook is a configurable CLI tool designed to run quality checks (formatting, linting, type checking, testing) on code changes and provide filtered, actionable feedback optimized for Claude Code and other LLMs.

## Overview

Quality Hook acts as a wrapper around your project's existing quality tools, intelligently filtering their output to provide only the most relevant error information. It supports any project type through configuration files and has special support for monorepos and file-aware execution.

## Key Features

- **Configuration-driven**: Support for any project type without hardcoding
- **Smart filtering**: Extracts only relevant error information from tool output
- **Monorepo support**: Path-based configuration for complex project structures
- **File-aware execution**: Runs only relevant checks based on changed files
- **LLM-optimized output**: Formatted for consumption by Claude Code and other AI assistants
- **Zero implementation**: Delegates all actual work to your existing project tools

## Installation

```bash
# Install from source
go install github.com/bebsworthy/qualhook/cmd/qualhook@latest

# Or download a pre-built binary from releases
```

## Quick Start

1. Initialize configuration for your project:
   ```bash
   qualhook config
   ```

2. Run quality checks:
   ```bash
   qualhook typecheck
   qualhook lint
   qualhook test
   ```

3. Use with Claude Code hooks:
   ```json
   // .claude/settings.json
   {
     "hooks": {
       "Stop": [
         {
           "matcher": "",
           "hooks": [
             {
               "type": "command",
               "command": "qualhook typecheck"
             },
             {
               "type": "command",
               "command": "qualhook lint"
             },
             {
               "type": "command",
               "command": "qualhook test"
             }
           ]
         }
       ]
     }
   }
   ```
   
   This example runs typecheck, lint, and test checks automatically when Claude Code stops editing files, ensuring code quality throughout your development session.

## Configuration

Quality Hook uses JSON configuration files to define how to run quality checks for your project. Configuration can be stored in:

- `.qualhook.json` in your project root
- `$HOME/.config/qualhook/config.json` for global settings
- Custom path via `QUALHOOK_CONFIG` environment variable

Example configuration:
```json
{
  "projectType": "node",
  "commands": {
    "format": {
      "command": "npm run format:check",
      "errorPatterns": ["Error:", "error TS\\d+:"]
    },
    "lint": {
      "command": "npm run lint",
      "errorPatterns": ["ERROR:", "Warning:"]
    }
  }
}
```

## Documentation

For detailed documentation, see the [documentation/features/quality-hook](documentation/features/quality-hook) directory.

## Development

```bash
# Clone the repository
git clone https://github.com/bebsworthy/qualhook.git
cd qualhook

# Install dependencies
go mod download

# Run tests
go test ./...

# Build
go build -o qualhook cmd/qualhook/main.go
```

### Test Categories

The test suite is organized into three categories using build tags:

- **Unit Tests** (`//go:build unit`): Test individual components in isolation
  - Fast execution
  - No external dependencies
  - Mock dependencies and test helpers
  - Run with: `make test-unit`

- **Integration Tests** (`//go:build integration`): Test multiple components working together
  - May use file system operations
  - Test component interactions
  - Security and validation tests
  - Run with: `make test-integration`

- **E2E Tests** (`//go:build e2e`): Test complete workflows from CLI perspective
  - Full command execution
  - Create temporary directories and config files
  - Test user-facing functionality
  - Run with: `make test-e2e`

Run specific test categories:
```bash
# Run only unit tests
make test-unit

# Run only integration tests
make test-integration

# Run only e2e tests
make test-e2e

# Run all tests (default behavior)
make test
```

## License

MIT