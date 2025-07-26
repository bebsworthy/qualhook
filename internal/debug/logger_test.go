//go:build unit

package debug

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestLogger(t *testing.T) {
	// Save original state
	originalEnabled := globalLogger.enabled
	originalWriter := globalLogger.writer
	defer func() {
		globalLogger.enabled = originalEnabled
		globalLogger.writer = originalWriter
	}()

	// Test buffer
	var buf bytes.Buffer
	SetWriter(&buf)

	// Test disabled logging
	Log("This should not appear")
	if buf.Len() > 0 {
		t.Error("Log wrote to buffer when disabled")
	}

	// Enable logging
	Enable()
	if !IsEnabled() {
		t.Error("IsEnabled() returned false after Enable()")
	}

	// Test basic logging
	buf.Reset()
	Log("Test message")
	output := buf.String()
	if !strings.Contains(output, "[DEBUG") {
		t.Error("Log output missing debug prefix")
	}
	if !strings.Contains(output, "Test message") {
		t.Error("Log output missing message")
	}
	if !strings.HasSuffix(output, "\n") {
		t.Error("Log output missing newline")
	}

	// Test formatting
	buf.Reset()
	Log("Formatted %s %d", "string", 42)
	output = buf.String()
	if !strings.Contains(output, "Formatted string 42") {
		t.Errorf("Log formatting failed: %q", output)
	}

	// Test message already ending with newline
	buf.Reset()
	Log("Message with newline\n")
	output = buf.String()
	if strings.Count(output, "\n") != 1 {
		t.Error("Log added extra newline")
	}
}

func TestLogSection(t *testing.T) {
	var buf bytes.Buffer
	SetWriter(&buf)
	Enable()

	buf.Reset()
	LogSection("Test Section")
	output := buf.String()
	if !strings.Contains(output, "=== Test Section ===") {
		t.Errorf("LogSection output incorrect: %q", output)
	}

	// Test when disabled
	globalLogger.enabled = false
	buf.Reset()
	LogSection("Should not appear")
	if buf.Len() > 0 {
		t.Error("LogSection wrote to buffer when disabled")
	}
}

func TestLogCommand(t *testing.T) {
	var buf bytes.Buffer
	SetWriter(&buf)
	Enable()

	buf.Reset()
	LogCommand("npm", []string{"run", "test"}, "/path/to/project")
	output := buf.String()

	expectedStrings := []string{
		"=== Command Execution ===",
		"Command: npm",
		"Arguments: [run test]",
		"Working Directory: /path/to/project",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("LogCommand missing %q in output: %q", expected, output)
		}
	}

	// Test without arguments and working directory
	buf.Reset()
	LogCommand("ls", []string{}, "")
	output = buf.String()
	if strings.Contains(output, "Arguments:") {
		t.Error("LogCommand should not log empty arguments")
	}
	if strings.Contains(output, "Working Directory:") {
		t.Error("LogCommand should not log empty working directory")
	}
}

func TestLogTiming(t *testing.T) {
	var buf bytes.Buffer
	SetWriter(&buf)
	Enable()

	tests := []struct {
		duration time.Duration
		expected string
	}{
		{500 * time.Microsecond, "500µs"},
		{10 * time.Millisecond, "10ms"},
		{1500 * time.Millisecond, "1.50s"},
	}

	for _, tt := range tests {
		buf.Reset()
		LogTiming("operation", tt.duration)
		output := buf.String()
		if !strings.Contains(output, "Timing: operation took "+tt.expected) {
			t.Errorf("LogTiming(%v) output incorrect: %q", tt.duration, output)
		}
	}
}

func TestLogPatternMatch(t *testing.T) {
	var buf bytes.Buffer
	SetWriter(&buf)
	Enable()

	// Test matched
	buf.Reset()
	LogPatternMatch("error.*", "error: something went wrong", true)
	output := buf.String()
	if !strings.Contains(output, "matched") {
		t.Error("LogPatternMatch should indicate match")
	}

	// Test not matched
	buf.Reset()
	LogPatternMatch("error.*", "warning: be careful", false)
	output = buf.String()
	if !strings.Contains(output, "no match") {
		t.Error("LogPatternMatch should indicate no match")
	}

	// Test truncation
	buf.Reset()
	longInput := strings.Repeat("a", 100)
	LogPatternMatch("test", longInput, false)
	output = buf.String()
	if strings.Contains(output, strings.Repeat("a", 100)) {
		t.Error("LogPatternMatch should truncate long input")
	}
	if !strings.Contains(output, "...") {
		t.Error("LogPatternMatch should add ellipsis for truncated input")
	}
}

func TestLogFilterProcess(t *testing.T) {
	var buf bytes.Buffer
	SetWriter(&buf)
	Enable()

	buf.Reset()
	LogFilterProcess(100, 25, 10)
	output := buf.String()
	if !strings.Contains(output, "Filter: 100 total lines -> 25 matched -> 10 output") {
		t.Errorf("LogFilterProcess output incorrect: %q", output)
	}
}

func TestLogError(t *testing.T) {
	var buf bytes.Buffer
	SetWriter(&buf)
	Enable()

	buf.Reset()
	testErr := errors.New("test error")
	LogError(testErr, "test operation")
	output := buf.String()
	if !strings.Contains(output, "Error in test operation: test error") {
		t.Errorf("LogError output incorrect: %q", output)
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		duration time.Duration
		expected string
	}{
		{100 * time.Microsecond, "100µs"},
		{999 * time.Microsecond, "999µs"},
		{1 * time.Millisecond, "1ms"},
		{999 * time.Millisecond, "999ms"},
		{1 * time.Second, "1.00s"},
		{2500 * time.Millisecond, "2.50s"},
	}

	for _, tt := range tests {
		result := formatDuration(tt.duration)
		if result != tt.expected {
			t.Errorf("formatDuration(%v) = %q, want %q", tt.duration, result, tt.expected)
		}
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"short", 10, "short"},
		{"exact length", 12, "exact length"},
		{"this is too long", 10, "this is..."},
		{"", 5, ""},
	}

	for _, tt := range tests {
		result := truncate(tt.input, tt.maxLen)
		if result != tt.expected {
			t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, result, tt.expected)
		}
	}
}

func TestTimingIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping timing test in short mode")
	}
	// Cannot run in parallel due to global state modification
	var buf bytes.Buffer
	SetWriter(&buf)
	Enable()

	// Simulate actual timing
	start := time.Now()
	time.Sleep(10 * time.Millisecond)
	duration := time.Since(start)

	buf.Reset()
	LogTiming("sleep operation", duration)
	output := buf.String()

	// Should contain timing info (exact value varies)
	if !strings.Contains(output, "Timing: sleep operation took") {
		t.Error("LogTiming integration test failed")
	}
	if !strings.Contains(output, "ms") && !strings.Contains(output, "s") {
		t.Error("LogTiming should include time unit")
	}
}

func TestDebugPrefix(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	SetWriter(&buf)
	Enable()

	// Wait a tiny bit to ensure elapsed time > 0
	time.Sleep(1 * time.Millisecond)

	buf.Reset()
	Log("Test")
	output := buf.String()

	// Should have the format: [DEBUG XXXms] Test
	if !strings.HasPrefix(output, "[DEBUG ") {
		t.Error("Log output should start with [DEBUG")
	}
	if !strings.Contains(output, "] Test") {
		t.Error("Log output should contain the message after the prefix")
	}
}
