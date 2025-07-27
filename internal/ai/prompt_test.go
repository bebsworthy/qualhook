package ai

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/bebsworthy/qualhook/pkg/config"
)

func TestNewPromptGenerator(t *testing.T) {
	pg := NewPromptGenerator()

	if pg == nil {
		t.Fatal("NewPromptGenerator returned nil")
	}

	// Verify it implements the interface
	var _ = pg
}

func TestGenerateConfigPrompt(t *testing.T) {
	tests := []struct {
		name       string
		workingDir string
		checks     []string
	}{
		{
			name:       "basic project directory",
			workingDir: "/home/user/myproject",
			checks: []string{
				"Analyze the project in the directory: /home/user/myproject",
				"Detect if this is a monorepo",
				"Identify the primary language(s) and framework(s)",
				"format: Code formatting",
				"lint: Static analysis",
				"typecheck: Type checking",
				"test: Running tests",
				"Respect .gitignore patterns",
				"Do not analyze or include information from .env files",
				"\"monorepo\":",
				"\"commands\":",
				"\"customCommands\":",
			},
		},
		{
			name:       "path with spaces",
			workingDir: "/home/user/my project",
			checks: []string{
				"Analyze the project in the directory: /home/user/my project",
			},
		},
		{
			name:       "Windows path",
			workingDir: "C:\\Users\\Developer\\Projects\\app",
			checks: []string{
				"Analyze the project in the directory: C:\\Users\\Developer\\Projects\\app",
			},
		},
	}

	pg := NewPromptGenerator()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt := pg.GenerateConfigPrompt(tt.workingDir)

			// Check that all expected content is present
			for _, check := range tt.checks {
				if !strings.Contains(prompt, check) {
					t.Errorf("Expected prompt to contain %q, but it didn't", check)
				}
			}

			// Verify JSON example is valid
			startIdx := strings.Index(prompt, "{")
			endIdx := strings.LastIndex(prompt, "}")
			if startIdx == -1 || endIdx == -1 {
				t.Fatal("Could not find JSON example in prompt")
			}

			jsonExample := prompt[startIdx : endIdx+1]
			var example map[string]interface{}
			if err := json.Unmarshal([]byte(jsonExample), &example); err != nil {
				t.Errorf("Invalid JSON example in prompt: %v", err)
			}

			// Verify example has required fields
			requiredFields := []string{"version", "projectType", "monorepo", "commands"}
			for _, field := range requiredFields {
				if _, ok := example[field]; !ok {
					t.Errorf("Example JSON missing required field: %s", field)
				}
			}
		})
	}
}

func TestGenerateCommandPrompt(t *testing.T) {
	tests := []struct {
		name        string
		commandType string
		context     ProjectContext
		checks      []string
	}{
		{
			name:        "format command for Go project",
			commandType: "format",
			context: ProjectContext{
				ProjectType: "go",
			},
			checks: []string{
				"Suggest a format command configuration",
				"Project Type: go",
				"most appropriate format command",
				"\"command\":",
				"\"args\":",
				"\"errorPatterns\":",
				"\"exitCodes\":",
				"\"explanation\":",
			},
		},
		{
			name:        "lint command with existing config",
			commandType: "lint",
			context: ProjectContext{
				ProjectType: "nodejs",
				ExistingConfig: &config.Config{
					Commands: map[string]*config.CommandConfig{
						"format": {
							Command: "prettier",
							Args:    []string{"--write", "."},
						},
						"test": {
							Command: "jest",
							Args:    []string{"--coverage"},
						},
					},
				},
			},
			checks: []string{
				"Suggest a lint command configuration",
				"Project Type: nodejs",
				"Existing commands:",
				"format: prettier --write .",
				"test: jest --coverage",
			},
		},
		{
			name:        "test command with custom commands",
			commandType: "test",
			context: ProjectContext{
				ProjectType:    "python",
				CustomCommands: []string{"build", "deploy", "security-scan"},
			},
			checks: []string{
				"Suggest a test command configuration",
				"Project Type: python",
				"Custom commands to consider:",
				"- build",
				"- deploy",
				"- security-scan",
			},
		},
		{
			name:        "typecheck command for empty context",
			commandType: "typecheck",
			context:     ProjectContext{},
			checks: []string{
				"Suggest a typecheck command configuration",
				"Available tools in the project",
			},
		},
	}

	pg := NewPromptGenerator()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt := pg.GenerateCommandPrompt(tt.commandType, tt.context)

			// Check that all expected content is present
			for _, check := range tt.checks {
				if !strings.Contains(prompt, check) {
					t.Errorf("Expected prompt to contain %q, but it didn't\nPrompt:\n%s", check, prompt)
				}
			}

			// Verify JSON example is valid
			startIdx := strings.Index(prompt, "{")
			endIdx := strings.LastIndex(prompt, "}")
			if startIdx == -1 || endIdx == -1 {
				t.Fatal("Could not find JSON example in prompt")
			}

			jsonExample := prompt[startIdx : endIdx+1]
			var suggestion map[string]interface{}
			if err := json.Unmarshal([]byte(jsonExample), &suggestion); err != nil {
				t.Errorf("Invalid JSON suggestion in prompt: %v", err)
			}

			// Verify suggestion has required fields
			requiredFields := []string{"command", "args", "errorPatterns", "exitCodes", "explanation"}
			for _, field := range requiredFields {
				if _, ok := suggestion[field]; !ok {
					t.Errorf("Suggestion JSON missing required field: %s", field)
				}
			}
		})
	}
}

func TestPromptContent(t *testing.T) {
	pg := NewPromptGenerator()

	t.Run("config prompt instructions", func(t *testing.T) {
		prompt := pg.GenerateConfigPrompt("/test/dir")

		// Verify important instructions are included
		instructions := []string{
			"Detect if this is a monorepo",
			"Identify common error patterns",
			"Respect .gitignore patterns",
			"Do not analyze or include information from .env files",
			"For monorepos, provide both root-level commands and workspace-specific overrides",
			"Include appropriate exit codes",
			"Use regex patterns that will match actual error output",
			"If a command type doesn't apply",
		}

		for _, instruction := range instructions {
			if !strings.Contains(prompt, instruction) {
				t.Errorf("Config prompt missing important instruction: %s", instruction)
			}
		}
	})

	t.Run("command prompt instructions", func(t *testing.T) {
		prompt := pg.GenerateCommandPrompt("lint", ProjectContext{})

		// Verify important instructions are included
		instructions := []string{
			"detected project type and framework",
			"Available tools in the project",
			"Existing configuration files",
			"Common conventions",
			"Use the most commonly adopted tool",
			"Include arguments that make the tool suitable for CI/CD",
			"Error patterns should use Go regex syntax (RE2)",
		}

		for _, instruction := range instructions {
			if !strings.Contains(prompt, instruction) {
				t.Errorf("Command prompt missing important instruction: %s", instruction)
			}
		}
	})
}

func TestMonorepoInstructions(t *testing.T) {
	pg := NewPromptGenerator()
	prompt := pg.GenerateConfigPrompt("/monorepo/project")

	// Check for monorepo-specific instructions
	monorepoChecks := []string{
		"\"monorepo\": {",
		"\"detected\":",
		"\"type\":",
		"\"workspaces\":",
		"yarn-workspaces",
		"npm-workspaces",
		"lerna",
		"nx",
		"turborepo",
		"pnpm-workspace",
		"\"paths\":",
		"path-specific command overrides",
	}

	for _, check := range monorepoChecks {
		if !strings.Contains(prompt, check) {
			t.Errorf("Config prompt missing monorepo instruction/example: %s", check)
		}
	}
}

func TestExampleFormats(t *testing.T) {
	t.Run("config example has all command types", func(t *testing.T) {
		example := createExampleResponse()
		commands, ok := example["commands"].(map[string]interface{})
		if !ok {
			t.Fatal("Example response missing commands")
		}

		expectedCommands := []string{"format", "lint", "typecheck", "test"}
		for _, cmd := range expectedCommands {
			if _, ok := commands[cmd]; !ok {
				t.Errorf("Example missing %s command", cmd)
			}
		}
	})

	t.Run("command examples for all types", func(t *testing.T) {
		commandTypes := []string{"format", "lint", "typecheck", "test", "unknown"}

		for _, cmdType := range commandTypes {
			example := createExampleCommandSuggestion(cmdType)

			// Verify required fields
			requiredFields := []string{"command", "args", "errorPatterns", "exitCodes", "explanation"}
			for _, field := range requiredFields {
				if _, ok := example[field]; !ok {
					t.Errorf("Example for %s missing field: %s", cmdType, field)
				}
			}

			// Verify explanation is not empty
			if explanation, ok := example["explanation"].(string); !ok || explanation == "" {
				t.Errorf("Example for %s has empty explanation", cmdType)
			}
		}
	})
}

func TestSecurityInstructions(t *testing.T) {
	pg := NewPromptGenerator()
	prompt := pg.GenerateConfigPrompt("/secure/project")

	// Verify security-related instructions
	securityChecks := []string{
		".gitignore",
		".env files",
		"credentials",
		"API keys",
	}

	for _, check := range securityChecks {
		if !strings.Contains(prompt, check) {
			t.Errorf("Config prompt missing security instruction about: %s", check)
		}
	}
}

func TestProjectScenarios(t *testing.T) {
	scenarios := []struct {
		name    string
		context ProjectContext
	}{
		{
			name: "Go monorepo",
			context: ProjectContext{
				ProjectType:    "go",
				CustomCommands: []string{"generate", "migrate"},
			},
		},
		{
			name: "Node.js with TypeScript",
			context: ProjectContext{
				ProjectType: "nodejs",
				ExistingConfig: &config.Config{
					Commands: map[string]*config.CommandConfig{
						"typecheck": {
							Command: "tsc",
							Args:    []string{"--noEmit"},
						},
					},
				},
			},
		},
		{
			name: "Python project",
			context: ProjectContext{
				ProjectType: "python",
			},
		},
		{
			name: "Rust workspace",
			context: ProjectContext{
				ProjectType:    "rust",
				CustomCommands: []string{"clippy", "doc"},
			},
		},
		{
			name:    "Empty/unknown project",
			context: ProjectContext{},
		},
	}

	pg := NewPromptGenerator()

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			// Test config prompt
			configPrompt := pg.GenerateConfigPrompt("/test/project")
			if configPrompt == "" {
				t.Error("Config prompt is empty")
			}

			// Test command prompts for each type
			for _, cmdType := range []string{"format", "lint", "typecheck", "test"} {
				cmdPrompt := pg.GenerateCommandPrompt(cmdType, scenario.context)
				if cmdPrompt == "" {
					t.Errorf("Command prompt for %s is empty", cmdType)
				}

				// Verify context is included
				if scenario.context.ProjectType != "" {
					if !strings.Contains(cmdPrompt, scenario.context.ProjectType) {
						t.Errorf("Command prompt doesn't include project type: %s", scenario.context.ProjectType)
					}
				}
			}
		})
	}
}
