//go:build performance || regression

// Package performance provides performance regression tests for qualhook.
// These tests establish baselines and detect performance degradation.
package performance

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/bebsworthy/qualhook/internal/config"
	"github.com/bebsworthy/qualhook/internal/detector"
	"github.com/bebsworthy/qualhook/internal/executor"
	"github.com/bebsworthy/qualhook/internal/filter"
	pkgconfig "github.com/bebsworthy/qualhook/pkg/config"
	"github.com/spf13/cobra"
)

// Performance baselines - these values should be tuned based on your hardware
// and acceptable performance thresholds
var performanceBaselines = struct {
	// Startup time baselines (in milliseconds)
	StartupHelp    float64
	StartupVersion float64
	StartupCommand float64

	// Pattern matching baselines (ops/second)
	PatternSimpleSmall  int64
	PatternSimpleLarge  int64
	PatternComplexSmall int64
	PatternComplexLarge int64
	PatternSetSmall     int64
	PatternSetLarge     int64

	// Memory usage baselines (bytes per operation)
	MemorySmallCommand   int64
	MemoryLargeCommand   int64
	MemoryConcurrent     int64
	MemoryPatternCompile int64

	// Concurrent execution baselines
	ConcurrentThroughput int64 // ops/second
	ConcurrentMaxMemory  int64 // max memory in bytes
}{
	// Startup baselines (milliseconds)
	StartupHelp:    50,  // 50ms max for help command
	StartupVersion: 30,  // 30ms max for version command
	StartupCommand: 100, // 100ms max for actual command startup

	// Pattern matching baselines (minimum ops/second)
	PatternSimpleSmall:  1000000, // 1M ops/sec for simple patterns on small input
	PatternSimpleLarge:  10000,   // 10K ops/sec for simple patterns on large input
	PatternComplexSmall: 500000,  // 500K ops/sec for complex patterns on small input
	PatternComplexLarge: 5000,    // 5K ops/sec for complex patterns on large input
	PatternSetSmall:     200000,  // 200K ops/sec for pattern sets on small input
	PatternSetLarge:     2000,    // 2K ops/sec for pattern sets on large input

	// Memory baselines (max bytes per operation)
	MemorySmallCommand:   1024,     // 1KB per small command
	MemoryLargeCommand:   1048576,  // 1MB per large command output
	MemoryConcurrent:     10485760, // 10MB for concurrent execution
	MemoryPatternCompile: 10240,    // 10KB per pattern compilation

	// Concurrent execution baselines
	ConcurrentThroughput: 100,       // 100 ops/sec minimum
	ConcurrentMaxMemory:  104857600, // 100MB max memory for concurrent ops
}

// Test data for pattern matching
var (
	testSmallInput  = "2024-01-15 10:30:45 ERROR: Failed to connect to database"
	testMediumInput = strings.Repeat("2024-01-15 10:30:45 INFO: Processing request\n", 100)
	testLargeInput  = strings.Repeat("2024-01-15 10:30:45 DEBUG: Verbose logging output with lots of details\n", 10000)
	testHugeInput   = strings.Repeat("2024-01-15 10:30:45 TRACE: Extremely verbose trace logging with extensive details and metadata\n", 100000)

	testPatterns = []*pkgconfig.RegexPattern{
		{Pattern: `error`, Flags: "i"},
		{Pattern: `warning`, Flags: "i"},
		{Pattern: `\d{4}-\d{2}-\d{2}\s+\d{2}:\d{2}:\d{2}`, Flags: ""},
		{Pattern: `(ERROR|WARN|INFO|DEBUG):`, Flags: ""},
		{Pattern: `Failed to \w+ to \w+`, Flags: "i"},
		{Pattern: `\S+\.(go|js|ts|py):\d+:\d+`, Flags: ""},
		{Pattern: `(?P<level>ERROR|WARN|INFO|DEBUG):\s+(?P<msg>.+)$`, Flags: "m"},
		{Pattern: `^[A-Z][a-z]+Error:`, Flags: "m"},
		{Pattern: `stack trace:|at \S+\(\S+:\d+:\d+\)`, Flags: "i"},
		{Pattern: `memory|heap|gc|allocation`, Flags: "i"},
	}
)

// newTestRootCmd creates a minimal root command for testing
func newTestRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "qualhook",
		Short:   "Performance test command",
		Version: "test",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 && args[0] == "invalid" {
				return fmt.Errorf("unknown command: %s", args[0])
			}
			return nil
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	// Disable output for performance testing
	cmd.SetOut(os.NewFile(0, os.DevNull))
	cmd.SetErr(os.NewFile(0, os.DevNull))

	// Add minimal subcommands for testing
	cmd.AddCommand(&cobra.Command{
		Use:   "test",
		Short: "Test command",
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	})

	return cmd
}

// TestStartupPerformance tests CLI startup time regression
func TestStartupPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance regression test in short mode")
	}

	testCases := []struct {
		name     string
		args     []string
		baseline float64 // milliseconds
	}{
		{"HelpCommand", []string{"--help"}, performanceBaselines.StartupHelp},
		{"VersionCommand", []string{"--version"}, performanceBaselines.StartupVersion},
		{"InvalidCommand", []string{"invalid"}, performanceBaselines.StartupCommand},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Warm up
			cmd := newTestRootCmd()
			cmd.SetArgs(tc.args)
			_ = cmd.Execute()

			// Measure startup time
			iterations := 100
			start := time.Now()
			for i := 0; i < iterations; i++ {
				cmd := newTestRootCmd()
				cmd.SetArgs(tc.args)
				_ = cmd.Execute()
			}
			elapsed := time.Since(start)

			avgMs := float64(elapsed.Nanoseconds()) / float64(iterations) / 1e6
			if avgMs > tc.baseline {
				t.Errorf("Startup performance regression: %s took %.2fms (baseline: %.2fms)",
					tc.name, avgMs, tc.baseline)
			}
			t.Logf("%s: %.2fms average (baseline: %.2fms)", tc.name, avgMs, tc.baseline)
		})
	}
}

// TestPatternMatchingPerformance tests regex pattern matching performance
func TestPatternMatchingPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance regression test in short mode")
	}

	testCases := []struct {
		name     string
		pattern  *pkgconfig.RegexPattern
		input    string
		baseline int64 // minimum ops/second
	}{
		{"SimplePattern_SmallInput", testPatterns[0], testSmallInput, performanceBaselines.PatternSimpleSmall},
		{"SimplePattern_LargeInput", testPatterns[0], testLargeInput, performanceBaselines.PatternSimpleLarge},
		{"ComplexPattern_SmallInput", testPatterns[5], testSmallInput, performanceBaselines.PatternComplexSmall},
		{"ComplexPattern_LargeInput", testPatterns[5], testLargeInput, performanceBaselines.PatternComplexLarge},
		{"ComplexPattern_HugeInput", testPatterns[6], testHugeInput, 500}, // 500 ops/sec for huge input
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			re, err := tc.pattern.Compile()
			if err != nil {
				t.Fatal(err)
			}

			// Warm up
			for i := 0; i < 100; i++ {
				_ = re.MatchString(tc.input)
			}

			// Measure performance
			start := time.Now()
			iterations := 0
			deadline := start.Add(time.Second)

			for time.Now().Before(deadline) {
				_ = re.MatchString(tc.input)
				iterations++
			}

			opsPerSec := int64(iterations)
			if opsPerSec < tc.baseline {
				t.Errorf("Pattern matching performance regression: %s achieved %d ops/sec (baseline: %d ops/sec)",
					tc.name, opsPerSec, tc.baseline)
			}
			t.Logf("%s: %d ops/sec (baseline: %d ops/sec)", tc.name, opsPerSec, tc.baseline)
		})
	}
}

// TestPatternSetPerformance tests pattern set matching performance
func TestPatternSetPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance regression test in short mode")
	}

	cache, err := filter.NewPatternCache()
	if err != nil {
		t.Fatal(err)
	}

	testCases := []struct {
		name     string
		patterns []*pkgconfig.RegexPattern
		input    string
		baseline int64 // minimum ops/second
	}{
		{"SmallSet_SmallInput", testPatterns[:3], testSmallInput, performanceBaselines.PatternSetSmall},
		{"SmallSet_LargeInput", testPatterns[:3], testLargeInput, performanceBaselines.PatternSetLarge},
		{"LargeSet_SmallInput", testPatterns, testSmallInput, 100000}, // 100K ops/sec
		{"LargeSet_LargeInput", testPatterns, testLargeInput, 1000},   // 1K ops/sec
		{"LargeSet_HugeInput", testPatterns, testHugeInput, 100},      // 100 ops/sec
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ps, err := filter.NewPatternSet(tc.patterns, cache)
			if err != nil {
				t.Fatal(err)
			}

			// Warm up
			for i := 0; i < 100; i++ {
				_ = ps.MatchAny(tc.input)
			}

			// Measure performance
			start := time.Now()
			iterations := 0
			deadline := start.Add(time.Second)

			for time.Now().Before(deadline) {
				_ = ps.MatchAny(tc.input)
				iterations++
			}

			opsPerSec := int64(iterations)
			if opsPerSec < tc.baseline {
				t.Errorf("Pattern set performance regression: %s achieved %d ops/sec (baseline: %d ops/sec)",
					tc.name, opsPerSec, tc.baseline)
			}
			t.Logf("%s: %d ops/sec (baseline: %d ops/sec)", tc.name, opsPerSec, tc.baseline)
		})
	}
}

// TestMemoryUsageRegression tests memory usage doesn't exceed baselines
func TestMemoryUsageRegression(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance regression test in short mode")
	}

	t.Run("SmallCommandExecution", func(t *testing.T) {
		exec := executor.NewCommandExecutor(5 * time.Second)
		opts := executor.ExecOptions{
			Timeout: 1 * time.Second,
		}

		// Force GC and get baseline
		runtime.GC()
		var m1 runtime.MemStats
		runtime.ReadMemStats(&m1)

		// Execute many small commands
		for i := 0; i < 100; i++ {
			_, _ = exec.Execute("echo", []string{fmt.Sprintf("test %d", i)}, opts)
		}

		// Force GC and measure
		runtime.GC()
		var m2 runtime.MemStats
		runtime.ReadMemStats(&m2)

		bytesPerOp := int64((m2.Alloc - m1.Alloc) / 100)
		if bytesPerOp > performanceBaselines.MemorySmallCommand {
			t.Errorf("Memory usage regression: small commands use %d bytes/op (baseline: %d bytes/op)",
				bytesPerOp, performanceBaselines.MemorySmallCommand)
		}
		t.Logf("Small command memory: %d bytes/op (baseline: %d bytes/op)",
			bytesPerOp, performanceBaselines.MemorySmallCommand)
	})

	t.Run("LargeCommandOutput", func(t *testing.T) {
		exec := executor.NewCommandExecutor(10 * time.Second)

		// Create script that generates large output
		tmpDir := t.TempDir()
		scriptPath := filepath.Join(tmpDir, "large_output.sh")
		script := `#!/bin/bash
for i in {1..1000}; do
    echo "Line $i: This is a test output line with some data"
done`
		os.WriteFile(scriptPath, []byte(script), 0755)

		// Force GC and get baseline
		runtime.GC()
		var m1 runtime.MemStats
		runtime.ReadMemStats(&m1)

		// Execute command with large output
		opts := executor.ExecOptions{
			Timeout: 5 * time.Second,
		}
		_, _ = exec.Execute("/bin/bash", []string{scriptPath}, opts)

		// Force GC and measure
		runtime.GC()
		var m2 runtime.MemStats
		runtime.ReadMemStats(&m2)

		memUsed := int64(m2.Alloc - m1.Alloc)
		if memUsed > performanceBaselines.MemoryLargeCommand {
			t.Errorf("Memory usage regression: large output used %d bytes (baseline: %d bytes)",
				memUsed, performanceBaselines.MemoryLargeCommand)
		}
		t.Logf("Large output memory: %d bytes (baseline: %d bytes)",
			memUsed, performanceBaselines.MemoryLargeCommand)
	})

	t.Run("PatternCompilation", func(t *testing.T) {
		// Force GC and get baseline
		runtime.GC()
		var m1 runtime.MemStats
		runtime.ReadMemStats(&m1)

		// Compile many patterns
		compiled := make([]*regexp.Regexp, 0, 100)
		for i := 0; i < 100; i++ {
			pattern := &pkgconfig.RegexPattern{
				Pattern: fmt.Sprintf(`pattern_%d_\d+`, i),
				Flags:   "i",
			}
			re, _ := pattern.Compile()
			compiled = append(compiled, re)
		}

		// Force GC and measure
		runtime.GC()
		var m2 runtime.MemStats
		runtime.ReadMemStats(&m2)

		bytesPerPattern := int64((m2.Alloc - m1.Alloc) / 100)
		if bytesPerPattern > performanceBaselines.MemoryPatternCompile {
			t.Errorf("Memory usage regression: pattern compilation uses %d bytes/pattern (baseline: %d bytes/pattern)",
				bytesPerPattern, performanceBaselines.MemoryPatternCompile)
		}
		t.Logf("Pattern compilation memory: %d bytes/pattern (baseline: %d bytes/pattern)",
			bytesPerPattern, performanceBaselines.MemoryPatternCompile)

		// Keep compiled patterns alive
		_ = compiled
	})
}

// TestConcurrentExecutionPerformance tests performance under concurrent load
func TestConcurrentExecutionPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance regression test in short mode")
	}

	cmdExec := executor.NewCommandExecutor(30 * time.Second)
	pe := executor.NewParallelExecutor(cmdExec, 4) // 4 concurrent workers

	// Create test commands
	commands := make([]executor.ParallelCommand, 8)
	for i := range commands {
		commands[i] = executor.ParallelCommand{
			ID:      fmt.Sprintf("cmd_%d", i),
			Command: "echo",
			Args:    []string{fmt.Sprintf("Concurrent command %d output", i)},
			Options: executor.ExecOptions{
				Timeout: 2 * time.Second,
			},
		}
	}

	// Warm up
	ctx := context.Background()
	_, _ = pe.Execute(ctx, commands[:4], nil)

	// Measure throughput
	start := time.Now()
	iterations := 0
	deadline := start.Add(10 * time.Second)

	// Track memory usage
	var maxMemory uint64
	memMonitor := make(chan struct{})
	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				var m runtime.MemStats
				runtime.ReadMemStats(&m)
				if m.Alloc > maxMemory {
					maxMemory = m.Alloc
				}
			case <-memMonitor:
				return
			}
		}
	}()

	for time.Now().Before(deadline) {
		_, err := pe.Execute(ctx, commands, nil)
		if err != nil {
			t.Logf("Concurrent execution error: %v", err)
		}
		iterations++
	}
	close(memMonitor)

	elapsed := time.Since(start).Seconds()
	opsPerSec := int64(float64(iterations) / elapsed)

	if opsPerSec < performanceBaselines.ConcurrentThroughput {
		t.Errorf("Concurrent execution performance regression: %d ops/sec (baseline: %d ops/sec)",
			opsPerSec, performanceBaselines.ConcurrentThroughput)
	}
	t.Logf("Concurrent throughput: %d ops/sec (baseline: %d ops/sec)",
		opsPerSec, performanceBaselines.ConcurrentThroughput)

	if int64(maxMemory) > performanceBaselines.ConcurrentMaxMemory {
		t.Errorf("Concurrent memory regression: %d bytes max (baseline: %d bytes)",
			maxMemory, performanceBaselines.ConcurrentMaxMemory)
	}
	t.Logf("Concurrent max memory: %d bytes (baseline: %d bytes)",
		maxMemory, performanceBaselines.ConcurrentMaxMemory)
}

// TestConfigLoadingPerformance tests configuration loading performance
func TestConfigLoadingPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance regression test in short mode")
	}

	// Create test configs
	tmpDir := t.TempDir()

	// Simple config
	simpleConfig := &pkgconfig.Config{
		Version: "1.0",
		Commands: map[string]*pkgconfig.CommandConfig{
			"lint": {
				Command:   "eslint",
				Args:      []string{"."},
				ExitCodes: []int{1},
			},
		},
	}
	simpleData, _ := pkgconfig.SaveConfig(simpleConfig)
	simplePath := filepath.Join(tmpDir, "simple.json")
	os.WriteFile(simplePath, simpleData, 0644)

	// Complex config with paths
	complexConfig := &pkgconfig.Config{
		Version: "1.0",
		Commands: map[string]*pkgconfig.CommandConfig{
			"format": {
				Command:   "prettier",
				Args:      []string{"--write", "."},
				ExitCodes: []int{1},
			},
			"lint": {
				Command:   "eslint",
				Args:      []string{"."},
				ExitCodes: []int{1},
				ErrorPatterns: []*pkgconfig.RegexPattern{
					{Pattern: `\d+ errors?`, Flags: "i"},
					{Pattern: `warning`, Flags: "i"},
				},
			},
			"test": {
				Command:   "jest",
				Args:      []string{"--coverage"},
				ExitCodes: []int{1},
			},
		},
		Paths: []*pkgconfig.PathConfig{
			{
				Path: "frontend/**/*.{js,jsx,ts,tsx}",
				Commands: map[string]*pkgconfig.CommandConfig{
					"lint": {
						Command: "eslint",
						Args:    []string{"--fix", "."},
					},
				},
			},
			{
				Path: "backend/**/*.go",
				Commands: map[string]*pkgconfig.CommandConfig{
					"lint": {
						Command: "golangci-lint",
						Args:    []string{"run"},
					},
				},
			},
		},
	}
	complexData, _ := pkgconfig.SaveConfig(complexConfig)
	complexPath := filepath.Join(tmpDir, "complex.json")
	os.WriteFile(complexPath, complexData, 0644)

	testCases := []struct {
		name         string
		configPath   string
		baselineMs   float64
		withValidate bool
	}{
		{"SimpleConfig", simplePath, 10.0, false},
		{"SimpleConfigWithValidation", simplePath, 15.0, true},
		{"ComplexConfig", complexPath, 20.0, false},
		{"ComplexConfigWithValidation", complexPath, 30.0, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			loader := config.NewLoader()
			validator := config.NewValidator()

			// Warm up
			for i := 0; i < 10; i++ {
				cfg, _ := loader.LoadFromPath(tc.configPath)
				if tc.withValidate {
					_ = validator.Validate(cfg)
				}
			}

			// Measure
			iterations := 100
			start := time.Now()
			for i := 0; i < iterations; i++ {
				cfg, err := loader.LoadFromPath(tc.configPath)
				if err != nil {
					t.Fatal(err)
				}
				if tc.withValidate {
					if err := validator.Validate(cfg); err != nil {
						t.Fatal(err)
					}
				}
			}
			elapsed := time.Since(start)

			avgMs := float64(elapsed.Nanoseconds()) / float64(iterations) / 1e6
			if avgMs > tc.baselineMs {
				t.Errorf("Config loading performance regression: %s took %.2fms (baseline: %.2fms)",
					tc.name, avgMs, tc.baselineMs)
			}
			t.Logf("%s: %.2fms average (baseline: %.2fms)", tc.name, avgMs, tc.baselineMs)
		})
	}
}

// TestProjectDetectionPerformance tests project detection performance
func TestProjectDetectionPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance regression test in short mode")
	}

	testCases := []struct {
		name       string
		files      map[string]string
		baselineMs float64
	}{
		{
			name: "NodeProject",
			files: map[string]string{
				"package.json":      `{"name": "test", "version": "1.0.0"}`,
				"package-lock.json": `{}`,
				".eslintrc.js":      `module.exports = {}`,
				"src/index.js":      `console.log("hello")`,
			},
			baselineMs: 5.0,
		},
		{
			name: "GoProject",
			files: map[string]string{
				"go.mod":       `module example.com/test`,
				"go.sum":       ``,
				"main.go":      `package main`,
				"Makefile":     `build:`,
				"cmd/main.go":  `package main`,
				"internal/foo": `package foo`,
			},
			baselineMs: 5.0,
		},
		{
			name: "Monorepo",
			files: map[string]string{
				"lerna.json":              `{"version": "1.0.0"}`,
				"package.json":            `{"workspaces": ["packages/*"]}`,
				"packages/a/package.json": `{"name": "a"}`,
				"packages/b/package.json": `{"name": "b"}`,
				"packages/c/package.json": `{"name": "c"}`,
				"backend/go.mod":          `module backend`,
				"frontend/package.json":   `{"name": "frontend"}`,
			},
			baselineMs: 10.0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create test directory structure
			tmpDir := t.TempDir()
			for path, content := range tc.files {
				fullPath := filepath.Join(tmpDir, path)
				os.MkdirAll(filepath.Dir(fullPath), 0755)
				os.WriteFile(fullPath, []byte(content), 0644)
			}

			pd := detector.New()

			// Warm up
			for i := 0; i < 10; i++ {
				_, _ = pd.Detect(tmpDir)
			}

			// Measure
			iterations := 100
			start := time.Now()
			for i := 0; i < iterations; i++ {
				_, err := pd.Detect(tmpDir)
				if err != nil {
					t.Fatal(err)
				}
			}
			elapsed := time.Since(start)

			avgMs := float64(elapsed.Nanoseconds()) / float64(iterations) / 1e6
			if avgMs > tc.baselineMs {
				t.Errorf("Project detection performance regression: %s took %.2fms (baseline: %.2fms)",
					tc.name, avgMs, tc.baselineMs)
			}
			t.Logf("%s: %.2fms average (baseline: %.2fms)", tc.name, avgMs, tc.baselineMs)
		})
	}
}

// TestEndToEndPerformance tests full command execution performance
func TestEndToEndPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance regression test in short mode")
	}

	// Create test environment
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".qualhook.json")

	testConfig := &pkgconfig.Config{
		Version: "1.0",
		Commands: map[string]*pkgconfig.CommandConfig{
			"echo-test": {
				Command:   "echo",
				Args:      []string{"test output"},
				ExitCodes: []int{1},
				ErrorPatterns: []*pkgconfig.RegexPattern{
					{Pattern: "error", Flags: "i"},
				},
				MaxOutput: 1000,
			},
			"date-test": {
				Command:   "date",
				Args:      []string{},
				ExitCodes: []int{1},
			},
		},
	}

	configData, err := pkgconfig.SaveConfig(testConfig)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, configData, 0644); err != nil {
		t.Fatal(err)
	}

	// For end-to-end test, we'll use the executor directly instead of going through CLI
	testCases := []struct {
		name       string
		command    string
		baselineMs float64
	}{
		{"SimpleEcho", "echo-test", 50.0},
		{"DateCommand", "date-test", 50.0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Load config
			loader := config.NewLoader()
			cfg, err := loader.LoadFromPath(configPath)
			if err != nil {
				t.Fatal(err)
			}

			cmdConfig, ok := cfg.Commands[tc.command]
			if !ok {
				t.Fatalf("Command %s not found in config", tc.command)
			}

			exec := executor.NewCommandExecutor(5 * time.Second)
			opts := executor.ExecOptions{
				Timeout: 2 * time.Second,
			}

			// Warm up
			_, _ = exec.Execute(cmdConfig.Command, cmdConfig.Args, opts)

			// Measure
			iterations := 50
			start := time.Now()
			for i := 0; i < iterations; i++ {
				_, _ = exec.Execute(cmdConfig.Command, cmdConfig.Args, opts)
			}
			elapsed := time.Since(start)

			avgMs := float64(elapsed.Nanoseconds()) / float64(iterations) / 1e6
			if avgMs > tc.baselineMs {
				t.Errorf("End-to-end performance regression: %s took %.2fms (baseline: %.2fms)",
					tc.name, avgMs, tc.baselineMs)
			}
			t.Logf("%s: %.2fms average (baseline: %.2fms)", tc.name, avgMs, tc.baselineMs)
		})
	}
}

// TestStressPerformance tests performance under stress conditions
func TestStressPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance regression test in short mode")
	}

	t.Run("ManyPatterns", func(t *testing.T) {
		// Create many patterns
		patterns := make([]*pkgconfig.RegexPattern, 100)
		for i := range patterns {
			patterns[i] = &pkgconfig.RegexPattern{
				Pattern: fmt.Sprintf(`pattern_%d_\w+`, i),
				Flags:   "i",
			}
		}

		cache, _ := filter.NewPatternCache()
		start := time.Now()

		// Compile all patterns
		for _, p := range patterns {
			_, err := cache.GetOrCompile(p)
			if err != nil {
				t.Fatal(err)
			}
		}

		elapsed := time.Since(start)
		if elapsed > 100*time.Millisecond {
			t.Errorf("Pattern compilation stress test failed: took %v (baseline: 100ms)", elapsed)
		}
		t.Logf("Compiled 100 patterns in %v", elapsed)
	})

	t.Run("ConcurrentPatternAccess", func(t *testing.T) {
		cache, _ := filter.NewPatternCache()
		pattern := &pkgconfig.RegexPattern{Pattern: `concurrent.*test`, Flags: "i"}
		re, _ := cache.GetOrCompile(pattern)

		var wg sync.WaitGroup
		workers := 10
		iterations := 10000
		start := time.Now()

		for i := 0; i < workers; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < iterations; j++ {
					_ = re.MatchString(testMediumInput)
				}
			}()
		}

		wg.Wait()
		elapsed := time.Since(start)
		totalOps := workers * iterations
		opsPerSec := int64(float64(totalOps) / elapsed.Seconds())

		if opsPerSec < 100000 { // 100K ops/sec minimum
			t.Errorf("Concurrent pattern access regression: %d ops/sec (baseline: 100000 ops/sec)", opsPerSec)
		}
		t.Logf("Concurrent pattern matching: %d ops/sec", opsPerSec)
	})

	t.Run("RapidCommandExecution", func(t *testing.T) {
		exec := executor.NewCommandExecutor(5 * time.Second)
		opts := executor.ExecOptions{
			Timeout: 500 * time.Millisecond,
		}

		start := time.Now()
		iterations := 100

		for i := 0; i < iterations; i++ {
			_, _ = exec.Execute("true", []string{}, opts)
		}

		elapsed := time.Since(start)
		avgMs := float64(elapsed.Nanoseconds()) / float64(iterations) / 1e6

		if avgMs > 10.0 { // 10ms per command max
			t.Errorf("Rapid command execution regression: %.2fms per command (baseline: 10.0ms)", avgMs)
		}
		t.Logf("Rapid command execution: %.2fms per command", avgMs)
	})
}

// BenchmarkRegressionSummary provides a summary benchmark for regression tracking
func BenchmarkRegressionSummary(b *testing.B) {
	b.Run("Startup", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			cmd := newTestRootCmd()
			cmd.SetArgs([]string{"--version"})
			_ = cmd.Execute()
		}
	})

	b.Run("PatternMatching", func(b *testing.B) {
		pattern := &pkgconfig.RegexPattern{Pattern: `error|warning|fatal`, Flags: "i"}
		re, _ := pattern.Compile()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = re.MatchString(testLargeInput)
		}
	})

	b.Run("ConfigLoading", func(b *testing.B) {
		tmpDir := b.TempDir()
		configPath := filepath.Join(tmpDir, ".qualhook.json")
		testConfig := &pkgconfig.Config{
			Version: "1.0",
			Commands: map[string]*pkgconfig.CommandConfig{
				"test": {Command: "echo", Args: []string{"test"}},
			},
		}
		data, _ := pkgconfig.SaveConfig(testConfig)
		os.WriteFile(configPath, data, 0644)

		loader := config.NewLoader()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = loader.LoadFromPath(configPath)
		}
	})

	b.Run("CommandExecution", func(b *testing.B) {
		exec := executor.NewCommandExecutor(1 * time.Second)
		opts := executor.ExecOptions{Timeout: 100 * time.Millisecond}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = exec.Execute("echo", []string{"bench"}, opts)
		}
	})
}
