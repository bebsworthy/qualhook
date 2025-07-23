package hook_test

import (
	"fmt"
	"strings"

	"github.com/bebsworthy/qualhook/internal/hook"
)

func ExampleParser_Parse() {
	// Example JSON input from Claude Code
	input := `{
		"session_id": "abc123",
		"transcript_path": "/tmp/claude-transcript.json",
		"cwd": "/home/user/myproject",
		"hook_event_name": "before_tool_use",
		"tool_use": {
			"name": "Edit",
			"input": {
				"file_path": "/home/user/myproject/main.go",
				"old_string": "fmt.Println(\"Hello\")",
				"new_string": "fmt.Println(\"Hello, World!\")"
			}
		}
	}`

	parser := hook.NewParser()
	hookInput, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Session ID: %s\n", hookInput.SessionID)
	fmt.Printf("Working Directory: %s\n", hookInput.CWD)
	fmt.Printf("Hook Event: %s\n", hookInput.HookEventName)
	
	// Extract edited files
	files, err := parser.ExtractEditedFiles(hookInput)
	if err != nil {
		fmt.Printf("Error extracting files: %v\n", err)
		return
	}
	
	fmt.Printf("Edited files: %v\n", files)
	// Output:
	// Session ID: abc123
	// Working Directory: /home/user/myproject
	// Hook Event: before_tool_use
	// Edited files: [/home/user/myproject/main.go]
}

func ExampleParser_ExtractAllEditedFiles() {
	// Multiple hook inputs from a Claude Code session
	inputs := []*hook.HookInput{
		{
			SessionID:     "session1",
			CWD:           "/home/user/project",
			HookEventName: "before_tool_use",
			ToolUse: &hook.ToolUse{
				Name:  "Edit",
				Input: []byte(`{"file_path": "/home/user/project/main.go"}`),
			},
		},
		{
			SessionID:     "session1",
			CWD:           "/home/user/project",
			HookEventName: "before_tool_use",
			ToolUse: &hook.ToolUse{
				Name:  "Write",
				Input: []byte(`{"file_path": "/home/user/project/config.json"}`),
			},
		},
		{
			SessionID:     "session1",
			CWD:           "/home/user/project",
			HookEventName: "before_tool_use",
			ToolUse: &hook.ToolUse{
				Name:  "Edit",
				Input: []byte(`{"file_path": "/home/user/project/main.go"}`), // Duplicate
			},
		},
	}

	parser := hook.NewParser()
	files, err := parser.ExtractAllEditedFiles(inputs)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("All edited files (deduplicated): %v\n", files)
	// Output:
	// All edited files (deduplicated): [/home/user/project/main.go /home/user/project/config.json]
}