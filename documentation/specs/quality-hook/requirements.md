# Requirements Document: Quality Hook

## Introduction

Quality Hook is a configurable command-line utility that serves as Claude Code hooks to enforce code quality for LLM coding agents. The tool acts as an intelligent wrapper around project-specific commands (format, lint, typecheck, test), filtering their output to provide only relevant error information to the LLM. It supports diverse project types and monorepos through a configuration-driven architecture.

### Research Context

Based on research findings:
- Claude Code uses exit code 2 to block actions and feed stderr back to the LLM
- Configuration-driven architecture enables support for any project type without hardcoding
- Intelligent output filtering is critical to avoid overwhelming the LLM with verbose output
- Monorepo support requires path-based configuration with clear precedence rules

## Requirements

### Requirement 1: Core CLI Commands

**User Story:** As a developer, I want to run quality checks through simple CLI commands, so that I can integrate them into Claude Code hooks.

#### Acceptance Criteria
1. WHEN a user runs `qualhook format` THEN the system SHALL execute the configured formatting command for the current project
2. WHEN a user runs `qualhook lint` THEN the system SHALL execute the configured linting command for the current project
3. WHEN a user runs `qualhook typecheck` THEN the system SHALL execute the configured type checking command for the current project
4. WHEN a user runs `qualhook test` THEN the system SHALL execute the configured test command for the current project
5. WHEN a user runs `qualhook <custom-command>` THEN the system SHALL execute any custom command defined in the configuration
6. WHEN any command executes THEN the system SHALL delegate to the actual project tool without reimplementing functionality

### Requirement 2: Claude Code Hook Integration

**User Story:** As a Claude Code user, I want quality hooks that properly communicate with the LLM, so that errors are automatically fixed.

#### Acceptance Criteria
1. WHEN a quality check detects errors THEN the system SHALL return exit code 2
2. WHEN returning exit code 2 THEN the system SHALL output filtered error information to stderr
3. WHEN no errors are detected THEN the system SHALL return exit code 0
4. WHEN the hook receives JSON input from Claude Code THEN the system SHALL parse and use it for context
5. WHEN outputting errors THEN the system SHALL prefix them with actionable prompts like "Fix the linting issues below:"
6. WHEN processing output THEN the system SHALL filter to include only relevant error information

### Requirement 3: Configuration System

**User Story:** As a developer, I want a flexible configuration system, so that I can support any project type without code changes.

#### Acceptance Criteria
1. WHEN configuration is needed THEN the system SHALL support JSON format
2. WHEN configuring a command THEN the system SHALL allow specifying:
   - The actual command to run
   - Error detection patterns (stdout/stderr regex, exit codes)
   - Output filtering rules
   - LLM prompt templates
3. WHEN no configuration exists THEN the system SHALL guide users through an interactive setup
4. WHEN configuration exists THEN the system SHALL validate it on startup
5. WHEN new project types are encountered THEN users SHALL be able to configure them without code changes

### Requirement 4: Project Type Detection

**User Story:** As a developer, I want automatic project type detection, so that configuration is quick and accurate.

#### Acceptance Criteria
1. WHEN running initial configuration THEN the system SHALL scan for project indicators (package.json, go.mod, Cargo.toml, etc.)
2. WHEN detecting a project type THEN the system SHALL suggest appropriate default configurations
3. WHEN multiple project types are detected THEN the system SHALL ask for user confirmation
4. WHEN suggesting configurations THEN the system SHALL allow users to modify before accepting
5. WHEN unknown project types are encountered THEN the system SHALL allow manual configuration

### Requirement 5: Monorepo Support

**User Story:** As a monorepo developer, I want path-based configurations, so that different parts of my repository use appropriate tools.

#### Acceptance Criteria
1. WHEN in a monorepo THEN the system SHALL support path-based configuration rules
2. WHEN multiple configurations match a path THEN the system SHALL use the most specific match
3. WHEN configuring a monorepo THEN the system SHALL support:
   - Root-level default configuration
   - Directory-specific overrides
   - Glob pattern matching for paths
4. WHEN detecting project structure THEN the system SHALL identify common monorepo patterns
5. WHEN executing commands in a monorepo THEN the system SHALL use the configuration for the current working directory

### Requirement 6: File-Aware Execution

**User Story:** As a monorepo developer, I want quality checks to run only on the relevant project components based on edited files, so that checks are faster and more targeted.

#### Acceptance Criteria
1. WHEN files are edited in a specific directory (e.g., frontend/) THEN the system SHALL run only the quality checks configured for that directory
2. WHEN files are edited across multiple project components THEN the system SHALL run the appropriate checks for each component
3. WHEN receiving file paths from Claude Code hooks THEN the system SHALL:
   - Map each file to its project component
   - Group files by component
   - Execute appropriate commands for each component
4. WHEN no files match any configured path THEN the system SHALL fall back to root configuration
5. WHEN running file-aware checks THEN the system SHALL report which checks were run for which files

### Requirement 7: Output Filtering

**User Story:** As an LLM, I want concise, relevant error information, so that I can effectively fix issues without being overwhelmed.

#### Acceptance Criteria
1. WHEN filtering output THEN the system SHALL use configurable regex patterns to identify errors
2. WHEN an error is detected THEN the system SHALL capture configurable lines of context
3. WHEN output exceeds limits THEN the system SHALL truncate intelligently while preserving error information
4. WHEN filtering THEN the system SHALL support:
   - Error detection patterns
   - Context inclusion patterns
   - Output size limits
   - Priority-based filtering (errors before warnings)
5. WHEN no errors match patterns THEN the system SHALL pass through a configurable amount of output

### Requirement 8: Configuration Management

**User Story:** As a developer, I want easy configuration management, so that I can quickly set up and modify project settings.

#### Acceptance Criteria
1. WHEN running `qualhook config` THEN the system SHALL start an interactive configuration wizard
2. WHEN running `qualhook config --validate` THEN the system SHALL check configuration validity
3. WHEN sharing configurations THEN the system SHALL support importing/exporting configuration templates

### Requirement 9: Error Handling and Diagnostics

**User Story:** As a developer, I want clear error messages and diagnostics, so that I can troubleshoot configuration issues.

#### Acceptance Criteria
1. WHEN configuration errors occur THEN the system SHALL provide specific error messages with fix suggestions
2. WHEN command execution fails THEN the system SHALL distinguish between configuration errors and tool errors
3. WHEN running `qualhook --debug` THEN the system SHALL output detailed execution information
4. WHEN timeouts occur THEN the system SHALL kill processes cleanly and report the timeout
5. WHEN parsing errors occur THEN the system SHALL show the problematic pattern and sample input

### Requirement 10: Performance

**User Story:** As a developer, I want fast execution with minimal overhead, so that quality checks don't slow down my workflow.

#### Acceptance Criteria
1. WHEN executing commands THEN the system SHALL add no more than 100ms overhead
2. WHEN processing large outputs THEN the system SHALL use streaming to avoid memory issues
3. WHEN caching is applicable THEN the system SHALL cache project detection results
4. WHEN timeouts are configured THEN the system SHALL enforce them reliably
5. WHEN running in CI/CD THEN the system SHALL support parallel execution without conflicts

### Requirement 11: Security

**User Story:** As a security-conscious developer, I want safe command execution, so that my system isn't vulnerable to injection attacks.

#### Acceptance Criteria
1. WHEN executing commands THEN the system SHALL properly escape all arguments
2. WHEN loading configuration THEN the system SHALL validate against command injection
3. WHEN handling file paths THEN the system SHALL prevent directory traversal attacks
4. WHEN storing configuration THEN the system SHALL use appropriate file permissions
5. WHEN executing subprocess THEN the system SHALL not inherit unnecessary environment variables

## Non-Functional Requirements

### Usability
- Installation via single binary or common package managers
- Clear, helpful error messages
- Comprehensive --help documentation
- Example configurations for common project types

### Compatibility
- Cross-platform support (Windows, macOS, Linux)
- Support for all major project types (Node.js, Go, Rust, Python, etc.)
- Compatible with Claude Code hook system
- Works with common CI/CD systems

### Maintainability
- Configuration-driven architecture with no hardcoded project logic
- Extensible plugin system for new commands
- Versioned configuration schema
- Comprehensive test coverage

### Documentation
- Quick start guide
- Configuration reference
- Example hook configurations
- Troubleshooting guide

---

Do the requirements look good? If so, we can move on to the design.