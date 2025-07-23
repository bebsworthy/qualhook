// Package filter provides pattern compilation and caching functionality.
package filter

import (
	"fmt"
	"strings"
	"testing"

	config "github.com/bebsworthy/qualhook/pkg/config"
)

// Common test patterns for benchmarking
var (
	benchmarkPatterns = []*config.RegexPattern{
		{Pattern: `error`, Flags: "i"},
		{Pattern: `\d+:\d+`, Flags: ""},
		{Pattern: `warning|error|fatal`, Flags: "i"},
		{Pattern: `^[A-Z]+\s+\d+:\d+:\d+`, Flags: ""},
		{Pattern: `\S+\.(go|js|ts|py):\d+:\d+`, Flags: ""},
		{Pattern: `^.*error.*$`, Flags: "im"},
		{Pattern: `(Error|Warning|Info):\s+(.+)`, Flags: ""},
		{Pattern: `\b(TODO|FIXME|XXX)\b`, Flags: ""},
	}

	// Sample inputs of various sizes
	smallInput  = "2024-01-15 10:30:45 ERROR: Failed to connect to database"
	mediumInput = strings.Repeat("2024-01-15 10:30:45 INFO: Processing request\n", 100)
	largeInput  = strings.Repeat("2024-01-15 10:30:45 DEBUG: Verbose logging output with lots of details\n", 1000)
)

// BenchmarkPatternCompilation measures the time to compile regex patterns
func BenchmarkPatternCompilation(b *testing.B) {
	patterns := []*config.RegexPattern{
		{Pattern: `simple`, Flags: ""},
		{Pattern: `(complex|pattern|with|groups)`, Flags: "i"},
		{Pattern: `^\d{4}-\d{2}-\d{2}\s+\d{2}:\d{2}:\d{2}\s+\w+:.*$`, Flags: "m"},
		{Pattern: `(?P<file>\S+\.(go|js|ts|py)):(?P<line>\d+):(?P<col>\d+)`, Flags: ""},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, pattern := range patterns {
			_, err := pattern.Compile()
			if err != nil {
				b.Fatal(err)
			}
		}
	}
}

// BenchmarkPatternCacheHit measures cache hit performance
func BenchmarkPatternCacheHit(b *testing.B) {
	cache, _ := NewPatternCache()
	pattern := &config.RegexPattern{Pattern: `error|warning|fatal`, Flags: "i"}
	
	// Pre-populate cache
	_, _ = cache.GetOrCompile(pattern)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := cache.GetOrCompile(pattern)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkPatternCacheMiss measures cache miss performance
func BenchmarkPatternCacheMiss(b *testing.B) {
	cache, _ := NewPatternCache()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pattern := &config.RegexPattern{Pattern: fmt.Sprintf(`pattern_%d`, i), Flags: ""}
		_, err := cache.GetOrCompile(pattern)
		if err != nil {
			b.Fatal(err)
		}
		// Clear cache to force miss on next iteration
		cache.Clear()
	}
}

// BenchmarkPatternMatching measures regex matching performance
func BenchmarkPatternMatching(b *testing.B) {
	testCases := []struct {
		name    string
		pattern *config.RegexPattern
		input   string
	}{
		{"SimplePattern_SmallInput", &config.RegexPattern{Pattern: `error`, Flags: "i"}, smallInput},
		{"SimplePattern_LargeInput", &config.RegexPattern{Pattern: `error`, Flags: "i"}, largeInput},
		{"ComplexPattern_SmallInput", &config.RegexPattern{Pattern: `\S+\.(go|js|ts|py):\d+:\d+`, Flags: ""}, smallInput},
		{"ComplexPattern_LargeInput", &config.RegexPattern{Pattern: `\S+\.(go|js|ts|py):\d+:\d+`, Flags: ""}, largeInput},
		{"AnchoredPattern_SmallInput", &config.RegexPattern{Pattern: `^ERROR:`, Flags: ""}, smallInput},
		{"AnchoredPattern_LargeInput", &config.RegexPattern{Pattern: `^ERROR:`, Flags: "m"}, largeInput},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			re, err := tc.pattern.Compile()
			if err != nil {
				b.Fatal(err)
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = re.MatchString(tc.input)
			}
		})
	}
}

// BenchmarkPatternSet measures performance of matching against multiple patterns
func BenchmarkPatternSet(b *testing.B) {
	cache, _ := NewPatternCache()
	
	testCases := []struct {
		name     string
		patterns []*config.RegexPattern
		input    string
	}{
		{"SmallSet_SmallInput", benchmarkPatterns[:3], smallInput},
		{"SmallSet_LargeInput", benchmarkPatterns[:3], largeInput},
		{"LargeSet_SmallInput", benchmarkPatterns, smallInput},
		{"LargeSet_LargeInput", benchmarkPatterns, largeInput},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			ps, err := NewPatternSet(tc.patterns, cache)
			if err != nil {
				b.Fatal(err)
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = ps.MatchAny(tc.input)
			}
		})
	}
}

// BenchmarkPatternFindAll measures performance of finding all matches
func BenchmarkPatternFindAll(b *testing.B) {
	cache, _ := NewPatternCache()
	patterns := []*config.RegexPattern{
		{Pattern: `\d+:\d+`, Flags: ""},
		{Pattern: `(error|warning)`, Flags: "i"},
		{Pattern: `\S+\.(go|js|ts|py)`, Flags: ""},
	}
	
	ps, _ := NewPatternSet(patterns, cache)
	
	// Input with multiple matches
	input := `
main.go:15:8: error: undefined variable
utils.js:42:15: warning: unused parameter
helper.py:108:3: error: syntax error
test.ts:23:11: warning: deprecated function
`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ps.FindAll(input)
	}
}

// BenchmarkConcurrentPatternMatching measures concurrent access performance
func BenchmarkConcurrentPatternMatching(b *testing.B) {
	cache, _ := NewPatternCache()
	pattern := &config.RegexPattern{Pattern: `error|warning|fatal`, Flags: "i"}
	
	// Pre-compile pattern
	re, _ := cache.GetOrCompile(pattern)
	
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = re.MatchString(mediumInput)
		}
	})
}

// BenchmarkMemoryUsage measures memory allocation during pattern operations
func BenchmarkMemoryUsage(b *testing.B) {
	b.Run("PatternCompilation", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			pattern := &config.RegexPattern{Pattern: `complex\s+pattern\s+\d+`, Flags: "i"}
			_, _ = pattern.Compile()
		}
	})

	b.Run("PatternCaching", func(b *testing.B) {
		cache, _ := NewPatternCache()
		pattern := &config.RegexPattern{Pattern: `cached\s+pattern`, Flags: ""}
		
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = cache.GetOrCompile(pattern)
		}
	})

	b.Run("PatternMatching", func(b *testing.B) {
		pattern := &config.RegexPattern{Pattern: `\berror\b`, Flags: "i"}
		re, _ := pattern.Compile()
		
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = re.FindAllString(largeInput, -1)
		}
	})
}