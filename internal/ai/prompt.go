// Package ai provides AI-powered configuration generation for qualhook.
package ai

import (
	"encoding/json"
	"fmt"
	"strings"
)

// promptGenerator implements the PromptGenerator interface
type promptGenerator struct {
	templates map[string]string
}

// NewPromptGenerator creates a new prompt generator instance
func NewPromptGenerator() PromptGenerator {
	pg := &promptGenerator{
		templates: make(map[string]string),
	}
	pg.initializeTemplates()
	return pg
}

// initializeTemplates sets up the prompt templates
func (p *promptGenerator) initializeTemplates() {
	p.templates["config"] = configPromptTemplate
	p.templates["command"] = commandPromptTemplate
}

// GenerateConfigPrompt creates a prompt for full configuration generation
func (p *promptGenerator) GenerateConfigPrompt(workingDir string) string {
	template := p.templates["config"]

	// Create the example response format
	exampleResponse := createExampleResponse()
	exampleJSON, err := json.MarshalIndent(exampleResponse, "", "  ")
	if err != nil {
		// This should never happen with our predefined structure
		exampleJSON = []byte("{}")
	}

	// Replace placeholders in template
	prompt := strings.ReplaceAll(template, "{{WORKING_DIR}}", workingDir)
	prompt = strings.ReplaceAll(prompt, "{{EXAMPLE_RESPONSE}}", string(exampleJSON))

	return prompt
}

// GenerateCommandPrompt creates a prompt for specific command suggestion
func (p *promptGenerator) GenerateCommandPrompt(commandType string, context ProjectContext) string {
	template := p.templates["command"]

	// Create context information
	var contextInfo strings.Builder
	if context.ProjectType != "" {
		contextInfo.WriteString(fmt.Sprintf("Project Type: %s\n", context.ProjectType))
	}

	if context.ExistingConfig != nil && len(context.ExistingConfig.Commands) > 0 {
		contextInfo.WriteString("\nExisting commands:\n")
		for cmdType, cmd := range context.ExistingConfig.Commands {
			contextInfo.WriteString(fmt.Sprintf("- %s: %s %s\n", cmdType, cmd.Command, strings.Join(cmd.Args, " ")))
		}
	}

	if len(context.CustomCommands) > 0 {
		contextInfo.WriteString("\nCustom commands to consider:\n")
		for _, cmd := range context.CustomCommands {
			contextInfo.WriteString(fmt.Sprintf("- %s\n", cmd))
		}
	}

	// Create example suggestion
	exampleSuggestion := createExampleCommandSuggestion(commandType)
	exampleJSON, err := json.MarshalIndent(exampleSuggestion, "", "  ")
	if err != nil {
		// This should never happen with our predefined structure
		exampleJSON = []byte("{}")
	}

	// Replace placeholders
	prompt := strings.ReplaceAll(template, "{{COMMAND_TYPE}}", commandType)
	prompt = strings.ReplaceAll(prompt, "{{CONTEXT_INFO}}", contextInfo.String())
	prompt = strings.ReplaceAll(prompt, "{{EXAMPLE_SUGGESTION}}", string(exampleJSON))

	return prompt
}

// Config generation prompt template
const configPromptTemplate = `Analyze the project in the directory: {{WORKING_DIR}}

Your task is to generate a comprehensive quality check configuration for qualhook. Please:

1. Detect if this is a monorepo and identify all workspaces/sub-projects
2. Identify the primary language(s) and framework(s) used
3. Detect the build system and package manager
4. Determine appropriate commands for:
   - format: Code formatting (e.g., prettier, gofmt, black)
   - lint: Static analysis and linting (e.g., eslint, golangci-lint, pylint)
   - typecheck: Type checking if applicable (e.g., tsc, mypy)
   - test: Running tests (e.g., jest, go test, pytest)
   - **Important**: If the project uses a system build tool such as Make, Maven, Gradle, NPM, Yarn, Pip, Grunt, Gulp, or any other you MUST prefer returning the relevant commands specified int the build tool configuration such as 'make format', 'mvn lint', 'gradle typecheck', 'npm test', etc.
5. Identify common error patterns for each command
6. Suggest any additional quality commands specific to this project

IMPORTANT Instructions:
- Respect .gitignore patterns - do not analyze files that would be ignored by git
- Do not analyze or include information from .env files, credentials, or API keys
- For monorepos, provide both root-level commands and workspace-specific overrides where needed
- Include appropriate exit codes that indicate failure (typically non-zero)
- Use regex patterns that will match actual error output from the tools
- If a command type doesn't apply (e.g., typecheck for JavaScript without TypeScript), omit it
- ALWAYS prefer system build tools commands over direct tool invocations unless no build tool is present

Return your response as a JSON object following this exact structure:

{{EXAMPLE_RESPONSE}}

Notes on the response format:
- "monorepo.detected": boolean indicating if this is a monorepo
- "monorepo.type": the monorepo tool used (e.g., "yarn-workspaces", "npm-workspaces", "lerna", "nx", "turborepo", "pnpm-workspace", "other")
- "monorepo.workspaces": array of workspace paths relative to the root
- "commands": the default commands that apply to the entire project
- "paths": array of path-specific command overrides for monorepo workspaces
- "customCommands": additional quality commands beyond the standard four, with explanations
- Error patterns should use Go regex syntax (RE2)
- The "flags" field in errorPatterns can be empty or contain "i" for case-insensitive

Analyze the project structure, configuration files, and build scripts to provide accurate suggestions.`

// Command suggestion prompt template
const commandPromptTemplate = `Suggest a {{COMMAND_TYPE}} command configuration for the current project.

{{CONTEXT_INFO}}

Your task is to suggest the most appropriate {{COMMAND_TYPE}} command for this project based on:
1. The detected project type and framework
2. Available tools in the project (check package.json, go.mod, requirements.txt, etc.)
3. Existing configuration files for the tools
4. Common conventions for this type of project
5. ALWAYS prefer system build tools commands over direct tool invocations unless no build tool is present

**Important**: 
If the project uses a system build tool such as Make, Maven, Gradle, NPM, Yarn, Pip, Grunt, Gulp, or any other you MUST prefer returning the relevant commands specified int the build tool configuration such as 'make format', 'mvn lint', 'gradle typecheck', 'npm test', etc.

For the {{COMMAND_TYPE}} command, provide:
- The base command to execute
- Appropriate arguments
- Regex patterns that match error output
- Exit codes that indicate failure
- A brief explanation of why this command was chosen

Return your response as a JSON object following this exact structure:

{{EXAMPLE_SUGGESTION}}

Notes:
- Use the most commonly adopted tool for this project type
- Include arguments that make the tool suitable for CI/CD environments
- Error patterns should use Go regex syntax (RE2)
- The explanation should be 1-2 sentences describing why this command is appropriate`

// createExampleResponse creates an example configuration response
func createExampleResponse() map[string]interface{} {
	return map[string]interface{}{
		"version":     "1.0",
		"projectType": "nodejs",
		"monorepo": map[string]interface{}{
			"detected":   true,
			"type":       "yarn-workspaces",
			"workspaces": []string{"packages/backend", "packages/frontend"},
		},
		"commands": map[string]interface{}{
			"format": map[string]interface{}{
				"command": "prettier",
				"args":    []string{"--write", "."},
				"errorPatterns": []map[string]string{
					{"pattern": "\\[error\\]", "flags": "i"},
				},
				"exitCodes": []int{1, 2},
			},
			"lint": map[string]interface{}{
				"command": "eslint",
				"args":    []string{".", "--fix"},
				"errorPatterns": []map[string]string{
					{"pattern": "\\d+ problems? \\(\\d+ errors?, \\d+ warnings?\\)", "flags": ""},
				},
				"exitCodes": []int{1},
			},
			"typecheck": map[string]interface{}{
				"command": "tsc",
				"args":    []string{"--noEmit"},
				"errorPatterns": []map[string]string{
					{"pattern": "error TS\\d+:", "flags": ""},
				},
				"exitCodes": []int{1, 2},
			},
			"test": map[string]interface{}{
				"command": "jest",
				"args":    []string{"--passWithNoTests"},
				"errorPatterns": []map[string]string{
					{"pattern": "FAIL", "flags": ""},
				},
				"exitCodes": []int{1},
			},
		},
		"paths": []map[string]interface{}{
			{
				"path": "packages/backend/**",
				"commands": map[string]interface{}{
					"test": map[string]interface{}{
						"command": "jest",
						"args":    []string{"--config", "packages/backend/jest.config.js"},
					},
				},
			},
		},
		"customCommands": map[string]interface{}{
			"build": map[string]interface{}{
				"command":     "yarn",
				"args":        []string{"build"},
				"explanation": "Detected build script in package.json",
			},
		},
	}
}

// createExampleCommandSuggestion creates an example command suggestion
func createExampleCommandSuggestion(commandType string) map[string]interface{} {
	examples := map[string]map[string]interface{}{
		"format": {
			"command": "prettier",
			"args":    []string{"--write", "."},
			"errorPatterns": []map[string]string{
				{"pattern": "\\[error\\]", "flags": "i"},
			},
			"exitCodes":   []int{1, 2},
			"explanation": "Prettier is the most widely used formatter for JavaScript/TypeScript projects",
		},
		"lint": {
			"command": "eslint",
			"args":    []string{".", "--max-warnings", "0"},
			"errorPatterns": []map[string]string{
				{"pattern": "\\d+ problems? \\(\\d+ errors?, \\d+ warnings?\\)", "flags": ""},
			},
			"exitCodes":   []int{1},
			"explanation": "ESLint with --max-warnings 0 ensures both errors and warnings fail the check",
		},
		"typecheck": {
			"command": "tsc",
			"args":    []string{"--noEmit"},
			"errorPatterns": []map[string]string{
				{"pattern": "error TS\\d+:", "flags": ""},
			},
			"exitCodes":   []int{1, 2},
			"explanation": "TypeScript compiler with --noEmit performs type checking without generating output files",
		},
		"test": {
			"command": "jest",
			"args":    []string{"--ci", "--coverage"},
			"errorPatterns": []map[string]string{
				{"pattern": "FAIL", "flags": ""},
				{"pattern": "Test Suites: \\d+ failed", "flags": ""},
			},
			"exitCodes":   []int{1},
			"explanation": "Jest with --ci flag is optimized for continuous integration environments",
		},
	}

	if example, ok := examples[commandType]; ok {
		return example
	}

	// Default example for unknown command types
	return map[string]interface{}{
		"command": "unknown",
		"args":    []string{},
		"errorPatterns": []map[string]string{
			{"pattern": "error", "flags": "i"},
		},
		"exitCodes":   []int{1},
		"explanation": "Please provide appropriate command for " + commandType,
	}
}
