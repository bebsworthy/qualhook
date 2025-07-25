// Package security provides environment variable sanitization
package security

import (
	"fmt"
	"os"
	"strings"
)

// SensitiveEnvVars contains environment variables that should not be passed to subprocesses
var SensitiveEnvVars = []string{
	// Authentication and secrets
	"AWS_SECRET_ACCESS_KEY",
	"AWS_SESSION_TOKEN",
	"GITHUB_TOKEN",
	"GITLAB_TOKEN",
	"NPM_TOKEN",
	"DOCKER_PASSWORD",
	"DATABASE_PASSWORD",
	"DATABASE_URL",
	"DB_PASSWORD",
	"DB_URL",
	"MONGODB_URI",
	"MYSQL_ROOT_PASSWORD",
	"REDIS_PASSWORD",
	"API_KEY",
	"API_SECRET",
	"SECRET_KEY",
	"PRIVATE_KEY",
	"SSH_PRIVATE_KEY",
	"GPG_PRIVATE_KEY",

	// System paths that could be exploited
	"LD_PRELOAD",
	"LD_LIBRARY_PATH",
	"DYLD_INSERT_LIBRARIES",
	"DYLD_LIBRARY_PATH",

	// Shell configuration
	"BASH_ENV",
	"ENV",
	"SHELL",
	"ZDOTDIR",
}

// SafeEnvVars contains environment variables that are safe to pass
var SafeEnvVars = []string{
	// Basic system info
	"HOME",
	"USER",
	"LANG",
	"LC_ALL",
	"TZ",
	"TMPDIR",
	"TEMP",
	"TMP",

	// Development tools
	"PATH",
	"NODE_ENV",
	"GOPATH",
	"GOROOT",
	"CARGO_HOME",
	"RUSTUP_HOME",
	"PYTHON_HOME",
	"JAVA_HOME",

	// CI/CD indicators (read-only)
	"CI",
	"CONTINUOUS_INTEGRATION",
	"GITHUB_ACTIONS",
	"GITLAB_CI",
	"JENKINS_HOME",
	"TRAVIS",
	"CIRCLECI",

	// Terminal settings
	"TERM",
	"COLORTERM",
	"COLUMNS",
	"LINES",
}

// SanitizeEnvironment filters environment variables for safe subprocess execution
func SanitizeEnvironment(env []string, allowInherit bool) []string {
	if !allowInherit {
		// Start with minimal environment
		return createMinimalEnvironment()
	}

	// Filter inherited environment
	filtered := make([]string, 0, len(env))
	sensitive := make(map[string]bool)

	// Build sensitive vars map
	for _, v := range SensitiveEnvVars {
		sensitive[v] = true
	}

	for _, envVar := range env {
		parts := strings.SplitN(envVar, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := parts[0]
		value := parts[1]

		// Skip sensitive variables
		if sensitive[key] {
			continue
		}

		// Skip variables with sensitive patterns
		if containsSensitivePattern(key) {
			continue
		}

		// Validate the value doesn't contain injection attempts
		if err := validateEnvValue(key, value); err != nil {
			// Skip variables with suspicious values
			continue
		}

		filtered = append(filtered, envVar)
	}

	return filtered
}

// createMinimalEnvironment creates a minimal safe environment
func createMinimalEnvironment() []string {
	env := []string{}

	// Add only essential variables from current environment
	for _, key := range SafeEnvVars {
		if value := os.Getenv(key); value != "" {
			// Validate even safe variables
			if err := validateEnvValue(key, value); err == nil {
				env = append(env, fmt.Sprintf("%s=%s", key, value))
			}
		}
	}

	return env
}

// containsSensitivePattern checks if a key contains sensitive patterns
func containsSensitivePattern(key string) bool {
	upperKey := strings.ToUpper(key)

	sensitivePatterns := []string{
		"PASSWORD",
		"SECRET",
		"TOKEN",
		"KEY",
		"AUTH",
		"CREDENTIAL",
		"PRIVATE",
		"ACCESS",
	}

	for _, pattern := range sensitivePatterns {
		if strings.Contains(upperKey, pattern) {
			return true
		}
	}

	return false
}

// validateEnvValue validates an environment variable value for safety
func validateEnvValue(key, value string) error {
	// Check for null bytes
	if strings.Contains(value, "\x00") {
		return fmt.Errorf("environment variable %s contains null byte", key)
	}

	// For PATH-like variables, check for suspicious entries
	if strings.HasSuffix(key, "PATH") {
		if err := validatePathEnv(value); err != nil {
			return fmt.Errorf("invalid %s: %w", key, err)
		}
	}

	// Check for command injection patterns
	dangerousPatterns := []string{
		"$(", "${", "`",
		"&&", "||", ";",
		"|", ">", "<",
		"\n", "\r",
	}

	for _, pattern := range dangerousPatterns {
		if strings.Contains(value, pattern) {
			return fmt.Errorf("environment variable %s contains dangerous pattern: %s", key, pattern)
		}
	}

	return nil
}

// validatePathEnv validates PATH-like environment variables
func validatePathEnv(value string) error {
	// Split by path separator
	separator := string(os.PathListSeparator)
	paths := strings.Split(value, separator)

	for _, path := range paths {
		// Skip empty entries
		if path == "" {
			continue
		}

		// Check for parent directory references first (more specific)
		if strings.Contains(path, "..") {
			return fmt.Errorf("parent directory reference in PATH: %s", path)
		}

		// Check for relative paths (security risk)
		if strings.HasPrefix(path, ".") {
			return fmt.Errorf("relative path in PATH variable: %s", path)
		}

		// Check for suspicious patterns
		if strings.Contains(path, "~") {
			return fmt.Errorf("home directory expansion in PATH: %s", path)
		}
	}

	return nil
}

// MergeEnvironment merges custom environment variables with a base environment
func MergeEnvironment(base []string, custom []string) ([]string, error) {
	envMap := make(map[string]string)

	// Start with base environment
	for _, env := range base {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}

	// Override with custom environment
	for _, env := range custom {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid environment variable format: %s", env)
		}

		key := parts[0]
		value := parts[1]

		// Validate custom environment variables
		if err := validateEnvValue(key, value); err != nil {
			return nil, fmt.Errorf("invalid custom environment variable: %w", err)
		}

		envMap[key] = value
	}

	// Convert back to slice
	result := make([]string, 0, len(envMap))
	for k, v := range envMap {
		result = append(result, fmt.Sprintf("%s=%s", k, v))
	}

	return result, nil
}
