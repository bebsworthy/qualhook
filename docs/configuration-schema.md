# Configuration Schema Reference

## Table of Contents

1. [Overview](#overview)
2. [Root Configuration](#root-configuration)
3. [Command Configuration](#command-configuration)
4. [Error Detection](#error-detection)
5. [Output Filtering](#output-filtering)
6. [Path Configuration](#path-configuration)
7. [Regular Expression Patterns](#regular-expression-patterns)
8. [Complete Schema](#complete-schema)
9. [Examples](#examples)

## Overview

Quality Hook uses JSON configuration files to define how to run and interpret quality commands. The configuration is designed to be flexible and support any project type without requiring code changes.

### Configuration File Location

Quality Hook looks for configuration in the following order:

1. `.qualhook.json` in the current directory
2. `qualhook.json` in the current directory
3. `.qualhook/config.json` in the current directory
4. Path specified by `QUALHOOK_CONFIG` environment variable

## Root Configuration

The root configuration object contains the following properties:

| Property | Type | Required | Description |
|----------|------|----------|-------------|
| `version` | string | Yes | Schema version (currently "1.0") |
| `projectType` | string | No | Optional project type hint (e.g., "nodejs", "go", "python") |
| `commands` | object | Yes | Map of command names to command configurations |
| `paths` | array | No | Path-specific configurations for monorepo support |

### Example Root Configuration

```json
{
  "version": "1.0",
  "projectType": "nodejs",
  "commands": {
    "format": { /* ... */ },
    "lint": { /* ... */ }
  },
  "paths": [ /* ... */ ]
}
```

## Command Configuration

Each command in the `commands` object has the following structure:

| Property | Type | Required | Description |
|----------|------|----------|-------------|
| `command` | string | Yes | The executable command to run |
| `args` | array | No | Array of command arguments |
| `errorDetection` | object | No | How to detect errors in command output |
| `outputFilter` | object | No | How to filter command output |
| `prompt` | string | No | LLM prompt template for this command |
| `timeout` | number | No | Command timeout in milliseconds (default: 120000) |
| `workingDir` | string | No | Working directory for command execution |
| `env` | object | No | Additional environment variables |

### Command Examples

#### Simple Command

```json
{
  "command": "npm",
  "args": ["run", "lint"]
}
```

#### Complex Command

```json
{
  "command": "eslint",
  "args": [".", "--ext", ".js,.jsx,.ts,.tsx", "--fix"],
  "timeout": 300000,
  "workingDir": "./src",
  "env": {
    "NODE_ENV": "development"
  }
}
```

## Error Detection

The `errorDetection` object determines when a command has found errors:

| Property | Type | Required | Description |
|----------|------|----------|-------------|
| `exitCodes` | array | No | Exit codes that indicate errors (default: [1]) |
| `patterns` | array | No | Regex patterns in output that indicate errors |

### Error Detection Logic

Quality Hook considers a command to have errors if:
1. The exit code matches one in `exitCodes` array, OR
2. The output matches any pattern in `patterns` array

### Examples

#### Exit Code Based

```json
{
  "errorDetection": {
    "exitCodes": [1, 2, 3]
  }
}
```

#### Pattern Based

```json
{
  "errorDetection": {
    "patterns": [
      { "pattern": "\\d+ errors?", "flags": "i" },
      { "pattern": "FAILED", "flags": "" }
    ]
  }
}
```

#### Combined

```json
{
  "errorDetection": {
    "exitCodes": [1],
    "patterns": [
      { "pattern": "error:", "flags": "i" }
    ]
  }
}
```

## Output Filtering

The `outputFilter` object controls how command output is processed:

| Property | Type | Required | Description |
|----------|------|----------|-------------|
| `errorPatterns` | array | Yes | Regex patterns to identify error lines |
| `contextLines` | number | No | Number of context lines around errors (default: 0) |
| `maxOutput` | number | No | Maximum number of output lines (default: 100) |
| `includePatterns` | array | No | Additional patterns to always include |
| `priority` | string | No | Filter priority: "errors", "warnings", "all" (default: "errors") |

### Filtering Process

1. Identify lines matching `errorPatterns`
2. Include `contextLines` before and after each match
3. Add lines matching `includePatterns`
4. Truncate to `maxOutput` lines if needed
5. Apply `priority` filtering if output is still too large

### Examples

#### Basic Filter

```json
{
  "outputFilter": {
    "errorPatterns": [
      { "pattern": "error", "flags": "i" }
    ],
    "maxOutput": 50
  }
}
```

#### With Context

```json
{
  "outputFilter": {
    "errorPatterns": [
      { "pattern": "^\\s*\\d+:\\d+", "flags": "m" }
    ],
    "contextLines": 2,
    "maxOutput": 100
  }
}
```

#### Advanced Filter

```json
{
  "outputFilter": {
    "errorPatterns": [
      { "pattern": "ERROR|FAIL", "flags": "" },
      { "pattern": "^\\s+at\\s+", "flags": "m" }
    ],
    "includePatterns": [
      { "pattern": "caused by:", "flags": "i" }
    ],
    "contextLines": 3,
    "maxOutput": 200,
    "priority": "errors"
  }
}
```

## Path Configuration

For monorepo support, the `paths` array contains path-specific configurations:

| Property | Type | Required | Description |
|----------|------|----------|-------------|
| `path` | string | Yes | Glob pattern for path matching |
| `extends` | string | No | Base configuration to extend |
| `commands` | object | Yes | Command overrides for this path |

### Path Matching Rules

1. Patterns are matched against file paths relative to the config file
2. More specific patterns take precedence
3. First matching pattern wins
4. Uses standard glob syntax (`*`, `**`, `?`, `[...]`)

### Examples

#### Basic Path Config

```json
{
  "paths": [
    {
      "path": "frontend/**",
      "commands": {
        "lint": {
          "command": "npm",
          "args": ["run", "lint", "--prefix", "frontend"]
        }
      }
    }
  ]
}
```

#### With Inheritance

```json
{
  "paths": [
    {
      "path": "packages/*/",
      "extends": "base",
      "commands": {
        "test": {
          "args": ["test", "--coverage"]
        }
      }
    }
  ]
}
```

## Regular Expression Patterns

Regex patterns are used throughout the configuration:

| Property | Type | Required | Description |
|----------|------|----------|-------------|
| `pattern` | string | Yes | Regular expression pattern |
| `flags` | string | No | Regex flags (e.g., "i" for case-insensitive) |

### Supported Flags

- `i`: Case-insensitive matching
- `m`: Multiline mode (^ and $ match line boundaries)
- `s`: Dot matches newlines
- `U`: Ungreedy quantifiers

### Pattern Examples

#### Case-Insensitive Error

```json
{
  "pattern": "error|warning|failed",
  "flags": "i"
}
```

#### Line Numbers

```json
{
  "pattern": "^\\s*\\d+:\\d+\\s+error",
  "flags": "m"
}
```

#### Stack Traces

```json
{
  "pattern": "^\\s+at\\s+\\S+\\s*\\([^)]+\\)",
  "flags": "m"
}
```

## Complete Schema

Here's the complete JSON Schema for Quality Hook configuration:

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "required": ["version", "commands"],
  "properties": {
    "version": {
      "type": "string",
      "enum": ["1.0"]
    },
    "projectType": {
      "type": "string"
    },
    "commands": {
      "type": "object",
      "additionalProperties": {
        "$ref": "#/definitions/commandConfig"
      }
    },
    "paths": {
      "type": "array",
      "items": {
        "$ref": "#/definitions/pathConfig"
      }
    }
  },
  "definitions": {
    "commandConfig": {
      "type": "object",
      "required": ["command"],
      "properties": {
        "command": {
          "type": "string"
        },
        "args": {
          "type": "array",
          "items": {
            "type": "string"
          }
        },
        "errorDetection": {
          "$ref": "#/definitions/errorDetection"
        },
        "outputFilter": {
          "$ref": "#/definitions/outputFilter"
        },
        "prompt": {
          "type": "string"
        },
        "timeout": {
          "type": "number",
          "minimum": 0
        },
        "workingDir": {
          "type": "string"
        },
        "env": {
          "type": "object",
          "additionalProperties": {
            "type": "string"
          }
        }
      }
    },
    "errorDetection": {
      "type": "object",
      "properties": {
        "exitCodes": {
          "type": "array",
          "items": {
            "type": "number"
          }
        },
        "patterns": {
          "type": "array",
          "items": {
            "$ref": "#/definitions/regexPattern"
          }
        }
      }
    },
    "outputFilter": {
      "type": "object",
      "required": ["errorPatterns"],
      "properties": {
        "errorPatterns": {
          "type": "array",
          "items": {
            "$ref": "#/definitions/regexPattern"
          }
        },
        "contextLines": {
          "type": "number",
          "minimum": 0
        },
        "maxOutput": {
          "type": "number",
          "minimum": 1
        },
        "includePatterns": {
          "type": "array",
          "items": {
            "$ref": "#/definitions/regexPattern"
          }
        },
        "priority": {
          "type": "string",
          "enum": ["errors", "warnings", "all"]
        }
      }
    },
    "regexPattern": {
      "type": "object",
      "required": ["pattern"],
      "properties": {
        "pattern": {
          "type": "string"
        },
        "flags": {
          "type": "string"
        }
      }
    },
    "pathConfig": {
      "type": "object",
      "required": ["path", "commands"],
      "properties": {
        "path": {
          "type": "string"
        },
        "extends": {
          "type": "string"
        },
        "commands": {
          "type": "object",
          "additionalProperties": {
            "$ref": "#/definitions/commandConfig"
          }
        }
      }
    }
  }
}
```

## Examples

### Node.js Project

```json
{
  "version": "1.0",
  "projectType": "nodejs",
  "commands": {
    "format": {
      "command": "prettier",
      "args": ["--write", "."],
      "errorDetection": {
        "exitCodes": [2]
      },
      "outputFilter": {
        "errorPatterns": [
          { "pattern": "\\[error\\]", "flags": "i" }
        ],
        "maxOutput": 50
      },
      "prompt": "Fix the formatting issues:"
    },
    "lint": {
      "command": "eslint",
      "args": [".", "--fix"],
      "errorDetection": {
        "exitCodes": [1],
        "patterns": [
          { "pattern": "\\d+ problems?", "flags": "i" }
        ]
      },
      "outputFilter": {
        "errorPatterns": [
          { "pattern": "^\\s*\\d+:\\d+", "flags": "m" }
        ],
        "contextLines": 1,
        "maxOutput": 100
      },
      "prompt": "Fix the linting errors:"
    },
    "typecheck": {
      "command": "tsc",
      "args": ["--noEmit"],
      "errorDetection": {
        "exitCodes": [1, 2]
      },
      "outputFilter": {
        "errorPatterns": [
          { "pattern": "error TS\\d+:", "flags": "" }
        ],
        "contextLines": 2,
        "maxOutput": 100
      },
      "prompt": "Fix the TypeScript errors:"
    },
    "test": {
      "command": "jest",
      "args": ["--passWithNoTests"],
      "timeout": 300000,
      "errorDetection": {
        "exitCodes": [1],
        "patterns": [
          { "pattern": "FAIL", "flags": "" }
        ]
      },
      "outputFilter": {
        "errorPatterns": [
          { "pattern": "FAIL", "flags": "" },
          { "pattern": "✕", "flags": "" }
        ],
        "includePatterns": [
          { "pattern": "Expected:", "flags": "" },
          { "pattern": "Received:", "flags": "" }
        ],
        "contextLines": 5,
        "maxOutput": 200
      },
      "prompt": "Fix the failing tests:"
    }
  }
}
```

### Go Project

```json
{
  "version": "1.0",
  "projectType": "go",
  "commands": {
    "format": {
      "command": "gofmt",
      "args": ["-w", "."],
      "errorDetection": {
        "exitCodes": []
      }
    },
    "lint": {
      "command": "golangci-lint",
      "args": ["run", "--fix"],
      "errorDetection": {
        "exitCodes": [1]
      },
      "outputFilter": {
        "errorPatterns": [
          { "pattern": "^[^:]+:\\d+:\\d+:", "flags": "m" }
        ],
        "maxOutput": 100
      },
      "prompt": "Fix the Go linting issues:"
    },
    "typecheck": {
      "command": "go",
      "args": ["build", "./..."],
      "errorDetection": {
        "exitCodes": [1]
      },
      "outputFilter": {
        "errorPatterns": [
          { "pattern": "^[^:]+:\\d+:\\d+:", "flags": "m" }
        ],
        "contextLines": 1,
        "maxOutput": 100
      }
    },
    "test": {
      "command": "go",
      "args": ["test", "./...", "-v"],
      "timeout": 600000,
      "errorDetection": {
        "exitCodes": [1],
        "patterns": [
          { "pattern": "FAIL", "flags": "" }
        ]
      },
      "outputFilter": {
        "errorPatterns": [
          { "pattern": "FAIL", "flags": "" },
          { "pattern": "--- FAIL:", "flags": "" }
        ],
        "contextLines": 10,
        "maxOutput": 200
      },
      "prompt": "Fix the failing Go tests:"
    }
  }
}
```

### Monorepo with Multiple Languages

```json
{
  "version": "1.0",
  "commands": {
    "format": {
      "command": "echo",
      "args": ["No formatter configured for this path"],
      "errorDetection": {
        "exitCodes": []
      }
    }
  },
  "paths": [
    {
      "path": "apps/web/**",
      "commands": {
        "format": {
          "command": "prettier",
          "args": ["--write", "apps/web"]
        },
        "lint": {
          "command": "npm",
          "args": ["run", "lint", "--workspace=web"]
        },
        "typecheck": {
          "command": "npm",
          "args": ["run", "typecheck", "--workspace=web"]
        }
      }
    },
    {
      "path": "apps/api/**",
      "commands": {
        "format": {
          "command": "gofmt",
          "args": ["-w", "apps/api"]
        },
        "lint": {
          "command": "golangci-lint",
          "args": ["run"],
          "workingDir": "apps/api"
        },
        "test": {
          "command": "go",
          "args": ["test", "./..."],
          "workingDir": "apps/api"
        }
      }
    },
    {
      "path": "packages/shared/**",
      "commands": {
        "typecheck": {
          "command": "tsc",
          "args": ["--project", "packages/shared/tsconfig.json", "--noEmit"]
        }
      }
    }
  ]
}
```