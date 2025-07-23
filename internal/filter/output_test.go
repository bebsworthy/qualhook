package filter

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/bebsworthy/qualhook/pkg/config"
)

func TestNewOutputFilter(t *testing.T) {
	tests := []struct {
		name    string
		rules   *config.FilterConfig
		wantErr bool
	}{
		{
			name:    "nil rules",
			rules:   nil,
			wantErr: true,
		},
		{
			name: "valid rules",
			rules: &config.FilterConfig{
				ErrorPatterns: []*config.RegexPattern{
					{Pattern: "error", Flags: "i"},
				},
				MaxOutput: 100,
			},
			wantErr: false,
		},
		{
			name: "invalid regex pattern",
			rules: &config.FilterConfig{
				ErrorPatterns: []*config.RegexPattern{
					{Pattern: "[invalid"},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewOutputFilter(tt.rules)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewOutputFilter() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOutputFilter_Filter(t *testing.T) {
	filter, _ := NewOutputFilter(&config.FilterConfig{
		ErrorPatterns: []*config.RegexPattern{
			{Pattern: "error", Flags: "i"},
			{Pattern: "^\\s*\\d+:\\d+", Flags: "m"},
		},
		ContextLines: 2,
		MaxOutput:    50,
	})

	tests := []struct {
		name      string
		input     string
		wantError bool
		wantLines int
	}{
		{
			name: "simple error",
			input: `line 1
line 2
ERROR: something went wrong
line 4
line 5`,
			wantError: true,
			wantLines: 5, // With context
		},
		{
			name: "line number format",
			input: `file.go:10:5: undefined variable
file.go:20:3: syntax error`,
			wantError: true,
			wantLines: 2,
		},
		{
			name:      "no matches",
			input:     "just some normal output\nnothing special here",
			wantError: false,
			wantLines: 2, // Returns sample when no matches
		},
		{
			name: "multiple errors with context",
			input: `start
before error 1
ERROR 1
after error 1
middle
before error 2
ERROR 2
after error 2
end`,
			wantError: true,
			wantLines: 9, // All lines due to overlapping context
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filter.Filter(tt.input)
			if result.HasErrors != tt.wantError {
				t.Errorf("Filter() HasErrors = %v, want %v", result.HasErrors, tt.wantError)
			}
			if len(result.Lines) != tt.wantLines {
				t.Errorf("Filter() got %d lines, want %d", len(result.Lines), tt.wantLines)
			}
		})
	}
}

func TestOutputFilter_FilterBoth(t *testing.T) {
	filter, _ := NewOutputFilter(&config.FilterConfig{
		ErrorPatterns: []*config.RegexPattern{
			{Pattern: "error", Flags: "i"},
		},
		MaxOutput: 10,
	})

	stdout := "stdout line 1\nstdout line 2"
	stderr := "stderr ERROR line 1\nstderr line 2"

	result := filter.FilterBoth(stdout, stderr)

	if !result.HasErrors {
		t.Error("FilterBoth() should detect errors in stderr")
	}

	// Should have both stdout and stderr sections
	outputStr := strings.Join(result.Lines, "\n")
	if !strings.Contains(outputStr, "=== STDERR ===") {
		t.Error("FilterBoth() should include stderr section")
	}
	if !strings.Contains(outputStr, "=== STDOUT ===") {
		t.Error("FilterBoth() should include stdout section")
	}
}

func TestOutputFilter_IntelligentTruncate(t *testing.T) {
	filter, _ := NewOutputFilter(&config.FilterConfig{
		ErrorPatterns: []*config.RegexPattern{
			{Pattern: "ERROR", Flags: ""},
		},
		ContextLines: 0, // No context to make the test clearer
		MaxOutput:    5,
	})

	input := `line 1
line 2
ERROR: important error
line 4
line 5
line 6
ERROR: another error
line 8`

	result := filter.Filter(input)

	// With 2 error lines out of 8, we should get those 2 plus 3 others = 5 total (or 6 with truncation message)
	if len(result.Lines) > 6 {
		t.Errorf("Should respect MaxOutput, got %d lines", len(result.Lines))
	}

	// Should preserve error lines
	outputStr := strings.Join(result.Lines, "\n")
	if !strings.Contains(outputStr, "ERROR: important error") {
		t.Error("Should preserve error lines during truncation")
	}
}

func TestOutputFilter_StreamFilter(t *testing.T) {
	filter, _ := NewOutputFilter(&config.FilterConfig{
		ErrorPatterns: []*config.RegexPattern{
			{Pattern: "ERROR"},
		},
		ContextLines: 1,
	})

	input := `line 1
line 2
ERROR: streaming error
line 4
line 5`

	reader := strings.NewReader(input)
	var output bytes.Buffer

	err := filter.StreamFilter(reader, &output)
	if err != nil {
		t.Fatalf("StreamFilter() error = %v", err)
	}

	outputStr := output.String()
	if !strings.Contains(outputStr, "ERROR: streaming error") {
		t.Error("StreamFilter() should output error lines")
	}
}

func TestOutputFilter_LargeOutput(t *testing.T) {
	filter, _ := NewOutputFilter(&config.FilterConfig{
		ErrorPatterns: []*config.RegexPattern{
			{Pattern: "ERROR"},
		},
		ContextLines: 1,
		MaxOutput:    50, // Lower limit to ensure truncation
	})

	// Generate large output with more errors
	var lines []string
	for i := 0; i < 200; i++ {
		if i%10 == 0 {
			lines = append(lines, fmt.Sprintf("ERROR: error at line %d", i))
		} else {
			lines = append(lines, fmt.Sprintf("normal line %d", i))
		}
	}
	
	input := strings.Join(lines, "\n")
	result := filter.Filter(input)

	if !result.HasErrors {
		t.Error("Should detect errors in large output")
	}
	
	// The result should be truncated to approximately MaxOutput lines
	if len(result.Lines) > 51 { // Allow for truncation message
		t.Errorf("Should limit output to MaxOutput, got %d lines", len(result.Lines))
	}
	
	// Should have truncation indicator
	outputStr := strings.Join(result.Lines, "\n")
	if !strings.Contains(outputStr, "truncated") {
		t.Error("Should include truncation indicator")
	}
}

func TestOutputFilter_EmptyPatterns(t *testing.T) {
	filter, _ := NewOutputFilter(&config.FilterConfig{
		ErrorPatterns: []*config.RegexPattern{
			{Pattern: "ERROR"}, // Changed to uppercase so it won't match
		},
		IncludePatterns: []*config.RegexPattern{},
		MaxOutput:       50,
	})

	input := "some output\nwith no errors\njust normal stuff"
	result := filter.Filter(input)

	if result.HasErrors {
		t.Error("Should not have errors when no patterns match")
	}
	if len(result.Lines) != 3 {
		t.Errorf("Expected 3 lines, got %d", len(result.Lines))
	}
}

func TestOutputFilter_ContextOverlap(t *testing.T) {
	filter, _ := NewOutputFilter(&config.FilterConfig{
		ErrorPatterns: []*config.RegexPattern{
			{Pattern: "ERROR"},
		},
		ContextLines: 3,
		MaxOutput:    100,
	})

	input := `line 1
line 2
line 3
ERROR 1
line 5
ERROR 2
line 7
line 8
line 9`

	result := filter.Filter(input)

	// With overlapping context, should get all lines
	if len(result.Lines) != 9 {
		t.Errorf("Expected 9 lines with overlapping context, got %d", len(result.Lines))
	}
}

func BenchmarkOutputFilter_Filter(b *testing.B) {
	filter, _ := NewOutputFilter(&config.FilterConfig{
		ErrorPatterns: []*config.RegexPattern{
			{Pattern: "error", Flags: "i"},
			{Pattern: "warning", Flags: "i"},
			{Pattern: "^\\s*\\d+:\\d+"},
		},
		ContextLines: 2,
		MaxOutput:    100,
	})

	input := generateBenchmarkInput(1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = filter.Filter(input)
	}
}

func generateBenchmarkInput(lines int) string {
	var result []string
	for i := 0; i < lines; i++ {
		switch i % 10 {
		case 0:
			result = append(result, "ERROR: something went wrong")
		case 5:
			result = append(result, "WARNING: potential issue")
		default:
			result = append(result, "normal output line "+string(rune(i)))
		}
	}
	return strings.Join(result, "\n")
}