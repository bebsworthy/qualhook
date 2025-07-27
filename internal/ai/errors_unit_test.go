package ai

import (
	"context"
	"errors"
	"testing"
)

// TestErrorRecoveryDirectly tests error recovery mechanisms directly
func TestErrorRecoveryDirectly(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name              string
		simulateError     func() error
		expectRecovery    bool
		expectSuggestions int
	}{
		{
			name: "no tools error has recovery",
			simulateError: func() error {
				return NewErrorWithRecovery(
					ErrTypeNoTools,
					msgNoToolsAvailable,
					nil,
					GetRecoverySuggestions(ErrTypeNoTools),
					nil,
				)
			},
			expectRecovery:    true,
			expectSuggestions: 3,
		},
		{
			name: "network error has recovery",
			simulateError: func() error {
				return HandleNetworkError(errors.New("dial tcp: connection refused"))
			},
			expectRecovery:    true,
			expectSuggestions: 4,
		},
		{
			name: "timeout error has recovery",
			simulateError: func() error {
				return NewErrorWithRecovery(
					ErrTypeTimeout,
					"Operation timed out",
					context.DeadlineExceeded,
					GetRecoverySuggestions(ErrTypeTimeout),
					nil,
				)
			},
			expectRecovery:    true,
			expectSuggestions: 3,
		},
		{
			name: "partial response has recovery",
			simulateError: func() error {
				partialData := map[string]bool{
					"format": true,
					"lint":   true,
				}
				return NewErrorWithRecovery(
					ErrTypeResponseInvalid,
					"Failed to parse complete response",
					errors.New("invalid JSON"),
					GetRecoverySuggestions(ErrTypeResponseInvalid),
					partialData,
				)
			},
			expectRecovery:    true,
			expectSuggestions: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.simulateError()

			// Check if it's an ErrorWithRecovery
			var errWithRecovery *ErrorWithRecovery
			if !errors.As(err, &errWithRecovery) && tt.expectRecovery {
				t.Errorf("Expected ErrorWithRecovery but got %T", err)
				return
			}

			if tt.expectRecovery {
				// Check suggestions count
				if len(errWithRecovery.RecoverySuggestions) < tt.expectSuggestions {
					t.Errorf("Expected at least %d suggestions, got %d",
						tt.expectSuggestions, len(errWithRecovery.RecoverySuggestions))
				}

				// Check if error is properly formatted
				formatted := FormatErrorWithSuggestions(err)
				if formatted == "" {
					t.Error("Expected formatted error message")
				}

				// For partial data case, check if data is preserved
				if errWithRecovery.PartialData != nil {
					if partialMap, ok := errWithRecovery.PartialData.(map[string]bool); ok {
						if len(partialMap) == 0 {
							t.Error("Expected partial data to be preserved")
						}
					}
				}
			}
		})
	}

	_ = ctx
}

// TestErrorMessageQuality tests that error messages are helpful
func TestErrorMessageQuality(t *testing.T) {
	errorScenarios := []struct {
		name           string
		err            error
		mustContain    []string
		mustNotContain []string
	}{
		{
			name: "no tools error is helpful",
			err: NewErrorWithRecovery(
				ErrTypeNoTools,
				msgNoToolsAvailable,
				nil,
				GetRecoverySuggestions(ErrTypeNoTools),
				nil,
			),
			mustContain: []string{
				"Claude CLI",
				"Gemini CLI",
				"brew install",
				"npm install",
			},
			mustNotContain: []string{},
		},
		{
			name: "execution error is helpful",
			err: NewErrorWithRecovery(
				ErrTypeExecutionFailed,
				msgExecutionFailed,
				errors.New("connection timeout"),
				GetRecoverySuggestions(ErrTypeExecutionFailed),
				nil,
			),
			mustContain: []string{
				"Check your internet connection",
				"Try a different AI tool",
			},
			mustNotContain: []string{},
		},
		{
			name: "sanitized error removes sensitive data",
			err: SanitizeErrorMessage(
				errors.New("Failed with API key sk-proj-1234567890"),
			),
			mustContain: []string{
				"[REDACTED]",
				"sensitive information removed",
			},
			mustNotContain: []string{
				"sk-proj",
				"1234567890",
			},
		},
	}

	for _, scenario := range errorScenarios {
		t.Run(scenario.name, func(t *testing.T) {
			formatted := FormatErrorWithSuggestions(scenario.err)

			// Check must contain
			for _, expected := range scenario.mustContain {
				if !contains(formatted, expected) {
					t.Errorf("Error message should contain %q but doesn't:\n%s", expected, formatted)
				}
			}

			// Check must not contain
			for _, forbidden := range scenario.mustNotContain {
				if contains(formatted, forbidden) {
					t.Errorf("Error message should not contain %q but does:\n%s", forbidden, formatted)
				}
			}
		})
	}
}

// TestErrorTypeClassification tests that errors are properly classified
func TestErrorTypeClassification(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		expectedType AIErrorType
		isRetryable  bool
		hasRecovery  bool
	}{
		{
			name:         "network errors are execution failures",
			err:          HandleNetworkError(errors.New("connection refused")),
			expectedType: ErrTypeExecutionFailed,
			isRetryable:  true,
			hasRecovery:  true,
		},
		{
			name: "user cancellation is not retryable",
			err: NewAIError(
				ErrTypeUserCanceled,
				"User pressed ESC",
				nil,
			),
			expectedType: ErrTypeUserCanceled,
			isRetryable:  false,
			hasRecovery:  false,
		},
		{
			name: "validation errors are retryable",
			err: NewAIError(
				ErrTypeValidationFailed,
				"Command failed validation",
				nil,
			),
			expectedType: ErrTypeValidationFailed,
			isRetryable:  true,
			hasRecovery:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Check error type
			var aiErr *AIError
			var errWithRecovery *ErrorWithRecovery

			switch {
			case errors.As(tt.err, &errWithRecovery):
				if errWithRecovery.Type != tt.expectedType {
					t.Errorf("Expected error type %v, got %v", tt.expectedType, errWithRecovery.Type)
				}
				if !tt.hasRecovery {
					t.Error("Did not expect ErrorWithRecovery type")
				}
			case errors.As(tt.err, &aiErr):
				if aiErr.Type != tt.expectedType {
					t.Errorf("Expected error type %v, got %v", tt.expectedType, aiErr.Type)
				}
				if tt.hasRecovery {
					t.Error("Expected ErrorWithRecovery type")
				}
			default:
				t.Errorf("Expected AI error type but got %T", tt.err)
			}

			// Check retryability
			if IsRetryableError(tt.err) != tt.isRetryable {
				t.Errorf("Expected IsRetryableError = %v", tt.isRetryable)
			}
		})
	}
}

// TestPartialConfigExtraction tests partial configuration extraction
func TestPartialConfigExtraction(t *testing.T) {
	tests := []struct {
		name           string
		response       string
		expectCommands int
		expectRecovery bool
	}{
		{
			name: "extract from JSON fragment",
			response: `{
				"commands": {
					"format": {"command": "prettier", "args": ["--write"]},
					"lint": {"command": "eslint"},
					// incomplete...`,
			expectCommands: 2,
			expectRecovery: true,
		},
		{
			name: "extract from natural language",
			response: `Based on my analysis:
			- format: Use prettier --write for formatting
			- lint: Run eslint . for linting
			- test: Execute jest for testing
			
			I couldn't determine the typecheck command.`,
			expectCommands: 3,
			expectRecovery: true,
		},
		{
			name:           "no commands found",
			response:       "This is a Go project using standard tooling.",
			expectCommands: 0,
			expectRecovery: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			commands, hint := ExtractPartialConfig(tt.response, nil)

			if len(commands) != tt.expectCommands {
				t.Errorf("Expected %d commands, got %d", tt.expectCommands, len(commands))
			}

			if tt.expectRecovery && hint == "" {
				t.Error("Expected recovery hint but got none")
			} else if !tt.expectRecovery && hint != "" {
				t.Error("Did not expect recovery hint")
			}
		})
	}
}
