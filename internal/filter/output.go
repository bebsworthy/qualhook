// Package filter provides output filtering and processing functionality for qualhook.
package filter

import (
	"bufio"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/bebsworthy/qualhook/internal/debug"
	"github.com/bebsworthy/qualhook/pkg/config"
)

// OutputFilter processes command output according to configured rules
type OutputFilter struct {
	rules         *FilterRules
	patternCache  *PatternCache
	maxBufferSize int
}

// FilteredOutput represents the result of filtering command output
type FilteredOutput struct {
	Lines      []string
	HasErrors  bool
	Truncated  bool
	TotalLines int
}

// NewOutputFilter creates a new output filter with the given rules
func NewOutputFilter(rules *FilterRules) (*OutputFilter, error) {
	if rules == nil {
		return nil, fmt.Errorf("filter rules cannot be nil")
	}

	cache, err := NewPatternCache()
	if err != nil {
		return nil, fmt.Errorf("failed to create pattern cache: %w", err)
	}

	// Pre-compile all patterns
	for _, pattern := range rules.ErrorPatterns {
		if _, err := cache.GetOrCompile(pattern); err != nil {
			return nil, fmt.Errorf("failed to compile error pattern %q: %w", pattern.Pattern, err)
		}
	}

	for _, pattern := range rules.ContextPatterns {
		if _, err := cache.GetOrCompile(pattern); err != nil {
			return nil, fmt.Errorf("failed to compile context pattern %q: %w", pattern.Pattern, err)
		}
	}

	return &OutputFilter{
		rules:         rules,
		patternCache:  cache,
		maxBufferSize: 10 * 1024 * 1024, // 10MB default
	}, nil
}

// Filter applies filtering rules to the given output
func (f *OutputFilter) Filter(output string) *FilteredOutput {
	reader := strings.NewReader(output)
	return f.FilterReader(reader)
}

// FilterReader applies filtering rules to output from a reader (supports streaming)
func (f *OutputFilter) FilterReader(reader io.Reader) *FilteredOutput {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 0, f.maxBufferSize), f.maxBufferSize)

	var (
		allLines     []string
		matchedLines []lineMatch
		totalLines   int
	)

	lineNum := 0
	for scanner.Scan() {
		line := scanner.Text()
		allLines = append(allLines, line)
		totalLines++
		lineNum++

		// Check if line matches any error pattern
		if f.matchesAnyPattern(line, f.rules.ErrorPatterns) {
			debug.LogPatternMatch("error patterns", line, true)
			matchedLines = append(matchedLines, lineMatch{
				lineNum: lineNum - 1, // 0-indexed
				line:    line,
				isError: true,
			})
		} else if f.matchesAnyPattern(line, f.rules.ContextPatterns) {
			debug.LogPatternMatch("include patterns", line, true)
			matchedLines = append(matchedLines, lineMatch{
				lineNum: lineNum - 1,
				line:    line,
				isError: false,
			})
		}
	}

	// Extract matched lines with context
	extractedLines := f.extractLinesWithContext(allLines, matchedLines)

	// Apply truncation if needed
	truncated := false
	if f.rules.MaxLines > 0 && len(extractedLines) > f.rules.MaxLines {
		// Re-map matched lines to their positions in extractedLines
		remappedMatches := f.remapMatches(allLines, extractedLines, matchedLines)
		extractedLines = f.intelligentTruncate(extractedLines, remappedMatches)
		truncated = true
	}

	return &FilteredOutput{
		Lines:      extractedLines,
		HasErrors:  f.hasErrors(matchedLines),
		Truncated:  truncated,
		TotalLines: totalLines,
	}
}

// FilterBoth filters both stdout and stderr, combining the results
func (f *OutputFilter) FilterBoth(stdout, stderr string) *FilteredOutput {
	stdoutResult := f.Filter(stdout)
	stderrResult := f.Filter(stderr)

	// Combine results, prioritizing stderr (typically contains errors)
	combined := &FilteredOutput{
		Lines:      make([]string, 0, len(stderrResult.Lines)+len(stdoutResult.Lines)),
		HasErrors:  stdoutResult.HasErrors || stderrResult.HasErrors,
		Truncated:  stdoutResult.Truncated || stderrResult.Truncated,
		TotalLines: stdoutResult.TotalLines + stderrResult.TotalLines,
	}

	// Add stderr lines first (higher priority)
	if len(stderrResult.Lines) > 0 {
		combined.Lines = append(combined.Lines, "=== STDERR ===")
		combined.Lines = append(combined.Lines, stderrResult.Lines...)
	}

	// Add stdout lines
	if len(stdoutResult.Lines) > 0 {
		if len(stderrResult.Lines) > 0 {
			combined.Lines = append(combined.Lines, "")
			combined.Lines = append(combined.Lines, "=== STDOUT ===")
		}
		combined.Lines = append(combined.Lines, stdoutResult.Lines...)
	}

	// Re-apply truncation to combined output
	if f.rules.MaxLines > 0 && len(combined.Lines) > f.rules.MaxLines {
		combined.Lines = combined.Lines[:f.rules.MaxLines]
		combined.Lines = append(combined.Lines, fmt.Sprintf("\n... truncated %d lines ...", len(combined.Lines)-f.rules.MaxLines))
		combined.Truncated = true
	}

	return combined
}

// StreamFilter filters output as it's being produced (for real-time processing)
func (f *OutputFilter) StreamFilter(reader io.Reader, writer io.Writer) error {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 0, f.maxBufferSize), f.maxBufferSize)

	bufWriter := bufio.NewWriter(writer)
	defer func() {
		_ = bufWriter.Flush() //nolint:errcheck // Best effort flush on defer
	}()

	var (
		buffer  []string
		lineNum int
		mu      sync.Mutex
	)

	// Process lines as they come
	for scanner.Scan() {
		line := scanner.Text()
		mu.Lock()
		buffer = append(buffer, line)
		lineNum++

		// Keep a sliding window of lines for context
		if len(buffer) > f.rules.ContextLines*2+100 {
			buffer = buffer[len(buffer)-f.rules.ContextLines*2-100:]
		}

		// Check if line matches patterns
		if f.matchesAnyPattern(line, f.rules.ErrorPatterns) {
			// Write the line with context immediately
			contextLines := f.getContextLines(buffer, len(buffer)-1, f.rules.ContextLines)
			for _, contextLine := range contextLines {
				_, _ = fmt.Fprintln(bufWriter, contextLine) //nolint:errcheck // Best effort output
			}
			_ = bufWriter.Flush() //nolint:errcheck // Best effort flush for immediate output
		}
		mu.Unlock()
	}

	return scanner.Err()
}

// Private helper types and methods

type lineMatch struct {
	lineNum int
	line    string
	isError bool
}

func (f *OutputFilter) matchesAnyPattern(line string, patterns []*config.RegexPattern) bool {
	for _, pattern := range patterns {
		re, err := f.patternCache.GetOrCompile(pattern)
		if err != nil {
			debug.LogError(err, "compiling pattern")
			continue // Skip invalid patterns
		}
		matched := re.MatchString(line)
		if debug.IsEnabled() && matched {
			debug.LogPatternMatch(pattern.Pattern, line, true)
		}
		if matched {
			return true
		}
	}
	return false
}

func (f *OutputFilter) extractLinesWithContext(allLines []string, matches []lineMatch) []string {
	if len(matches) == 0 {
		// No matches, return all lines if MaxOutput not set or small enough
		if f.rules.MaxLines <= 0 || len(allLines) <= f.rules.MaxLines {
			return allLines
		}
		// Otherwise return a sample
		if len(allLines) <= 10 {
			return allLines
		}
		return allLines[:10]
	}

	// Use a set to track which lines to include
	includeSet := make(map[int]bool)

	// Add matched lines and their context
	for _, match := range matches {
		// Add the matched line
		includeSet[match.lineNum] = true

		// Add context lines before
		for i := 1; i <= f.rules.ContextLines && match.lineNum-i >= 0; i++ {
			includeSet[match.lineNum-i] = true
		}

		// Add context lines after
		for i := 1; i <= f.rules.ContextLines && match.lineNum+i < len(allLines); i++ {
			includeSet[match.lineNum+i] = true
		}
	}

	// Extract lines in order
	var result []string
	lastIncluded := -1

	for i := 0; i < len(allLines); i++ {
		if includeSet[i] {
			// Add separator if there's a gap
			if lastIncluded >= 0 && i-lastIncluded > 1 {
				result = append(result, "...")
			}
			result = append(result, allLines[i])
			lastIncluded = i
		}
	}

	return result
}

func (f *OutputFilter) remapMatches(allLines, extractedLines []string, originalMatches []lineMatch) []lineMatch {
	// Create a map from line content to indices in extractedLines
	lineToIndex := make(map[string][]int)
	for i, line := range extractedLines {
		lineToIndex[line] = append(lineToIndex[line], i)
	}

	var remapped []lineMatch
	for _, match := range originalMatches {
		if match.lineNum < len(allLines) {
			line := allLines[match.lineNum]
			if indices, ok := lineToIndex[line]; ok && len(indices) > 0 {
				// Use the first occurrence
				remapped = append(remapped, lineMatch{
					lineNum: indices[0],
					line:    line,
					isError: match.isError,
				})
				// Remove used index to handle duplicates
				lineToIndex[line] = indices[1:]
			}
		}
	}

	return remapped
}

func (f *OutputFilter) intelligentTruncate(lines []string, matches []lineMatch) []string {
	if len(lines) <= f.rules.MaxLines {
		return lines
	}

	// Prioritize error lines
	errorIndices := make(map[int]bool)
	for _, match := range matches {
		if match.isError {
			errorIndices[match.lineNum] = true
		}
	}

	// Build result prioritizing errors
	var result []string
	includedIndices := make(map[int]bool)
	errorCount := 0

	// First pass: include all error lines
	for i, line := range lines {
		if errorIndices[i] && len(result) < f.rules.MaxLines-1 { // Reserve space for truncation message
			result = append(result, line)
			includedIndices[i] = true
			errorCount++
		}
	}

	// Second pass: fill remaining space with other lines
	remaining := f.rules.MaxLines - len(result) - 1 // Reserve space for truncation message
	for i, line := range lines {
		if remaining <= 0 {
			break
		}
		if !includedIndices[i] {
			result = append(result, line)
			includedIndices[i] = true
			remaining--
		}
	}

	// Add truncation indicator
	truncatedCount := len(lines) - len(result)
	if truncatedCount > 0 {
		result = append(result, fmt.Sprintf("... truncated %d lines (preserved %d error lines) ...", truncatedCount, errorCount))
	}

	return result
}

func (f *OutputFilter) getContextLines(buffer []string, matchIndex int, contextSize int) []string {
	start := matchIndex - contextSize
	if start < 0 {
		start = 0
	}

	end := matchIndex + contextSize + 1
	if end > len(buffer) {
		end = len(buffer)
	}

	return buffer[start:end]
}

func (f *OutputFilter) hasErrors(matches []lineMatch) bool {
	for _, match := range matches {
		if match.isError {
			return true
		}
	}
	return false
}

// SetMaxBufferSize sets the maximum buffer size for streaming operations
func (f *OutputFilter) SetMaxBufferSize(size int) {
	f.maxBufferSize = size
}

// FilterRules defines the rules for filtering output
type FilterRules struct {
	ErrorPatterns   []*config.RegexPattern
	ContextPatterns []*config.RegexPattern
	MaxLines        int
	ContextLines    int
	Priority        string
}

// NewSimpleOutputFilter creates a new output filter without rules (for simple filtering)
func NewSimpleOutputFilter() *OutputFilter {
	cache, err := NewPatternCache()
	if err != nil {
		// Pattern cache is optional for simple filtering, continue without it
		cache = nil
	}
	return &OutputFilter{
		patternCache:  cache,
		maxBufferSize: 10 * 1024 * 1024, // 10MB default
	}
}

// FilterWithRules applies filtering rules to the given output
func (o *OutputFilter) FilterWithRules(output string, rules *FilterRules) *FilteredOutput {
	// Create a temporary filter with the rules
	filter := &OutputFilter{
		rules:         rules,
		patternCache:  o.patternCache,
		maxBufferSize: o.maxBufferSize,
	}

	return filter.Filter(output)
}
