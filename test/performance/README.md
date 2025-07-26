# Performance Regression Tests

This directory contains performance regression tests for qualhook to detect performance degradation over time.

## Running Performance Tests

### Run all performance regression tests:
```bash
go test -v -tags=performance ./test/performance/...
```

### Run specific test categories:
```bash
# Startup performance
go test -v -tags=performance -run TestStartupPerformance ./test/performance/...

# Pattern matching performance
go test -v -tags=performance -run TestPatternMatchingPerformance ./test/performance/...

# Memory usage regression
go test -v -tags=performance -run TestMemoryUsageRegression ./test/performance/...

# Concurrent execution performance
go test -v -tags=performance -run TestConcurrentExecutionPerformance ./test/performance/...
```

### Run benchmarks:
```bash
# All benchmarks
go test -bench=. -tags=performance -run=^$ ./test/performance/...

# Summary benchmark for tracking
go test -bench=BenchmarkRegressionSummary -tags=performance -run=^$ ./test/performance/...
```

## Performance Baselines

The tests use predefined performance baselines that may need adjustment based on your hardware:

### Startup Time Baselines (milliseconds)
- Help command: 50ms
- Version command: 30ms
- Command startup: 100ms

### Pattern Matching Baselines (ops/second)
- Simple pattern, small input: 1,000,000 ops/sec
- Simple pattern, large input: 10,000 ops/sec
- Complex pattern, small input: 500,000 ops/sec
- Complex pattern, large input: 5,000 ops/sec

### Memory Usage Baselines (bytes)
- Small command execution: 1KB per operation
- Large command output: 1MB per operation
- Pattern compilation: 10KB per pattern
- Concurrent execution: 10MB total

### Concurrent Execution Baselines
- Throughput: 100 ops/sec minimum
- Max memory: 100MB

## Adjusting Baselines

If tests fail due to hardware differences, adjust the baselines in `regression_test.go`:

```go
var performanceBaselines = struct {
    StartupHelp    float64
    StartupVersion float64
    // ... other baselines
}{
    StartupHelp:    50,  // Adjust based on your hardware
    StartupVersion: 30,  // Adjust based on your hardware
    // ...
}
```

## Test Categories

### 1. Startup Performance (`TestStartupPerformance`)
Tests CLI startup time for various commands to ensure quick response times.

### 2. Pattern Matching Performance (`TestPatternMatchingPerformance`)
Tests regex pattern compilation and matching performance with various input sizes.

### 3. Pattern Set Performance (`TestPatternSetPerformance`)
Tests performance when matching against multiple patterns simultaneously.

### 4. Memory Usage Regression (`TestMemoryUsageRegression`)
Tests memory allocation and usage for various operations:
- Small command execution
- Large command output handling
- Pattern compilation
- Concurrent operations

### 5. Concurrent Execution Performance (`TestConcurrentExecutionPerformance`)
Tests performance under concurrent load with multiple parallel commands.

### 6. Config Loading Performance (`TestConfigLoadingPerformance`)
Tests configuration file loading and validation performance.

### 7. Project Detection Performance (`TestProjectDetectionPerformance`)
Tests project type detection performance for various project structures.

### 8. End-to-End Performance (`TestEndToEndPerformance`)
Tests full command execution performance from config loading to result output.

### 9. Stress Performance (`TestStressPerformance`)
Tests performance under stress conditions:
- Many patterns compilation
- Concurrent pattern access
- Rapid command execution

## Continuous Integration

To run performance tests in CI:

```yaml
- name: Run Performance Tests
  run: go test -v -tags=performance ./test/performance/... -timeout 5m
  
- name: Run Performance Benchmarks
  run: go test -bench=BenchmarkRegressionSummary -tags=performance -run=^$ ./test/performance/... -benchtime=30s
```

## Interpreting Results

- **PASS**: Performance is within acceptable baselines
- **FAIL**: Performance has regressed beyond acceptable thresholds
- Check the test output for actual vs baseline values
- Consider hardware differences when interpreting results

## Adding New Performance Tests

1. Add test data or scenarios to the test constants
2. Create a new test function following the pattern:
   ```go
   func TestNewPerformanceArea(t *testing.T) {
       if testing.Short() {
           t.Skip("Skipping performance regression test in short mode")
       }
       // ... test implementation
   }
   ```
3. Define appropriate baselines in `performanceBaselines`
4. Measure performance and compare against baselines
5. Use `t.Errorf` to fail if performance regresses
6. Use `t.Logf` to log actual performance metrics