//go:build unit

package hook

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestNewParser(t *testing.T) {
	parser := NewParser()
	if parser == nil {
		t.Fatal("NewParser() returned nil")
	}
}

func TestParser_Parse(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    *HookInput
		wantErr bool
	}{
		{
			name: "valid input with Edit tool",
			input: `{
				"session_id": "test-session",
				"transcript_path": "/tmp/transcript.json",
				"cwd": "/home/user/project",
				"hook_event_name": "before_tool_use",
				"tool_use": {
					"name": "Edit",
					"input": {"file_path": "/home/user/project/main.go"}
				}
			}`,
			want: &HookInput{
				SessionID:      "test-session",
				TranscriptPath: "/tmp/transcript.json",
				CWD:            "/home/user/project",
				HookEventName:  "before_tool_use",
				ToolUse: &ToolUse{
					Name:  "Edit",
					Input: json.RawMessage(`{"file_path": "/home/user/project/main.go"}`),
				},
			},
			wantErr: false,
		},
		{
			name: "valid input without tool_use",
			input: `{
				"session_id": "test-session",
				"transcript_path": "/tmp/transcript.json",
				"cwd": "/home/user/project",
				"hook_event_name": "after_tool_use"
			}`,
			want: &HookInput{
				SessionID:      "test-session",
				TranscriptPath: "/tmp/transcript.json",
				CWD:            "/home/user/project",
				HookEventName:  "after_tool_use",
				ToolUse:        nil,
			},
			wantErr: false,
		},
		{
			name: "invalid JSON",
			input: `{
				"session_id": "test-session",
				"transcript_path": "/tmp/transcript.json"
				invalid json
			}`,
			want:    nil,
			wantErr: true,
		},
		{
			name: "missing session_id",
			input: `{
				"transcript_path": "/tmp/transcript.json",
				"cwd": "/home/user/project",
				"hook_event_name": "before_tool_use"
			}`,
			want:    nil,
			wantErr: true,
		},
		{
			name: "missing cwd",
			input: `{
				"session_id": "test-session",
				"transcript_path": "/tmp/transcript.json",
				"hook_event_name": "before_tool_use"
			}`,
			want:    nil,
			wantErr: true,
		},
		{
			name: "missing hook_event_name",
			input: `{
				"session_id": "test-session",
				"transcript_path": "/tmp/transcript.json",
				"cwd": "/home/user/project"
			}`,
			want:    nil,
			wantErr: true,
		},
		{
			name: "empty session_id",
			input: `{
				"session_id": "",
				"transcript_path": "/tmp/transcript.json",
				"cwd": "/home/user/project",
				"hook_event_name": "before_tool_use"
			}`,
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser()
			got, err := p.Parse(strings.NewReader(tt.input))
			if (err != nil) != tt.wantErr {
				t.Errorf("Parser.Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !equalHookInput(got, tt.want) {
				t.Errorf("Parser.Parse() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParser_ParseJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   []byte
		want    *HookInput
		wantErr bool
	}{
		{
			name: "valid JSON",
			input: []byte(`{
				"session_id": "test-session",
				"transcript_path": "/tmp/transcript.json",
				"cwd": "/home/user/project",
				"hook_event_name": "before_tool_use"
			}`),
			want: &HookInput{
				SessionID:      "test-session",
				TranscriptPath: "/tmp/transcript.json",
				CWD:            "/home/user/project",
				HookEventName:  "before_tool_use",
			},
			wantErr: false,
		},
		{
			name:    "invalid JSON",
			input:   []byte(`{"invalid": json}`),
			want:    nil,
			wantErr: true,
		},
		{
			name:    "empty input",
			input:   []byte{},
			want:    nil,
			wantErr: true,
		},
		{
			name:    "null input",
			input:   []byte("null"),
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser()
			got, err := p.ParseJSON(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parser.ParseJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !equalHookInput(got, tt.want) {
				t.Errorf("Parser.ParseJSON() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParser_ExtractEditedFiles(t *testing.T) {
	tests := []struct {
		name    string
		input   *HookInput
		want    []string
		wantErr bool
	}{
		{
			name:    "nil input",
			input:   nil,
			want:    nil,
			wantErr: true,
		},
		{
			name: "no tool use",
			input: &HookInput{
				SessionID:     "test",
				CWD:           "/home/user/project",
				HookEventName: "before_tool_use",
			},
			want:    []string{},
			wantErr: false,
		},
		{
			name: "Edit tool with file path",
			input: &HookInput{
				SessionID:     "test",
				CWD:           "/home/user/project",
				HookEventName: "before_tool_use",
				ToolUse: &ToolUse{
					Name:  "Edit",
					Input: json.RawMessage(`{"file_path": "/home/user/project/main.go", "old_string": "foo", "new_string": "bar"}`),
				},
			},
			want:    []string{"/home/user/project/main.go"},
			wantErr: false,
		},
		{
			name: "Write tool with file path",
			input: &HookInput{
				SessionID:     "test",
				CWD:           "/home/user/project",
				HookEventName: "before_tool_use",
				ToolUse: &ToolUse{
					Name:  "Write",
					Input: json.RawMessage(`{"file_path": "/home/user/project/new.go", "content": "package main"}`),
				},
			},
			want:    []string{"/home/user/project/new.go"},
			wantErr: false,
		},
		{
			name: "MultiEdit tool with file path",
			input: &HookInput{
				SessionID:     "test",
				CWD:           "/home/user/project",
				HookEventName: "before_tool_use",
				ToolUse: &ToolUse{
					Name:  "MultiEdit",
					Input: json.RawMessage(`{"file_path": "/home/user/project/config.go", "edits": []}`),
				},
			},
			want:    []string{"/home/user/project/config.go"},
			wantErr: false,
		},
		{
			name: "case insensitive tool name",
			input: &HookInput{
				SessionID:     "test",
				CWD:           "/home/user/project",
				HookEventName: "before_tool_use",
				ToolUse: &ToolUse{
					Name:  "edit",
					Input: json.RawMessage(`{"file_path": "/home/user/project/main.go"}`),
				},
			},
			want:    []string{"/home/user/project/main.go"},
			wantErr: false,
		},
		{
			name: "unknown tool type",
			input: &HookInput{
				SessionID:     "test",
				CWD:           "/home/user/project",
				HookEventName: "before_tool_use",
				ToolUse: &ToolUse{
					Name:  "UnknownTool",
					Input: json.RawMessage(`{"some_field": "value"}`),
				},
			},
			want:    []string{},
			wantErr: false,
		},
		{
			name: "Edit tool with empty file path",
			input: &HookInput{
				SessionID:     "test",
				CWD:           "/home/user/project",
				HookEventName: "before_tool_use",
				ToolUse: &ToolUse{
					Name:  "Edit",
					Input: json.RawMessage(`{"file_path": "", "old_string": "foo", "new_string": "bar"}`),
				},
			},
			want:    []string{},
			wantErr: false,
		},
		{
			name: "Edit tool with missing file path",
			input: &HookInput{
				SessionID:     "test",
				CWD:           "/home/user/project",
				HookEventName: "before_tool_use",
				ToolUse: &ToolUse{
					Name:  "Edit",
					Input: json.RawMessage(`{"old_string": "foo", "new_string": "bar"}`),
				},
			},
			want:    []string{},
			wantErr: false,
		},
		{
			name: "invalid tool input JSON",
			input: &HookInput{
				SessionID:     "test",
				CWD:           "/home/user/project",
				HookEventName: "before_tool_use",
				ToolUse: &ToolUse{
					Name:  "Edit",
					Input: json.RawMessage(`invalid json`),
				},
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser()
			got, err := p.ExtractEditedFiles(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parser.ExtractEditedFiles() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !equalStringSlices(got, tt.want) {
				t.Errorf("Parser.ExtractEditedFiles() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParser_ExtractAllEditedFiles(t *testing.T) {
	tests := []struct {
		name    string
		inputs  []*HookInput
		want    []string
		wantErr bool
	}{
		{
			name:    "empty inputs",
			inputs:  []*HookInput{},
			want:    []string{},
			wantErr: false,
		},
		{
			name:    "nil inputs",
			inputs:  nil,
			want:    []string{},
			wantErr: false,
		},
		{
			name: "single input with file",
			inputs: []*HookInput{
				{
					SessionID:     "test",
					CWD:           "/home/user/project",
					HookEventName: "before_tool_use",
					ToolUse: &ToolUse{
						Name:  "Edit",
						Input: json.RawMessage(`{"file_path": "/home/user/project/main.go"}`),
					},
				},
			},
			want:    []string{"/home/user/project/main.go"},
			wantErr: false,
		},
		{
			name: "multiple inputs with different files",
			inputs: []*HookInput{
				{
					SessionID:     "test1",
					CWD:           "/home/user/project",
					HookEventName: "before_tool_use",
					ToolUse: &ToolUse{
						Name:  "Edit",
						Input: json.RawMessage(`{"file_path": "/home/user/project/main.go"}`),
					},
				},
				{
					SessionID:     "test2",
					CWD:           "/home/user/project",
					HookEventName: "before_tool_use",
					ToolUse: &ToolUse{
						Name:  "Write",
						Input: json.RawMessage(`{"file_path": "/home/user/project/config.go"}`),
					},
				},
			},
			want:    []string{"/home/user/project/main.go", "/home/user/project/config.go"},
			wantErr: false,
		},
		{
			name: "duplicate files are deduplicated",
			inputs: []*HookInput{
				{
					SessionID:     "test1",
					CWD:           "/home/user/project",
					HookEventName: "before_tool_use",
					ToolUse: &ToolUse{
						Name:  "Edit",
						Input: json.RawMessage(`{"file_path": "/home/user/project/main.go"}`),
					},
				},
				{
					SessionID:     "test2",
					CWD:           "/home/user/project",
					HookEventName: "before_tool_use",
					ToolUse: &ToolUse{
						Name:  "Edit",
						Input: json.RawMessage(`{"file_path": "/home/user/project/main.go"}`),
					},
				},
			},
			want:    []string{"/home/user/project/main.go"},
			wantErr: false,
		},
		{
			name: "mixed valid and no tool use",
			inputs: []*HookInput{
				{
					SessionID:     "test1",
					CWD:           "/home/user/project",
					HookEventName: "before_tool_use",
					ToolUse: &ToolUse{
						Name:  "Edit",
						Input: json.RawMessage(`{"file_path": "/home/user/project/main.go"}`),
					},
				},
				{
					SessionID:     "test2",
					CWD:           "/home/user/project",
					HookEventName: "before_tool_use",
					ToolUse:       nil,
				},
			},
			want:    []string{"/home/user/project/main.go"},
			wantErr: false,
		},
		{
			name: "error in one input",
			inputs: []*HookInput{
				{
					SessionID:     "test1",
					CWD:           "/home/user/project",
					HookEventName: "before_tool_use",
					ToolUse: &ToolUse{
						Name:  "Edit",
						Input: json.RawMessage(`invalid json`),
					},
				},
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser()
			got, err := p.ExtractAllEditedFiles(tt.inputs)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parser.ExtractAllEditedFiles() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !equalStringSlices(got, tt.want) {
				t.Errorf("Parser.ExtractAllEditedFiles() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Helper functions

func equalHookInput(a, b *HookInput) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if a.SessionID != b.SessionID || a.TranscriptPath != b.TranscriptPath ||
		a.CWD != b.CWD || a.HookEventName != b.HookEventName {
		return false
	}
	if (a.ToolUse == nil) != (b.ToolUse == nil) {
		return false
	}
	if a.ToolUse != nil && b.ToolUse != nil {
		if a.ToolUse.Name != b.ToolUse.Name {
			return false
		}
		// Compare JSON content
		var aJSON, bJSON interface{}
		if err := json.Unmarshal(a.ToolUse.Input, &aJSON); err != nil {
			return false
		}
		if err := json.Unmarshal(b.ToolUse.Input, &bJSON); err != nil {
			return false
		}
		aBytes, _ := json.Marshal(aJSON)
		bBytes, _ := json.Marshal(bJSON)
		return bytes.Equal(aBytes, bBytes)
	}
	return true
}

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
