// Package filter provides pattern compilation and caching functionality.
package filter

import (
	"regexp"
	"testing"
	
	config "github.com/bebsworthy/qualhook/pkg/config"
)

// BenchmarkOptimizedPatternSet compares optimized vs regular pattern matching
func BenchmarkOptimizedPatternSet(b *testing.B) {
	// Mix of simple and complex patterns
	patterns := []*config.RegexPattern{
		// Literals
		{Pattern: "ERROR", Flags: ""},
		{Pattern: "WARNING", Flags: ""},
		{Pattern: "FATAL", Flags: ""},
		// Prefixes
		{Pattern: "^DEBUG:", Flags: ""},
		{Pattern: "INFO:.*", Flags: ""},
		// Suffixes
		{Pattern: ".*\\.go$", Flags: ""},
		{Pattern: ".js$", Flags: ""},
		// Complex patterns
		{Pattern: `\d+:\d+:\d+`, Flags: ""},
		{Pattern: `(error|warning|fatal)`, Flags: "i"},
		{Pattern: `\S+\.(go|js|ts|py):\d+:\d+`, Flags: ""},
	}
	
	// Test inputs
	testInputs := []string{
		"ERROR",
		"WARNING at line 42",
		"DEBUG: Starting process",
		"INFO: Processing complete",
		"main.go",
		"script.js",
		"2024-01-15 10:30:45",
		"error: failed to connect",
		"app.go:15:8: undefined variable",
		"Regular log line without patterns",
	}
	
	b.Run("RegularPatternSet", func(b *testing.B) {
		cache, _ := NewPatternCache()
		ps, _ := NewPatternSet(patterns, cache)
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for _, input := range testInputs {
				_ = ps.MatchAny(input)
			}
		}
	})
	
	b.Run("OptimizedPatternSet", func(b *testing.B) {
		ops, _ := NewOptimizedPatternSet(patterns)
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for _, input := range testInputs {
				_ = ops.MatchAnyOptimized(input)
			}
		}
	})
}

// BenchmarkLiteralMatching tests performance of literal pattern matching
func BenchmarkLiteralMatching(b *testing.B) {
	literalPatterns := []*config.RegexPattern{
		{Pattern: "ERROR", Flags: ""},
		{Pattern: "WARNING", Flags: ""},
		{Pattern: "INFO", Flags: ""},
		{Pattern: "DEBUG", Flags: ""},
		{Pattern: "FATAL", Flags: ""},
	}
	
	b.Run("RegexLiterals", func(b *testing.B) {
		cache, _ := NewPatternCache()
		ps, _ := NewPatternSet(literalPatterns, cache)
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = ps.MatchAny("ERROR")
			_ = ps.MatchAny("INFO")
			_ = ps.MatchAny("NOTFOUND")
		}
	})
	
	b.Run("OptimizedLiterals", func(b *testing.B) {
		ops, _ := NewOptimizedPatternSet(literalPatterns)
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = ops.MatchAnyOptimized("ERROR")
			_ = ops.MatchAnyOptimized("INFO")
			_ = ops.MatchAnyOptimized("NOTFOUND")
		}
	})
}

// BenchmarkPrefixSuffixMatching tests prefix/suffix optimization
func BenchmarkPrefixSuffixMatching(b *testing.B) {
	patterns := []*config.RegexPattern{
		{Pattern: "^ERROR:", Flags: ""},
		{Pattern: "^WARNING:", Flags: ""},
		{Pattern: "INFO:.*", Flags: ""},
		{Pattern: ".*\\.go$", Flags: ""},
		{Pattern: ".*\\.js$", Flags: ""},
		{Pattern: ".py$", Flags: ""},
	}
	
	testInputs := []string{
		"ERROR: Connection failed",
		"WARNING: Deprecated function",
		"INFO: Server started",
		"main.go",
		"app.js",
		"script.py",
		"Random text without pattern",
	}
	
	b.Run("RegexPrefixSuffix", func(b *testing.B) {
		cache, _ := NewPatternCache()
		ps, _ := NewPatternSet(patterns, cache)
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for _, input := range testInputs {
				_ = ps.MatchAny(input)
			}
		}
	})
	
	b.Run("OptimizedPrefixSuffix", func(b *testing.B) {
		ops, _ := NewOptimizedPatternSet(patterns)
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for _, input := range testInputs {
				_ = ops.MatchAnyOptimized(input)
			}
		}
	})
}

// BenchmarkBatchMatching tests batch matching performance
func BenchmarkBatchMatching(b *testing.B) {
	patterns := []*config.RegexPattern{
		{Pattern: "error", Flags: "i"},
		{Pattern: "warning", Flags: "i"},
		{Pattern: `\d+:\d+`, Flags: ""},
		{Pattern: "failed|failure", Flags: "i"},
	}
	
	// Generate test lines
	lines := make([]string, 1000)
	for i := range lines {
		switch i % 10 {
		case 0:
			lines[i] = "ERROR: Failed operation"
		case 1:
			lines[i] = "WARNING: Deprecated usage"
		case 2:
			lines[i] = "12:34: Syntax error"
		default:
			lines[i] = "Normal log line"
		}
	}
	
	b.Run("SequentialMatching", func(b *testing.B) {
		cache, _ := NewPatternCache()
		compiled := make([]*regexp.Regexp, len(patterns))
		for i, p := range patterns {
			compiled[i], _ = cache.GetOrCompile(p)
		}
		
		b.ResetTimer()
		var matchCount int
		for i := 0; i < b.N; i++ {
			matches := make([]int, 0)
			for lineNum, line := range lines {
				for _, re := range compiled {
					if re.MatchString(line) {
						matches = append(matches, lineNum)
						break
					}
				}
			}
			matchCount = len(matches) // Prevent compiler optimization
		}
		_ = matchCount
	})
	
	b.Run("BatchMatching", func(b *testing.B) {
		cache, _ := NewPatternCache()
		bm := NewBatchMatcher(cache)
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = bm.MatchLines(lines, patterns)
		}
	})
}

// BenchmarkMemoryPooling tests buffer pooling effectiveness
func BenchmarkMemoryPooling(b *testing.B) {
	cache, _ := NewPatternCache()
	bm := NewBatchMatcher(cache)
	
	b.Run("WithoutPooling", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			buf := make([]byte, 0, 4096)
			// Simulate some work
			buf = append(buf, "test data"...)
			_ = buf
		}
	})
	
	b.Run("WithPooling", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			buf := bm.GetBuffer()
			// Simulate some work
			buf = append(buf, "test data"...)
			bm.PutBuffer(buf)
		}
	})
}