// Package security provides security configuration options
package security

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Config represents security configuration options
type Config struct {
	// AllowedCommands is a whitelist of allowed commands
	// Empty list means all commands are allowed
	AllowedCommands []string `json:"allowedCommands,omitempty"`

	// MaxTimeout is the maximum allowed timeout for command execution
	MaxTimeout string `json:"maxTimeout,omitempty"`

	// MaxRegexLength is the maximum allowed regex pattern length
	MaxRegexLength int `json:"maxRegexLength,omitempty"`

	// MaxOutputSize is the maximum allowed output size in bytes
	MaxOutputSize int64 `json:"maxOutputSize,omitempty"`

	// BannedPaths is a list of paths that are forbidden to access
	BannedPaths []string `json:"bannedPaths,omitempty"`

	// EnableStrictMode enables stricter security checks
	EnableStrictMode bool `json:"enableStrictMode,omitempty"`
}

// LoadConfig loads security configuration from a file
func LoadConfig(configPath string) (*Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Return default config if file doesn't exist
			return DefaultConfig(), nil
		}
		return nil, fmt.Errorf("failed to read security config: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse security config: %w", err)
	}

	// Apply defaults for missing values
	config.applyDefaults()

	return &config, nil
}

// DefaultConfig returns the default security configuration
func DefaultConfig() *Config {
	config := &Config{
		AllowedCommands: []string{}, // Empty means all allowed
		MaxTimeout:      "1h",
		MaxRegexLength:  500,
		MaxOutputSize:   10 * 1024 * 1024, // 10MB
		BannedPaths: []string{
			"/etc",
			"/sys",
			"/proc",
			"/dev",
			"C:\\Windows",
			"C:\\System32",
		},
		EnableStrictMode: false,
	}
	return config
}

// StrictConfig returns a strict security configuration
func StrictConfig() *Config {
	return &Config{
		AllowedCommands: []string{
			// Node.js ecosystem
			"node", "npm", "yarn", "pnpm", "npx",
			// Go ecosystem
			"go", "gofmt", "golint",
			// Rust ecosystem
			"cargo", "rustc", "rustfmt",
			// Python ecosystem
			"python", "python3", "pip", "pip3", "mypy", "pylint", "black", "flake8", "pytest",
			// Common linters and formatters
			"eslint", "prettier", "tslint", "stylelint",
			// Testing tools
			"jest", "mocha", "karma", "cypress",
			// TypeScript
			"tsc", "typescript",
			// Version control (read-only operations)
			"git",
		},
		MaxTimeout:     "5m",
		MaxRegexLength: 200,
		MaxOutputSize:  1 * 1024 * 1024, // 1MB
		BannedPaths: []string{
			"/",
			"/etc",
			"/sys",
			"/proc",
			"/dev",
			"/usr",
			"/bin",
			"/sbin",
			"/var",
			"/tmp",
			"~",
			"$HOME",
			"C:\\",
			"C:\\Windows",
			"C:\\System32",
			"C:\\Program Files",
			"C:\\Program Files (x86)",
		},
		EnableStrictMode: true,
	}
}

// applyDefaults applies default values to missing configuration fields
func (c *Config) applyDefaults() {
	defaults := DefaultConfig()

	if c.MaxTimeout == "" {
		c.MaxTimeout = defaults.MaxTimeout
	}

	if c.MaxRegexLength == 0 {
		c.MaxRegexLength = defaults.MaxRegexLength
	}

	if c.MaxOutputSize == 0 {
		c.MaxOutputSize = defaults.MaxOutputSize
	}

	if len(c.BannedPaths) == 0 {
		c.BannedPaths = defaults.BannedPaths
	}
}

// ParseTimeout parses the timeout string into a time.Duration
func (c *Config) ParseTimeout() (time.Duration, error) {
	return time.ParseDuration(c.MaxTimeout)
}

// Save saves the security configuration to a file
func (c *Config) Save(configPath string) error {
	// Ensure directory exists
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Marshal config to JSON
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal security config: %w", err)
	}

	// Write to file with restricted permissions
	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write security config: %w", err)
	}

	return nil
}

// ApplyToValidator applies the configuration to a SecurityValidator
func (c *Config) ApplyToValidator(v *SecurityValidator) error {
	// Set allowed commands
	if len(c.AllowedCommands) > 0 {
		v.SetAllowedCommands(c.AllowedCommands)
	}

	// Parse and set max timeout
	timeout, err := c.ParseTimeout()
	if err != nil {
		return fmt.Errorf("invalid maxTimeout: %w", err)
	}
	v.SetMaxTimeout(timeout)

	// Set other limits
	v.SetMaxRegexLength(c.MaxRegexLength)
	v.SetMaxOutputSize(c.MaxOutputSize)

	// Update banned paths
	v.bannedPaths = c.BannedPaths

	return nil
}