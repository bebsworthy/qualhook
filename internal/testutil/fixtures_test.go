package testutil

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFixturePath(t *testing.T) {
	tests := []struct {
		name         string
		relativePath string
		shouldExist  bool
	}{
		{
			name:         "config fixture",
			relativePath: "configs/basic.qualhook.json",
			shouldExist:  true,
		},
		{
			name:         "project fixture",
			relativePath: "projects/golang",
			shouldExist:  true,
		},
		{
			name:         "output fixture",
			relativePath: "outputs/error_output.txt",
			shouldExist:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := FixturePath(t, tt.relativePath)

			// Check path contains expected components
			if !strings.Contains(path, "test/fixtures") {
				t.Errorf("Path doesn't contain test/fixtures: %s", path)
			}

			// Verify absolute path
			if !filepath.IsAbs(path) {
				t.Errorf("Path is not absolute: %s", path)
			}

			// Check existence
			_, err := os.Stat(path)
			if tt.shouldExist && err != nil {
				t.Errorf("Expected fixture to exist: %s", path)
			}
		})
	}
}

func TestLoadFixture(t *testing.T) {
	tests := []struct {
		name         string
		relativePath string
		contains     string
	}{
		{
			name:         "load config",
			relativePath: "configs/basic.qualhook.json",
			contains:     `"commands"`,
		},
		{
			name:         "load output",
			relativePath: "outputs/error_output.txt",
			contains:     "ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := LoadFixture(t, tt.relativePath)

			if len(data) == 0 {
				t.Error("Loaded empty fixture")
			}

			if !strings.Contains(string(data), tt.contains) {
				t.Errorf("Fixture doesn't contain expected string '%s'", tt.contains)
			}
		})
	}
}

func TestConfigFixture(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "with extension",
			input:    "basic.qualhook.json",
			expected: "configs/basic.qualhook.json",
		},
		{
			name:     "without extension",
			input:    "basic",
			expected: "configs/basic.qualhook.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := ConfigFixture(t, tt.input)

			if !strings.HasSuffix(path, tt.expected) {
				t.Errorf("Path doesn't end with %s: %s", tt.expected, path)
			}
		})
	}
}

func TestCreateTempFixture(t *testing.T) {
	// Test copying a config file
	t.Run("copy config", func(t *testing.T) {
		tempPath := CreateTempFixture(t, "configs/basic.qualhook.json")

		// Check temp directory was created
		if !strings.HasPrefix(tempPath, os.TempDir()) {
			t.Errorf("Temp path not in temp directory: %s", tempPath)
		}

		// Check file was copied
		copiedFile := filepath.Join(tempPath, "basic.qualhook.json")
		if _, err := os.Stat(copiedFile); err != nil {
			t.Errorf("Copied file not found: %s", copiedFile)
		}
	})

	// Test copying a project directory
	t.Run("copy project", func(t *testing.T) {
		tempPath := CreateTempFixture(t, "projects/golang")

		// Check files were copied
		expectedFiles := []string{"go.mod", "main.go", "main_test.go", ".qualhook.json"}
		for _, file := range expectedFiles {
			path := filepath.Join(tempPath, file)
			if _, err := os.Stat(path); err != nil {
				t.Errorf("Expected file not found: %s", path)
			}
		}
	})
}
