package ai

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestNewErrorWithRecovery(t *testing.T) {
	tests := []struct {
		name        string
		errType     AIErrorType
		message     string
		cause       error
		suggestions []string
		partialData interface{}
	}{
		{
			name:    "no tools error with recovery",
			errType: ErrTypeNoTools,
			message: "No AI tools available",
			cause:   nil,
			suggestions: []string{
				"Install Claude CLI",
				"Install Gemini CLI",
				"Use manual configuration",
			},
			partialData: nil,
		},
		{
			name:    "execution failed with partial data",
			errType: ErrTypeExecutionFailed,
			message: "AI tool failed",
			cause:   errors.New("connection timeout"),
			suggestions: []string{
				"Check network",
				"Retry",
			},
			partialData: map[string]string{"format": "prettier"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewErrorWithRecovery(tt.errType, tt.message, tt.cause, tt.suggestions, tt.partialData)

			if err.Type != tt.errType {
				t.Errorf("Expected error type %v, got %v", tt.errType, err.Type)
			}

			if err.Message != tt.message {
				t.Errorf("Expected message %q, got %q", tt.message, err.Message)
			}

			if len(err.RecoverySuggestions) != len(tt.suggestions) {
				t.Errorf("Expected %d suggestions, got %d", len(tt.suggestions), len(err.RecoverySuggestions))
			}

			if tt.partialData != nil && err.PartialData == nil {
				t.Error("Expected partial data, got nil")
			}
		})
	}
}

func TestGetRecoverySuggestions(t *testing.T) {
	tests := []struct {
		name     string
		errType  AIErrorType
		minCount int // Minimum expected suggestions
	}{
		{
			name:     "no tools suggestions",
			errType:  ErrTypeNoTools,
			minCount: 3,
		},
		{
			name:     "tool not found suggestions",
			errType:  ErrTypeToolNotFound,
			minCount: 3,
		},
		{
			name:     "execution failed suggestions",
			errType:  ErrTypeExecutionFailed,
			minCount: 4,
		},
		{
			name:     "response invalid suggestions",
			errType:  ErrTypeResponseInvalid,
			minCount: 4,
		},
		{
			name:     "timeout suggestions",
			errType:  ErrTypeTimeout,
			minCount: 3,
		},
		{
			name:     "validation failed suggestions",
			errType:  ErrTypeValidationFailed,
			minCount: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suggestions := GetRecoverySuggestions(tt.errType)
			if len(suggestions) < tt.minCount {
				t.Errorf("Expected at least %d suggestions for %v, got %d", tt.minCount, tt.errType, len(suggestions))
			}

			// Verify suggestions are not empty
			for i, suggestion := range suggestions {
				if suggestion == "" {
					t.Errorf("Empty suggestion at index %d", i)
				}
			}
		})
	}
}

func TestFormatErrorWithSuggestions(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		contains []string
	}{
		{
			name: "AI error with default suggestions",
			err:  NewAIError(ErrTypeNoTools, "No tools found", nil),
			contains: []string{
				"Error: No tools found",
				"Suggested actions:",
				"Install Claude CLI",
			},
		},
		{
			name: "Error with recovery and partial data",
			err: NewErrorWithRecovery(
				ErrTypeResponseInvalid,
				"Parse failed",
				nil,
				[]string{"Retry", "Use manual config"},
				map[string]string{"format": "prettier"},
			),
			contains: []string{
				"Error: Parse failed",
				"Suggested actions:",
				"1. Retry",
				"2. Use manual config",
				"Partial data was recovered",
			},
		},
		{
			name: "Regular error",
			err:  errors.New("generic error"),
			contains: []string{
				"Error: generic error",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatted := FormatErrorWithSuggestions(tt.err)
			for _, expected := range tt.contains {
				if !strings.Contains(formatted, expected) {
					t.Errorf("Expected formatted error to contain %q, got:\n%s", expected, formatted)
				}
			}
		})
	}
}

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		retryable bool
	}{
		{
			name:      "execution failed is retryable",
			err:       NewAIError(ErrTypeExecutionFailed, "failed", nil),
			retryable: true,
		},
		{
			name:      "timeout is retryable",
			err:       NewAIError(ErrTypeTimeout, "timeout", nil),
			retryable: true,
		},
		{
			name:      "response invalid is retryable",
			err:       NewAIError(ErrTypeResponseInvalid, "invalid", nil),
			retryable: true,
		},
		{
			name:      "user canceled is not retryable",
			err:       NewAIError(ErrTypeUserCanceled, "canceled", nil),
			retryable: false,
		},
		{
			name:      "no tools is not retryable",
			err:       NewAIError(ErrTypeNoTools, "no tools", nil),
			retryable: false,
		},
		{
			name:      "network error pattern is retryable",
			err:       errors.New("connection timeout"),
			retryable: true,
		},
		{
			name:      "rate limit error is retryable",
			err:       errors.New("rate limit exceeded"),
			retryable: true,
		},
		{
			name:      "nil error is not retryable",
			err:       nil,
			retryable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			retryable := IsRetryableError(tt.err)
			if retryable != tt.retryable {
				t.Errorf("Expected IsRetryableError(%v) = %v, got %v", tt.err, tt.retryable, retryable)
			}
		})
	}
}

func TestHandleNetworkError(t *testing.T) {
	tests := []struct {
		name             string
		err              error
		expectNetworkErr bool
	}{
		{
			name:             "no such host",
			err:              errors.New("dial tcp: lookup example.com: no such host"),
			expectNetworkErr: true,
		},
		{
			name:             "connection refused",
			err:              errors.New("dial tcp 127.0.0.1:8080: connection refused"),
			expectNetworkErr: true,
		},
		{
			name:             "timeout error",
			err:              errors.New("request timeout after 30s"),
			expectNetworkErr: true,
		},
		{
			name:             "non-network error",
			err:              errors.New("invalid JSON"),
			expectNetworkErr: false,
		},
		{
			name:             "nil error",
			err:              nil,
			expectNetworkErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handled := HandleNetworkError(tt.err)

			if tt.err == nil && handled != nil {
				t.Error("Expected nil for nil input")
				return
			}

			if tt.expectNetworkErr {
				var errWithRecovery *ErrorWithRecovery
				if !errors.As(handled, &errWithRecovery) {
					t.Error("Expected ErrorWithRecovery for network error")
				} else if errWithRecovery.Type != ErrTypeExecutionFailed {
					t.Errorf("Expected ErrTypeExecutionFailed, got %v", errWithRecovery.Type)
				}
			} else if handled != tt.err {
				t.Error("Expected original error for non-network error")
			}
		})
	}
}

func TestExtractPartialConfig(t *testing.T) {
	tests := []struct {
		name               string
		response           string
		expectedCommands   map[string]bool
		expectRecoveryHint bool
	}{
		{
			name: "response with multiple commands",
			response: `{
				"format": {"command": "prettier"},
				"lint": {"command": "eslint"},
				"test": {"command": "jest"}
			}`,
			expectedCommands: map[string]bool{
				"format": true,
				"lint":   true,
				"test":   true,
			},
			expectRecoveryHint: true,
		},
		{
			name: "response with command mentions",
			response: `AI suggestion:
			format: prettier --write
			lint: eslint .
			`,
			expectedCommands: map[string]bool{
				"format": true,
				"lint":   true,
			},
			expectRecoveryHint: true,
		},
		{
			name:               "empty response",
			response:           "",
			expectedCommands:   map[string]bool{},
			expectRecoveryHint: false,
		},
		{
			name:               "response without commands",
			response:           "This project appears to be a Go project",
			expectedCommands:   map[string]bool{},
			expectRecoveryHint: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			partialCommands, recoveryHint := ExtractPartialConfig(tt.response, nil)

			if len(partialCommands) != len(tt.expectedCommands) {
				t.Errorf("Expected %d commands, got %d", len(tt.expectedCommands), len(partialCommands))
			}

			for cmd, expected := range tt.expectedCommands {
				if partialCommands[cmd] != expected {
					t.Errorf("Expected command %s = %v, got %v", cmd, expected, partialCommands[cmd])
				}
			}

			if tt.expectRecoveryHint && recoveryHint == "" {
				t.Error("Expected recovery hint, got empty string")
			} else if !tt.expectRecoveryHint && recoveryHint != "" {
				t.Errorf("Expected no recovery hint, got %q", recoveryHint)
			}
		})
	}
}

func TestSanitizeErrorMessage(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		shouldSanitize bool
	}{
		{
			name:           "error with API key",
			err:            errors.New("Authentication failed with key sk-1234567890abcdef"),
			shouldSanitize: true,
		},
		{
			name:           "error with user path",
			err:            errors.New("File not found: /Users/johndoe/project/config.json"),
			shouldSanitize: true,
		},
		{
			name:           "error with token",
			err:            errors.New("Invalid token: token_abc123xyz"),
			shouldSanitize: true,
		},
		{
			name:           "normal error",
			err:            errors.New("Command not found"),
			shouldSanitize: false,
		},
		{
			name:           "nil error",
			err:            nil,
			shouldSanitize: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sanitized := SanitizeErrorMessage(tt.err)

			if tt.err == nil && sanitized != nil {
				t.Error("Expected nil for nil input")
				return
			}

			if tt.err == nil {
				return
			}

			sanitizedMsg := sanitized.Error()
			originalMsg := tt.err.Error()

			if tt.shouldSanitize {
				// Should not contain sensitive patterns
				sensitivePatterns := []string{"sk-", "johndoe", "token_abc", "api_key"}
				for _, pattern := range sensitivePatterns {
					if strings.Contains(sanitizedMsg, pattern) {
						t.Errorf("Sanitized message still contains sensitive pattern %q: %s", pattern, sanitizedMsg)
					}
				}

				// Should indicate sanitization occurred
				if !strings.Contains(sanitizedMsg, "[REDACTED]") && !strings.Contains(sanitizedMsg, "[USER_PATH]") && !strings.Contains(sanitizedMsg, "sensitive information removed") {
					t.Error("Sanitized message should indicate sanitization occurred")
				}
			} else if sanitizedMsg != originalMsg {
				t.Errorf("Expected message to remain unchanged, got %q", sanitizedMsg)
			}
		})
	}
}

func TestWrapErrorWithContext(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		context string
		want    string
	}{
		{
			name:    "wrap error with context",
			err:     errors.New("original error"),
			context: "during AI execution",
			want:    "during AI execution: original error",
		},
		{
			name:    "nil error returns nil",
			err:     nil,
			context: "some context",
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrapped := WrapErrorWithContext(tt.err, tt.context)

			if tt.err == nil && wrapped != nil {
				t.Error("Expected nil for nil input")
				return
			}

			if tt.err != nil && wrapped.Error() != tt.want {
				t.Errorf("Expected %q, got %q", tt.want, wrapped.Error())
			}
		})
	}
}

// Test error type string representation
func TestAIErrorTypeString(t *testing.T) {
	// Ensure all error types have proper handling
	errorTypes := []AIErrorType{
		ErrTypeNoTools,
		ErrTypeToolNotFound,
		ErrTypeExecutionFailed,
		ErrTypeResponseInvalid,
		ErrTypeUserCanceled,
		ErrTypeTimeout,
		ErrTypeValidationFailed,
	}

	for _, errType := range errorTypes {
		t.Run(fmt.Sprintf("error_type_%d", errType), func(t *testing.T) {
			suggestions := GetRecoverySuggestions(errType)
			if len(suggestions) == 0 {
				t.Errorf("No recovery suggestions for error type %v", errType)
			}
		})
	}
}
