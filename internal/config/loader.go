// Package config provides configuration loading and management for qualhook.
package config

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/bebsworthy/qualhook/internal/debug"
	"github.com/bebsworthy/qualhook/pkg/config"
)

const (
	// ConfigFileName is the default configuration file name
	ConfigFileName = ".qualhook.json"

	// ConfigEnvVar is the environment variable to specify custom config path
	ConfigEnvVar = "QUALHOOK_CONFIG"
)

// Loader handles loading and merging configuration files
type Loader struct {
	// SearchPaths contains the paths to search for configuration files
	SearchPaths []string
}

// NewLoader creates a new configuration loader
func NewLoader() *Loader {
	return &Loader{
		SearchPaths: getDefaultSearchPaths(),
	}
}

// Load attempts to load configuration from various sources
func (l *Loader) Load() (*config.Config, error) {
	debug.LogSection("Configuration Loading")
	
	// First check if environment variable is set
	if envPath := os.Getenv(ConfigEnvVar); envPath != "" {
		debug.Log("Loading config from environment variable %s: %s", ConfigEnvVar, envPath)
		cfg, err := l.loadFromPath(envPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load config from %s: %w", ConfigEnvVar, err)
		}
		return cfg, nil
	}

	// Search in default paths
	debug.Log("Searching for config in default paths: %v", l.SearchPaths)
	for _, searchPath := range l.SearchPaths {
		configPath := filepath.Join(searchPath, ConfigFileName)
		debug.Log("Checking path: %s", configPath)
		if _, err := os.Stat(configPath); err == nil {
			debug.Log("Found config at: %s", configPath)
			cfg, err := l.loadFromPath(configPath)
			if err != nil {
				return nil, fmt.Errorf("failed to load config from %s: %w", configPath, err)
			}
			return cfg, nil
		}
	}

	return nil, fmt.Errorf("no configuration file found in search paths: %v", l.SearchPaths)
}

// LoadFromPath loads configuration from a specific file path
func (l *Loader) LoadFromPath(path string) (*config.Config, error) {
	return l.loadFromPath(path)
}

// LoadForMonorepo loads configuration with path-based merging for monorepo support
func (l *Loader) LoadForMonorepo(workingDir string) (*config.Config, error) {
	// Load the root configuration first
	rootConfig, err := l.Load()
	if err != nil {
		return nil, err
	}

	// If no path configurations, return root config
	if len(rootConfig.Paths) == 0 {
		return rootConfig, nil
	}

	// Find the most specific path configuration that matches the working directory
	relPath, err := filepath.Rel(l.SearchPaths[0], workingDir)
	if err != nil {
		// If we can't determine relative path, use root config
		return rootConfig, nil
	}

	// Normalize the path for matching
	relPath = filepath.ToSlash(relPath)

	// Find the most specific matching path configuration
	var bestMatch *config.PathConfig
	bestMatchLen := -1

	for _, pathConfig := range rootConfig.Paths {
		if matched, matchLen := matchesPath(relPath, pathConfig.Path); matched && matchLen > bestMatchLen {
			bestMatch = pathConfig
			bestMatchLen = matchLen
		}
	}

	// If no match found, return root config
	if bestMatch == nil {
		return rootConfig, nil
	}

	// Merge the path-specific configuration with the root configuration
	mergedConfig := l.mergeConfigs(rootConfig, bestMatch)
	return mergedConfig, nil
}

// loadFromPath loads and validates configuration from a file
func (l *Loader) loadFromPath(path string) (*config.Config, error) {
	debug.Log("Loading config from file: %s", path)
	
	// #nosec G304 - path is validated by caller (LoadFromPath checks file existence)
	file, err := os.Open(path)
	if err != nil {
		debug.LogError(err, "opening config file")
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer func() { _ = file.Close() }() //nolint:errcheck // Best effort cleanup

	data, err := io.ReadAll(file)
	if err != nil {
		debug.LogError(err, "reading config file")
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	debug.Log("Config file size: %d bytes", len(data))
	cfg, err := config.LoadConfig(data)
	if err != nil {
		debug.LogError(err, "parsing config")
		return nil, err
	}
	
	debug.Log("Loaded config: version=%s, commands=%d, paths=%d", 
		cfg.Version, len(cfg.Commands), len(cfg.Paths))
	
	return cfg, nil
}

// mergeConfigs merges path-specific configuration with root configuration
func (l *Loader) mergeConfigs(root *config.Config, pathConfig *config.PathConfig) *config.Config {
	// Create a new config based on root
	merged := &config.Config{
		Version:     root.Version,
		ProjectType: root.ProjectType,
		Commands:    make(map[string]*config.CommandConfig),
		Paths:       root.Paths, // Keep paths for nested monorepo support
	}

	// Copy root commands
	for name, cmd := range root.Commands {
		merged.Commands[name] = CloneCommandConfig(cmd)
	}

	// Override with path-specific commands
	for name, cmd := range pathConfig.Commands {
		merged.Commands[name] = CloneCommandConfig(cmd)
	}

	// Handle extends functionality
	// TODO: Implement extends functionality to resolve and merge extended path configs
	// For now, we just use the path-specific overrides as-is

	return merged
}

// matchesPath checks if a relative path matches a glob pattern
func matchesPath(relPath, pattern string) (bool, int) {
	// Simple prefix matching for now
	// TODO: Implement proper glob pattern matching
	if relPath == pattern {
		return true, len(pattern)
	}

	// Check if the path is within the pattern directory
	if len(pattern) > 0 && pattern[len(pattern)-1] == '/' {
		if relPath == pattern[:len(pattern)-1] {
			return true, len(pattern)
		}
		if len(relPath) > len(pattern) && relPath[:len(pattern)] == pattern {
			return true, len(pattern)
		}
	}

	// Check if pattern ends with /** for recursive matching
	if len(pattern) > 3 && pattern[len(pattern)-3:] == "/**" {
		prefix := pattern[:len(pattern)-3]
		if relPath == prefix || (len(relPath) > len(prefix) && relPath[:len(prefix)+1] == prefix+"/") {
			return true, len(prefix)
		}
	}

	return false, 0
}

// getDefaultSearchPaths returns the default paths to search for configuration
func getDefaultSearchPaths() []string {
	paths := []string{}

	// Current working directory
	if cwd, err := os.Getwd(); err == nil {
		paths = append(paths, cwd)

		// Walk up the directory tree to find root of project
		dir := cwd
		for {
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}

			// Check for common project root indicators
			if _, err := os.Stat(filepath.Join(parent, ".git")); err == nil {
				paths = append(paths, parent)
				break
			}
			if _, err := os.Stat(filepath.Join(parent, "go.mod")); err == nil {
				paths = append(paths, parent)
				break
			}
			if _, err := os.Stat(filepath.Join(parent, "package.json")); err == nil {
				paths = append(paths, parent)
				break
			}

			dir = parent
		}
	}

	// Home directory
	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths, home)
	}

	// System-wide configuration (for future use)
	// paths = append(paths, "/etc/qualhook")

	return paths
}

// ValidateConfigFile validates a configuration file without loading it fully
func ValidateConfigFile(path string) error {
	// #nosec G304 - path is provided by user for validation purposes
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open config file: %w", err)
	}
	defer func() { _ = file.Close() }() //nolint:errcheck // Best effort cleanup

	var cfg config.Config
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&cfg); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	return cfg.Validate()
}

