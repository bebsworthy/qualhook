# Feature: AI-Assisted Configuration

## Feature Name
ai-assisted-config

## Description
Enable AI-powered configuration of .qualhook.json files using Claude or Gemini to automatically detect and configure the appropriate commands for a project. This feature addresses the challenge that default configurations often don't match project-specific requirements (e.g., using `make test` instead of `go test ./...`).

## Key Capabilities
1. **Interactive Configuration Wizard Enhancement**
   - Mandatory review of all command types (standard and custom)
   - Option to use AI assistance for command suggestions
   - Add new custom commands beyond the standard four
   - Test-run commands before saving configuration
   - User selection of AI tool (Claude or Gemini) with explicit consent
   - Cancel AI analysis at any time with ESC key

2. **AI Config Command**
   - New `ai-config` command for fully automated configuration
   - AI tools run directly in project directory to analyze codebase
   - Complete monorepo discovery with per-workspace configurations
   - Generates complete .qualhook.json with proper commands and inheritance
   - Test-run validation of suggested commands before saving
   - Progress indicators with elapsed time and cancellation option

3. **AI Integration Approach**
   - AI tools analyze project directly rather than receiving pre-analyzed data
   - Clear prompts instruct AI to identify languages, build systems, and commands
   - AI handles monorepo detection and workspace-specific configurations
   - Response validation includes actual command execution with user approval
   - Graceful fallback to manual configuration on any failure

## Problem Statement
- Current default configurations often don't match project-specific build systems
- Manual maintenance of .qualhook.json is tedious as projects evolve
- Monorepos require complex path-specific configurations
- Users need to understand their project's build toolchain to configure qualhook properly

## Target Users
- Developers working on diverse projects with different build systems
- Teams managing monorepos with multiple technologies
- Users who want automated, accurate configuration without manual research

## User Control & Privacy
- No AI tool runs without explicit user permission
- Users can cancel AI execution at any time
- AI instructed to respect .gitignore and avoid sensitive files
- Command testing requires user approval before execution
- Clear progress indicators and elapsed time display