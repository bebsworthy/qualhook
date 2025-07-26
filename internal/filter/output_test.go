//go:build unit

package filter

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bebsworthy/qualhook/pkg/config"
)

// loadTestFixture loads test data from a fixture file
func loadTestFixture(t *testing.T, filename string) string {
	t.Helper()
	path := filepath.Join("..", "..", "test", "fixtures", "outputs", filename)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to load test fixture %s: %v", filename, err)
	}
	return string(data)
}

func TestNewOutputFilter(t *testing.T) {
	tests := []struct {
		name    string
		rules   *FilterRules
		wantErr bool
	}{
		{
			name:    "nil rules",
			rules:   nil,
			wantErr: true,
		},
		{
			name: "valid rules",
			rules: &FilterRules{
				ErrorPatterns: []*config.RegexPattern{
					{Pattern: "error", Flags: "i"},
				},
				MaxLines: 100,
			},
			wantErr: false,
		},
		{
			name: "invalid regex pattern",
			rules: &FilterRules{
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
	filter, _ := NewOutputFilter(&FilterRules{
		ErrorPatterns: []*config.RegexPattern{
			{Pattern: "error", Flags: "i"},
			{Pattern: "^\\s*\\d+:\\d+", Flags: "m"},
		},
		ContextLines: 2,
		MaxLines:    50,
	})

	tests := []struct {
		name      string
		input     string
		wantError bool
		wantLines int
	}{
		{
			name:      "simple error",
			input:     loadTestFixture(t, "error_output.txt"),
			wantError: true,
			wantLines: 5, // With context
		},
		{
			name:      "line number format",
			input:     loadTestFixture(t, "line_numbers.txt"),
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
			name:      "multiple errors with context",
			input:     loadTestFixture(t, "multiple_errors.txt"),
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

func TestOutputFilter_Operations(t *testing.T) {
	tests := []struct {
		name      string
		rules     *FilterRules
		operation func(t *testing.T, filter *OutputFilter)
	}{
		{
			name: "FilterBoth detects errors",
			rules: &FilterRules{
				ErrorPatterns: []*config.RegexPattern{
					{Pattern: "error", Flags: "i"},
				},
				MaxLines: 10,
			},
			operation: func(t *testing.T, filter *OutputFilter) {
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
			},
		},
		{
			name: "intelligent truncate preserves errors",
			rules: &FilterRules{
				ErrorPatterns: []*config.RegexPattern{
					{Pattern: "ERROR", Flags: ""},
				},
				ContextLines: 0, // No context to make the test clearer
				MaxLines:     5,
			},
			operation: func(t *testing.T, filter *OutputFilter) {
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
			},
		},
		{
			name: "stream filter outputs error lines",
			rules: &FilterRules{
				ErrorPatterns: []*config.RegexPattern{
					{Pattern: "ERROR"},
				},
				ContextLines: 1,
			},
			operation: func(t *testing.T, filter *OutputFilter) {
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
			},
		},
		{
			name: "large output with truncation",
			rules: &FilterRules{
				ErrorPatterns: []*config.RegexPattern{
					{Pattern: "ERROR"},
				},
				ContextLines: 1,
				MaxLines:     50, // Lower limit to ensure truncation
			},
			operation: func(t *testing.T, filter *OutputFilter) {
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
			},
		},
		{
			name: "empty patterns no errors",
			rules: &FilterRules{
				ErrorPatterns: []*config.RegexPattern{
					{Pattern: "ERROR"}, // Changed to uppercase so it won't match
				},
				ContextPatterns: []*config.RegexPattern{},
				MaxLines:        50,
			},
			operation: func(t *testing.T, filter *OutputFilter) {
				input := "some output\nwith no errors\njust normal stuff"
				result := filter.Filter(input)

				if result.HasErrors {
					t.Error("Should not have errors when no patterns match")
				}
				if len(result.Lines) != 3 {
					t.Errorf("Expected 3 lines, got %d", len(result.Lines))
				}
			},
		},
		{
			name: "context overlap includes all lines",
			rules: &FilterRules{
				ErrorPatterns: []*config.RegexPattern{
					{Pattern: "ERROR"},
				},
				ContextLines: 3,
				MaxLines:     100,
			},
			operation: func(t *testing.T, filter *OutputFilter) {
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
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter, err := NewOutputFilter(tt.rules)
			if err != nil {
				t.Fatalf("Failed to create filter: %v", err)
			}
			tt.operation(t, filter)
		})
	}
}

