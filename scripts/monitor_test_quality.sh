#!/bin/bash
# Monitor test quality metrics and alert on degradation

set -e

# Configuration
METRICS_DIR="test_metrics"
ALERTS_FILE="$METRICS_DIR/alerts.log"
SLACK_WEBHOOK="${SLACK_WEBHOOK:-}"  # Set via environment variable
EMAIL_TO="${TEST_ALERTS_EMAIL:-}"    # Set via environment variable

# Colors
RED='\033[0;31m'
YELLOW='\033[1;33m'
GREEN='\033[0;32m'
NC='\033[0m'

# Thresholds
COVERAGE_THRESHOLD=70
QUALITY_THRESHOLD=60
FLAKY_THRESHOLD=5
COVERAGE_DROP_THRESHOLD=5  # Alert if coverage drops by more than 5%
QUALITY_DROP_THRESHOLD=10  # Alert if quality drops by more than 10 points

# Create alerts file if it doesn't exist
mkdir -p "$METRICS_DIR"
touch "$ALERTS_FILE"

echo "=== Test Quality Monitor ==="
echo "Checking test quality metrics..."

# Get latest metrics
LATEST_METRICS=$(ls "$METRICS_DIR"/results/metrics_*.json 2>/dev/null | tail -1)
if [ -z "$LATEST_METRICS" ]; then
    echo "No metrics found. Run './scripts/test_metrics.sh' first."
    exit 1
fi

# Get trend data
TREND_FILE="$METRICS_DIR/trend_data.json"
if [ ! -f "$TREND_FILE" ]; then
    echo "No trend data found."
    exit 1
fi

# Extract current metrics
CURRENT_COVERAGE=$(jq '.metrics.coverage.average' "$LATEST_METRICS")
CURRENT_QUALITY=$(jq '.metrics.quality_score.overall' "$LATEST_METRICS")
CURRENT_FLAKY=$(jq '.metrics.flakiness.flaky_test_count' "$LATEST_METRICS")

# Get previous metrics (from 1 day ago if available)
PREVIOUS_DATA=$(jq '.[-2] // .[-1]' "$TREND_FILE")
PREVIOUS_COVERAGE=$(echo "$PREVIOUS_DATA" | jq '.coverage // 0')
PREVIOUS_QUALITY=$(echo "$PREVIOUS_DATA" | jq '.quality_score // 0')

# Calculate changes
COVERAGE_CHANGE=$(echo "$CURRENT_COVERAGE - $PREVIOUS_COVERAGE" | bc)
QUALITY_CHANGE=$(echo "$CURRENT_QUALITY - $PREVIOUS_QUALITY" | bc)

# Initialize alerts array
ALERTS=()

# Check absolute thresholds
if (( $(echo "$CURRENT_COVERAGE < $COVERAGE_THRESHOLD" | bc -l) )); then
    ALERTS+=("CRITICAL: Test coverage ($CURRENT_COVERAGE%) is below threshold ($COVERAGE_THRESHOLD%)")
fi

if (( $(echo "$CURRENT_QUALITY < $QUALITY_THRESHOLD" | bc -l) )); then
    ALERTS+=("CRITICAL: Quality score ($CURRENT_QUALITY) is below threshold ($QUALITY_THRESHOLD)")
fi

if [ "$CURRENT_FLAKY" -gt "$FLAKY_THRESHOLD" ]; then
    ALERTS+=("WARNING: Too many flaky tests detected ($CURRENT_FLAKY > $FLAKY_THRESHOLD)")
fi

# Check for degradation
if (( $(echo "$COVERAGE_CHANGE < -$COVERAGE_DROP_THRESHOLD" | bc -l) )); then
    ALERTS+=("WARNING: Test coverage dropped by ${COVERAGE_CHANGE#-}% (from $PREVIOUS_COVERAGE% to $CURRENT_COVERAGE%)")
fi

if (( $(echo "$QUALITY_CHANGE < -$QUALITY_DROP_THRESHOLD" | bc -l) )); then
    ALERTS+=("WARNING: Quality score dropped by ${QUALITY_CHANGE#-} points (from $PREVIOUS_QUALITY to $CURRENT_QUALITY)")
fi

# Display current status
echo ""
echo "Current Metrics:"
echo "  Coverage: $CURRENT_COVERAGE% (change: $COVERAGE_CHANGE%)"
echo "  Quality Score: $CURRENT_QUALITY (change: $QUALITY_CHANGE)"
echo "  Flaky Tests: $CURRENT_FLAKY"
echo ""

# Process alerts
if [ ${#ALERTS[@]} -eq 0 ]; then
    echo -e "${GREEN}✅ All metrics are healthy!${NC}"
else
    echo -e "${RED}⚠️  ${#ALERTS[@]} alerts detected:${NC}"
    
    # Log alerts
    TIMESTAMP=$(date -u +%Y-%m-%dT%H:%M:%SZ)
    for alert in "${ALERTS[@]}"; do
        echo -e "${YELLOW}  - $alert${NC}"
        echo "[$TIMESTAMP] $alert" >> "$ALERTS_FILE"
    done
    
    # Send notifications
    send_notifications
fi

# Function to send notifications
send_notifications() {
    # Prepare alert message
    ALERT_MESSAGE="Test Quality Alert\n\n"
    for alert in "${ALERTS[@]}"; do
        ALERT_MESSAGE+="- $alert\n"
    done
    ALERT_MESSAGE+="\nView dashboard: file://$PWD/test_metrics/dashboard.html"
    
    # Send Slack notification if webhook is configured
    if [ -n "$SLACK_WEBHOOK" ]; then
        echo ""
        echo "Sending Slack notification..."
        
        SLACK_PAYLOAD=$(jq -n \
            --arg text "$ALERT_MESSAGE" \
            --arg color "danger" \
            '{
                "attachments": [{
                    "color": $color,
                    "title": "Qualhook Test Quality Alert",
                    "text": $text,
                    "fields": [
                        {"title": "Coverage", "value": "'$CURRENT_COVERAGE'%", "short": true},
                        {"title": "Quality Score", "value": "'$CURRENT_QUALITY'/100", "short": true},
                        {"title": "Flaky Tests", "value": "'$CURRENT_FLAKY'", "short": true}
                    ],
                    "footer": "Test Quality Monitor",
                    "ts": '$(date +%s)'
                }]
            }')
        
        curl -X POST -H 'Content-type: application/json' \
            --data "$SLACK_PAYLOAD" \
            "$SLACK_WEBHOOK" 2>/dev/null || echo "Failed to send Slack notification"
    fi
    
    # Send email notification if configured
    if [ -n "$EMAIL_TO" ]; then
        echo ""
        echo "Sending email notification to $EMAIL_TO..."
        
        echo -e "$ALERT_MESSAGE" | mail -s "Qualhook Test Quality Alert" "$EMAIL_TO" 2>/dev/null || \
            echo "Failed to send email notification"
    fi
}

# Generate recommendations
echo ""
echo "Recommendations:"

if (( $(echo "$CURRENT_COVERAGE < 80" | bc -l) )); then
    echo "  📈 Increase test coverage:"
    
    # Find packages with lowest coverage
    LOW_COVERAGE_PACKAGES=$(jq -r '.metrics.coverage.by_package[] | select(.coverage < 50) | .package' "$LATEST_METRICS" | head -5)
    if [ -n "$LOW_COVERAGE_PACKAGES" ]; then
        echo "     Focus on these packages:"
        echo "$LOW_COVERAGE_PACKAGES" | while read pkg; do
            echo "       - $pkg"
        done
    fi
fi

if [ "$CURRENT_FLAKY" -gt 0 ]; then
    echo "  🔧 Fix flaky tests:"
    
    # List flaky tests
    FLAKY_TESTS=$(jq -r '.metrics.flakiness.flaky_tests[] | "     - \(.test) in \(.package)"' "$LATEST_METRICS" | head -5)
    if [ -n "$FLAKY_TESTS" ]; then
        echo "$FLAKY_TESTS"
    fi
fi

# Check for slow tests
SLOW_TESTS=$(jq -r '.metrics.execution_time.slowest_tests[] | select(.duration > 1) | "     - \(.name) (\(.duration)s)"' "$LATEST_METRICS" | head -3)
if [ -n "$SLOW_TESTS" ]; then
    echo "  ⚡ Optimize slow tests:"
    echo "$SLOW_TESTS"
fi

echo ""

# Exit with error if alerts were found
if [ ${#ALERTS[@]} -gt 0 ]; then
    exit 1
fi