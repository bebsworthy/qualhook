// Command benchmark provides test benchmarking and analysis tools
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/bebsworthy/qualhook/test/benchmarks"
)

func main() {
	var (
		action          = flag.String("action", "report", "Action to perform: report, track, analyze-flakiness, coverage")
		outputDir       = flag.String("output", "test/benchmarks/reports", "Output directory for reports")
		coverageDir     = flag.String("coverage-dir", "test/benchmarks/coverage", "Directory containing coverage files")
		trackingFile    = flag.String("tracking-file", "", "Test tracking data file")
		flakeThreshold  = flag.Float64("flake-threshold", 0.1, "Flakiness threshold (0.0-1.0)")
		format          = flag.String("format", "text", "Output format: text, json, html")
	)
	
	flag.Parse()
	
	switch *action {
	case "report":
		if err := generateReport(*outputDir, *coverageDir, *trackingFile, *format); err != nil {
			log.Fatal(err)
		}
		
	case "track":
		fmt.Println("Test tracking should be integrated into your test runner")
		fmt.Println("See the documentation for examples")
		
	case "analyze-flakiness":
		if err := analyzeFlakinessOnly(*trackingFile, *flakeThreshold); err != nil {
			log.Fatal(err)
		}
		
	case "coverage":
		if err := generateCoverageScript(*outputDir); err != nil {
			log.Fatal(err)
		}
		
	default:
		log.Fatalf("Unknown action: %s", *action)
	}
}

func generateReport(outputDir, coverageDir, trackingFile, format string) error {
	// Create components
	tracker := benchmarks.NewTracker()
	flakeDetector := benchmarks.NewFlakeDetector(0.1)
	coverageAnalyzer := benchmarks.NewCoverageAnalyzer()
	
	// Load tracking data if provided
	if trackingFile != "" {
		if err := tracker.LoadResults(trackingFile); err != nil {
			log.Printf("Warning: Could not load tracking file: %v", err)
		}
	}
	
	// Parse coverage files
	categories := []string{"unit", "integration", "e2e"}
	for _, category := range categories {
		coverFile := filepath.Join(coverageDir, category+".cover")
		if _, err := os.Stat(coverFile); err == nil {
			if err := coverageAnalyzer.ParseCoverageFile(coverFile, category); err != nil {
				log.Printf("Warning: Could not parse %s coverage: %v", category, err)
			}
		}
	}
	
	// Generate report
	reporter := benchmarks.NewReporter(tracker, flakeDetector, coverageAnalyzer)
	summary := reporter.GenerateSummary()
	
	// Output report
	if err := os.MkdirAll(outputDir, 0750); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}
	
	switch format {
	case "text":
		return reporter.WriteTextReport(os.Stdout, summary)
		
	case "json":
		filename := filepath.Join(outputDir, "test-suite-report.json")
		if err := reporter.WriteJSONReport(filename, summary); err != nil {
			return err
		}
		fmt.Printf("JSON report written to: %s\n", filename)
		
	case "html":
		filename := filepath.Join(outputDir, "test-suite-report.html")
		if err := reporter.WriteHTMLReport(filename, summary); err != nil {
			return err
		}
		fmt.Printf("HTML report written to: %s\n", filename)
		
	default:
		return fmt.Errorf("unknown format: %s", format)
	}
	
	return nil
}

func analyzeFlakinessOnly(trackingFile string, threshold float64) error {
	if trackingFile == "" {
		return fmt.Errorf("tracking file required for flakiness analysis")
	}
	
	tracker := benchmarks.NewTracker()
	if err := tracker.LoadResults(trackingFile); err != nil {
		return fmt.Errorf("failed to load tracking file: %w", err)
	}
	
	flakeDetector := benchmarks.NewFlakeDetector(threshold)
	
	// TODO: Populate flake detector from tracker data
	// This would require extending the tracker to record individual test runs
	
	report := flakeDetector.AnalyzeFlakiness()
	
	fmt.Printf("Flakiness Analysis Report\n")
	fmt.Printf("========================\n")
	fmt.Printf("Total Tests: %d\n", report.TotalTests)
	fmt.Printf("Flaky Tests: %d\n", report.TotalFlakyTests)
	fmt.Printf("Threshold: %.1f%%\n\n", report.ThresholdPercent)
	
	if len(report.FlakyTests) > 0 {
		fmt.Println("Flaky Tests:")
		for _, test := range report.FlakyTests {
			fmt.Printf("- %s.%s\n", test.Package, test.TestName)
			fmt.Printf("  Flake Score: %.2f\n", test.FlakeScore)
			fmt.Printf("  Failure Rate: %.1f%% (%d/%d)\n", 
				test.FailureRate*100, test.Failures, test.TotalRuns)
			if len(test.FailureReasons) > 0 {
				fmt.Println("  Failure Reasons:")
				for reason, count := range test.FailureReasons {
					fmt.Printf("    - %s (%d times)\n", reason, count)
				}
			}
			fmt.Println()
		}
		
		// Suggest retry strategies
		strategies := flakeDetector.SuggestRetries()
		if len(strategies) > 0 {
			fmt.Println("Suggested Retry Strategies:")
			for _, strategy := range strategies {
				fmt.Printf("- %s.%s: %d retries", strategy.Package, strategy.TestName, strategy.MaxRetries)
				if strategy.TimeoutMultiplier > 1 {
					fmt.Printf(", %.1fx timeout", strategy.TimeoutMultiplier)
				}
				fmt.Println()
				for _, note := range strategy.Notes {
					fmt.Printf("  Note: %s\n", note)
				}
			}
		}
	}
	
	return nil
}

func generateCoverageScript(outputDir string) error {
	if err := benchmarks.GenerateCoverageScript(outputDir); err != nil {
		return err
	}
	
	scriptPath := filepath.Join(outputDir, "collect_coverage.sh")
	fmt.Printf("Coverage collection script generated: %s\n", scriptPath)
	fmt.Println("\nUsage:")
	fmt.Printf("  chmod +x %s\n", scriptPath)
	fmt.Printf("  ./%s [output-dir]\n", scriptPath)
	fmt.Println("\nThis will run tests with coverage for each category and generate reports.")
	
	return nil
}