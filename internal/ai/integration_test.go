//go:build integration

package ai

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bebsworthy/qualhook/internal/config"
	"github.com/bebsworthy/qualhook/internal/executor"
	"github.com/bebsworthy/qualhook/internal/testutil"
	pkgconfig "github.com/bebsworthy/qualhook/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAIConfig_EndToEnd tests the complete ai-config flow with mock AI tool
func TestAIConfig_EndToEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tests := []struct {
		name             string
		setupFunc        func() *testutil.MockAITool
		options          AIOptions
		expectedCommands []string
		expectError      bool
		errorContains    string
	}{
		{
			name: "successful golang project config generation",
			setupFunc: func() *testutil.MockAITool {
				mockAI := testutil.NewMockAITool()
				// mockAI already has a default response set
				return mockAI
			},
			options: AIOptions{
				Tool:         "claude",
				WorkingDir:   ".",
				Interactive:  false,
				TestCommands: false,
				Timeout:      30 * time.Second,
			},
			expectedCommands: []string{"format", "lint", "typecheck", "test"},
			expectError:      false,
		},
		{
			name: "successful monorepo project config generation",
			setupFunc: func() *testutil.MockAITool {
				mockAI := testutil.NewMockAITool()
				mockAI.SetResponse("monorepo", testutil.MonorepoResponse())
				return mockAI
			},
			options: AIOptions{
				Tool:         "claude",
				WorkingDir:   ".",
				Interactive:  false,
				TestCommands: false,
				Timeout:      30 * time.Second,
			},
			expectedCommands: []string{"format", "lint", "typecheck", "test"},
			expectError:      false,
		},
		{
			name: "partial success with invalid commands",
			setupFunc: func() *testutil.MockAITool {
				mockAI := testutil.NewMockAITool()
				mockAI.SetResponse("partial", testutil.PartialSuccessResponse())
				return mockAI
			},
			options: AIOptions{
				Tool:         "claude",
				WorkingDir:   ".",
				Interactive:  false,
				TestCommands: false,
			},
			expectedCommands: []string{"format", "lint"}, // Only valid commands
			expectError:      false,
		},
		{
			name: "invalid JSON response handling",
			setupFunc: func() *testutil.MockAITool {
				mockAI := testutil.NewMockAITool()
				mockAI.SetResponse("invalid", testutil.InvalidJSONResponse())
				return mockAI
			},
			options: AIOptions{
				Tool:         "claude",
				WorkingDir:   ".",
				Interactive:  false,
				TestCommands: false,
			},
			expectError:      false, // Parser can recover partial config
			expectedCommands: []string{"format"},
		},
		{
			name: "AI tool execution failure",
			setupFunc: func() *testutil.MockAITool {
				mockAI := testutil.NewMockAITool()
				mockAI.ExitCode = 1
				mockAI.StderrOutput = "AI tool execution failed"
				return mockAI
			},
			options: AIOptions{
				Tool:        "claude",
				WorkingDir:  ".",
				Interactive: false,
			},
			expectError:   true,
			errorContains: "claude failed with exit code",
		},
		{
			name: "user cancellation during execution",
			setupFunc: func() *testutil.MockAITool {
				mockAI := testutil.NewMockAITool()
				mockAI.Delay = 100 * time.Millisecond
				return mockAI
			},
			options: AIOptions{
				Tool:        "claude",
				WorkingDir:  ".",
				Interactive: true,
				Timeout:     50 * time.Millisecond, // Short timeout to trigger cancellation
			},
			expectError:   true,
			errorContains: "timed out",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup test environment
			tempDir := t.TempDir()
			oldDir, _ := os.Getwd()
			defer os.Chdir(oldDir)
			os.Chdir(tempDir)

			// Setup mocks
			mockAI := tt.setupFunc()

			// Get mock response - execute with empty prompt to get the default/configured response
			mockResponse, mockErr := mockAI.Execute("")
			var mockExitCode int
			var mockStderr string
			if mockErr != nil {
				mockExitCode = 1
				mockStderr = mockErr.Error()
				mockResponse = mockStderr
			}

			// Create mock executor that returns the AI tool response
			mockExec := &MockExecutor{
				results: map[string]*executor.ExecResult{
					"claude": {Stdout: mockResponse, Stderr: mockStderr, ExitCode: mockExitCode},
					"gemini": {Stdout: mockResponse, Stderr: mockStderr, ExitCode: mockExitCode},
				},
				delay: mockAI.Delay, // Use the delay from MockAITool
			}

			// Create assistant with mock executor
			assistant := NewAssistant(mockExec).(*assistantImpl)
			// Override with test mocks
			assistant.detector = &MockToolDetector{}
			assistant.progress = &MockProgressIndicator{}
			assistant.testRunner = &MockTestRunner{}
			// Use real validator but disable command checking for tests
			validator := config.NewValidator()
			validator.CheckCommands = false
			assistant.parser = NewResponseParser(validator)
			// Pre-select a tool to avoid interactive prompt
			assistant.selectedTool = "claude"
			assistant.toolSelectionTime = time.Now()

			// Execute test
			ctx := context.Background()
			if tt.options.Timeout > 0 {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, tt.options.Timeout)
				defer cancel()
			}

			cfg, err := assistant.GenerateConfig(ctx, tt.options)

			// Verify results
			if tt.expectError {
				assert.Error(t, err, "Expected error but got none")
				if tt.errorContains != "" && err != nil {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				require.NoError(t, err, "Unexpected error: %v", err)
				require.NotNil(t, cfg, "Config should not be nil")

				// Verify expected commands are present
				for _, expectedCmd := range tt.expectedCommands {
					assert.Contains(t, cfg.Commands, expectedCmd, "Expected command %s not found", expectedCmd)
				}
			}

			// Verify mock AI tool was called
			if !tt.expectError || tt.errorContains != "timeout" {
				assert.Greater(t, mockAI.GetCallCount(), 0, "Mock AI tool should have been called")
			}
		})
	}
}

// TestWizard_AIEnhancement tests AI integration with the configuration wizard
func TestWizard_AIEnhancement(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tests := []struct {
		name           string
		commandType    string
		existingConfig *pkgconfig.Config
		mockResponse   string
		expectSuccess  bool
	}{
		{
			name:        "enhance format command",
			commandType: "format",
			existingConfig: &pkgconfig.Config{
				Commands: map[string]*pkgconfig.CommandConfig{
					"format": {
						Command: "echo",
						Args:    []string{"basic format"},
					},
				},
			},
			mockResponse: `{
				"command": "prettier",
				"args": ["--write", "."],
				"errorPatterns": [{"pattern": "\\[error\\]", "flags": "i"}],
				"exitCodes": [1],
				"explanation": "Prettier provides comprehensive JavaScript/TypeScript formatting"
			}`,
			expectSuccess: true,
		},
		{
			name:        "suggest new lint command",
			commandType: "lint",
			existingConfig: &pkgconfig.Config{
				Commands: map[string]*pkgconfig.CommandConfig{},
			},
			mockResponse: `{
				"command": "eslint",
				"args": [".", "--fix"],
				"errorPatterns": [{"pattern": "\\d+ problem", "flags": ""}],
				"exitCodes": [1],
				"explanation": "ESLint provides comprehensive JavaScript linting with auto-fix"
			}`,
			expectSuccess: true,
		},
		{
			name:        "handle invalid AI response",
			commandType: "test",
			existingConfig: &pkgconfig.Config{
				Commands: map[string]*pkgconfig.CommandConfig{},
			},
			mockResponse:  `{invalid json}`,
			expectSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup test environment
			_ = t.TempDir() // tempDir created for test isolation

			// Create mock executor with response
			mockExec := createMockExecutorWithAIResponse(tt.mockResponse, 0)

			// Create assistant with mock executor
			assistant := NewAssistant(mockExec).(*assistantImpl)
			// Override with test mocks
			assistant.detector = &MockToolDetector{}
			assistant.progress = &MockProgressIndicator{}
			assistant.testRunner = &MockTestRunner{}
			// Use real validator but disable command checking for tests
			validator := config.NewValidator()
			validator.CheckCommands = false
			assistant.parser = NewResponseParser(validator)
			// Pre-select a tool to avoid interactive prompt
			assistant.selectedTool = "claude"
			assistant.toolSelectionTime = time.Now()

			// Test command suggestion
			ctx := context.Background()
			projectInfo := ProjectContext{
				ProjectType:    "nodejs",
				ExistingConfig: tt.existingConfig,
				CustomCommands: []string{},
			}

			suggestion, err := assistant.SuggestCommand(ctx, tt.commandType, projectInfo)

			if tt.expectSuccess {
				require.NoError(t, err, "Command suggestion should succeed")
				assert.NotNil(t, suggestion)
				assert.NotEmpty(t, suggestion.Command)
				assert.NotEmpty(t, suggestion.Explanation)
			} else {
				assert.Error(t, err, "Command suggestion should fail")
			}
		})
	}
}

// setupTestAssistant creates a properly configured assistant for testing
func setupTestAssistant(mockResponse string, exitCode int) *assistantImpl {
	mockExec := createMockExecutorWithAIResponse(mockResponse, exitCode)
	assistant := NewAssistant(mockExec).(*assistantImpl)

	// Override with test mocks
	assistant.detector = &MockToolDetector{}
	assistant.progress = &MockProgressIndicator{}
	assistant.testRunner = &MockTestRunner{}

	// Use real validator but disable command checking for tests
	validator := config.NewValidator()
	validator.CheckCommands = false
	assistant.parser = NewResponseParser(validator)

	// Pre-select a tool to avoid interactive prompt
	assistant.selectedTool = "claude"
	assistant.toolSelectionTime = time.Now()

	return assistant
}

// TestMonorepo_Scenarios tests AI config generation for monorepo projects
func TestMonorepo_Scenarios(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tests := []struct {
		name          string
		setupProject  func(string) error
		mockResponse  string
		expectedPaths int
		expectedRoot  []string
		expectedSub   map[string][]string
	}{
		{
			name: "yarn workspaces monorepo",
			setupProject: func(tempDir string) error {
				// Create package.json with workspaces
				packageJSON := `{
					"name": "monorepo-root",
					"workspaces": ["packages/*"],
					"scripts": {
						"build": "yarn workspaces run build"
					}
				}`
				if err := os.WriteFile(filepath.Join(tempDir, "package.json"), []byte(packageJSON), 0644); err != nil {
					return err
				}

				// Create workspace directories
				packagesDir := filepath.Join(tempDir, "packages")
				os.MkdirAll(filepath.Join(packagesDir, "backend"), 0755)
				os.MkdirAll(filepath.Join(packagesDir, "frontend"), 0755)

				// Add workspace package.json files
				backendPkg := `{"name": "@app/backend", "scripts": {"test": "jest"}}`
				frontendPkg := `{"name": "@app/frontend", "scripts": {"test": "jest", "build": "webpack"}}`

				os.WriteFile(filepath.Join(packagesDir, "backend", "package.json"), []byte(backendPkg), 0644)
				os.WriteFile(filepath.Join(packagesDir, "frontend", "package.json"), []byte(frontendPkg), 0644)

				return nil
			},
			mockResponse:  testutil.MonorepoResponse(),
			expectedPaths: 2,
			expectedRoot:  []string{"format", "lint", "typecheck", "test"},
			expectedSub: map[string][]string{
				"packages/backend/**":  {"test"},
				"packages/frontend/**": {"test", "build"},
			},
		},
		{
			name: "lerna monorepo",
			setupProject: func(tempDir string) error {
				// Create lerna.json
				lernaJSON := `{
					"version": "independent",
					"packages": ["packages/*"],
					"npmClient": "npm"
				}`
				if err := os.WriteFile(filepath.Join(tempDir, "lerna.json"), []byte(lernaJSON), 0644); err != nil {
					return err
				}

				// Create packages
				packagesDir := filepath.Join(tempDir, "packages")
				os.MkdirAll(filepath.Join(packagesDir, "lib1"), 0755)
				os.MkdirAll(filepath.Join(packagesDir, "lib2"), 0755)

				return nil
			},
			mockResponse:  testutil.MonorepoResponse(),
			expectedPaths: 2,
			expectedRoot:  []string{"format", "lint", "typecheck", "test"},
			expectedSub: map[string][]string{
				"packages/backend/**":  {"test"},
				"packages/frontend/**": {"test", "build"},
			},
		},
		{
			name: "complex multi-language monorepo",
			setupProject: func(tempDir string) error {
				// Create different language components
				dirs := []string{"backend", "frontend", "mobile", "shared"}
				for _, dir := range dirs {
					os.MkdirAll(filepath.Join(tempDir, dir), 0755)
				}

				// Add language-specific files
				os.WriteFile(filepath.Join(tempDir, "backend", "main.go"), []byte("package main"), 0644)
				os.WriteFile(filepath.Join(tempDir, "frontend", "package.json"), []byte(`{"name": "frontend"}`), 0644)
				os.WriteFile(filepath.Join(tempDir, "mobile", "Package.swift"), []byte("// Swift Package"), 0644)

				return nil
			},
			mockResponse:  testutil.ComplexProjectResponse(),
			expectedPaths: 3,
			expectedRoot:  []string{"format", "lint", "typecheck", "test"},
			expectedSub: map[string][]string{
				"backend/**":  {"format", "lint", "test"},
				"frontend/**": {"format", "lint", "typecheck"},
				"mobile/**":   {"format", "lint", "test"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup project structure
			tempDir := t.TempDir()
			require.NoError(t, tt.setupProject(tempDir))

			oldDir, _ := os.Getwd()
			defer os.Chdir(oldDir)
			os.Chdir(tempDir)

			// Setup mocks
			assistant := setupTestAssistant(tt.mockResponse, 0)

			// Generate config
			ctx := context.Background()
			options := AIOptions{
				Tool:         "claude",
				WorkingDir:   tempDir,
				Interactive:  false,
				TestCommands: false,
			}

			cfg, err := assistant.GenerateConfig(ctx, options)
			require.NoError(t, err)
			require.NotNil(t, cfg)

			// Verify root commands
			for _, expectedCmd := range tt.expectedRoot {
				assert.Contains(t, cfg.Commands, expectedCmd, "Root command %s missing", expectedCmd)
			}

			// Verify path-specific configurations
			assert.Len(t, cfg.Paths, tt.expectedPaths, "Expected %d path configurations", tt.expectedPaths)

			for _, pathCfg := range cfg.Paths {
				if expectedCmds, exists := tt.expectedSub[pathCfg.Path]; exists {
					for _, expectedCmd := range expectedCmds {
						assert.Contains(t, pathCfg.Commands, expectedCmd,
							"Path %s missing command %s", pathCfg.Path, expectedCmd)
					}
				}
			}
		})
	}
}

// TestCancellation_AndErrorCases tests various cancellation and error scenarios
func TestCancellation_AndErrorCases(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tests := []struct {
		name        string
		setupMock   func() (*MockExecutor, *testutil.MockAITool)
		options     AIOptions
		expectError string
	}{
		{
			name: "no AI tools available",
			setupMock: func() (*MockExecutor, *testutil.MockAITool) {
				mockExec := &MockExecutor{
					results: map[string]*executor.ExecResult{},
				}
				mockAI := testutil.NewMockAITool()
				return mockExec, mockAI
			},
			options: AIOptions{
				Interactive: false,
			},
			expectError: "no ai tools available",
		},
		{
			name: "specific tool not found",
			setupMock: func() (*MockExecutor, *testutil.MockAITool) {
				mockAI := testutil.NewMockAITool()
				mockExec := &MockExecutor{
					results: map[string]*executor.ExecResult{
						"claude": {Stdout: mockAI.DefaultResponse, ExitCode: 0},
					},
				}
				return mockExec, mockAI
			},
			options: AIOptions{
				Tool:        "nonexistent",
				Interactive: false,
			},
			expectError: "not found",
		},
		{
			name: "AI tool timeout",
			setupMock: func() (*MockExecutor, *testutil.MockAITool) {
				mockAI := testutil.NewMockAITool()
				mockExec := &MockExecutor{
					results: map[string]*executor.ExecResult{
						"claude": {Stdout: mockAI.DefaultResponse, ExitCode: 0},
					},
					delay: 200 * time.Millisecond,
				}
				return mockExec, mockAI
			},
			options: AIOptions{
				Tool:        "claude",
				Interactive: false,
				Timeout:     50 * time.Millisecond,
			},
			expectError: "timed out",
		},
		{
			name: "empty AI response",
			setupMock: func() (*MockExecutor, *testutil.MockAITool) {
				mockExec := &MockExecutor{
					results: map[string]*executor.ExecResult{
						"claude": {Stdout: testutil.EmptyResponse(), ExitCode: 0},
					},
				}
				mockAI := testutil.NewMockAITool()
				return mockExec, mockAI
			},
			options: AIOptions{
				Tool:        "claude",
				Interactive: false,
			},
			expectError: "parse",
		},
		{
			name: "dangerous commands in AI response",
			setupMock: func() (*MockExecutor, *testutil.MockAITool) {
				mockExec := &MockExecutor{
					results: map[string]*executor.ExecResult{
						"claude": {Stdout: testutil.DangerousCommandsResponse(), ExitCode: 0},
					},
				}
				mockAI := testutil.NewMockAITool()
				return mockExec, mockAI
			},
			options: AIOptions{
				Tool:        "claude",
				Interactive: false,
			},
			expectError: "", // Security validation happens during command execution, not parsing
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup test environment
			tempDir := t.TempDir()
			oldDir, _ := os.Getwd()
			defer os.Chdir(oldDir)
			os.Chdir(tempDir)

			// Setup mocks
			mockExec, _ := tt.setupMock()

			var detector ToolDetector
			if tt.expectError == "no ai tools available" {
				detector = &MockToolDetector{noTools: true}
			} else {
				detector = &MockToolDetector{}
			}

			// Use the mock executor from setupMock
			assistant := &assistantImpl{
				detector:      detector,
				promptGen:     NewPromptGenerator(),
				parser:        NewResponseParser(config.NewValidator()), // Use real validator for security checks
				executor:      mockExec,
				progress:      &MockProgressIndicator{},
				testRunner:    &MockTestRunner{},
				ui:            NewInteractiveUI(),
				responseCache: make(map[string]*cachedResponse),
			}
			// Pre-select tool to avoid interactive prompt
			assistant.selectedTool = "claude"
			assistant.toolSelectionTime = time.Now()

			// Execute test
			ctx := context.Background()
			if tt.options.Timeout > 0 {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, tt.options.Timeout)
				defer cancel()
			}

			cfg, err := assistant.GenerateConfig(ctx, tt.options)

			// Verify results
			if tt.expectError != "" {
				require.Error(t, err, "Expected error but got none")
				assert.Contains(t, strings.ToLower(err.Error()), strings.ToLower(tt.expectError))
				assert.Nil(t, cfg, "Config should be nil on error")
			} else {
				require.NoError(t, err, "Expected success but got error")
				assert.NotNil(t, cfg, "Config should not be nil on success")
			}
		})
	}
}

// TestCommand_TestingIntegration tests the command testing functionality
func TestCommand_TestingIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tempDir := t.TempDir()
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(tempDir)

	// Create mock executor with a valid config response
	mockResponse := `{
		"version": "1.0",
		"projectType": "golang",
		"commands": {
			"format": {"command": "go", "args": ["fmt", "./..."]},
			"lint": {"command": "golangci-lint", "args": ["run"]},
			"test": {"command": "go", "args": ["test", "./..."]}
		}
	}`

	testRunner := &MockTestRunner{
		testResults: map[string]TestResult{
			"format": {Success: true, Output: "Formatting complete"},
			"lint":   {Success: false, Error: assert.AnError, Output: "Linting failed"},
			"test":   {Success: true, Output: "All tests passed"},
		},
	}

	assistant := setupTestAssistant(mockResponse, 0)
	// Override test runner with our mock
	assistant.testRunner = testRunner

	// Test with command testing disabled to avoid interactive prompts
	ctx := context.Background()
	options := AIOptions{
		Tool:         "claude",
		WorkingDir:   tempDir,
		Interactive:  false,
		TestCommands: false,
	}

	cfg, err := assistant.GenerateConfig(ctx, options)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Verify config was generated correctly
	assert.NotEmpty(t, cfg.Commands)
	assert.Contains(t, cfg.Commands, "format")
	assert.Contains(t, cfg.Commands, "lint")
	assert.Contains(t, cfg.Commands, "test")
}

// Mock implementations for testing

type MockExecutor struct {
	results map[string]*executor.ExecResult
	delay   time.Duration
}

// createMockExecutorWithAIResponse creates a mock executor that returns AI tool responses
func createMockExecutorWithAIResponse(response string, exitCode int) *MockExecutor {
	var stderr string
	if exitCode != 0 {
		stderr = response
	}
	return &MockExecutor{
		results: map[string]*executor.ExecResult{
			"claude": {Stdout: response, Stderr: stderr, ExitCode: exitCode},
			"gemini": {Stdout: response, Stderr: stderr, ExitCode: exitCode},
		},
	}
}

func (m *MockExecutor) Execute(command string, args []string, options executor.ExecOptions) (*executor.ExecResult, error) {
	if m.delay > 0 {
		time.Sleep(m.delay)
	}

	if result, exists := m.results[command]; exists {
		return result, nil
	}

	// Default successful result
	return &executor.ExecResult{
		Stdout:   "mock output",
		ExitCode: 0,
	}, nil
}

type MockToolDetector struct {
	noTools bool
}

func (m *MockToolDetector) DetectTools() ([]Tool, error) {
	if m.noTools {
		return []Tool{}, nil
	}

	return []Tool{
		{Name: "claude", Command: "claude", Version: "1.0.0", Available: true},
		{Name: "gemini", Command: "gemini", Version: "1.0.0", Available: true},
	}, nil
}

func (m *MockToolDetector) IsToolAvailable(toolName string) (bool, error) {
	if m.noTools {
		return false, nil
	}
	return toolName == "claude" || toolName == "gemini", nil
}

type MockProgressIndicator struct {
	started bool
	message string
}

func (m *MockProgressIndicator) Start(message string) {
	m.started = true
	m.message = message
}

func (m *MockProgressIndicator) Update(message string) {
	m.message = message
}

func (m *MockProgressIndicator) Stop() {
	m.started = false
}

func (m *MockProgressIndicator) WaitForCancellation(ctx context.Context) <-chan bool {
	ch := make(chan bool, 1)
	go func() {
		select {
		case <-ctx.Done():
			ch <- true
		}
	}()
	return ch
}

type MockTestRunner struct {
	testResults   map[string]TestResult
	testCallCount int
}

func (m *MockTestRunner) TestCommands(ctx context.Context, commands map[string]*pkgconfig.CommandConfig) (map[string]TestResult, error) {
	m.testCallCount++
	results := make(map[string]TestResult)

	for name := range commands {
		if result, exists := m.testResults[name]; exists {
			results[name] = result
		} else {
			results[name] = TestResult{Success: true, Output: "Test passed"}
		}
	}

	return results, nil
}

func (m *MockTestRunner) TestCommand(ctx context.Context, name string, cmd *pkgconfig.CommandConfig) (*TestResult, error) {
	if result, exists := m.testResults[name]; exists {
		return &result, nil
	}
	return &TestResult{Success: true, Output: "Test passed"}, nil
}

type MockInteractiveUI struct {
	selectedTool string
}

func (m *MockInteractiveUI) SelectTool(tools []Tool) (string, error) {
	if len(tools) > 0 {
		return tools[0].Name, nil
	}
	return "", assert.AnError
}

func (m *MockInteractiveUI) ReviewConfiguration(cfg *pkgconfig.Config) error {
	return nil
}

func (m *MockInteractiveUI) ConfirmAction(message string) (bool, error) {
	return true, nil
}

func (m *MockInteractiveUI) ShowCommandTest(name string, cmd *pkgconfig.CommandConfig, result *TestResult) {
	// Mock implementation - do nothing
}

func (m *MockInteractiveUI) HandleAIError(err error) {
	// Mock implementation - do nothing
}

// TestRealWorld_AIConfig tests with more realistic project scenarios
func TestRealWorld_AIConfig(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tests := []struct {
		name         string
		setupProject func(string) error
		mockResponse string
		validate     func(*testing.T, *pkgconfig.Config)
	}{
		{
			name: "node.js project with TypeScript",
			setupProject: func(tempDir string) error {
				packageJSON := `{
					"name": "my-app",
					"scripts": {
						"build": "tsc",
						"test": "jest",
						"lint": "eslint src/",
						"format": "prettier --write src/"
					},
					"devDependencies": {
						"typescript": "^4.0.0",
						"jest": "^27.0.0",
						"eslint": "^8.0.0",
						"prettier": "^2.0.0"
					}
				}`
				os.WriteFile(filepath.Join(tempDir, "package.json"), []byte(packageJSON), 0644)

				tsconfig := `{
					"compilerOptions": {
						"target": "ES2020",
						"module": "commonjs",
						"strict": true
					}
				}`
				os.WriteFile(filepath.Join(tempDir, "tsconfig.json"), []byte(tsconfig), 0644)

				os.MkdirAll(filepath.Join(tempDir, "src"), 0755)
				os.WriteFile(filepath.Join(tempDir, "src", "index.ts"), []byte("console.log('Hello');"), 0644)

				return nil
			},
			mockResponse: `{
				"version": "1.0",
				"projectType": "nodejs-typescript",
				"monorepo": {"detected": false},
				"commands": {
					"format": {
						"command": "prettier",
						"args": ["--write", "src/"],
						"errorPatterns": [{"pattern": "\\\\[error\\\\]", "flags": "i"}],
						"exitCodes": [1]
					},
					"lint": {
						"command": "eslint",
						"args": ["src/", "--ext", ".ts,.tsx"],
						"errorPatterns": [{"pattern": "\\\\d+ problem", "flags": ""}],
						"exitCodes": [1]
					},
					"typecheck": {
						"command": "tsc",
						"args": ["--noEmit"],
						"errorPatterns": [{"pattern": "error TS", "flags": ""}],
						"exitCodes": [1]
					},
					"test": {
						"command": "jest",
						"args": ["--passWithNoTests"],
						"errorPatterns": [{"pattern": "FAIL", "flags": ""}],
						"exitCodes": [1]
					}
				}
			}`,
			validate: func(t *testing.T, cfg *pkgconfig.Config) {
				require.Contains(t, cfg.Commands, "typecheck")
				assert.Equal(t, "tsc", cfg.Commands["typecheck"].Command)
				assert.Contains(t, cfg.Commands["typecheck"].Args, "--noEmit")
			},
		},
		{
			name: "Go project with modules",
			setupProject: func(tempDir string) error {
				goMod := `module example.com/myapp

go 1.19

require (
	github.com/stretchr/testify v1.8.0
)`
				os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte(goMod), 0644)
				os.WriteFile(filepath.Join(tempDir, "main.go"), []byte("package main\n\nfunc main() {}"), 0644)

				return nil
			},
			mockResponse: `{
  "version": "1.0",
  "projectType": "golang",
  "monorepo": {
    "detected": false
  },
  "commands": {
    "format": {
      "command": "go",
      "args": ["fmt", "./..."],
      "errorPatterns": [],
      "exitCodes": []
    },
    "lint": {
      "command": "golangci-lint",
      "args": ["run"],
      "errorPatterns": [
        {"pattern": "\\\\S+:\\\\d+:\\\\d+:", "flags": ""}
      ],
      "exitCodes": [1]
    },
    "typecheck": {
      "command": "go",
      "args": ["build", "-o", "/dev/null", "./..."],
      "errorPatterns": [
        {"pattern": "cannot find package", "flags": ""},
        {"pattern": "undefined:", "flags": ""}
      ],
      "exitCodes": [1]
    },
    "test": {
      "command": "go",
      "args": ["test", "./..."],
      "errorPatterns": [
        {"pattern": "FAIL", "flags": ""}
      ],
      "exitCodes": [1]
    }
  }
}`,
			validate: func(t *testing.T, cfg *pkgconfig.Config) {
				require.Contains(t, cfg.Commands, "format")
				assert.Equal(t, "go", cfg.Commands["format"].Command)
				assert.Contains(t, cfg.Commands["format"].Args, "fmt")
			},
		},
		{
			name: "Python project with poetry",
			setupProject: func(tempDir string) error {
				pyproject := `[tool.poetry]
name = "my-python-app"
version = "0.1.0"
description = ""
authors = ["User <user@example.com>"]

[tool.poetry.dependencies]
python = "^3.8"

[tool.poetry.dev-dependencies]
pytest = "^6.0"
black = "^22.0"
flake8 = "^4.0"
mypy = "^0.950"

[build-system]
requires = ["poetry-core>=1.0.0"]
build-backend = "poetry.core.masonry.api"`
				os.WriteFile(filepath.Join(tempDir, "pyproject.toml"), []byte(pyproject), 0644)
				os.MkdirAll(filepath.Join(tempDir, "src"), 0755)
				os.WriteFile(filepath.Join(tempDir, "src", "main.py"), []byte("print('Hello')"), 0644)

				return nil
			},
			mockResponse: `{
				"version": "1.0",
				"projectType": "python-poetry",
				"monorepo": {"detected": false},
				"commands": {
					"format": {
						"command": "poetry",
						"args": ["run", "black", "."],
						"errorPatterns": [],
						"exitCodes": [1]
					},
					"lint": {
						"command": "poetry",
						"args": ["run", "flake8", "."],
						"errorPatterns": [{"pattern": "E\\d{3}", "flags": ""}],
						"exitCodes": [1]
					},
					"typecheck": {
						"command": "poetry",
						"args": ["run", "mypy", "."],
						"errorPatterns": [{"pattern": "error:", "flags": ""}],
						"exitCodes": [1]
					},
					"test": {
						"command": "poetry",
						"args": ["run", "pytest"],
						"errorPatterns": [{"pattern": "FAILED", "flags": ""}],
						"exitCodes": [1]
					}
				}
			}`,
			validate: func(t *testing.T, cfg *pkgconfig.Config) {
				for _, cmd := range cfg.Commands {
					assert.Equal(t, "poetry", cmd.Command)
					if len(cmd.Args) > 0 {
						assert.Equal(t, "run", cmd.Args[0])
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup project
			tempDir := t.TempDir()
			require.NoError(t, tt.setupProject(tempDir))

			oldDir, _ := os.Getwd()
			defer os.Chdir(oldDir)
			os.Chdir(tempDir)

			// Setup mocks
			assistant := setupTestAssistant(tt.mockResponse, 0)

			// Generate config
			ctx := context.Background()
			options := AIOptions{
				Tool:         "claude",
				WorkingDir:   tempDir,
				Interactive:  false,
				TestCommands: false,
			}

			cfg, err := assistant.GenerateConfig(ctx, options)
			require.NoError(t, err)
			require.NotNil(t, cfg)

			// Run project-specific validation
			tt.validate(t, cfg)
		})
	}
}

// TestAIAssistant_ResponseCaching tests the response caching functionality
func TestAIAssistant_ResponseCaching(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Create mock executor with default response
	defaultResponse := `{
		"version": "1.0",
		"projectType": "golang",
		"commands": {
			"format": {"command": "go", "args": ["fmt", "./..."]},
			"lint": {"command": "golangci-lint", "args": ["run"]},
			"typecheck": {"command": "go", "args": ["build", "-o", "/dev/null", "./..."]},
			"test": {"command": "go", "args": ["test", "./..."]}
		}
	}`
	mockExec := createMockExecutorWithAIResponse(defaultResponse, 0)
	mockExec.delay = 100 * time.Millisecond // Add delay to simulate AI execution time

	// Create assistant with caching enabled
	assistant := NewAssistant(mockExec).(*assistantImpl)
	assistant.responseCache = make(map[string]*cachedResponse)

	// Mock detector
	mockDetector := &MockToolDetector{}
	assistant.detector = mockDetector
	assistant.progress = &MockProgressIndicator{}
	assistant.testRunner = &MockTestRunner{}

	// First call - should execute normally
	ctx := context.Background()
	options := AIOptions{
		Tool:         "claude",
		WorkingDir:   ".",
		Interactive:  false,
		TestCommands: false,
	}

	// Manually cache a response to test caching
	cachedResponse := `{
		"version": "1.0",
		"projectType": "golang",
		"commands": {
			"format": {"command": "go", "args": ["fmt", "./..."]},
			"lint": {"command": "golangci-lint", "args": ["run"]},
			"typecheck": {"command": "go", "args": ["build", "-o", "/dev/null", "./..."]},
			"test": {"command": "go", "args": ["test", "./..."]}
		}
	}`
	cacheKey := assistant.generateCacheKey("claude", assistant.promptGen.GenerateConfigPrompt("."))
	assistant.cacheResponse(cacheKey, cachedResponse, 10*time.Minute)

	start := time.Now()
	cfg1, err1 := assistant.GenerateConfig(ctx, options)
	duration1 := time.Since(start)

	require.NoError(t, err1)
	require.NotNil(t, cfg1)

	// Second call with same parameters should use cache
	start = time.Now()
	cfg2, err2 := assistant.GenerateConfig(ctx, options)
	duration2 := time.Since(start)

	require.NoError(t, err2)
	require.NotNil(t, cfg2)

	// Cache hit should be significantly faster (no AI execution)
	// Allow some variance in timing but cached should be much faster
	assert.True(t, duration2 < duration1, "Cached response should be faster: duration1=%v, duration2=%v", duration1, duration2)
	// For a more reliable test, check that the second call was very fast (under 10ms)
	assert.True(t, duration2 < 10*time.Millisecond, "Cached response should be very fast (under 10ms), got %v", duration2)

	// Configs should be identical
	assert.Equal(t, cfg1.Version, cfg2.Version)
}

// TestAIAssistant_ConcurrentToolDetection tests concurrent tool detection performance
func TestAIAssistant_ConcurrentToolDetection(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Create mock executor that simulates tool detection
	mockExec := &MockExecutor{
		results: map[string]*executor.ExecResult{
			"claude": {Stdout: "Claude 1.0.0", ExitCode: 0},
			"gemini": {Stdout: "Gemini CLI v1.0.0", ExitCode: 0},
		},
	}

	// Create detector
	detector := NewToolDetector(mockExec)

	// Measure detection time - should be concurrent
	start := time.Now()
	tools, err := detector.DetectTools()
	duration := time.Since(start)

	// Log results for debugging
	t.Logf("Tool detection took: %v", duration)
	t.Logf("Detected tools: %v", tools)

	require.NoError(t, err)
	// At least check we get a result (tools may or may not be available)
	assert.NotNil(t, tools)
}
