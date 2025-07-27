package ai

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/bebsworthy/qualhook/internal/executor"
	"github.com/bebsworthy/qualhook/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAssistant_AIError(t *testing.T) {
	// Test error creation and unwrapping
	cause := errors.New("underlying error")
	err := NewAIError(ErrTypeExecutionFailed, "execution failed", cause)

	assert.Equal(t, "execution failed: underlying error", err.Error())
	assert.Equal(t, ErrTypeExecutionFailed, err.Type)
	assert.Equal(t, cause, err.Unwrap())

	// Test error without cause
	err2 := NewAIError(ErrTypeNoTools, "no tools", nil)
	assert.Equal(t, "no tools", err2.Error())
	assert.Nil(t, err2.Unwrap())
}

func TestAssistant_extractCommandFromResponse(t *testing.T) {
	tests := []struct {
		name     string
		response string
		cmdType  string
		expected *config.CommandConfig
	}{
		{
			name:     "Simple format",
			response: "format: prettier --write .",
			cmdType:  "format",
			expected: &config.CommandConfig{
				Command: "prettier",
				Args:    []string{"--write", "."},
			},
		},
		{
			name:     "With command label",
			response: "The format command: gofmt -w .",
			cmdType:  "format",
			expected: &config.CommandConfig{
				Command: "gofmt",
				Args:    []string{"-w", "."},
			},
		},
		{
			name:     "JSON style",
			response: `"lint": "eslint . --fix"`,
			cmdType:  "lint",
			expected: &config.CommandConfig{
				Command: "eslint",
				Args:    []string{".", "--fix"},
			},
		},
		{
			name:     "Arrow notation",
			response: "test -> jest --coverage",
			cmdType:  "test",
			expected: &config.CommandConfig{
				Command: "jest",
				Args:    []string{"--coverage"},
			},
		},
		{
			name:     "Not found",
			response: "Some other text without commands",
			cmdType:  "format",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractCommandFromResponse(tt.response, tt.cmdType)
			if tt.expected == nil {
				assert.Nil(t, result)
			} else {
				require.NotNil(t, result)
				assert.Equal(t, tt.expected.Command, result.Command)
				assert.Equal(t, tt.expected.Args, result.Args)
			}
		})
	}
}

func TestAssistant_buildAIToolArgs(t *testing.T) {
	tests := []struct {
		toolName string
		prompt   string
		expected []string
	}{
		{
			toolName: "claude",
			prompt:   "test prompt",
			expected: []string{"test prompt"},
		},
		{
			toolName: "gemini",
			prompt:   "test prompt",
			expected: []string{"--prompt", "test prompt"},
		},
		{
			toolName: "unknown",
			prompt:   "test prompt",
			expected: []string{"test prompt"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.toolName, func(t *testing.T) {
			result := buildAIToolArgs(tt.toolName, tt.prompt)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAssistant_NewAssistant(t *testing.T) {
	// Test that NewAssistant creates a properly initialized assistant
	exec := executor.NewCommandExecutor(30 * time.Second)
	assistant := NewAssistant(exec)

	assert.NotNil(t, assistant)

	// Verify it implements the interface
	var _ = assistant
}

func TestAssistant_NoToolsError(t *testing.T) {
	// This tests the actual implementation with no tools available
	exec := executor.NewCommandExecutor(2 * time.Second)
	assistant := NewAssistant(exec).(*assistantImpl)

	// Mock the detector to return no tools
	assistant.detector = &mockToolDetectorSimple{
		tools: []Tool{},
		err:   nil,
	}

	ctx := context.Background()
	options := AIOptions{
		WorkingDir: "/test/project",
	}

	cfg, err := assistant.GenerateConfig(ctx, options)

	assert.Nil(t, cfg)
	assert.Error(t, err)

	aiErr, ok := err.(*AIError)
	require.True(t, ok)
	assert.Equal(t, ErrTypeNoTools, aiErr.Type)
}

// Simple mock for basic testing
type mockToolDetectorSimple struct {
	tools []Tool
	err   error
}

func (m *mockToolDetectorSimple) DetectTools() ([]Tool, error) {
	return m.tools, m.err
}

func (m *mockToolDetectorSimple) IsToolAvailable(toolName string) (bool, error) {
	for _, tool := range m.tools {
		if tool.Name == toolName {
			return tool.Available, nil
		}
	}
	return false, nil
}

// Benchmark partial response extraction
func BenchmarkExtractCommandFromResponse(b *testing.B) {
	response := strings.Repeat(`
		Here are the recommended commands:
		format: prettier --write .
		lint: eslint . --fix
		test: jest --coverage
		typecheck: tsc --noEmit
	`, 10)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		extractCommandFromResponse(response, "format")
		extractCommandFromResponse(response, "lint")
		extractCommandFromResponse(response, "test")
		extractCommandFromResponse(response, "typecheck")
	}
}
