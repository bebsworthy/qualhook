// Package ai provides AI-powered configuration generation for qualhook.
package ai

import (
	"fmt"
	"runtime"
	"strings"
)

// GetInstallInstructions returns platform-specific installation instructions for the given AI tool
func GetInstallInstructions(toolName string) string {
	platform := runtime.GOOS
	
	switch strings.ToLower(toolName) {
	case "claude":
		return getClaudeInstructions(platform)
	case "gemini":
		return getGeminiInstructions(platform)
	default:
		return fmt.Sprintf("Unknown AI tool: %s", toolName)
	}
}

// getClaudeInstructions returns Claude CLI installation instructions for the platform
func getClaudeInstructions(platform string) string {
	var instructions strings.Builder
	
	instructions.WriteString("Claude CLI is not installed. Installation instructions:\n\n")
	
	switch platform {
	case "darwin": // macOS
		instructions.WriteString("macOS:\n")
		instructions.WriteString("  1. Install using Homebrew:\n")
		instructions.WriteString("     brew tap anthropics/tap\n")
		instructions.WriteString("     brew install claude\n\n")
		instructions.WriteString("  2. Or download from: https://claude.ai/cli\n")
		instructions.WriteString("     Then add to PATH\n\n")
		instructions.WriteString("  3. Authenticate:\n")
		instructions.WriteString("     claude auth login\n")
		
	case "linux":
		instructions.WriteString("Linux:\n")
		instructions.WriteString("  1. Download the latest release:\n")
		instructions.WriteString("     curl -L https://github.com/anthropics/claude-cli/releases/latest/download/claude-linux-amd64 -o claude\n")
		instructions.WriteString("     chmod +x claude\n")
		instructions.WriteString("     sudo mv claude /usr/local/bin/\n\n")
		instructions.WriteString("  2. Or use the install script:\n")
		instructions.WriteString("     curl -fsSL https://claude.ai/install.sh | sh\n\n")
		instructions.WriteString("  3. Authenticate:\n")
		instructions.WriteString("     claude auth login\n")
		
	case "windows":
		instructions.WriteString("Windows:\n")
		instructions.WriteString("  1. Using PowerShell (as Administrator):\n")
		instructions.WriteString("     irm https://claude.ai/install.ps1 | iex\n\n")
		instructions.WriteString("  2. Or download manually:\n")
		instructions.WriteString("     - Download from: https://github.com/anthropics/claude-cli/releases\n")
		instructions.WriteString("     - Extract claude.exe to a directory\n")
		instructions.WriteString("     - Add the directory to your PATH\n\n")
		instructions.WriteString("  3. Authenticate:\n")
		instructions.WriteString("     claude auth login\n")
		
	default:
		instructions.WriteString(fmt.Sprintf("Platform %s:\n", platform))
		instructions.WriteString("  Visit https://claude.ai/cli for installation instructions\n")
	}
	
	return instructions.String()
}

// getGeminiInstructions returns Gemini CLI installation instructions for the platform
func getGeminiInstructions(platform string) string {
	var instructions strings.Builder
	
	instructions.WriteString("Gemini CLI is not installed. Installation instructions:\n\n")
	
	switch platform {
	case "darwin": // macOS
		instructions.WriteString("macOS:\n")
		instructions.WriteString("  1. Install using Homebrew:\n")
		instructions.WriteString("     brew tap google/gemini\n")
		instructions.WriteString("     brew install gemini\n\n")
		instructions.WriteString("  2. Or install via npm:\n")
		instructions.WriteString("     npm install -g @google/gemini-cli\n\n")
		instructions.WriteString("  3. Authenticate:\n")
		instructions.WriteString("     gemini auth login\n")
		instructions.WriteString("     # Or set API key:\n")
		instructions.WriteString("     export GEMINI_API_KEY=\"your-api-key\"\n")
		
	case "linux":
		instructions.WriteString("Linux:\n")
		instructions.WriteString("  1. Install via npm:\n")
		instructions.WriteString("     npm install -g @google/gemini-cli\n\n")
		instructions.WriteString("  2. Or download binary:\n")
		instructions.WriteString("     curl -L https://storage.googleapis.com/gemini-cli/latest/gemini-linux-amd64 -o gemini\n")
		instructions.WriteString("     chmod +x gemini\n")
		instructions.WriteString("     sudo mv gemini /usr/local/bin/\n\n")
		instructions.WriteString("  3. Authenticate:\n")
		instructions.WriteString("     gemini auth login\n")
		instructions.WriteString("     # Or set API key:\n")
		instructions.WriteString("     export GEMINI_API_KEY=\"your-api-key\"\n")
		
	case "windows":
		instructions.WriteString("Windows:\n")
		instructions.WriteString("  1. Using npm:\n")
		instructions.WriteString("     npm install -g @google/gemini-cli\n\n")
		instructions.WriteString("  2. Or using PowerShell:\n")
		instructions.WriteString("     Invoke-WebRequest -Uri https://storage.googleapis.com/gemini-cli/latest/gemini-windows-amd64.exe -OutFile gemini.exe\n")
		instructions.WriteString("     # Add to PATH manually\n\n")
		instructions.WriteString("  3. Authenticate:\n")
		instructions.WriteString("     gemini auth login\n")
		instructions.WriteString("     # Or set API key:\n")
		instructions.WriteString("     set GEMINI_API_KEY=your-api-key\n")
		
	default:
		instructions.WriteString(fmt.Sprintf("Platform %s:\n", platform))
		instructions.WriteString("  Visit https://ai.google/tools/gemini-cli for installation instructions\n")
	}
	
	return instructions.String()
}

// FormatToolNotFoundError formats an error message when an AI tool is not found
func FormatToolNotFoundError(toolName string, err error) string {
	var msg strings.Builder
	
	msg.WriteString(fmt.Sprintf("AI tool '%s' not found", toolName))
	
	if err != nil {
		msg.WriteString(fmt.Sprintf(": %v", err))
	}
	
	msg.WriteString("\n\n")
	msg.WriteString(GetInstallInstructions(toolName))
	msg.WriteString("\nAfter installation, run this command again.")
	
	return msg.String()
}

// FormatNoToolsAvailableError formats an error when no AI tools are available
func FormatNoToolsAvailableError() string {
	var msg strings.Builder
	
	msg.WriteString("No AI tools available. Please install Claude or Gemini CLI.\n\n")
	msg.WriteString("Option 1: Install Claude CLI\n")
	msg.WriteString(strings.TrimPrefix(GetInstallInstructions("claude"), "Claude CLI is not installed. Installation instructions:\n\n"))
	msg.WriteString("\n")
	msg.WriteString("Option 2: Install Gemini CLI\n")
	msg.WriteString(strings.TrimPrefix(GetInstallInstructions("gemini"), "Gemini CLI is not installed. Installation instructions:\n\n"))
	
	return msg.String()
}

// GetToolSelectionPrompt formats a prompt for tool selection
func GetToolSelectionPrompt(availableTools []Tool) string {
	var msg strings.Builder
	
	msg.WriteString("Multiple AI tools are available. Please select one:\n\n")
	
	for i, tool := range availableTools {
		msg.WriteString(fmt.Sprintf("%d. %s", i+1, tool.Name))
		if tool.Version != "" {
			msg.WriteString(fmt.Sprintf(" (v%s)", tool.Version))
		}
		msg.WriteString("\n")
	}
	
	msg.WriteString("\nEnter your choice (1-%d): ")
	
	return fmt.Sprintf(msg.String(), len(availableTools))
}

// GetHelpDocumentation returns help documentation about AI tools for the help command
func GetHelpDocumentation() string {
	return `AI-Assisted Configuration

Qualhook can use AI tools (Claude or Gemini) to automatically analyze your project
and generate appropriate quality check configurations.

Commands:
  qualhook ai-config          Generate complete configuration using AI
  qualhook ai-config --tool   Specify which AI tool to use (claude/gemini)
  qualhook config             Interactive wizard with AI assistance option

Requirements:
  - Claude CLI: Install from https://claude.ai/cli
  - Gemini CLI: Install from https://ai.google/tools/gemini-cli

The AI will analyze your project structure, detect build tools and frameworks,
and suggest appropriate commands for formatting, linting, type checking, and testing.

Example:
  $ qualhook ai-config
  $ qualhook ai-config --tool claude
  $ qualhook config  # Select "AI assistance" when prompted

Security:
  - AI tools run locally with restricted permissions
  - All suggested commands require user approval
  - Sensitive files (.env, credentials) are excluded from analysis`
}