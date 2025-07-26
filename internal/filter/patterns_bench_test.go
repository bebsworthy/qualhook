//go:build unit

// Package filter provides pattern compilation and caching functionality.
package filter

import (
	"fmt"
	"testing"

	config "github.com/bebsworthy/qualhook/pkg/config"
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
		{"SimplePattern_SmallInput", &config.RegexPattern{Pattern: `error`, Flags: "i"}, SmallInput},
		{"SimplePattern_LargeInput", &config.RegexPattern{Pattern: `error`, Flags: "i"}, LargeInput},
		{"ComplexPattern_SmallInput", &config.RegexPattern{Pattern: `\S+\.(go|js|ts|py):\d+:\d+`, Flags: ""}, SmallInput},
		{"ComplexPattern_LargeInput", &config.RegexPattern{Pattern: `\S+\.(go|js|ts|py):\d+:\d+`, Flags: ""}, LargeInput},
		{"AnchoredPattern_SmallInput", &config.RegexPattern{Pattern: `^ERROR:`, Flags: ""}, SmallInput},
		{"AnchoredPattern_LargeInput", &config.RegexPattern{Pattern: `^ERROR:`, Flags: "m"}, LargeInput},
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
		{"SmallSet_SmallInput", ComplexPatterns[:3], SmallInput},
		{"SmallSet_LargeInput", ComplexPatterns[:3], LargeInput},
		{"LargeSet_SmallInput", ComplexPatterns, SmallInput},
		{"LargeSet_LargeInput", ComplexPatterns, LargeInput},
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
			_ = re.MatchString(MediumInput)
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
		pattern := &config.RegexPattern{Pattern: `error`, Flags: "i"}
		re, _ := pattern.Compile()

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = re.FindAllString(LargeInput, -1)
		}
	})
}