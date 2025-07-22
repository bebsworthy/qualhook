# Troubleshooting Guide

## Table of Contents

1. [Common Issues](#common-issues)
2. [Configuration Problems](#configuration-problems)
3. [Command Execution Issues](#command-execution-issues)
4. [Output Filtering Problems](#output-filtering-problems)
5. [Monorepo Issues](#monorepo-issues)
6. [Claude Code Integration Issues](#claude-code-integration-issues)
7. [Performance Problems](#performance-problems)
8. [Debug Mode](#debug-mode)
9. [Getting Help](#getting-help)

## Common Issues

### Quality Hook command not found

**Problem**: After installation, `qualhook` command is not recognized.

**Solutions**:

1. **Check PATH**: Ensure the installation directory is in your PATH
   ```bash
   echo $PATH
   which qualhook
   ```

2. **Go installation**: If installed via Go, ensure Go's bin directory is in PATH
   ```bash
   export PATH=$PATH:$(go env GOPATH)/bin
   ```

3. **Direct execution**: Use the full path to the binary
   ```bash
   /usr/local/bin/qualhook --version
   ```

### No configuration file found

**Problem**: Quality Hook reports "configuration file not found".

**Solutions**:

1. **Initialize configuration**:
   ```bash
   qualhook config
   ```

2. **Check file location**: Ensure `.qualhook.json` exists in the current directory
   ```bash
   ls -la .qualhook.json
   ```

3. **Use environment variable**:
   ```bash
   export QUALHOOK_CONFIG=/path/to/config.json
   qualhook lint
   ```

### Permission denied errors

**Problem**: Getting permission errors when running commands.

**Solutions**:

1. **Check file permissions**:
   ```bash
   chmod +x qualhook
   ```

2. **Check configuration file permissions**:
   ```bash
   chmod 644 .qualhook.json
   ```

3. **Run with appropriate permissions** (avoid if possible):
   ```bash
   sudo qualhook lint
   ```

## Configuration Problems

### Invalid JSON syntax

**Problem**: Configuration file has JSON syntax errors.

**Example error**:
```
[QUALHOOK ERROR] Configuration: Invalid JSON syntax
Details:
- unexpected end of JSON input
- Check for missing commas, brackets, or quotes
```

**Solutions**:

1. **Validate JSON**:
   ```bash
   cat .qualhook.json | jq .
   ```

2. **Common issues to check**:
   - Missing commas between properties
   - Unclosed brackets or braces
   - Invalid escape sequences in regex patterns
   - Trailing commas (not allowed in JSON)

3. **Example fix**:
   ```json
   // ❌ Bad - trailing comma
   {
     "version": "1.0",
     "commands": {
       "lint": { /* ... */ },  // Remove this comma
     }
   }
   
   // ✅ Good
   {
     "version": "1.0",
     "commands": {
       "lint": { /* ... */ }
     }
   }
   ```

### Schema validation errors

**Problem**: Configuration doesn't match expected schema.

**Example error**:
```
[QUALHOOK ERROR] Configuration: Schema validation failed
Details:
- commands.lint: missing required field 'command'
- version: must be "1.0"
```

**Solutions**:

1. **Check required fields**:
   ```json
   {
     "version": "1.0",  // Required
     "commands": {      // Required
       "lint": {
         "command": "eslint"  // Required for each command
       }
     }
   }
   ```

2. **Validate configuration**:
   ```bash
   qualhook config --validate
   ```

### Regex pattern errors

**Problem**: Invalid regular expression patterns in configuration.

**Example error**:
```
[QUALHOOK ERROR] Configuration: Invalid regex pattern
Details:
- Pattern: "[error"
- Error: missing closing bracket
```

**Solutions**:

1. **Escape special characters**:
   ```json
   {
     "pattern": "\\[error\\]",  // Escape brackets
     "flags": "i"
   }
   ```

2. **Test patterns**:
   ```bash
   # Use online regex testers or debug mode
   qualhook --debug lint
   ```

3. **Common escaping needs**:
   - Backslashes: `\\` → `\\\\`
   - Brackets: `[` → `\\[`
   - Dots: `.` → `\\.` (when matching literal dots)

## Command Execution Issues

### Command not found

**Problem**: External tool command cannot be found.

**Example error**:
```
[QUALHOOK ERROR] Execution: Command not found
Details:
- Command: eslint
- Make sure the command is installed and in PATH
```

**Solutions**:

1. **Verify installation**:
   ```bash
   which eslint
   npm list -g eslint
   ```

2. **Use full path**:
   ```json
   {
     "command": "/usr/local/bin/eslint",
     "args": ["."]
   }
   ```

3. **Use npm/yarn scripts**:
   ```json
   {
     "command": "npm",
     "args": ["run", "lint"]
   }
   ```

### Command timeout

**Problem**: Command exceeds timeout limit.

**Example error**:
```
[QUALHOOK ERROR] Execution: Command timed out
Details:
- Command: npm test
- Timeout: 120000ms (2 minutes)
- Process was killed
```

**Solutions**:

1. **Increase timeout**:
   ```json
   {
     "command": "npm",
     "args": ["test"],
     "timeout": 600000  // 10 minutes
   }
   ```

2. **Use environment variable**:
   ```bash
   QUALHOOK_TIMEOUT=600000 qualhook test
   ```

3. **Run tests in parallel** (if supported):
   ```json
   {
     "args": ["test", "--parallel"]
   }
   ```

### Working directory issues

**Problem**: Command runs in wrong directory.

**Solutions**:

1. **Specify working directory**:
   ```json
   {
     "command": "npm",
     "args": ["test"],
     "workingDir": "./frontend"
   }
   ```

2. **Use relative paths in args**:
   ```json
   {
     "command": "eslint",
     "args": ["./src"]
   }
   ```

## Output Filtering Problems

### No errors detected when there should be

**Problem**: Quality Hook returns exit code 0 despite errors in output.

**Solutions**:

1. **Check error detection config**:
   ```json
   {
     "errorDetection": {
       "exitCodes": [1, 2],  // Add all error exit codes
       "patterns": [
         { "pattern": "error", "flags": "i" },
         { "pattern": "failed", "flags": "i" }
       ]
     }
   }
   ```

2. **Debug pattern matching**:
   ```bash
   qualhook --debug lint 2>&1 | grep "Pattern match"
   ```

3. **Add more patterns**:
   ```json
   {
     "patterns": [
       { "pattern": "\\d+ errors?", "flags": "i" },
       { "pattern": "ERROR:", "flags": "" },
       { "pattern": "✗", "flags": "" }
     ]
   }
   ```

### Too much output / noise

**Problem**: Output includes too much irrelevant information.

**Solutions**:

1. **Reduce max output**:
   ```json
   {
     "outputFilter": {
       "maxOutput": 50  // Reduce from default 100
     }
   }
   ```

2. **Tighten error patterns**:
   ```json
   {
     "errorPatterns": [
       { "pattern": "^\\s*\\d+:\\d+\\s+error", "flags": "m" }
       // More specific than just "error"
     ]
   }
   ```

3. **Reduce context lines**:
   ```json
   {
     "contextLines": 0  // No context, just error lines
   }
   ```

### Missing important errors

**Problem**: Some errors are filtered out that should be shown.

**Solutions**:

1. **Add include patterns**:
   ```json
   {
     "outputFilter": {
       "errorPatterns": [/* ... */],
       "includePatterns": [
         { "pattern": "caused by:", "flags": "i" },
         { "pattern": "stack trace:", "flags": "i" }
       ]
     }
   }
   ```

2. **Increase context lines**:
   ```json
   {
     "contextLines": 5  // Show more surrounding context
   }
   ```

3. **Increase max output**:
   ```json
   {
     "maxOutput": 200  // Allow more lines
   }
   ```

## Monorepo Issues

### Wrong configuration being used

**Problem**: Commands use root config instead of path-specific config.

**Solutions**:

1. **Check path patterns**:
   ```json
   {
     "paths": [
       {
         "path": "frontend/**",  // Use ** for recursive matching
         "commands": { /* ... */ }
       }
     ]
   }
   ```

2. **Verify current directory**:
   ```bash
   pwd  # Ensure you're in the right directory
   qualhook --debug lint  # Check which config is used
   ```

3. **Use more specific patterns**:
   ```json
   {
     "paths": [
       {
         "path": "apps/web/src/**",  // More specific
         "commands": { /* ... */ }
       },
       {
         "path": "apps/web/**",  // Less specific fallback
         "commands": { /* ... */ }
       }
     ]
   }
   ```

### Commands run in wrong directory

**Problem**: Commands execute in root instead of component directory.

**Solutions**:

1. **Set working directory**:
   ```json
   {
     "paths": [
       {
         "path": "backend/**",
         "commands": {
           "test": {
             "command": "go",
             "args": ["test", "./..."],
             "workingDir": "backend"  // Explicit working dir
           }
         }
       }
     ]
   }
   ```

2. **Use prefix arguments**:
   ```json
   {
     "command": "npm",
     "args": ["run", "test", "--prefix", "frontend"]
   }
   ```

## Claude Code Integration Issues

### Hook not triggering

**Problem**: Quality Hook doesn't run when expected in Claude Code.

**Solutions**:

1. **Check hook configuration** in Claude Code settings:
   ```json
   {
     "hooks": {
       "post-edit": "qualhook"
     }
   }
   ```

2. **Verify hook executable**:
   - Ensure `qualhook` is in PATH
   - Use absolute path if needed

3. **Check hook permissions**:
   ```bash
   chmod +x $(which qualhook)
   ```

### File-aware execution not working

**Problem**: All checks run instead of just affected components.

**Solutions**:

1. **Enable debug mode** to see file mapping:
   ```bash
   QUALHOOK_DEBUG=1 qualhook
   ```

2. **Check path patterns** match your file structure:
   ```json
   {
     "paths": [
       {
         "path": "src/**/*.js",  // Ensure patterns match actual files
         "commands": { /* ... */ }
       }
     ]
   }
   ```

### Error output not appearing in Claude

**Problem**: Errors don't show up in Claude Code interface.

**Solutions**:

1. **Ensure exit code 2** for errors:
   ```json
   {
     "errorDetection": {
       "exitCodes": [1],  // Tool exits with 1
       // Quality Hook will convert to exit code 2
     }
   }
   ```

2. **Check stderr output**:
   ```bash
   qualhook lint 2>&1  # Verify errors go to stderr
   ```

3. **Add prompts** for clarity:
   ```json
   {
     "prompt": "Fix the following linting errors:"
   }
   ```

## Performance Problems

### Slow startup

**Problem**: Quality Hook takes too long to start.

**Solutions**:

1. **Check configuration size**: Large configs take longer to parse
   - Split monorepo configs if very large

2. **Disable project detection** if not needed:
   ```json
   {
     "projectType": "nodejs"  // Skip auto-detection
   }
   ```

3. **Use faster tools**: Some tools are inherently slow
   - Consider faster alternatives (e.g., `swc` instead of `tsc`)

### High memory usage

**Problem**: Quality Hook uses too much memory with large outputs.

**Solutions**:

1. **Reduce output limits**:
   ```json
   {
     "outputFilter": {
       "maxOutput": 50  // Limit output size
     }
   }
   ```

2. **Avoid capturing all output**:
   ```json
   {
     "outputFilter": {
       "priority": "errors"  // Only capture errors, not warnings
     }
   }
   ```

## Debug Mode

### Enabling debug mode

Debug mode provides detailed information about Quality Hook's operation:

```bash
# Via command line flag
qualhook --debug lint

# Via environment variable
QUALHOOK_DEBUG=1 qualhook lint

# Redirect to file for analysis
qualhook --debug lint > debug.log 2>&1
```

### Understanding debug output

Debug output includes:

1. **Configuration loading**:
   ```
   [DEBUG] Loading config from: .qualhook.json
   [DEBUG] Config validated successfully
   ```

2. **Path matching** (for monorepos):
   ```
   [DEBUG] Checking path: frontend/src/App.tsx
   [DEBUG] Matched pattern: frontend/**
   [DEBUG] Using config for: frontend
   ```

3. **Command execution**:
   ```
   [DEBUG] Executing: npm run lint
   [DEBUG] Working directory: /project/frontend
   [DEBUG] Timeout: 120000ms
   ```

4. **Pattern matching**:
   ```
   [DEBUG] Testing pattern: "error" (flags: i)
   [DEBUG] Pattern matched at line 15
   ```

5. **Output filtering**:
   ```
   [DEBUG] Total output lines: 523
   [DEBUG] Filtered to: 47 lines
   [DEBUG] Applied max output limit: 50
   ```

### Common debug scenarios

1. **Check which config is used**:
   ```bash
   qualhook --debug lint 2>&1 | grep "Loading config"
   ```

2. **Verify pattern matching**:
   ```bash
   qualhook --debug lint 2>&1 | grep "Pattern"
   ```

3. **Monitor command execution**:
   ```bash
   qualhook --debug test 2>&1 | grep "Executing"
   ```

## Getting Help

### Resources

1. **Documentation**:
   - [User Guide](user-guide.md)
   - [Configuration Schema](configuration-schema.md)
   - [Claude Code Integration](claude-code-integration.md)

2. **Examples**:
   - Check the `examples/` directory for real-world configurations
   - Look for configurations matching your tech stack

3. **Community**:
   - GitHub Issues: Report bugs or request features
   - Discussions: Ask questions and share configurations

### Reporting issues

When reporting issues, please include:

1. **Quality Hook version**:
   ```bash
   qualhook --version
   ```

2. **Configuration file** (sanitized if needed):
   ```bash
   cat .qualhook.json
   ```

3. **Debug output**:
   ```bash
   qualhook --debug [command] > debug.log 2>&1
   ```

4. **System information**:
   ```bash
   uname -a  # OS info
   echo $SHELL  # Shell being used
   ```

5. **Steps to reproduce**:
   - Exact commands run
   - Expected behavior
   - Actual behavior

### Quick fixes checklist

When encountering issues, try these steps in order:

1. ✓ Update to latest version
2. ✓ Validate configuration: `qualhook config --validate`
3. ✓ Run in debug mode: `qualhook --debug [command]`
4. ✓ Check tool installation: `which [tool-command]`
5. ✓ Verify file permissions
6. ✓ Test with simplified configuration
7. ✓ Check environment variables
8. ✓ Review error patterns with sample output

### Common error messages reference

| Error Message | Likely Cause | Quick Fix |
|--------------|--------------|-----------|
| "configuration file not found" | No .qualhook.json | Run `qualhook config` |
| "invalid JSON syntax" | Malformed JSON | Validate with `jq` |
| "command not found" | Tool not installed | Install tool or use full path |
| "permission denied" | File permissions | Check execute permissions |
| "timeout exceeded" | Long-running command | Increase timeout value |
| "no matches found" | Pattern too strict | Broaden error patterns |
| "output truncated" | Too much output | Reduce maxOutput value |