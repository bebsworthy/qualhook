# Quality Hook Examples

This directory contains example configurations for various project types and use cases.

## Directory Structure

```
examples/
├── simple/                     # Basic single-project configurations
│   ├── nodejs-basic.json      # Node.js with npm
│   ├── nodejs-with-yarn.json  # Node.js with Yarn
│   ├── go-project.json        # Go project
│   ├── python-project.json    # Python project
│   └── rust-project.json      # Rust project
├── monorepo/                   # Monorepo configurations
│   ├── mixed-stack.json       # Frontend/Backend/Services
│   └── lerna-monorepo.json    # Lerna-based monorepo
├── custom-commands/            # Beyond standard commands
│   ├── security-checks.json   # Security scanning
│   ├── documentation-checks.json # Documentation quality
│   └── database-migrations.json  # Database validation
└── CONFIGURATION_REFERENCE.md  # Complete configuration guide
```

## Quick Start

1. Find an example that matches your project type
2. Copy it to your project root as `.qualhook.json`
3. Customize the commands and patterns for your specific tools
4. Test with `qualhook lint --debug` to see full output

## Example Usage

### Simple Project

```bash
# Copy a basic configuration
cp examples/simple/nodejs-basic.json .qualhook.json

# Run commands
qualhook format
qualhook lint
qualhook typecheck
qualhook test
```

### Monorepo

```bash
# Copy monorepo configuration
cp examples/monorepo/mixed-stack.json .qualhook.json

# Commands run appropriate tools based on edited files
# Edit frontend/src/App.tsx
qualhook lint  # Runs frontend linter

# Edit backend/main.go
qualhook lint  # Runs Go linter
```

### Custom Commands

```bash
# Copy and extend with custom commands
cp examples/custom-commands/security-checks.json .qualhook.json

# Run custom commands
qualhook security      # npm audit
qualhook sast         # Semgrep scan
qualhook license-check # License validation
```

## Common Patterns

### Error Detection Patterns

**JavaScript/TypeScript:**
```json
{
  "pattern": "error", "flags": "i"
}
{
  "pattern": "^\\s*\\d+:\\d+", "flags": "m"
}
{
  "pattern": "error TS\\d+:", "flags": ""
}
```

**Go:**
```json
{
  "pattern": "^[^:]+:\\d+:\\d+:", "flags": "m"
}
{
  "pattern": "undefined:", "flags": ""
}
{
  "pattern": "cannot use", "flags": ""
}
```

**Python:**
```json
{
  "pattern": "[EWRCIF]\\d{4}:", "flags": ""
}
{
  "pattern": "^[^:]+:\\d+:", "flags": "m"
}
{
  "pattern": "AssertionError", "flags": ""
}
```

**Rust:**
```json
{
  "pattern": "error\\[E\\d+\\]:", "flags": ""
}
{
  "pattern": "^\\s+--> ", "flags": "m"
}
{
  "pattern": "thread .+ panicked", "flags": ""
}
```

## Customization Tips

1. **Adjust Timeouts**: Increase for slow tools or large codebases
2. **Fine-tune Patterns**: Start broad, then refine based on actual output
3. **Context Lines**: More context helps LLM understand errors better
4. **Output Limits**: Balance between completeness and LLM token limits
5. **Custom Prompts**: Guide the LLM with specific fix instructions

## Testing Your Configuration

```bash
# See what configuration is loaded
qualhook config --validate

# Test with debug output
qualhook lint --debug

# Force an error to test patterns
echo "bad code" > test.js
qualhook lint
```

## Contributing

Have a useful configuration? Please contribute:

1. Add your example to the appropriate directory
2. Include comments explaining non-obvious patterns
3. Test thoroughly with your project
4. Submit a pull request

## See Also

- [Configuration Reference](CONFIGURATION_REFERENCE.md) - Complete configuration documentation
- [Quality Hook README](../README.md) - Main project documentation