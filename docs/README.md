# Quality Hook Documentation

Welcome to the Quality Hook documentation! Quality Hook is a configurable CLI tool that wraps your project's quality commands and intelligently filters their output for optimal consumption by Claude Code and other LLMs.

## Documentation Overview

### Getting Started

- **[User Guide](user-guide.md)** - Complete guide to installing, configuring, and using Quality Hook
- **[Claude Code Integration](claude-code-integration.md)** - Step-by-step guide for integrating with Claude Code hooks

### Reference

- **[Configuration Schema](configuration-schema.md)** - Detailed reference for all configuration options
- **[Troubleshooting Guide](troubleshooting.md)** - Solutions to common problems and debugging techniques

### Quick Links

- **Installation**: See the [Installation section](user-guide.md#installation) in the User Guide
- **Quick Start**: Jump to [Getting Started](user-guide.md#getting-started)
- **Examples**: Check the [Examples directory](../examples/) for real-world configurations

## What is Quality Hook?

Quality Hook is a wrapper around your existing project tools (formatters, linters, type checkers, test runners) that:

1. **Runs your existing tools** - No reimplementation, just smart orchestration
2. **Filters output intelligently** - Shows only relevant errors, reducing noise
3. **Integrates with Claude Code** - Provides AI-friendly error messages
4. **Supports any project type** - Configuration-driven architecture
5. **Handles monorepos** - Path-based rules for complex projects

## Key Features

### Universal Project Support

Works with any language or framework through configuration:
- JavaScript/TypeScript (ESLint, Prettier, Jest, etc.)
- Go (gofmt, golangci-lint, go test)
- Python (Black, Flake8, mypy, pytest)
- Rust (rustfmt, clippy, cargo test)
- And many more...

### Smart Output Filtering

Reduces thousands of lines of output to just the relevant errors:
- Regex-based error detection
- Configurable context lines
- Priority-based filtering
- Size limits to prevent overwhelming

### Monorepo Support

Different configurations for different parts of your codebase:
- Path-based configuration rules
- File-aware execution
- Component-specific commands
- Precedence handling

### Claude Code Integration

Designed specifically for AI-assisted development:
- Exit code 2 for error signaling
- Filtered stderr output
- Actionable error messages
- File-aware hook support

## Quick Example

1. **Install Quality Hook**:
   ```bash
   go install github.com/bebsworthy/qualhook/cmd/qualhook@latest
   ```

2. **Configure your project**:
   ```bash
   qualhook config
   ```

3. **Add to Claude Code**:
   ```json
   {
     "hooks": {
       "post-edit": "qualhook"
     }
   }
   ```

4. **Let Claude fix errors automatically**!

## Navigation

- Next: [User Guide](user-guide.md) - Learn how to use Quality Hook
- Or: [Configuration Schema](configuration-schema.md) - Dive into configuration details
- Having issues? See the [Troubleshooting Guide](troubleshooting.md)