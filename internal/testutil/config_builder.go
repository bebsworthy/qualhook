// Package testutil provides common test utilities and helpers for the qualhook test suite.
package testutil

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/bebsworthy/qualhook/pkg/config"
)

// ConfigBuilder provides a fluent interface for building test configurations.
type ConfigBuilder struct {
	config *config.Config
}

// NewConfigBuilder creates a new ConfigBuilder with default test configuration.
func NewConfigBuilder() *ConfigBuilder {
	return &ConfigBuilder{
		config: &config.Config{
			Version:  "1.0",
			Commands: make(map[string]*config.CommandConfig),
			Paths:    []*config.PathConfig{},
		},
	}
}

// WithVersion sets the configuration version.
func (b *ConfigBuilder) WithVersion(version string) *ConfigBuilder {
	b.config.Version = version
	return b
}

// WithCommand adds a command configuration.
func (b *ConfigBuilder) WithCommand(name string, cmd *config.CommandConfig) *ConfigBuilder {
	if b.config.Commands == nil {
		b.config.Commands = make(map[string]*config.CommandConfig)
	}
	b.config.Commands[name] = cmd
	return b
}

// WithSimpleCommand adds a simple command configuration with common defaults.
func (b *ConfigBuilder) WithSimpleCommand(name, command string, args ...string) *ConfigBuilder {
	return b.WithCommand(name, &config.CommandConfig{
		Command:       command,
		Args:          args,
		ExitCodes:     []int{1},
		ErrorPatterns: []*config.RegexPattern{{Pattern: "error", Flags: "i"}},
		MaxOutput:     100,
	})
}

// WithPath adds a path-specific configuration.
func (b *ConfigBuilder) WithPath(pathConfig *config.PathConfig) *ConfigBuilder {
	b.config.Paths = append(b.config.Paths, pathConfig)
	return b
}

// WithPathCommand adds a path-specific command configuration.
func (b *ConfigBuilder) WithPathCommand(pattern string, commands map[string]*config.CommandConfig) *ConfigBuilder {
	return b.WithPath(&config.PathConfig{
		Path:     pattern,
		Commands: commands,
	})
}

// Build returns the constructed configuration.
func (b *ConfigBuilder) Build() *config.Config {
	return b.config
}

// WriteToFile writes the configuration to a JSON file.
func (b *ConfigBuilder) WriteToFile(path string) error {
	data, err := json.MarshalIndent(b.config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

// DefaultTestConfig returns a basic test configuration with common commands.
func DefaultTestConfig() *config.Config {
	return NewConfigBuilder().
		WithSimpleCommand("lint", "echo", "Linting...").
		WithSimpleCommand("format", "echo", "Formatting...").
		WithSimpleCommand("test", "echo", "Testing...").
		WithSimpleCommand("typecheck", "echo", "Type checking...").
		Build()
}

// SafeCommandConfig returns a CommandConfig using safe test commands.
func SafeCommandConfig(args ...string) *config.CommandConfig {
	return &config.CommandConfig{
		Command:       "echo",
		Args:          args,
		ExitCodes:     []int{1},
		ErrorPatterns: []*config.RegexPattern{{Pattern: "error", Flags: "i"}},
		MaxOutput:     100,
		Timeout:       5000, // 5 seconds default timeout for tests
	}
}

// FailingCommandConfig returns a CommandConfig that will fail with exit code 1.
func FailingCommandConfig() *config.CommandConfig {
	return &config.CommandConfig{
		Command:   "sh",
		Args:      []string{"-c", "echo 'error: test failure' >&2; exit 1"},
		ExitCodes: []int{0}, // Expecting 0, but will get 1
		ErrorPatterns: []*config.RegexPattern{
			{Pattern: "error", Flags: "i"},
		},
		MaxOutput: 100,
		Timeout:   5000,
	}
}

// TimeoutCommandConfig returns a CommandConfig that will timeout.
func TimeoutCommandConfig() *config.CommandConfig {
	return &config.CommandConfig{
		Command:   "sleep",
		Args:      []string{"10"}, // Sleep for 10 seconds
		ExitCodes: []int{0},
		Timeout:   100, // But timeout after 100ms
		MaxOutput: 100,
	}
}

// CreateTestConfigFile creates a temporary config file and returns its path.
func CreateTestConfigFile(dir string, cfg *config.Config) (string, error) {
	if cfg == nil {
		cfg = DefaultTestConfig()
	}

	configPath := filepath.Join(dir, ".qualhook.json")
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return "", err
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return "", err
	}

	return configPath, nil
}
