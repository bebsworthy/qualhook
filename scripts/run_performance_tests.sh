#!/bin/bash
# Focused performance benchmark script for Task 27

set -e

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Create benchmark results directory
RESULTS_DIR="benchmark_results"
mkdir -p "$RESULTS_DIR"

# Timestamp for results
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
RESULTS_FILE="$RESULTS_DIR/performance_$TIMESTAMP.txt"

echo -e "${GREEN}Running Qualhook Performance Benchmarks${NC}"
echo "Results will be saved to: $RESULTS_FILE"
echo ""

# Header
echo "Qualhook Performance Benchmark Results" > "$RESULTS_FILE"
echo "=====================================" >> "$RESULTS_FILE"
echo "Date: $(date)" >> "$RESULTS_FILE"
echo "Go Version: $(go version)" >> "$RESULTS_FILE"
echo "" >> "$RESULTS_FILE"

# 1. Pattern Matching Benchmarks
echo -e "${YELLOW}1. Running Pattern Matching Benchmarks...${NC}"
echo "## Pattern Matching Performance" >> "$RESULTS_FILE"
echo "--------------------------------" >> "$RESULTS_FILE"
go test -bench="BenchmarkPattern" -benchmem -benchtime=2s ./internal/filter >> "$RESULTS_FILE" 2>&1
echo "" >> "$RESULTS_FILE"

# 2. Output Filtering Benchmarks (especially for large outputs)
echo -e "${YELLOW}2. Running Output Filtering Benchmarks...${NC}"
echo "## Output Filtering Performance" >> "$RESULTS_FILE"
echo "--------------------------------" >> "$RESULTS_FILE"
go test -bench="BenchmarkOutputFiltering|BenchmarkStreamFiltering" -benchmem -benchtime=2s ./internal/filter >> "$RESULTS_FILE" 2>&1
echo "" >> "$RESULTS_FILE"

# 3. Optimization Benchmarks
echo -e "${YELLOW}3. Running Optimization Benchmarks...${NC}"
echo "## Optimization Performance" >> "$RESULTS_FILE"
echo "------------------------------" >> "$RESULTS_FILE"
go test -bench="BenchmarkOptimized|BenchmarkLiteral|BenchmarkBatch" -benchmem -benchtime=2s ./internal/filter >> "$RESULTS_FILE" 2>&1
echo "" >> "$RESULTS_FILE"

# 4. Startup Time Benchmarks (if no build errors)
echo -e "${YELLOW}4. Testing Startup Time...${NC}"
echo "## Startup Time Measurement" >> "$RESULTS_FILE"
echo "------------------------------" >> "$RESULTS_FILE"

# Create a simple timing test
cat > "$RESULTS_DIR/startup_test.go" << 'EOF'
package main

import (
    "fmt"
    "os/exec"
    "time"
)

func main() {
    // Test help command startup
    times := make([]time.Duration, 10)
    for i := 0; i < 10; i++ {
        start := time.Now()
        cmd := exec.Command("./qualhook", "--help")
        _ = cmd.Run()
        times[i] = time.Since(start)
    }
    
    // Calculate average
    var total time.Duration
    for _, t := range times {
        total += t
    }
    avg := total / 10
    
    fmt.Printf("Average startup time (--help): %v\n", avg)
    fmt.Printf("Average startup time (ms): %.2f\n", float64(avg.Nanoseconds())/1e6)
}
EOF

# Build qualhook if not exists
if [ ! -f "./qualhook" ]; then
    echo "Building qualhook binary..." >> "$RESULTS_FILE"
    go build -o qualhook ./cmd/qualhook >> "$RESULTS_FILE" 2>&1
fi

# Run startup test
go run "$RESULTS_DIR/startup_test.go" >> "$RESULTS_FILE" 2>&1
echo "" >> "$RESULTS_FILE"

# 5. Memory Usage Analysis
echo -e "${YELLOW}5. Analyzing Memory Usage Patterns...${NC}"
echo "## Memory Usage Analysis" >> "$RESULTS_FILE"
echo "------------------------" >> "$RESULTS_FILE"

# Extract key memory metrics
echo "### Pattern Matching Memory Usage:" >> "$RESULTS_FILE"
grep -A 1 "BenchmarkPatternMatching" "$RESULTS_FILE" | grep -E "B/op|allocs/op" | head -10 >> "$RESULTS_FILE" || true
echo "" >> "$RESULTS_FILE"

echo "### Output Filtering Memory Usage (Large Outputs):" >> "$RESULTS_FILE"
grep -A 1 "LargeOutput" "$RESULTS_FILE" | grep -E "B/op|allocs/op" | head -10 >> "$RESULTS_FILE" || true
echo "" >> "$RESULTS_FILE"

# Performance Requirements Check
echo -e "${GREEN}Checking Performance Requirements...${NC}"
echo "" >> "$RESULTS_FILE"
echo "## Performance Requirements Check" >> "$RESULTS_FILE"
echo "---------------------------------" >> "$RESULTS_FILE"

# Check if startup time is under 100ms
STARTUP_MS=$(grep "Average startup time (ms):" "$RESULTS_FILE" | awk '{print $5}')
if [ ! -z "$STARTUP_MS" ]; then
    echo "Startup Time: ${STARTUP_MS}ms (Requirement: <100ms)" >> "$RESULTS_FILE"
    if (( $(echo "$STARTUP_MS < 100" | bc -l 2>/dev/null || echo 0) )); then
        echo "✓ PASS: Startup time meets requirement" >> "$RESULTS_FILE"
        echo -e "${GREEN}✓ Startup time: ${STARTUP_MS}ms (PASS)${NC}"
    else
        echo "✗ FAIL: Startup time exceeds requirement" >> "$RESULTS_FILE"
        echo -e "${RED}✗ Startup time: ${STARTUP_MS}ms (FAIL)${NC}"
    fi
fi

# Summary
echo "" >> "$RESULTS_FILE"
echo "## Summary" >> "$RESULTS_FILE"
echo "----------" >> "$RESULTS_FILE"
echo "Benchmarks completed at $(date)" >> "$RESULTS_FILE"

echo -e "${GREEN}Performance benchmarks completed!${NC}"
echo "Results saved to: $RESULTS_FILE"

# Display key results
echo ""
echo -e "${YELLOW}Key Results:${NC}"
echo "- Pattern matching performance: Check $RESULTS_FILE"
echo "- Memory usage with large outputs: Check $RESULTS_FILE"
echo "- Startup overhead: ${STARTUP_MS}ms"

# Clean up
rm -f "$RESULTS_DIR/startup_test.go"