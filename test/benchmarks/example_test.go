//go:build unit

package benchmarks_test

import (
	"os"
	"testing"
	"time"

	"github.com/bebsworthy/qualhook/test/benchmarks"
)

// Example of how to use the benchmarking infrastructure in tests
var (
	tracker       = benchmarks.NewTracker()
	flakeDetector = benchmarks.NewFlakeDetector(0.1)
	testRunID     = "example-run-" + time.Now().Format("20060102-150405")
)

func TestExample(t *testing.T) {
	// Track test execution
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		passed := !t.Failed()

		// Record in tracker
		result := benchmarks.TestResult{
			Package:   "benchmarks_test",
			Test:      "TestExample",
			Category:  "unit",
			StartTime: start,
			Duration:  duration,
			Passed:    passed,
			Skipped:   t.Skipped(),
		}
		tracker.RecordTest(result)

		// Record in flake detector
		var errMsg string
		if !passed {
			errMsg = "test failed"
		}
		flakeDetector.RecordExecution(
			"benchmarks_test",
			"TestExample",
			testRunID,
			passed,
			duration,
			errMsg,
		)
	}()

	// Your actual test code here
	time.Sleep(10 * time.Millisecond) // Simulate some work

	if testing.Short() {
		t.Skip("Skipping in short mode")
	}
}

func TestExampleFlaky(t *testing.T) {
	// Example of a potentially flaky test
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		passed := !t.Failed()

		result := benchmarks.TestResult{
			Package:   "benchmarks_test",
			Test:      "TestExampleFlaky",
			Category:  "unit",
			StartTime: start,
			Duration:  duration,
			Passed:    passed,
			Skipped:   t.Skipped(),
		}
		tracker.RecordTest(result)

		var errMsg string
		if !passed {
			errMsg = "random failure"
		}
		flakeDetector.RecordExecution(
			"benchmarks_test",
			"TestExampleFlaky",
			testRunID,
			passed,
			duration,
			errMsg,
		)
	}()

	// Skip the flaky test simulation in normal runs
	t.Skip("Skipping flaky test simulation")
}

func TestExampleSlow(t *testing.T) {
	// Example of a slow test
	start := time.Now()
	defer func() {
		duration := time.Since(start)

		result := benchmarks.TestResult{
			Package:   "benchmarks_test",
			Test:      "TestExampleSlow",
			Category:  "unit",
			StartTime: start,
			Duration:  duration,
			Passed:    !t.Failed(),
			Skipped:   t.Skipped(),
		}
		tracker.RecordTest(result)
	}()

	if testing.Short() {
		t.Skip("Skipping slow test in short mode")
	}

	// Simulate a slow operation
	time.Sleep(100 * time.Millisecond)
}

// TestMain demonstrates how to save benchmarking data
func TestMain(m *testing.M) {
	// Start tracking the test run
	run := tracker.StartRun(testRunID)

	// Run tests
	code := m.Run()

	// Finish tracking
	tracker.FinishRun(run)

	// Save results (optional - you might want to do this only in CI)
	if os.Getenv("SAVE_BENCHMARK_DATA") == "true" {
		if err := tracker.SaveResults("test/benchmarks/example-results.json"); err != nil {
			os.Stderr.WriteString("Failed to save benchmark results: " + err.Error() + "\n")
		}

		// Generate a simple report
		reporter := benchmarks.NewReporter(tracker, flakeDetector, benchmarks.NewCoverageAnalyzer())
		summary := reporter.GenerateSummary()

		if err := reporter.WriteTextReport(os.Stdout, summary); err != nil {
			os.Stderr.WriteString("Failed to write report: " + err.Error() + "\n")
		}
	}

	os.Exit(code)
}
