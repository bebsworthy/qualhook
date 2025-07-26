// Package benchmarks provides test suite reporting functionality
package benchmarks

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"
)

// Reporter generates test suite health reports
type Reporter struct {
	tracker      *Tracker
	flakeDetector *FlakeDetector
	coverageAnalyzer *CoverageAnalyzer
}

// NewReporter creates a new test suite reporter
func NewReporter(tracker *Tracker, flakeDetector *FlakeDetector, coverageAnalyzer *CoverageAnalyzer) *Reporter {
	return &Reporter{
		tracker:          tracker,
		flakeDetector:    flakeDetector,
		coverageAnalyzer: coverageAnalyzer,
	}
}

// GenerateSummary generates a comprehensive test suite summary
func (r *Reporter) GenerateSummary() TestSuiteSummary {
	stats := r.tracker.GetStats()
	stats.CalculateAverages()
	
	flakinessReport := r.flakeDetector.AnalyzeFlakiness()
	coverageSummary := r.coverageAnalyzer.GetSummaryReport()
	
	return TestSuiteSummary{
		GeneratedAt:      time.Now(),
		TestStats:        stats,
		FlakinessReport:  flakinessReport,
		CoverageSummary:  coverageSummary,
		HealthScore:      r.calculateHealthScore(stats, flakinessReport, coverageSummary),
		Recommendations:  r.generateRecommendations(stats, flakinessReport, coverageSummary),
	}
}

// fprintf is a helper that ignores the error from fmt.Fprintf
func fprintf(w io.Writer, format string, a ...interface{}) {
	_, _ = fmt.Fprintf(w, format, a...) //nolint:errcheck
}

// fprintln is a helper that ignores the error from fmt.Fprintln
func fprintln(w io.Writer, a ...interface{}) {
	_, _ = fmt.Fprintln(w, a...) //nolint:errcheck
}

// WriteTextReport writes a human-readable text report
func (r *Reporter) WriteTextReport(w io.Writer, summary TestSuiteSummary) error {
	fprintf(w, "Test Suite Health Report\n")
	fprintf(w, "Generated: %s\n\n", summary.GeneratedAt.Format("2006-01-02 15:04:05"))
	
	// Health Score
	fprintf(w, "Overall Health Score: %.1f%%\n\n", summary.HealthScore)
	
	// Test Statistics
	fprintf(w, "Test Execution Summary\n")
	fprintf(w, "======================\n")
	fprintf(w, "Total Tests: %d\n", summary.TestStats.TotalTests)
	fprintf(w, "Passed: %d (%.1f%%)\n", summary.TestStats.Passed, 
		float64(summary.TestStats.Passed)/float64(summary.TestStats.TotalTests)*100)
	fprintf(w, "Failed: %d\n", summary.TestStats.GetFailed())
	fprintf(w, "Skipped: %d\n", summary.TestStats.Skipped)
	fprintf(w, "Total Duration: %s\n\n", summary.TestStats.TotalDuration)
	
	// Category Breakdown
	fprintf(w, "Tests by Category\n")
	fprintf(w, "-----------------\n")
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fprintf(tw, "Category\tTotal\tPassed\tFailed\tAvg Duration\n")
	
	for category, stats := range summary.TestStats.ByCategory {
		failed := stats.Total - stats.Passed - stats.Skipped
		fprintf(tw, "%s\t%d\t%d\t%d\t%s\n", 
			category, stats.Total, stats.Passed, failed, stats.AvgDuration)
	}
	_ = tw.Flush() //nolint:errcheck
	fprintln(w)
	
	// Slowest Tests
	fprintf(w, "Slowest Tests (Top 5)\n")
	fprintf(w, "---------------------\n")
	for i, test := range summary.TestStats.SlowestTests {
		if i >= 5 {
			break
		}
		fprintf(w, "%d. %s.%s - %s\n", i+1, test.Package, test.Test, test.Duration)
	}
	fprintln(w)
	
	// Flakiness Report
	fprintf(w, "Flakiness Analysis\n")
	fprintf(w, "==================\n")
	fprintf(w, "Total Flaky Tests: %d\n", summary.FlakinessReport.TotalFlakyTests)
	
	if len(summary.FlakinessReport.FlakyTests) > 0 {
		fprintf(w, "\nFlaky Tests:\n")
		for _, flaky := range summary.FlakinessReport.FlakyTests {
			fprintf(w, "- %s.%s (Flake Score: %.2f, Failure Rate: %.1f%%)\n",
				flaky.Package, flaky.TestName, flaky.FlakeScore, flaky.FailureRate*100)
		}
	}
	fprintln(w)
	
	// Coverage Summary
	fprintf(w, "Coverage Summary\n")
	fprintf(w, "================\n")
	fprintf(w, "Overall Coverage: %.1f%%\n\n", summary.CoverageSummary.OverallCoverage)
	
	fprintf(w, "Coverage by Category:\n")
	for category, catSummary := range summary.CoverageSummary.Categories {
		fprintf(w, "- %s: %.1f%% (%d/%d lines)\n", 
			category, catSummary.Percentage, catSummary.CoveredLines, catSummary.TotalLines)
	}
	fprintln(w)
	
	// Recommendations
	if len(summary.Recommendations) > 0 {
		fprintf(w, "Recommendations\n")
		fprintf(w, "===============\n")
		for i, rec := range summary.Recommendations {
			fprintf(w, "%d. %s\n", i+1, rec)
		}
	}
	
	return nil
}

// WriteJSONReport writes a JSON report
func (r *Reporter) WriteJSONReport(filename string, summary TestSuiteSummary) error {
	data, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal summary: %w", err)
	}
	
	if err := os.MkdirAll(filepath.Dir(filename), 0750); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	
	if err := os.WriteFile(filename, data, 0600); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}
	
	return nil
}

// WriteHTMLReport writes an HTML report
func (r *Reporter) WriteHTMLReport(filename string, summary TestSuiteSummary) error {
	html := r.generateHTMLReport(summary)
	
	if err := os.MkdirAll(filepath.Dir(filename), 0750); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	
	if err := os.WriteFile(filename, []byte(html), 0600); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}
	
	return nil
}

// calculateHealthScore calculates an overall health score for the test suite
func (r *Reporter) calculateHealthScore(stats TestStats, flakiness FlakinessReport, coverage CoverageSummary) float64 {
	score := 100.0
	
	// Deduct points for test failures
	if stats.TotalTests > 0 {
		failureRate := float64(stats.GetFailed()) / float64(stats.TotalTests)
		score -= failureRate * 20
	}
	
	// Deduct points for flaky tests
	if flakiness.TotalTests > 0 {
		flakyRate := float64(flakiness.TotalFlakyTests) / float64(flakiness.TotalTests)
		score -= flakyRate * 15
	}
	
	// Deduct points for low coverage
	if coverage.OverallCoverage < 80 {
		score -= (80 - coverage.OverallCoverage) * 0.5
	}
	
	// Ensure score is between 0 and 100
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}
	
	return score
}

// generateRecommendations generates actionable recommendations
func (r *Reporter) generateRecommendations(stats TestStats, flakiness FlakinessReport, coverage CoverageSummary) []string {
	recommendations := make([]string, 0)
	
	// Check for test failures
	failureRate := float64(stats.GetFailed()) / float64(stats.TotalTests)
	if failureRate > 0.05 {
		recommendations = append(recommendations, 
			fmt.Sprintf("High test failure rate (%.1f%%). Investigate and fix failing tests.", failureRate*100))
	}
	
	// Check for flaky tests
	if flakiness.TotalFlakyTests > 0 {
		recommendations = append(recommendations,
			fmt.Sprintf("Found %d flaky tests. Consider adding retry logic or investigating root causes.", 
				flakiness.TotalFlakyTests))
	}
	
	// Check coverage
	if coverage.OverallCoverage < 80 {
		recommendations = append(recommendations,
			fmt.Sprintf("Test coverage is below 80%% (%.1f%%). Add more tests to improve coverage.", 
				coverage.OverallCoverage))
	}
	
	// Check for slow tests
	if len(stats.SlowestTests) > 0 && stats.SlowestTests[0].Duration > 5*time.Second {
		recommendations = append(recommendations,
			"Some tests are taking more than 5 seconds. Consider optimizing or moving to integration tests.")
	}
	
	// Check category balance
	for category, catStats := range stats.ByCategory {
		if catStats.Total < 10 {
			recommendations = append(recommendations,
				fmt.Sprintf("Low number of %s tests (%d). Consider adding more tests in this category.", 
					category, catStats.Total))
		}
	}
	
	return recommendations
}

// generateHTMLReport generates an HTML report
func (r *Reporter) generateHTMLReport(summary TestSuiteSummary) string {
	var sb strings.Builder
	
	sb.WriteString(`<!DOCTYPE html>
<html>
<head>
    <title>Test Suite Health Report</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        h1, h2 { color: #333; }
        .metric { display: inline-block; margin: 10px; padding: 10px; background: #f0f0f0; border-radius: 5px; }
        .metric-value { font-size: 24px; font-weight: bold; }
        .metric-label { font-size: 14px; color: #666; }
        table { border-collapse: collapse; width: 100%; margin: 20px 0; }
        th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
        th { background-color: #f2f2f2; }
        .health-score { 
            font-size: 48px; 
            font-weight: bold; 
            padding: 20px; 
            text-align: center;
            border-radius: 10px;
        }
        .health-good { background: #4CAF50; color: white; }
        .health-warning { background: #FF9800; color: white; }
        .health-bad { background: #F44336; color: white; }
        .recommendation { 
            background: #FFF3CD; 
            border: 1px solid #FFEAA7; 
            padding: 10px; 
            margin: 5px 0;
            border-radius: 5px;
        }
    </style>
</head>
<body>
`)
	
	// Header
	sb.WriteString("<h1>Test Suite Health Report</h1>\n")
	sb.WriteString(fmt.Sprintf("<p>Generated: %s</p>\n", summary.GeneratedAt.Format("2006-01-02 15:04:05")))
	
	// Health Score
	healthClass := "health-good"
	if summary.HealthScore < 70 {
		healthClass = "health-bad"
	} else if summary.HealthScore < 85 {
		healthClass = "health-warning"
	}
	
	sb.WriteString(fmt.Sprintf(`<div class="health-score %s">Health Score: %.1f%%</div>`, 
		healthClass, summary.HealthScore))
	
	// Key Metrics
	sb.WriteString("<h2>Key Metrics</h2>\n")
	sb.WriteString("<div>\n")
	sb.WriteString(fmt.Sprintf(`<div class="metric"><div class="metric-value">%d</div><div class="metric-label">Total Tests</div></div>`,
		summary.TestStats.TotalTests))
	sb.WriteString(fmt.Sprintf(`<div class="metric"><div class="metric-value">%.1f%%</div><div class="metric-label">Pass Rate</div></div>`,
		float64(summary.TestStats.Passed)/float64(summary.TestStats.TotalTests)*100))
	sb.WriteString(fmt.Sprintf(`<div class="metric"><div class="metric-value">%.1f%%</div><div class="metric-label">Coverage</div></div>`,
		summary.CoverageSummary.OverallCoverage))
	sb.WriteString(fmt.Sprintf(`<div class="metric"><div class="metric-value">%d</div><div class="metric-label">Flaky Tests</div></div>`,
		summary.FlakinessReport.TotalFlakyTests))
	sb.WriteString("</div>\n")
	
	// Test Results by Category
	sb.WriteString("<h2>Test Results by Category</h2>\n")
	sb.WriteString("<table>\n")
	sb.WriteString("<tr><th>Category</th><th>Total</th><th>Passed</th><th>Failed</th><th>Coverage</th></tr>\n")
	
	for category, stats := range summary.TestStats.ByCategory {
		failed := stats.Total - stats.Passed - stats.Skipped
		coverage := summary.CoverageSummary.Categories[category].Percentage
		sb.WriteString(fmt.Sprintf("<tr><td>%s</td><td>%d</td><td>%d</td><td>%d</td><td>%.1f%%</td></tr>\n",
			category, stats.Total, stats.Passed, failed, coverage))
	}
	sb.WriteString("</table>\n")
	
	// Recommendations
	if len(summary.Recommendations) > 0 {
		sb.WriteString("<h2>Recommendations</h2>\n")
		for _, rec := range summary.Recommendations {
			sb.WriteString(fmt.Sprintf(`<div class="recommendation">%s</div>`, rec))
		}
	}
	
	sb.WriteString("</body>\n</html>")
	
	return sb.String()
}

// TestSuiteSummary represents a comprehensive test suite summary
type TestSuiteSummary struct {
	GeneratedAt      time.Time          `json:"generated_at"`
	TestStats        TestStats          `json:"test_stats"`
	FlakinessReport  FlakinessReport    `json:"flakiness_report"`
	CoverageSummary  CoverageSummary    `json:"coverage_summary"`
	HealthScore      float64            `json:"health_score"`
	Recommendations  []string           `json:"recommendations"`
}