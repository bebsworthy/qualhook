// Package watcher provides file mapping functionality for monorepo support in qualhook.
package watcher

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bebsworthy/qualhook/pkg/config"
	"github.com/bmatcuk/doublestar/v4"
)

// ComponentGroup represents a group of files mapped to a component with its configuration
type ComponentGroup struct {
	// Path is the configuration path pattern that matched
	Path string
	// Files are the files that belong to this component
	Files []string
	// Config is the merged configuration for this component
	Config map[string]*config.CommandConfig
}

// FileMapper maps files to their corresponding component configurations
type FileMapper struct {
	// rootConfig is the base configuration
	rootConfig *config.Config
}

// NewFileMapper creates a new file mapper
func NewFileMapper(cfg *config.Config) *FileMapper {
	return &FileMapper{
		rootConfig: cfg,
	}
}

// MapFilesToComponents maps a list of file paths to their component groups
func (m *FileMapper) MapFilesToComponents(files []string) ([]ComponentGroup, error) {
	if len(files) == 0 {
		return nil, nil
	}

	// If no paths are configured, all files go to root
	if len(m.rootConfig.Paths) == 0 {
		return []ComponentGroup{
			{
				Path:   ".",
				Files:  files,
				Config: m.rootConfig.Commands,
			},
		}, nil
	}

	// First, determine which path config each file belongs to
	fileToPath := make(map[string]*config.PathConfig)
	fileToPathPattern := make(map[string]string)

	for _, file := range files {
		// Clean and normalize the file path
		cleanFile := filepath.Clean(file)

		// Find the most specific matching path config
		var bestMatch *config.PathConfig
		var bestPattern string
		bestSpecificity := -1

		// Check each path configuration
		for _, pathConfig := range m.rootConfig.Paths {
			if match, specificity := m.matchesPath(cleanFile, pathConfig.Path); match {
				if specificity > bestSpecificity {
					bestMatch = pathConfig
					bestPattern = pathConfig.Path
					bestSpecificity = specificity
				}
			}
		}

		if bestMatch != nil {
			fileToPath[cleanFile] = bestMatch
			fileToPathPattern[cleanFile] = bestPattern
		}
		// If no specific path matches, the file will use root config
	}

	// Group files by their component (path pattern)
	componentFiles := make(map[string][]string)
	componentConfigs := make(map[string]*config.PathConfig)

	// First, add files that match specific paths
	for file, pathConfig := range fileToPath {
		pattern := fileToPathPattern[file]
		componentFiles[pattern] = append(componentFiles[pattern], file)
		componentConfigs[pattern] = pathConfig
	}

	// Then, add files that don't match any path to the root component
	rootFiles := []string{}
	for _, file := range files {
		cleanFile := filepath.Clean(file)
		if _, hasPath := fileToPath[cleanFile]; !hasPath {
			rootFiles = append(rootFiles, file)
		}
	}

	// Build component groups
	var groups []ComponentGroup

	// Add path-specific components
	for pattern, files := range componentFiles {
		pathConfig := componentConfigs[pattern]
		mergedConfig := m.mergeConfigs(pathConfig)

		groups = append(groups, ComponentGroup{
			Path:   pattern,
			Files:  files,
			Config: mergedConfig,
		})
	}

	// Add root component if there are files
	if len(rootFiles) > 0 {
		groups = append(groups, ComponentGroup{
			Path:   ".",
			Files:  rootFiles,
			Config: m.rootConfig.Commands,
		})
	}

	// Sort groups by path for consistent ordering
	sort.Slice(groups, func(i, j int) bool {
		return groups[i].Path < groups[j].Path
	})

	return groups, nil
}

// matchesPath checks if a file path matches a glob pattern and returns the specificity
func (m *FileMapper) matchesPath(filePath, pattern string) (bool, int) {
	// Normalize paths for comparison
	filePath = filepath.ToSlash(filepath.Clean(filePath))
	pattern = filepath.ToSlash(pattern)

	// Use doublestar for glob matching
	matched, err := doublestar.Match(pattern, filePath)
	if err != nil || !matched {
		return false, -1
	}

	// Calculate specificity based on pattern complexity
	specificity := calculateSpecificity(pattern)
	return true, specificity
}

// calculateSpecificity calculates how specific a pattern is
// More specific patterns have higher values
func calculateSpecificity(pattern string) int {
	specificity := 0

	// Count path separators (deeper paths are more specific)
	specificity += strings.Count(pattern, "/") * 10

	// Count non-wildcard characters
	for _, ch := range pattern {
		switch ch {
		case '*', '?', '[', ']':
			// Wildcards reduce specificity slightly
			specificity -= 1
		default:
			// Regular characters increase specificity
			specificity += 1
		}
	}

	// Patterns ending with /** are less specific
	if strings.HasSuffix(pattern, "/**") {
		specificity -= 20
	}

	// Patterns with ** in the middle are less specific
	if strings.Contains(pattern, "/**/") {
		specificity -= 10
	}

	// Ensure we don't return negative values for simple patterns
	if specificity < 0 {
		specificity = 0
	}

	return specificity
}

// mergeConfigs merges a path config with its extends base and root config
func (m *FileMapper) mergeConfigs(pathConfig *config.PathConfig) map[string]*config.CommandConfig {
	// Start with a copy of root commands
	merged := make(map[string]*config.CommandConfig)
	for name, cmd := range m.rootConfig.Commands {
		if cmd != nil {
			merged[name] = cmd.Clone()
		}
	}

	// If path extends another path, apply that first
	if pathConfig.Extends != "" {
		// Find the extended path config
		for _, extPath := range m.rootConfig.Paths {
			if extPath.Path == pathConfig.Extends {
				for name, cmd := range extPath.Commands {
					if cmd != nil {
						merged[name] = cmd.Clone()
					}
				}
				break
			}
		}
	}

	// Finally, apply the path-specific commands (these override everything)
	for name, cmd := range pathConfig.Commands {
		if cmd != nil {
			merged[name] = cmd.Clone()
		}
	}

	return merged
}

// GetComponentForFile returns the component group for a single file
func (m *FileMapper) GetComponentForFile(filePath string) (*ComponentGroup, error) {
	groups, err := m.MapFilesToComponents([]string{filePath})
	if err != nil {
		return nil, err
	}

	if len(groups) == 0 {
		return nil, fmt.Errorf("no component found for file: %s", filePath)
	}

	return &groups[0], nil
}

// ListAllComponents returns all configured components (paths + root)
func (m *FileMapper) ListAllComponents() []string {
	components := []string{"."}

	for _, pathConfig := range m.rootConfig.Paths {
		components = append(components, pathConfig.Path)
	}

	return components
}
