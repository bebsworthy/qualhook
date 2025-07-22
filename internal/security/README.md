# Security Package

The security package provides comprehensive security validation and protection mechanisms for qualhook to prevent command injection, path traversal, ReDoS attacks, and other security vulnerabilities.

## Features

### 1. Command Validation
- **Command whitelisting**: Restrict execution to approved commands only
- **Shell injection prevention**: Detect and block shell metacharacters
- **Dangerous command detection**: Special handling for potentially dangerous commands like `rm`, `curl`, etc.

### 2. Path Validation  
- **Directory traversal prevention**: Block `..` sequences and path manipulation attempts
- **Banned path protection**: Prevent access to system directories
- **Project boundary enforcement**: Ensure all file operations stay within project directory

### 3. Regex Pattern Validation
- **ReDoS prevention**: Detect patterns that could cause catastrophic backtracking
- **Pattern complexity limits**: Enforce maximum pattern length and complexity
- **Safe compilation**: Timeout protection during pattern compilation

### 4. Environment Sanitization
- **Sensitive variable filtering**: Remove passwords, tokens, and secrets
- **Injection prevention**: Block command substitution in environment values
- **Minimal environment mode**: Start with only essential variables

### 5. Resource Limits
- **Output size limiting**: Prevent excessive memory usage from command output
- **Timeout enforcement**: Prevent runaway processes
- **Rate limiting**: Control command execution frequency

## Usage

### Basic Security Validation

```go
import "github.com/qualhook/qualhook/internal/security"

// Create a security validator
validator := security.NewSecurityValidator()

// Validate a command
err := validator.ValidateCommand("npm", []string{"test"})
if err != nil {
    // Command is not safe to execute
}

// Validate a file path
err = validator.ValidatePath("src/main.go")
if err != nil {
    // Path is not safe to access
}

// Validate a regex pattern
err = validator.ValidateRegexPattern("error:\\s+(.*)")
if err != nil {
    // Pattern could cause performance issues
}
```

### Environment Sanitization

```go
// Sanitize environment for subprocess
env := security.SanitizeEnvironment(os.Environ(), true)

// Create minimal environment
minimalEnv := security.SanitizeEnvironment(nil, false)

// Merge custom variables safely
merged, err := security.MergeEnvironment(baseEnv, customVars)
```

### Resource Limiting

```go
// Create a limited writer to cap output size
var buf bytes.Buffer
limited := security.NewLimitedWriter(&buf, 1024*1024) // 1MB limit

// Use with command execution
cmd.Stdout = limited
```

### Security Configuration

```go
// Load security configuration
config, err := security.LoadConfig("security.json")

// Apply to validator
err = config.ApplyToValidator(validator)

// Use strict mode for production
strictConfig := security.StrictConfig()
```

## Security Configuration File

Create a `security.json` file to customize security settings:

```json
{
  "allowedCommands": [
    "npm", "yarn", "go", "python", "git"
  ],
  "maxTimeout": "5m",
  "maxRegexLength": 200,
  "maxOutputSize": 5242880,
  "bannedPaths": [
    "/etc", "/sys", "/proc", 
    "C:\\Windows", "C:\\System32"
  ],
  "enableStrictMode": true
}
```

## Best Practices

1. **Use whitelisting in production**: Explicitly list allowed commands rather than blacklisting
2. **Validate all inputs**: Always validate commands, paths, and patterns before use
3. **Use minimal environments**: Don't inherit full environment unless necessary
4. **Set appropriate limits**: Configure timeouts and output limits based on your needs
5. **Monitor for violations**: Log security validation failures for analysis

## Security Considerations

- The validator operates on a "fail-closed" principle - when in doubt, it rejects
- Path validation is strict and prevents access outside the project directory
- Environment sanitization removes variables that could leak secrets
- ReDoS prevention may reject some legitimate but complex regex patterns

## Testing

The package includes comprehensive tests for all security features:

```bash
go test ./internal/security/...
```

## Contributing

When adding new security features:
1. Follow the existing validation patterns
2. Add comprehensive tests including attack scenarios
3. Document any new configuration options
4. Consider backward compatibility