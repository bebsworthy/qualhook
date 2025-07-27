package ai

import (
	"runtime"
	"strings"
	"testing"
)

const (
	osDarwin  = "darwin"
	osLinux   = "linux"
	osWindows = "windows"
)

func TestGetInstallInstructions(t *testing.T) {
	tests := []struct {
		name     string
		tool     string
		contains []string
	}{
		{
			name: "Claude instructions",
			tool: "claude",
			contains: []string{
				"Claude CLI is not installed",
				"Installation instructions:",
				"Authenticate:",
				"claude auth login",
			},
		},
		{
			name: "Gemini instructions",
			tool: "gemini",
			contains: []string{
				"Gemini CLI is not installed",
				"Installation instructions:",
				"Authenticate:",
				"gemini auth login",
			},
		},
		{
			name: "Unknown tool",
			tool: "unknown",
			contains: []string{
				"Unknown AI tool: unknown",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetInstallInstructions(tt.tool)

			for _, expected := range tt.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("Expected result to contain %q, but it didn't.\nGot: %s", expected, result)
				}
			}
		})
	}
}

func TestGetInstallInstructions_PlatformSpecific(t *testing.T) {
	platform := runtime.GOOS

	// Test Claude
	claudeInstructions := GetInstallInstructions("claude")

	switch platform {
	case osDarwin:
		if !strings.Contains(claudeInstructions, "brew") {
			t.Error("macOS Claude instructions should mention Homebrew")
		}
	case osLinux:
		if !strings.Contains(claudeInstructions, "curl") {
			t.Error("Linux Claude instructions should mention curl")
		}
	case osWindows:
		if !strings.Contains(claudeInstructions, "PowerShell") {
			t.Error("Windows Claude instructions should mention PowerShell")
		}
	}

	// Test Gemini
	geminiInstructions := GetInstallInstructions("gemini")

	switch platform {
	case osDarwin:
		if !strings.Contains(geminiInstructions, "brew") && !strings.Contains(geminiInstructions, "npm") {
			t.Error("macOS Gemini instructions should mention Homebrew or npm")
		}
	case osLinux:
		if !strings.Contains(geminiInstructions, "npm") && !strings.Contains(geminiInstructions, "curl") {
			t.Error("Linux Gemini instructions should mention npm or curl")
		}
	case osWindows:
		if !strings.Contains(geminiInstructions, "npm") {
			t.Error("Windows Gemini instructions should mention npm")
		}
	}
}

func TestFormatToolNotFoundError(t *testing.T) {
	tests := []struct {
		name     string
		tool     string
		err      error
		contains []string
	}{
		{
			name: "Claude not found",
			tool: "claude",
			err:  nil,
			contains: []string{
				"AI tool 'claude' not found",
				"Claude CLI is not installed",
				"After installation, run this command again",
			},
		},
		{
			name: "Gemini not found with error",
			tool: "gemini",
			err:  &testError{"command not found"},
			contains: []string{
				"AI tool 'gemini' not found: command not found",
				"Gemini CLI is not installed",
				"After installation, run this command again",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatToolNotFoundError(tt.tool, tt.err)

			for _, expected := range tt.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("Expected result to contain %q, but it didn't.\nGot: %s", expected, result)
				}
			}
		})
	}
}

func TestFormatNoToolsAvailableError(t *testing.T) {
	result := FormatNoToolsAvailableError()

	expectedContents := []string{
		"No AI tools available",
		"Option 1: Install Claude CLI",
		"Option 2: Install Gemini CLI",
		"claude auth login",
		"gemini auth login",
	}

	for _, expected := range expectedContents {
		if !strings.Contains(result, expected) {
			t.Errorf("Expected result to contain %q, but it didn't.\nGot: %s", expected, result)
		}
	}

	// Should not contain the redundant "is not installed" messages
	if strings.Contains(result, "Claude CLI is not installed") {
		t.Error("Should not contain redundant 'Claude CLI is not installed' message")
	}
	if strings.Contains(result, "Gemini CLI is not installed") {
		t.Error("Should not contain redundant 'Gemini CLI is not installed' message")
	}
}

func TestGetToolSelectionPrompt(t *testing.T) {
	tools := []Tool{
		{Name: "claude", Version: "1.0.0", Available: true},
		{Name: "gemini", Version: "2.3.4", Available: true},
	}

	result := GetToolSelectionPrompt(tools)

	expectedContents := []string{
		"Multiple AI tools are available",
		"1. claude (v1.0.0)",
		"2. gemini (v2.3.4)",
		"Enter your choice (1-2):",
	}

	for _, expected := range expectedContents {
		if !strings.Contains(result, expected) {
			t.Errorf("Expected result to contain %q, but it didn't.\nGot: %s", expected, result)
		}
	}
}

func TestGetHelpDocumentation(t *testing.T) {
	result := GetHelpDocumentation()

	expectedContents := []string{
		"AI-Assisted Configuration",
		"qualhook ai-config",
		"qualhook config",
		"Claude CLI:",
		"Gemini CLI:",
		"Security:",
		"Example:",
	}

	for _, expected := range expectedContents {
		if !strings.Contains(result, expected) {
			t.Errorf("Expected help documentation to contain %q, but it didn't", expected)
		}
	}
}

// testError is a simple error type for testing
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
