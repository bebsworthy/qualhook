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
go install github.com/boyd/qualhook/cmd/qualhook@latest

# Or download a pre-built binary from releases
```

## Quick Start

1. Initialize configuration for your project:
   ```bash
   qualhook config
   ```

2. Run quality checks:
   ```bash
   # Run all checks
   qualhook
   
   # Run specific check
   qualhook lint
   qualhook test
   ```

3. Use with Claude Code hooks:
   ```bash
   # Add to your Claude Code settings
   "hooks": {
     "post-edit": "qualhook"
   }
   ```

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
git clone https://github.com/boyd/qualhook.git
cd qualhook

# Install dependencies
go mod download

# Run tests
go test ./...

# Build
go build -o qualhook cmd/qualhook/main.go
```

## License

MIT