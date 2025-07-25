//go:build unit

// Package filter provides output filtering and processing functionality for qualhook.
package filter

import (
	"fmt"
	"strings"
	"testing"
)

// BenchmarkOptimizedFiltering compares original vs optimized filtering
func BenchmarkOptimizedFiltering(b *testing.B) {
	filterRules := TestFilterRules.Complex

	testCases := []struct {
		name      string
		lines     int
		errorRate float64
	}{
		{"SmallOutput", 100, 0.1},
		{"MediumOutput", 1000, 0.1},
		{"LargeOutput", 10000, 0.01},
		{"VeryLargeOutput", 100000, 0.001},
	}

	for _, tc := range testCases {
		output := GenerateTestOutput(tc.lines, tc.errorRate)

		b.Run(tc.name+"_Original", func(b *testing.B) {
			filter, _ := NewOutputFilter(filterRules)

			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_ = filter.Filter(output)
			}
		})

		b.Run(tc.name+"_Optimized", func(b *testing.B) {
			filter, _ := NewOptimizedOutputFilter(filterRules)

			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_ = filter.FilterOptimized(output)
			}
		})
	}
}

// BenchmarkStreamingComparison compares streaming implementations
func BenchmarkStreamingComparison(b *testing.B) {
	filterRules := TestFilterRules.Strict

	// Generate 1MB of output
	var builder strings.Builder
	for i := 0; i < 10000; i++ {
		if i%100 == 0 {
			builder.WriteString(fmt.Sprintf("Line %d: ERROR: Something went wrong\n", i))
		} else {
			builder.WriteString(fmt.Sprintf("Line %d: Normal log output with some content\n", i))
		}
	}
	output := builder.String()

	b.Run("Original_Streaming", func(b *testing.B) {
		filter, _ := NewOutputFilter(filterRules)

		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			reader := strings.NewReader(output)
			_ = filter.FilterReader(reader)
		}
	})

	b.Run("Optimized_Streaming", func(b *testing.B) {
		filter, _ := NewOptimizedOutputFilter(filterRules)

		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			reader := strings.NewReader(output)
			_ = filter.FilterReaderOptimized(reader)
		}
	})
}

// BenchmarkWorstCaseOptimized tests worst-case scenarios with optimization
func BenchmarkWorstCaseOptimized(b *testing.B) {
	// All lines match - worst case for memory
	var builder strings.Builder
	for i := 0; i < 10000; i++ {
		builder.WriteString(fmt.Sprintf("ERROR: Line %d failed with error\n", i))
	}
	output := builder.String()

	filterRules := TestFilterRules.Strict

	b.Run("Original_AllMatch", func(b *testing.B) {
		filter, _ := NewOutputFilter(filterRules)

		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = filter.Filter(output)
		}
	})

	b.Run("Optimized_AllMatch", func(b *testing.B) {
		filter, _ := NewOptimizedOutputFilter(filterRules)

		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = filter.FilterOptimized(output)
		}
	})
}

// BenchmarkMemoryScaling tests how memory scales with input size
func BenchmarkMemoryScaling(b *testing.B) {
	filterRules := TestFilterRules.Basic

	sizes := []int{100, 1000, 10000, 100000}

	for _, size := range sizes {
		output := GenerateTestOutput(size, 0.01)

		b.Run(fmt.Sprintf("Original_%dLines", size), func(b *testing.B) {
			filter, _ := NewOutputFilter(filterRules)

			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = filter.Filter(output)
			}
		})

		b.Run(fmt.Sprintf("Optimized_%dLines", size), func(b *testing.B) {
			filter, _ := NewOptimizedOutputFilter(filterRules)

			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = filter.FilterOptimized(output)
			}
		})
	}
}
