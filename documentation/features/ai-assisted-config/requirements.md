# Requirements Document

## Introduction
The AI-Assisted Configuration feature enables automated generation and optimization of .qualhook.json configuration files using AI tools (Claude or Gemini). This feature addresses the challenge that default configurations often don't match project-specific build systems, requiring manual maintenance and deep understanding of each project's toolchain.

### Research Context
Based on the research findings:
- The existing configuration wizard supports three methods: default templates, customization, and manual creation
- Project detection already identifies 8 language types with confidence scoring
- Command execution infrastructure exists for secure external tool invocation
- Monorepo support is already implemented with workspace-specific configurations

## Requirements

### Requirement 1: Interactive Wizard Enhancement
**User Story:** As a developer, I want the configuration wizard to review all command types with me, so that I don't miss configuring important quality tools.

#### Acceptance Criteria
1. WHEN a user runs `qualhook config` THEN the system SHALL display all standard command types (format, lint, typecheck, test) plus any existing custom commands in a review list
2. IF a command type is not configured THEN the system SHALL clearly mark it as "Not Configured" with a visual indicator
3. WHEN viewing each command type THEN the user SHALL have options to: configure manually, use default, use AI assistance, or explicitly skip
4. IF the user selects "use AI assistance" AND no tool was pre-selected THEN the system SHALL ask which AI tool to use before proceeding
5. WHEN AI suggestions are received THEN the system SHALL allow the user to review, modify, or reject them before saving
6. WHEN reviewing commands THEN the user SHALL have the option to add new custom commands beyond the standard ones
7. IF the user adds a custom command THEN the system SHALL include it in the AI context for consistency with other commands

### Requirement 2: AI Integration Service
**User Story:** As a system component, I want to safely interact with AI tools, so that I can generate accurate configuration suggestions.

#### Acceptance Criteria
1. WHEN initializing AI assistance THEN the system SHALL detect if `claude` or `gemini` CLI tools are available
2. IF AI tools are not available THEN the system SHALL provide clear installation instructions and fall back to manual configuration
3. WHEN generating prompts THEN the system SHALL provide clear instructions asking the AI to:
   - Identify if the project is a monorepo and locate all workspaces/sub-projects
   - For each workspace/sub-project (or the root if not a monorepo):
     - Identify the project language and framework
     - Detect the build system and package manager
     - Determine the appropriate commands for: formatting, linting, type checking, and testing
     - Identify common error patterns for each command
     - Suggest any additional quality commands specific to that workspace
   - Return the configuration in a specified JSON format with proper workspace paths and inheritance
4. WHEN the AI tool is executing THEN the system SHALL display a progress indicator and allow the user to cancel with ESC key
5. WHEN parsing AI responses THEN the system SHALL validate JSON structure and regex patterns
6. IF basic validation passes THEN the system SHALL offer to test-run each command with user approval
7. WHEN test-running commands THEN the system SHALL:
   - Display the command to be tested
   - Ask for user confirmation before running
   - Run the command and show the output
   - Verify the command executes without critical errors
   - Allow the user to modify the command if it fails
8. IF AI response validation or test runs fail THEN the system SHALL allow the user to fix issues before proceeding

### Requirement 3: New AI Config Command
**User Story:** As a developer, I want to generate a complete .qualhook.json automatically, so that I can quickly set up quality checks without manual configuration.

#### Acceptance Criteria
1. WHEN a user runs `qualhook ai-config` THEN the system SHALL execute the AI tool to analyze and generate a complete configuration
2. WHEN presenting the generated configuration THEN the system SHALL display a summary (including any detected workspaces) and ask for user confirmation before saving
3. IF the user has an existing .qualhook.json THEN the system SHALL prompt to overwrite or merge configurations
4. WHEN the `--tool` flag is provided THEN the system SHALL use the specified AI tool (claude or gemini)
5. IF no tool is specified THEN the system SHALL:
   - Check for available AI tools (claude and gemini)
   - Display which tools are available
   - Ask the user to select which tool to use
   - Remember the user's choice for the current session

### Requirement 4: Security and Privacy
**User Story:** As a security-conscious developer, I want my project information to be handled safely, so that sensitive data is not exposed to AI tools.

#### Acceptance Criteria
1. WHEN constructing AI prompts THEN the system SHALL instruct the AI to exclude analysis of: .env files, credentials, API keys, and files matching .gitignore patterns
2. IF the AI response contains sensitive information THEN the system SHALL sanitize it before displaying or logging
3. WHEN executing AI tools THEN the system SHALL use the existing CommandExecutor with security validation
4. IF the AI suggests suspicious commands THEN the system SHALL reject them based on existing security validators
5. WHEN logging AI interactions THEN the system SHALL sanitize any potentially sensitive information from both prompts and responses

### Requirement 5: Error Handling and Fallback
**User Story:** As a developer, I want the AI configuration to handle errors gracefully, so that I can always complete my configuration setup.

#### Acceptance Criteria
1. IF the AI tool returns invalid JSON THEN the system SHALL log the error and attempt to extract useful information
2. WHEN network connectivity issues occur THEN the system SHALL detect them and suggest offline alternatives
3. IF AI suggestions are partially valid THEN the system SHALL use valid portions and prompt for manual completion of invalid sections
4. WHEN any AI-related error occurs THEN the system SHALL provide clear error messages and actionable next steps
5. IF the user cancels AI assistance THEN the system SHALL seamlessly continue with manual configuration

### Requirement 6: Configuration Presentation and Review
**User Story:** As a developer, I want to review and understand the AI-generated configuration, so that I can trust and maintain it.

#### Acceptance Criteria
1. WHEN presenting AI-generated configurations THEN the system SHALL display a clear summary showing all detected workspaces (if monorepo) and their configurations
2. IF the configuration includes workspace-specific overrides THEN the system SHALL clearly show the inheritance hierarchy
3. WHEN displaying commands THEN the system SHALL group them by workspace and highlight any differences from the root configuration
4. IF the AI suggests custom commands beyond the standard four THEN the system SHALL explain why these commands were recommended
5. WHEN the user reviews the configuration THEN they SHALL have options to: accept all, modify specific commands, test commands, or regenerate with different parameters

## Non-Functional Requirements

### Performance
1. AI tool execution SHALL show progress and be user-cancellable at any time
2. The system SHALL display elapsed time during AI analysis
3. Configuration validation SHALL complete within 1 second

### Usability
1. Error messages SHALL be clear and provide actionable guidance
2. AI tool installation instructions SHALL include platform-specific commands
3. Progress indicators SHALL show: spinner, elapsed time, and "Press ESC to cancel" message

### Compatibility
1. The feature SHALL support both Claude and Gemini CLI tools
2. The feature SHALL work with all existing project types (Go, Node.js, Python, Rust, Java, Ruby, PHP, .NET)
3. The feature SHALL maintain backward compatibility with existing .qualhook.json files

### Security
1. The system SHALL execute AI tools in the project directory, allowing them to analyze files directly rather than sending file contents through prompts
2. The system SHALL validate all AI-suggested commands against existing security rules
3. The system SHALL instruct the AI to respect .gitignore patterns and avoid analyzing sensitive files