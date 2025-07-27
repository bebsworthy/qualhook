//go:build unit

// Package filter provides output filtering and processing functionality for qualhook.
package filter

import (
	"fmt"
	"strings"
	"testing"

	"github.com/bebsworthy/qualhook/pkg/config"
)

// BenchmarkOutputFiltering measures filtering performance with various output sizes
func BenchmarkOutputFiltering(b *testing.B) {
	filterRules := TestFilterRules.Complex

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
			output := GenerateTestOutput(tc.lines, tc.errorRate)
			filter, err := NewOutputFilter(filterRules)
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
	filterRules := TestFilterRules.Basic

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
			output := GenerateTestOutput(tc.lines, 0.1)
			filter, _ := NewOutputFilter(filterRules)

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
				rules: &FilterRules{
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
				rules: &FilterRules{
					MaxLines: tc.maxOutput,
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
	filterRules := TestFilterRules.Basic

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
			stdout := GenerateTestOutput(tc.stdoutLines, 0.05)
			stderr := GenerateTestOutput(tc.stderrLines, 0.3)

			filter, _ := NewOutputFilter(filterRules)

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
	filterRules := TestFilterRules.Strict

	b.Run("SmallOutput", func(b *testing.B) {
		output := GenerateTestOutput(100, 0.1)
		filter, _ := NewOutputFilter(filterRules)

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = filter.Filter(output)
		}
	})

	b.Run("LargeOutput", func(b *testing.B) {
		output := GenerateTestOutput(10000, 0.01)
		filter, _ := NewOutputFilter(filterRules)

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = filter.Filter(output)
		}
	})

	b.Run("Streaming", func(b *testing.B) {
		output := GenerateTestOutput(10000, 0.01)
		filter, _ := NewOutputFilter(filterRules)

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

		filterRules := &FilterRules{
			ErrorPatterns: []*config.RegexPattern{
				{Pattern: `ERROR:`, Flags: ""},
			},
			ContextLines: 5,
			MaxLines:     100,
		}

		filter, _ := NewOutputFilter(filterRules)

		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = filter.Filter(output)
		}
	})

	b.Run("ComplexPatterns", func(b *testing.B) {
		output := GenerateTestOutput(1000, 0.1)

		// Use complex patterns that may be slow
		filterRules := TestFilterRules.Complex

		filter, _ := NewOutputFilter(filterRules)

		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = filter.Filter(output)
		}
	})
}
