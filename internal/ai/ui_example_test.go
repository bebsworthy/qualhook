package ai_test

import (
	"fmt"

	"github.com/bebsworthy/qualhook/internal/ai"
	"github.com/bebsworthy/qualhook/pkg/config"
)

func ExampleInteractiveUI_ReviewConfiguration() {
	ui := ai.NewInteractiveUI()

	// Example configuration to review
	cfg := &config.Config{
		ProjectType: "nodejs",
		Commands: map[string]*config.CommandConfig{
			"format": {
				Command: "prettier",
				Args:    []string{"--write", "."},
			},
			"lint": {
				Command: "eslint",
				Args:    []string{".", "--fix"},
			},
			"test": {
				Command: "jest",
			},
		},
	}

	// Review the configuration (in real usage, this would display to stdout)
	_ = ui.ReviewConfiguration(cfg)

	// Note: Output is captured by the test framework
}

func ExampleInteractiveUI_DisplayCommandComparison() {
	ui := ai.NewInteractiveUI()

	original := &config.CommandConfig{
		Command: "npm",
		Args:    []string{"run", "test"},
	}

	suggested := &config.CommandConfig{
		Command: "jest",
		Args:    []string{"--coverage"},
		ErrorPatterns: []*config.RegexPattern{
			{Pattern: "FAIL", Flags: ""},
		},
	}

	// Compare original and suggested commands
	_ = ui.DisplayCommandComparison(original, suggested, "test")
}

func ExampleInteractiveUI_ShowAIError() {
	ui := ai.NewInteractiveUI()

	// Show different types of errors with context
	err := fmt.Errorf("claude not found in PATH")
	ui.ShowAIError(err, "claude")

	// The output would include installation instructions
}

func ExampleInteractiveUI_ShowInstallInstructions() {
	ui := ai.NewInteractiveUI()

	// Show installation instructions for different tools
	ui.ShowInstallInstructions("claude")
	fmt.Println()
	ui.ShowInstallInstructions("gemini")
}