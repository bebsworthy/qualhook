# AI-Assisted Configuration Research

## Overview
This document captures research findings for implementing AI-assisted configuration of .qualhook.json files using Claude or Gemini.

## Current Architecture Analysis

### 1. Configuration System
- **Config Command** (`cmd/qualhook/config.go`):
  - Entry point for configuration wizard
  - Supports validation and force overwrite
  - Delegates to wizard implementation

- **Configuration Wizard** (`internal/wizard/config.go`):
  - Interactive prompts using AlecAivazis/survey
  - Three configuration methods:
    - `createFromDefault()` - Uses default templates
    - `customizeConfiguration()` - Modifies existing config
    - `createManualConfiguration()` - Creates from scratch
  - Monorepo support with path-specific configurations

- **Default Configurations** (`internal/config/defaults.go`):
  - Embedded JSON templates for multiple languages
  - Project types: Go, Node.js, Python, Rust
  - Each template includes commands with error patterns and exit codes

### 2. Project Detection System
- **Project Detector** (`internal/detector/project.go`):
  - Marker-based detection with confidence scores
  - Supports 8 project types (Go, Node.js, Python, Rust, Java, Ruby, PHP, .NET)
  - Monorepo detection for multiple systems (lerna, nx, rush, pnpm, yarn, turborepo, bazel)
  - Workspace scanning with sub-project type detection

### 3. Command Execution
- **Command Executor** (`internal/executor/command.go`):
  - Secure command execution with validation
  - Timeout support and context cancellation
  - Environment variable handling
  - Output capture (stdout/stderr)

## Integration Points for AI Assistance

### 1. Interactive Wizard Enhancement
**Location**: `internal/wizard/config.go`

#### a. Command Customization Phase
In `customizeConfiguration()` method (line 140), after listing current commands:
- Add option: "Use AI to suggest better commands"
- Execute AI tool to analyze project and suggest commands
- Parse AI response and update command configurations

#### b. Manual Configuration Phase
In `configureStandardCommand()` method (line 515), when prompting for command:
- Add option: "Let AI suggest the command"
- Execute AI with project context
- Pre-fill the command suggestion

#### c. Mandatory Review Flow
Modify `createConfiguration()` method (line 360) to:
- Always show all command types (format, lint, typecheck, test)
- Mark each as "configured" or "not configured"
- Require user to explicitly skip or configure each

### 2. New AI Config Command
**Location**: New file `cmd/qualhook/ai-config.go`

Command structure:
```go
var aiConfigCmd = &cobra.Command{
    Use:   "ai-config",
    Short: "Generate configuration using AI assistance",
    Long:  "Automatically generate .qualhook.json by analyzing your project with Claude or Gemini",
    RunE:  runAIConfig,
}
```

### 3. AI Integration Service
**Location**: New package `internal/ai/assistant.go`

Core functionality:
- Project context gathering (files, structure, existing tools)
- Prompt generation for AI tools
- Command execution (`claude -p` or `gemini -p`)
- Response parsing and validation
- Configuration generation

## Implementation Strategy

### 1. Command Execution Pattern
Use existing `CommandExecutor` to run AI tools:
```go
executor := executor.NewCommandExecutor(30 * time.Second)
result, err := executor.Execute("claude", []string{"-p", prompt}, options)
```

### 2. Prompt Generation
Create context-aware prompts including:
- Project type and structure
- Detected build files
- Common patterns for the language
- Request for specific command format

### 3. Response Parsing
- Expect JSON or structured format from AI
- Validate suggested commands
- Ensure error patterns are valid regex
- Verify exit codes are reasonable

### 4. Configuration Validation
- Use existing `config.Validator` to validate AI suggestions
- Provide feedback if suggestions are invalid
- Allow user to review and modify before saving

## Technical Considerations

### 1. Tool Detection
- Check if `claude` or `gemini` CLI tools are available
- Provide clear error messages if not installed
- Offer fallback to manual configuration

### 2. Security
- Sanitize project information sent to AI
- Don't include sensitive file contents
- Use existing security validators for commands

### 3. Monorepo Handling
- Generate prompts for each workspace
- Allow AI to suggest workspace-specific overrides
- Maintain configuration inheritance

### 4. Error Handling
- AI tool timeouts
- Invalid AI responses
- Network connectivity issues
- Graceful fallback to manual configuration

## Next Steps

1. **Requirements Phase**: Define detailed requirements for both wizard enhancement and ai-config command
2. **Design Phase**: Create technical design for AI integration service
3. **Implementation Phase**: 
   - Phase 1: AI integration service and prompt generation
   - Phase 2: Wizard enhancement with mandatory review
   - Phase 3: New ai-config command
   - Phase 4: Testing and refinement

## Open Questions

1. Should we support both Claude and Gemini, or start with one?
2. How to handle API key configuration for the CLI tools?
3. Should AI suggestions be cached for repeated runs?
4. What level of project analysis should we include in prompts?