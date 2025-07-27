package testutil

import (
	"strings"
	"testing"
	"time"
)

func TestMockAITool_Execute(t *testing.T) {
	t.Run("default response", func(t *testing.T) {
		mock := NewMockAITool()

		response, err := mock.Execute("analyze my project")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Should return valid JSON
		if !strings.Contains(response, `"projectType": "golang"`) {
			t.Errorf("expected golang project type in default response")
		}
	})

	t.Run("keyword matching", func(t *testing.T) {
		mock := NewMockAITool()
		mock.SetResponse("monorepo", MonorepoResponse())

		response, err := mock.Execute("analyze this monorepo project")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Should return monorepo response
		if !strings.Contains(response, `"detected": true`) {
			t.Errorf("expected monorepo detection in response")
		}
		if !strings.Contains(response, `"type": "yarn-workspaces"`) {
			t.Errorf("expected yarn-workspaces in response")
		}
	})

	t.Run("error on prompt", func(t *testing.T) {
		mock := NewMockAITool()
		mock.ErrorOnPrompt = "trigger-error"

		_, err := mock.Execute("please trigger-error now")
		if err == nil {
			t.Fatal("expected error but got none")
		}
		if !strings.Contains(err.Error(), "mock AI error") {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("non-zero exit code", func(t *testing.T) {
		mock := NewMockAITool()
		mock.ExitCode = 1
		mock.StderrOutput = "AI tool failed"

		response, err := mock.Execute("analyze project")
		if err == nil {
			t.Fatal("expected error but got none")
		}
		if !strings.Contains(err.Error(), "exit status 1") {
			t.Errorf("unexpected error message: %v", err)
		}
		if response != "AI tool failed" {
			t.Errorf("expected stderr output, got: %s", response)
		}
	})

	t.Run("delay simulation", func(t *testing.T) {
		mock := NewMockAITool()
		mock.Delay = 50 * time.Millisecond

		start := time.Now()
		_, err := mock.Execute("analyze project")
		duration := time.Since(start)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if duration < 50*time.Millisecond {
			t.Errorf("expected delay of at least 50ms, got %v", duration)
		}
	})

	t.Run("call tracking", func(t *testing.T) {
		mock := NewMockAITool()

		// Initial state
		if mock.GetCallCount() != 0 {
			t.Errorf("expected initial call count 0, got %d", mock.GetCallCount())
		}

		// First call
		mock.Execute("first prompt")
		if mock.GetCallCount() != 1 {
			t.Errorf("expected call count 1, got %d", mock.GetCallCount())
		}
		if mock.GetLastPrompt() != "first prompt" {
			t.Errorf("expected last prompt 'first prompt', got %s", mock.GetLastPrompt())
		}

		// Second call
		mock.Execute("second prompt")
		if mock.GetCallCount() != 2 {
			t.Errorf("expected call count 2, got %d", mock.GetCallCount())
		}
		if mock.GetLastPrompt() != "second prompt" {
			t.Errorf("expected last prompt 'second prompt', got %s", mock.GetLastPrompt())
		}

		// Reset
		mock.Reset()
		if mock.GetCallCount() != 0 {
			t.Errorf("expected call count 0 after reset, got %d", mock.GetCallCount())
		}
		if mock.GetLastPrompt() != "" {
			t.Errorf("expected empty last prompt after reset, got %s", mock.GetLastPrompt())
		}
	})

	t.Run("case insensitive keyword matching", func(t *testing.T) {
		mock := NewMockAITool()
		mock.SetResponse("PYTHON", PartialSuccessResponse())

		response, err := mock.Execute("analyze this python project")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Should match despite case difference
		if !strings.Contains(response, `"projectType": "python"`) {
			t.Errorf("expected python project type in response")
		}
	})

	t.Run("multiple keywords", func(t *testing.T) {
		mock := NewMockAITool()
		mock.SetResponse("complex", ComplexProjectResponse())
		mock.SetResponse("simple", MinimalResponse())

		// Test complex
		response, err := mock.Execute("analyze this complex project")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(response, `"projectType": "mixed"`) {
			t.Errorf("expected mixed project type for complex")
		}

		// Test simple
		response, err = mock.Execute("analyze this simple project")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(response, `"projectType": "unknown"`) {
			t.Errorf("expected unknown project type for simple")
		}
	})
}

func TestPredefinedResponses(t *testing.T) {
	tests := []struct {
		name     string
		response string
		checks   []string
	}{
		{
			name:     "MonorepoResponse",
			response: MonorepoResponse(),
			checks: []string{
				`"detected": true`,
				`"type": "yarn-workspaces"`,
				`"workspaces": ["packages/backend"`,
				`"customCommands"`,
			},
		},
		{
			name:     "PartialSuccessResponse",
			response: PartialSuccessResponse(),
			checks: []string{
				`"projectType": "python"`,
				`"command": "black"`,
				`"command": "nonexistent-typechecker"`,
				`"test": null`,
			},
		},
		{
			name:     "InvalidJSONResponse",
			response: InvalidJSONResponse(),
			checks: []string{
				`// This comment makes the JSON invalid`,
			},
		},
		{
			name:     "DangerousCommandsResponse",
			response: DangerousCommandsResponse(),
			checks: []string{
				`"command": "rm"`,
				`"args": ["-rf", "/"]`,
				`"command": "curl"`,
				`http://malicious.com`,
			},
		},
		{
			name:     "ComplexProjectResponse",
			response: ComplexProjectResponse(),
			checks: []string{
				`"projectType": "mixed"`,
				`"workspaces": ["backend", "frontend", "mobile", "shared"]`,
				`"path": "backend/**"`,
				`"path": "frontend/**"`,
				`"path": "mobile/**"`,
				`"command": "swift-format"`,
				`"command": "xcodebuild"`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, check := range tt.checks {
				if !strings.Contains(tt.response, check) {
					t.Errorf("expected %q in response", check)
				}
			}
		})
	}
}

func TestMockAITool_ThreadSafety(t *testing.T) {
	mock := NewMockAITool()
	mock.SetResponse("test", "test response")

	// Run concurrent operations
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(_ int) {
			prompt := "test prompt"
			mock.Execute(prompt)
			mock.GetCallCount()
			mock.GetLastPrompt()
			mock.SetResponse("key", "value")
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Should have executed without race conditions
	if mock.GetCallCount() != 10 {
		t.Errorf("expected 10 calls, got %d", mock.GetCallCount())
	}
}
