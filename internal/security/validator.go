// Package security provides security validation and protection mechanisms for qualhook.
package security

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// SecurityValidator provides comprehensive security validation
type SecurityValidator struct {
	// Command whitelist - empty means all commands allowed
	allowedCommands map[string]bool
	// Maximum allowed timeout
	maxTimeout time.Duration
	// Maximum regex pattern length
	maxRegexLength int
	// Maximum output size in bytes
	maxOutputSize int64
	// Banned path patterns
	bannedPaths []string
}

// NewSecurityValidator creates a new security validator with default settings
func NewSecurityValidator() *SecurityValidator {
	return &SecurityValidator{
		allowedCommands: getDefaultAllowedCommands(),
		maxTimeout:      1 * time.Hour,
		maxRegexLength:  500,
		maxOutputSize:   10 * 1024 * 1024, // 10MB
		bannedPaths: []string{
			"/etc",
			"/sys",
			"/proc",
			"/dev",
			"C:\\Windows",
			"C:\\System32",
		},
	}
}

// ValidateCommand checks if a command is allowed and safe to execute
func (v *SecurityValidator) ValidateCommand(command string, args []string) error {
	// Check if command is empty
	if command == "" {
		return fmt.Errorf("command cannot be empty")
	}

	// Extract base command name
	baseCommand := filepath.Base(command)

	// Check against whitelist if configured
	if len(v.allowedCommands) > 0 {
		if !v.allowedCommands[command] && !v.allowedCommands[baseCommand] {
			return fmt.Errorf("command '%s' is not in the allowed command list", command)
		}
	}

	// Check for shell injection attempts in command
	if err := v.checkForShellInjection(command); err != nil {
		return fmt.Errorf("potential shell injection in command: %w", err)
	}

	// Validate all arguments
	for i, arg := range args {
		if err := v.checkForShellInjection(arg); err != nil {
			return fmt.Errorf("potential shell injection in argument %d: %w", i, err)
		}
	}

	// Check for dangerous command patterns
	if err := v.checkDangerousCommands(command, args); err != nil {
		return err
	}

	return nil
}

// ValidatePath validates a file path to prevent directory traversal attacks
func (v *SecurityValidator) ValidatePath(path string) error {
	// Basic path validation
	if err := v.validateBasicPath(path); err != nil {
		return err
	}

	// Clean the path
	cleanPath := filepath.Clean(path)

	// Check for directory traversal attempts
	if strings.Contains(cleanPath, "..") {
		return fmt.Errorf("path contains directory traversal sequence")
	}

	// Convert to absolute path for validation
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return fmt.Errorf("cannot resolve path: %w", err)
	}

	// Check against banned paths
	if err := v.checkBannedPaths(absPath); err != nil {
		return err
	}

	// Check Windows-specific paths
	if err := v.checkWindowsPaths(path); err != nil {
		return err
	}

	// Ensure path is within allowed directories
	if err := v.checkPathScope(absPath, path); err != nil {
		return err
	}

	return nil
}

// ValidateRegexPattern validates a regex pattern for safety
func (v *SecurityValidator) ValidateRegexPattern(pattern string) error {
	// Check pattern length
	if len(pattern) > v.maxRegexLength {
		return fmt.Errorf("regex pattern too long (%d chars), maximum is %d", len(pattern), v.maxRegexLength)
	}

	// Check for ReDoS vulnerable patterns
	if err := v.checkReDoSPattern(pattern); err != nil {
		return fmt.Errorf("potentially vulnerable regex pattern: %w", err)
	}

	// Try to compile the pattern with a timeout
	// This prevents hanging on malicious patterns
	done := make(chan error, 1)
	go func() {
		_, err := regexp.Compile(pattern)
		done <- err
	}()

	select {
	case err := <-done:
		if err != nil {
			return fmt.Errorf("invalid regex pattern: %w", err)
		}
	case <-time.After(100 * time.Millisecond):
		return fmt.Errorf("regex compilation timeout - pattern may be too complex")
	}

	return nil
}

// ValidateTimeout validates a timeout duration
func (v *SecurityValidator) ValidateTimeout(timeout time.Duration) error {
	if timeout < 0 {
		return fmt.Errorf("timeout cannot be negative")
	}

	if timeout > v.maxTimeout {
		return fmt.Errorf("timeout %v exceeds maximum allowed %v", timeout, v.maxTimeout)
	}

	// Warn about very short timeouts
	if timeout > 0 && timeout < 100*time.Millisecond {
		return fmt.Errorf("timeout %v is too short, minimum recommended is 100ms", timeout)
	}

	return nil
}

// ValidateResourceLimits validates resource consumption limits
func (v *SecurityValidator) ValidateResourceLimits(outputSize int64, memoryLimit int64) error {
	if outputSize > v.maxOutputSize {
		return fmt.Errorf("output size %d exceeds maximum %d bytes", outputSize, v.maxOutputSize)
	}

	if memoryLimit > 0 {
		// Check reasonable memory limits (e.g., 1GB max)
		maxMemory := int64(1024 * 1024 * 1024) // 1GB
		if memoryLimit > maxMemory {
			return fmt.Errorf("memory limit %d exceeds maximum %d bytes", memoryLimit, maxMemory)
		}
	}

	return nil
}

// checkForShellInjection checks for potential shell injection attempts
func (v *SecurityValidator) checkForShellInjection(input string) error {
	// Check for null bytes first (special case)
	if strings.Contains(input, "\x00") {
		return fmt.Errorf("contains null byte")
	}

	// List of dangerous characters and patterns
	dangerousPatterns := []string{
		";", "&&", "||", "|", "`", "$(",
		"${", "\n", "\r", "\\x00",
		">>", "<<", ">", "<",
	}

	for _, pattern := range dangerousPatterns {
		if strings.Contains(input, pattern) {
			return fmt.Errorf("contains potentially dangerous pattern '%s'", pattern)
		}
	}

	// Check for encoded characters
	if strings.Contains(input, "%") {
		// Could be URL encoding attempt
		if strings.Contains(input, "%3B") || // ;
			strings.Contains(input, "%7C") || // |
			strings.Contains(input, "%26") || // &
			strings.Contains(input, "%24") { // $
			return fmt.Errorf("contains encoded shell metacharacters")
		}
	}

	return nil
}

// checkDangerousCommands checks for inherently dangerous commands
func (v *SecurityValidator) checkDangerousCommands(command string, args []string) error {
	baseCmd := filepath.Base(command)

	// Check if command is dangerous
	if !isDangerousCommand(baseCmd) {
		return nil
	}

	// Validate dangerous command based on type
	switch baseCmd {
	case "rm", "del":
		return v.validateRemoveCommand(baseCmd, args)
	case "curl", "wget":
		return v.validateDownloadCommand(baseCmd, args)
	default:
		// Other dangerous commands require additional scrutiny
		return nil
	}
}

// checkReDoSPattern checks for ReDoS vulnerable regex patterns
func (v *SecurityValidator) checkReDoSPattern(pattern string) error {
	// Patterns that can cause catastrophic backtracking
	vulnerablePatterns := []struct {
		pattern string
		name    string
	}{
		{`\(\.\*\)\*`, "nested quantifiers (.*)* "},
		{`\(\.\+\)\*`, "nested quantifiers (.+)* "},
		{`\(\.\*\)\+`, "nested quantifiers (.*)+ "},
		{`\(\.\+\)\+`, "nested quantifiers (.+)+ "},
		{`\([^)]*\+\)\+`, "nested quantifiers (x+)+"},
		{`\([^)]*\*\)\*`, "nested quantifiers (x*)*"},
		{`\([^)]*\+\)\*`, "nested quantifiers (x+)*"},
		{`\([^)]*\*\)\+`, "nested quantifiers (x*)+"},
		{`\(\([^)]+\)\)\+`, "nested groups ((x))+"},
		{`\([^)]*\|[^)]*\)\*`, "alternation with star (a|b)*"},
	}

	for _, vp := range vulnerablePatterns {
		matched, err := regexp.MatchString(vp.pattern, pattern)
		if err != nil {
			// Invalid pattern in our check, skip it
			continue
		}
		if matched {
			return fmt.Errorf("pattern contains %s which can cause catastrophic backtracking", vp.name)
		}
	}

	// Check for excessive alternation
	pipeCount := strings.Count(pattern, "|")
	if pipeCount > 10 {
		return fmt.Errorf("pattern contains too many alternations (%d), which can be slow", pipeCount)
	}

	// Check for excessive capturing groups
	openParens := strings.Count(pattern, "(")
	nonCapturing := strings.Count(pattern, "(?:")
	capturingGroups := openParens - nonCapturing
	if capturingGroups > 10 {
		return fmt.Errorf("pattern contains too many capturing groups (%d), maximum is 10", capturingGroups)
	}

	return nil
}

// getDefaultAllowedCommands returns the default whitelist of allowed commands
func getDefaultAllowedCommands() map[string]bool {
	// Common development tools that are generally safe
	commands := []string{
		// Node.js ecosystem
		"node", "npm", "yarn", "pnpm", "npx",
		// Go ecosystem
		"go", "gofmt", "golint", "go-staticcheck",
		// Rust ecosystem
		"cargo", "rustc", "rustfmt", "clippy-driver",
		// Python ecosystem
		"python", "python3", "pip", "pip3", "mypy", "pylint", "black", "flake8", "pytest",
		// Ruby ecosystem
		"ruby", "bundle", "rubocop", "rspec",
		// Java ecosystem
		"java", "javac", "gradle", "mvn", "maven",
		// .NET ecosystem
		"dotnet", "msbuild",
		// Version control
		"git", "hg", "svn",
		// Common build tools
		"make", "cmake", "bazel",
		// Linters and formatters
		"eslint", "prettier", "tslint", "stylelint",
		// Testing tools
		"jest", "mocha", "karma", "cypress",
		// TypeScript
		"tsc", "typescript",
		// Shell utilities (safe subset)
		"echo", "pwd", "which", "where", "type",
	}

	allowed := make(map[string]bool)
	for _, cmd := range commands {
		allowed[cmd] = true
	}

	// Return empty map to allow all commands by default
	// Users can configure this for stricter security
	return map[string]bool{}
}

// SetAllowedCommands updates the list of allowed commands
func (v *SecurityValidator) SetAllowedCommands(commands []string) {
	v.allowedCommands = make(map[string]bool)
	for _, cmd := range commands {
		v.allowedCommands[cmd] = true
	}
}

// SetMaxTimeout updates the maximum allowed timeout
func (v *SecurityValidator) SetMaxTimeout(timeout time.Duration) {
	v.maxTimeout = timeout
}

// SetMaxRegexLength updates the maximum allowed regex pattern length
func (v *SecurityValidator) SetMaxRegexLength(length int) {
	v.maxRegexLength = length
}

// SetMaxOutputSize updates the maximum allowed output size
func (v *SecurityValidator) SetMaxOutputSize(size int64) {
	v.maxOutputSize = size
}

// validateBasicPath performs basic path validation checks
func (v *SecurityValidator) validateBasicPath(path string) error {
	// Empty path is invalid
	if path == "" {
		return fmt.Errorf("path cannot be empty")
	}

	// Check for null bytes
	if strings.Contains(path, "\x00") {
		return fmt.Errorf("path contains null byte")
	}

	return nil
}

// checkBannedPaths checks if the path matches any banned paths
func (v *SecurityValidator) checkBannedPaths(absPath string) error {
	for _, banned := range v.bannedPaths {
		// Normalize the banned path for comparison
		normalizedBanned := filepath.Clean(banned)
		if strings.HasPrefix(absPath, normalizedBanned) || 
		   strings.HasPrefix(strings.ToLower(absPath), strings.ToLower(normalizedBanned)) {
			return fmt.Errorf("access to path '%s' is forbidden", banned)
		}
	}
	return nil
}

// checkWindowsPaths checks Windows-specific path restrictions
func (v *SecurityValidator) checkWindowsPaths(path string) error {
	// Check if it's a Windows path on any system (for cross-platform validation)
	if len(path) >= 3 && path[1] == ':' && (path[2] == '\\' || path[2] == '/') {
		// This is a Windows absolute path like C:\ or D:/
		drive := strings.ToUpper(string(path[0]))
		if drive >= "A" && drive <= "Z" {
			// Check against Windows banned paths
			for _, banned := range v.bannedPaths {
				if strings.Contains(banned, ":\\") || strings.Contains(banned, ":/") {
					if strings.HasPrefix(strings.ToLower(path), strings.ToLower(banned)) {
						return fmt.Errorf("access to path '%s' is forbidden", banned)
					}
				}
			}
		}
	}
	return nil
}

// checkPathScope ensures path is within allowed directories
func (v *SecurityValidator) checkPathScope(absPath, originalPath string) error {
	// Ensure path is within working directory or its subdirectories
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("cannot get working directory: %w", err)
	}

	// Allow paths within current directory
	if !strings.HasPrefix(absPath, cwd) {
		// Also allow temp directories for certain operations
		tempDir := os.TempDir()
		if !strings.HasPrefix(absPath, tempDir) {
			return fmt.Errorf("path '%s' is outside project directory", originalPath)
		}
	}

	return nil
}

// isDangerousCommand checks if a command is in the dangerous commands list
func isDangerousCommand(baseCmd string) bool {
	dangerousCommands := map[string]bool{
		"rm":     true,
		"del":    true,
		"format": true,
		"dd":     true,
		"mkfs":   true,
		"fdisk":  true,
		"curl":   true,
		"wget":   true,
		"nc":     true,
		"netcat": true,
	}
	return dangerousCommands[baseCmd]
}

// validateRemoveCommand validates rm/del commands for dangerous flags
func (v *SecurityValidator) validateRemoveCommand(baseCmd string, args []string) error {
	hasRecursive := false
	hasForce := false
	
	for _, arg := range args {
		// Check for combined flags
		if arg == "-rf" || arg == "-fr" {
			return fmt.Errorf("dangerous %s command with force/recursive flags", baseCmd)
		}
		
		// Check individual flags
		if arg == "-r" || arg == "-R" || arg == "--recursive" {
			hasRecursive = true
		}
		if arg == "-f" || arg == "--force" {
			hasForce = true
		}
	}
	
	if hasRecursive && hasForce {
		return fmt.Errorf("dangerous %s command with force/recursive flags", baseCmd)
	}
	
	return nil
}

// validateDownloadCommand validates curl/wget commands for dangerous output paths
func (v *SecurityValidator) validateDownloadCommand(baseCmd string, args []string) error {
	for i, arg := range args {
		if arg == "-o" || arg == "--output" {
			if i+1 < len(args) {
				outputPath := args[i+1]
				if err := v.ValidatePath(outputPath); err != nil {
					return fmt.Errorf("dangerous output path for %s: %w", baseCmd, err)
				}
			}
		}
	}
	return nil
}