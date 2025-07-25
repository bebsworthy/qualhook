package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// TestCommandStructure tests the command structure without execution
func TestCommandStructure(t *testing.T) {
	tests := []struct {
		name       string
		cmd        *cobra.Command
		wantUse    string
		wantShort  string
		hasExample bool
	}{
		{
			name:       "format command",
			cmd:        formatCmd,
			wantUse:    "format [files...]",
			wantShort:  "Run the configured formatting command",
			hasExample: true,
		},
		{
			name:       "lint command",
			cmd:        lintCmd,
			wantUse:    "lint [files...]",
			wantShort:  "Run the configured linting command",
			hasExample: true,
		},
		{
			name:       "typecheck command",
			cmd:        typecheckCmd,
			wantUse:    "typecheck [files...]",
			wantShort:  "Run the configured type checking command",
			hasExample: true,
		},
		{
			name:       "test command",
			cmd:        testCmd,
			wantUse:    "test [test-files-or-patterns...]",
			wantShort:  "Run the configured test command",
			hasExample: true,
		},
		{
			name:       "config command",
			cmd:        configCmd,
			wantUse:    "config",
			wantShort:  "Configure qualhook for your project",
			hasExample: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.cmd.Use != tt.wantUse {
				t.Errorf("Use = %q, want %q", tt.cmd.Use, tt.wantUse)
			}
			if tt.cmd.Short != tt.wantShort {
				t.Errorf("Short = %q, want %q", tt.cmd.Short, tt.wantShort)
			}
			if tt.hasExample && tt.cmd.Example == "" && tt.cmd.Long == "" {
				t.Error("Expected command to have examples")
			}
			if tt.cmd.RunE == nil {
				t.Error("Expected command to have RunE function")
			}
		})
	}
}

// TestCommandHelp tests that help text is properly formatted
func TestCommandHelp(t *testing.T) {
	commands := []*cobra.Command{
		formatCmd,
		lintCmd,
		typecheckCmd,
		testCmd,
		configCmd,
	}

	for _, cmd := range commands {
		t.Run(cmd.Use, func(t *testing.T) {
			var buf bytes.Buffer
			cmd.SetOut(&buf)
			cmd.SetErr(&buf)
			cmd.SetArgs([]string{"--help"})

			// Help should not error
			err := cmd.Help()
			if err != nil {
				t.Errorf("Help() error = %v", err)
			}

			help := buf.String()

			// Check for key sections
			if !strings.Contains(help, "Usage:") {
				t.Error("Help should contain Usage section")
			}
			if !strings.Contains(help, cmd.Short) {
				t.Error("Help should contain short description")
			}
			if cmd.Long != "" && !strings.Contains(help, cmd.Long) {
				t.Error("Help should contain long description")
			}
			if cmd.Example != "" && !strings.Contains(help, "Examples:") {
				t.Error("Help should contain Examples section")
			}
		})
	}
}

// TestConfigCommandFlags tests the config command flags
func TestConfigCommandFlags(t *testing.T) {
	// Test that validate flag exists
	validateFlag := configCmd.Flags().Lookup("validate")
	if validateFlag == nil {
		t.Error("Config command should have --validate flag")
		return
	}
	if validateFlag.DefValue != "false" {
		t.Errorf("--validate flag default should be false, got %s", validateFlag.DefValue)
	}
}

// TestTemplateCommandSubcommands tests template subcommands
func TestTemplateCommandSubcommands(t *testing.T) {
	// Check that template command has subcommands
	if !templateCmd.HasSubCommands() {
		t.Error("Template command should have subcommands")
	}

	// Check for specific subcommands
	subcommands := []string{"list", "export", "import"}
	for _, sub := range subcommands {
		found := false
		for _, cmd := range templateCmd.Commands() {
			// Check if the Use field starts with the subcommand name
			if strings.HasPrefix(cmd.Use, sub) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Template command should have %q subcommand", sub)
		}
	}
}
