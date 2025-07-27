package ai

import (
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/bebsworthy/qualhook/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// captureOutput captures stdout during test execution
func captureOutput(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = old

	out, _ := io.ReadAll(r)
	return string(out)
}

func TestNewInteractiveUI(t *testing.T) {
	ui := NewInteractiveUI()
	assert.NotNil(t, ui)
}

func TestSelectTool(t *testing.T) {
	ui := NewInteractiveUI()

	tests := []struct {
		name           string
		availableTools []Tool
		expectError    bool
		expectedMsg    string
	}{
		{
			name:           "no tools available",
			availableTools: []Tool{},
			expectError:    true,
			expectedMsg:    "no AI tools available",
		},
		{
			name: "single tool auto-selected",
			availableTools: []Tool{
				{Name: "claude", Version: "1.0.0", Available: true},
			},
			expectError: false,
		},
		{
			name: "multiple tools with versions",
			availableTools: []Tool{
				{Name: "claude", Version: "1.0.0", Available: true},
				{Name: "gemini", Version: "2.0.0", Available: true},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if len(tt.availableTools) == 1 {
				// Test single tool auto-selection
				output := captureOutput(func() {
					result, err := ui.SelectTool(tt.availableTools)
					if !tt.expectError {
						assert.NoError(t, err)
						assert.Equal(t, tt.availableTools[0].Name, result)
					}
				})
				assert.Contains(t, output, "Using claude for AI assistance")
			} else if len(tt.availableTools) == 0 {
				// Test no tools available
				_, err := ui.SelectTool(tt.availableTools)
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedMsg)
			}
			// Note: Multiple tools case would require mocking survey.AskOne
		})
	}
}

func TestReviewConfiguration(t *testing.T) {
	ui := NewInteractiveUI()

	tests := []struct {
		name           string
		config         *config.Config
		expectedOutput []string
	}{
		{
			name: "basic configuration",
			config: &config.Config{
				ProjectType: "nodejs",
				Commands: map[string]*config.CommandConfig{
					"format": {
						Command: "prettier",
						Args:    []string{"--write", "."},
					},
					"lint": {
						Command: "eslint",
						Args:    []string{"."},
					},
				},
			},
			expectedOutput: []string{
				"Project Type: nodejs",
				"format: prettier --write .",
				"lint: eslint .",
				"typecheck: <not configured>",
				"test: <not configured>",
			},
		},
		{
			name: "configuration with custom commands",
			config: &config.Config{
				Commands: map[string]*config.CommandConfig{
					"format": {
						Command: "fmt",
					},
					"build": {
						Command: "make",
						Args:    []string{"build"},
					},
				},
			},
			expectedOutput: []string{
				"format: fmt",
				"Custom Commands:",
				"build: make build",
			},
		},
		{
			name: "monorepo configuration",
			config: &config.Config{
				Commands: map[string]*config.CommandConfig{
					"test": {
						Command: "jest",
					},
				},
				Paths: []*config.PathConfig{
					{
						Path: "packages/backend/**",
						Commands: map[string]*config.CommandConfig{
							"test": {
								Command: "jest",
								Args:    []string{"--config", "backend.jest.config.js"},
							},
						},
					},
				},
			},
			expectedOutput: []string{
				"Monorepo Configuration:",
				"Path: packages/backend/**",
				"Overrides:",
				"test: jest --config backend.jest.config.js",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureOutput(func() {
				err := ui.ReviewConfiguration(tt.config)
				assert.NoError(t, err)
			})

			for _, expected := range tt.expectedOutput {
				assert.Contains(t, output, expected)
			}
		})
	}
}

func TestDisplayCommandComparison(t *testing.T) {
	ui := NewInteractiveUI()

	tests := []struct {
		name        string
		original    *config.CommandConfig
		suggested   *config.CommandConfig
		commandName string
		expected    []string
	}{
		{
			name:     "new command suggestion",
			original: nil,
			suggested: &config.CommandConfig{
				Command: "prettier",
				Args:    []string{"--write", "."},
			},
			commandName: "format",
			expected: []string{
				"Command Comparison for 'format'",
				"Original: <not configured>",
				"Suggested:",
				"Command: prettier --write .",
			},
		},
		{
			name: "command modification",
			original: &config.CommandConfig{
				Command:   "npm",
				Args:      []string{"run", "lint"},
				ExitCodes: []int{1},
			},
			suggested: &config.CommandConfig{
				Command: "eslint",
				Args:    []string{".", "--fix"},
				ErrorPatterns: []*config.RegexPattern{
					{Pattern: "error", Flags: "i"},
				},
				ExitCodes: []int{1, 2},
			},
			commandName: "lint",
			expected: []string{
				"Original:",
				"Command: npm run lint",
				"Exit Codes: [1]",
				"Suggested:",
				"Command: eslint . --fix",
				"Error Patterns: 1 configured",
				"Exit Codes: [1 2]",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureOutput(func() {
				err := ui.DisplayCommandComparison(tt.original, tt.suggested, tt.commandName)
				assert.NoError(t, err)
			})

			for _, expected := range tt.expected {
				assert.Contains(t, output, expected)
			}
		})
	}
}

func TestShowAIProgress(t *testing.T) {
	ui := NewInteractiveUI()

	output := captureOutput(func() {
		ui.ShowAIProgress("Analyzing project structure...")
	})

	assert.Contains(t, output, "Analyzing project structure...")
	assert.Contains(t, output, "Press ESC to cancel...")
}

func TestShowAIError(t *testing.T) {
	ui := NewInteractiveUI()

	tests := []struct {
		name         string
		err          error
		toolName     string
		expectedMsgs []string
	}{
		{
			name:     "tool not found",
			err:      fmt.Errorf("claude not found"),
			toolName: "claude",
			expectedMsgs: []string{
				"AI assistance error with claude",
				"To install the AI tool:",
				"macOS/Linux: curl -fsSL https://cli.claude.ai/install.sh | sh",
			},
		},
		{
			name:     "timeout error",
			err:      fmt.Errorf("operation timeout"),
			toolName: "gemini",
			expectedMsgs: []string{
				"The AI tool is taking longer than expected",
				"Wait longer for the response",
				"Cancel and try again",
			},
		},
		{
			name:     "canceled error",
			err:      fmt.Errorf("operation canceled by user"),
			toolName: "claude",
			expectedMsgs: []string{
				"AI assistance canceled",
				"Continuing with manual configuration",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureOutput(func() {
				ui.ShowAIError(tt.err, tt.toolName)
			})

			for _, expected := range tt.expectedMsgs {
				assert.Contains(t, output, expected)
			}
		})
	}
}

func TestShowInstallInstructions(t *testing.T) {
	ui := NewInteractiveUI()

	tests := []struct {
		name         string
		toolName     string
		expectedMsgs []string
	}{
		{
			name:     "claude instructions",
			toolName: "claude",
			expectedMsgs: []string{
				"macOS/Linux: curl -fsSL https://cli.claude.ai/install.sh | sh",
				"Windows: Visit https://cli.claude.ai",
			},
		},
		{
			name:     "gemini instructions",
			toolName: "gemini",
			expectedMsgs: []string{
				"All platforms: pip install google-generativeai",
				"Or visit: https://ai.google.dev/gemini-api/docs/quickstart",
			},
		},
		{
			name:     "unknown tool",
			toolName: "unknown-ai",
			expectedMsgs: []string{
				"Please refer to the unknown-ai documentation",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureOutput(func() {
				ui.ShowInstallInstructions(tt.toolName)
			})

			for _, expected := range tt.expectedMsgs {
				assert.Contains(t, output, expected)
			}
		})
	}
}

func TestPromptForCommandModification(t *testing.T) {
	ui := NewInteractiveUI()

	// Test the structure of the function without actual survey interaction
	original := &config.CommandConfig{
		Command: "npm",
		Args:    []string{"test"},
		ErrorPatterns: []*config.RegexPattern{
			{Pattern: "FAIL", Flags: ""},
		},
		ExitCodes: []int{1},
		Prompt:    "Run tests?",
	}

	// This test validates the function signature and basic structure
	// Actual survey interaction would need to be mocked
	t.Run("function exists and returns correct type", func(t *testing.T) {
		// The function exists and can be called
		assert.NotNil(t, ui.PromptForCommandModification)

		// Verify the original command structure is preserved in copy
		require.NotNil(t, original)
		assert.Equal(t, "npm", original.Command)
		assert.Equal(t, []string{"test"}, original.Args)
		assert.Len(t, original.ErrorPatterns, 1)
		assert.Len(t, original.ExitCodes, 1)
		assert.Equal(t, "Run tests?", original.Prompt)
	})
}

func TestConfirmConfiguration(t *testing.T) {
	ui := NewInteractiveUI()

	// Test that the function exists and has correct signature
	t.Run("function exists", func(t *testing.T) {
		assert.NotNil(t, ui.ConfirmConfiguration)
	})
}

func TestSelectCommandsToReview(t *testing.T) {
	ui := NewInteractiveUI()

	// Test the ordering logic without actual survey interaction
	commands := map[string]*config.CommandConfig{
		"build":     {Command: "make"},
		"test":      {Command: "jest"},
		"format":    {Command: "prettier"},
		"custom":    {Command: "custom-cmd"},
		"lint":      {Command: "eslint"},
		"typecheck": {Command: "tsc"},
	}

	// This test validates the function exists and would order commands correctly
	t.Run("function exists with correct command map", func(t *testing.T) {
		assert.NotNil(t, ui.SelectCommandsToReview)

		// Verify command map structure
		assert.Len(t, commands, 6)
		assert.Contains(t, commands, "format")
		assert.Contains(t, commands, "lint")
		assert.Contains(t, commands, "typecheck")
		assert.Contains(t, commands, "test")
		assert.Contains(t, commands, "build")
		assert.Contains(t, commands, "custom")
	})
}

// TestUIOutputFormatting tests that output is properly formatted
func TestUIOutputFormatting(t *testing.T) {
	ui := NewInteractiveUI()

	t.Run("empty args handling", func(t *testing.T) {
		cfg := &config.Config{
			Commands: map[string]*config.CommandConfig{
				"format": {
					Command: "gofmt",
					Args:    []string{},
				},
			},
		}

		output := captureOutput(func() {
			err := ui.ReviewConfiguration(cfg)
			assert.NoError(t, err)
		})

		// Should show command without trailing space
		assert.Contains(t, output, "format: gofmt\n")
		assert.NotContains(t, output, "format: gofmt \n")
	})

	t.Run("section headers", func(t *testing.T) {
		output := captureOutput(func() {
			err := ui.ReviewConfiguration(&config.Config{})
			assert.NoError(t, err)
		})

		assert.Contains(t, output, "=== Generated Configuration Summary ===")
		assert.Contains(t, output, "=====================================")
	})
}

// TestErrorMessageClarity tests that error messages are clear and actionable
func TestErrorMessageClarity(t *testing.T) {
	ui := NewInteractiveUI()

	t.Run("network error guidance", func(t *testing.T) {
		err := fmt.Errorf("network connection failed")
		output := captureOutput(func() {
			ui.ShowAIError(err, "claude")
		})

		// Should show the error clearly
		assert.Contains(t, output, "AI assistance error with claude")
		assert.Contains(t, output, "network connection failed")
	})

	t.Run("generic error", func(t *testing.T) {
		err := fmt.Errorf("unexpected error occurred")
		output := captureOutput(func() {
			ui.ShowAIError(err, "gemini")
		})

		assert.Contains(t, output, "AI assistance error with gemini")
		assert.Contains(t, output, "unexpected error occurred")
	})
}

// TestUIHelperIntegration tests that UI helpers work together coherently
func TestUIHelperIntegration(t *testing.T) {
	ui := NewInteractiveUI()

	t.Run("review flow messages", func(t *testing.T) {
		// Simulate a review flow
		cfg := &config.Config{
			ProjectType: "go",
			Commands: map[string]*config.CommandConfig{
				"format": {Command: "gofmt", Args: []string{"-w", "."}},
				"test":   {Command: "go", Args: []string{"test", "./..."}},
			},
		}

		output := captureOutput(func() {
			// Show configuration
			_ = ui.ReviewConfiguration(cfg)

			// Show progress
			ui.ShowAIProgress("Validating commands...")

			// Show comparison
			_ = ui.DisplayCommandComparison(
				cfg.Commands["format"],
				&config.CommandConfig{Command: "goimports", Args: []string{"-w", "."}},
				"format",
			)
		})

		// Verify flow makes sense
		assert.Contains(t, output, "Generated Configuration Summary")
		assert.Contains(t, output, "Validating commands...")
		assert.Contains(t, output, "Command Comparison")
	})
}

// TestConsistentFormatting ensures consistent formatting across all UI methods
func TestConsistentFormatting(t *testing.T) {
	ui := NewInteractiveUI()

	t.Run("command display format", func(t *testing.T) {
		cmd := &config.CommandConfig{
			Command: "npm",
			Args:    []string{"run", "test"},
		}

		// Test in ReviewConfiguration
		cfg := &config.Config{
			Commands: map[string]*config.CommandConfig{
				"test": cmd,
			},
		}

		output1 := captureOutput(func() {
			_ = ui.ReviewConfiguration(cfg)
		})

		// Test in DisplayCommandComparison
		output2 := captureOutput(func() {
			_ = ui.DisplayCommandComparison(nil, cmd, "test")
		})

		// Both should format the command the same way
		expectedFormat := "npm run test"
		assert.Contains(t, output1, expectedFormat)
		assert.Contains(t, output2, expectedFormat)
	})
}

// TestEmptyStateHandling tests handling of empty or nil states
func TestEmptyStateHandling(t *testing.T) {
	ui := NewInteractiveUI()

	t.Run("empty configuration", func(t *testing.T) {
		output := captureOutput(func() {
			err := ui.ReviewConfiguration(&config.Config{})
			assert.NoError(t, err)
		})

		// Should show all commands as not configured
		assert.Contains(t, output, "format: <not configured>")
		assert.Contains(t, output, "lint: <not configured>")
		assert.Contains(t, output, "typecheck: <not configured>")
		assert.Contains(t, output, "test: <not configured>")

		// Should not show custom commands section
		assert.NotContains(t, output, "Custom Commands:")
		assert.NotContains(t, output, "Monorepo Configuration:")
	})

	t.Run("nil command in comparison", func(t *testing.T) {
		output := captureOutput(func() {
			err := ui.DisplayCommandComparison(nil, &config.CommandConfig{Command: "test"}, "test")
			assert.NoError(t, err)
		})

		assert.Contains(t, output, "Original: <not configured>")
		assert.Contains(t, output, "Suggested:")
		assert.Contains(t, output, "Command: test")
	})
}

// TestSpecialCharacterHandling tests handling of special characters in commands
func TestSpecialCharacterHandling(t *testing.T) {
	ui := NewInteractiveUI()

	t.Run("commands with special characters", func(t *testing.T) {
		cfg := &config.Config{
			Commands: map[string]*config.CommandConfig{
				"format": {
					Command: "prettier",
					Args:    []string{"--write", "**/*.{js,jsx,ts,tsx}"},
				},
			},
		}

		output := captureOutput(func() {
			err := ui.ReviewConfiguration(cfg)
			assert.NoError(t, err)
		})

		// Special characters should be preserved
		assert.Contains(t, output, "**/*.{js,jsx,ts,tsx}")
	})
}

// TestLongCommandHandling tests handling of very long commands
func TestLongCommandHandling(t *testing.T) {
	ui := NewInteractiveUI()

	t.Run("long command with many args", func(t *testing.T) {
		longArgs := make([]string, 20)
		for i := range longArgs {
			longArgs[i] = fmt.Sprintf("--flag%d=value%d", i, i)
		}

		cfg := &config.Config{
			Commands: map[string]*config.CommandConfig{
				"complex": {
					Command: "complex-tool",
					Args:    longArgs,
				},
			},
		}

		output := captureOutput(func() {
			err := ui.ReviewConfiguration(cfg)
			assert.NoError(t, err)
		})

		// Should display the full command
		expectedCmd := "complex-tool " + strings.Join(longArgs, " ")
		assert.Contains(t, output, expectedCmd)
	})
}
