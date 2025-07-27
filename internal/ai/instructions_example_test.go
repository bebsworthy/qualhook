package ai_test

import (
	"errors"
	"fmt"
	"log"

	"github.com/bebsworthy/qualhook/internal/ai"
)

func ExampleGetInstallInstructions() {
	// Get installation instructions for Claude CLI
	instructions := ai.GetInstallInstructions("claude")
	fmt.Println(instructions)
}

func ExampleFormatToolNotFoundError() {
	// When a tool is not found, provide helpful instructions
	err := errors.New("command not found: claude")
	message := ai.FormatToolNotFoundError("claude", err)
	log.Print(message)
}

func ExampleFormatNoToolsAvailableError() {
	// When no AI tools are available, offer both options
	message := ai.FormatNoToolsAvailableError()
	fmt.Println(message)
}

func ExampleGetToolSelectionPrompt() {
	// When multiple tools are available, let user choose
	tools := []ai.Tool{
		{Name: "claude", Version: "1.0.0", Available: true},
		{Name: "gemini", Version: "2.3.4", Available: true},
	}

	prompt := ai.GetToolSelectionPrompt(tools)
	fmt.Print(prompt)
	// User would input their choice here
}
