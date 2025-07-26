// Package ai provides AI-powered configuration generation for qualhook.
package ai

import (
	"context"
	"time"

	"github.com/bebsworthy/qualhook/pkg/config"
)

// AIOptions configures AI assistance behavior
type AIOptions struct {
	// Tool specifies which AI tool to use ("claude" or "gemini")
	// Empty string prompts user selection
	Tool string

	// WorkingDir is the project directory to analyze
	WorkingDir string

	// Interactive indicates whether to show progress and allow cancellation
	Interactive bool

	// TestCommands indicates whether to test commands before returning
	TestCommands bool

	// Timeout is the maximum time for AI execution (0 = no limit)
	Timeout time.Duration
}

// Tool represents an available AI CLI tool
type Tool struct {
	// Name of the tool ("claude" or "gemini")
	Name string

	// Command is the actual command to execute
	Command string

	// Version is the tool version if available
	Version string

	// Available indicates whether tool is installed and accessible
	Available bool
}

// CommandSuggestion represents an AI-suggested command configuration
type CommandSuggestion struct {
	// Command is the base command to execute
	Command string

	// Args are the command arguments
	Args []string

	// ErrorPatterns are regex patterns to match error output
	ErrorPatterns []config.RegexPattern

	// ExitCodes are non-zero exit codes that indicate failure
	ExitCodes []int

	// Explanation describes why this command was suggested
	Explanation string
}

// ProjectContext provides context for AI prompt generation
type ProjectContext struct {
	// ProjectType identifies the project language/framework
	ProjectType string

	// ExistingConfig is the current configuration if any
	ExistingConfig *config.Config

	// CustomCommands are additional command types beyond the standard ones
	CustomCommands []string
}

// TestResult contains the results of testing a command
type TestResult struct {
	// Success indicates whether the command executed successfully
	Success bool

	// Output contains the command output
	Output string

	// Error contains any execution error
	Error error

	// Modified indicates whether the user modified the command
	Modified bool

	// FinalCommand is the command configuration after any modifications
	FinalCommand *config.CommandConfig
}

// ProgressState represents the current state of AI processing
type ProgressState struct {
	// Phase describes the current operation phase
	// Values: "detecting", "analyzing", "parsing", "testing"
	Phase string

	// StartTime is when the operation started
	StartTime time.Time

	// ElapsedTime is the duration since start
	ElapsedTime time.Duration

	// Cancellable indicates if the operation can be canceled
	Cancellable bool

	// Message is the current progress message to display
	Message string
}

// ToolSelectionState tracks AI tool selection for the session
type ToolSelectionState struct {
	// AvailableTools lists all detected AI tools
	AvailableTools []Tool

	// SelectedTool is the user's chosen tool
	SelectedTool string

	// UserConsent indicates whether the user approved AI usage
	UserConsent bool

	// Timestamp is when the selection was made
	Timestamp time.Time
}

// Assistant is the main interface for AI-powered configuration generation
type Assistant interface {
	// GenerateConfig generates a complete configuration using AI
	GenerateConfig(ctx context.Context, options AIOptions) (*config.Config, error)

	// SuggestCommand suggests a command for a specific purpose
	SuggestCommand(ctx context.Context, commandType string, projectInfo ProjectContext) (*CommandSuggestion, error)
}

// ToolDetector detects available AI CLI tools
type ToolDetector interface {
	// DetectTools returns all available AI tools
	DetectTools() ([]Tool, error)

	// IsToolAvailable checks if a specific tool is available
	IsToolAvailable(toolName string) (bool, error)
}

// PromptGenerator creates prompts for AI tools
type PromptGenerator interface {
	// GenerateConfigPrompt creates a prompt for full configuration generation
	GenerateConfigPrompt(workingDir string) string

	// GenerateCommandPrompt creates a prompt for specific command suggestion
	GenerateCommandPrompt(commandType string, context ProjectContext) string
}

// ResponseParser parses AI tool responses
type ResponseParser interface {
	// ParseConfigResponse parses a full configuration response
	ParseConfigResponse(response string) (*config.Config, error)

	// ParseCommandResponse parses a single command suggestion
	ParseCommandResponse(response string) (*CommandSuggestion, error)
}

// ProgressIndicator manages progress display for AI operations
type ProgressIndicator interface {
	// Start begins showing progress with the given message
	Start(message string)

	// Update updates the progress message
	Update(message string)

	// Stop stops the progress indicator
	Stop()

	// WaitForCancellation returns a channel that receives true when user cancels
	WaitForCancellation(ctx context.Context) <-chan bool
}

// TestRunner validates commands by executing them
type TestRunner interface {
	// TestCommands tests a set of commands with user approval
	TestCommands(ctx context.Context, commands map[string]*config.CommandConfig) (map[string]TestResult, error)

	// TestCommand tests a single command
	TestCommand(ctx context.Context, name string, cmd *config.CommandConfig) (*TestResult, error)
}

// AIError represents errors specific to AI operations
type AIError struct {
	// Type categorizes the error
	Type AIErrorType

	// Message is the error message
	Message string

	// Cause is the underlying error if any
	Cause error
}

// AIErrorType categorizes AI-related errors
type AIErrorType int

const (
	// ErrTypeNoTools indicates no AI tools are available
	ErrTypeNoTools AIErrorType = iota

	// ErrTypeToolNotFound indicates the specified tool wasn't found
	ErrTypeToolNotFound

	// ErrTypeExecutionFailed indicates AI tool execution failed
	ErrTypeExecutionFailed

	// ErrTypeResponseInvalid indicates AI response parsing failed
	ErrTypeResponseInvalid

	// ErrTypeUserCanceled indicates the user canceled the operation
	ErrTypeUserCanceled

	// ErrTypeTimeout indicates the operation timed out
	ErrTypeTimeout

	// ErrTypeValidationFailed indicates command validation failed
	ErrTypeValidationFailed
)

// Error implements the error interface
func (e *AIError) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}

// Unwrap returns the underlying error
func (e *AIError) Unwrap() error {
	return e.Cause
}

// NewAIError creates a new AI error
func NewAIError(errorType AIErrorType, message string, cause error) *AIError {
	return &AIError{
		Type:    errorType,
		Message: message,
		Cause:   cause,
	}
}