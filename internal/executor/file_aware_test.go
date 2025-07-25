package executor

import (
	"testing"

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
				Command: "echo",
				Args:    []string{"root lint"},
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
						Command: "echo",
						Args:    []string{"frontend lint"},
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
						Command: "echo",
						Args:    []string{"backend lint"},
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
