// Package benchmarks provides test coverage analysis by category
package benchmarks

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// CoverageAnalyzer analyzes test coverage by category
type CoverageAnalyzer struct {
	profiles map[string]*CoverageProfile // key: category (unit, integration, e2e)
}

// CoverageProfile represents coverage data for a test category
type CoverageProfile struct {
	Category     string                     `json:"category"`
	TotalLines   int                        `json:"total_lines"`
	CoveredLines int                        `json:"covered_lines"`
	Percentage   float64                    `json:"percentage"`
	Packages     map[string]PackageCoverage `json:"packages"`
}

// PackageCoverage represents coverage for a single package
type PackageCoverage struct {
	Package      string         `json:"package"`
	TotalLines   int            `json:"total_lines"`
	CoveredLines int            `json:"covered_lines"`
	Percentage   float64        `json:"percentage"`
	Files        []FileCoverage `json:"files"`
}

// FileCoverage represents coverage for a single file
type FileCoverage struct {
	Filename       string  `json:"filename"`
	TotalLines     int     `json:"total_lines"`
	CoveredLines   int     `json:"covered_lines"`
	Percentage     float64 `json:"percentage"`
	UncoveredLines []int   `json:"uncovered_lines,omitempty"`
}

// NewCoverageAnalyzer creates a new coverage analyzer
func NewCoverageAnalyzer() *CoverageAnalyzer {
	return &CoverageAnalyzer{
		profiles: make(map[string]*CoverageProfile),
	}
}

// ParseCoverageFile parses a Go coverage profile file
//
//nolint:gocyclo // Complex function parsing various coverage profile formats
func (ca *CoverageAnalyzer) ParseCoverageFile(filename, category string) error {
	file, err := os.Open(filename) // #nosec G304 - filename comes from coverage profile
	if err != nil {
		return fmt.Errorf("failed to open coverage file: %w", err)
	}
	defer func() { _ = file.Close() }() //nolint:errcheck

	if _, exists := ca.profiles[category]; !exists {
		ca.profiles[category] = &CoverageProfile{
			Category: category,
			Packages: make(map[string]PackageCoverage),
		}
	}

	profile := ca.profiles[category]
	scanner := bufio.NewScanner(file)

	// Skip the mode line
	if scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "mode:") {
			return fmt.Errorf("invalid coverage file format")
		}
	}

	// Parse coverage data
	// Format: github.com/user/pkg/file.go:startLine.startCol,endLine.endCol numStmt count
	lineRegex := regexp.MustCompile(`^(.+):(\d+)\.(\d+),(\d+)\.(\d+)\s+(\d+)\s+(\d+)$`)

	fileData := make(map[string]map[int]bool) // file -> line -> covered

	for scanner.Scan() {
		line := scanner.Text()
		matches := lineRegex.FindStringSubmatch(line)
		if matches == nil {
			continue
		}

		file := matches[1]
		startLine, err := strconv.Atoi(matches[2])
		if err != nil {
			continue
		}
		endLine, err := strconv.Atoi(matches[4])
		if err != nil {
			continue
		}
		count, err := strconv.Atoi(matches[7])
		if err != nil {
			continue
		}

		if _, exists := fileData[file]; !exists {
			fileData[file] = make(map[int]bool)
		}

		// Mark lines as covered or not
		for line := startLine; line <= endLine; line++ {
			if count > 0 {
				fileData[file][line] = true
			} else if _, exists := fileData[file][line]; !exists {
				fileData[file][line] = false
			}
		}
	}

	// Process the collected data
	for file, lines := range fileData {
		pkg := getPackageFromFile(file)

		if _, exists := profile.Packages[pkg]; !exists {
			profile.Packages[pkg] = PackageCoverage{
				Package: pkg,
				Files:   make([]FileCoverage, 0),
			}
		}

		fileCov := FileCoverage{
			Filename:       file,
			TotalLines:     len(lines),
			CoveredLines:   0,
			UncoveredLines: make([]int, 0),
		}

		for line, covered := range lines {
			if covered {
				fileCov.CoveredLines++
			} else {
				fileCov.UncoveredLines = append(fileCov.UncoveredLines, line)
			}
		}

		sort.Ints(fileCov.UncoveredLines)

		if fileCov.TotalLines > 0 {
			fileCov.Percentage = float64(fileCov.CoveredLines) / float64(fileCov.TotalLines) * 100
		}

		pkgCov := profile.Packages[pkg]
		pkgCov.Files = append(pkgCov.Files, fileCov)
		pkgCov.TotalLines += fileCov.TotalLines
		pkgCov.CoveredLines += fileCov.CoveredLines
		profile.Packages[pkg] = pkgCov
	}

	// Calculate package percentages
	for pkg, pkgCov := range profile.Packages {
		if pkgCov.TotalLines > 0 {
			pkgCov.Percentage = float64(pkgCov.CoveredLines) / float64(pkgCov.TotalLines) * 100
			profile.Packages[pkg] = pkgCov
		}

		profile.TotalLines += pkgCov.TotalLines
		profile.CoveredLines += pkgCov.CoveredLines
	}

	// Calculate overall percentage
	if profile.TotalLines > 0 {
		profile.Percentage = float64(profile.CoveredLines) / float64(profile.TotalLines) * 100
	}

	return scanner.Err()
}

// GetCategoryReport returns coverage report for a specific category
func (ca *CoverageAnalyzer) GetCategoryReport(category string) (*CoverageProfile, bool) {
	profile, exists := ca.profiles[category]
	return profile, exists
}

// GetSummaryReport returns a summary of coverage across all categories
func (ca *CoverageAnalyzer) GetSummaryReport() CoverageSummary {
	summary := CoverageSummary{
		Categories:       make(map[string]CategorySummary),
		TotalLines:       0,
		TotalCovered:     0,
		OverallCoverage:  0,
		PackageSummaries: make(map[string]PackageSummary),
	}

	// Aggregate by category
	for category, profile := range ca.profiles {
		summary.Categories[category] = CategorySummary{
			Category:     category,
			TotalLines:   profile.TotalLines,
			CoveredLines: profile.CoveredLines,
			Percentage:   profile.Percentage,
			PackageCount: len(profile.Packages),
		}

		summary.TotalLines += profile.TotalLines
		summary.TotalCovered += profile.CoveredLines

		// Aggregate by package across categories
		for pkg, pkgCov := range profile.Packages {
			if _, exists := summary.PackageSummaries[pkg]; !exists {
				summary.PackageSummaries[pkg] = PackageSummary{
					Package:            pkg,
					CoverageByCategory: make(map[string]float64),
				}
			}

			pkgSummary := summary.PackageSummaries[pkg]
			pkgSummary.CoverageByCategory[category] = pkgCov.Percentage
			summary.PackageSummaries[pkg] = pkgSummary
		}
	}

	// Calculate overall coverage
	if summary.TotalLines > 0 {
		summary.OverallCoverage = float64(summary.TotalCovered) / float64(summary.TotalLines) * 100
	}

	return summary
}

// FindUncoveredCode identifies code that lacks test coverage
func (ca *CoverageAnalyzer) FindUncoveredCode() []UncoveredCode {
	uncovered := make([]UncoveredCode, 0)

	for category, profile := range ca.profiles {
		for _, pkgCov := range profile.Packages {
			for _, fileCov := range pkgCov.Files {
				if len(fileCov.UncoveredLines) > 0 {
					uncovered = append(uncovered, UncoveredCode{
						Category:       category,
						Package:        pkgCov.Package,
						File:           fileCov.Filename,
						UncoveredLines: fileCov.UncoveredLines,
						Percentage:     fileCov.Percentage,
					})
				}
			}
		}
	}

	// Sort by coverage percentage (lowest first)
	sort.Slice(uncovered, func(i, j int) bool {
		return uncovered[i].Percentage < uncovered[j].Percentage
	})

	return uncovered
}

// CoverageSummary provides an overall coverage summary
type CoverageSummary struct {
	Categories       map[string]CategorySummary `json:"categories"`
	PackageSummaries map[string]PackageSummary  `json:"packages"`
	TotalLines       int                        `json:"total_lines"`
	TotalCovered     int                        `json:"total_covered"`
	OverallCoverage  float64                    `json:"overall_coverage"`
}

// CategorySummary provides coverage summary for a test category
type CategorySummary struct {
	Category     string  `json:"category"`
	TotalLines   int     `json:"total_lines"`
	CoveredLines int     `json:"covered_lines"`
	Percentage   float64 `json:"percentage"`
	PackageCount int     `json:"package_count"`
}

// PackageSummary provides coverage summary for a package across categories
type PackageSummary struct {
	Package            string             `json:"package"`
	CoverageByCategory map[string]float64 `json:"coverage_by_category"`
}

// UncoveredCode represents code without test coverage
type UncoveredCode struct {
	Category       string  `json:"category"`
	Package        string  `json:"package"`
	File           string  `json:"file"`
	UncoveredLines []int   `json:"uncovered_lines"`
	Percentage     float64 `json:"percentage"`
}

// Helper function to extract package name from file path
func getPackageFromFile(file string) string {
	// Convert file path to package path
	// e.g., github.com/bebsworthy/qualhook/internal/executor/command.go -> github.com/bebsworthy/qualhook/internal/executor
	dir := filepath.Dir(file)
	return strings.ReplaceAll(dir, string(filepath.Separator), "/")
}

// GenerateCoverageScript generates a script to collect coverage by category
func GenerateCoverageScript(outputDir string) error {
	script := `#!/bin/bash
# Coverage collection script for test categories

set -e

OUTPUT_DIR="${1:-test/benchmarks/coverage}"
mkdir -p "$OUTPUT_DIR"

echo "Collecting test coverage by category..."

# Unit tests
echo "Running unit tests with coverage..."
go test -tags=unit -coverprofile="$OUTPUT_DIR/unit.cover" ./... > /dev/null 2>&1 || true

# Integration tests
echo "Running integration tests with coverage..."
go test -tags=integration -coverprofile="$OUTPUT_DIR/integration.cover" ./... > /dev/null 2>&1 || true

# E2E tests
echo "Running e2e tests with coverage..."
go test -tags=e2e -coverprofile="$OUTPUT_DIR/e2e.cover" ./... > /dev/null 2>&1 || true

echo "Coverage collection complete. Files saved to $OUTPUT_DIR"

# Generate HTML reports
for category in unit integration e2e; do
    if [ -f "$OUTPUT_DIR/$category.cover" ]; then
        go tool cover -html="$OUTPUT_DIR/$category.cover" -o "$OUTPUT_DIR/$category.html" 2>/dev/null || true
        echo "Generated HTML report: $OUTPUT_DIR/$category.html"
    fi
done
`

	scriptPath := filepath.Join(outputDir, "collect_coverage.sh")
	if err := os.WriteFile(scriptPath, []byte(script), 0600); err != nil {
		return fmt.Errorf("failed to write coverage script: %w", err)
	}

	return nil
}
