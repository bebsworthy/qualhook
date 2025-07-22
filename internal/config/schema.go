// Package config provides schema versioning for qualhook configurations
package config

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/qualhook/qualhook/internal/debug"
	pkgconfig "github.com/qualhook/qualhook/pkg/config"
)

// CurrentSchemaVersion is the current configuration schema version
const CurrentSchemaVersion = "1.0"

// SchemaVersioner handles configuration schema versioning and migration
type SchemaVersioner struct {
	// Map of version to migration function
	migrations map[string]MigrationFunc
}

// MigrationFunc migrates a configuration from one version to another
type MigrationFunc func(cfg *pkgconfig.Config) (*pkgconfig.Config, error)

// NewSchemaVersioner creates a new schema versioner
func NewSchemaVersioner() *SchemaVersioner {
	sv := &SchemaVersioner{
		migrations: make(map[string]MigrationFunc),
	}
	
	// Register migrations here as schema evolves
	// Example: sv.RegisterMigration("0.9", "1.0", migrateV0_9ToV1_0)
	
	return sv
}

// RegisterMigration registers a migration function for a version transition
func (sv *SchemaVersioner) RegisterMigration(fromVersion, toVersion string, fn MigrationFunc) {
	key := fromVersion + "->" + toVersion
	sv.migrations[key] = fn
}

// ValidateVersion checks if a configuration version is valid
func (sv *SchemaVersioner) ValidateVersion(version string) error {
	if version == "" {
		return fmt.Errorf("configuration version is required")
	}
	
	// Parse version
	major, minor, err := parseVersion(version)
	if err != nil {
		return fmt.Errorf("invalid version format: %w", err)
	}
	
	// Parse current version
	currentMajor, currentMinor, _ := parseVersion(CurrentSchemaVersion)
	
	// Check if version is from the future
	if major > currentMajor || (major == currentMajor && minor > currentMinor) {
		return fmt.Errorf("configuration version %s is newer than supported version %s", version, CurrentSchemaVersion)
	}
	
	return nil
}

// MigrateConfig migrates a configuration to the current schema version
func (sv *SchemaVersioner) MigrateConfig(cfg *pkgconfig.Config) (*pkgconfig.Config, error) {
	debug.LogSection("Schema Migration")
	debug.Log("Current config version: %s", cfg.Version)
	debug.Log("Target version: %s", CurrentSchemaVersion)
	
	if cfg.Version == CurrentSchemaVersion {
		debug.Log("Configuration is already at current version")
		return cfg, nil
	}
	
	// Determine migration path
	path, err := sv.findMigrationPath(cfg.Version, CurrentSchemaVersion)
	if err != nil {
		return nil, err
	}
	
	if len(path) == 0 {
		// No migration needed, just update version
		cfg.Version = CurrentSchemaVersion
		return cfg, nil
	}
	
	// Apply migrations in sequence
	result := cfg
	for i := 0; i < len(path)-1; i++ {
		fromVer := path[i]
		toVer := path[i+1]
		key := fromVer + "->" + toVer
		
		migrationFn, exists := sv.migrations[key]
		if !exists {
			// No specific migration, just update version
			debug.Log("No migration needed from %s to %s", fromVer, toVer)
			result.Version = toVer
			continue
		}
		
		debug.Log("Applying migration from %s to %s", fromVer, toVer)
		result, err = migrationFn(result)
		if err != nil {
			return nil, fmt.Errorf("migration from %s to %s failed: %w", fromVer, toVer, err)
		}
		result.Version = toVer
	}
	
	debug.Log("Migration complete. New version: %s", result.Version)
	return result, nil
}

// findMigrationPath finds the migration path between two versions
func (sv *SchemaVersioner) findMigrationPath(from, to string) ([]string, error) {
	if from == to {
		return []string{}, nil
	}
	
	// For now, we support simple linear versioning
	// In the future, this could use graph traversal for complex migrations
	fromMajor, fromMinor, err := parseVersion(from)
	if err != nil {
		return nil, fmt.Errorf("invalid from version: %w", err)
	}
	
	toMajor, toMinor, err := parseVersion(to)
	if err != nil {
		return nil, fmt.Errorf("invalid to version: %w", err)
	}
	
	// Build path
	var path []string
	
	// Start with current version
	path = append(path, from)
	
	// Simple linear progression
	currentMajor, currentMinor := fromMajor, fromMinor
	
	for currentMajor < toMajor || (currentMajor == toMajor && currentMinor < toMinor) {
		if currentMinor < 9 {
			currentMinor++
		} else {
			currentMajor++
			currentMinor = 0
		}
		
		version := fmt.Sprintf("%d.%d", currentMajor, currentMinor)
		path = append(path, version)
		
		if currentMajor == toMajor && currentMinor == toMinor {
			break
		}
	}
	
	return path, nil
}

// parseVersion parses a version string into major and minor components
func parseVersion(version string) (int, int, error) {
	parts := strings.Split(version, ".")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("version must be in format X.Y")
	}
	
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid major version: %w", err)
	}
	
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid minor version: %w", err)
	}
	
	return major, minor, nil
}

// Example migration functions (to be implemented as schema evolves)

/*
// Example: Migration from 0.9 to 1.0
func migrateV0_9ToV1_0(cfg *pkgconfig.Config) (*pkgconfig.Config, error) {
	// Perform migration logic
	// For example, rename fields, change structure, etc.
	
	// Update version
	cfg.Version = "1.0"
	
	return cfg, nil
}
*/