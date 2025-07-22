# Quality Hook Configuration Reference

This document provides a comprehensive reference for configuring Quality Hook.

## Table of Contents

- [Configuration File Format](#configuration-file-format)
- [Configuration Schema](#configuration-schema)
- [Command Configuration](#command-configuration)
- [Error Detection](#error-detection)
- [Output Filtering](#output-filtering)
- [Monorepo Support](#monorepo-support)
- [Custom Commands](#custom-commands)
- [Examples](#examples)

## Configuration File Format

Quality Hook uses JSON configuration files. The default location is `.qualhook.json` in your project root.

### Basic Structure

```json
{
  "version": "1.0",
  "projectType": "nodejs",
  "commands": {
    "format": { /* command config */ },
    "lint": { /* command config */ },
    "typecheck": { /* command config */ },
    "test": { /* command config */ }
  },
  "paths": [ /* monorepo path configs */ ]
}
```

## Configuration Schema

### Root Configuration

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `version` | string | Yes | Configuration schema version (currently "1.0") |
| `projectType` | string | No | Project type hint (e.g., "nodejs", "go", "rust", "python") |
| `commands` | object | Yes* | Map of command names to command configurations |
| `paths` | array | No | Path-specific configurations for monorepos |

*Either `commands` or `paths` must be specified.

## Command Configuration

Each command in the `commands` object has the following structure:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `command` | string | Yes | The executable to run |
| `args` | string[] | No | Arguments to pass to the command |
| `errorDetection` | object | Yes | How to detect errors |
| `outputFilter` | object | Yes | How to filter output |
| `prompt` | string | No | Custom prompt for LLM |
| `timeout` | number | No | Command timeout in milliseconds |

### Example Command Configuration

```json
{
  "lint": {
    "command": "npm",
    "args": ["run", "lint"],
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
      "contextLines": 3,
      "maxOutput": 150
    },
    "prompt": "Fix the linting errors below:",
    "timeout": 60000
  }
}
```

## Error Detection

The `errorDetection` object determines when a command has failed:

| Field | Type | Description |
|-------|------|-------------|
| `exitCodes` | number[] | Exit codes that indicate errors |
| `patterns` | RegexPattern[] | Patterns in output that indicate errors |

### RegexPattern Object

| Field | Type | Description |
|-------|------|-------------|
| `pattern` | string | Regular expression pattern |
| `flags` | string | Regex flags (i, m, s, U) |

### Common Regex Flags

- `i`: Case insensitive
- `m`: Multiline mode (^ and $ match line boundaries)
- `s`: Dot matches newline
- `U`: Ungreedy mode

## Output Filtering

The `outputFilter` object controls what output is shown to the LLM:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `errorPatterns` | RegexPattern[] | Yes | Patterns to identify error lines |
| `contextLines` | number | No | Lines of context around errors (default: 0) |
| `maxOutput` | number | No | Maximum lines to output (default: unlimited) |
| `includePatterns` | RegexPattern[] | No | Additional patterns to always include |

### Filtering Strategy

1. Lines matching `errorPatterns` are included
2. `contextLines` before/after each error are included
3. Lines matching `includePatterns` are always included
4. Output is truncated to `maxOutput` lines if exceeded

## Monorepo Support

Use the `paths` array for path-specific configurations:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `path` | string | Yes | Glob pattern for path matching |
| `extends` | string | No | Base path config to extend |
| `commands` | object | Yes | Command overrides for this path |

### Path Matching Rules

1. Paths are matched using glob patterns
2. More specific paths take precedence
3. Commands can be partially overridden
4. Use `extends` to inherit from another path

### Example Monorepo Configuration

```json
{
  "version": "1.0",
  "commands": {
    "lint": { /* root lint config */ }
  },
  "paths": [
    {
      "path": "frontend/**",
      "commands": {
        "lint": {
          "command": "npm",
          "args": ["run", "lint", "--prefix", "frontend"],
          "errorDetection": { /* ... */ },
          "outputFilter": { /* ... */ }
        }
      }
    },
    {
      "path": "backend/**",
      "commands": {
        "lint": {
          "command": "go",
          "args": ["vet", "./..."],
          "errorDetection": { /* ... */ },
          "outputFilter": { /* ... */ }
        }
      }
    }
  ]
}
```

## Custom Commands

You can define any custom command beyond the standard format/lint/typecheck/test:

```json
{
  "commands": {
    "security": {
      "command": "npm",
      "args": ["audit", "--audit-level=moderate"],
      "errorDetection": {
        "exitCodes": [1],
        "patterns": [
          { "pattern": "found \\d+ vulnerabilities", "flags": "" }
        ]
      },
      "outputFilter": {
        "errorPatterns": [
          { "pattern": "Severity:", "flags": "" },
          { "pattern": "Package:", "flags": "" }
        ],
        "contextLines": 1,
        "maxOutput": 200
      },
      "prompt": "Fix the security vulnerabilities below:"
    }
  }
}
```

Then run with: `qualhook security`

## Examples

### Minimal Configuration

```json
{
  "version": "1.0",
  "commands": {
    "format": {
      "command": "prettier",
      "args": ["--check", "."],
      "errorDetection": {
        "exitCodes": [1]
      },
      "outputFilter": {
        "errorPatterns": [
          { "pattern": "\\[warn\\]", "flags": "" }
        ],
        "maxOutput": 50
      }
    }
  }
}
```

### TypeScript Project with Multiple Tools

```json
{
  "version": "1.0",
  "projectType": "nodejs",
  "commands": {
    "format": {
      "command": "prettier",
      "args": ["--check", "src/**/*.{ts,tsx}"],
      "errorDetection": {
        "exitCodes": [1]
      },
      "outputFilter": {
        "errorPatterns": [
          { "pattern": "\\[warn\\]", "flags": "" }
        ],
        "contextLines": 0,
        "maxOutput": 100
      },
      "prompt": "Format these files with Prettier:"
    },
    "lint": {
      "command": "eslint",
      "args": ["src", "--ext", ".ts,.tsx"],
      "errorDetection": {
        "exitCodes": [1],
        "patterns": [
          { "pattern": "\\d+ problems?", "flags": "" }
        ]
      },
      "outputFilter": {
        "errorPatterns": [
          { "pattern": "^\\s*\\d+:\\d+", "flags": "m" },
          { "pattern": "error", "flags": "i" }
        ],
        "contextLines": 2,
        "maxOutput": 200
      },
      "prompt": "Fix these ESLint errors:"
    }
  }
}
```

### Go Monorepo with Services

```json
{
  "version": "1.0",
  "projectType": "go",
  "commands": {
    "format": {
      "command": "gofmt",
      "args": ["-l", "."],
      "errorDetection": {
        "exitCodes": [1]
      },
      "outputFilter": {
        "errorPatterns": [
          { "pattern": "\\.go$", "flags": "" }
        ],
        "maxOutput": 50
      }
    }
  },
  "paths": [
    {
      "path": "services/api/**",
      "commands": {
        "test": {
          "command": "go",
          "args": ["test", "./...", "-v", "-race"],
          "errorDetection": {
            "exitCodes": [1],
            "patterns": [
              { "pattern": "FAIL", "flags": "" }
            ]
          },
          "outputFilter": {
            "errorPatterns": [
              { "pattern": "--- FAIL:", "flags": "" },
              { "pattern": "DATA RACE", "flags": "" }
            ],
            "contextLines": 10,
            "maxOutput": 300
          }
        }
      }
    },
    {
      "path": "tools/**",
      "commands": {
        "lint": {
          "command": "staticcheck",
          "args": ["./..."],
          "errorDetection": {
            "exitCodes": [1]
          },
          "outputFilter": {
            "errorPatterns": [
              { "pattern": "^[^:]+:\\d+:\\d+:", "flags": "m" }
            ],
            "contextLines": 2,
            "maxOutput": 150
          }
        }
      }
    }
  ]
}
```

### Python Project with Custom Commands

```json
{
  "version": "1.0",
  "projectType": "python",
  "commands": {
    "format": {
      "command": "black",
      "args": [".", "--check"],
      "errorDetection": {
        "exitCodes": [1]
      },
      "outputFilter": {
        "errorPatterns": [
          { "pattern": "would reformat", "flags": "" }
        ],
        "maxOutput": 100
      }
    },
    "docstring": {
      "command": "pydocstyle",
      "args": ["--count"],
      "errorDetection": {
        "exitCodes": [1],
        "patterns": [
          { "pattern": "D\\d{3}:", "flags": "" }
        ]
      },
      "outputFilter": {
        "errorPatterns": [
          { "pattern": "^[^:]+:\\d+", "flags": "m" },
          { "pattern": "D\\d{3}:", "flags": "" }
        ],
        "contextLines": 1,
        "maxOutput": 150
      },
      "prompt": "Fix the docstring issues:"
    },
    "complexity": {
      "command": "radon",
      "args": ["cc", ".", "-s", "-nc"],
      "errorDetection": {
        "exitCodes": [1],
        "patterns": [
          { "pattern": "\\(\\d+\\)", "flags": "" }
        ]
      },
      "outputFilter": {
        "errorPatterns": [
          { "pattern": "C \\(\\d+\\)", "flags": "" },
          { "pattern": "D \\(\\d+\\)", "flags": "" },
          { "pattern": "E \\(\\d+\\)", "flags": "" },
          { "pattern": "F \\(\\d+\\)", "flags": "" }
        ],
        "contextLines": 0,
        "maxOutput": 100
      },
      "prompt": "Reduce complexity in these functions:"
    }
  }
}
```

## Tips and Best Practices

1. **Start Simple**: Begin with basic error detection and refine patterns based on actual output
2. **Test Patterns**: Use regex testers to validate your patterns
3. **Context Matters**: Add enough context lines for the LLM to understand errors
4. **Limit Output**: Set reasonable `maxOutput` to avoid overwhelming the LLM
5. **Custom Prompts**: Use prompts to guide the LLM's response
6. **Timeout Values**: Set appropriate timeouts for long-running commands
7. **Monorepo Paths**: Use specific paths to avoid running unnecessary checks
8. **Exit Codes**: Most tools use exit code 1 for errors, but some vary
9. **Pattern Flags**: Use multiline flag (`m`) for line-based patterns
10. **Incremental Migration**: You can start with one command and add more later

## Troubleshooting

### Common Issues

1. **No errors detected**: Check exit codes and patterns match actual output
2. **Too much output**: Reduce `maxOutput` or refine `errorPatterns`
3. **Missing errors**: Add more patterns or check `contextLines`
4. **Timeout errors**: Increase timeout value for slow commands
5. **Monorepo confusion**: Check path patterns and precedence

### Debug Mode

Run with `--debug` flag to see:
- Which configuration is loaded
- Full command output before filtering
- Pattern matching details
- Execution timing

```bash
qualhook lint --debug
```