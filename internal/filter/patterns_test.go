//go:build unit

package filter

import (
	"sync"
	"testing"

	"github.com/bebsworthy/qualhook/pkg/config"
)

func TestNewPatternCache(t *testing.T) {
	cache, err := NewPatternCache()
	if err != nil {
		t.Fatalf("NewPatternCache() error = %v", err)
	}
	if cache == nil {
		t.Fatal("NewPatternCache() returned nil")
	}
	if cache.Size() != 0 {
		t.Errorf("New cache should be empty, got size %d", cache.Size())
	}
}

func TestPatternCache_GetOrCompile(t *testing.T) {
	cache, _ := NewPatternCache()

	tests := []struct {
		name    string
		pattern *config.RegexPattern
		wantErr bool
	}{
		{
			name:    "nil pattern",
			pattern: nil,
			wantErr: true,
		},
		{
			name:    "valid pattern",
			pattern: &config.RegexPattern{Pattern: "test", Flags: "i"},
			wantErr: false,
		},
		{
			name:    "invalid pattern",
			pattern: &config.RegexPattern{Pattern: "[invalid"},
			wantErr: true,
		},
		{
			name:    "pattern with flags",
			pattern: &config.RegexPattern{Pattern: "TEST", Flags: "i"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := cache.GetOrCompile(tt.pattern)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetOrCompile() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPatternCache_Caching(t *testing.T) {
	cache, _ := NewPatternCache()
	cache.ResetStats()

	pattern := &config.RegexPattern{Pattern: "test", Flags: "i"}

	// First call should be a miss
	_, err := cache.GetOrCompile(pattern)
	if err != nil {
		t.Fatalf("First GetOrCompile() failed: %v", err)
	}

	stats := cache.GetStats()
	if stats.Misses != 1 || stats.Hits != 0 {
		t.Errorf("Expected 1 miss, 0 hits, got %d misses, %d hits", stats.Misses, stats.Hits)
	}

	// Second call should be a hit
	_, err = cache.GetOrCompile(pattern)
	if err != nil {
		t.Fatalf("Second GetOrCompile() failed: %v", err)
	}

	stats = cache.GetStats()
	if stats.Misses != 1 || stats.Hits != 1 {
		t.Errorf("Expected 1 miss, 1 hit, got %d misses, %d hits", stats.Misses, stats.Hits)
	}
}

func TestPatternCache_Concurrent(t *testing.T) {
	cache, _ := NewPatternCache()
	pattern := &config.RegexPattern{Pattern: "concurrent.*test", Flags: "i"}

	var wg sync.WaitGroup
	errors := make(chan error, 100)

	// Launch 100 goroutines trying to compile the same pattern
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := cache.GetOrCompile(pattern)
			if err != nil {
				errors <- err
			}
		}()
	}

	wg.Wait()
	close(errors)

	// Check for any errors
	for err := range errors {
		t.Errorf("Concurrent GetOrCompile() error: %v", err)
	}

	// Should have exactly one pattern cached
	if cache.Size() != 1 {
		t.Errorf("Expected 1 cached pattern, got %d", cache.Size())
	}
}

func TestPatternCache_Precompile(t *testing.T) {
	cache, _ := NewPatternCache()

	patterns := []*config.RegexPattern{
		{Pattern: "error", Flags: "i"},
		{Pattern: "warning", Flags: "i"},
		{Pattern: "^\\d+:\\d+"},
	}

	err := cache.Precompile(patterns)
	if err != nil {
		t.Fatalf("Precompile() error = %v", err)
	}

	if cache.Size() != 3 {
		t.Errorf("Expected 3 cached patterns, got %d", cache.Size())
	}
}

func TestPatternCache_Clear(t *testing.T) {
	cache, _ := NewPatternCache()

	// Add some patterns
	patterns := []*config.RegexPattern{
		{Pattern: "test1"},
		{Pattern: "test2"},
	}
	_ = cache.Precompile(patterns)

	if cache.Size() != 2 {
		t.Errorf("Expected 2 cached patterns, got %d", cache.Size())
	}

	cache.Clear()

	if cache.Size() != 0 {
		t.Errorf("Expected 0 cached patterns after clear, got %d", cache.Size())
	}
}

func TestPatternValidator_Validate(t *testing.T) {
	t.Parallel()
	validator := NewPatternValidator(nil)

	tests := []struct {
		name    string
		pattern *config.RegexPattern
		wantErr bool
	}{
		{
			name:    "nil pattern",
			pattern: nil,
			wantErr: true,
		},
		{
			name:    "empty pattern",
			pattern: &config.RegexPattern{Pattern: ""},
			wantErr: true,
		},
		{
			name:    "valid pattern",
			pattern: &config.RegexPattern{Pattern: "valid.*pattern"},
			wantErr: false,
		},
		{
			name:    "invalid regex",
			pattern: &config.RegexPattern{Pattern: "[unclosed"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(tt.pattern)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPatternValidator_TestPattern(t *testing.T) {
	validator := NewPatternValidator(nil)

	pattern := &config.RegexPattern{Pattern: "error:\\s*(\\w+)", Flags: "i"}
	input := "ERROR: undefined variable"

	result, err := validator.TestPattern(pattern, input)
	if err != nil {
		t.Fatalf("TestPattern() error = %v", err)
	}

	if result.MatchCount != 1 {
		t.Errorf("Expected 1 match, got %d", result.MatchCount)
	}

	if len(result.Matches) != 1 || result.Matches[0] != "ERROR: undefined" {
		t.Errorf("Unexpected matches: %v", result.Matches)
	}
}

func TestPatternValidator_TestPatternBatch(t *testing.T) {
	validator := NewPatternValidator(nil)

	pattern := &config.RegexPattern{Pattern: "\\d+", Flags: ""}
	inputs := []string{
		"line 123",
		"no numbers here",
		"multiple 456 numbers 789",
	}

	results, err := validator.TestPatternBatch(pattern, inputs)
	if err != nil {
		t.Fatalf("TestPatternBatch() error = %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("Expected 3 results, got %d", len(results))
	}

	// Check match counts
	expectedCounts := []int{1, 0, 2}
	for i, expected := range expectedCounts {
		if results[i].MatchCount != expected {
			t.Errorf("Input %d: expected %d matches, got %d", i, expected, results[i].MatchCount)
		}
	}
}

func TestPatternValidator_OptimizePattern(t *testing.T) {
	validator := NewPatternValidator(nil)

	tests := []struct {
		name            string
		pattern         *config.RegexPattern
		wantSuggestions int
	}{
		{
			name:            "pattern without anchors",
			pattern:         &config.RegexPattern{Pattern: "error"},
			wantSuggestions: 1, // Should suggest anchors
		},
		{
			name:            "pattern with greedy quantifier",
			pattern:         &config.RegexPattern{Pattern: ".*error"},
			wantSuggestions: 2, // Anchors + greedy
		},
		{
			name:            "already optimized",
			pattern:         &config.RegexPattern{Pattern: "^error$"},
			wantSuggestions: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, suggestions := validator.OptimizePattern(tt.pattern)
			if len(suggestions) != tt.wantSuggestions {
				t.Errorf("Expected %d suggestions, got %d: %v", tt.wantSuggestions, len(suggestions), suggestions)
			}
		})
	}
}

func TestPatternSet_NewPatternSet(t *testing.T) {
	patterns := []*config.RegexPattern{
		{Pattern: "error", Flags: "i"},
		{Pattern: "warning", Flags: "i"},
	}

	set, err := NewPatternSet(patterns, nil)
	if err != nil {
		t.Fatalf("NewPatternSet() error = %v", err)
	}

	if len(set.patterns) != 2 {
		t.Errorf("Expected 2 patterns in set, got %d", len(set.patterns))
	}
}

func TestPatternSet_MatchAny(t *testing.T) {
	patterns := []*config.RegexPattern{
		{Pattern: "error", Flags: "i"},
		{Pattern: "warning", Flags: "i"},
	}

	set, _ := NewPatternSet(patterns, nil)

	tests := []struct {
		input     string
		wantMatch bool
	}{
		{"This is an ERROR message", true},
		{"Just a warning here", true},
		{"Normal output", false},
		{"error and warning", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := set.MatchAny(tt.input); got != tt.wantMatch {
				t.Errorf("MatchAny(%q) = %v, want %v", tt.input, got, tt.wantMatch)
			}
		})
	}
}

func TestPatternSet_MatchAll(t *testing.T) {
	patterns := []*config.RegexPattern{
		{Pattern: "error", Flags: "i"},
		{Pattern: "line \\d+"},
		{Pattern: "warning", Flags: "i"},
	}

	set, _ := NewPatternSet(patterns, nil)

	input := "ERROR on line 42: warning condition"
	matches := set.MatchAll(input)

	// Should match all three patterns
	if len(matches) != 3 {
		t.Errorf("Expected 3 matches, got %d", len(matches))
	}

	// Check that we got the right pattern indices
	expectedIndices := map[int]bool{0: true, 1: true, 2: true}
	for _, idx := range matches {
		if !expectedIndices[idx] {
			t.Errorf("Unexpected pattern index: %d", idx)
		}
	}
}

func TestPatternSet_FindAll(t *testing.T) {
	patterns := []*config.RegexPattern{
		{Pattern: "\\d+"},
		{Pattern: "error", Flags: "i"},
	}

	set, _ := NewPatternSet(patterns, nil)

	input := "Error 1: line 42 has error 2"
	matches := set.FindAll(input)

	// Pattern 0 (numbers) should find "1", "42", "2"
	if nums, ok := matches[0]; !ok || len(nums) != 3 {
		t.Errorf("Pattern 0: expected 3 number matches, got %v", nums)
	}

	// Pattern 1 (error) should find "Error" and "error"
	if errors, ok := matches[1]; !ok || len(errors) != 2 {
		t.Errorf("Pattern 1: expected 2 error matches, got %v", errors)
	}
}

