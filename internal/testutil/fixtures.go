// Package testutil provides utilities for loading test fixtures and managing test data.
package testutil

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// FixturePath returns the absolute path to a fixture file or directory.
// The path is relative to the test/fixtures directory.
func FixturePath(t *testing.T, relativePath string) string {
	t.Helper()

	// Get the directory of this source file
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("Failed to get source file path")
	}

	// Navigate to project root
	projectRoot := filepath.Join(filepath.Dir(filename), "..", "..")
	fixturePath := filepath.Join(projectRoot, "test", "fixtures", relativePath)

	// Clean the path
	fixturePath = filepath.Clean(fixturePath)

	// Verify the fixture exists
	if _, err := os.Stat(fixturePath); err != nil {
		t.Fatalf("Fixture not found: %s", fixturePath)
	}

	return fixturePath
}

// LoadFixture reads and returns the contents of a fixture file.
func LoadFixture(t *testing.T, relativePath string) []byte {
	t.Helper()

	path := FixturePath(t, relativePath)
	data, err := os.ReadFile(path) // #nosec G304 - paths are controlled by tests
	if err != nil {
		t.Fatalf("Failed to read fixture %s: %v", relativePath, err)
	}

	return data
}

// LoadFixtureString reads and returns the contents of a fixture file as a string.
func LoadFixtureString(t *testing.T, relativePath string) string {
	t.Helper()
	return string(LoadFixture(t, relativePath))
}

// ConfigFixture returns the path to a configuration fixture.
func ConfigFixture(t *testing.T, name string) string {
	t.Helper()
	if !strings.HasSuffix(name, ".json") {
		name += ".qualhook.json"
	}
	return FixturePath(t, filepath.Join("configs", name))
}

// ProjectFixture returns the path to a project fixture directory.
func ProjectFixture(t *testing.T, projectType string) string {
	t.Helper()
	return FixturePath(t, filepath.Join("projects", projectType))
}

// OutputFixture returns the contents of an output fixture file.
func OutputFixture(t *testing.T, name string) string {
	t.Helper()
	if !strings.HasSuffix(name, ".txt") {
		name += ".txt"
	}
	return LoadFixtureString(t, filepath.Join("outputs", name))
}

// CreateTempFixture copies a fixture to a temporary directory and returns the path.
// The temporary directory is automatically cleaned up when the test completes.
func CreateTempFixture(t *testing.T, fixturePath string) string {
	t.Helper()

	// Create temporary directory
	tempDir := t.TempDir()

	// Get the source path
	srcPath := FixturePath(t, fixturePath)

	// Copy fixture to temp directory
	if err := copyDir(srcPath, tempDir); err != nil {
		t.Fatalf("Failed to copy fixture to temp dir: %v", err)
	}

	return tempDir
}

// copyDir recursively copies a directory.
func copyDir(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	// If source is a file, copy it directly
	if !srcInfo.IsDir() {
		srcData, err := os.ReadFile(src) // #nosec G304 - paths are controlled by tests
		if err != nil {
			return err
		}
		dstPath := filepath.Join(dst, filepath.Base(src))
		return os.WriteFile(dstPath, srcData, srcInfo.Mode())
	}

	// Create destination directory
	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return err
	}

	// Read directory contents
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	// Copy each entry
	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			srcData, err := os.ReadFile(srcPath) // #nosec G304 - paths are controlled by tests
			if err != nil {
				return err
			}
			info, err := entry.Info()
			if err != nil {
				return err
			}
			if err := os.WriteFile(dstPath, srcData, info.Mode()); err != nil {
				return err
			}
		}
	}

	return nil
}
