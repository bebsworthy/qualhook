//go:build unit

package ai

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/bebsworthy/qualhook/internal/executor"
)

// mockCommandExecutor is a mock implementation of CommandExecutor for testing
type mockCommandExecutor struct {
	responses map[string]*executor.ExecResult
	errors    map[string]error
}

func newMockCommandExecutor() *mockCommandExecutor {
	return &mockCommandExecutor{
		responses: make(map[string]*executor.ExecResult),
		errors:    make(map[string]error),
	}
}

func (m *mockCommandExecutor) Execute(command string, args []string, options executor.ExecOptions) (*executor.ExecResult, error) {
	key := fmt.Sprintf("%s %s", command, strings.Join(args, " "))

	if err, exists := m.errors[key]; exists {
		return nil, err
	}

	if result, exists := m.responses[key]; exists {
		return result, nil
	}

	// Default to command not found
	return nil, fmt.Errorf("command not found: %s", command)
}

// trackingExecutor wraps another executor to track calls
type trackingExecutor struct {
	wrapped   commandExecutor
	callCount *int
	mu        sync.Mutex
}

func (t *trackingExecutor) Execute(command string, args []string, options executor.ExecOptions) (*executor.ExecResult, error) {
	t.mu.Lock()
	*t.callCount++
	t.mu.Unlock()
	return t.wrapped.Execute(command, args, options)
}

func TestNewToolDetector(t *testing.T) {
	t.Parallel()
	exec := executor.NewCommandExecutor(time.Minute)
	detector := NewToolDetector(exec)

	if detector == nil {
		t.Fatal("NewToolDetector returned nil")
	}

	// Type assertion to access internal fields
	td, ok := detector.(*toolDetector)
	if !ok {
		t.Fatal("NewToolDetector did not return *toolDetector")
	}

	if td.executor == nil {
		t.Error("executor should not be nil")
	}

	if td.cacheDuration != 5*time.Minute {
		t.Errorf("expected cache duration 5m, got %v", td.cacheDuration)
	}
}

func TestDetectTools_BothAvailable(t *testing.T) {
	t.Parallel()
	mockExec := newMockCommandExecutor()

	// Set up responses for both tools
	mockExec.responses["claude --version"] = &executor.ExecResult{
		Stdout:   "claude version 1.2.3",
		ExitCode: 0,
	}
	mockExec.responses["gemini --version"] = &executor.ExecResult{
		Stdout:   "Gemini CLI v2.0.0-beta.1",
		ExitCode: 0,
	}

	detector := &toolDetector{
		executor:      mockExec,
		cacheDuration: 5 * time.Minute,
	}

	tools, err := detector.DetectTools()
	if err != nil {
		t.Fatalf("DetectTools failed: %v", err)
	}

	if len(tools) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(tools))
	}

	// Check Claude
	claude := tools[0]
	if claude.Name != "claude" {
		t.Errorf("expected tool name 'claude', got %s", claude.Name)
	}
	if !claude.Available {
		t.Error("claude should be available")
	}
	if claude.Version != "1.2.3" {
		t.Errorf("expected claude version '1.2.3', got %s", claude.Version)
	}

	// Check Gemini
	gemini := tools[1]
	if gemini.Name != "gemini" {
		t.Errorf("expected tool name 'gemini', got %s", gemini.Name)
	}
	if !gemini.Available {
		t.Error("gemini should be available")
	}
	if gemini.Version != "2.0.0-beta.1" {
		t.Errorf("expected gemini version '2.0.0-beta.1', got %s", gemini.Version)
	}
}

func TestDetectTools_NoneAvailable(t *testing.T) {
	t.Parallel()
	mockExec := newMockCommandExecutor()

	// All commands fail
	mockExec.errors["claude --version"] = errors.New("command not found")
	mockExec.errors["claude version"] = errors.New("command not found")
	mockExec.errors["claude --help"] = errors.New("command not found")
	mockExec.errors["gemini --version"] = errors.New("command not found")
	mockExec.errors["gemini version"] = errors.New("command not found")
	mockExec.errors["gemini --help"] = errors.New("command not found")

	detector := &toolDetector{
		executor:      mockExec,
		cacheDuration: 5 * time.Minute,
	}

	tools, err := detector.DetectTools()
	if err != nil {
		t.Fatalf("DetectTools failed: %v", err)
	}

	if len(tools) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(tools))
	}

	for _, tool := range tools {
		if tool.Available {
			t.Errorf("tool %s should not be available", tool.Name)
		}
		if tool.Version != "" {
			t.Errorf("unavailable tool %s should not have version", tool.Name)
		}
	}
}

func TestDetectTools_AlternateVersionCommand(t *testing.T) {
	t.Parallel()
	mockExec := newMockCommandExecutor()

	// Claude uses --version
	mockExec.responses["claude --version"] = &executor.ExecResult{
		Stdout:   "1.0.0",
		ExitCode: 0,
	}

	// Gemini uses version (without --)
	mockExec.errors["gemini --version"] = errors.New("unknown flag")
	mockExec.responses["gemini version"] = &executor.ExecResult{
		Stdout:   "v2.1.0",
		ExitCode: 0,
	}

	detector := &toolDetector{
		executor:      mockExec,
		cacheDuration: 5 * time.Minute,
	}

	tools, err := detector.DetectTools()
	if err != nil {
		t.Fatalf("DetectTools failed: %v", err)
	}

	// Check both tools are available
	claude := tools[0]
	if !claude.Available || claude.Version != "1.0.0" {
		t.Errorf("claude detection failed: available=%v, version=%s", claude.Available, claude.Version)
	}

	gemini := tools[1]
	if !gemini.Available || gemini.Version != "2.1.0" {
		t.Errorf("gemini detection failed: available=%v, version=%s", gemini.Available, gemini.Version)
	}
}

func TestDetectTools_Caching(t *testing.T) {
	t.Parallel()
	callCount := 0
	mockExec := newMockCommandExecutor()

	// Track execution calls by wrapping the mock
	wrappedMock := &trackingExecutor{
		wrapped:   mockExec,
		callCount: &callCount,
	}

	mockExec.responses["claude --version"] = &executor.ExecResult{
		Stdout:   "1.0.0",
		ExitCode: 0,
	}
	mockExec.responses["gemini --version"] = &executor.ExecResult{
		Stdout:   "2.0.0",
		ExitCode: 0,
	}

	detector := &toolDetector{
		executor:      wrappedMock,
		cacheDuration: 5 * time.Minute,
	}

	// First call should execute commands
	tools1, err := detector.DetectTools()
	if err != nil {
		t.Fatalf("First DetectTools failed: %v", err)
	}
	firstCallCount := callCount

	// Second call should use cache
	tools2, err := detector.DetectTools()
	if err != nil {
		t.Fatalf("Second DetectTools failed: %v", err)
	}

	if callCount != firstCallCount {
		t.Errorf("expected no additional calls due to cache, got %d extra calls", callCount-firstCallCount)
	}

	// Results should be the same
	if len(tools1) != len(tools2) {
		t.Errorf("cached results differ in length: %d vs %d", len(tools1), len(tools2))
	}

	// Invalidate cache
	detector.lastDetection = time.Now().Add(-10 * time.Minute)

	// Third call should execute commands again
	tools3, err := detector.DetectTools()
	if err != nil {
		t.Fatalf("Third DetectTools failed: %v", err)
	}

	if callCount == firstCallCount {
		t.Error("expected additional calls after cache expiry")
	}

	_ = tools3 // Just to use the variable
}

func TestIsToolAvailable(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		toolName       string
		availableTools []Tool
		expectedResult bool
	}{
		{
			name:     "claude available",
			toolName: "claude",
			availableTools: []Tool{
				{Name: "claude", Available: true},
				{Name: "gemini", Available: false},
			},
			expectedResult: true,
		},
		{
			name:     "gemini not available",
			toolName: "gemini",
			availableTools: []Tool{
				{Name: "claude", Available: true},
				{Name: "gemini", Available: false},
			},
			expectedResult: false,
		},
		{
			name:     "case insensitive",
			toolName: "CLAUDE",
			availableTools: []Tool{
				{Name: "claude", Available: true},
			},
			expectedResult: true,
		},
		{
			name:     "unknown tool",
			toolName: "unknown",
			availableTools: []Tool{
				{Name: "claude", Available: true},
				{Name: "gemini", Available: true},
			},
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mockExec := newMockCommandExecutor()
			detector := &toolDetector{
				executor:      mockExec,
				cacheDuration: 5 * time.Minute,
				detectedTools: tt.availableTools,
				lastDetection: time.Now(),
			}

			available, err := detector.IsToolAvailable(tt.toolName)
			if err != nil {
				t.Fatalf("IsToolAvailable failed: %v", err)
			}

			if available != tt.expectedResult {
				t.Errorf("expected %v, got %v", tt.expectedResult, available)
			}
		})
	}
}

func TestExtractVersion(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name            string
		input           string
		expectedVersion string
	}{
		{
			name:            "semantic version",
			input:           "claude version 1.2.3",
			expectedVersion: "1.2.3",
		},
		{
			name:            "version with v prefix",
			input:           "v2.0.0",
			expectedVersion: "2.0.0",
		},
		{
			name:            "version with pre-release",
			input:           "Gemini CLI v2.0.0-beta.1",
			expectedVersion: "2.0.0-beta.1",
		},
		{
			name:            "version with build metadata",
			input:           "1.0.0+20130313144700",
			expectedVersion: "1.0.0+20130313144700",
		},
		{
			name:            "simple version",
			input:           "version 3.14",
			expectedVersion: "3.14",
		},
		{
			name:            "multiline output",
			input:           "Claude CLI\nVersion: 1.5.0\nBuilt: 2024-01-01",
			expectedVersion: "1.5.0",
		},
		{
			name:            "no version found",
			input:           "This is some random text",
			expectedVersion: "",
		},
		{
			name:            "version in path",
			input:           "/usr/local/bin/claude-1.2.3",
			expectedVersion: "1.2.3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			version := extractVersion(tt.input)
			if version != tt.expectedVersion {
				t.Errorf("expected version %q, got %q", tt.expectedVersion, version)
			}
		})
	}
}

func TestGetAvailableTools(t *testing.T) {
	t.Parallel()
	tools := []Tool{
		{Name: "claude", Available: true},
		{Name: "gemini", Available: false},
		{Name: "gpt", Available: true},
		{Name: "bard", Available: false},
	}

	available := GetAvailableTools(tools)

	if len(available) != 2 {
		t.Fatalf("expected 2 available tools, got %d", len(available))
	}

	expectedNames := map[string]bool{"claude": true, "gpt": true}
	for _, tool := range available {
		if !expectedNames[tool.Name] {
			t.Errorf("unexpected available tool: %s", tool.Name)
		}
		if !tool.Available {
			t.Errorf("tool %s should be marked as available", tool.Name)
		}
	}
}

func TestFormatToolsStatus(t *testing.T) {
	t.Parallel()
	tools := []Tool{
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

	status := FormatToolsStatus(tools)

	// Check for expected content
	expectedStrings := []string{
		"AI Tool Detection Results:",
		"claude:",
		"Status: ✓ Available",
		"Version: 1.2.3",
		"Command: claude",
		"gemini:",
		"Status: ✗ Not found",
		"Install: Run 'qualhook ai-config' for installation instructions",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(status, expected) {
			t.Errorf("status missing expected string: %q", expected)
		}
	}
}

func TestDetectTools_ErrorHandling(t *testing.T) {
	t.Parallel()
	mockExec := newMockCommandExecutor()

	// Claude command exists but version check fails with non-zero exit
	mockExec.responses["claude --version"] = &executor.ExecResult{
		Stderr:   "error: not authenticated",
		ExitCode: 1,
	}
	mockExec.responses["claude version"] = &executor.ExecResult{
		Stderr:   "error: not authenticated",
		ExitCode: 1,
	}
	// But help works
	mockExec.responses["claude --help"] = &executor.ExecResult{
		Stdout:   "Claude CLI help...",
		ExitCode: 0,
	}

	// Gemini has security validation error on all commands
	mockExec.errors["gemini --version"] = errors.New("command validation failed: suspicious command")
	mockExec.errors["gemini version"] = errors.New("command validation failed: suspicious command")
	mockExec.errors["gemini --help"] = errors.New("command validation failed: suspicious command")

	detector := &toolDetector{
		executor:      mockExec,
		cacheDuration: 5 * time.Minute,
	}

	tools, err := detector.DetectTools()
	if err != nil {
		t.Fatalf("DetectTools should not fail on individual tool errors: %v", err)
	}

	// Claude should be marked as available (command exists even if version fails)
	claude := tools[0]
	if !claude.Available {
		t.Error("claude should be available even if version check fails")
	}
	if claude.Version != "" {
		t.Errorf("claude should not have version when version check fails, got %s", claude.Version)
	}

	// Gemini should not be available due to security error
	gemini := tools[1]
	if gemini.Available {
		t.Error("gemini should not be available when security validation fails")
	}
}
