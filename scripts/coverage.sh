#!/bin/bash
# Run tests with coverage for all build tags

set -e

echo "Running tests with coverage for all build tags..."
echo ""

# Run tests with all tags to ensure maximum coverage
go test -tags="unit integration e2e test" \
    -coverprofile=coverage.out \
    -covermode=atomic \
    ./... 2>&1 | grep -v "no test files" || true

echo ""
echo "Coverage Summary:"
echo "================="

# Show total coverage
TOTAL_COVERAGE=$(go tool cover -func=coverage.out 2>/dev/null | grep "total:" | awk '{print $3}')
echo "Total Coverage: $TOTAL_COVERAGE"
echo ""

# Show coverage by package
echo "Coverage by Package:"
echo "-------------------"
go test -tags="unit integration e2e test" ./... 2>&1 | \
    grep -E "^(ok|FAIL).*coverage:" | \
    grep -v "no test files" | \
    sort | \
    awk '{
        if ($1 == "ok") {
            printf "✓ %-60s %s\n", $2, $4
        } else {
            printf "✗ %-60s %s\n", $2, "FAILED"
        }
    }'

# Generate HTML report
if [ "$1" = "--html" ]; then
    echo ""
    echo "Generating HTML coverage report..."
    go tool cover -html=coverage.out -o coverage.html
    echo "Coverage report saved to: coverage.html"
    
    # Try to open in browser
    if command -v open >/dev/null 2>&1; then
        open coverage.html
    elif command -v xdg-open >/dev/null 2>&1; then
        xdg-open coverage.html
    fi
fi

# Check if coverage meets threshold
THRESHOLD=50
COVERAGE_NUM=$(echo "$TOTAL_COVERAGE" | sed 's/%//')
if (( $(echo "$COVERAGE_NUM >= $THRESHOLD" | bc -l) )); then
    echo ""
    echo "✅ Coverage meets threshold of $THRESHOLD%"
    exit 0
else
    echo ""
    echo "❌ Coverage ($TOTAL_COVERAGE) is below threshold of $THRESHOLD%"
    exit 1
fi