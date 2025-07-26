package ai_test

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/bebsworthy/qualhook/internal/ai"
	"github.com/bebsworthy/qualhook/internal/executor"
)

func ExampleAssistant_GenerateConfig() {
	// Create command executor with timeout
	exec := executor.NewCommandExecutor(2 * time.Minute)
	
	// Create AI assistant
	assistant := ai.NewAssistant(exec)
	
	// Configure AI options
	options := ai.AIOptions{
		Tool:         "",           // Let user select
		WorkingDir:   "/my/project", // Project to analyze
		Interactive:  true,         // Show progress
		TestCommands: true,         // Test generated commands
		Timeout:      60 * time.Second,
	}
	
	// Generate configuration
	ctx := context.Background()
	config, err := assistant.GenerateConfig(ctx, options)
	if err != nil {
		log.Fatalf("Failed to generate config: %v", err)
	}
	
	// Use the generated configuration
	fmt.Printf("Generated config with %d commands\n", len(config.Commands))
	for name, cmd := range config.Commands {
		fmt.Printf("  %s: %s %v\n", name, cmd.Command, cmd.Args)
	}
}

func ExampleAssistant_SuggestCommand() {
	// Create command executor
	exec := executor.NewCommandExecutor(30 * time.Second)
	
	// Create AI assistant
	assistant := ai.NewAssistant(exec)
	
	// Define project context
	projectInfo := ai.ProjectContext{
		ProjectType: "nodejs",
		CustomCommands: []string{"build", "deploy"},
	}
	
	// Get suggestion for format command
	ctx := context.Background()
	suggestion, err := assistant.SuggestCommand(ctx, "format", projectInfo)
	if err != nil {
		log.Fatalf("Failed to get suggestion: %v", err)
	}
	
	// Use the suggestion
	fmt.Printf("Suggested command: %s %v\n", suggestion.Command, suggestion.Args)
	fmt.Printf("Explanation: %s\n", suggestion.Explanation)
}

func ExampleAIError() {
	// Create an AI error
	err := ai.NewAIError(
		ai.ErrTypeNoTools,
		"No AI tools available",
		nil,
	)
	
	// Check error type - err is already *ai.AIError
	switch err.Type {
		case ai.ErrTypeNoTools:
			fmt.Println("Please install Claude or Gemini CLI")
		case ai.ErrTypeTimeout:
			fmt.Println("AI analysis timed out")
		case ai.ErrTypeUserCanceled:
			fmt.Println("Operation canceled by user")
	}
	
	// Output:
	// Please install Claude or Gemini CLI
}

func ExampleAssistant_GenerateConfig_monorepo() {
	// Example for monorepo project
	exec := executor.NewCommandExecutor(3 * time.Minute)
	assistant := ai.NewAssistant(exec)
	
	// Configure for monorepo analysis
	options := ai.AIOptions{
		Tool:         "claude",
		WorkingDir:   "/my/monorepo",
		Interactive:  true,
		TestCommands: false, // Skip testing for example
		Timeout:      2 * time.Minute,
	}
	
	ctx := context.Background()
	config, err := assistant.GenerateConfig(ctx, options)
	if err != nil {
		log.Fatalf("Failed: %v", err)
	}
	
	// Check for workspace-specific configurations
	if len(config.Paths) > 0 {
		fmt.Printf("Detected monorepo with %d workspaces\n", len(config.Paths))
		for _, path := range config.Paths {
			fmt.Printf("  Workspace: %s\n", path.Path)
		}
	}
}