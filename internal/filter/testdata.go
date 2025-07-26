//go:build unit

package filter

import (
	"fmt"
	"strings"

	"github.com/bebsworthy/qualhook/pkg/config"
)

// Common test patterns used across multiple test files
var (
	// Basic error detection patterns
	BasicErrorPatterns = []*config.RegexPattern{
		{Pattern: `error`, Flags: "i"},
		{Pattern: `warning`, Flags: "i"},
		{Pattern: `^\\s*\\d+:\\d+`, Flags: "m"},
	}

	// Complex patterns for performance testing
	ComplexPatterns = []*config.RegexPattern{
		{Pattern: `error`, Flags: "i"},
		{Pattern: `warning`, Flags: "i"},
		{Pattern: `\d+:\d+`, Flags: ""},
		{Pattern: `failed|failure`, Flags: "i"},
		{Pattern: `^[A-Z]+\s+\d+:\d+:\d+`, Flags: ""},
		{Pattern: `\S+\.(go|js|ts|py):\d+:\d+`, Flags: ""},
	}

	// Literal patterns for optimization testing
	LiteralPatterns = []*config.RegexPattern{
		{Pattern: "ERROR", Flags: ""},
		{Pattern: "WARNING", Flags: ""},
		{Pattern: "INFO", Flags: ""},
		{Pattern: "DEBUG", Flags: ""},
		{Pattern: "FATAL", Flags: ""},
	}

	// Prefix/suffix patterns
	PrefixSuffixPatterns = []*config.RegexPattern{
		{Pattern: "^ERROR:", Flags: ""},
		{Pattern: "^WARNING:", Flags: ""},
		{Pattern: "INFO:.*", Flags: ""},
		{Pattern: ".*\\.go$", Flags: ""},
		{Pattern: ".*\\.js$", Flags: ""},
		{Pattern: ".py$", Flags: ""},
	}
)

// GenerateTestOutput creates test output with configurable error rate
func GenerateTestOutput(lines int, errorRate float64) string {
	var builder strings.Builder
	errorInterval := int(1.0 / errorRate)
	if errorRate == 0 {
		errorInterval = lines + 1 // Never add errors
	}

	for i := 0; i < lines; i++ {
		if errorRate > 0 && i%errorInterval == 0 {
			builder.WriteString(fmt.Sprintf("file.go:%d:10: error: undefined variable 'x'\n", i+1))
		} else {
			builder.WriteString(fmt.Sprintf("INFO [%05d] Processing item successfully\n", i))
		}
	}

	return builder.String()
}

// GenerateMixedOutput creates output with various log levels
func GenerateMixedOutput(lines int) string {
	var result []string
	for i := 0; i < lines; i++ {
		switch i % 10 {
		case 0:
			result = append(result, "ERROR: something went wrong")
		case 5:
			result = append(result, "WARNING: potential issue")
		default:
			result = append(result, fmt.Sprintf("normal output line %d", i))
		}
	}
	return strings.Join(result, "\n")
}

// TestFilterRules provides common filter rule configurations
var TestFilterRules = struct {
	Basic    *FilterRules
	Complex  *FilterRules
	Strict   *FilterRules
	Minimal  *FilterRules
}{
	Basic: &FilterRules{
		ErrorPatterns: BasicErrorPatterns[:2], // error, warning
		ContextLines:  2,
		MaxLines:      100,
	},
	Complex: &FilterRules{
		ErrorPatterns:   ComplexPatterns,
		ContextPatterns: []*config.RegexPattern{{Pattern: `WARN`, Flags: ""}},
		ContextLines:    3,
		MaxLines:        200,
	},
	Strict: &FilterRules{
		ErrorPatterns: []*config.RegexPattern{{Pattern: `ERROR`, Flags: ""}},
		ContextLines:  0,
		MaxLines:      50,
	},
	Minimal: &FilterRules{
		ErrorPatterns: []*config.RegexPattern{{Pattern: `error`, Flags: "i"}},
		ContextLines:  1,
		MaxLines:      10,
	},
}

// Common test inputs
var (
	SmallInput  = "2024-01-15 10:30:45 ERROR: Failed to connect to database"
	MediumInput = strings.Repeat("2024-01-15 10:30:45 INFO: Processing request\n", 100)
	LargeInput  = strings.Repeat("2024-01-15 10:30:45 DEBUG: Verbose logging output with lots of details\n", 1000)
)