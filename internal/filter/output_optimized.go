// Package filter provides output filtering and processing functionality for qualhook.
package filter

import (
	"bufio"
	"io"
	"strings"
	"sync"

	"github.com/bebsworthy/qualhook/pkg/config"
)

// OptimizedOutputFilter provides memory-efficient output filtering
type OptimizedOutputFilter struct {
	rules         *config.FilterConfig
	patternCache  *PatternCache
	bufferPool    *sync.Pool
	maxBufferSize int
}

// NewOptimizedOutputFilter creates a new optimized output filter
func NewOptimizedOutputFilter(rules *config.FilterConfig) (*OptimizedOutputFilter, error) {
	cache, err := NewPatternCache()
	if err != nil {
		return nil, err
	}

	// Pre-compile all patterns
	for _, pattern := range rules.ErrorPatterns {
		if _, err := cache.GetOrCompile(pattern); err != nil {
			return nil, err
		}
	}

	for _, pattern := range rules.IncludePatterns {
		if _, err := cache.GetOrCompile(pattern); err != nil {
			return nil, err
		}
	}

	return &OptimizedOutputFilter{
		rules:         rules,
		patternCache:  cache,
		maxBufferSize: 64 * 1024, // 64KB buffer
		bufferPool: &sync.Pool{
			New: func() interface{} {
				return &strings.Builder{}
			},
		},
	}, nil
}

// FilterOptimized applies filtering with minimal memory allocation
func (f *OptimizedOutputFilter) FilterOptimized(output string) *FilteredOutput {
	reader := strings.NewReader(output)
	return f.FilterReaderOptimized(reader)
}

// FilterReaderOptimized applies filtering with true streaming
func (f *OptimizedOutputFilter) FilterReaderOptimized(reader io.Reader) *FilteredOutput {
	scanner := bufio.NewScanner(reader)
	buf := make([]byte, 0, f.maxBufferSize)
	scanner.Buffer(buf, f.maxBufferSize)

	// Use circular buffer for context lines
	contextBuffer := NewCircularBuffer(f.rules.ContextLines * 2 + 1)
	
	result := &FilteredOutput{
		Lines:     make([]string, 0, 100), // Pre-allocate reasonable size
		HasErrors: false,
		Truncated: false,
	}

	var (
		totalLines     int
		capturedLines  int
		pendingContext int
	)

	for scanner.Scan() {
		line := scanner.Text()
		totalLines++

		// Add to circular buffer
		contextBuffer.Add(line)

		// Check if line matches patterns
		isError := f.matchesAnyPattern(line, f.rules.ErrorPatterns)
		isInclude := !isError && f.matchesAnyPattern(line, f.rules.IncludePatterns)

		if isError || isInclude {
			result.HasErrors = result.HasErrors || isError

			// Add context lines before the match
			contextLines := contextBuffer.GetContext(f.rules.ContextLines)
			for _, contextLine := range contextLines {
				if capturedLines < f.rules.MaxOutput {
					result.Lines = append(result.Lines, contextLine)
					capturedLines++
				}
			}

			// Set pending context for lines after the match
			pendingContext = f.rules.ContextLines
		} else if pendingContext > 0 {
			// Capture context lines after a match
			if capturedLines < f.rules.MaxOutput {
				result.Lines = append(result.Lines, line)
				capturedLines++
			}
			pendingContext--
		}

		// Check if we've hit the output limit
		if capturedLines >= f.rules.MaxOutput {
			result.Truncated = true
			// Continue scanning to count total lines but don't store
		}
	}

	result.TotalLines = totalLines
	return result
}

// CircularBuffer implements a fixed-size circular buffer for context lines
type CircularBuffer struct {
	buffer []string
	size   int
	head   int
	count  int
}

// NewCircularBuffer creates a new circular buffer
func NewCircularBuffer(size int) *CircularBuffer {
	return &CircularBuffer{
		buffer: make([]string, size),
		size:   size,
	}
}

// Add adds a line to the buffer
func (cb *CircularBuffer) Add(line string) {
	cb.buffer[cb.head] = line
	cb.head = (cb.head + 1) % cb.size
	if cb.count < cb.size {
		cb.count++
	}
}

// GetContext returns up to n lines before the current position
func (cb *CircularBuffer) GetContext(n int) []string {
	if n > cb.count {
		n = cb.count
	}
	
	result := make([]string, 0, n)
	start := (cb.head - n + cb.size) % cb.size
	
	for i := 0; i < n; i++ {
		idx := (start + i) % cb.size
		if cb.buffer[idx] != "" {
			result = append(result, cb.buffer[idx])
		}
	}
	
	return result
}

// matchesAnyPattern checks if a line matches any of the given patterns
func (f *OptimizedOutputFilter) matchesAnyPattern(line string, patterns []*config.RegexPattern) bool {
	for _, pattern := range patterns {
		re, err := f.patternCache.GetOrCompile(pattern)
		if err != nil {
			continue
		}
		if re.MatchString(line) {
			return true
		}
	}
	return false
}