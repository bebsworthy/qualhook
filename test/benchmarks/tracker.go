// Package benchmarks provides test execution tracking and analysis tools
package benchmarks

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// TestResult represents the result of a single test execution
type TestResult struct {
	Package     string        `json:"package"`
	Test        string        `json:"test"`
	Category    string        `json:"category"` // unit, integration, e2e
	StartTime   time.Time     `json:"start_time"`
	Duration    time.Duration `json:"duration"`
	Passed      bool          `json:"passed"`
	Skipped     bool          `json:"skipped"`
	Error       string        `json:"error,omitempty"`
	Output      string        `json:"output,omitempty"`
	Allocations int64         `json:"allocations,omitempty"`
	MemoryBytes int64         `json:"memory_bytes,omitempty"`
}

// TestRun represents a complete test run
type TestRun struct {
	ID        string        `json:"id"`
	StartTime time.Time     `json:"start_time"`
	EndTime   time.Time     `json:"end_time"`
	Duration  time.Duration `json:"duration"`
	Results   []TestResult  `json:"results"`
}

// Tracker tracks test execution metrics
type Tracker struct {
	mu      sync.RWMutex
	results []TestResult
	runs    []TestRun
}

// NewTracker creates a new test execution tracker
func NewTracker() *Tracker {
	return &Tracker{
		results: make([]TestResult, 0),
		runs:    make([]TestRun, 0),
	}
}

// RecordTest records a test result
func (t *Tracker) RecordTest(result TestResult) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.results = append(t.results, result)
}

// StartRun starts tracking a new test run
func (t *Tracker) StartRun(id string) *TestRun {
	run := &TestRun{
		ID:        id,
		StartTime: time.Now(),
		Results:   make([]TestResult, 0),
	}
	return run
}

// FinishRun completes a test run
func (t *Tracker) FinishRun(run *TestRun) {
	t.mu.Lock()
	defer t.mu.Unlock()
	
	run.EndTime = time.Now()
	run.Duration = run.EndTime.Sub(run.StartTime)
	run.Results = t.results
	
	t.runs = append(t.runs, *run)
}

// GetStats returns statistics for the tracked tests
func (t *Tracker) GetStats() TestStats {
	t.mu.RLock()
	defer t.mu.RUnlock()
	
	stats := TestStats{
		TotalTests:   len(t.results),
		ByCategory:   make(map[string]CategoryStats),
		ByPackage:    make(map[string]PackageStats),
		SlowestTests: make([]TestResult, 0),
	}
	
	// Calculate statistics
	for _, result := range t.results {
		// Update category stats
		catStats := stats.ByCategory[result.Category]
		catStats.Total++
		if result.Passed {
			catStats.Passed++
		}
		if result.Skipped {
			catStats.Skipped++
		}
		catStats.TotalDuration += result.Duration
		stats.ByCategory[result.Category] = catStats
		
		// Update package stats
		pkgStats := stats.ByPackage[result.Package]
		pkgStats.Total++
		if result.Passed {
			pkgStats.Passed++
		}
		if result.Skipped {
			pkgStats.Skipped++
		}
		pkgStats.TotalDuration += result.Duration
		stats.ByPackage[result.Package] = pkgStats
		
		// Update totals
		if result.Passed {
			stats.Passed++
		}
		if result.Skipped {
			stats.Skipped++
		}
		stats.TotalDuration += result.Duration
	}
	
	// Find slowest tests
	stats.SlowestTests = t.findSlowestTests(10)
	
	return stats
}

// findSlowestTests returns the n slowest tests
func (t *Tracker) findSlowestTests(n int) []TestResult {
	if len(t.results) == 0 {
		return []TestResult{}
	}
	
	// Create a copy and sort by duration
	sorted := make([]TestResult, len(t.results))
	copy(sorted, t.results)
	
	// Simple bubble sort for small datasets
	for i := 0; i < len(sorted)-1; i++ {
		for j := 0; j < len(sorted)-i-1; j++ {
			if sorted[j].Duration < sorted[j+1].Duration {
				sorted[j], sorted[j+1] = sorted[j+1], sorted[j]
			}
		}
	}
	
	if n > len(sorted) {
		n = len(sorted)
	}
	
	return sorted[:n]
}

// SaveResults saves test results to a JSON file
func (t *Tracker) SaveResults(filename string) error {
	t.mu.RLock()
	defer t.mu.RUnlock()
	
	data, err := json.MarshalIndent(t.runs, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal results: %w", err)
	}
	
	if err := os.MkdirAll(filepath.Dir(filename), 0750); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	
	if err := os.WriteFile(filename, data, 0600); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}
	
	return nil
}

// LoadResults loads test results from a JSON file
func (t *Tracker) LoadResults(filename string) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	
	data, err := os.ReadFile(filename) // #nosec G304 - filename provided by caller
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}
	
	var runs []TestRun
	if err := json.Unmarshal(data, &runs); err != nil {
		return fmt.Errorf("failed to unmarshal results: %w", err)
	}
	
	t.runs = runs
	
	// Rebuild results from runs
	t.results = make([]TestResult, 0)
	for _, run := range runs {
		t.results = append(t.results, run.Results...)
	}
	
	return nil
}

// TestStats represents aggregated test statistics
type TestStats struct {
	TotalTests    int                       `json:"total_tests"`
	Passed        int                       `json:"passed"`
	Failed        int                       `json:"failed"`
	Skipped       int                       `json:"skipped"`
	TotalDuration time.Duration             `json:"total_duration"`
	ByCategory    map[string]CategoryStats  `json:"by_category"`
	ByPackage     map[string]PackageStats   `json:"by_package"`
	SlowestTests  []TestResult              `json:"slowest_tests"`
}

// CategoryStats represents statistics for a test category
type CategoryStats struct {
	Total         int           `json:"total"`
	Passed        int           `json:"passed"`
	Failed        int           `json:"failed"`
	Skipped       int           `json:"skipped"`
	TotalDuration time.Duration `json:"total_duration"`
	AvgDuration   time.Duration `json:"avg_duration"`
}

// PackageStats represents statistics for a package
type PackageStats struct {
	Total         int           `json:"total"`
	Passed        int           `json:"passed"`
	Failed        int           `json:"failed"`
	Skipped       int           `json:"skipped"`
	TotalDuration time.Duration `json:"total_duration"`
	AvgDuration   time.Duration `json:"avg_duration"`
}

// GetFailed returns the number of failed tests
func (s TestStats) GetFailed() int {
	return s.TotalTests - s.Passed - s.Skipped
}

// Calculate average durations
func (s *TestStats) CalculateAverages() {
	for cat, stats := range s.ByCategory {
		if stats.Total > 0 {
			stats.AvgDuration = stats.TotalDuration / time.Duration(stats.Total)
			s.ByCategory[cat] = stats
		}
	}
	
	for pkg, stats := range s.ByPackage {
		if stats.Total > 0 {
			stats.AvgDuration = stats.TotalDuration / time.Duration(stats.Total)
			s.ByPackage[pkg] = stats
		}
	}
}