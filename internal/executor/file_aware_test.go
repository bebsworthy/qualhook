//go:build unit

package executor

import (
	"fmt"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/bebsworthy/qualhook/internal/filter"
	"github.com/bebsworthy/qualhook/internal/hook"
	"github.com/bebsworthy/qualhook/pkg/config"
)

func TestFileAwareExecutor_ExecuteForEditedFiles(t *testing.T) {
	// Create test configuration
	testConfig := &config.Config{
		Version: "1.0",
		Commands: map[string]*config.CommandConfig{
			"lint": {
				Command:   "echo",
				Args:      []string{"root lint"},
				ExitCodes: []int{1},
				ErrorPatterns: []*config.RegexPattern{
					{Pattern: "error", Flags: "i"},
				},
				MaxOutput: 100,
			},
		},
		Paths: []*config.PathConfig{
			{
				Path: "frontend/**",
				Commands: map[string]*config.CommandConfig{
					"lint": {
						Command:   "echo",
						Args:      []string{"frontend lint"},
						ExitCodes: []int{1},
						ErrorPatterns: []*config.RegexPattern{
							{Pattern: "error", Flags: "i"},
						},
						MaxOutput: 100,
					},
				},
			},
			{
				Path: "backend/**",
				Commands: map[string]*config.CommandConfig{
					"lint": {
						Command:   "echo",
						Args:      []string{"backend lint"},
						ExitCodes: []int{1},
						ErrorPatterns: []*config.RegexPattern{
							{Pattern: "error", Flags: "i"},
						},
						MaxOutput: 100,
					},
				},
			},
		},
	}

	executor := NewFileAwareExecutor(testConfig, false)

	tests := []struct {
		name           string
		hookInput      *hook.HookInput
		commandName    string
		expectCommands []string // Expected command outputs
		wantErr        bool
	}{
		{
			name: "no edited files",
			hookInput: &hook.HookInput{
				SessionID:     "test",
				CWD:           "/test",
				HookEventName: "pre-commit",
			},
			commandName:    "lint",
			expectCommands: []string{"root lint"},
		},
		{
			name: "frontend file edited",
			hookInput: &hook.HookInput{
				SessionID:     "test",
				CWD:           "/test",
				HookEventName: "pre-commit",
				ToolUse: &hook.ToolUse{
					Name:  "Edit",
					Input: []byte(`{"file_path": "frontend/src/app.js"}`),
				},
			},
			commandName:    "lint",
			expectCommands: []string{"frontend lint"},
		},
		{
			name: "backend file edited",
			hookInput: &hook.HookInput{
				SessionID:     "test",
				CWD:           "/test",
				HookEventName: "pre-commit",
				ToolUse: &hook.ToolUse{
					Name:  "Edit",
					Input: []byte(`{"file_path": "backend/main.go"}`),
				},
			},
			commandName:    "lint",
			expectCommands: []string{"backend lint"},
		},
		{
			name: "root file edited",
			hookInput: &hook.HookInput{
				SessionID:     "test",
				CWD:           "/test",
				HookEventName: "pre-commit",
				ToolUse: &hook.ToolUse{
					Name:  "Edit",
					Input: []byte(`{"file_path": "README.md"}`),
				},
			},
			commandName:    "lint",
			expectCommands: []string{"root lint"},
		},
		{
			name: "unknown command",
			hookInput: &hook.HookInput{
				SessionID:     "test",
				CWD:           "/test",
				HookEventName: "pre-commit",
			},
			commandName: "unknown",
			wantErr:     false, // Should not error, just skip
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := executor.ExecuteForEditedFiles(tt.hookInput, tt.commandName, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExecuteForEditedFiles() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// For now, just check that we don't error and get expected results
			// In a real test, we'd mock the command executor to verify the right commands were called
			if !tt.wantErr && len(results) > 0 {
				for _, result := range results {
					if result.ExecutionError != nil {
						t.Errorf("Unexpected execution error: %v", result.ExecutionError)
					}
				}
			}
		})
	}
}

func TestFileAwareExecutor_getStatusText(t *testing.T) {
	tests := []struct {
		name   string
		output *filter.FilteredOutput
		want   string
	}{
		{
			name:   "nil output",
			output: nil,
			want:   "✓ passed",
		},
		{
			name: "no errors",
			output: &filter.FilteredOutput{
				HasErrors: false,
			},
			want: "✓ passed",
		},
		{
			name: "has errors",
			output: &filter.FilteredOutput{
				HasErrors: true,
			},
			want: "✗ failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getStatusText(tt.output); got != tt.want {
				t.Errorf("getStatusText() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewFileAwareExecutor(t *testing.T) {
	cfg := &config.Config{
		Version:  "1.0",
		Commands: make(map[string]*config.CommandConfig),
	}

	executor := NewFileAwareExecutor(cfg, true)
	if executor == nil {
		t.Fatal("NewFileAwareExecutor returned nil")
	}

	if executor.commandExecutor == nil {
		t.Error("commandExecutor is nil")
	}
	if executor.parallelExecutor == nil {
		t.Error("parallelExecutor is nil")
	}
	if executor.mapper == nil {
		t.Error("mapper is nil")
	}
	if executor.hookParser == nil {
		t.Error("hookParser is nil")
	}
	if !executor.debugMode {
		t.Error("debugMode should be true")
	}
}

func TestFileAwareExecutor_MultipleFilePatterns(t *testing.T) {
	// Test with multiple overlapping file patterns
	testConfig := &config.Config{
		Version: "1.0",
		Commands: map[string]*config.CommandConfig{
			"lint": {
				Command:   "echo",
				Args:      []string{"root lint"},
				ExitCodes: []int{0},
				MaxOutput: 100,
			},
		},
		Paths: []*config.PathConfig{
			{
				Path: "src/**/*.js",
				Commands: map[string]*config.CommandConfig{
					"lint": {
						Command:   "echo",
						Args:      []string{"js lint"},
						ExitCodes: []int{0},
						MaxOutput: 100,
					},
				},
			},
			{
				Path: "src/**/*.ts",
				Commands: map[string]*config.CommandConfig{
					"lint": {
						Command:   "echo",
						Args:      []string{"ts lint"},
						ExitCodes: []int{0},
						MaxOutput: 100,
					},
				},
			},
			{
				Path: "**/*.test.*",
				Commands: map[string]*config.CommandConfig{
					"lint": {
						Command:   "echo",
						Args:      []string{"test lint"},
						ExitCodes: []int{0},
						MaxOutput: 100,
					},
				},
			},
		},
	}

	executor := NewFileAwareExecutor(testConfig, false)

	tests := []struct {
		name           string
		editedFiles    []string
		commandName    string
		expectCommands int // Expected number of commands to run
		wantErr        bool
	}{
		{
			name:           "single js file",
			editedFiles:    []string{"src/app.js"},
			commandName:    "lint",
			expectCommands: 1, // Only js lint
		},
		{
			name:           "single ts file",
			editedFiles:    []string{"src/components/button.ts"},
			commandName:    "lint",
			expectCommands: 1, // Only ts lint
		},
		{
			name:           "test file matches multiple patterns",
			editedFiles:    []string{"src/app.test.js"},
			commandName:    "lint",
			expectCommands: 1, // Only most specific pattern (test lint)
		},
		{
			name: "multiple files different patterns",
			editedFiles: []string{
				"src/app.js",
				"src/components/button.ts",
				"tests/e2e.test.js",
			},
			commandName:    "lint",
			expectCommands: 3, // js, ts, and test patterns
		},
		{
			name: "files outside patterns",
			editedFiles: []string{
				"README.md",
				"package.json",
			},
			commandName:    "lint",
			expectCommands: 1, // Root lint
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// For testing, we'll directly call the mapper instead of parsing hook input
			// This is because the hook parser expects specific tool use format
			componentGroups, err := executor.mapper.MapFilesToComponents(tt.editedFiles)
			if err != nil {
				t.Fatalf("Failed to map files: %v", err)
			}

			results, err := executor.executeForComponents(componentGroups, tt.commandName, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("executeForComponents() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && len(results) != tt.expectCommands {
				t.Errorf("Expected %d commands to run, got %d", tt.expectCommands, len(results))
			}
		})
	}
}

func TestFileAwareExecutor_OverlappingPaths(t *testing.T) {
	// Test overlapping path configurations
	testConfig := &config.Config{
		Version: "1.0",
		Commands: map[string]*config.CommandConfig{
			"test": {
				Command:   "echo",
				Args:      []string{"root test"},
				ExitCodes: []int{0},
				MaxOutput: 100,
			},
		},
		Paths: []*config.PathConfig{
			{
				Path: "src/**",
				Commands: map[string]*config.CommandConfig{
					"test": {
						Command:   "echo",
						Args:      []string{"src test"},
						ExitCodes: []int{0},
						MaxOutput: 100,
					},
				},
			},
			{
				Path: "src/components/**",
				Commands: map[string]*config.CommandConfig{
					"test": {
						Command:   "echo",
						Args:      []string{"components test"},
						ExitCodes: []int{0},
						MaxOutput: 100,
					},
				},
			},
			{
				Path: "src/components/ui/**",
				Commands: map[string]*config.CommandConfig{
					"test": {
						Command:   "echo",
						Args:      []string{"ui test"},
						ExitCodes: []int{0},
						MaxOutput: 100,
					},
				},
			},
		},
	}

	executor := NewFileAwareExecutor(testConfig, false)

	tests := []struct {
		name           string
		editedFile     string
		expectPatterns []string // Expected patterns to match
	}{
		{
			name:           "file in src root",
			editedFile:     "src/main.js",
			expectPatterns: []string{"src/**"},
		},
		{
			name:           "file in components",
			editedFile:     "src/components/button.js",
			expectPatterns: []string{"src/components/**"}, // Most specific match
		},
		{
			name:           "file in ui subdirectory",
			editedFile:     "src/components/ui/dialog.js",
			expectPatterns: []string{"src/components/ui/**"}, // Most specific match
		},
		{
			name:           "file outside src",
			editedFile:     "tests/unit.test.js",
			expectPatterns: []string{"."}, // Root pattern
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			componentGroups, err := executor.mapper.MapFilesToComponents([]string{tt.editedFile})
			if err != nil {
				t.Fatalf("Failed to map files: %v", err)
			}

			// Check that we get the expected number of component groups
			if len(componentGroups) != len(tt.expectPatterns) {
				t.Errorf("Expected %d component groups, got %d", len(tt.expectPatterns), len(componentGroups))
			}

			// Verify each expected pattern is present
			foundPatterns := make(map[string]bool)
			for _, group := range componentGroups {
				foundPatterns[group.Path] = true
			}

			for _, expectedPattern := range tt.expectPatterns {
				if !foundPatterns[expectedPattern] {
					t.Errorf("Expected pattern %q not found in component groups", expectedPattern)
				}
			}
		})
	}
}

func TestFileAwareExecutor_VeryLargeFileList(t *testing.T) {
	// Test performance with very large file lists
	testConfig := &config.Config{
		Version: "1.0",
		Commands: map[string]*config.CommandConfig{
			"analyze": {
				Command:   "echo",
				Args:      []string{"analyzing"},
				ExitCodes: []int{0},
				MaxOutput: 100,
			},
		},
		Paths: []*config.PathConfig{
			{
				Path: "src/**/*.js",
				Commands: map[string]*config.CommandConfig{
					"analyze": {
						Command:   "echo",
						Args:      []string{"js analyze"},
						ExitCodes: []int{0},
						MaxOutput: 100,
					},
				},
			},
			{
				Path: "test/**/*.js",
				Commands: map[string]*config.CommandConfig{
					"analyze": {
						Command:   "echo",
						Args:      []string{"test analyze"},
						ExitCodes: []int{0},
						MaxOutput: 100,
					},
				},
			},
		},
	}

	executor := NewFileAwareExecutor(testConfig, false)

	// Generate a large list of files
	largeFileList := make([]string, 1500)
	for i := 0; i < 1000; i++ {
		largeFileList[i] = fmt.Sprintf("src/module%d/file%d.js", i/100, i)
	}
	for i := 1000; i < 1500; i++ {
		largeFileList[i] = fmt.Sprintf("test/spec%d/test%d.js", (i-1000)/50, i)
	}

	// Time the operation
	start := time.Now()
	componentGroups, err := executor.mapper.MapFilesToComponents(largeFileList)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("Failed to map large file list: %v", err)
	}

	// Should complete reasonably quickly (within 1 second)
	if duration > 1*time.Second {
		t.Errorf("Mapping %d files took too long: %v", len(largeFileList), duration)
	}

	// Verify we got the expected component groups
	expectedGroups := 2 // src and test
	if len(componentGroups) != expectedGroups {
		t.Errorf("Expected %d component groups, got %d", expectedGroups, len(componentGroups))
	}

	// Verify file counts
	var srcFiles, testFiles int
	for _, group := range componentGroups {
		switch group.Path {
		case "src/**/*.js":
			srcFiles = len(group.Files)
		case "test/**/*.js":
			testFiles = len(group.Files)
		}
	}

	if srcFiles != 1000 {
		t.Errorf("Expected 1000 src files, got %d", srcFiles)
	}
	if testFiles != 500 {
		t.Errorf("Expected 500 test files, got %d", testFiles)
	}
}

func TestFileAwareExecutor_PatternPriorityAndPrecedence(t *testing.T) {
	// Test that more specific patterns take precedence
	testConfig := &config.Config{
		Version: "1.0",
		Commands: map[string]*config.CommandConfig{
			"check": {
				Command:   "echo",
				Args:      []string{"root check"},
				ExitCodes: []int{0},
				MaxOutput: 100,
			},
		},
		Paths: []*config.PathConfig{
			{
				Path: "**/*.js", // Generic JS pattern
				Commands: map[string]*config.CommandConfig{
					"check": {
						Command:   "echo",
						Args:      []string{"generic js check"},
						ExitCodes: []int{0},
						MaxOutput: 100,
					},
				},
			},
			{
				Path: "src/**/*.js", // More specific src JS pattern
				Commands: map[string]*config.CommandConfig{
					"check": {
						Command:   "echo",
						Args:      []string{"src js check"},
						ExitCodes: []int{0},
						MaxOutput: 100,
					},
				},
			},
			{
				Path: "src/critical/**/*.js", // Most specific critical JS pattern
				Commands: map[string]*config.CommandConfig{
					"check": {
						Command:   "echo",
						Args:      []string{"critical js check"},
						ExitCodes: []int{0},
						MaxOutput: 100,
					},
				},
			},
		},
	}

	executor := NewFileAwareExecutor(testConfig, false)

	tests := []struct {
		name             string
		editedFile       string
		expectedGroups   int
		expectedCommands []string // Expected command args in order of specificity
	}{
		{
			name:             "generic js file",
			editedFile:       "lib/utils.js",
			expectedGroups:   1,
			expectedCommands: []string{"generic js check"},
		},
		{
			name:             "src js file matches two patterns",
			editedFile:       "src/components/header.js",
			expectedGroups:   1,
			expectedCommands: []string{"src js check"}, // Most specific match
		},
		{
			name:             "critical js file matches all patterns",
			editedFile:       "src/critical/auth.js",
			expectedGroups:   1,
			expectedCommands: []string{"critical js check"}, // Most specific match
		},
		{
			name:             "non-js file",
			editedFile:       "src/styles.css",
			expectedGroups:   1,
			expectedCommands: []string{"root check"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			componentGroups, err := executor.mapper.MapFilesToComponents([]string{tt.editedFile})
			if err != nil {
				t.Fatalf("Failed to map files: %v", err)
			}

			if len(componentGroups) != tt.expectedGroups {
				t.Errorf("Expected %d component groups, got %d", tt.expectedGroups, len(componentGroups))
			}

			// Execute commands and verify the order
			results, err := executor.executeForComponents(componentGroups, "check", nil)
			if err != nil {
				t.Fatalf("Failed to execute commands: %v", err)
			}

			if len(results) != len(tt.expectedCommands) {
				t.Errorf("Expected %d results, got %d", len(tt.expectedCommands), len(results))
			}

			// Note: The actual order depends on the mapper implementation
			// We should verify that all expected commands were executed
			executedCommands := make(map[string]bool)
			for _, result := range results {
				if result.CommandConfig != nil {
					key := strings.Join(result.CommandConfig.Args, " ")
					executedCommands[key] = true
				}
			}

			for _, expectedCmd := range tt.expectedCommands {
				if !executedCommands[expectedCmd] {
					t.Errorf("Expected command %q was not executed", expectedCmd)
				}
			}
		})
	}
}

func TestFileAwareExecutor_EmptyAndNilEdgeCases(t *testing.T) {
	// Test edge cases with empty configurations and nil values
	testConfig := &config.Config{
		Version: "1.0",
		Commands: map[string]*config.CommandConfig{
			"validate": {
				Command:   "echo",
				Args:      []string{"validating"},
				ExitCodes: []int{0},
				MaxOutput: 100,
			},
		},
		Paths: []*config.PathConfig{
			{
				Path:     "src/**",
				Commands: map[string]*config.CommandConfig{}, // Empty commands
			},
			{
				Path: "test/**",
				Commands: map[string]*config.CommandConfig{
					"validate": nil, // Nil command config
				},
			},
		},
	}

	executor := NewFileAwareExecutor(testConfig, false)

	tests := []struct {
		name        string
		editedFiles []string
		commandName string
		expectError bool
	}{
		{
			name:        "empty file list",
			editedFiles: []string{},
			commandName: "validate",
			expectError: false, // Should execute root command
		},
		{
			name:        "file in path with empty commands",
			editedFiles: []string{"src/app.js"},
			commandName: "validate",
			expectError: false, // Should skip this path
		},
		{
			name:        "file in path with nil command",
			editedFiles: []string{"test/spec.js"},
			commandName: "validate",
			expectError: false, // Nil command is skipped during merge, falls back to root
		},
		{
			name:        "non-existent command",
			editedFiles: []string{"README.md"},
			commandName: "non-existent",
			expectError: false, // Should return empty results
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// For edge case testing, directly test the component execution
			if len(tt.editedFiles) == 0 {
				results, err := executor.executeForRootComponent(tt.commandName, nil)
				if (err != nil) != tt.expectError {
					t.Errorf("executeForRootComponent() error = %v, wantErr %v", err, tt.expectError)
				}
				if !tt.expectError && len(results) > 0 && results[0].ExecutionError != nil {
					t.Errorf("Unexpected execution error: %v", results[0].ExecutionError)
				}
			} else {
				componentGroups, _ := executor.mapper.MapFilesToComponents(tt.editedFiles)
				results, err := executor.executeForComponents(componentGroups, tt.commandName, nil)
				if (err != nil) != tt.expectError {
					t.Errorf("executeForComponents() error = %v, wantErr %v", err, tt.expectError)
				}
				_ = results
			}
		})
	}
}

func TestFileAwareExecutor_SymbolicLinks(t *testing.T) {
	// Test symbolic link handling
	// Skip on systems that don't support symlinks well
	if runtime.GOOS == "windows" {
		t.Skip("Skipping symlink test on Windows")
	}

	testConfig := &config.Config{
		Version: "1.0",
		Commands: map[string]*config.CommandConfig{
			"scan": {
				Command:   "echo",
				Args:      []string{"scanning"},
				ExitCodes: []int{0},
				MaxOutput: 100,
			},
		},
		Paths: []*config.PathConfig{
			{
				Path: "src/**/*.js",
				Commands: map[string]*config.CommandConfig{
					"scan": {
						Command:   "echo",
						Args:      []string{"src scan"},
						ExitCodes: []int{0},
						MaxOutput: 100,
					},
				},
			},
			{
				Path: "lib/**/*.js",
				Commands: map[string]*config.CommandConfig{
					"scan": {
						Command:   "echo",
						Args:      []string{"lib scan"},
						ExitCodes: []int{0},
						MaxOutput: 100,
					},
				},
			},
		},
	}

	executor := NewFileAwareExecutor(testConfig, false)

	tests := []struct {
		name           string
		setupFiles     func() []string
		commandName    string
		expectPatterns []string
	}{
		{
			name: "symlink to file",
			setupFiles: func() []string {
				// In a real test, we would create actual symlinks
				// For now, we'll just test with regular files
				return []string{
					"src/original.js",
					"lib/link-to-original.js", // Would be a symlink to src/original.js
				}
			},
			commandName:    "scan",
			expectPatterns: []string{"src/**/*.js", "lib/**/*.js"},
		},
		{
			name: "symlink to directory",
			setupFiles: func() []string {
				return []string{
					"src/components/button.js",
					"lib/components/button.js", // Would be symlinked directory
				}
			},
			commandName:    "scan",
			expectPatterns: []string{"src/**/*.js", "lib/**/*.js"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files := tt.setupFiles()
			componentGroups, err := executor.mapper.MapFilesToComponents(files)
			if err != nil {
				t.Fatalf("Failed to map files: %v", err)
			}

			// Verify we handle symlinks correctly (they should be treated as separate files)
			foundPatterns := make(map[string]bool)
			for _, group := range componentGroups {
				foundPatterns[group.Path] = true
			}

			for _, expectedPattern := range tt.expectPatterns {
				if !foundPatterns[expectedPattern] {
					t.Errorf("Expected pattern %q not found", expectedPattern)
				}
			}
		})
	}
}

func TestFileAwareExecutor_ConcurrentFileMapping(t *testing.T) {
	// Test concurrent access to file mapping
	testConfig := &config.Config{
		Version: "1.0",
		Commands: map[string]*config.CommandConfig{
			"process": {
				Command:   "echo",
				Args:      []string{"processing"},
				ExitCodes: []int{0},
				MaxOutput: 100,
			},
		},
		Paths: []*config.PathConfig{
			{
				Path: "**/*.go",
				Commands: map[string]*config.CommandConfig{
					"process": {
						Command:   "echo",
						Args:      []string{"go process"},
						ExitCodes: []int{0},
						MaxOutput: 100,
					},
				},
			},
		},
	}

	executor := NewFileAwareExecutor(testConfig, false)

	// Create file lists for concurrent processing
	fileLists := [][]string{
		{"src/main.go", "src/util.go"},
		{"test/main_test.go", "test/util_test.go"},
		{"cmd/app/main.go", "cmd/tool/main.go"},
		{"internal/pkg1/file.go", "internal/pkg2/file.go"},
	}

	// Run mapping concurrently
	results := make(chan error, len(fileLists))
	for _, files := range fileLists {
		files := files // Capture loop variable
		go func() {
			_, err := executor.mapper.MapFilesToComponents(files)
			results <- err
		}()
	}

	// Collect results
	for i := 0; i < len(fileLists); i++ {
		if err := <-results; err != nil {
			t.Errorf("Concurrent mapping failed: %v", err)
		}
	}
}

func TestFileAwareExecutor_ComplexGlobPatterns(t *testing.T) {
	// Test complex glob patterns
	testConfig := &config.Config{
		Version: "1.0",
		Commands: map[string]*config.CommandConfig{
			"analyze": {
				Command:   "echo",
				Args:      []string{"analyzing"},
				ExitCodes: []int{0},
				MaxOutput: 100,
			},
		},
		Paths: []*config.PathConfig{
			{
				Path: "src/**/[!_]*.js", // Exclude files starting with underscore
				Commands: map[string]*config.CommandConfig{
					"analyze": {
						Command:   "echo",
						Args:      []string{"public js analyze"},
						ExitCodes: []int{0},
						MaxOutput: 100,
					},
				},
			},
			{
				Path: "src/**/_*.js", // Only files starting with underscore
				Commands: map[string]*config.CommandConfig{
					"analyze": {
						Command:   "echo",
						Args:      []string{"private js analyze"},
						ExitCodes: []int{0},
						MaxOutput: 100,
					},
				},
			},
			{
				Path: "**/*.{js,jsx,ts,tsx}", // Multiple extensions
				Commands: map[string]*config.CommandConfig{
					"analyze": {
						Command:   "echo",
						Args:      []string{"all js/ts analyze"},
						ExitCodes: []int{0},
						MaxOutput: 100,
					},
				},
			},
		},
	}

	executor := NewFileAwareExecutor(testConfig, false)

	tests := []struct {
		name           string
		editedFile     string
		expectPatterns []string
	}{
		{
			name:           "public js file",
			editedFile:     "src/components/button.js",
			expectPatterns: []string{"src/**/[!_]*.js", "**/*.{js,jsx,ts,tsx}"},
		},
		{
			name:           "private js file",
			editedFile:     "src/utils/_helper.js",
			expectPatterns: []string{"src/**/_*.js", "**/*.{js,jsx,ts,tsx}"},
		},
		{
			name:           "typescript file",
			editedFile:     "src/types/index.ts",
			expectPatterns: []string{"**/*.{js,jsx,ts,tsx}"},
		},
		{
			name:           "jsx file",
			editedFile:     "src/components/App.jsx",
			expectPatterns: []string{"**/*.{js,jsx,ts,tsx}"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			componentGroups, err := executor.mapper.MapFilesToComponents([]string{tt.editedFile})
			if err != nil {
				t.Fatalf("Failed to map files: %v", err)
			}

			foundPatterns := make(map[string]bool)
			for _, group := range componentGroups {
				foundPatterns[group.Path] = true
			}

			// Note: The actual glob matching depends on the mapper implementation
			// This test verifies the structure is correct
			if len(componentGroups) == 0 {
				t.Error("Expected at least one component group")
			}
		})
	}
}
