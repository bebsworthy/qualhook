// Package wizard provides interactive configuration wizards for qualhook
package wizard

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/qualhook/qualhook/internal/config"
	"github.com/qualhook/qualhook/internal/debug"
	"github.com/qualhook/qualhook/internal/detector"
	pkgconfig "github.com/qualhook/qualhook/pkg/config"
)

// ConfigWizard provides an interactive configuration wizard
type ConfigWizard struct {
	projectDetector *detector.ProjectDetector
	defaults        *config.DefaultConfigs
}

// NewConfigWizard creates a new configuration wizard
func NewConfigWizard() (*ConfigWizard, error) {
	defaults, err := config.NewDefaultConfigs()
	if err != nil {
		return nil, fmt.Errorf("failed to load default configs: %w", err)
	}

	return &ConfigWizard{
		projectDetector: detector.New(),
		defaults:        defaults,
	}, nil
}

// Run runs the interactive configuration wizard
func (w *ConfigWizard) Run(outputPath string, force bool) error {
	debug.LogSection("Configuration Wizard")
	
	// Determine output path
	if outputPath == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
		outputPath = filepath.Join(cwd, config.ConfigFileName)
	}

	// Check if configuration already exists
	if _, err := os.Stat(outputPath); err == nil && !force {
		overwrite := false
		prompt := &survey.Confirm{
			Message: fmt.Sprintf("Configuration already exists at %s. Overwrite?", outputPath),
			Default: false,
		}
		if err := survey.AskOne(prompt, &overwrite); err != nil {
			return err
		}
		if !overwrite {
			fmt.Println("Configuration wizard cancelled.")
			return nil
		}
	}

	// Welcome message
	fmt.Println("🚀 Welcome to the qualhook configuration wizard!")
	fmt.Println("This wizard will help you set up qualhook for your project.\n")

	// Detect project type
	projectDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	detectedTypes, err := w.projectDetector.Detect(projectDir)
	if err != nil {
		debug.LogError(err, "detecting project type")
	}

	// Check for monorepo
	monorepoInfo, err := w.projectDetector.DetectMonorepo(projectDir)
	if err != nil {
		debug.LogError(err, "detecting monorepo")
	}

	// Display detection results
	if len(detectedTypes) > 0 {
		fmt.Println("📦 Detected project types:")
		for _, dt := range detectedTypes {
			fmt.Printf("   • %s (confidence: %.0f%%)\n", dt.Name, dt.Confidence*100)
		}
		fmt.Println()
	}

	if monorepoInfo.IsMonorepo {
		fmt.Printf("🏢 Monorepo detected: %s\n", monorepoInfo.Type)
		if len(monorepoInfo.Workspaces) > 0 {
			fmt.Println("   Workspaces found:")
			for _, ws := range monorepoInfo.Workspaces {
				fmt.Printf("   • %s\n", ws)
			}
		}
		fmt.Println()
	}

	// Choose configuration approach
	var cfg *pkgconfig.Config

	if len(detectedTypes) > 0 {
		useDefault := false
		prompt := &survey.Confirm{
			Message: fmt.Sprintf("Would you like to use the default configuration for %s?", detectedTypes[0].Name),
			Default: true,
		}
		if err := survey.AskOne(prompt, &useDefault); err != nil {
			return err
		}

		if useDefault {
			cfg, err = w.createFromDefault(detectedTypes[0].Name)
			if err != nil {
				return err
			}

			// Ask about customization
			customize := false
			prompt := &survey.Confirm{
				Message: "Would you like to customize the configuration?",
				Default: false,
			}
			if err := survey.AskOne(prompt, &customize); err != nil {
				return err
			}

			if customize {
				cfg, err = w.customizeConfiguration(cfg)
				if err != nil {
					return err
				}
			}
		} else {
			cfg, err = w.createManualConfiguration()
			if err != nil {
				return err
			}
		}
	} else {
		fmt.Println("📝 No project type detected. Let's create a custom configuration.")
		cfg, err = w.createManualConfiguration()
		if err != nil {
			return err
		}
	}

	// Handle monorepo configuration
	if monorepoInfo.IsMonorepo {
		configureMonorepo := false
		prompt := &survey.Confirm{
			Message: "Would you like to configure monorepo paths?",
			Default: true,
		}
		if err := survey.AskOne(prompt, &configureMonorepo); err != nil {
			return err
		}

		if configureMonorepo {
			cfg, err = w.configureMonorepoPaths(cfg, monorepoInfo)
			if err != nil {
				return err
			}
		}
	}

	// Validate configuration
	validator := config.NewValidator()
	if err := validator.Validate(cfg); err != nil {
		fmt.Printf("\n⚠️  Configuration validation warning: %v\n", err)
		
		saveAnyway := false
		prompt := &survey.Confirm{
			Message: "Do you want to save anyway?",
			Default: false,
		}
		if err := survey.AskOne(prompt, &saveAnyway); err != nil {
			return err
		}
		if !saveAnyway {
			return fmt.Errorf("configuration validation failed")
		}
	}

	// Save configuration
	data, err := pkgconfig.SaveConfig(cfg)
	if err != nil {
		return fmt.Errorf("failed to serialize configuration: %w", err)
	}

	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write configuration file: %w", err)
	}

	fmt.Printf("\n✅ Configuration saved to: %s\n", outputPath)
	fmt.Println("\n🎉 Setup complete! You can now use qualhook commands:")
	fmt.Println("   • qualhook format")
	fmt.Println("   • qualhook lint")
	fmt.Println("   • qualhook typecheck")
	fmt.Println("   • qualhook test")

	return nil
}

// createFromDefault creates a configuration from default template
func (w *ConfigWizard) createFromDefault(projectType string) (*pkgconfig.Config, error) {
	// Map detected project type to internal type
	var pType config.ProjectType
	switch projectType {
	case "nodejs":
		pType = config.ProjectTypeNodeJS
	case "go":
		pType = config.ProjectTypeGo
	case "python":
		pType = config.ProjectTypePython
	case "rust":
		pType = config.ProjectTypeRust
	default:
		return nil, fmt.Errorf("no default configuration for project type: %s", projectType)
	}

	return w.defaults.GetConfig(pType)
}

// createManualConfiguration creates a configuration through manual input
func (w *ConfigWizard) createManualConfiguration() (*pkgconfig.Config, error) {
	cfg := &pkgconfig.Config{
		Version:  "1.0",
		Commands: make(map[string]*pkgconfig.CommandConfig),
	}

	// Get project type
	projectType := ""
	prompt := &survey.Input{
		Message: "Project type (optional, e.g., nodejs, go, python):",
	}
	if err := survey.AskOne(prompt, &projectType, survey.WithValidator(survey.Required)); err != nil {
		if err.Error() != "interrupt" {
			// User pressed enter without input, which is fine
			projectType = ""
		} else {
			return nil, err
		}
	}
	if projectType != "" {
		cfg.ProjectType = projectType
	}

	// Configure standard commands
	fmt.Println("\nLet's configure the standard commands.")
	
	standardCommands := []struct {
		name   string
		prompt string
		defaultPrompt string
	}{
		{"format", "Formatting command", "Fix the formatting issues below:"},
		{"lint", "Linting command", "Fix the linting errors below:"},
		{"typecheck", "Type checking command", "Fix the type errors below:"},
		{"test", "Testing command", "Fix the failing tests below:"},
	}

	for _, cmd := range standardCommands {
		fmt.Printf("\n📝 Configuring '%s' command:\n", cmd.name)
		
		command := ""
		commandPrompt := &survey.Input{
			Message: cmd.prompt + " (leave empty to skip):",
		}
		if err := survey.AskOne(commandPrompt, &command); err != nil {
			return nil, err
		}
		
		if command == "" {
			fmt.Printf("  Skipping %s command.\n", cmd.name)
			continue
		}

		cmdConfig := &pkgconfig.CommandConfig{
			Command: command,
			Prompt:  cmd.defaultPrompt,
		}

		// Get additional arguments
		args := ""
		argsPrompt := &survey.Input{
			Message: "Additional arguments (space-separated):",
		}
		if err := survey.AskOne(argsPrompt, &args); err != nil {
			return nil, err
		}
		if args != "" {
			cmdConfig.Args = strings.Fields(args)
		}

		// Configure error detection
		exitCodes := []string{"1"}
		exitCodesPrompt := &survey.MultiSelect{
			Message: "Exit codes that indicate errors:",
			Options: []string{"0", "1", "2", "3", "4", "5"},
			Default: []string{"1"},
		}
		if err := survey.AskOne(exitCodesPrompt, &exitCodes); err != nil {
			return nil, err
		}
		
		cmdConfig.ErrorDetection = &pkgconfig.ErrorDetection{
			ExitCodes: make([]int, 0, len(exitCodes)),
		}
		for _, code := range exitCodes {
			var exitCode int
			fmt.Sscanf(code, "%d", &exitCode)
			cmdConfig.ErrorDetection.ExitCodes = append(cmdConfig.ErrorDetection.ExitCodes, exitCode)
		}

		// Configure output filter
		errorPattern := ""
		errorPatternPrompt := &survey.Input{
			Message: "Error pattern (regex, default: error):",
			Default: "error",
		}
		if err := survey.AskOne(errorPatternPrompt, &errorPattern); err != nil {
			return nil, err
		}
		
		cmdConfig.OutputFilter = &pkgconfig.FilterConfig{
			ErrorPatterns: []*pkgconfig.RegexPattern{
				{Pattern: errorPattern, Flags: "i"},
			},
			MaxOutput: 100,
		}

		cfg.Commands[cmd.name] = cmdConfig
	}

	// Custom commands
	addCustom := false
	customPrompt := &survey.Confirm{
		Message: "Would you like to add custom commands?",
		Default: false,
	}
	if err := survey.AskOne(customPrompt, &addCustom); err != nil {
		return nil, err
	}

	if addCustom {
		for {
			cmdName := ""
			namePrompt := &survey.Input{
				Message: "Custom command name (empty to finish):",
			}
			if err := survey.AskOne(namePrompt, &cmdName); err != nil {
				return nil, err
			}
			if cmdName == "" {
				break
			}

			command := ""
			commandPrompt := &survey.Input{
				Message: "Command to run:",
			}
			if err := survey.AskOne(commandPrompt, &command, survey.WithValidator(survey.Required)); err != nil {
				return nil, err
			}

			cfg.Commands[cmdName] = &pkgconfig.CommandConfig{
				Command: command,
				ErrorDetection: &pkgconfig.ErrorDetection{
					ExitCodes: []int{1},
				},
				OutputFilter: &pkgconfig.FilterConfig{
					ErrorPatterns: []*pkgconfig.RegexPattern{
						{Pattern: "error", Flags: "i"},
					},
					MaxOutput: 100,
				},
			}
		}
	}

	return cfg, nil
}

// customizeConfiguration allows customization of an existing configuration
func (w *ConfigWizard) customizeConfiguration(cfg *pkgconfig.Config) (*pkgconfig.Config, error) {
	fmt.Println("\n🛠  Customizing configuration...")

	// List current commands
	fmt.Println("\nCurrent commands:")
	var commandNames []string
	for name, cmd := range cfg.Commands {
		fmt.Printf("  • %s: %s %s\n", name, cmd.Command, strings.Join(cmd.Args, " "))
		commandNames = append(commandNames, name)
	}

	// Select commands to modify
	selectedCommands := []string{}
	selectPrompt := &survey.MultiSelect{
		Message: "Select commands to modify:",
		Options: commandNames,
	}
	if err := survey.AskOne(selectPrompt, &selectedCommands); err != nil {
		return nil, err
	}

	// Modify selected commands
	for _, cmdName := range selectedCommands {
		cmd := cfg.Commands[cmdName]
		fmt.Printf("\n📝 Modifying '%s' command:\n", cmdName)
		fmt.Printf("Current: %s %s\n", cmd.Command, strings.Join(cmd.Args, " "))

		// Modify command
		newCommand := cmd.Command
		commandPrompt := &survey.Input{
			Message: "New command:",
			Default: cmd.Command,
		}
		if err := survey.AskOne(commandPrompt, &newCommand); err != nil {
			return nil, err
		}
		cmd.Command = newCommand

		// Modify arguments
		newArgs := strings.Join(cmd.Args, " ")
		argsPrompt := &survey.Input{
			Message: "New arguments:",
			Default: newArgs,
		}
		if err := survey.AskOne(argsPrompt, &newArgs); err != nil {
			return nil, err
		}
		if newArgs != "" {
			cmd.Args = strings.Fields(newArgs)
		} else {
			cmd.Args = nil
		}

		// Modify prompt if present
		if cmd.Prompt != "" {
			newPrompt := cmd.Prompt
			promptPrompt := &survey.Input{
				Message: "New prompt:",
				Default: cmd.Prompt,
			}
			if err := survey.AskOne(promptPrompt, &newPrompt); err != nil {
				return nil, err
			}
			cmd.Prompt = newPrompt
		}
	}

	return cfg, nil
}

// configureMonorepoPaths configures monorepo path-specific settings
func (w *ConfigWizard) configureMonorepoPaths(cfg *pkgconfig.Config, info *detector.MonorepoInfo) (*pkgconfig.Config, error) {
	fmt.Println("\n🏢 Configuring monorepo paths...")

	cfg.Paths = make([]*pkgconfig.PathConfig, 0)

	for _, workspace := range info.Workspaces {
		if projects, exists := info.SubProjects[workspace]; exists && len(projects) > 0 {
			fmt.Printf("\nWorkspace: %s (detected: %s)\n", workspace, projects[0].Name)

			configure := false
			configPrompt := &survey.Confirm{
				Message: "Configure specific commands for this workspace?",
				Default: false,
			}
			if err := survey.AskOne(configPrompt, &configure); err != nil {
				return nil, err
			}

			if configure {
				pathConfig := &pkgconfig.PathConfig{
					Path:     workspace + "/**",
					Commands: make(map[string]*pkgconfig.CommandConfig),
				}

				// Select commands to override
				var commandNames []string
				for name := range cfg.Commands {
					commandNames = append(commandNames, name)
				}

				selectedCommands := []string{}
				selectPrompt := &survey.MultiSelect{
					Message: "Select commands to override for this workspace:",
					Options: commandNames,
				}
				if err := survey.AskOne(selectPrompt, &selectedCommands); err != nil {
					return nil, err
				}

				// Configure each selected command
				for _, cmdName := range selectedCommands {
					command := ""
					commandPrompt := &survey.Input{
						Message: fmt.Sprintf("Override command for '%s':", cmdName),
					}
					if err := survey.AskOne(commandPrompt, &command, survey.WithValidator(survey.Required)); err != nil {
						return nil, err
					}

					pathConfig.Commands[cmdName] = &pkgconfig.CommandConfig{
						Command: command,
						ErrorDetection: &pkgconfig.ErrorDetection{
							ExitCodes: []int{1},
						},
						OutputFilter: &pkgconfig.FilterConfig{
							ErrorPatterns: []*pkgconfig.RegexPattern{
								{Pattern: "error", Flags: "i"},
							},
							MaxOutput: 100,
						},
					}
				}

				if len(pathConfig.Commands) > 0 {
					cfg.Paths = append(cfg.Paths, pathConfig)
				}
			}
		}
	}

	return cfg, nil
}