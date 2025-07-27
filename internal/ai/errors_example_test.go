package ai_test

import (
	"context"
	"errors"
	"fmt"

	"github.com/bebsworthy/qualhook/internal/ai"
)

// Example_errorRecovery demonstrates how to handle AI errors with recovery suggestions
func Example_errorRecovery() {
	// Simulate an AI error scenario
	err := ai.NewErrorWithRecovery(
		ai.ErrTypeExecutionFailed,
		"Failed to connect to AI service",
		errors.New("connection timeout"),
		[]string{
			"Check your internet connection",
			"Try a different AI tool",
			"Use manual configuration",
		},
		nil,
	)

	// Format the error with recovery suggestions
	formatted := ai.FormatErrorWithSuggestions(err)
	fmt.Println(formatted)

	// Output:
	// Error: Failed to connect to AI service: connection timeout
	//
	// Suggested actions:
	//   1. Check your internet connection
	//   2. Try a different AI tool
	//   3. Use manual configuration
}

// Example_retryableError demonstrates checking if an error is retryable
func Example_retryableError() {
	// Test various error types
	errors := []error{
		ai.NewAIError(ai.ErrTypeTimeout, "Request timed out", nil),
		ai.NewAIError(ai.ErrTypeUserCanceled, "User canceled", nil),
		fmt.Errorf("network connection failed"),
	}

	for _, err := range errors {
		if ai.IsRetryableError(err) {
			fmt.Printf("%v is retryable\n", err)
		} else {
			fmt.Printf("%v is not retryable\n", err)
		}
	}

	// Output:
	// Request timed out is retryable
	// User canceled is not retryable
	// network connection failed is retryable
}

// Example_partialConfigExtraction demonstrates extracting partial configuration
func Example_partialConfigExtraction() {
	// Simulate a partial AI response
	response := `
	The project appears to use:
	- format: prettier --write
	- lint: eslint .
	- test: jest
	
	But I couldn't determine the typecheck command.
	`

	partialCommands, recoveryHint := ai.ExtractPartialConfig(response, nil)

	fmt.Printf("Found %d partial commands\n", len(partialCommands))
	if recoveryHint != "" {
		fmt.Println("Recovery hint available")
	}

	// Output:
	// Found 3 partial commands
	// Recovery hint available
}

// Example_networkErrorHandling demonstrates network error handling
func Example_networkErrorHandling() {
	// Simulate various network errors
	networkErr := errors.New("dial tcp: lookup api.example.com: no such host")

	// Handle the network error
	handled := ai.HandleNetworkError(networkErr)

	// Check if it was recognized as a network error
	var errWithRecovery *ai.ErrorWithRecovery
	if errors.As(handled, &errWithRecovery) {
		fmt.Println("Network error detected with recovery suggestions")
		fmt.Printf("Type: %v\n", errWithRecovery.Type)
		fmt.Printf("Suggestions: %d\n", len(errWithRecovery.RecoverySuggestions))
	}

	// Output:
	// Network error detected with recovery suggestions
	// Type: 2
	// Suggestions: 4
}

// Example_installInstructions demonstrates getting platform-specific installation instructions
func Example_installInstructions() {
	tools := []string{"claude", "gemini", "unknown"}

	for _, tool := range tools {
		instructions := ai.GetInstallInstructions(tool)
		fmt.Printf("%s: %v\n", tool, instructions != "")
	}

	// Output:
	// claude: true
	// gemini: true
	// unknown: true
}

// Example_errorSanitization demonstrates error message sanitization
func Example_errorSanitization() {
	// Create an error with sensitive information
	sensitiveErr := errors.New("Authentication failed with key sk-1234567890abcdef")

	// Sanitize the error
	sanitized := ai.SanitizeErrorMessage(sensitiveErr)

	// The sensitive information should be removed
	fmt.Println(sanitized.Error())

	// Output:
	// [REDACTED] (sensitive information removed)
}

// Example_assistantErrorHandling demonstrates how the assistant handles errors
func Example_assistantErrorHandling() {
	// This example shows the expected error handling flow
	ctx := context.Background()

	// Mock scenario where no AI tools are available
	// In real usage, this would be handled by the assistant
	options := ai.AIOptions{
		WorkingDir:  ".",
		Interactive: true,
	}

	// The assistant would detect no tools and return an appropriate error
	// This demonstrates the expected error type
	err := ai.NewErrorWithRecovery(
		ai.ErrTypeNoTools,
		"No AI tools detected",
		nil,
		ai.GetRecoverySuggestions(ai.ErrTypeNoTools),
		nil,
	)

	// Format and display the error
	if err != nil {
		fmt.Println("Error handling demonstration:")
		fmt.Printf("- Error type provides context\n")
		fmt.Printf("- Recovery suggestions guide next steps\n")
		fmt.Printf("- User can fallback to manual configuration\n")
	}

	_ = ctx
	_ = options

	// Output:
	// Error handling demonstration:
	// - Error type provides context
	// - Recovery suggestions guide next steps
	// - User can fallback to manual configuration
}
