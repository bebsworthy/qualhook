// Package filter provides pattern compilation and caching functionality.
package filter

import (
	"fmt"
	"regexp"
	"sync"

	"github.com/qualhook/qualhook/pkg/config"
)

// PatternCache manages compiled regex patterns with thread-safe caching
type PatternCache struct {
	cache map[string]*regexp.Regexp
	mu    sync.RWMutex
	stats *CacheStats
}

// CacheStats tracks pattern cache performance metrics
type CacheStats struct {
	Hits        int64
	Misses      int64
	CompileTime int64 // Total time spent compiling patterns (nanoseconds)
	mu          sync.Mutex
}

// PatternValidator provides pattern validation and testing functionality
type PatternValidator struct {
	cache *PatternCache
}

// NewPatternCache creates a new pattern cache
func NewPatternCache() (*PatternCache, error) {
	return &PatternCache{
		cache: make(map[string]*regexp.Regexp),
		stats: &CacheStats{},
	}, nil
}

// GetOrCompile retrieves a compiled pattern from cache or compiles it
func (pc *PatternCache) GetOrCompile(pattern *config.RegexPattern) (*regexp.Regexp, error) {
	if pattern == nil {
		return nil, fmt.Errorf("pattern cannot be nil")
	}

	// Create cache key from pattern and flags
	key := pc.getCacheKey(pattern)

	// Try to get from cache first (read lock)
	pc.mu.RLock()
	if compiled, exists := pc.cache[key]; exists {
		pc.mu.RUnlock()
		pc.recordHit()
		return compiled, nil
	}
	pc.mu.RUnlock()

	// Not in cache, need to compile (write lock)
	pc.mu.Lock()
	defer pc.mu.Unlock()

	// Double-check in case another goroutine compiled it
	if compiled, exists := pc.cache[key]; exists {
		pc.recordHit()
		return compiled, nil
	}

	// Compile the pattern
	pc.recordMiss()
	compiled, err := pattern.Compile()
	if err != nil {
		return nil, fmt.Errorf("failed to compile pattern %q: %w", pattern.Pattern, err)
	}

	// Cache the compiled pattern
	pc.cache[key] = compiled
	return compiled, nil
}

// Precompile compiles and caches multiple patterns
func (pc *PatternCache) Precompile(patterns []*config.RegexPattern) error {
	var errors []error

	for _, pattern := range patterns {
		if _, err := pc.GetOrCompile(pattern); err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to precompile %d patterns", len(errors))
	}

	return nil
}

// Clear removes all cached patterns
func (pc *PatternCache) Clear() {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	pc.cache = make(map[string]*regexp.Regexp)
}

// Size returns the number of cached patterns
func (pc *PatternCache) Size() int {
	pc.mu.RLock()
	defer pc.mu.RUnlock()

	return len(pc.cache)
}

// GetStats returns cache performance statistics
func (pc *PatternCache) GetStats() CacheStats {
	pc.stats.mu.Lock()
	defer pc.stats.mu.Unlock()

	return CacheStats{
		Hits:        pc.stats.Hits,
		Misses:      pc.stats.Misses,
		CompileTime: pc.stats.CompileTime,
	}
}

// ResetStats resets cache performance statistics
func (pc *PatternCache) ResetStats() {
	pc.stats.mu.Lock()
	defer pc.stats.mu.Unlock()

	pc.stats.Hits = 0
	pc.stats.Misses = 0
	pc.stats.CompileTime = 0
}

// NewPatternValidator creates a new pattern validator
func NewPatternValidator(cache *PatternCache) *PatternValidator {
	if cache == nil {
		cache, _ = NewPatternCache()
	}
	return &PatternValidator{cache: cache}
}

// Validate checks if a pattern is valid and can be compiled
func (pv *PatternValidator) Validate(pattern *config.RegexPattern) error {
	if pattern == nil {
		return fmt.Errorf("pattern cannot be nil")
	}

	if pattern.Pattern == "" {
		return fmt.Errorf("pattern cannot be empty")
	}

	// Try to compile the pattern
	_, err := pv.cache.GetOrCompile(pattern)
	if err != nil {
		return fmt.Errorf("invalid pattern: %w", err)
	}

	return nil
}

// ValidateAll validates multiple patterns
func (pv *PatternValidator) ValidateAll(patterns []*config.RegexPattern) []error {
	var errors []error

	for i, pattern := range patterns {
		if err := pv.Validate(pattern); err != nil {
			errors = append(errors, fmt.Errorf("pattern %d: %w", i, err))
		}
	}

	return errors
}

// TestPattern tests a pattern against sample input
func (pv *PatternValidator) TestPattern(pattern *config.RegexPattern, input string) (*PatternTestResult, error) {
	re, err := pv.cache.GetOrCompile(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to compile pattern: %w", err)
	}

	matches := re.FindAllStringIndex(input, -1)
	matchStrings := re.FindAllString(input, -1)

	return &PatternTestResult{
		Pattern:       pattern.Pattern,
		Input:         input,
		Matches:       matchStrings,
		MatchCount:    len(matches),
		MatchIndices:  matches,
		IsValid:       true,
		CompiledRegex: re.String(),
	}, nil
}

// TestPatternBatch tests a pattern against multiple inputs
func (pv *PatternValidator) TestPatternBatch(pattern *config.RegexPattern, inputs []string) ([]*PatternTestResult, error) {
	results := make([]*PatternTestResult, len(inputs))

	for i, input := range inputs {
		result, err := pv.TestPattern(pattern, input)
		if err != nil {
			return nil, fmt.Errorf("failed to test pattern on input %d: %w", i, err)
		}
		results[i] = result
	}

	return results, nil
}

// OptimizePattern attempts to optimize a pattern for better performance
func (pv *PatternValidator) OptimizePattern(pattern *config.RegexPattern) (*config.RegexPattern, []string) {
	var suggestions []string
	optimized := &config.RegexPattern{
		Pattern: pattern.Pattern,
		Flags:   pattern.Flags,
	}

	// Check for common optimization opportunities
	if hasAnchorOptimization(pattern.Pattern) {
		suggestions = append(suggestions, "Consider adding ^ or $ anchors to improve performance")
	}

	if hasGreedyQuantifiers(pattern.Pattern) {
		suggestions = append(suggestions, "Consider using non-greedy quantifiers (.*? instead of .*)")
	}

	if hasUnescapedDots(pattern.Pattern) {
		suggestions = append(suggestions, "Consider escaping literal dots (\\. instead of .)")
	}

	return optimized, suggestions
}

// PatternTestResult contains the results of testing a pattern
type PatternTestResult struct {
	Pattern       string
	Input         string
	Matches       []string
	MatchCount    int
	MatchIndices  [][]int
	IsValid       bool
	CompiledRegex string
	Error         error
}

// Private helper methods

func (pc *PatternCache) getCacheKey(pattern *config.RegexPattern) string {
	if pattern.Flags == "" {
		return pattern.Pattern
	}
	return fmt.Sprintf("(?%s)%s", pattern.Flags, pattern.Pattern)
}

func (pc *PatternCache) recordHit() {
	pc.stats.mu.Lock()
	defer pc.stats.mu.Unlock()
	pc.stats.Hits++
}

func (pc *PatternCache) recordMiss() {
	pc.stats.mu.Lock()
	defer pc.stats.mu.Unlock()
	pc.stats.Misses++
}

// Pattern optimization helpers

func hasAnchorOptimization(pattern string) bool {
	// Check if pattern could benefit from anchors
	return len(pattern) > 0 && pattern[0] != '^' && pattern[len(pattern)-1] != '$'
}

func hasGreedyQuantifiers(pattern string) bool {
	// Simple check for greedy quantifiers
	greedyPatterns := []string{".*", ".+", ".*?", ".+?"}
	for _, greedy := range greedyPatterns[:2] { // Only check actual greedy ones
		if regexp.MustCompile(regexp.QuoteMeta(greedy)).MatchString(pattern) {
			return true
		}
	}
	return false
}

func hasUnescapedDots(pattern string) bool {
	// Check for unescaped dots that might be meant as literals
	// This is a simplified check
	re := regexp.MustCompile(`[^\\]\.`)
	return re.MatchString(pattern)
}

// PatternSet manages a collection of patterns with efficient matching
type PatternSet struct {
	patterns []*config.RegexPattern
	compiled []*regexp.Regexp
	cache    *PatternCache
}

// NewPatternSet creates a new pattern set
func NewPatternSet(patterns []*config.RegexPattern, cache *PatternCache) (*PatternSet, error) {
	if cache == nil {
		cache, _ = NewPatternCache()
	}

	ps := &PatternSet{
		patterns: patterns,
		compiled: make([]*regexp.Regexp, len(patterns)),
		cache:    cache,
	}

	// Precompile all patterns
	for i, pattern := range patterns {
		compiled, err := cache.GetOrCompile(pattern)
		if err != nil {
			return nil, fmt.Errorf("failed to compile pattern %d: %w", i, err)
		}
		ps.compiled[i] = compiled
	}

	return ps, nil
}

// MatchAny returns true if the input matches any pattern in the set
func (ps *PatternSet) MatchAny(input string) bool {
	for _, re := range ps.compiled {
		if re.MatchString(input) {
			return true
		}
	}
	return false
}

// MatchAll returns all patterns that match the input
func (ps *PatternSet) MatchAll(input string) []int {
	var matches []int
	for i, re := range ps.compiled {
		if re.MatchString(input) {
			matches = append(matches, i)
		}
	}
	return matches
}

// FindAll returns all matches from all patterns
func (ps *PatternSet) FindAll(input string) map[int][]string {
	matches := make(map[int][]string)
	for i, re := range ps.compiled {
		if found := re.FindAllString(input, -1); len(found) > 0 {
			matches[i] = found
		}
	}
	return matches
}