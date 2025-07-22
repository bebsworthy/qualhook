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
	Command        string          `json:"command"`
	Args           []string        `json:"args,omitempty"`
	ErrorDetection *ErrorDetection `json:"errorDetection"`
	OutputFilter   *FilterConfig   `json:"outputFilter"`
	Prompt         string          `json:"prompt,omitempty"`
	Timeout        int             `json:"timeout,omitempty"` // milliseconds
}

// PathConfig defines path-specific configuration for monorepo support
type PathConfig struct {
	Path     string                    `json:"path"`
	Extends  string                    `json:"extends,omitempty"`
	Commands map[string]*CommandConfig `json:"commands"`
}

// ErrorDetection defines how to detect errors in command output
type ErrorDetection struct {
	ExitCodes []int           `json:"exitCodes,omitempty"`
	Patterns  []*RegexPattern `json:"patterns,omitempty"`
}

// FilterConfig defines output filtering rules
type FilterConfig struct {
	ErrorPatterns   []*RegexPattern `json:"errorPatterns"`
	ContextLines    int             `json:"contextLines,omitempty"`
	MaxOutput       int             `json:"maxOutput,omitempty"`
	IncludePatterns []*RegexPattern `json:"includePatterns,omitempty"`
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

	if c.ErrorDetection != nil {
		if err := c.ErrorDetection.Validate(); err != nil {
			return fmt.Errorf("error detection: %w", err)
		}
	}

	if c.OutputFilter != nil {
		if err := c.OutputFilter.Validate(); err != nil {
			return fmt.Errorf("output filter: %w", err)
		}
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

// Validate performs validation on the ErrorDetection
func (e *ErrorDetection) Validate() error {
	for i, pattern := range e.Patterns {
		if err := pattern.Validate(); err != nil {
			return fmt.Errorf("pattern %d: %w", i, err)
		}
	}

	return nil
}

// Validate performs validation on the FilterConfig
func (f *FilterConfig) Validate() error {
	if len(f.ErrorPatterns) == 0 {
		return fmt.Errorf("at least one error pattern is required")
	}

	for i, pattern := range f.ErrorPatterns {
		if err := pattern.Validate(); err != nil {
			return fmt.Errorf("error pattern %d: %w", i, err)
		}
	}

	for i, pattern := range f.IncludePatterns {
		if err := pattern.Validate(); err != nil {
			return fmt.Errorf("include pattern %d: %w", i, err)
		}
	}

	if f.ContextLines < 0 {
		return fmt.Errorf("context lines must be non-negative")
	}

	if f.MaxOutput < 0 {
		return fmt.Errorf("max output must be non-negative")
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
		Command: c.Command,
		Prompt:  c.Prompt,
		Timeout: c.Timeout,
	}

	if c.Args != nil {
		clone.Args = make([]string, len(c.Args))
		copy(clone.Args, c.Args)
	}

	if c.ErrorDetection != nil {
		clone.ErrorDetection = c.ErrorDetection.Clone()
	}

	if c.OutputFilter != nil {
		clone.OutputFilter = c.OutputFilter.Clone()
	}

	return clone
}

// Clone creates a deep copy of the ErrorDetection
func (e *ErrorDetection) Clone() *ErrorDetection {
	if e == nil {
		return nil
	}

	clone := &ErrorDetection{}

	if e.ExitCodes != nil {
		clone.ExitCodes = make([]int, len(e.ExitCodes))
		copy(clone.ExitCodes, e.ExitCodes)
	}

	if e.Patterns != nil {
		clone.Patterns = make([]*RegexPattern, len(e.Patterns))
		for i, p := range e.Patterns {
			if p != nil {
				clone.Patterns[i] = &RegexPattern{
					Pattern: p.Pattern,
					Flags:   p.Flags,
				}
			}
		}
	}

	return clone
}

// Clone creates a deep copy of the FilterConfig
func (f *FilterConfig) Clone() *FilterConfig {
	if f == nil {
		return nil
	}

	clone := &FilterConfig{
		ContextLines: f.ContextLines,
		MaxOutput:    f.MaxOutput,
	}

	if f.ErrorPatterns != nil {
		clone.ErrorPatterns = make([]*RegexPattern, len(f.ErrorPatterns))
		for i, p := range f.ErrorPatterns {
			if p != nil {
				clone.ErrorPatterns[i] = &RegexPattern{
					Pattern: p.Pattern,
					Flags:   p.Flags,
				}
			}
		}
	}

	if f.IncludePatterns != nil {
		clone.IncludePatterns = make([]*RegexPattern, len(f.IncludePatterns))
		for i, p := range f.IncludePatterns {
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
