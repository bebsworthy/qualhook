// Package config provides default configuration templates for common project types
package config

import (
	_ "embed"
	"encoding/json"
	"fmt"

	"github.com/bebsworthy/qualhook/pkg/config"
)

// Embedded default configuration files
var (
	//go:embed defaults/nodejs.json
	defaultNodeJSConfig string

	//go:embed defaults/golang.json
	defaultGoConfig string

	//go:embed defaults/python.json
	defaultPythonConfig string

	//go:embed defaults/rust.json
	defaultRustConfig string
)

// ProjectType represents a supported project type
type ProjectType string

const (
	// ProjectTypeNodeJS represents a Node.js project
	ProjectTypeNodeJS ProjectType = "nodejs"

	// ProjectTypeGo represents a Go project
	ProjectTypeGo ProjectType = "go"

	// ProjectTypePython represents a Python project
	ProjectTypePython ProjectType = "python"

	// ProjectTypeRust represents a Rust project
	ProjectTypeRust ProjectType = "rust"

	// ProjectTypeUnknown represents an unknown project type
	ProjectTypeUnknown ProjectType = "unknown"
)

// DefaultConfigs provides access to default configuration templates
type DefaultConfigs struct {
	configs map[ProjectType]*config.Config
}

// NewDefaultConfigs creates a new instance with all default configurations loaded
func NewDefaultConfigs() (*DefaultConfigs, error) {
	dc := &DefaultConfigs{
		configs: make(map[ProjectType]*config.Config),
	}

	// Load all embedded configurations
	configs := map[ProjectType]string{
		ProjectTypeNodeJS: defaultNodeJSConfig,
		ProjectTypeGo:     defaultGoConfig,
		ProjectTypePython: defaultPythonConfig,
		ProjectTypeRust:   defaultRustConfig,
	}

	for projectType, configJSON := range configs {
		cfg, err := config.LoadConfig([]byte(configJSON))
		if err != nil {
			return nil, fmt.Errorf("failed to load default config for %s: %w", projectType, err)
		}
		dc.configs[projectType] = cfg
	}

	return dc, nil
}

// GetConfig returns the default configuration for a project type
func (dc *DefaultConfigs) GetConfig(projectType ProjectType) (*config.Config, error) {
	cfg, ok := dc.configs[projectType]
	if !ok {
		return nil, fmt.Errorf("no default configuration for project type: %s", projectType)
	}

	// Return a deep copy to prevent modification of the default
	return dc.cloneConfig(cfg), nil
}

// GetAllTypes returns all supported project types
func (dc *DefaultConfigs) GetAllTypes() []ProjectType {
	types := make([]ProjectType, 0, len(dc.configs))
	for t := range dc.configs {
		types = append(types, t)
	}
	return types
}

// GetCommonErrorPatterns returns common error patterns for a project type
func (dc *DefaultConfigs) GetCommonErrorPatterns(projectType ProjectType) ([]*config.RegexPattern, error) {
	cfg, err := dc.GetConfig(projectType)
	if err != nil {
		return nil, err
	}

	patterns := []*config.RegexPattern{}
	
	// Collect error patterns from all commands
	for _, cmd := range cfg.Commands {
		if cmd.OutputFilter != nil && cmd.OutputFilter.ErrorPatterns != nil {
			patterns = append(patterns, cmd.OutputFilter.ErrorPatterns...)
		}
		if cmd.ErrorDetection != nil && cmd.ErrorDetection.Patterns != nil {
			patterns = append(patterns, cmd.ErrorDetection.Patterns...)
		}
	}

	return patterns, nil
}

// MergeWithDefaults merges a user configuration with defaults for a project type
func (dc *DefaultConfigs) MergeWithDefaults(userConfig *config.Config, projectType ProjectType) (*config.Config, error) {
	defaultCfg, err := dc.GetConfig(projectType)
	if err != nil {
		// If no default for this type, just return the user config
		return userConfig, nil
	}

	// Start with a copy of the default
	merged := dc.cloneConfig(defaultCfg)

	// Override with user values
	if userConfig.Version != "" {
		merged.Version = userConfig.Version
	}
	if userConfig.ProjectType != "" {
		merged.ProjectType = userConfig.ProjectType
	}

	// Merge commands
	for name, cmd := range userConfig.Commands {
		merged.Commands[name] = cmd.Clone()
	}

	// Merge paths
	if len(userConfig.Paths) > 0 {
		merged.Paths = make([]*config.PathConfig, len(userConfig.Paths))
		for i, p := range userConfig.Paths {
			merged.Paths[i] = dc.clonePathConfig(p)
		}
	}

	return merged, nil
}

// cloneConfig creates a deep copy of a configuration
func (dc *DefaultConfigs) cloneConfig(cfg *config.Config) *config.Config {
	clone := &config.Config{
		Version:     cfg.Version,
		ProjectType: cfg.ProjectType,
		Commands:    make(map[string]*config.CommandConfig),
	}

	for name, cmd := range cfg.Commands {
		clone.Commands[name] = cmd.Clone()
	}

	if len(cfg.Paths) > 0 {
		clone.Paths = make([]*config.PathConfig, len(cfg.Paths))
		for i, p := range cfg.Paths {
			clone.Paths[i] = dc.clonePathConfig(p)
		}
	}

	return clone
}

// clonePathConfig creates a deep copy of a path configuration
func (dc *DefaultConfigs) clonePathConfig(p *config.PathConfig) *config.PathConfig {
	clone := &config.PathConfig{
		Path:     p.Path,
		Extends:  p.Extends,
		Commands: make(map[string]*config.CommandConfig),
	}

	for name, cmd := range p.Commands {
		clone.Commands[name] = cmd.Clone()
	}

	return clone
}

// DetectProjectType attempts to detect the project type from marker files
func DetectProjectType(markers []string) ProjectType {
	// Check for specific marker files
	for _, marker := range markers {
		switch marker {
		case "package.json", "package-lock.json", "yarn.lock", "pnpm-lock.yaml":
			return ProjectTypeNodeJS
		case "go.mod", "go.sum":
			return ProjectTypeGo
		case "requirements.txt", "setup.py", "Pipfile", "pyproject.toml":
			return ProjectTypePython
		case "Cargo.toml", "Cargo.lock":
			return ProjectTypeRust
		}
	}

	return ProjectTypeUnknown
}

// ExportTemplate exports a default configuration as formatted JSON
func (dc *DefaultConfigs) ExportTemplate(projectType ProjectType) ([]byte, error) {
	cfg, err := dc.GetConfig(projectType)
	if err != nil {
		return nil, err
	}

	return json.MarshalIndent(cfg, "", "  ")
}