// Package main provides the entry point for the qualhook CLI tool.
package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/qualhook/qualhook/internal/config"
	"github.com/qualhook/qualhook/internal/detector"
	"github.com/qualhook/qualhook/internal/executor"
	pkgconfig "github.com/qualhook/qualhook/pkg/config"
)

// BenchmarkCLIStartup measures the startup overhead of the CLI
func BenchmarkCLIStartup(b *testing.B) {
	// Save original args
	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	testCases := []struct {
		name string
		args []string
	}{
		{"HelpCommand", []string{"qualhook", "--help"}},
		{"VersionCommand", []string{"qualhook", "--version"}},
		{"InvalidCommand", []string{"qualhook", "invalid"}},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				os.Args = tc.args
				// Reset root command for each iteration
				cmd := newRootCmd()
				_ = cmd.Execute()
			}
		})
	}
}

// BenchmarkConfigLoading measures configuration loading performance
func BenchmarkConfigLoading(b *testing.B) {
	// Create a temporary config file
	tmpDir := b.TempDir()
	configPath := filepath.Join(tmpDir, ".qualhook.json")
	
	sampleConfig := &pkgconfig.Config{
		Version: "1.0",
		Commands: map[string]*pkgconfig.CommandConfig{
			"format": {
				Command: "prettier",
				Args:    []string{"--write", "."},
				ErrorDetection: &pkgconfig.ErrorDetection{
					ExitCodes: []int{1},
				},
			},
			"lint": {
				Command: "eslint",
				Args:    []string{"."},
				ErrorDetection: &pkgconfig.ErrorDetection{
					ExitCodes: []int{1},
					Patterns: []*pkgconfig.RegexPattern{
						{Pattern: `\d+ errors?`, Flags: "i"},
					},
				},
			},
		},
	}

	// Write config file
	configData, err := pkgconfig.SaveConfig(sampleConfig)
	if err != nil {
		b.Fatal(err)
	}
	if err := os.WriteFile(configPath, configData, 0644); err != nil {
		b.Fatal(err)
	}

	b.Run("SimpleConfig", func(b *testing.B) {
		loader := config.NewLoader()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = loader.LoadFromPath(configPath)
		}
	})

	b.Run("ComplexConfig", func(b *testing.B) {
		// Create a more complex config with multiple paths
		complexConfig := &pkgconfig.Config{
			Version:  "1.0",
			Commands: sampleConfig.Commands,
			Paths: []*pkgconfig.PathConfig{
				{
					Path:     "frontend/**",
					Commands: sampleConfig.Commands,
				},
				{
					Path:     "backend/**",
					Commands: sampleConfig.Commands,
				},
				{
					Path:     "packages/*/src/**",
					Commands: sampleConfig.Commands,
				},
			},
		}
		
		complexPath := filepath.Join(tmpDir, ".qualhook-complex.json")
		complexData, err := pkgconfig.SaveConfig(complexConfig)
		if err != nil {
			b.Fatal(err)
		}
		if err := os.WriteFile(complexPath, complexData, 0644); err != nil {
			b.Fatal(err)
		}

		loader := config.NewLoader()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = loader.LoadFromPath(complexPath)
		}
	})

	b.Run("WithValidation", func(b *testing.B) {
		loader := config.NewLoader()
		validator := config.NewValidator()
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			cfg, err := loader.LoadFromPath(configPath)
			if err == nil {
				_ = validator.Validate(cfg)
			}
		}
	})
}

// BenchmarkProjectDetection measures project detection performance
func BenchmarkProjectDetection(b *testing.B) {
	testCases := []struct {
		name  string
		files map[string]string
	}{
		{
			name: "NodeProject",
			files: map[string]string{
				"package.json":      `{"name": "test", "version": "1.0.0"}`,
				"package-lock.json": `{}`,
				".eslintrc.js":      `module.exports = {}`,
			},
		},
		{
			name: "GoProject",
			files: map[string]string{
				"go.mod":     `module example.com/test`,
				"go.sum":     ``,
				"main.go":    `package main`,
				"Makefile":   `build:`,
			},
		},
		{
			name: "Monorepo",
			files: map[string]string{
				"lerna.json":              `{"version": "1.0.0"}`,
				"package.json":            `{"workspaces": ["packages/*"]}`,
				"packages/a/package.json": `{"name": "a"}`,
				"packages/b/package.json": `{"name": "b"}`,
				"backend/go.mod":          `module backend`,
			},
		},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			// Create test directory structure
			tmpDir := b.TempDir()
			for path, content := range tc.files {
				fullPath := filepath.Join(tmpDir, path)
				os.MkdirAll(filepath.Dir(fullPath), 0755)
				os.WriteFile(fullPath, []byte(content), 0644)
			}

			pd := detector.New()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = pd.Detect(tmpDir)
			}
		})
	}
}

// BenchmarkCommandExecution measures command execution overhead
func BenchmarkCommandExecution(b *testing.B) {
	exec := executor.NewCommandExecutor(2 * time.Minute)

	testCases := []struct {
		name    string
		command string
		args    []string
	}{
		{"EchoCommand", "echo", []string{"test"}},
		{"TrueCommand", "true", []string{}},
		{"DateCommand", "date", []string{}},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			opts := executor.ExecOptions{
				Timeout: 5 * time.Second,
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = exec.Execute(tc.command, tc.args, opts)
			}
		})
	}
}

// BenchmarkEndToEnd measures full command execution from CLI to result
func BenchmarkEndToEnd(b *testing.B) {
	// Create a minimal config
	tmpDir := b.TempDir()
	configPath := filepath.Join(tmpDir, ".qualhook.json")
	
	minimalConfig := &pkgconfig.Config{
		Version: "1.0",
		Commands: map[string]*pkgconfig.CommandConfig{
			"test": {
				Command: "echo",
				Args:    []string{"test output"},
				ErrorDetection: &pkgconfig.ErrorDetection{
					ExitCodes: []int{1},
				},
				OutputFilter: &pkgconfig.FilterConfig{
					ErrorPatterns: []*pkgconfig.RegexPattern{
						{Pattern: "error", Flags: "i"},
					},
					MaxOutput: 100,
				},
			},
		},
	}

	configData, err := pkgconfig.SaveConfig(minimalConfig)
	if err != nil {
		b.Fatal(err)
	}
	if err := os.WriteFile(configPath, configData, 0644); err != nil {
		b.Fatal(err)
	}

	// Change to test directory
	origWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origWd)

	// Save original args
	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		os.Args = []string{"qualhook", "test"}
		rootCmd = newRootCmd()
		_ = rootCmd.Execute()
	}
}

// BenchmarkMemoryUsage measures memory allocations during startup
func BenchmarkMemoryUsage(b *testing.B) {
	b.Run("CLIInitialization", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = newRootCmd()
		}
	})

	b.Run("ConfigLoading", func(b *testing.B) {
		tmpDir := b.TempDir()
		configPath := filepath.Join(tmpDir, ".qualhook.json")
		
		sampleConfig := &pkgconfig.Config{
			Version: "1.0",
			Commands: map[string]*pkgconfig.CommandConfig{
				"lint": {
					Command: "eslint",
					Args:    []string{"."},
				},
			},
		}
		
		configData, _ := pkgconfig.SaveConfig(sampleConfig)
		os.WriteFile(configPath, configData, 0644)
		loader := config.NewLoader()
		
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = loader.LoadFromPath(configPath)
		}
	})
}

// BenchmarkConcurrentExecution measures performance under concurrent load
func BenchmarkConcurrentExecution(b *testing.B) {
	exec := executor.NewCommandExecutor(2 * time.Minute)
	opts := executor.ExecOptions{
		Timeout: 1 * time.Second,
	}

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = exec.Execute("echo", []string{"concurrent test"}, opts)
		}
	})
}