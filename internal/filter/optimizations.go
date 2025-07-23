// Package filter provides pattern compilation and caching functionality.
package filter

import (
	"regexp"
	"sync"
	
	"github.com/qualhook/qualhook/pkg/config"
)

// OptimizedPatternSet provides optimized pattern matching for common cases
type OptimizedPatternSet struct {
	// Literal strings that can be matched without regex
	literals map[string]bool
	// Simple prefix patterns (e.g., "ERROR:")
	prefixes []string
	// Simple suffix patterns (e.g., ".go")
	suffixes []string
	// Compiled regex patterns for complex cases
	patterns []*regexp.Regexp
	// Original pattern configurations
	configs []*config.RegexPattern
	mu      sync.RWMutex
}

// NewOptimizedPatternSet creates an optimized pattern set
func NewOptimizedPatternSet(patterns []*config.RegexPattern) (*OptimizedPatternSet, error) {
	ops := &OptimizedPatternSet{
		literals: make(map[string]bool),
		prefixes: make([]string, 0),
		suffixes: make([]string, 0),
		patterns: make([]*regexp.Regexp, 0),
		configs:  patterns,
	}
	
	for _, pattern := range patterns {
		if err := ops.addPattern(pattern); err != nil {
			return nil, err
		}
	}
	
	return ops, nil
}

// addPattern analyzes a pattern and adds it to the appropriate optimization bucket
func (ops *OptimizedPatternSet) addPattern(pattern *config.RegexPattern) error {
	// Check if it's a simple literal
	if isLiteral(pattern.Pattern) {
		ops.literals[pattern.Pattern] = true
		return nil
	}
	
	// Check if it's a simple prefix pattern
	if prefix, ok := isSimplePrefix(pattern.Pattern); ok {
		ops.prefixes = append(ops.prefixes, prefix)
		return nil
	}
	
	// Check if it's a simple suffix pattern
	if suffix, ok := isSimpleSuffix(pattern.Pattern); ok {
		ops.suffixes = append(ops.suffixes, suffix)
		return nil
	}
	
	// Fall back to regex compilation
	compiled, err := pattern.Compile()
	if err != nil {
		return err
	}
	ops.patterns = append(ops.patterns, compiled)
	
	return nil
}

// MatchAnyOptimized uses optimized matching strategies
func (ops *OptimizedPatternSet) MatchAnyOptimized(input string) bool {
	ops.mu.RLock()
	defer ops.mu.RUnlock()
	
	// First check literals (O(1) lookup)
	if ops.literals[input] {
		return true
	}
	
	// Check prefixes (optimized string operations)
	for _, prefix := range ops.prefixes {
		if len(input) >= len(prefix) && input[:len(prefix)] == prefix {
			return true
		}
	}
	
	// Check suffixes (optimized string operations)
	for _, suffix := range ops.suffixes {
		if len(input) >= len(suffix) && input[len(input)-len(suffix):] == suffix {
			return true
		}
	}
	
	// Fall back to regex matching for complex patterns
	for _, re := range ops.patterns {
		if re.MatchString(input) {
			return true
		}
	}
	
	return false
}

// Helper functions to analyze patterns

func isLiteral(pattern string) bool {
	// Check if the pattern contains any regex metacharacters
	metacharacters := `\.+*?[]{},^$|():`
	for _, char := range metacharacters {
		if containsRune(pattern, char) {
			return false
		}
	}
	return true
}

func isSimplePrefix(pattern string) (string, bool) {
	// Check if pattern is like "^PREFIX" or "PREFIX.*"
	if len(pattern) > 1 && pattern[0] == '^' {
		rest := pattern[1:]
		if isLiteral(rest) {
			return rest, true
		}
	}
	
	if len(pattern) > 2 && pattern[len(pattern)-2:] == ".*" {
		prefix := pattern[:len(pattern)-2]
		if isLiteral(prefix) {
			return prefix, true
		}
	}
	
	return "", false
}

func isSimpleSuffix(pattern string) (string, bool) {
	// Check if pattern is like "SUFFIX$" or ".*SUFFIX"
	if len(pattern) > 1 && pattern[len(pattern)-1] == '$' {
		rest := pattern[:len(pattern)-1]
		if isLiteral(rest) {
			return rest, true
		}
	}
	
	if len(pattern) > 2 && pattern[:2] == ".*" {
		suffix := pattern[2:]
		if isLiteral(suffix) {
			return suffix, true
		}
	}
	
	return "", false
}

func containsRune(s string, r rune) bool {
	for _, char := range s {
		if char == r {
			return true
		}
	}
	return false
}

// BatchMatcher provides optimized batch matching operations
type BatchMatcher struct {
	cache       *PatternCache
	bufferPool  sync.Pool
	maxLineSize int
}

// NewBatchMatcher creates a new batch matcher
func NewBatchMatcher(cache *PatternCache) *BatchMatcher {
	return &BatchMatcher{
		cache:       cache,
		maxLineSize: 4096, // Default max line size
		bufferPool: sync.Pool{
			New: func() interface{} {
				return make([]byte, 0, 4096)
			},
		},
	}
}

// MatchLines efficiently matches multiple lines against patterns
func (bm *BatchMatcher) MatchLines(lines []string, patterns []*config.RegexPattern) []int {
	// Pre-compile all patterns
	compiled := make([]*regexp.Regexp, len(patterns))
	for i, pattern := range patterns {
		re, err := bm.cache.GetOrCompile(pattern)
		if err != nil {
			continue
		}
		compiled[i] = re
	}
	
	// Match lines
	matches := make([]int, 0, len(lines)/10) // Preallocate for ~10% match rate
	
	for lineNum, line := range lines {
		for _, re := range compiled {
			if re != nil && re.MatchString(line) {
				matches = append(matches, lineNum)
				break // Move to next line after first match
			}
		}
	}
	
	return matches
}

// GetBuffer gets a buffer from the pool
func (bm *BatchMatcher) GetBuffer() []byte {
	return bm.bufferPool.Get().([]byte)
}

// PutBuffer returns a buffer to the pool
func (bm *BatchMatcher) PutBuffer(buf []byte) {
	buf = buf[:0] // Reset slice
	//nolint:staticcheck // SA6002: slices are already reference types, reusing backing array
	bm.bufferPool.Put(buf)
}