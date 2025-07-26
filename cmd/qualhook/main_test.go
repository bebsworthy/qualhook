//go:build unit

package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"encoding/json"
	"github.com/bebsworthy/qualhook/pkg/config"
)

func TestNewRootCmd(t *testing.T) {
	cmd := newRootCmd()

	if cmd == nil {
		t.Fatal("newRootCmd() returned nil")
	}

	if cmd.Use != "qualhook" {
		t.Errorf("Expected Use to be 'qualhook', got %s", cmd.Use)
	}

	if cmd.Version != Version {
		t.Errorf("Expected Version to be %s, got %s", Version, cmd.Version)
	}
}

func TestParseGlobalFlags(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		expectedDebug  bool
		expectedConfig string
	}{
		{
			name:           "no flags",
			args:           []string{"qualhook", "lint"},
			expectedDebug:  false,
			expectedConfig: "",
		},
		{
			name:           "debug flag",
			args:           []string{"qualhook", "--debug", "lint"},
			expectedDebug:  true,
			expectedConfig: "",
		},
		{
			name:           "config flag",
			args:           []string{"qualhook", "--config", "custom.json", "lint"},
			expectedDebug:  false,
			expectedConfig: "custom.json",
		},
		{
			name:           "both flags",
			args:           []string{"qualhook", "--debug", "--config", "custom.json", "lint"},
			expectedDebug:  true,
			expectedConfig: "custom.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset global flags
			debugFlag = false
			configPath = ""

			// Set os.Args temporarily
			oldArgs := os.Args
			os.Args = tt.args
			defer func() { os.Args = oldArgs }()

			parseGlobalFlags()

			if debugFlag != tt.expectedDebug {
				t.Errorf("Expected debugFlag to be %v, got %v", tt.expectedDebug, debugFlag)
			}

			if configPath != tt.expectedConfig {
				t.Errorf("Expected configPath to be %q, got %q", tt.expectedConfig, configPath)
			}
		})
	}
}

func TestExtractNonFlagArgs(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected []string
	}{
		{
			name:     "no args",
			args:     []string{},
			expected: []string{},
		},
		{
			name:     "only non-flag args",
			args:     []string{"arg1", "arg2"},
			expected: []string{"arg1", "arg2"},
		},
		{
			name:     "mixed args",
			args:     []string{"--debug", "arg1", "--config", "test.json", "arg2"},
			expected: []string{"arg1", "arg2"},
		},
		{
			name:     "config flag with value",
			args:     []string{"--config", "test.json", "arg1"},
			expected: []string{"arg1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractNonFlagArgs(tt.args)

			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d args, got %d", len(tt.expected), len(result))
				return
			}

			for i, arg := range result {
				if arg != tt.expected[i] {
					t.Errorf("Expected arg[%d] to be %q, got %q", i, tt.expected[i], arg)
				}
			}
		})
	}
}

func TestTryCustomCommand(t *testing.T) {
	// Create temp directory with config
	tempDir := t.TempDir()

	cfg := &config.Config{
		Version: "1.0",
		Commands: map[string]*config.CommandConfig{
			"custom-cmd": {
				Command: "echo",
				Args:    []string{"custom command executed"},
				ExitCodes: []int{1},
				ErrorPatterns: []*config.RegexPattern{
					{Pattern: "error"},
				},
				MaxOutput: 100,
			},
		},
	}

	configData, _ := json.Marshal(cfg)
	configFile := filepath.Join(tempDir, ".qualhook.json")
	os.WriteFile(configFile, configData, 0644)

	// Change to temp directory
	oldDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldDir)

	// Reset global config path
	oldConfigPath := configPath
	configPath = ""
	defer func() { configPath = oldConfigPath }()

	tests := []struct {
		name        string
		cmdName     string
		args        []string
		shouldError bool
	}{
		{
			name:        "existing custom command",
			cmdName:     "custom-cmd",
			args:        []string{},
			shouldError: false,
		},
		{
			name:        "non-existing command",
			cmdName:     "unknown-cmd",
			args:        []string{},
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tryCustomCommand(tt.cmdName, tt.args)

			if tt.shouldError && err == nil {
				t.Error("Expected error but got none")
			}

			if !tt.shouldError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestRootCommand(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		shouldError bool
		checkOutput func(t *testing.T, stdout, stderr string)
	}{
		{
			name:        "help command",
			args:        []string{"--help"},
			shouldError: false,
			checkOutput: func(t *testing.T, stdout, stderr string) {
				if !strings.Contains(stdout, "Qualhook is a configurable command-line utility") {
					t.Error("Expected help text not found")
				}
			},
		},
		{
			name:        "version command",
			args:        []string{"--version"},
			shouldError: false,
			checkOutput: func(t *testing.T, stdout, stderr string) {
				if !strings.Contains(stdout, Version) {
					t.Error("Expected version not found")
				}
			},
		},
		{
			name:        "unknown command",
			args:        []string{"unknown-command"},
			shouldError: true,
			checkOutput: func(t *testing.T, stdout, stderr string) {
				if !strings.Contains(stderr, "unknown command") {
					t.Error("Expected error message not found")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := newRootCmd()
			cmd.SetArgs(tt.args)

			var stdout, stderr bytes.Buffer
			cmd.SetOut(&stdout)
			cmd.SetErr(&stderr)

			err := cmd.Execute()

			if tt.shouldError && err == nil {
				t.Error("Expected error but got none")
			}

			if !tt.shouldError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if tt.checkOutput != nil {
				tt.checkOutput(t, stdout.String(), stderr.String())
			}
		})
	}
}

// Command execution tests are in integration_test_simple.go
// These tests verify the command structure without execution
