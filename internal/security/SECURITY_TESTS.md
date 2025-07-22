# Security Tests Documentation

This document describes the comprehensive security tests implemented for the qualhook project.

## Overview

The security tests ensure that qualhook is protected against various attack vectors including:
- Command injection
- Path traversal
- Environment variable manipulation
- ReDoS (Regular Expression Denial of Service)
- Resource exhaustion
- Configuration tampering

## Test Files Created

### 1. `/internal/executor/security_test.go`
Tests for command execution security:
- **TestExecute_CommandInjectionPrevention**: Tests protection against shell injection in commands and arguments
- **TestExecute_PathTraversalPrevention**: Tests protection against directory traversal attacks
- **TestExecute_EnvironmentVariableFiltering**: Tests that sensitive environment variables are filtered
- **TestExecute_DangerousCommands**: Tests prevention of dangerous command execution
- **TestExecute_CommandWhitelist**: Tests command whitelisting functionality
- **TestExecute_TimeoutValidation**: Tests timeout value validation
- **TestExecute_EnvironmentInjection**: Tests protection against environment variable injection
- **TestExecute_ConcurrentSecurity**: Tests security measures under concurrent load

### 2. `/internal/config/security_test.go`
Tests for configuration validation security:
- **TestValidateCommand_SecurityChecks**: Tests security validation for commands
- **TestValidate_MaliciousRegexPatterns**: Tests validation of potentially malicious regex patterns
- **TestValidate_PathTraversalInPatterns**: Tests path traversal prevention in path patterns
- **TestValidate_TimeoutLimits**: Tests validation of timeout values
- **TestValidate_CommandWhitelist**: Tests command whitelisting in configuration
- **TestValidate_ComplexMaliciousConfig**: Tests complex configurations with multiple security issues
- **TestValidate_ResourceExhaustion**: Tests protection against resource exhaustion attacks
- **TestSuggestFixes_SecurityErrors**: Tests fix suggestions for security errors

### 3. `/internal/filter/patterns_security_test.go`
Tests for regex pattern security:
- **TestPatternCache_ReDoSPrevention**: Tests that ReDoS vulnerable patterns are rejected
- **TestPatternCache_PatternTimeout**: Tests pattern compilation timeouts
- **TestPatternCache_MaliciousPatterns**: Tests various malicious regex patterns
- **TestPatternCache_SafePatterns**: Tests that legitimate patterns are not blocked
- **TestPatternCache_ConcurrentAccess**: Tests thread safety of pattern compilation
- **TestPatternValidation_SecurityPatterns**: Tests validation of security-relevant patterns

### 4. `/internal/security/environment_security_test.go`
Tests for environment variable security:
- **TestSanitizeEnvironment_ComprehensiveSecurity**: Tests comprehensive environment sanitization
- **TestMergeEnvironment_Security**: Tests secure merging of environment variables
- **TestValidatePathEnv_Security**: Tests PATH environment variable validation
- **TestCreateMinimalEnvironment**: Tests creation of minimal safe environment
- **TestEnvironmentSanitization_RealWorldScenarios**: Tests against real-world attack scenarios

### 5. `/internal/security/integration_test.go`
Integration tests for end-to-end security:
- **TestSecurityIntegration_CommandExecution**: Tests end-to-end security for command execution
- **TestSecurityIntegration_ConfigurationValidation**: Tests comprehensive config validation
- **TestSecurityIntegration_DefenseInDepth**: Tests multiple layers of security
- **TestSecurityIntegration_RealWorldScenarios**: Tests against real-world attack patterns

## Security Measures Tested

### 1. Command Injection Prevention
- Semicolon injection (`;`)
- Pipe injection (`|`)
- Command substitution (`$()`, backticks)
- Logical operators (`&&`, `||`)
- Redirection operators (`>`, `<`, `>>`)
- Newline injection
- Null byte injection
- URL-encoded shell metacharacters

### 2. Path Traversal Protection
- Directory traversal sequences (`../`)
- Absolute paths to system directories
- Windows system paths
- Null bytes in paths
- Symlink attacks

### 3. Environment Variable Security
- Filtering of sensitive variables (AWS keys, tokens, passwords)
- Prevention of LD_PRELOAD hijacking
- Shell configuration protection
- Command injection in environment values
- PATH manipulation prevention

### 4. ReDoS Prevention
- Nested quantifiers detection
- Catastrophic backtracking patterns
- Excessive alternation
- Pattern complexity limits
- Compilation timeouts

### 5. Resource Limits
- Command timeout validation
- Output size limits
- Memory usage limits
- Pattern length restrictions

### 6. Defense in Depth
- Multiple security layers
- Fail-safe defaults
- Comprehensive validation
- Real-world attack scenarios

## Running the Security Tests

To run all security tests:
```bash
# Run all security tests
go test -v ./internal/... -run Security

# Run specific security test suites
go test -v ./internal/executor -run Security
go test -v ./internal/config -run Security
go test -v ./internal/filter -run Security
go test -v ./internal/security -run Security

# Run integration tests
go test -v ./internal/security -run Integration
```

## Test Coverage

The security tests provide comprehensive coverage of:
- All user input validation points
- All external command execution paths
- All configuration parsing and validation
- All regex pattern compilation
- All environment variable handling
- Real-world attack scenarios

## Continuous Security Testing

These tests should be:
1. Run on every commit
2. Included in CI/CD pipelines
3. Extended when new features are added
4. Updated when new attack vectors are discovered
5. Used as regression tests for security fixes