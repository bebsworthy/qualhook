// Package benchmarks provides flakiness detection for tests
package benchmarks

import (
	"fmt"
	"sync"
	"time"
)

// FlakeDetector tracks test flakiness across multiple runs
type FlakeDetector struct {
	mu        sync.RWMutex
	history   map[string]*TestHistory
	threshold float64 // Percentage threshold for considering a test flaky
}

// TestHistory tracks the execution history of a single test
type TestHistory struct {
	TestName    string          `json:"test_name"`
	Package     string          `json:"package"`
	Executions  []TestExecution `json:"executions"`
	FlakeScore  float64         `json:"flake_score"`
	IsFlaky     bool            `json:"is_flaky"`
	LastUpdated time.Time       `json:"last_updated"`
}

// TestExecution represents a single test execution
type TestExecution struct {
	RunID     string        `json:"run_id"`
	Timestamp time.Time     `json:"timestamp"`
	Passed    bool          `json:"passed"`
	Duration  time.Duration `json:"duration"`
	Error     string        `json:"error,omitempty"`
}

// NewFlakeDetector creates a new flakiness detector
func NewFlakeDetector(threshold float64) *FlakeDetector {
	if threshold <= 0 || threshold >= 1 {
		threshold = 0.1 // Default 10% failure rate threshold
	}

	return &FlakeDetector{
		history:   make(map[string]*TestHistory),
		threshold: threshold,
	}
}

// RecordExecution records a test execution
func (fd *FlakeDetector) RecordExecution(pkg, test, runID string, passed bool, duration time.Duration, err string) {
	fd.mu.Lock()
	defer fd.mu.Unlock()

	key := fmt.Sprintf("%s.%s", pkg, test)

	if _, exists := fd.history[key]; !exists {
		fd.history[key] = &TestHistory{
			TestName:   test,
			Package:    pkg,
			Executions: make([]TestExecution, 0),
		}
	}

	execution := TestExecution{
		RunID:     runID,
		Timestamp: time.Now(),
		Passed:    passed,
		Duration:  duration,
		Error:     err,
	}

	fd.history[key].Executions = append(fd.history[key].Executions, execution)
	fd.history[key].LastUpdated = time.Now()

	// Update flake score
	fd.updateFlakeScore(fd.history[key])
}

// updateFlakeScore calculates the flake score for a test
func (fd *FlakeDetector) updateFlakeScore(history *TestHistory) {
	if len(history.Executions) < 2 {
		history.FlakeScore = 0
		history.IsFlaky = false
		return
	}

	// Count failures and check for inconsistent results
	failures := 0
	hasPass := false
	hasFail := false

	// Look at recent executions (last 20 runs)
	start := 0
	if len(history.Executions) > 20 {
		start = len(history.Executions) - 20
	}

	recentExecutions := history.Executions[start:]

	for _, exec := range recentExecutions {
		if exec.Passed {
			hasPass = true
		} else {
			hasFail = true
			failures++
		}
	}

	// Calculate flake score
	if hasPass && hasFail {
		// Test has both passed and failed - potentially flaky
		history.FlakeScore = float64(failures) / float64(len(recentExecutions))
		history.IsFlaky = history.FlakeScore > fd.threshold && history.FlakeScore < (1-fd.threshold)
	} else {
		// Test consistently passes or fails
		history.FlakeScore = 0
		history.IsFlaky = false
	}
}

// GetFlakyTests returns all tests identified as flaky
func (fd *FlakeDetector) GetFlakyTests() []TestHistory {
	fd.mu.RLock()
	defer fd.mu.RUnlock()

	flaky := make([]TestHistory, 0)

	for _, history := range fd.history {
		if history.IsFlaky {
			flaky = append(flaky, *history)
		}
	}

	return flaky
}

// GetTestHistory returns the history for a specific test
func (fd *FlakeDetector) GetTestHistory(pkg, test string) (*TestHistory, bool) {
	fd.mu.RLock()
	defer fd.mu.RUnlock()

	key := fmt.Sprintf("%s.%s", pkg, test)
	history, exists := fd.history[key]
	if !exists {
		return nil, false
	}

	return history, true
}

// AnalyzeFlakiness performs a detailed analysis of test flakiness
func (fd *FlakeDetector) AnalyzeFlakiness() FlakinessReport {
	fd.mu.RLock()
	defer fd.mu.RUnlock()

	report := FlakinessReport{
		TotalTests:       len(fd.history),
		FlakyTests:       make([]FlakyTestDetail, 0),
		Timestamp:        time.Now(),
		ThresholdPercent: fd.threshold * 100,
	}

	for _, history := range fd.history {
		if history.IsFlaky {
			detail := FlakyTestDetail{
				TestName:       history.TestName,
				Package:        history.Package,
				FlakeScore:     history.FlakeScore,
				TotalRuns:      len(history.Executions),
				Failures:       0,
				LastFailure:    time.Time{},
				FailureReasons: make(map[string]int),
			}

			// Analyze failures
			for _, exec := range history.Executions {
				if !exec.Passed {
					detail.Failures++
					if exec.Timestamp.After(detail.LastFailure) {
						detail.LastFailure = exec.Timestamp
					}
					if exec.Error != "" {
						detail.FailureReasons[exec.Error]++
					}
				}
			}

			detail.FailureRate = float64(detail.Failures) / float64(detail.TotalRuns)
			report.FlakyTests = append(report.FlakyTests, detail)
			report.TotalFlakyTests++
		}
	}

	return report
}

// FlakinessReport represents a comprehensive flakiness analysis
type FlakinessReport struct {
	Timestamp        time.Time         `json:"timestamp"`
	TotalTests       int               `json:"total_tests"`
	TotalFlakyTests  int               `json:"total_flaky_tests"`
	ThresholdPercent float64           `json:"threshold_percent"`
	FlakyTests       []FlakyTestDetail `json:"flaky_tests"`
}

// FlakyTestDetail provides detailed information about a flaky test
type FlakyTestDetail struct {
	TestName       string         `json:"test_name"`
	Package        string         `json:"package"`
	FlakeScore     float64        `json:"flake_score"`
	FailureRate    float64        `json:"failure_rate"`
	TotalRuns      int            `json:"total_runs"`
	Failures       int            `json:"failures"`
	LastFailure    time.Time      `json:"last_failure"`
	FailureReasons map[string]int `json:"failure_reasons"`
}

// SuggestRetries suggests retry strategies for flaky tests
func (fd *FlakeDetector) SuggestRetries() map[string]RetryStrategy {
	fd.mu.RLock()
	defer fd.mu.RUnlock()

	strategies := make(map[string]RetryStrategy)

	for key, history := range fd.history {
		if history.IsFlaky {
			strategy := RetryStrategy{
				TestName: history.TestName,
				Package:  history.Package,
			}

			// Determine retry count based on flake score
			switch {
			case history.FlakeScore < 0.2:
				strategy.MaxRetries = 1
			case history.FlakeScore < 0.4:
				strategy.MaxRetries = 2
			default:
				strategy.MaxRetries = 3
			}

			// Check for timing-related flakiness
			var durations []time.Duration
			for _, exec := range history.Executions {
				durations = append(durations, exec.Duration)
			}

			avgDuration := calculateAverage(durations)
			maxDuration := calculateMax(durations)

			// If max duration is significantly higher than average, suggest timeout increase
			if maxDuration > avgDuration*2 {
				strategy.TimeoutMultiplier = 2.0
				strategy.Notes = append(strategy.Notes, "Consider increasing timeout - high duration variance detected")
			}

			strategies[key] = strategy
		}
	}

	return strategies
}

// RetryStrategy suggests retry configuration for a flaky test
type RetryStrategy struct {
	TestName          string   `json:"test_name"`
	Package           string   `json:"package"`
	MaxRetries        int      `json:"max_retries"`
	TimeoutMultiplier float64  `json:"timeout_multiplier"`
	Notes             []string `json:"notes"`
}

// Helper functions
func calculateAverage(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}

	var total time.Duration
	for _, d := range durations {
		total += d
	}

	return total / time.Duration(len(durations))
}

func calculateMax(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}

	max := durations[0]
	for _, d := range durations[1:] {
		if d > max {
			max = d
		}
	}

	return max
}
