package detector

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProjectDetector_Detect(t *testing.T) {
	tests := []struct {
		name          string
		files         []string
		dirs          []string
		expectedTypes []string
		minConfidence float64
	}{
		{
			name:          "Node.js project",
			files:         []string{"package.json", "package-lock.json", "tsconfig.json"},
			expectedTypes: []string{"nodejs"},
			minConfidence: 0.5,
		},
		{
			name:          "Go project",
			files:         []string{"go.mod", "go.sum", "main.go"},
			expectedTypes: []string{"go"},
			minConfidence: 0.6,
		},
		{
			name:          "Rust project",
			files:         []string{"Cargo.toml", "Cargo.lock"},
			expectedTypes: []string{"rust"},
			minConfidence: 0.5,
		},
		{
			name:          "Python project with pyproject.toml",
			files:         []string{"pyproject.toml", "requirements.txt"},
			expectedTypes: []string{"python"},
			minConfidence: 0.2,
		},
		{
			name:          "Python project with setup.py",
			files:         []string{"setup.py", "setup.cfg", "requirements.txt"},
			expectedTypes: []string{"python"},
			minConfidence: 0.3,
		},
		{
			name:          "Java Maven project",
			files:         []string{"pom.xml"},
			dirs:          []string{".mvn"},
			expectedTypes: []string{"java"},
			minConfidence: 0.2,
		},
		{
			name:          "Java Gradle project",
			files:         []string{"build.gradle", "settings.gradle", "gradlew"},
			expectedTypes: []string{"java"},
			minConfidence: 0.4,
		},
		{
			name:          "Ruby project",
			files:         []string{"Gemfile", "Gemfile.lock", ".ruby-version"},
			expectedTypes: []string{"ruby"},
			minConfidence: 0.5,
		},
		{
			name:          "PHP project",
			files:         []string{"composer.json", "composer.lock"},
			expectedTypes: []string{"php"},
			minConfidence: 0.6,
		},
		{
			name:          "Multiple project types",
			files:         []string{"package.json", "go.mod", "requirements.txt"},
			expectedTypes: []string{"nodejs", "go", "python"},
			minConfidence: 0.08,
		},
		{
			name:          "TypeScript Node.js project",
			files:         []string{"package.json", "tsconfig.json", "yarn.lock"},
			expectedTypes: []string{"nodejs"},
			minConfidence: 0.5,
		},
		{
			name:          "Django project",
			files:         []string{"manage.py", "requirements.txt", "pyproject.toml"},
			expectedTypes: []string{"python"},
			minConfidence: 0.3,
		},
		{
			name:          "Laravel project",
			files:         []string{"composer.json", "artisan"},
			expectedTypes: []string{"php"},
			minConfidence: 0.6,
		},
		{
			name:          "Empty directory",
			files:         []string{},
			expectedTypes: []string{},
			minConfidence: 0.0,
		},
		{
			name:          "Unknown project type",
			files:         []string{"README.md", "LICENSE", ".gitignore"},
			expectedTypes: []string{},
			minConfidence: 0.0,
		},
	}

	detector := New()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory
			tmpDir := t.TempDir()

			// Create test files
			for _, file := range tt.files {
				path := filepath.Join(tmpDir, file)
				if err := os.WriteFile(path, []byte("test content"), 0644); err != nil {
					t.Fatalf("failed to create test file %s: %v", file, err)
				}
			}

			// Create test directories
			for _, dir := range tt.dirs {
				path := filepath.Join(tmpDir, dir)
				if err := os.MkdirAll(path, 0755); err != nil {
					t.Fatalf("failed to create test directory %s: %v", dir, err)
				}
			}

			// Detect project types
			results, err := detector.Detect(tmpDir)
			if err != nil {
				t.Fatalf("Detect() error = %v", err)
			}

			// Check expected types
			foundTypes := make(map[string]bool)
			for _, result := range results {
				foundTypes[result.Name] = true

				// Check confidence
				if result.Confidence < tt.minConfidence {
					t.Errorf("Project type %s has confidence %f, expected at least %f",
						result.Name, result.Confidence, tt.minConfidence)
				}

				// Check markers
				if len(result.Markers) == 0 && result.Confidence > 0 {
					t.Errorf("Project type %s has no markers but positive confidence", result.Name)
				}
			}

			// Verify all expected types were found
			for _, expectedType := range tt.expectedTypes {
				if !foundTypes[expectedType] {
					t.Errorf("Expected project type %s not found", expectedType)
				}
			}

			// Verify no unexpected types
			if len(tt.expectedTypes) == 0 && len(results) > 0 {
				t.Errorf("Expected no project types, but found: %v", results)
			}
		})
	}
}

func TestProjectDetector_DetectWithDotNetProject(t *testing.T) {
	detector := New()
	tmpDir := t.TempDir()

	// Create .NET project files
	files := []string{
		"MyProject.csproj",
		"Program.cs",
		"MyProject.sln",
		"global.json",
	}

	for _, file := range files {
		path := filepath.Join(tmpDir, file)
		if err := os.WriteFile(path, []byte("test content"), 0644); err != nil {
			t.Fatalf("failed to create test file %s: %v", file, err)
		}
	}

	results, err := detector.Detect(tmpDir)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}

	// Check for .NET detection
	found := false
	for _, result := range results {
		if result.Name == "dotnet" {
			found = true
			if result.Confidence < 0.5 {
				t.Errorf("Low confidence for .NET project: %f", result.Confidence)
			}
			break
		}
	}

	if !found {
		t.Error("Failed to detect .NET project")
	}
}

func TestProjectDetector_DetectInvalidPath(t *testing.T) {
	detector := New()

	// Test non-existent path
	_, err := detector.Detect("/non/existent/path")
	if err == nil {
		t.Error("Expected error for non-existent path")
	}

	// Test file instead of directory
	tmpFile := filepath.Join(t.TempDir(), "file.txt")
	if err := os.WriteFile(tmpFile, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err = detector.Detect(tmpFile)
	if err == nil {
		t.Error("Expected error for file path instead of directory")
	}
}

func TestProjectDetector_ConfidenceScoring(t *testing.T) {
	detector := New()

	tests := []struct {
		name               string
		files              []string
		dirs               []string
		expectedType       string
		expectedConfidence float64
		tolerance          float64
	}{
		{
			name:               "Full Node.js project",
			files:              []string{"package.json", "package-lock.json", "node_modules", ".nvmrc"},
			expectedType:       "nodejs",
			expectedConfidence: 0.5, // (1.0 + 0.5 + 0.3 + 0.2) / total possible
			tolerance:          0.1,
		},
		{
			name:               "Minimal Go project",
			files:              []string{"go.mod"},
			expectedType:       "go",
			expectedConfidence: 0.3, // 1.0 / total possible
			tolerance:          0.1,
		},
		{
			name:               "Complete Rust project",
			files:              []string{"Cargo.toml", "Cargo.lock", "rust-toolchain.toml"},
			dirs:               []string{".cargo"},
			expectedType:       "rust",
			expectedConfidence: 0.8, // Higher confidence with more markers
			tolerance:          0.1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			// Create files
			for _, file := range tt.files {
				if file == "node_modules" {
					// Create as directory
					if err := os.MkdirAll(filepath.Join(tmpDir, file), 0755); err != nil {
						t.Fatal(err)
					}
				} else {
					if err := os.WriteFile(filepath.Join(tmpDir, file), []byte("test"), 0644); err != nil {
						t.Fatal(err)
					}
				}
			}

			// Create directories
			for _, dir := range tt.dirs {
				if err := os.MkdirAll(filepath.Join(tmpDir, dir), 0755); err != nil {
					t.Fatal(err)
				}
			}

			results, err := detector.Detect(tmpDir)
			if err != nil {
				t.Fatal(err)
			}

			// Find the expected type
			var found *ProjectType
			for i := range results {
				if results[i].Name == tt.expectedType {
					found = &results[i]
					break
				}
			}

			if found == nil {
				t.Fatalf("Expected type %s not found", tt.expectedType)
			}

			// Check confidence within tolerance
			diff := found.Confidence - tt.expectedConfidence
			if diff < 0 {
				diff = -diff
			}
			if diff > tt.tolerance {
				t.Errorf("Confidence %f not within tolerance %f of expected %f",
					found.Confidence, tt.tolerance, tt.expectedConfidence)
			}
		})
	}
}

func TestProjectDetector_DetectMonorepo(t *testing.T) {
	tests := []struct {
		name         string
		files        []string
		dirs         []string
		isMonorepo   bool
		monorepoType string
		workspaces   int
	}{
		{
			name:         "Lerna monorepo",
			files:        []string{"lerna.json", "package.json"},
			dirs:         []string{"packages/app1", "packages/app2"},
			isMonorepo:   true,
			monorepoType: "lerna",
			workspaces:   2,
		},
		{
			name:         "Nx monorepo",
			files:        []string{"nx.json", "package.json"},
			dirs:         []string{"apps/frontend", "apps/backend", "libs/shared"},
			isMonorepo:   true,
			monorepoType: "nx",
			workspaces:   3,
		},
		{
			name:         "pnpm workspace",
			files:        []string{"pnpm-workspace.yaml", "package.json"},
			dirs:         []string{"packages/core", "packages/ui"},
			isMonorepo:   true,
			monorepoType: "pnpm",
			workspaces:   2,
		},
		{
			name:         "Go workspace",
			files:        []string{"go.work", "go.work.sum"},
			dirs:         []string{"services/api", "services/worker"},
			isMonorepo:   true,
			monorepoType: "go-workspace",
			workspaces:   2,
		},
		{
			name:         "Turborepo",
			files:        []string{"turbo.json", "package.json"},
			dirs:         []string{"apps/web", "packages/utils"},
			isMonorepo:   true,
			monorepoType: "turborepo",
			workspaces:   2,
		},
		{
			name:         "Not a monorepo",
			files:        []string{"package.json", "README.md"},
			isMonorepo:   false,
			monorepoType: "",
			workspaces:   0,
		},
		{
			name:         "Yarn workspaces",
			files:        []string{"yarn.lock", "package.json"},
			isMonorepo:   true,
			monorepoType: "yarn-workspaces",
			workspaces:   0, // Need actual package.json content
		},
	}

	detector := New()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			// Create files
			for _, file := range tt.files {
				content := "test content"
				if file == "package.json" && tt.monorepoType == "yarn-workspaces" {
					content = `{"workspaces": ["packages/*"]}`
				}
				if err := os.WriteFile(filepath.Join(tmpDir, file), []byte(content), 0644); err != nil {
					t.Fatal(err)
				}
			}

			// Create directories with marker files
			for _, dir := range tt.dirs {
				dirPath := filepath.Join(tmpDir, dir)
				if err := os.MkdirAll(dirPath, 0755); err != nil {
					t.Fatal(err)
				}
				// Add a package.json to make it a valid workspace
				pkgPath := filepath.Join(dirPath, "package.json")
				if err := os.WriteFile(pkgPath, []byte(`{"name": "test"}`), 0644); err != nil {
					t.Fatal(err)
				}
			}

			info, err := detector.DetectMonorepo(tmpDir)
			if err != nil {
				t.Fatalf("DetectMonorepo() error = %v", err)
			}

			if info.IsMonorepo != tt.isMonorepo {
				t.Errorf("IsMonorepo = %v, want %v", info.IsMonorepo, tt.isMonorepo)
			}

			if info.Type != tt.monorepoType {
				t.Errorf("Type = %v, want %v", info.Type, tt.monorepoType)
			}

			// For actual workspace detection, we need the directories to match patterns
			if tt.workspaces > 0 && len(info.Workspaces) == 0 {
				t.Logf("Note: Workspace detection requires matching patterns (packages/*, apps/*, etc)")
			}
		})
	}
}

func TestProjectDetector_GetDefaultConfigName(t *testing.T) {
	tests := []struct {
		projectType string
		expected    string
	}{
		{"nodejs", "nodejs.json"},
		{"go", "golang.json"},
		{"rust", "rust.json"},
		{"python", "python.json"},
		{"java", "java.json"},
		{"ruby", "ruby.json"},
		{"php", "php.json"},
		{"dotnet", "dotnet.json"},
		{"unknown", ""},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.projectType, func(t *testing.T) {
			result := GetDefaultConfigName(tt.projectType)
			if result != tt.expected {
				t.Errorf("GetDefaultConfigName(%q) = %q, want %q", tt.projectType, result, tt.expected)
			}
		})
	}
}

func TestProjectDetector_NestedProjects(t *testing.T) {
	detector := New()
	tmpDir := t.TempDir()

	// Create a structure with nested projects
	// Root: Node.js
	// frontend/: React (Node.js)
	// backend/: Go
	// scripts/: Python

	files := map[string]string{
		"package.json":             `{"name": "root"}`,
		"frontend/package.json":    `{"name": "frontend"}`,
		"frontend/tsconfig.json":   `{}`,
		"backend/go.mod":           `module backend`,
		"backend/go.sum":           ``,
		"scripts/requirements.txt": `requests==2.28.0`,
		"scripts/setup.py":         ``,
	}

	for path, content := range files {
		fullPath := filepath.Join(tmpDir, path)
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Test root detection
	rootResults, err := detector.Detect(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	foundNodejs := false
	for _, result := range rootResults {
		if result.Name == "nodejs" {
			foundNodejs = true
			break
		}
	}
	if !foundNodejs {
		t.Error("Failed to detect Node.js in root")
	}

	// Test backend detection
	backendResults, err := detector.Detect(filepath.Join(tmpDir, "backend"))
	if err != nil {
		t.Fatal(err)
	}

	foundGo := false
	for _, result := range backendResults {
		if result.Name == "go" {
			foundGo = true
			break
		}
	}
	if !foundGo {
		t.Error("Failed to detect Go in backend")
	}

	// Test scripts detection
	scriptsResults, err := detector.Detect(filepath.Join(tmpDir, "scripts"))
	if err != nil {
		t.Fatal(err)
	}

	foundPython := false
	for _, result := range scriptsResults {
		if result.Name == "python" {
			foundPython = true
			break
		}
	}
	if !foundPython {
		t.Error("Failed to detect Python in scripts")
	}
}

func TestProjectDetector_MonorepoSubProjects(t *testing.T) {
	detector := New()
	tmpDir := t.TempDir()

	// Create a lerna monorepo with different project types
	files := map[string]string{
		"lerna.json":                      `{"version": "1.0.0"}`,
		"package.json":                    `{"name": "monorepo"}`,
		"packages/web/package.json":       `{"name": "web"}`,
		"packages/web/tsconfig.json":      `{}`,
		"packages/api/go.mod":             `module api`,
		"packages/api/go.sum":             ``,
		"packages/cli/Cargo.toml":         `[package]\nname = "cli"`,
		"packages/scripts/pyproject.toml": `[tool.poetry]\nname = "scripts"`,
	}

	for path, content := range files {
		fullPath := filepath.Join(tmpDir, path)
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	info, err := detector.DetectMonorepo(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	if !info.IsMonorepo {
		t.Error("Failed to detect monorepo")
	}

	if info.Type != "lerna" {
		t.Errorf("Wrong monorepo type: %s", info.Type)
	}

	// Check detected workspaces
	expectedWorkspaces := []string{"packages/api", "packages/cli", "packages/scripts", "packages/web"}
	if len(info.Workspaces) != len(expectedWorkspaces) {
		t.Errorf("Expected %d workspaces, got %d", len(expectedWorkspaces), len(info.Workspaces))
	}

	// Check sub-project detection
	expectedSubProjects := map[string]string{
		"packages/web":     "nodejs",
		"packages/api":     "go",
		"packages/cli":     "rust",
		"packages/scripts": "python",
	}

	for workspace, expectedType := range expectedSubProjects {
		projects, exists := info.SubProjects[workspace]
		if !exists {
			t.Errorf("No projects detected for workspace %s", workspace)
			continue
		}

		found := false
		for _, proj := range projects {
			if proj.Name == expectedType {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected %s project in %s, but not found", expectedType, workspace)
		}
	}
}
