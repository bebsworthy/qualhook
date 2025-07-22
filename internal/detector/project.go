// Package detector provides project type detection functionality for qualhook
package detector

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/qualhook/qualhook/internal/debug"
)

// ProjectType represents a detected project type with confidence score
type ProjectType struct {
	Name       string   // Project type name (e.g., "nodejs", "go", "rust")
	Confidence float64  // Confidence score between 0 and 1
	Markers    []string // Files that identified this type
}

// ProjectDetector handles project type detection
type ProjectDetector struct {
	// MarkerFiles maps project types to their characteristic files
	markerFiles map[string][]markerFile
}

// markerFile represents a file marker with its weight
type markerFile struct {
	name   string
	weight float64
}

// New creates a new ProjectDetector with default marker configurations
func New() *ProjectDetector {
	return &ProjectDetector{
		markerFiles: map[string][]markerFile{
			"nodejs": {
				{name: "package.json", weight: 1.0},
				{name: "package-lock.json", weight: 0.5},
				{name: "yarn.lock", weight: 0.5},
				{name: "pnpm-lock.yaml", weight: 0.5},
				{name: "node_modules", weight: 0.3},
				{name: ".nvmrc", weight: 0.2},
				{name: "tsconfig.json", weight: 0.4},
				{name: "jsconfig.json", weight: 0.4},
			},
			"go": {
				{name: "go.mod", weight: 1.0},
				{name: "go.sum", weight: 0.7},
				{name: "main.go", weight: 0.4},
				{name: "go.work", weight: 0.8},
				{name: "go.work.sum", weight: 0.6},
			},
			"rust": {
				{name: "Cargo.toml", weight: 1.0},
				{name: "Cargo.lock", weight: 0.7},
				{name: "rust-toolchain", weight: 0.5},
				{name: "rust-toolchain.toml", weight: 0.5},
				{name: ".cargo", weight: 0.4},
			},
			"python": {
				{name: "setup.py", weight: 0.8},
				{name: "setup.cfg", weight: 0.7},
				{name: "pyproject.toml", weight: 1.0},
				{name: "requirements.txt", weight: 0.6},
				{name: "Pipfile", weight: 0.8},
				{name: "Pipfile.lock", weight: 0.6},
				{name: "poetry.lock", weight: 0.8},
				{name: "tox.ini", weight: 0.5},
				{name: ".python-version", weight: 0.3},
				{name: "manage.py", weight: 0.7}, // Django
			},
			"java": {
				{name: "pom.xml", weight: 1.0},      // Maven
				{name: "build.gradle", weight: 1.0}, // Gradle
				{name: "build.gradle.kts", weight: 1.0},
				{name: "settings.gradle", weight: 0.7},
				{name: "settings.gradle.kts", weight: 0.7},
				{name: "gradlew", weight: 0.5},
				{name: ".mvn", weight: 0.4},
			},
			"ruby": {
				{name: "Gemfile", weight: 1.0},
				{name: "Gemfile.lock", weight: 0.7},
				{name: "Rakefile", weight: 0.6},
				{name: ".ruby-version", weight: 0.4},
				{name: ".rvmrc", weight: 0.3},
				{name: "config.ru", weight: 0.5}, // Rack
			},
			"php": {
				{name: "composer.json", weight: 1.0},
				{name: "composer.lock", weight: 0.7},
				{name: "artisan", weight: 0.8}, // Laravel
				{name: ".php-version", weight: 0.3},
			},
			"dotnet": {
				{name: "*.csproj", weight: 1.0},
				{name: "*.fsproj", weight: 1.0},
				{name: "*.vbproj", weight: 1.0},
				{name: "*.sln", weight: 0.9},
				{name: "global.json", weight: 0.6},
				{name: "nuget.config", weight: 0.5},
			},
		},
	}
}

// Detect scans the given path for project type indicators
func (d *ProjectDetector) Detect(path string) ([]ProjectType, error) {
	debug.LogSection("Project Detection")
	debug.Log("Scanning path: %s", path)
	
	// Validate path
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("invalid path: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("path must be a directory")
	}

	// Scan for markers
	projectScores := make(map[string]*projectScore)
	
	err = d.scanDirectory(path, projectScores)
	if err != nil {
		return nil, fmt.Errorf("error scanning directory: %w", err)
	}

	// Convert scores to ProjectType slice
	var results []ProjectType
	for projectName, score := range projectScores {
		if score.totalWeight > 0 {
			confidence := score.score / score.maxPossibleScore()
			results = append(results, ProjectType{
				Name:       projectName,
				Confidence: confidence,
				Markers:    score.foundMarkers,
			})
		}
	}

	// Sort by confidence (highest first)
	sort.Slice(results, func(i, j int) bool {
		return results[i].Confidence > results[j].Confidence
	})

	debug.Log("Detected %d project types", len(results))
	for _, result := range results {
		debug.Log("  %s (confidence: %.2f%%, markers: %v)", 
			result.Name, result.Confidence*100, result.Markers)
	}

	return results, nil
}

// projectScore tracks scoring for a project type
type projectScore struct {
	score        float64
	totalWeight  float64
	foundMarkers []string
	markerFiles  []markerFile
}

func (p *projectScore) maxPossibleScore() float64 {
	max := 0.0
	for _, marker := range p.markerFiles {
		max += marker.weight
	}
	return max
}

// scanDirectory scans a directory for marker files
func (d *ProjectDetector) scanDirectory(dir string, scores map[string]*projectScore) error {
	// Initialize scores for all project types
	for projectType, markers := range d.markerFiles {
		if _, exists := scores[projectType]; !exists {
			scores[projectType] = &projectScore{
				markerFiles: markers,
			}
		}
	}

	// Read directory contents
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	// Check each entry against markers
	for _, entry := range entries {
		name := entry.Name()
		
		// Skip hidden directories (except specific ones we're looking for)
		if strings.HasPrefix(name, ".") && entry.IsDir() {
			if !d.isRelevantHiddenDir(name) {
				continue
			}
		}

		// Check against each project type's markers
		for projectType, score := range scores {
			for _, marker := range d.markerFiles[projectType] {
				if d.matchesMarker(name, marker.name) {
					debug.Log("Found marker %q for %s (weight: %.1f)", name, projectType, marker.weight)
					score.score += marker.weight
					score.totalWeight += marker.weight
					score.foundMarkers = append(score.foundMarkers, name)
					break // Don't count the same file multiple times for one project type
				}
			}
		}
	}

	return nil
}

// matchesMarker checks if a filename matches a marker pattern
func (d *ProjectDetector) matchesMarker(filename, pattern string) bool {
	// Handle glob patterns
	if strings.Contains(pattern, "*") {
		matched, _ := filepath.Match(pattern, filename)
		return matched
	}
	// Exact match
	return filename == pattern
}

// isRelevantHiddenDir checks if a hidden directory is relevant for detection
func (d *ProjectDetector) isRelevantHiddenDir(name string) bool {
	relevantDirs := []string{".cargo", ".mvn", ".python-version", ".ruby-version", ".php-version", ".nvmrc", ".rvmrc"}
	for _, dir := range relevantDirs {
		if name == dir {
			return true
		}
	}
	return false
}

// DetectMonorepo checks for common monorepo patterns
func (d *ProjectDetector) DetectMonorepo(path string) (*MonorepoInfo, error) {
	info := &MonorepoInfo{
		Path:        path,
		IsMonorepo:  false,
		Type:        "",
		Workspaces:  []string{},
		SubProjects: make(map[string][]ProjectType),
	}

	// Check for monorepo indicator files
	monorepoMarkers := map[string]string{
		"lerna.json":             "lerna",
		"nx.json":                "nx",
		"rush.json":              "rush",
		"pnpm-workspace.yaml":    "pnpm",
		"yarn.lock":              "yarn-workspaces", // Need to check package.json for workspaces
		"turbo.json":             "turborepo",
		"workspace.json":         "nx-legacy",
		".yarnrc.yml":            "yarn-berry",
		"WORKSPACE":              "bazel",
		"WORKSPACE.bazel":        "bazel",
	}

	for marker, repoType := range monorepoMarkers {
		markerPath := filepath.Join(path, marker)
		if _, err := os.Stat(markerPath); err == nil {
			info.IsMonorepo = true
			info.Type = repoType
			
			// Special case: check package.json for yarn workspaces
			if repoType == "yarn-workspaces" {
				if hasYarnWorkspaces := d.checkYarnWorkspaces(path); !hasYarnWorkspaces {
					info.IsMonorepo = false
					info.Type = ""
					continue
				}
			}
			break
		}
	}

	// Check for Go workspace
	if _, err := os.Stat(filepath.Join(path, "go.work")); err == nil {
		info.IsMonorepo = true
		info.Type = "go-workspace"
	}

	// If monorepo detected, scan for workspaces
	if info.IsMonorepo {
		if err := d.scanWorkspaces(info); err != nil {
			return info, fmt.Errorf("error scanning workspaces: %w", err)
		}
	}

	return info, nil
}

// MonorepoInfo contains information about a detected monorepo
type MonorepoInfo struct {
	Path        string
	IsMonorepo  bool
	Type        string
	Workspaces  []string
	SubProjects map[string][]ProjectType // Maps workspace path to detected project types
}

// checkYarnWorkspaces checks if package.json contains workspaces field
func (d *ProjectDetector) checkYarnWorkspaces(path string) bool {
	packageJSON := filepath.Join(path, "package.json")
	data, err := os.ReadFile(packageJSON)
	if err != nil {
		return false
	}
	
	// Simple check for workspaces field
	return strings.Contains(string(data), `"workspaces"`)
}

// scanWorkspaces scans for workspace directories in a monorepo
func (d *ProjectDetector) scanWorkspaces(info *MonorepoInfo) error {
	// Common workspace patterns
	workspacePatterns := []string{
		"packages/*",
		"apps/*",
		"services/*",
		"libs/*",
		"modules/*",
		"projects/*",
	}

	// Scan each pattern
	for _, pattern := range workspacePatterns {
		matches, err := filepath.Glob(filepath.Join(info.Path, pattern))
		if err != nil {
			continue
		}

		for _, match := range matches {
			// Check if it's a directory
			stat, err := os.Stat(match)
			if err != nil || !stat.IsDir() {
				continue
			}

			// Get relative path
			relPath, err := filepath.Rel(info.Path, match)
			if err != nil {
				continue
			}

			info.Workspaces = append(info.Workspaces, relPath)

			// Detect project types in this workspace
			projects, err := d.Detect(match)
			if err == nil && len(projects) > 0 {
				info.SubProjects[relPath] = projects
			}
		}
	}

	// Sort workspaces for consistent output
	sort.Strings(info.Workspaces)

	return nil
}

// GetDefaultConfigName returns the default configuration name for a project type
func GetDefaultConfigName(projectType string) string {
	// Map project types to config file names
	configNames := map[string]string{
		"nodejs": "nodejs.json",
		"go":     "golang.json",
		"rust":   "rust.json",
		"python": "python.json",
		"java":   "java.json",
		"ruby":   "ruby.json",
		"php":    "php.json",
		"dotnet": "dotnet.json",
	}

	if name, exists := configNames[projectType]; exists {
		return name
	}
	return ""
}