# Quality Hook Feature Context

## Feature Name
quality-hook

## Description
A configurable command-line utility that serves as Claude Code hooks to enforce code quality for LLM coding agents. The tool wraps project-specific commands (format, lint, typecheck, test) and intelligently filters their output to provide relevant error information to the LLM.

## Key Requirements
- Support multiple project types (Node.js, Go, Kotlin, Rust, etc.) through configuration
- Support monorepos with multiple technologies (e.g., Node.js/Vite/React frontend, Go backend, Rust service, Jupyter notebooks)
- File-aware execution: automatically run only the relevant quality checks based on which files were edited
- Auto-detection and configuration of project type with user confirmation
- Extensible command system allowing custom commands
- Intelligent output filtering to extract only relevant error information
- Return exit code 2 when errors need LLM attention
- Configuration-driven architecture with no hardcoded project type logic

## Integration Points
- Claude Code hooks system
- Project-specific tooling (linters, formatters, test runners)
- Configuration file (JSON)

## External References
- https://docs.anthropic.com/en/docs/claude-code/hooks-guide
- https://docs.anthropic.com/en/docs/claude-code/hooks