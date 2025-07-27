package ai

import (
	"strings"
	"testing"

	"github.com/bebsworthy/qualhook/internal/config"
	pkgconfig "github.com/bebsworthy/qualhook/pkg/config"
)

const (
	versionString   = "1.0"
	prettierCommand = "prettier"
)

func TestResponseParser_ParseConfigResponse(t *testing.T) {
	validator := config.NewValidator()
	validator.CheckCommands = false // Disable command existence checking for tests
	parser := NewResponseParser(validator)

	tests := []struct {
		name        string
		response    string
		wantErr     bool
		errType     AIErrorType
		validateCfg func(t *testing.T, cfg *pkgconfig.Config)
	}{
		{
			name: "valid_json_response",
			response: `{
				"version": "1.0",
				"projectType": "nodejs",
				"commands": {
					"format": {
						"command": "prettier",
						"args": ["--write", "."],
						"errorPatterns": [
							{"pattern": "\\[error\\]", "flags": "i"}
						],
						"exitCodes": [1, 2]
					},
					"lint": {
						"command": "eslint",
						"args": [".", "--fix"],
						"errorPatterns": [
							{"pattern": "\\d+ problems? \\(\\d+ errors?, \\d+ warnings?\\)", "flags": ""}
						],
						"exitCodes": [1]
					}
				}
			}`,
			wantErr: false,
			validateCfg: func(t *testing.T, cfg *pkgconfig.Config) {
				if cfg.Version != versionString {
					t.Errorf("expected version 1.0, got %s", cfg.Version)
				}
				if cfg.ProjectType != "nodejs" {
					t.Errorf("expected projectType nodejs, got %s", cfg.ProjectType)
				}
				if len(cfg.Commands) != 2 {
					t.Errorf("expected 2 commands, got %d", len(cfg.Commands))
				}
				if cmd, ok := cfg.Commands["format"]; ok {
					if cmd.Command != prettierCommand {
						t.Errorf("expected prettier command, got %s", cmd.Command)
					}
					if len(cmd.Args) != 2 {
						t.Errorf("expected 2 args, got %d", len(cmd.Args))
					}
				} else {
					t.Error("format command not found")
				}
			},
		},
		{
			name:     "json_in_code_block",
			response: "Here's the configuration for your project:\n\n```json\n{\n\t\"version\": \"1.0\",\n\t\"projectType\": \"go\",\n\t\"commands\": {\n\t\t\"format\": {\n\t\t\t\"command\": \"gofmt\",\n\t\t\t\"args\": [\"-w\", \".\"],\n\t\t\t\"exitCodes\": [1]\n\t\t}\n\t}\n}\n```\n\nThis configuration will format your Go code.",
			wantErr:  false,
			validateCfg: func(t *testing.T, cfg *pkgconfig.Config) {
				if cfg.ProjectType != "go" {
					t.Errorf("expected projectType go, got %s", cfg.ProjectType)
				}
				if cmd, ok := cfg.Commands["format"]; ok {
					if cmd.Command != "gofmt" {
						t.Errorf("expected gofmt command, got %s", cmd.Command)
					}
				} else {
					t.Error("format command not found")
				}
			},
		},
		{
			name: "monorepo_configuration",
			response: `{
				"version": "1.0",
				"projectType": "nodejs",
				"monorepo": {
					"detected": true,
					"type": "yarn-workspaces",
					"workspaces": ["packages/backend", "packages/frontend"]
				},
				"commands": {
					"format": {
						"command": "prettier",
						"args": ["--write", "."]
					}
				},
				"paths": [
					{
						"path": "packages/backend/**",
						"commands": {
							"test": {
								"command": "jest",
								"args": ["--config", "packages/backend/jest.config.js"]
							}
						}
					}
				]
			}`,
			wantErr: false,
			validateCfg: func(t *testing.T, cfg *pkgconfig.Config) {
				if len(cfg.Paths) != 1 {
					t.Errorf("expected 1 path config, got %d", len(cfg.Paths))
				}
				if cfg.Paths[0].Path != "packages/backend/**" {
					t.Errorf("expected path packages/backend/**, got %s", cfg.Paths[0].Path)
				}
			},
		},
		{
			name: "custom_commands",
			response: `{
				"version": "1.0",
				"projectType": "nodejs",
				"commands": {
					"format": {
						"command": "prettier",
						"args": ["--write", "."]
					}
				},
				"customCommands": {
					"build": {
						"command": "npm",
						"args": ["run", "build"],
						"explanation": "Build the project"
					}
				}
			}`,
			wantErr: false,
			validateCfg: func(t *testing.T, cfg *pkgconfig.Config) {
				if _, ok := cfg.Commands["build"]; !ok {
					t.Error("custom build command not found")
				}
			},
		},
		{
			name: "partial_json_recovery",
			response: `{
				"version": "1.0",
				"commands": {
					"format": {
						"command": "prettier",
						"args": ["--write", "."]
					},
					"lint": {
						"command": "eslint"`, // Missing closing braces
			wantErr: false,
			validateCfg: func(t *testing.T, cfg *pkgconfig.Config) {
				// Should recover with fixed JSON
				if cfg.Version != versionString {
					t.Errorf("expected version 1.0, got %s", cfg.Version)
				}
				if _, ok := cfg.Commands["format"]; !ok {
					t.Error("format command not recovered")
				}
			},
		},
		{
			name:     "invalid_json_no_recovery",
			response: `This is not JSON at all, just plain text`,
			wantErr:  true,
			errType:  ErrTypeResponseInvalid,
		},
		{
			name:     "empty_response",
			response: "",
			wantErr:  true,
			errType:  ErrTypeResponseInvalid,
		},
		{
			name: "dangerous_command",
			response: `{
				"version": "1.0",
				"commands": {
					"danger": {
						"command": "rm",
						"args": ["-rf", "/"]
					}
				}
			}`,
			wantErr: true,
			errType: ErrTypeValidationFailed,
		},
		{
			name: "invalid_regex_pattern",
			response: `{
				"version": "1.0",
				"commands": {
					"lint": {
						"command": "eslint",
						"errorPatterns": [
							{"pattern": "[invalid(regex", "flags": ""}
						]
					}
				}
			}`,
			wantErr: false, // Should skip invalid patterns but not fail
			validateCfg: func(t *testing.T, cfg *pkgconfig.Config) {
				if cmd, ok := cfg.Commands["lint"]; ok {
					if len(cmd.ErrorPatterns) != 0 {
						t.Errorf("expected invalid patterns to be skipped, got %d patterns", len(cmd.ErrorPatterns))
					}
				}
			},
		},
		{
			name: "default_exit_codes",
			response: `{
				"version": "1.0",
				"commands": {
					"test": {
						"command": "npm",
						"args": ["test"]
					}
				}
			}`,
			wantErr: false,
			validateCfg: func(t *testing.T, cfg *pkgconfig.Config) {
				if cmd, ok := cfg.Commands["test"]; ok {
					if len(cmd.ExitCodes) == 0 {
						t.Error("expected default exit codes to be set")
					}
					if cmd.ExitCodes[0] != 1 {
						t.Errorf("expected default exit code 1, got %d", cmd.ExitCodes[0])
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := parser.ParseConfigResponse(tt.response)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got none")
				} else if aiErr, ok := err.(*AIError); ok {
					if aiErr.Type != tt.errType {
						t.Errorf("expected error type %v, got %v", tt.errType, aiErr.Type)
					}
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				} else if tt.validateCfg != nil {
					tt.validateCfg(t, cfg)
				}
			}
		})
	}
}

func TestResponseParser_ParseCommandResponse(t *testing.T) {
	validator := config.NewValidator()
	validator.CheckCommands = false // Disable command existence checking for tests
	parser := NewResponseParser(validator)

	tests := []struct {
		name     string
		response string
		wantErr  bool
		validate func(t *testing.T, cmd *CommandSuggestion)
	}{
		{
			name: "valid_command_json",
			response: `{
				"command": "prettier",
				"args": ["--write", "."],
				"errorPatterns": [
					{"pattern": "\\[error\\]", "flags": "i"}
				],
				"exitCodes": [1, 2],
				"explanation": "Format code with Prettier"
			}`,
			wantErr: false,
			validate: func(t *testing.T, cmd *CommandSuggestion) {
				if cmd.Command != prettierCommand {
					t.Errorf("expected prettier, got %s", cmd.Command)
				}
				if len(cmd.Args) != 2 {
					t.Errorf("expected 2 args, got %d", len(cmd.Args))
				}
				if cmd.Explanation != "Format code with Prettier" {
					t.Errorf("unexpected explanation: %s", cmd.Explanation)
				}
			},
		},
		{
			name: "simple_command_text",
			response: `To format your code, run the following command:

prettier --write .

This will format all files in the current directory.`,
			wantErr: false,
			validate: func(t *testing.T, cmd *CommandSuggestion) {
				if cmd.Command != prettierCommand {
					t.Errorf("expected prettier, got %s", cmd.Command)
				}
				if len(cmd.Args) != 2 || cmd.Args[0] != "--write" || cmd.Args[1] != "." {
					t.Errorf("unexpected args: %v", cmd.Args)
				}
			},
		},
		{
			name:     "command_in_code_block",
			response: "Run this command:\n```\nnpm test\n```",
			wantErr:  false,
			validate: func(t *testing.T, cmd *CommandSuggestion) {
				if cmd.Command != "npm" {
					t.Errorf("expected npm, got %s", cmd.Command)
				}
				if len(cmd.Args) != 1 || cmd.Args[0] != "test" {
					t.Errorf("unexpected args: %v", cmd.Args)
				}
			},
		},
		{
			name:     "no_command_found",
			response: "This response contains no commands at all",
			wantErr:  true,
		},
		{
			name:     "dangerous_command",
			response: `{"command": "rm", "args": ["-rf", "/"]}`,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, err := parser.ParseCommandResponse(tt.response)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				} else if tt.validate != nil {
					tt.validate(t, cmd)
				}
			}
		})
	}
}

func TestResponseParser_extractJSON(t *testing.T) {
	parser := &ResponseParserImpl{}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "json_in_code_block",
			input:    "```json\n{\"test\": true}\n```",
			expected: `{"test": true}`,
		},
		{
			name:     "json_in_plain_code_block",
			input:    "```\n{\"test\": true}\n```",
			expected: `{"test": true}`,
		},
		{
			name:     "raw_json",
			input:    `{"test": true}`,
			expected: `{"test": true}`,
		},
		{
			name:     "json_with_text",
			input:    "Here's the config:\n{\"test\": true}\nThat's it!",
			expected: `{"test": true}`,
		},
		{
			name:     "nested_json",
			input:    `{"outer": {"inner": "value"}}`,
			expected: `{"outer": {"inner": "value"}}`,
		},
		{
			name:     "no_json",
			input:    "This is just plain text",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.extractJSON(tt.input)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestResponseParser_fixCommonJSONIssues(t *testing.T) {
	parser := &ResponseParserImpl{}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "trailing_comma_in_array",
			input:    `["a", "b",]`,
			expected: `["a", "b"]`,
		},
		{
			name:     "trailing_comma_in_object",
			input:    `{"a": 1, "b": 2,}`,
			expected: `{"a": 1, "b": 2}`,
		},
		{
			name:     "missing_closing_brace",
			input:    `{"a": 1`,
			expected: `{"a": 1}`,
		},
		{
			name:     "missing_closing_bracket",
			input:    `["a", "b"`,
			expected: `["a", "b"]`,
		},
		{
			name:     "multiple_issues",
			input:    `{"arr": ["a", "b",], "obj": {"x": 1,}`,
			expected: `{"arr": ["a", "b"], "obj": {"x": 1}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.fixCommonJSONIssues(tt.input)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestResponseParser_recoverPartialResponse(t *testing.T) {
	parser := &ResponseParserImpl{}

	tests := []struct {
		name      string
		input     string
		wantErr   bool
		checkResp func(t *testing.T, resp *aiResponse)
	}{
		{
			name: "recover_commands_only",
			input: `{
				"commands": {
					"format": {"command": "prettier", "args": ["--write", "."]}
				}
			}`,
			wantErr: false,
			checkResp: func(t *testing.T, resp *aiResponse) {
				if resp.Version != versionString {
					t.Error("expected default version 1.0")
				}
				if len(resp.Commands) != 1 {
					t.Errorf("expected 1 command, got %d", len(resp.Commands))
				}
			},
		},
		{
			name: "extract_commands_from_broken_json",
			input: `{
				"version": "1.0",
				"projectType": "nodejs",
				"commands": {
					"format": {"command": "prettier", "args": ["--write", "."]},
					"lint": {"command": "eslint"}
				}`,
			wantErr: false,
			checkResp: func(t *testing.T, resp *aiResponse) {
				if resp.Version != versionString {
					t.Errorf("expected version 1.0, got %s", resp.Version)
				}
				if len(resp.Commands) < 2 {
					t.Errorf("expected at least 2 commands, got %d", len(resp.Commands))
				}
			},
		},
		{
			name:    "completely_invalid",
			input:   "not json at all",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := parser.recoverPartialResponse(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				} else if tt.checkResp != nil {
					tt.checkResp(t, resp)
				}
			}
		})
	}
}

func TestResponseParser_attemptAutoFix(t *testing.T) {
	validator := config.NewValidator()
	validator.CheckCommands = false // Disable command existence checking for tests
	parser := &ResponseParserImpl{validator: validator}

	tests := []struct {
		name        string
		cfg         *pkgconfig.Config
		err         error
		shouldFix   bool
		validateFix func(t *testing.T, fixed *pkgconfig.Config)
	}{
		{
			name: "fix_negative_timeout",
			cfg: &pkgconfig.Config{
				Version: "1.0",
				Commands: map[string]*pkgconfig.CommandConfig{
					"test": {
						Command: "npm",
						Args:    []string{"test"},
						Timeout: -1000,
					},
				},
			},
			err:       validator.Validate(&pkgconfig.Config{}), // Mock error with "timeout"
			shouldFix: true,
			validateFix: func(t *testing.T, fixed *pkgconfig.Config) {
				if fixed.Commands["test"].Timeout != 0 {
					t.Errorf("expected timeout to be fixed to 0, got %d", fixed.Commands["test"].Timeout)
				}
			},
		},
		{
			name: "fix_excessive_timeout",
			cfg: &pkgconfig.Config{
				Version: "1.0",
				Commands: map[string]*pkgconfig.CommandConfig{
					"test": {
						Command: "npm",
						Args:    []string{"test"},
						Timeout: 9999999,
					},
				},
			},
			err:       validator.Validate(&pkgconfig.Config{}),
			shouldFix: true,
			validateFix: func(t *testing.T, fixed *pkgconfig.Config) {
				if fixed.Commands["test"].Timeout != 3600000 {
					t.Errorf("expected timeout to be capped at 3600000, got %d", fixed.Commands["test"].Timeout)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock error that contains the keywords we're looking for
			mockErr := strings.Builder{}
			if strings.Contains(tt.name, "timeout") {
				mockErr.WriteString("timeout validation failed")
			} else if strings.Contains(tt.name, "regex") {
				mockErr.WriteString("invalid regex pattern")
			}

			fixed := parser.attemptAutoFix(tt.cfg, &AIError{Message: mockErr.String()})

			switch {
			case tt.shouldFix && fixed == nil:
				t.Error("expected fix but got nil")
			case !tt.shouldFix && fixed != nil:
				t.Error("expected no fix but got one")
			case fixed != nil && tt.validateFix != nil:
				tt.validateFix(t, fixed)
			}
		})
	}
}

func TestResponseParser_looksLikeCommand(t *testing.T) {
	parser := &ResponseParserImpl{}

	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"npm_command", "npm test", true},
		{"yarn_command", "yarn build", true},
		{"go_command", "go test ./...", true},
		{"prettier_command", "prettier --write .", true},
		{"eslint_command", "eslint .", true},
		{"not_a_command", "this is not a command", false},
		{"empty_string", "", false},
		{"just_text", "hello world", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.looksLikeCommand(tt.input)
			if result != tt.expected {
				t.Errorf("expected %v for %q, got %v", tt.expected, tt.input, result)
			}
		})
	}
}

func TestResponseParser_EdgeCases(t *testing.T) {
	validator := config.NewValidator()
	parser := NewResponseParser(validator)

	t.Run("empty_commands_map", func(t *testing.T) {
		response := `{"version": "1.0", "commands": {}}`
		cfg, err := parser.ParseConfigResponse(response)
		if err == nil {
			t.Error("expected error for empty commands")
		}
		if cfg != nil {
			t.Error("expected nil config for empty commands")
		}
	})

	t.Run("null_values", func(t *testing.T) {
		response := `{
			"version": "1.0",
			"commands": {
				"test": {
					"command": "npm",
					"args": null,
					"errorPatterns": null
				}
			}
		}`
		cfg, err := parser.ParseConfigResponse(response)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if cfg != nil && cfg.Commands["test"] != nil {
			if cfg.Commands["test"].Args != nil {
				t.Error("expected nil args")
			}
			if cfg.Commands["test"].ErrorPatterns != nil {
				t.Error("expected nil error patterns")
			}
		}
	})

	t.Run("unicode_in_explanation", func(t *testing.T) {
		response := `{
			"command": "npm",
			"args": ["test"],
			"explanation": "Run tests 🧪 with coverage 📊"
		}`
		cmd, err := parser.ParseCommandResponse(response)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if cmd != nil && !strings.Contains(cmd.Explanation, "🧪") {
			t.Error("expected unicode to be preserved")
		}
	})

	t.Run("very_long_pattern", func(t *testing.T) {
		longPattern := strings.Repeat("a", 1000)
		response := `{
			"version": "1.0",
			"commands": {
				"test": {
					"command": "npm",
					"args": ["test"],
					"errorPatterns": [{"pattern": "` + longPattern + `"}]
				}
			}
		}`
		_, err := parser.ParseConfigResponse(response)
		if err == nil {
			t.Error("expected error for very long pattern")
		}
	})
}
