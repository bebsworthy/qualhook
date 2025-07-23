// Package filter provides output filtering and processing functionality for qualhook.
package filter

import (
	"fmt"
	"strings"
	"testing"

	config "github.com/bebsworthy/qualhook/pkg/config"
)

// Generate test data of various sizes
func generateOutput(lines int, errorRate float64) string {
	var builder strings.Builder
	errorInterval := int(1.0 / errorRate)
	
	for i := 0; i < lines; i++ {
		if errorRate > 0 && i%errorInterval == 0 {
			builder.WriteString(fmt.Sprintf("file.go:%d:10: error: undefined variable 'x'\n", i+1))
		} else {
			builder.WriteString(fmt.Sprintf("INFO [%05d] Processing item successfully\n", i))
		}
	}
	
	return builder.String()
}

// BenchmarkOutputFiltering measures filtering performance with various output sizes
func BenchmarkOutputFiltering(b *testing.B) {
	filterConfig := &config.FilterConfig{
		ErrorPatterns: []*config.RegexPattern{
			{Pattern: `error:`, Flags: "i"},
			{Pattern: `\d+:\d+:`, Flags: ""},
			{Pattern: `failed|failure`, Flags: "i"},
		},
		IncludePatterns: []*config.RegexPattern{
			{Pattern: `warning:`, Flags: "i"},
			{Pattern: `WARN`, Flags: ""},
		},
		ContextLines: 2,
		MaxOutput:    100,
	}

	testCases := []struct {
		name      string
		lines     int
		errorRate float64
	}{
		{"SmallOutput_NoErrors", 100, 0},
		{"SmallOutput_FewErrors", 100, 0.1},
		{"SmallOutput_ManyErrors", 100, 0.5},
		{"MediumOutput_NoErrors", 1000, 0},
		{"MediumOutput_FewErrors", 1000, 0.1},
		{"MediumOutput_ManyErrors", 1000, 0.5},
		{"LargeOutput_NoErrors", 10000, 0},
		{"LargeOutput_FewErrors", 10000, 0.01},
		{"LargeOutput_ManyErrors", 10000, 0.1},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			output := generateOutput(tc.lines, tc.errorRate)
			filter, err := NewOutputFilter(filterConfig)
			if err != nil {
				b.Fatal(err)
			}

			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_ = filter.Filter(output)
			}
		})
	}
}

// BenchmarkStreamFiltering measures streaming filter performance
func BenchmarkStreamFiltering(b *testing.B) {
	filterConfig := &config.FilterConfig{
		ErrorPatterns: []*config.RegexPattern{
			{Pattern: `error:`, Flags: "i"},
			{Pattern: `\d+:\d+:`, Flags: ""},
		},
		ContextLines: 2,
		MaxOutput:    1000,
	}

	testCases := []struct {
		name  string
		lines int
	}{
		{"SmallStream", 100},
		{"MediumStream", 1000},
		{"LargeStream", 10000},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			output := generateOutput(tc.lines, 0.1)
			filter, _ := NewOutputFilter(filterConfig)

			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				reader := strings.NewReader(output)
				_ = filter.FilterReader(reader)
			}
		})
	}
}

// BenchmarkContextExtraction measures performance of context line extraction
func BenchmarkContextExtraction(b *testing.B) {
	testCases := []struct {
		name         string
		lines        int
		matches      int
		contextLines int
	}{
		{"FewMatches_SmallContext", 1000, 10, 2},
		{"FewMatches_LargeContext", 1000, 10, 10},
		{"ManyMatches_SmallContext", 1000, 100, 2},
		{"ManyMatches_LargeContext", 1000, 100, 10},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			// Generate lines
			allLines := make([]string, tc.lines)
			for i := range allLines {
				allLines[i] = fmt.Sprintf("Line %d: some content", i)
			}

			// Generate matches
			matches := make([]lineMatch, tc.matches)
			for i := range matches {
				matches[i] = lineMatch{
					lineNum: i * (tc.lines / tc.matches),
					line:    allLines[i*(tc.lines/tc.matches)],
					isError: i%2 == 0,
				}
			}

			filter := &OutputFilter{
				rules: &config.FilterConfig{
					ContextLines: tc.contextLines,
				},
			}

			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_ = filter.extractLinesWithContext(allLines, matches)
			}
		})
	}
}

// BenchmarkIntelligentTruncation measures truncation performance
func BenchmarkIntelligentTruncation(b *testing.B) {
	testCases := []struct {
		name      string
		lines     int
		errors    int
		maxOutput int
	}{
		{"SmallTruncation", 200, 20, 100},
		{"MediumTruncation", 1000, 100, 200},
		{"LargeTruncation", 5000, 500, 500},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			// Generate lines
			lines := make([]string, tc.lines)
			matches := make([]lineMatch, tc.errors)
			
			for i := range lines {
				lines[i] = fmt.Sprintf("Line %d content", i)
			}
			
			for i := range matches {
				matches[i] = lineMatch{
					lineNum: i * (tc.lines / tc.errors),
					line:    lines[i*(tc.lines/tc.errors)],
					isError: true,
				}
			}

			filter := &OutputFilter{
				rules: &config.FilterConfig{
					MaxOutput: tc.maxOutput,
				},
			}

			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_ = filter.intelligentTruncate(lines, matches)
			}
		})
	}
}

// BenchmarkFilterBoth measures performance of filtering both stdout and stderr
func BenchmarkFilterBoth(b *testing.B) {
	filterConfig := &config.FilterConfig{
		ErrorPatterns: []*config.RegexPattern{
			{Pattern: `error:`, Flags: "i"},
			{Pattern: `failed`, Flags: "i"},
		},
		ContextLines: 2,
		MaxOutput:    200,
	}

	testCases := []struct {
		name        string
		stdoutLines int
		stderrLines int
	}{
		{"SmallBoth", 100, 50},
		{"MediumBoth", 500, 200},
		{"LargeBoth", 1000, 500},
		{"StdoutOnly", 1000, 0},
		{"StderrOnly", 0, 1000},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			stdout := generateOutput(tc.stdoutLines, 0.05)
			stderr := generateOutput(tc.stderrLines, 0.3)
			
			filter, _ := NewOutputFilter(filterConfig)

			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_ = filter.FilterBoth(stdout, stderr)
			}
		})
	}
}

// BenchmarkMemoryUsageFiltering measures memory allocation during filtering
func BenchmarkMemoryUsageFiltering(b *testing.B) {
	filterConfig := &config.FilterConfig{
		ErrorPatterns: []*config.RegexPattern{
			{Pattern: `error:`, Flags: "i"},
		},
		ContextLines: 5,
		MaxOutput:    500,
	}

	b.Run("SmallOutput", func(b *testing.B) {
		output := generateOutput(100, 0.1)
		filter, _ := NewOutputFilter(filterConfig)
		
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = filter.Filter(output)
		}
	})

	b.Run("LargeOutput", func(b *testing.B) {
		output := generateOutput(10000, 0.01)
		filter, _ := NewOutputFilter(filterConfig)
		
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = filter.Filter(output)
		}
	})

	b.Run("Streaming", func(b *testing.B) {
		output := generateOutput(10000, 0.01)
		filter, _ := NewOutputFilter(filterConfig)
		
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			reader := strings.NewReader(output)
			_ = filter.FilterReader(reader)
		}
	})
}

// BenchmarkWorstCaseScenarios tests performance in worst-case scenarios
func BenchmarkWorstCaseScenarios(b *testing.B) {
	b.Run("AllLinesMatch", func(b *testing.B) {
		// Every line matches the error pattern
		var builder strings.Builder
		for i := 0; i < 1000; i++ {
			builder.WriteString(fmt.Sprintf("ERROR: Line %d failed\n", i))
		}
		output := builder.String()
		
		filterConfig := &config.FilterConfig{
			ErrorPatterns: []*config.RegexPattern{
				{Pattern: `ERROR:`, Flags: ""},
			},
			ContextLines: 5,
			MaxOutput:    100,
		}
		
		filter, _ := NewOutputFilter(filterConfig)
		
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = filter.Filter(output)
		}
	})

	b.Run("ComplexPatterns", func(b *testing.B) {
		output := generateOutput(1000, 0.1)
		
		// Use complex patterns that may be slow
		filterConfig := &config.FilterConfig{
			ErrorPatterns: []*config.RegexPattern{
				{Pattern: `.*error.*`, Flags: "i"},
				{Pattern: `(.*)\s+(\d+):(\d+):\s+(.*)`, Flags: ""},
				{Pattern: `(?:error|warning|fatal|panic).*(?:failed|failure|exception)`, Flags: "i"},
			},
			ContextLines: 3,
			MaxOutput:    200,
		}
		
		filter, _ := NewOutputFilter(filterConfig)
		
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = filter.Filter(output)
		}
	})
}