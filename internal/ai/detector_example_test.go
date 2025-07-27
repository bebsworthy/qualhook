//go:build unit

package ai_test

import (
	"fmt"
	"time"

	"github.com/bebsworthy/qualhook/internal/ai"
	"github.com/bebsworthy/qualhook/internal/executor"
)

func ExampleNewToolDetector() {
	// Create a command executor with a 30-second timeout
	cmdExecutor := executor.NewCommandExecutor(30 * time.Second)

	// Create a tool detector
	detector := ai.NewToolDetector(cmdExecutor)

	// Detect available AI tools
	tools, err := detector.DetectTools()
	if err != nil {
		fmt.Printf("Error detecting tools: %v\n", err)
		return
	}

	// Check which tools are available
	for _, tool := range tools {
		if tool.Available {
			fmt.Printf("%s is available (version: %s)\n", tool.Name, tool.Version)
		} else {
			fmt.Printf("%s is not installed\n", tool.Name)
		}
	}

	// Check if a specific tool is available
	claudeAvailable, _ := detector.IsToolAvailable("claude")
	if !claudeAvailable {
		fmt.Println("\nTo use AI-assisted configuration, please install Claude CLI:")
		fmt.Println(ai.GetInstallInstructions("claude"))
	}
}

func ExampleFormatToolsStatus() {
	// Example tools detection results
	tools := []ai.Tool{
		{
			Name:      "claude",
			Command:   "claude",
			Version:   "1.2.3",
			Available: true,
		},
		{
			Name:      "gemini",
			Command:   "gemini",
			Available: false,
		},
	}

	// Format and display the status
	status := ai.FormatToolsStatus(tools)
	fmt.Println(status)

	// Output:
	// AI Tool Detection Results:
	//
	// claude:
	//   Status: ✓ Available
	//   Version: 1.2.3
	//   Command: claude
	//
	// gemini:
	//   Status: ✗ Not found
	//   Install: Run 'qualhook ai-config' for installation instructions
}

func ExampleGetAvailableTools() {
	// Example tools with mixed availability
	allTools := []ai.Tool{
		{Name: "claude", Available: true},
		{Name: "gemini", Available: false},
		{Name: "gpt", Available: true},
	}

	// Get only available tools
	availableTools := ai.GetAvailableTools(allTools)

	fmt.Printf("Available tools: %d of %d\n", len(availableTools), len(allTools))
	for _, tool := range availableTools {
		fmt.Printf("- %s\n", tool.Name)
	}

	// Output:
	// Available tools: 2 of 3
	// - claude
	// - gpt
}
