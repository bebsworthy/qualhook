// Package main provides template management commands for qualhook
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"github.com/bebsworthy/qualhook/internal/config"
	pkgconfig "github.com/bebsworthy/qualhook/pkg/config"
)

var (
	templateName        string
	templateDescription string
	templateDir         string
	mergeFlag           bool
)

// templateCmd represents the template command
var templateCmd = &cobra.Command{
	Use:   "template",
	Short: "Manage configuration templates",
	Long: `Manage configuration templates for easy sharing and reuse.

Templates allow you to export your configuration for reuse in other projects
or share with your team. You can also import templates and merge them with
existing configurations.`,
}

// exportCmd exports current configuration as a template
var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export current configuration as a template",
	Long: `Export the current configuration as a reusable template.

Examples:
  # Export current config as a template
  qualhook template export --name myproject-config

  # Export with description
  qualhook template export --name nodejs-eslint --description "Node.js with ESLint and Prettier"

  # Export to custom directory
  qualhook template export --name team-config --dir ./templates`,
	RunE: runExportTemplate,
}

// importCmd imports a configuration template
var importCmd = &cobra.Command{
	Use:   "import [template-name or path]",
	Short: "Import a configuration template",
	Long: `Import a configuration template by name or path.

Examples:
  # Import a template by name
  qualhook template import nodejs-eslint

  # Import from a file path
  qualhook template import ./templates/team-config.json

  # Import and merge with existing config
  qualhook template import nodejs-eslint --merge`,
	Args: cobra.ExactArgs(1),
	RunE: runImportTemplate,
}

// listCmd lists available templates
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List available templates",
	Long: `List all available configuration templates.

Examples:
  # List all templates
  qualhook template list

  # List templates from custom directory
  qualhook template list --dir ./templates`,
	RunE: runListTemplates,
}

func init() {
	// Add template command to root
	rootCmd.AddCommand(templateCmd)
	
	// Add subcommands
	templateCmd.AddCommand(exportCmd)
	templateCmd.AddCommand(importCmd)
	templateCmd.AddCommand(listCmd)
	
	// Export flags
	exportCmd.Flags().StringVar(&templateName, "name", "", "Template name (required)")
	exportCmd.Flags().StringVar(&templateDescription, "description", "", "Template description")
	exportCmd.Flags().StringVar(&templateDir, "dir", "", "Directory to export template to")
	if err := exportCmd.MarkFlagRequired("name"); err != nil {
		// This is a programming error and should never happen
		panic(fmt.Sprintf("failed to mark 'name' flag as required: %v", err))
	}
	
	// Import flags
	importCmd.Flags().BoolVar(&mergeFlag, "merge", false, "Merge with existing configuration")
	importCmd.Flags().StringVar(&templateDir, "dir", "", "Directory to import template from")
	importCmd.Flags().BoolVar(&forceFlag, "force", false, "Force overwrite existing configuration")
	
	// List flags
	listCmd.Flags().StringVar(&templateDir, "dir", "", "Directory to list templates from")
}

func runExportTemplate(cmd *cobra.Command, args []string) error {
	// Load current configuration
	loader := config.NewLoader()
	var cfg *pkgconfig.Config
	var err error
	
	if configPath != "" {
		cfg, err = loader.LoadFromPath(configPath)
	} else {
		cfg, err = loader.Load()
	}
	
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}
	
	// Create template manager
	tm := config.NewTemplateManager()
	if templateDir != "" {
		tm.SetTemplateDir(templateDir)
	}
	
	// Validate template
	if err := tm.ValidateTemplate(cfg); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}
	
	// Export template
	if err := tm.ExportTemplate(cfg, templateName, templateDescription); err != nil {
		return fmt.Errorf("failed to export template: %w", err)
	}
	
	fmt.Printf("✅ Template '%s' exported successfully!\n", templateName)
	
	return nil
}

func runImportTemplate(cmd *cobra.Command, args []string) error {
	templateNameOrPath := args[0]
	
	// Create template manager
	tm := config.NewTemplateManager()
	if templateDir != "" {
		tm.SetTemplateDir(templateDir)
	}
	
	// Import template
	importedCfg, err := tm.ImportTemplate(templateNameOrPath)
	if err != nil {
		return fmt.Errorf("failed to import template: %w", err)
	}
	
	// Determine output path
	outputPath := configPath
	if outputPath == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
		outputPath = filepath.Join(cwd, config.ConfigFileName)
	}
	
	// Handle merge if requested
	var finalCfg *pkgconfig.Config
	if mergeFlag {
		// Load existing configuration
		loader := config.NewLoader()
		existingCfg, err := loader.LoadFromPath(outputPath)
		if err != nil {
			if !os.IsNotExist(err) {
				return fmt.Errorf("failed to load existing configuration: %w", err)
			}
			// No existing config, just use imported
			finalCfg = importedCfg
		} else {
			// Merge configurations
			finalCfg = tm.MergeConfigs(existingCfg, importedCfg)
			fmt.Println("📋 Merging with existing configuration...")
		}
	} else {
		// Check if config exists
		if _, err := os.Stat(outputPath); err == nil && !forceFlag {
			return fmt.Errorf("configuration already exists at %s. Use --force to overwrite or --merge to merge", outputPath)
		}
		finalCfg = importedCfg
	}
	
	// Validate final configuration
	validator := config.NewValidator()
	if err := validator.Validate(finalCfg); err != nil {
		fmt.Printf("⚠️  Configuration validation warning: %v\n", err)
	}
	
	// Save configuration
	data, err := pkgconfig.SaveConfig(finalCfg)
	if err != nil {
		return fmt.Errorf("failed to serialize configuration: %w", err)
	}
	
	if err := os.WriteFile(outputPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write configuration file: %w", err)
	}
	
	fmt.Printf("✅ Template imported successfully to: %s\n", outputPath)
	
	// Show summary
	fmt.Printf("\n📋 Configuration Summary:\n")
	fmt.Printf("   Version: %s\n", finalCfg.Version)
	if finalCfg.ProjectType != "" {
		fmt.Printf("   Project Type: %s\n", finalCfg.ProjectType)
	}
	fmt.Printf("   Commands: %d configured\n", len(finalCfg.Commands))
	if len(finalCfg.Paths) > 0 {
		fmt.Printf("   Monorepo Paths: %d configured\n", len(finalCfg.Paths))
	}
	
	return nil
}

func runListTemplates(cmd *cobra.Command, args []string) error {
	// Create template manager
	tm := config.NewTemplateManager()
	if templateDir != "" {
		tm.SetTemplateDir(templateDir)
	}
	
	// List templates
	templates, err := tm.ListTemplates()
	if err != nil {
		return fmt.Errorf("failed to list templates: %w", err)
	}
	
	if len(templates) == 0 {
		fmt.Println("No templates found.")
		fmt.Println("\nCreate a template with: qualhook template export --name <name>")
		return nil
	}
	
	// Display templates in a table
	fmt.Printf("📋 Available templates (%d):\n\n", len(templates))
	
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "NAME\tDESCRIPTION\tCREATED") //nolint:errcheck // Best effort table output
	_, _ = fmt.Fprintln(w, "----\t-----------\t-------") //nolint:errcheck // Best effort table output
	
	for _, tmpl := range templates {
		desc := tmpl.Description
		if desc == "" {
			desc = "-"
		}
		if len(desc) > 50 {
			desc = desc[:47] + "..."
		}
		
		created := tmpl.CreatedAt
		if created != "" {
			// Parse and format the date
			if t, err := time.Parse(time.RFC3339, created); err == nil {
				created = t.Format("2006-01-02")
			}
		}
		
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", tmpl.Name, desc, created) //nolint:errcheck // Best effort table output
	}
	
	if err := w.Flush(); err != nil {
		return fmt.Errorf("failed to flush table output: %w", err)
	}
	
	fmt.Println("\nImport a template with: qualhook template import <name>")
	
	return nil
}