// Package ai provides AI-powered configuration generation for qualhook.
package ai

import (
	"errors"
	"fmt"
	"strings"
)

// Common error messages with helpful next steps
const (
	msgNoToolsAvailable = `No AI tools detected. To use AI-assisted configuration, please install one of the following:

Claude CLI:
  • macOS/Linux: brew install claude
  • Visit: https://claude.ai/cli for other platforms
  
Gemini CLI:
  • All platforms: npm install -g @google/generative-ai-cli
  • Visit: https://ai.google.dev/gemini-api/docs/cli

After installation, run 'qualhook ai-config' again.`

	// msgToolNotFound - removed, was unused

	msgExecutionFailed = `Failed to execute AI tool: %s

Possible causes:
• Network connectivity issues
• AI service temporarily unavailable
• Invalid API credentials

Next steps:
1. Check your internet connection
2. Verify your AI tool credentials are configured
3. Try again with --tool flag to select a different AI tool
4. Use 'qualhook config' for manual configuration`

	msgResponseInvalid = `The AI tool returned an invalid or unparseable response.

This can happen when:
• The AI service is overloaded
• The project structure is too complex
• Network issues corrupted the response

Next steps:
1. Try running the command again
2. Use a different AI tool with --tool flag
3. Use 'qualhook config' for guided manual configuration
4. Check the debug log for more details: qualhook --debug ai-config`

	// msgTimeout - removed, was unused

	// msgValidationFailed - removed, was unused

	msgPartialSuccess = `Partial configuration extracted from AI response.

Successfully configured:
%s

Failed to configure:
%s

You can:
1. Accept the partial configuration and add missing commands manually
2. Try again with a different AI tool
3. Use 'qualhook config' to complete the configuration`
)

// ErrorWithRecovery provides detailed error information with recovery suggestions
type ErrorWithRecovery struct {
	*AIError
	RecoverySuggestions []string
	PartialData         interface{} // Any partial data that could be salvaged
}

// NewErrorWithRecovery creates an error with recovery suggestions
func NewErrorWithRecovery(errType AIErrorType, message string, cause error, suggestions []string, partialData interface{}) *ErrorWithRecovery {
	return &ErrorWithRecovery{
		AIError:             NewAIError(errType, message, cause),
		RecoverySuggestions: suggestions,
		PartialData:         partialData,
	}
}

// GetRecoverySuggestions returns recovery suggestions for an error type
func GetRecoverySuggestions(errType AIErrorType) []string {
	switch errType {
	case ErrTypeNoTools:
		return []string{
			"Install Claude CLI: brew install claude (macOS)",
			"Install Gemini CLI: npm install -g @google/generative-ai-cli",
			"Use 'qualhook config' for manual configuration",
		}
	case ErrTypeToolNotFound:
		return []string{
			"Check available tools with 'qualhook ai-config' without --tool flag",
			"Install the missing tool",
			"Use a different AI tool",
		}
	case ErrTypeExecutionFailed:
		return []string{
			"Check your internet connection",
			"Verify AI tool credentials",
			"Try a different AI tool",
			"Use manual configuration as fallback",
		}
	case ErrTypeResponseInvalid:
		return []string{
			"Retry the command",
			"Try a different AI tool",
			"Use manual configuration",
			"Enable debug mode for more details",
		}
	case ErrTypeTimeout:
		return []string{
			"Retry with a longer timeout",
			"Use manual configuration for specific commands",
			"Create a basic config manually",
		}
	case ErrTypeValidationFailed:
		return []string{
			"Modify the command during review",
			"Check if required tools are installed",
			"Configure the command manually",
		}
	default:
		return []string{
			"Try the operation again",
			"Use 'qualhook config' for manual configuration",
			"Check the documentation for more help",
		}
	}
}

// FormatErrorWithSuggestions formats an error with recovery suggestions
func FormatErrorWithSuggestions(err error) string {
	var builder strings.Builder

	// Write the main error
	builder.WriteString("Error: ")
	builder.WriteString(err.Error())
	builder.WriteString("\n")

	// Check if it's an ErrorWithRecovery
	if errWithRecovery, ok := err.(*ErrorWithRecovery); ok {
		if len(errWithRecovery.RecoverySuggestions) > 0 {
			builder.WriteString("\nSuggested actions:\n")
			for i, suggestion := range errWithRecovery.RecoverySuggestions {
				builder.WriteString(fmt.Sprintf("  %d. %s\n", i+1, suggestion))
			}
		}

		// If there's partial data, mention it
		if errWithRecovery.PartialData != nil {
			builder.WriteString("\nNote: Partial data was recovered and can be used.\n")
		}
	} else if aiErr, ok := err.(*AIError); ok {
		// Get default suggestions for the error type
		suggestions := GetRecoverySuggestions(aiErr.Type)
		if len(suggestions) > 0 {
			builder.WriteString("\nSuggested actions:\n")
			for i, suggestion := range suggestions {
				builder.WriteString(fmt.Sprintf("  %d. %s\n", i+1, suggestion))
			}
		}
	}

	return builder.String()
}

// WrapErrorWithContext wraps an error with additional context
func WrapErrorWithContext(err error, context string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", context, err)
}

// IsRetryableError determines if an error is retryable
func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Unwrap to get the root cause
	var aiErr *AIError
	if errors.As(err, &aiErr) {
		switch aiErr.Type {
		case ErrTypeExecutionFailed, ErrTypeTimeout, ErrTypeResponseInvalid:
			return true
		case ErrTypeUserCanceled, ErrTypeNoTools, ErrTypeToolNotFound:
			return false
		case ErrTypeValidationFailed:
			// Validation failures might be retryable if they're due to transient issues
			return true
		}
	}

	// Check for common retryable error patterns
	errMsg := strings.ToLower(err.Error())
	retryablePatterns := []string{
		"timeout",
		"network",
		"connection",
		"temporary",
		"unavailable",
		"rate limit",
	}

	for _, pattern := range retryablePatterns {
		if strings.Contains(errMsg, pattern) {
			return true
		}
	}

	return false
}

// HandleNetworkError provides specific handling for network-related errors
func HandleNetworkError(err error) error {
	if err == nil {
		return nil
	}

	errMsg := strings.ToLower(err.Error())
	networkPatterns := []string{
		"no such host",
		"connection refused",
		"connection reset",
		"network is unreachable",
		"timeout",
		"dns",
	}

	for _, pattern := range networkPatterns {
		if strings.Contains(errMsg, pattern) {
			return NewErrorWithRecovery(
				ErrTypeExecutionFailed,
				"Network connectivity issue detected",
				err,
				[]string{
					"Check your internet connection",
					"Check if you're behind a firewall or proxy",
					"Try again in a few moments",
					"Use 'qualhook config' for offline configuration",
				},
				nil,
			)
		}
	}

	return err
}

// ExtractPartialConfig attempts to extract partial configuration from an error scenario
func ExtractPartialConfig(response string, err error) (partialCommands map[string]bool, recoveryHint string) {
	partialCommands = make(map[string]bool)

	// Check what command types are mentioned in the response
	commandTypes := []string{"format", "lint", "typecheck", "test"}
	for _, cmdType := range commandTypes {
		if strings.Contains(strings.ToLower(response), cmdType) {
			// Check if there's actual command data for this type
			if strings.Contains(response, `"`+cmdType+`"`) || strings.Contains(response, cmdType+":") {
				partialCommands[cmdType] = true
			}
		}
	}

	if len(partialCommands) > 0 {
		configured := []string{}
		missing := []string{}

		for _, cmdType := range commandTypes {
			if partialCommands[cmdType] {
				configured = append(configured, cmdType)
			} else {
				missing = append(missing, cmdType)
			}
		}

		recoveryHint = fmt.Sprintf(msgPartialSuccess,
			strings.Join(configured, ", "),
			strings.Join(missing, ", "))
	}

	return partialCommands, recoveryHint
}

// SanitizeErrorMessage removes potentially sensitive information from error messages
func SanitizeErrorMessage(err error) error {
	if err == nil {
		return nil
	}

	msg := err.Error()
	sanitized := msg

	// Check for API keys or tokens
	if containsAPIKeyPattern(msg) {
		sanitized = "[REDACTED]"
	}

	// Sanitize file paths with usernames
	sanitized = sanitizePaths(sanitized)

	if sanitized != msg {
		return fmt.Errorf("%s (sensitive information removed)", sanitized)
	}

	return err
}

// containsAPIKeyPattern checks if the message contains API key patterns
func containsAPIKeyPattern(msg string) bool {
	lowerMsg := strings.ToLower(msg)
	if !strings.Contains(lowerMsg, "key") && !strings.Contains(lowerMsg, "token") {
		return false
	}

	patterns := []string{"sk-", "api_key", "api-key", "token_", "token-"}
	for _, pattern := range patterns {
		if strings.Contains(msg, pattern) {
			return true
		}
	}
	return false
}

// sanitizePaths removes usernames from common path patterns
func sanitizePaths(msg string) string {
	type pathPattern struct {
		prefix    string
		prefixLen int
		separator string
	}

	patterns := []pathPattern{
		{"/Users/", 7, "/"},
		{"/home/", 6, "/"},
		{"C:\\Users\\", 10, "\\"},
	}

	result := msg
	for _, pattern := range patterns {
		result = sanitizePath(result, pattern.prefix, pattern.prefixLen, pattern.separator)
	}

	return result
}

// sanitizePath removes username from a specific path pattern
func sanitizePath(msg, prefix string, prefixLen int, separator string) string {
	if !strings.Contains(msg, prefix) {
		return msg
	}

	start := strings.Index(msg, prefix)
	if start < 0 {
		return msg
	}

	end := strings.Index(msg[start+prefixLen:], separator)
	if end <= 0 {
		return msg
	}

	username := msg[start+prefixLen : start+prefixLen+end]
	placeholder := strings.ReplaceAll(prefix, separator, "") + "[USER]"
	return strings.ReplaceAll(msg, prefix+username, placeholder)
}
