#!/bin/bash
# Script to run all performance benchmarks and generate reports

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Create benchmark results directory
RESULTS_DIR="benchmark_results"
mkdir -p "$RESULTS_DIR"

# Timestamp for results
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
RESULTS_FILE="$RESULTS_DIR/benchmarks_$TIMESTAMP.txt"
SUMMARY_FILE="$RESULTS_DIR/summary_$TIMESTAMP.md"

echo -e "${GREEN}Running Qualhook Performance Benchmarks${NC}"
echo "Results will be saved to: $RESULTS_FILE"
echo ""

# Function to run benchmarks for a package
run_benchmark() {
    local package=$1
    local name=$2
    
    echo -e "${YELLOW}Running $name benchmarks...${NC}"
    echo "=== $name Benchmarks ===" >> "$RESULTS_FILE"
    echo "" >> "$RESULTS_FILE"
    
    # Run benchmarks with different configurations
    # Basic run
    go test -bench=. -benchmem -benchtime=10s "$package" >> "$RESULTS_FILE" 2>&1
    
    # Run with CPU profiling for selected benchmarks
    if [[ "$name" == "Pattern Matching" || "$name" == "Output Filtering" ]]; then
        echo -e "${YELLOW}  Running CPU profiling for $name...${NC}"
        go test -bench="BenchmarkPattern|BenchmarkOutput" -benchmem -benchtime=30s -cpuprofile="$RESULTS_DIR/cpu_${name// /_}_$TIMESTAMP.prof" "$package" > /dev/null 2>&1
    fi
    
    echo "" >> "$RESULTS_FILE"
    echo "" >> "$RESULTS_FILE"
}

# Run all benchmarks
echo "Starting benchmark suite..."
echo "Benchmark Results - $(date)" > "$RESULTS_FILE"
echo "======================================" >> "$RESULTS_FILE"
echo "" >> "$RESULTS_FILE"

# Pattern matching benchmarks
run_benchmark "./internal/filter" "Pattern Matching"

# Output filtering benchmarks
run_benchmark "./internal/filter" "Output Filtering"

# Startup and command execution benchmarks
run_benchmark "./cmd/qualhook" "Startup Time"

# Memory usage benchmarks
run_benchmark "./internal/executor" "Memory Usage"

# Generate summary report
echo -e "${GREEN}Generating summary report...${NC}"

cat > "$SUMMARY_FILE" << EOF
# Qualhook Performance Benchmark Summary

**Date:** $(date)  
**Go Version:** $(go version)  
**OS:** $(uname -s) $(uname -r)  
**CPU:** $(sysctl -n machdep.cpu.brand_string 2>/dev/null || grep "model name" /proc/cpuinfo | head -1 | cut -d: -f2 | xargs)

## Key Performance Metrics

### 1. Regex Pattern Matching
EOF

# Extract key metrics from results
echo '```' >> "$SUMMARY_FILE"
grep -A 5 "BenchmarkPatternMatching" "$RESULTS_FILE" | grep -E "BenchmarkPattern|ns/op|B/op|allocs/op" >> "$SUMMARY_FILE" || true
echo '```' >> "$SUMMARY_FILE"

cat >> "$SUMMARY_FILE" << EOF

### 2. Output Filtering
EOF

echo '```' >> "$SUMMARY_FILE"
grep -A 5 "BenchmarkOutputFiltering" "$RESULTS_FILE" | grep -E "BenchmarkOutput|ns/op|B/op|allocs/op" >> "$SUMMARY_FILE" || true
echo '```' >> "$SUMMARY_FILE"

cat >> "$SUMMARY_FILE" << EOF

### 3. Startup Time
EOF

echo '```' >> "$SUMMARY_FILE"
grep -A 3 "BenchmarkCLIStartup\|BenchmarkEndToEnd" "$RESULTS_FILE" | grep -E "Benchmark|ns/op" >> "$SUMMARY_FILE" || true
echo '```' >> "$SUMMARY_FILE"

cat >> "$SUMMARY_FILE" << EOF

### 4. Memory Usage with Large Outputs
EOF

echo '```' >> "$SUMMARY_FILE"
grep -A 3 "BenchmarkLargeOutputMemory\|BenchmarkWorstCaseMemory" "$RESULTS_FILE" | grep -E "Benchmark|B/op|allocs/op" >> "$SUMMARY_FILE" || true
echo '```' >> "$SUMMARY_FILE"

cat >> "$SUMMARY_FILE" << EOF

## Performance Analysis

### Startup Overhead
Based on the benchmarks, the CLI startup overhead is measured by the \`BenchmarkCLIStartup\` and \`BenchmarkEndToEnd\` tests.

### Regex Performance
Pattern compilation and matching performance is critical for output filtering. The benchmarks show:
- Simple patterns vs complex patterns
- Impact of pattern caching
- Performance with large inputs

### Memory Efficiency
Memory usage is tested with various output sizes:
- Small outputs (< 1MB)
- Medium outputs (1-10MB)
- Large outputs (> 10MB)

### Optimization Opportunities
1. **Pattern Caching**: Pre-compile frequently used patterns
2. **Streaming Processing**: Use streaming for large outputs to reduce memory usage
3. **Parallel Execution**: Leverage goroutines for monorepo scenarios

## Full Results
See the complete benchmark results in: \`$RESULTS_FILE\`
EOF

echo -e "${GREEN}Benchmarks completed!${NC}"
echo ""
echo "Results saved to:"
echo "  - Full results: $RESULTS_FILE"
echo "  - Summary: $SUMMARY_FILE"
echo ""

# Check if we meet performance requirements
echo -e "${YELLOW}Checking performance requirements...${NC}"

# Check startup overhead (requirement: < 100ms)
STARTUP_TIME=$(grep "BenchmarkEndToEnd" "$RESULTS_FILE" | awk '{print $3}' | sed 's/ns\/op//' | head -1)
if [ ! -z "$STARTUP_TIME" ]; then
    STARTUP_MS=$(echo "scale=2; $STARTUP_TIME / 1000000" | bc 2>/dev/null || echo "N/A")
    echo "  - Startup overhead: ${STARTUP_MS}ms (requirement: < 100ms)"
    
    if [ "$STARTUP_MS" != "N/A" ]; then
        if (( $(echo "$STARTUP_MS < 100" | bc -l 2>/dev/null || echo 0) )); then
            echo -e "    ${GREEN}✓ PASS${NC}"
        else
            echo -e "    ${RED}✗ FAIL${NC}"
        fi
    fi
fi

# Display summary
echo ""
echo -e "${GREEN}Benchmark suite completed successfully!${NC}"