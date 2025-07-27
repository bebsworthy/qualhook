// Package config provides the core configuration types and validation logic for qualhook.
package config

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// Config represents the main configuration structure for qualhook
type Config struct {
	Version     string                    `json:"version"`
	ProjectType string                    `json:"projectType,omitempty"`
	Commands    map[string]*CommandConfig `json:"commands"`
	Paths       []*PathConfig             `json:"paths,omitempty"`
}

// CommandConfig defines configuration for a single command
type CommandConfig struct {
	Command         string          `json:"command"`
	Args            []string        `json:"args,omitempty"`
	Prompt          string          `json:"prompt,omitempty"`
	Timeout         int             `json:"timeout,omitempty"` // milliseconds
	ExitCodes       []int           `json:"exitCodes,omitempty"`
	ErrorPatterns   []*RegexPattern `json:"errorPatterns,omitempty"`
	ContextLines    int             `json:"contextLines,omitempty"`
	MaxOutput       int             `json:"maxOutput,omitempty"`
	IncludePatterns []*RegexPattern `json:"includePatterns,omitempty"`
}

// PathConfig defines path-specific configuration for monorepo support
type PathConfig struct {
	Path     string                    `json:"path"`
	Extends  string                    `json:"extends,omitempty"`
	Commands map[string]*CommandConfig `json:"commands"`
}

// RegexPattern represents a regex pattern with optional flags
type RegexPattern struct {
	Pattern string `json:"pattern"`
	Flags   string `json:"flags,omitempty"`
}

// Validate performs validation on the Config
func (c *Config) Validate() error {
	if c.Version == "" {
		return fmt.Errorf("version is required")
	}

	if len(c.Commands) == 0 && len(c.Paths) == 0 {
		return fmt.Errorf("at least one command or path configuration is required")
	}

	// Validate commands
	for name, cmd := range c.Commands {
		if err := cmd.Validate(); err != nil {
			return fmt.Errorf("command %q: %w", name, err)
		}
	}

	// Validate paths
	for i, path := range c.Paths {
		if err := path.Validate(); err != nil {
			return fmt.Errorf("path config %d: %w", i, err)
		}
	}

	return nil
}

// Validate performs validation on the CommandConfig
func (c *CommandConfig) Validate() error {
	if c.Command == "" {
		return fmt.Errorf("command is required")
	}

	// Validate error patterns
	for i, pattern := range c.ErrorPatterns {
		if err := pattern.Validate(); err != nil {
			return fmt.Errorf("error pattern %d: %w", i, err)
		}
	}

	// Validate include patterns
	for i, pattern := range c.IncludePatterns {
		if err := pattern.Validate(); err != nil {
			return fmt.Errorf("include pattern %d: %w", i, err)
		}
	}

	if c.ContextLines < 0 {
		return fmt.Errorf("context lines must be non-negative")
	}

	if c.MaxOutput < 0 {
		return fmt.Errorf("max output must be non-negative")
	}

	if c.Timeout < 0 {
		return fmt.Errorf("timeout must be non-negative")
	}

	return nil
}

// Validate performs validation on the PathConfig
func (p *PathConfig) Validate() error {
	if p.Path == "" {
		return fmt.Errorf("path is required")
	}

	for name, cmd := range p.Commands {
		if err := cmd.Validate(); err != nil {
			return fmt.Errorf("command %q: %w", name, err)
		}
	}

	return nil
}

// Validate performs validation on the RegexPattern
func (r *RegexPattern) Validate() error {
	if r.Pattern == "" {
		return fmt.Errorf("pattern is required")
	}

	// Validate the regex pattern
	if err := r.validateRegex(); err != nil {
		return fmt.Errorf("invalid regex pattern: %w", err)
	}

	// Validate flags
	if r.Flags != "" {
		validFlags := "imsU"
		for _, flag := range r.Flags {
			if !strings.ContainsRune(validFlags, flag) {
				return fmt.Errorf("invalid regex flag: %c", flag)
			}
		}
	}

	return nil
}

// validateRegex checks if the regex pattern is valid
func (r *RegexPattern) validateRegex() error {
	pattern := r.Pattern

	// Add flags to pattern if specified
	if r.Flags != "" {
		pattern = "(?" + r.Flags + ")" + pattern
	}

	_, err := regexp.Compile(pattern)
	return err
}

// Compile returns a compiled regular expression
func (r *RegexPattern) Compile() (*regexp.Regexp, error) {
	pattern := r.Pattern

	// Add flags to pattern if specified
	if r.Flags != "" {
		pattern = "(?" + r.Flags + ")" + pattern
	}

	return regexp.Compile(pattern)
}

// LoadConfig loads a configuration from JSON data
func LoadConfig(data []byte) (*Config, error) {
	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &config, nil
}

// SaveConfig serializes a configuration to JSON
func SaveConfig(config *Config) ([]byte, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	return data, nil
}

// Clone creates a deep copy of the CommandConfig
func (c *CommandConfig) Clone() *CommandConfig {
	if c == nil {
		return nil
	}

	clone := &CommandConfig{
		Command:      c.Command,
		Prompt:       c.Prompt,
		Timeout:      c.Timeout,
		ContextLines: c.ContextLines,
		MaxOutput:    c.MaxOutput,
	}

	if c.Args != nil {
		clone.Args = make([]string, len(c.Args))
		copy(clone.Args, c.Args)
	}

	if c.ExitCodes != nil {
		clone.ExitCodes = make([]int, len(c.ExitCodes))
		copy(clone.ExitCodes, c.ExitCodes)
	}

	if c.ErrorPatterns != nil {
		clone.ErrorPatterns = make([]*RegexPattern, len(c.ErrorPatterns))
		for i, p := range c.ErrorPatterns {
			if p != nil {
				clone.ErrorPatterns[i] = &RegexPattern{
					Pattern: p.Pattern,
					Flags:   p.Flags,
				}
			}
		}
	}

	if c.IncludePatterns != nil {
		clone.IncludePatterns = make([]*RegexPattern, len(c.IncludePatterns))
		for i, p := range c.IncludePatterns {
			if p != nil {
				clone.IncludePatterns[i] = &RegexPattern{
					Pattern: p.Pattern,
					Flags:   p.Flags,
				}
			}
		}
	}

	return clone
}
