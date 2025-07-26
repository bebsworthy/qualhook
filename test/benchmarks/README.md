# Test Benchmarking Infrastructure

This directory contains tools for tracking and analyzing test suite health, including execution time tracking, flakiness detection, and coverage reporting by category.

## Components

### 1. Test Execution Tracker (`tracker.go`)
Tracks test execution metrics including:
- Test duration
- Pass/fail status
- Memory allocations
- Output capture

### 2. Flakiness Detector (`flakiness.go`)
Identifies and analyzes flaky tests:
- Tracks test execution history
- Calculates flake scores
- Suggests retry strategies
- Identifies common failure patterns

### 3. Coverage Analyzer (`coverage.go`)
Analyzes test coverage by category:
- Parses Go coverage profiles
- Aggregates coverage by package and category
- Identifies uncovered code
- Generates coverage reports

### 4. Reporter (`reporter.go`)
Generates comprehensive test suite health reports:
- Text, JSON, and HTML formats
- Health score calculation
- Actionable recommendations
- Trend analysis

## Usage

### Command Line Tool

```bash
# Generate a comprehensive report
go run test/benchmarks/cmd/benchmark/main.go -action=report

# Analyze flakiness only
go run test/benchmarks/cmd/benchmark/main.go -action=analyze-flakiness -tracking-file=test-results.json

# Generate coverage collection script
go run test/benchmarks/cmd/benchmark/main.go -action=coverage
```

### Collecting Coverage by Category

1. Generate the coverage collection script:
```bash
go run test/benchmarks/cmd/benchmark/main.go -action=coverage
```

2. Run the script:
```bash
chmod +x test/benchmarks/collect_coverage.sh
./test/benchmarks/collect_coverage.sh
```

This will generate coverage files for each test category (unit, integration, e2e).

### Integration with Test Runner

To track test execution in your tests:

```go
package mypackage_test

import (
    "testing"
    "time"
    "github.com/bebsworthy/qualhook/test/benchmarks"
)

var tracker = benchmarks.NewTracker()

func TestMyFeature(t *testing.T) {
    start := time.Now()
    defer func() {
        result := benchmarks.TestResult{
            Package:   "mypackage",
            Test:      "TestMyFeature",
            Category:  "unit",
            StartTime: start,
            Duration:  time.Since(start),
            Passed:    !t.Failed(),
            Skipped:   t.Skipped(),
        }
        tracker.RecordTest(result)
    }()
    
    // Your test code here
}

func TestMain(m *testing.M) {
    run := tracker.StartRun("test-run-" + time.Now().Format("20060102-150405"))
    code := m.Run()
    tracker.FinishRun(run)
    
    // Save results
    tracker.SaveResults("test-results.json")
    
    os.Exit(code)
}
```

### Flakiness Detection

The flakiness detector tracks test execution history and identifies tests that fail intermittently:

```go
detector := benchmarks.NewFlakeDetector(0.1) // 10% threshold

// Record test executions
detector.RecordExecution("pkg", "TestFlaky", "run1", true, 100*time.Millisecond, "")
detector.RecordExecution("pkg", "TestFlaky", "run2", false, 150*time.Millisecond, "timeout")
detector.RecordExecution("pkg", "TestFlaky", "run3", true, 120*time.Millisecond, "")

// Analyze flakiness
report := detector.AnalyzeFlakiness()
```

## Report Formats

### Text Report
Human-readable summary with:
- Overall health score
- Test execution statistics
- Coverage by category
- Flaky test identification
- Recommendations

### JSON Report
Machine-readable format for:
- CI/CD integration
- Historical tracking
- Custom analysis tools

### HTML Report
Interactive web report with:
- Visual health indicators
- Sortable tables
- Detailed metrics
- Actionable insights

## Health Score Calculation

The health score (0-100%) is calculated based on:
- Test pass rate (weight: 20%)
- Flakiness rate (weight: 15%)
- Coverage percentage (weight: 50% of deficit below 80%)
- Test execution time

## Recommendations

The system provides actionable recommendations:
- Fix failing tests when failure rate > 5%
- Address flaky tests with retry strategies
- Improve coverage when below 80%
- Optimize slow tests (> 5 seconds)
- Balance test categories

## Best Practices

1. **Regular Monitoring**: Run benchmarking reports regularly (e.g., daily in CI)
2. **Track Trends**: Store historical data to identify trends
3. **Set Thresholds**: Define acceptable thresholds for your project
4. **Act on Recommendations**: Prioritize fixing identified issues
5. **Category Balance**: Maintain appropriate test distribution across categories

## Future Enhancements

- [ ] Historical trend analysis
- [ ] Integration with popular CI/CD platforms
- [ ] Real-time test monitoring dashboard
- [ ] Predictive flakiness detection
- [ ] Test impact analysis