# Qualhook Test Quality Metrics

This directory contains the test quality metrics system for Qualhook, providing comprehensive insights into test coverage, quality, performance, and reliability.

## Overview

The test metrics system collects and tracks:
- **Test Coverage**: Package-level and overall coverage percentages
- **Test Execution Times**: Performance metrics for all tests
- **Test Flakiness**: Detection of unreliable tests
- **Test Quality Score**: Composite metric based on multiple factors
- **Test Distribution**: Types of tests (unit, integration, e2e, benchmarks)
- **Code-to-Test Ratio**: Balance between source and test code

## Quick Start

### Collect Metrics Locally
```bash
# Run comprehensive test metrics collection
make test-metrics

# Or run the script directly
./scripts/test_metrics.sh
```

### View Dashboard
After running metrics collection, open the generated dashboard:
```bash
open test_metrics/dashboard.html
```

### Monitor Quality
Check for quality degradation:
```bash
./scripts/monitor_test_quality.sh
```

### Visualize Trends
Generate trend visualizations (requires matplotlib):
```bash
pip install matplotlib
./scripts/visualize_test_trends.py test_metrics/trend_data.json
```

## CI Integration

### GitHub Actions Workflows

1. **Automated Collection**: Metrics are automatically collected on:
   - Every push to main branch
   - Every pull request
   - Daily scheduled runs (2 AM UTC)

2. **PR Comments**: Pull requests receive automatic comments with:
   - Quality score
   - Test coverage
   - Flaky test count
   - Pass/fail status for quality gates

3. **Quality Gates**: PRs fail if:
   - Quality score < 60
   - Coverage < 50%
   - Flaky tests > 5

### Viewing CI Metrics

- **Workflow Artifacts**: Download full reports from GitHub Actions
- **GitHub Pages**: View latest dashboard at `https://[your-username].github.io/qualhook/`
- **PR Comments**: See summary directly in pull requests

## Metrics Explained

### Quality Score (0-100)
Composite score calculated from:
- Test coverage (33.3%)
- Test-to-code ratio (33.3%)
- Package test presence (33.3%)
- Flaky test penalty (-5 points per flaky test)

### Coverage Metrics
- **Average Coverage**: Mean coverage across all packages
- **Package Coverage**: Individual package test coverage
- **Coverage Trend**: Historical coverage changes

### Test Performance
- **Execution Time**: Time taken by each test
- **Slowest Tests**: Tests taking >1 second
- **Performance Trends**: Historical execution times

### Test Reliability
- **Flaky Tests**: Tests that pass/fail inconsistently
- **Detection Method**: Multiple test runs with result comparison

## Configuration

### Environment Variables

```bash
# For Slack notifications
export SLACK_WEBHOOK="https://hooks.slack.com/services/..."

# For email alerts
export TEST_ALERTS_EMAIL="team@example.com"
```

### Thresholds

Edit thresholds in `scripts/monitor_test_quality.sh`:
```bash
COVERAGE_THRESHOLD=70      # Minimum coverage %
QUALITY_THRESHOLD=60       # Minimum quality score
FLAKY_THRESHOLD=5         # Maximum flaky tests
COVERAGE_DROP_THRESHOLD=5  # Alert if drops by >5%
QUALITY_DROP_THRESHOLD=10  # Alert if drops by >10
```

## Directory Structure

```
test_metrics/
├── README.md           # This file
├── results/           # Raw metrics data (JSON)
├── coverage/          # Coverage profiles
├── reports/           # Generated reports
├── dashboard.html     # Latest dashboard
├── trend_data.json    # Historical trends
├── alerts.log         # Alert history
└── visualizations/    # Trend charts
```

## Scripts

### test_metrics.sh
Main collection script that:
- Runs tests with coverage
- Analyzes execution times
- Detects flaky tests
- Calculates quality metrics
- Generates HTML dashboard
- Updates trend data

### monitor_test_quality.sh
Monitoring script that:
- Checks current metrics
- Compares with thresholds
- Detects degradation
- Sends alerts (Slack/email)
- Provides recommendations

### generate_test_dashboard.py
Dashboard generator that:
- Creates interactive HTML
- Shows key metrics
- Displays trends
- Lists problem areas

### visualize_test_trends.py
Visualization script that:
- Generates trend charts
- Creates summary reports
- Identifies patterns

## Best Practices

1. **Regular Monitoring**: Run metrics at least daily
2. **Fix Issues Promptly**: Address flaky tests immediately
3. **Maintain Coverage**: Keep coverage above 70%
4. **Review Trends**: Check weekly for degradation
5. **Update Thresholds**: Adjust based on project needs

## Troubleshooting

### No Metrics Found
```bash
# Ensure tests can run
go test ./...

# Check for test files
find . -name "*_test.go" | wc -l
```

### Dashboard Not Opening
```bash
# Check file exists
ls -la test_metrics/dashboard.html

# Open manually
open test_metrics/dashboard.html
```

### Metrics Collection Fails
```bash
# Check dependencies
which jq bc python3

# Install missing tools
brew install jq
apt-get install bc python3
```

## Contributing

To improve the test metrics system:

1. **Add New Metrics**: Update `test_metrics.sh`
2. **Improve Dashboard**: Modify `generate_test_dashboard.py`
3. **Add Visualizations**: Enhance `visualize_test_trends.py`
4. **Update Thresholds**: Adjust in monitoring scripts

## Future Enhancements

- [ ] Integration with more CI platforms
- [ ] Real-time metrics streaming
- [ ] Test impact analysis
- [ ] Mutation testing metrics
- [ ] Performance regression detection
- [ ] Custom metric plugins