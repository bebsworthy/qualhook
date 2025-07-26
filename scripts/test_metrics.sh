#!/bin/bash
# Test Quality Metrics Collection Script
# Collects comprehensive test metrics including coverage, execution time, and quality indicators

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
METRICS_DIR="test_metrics"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
RESULTS_DIR="$METRICS_DIR/results"
COVERAGE_DIR="$METRICS_DIR/coverage"
REPORTS_DIR="$METRICS_DIR/reports"
DASHBOARD_FILE="$METRICS_DIR/dashboard.html"
JSON_REPORT="$RESULTS_DIR/metrics_$TIMESTAMP.json"

# Create directories
mkdir -p "$RESULTS_DIR" "$COVERAGE_DIR" "$REPORTS_DIR"

# Initialize JSON report
cat > "$JSON_REPORT" << EOF
{
  "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "go_version": "$(go version | awk '{print $3}')",
  "os": "$(uname -s)",
  "arch": "$(uname -m)",
  "metrics": {}
}
EOF

echo -e "${BLUE}=== Qualhook Test Quality Metrics Collection ===${NC}"
echo "Starting at: $(date)"
echo ""

# Function to update JSON metrics
update_json() {
    local key=$1
    local value=$2
    local temp_file=$(mktemp)
    jq ".metrics.$key = $value" "$JSON_REPORT" > "$temp_file" && mv "$temp_file" "$JSON_REPORT"
}

# 1. Collect Package-Level Test Coverage
echo -e "${YELLOW}1. Collecting package-level test coverage...${NC}"

# Get list of all packages
PACKAGES=$(go list ./... | grep -v /vendor/ | grep -v /test/)
TOTAL_PACKAGES=$(echo "$PACKAGES" | wc -l | xargs)

# Initialize coverage data
COVERAGE_DATA="[]"
TOTAL_COVERAGE=0
PACKAGES_WITH_TESTS=0
PACKAGES_WITHOUT_TESTS=0

for pkg in $PACKAGES; do
    echo -n "  Testing $pkg... "
    
    # Run tests with coverage for this package
    COVERAGE_FILE="$COVERAGE_DIR/$(echo $pkg | tr '/' '_').coverage"
    if go test -tags="unit,integration,e2e" -coverprofile="$COVERAGE_FILE" -json "$pkg" > "$COVERAGE_DIR/$(echo $pkg | tr '/' '_').json" 2>/dev/null; then
        # Extract coverage percentage
        COVERAGE=$(go tool cover -func="$COVERAGE_FILE" 2>/dev/null | grep "total:" | awk '{print $3}' | sed 's/%//')
        if [ -z "$COVERAGE" ]; then
            COVERAGE="0.0"
        fi
        
        # Count test files
        PKG_DIR=$(go list -f '{{.Dir}}' "$pkg")
        TEST_FILES=$(find "$PKG_DIR" -maxdepth 1 -name "*_test.go" 2>/dev/null | wc -l | xargs)
        
        if [ "$TEST_FILES" -gt 0 ]; then
            PACKAGES_WITH_TESTS=$((PACKAGES_WITH_TESTS + 1))
            echo -e "${GREEN}✓${NC} Coverage: ${COVERAGE}%"
        else
            PACKAGES_WITHOUT_TESTS=$((PACKAGES_WITHOUT_TESTS + 1))
            echo -e "${YELLOW}⚠${NC} No tests"
        fi
        
        # Add to coverage data
        COVERAGE_DATA=$(echo "$COVERAGE_DATA" | jq ". += [{\"package\": \"$pkg\", \"coverage\": $COVERAGE, \"test_files\": $TEST_FILES}]")
        TOTAL_COVERAGE=$(echo "$TOTAL_COVERAGE + $COVERAGE" | bc)
    else
        echo -e "${RED}✗${NC} Failed"
        PACKAGES_WITHOUT_TESTS=$((PACKAGES_WITHOUT_TESTS + 1))
        COVERAGE_DATA=$(echo "$COVERAGE_DATA" | jq ". += [{\"package\": \"$pkg\", \"coverage\": 0, \"test_files\": 0, \"error\": true}]")
    fi
done

# Calculate average coverage
AVG_COVERAGE=$(echo "scale=2; $TOTAL_COVERAGE / $TOTAL_PACKAGES" | bc)

# Update JSON with coverage metrics
update_json "coverage" "{
    \"average\": $AVG_COVERAGE,
    \"packages_total\": $TOTAL_PACKAGES,
    \"packages_with_tests\": $PACKAGES_WITH_TESTS,
    \"packages_without_tests\": $PACKAGES_WITHOUT_TESTS,
    \"by_package\": $COVERAGE_DATA
}"

echo ""
echo -e "${GREEN}Coverage Summary:${NC}"
echo "  Average Coverage: ${AVG_COVERAGE}%"
echo "  Packages with tests: $PACKAGES_WITH_TESTS/$TOTAL_PACKAGES"
echo ""

# 2. Test Execution Time Analysis
echo -e "${YELLOW}2. Analyzing test execution times...${NC}"

# Run all tests with JSON output to capture timing
TEST_OUTPUT="$RESULTS_DIR/test_output_$TIMESTAMP.json"
go test -tags="unit,integration,e2e" -json -v ./... > "$TEST_OUTPUT" 2>&1 || true

# Parse test execution times
TEST_TIMES=$(cat "$TEST_OUTPUT" | jq -s '
    map(select(.Action == "pass" and .Test != null)) |
    group_by(.Package) |
    map({
        package: .[0].Package,
        tests: map({name: .Test, duration: .Elapsed}),
        total_duration: map(.Elapsed) | add,
        test_count: length
    })
')

# Find slowest tests
SLOWEST_TESTS=$(cat "$TEST_OUTPUT" | jq -s '
    map(select(.Action == "pass" and .Test != null)) |
    sort_by(.Elapsed) |
    reverse |
    .[0:10] |
    map({name: .Test, package: .Package, duration: .Elapsed})
')

update_json "execution_time" "{
    \"by_package\": $TEST_TIMES,
    \"slowest_tests\": $SLOWEST_TESTS
}"

echo "  Execution time analysis complete"
echo ""

# 3. Test Flakiness Detection
echo -e "${YELLOW}3. Detecting test flakiness...${NC}"

FLAKY_TESTS="[]"
FLAKY_COUNT=0

# Run tests multiple times to detect flakiness
for i in {1..3}; do
    echo -n "  Run $i/3... "
    go test -tags="unit,integration,e2e" ./... -json > "$RESULTS_DIR/flaky_run_$i.json" 2>&1 || true
    echo "done"
done

# Compare results to find flaky tests
echo "  Analyzing flakiness..."
FLAKY_ANALYSIS=$(cat "$RESULTS_DIR"/flaky_run_*.json | jq -s '
    # Flatten all test results from multiple runs
    flatten |
    # Select only pass/fail/skip actions for actual tests
    map(select(.Test != null and (.Action == "pass" or .Action == "fail" or .Action == "skip"))) |
    # Group by package and test name
    group_by(.Package + "." + .Test) |
    # Analyze each test for flakiness
    map({
        test: .[0].Test,
        package: .[0].Package,
        runs: length,
        # Get unique actions (pass/fail/skip)
        actions: map(.Action) | unique,
        # Test is flaky if it has different results across runs
        flaky: (map(.Action) | unique | length > 1)
    }) |
    # Keep only flaky tests
    map(select(.flaky))
')

FLAKY_COUNT=$(echo "$FLAKY_ANALYSIS" | jq 'length')

update_json "flakiness" "{
    \"flaky_test_count\": $FLAKY_COUNT,
    \"flaky_tests\": $FLAKY_ANALYSIS
}"

echo "  Found $FLAKY_COUNT flaky tests"
echo ""

# 4. Test Type Distribution
echo -e "${YELLOW}4. Analyzing test type distribution...${NC}"

# Count different test types
UNIT_TESTS=$(find . -name "*_test.go" -exec grep -l "//go:build unit\|// +build unit" {} \; 2>/dev/null | wc -l | xargs)
INTEGRATION_TESTS=$(find . -name "*_test.go" -exec grep -l "//go:build integration\|// +build integration" {} \; 2>/dev/null | wc -l | xargs)
E2E_TESTS=$(find . -name "*_test.go" -exec grep -l "//go:build e2e\|// +build e2e" {} \; 2>/dev/null | wc -l | xargs)
BENCHMARK_TESTS=$(find . -name "*_test.go" -exec grep -l "func Benchmark" {} \; 2>/dev/null | wc -l | xargs)
EXAMPLE_TESTS=$(find . -name "*_test.go" -exec grep -l "func Example" {} \; 2>/dev/null | wc -l | xargs)

# Total test files
TOTAL_TEST_FILES=$(find . -name "*_test.go" | wc -l | xargs)

update_json "test_types" "{
    \"total_test_files\": $TOTAL_TEST_FILES,
    \"unit_test_files\": $UNIT_TESTS,
    \"integration_test_files\": $INTEGRATION_TESTS,
    \"e2e_test_files\": $E2E_TESTS,
    \"benchmark_files\": $BENCHMARK_TESTS,
    \"example_files\": $EXAMPLE_TESTS
}"

echo "  Total test files: $TOTAL_TEST_FILES"
echo "  Unit tests: $UNIT_TESTS"
echo "  Integration tests: $INTEGRATION_TESTS"
echo "  E2E tests: $E2E_TESTS"
echo "  Benchmarks: $BENCHMARK_TESTS"
echo "  Examples: $EXAMPLE_TESTS"
echo ""

# 5. Code to Test Ratio
echo -e "${YELLOW}5. Calculating code to test ratio...${NC}"

# Count lines of code
SOURCE_LINES=$(find . -name "*.go" -not -name "*_test.go" -not -path "./vendor/*" -not -path "./test/*" -exec wc -l {} + | tail -1 | awk '{print $1}')
TEST_LINES=$(find . -name "*_test.go" -not -path "./vendor/*" -exec wc -l {} + | tail -1 | awk '{print $1}')
RATIO=$(echo "scale=2; $TEST_LINES / $SOURCE_LINES" | bc)

update_json "code_ratio" "{
    \"source_lines\": $SOURCE_LINES,
    \"test_lines\": $TEST_LINES,
    \"test_to_code_ratio\": $RATIO
}"

echo "  Source lines: $SOURCE_LINES"
echo "  Test lines: $TEST_LINES"
echo "  Test-to-code ratio: $RATIO"
echo ""

# 6. Generate Quality Score
echo -e "${YELLOW}6. Calculating test quality score...${NC}"

# Calculate quality score (0-100)
COVERAGE_SCORE=$(echo "scale=2; $AVG_COVERAGE" | bc)
RATIO_SCORE=$(echo "scale=2; $RATIO * 100" | bc | cut -d. -f1)
# Cap flaky penalty at 20 points max, and scale by percentage of tests that are flaky
TOTAL_TESTS=$(cat "$TEST_OUTPUT" | jq -s 'map(select(.Test != null and .Action == "pass")) | length')
if [ "$TOTAL_TESTS" -gt 0 ]; then
    FLAKY_PERCENTAGE=$(echo "scale=2; ($FLAKY_COUNT / $TOTAL_TESTS) * 100" | bc)
    # Cap flaky penalty at 20 points max
    if (( $(echo "$FLAKY_PERCENTAGE > 20" | bc -l) )); then
        FLAKY_PENALTY=20
    else
        FLAKY_PENALTY=$FLAKY_PERCENTAGE
    fi
else
    FLAKY_PENALTY=0
fi
TEST_PRESENCE_SCORE=$(echo "scale=2; ($PACKAGES_WITH_TESTS / $TOTAL_PACKAGES) * 100" | bc)

QUALITY_SCORE=$(echo "scale=2; ($COVERAGE_SCORE + $RATIO_SCORE + $TEST_PRESENCE_SCORE) / 3 - $FLAKY_PENALTY" | bc)
if (( $(echo "$QUALITY_SCORE < 0" | bc -l) )); then
    QUALITY_SCORE=0
fi

update_json "quality_score" "{
    \"overall\": $QUALITY_SCORE,
    \"components\": {
        \"coverage_score\": $COVERAGE_SCORE,
        \"ratio_score\": $RATIO_SCORE,
        \"test_presence_score\": $TEST_PRESENCE_SCORE,
        \"flaky_penalty\": $FLAKY_PENALTY
    }
}"

echo "  Quality Score: $QUALITY_SCORE/100"
echo ""

# 7. Generate Trend Data
echo -e "${YELLOW}7. Updating trend data...${NC}"

TREND_FILE="$METRICS_DIR/trend_data.json"
if [ ! -f "$TREND_FILE" ]; then
    echo "[]" > "$TREND_FILE"
fi

# Add current metrics to trend
CURRENT_METRICS=$(jq '{
    timestamp: .timestamp,
    coverage: .metrics.coverage.average,
    quality_score: .metrics.quality_score.overall,
    test_count: .metrics.test_types.total_test_files,
    flaky_count: .metrics.flakiness.flaky_test_count
}' "$JSON_REPORT")

jq ". += [$CURRENT_METRICS]" "$TREND_FILE" > "$TREND_FILE.tmp" && mv "$TREND_FILE.tmp" "$TREND_FILE"

# Keep only last 30 data points
jq '.[-30:]' "$TREND_FILE" > "$TREND_FILE.tmp" && mv "$TREND_FILE.tmp" "$TREND_FILE"

echo "  Trend data updated"
echo ""

# 8. Generate HTML Dashboard
echo -e "${YELLOW}8. Generating HTML dashboard...${NC}"

python3 scripts/generate_test_dashboard.py "$JSON_REPORT" "$TREND_FILE" "$DASHBOARD_FILE"

echo "  Dashboard generated: $DASHBOARD_FILE"
echo ""

# 9. Generate GitHub Actions Summary (if in CI)
if [ -n "$GITHUB_STEP_SUMMARY" ]; then
    echo -e "${YELLOW}9. Generating GitHub Actions summary...${NC}"
    
    cat >> "$GITHUB_STEP_SUMMARY" << EOF
## Test Quality Metrics

### Overall Quality Score: $QUALITY_SCORE/100

#### Coverage
- **Average Coverage:** ${AVG_COVERAGE}%
- **Packages with tests:** $PACKAGES_WITH_TESTS/$TOTAL_PACKAGES

#### Test Distribution
- **Total test files:** $TOTAL_TEST_FILES
- **Test-to-code ratio:** $RATIO

#### Issues
- **Flaky tests:** $FLAKY_COUNT

[View Full Dashboard](https://github.com/$GITHUB_REPOSITORY/blob/$GITHUB_SHA/$DASHBOARD_FILE)
EOF

    echo "  GitHub Actions summary generated"
fi

# 10. Check Quality Gates
echo -e "${BLUE}=== Quality Gates ===${NC}"

FAILED_GATES=0

# Coverage gate
if (( $(echo "$AVG_COVERAGE < 70" | bc -l) )); then
    echo -e "${RED}✗ Coverage below 70% threshold (${AVG_COVERAGE}%)${NC}"
    FAILED_GATES=$((FAILED_GATES + 1))
else
    echo -e "${GREEN}✓ Coverage meets threshold (${AVG_COVERAGE}%)${NC}"
fi

# Flaky test gate (only fail if more than 5% of tests are flaky)
FLAKY_THRESHOLD=5
if [ "$TOTAL_TESTS" -gt 0 ]; then
    FLAKY_PERCENTAGE_INT=$(echo "scale=0; ($FLAKY_COUNT * 100) / $TOTAL_TESTS" | bc)
    if [ "$FLAKY_PERCENTAGE_INT" -gt "$FLAKY_THRESHOLD" ]; then
        echo -e "${RED}✗ Found $FLAKY_COUNT flaky tests (${FLAKY_PERCENTAGE_INT}% of total)${NC}"
        FAILED_GATES=$((FAILED_GATES + 1))
    else
        if [ "$FLAKY_COUNT" -gt 0 ]; then
            echo -e "${YELLOW}⚠ Found $FLAKY_COUNT flaky tests (${FLAKY_PERCENTAGE_INT}% of total)${NC}"
        else
            echo -e "${GREEN}✓ No flaky tests detected${NC}"
        fi
    fi
else
    echo -e "${GREEN}✓ No tests to check for flakiness${NC}"
fi

# Quality score gate
if (( $(echo "$QUALITY_SCORE < 60" | bc -l) )); then
    echo -e "${RED}✗ Quality score below 60 threshold ($QUALITY_SCORE)${NC}"
    FAILED_GATES=$((FAILED_GATES + 1))
else
    echo -e "${GREEN}✓ Quality score meets threshold ($QUALITY_SCORE)${NC}"
fi

echo ""
echo -e "${BLUE}=== Test Metrics Collection Complete ===${NC}"
echo "Results saved to:"
echo "  - JSON Report: $JSON_REPORT"
echo "  - Dashboard: $DASHBOARD_FILE"
echo "  - Trend Data: $TREND_FILE"

# Exit with error if quality gates failed
if [ "$FAILED_GATES" -gt 0 ]; then
    echo ""
    echo -e "${RED}⚠️  $FAILED_GATES quality gates failed!${NC}"
    exit 1
fi

echo ""
echo -e "${GREEN}✅ All quality gates passed!${NC}"