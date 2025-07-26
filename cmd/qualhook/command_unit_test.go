//go:build unit

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
		hasRunE    bool
	}{
		{
			name:       "format command",
			cmd:        formatCmd,
			wantUse:    "format [files...]",
			wantShort:  "Run the configured formatting command",
			hasExample: true,
			hasRunE:    true,
		},
		{
			name:       "lint command",
			cmd:        lintCmd,
			wantUse:    "lint [files...]",
			wantShort:  "Run the configured linting command",
			hasExample: true,
			hasRunE:    true,
		},
		{
			name:       "typecheck command",
			cmd:        typecheckCmd,
			wantUse:    "typecheck [files...]",
			wantShort:  "Run the configured type checking command",
			hasExample: true,
			hasRunE:    true,
		},
		{
			name:       "test command",
			cmd:        testCmd,
			wantUse:    "test [test-files-or-patterns...]",
			wantShort:  "Run the configured test command",
			hasExample: true,
			hasRunE:    true,
		},
		{
			name:       "config command",
			cmd:        configCmd,
			wantUse:    "config",
			wantShort:  "Configure qualhook for your project",
			hasExample: true,
			hasRunE:    true,
		},
		{
			name:       "template command",
			cmd:        templateCmd,
			wantUse:    "template",
			wantShort:  "Manage configuration templates",
			hasExample: false,
			hasRunE:    false, // Parent command, doesn't have RunE
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
				t.Error("Expected command to have examples or long description")
			}
			if tt.hasRunE && tt.cmd.RunE == nil {
				t.Error("Expected command to have RunE function")
			}
		})
	}
}

// TestCommandHelp tests that help text is properly formatted
func TestCommandHelp(t *testing.T) {
	tests := []struct {
		name     string
		cmd      *cobra.Command
		contains []string
	}{
		{
			name: "format help",
			cmd:  formatCmd,
			contains: []string{
				"Usage:",
				"Run the configured formatting command",
				"Examples:",
			},
		},
		{
			name: "lint help",
			cmd:  lintCmd,
			contains: []string{
				"Usage:",
				"Run the configured linting command",
				"Examples:",
			},
		},
		{
			name: "typecheck help",
			cmd:  typecheckCmd,
			contains: []string{
				"Usage:",
				"Run the configured type checking command",
				"Examples:",
			},
		},
		{
			name: "test help",
			cmd:  testCmd,
			contains: []string{
				"Usage:",
				"Run the configured test command",
				"Examples:",
			},
		},
		{
			name: "config help",
			cmd:  configCmd,
			contains: []string{
				"Usage:",
				"Configure qualhook for your project",
				"Examples:",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			tt.cmd.SetOut(&buf)
			tt.cmd.SetErr(&buf)
			tt.cmd.SetArgs([]string{"--help"})

			// Help should not error
			err := tt.cmd.Help()
			if err != nil {
				t.Errorf("Help() error = %v", err)
			}

			help := buf.String()

			// Check for key sections
			for _, expected := range tt.contains {
				if !strings.Contains(help, expected) {
					t.Errorf("Help should contain %q", expected)
				}
			}
		})
	}
}

// TestCommandFlags tests command-specific flags
func TestCommandFlags(t *testing.T) {
	tests := []struct {
		name     string
		cmd      *cobra.Command
		flagName string
		defValue string
		usage    string
	}{
		{
			name:     "config validate flag",
			cmd:      configCmd,
			flagName: "validate",
			defValue: "false",
			usage:    "Validate existing configuration",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := tt.cmd.Flags().Lookup(tt.flagName)
			if flag == nil {
				t.Errorf("%s command should have --%s flag", tt.cmd.Use, tt.flagName)
				return
			}
			if flag.DefValue != tt.defValue {
				t.Errorf("--%s flag default should be %s, got %s", tt.flagName, tt.defValue, flag.DefValue)
			}
			if !strings.Contains(flag.Usage, tt.usage) {
				t.Errorf("--%s flag usage should contain %q, got %q", tt.flagName, tt.usage, flag.Usage)
			}
		})
	}
}

// TestTemplateSubcommands tests template subcommands structure
func TestTemplateSubcommands(t *testing.T) {
	// Check that template command has subcommands
	if !templateCmd.HasSubCommands() {
		t.Error("Template command should have subcommands")
	}

	// Check for specific subcommands
	subcommands := []struct {
		name      string
		use       string
		shortDesc string
	}{
		{
			name:      "list",
			use:       "list",
			shortDesc: "List available templates",
		},
		{
			name:      "export",
			use:       "export",
			shortDesc: "Export current configuration as a template",
		},
		{
			name:      "import",
			use:       "import",
			shortDesc: "Import a configuration template",
		},
	}

	for _, sub := range subcommands {
		t.Run(sub.name+" subcommand", func(t *testing.T) {
			found := false
			for _, cmd := range templateCmd.Commands() {
				if strings.HasPrefix(cmd.Use, sub.use) {
					found = true
					if cmd.Short != sub.shortDesc {
						t.Errorf("Expected short description %q, got %q", sub.shortDesc, cmd.Short)
					}
					break
				}
			}
			if !found {
				t.Errorf("Template command should have %q subcommand", sub.name)
			}
		})
	}
}


// TestRootCommandStructure tests the root command structure
func TestRootCommandStructure(t *testing.T) {
	rootCmd := newRootCmd()

	// Test basic properties
	if rootCmd.Use != "qualhook" {
		t.Errorf("Root command Use = %q, want %q", rootCmd.Use, "qualhook")
	}

	if !strings.Contains(rootCmd.Short, "Quality checks for Claude Code") {
		t.Errorf("Root command short description should be 'Quality checks for Claude Code', got: %s", rootCmd.Short)
	}

	// Test persistent flags
	debugFlag := rootCmd.PersistentFlags().Lookup("debug")
	if debugFlag == nil {
		t.Error("Root command should have --debug flag")
	}

	configFlag := rootCmd.PersistentFlags().Lookup("config")
	if configFlag == nil {
		t.Error("Root command should have --config flag")
	}

	// Test that all expected subcommands are present
	expectedCommands := []string{"format", "lint", "typecheck", "test", "config", "template"}
	for _, cmdName := range expectedCommands {
		t.Run("has "+cmdName+" command", func(t *testing.T) {
			found := false
			for _, cmd := range rootCmd.Commands() {
				if cmd.Use == cmdName || strings.HasPrefix(cmd.Use, cmdName+" ") {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Root command should have %s subcommand", cmdName)
			}
		})
	}
}