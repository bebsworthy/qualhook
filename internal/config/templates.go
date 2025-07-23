// Package config provides configuration templates management for qualhook
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bebsworthy/qualhook/internal/debug"
	pkgconfig "github.com/bebsworthy/qualhook/pkg/config"
)

// TemplateManager handles configuration template import/export
type TemplateManager struct {
	// Directory to store/load templates
	templateDir string
}

// NewTemplateManager creates a new template manager
func NewTemplateManager() *TemplateManager {
	// Default to ~/.qualhook/templates
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	
	return &TemplateManager{
		templateDir: filepath.Join(home, ".qualhook", "templates"),
	}
}

// SetTemplateDir sets a custom template directory
func (tm *TemplateManager) SetTemplateDir(dir string) {
	tm.templateDir = dir
}

// ExportTemplate exports a configuration as a reusable template
func (tm *TemplateManager) ExportTemplate(cfg *pkgconfig.Config, name, description string) error {
	debug.LogSection("Export Template")
	debug.Log("Exporting template: %s", name)
	
	// Validate name
	if name == "" {
		return fmt.Errorf("template name cannot be empty")
	}
	if strings.ContainsAny(name, "/\\:*?\"<>|") {
		return fmt.Errorf("template name contains invalid characters")
	}

	// Create template metadata
	template := &ConfigTemplate{
		Name:        name,
		Description: description,
		Version:     "1.0",
		Config:      cfg,
		Metadata: TemplateMetadata{
			CreatedAt:   time.Now().Format(time.RFC3339),
			QualhookMin: "0.1.0", // Minimum qualhook version
		},
	}

	// Ensure template directory exists
	if err := os.MkdirAll(tm.templateDir, 0750); err != nil {
		return fmt.Errorf("failed to create template directory: %w", err)
	}

	// Save template
	templatePath := filepath.Join(tm.templateDir, name+".json")
	data, err := json.MarshalIndent(template, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal template: %w", err)
	}

	if err := os.WriteFile(templatePath, data, 0600); err != nil {
		return fmt.Errorf("failed to write template file: %w", err)
	}

	debug.Log("Template exported to: %s", templatePath)
	return nil
}

// ImportTemplate imports a configuration template
func (tm *TemplateManager) ImportTemplate(nameOrPath string) (*pkgconfig.Config, error) {
	debug.LogSection("Import Template")
	debug.Log("Importing template: %s", nameOrPath)
	
	// Determine if it's a path or template name
	var templatePath string
	if strings.Contains(nameOrPath, string(os.PathSeparator)) || strings.HasSuffix(nameOrPath, ".json") {
		// It's a path
		templatePath = nameOrPath
	} else {
		// It's a template name, look in template directory
		templatePath = filepath.Join(tm.templateDir, nameOrPath+".json")
	}

	// Read template file
	// #nosec G304 - templatePath is constructed from controlled inputs
	data, err := os.ReadFile(templatePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("template not found: %s", nameOrPath)
		}
		return nil, fmt.Errorf("failed to read template: %w", err)
	}

	// Parse template
	var template ConfigTemplate
	if err := json.Unmarshal(data, &template); err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}

	debug.Log("Imported template: %s (version: %s)", template.Name, template.Version)
	
	// Validate config
	if template.Config == nil {
		return nil, fmt.Errorf("template contains no configuration")
	}

	return template.Config, nil
}

// ListTemplates lists available templates
func (tm *TemplateManager) ListTemplates() ([]TemplateInfo, error) {
	debug.LogSection("List Templates")
	
	// Check if template directory exists
	if _, err := os.Stat(tm.templateDir); os.IsNotExist(err) {
		debug.Log("Template directory does not exist: %s", tm.templateDir)
		return []TemplateInfo{}, nil
	}

	// Read directory
	entries, err := os.ReadDir(tm.templateDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read template directory: %w", err)
	}

	var templates []TemplateInfo
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		// Read template file to get info
		templatePath := filepath.Join(tm.templateDir, entry.Name())
		// #nosec G304 - templatePath is constructed from directory listing
		data, err := os.ReadFile(templatePath)
		if err != nil {
			debug.LogError(err, "reading template file")
			continue
		}

		var template ConfigTemplate
		if err := json.Unmarshal(data, &template); err != nil {
			debug.LogError(err, "parsing template file")
			continue
		}

		templates = append(templates, TemplateInfo{
			Name:        template.Name,
			Description: template.Description,
			Path:        templatePath,
			CreatedAt:   template.Metadata.CreatedAt,
		})
	}

	debug.Log("Found %d templates", len(templates))
	return templates, nil
}

// MergeConfigs merges two configurations, with the source overriding the target
func (tm *TemplateManager) MergeConfigs(target, source *pkgconfig.Config) *pkgconfig.Config {
	debug.LogSection("Merge Configurations")
	
	// Create a new config based on target
	merged := &pkgconfig.Config{
		Version:     source.Version, // Use source version
		ProjectType: source.ProjectType,
		Commands:    make(map[string]*pkgconfig.CommandConfig),
		Paths:       make([]*pkgconfig.PathConfig, 0),
	}

	// If target project type is empty, use source
	if merged.ProjectType == "" && target.ProjectType != "" {
		merged.ProjectType = target.ProjectType
	}

	// Copy target commands
	for name, cmd := range target.Commands {
		merged.Commands[name] = CloneCommandConfig(cmd)
	}

	// Override with source commands
	for name, cmd := range source.Commands {
		debug.Log("Merging command: %s", name)
		merged.Commands[name] = CloneCommandConfig(cmd)
	}

	// Merge paths (source paths take precedence)
	// First add source paths
	for _, path := range source.Paths {
		merged.Paths = append(merged.Paths, clonePathConfig(path))
	}

	// Then add target paths that don't conflict
	for _, targetPath := range target.Paths {
		conflict := false
		for _, sourcePath := range source.Paths {
			if targetPath.Path == sourcePath.Path {
				conflict = true
				break
			}
		}
		if !conflict {
			merged.Paths = append(merged.Paths, clonePathConfig(targetPath))
		}
	}

	debug.Log("Merged config: %d commands, %d paths", len(merged.Commands), len(merged.Paths))
	return merged
}

// ValidateTemplate validates a template before export
func (tm *TemplateManager) ValidateTemplate(cfg *pkgconfig.Config) error {
	if cfg == nil {
		return fmt.Errorf("configuration is nil")
	}

	if cfg.Version == "" {
		return fmt.Errorf("configuration version is required")
	}

	if len(cfg.Commands) == 0 {
		return fmt.Errorf("configuration must have at least one command")
	}

	// Use the standard validator
	validator := NewValidator()
	return validator.Validate(cfg)
}

// ConfigTemplate represents a configuration template with metadata
type ConfigTemplate struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Version     string                 `json:"version"`
	Config      *pkgconfig.Config      `json:"config"`
	Metadata    TemplateMetadata       `json:"metadata"`
}

// TemplateMetadata contains template metadata
type TemplateMetadata struct {
	CreatedAt   string `json:"created_at"`
	QualhookMin string `json:"qualhook_min"` // Minimum qualhook version
}

// TemplateInfo provides basic information about a template
type TemplateInfo struct {
	Name        string
	Description string
	Path        string
	CreatedAt   string
}

// CloneCommandConfig creates a deep copy of CommandConfig
func CloneCommandConfig(c *pkgconfig.CommandConfig) *pkgconfig.CommandConfig {
	if c == nil {
		return nil
	}

	clone := &pkgconfig.CommandConfig{
		Command: c.Command,
		Prompt:  c.Prompt,
		Timeout: c.Timeout,
	}

	// Clone args
	if c.Args != nil {
		clone.Args = make([]string, len(c.Args))
		copy(clone.Args, c.Args)
	}

	// Clone error detection
	if c.ErrorDetection != nil {
		clone.ErrorDetection = &pkgconfig.ErrorDetection{}
		if c.ErrorDetection.ExitCodes != nil {
			clone.ErrorDetection.ExitCodes = make([]int, len(c.ErrorDetection.ExitCodes))
			copy(clone.ErrorDetection.ExitCodes, c.ErrorDetection.ExitCodes)
		}
		if c.ErrorDetection.Patterns != nil {
			clone.ErrorDetection.Patterns = make([]*pkgconfig.RegexPattern, len(c.ErrorDetection.Patterns))
			for i, p := range c.ErrorDetection.Patterns {
				clone.ErrorDetection.Patterns[i] = &pkgconfig.RegexPattern{
					Pattern: p.Pattern,
					Flags:   p.Flags,
				}
			}
		}
	}

	// Clone output filter
	if c.OutputFilter != nil {
		clone.OutputFilter = &pkgconfig.FilterConfig{
			MaxOutput:    c.OutputFilter.MaxOutput,
			ContextLines: c.OutputFilter.ContextLines,
		}
		
		// Clone error patterns
		if c.OutputFilter.ErrorPatterns != nil {
			clone.OutputFilter.ErrorPatterns = make([]*pkgconfig.RegexPattern, len(c.OutputFilter.ErrorPatterns))
			for i, p := range c.OutputFilter.ErrorPatterns {
				clone.OutputFilter.ErrorPatterns[i] = &pkgconfig.RegexPattern{
					Pattern: p.Pattern,
					Flags:   p.Flags,
				}
			}
		}
		
		// Clone include patterns
		if c.OutputFilter.IncludePatterns != nil {
			clone.OutputFilter.IncludePatterns = make([]*pkgconfig.RegexPattern, len(c.OutputFilter.IncludePatterns))
			for i, p := range c.OutputFilter.IncludePatterns {
				clone.OutputFilter.IncludePatterns[i] = &pkgconfig.RegexPattern{
					Pattern: p.Pattern,
					Flags:   p.Flags,
				}
			}
		}
	}

	return clone
}

// clonePathConfig creates a deep copy of PathConfig
func clonePathConfig(p *pkgconfig.PathConfig) *pkgconfig.PathConfig {
	if p == nil {
		return nil
	}

	clone := &pkgconfig.PathConfig{
		Path:     p.Path,
		Extends:  p.Extends,
		Commands: make(map[string]*pkgconfig.CommandConfig),
	}

	// Clone commands
	for name, cmd := range p.Commands {
		clone.Commands[name] = CloneCommandConfig(cmd)
	}

	return clone
}
