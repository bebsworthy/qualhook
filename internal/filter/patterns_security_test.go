package filter

import (
	"strings"
	"testing"
	"time"

	"github.com/bebsworthy/qualhook/internal/security"
	"github.com/bebsworthy/qualhook/pkg/config"
)

// TestPatternCache_ReDoSPrevention tests that ReDoS vulnerable patterns are rejected
func TestPatternCache_ReDoSPrevention(t *testing.T) {
	pc, err := NewPatternCache()
	if err != nil {
		t.Fatalf("Failed to create pattern cache: %v", err)
	}
	
	secValidator := security.NewSecurityValidator()

	tests := []struct {
		name    string
		pattern string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "nested quantifiers (.*)*",
			pattern: "(.*)*",
			wantErr: true,
			errMsg:  "catastrophic backtracking",
		},
		{
			name:    "nested quantifiers (.+)+",
			pattern: "(.+)+",
			wantErr: true,
			errMsg:  "catastrophic backtracking",
		},
		{
			name:    "alternation with quantifier (a|a)*",
			pattern: "(a|a)*",
			wantErr: true,
			errMsg:  "catastrophic backtracking",
		},
		{
			name:    "complex nested groups ((a+)+)+",
			pattern: "((a+)+)+",
			wantErr: true,
			errMsg:  "catastrophic backtracking",
		},
		{
			name:    "safe pattern",
			pattern: "^error: (.+)$",
			wantErr: false,
		},
		{
			name:    "safe alternation",
			pattern: "(error|warning): (.+)",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// First validate with security validator
			err := secValidator.ValidateRegexPattern(tt.pattern)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateRegexPattern() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("ValidateRegexPattern() error = %v, want error containing %q", err, tt.errMsg)
			}

			// If validation passes, try to compile using cache
			if err == nil {
				rp := &config.RegexPattern{Pattern: tt.pattern}
				compiledRe, compileErr := pc.GetOrCompile(rp)
				if compileErr != nil {
					t.Errorf("Failed to compile safe pattern: %v", compileErr)
				} else if compiledRe == nil {
					t.Error("GetOrCompile returned nil for safe pattern")
				}
			}
		})
	}
}

// TestPatternCache_PatternTimeout tests that pattern compilation has reasonable timeouts
func TestPatternCache_PatternTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping timeout test in short mode")
	}

	pc, err := NewPatternCache()
	if err != nil {
		t.Fatalf("Failed to create pattern cache: %v", err)
	}

	// This pattern is complex but not necessarily ReDoS vulnerable
	// It's just meant to test timeout handling
	complexPattern := strings.Repeat("(a?){1,}", 50) + strings.Repeat("a", 50)

	done := make(chan bool, 1)
	var compileErr error

	go func() {
		rp := &config.RegexPattern{Pattern: complexPattern}
		_, compileErr = pc.GetOrCompile(rp)
		done <- true
	}()

	select {
	case <-done:
		// Compilation completed (either success or error)
		if compileErr == nil {
			t.Log("Complex pattern compiled successfully")
		} else {
			t.Logf("Complex pattern failed to compile: %v", compileErr)
		}
	case <-time.After(200 * time.Millisecond):
		// If we're still waiting after 200ms, that's concerning
		t.Error("Pattern compilation took too long (possible ReDoS)")
	}
}

// TestPatternCache_MaliciousPatterns tests various malicious regex patterns
func TestPatternCache_MaliciousPatterns(t *testing.T) {
	secValidator := security.NewSecurityValidator()

	maliciousPatterns := []struct {
		name        string
		pattern     string
		description string
	}{
		{
			name:        "exponential backtracking",
			pattern:     "(a*)*b",
			description: "causes exponential backtracking on non-matching input",
		},
		{
			name:        "nested groups with alternation",
			pattern:     "((a|b)*)*c",
			description: "nested quantifiers with alternation",
		},
		{
			name:        "overlapping alternation",
			pattern:     "(a|ab)*c",
			description: "overlapping alternatives with quantifier",
		},
		{
			name:        "complex lookahead",
			pattern:     "(?=a*)*b",
			description: "quantified lookahead assertion",
		},
		{
			name:        "deeply nested groups",
			pattern:     "((((a*)*)*)*)*",
			description: "deeply nested quantifiers",
		},
		{
			name:        "alternation of similar patterns",
			pattern:     "(x+x+)+y",
			description: "redundant repetition",
		},
	}

	for _, mp := range maliciousPatterns {
		t.Run(mp.name, func(t *testing.T) {
			err := secValidator.ValidateRegexPattern(mp.pattern)
			if err == nil {
				t.Errorf("Pattern %q (%s) should have been rejected", mp.pattern, mp.description)
			} else {
				t.Logf("Successfully rejected %s: %v", mp.description, err)
			}
		})
	}
}

// TestPatternCache_SafePatterns tests that legitimate patterns are not blocked
func TestPatternCache_SafePatterns(t *testing.T) {
	pc, err := NewPatternCache()
	if err != nil {
		t.Fatalf("Failed to create pattern cache: %v", err)
	}
	
	secValidator := security.NewSecurityValidator()

	safePatterns := []struct {
		name    string
		pattern string
		test    string
		match   bool
	}{
		{
			name:    "simple error pattern",
			pattern: `error:\s*(.+)`,
			test:    "error: something went wrong",
			match:   true,
		},
		{
			name:    "line number pattern",
			pattern: `^\s*at\s+(.+?)\s*\((.+?):(\d+):(\d+)\)`,
			test:    "  at function (file.js:10:5)",
			match:   true,
		},
		{
			name:    "log level pattern",
			pattern: `^(ERROR|WARN|INFO|DEBUG):\s*(.+)`,
			test:    "ERROR: Database connection failed",
			match:   true,
		},
		{
			name:    "file path pattern",
			pattern: `^([a-zA-Z]:)?[/\\]?(?:[^/\\]+[/\\])*[^/\\]+\.[a-zA-Z]+$`,
			test:    "/home/user/project/file.go",
			match:   true,
		},
		{
			name:    "test result pattern",
			pattern: `^(PASS|FAIL|SKIP):\s*(.+?)\s*\(([0-9.]+)s\)`,
			test:    "PASS: TestFunction (0.05s)",
			match:   true,
		},
	}

	for _, sp := range safePatterns {
		t.Run(sp.name, func(t *testing.T) {
			// Validate pattern
			err := secValidator.ValidateRegexPattern(sp.pattern)
			if err != nil {
				t.Fatalf("Safe pattern rejected: %v", err)
			}

			// Compile pattern
			rp := &config.RegexPattern{Pattern: sp.pattern}
			re, err := pc.GetOrCompile(rp)
			if err != nil {
				t.Fatalf("Failed to compile safe pattern: %v", err)
			}

			// Test pattern matching
			matched := re.MatchString(sp.test)
			if matched != sp.match {
				t.Errorf("Pattern match = %v, want %v for test string %q", matched, sp.match, sp.test)
			}
		})
	}
}

// TestPatternCache_ConcurrentAccess tests thread safety of pattern compilation and caching
func TestPatternCache_ConcurrentAccess(t *testing.T) {
	pc, err := NewPatternCache()
	if err != nil {
		t.Fatalf("Failed to create pattern cache: %v", err)
	}
	
	patterns := []*config.RegexPattern{
		{Pattern: `error:\s*(.+)`},
		{Pattern: `warning:\s*(.+)`},
		{Pattern: `info:\s*(.+)`},
	}

	// Number of concurrent goroutines
	concurrency := 20
	iterations := 100
	done := make(chan bool, concurrency)

	for i := 0; i < concurrency; i++ {
		go func(id int) {
			defer func() { done <- true }()

			for j := 0; j < iterations; j++ {
				// Compile patterns
				for _, p := range patterns {
					re, err := pc.GetOrCompile(p)
					if err != nil {
						t.Errorf("Goroutine %d: Failed to compile pattern: %v", id, err)
						return
					}

					// Test pattern matching
					testStr := "error: test message"
					if !re.MatchString(testStr) && p.Pattern == `error:\s*(.+)` {
						t.Errorf("Goroutine %d: Pattern should have matched", id)
					}
				}
			}
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < concurrency; i++ {
		<-done
	}

	// Verify cache stats
	stats := pc.GetStats()
	if stats.Hits == 0 {
		t.Error("Expected cache hits but got none")
	}
	t.Logf("Cache stats - Hits: %d, Misses: %d", stats.Hits, stats.Misses)
}

// TestPatternValidation_SecurityPatterns tests validation of security-relevant patterns
func TestPatternValidation_SecurityPatterns(t *testing.T) {
	secValidator := security.NewSecurityValidator()

	// Patterns that might be used in security contexts
	securityPatterns := []struct {
		name    string
		pattern string
		purpose string
		valid   bool
	}{
		{
			name:    "SQL injection detection",
			pattern: `(?i)(union|select|insert|update|delete|drop)\s+`,
			purpose: "detect SQL keywords",
			valid:   true,
		},
		{
			name:    "XSS detection",
			pattern: `<script[^>]*>.*?</script>`,
			purpose: "detect script tags",
			valid:   true,
		},
		{
			name:    "Path traversal detection",
			pattern: `\.\.[\\/]`,
			purpose: "detect directory traversal",
			valid:   true,
		},
		{
			name:    "Command injection detection",
			pattern: `[;&|]\s*(?:rm|del|format|dd)`,
			purpose: "detect dangerous commands",
			valid:   true,
		},
		{
			name:    "Overly broad pattern",
			pattern: `.*`,
			purpose: "matches everything",
			valid:   true, // Valid regex but might be flagged as too generic
		},
	}

	for _, sp := range securityPatterns {
		t.Run(sp.name, func(t *testing.T) {
			err := secValidator.ValidateRegexPattern(sp.pattern)
			if sp.valid && err != nil {
				t.Errorf("Valid security pattern rejected: %v", err)
			}
			if !sp.valid && err == nil {
				t.Errorf("Invalid security pattern accepted")
			}
			if err == nil {
				t.Logf("Pattern for %s validated successfully", sp.purpose)
			}
		})
	}
}