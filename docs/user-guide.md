# Quality Hook User Guide

## Table of Contents

1. [Introduction](#introduction)
2. [Installation](#installation)
3. [Getting Started](#getting-started)
4. [Basic Usage](#basic-usage)
5. [Monorepo Support](#monorepo-support)
6. [File-Aware Execution](#file-aware-execution)
7. [Custom Commands](#custom-commands)
8. [Advanced Usage](#advanced-usage)
9. [Best Practices](#best-practices)

## Introduction

Quality Hook is a configurable CLI tool that wraps your project's quality commands (format, lint, typecheck, test) and intelligently filters their output for optimal consumption by Claude Code and other LLMs. It supports any project type through configuration files without requiring any code changes.

### Key Benefits

- **Universal Support**: Works with any language or framework through configuration
- **Smart Filtering**: Shows only relevant errors, reducing noise for LLMs
- **Monorepo Ready**: Different configurations for different parts of your codebase
- **Zero Setup**: Auto-detects common project types and suggests configurations
- **LLM Optimized**: Output formatted specifically for AI understanding

## Installation

### From Source

```bash
# Requires Go 1.21 or later
go install github.com/boyd/qualhook/cmd/qualhook@latest
```

### Pre-built Binaries

Download the appropriate binary for your platform from the [releases page](https://github.com/boyd/qualhook/releases).

```bash
# macOS (Apple Silicon)
curl -L https://github.com/boyd/qualhook/releases/latest/download/qualhook-darwin-arm64 -o qualhook
chmod +x qualhook
sudo mv qualhook /usr/local/bin/

# macOS (Intel)
curl -L https://github.com/boyd/qualhook/releases/latest/download/qualhook-darwin-amd64 -o qualhook
chmod +x qualhook
sudo mv qualhook /usr/local/bin/

# Linux (x64)
curl -L https://github.com/boyd/qualhook/releases/latest/download/qualhook-linux-amd64 -o qualhook
chmod +x qualhook
sudo mv qualhook /usr/local/bin/

# Windows
# Download qualhook-windows-amd64.exe and add to your PATH
```

### Verify Installation

```bash
qualhook --version
```

## Getting Started

### 1. Initialize Configuration

Run the configuration wizard to set up Quality Hook for your project:

```bash
qualhook config
```

The wizard will:
- Detect your project type (Node.js, Go, Rust, Python, etc.)
- Suggest appropriate commands for format, lint, typecheck, and test
- Allow you to customize each command
- Create a `.qualhook.json` configuration file

### 2. Example Configuration Session

```
$ qualhook config

Quality Hook Configuration Wizard
=================================

Detecting project type...
✓ Found: Node.js project (package.json detected)

Suggested commands:
- format: npm run format
- lint: npm run lint
- typecheck: npm run typecheck
- test: npm test

Accept these defaults? [Y/n] n

Configure 'format' command:
Command to run [npm run format]: prettier --write .
Error detection pattern [non-zero exit code]: 
Output filter pattern [error]: 
✓ Configured format command

Configure 'lint' command:
Command to run [npm run lint]: eslint . --fix
Error detection pattern [non-zero exit code]: 
Output filter pattern [error|warning]: 
✓ Configured lint command

[... continues for typecheck and test ...]

Configuration saved to .qualhook.json
```

### 3. Basic Configuration Structure

After initialization, you'll have a `.qualhook.json` file:

```json
{
  "version": "1.0",
  "commands": {
    "format": {
      "command": "prettier",
      "args": ["--write", "."],
      "errorDetection": {
        "exitCodes": [1, 2]
      },
      "outputFilter": {
        "errorPatterns": [
          { "pattern": "\\[error\\]", "flags": "i" }
        ],
        "maxOutput": 50
      },
      "prompt": "Fix the formatting issues below:"
    },
    "lint": {
      "command": "eslint",
      "args": [".", "--fix"],
      "errorDetection": {
        "exitCodes": [1],
        "patterns": [
          { "pattern": "\\d+ errors?", "flags": "i" }
        ]
      },
      "outputFilter": {
        "errorPatterns": [
          { "pattern": "error", "flags": "i" },
          { "pattern": "^\\s*\\d+:\\d+", "flags": "m" }
        ],
        "contextLines": 2,
        "maxOutput": 100
      },
      "prompt": "Fix the linting errors below:"
    }
  }
}
```

## Basic Usage

### Running Individual Commands

```bash
# Format code
qualhook format

# Run linter
qualhook lint

# Run type checker
qualhook typecheck

# Run tests
qualhook test
```

### Running All Checks

```bash
# Run all configured commands
qualhook

# Equivalent to running:
# qualhook format && qualhook lint && qualhook typecheck && qualhook test
```

### Command Output

When no errors are found:
```
$ qualhook lint
✓ Lint check passed
```

When errors are detected:
```
$ qualhook lint
Fix the linting errors below:

src/utils/parser.js
  12:5  error  'unusedVar' is defined but never used  no-unused-vars
  24:1  error  Missing semicolon                       semi

✖ 2 problems (2 errors, 0 warnings)
```

### Exit Codes

- `0`: Success, no errors found
- `1`: Configuration or execution error
- `2`: Quality check failed (errors found)

## Monorepo Support

Quality Hook excels at handling monorepos with different tools for different parts of your codebase.

### Path-Based Configuration

Create a configuration that uses different commands for different directories:

```json
{
  "version": "1.0",
  "commands": {
    "lint": {
      "command": "echo",
      "args": ["No default linter configured"],
      "errorDetection": { "exitCodes": [1] }
    }
  },
  "paths": [
    {
      "path": "frontend/**",
      "commands": {
        "lint": {
          "command": "npm",
          "args": ["run", "lint", "--prefix", "frontend"],
          "errorDetection": { "exitCodes": [1] },
          "outputFilter": {
            "errorPatterns": [
              { "pattern": "error", "flags": "i" }
            ]
          }
        },
        "test": {
          "command": "npm",
          "args": ["test", "--prefix", "frontend"]
        }
      }
    },
    {
      "path": "backend/**",
      "commands": {
        "lint": {
          "command": "go",
          "args": ["vet", "./..."],
          "errorDetection": { "exitCodes": [1] }
        },
        "test": {
          "command": "go",
          "args": ["test", "./..."]
        }
      }
    },
    {
      "path": "docs/**",
      "commands": {
        "lint": {
          "command": "markdownlint",
          "args": ["**/*.md"],
          "errorDetection": { "exitCodes": [1] }
        }
      }
    }
  ]
}
```

### Path Matching Rules

1. **Specificity**: More specific paths take precedence
   - `frontend/src/components/**` overrides `frontend/**`
   - `frontend/**` overrides root configuration

2. **Glob Patterns**: Uses standard glob syntax
   - `*` matches any characters except `/`
   - `**` matches any characters including `/`
   - `?` matches single character
   - `[abc]` matches any character in brackets

3. **Working Directory**: Commands run in the matched directory by default

### Monorepo Example

For a typical full-stack monorepo:

```
myproject/
├── frontend/          # React app
│   ├── package.json
│   └── src/
├── backend/           # Go API
│   ├── go.mod
│   └── cmd/
├── shared/            # Shared TypeScript types
│   ├── package.json
│   └── types/
└── .qualhook.json
```

Configuration:

```json
{
  "version": "1.0",
  "paths": [
    {
      "path": "frontend/**",
      "commands": {
        "format": {
          "command": "npm",
          "args": ["run", "format", "--prefix", "frontend"]
        },
        "lint": {
          "command": "npm",
          "args": ["run", "lint", "--prefix", "frontend"]
        },
        "typecheck": {
          "command": "npm",
          "args": ["run", "typecheck", "--prefix", "frontend"]
        }
      }
    },
    {
      "path": "backend/**",
      "commands": {
        "format": {
          "command": "gofmt",
          "args": ["-w", "."]
        },
        "lint": {
          "command": "golangci-lint",
          "args": ["run"]
        },
        "test": {
          "command": "go",
          "args": ["test", "./..."]
        }
      }
    },
    {
      "path": "shared/**",
      "commands": {
        "typecheck": {
          "command": "npm",
          "args": ["run", "typecheck", "--prefix", "shared"]
        }
      }
    }
  ]
}
```

## File-Aware Execution

When integrated with Claude Code, Quality Hook can run checks only on the parts of your codebase that were actually modified.

### How It Works

1. Claude Code provides a list of modified files via hook input
2. Quality Hook maps files to their respective project components
3. Only runs checks for affected components
4. Reports which checks were run for which files

### Example Scenario

If Claude Code modifies:
- `frontend/src/App.tsx`
- `backend/api/users.go`

Quality Hook will:
1. Run frontend checks for the TypeScript file
2. Run backend checks for the Go file
3. Skip checks for unmodified components

### Configuration for File-Aware Mode

No special configuration needed! Quality Hook automatically detects when it receives file information from Claude Code hooks.

## Custom Commands

Beyond the standard commands (format, lint, typecheck, test), you can define custom commands for project-specific needs.

### Adding Custom Commands

```json
{
  "version": "1.0",
  "commands": {
    "security": {
      "command": "npm",
      "args": ["audit"],
      "errorDetection": {
        "patterns": [
          { "pattern": "found \\d+ vulnerabilities", "flags": "i" }
        ]
      },
      "outputFilter": {
        "errorPatterns": [
          { "pattern": "\\b(critical|high|moderate)\\b", "flags": "i" }
        ],
        "maxOutput": 100
      },
      "prompt": "Fix the security vulnerabilities below:"
    },
    "deps": {
      "command": "npm",
      "args": ["outdated"],
      "errorDetection": {
        "exitCodes": [1]
      },
      "prompt": "Update the outdated dependencies below:"
    }
  }
}
```

### Running Custom Commands

```bash
# Run security audit
qualhook security

# Check for outdated dependencies
qualhook deps

# Any command name defined in config
qualhook <custom-command>
```

## Advanced Usage

### Debug Mode

Get detailed execution information:

```bash
qualhook --debug lint
```

Debug output includes:
- Configuration loading details
- Command execution information
- Pattern matching results
- Output filtering steps

### Validation

Validate your configuration without running commands:

```bash
qualhook config --validate
```

### Environment Variables

```bash
# Override configuration file location
QUALHOOK_CONFIG=/path/to/config.json qualhook lint

# Enable debug mode
QUALHOOK_DEBUG=1 qualhook

# Set command timeout (milliseconds)
QUALHOOK_TIMEOUT=300000 qualhook test
```

### Timeout Configuration

Configure timeouts for long-running commands:

```json
{
  "commands": {
    "test": {
      "command": "npm",
      "args": ["test"],
      "timeout": 300000,  // 5 minutes
      "errorDetection": {
        "exitCodes": [1]
      }
    }
  }
}
```

### Output Filtering Examples

#### Basic Error Pattern

```json
{
  "outputFilter": {
    "errorPatterns": [
      { "pattern": "error", "flags": "i" }
    ]
  }
}
```

#### Line Number Pattern

```json
{
  "outputFilter": {
    "errorPatterns": [
      { "pattern": "^\\s*\\d+:\\d+", "flags": "m" }
    ],
    "contextLines": 2
  }
}
```

#### Complex Filtering

```json
{
  "outputFilter": {
    "errorPatterns": [
      { "pattern": "ERROR|FAIL|Error:", "flags": "" },
      { "pattern": "^\\s+at\\s+", "flags": "m" }
    ],
    "includePatterns": [
      { "pattern": "test.*failed", "flags": "i" }
    ],
    "maxOutput": 200,
    "contextLines": 3
  }
}
```

## Best Practices

### 1. Start Simple

Begin with basic configurations and add complexity as needed:

```json
{
  "version": "1.0",
  "commands": {
    "lint": {
      "command": "npm",
      "args": ["run", "lint"],
      "errorDetection": {
        "exitCodes": [1]
      }
    }
  }
}
```

### 2. Use Project Scripts

Leverage existing npm scripts or Makefiles:

```json
{
  "commands": {
    "lint": {
      "command": "npm",
      "args": ["run", "lint"]
    }
  }
}
```

### 3. Test Patterns Carefully

Use the debug mode to verify your patterns catch the right errors:

```bash
qualhook --debug lint > debug.log 2>&1
```

### 4. Optimize for LLMs

Keep error output concise and actionable:

```json
{
  "outputFilter": {
    "maxOutput": 100,
    "errorPatterns": [
      { "pattern": "^[^:]+:\\d+:\\d+:", "flags": "m" }
    ]
  },
  "prompt": "Fix these specific issues:"
}
```

### 5. Version Control

Always commit your `.qualhook.json` file:

```bash
git add .qualhook.json
git commit -m "Add Quality Hook configuration"
```

### 6. CI/CD Integration

Quality Hook works great in CI pipelines:

```yaml
# GitHub Actions example
- name: Run Quality Checks
  run: |
    qualhook format
    qualhook lint
    qualhook typecheck
    qualhook test
```

### 7. Custom Error Messages

Provide context-specific prompts for different commands:

```json
{
  "commands": {
    "format": {
      "prompt": "Apply these formatting changes:"
    },
    "lint": {
      "prompt": "Fix these code quality issues:"
    },
    "typecheck": {
      "prompt": "Resolve these type errors:"
    },
    "test": {
      "prompt": "Fix these failing tests:"
    }
  }
}
```

## Next Steps

- Read the [Configuration Schema](configuration-schema.md) for detailed configuration options
- Check the [Troubleshooting Guide](troubleshooting.md) for common issues
- Learn about [Claude Code Integration](claude-code-integration.md)
- See [Examples](../examples/) for real-world configurations