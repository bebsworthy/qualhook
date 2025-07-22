// Package hook provides Claude Code hook integration functionality for qualhook.
package hook

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// HookInput represents the JSON input from Claude Code hooks
type HookInput struct {
	// Session ID for the Claude Code session
	SessionID string `json:"session_id"`
	// Path to the transcript file
	TranscriptPath string `json:"transcript_path"`
	// Current working directory
	CWD string `json:"cwd"`
	// Name of the hook event
	HookEventName string `json:"hook_event_name"`
	// Tool use information (optional)
	ToolUse *ToolUse `json:"tool_use,omitempty"`
}

// ToolUse represents tool usage information from Claude Code
type ToolUse struct {
	// Name of the tool (Edit, Write, MultiEdit, etc.)
	Name string `json:"name"`
	// Input parameters for the tool
	Input json.RawMessage `json:"input"`
}

// EditToolInput represents the input for Edit tool
type EditToolInput struct {
	FilePath string `json:"file_path"`
	// Other fields are not needed for file extraction
}

// WriteToolInput represents the input for Write tool
type WriteToolInput struct {
	FilePath string `json:"file_path"`
	// Other fields are not needed for file extraction
}

// MultiEditToolInput represents the input for MultiEdit tool
type MultiEditToolInput struct {
	FilePath string `json:"file_path"`
	// Other fields are not needed for file extraction
}

// Parser handles parsing of Claude Code hook input
type Parser struct{}

// NewParser creates a new hook input parser
func NewParser() *Parser {
	return &Parser{}
}

// Parse parses the JSON input from Claude Code
func (p *Parser) Parse(r io.Reader) (*HookInput, error) {
	var input HookInput
	decoder := json.NewDecoder(r)
	if err := decoder.Decode(&input); err != nil {
		return nil, fmt.Errorf("failed to parse hook input: %w", err)
	}

	// Validate required fields
	if input.SessionID == "" {
		return nil, fmt.Errorf("session_id is required")
	}
	if input.CWD == "" {
		return nil, fmt.Errorf("cwd is required")
	}
	if input.HookEventName == "" {
		return nil, fmt.Errorf("hook_event_name is required")
	}

	return &input, nil
}

// ParseJSON parses JSON input from a byte slice
func (p *Parser) ParseJSON(data []byte) (*HookInput, error) {
	var input HookInput
	if err := json.Unmarshal(data, &input); err != nil {
		return nil, fmt.Errorf("failed to parse hook input: %w", err)
	}

	// Validate required fields
	if input.SessionID == "" {
		return nil, fmt.Errorf("session_id is required")
	}
	if input.CWD == "" {
		return nil, fmt.Errorf("cwd is required")
	}
	if input.HookEventName == "" {
		return nil, fmt.Errorf("hook_event_name is required")
	}

	return &input, nil
}

// ExtractEditedFiles extracts file paths from the tool_use field
func (p *Parser) ExtractEditedFiles(input *HookInput) ([]string, error) {
	if input == nil {
		return nil, fmt.Errorf("input cannot be nil")
	}

	// If no tool use, return empty list
	if input.ToolUse == nil {
		return []string{}, nil
	}

	// Extract file path based on tool type
	var filePath string
	switch strings.ToLower(input.ToolUse.Name) {
	case "edit":
		var editInput EditToolInput
		if err := json.Unmarshal(input.ToolUse.Input, &editInput); err != nil {
			return nil, fmt.Errorf("failed to parse Edit tool input: %w", err)
		}
		filePath = editInput.FilePath

	case "write":
		var writeInput WriteToolInput
		if err := json.Unmarshal(input.ToolUse.Input, &writeInput); err != nil {
			return nil, fmt.Errorf("failed to parse Write tool input: %w", err)
		}
		filePath = writeInput.FilePath

	case "multiedit":
		var multiEditInput MultiEditToolInput
		if err := json.Unmarshal(input.ToolUse.Input, &multiEditInput); err != nil {
			return nil, fmt.Errorf("failed to parse MultiEdit tool input: %w", err)
		}
		filePath = multiEditInput.FilePath

	default:
		// Unknown tool type, return empty list
		return []string{}, nil
	}

	// Return file path if found
	if filePath != "" {
		return []string{filePath}, nil
	}

	return []string{}, nil
}

// ExtractAllEditedFiles extracts all edited files from multiple hook inputs
func (p *Parser) ExtractAllEditedFiles(inputs []*HookInput) ([]string, error) {
	fileMap := make(map[string]bool)
	var files []string

	for _, input := range inputs {
		editedFiles, err := p.ExtractEditedFiles(input)
		if err != nil {
			return nil, fmt.Errorf("failed to extract files from input: %w", err)
		}

		// Deduplicate files
		for _, file := range editedFiles {
			if !fileMap[file] {
				fileMap[file] = true
				files = append(files, file)
			}
		}
	}

	return files, nil
}