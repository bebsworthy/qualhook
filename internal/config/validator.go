// Package config provides configuration validation utilities
package config

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/qualhook/qualhook/pkg/config"
)

// Validator provides enhanced validation for configurations
type Validator struct {
	// CheckCommands indicates whether to validate command existence in PATH
	CheckCommands bool

	// AllowedCommands is a whitelist of allowed commands (empty means all allowed)
	AllowedCommands []string
}

// NewValidator creates a new configuration validator
func NewValidator() *Validator {
	return &Validator{
		CheckCommands: true,
		AllowedCommands: getDefaultAllowedCommands(),
	}
}

// Validate performs comprehensive validation on a configuration
func (v *Validator) Validate(cfg *config.Config) error {
	// Basic validation is already done by config.Validate()
	if err := cfg.Validate(); err != nil {
		return err
	}

	// Validate all commands
	for name, cmd := range cfg.Commands {
		if err := v.validateCommand(name, cmd); err != nil {
			return fmt.Errorf("command %q: %w", name, err)
		}
	}

	// Validate path configurations
	for i, pathCfg := range cfg.Paths {
		if err := v.validatePathConfig(pathCfg); err != nil {
			return fmt.Errorf("path config %d (%s): %w", i, pathCfg.Path, err)
		}
	}

	return nil
}

// ValidateCommand validates a single command configuration
func (v *Validator) ValidateCommand(cmd *config.CommandConfig) error {
	return v.validateCommand("", cmd)
}

// validateCommand performs validation on a command configuration
func (v *Validator) validateCommand(name string, cmd *config.CommandConfig) error {
	// Check if command is in allowed list
	if len(v.AllowedCommands) > 0 && !v.isCommandAllowed(cmd.Command) {
		return fmt.Errorf("command %q is not in allowed list", cmd.Command)
	}

	// Check if command exists in PATH
	if v.CheckCommands {
		if err := v.checkCommandExists(cmd.Command); err != nil {
			return err
		}
	}

	// Validate regex patterns
	if cmd.ErrorDetection != nil {
		for i, pattern := range cmd.ErrorDetection.Patterns {
			if err := v.validateRegexPattern(pattern); err != nil {
				return fmt.Errorf("error detection pattern %d: %w", i, err)
			}
		}
	}

	if cmd.OutputFilter != nil {
		for i, pattern := range cmd.OutputFilter.ErrorPatterns {
			if err := v.validateRegexPattern(pattern); err != nil {
				return fmt.Errorf("error pattern %d: %w", i, err)
			}
		}

		for i, pattern := range cmd.OutputFilter.IncludePatterns {
			if err := v.validateRegexPattern(pattern); err != nil {
				return fmt.Errorf("include pattern %d: %w", i, err)
			}
		}
	}

	// Validate timeout
	if cmd.Timeout > 0 && cmd.Timeout < 100 {
		return fmt.Errorf("timeout %dms is too short, minimum is 100ms", cmd.Timeout)
	}

	if cmd.Timeout > 3600000 { // 1 hour
		return fmt.Errorf("timeout %dms exceeds maximum of 1 hour", cmd.Timeout)
	}

	return nil
}

// validatePathConfig validates a path configuration
func (v *Validator) validatePathConfig(pathCfg *config.PathConfig) error {
	// Validate path pattern
	if err := v.validatePathPattern(pathCfg.Path); err != nil {
		return fmt.Errorf("invalid path pattern: %w", err)
	}

	// Validate commands
	for name, cmd := range pathCfg.Commands {
		if err := v.validateCommand(name, cmd); err != nil {
			return fmt.Errorf("command %q: %w", name, err)
		}
	}

	return nil
}

// validateRegexPattern validates a regex pattern thoroughly
func (v *Validator) validateRegexPattern(pattern *config.RegexPattern) error {
	if pattern == nil {
		return nil
	}

	// Basic validation is done by pattern.Validate()
	if err := pattern.Validate(); err != nil {
		return err
	}

	// Check for potentially problematic patterns
	if err := v.checkDangerousRegex(pattern.Pattern); err != nil {
		return fmt.Errorf("potentially dangerous regex: %w", err)
	}

	// Try to compile and test the pattern
	re, err := pattern.Compile()
	if err != nil {
		return fmt.Errorf("failed to compile regex: %w", err)
	}

	// Check for patterns that might match too much
	if v.isTooGenericPattern(re, pattern.Pattern) {
		return fmt.Errorf("pattern %q is too generic and might match too much output", pattern.Pattern)
	}

	return nil
}

// checkDangerousRegex checks for regex patterns that could cause performance issues
func (v *Validator) checkDangerousRegex(pattern string) error {
	// Check for catastrophic backtracking patterns
	// Look for specific dangerous constructs
	if strings.Contains(pattern, "(.*)*") || 
	   strings.Contains(pattern, "(.+)*") ||
	   strings.Contains(pattern, "(\\s*)*") ||
	   regexp.MustCompile(`\([^)]*\+\)\+`).MatchString(pattern) || // (x+)+
	   regexp.MustCompile(`\([^)]*\*\)\*`).MatchString(pattern) {   // (x*)*
		return fmt.Errorf("pattern contains potential catastrophic backtracking")
	}

	// Check for overly complex patterns
	if len(pattern) > 500 {
		return fmt.Errorf("pattern is too long (%d chars), maximum is 500", len(pattern))
	}

	// Count capturing groups
	captureCount := strings.Count(pattern, "(") - strings.Count(pattern, "(?:")
	if captureCount > 10 {
		return fmt.Errorf("too many capturing groups (%d), maximum is 10", captureCount)
	}

	return nil
}

// isTooGenericPattern checks if a pattern might match too broadly
func (v *Validator) isTooGenericPattern(re *regexp.Regexp, pattern string) bool {
	// List of patterns that are too generic
	tooGeneric := []string{
		"^.*$",
		"^.+$",
		".*",
		".+",
		"\\w+",
		"\\s+",
	}

	cleanPattern := strings.TrimSpace(pattern)
	for _, generic := range tooGeneric {
		if cleanPattern == generic {
			return true
		}
	}

	// Test the pattern against common output to see if it matches everything
	testStrings := []string{
		"normal output line",
		"Error: something went wrong",
		"Warning: deprecated function",
		"   at file.js:10:5",
		"✓ Test passed",
		"",
	}

	matchCount := 0
	for _, test := range testStrings {
		if re.MatchString(test) {
			matchCount++
		}
	}

	// If pattern matches most test strings, it's probably too generic
	return matchCount >= len(testStrings)-1
}

// validatePathPattern validates a path glob pattern
func (v *Validator) validatePathPattern(pattern string) error {
	if pattern == "" {
		return fmt.Errorf("path pattern cannot be empty")
	}

	// Check for invalid characters
	if strings.ContainsAny(pattern, "\x00") {
		return fmt.Errorf("path pattern contains null character")
	}

	// Check for absolute paths (security risk)
	if filepath.IsAbs(pattern) {
		return fmt.Errorf("absolute paths are not allowed in patterns")
	}

	// Check for parent directory references
	if strings.Contains(pattern, "..") {
		return fmt.Errorf("parent directory references (..) are not allowed")
	}

	// Validate glob syntax
	if _, err := filepath.Match(pattern, "test"); err != nil {
		return fmt.Errorf("invalid glob pattern: %w", err)
	}

	return nil
}

// checkCommandExists verifies that a command exists in PATH
func (v *Validator) checkCommandExists(command string) error {
	// Don't check for shell built-ins
	shellBuiltins := []string{"echo", "cd", "pwd", "exit", "export", "alias"}
	for _, builtin := range shellBuiltins {
		if command == builtin {
			return nil
		}
	}

	// Special handling for commands with paths
	if strings.Contains(command, "/") || strings.Contains(command, "\\") {
		// Check if it's a relative path that might exist
		if _, err := os.Stat(command); err == nil {
			return nil
		}
		return fmt.Errorf("command %q not found at specified path", command)
	}

	// Use exec.LookPath to find the command
	path, err := exec.LookPath(command)
	if err != nil {
		// Provide helpful error message
		if runtime.GOOS == "windows" {
			return fmt.Errorf("command %q not found in PATH (did you mean %s.exe?)", command, command)
		}
		return fmt.Errorf("command %q not found in PATH", command)
	}

	// Verify the found path is executable
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("cannot stat command %q: %w", command, err)
	}

	if runtime.GOOS != "windows" && info.Mode()&0111 == 0 {
		return fmt.Errorf("command %q is not executable", command)
	}

	return nil
}

// isCommandAllowed checks if a command is in the allowed list
func (v *Validator) isCommandAllowed(command string) bool {
	// Extract base command name (e.g., "npm" from "/usr/local/bin/npm")
	baseCommand := filepath.Base(command)
	
	for _, allowed := range v.AllowedCommands {
		if allowed == command || allowed == baseCommand {
			return true
		}
	}
	return false
}

// getDefaultAllowedCommands returns a default list of allowed commands
func getDefaultAllowedCommands() []string {
	// Empty list means all commands are allowed
	// This can be overridden by users who want stricter security
	return []string{}
}

// SuggestFixes provides suggestions for common configuration errors
func (v *Validator) SuggestFixes(err error) []string {
	errStr := err.Error()
	suggestions := []string{}

	// Command not found suggestions
	if strings.Contains(errStr, "not found in PATH") {
		suggestions = append(suggestions, 
			"Make sure the command is installed and available in your PATH",
			"Try running 'which <command>' (Unix) or 'where <command>' (Windows) to verify",
		)
		
		// Specific suggestions for common commands
		if strings.Contains(errStr, "npm") {
			suggestions = append(suggestions, "Install Node.js from https://nodejs.org/")
		} else if strings.Contains(errStr, "go") {
			suggestions = append(suggestions, "Install Go from https://golang.org/")
		} else if strings.Contains(errStr, "cargo") {
			suggestions = append(suggestions, "Install Rust from https://rustup.rs/")
		} else if strings.Contains(errStr, "python") {
			suggestions = append(suggestions, "Install Python from https://python.org/")
		}
	}

	// Regex pattern errors
	if strings.Contains(errStr, "regex") || strings.Contains(errStr, "pattern") {
		suggestions = append(suggestions,
			"Check your regex pattern syntax",
			"Test your pattern at https://regex101.com/",
			"Escape special characters like '.', '*', '+', '?', '[', ']', '(', ')', '{', '}'",
		)
	}

	// Timeout errors
	if strings.Contains(errStr, "timeout") {
		suggestions = append(suggestions,
			"Use a timeout between 100ms and 3600000ms (1 hour)",
			"Consider if your command really needs a long timeout",
		)
	}

	// Path pattern errors
	if strings.Contains(errStr, "path pattern") {
		suggestions = append(suggestions,
			"Use relative paths only (no leading /)",
			"Use ** for recursive matching (e.g., 'src/**/*.js')",
			"Avoid using .. in path patterns",
		)
	}

	return suggestions
}