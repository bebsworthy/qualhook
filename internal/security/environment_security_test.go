//go:build unit

package security

import (
	"os"
	"strings"
	"testing"
)

// TestSanitizeEnvironment_ComprehensiveSecurity tests comprehensive environment sanitization
func TestSanitizeEnvironment_ComprehensiveSecurity(t *testing.T) {
	tests := []struct {
		name         string
		env          []string
		allowInherit bool
		checkFor     map[string]bool // Variables that should/shouldn't exist
		checkNotIn   []string        // Values that should NOT be in output
	}{
		{
			name: "filter all sensitive variables",
			env: []string{
				"PATH=/usr/bin:/usr/local/bin",
				"HOME=/home/user",
				"AWS_SECRET_ACCESS_KEY=AKIAIOSFODNN7EXAMPLE",
				"GITHUB_TOKEN=ghp_1234567890abcdef",
				"API_KEY=super-secret-key",
				"DATABASE_PASSWORD=admin123",
				"LD_PRELOAD=/evil/library.so",
				"NORMAL_VAR=safe-value",
			},
			allowInherit: true,
			checkFor: map[string]bool{
				"PATH":       true,
				"HOME":       true,
				"NORMAL_VAR": true,
			},
			checkNotIn: []string{
				"AWS_SECRET_ACCESS_KEY",
				"GITHUB_TOKEN",
				"API_KEY",
				"DATABASE_PASSWORD",
				"LD_PRELOAD",
				"AKIAIOSFODNN7EXAMPLE",
				"ghp_1234567890abcdef",
				"super-secret-key",
				"admin123",
				"/evil/library.so",
			},
		},
		{
			name: "filter variables with sensitive patterns",
			env: []string{
				"MY_SECRET_TOKEN=hidden",
				"APP_PRIVATE_KEY=private123",
				"USER_PASSWORD=pass123",
				"AUTH_CREDENTIAL=cred456",
				"ACCESS_TOKEN=token789",
				"SAFE_CONFIG=value",
			},
			allowInherit: true,
			checkFor: map[string]bool{
				"SAFE_CONFIG": true,
			},
			checkNotIn: []string{
				"SECRET",
				"PRIVATE",
				"PASSWORD",
				"CREDENTIAL",
				"ACCESS_TOKEN",
				"hidden",
				"private123",
				"pass123",
				"cred456",
				"token789",
			},
		},
		{
			name: "filter malicious PATH entries",
			env: []string{
				"PATH=.:/tmp:../../../bin:/usr/bin",
				"GOPATH=/home/user/go",
			},
			allowInherit: true,
			checkFor: map[string]bool{
				"GOPATH": true,
			},
			checkNotIn: []string{
				"PATH=.:",
				"../../../bin",
			},
		},
		{
			name: "minimal environment when not inheriting",
			env: []string{
				"SECRET_KEY=secret",
				"PATH=/usr/bin",
				"HOME=/home/user",
			},
			allowInherit: false,
			checkFor: map[string]bool{
				"PATH": true,
				"HOME": true,
			},
			checkNotIn: []string{
				"SECRET_KEY",
				"secret",
			},
		},
		{
			name: "filter command injection in values",
			env: []string{
				"VAR1=$(whoami)",
				"VAR2=`id`",
				"VAR3=value;rm -rf /",
				"VAR4=normal&&malicious",
				"VAR5=pipe|command",
				"SAFE_VAR=normal_value",
			},
			allowInherit: true,
			checkFor: map[string]bool{
				"SAFE_VAR": true,
			},
			checkNotIn: []string{
				"$(whoami)",
				"`id`",
				"rm -rf",
				"&&",
				"|command",
				"VAR1",
				"VAR2",
				"VAR3",
				"VAR4",
				"VAR5",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeEnvironment(tt.env, tt.allowInherit)

			// Build a map of resulting environment
			resultMap := make(map[string]string)
			for _, env := range result {
				parts := strings.SplitN(env, "=", 2)
				if len(parts) == 2 {
					resultMap[parts[0]] = parts[1]
				}
			}

			// Check expected variables
			for varName, shouldExist := range tt.checkFor {
				_, exists := resultMap[varName]
				if exists != shouldExist {
					t.Errorf("Variable %s existence = %v, want %v", varName, exists, shouldExist)
				}
			}

			// Check that sensitive values are not present
			resultStr := strings.Join(result, " ")
			for _, sensitive := range tt.checkNotIn {
				if strings.Contains(resultStr, sensitive) {
					t.Errorf("Sensitive value %q found in sanitized environment", sensitive)
				}
			}
		})
	}
}

// TestMergeEnvironment_Security tests secure merging of environment variables
func TestMergeEnvironment_Security(t *testing.T) {
	tests := []struct {
		name      string
		base      []string
		custom    []string
		wantErr   bool
		errMsg    string
		checkVals map[string]string
	}{
		{
			name: "valid merge",
			base: []string{
				"PATH=/usr/bin",
				"HOME=/home/user",
			},
			custom: []string{
				"NODE_ENV=production",
				"PORT=3000",
			},
			wantErr: false,
			checkVals: map[string]string{
				"PATH":     "/usr/bin",
				"NODE_ENV": "production",
				"PORT":     "3000",
			},
		},
		{
			name: "override base with custom",
			base: []string{
				"NODE_ENV=development",
			},
			custom: []string{
				"NODE_ENV=production",
			},
			wantErr: false,
			checkVals: map[string]string{
				"NODE_ENV": "production",
			},
		},
		{
			name: "reject malformed custom env",
			base: []string{
				"PATH=/usr/bin",
			},
			custom: []string{
				"INVALID_NO_EQUALS",
			},
			wantErr: true,
			errMsg:  "invalid environment variable format",
		},
		{
			name: "reject dangerous custom values",
			base: []string{
				"PATH=/usr/bin",
			},
			custom: []string{
				"CMD=$(rm -rf /)",
			},
			wantErr: true,
			errMsg:  "dangerous pattern",
		},
		{
			name: "reject null bytes in custom env",
			base: []string{
				"PATH=/usr/bin",
			},
			custom: []string{
				"VAR=test\x00hack",
			},
			wantErr: true,
			errMsg:  "null byte",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := MergeEnvironment(tt.base, tt.custom)

			if (err != nil) != tt.wantErr {
				t.Errorf("MergeEnvironment() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err != nil && tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("MergeEnvironment() error = %v, want error containing %q", err, tt.errMsg)
			}

			if err == nil && tt.checkVals != nil {
				// Check merged values
				resultMap := make(map[string]string)
				for _, env := range result {
					parts := strings.SplitN(env, "=", 2)
					if len(parts) == 2 {
						resultMap[parts[0]] = parts[1]
					}
				}

				for k, v := range tt.checkVals {
					if resultMap[k] != v {
						t.Errorf("Expected %s=%s, got %s", k, v, resultMap[k])
					}
				}
			}
		})
	}
}

// TestValidatePathEnv_Security tests PATH environment variable validation
func TestValidatePathEnv_Security(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid PATH",
			path:    "/usr/bin:/usr/local/bin:/home/user/bin",
			wantErr: false,
		},
		{
			name:    "PATH with current directory",
			path:    ".:/usr/bin",
			wantErr: true,
			errMsg:  "relative path",
		},
		{
			name:    "PATH with parent directory traversal",
			path:    "/usr/bin:../../../etc",
			wantErr: true,
			errMsg:  "parent directory reference",
		},
		{
			name:    "PATH with home expansion",
			path:    "~/bin:/usr/bin",
			wantErr: true,
			errMsg:  "home directory expansion",
		},
		{
			name:    "empty PATH entries",
			path:    "/usr/bin::/usr/local/bin",
			wantErr: false, // Empty entries are skipped
		},
		{
			name:    "Windows-style PATH",
			path:    "C:\\Program Files\\Git\\bin;C:\\Windows\\System32",
			wantErr: false,
		},
		{
			name:    "Windows PATH with traversal",
			path:    "C:\\Windows\\System32;..\\..\\",
			wantErr: true,
			errMsg:  "parent directory reference",
		},
	}

	// Adjust path separator for Windows
	if os.PathListSeparator == ';' {
		// Update test cases for Windows
		for i := range tests {
			tests[i].path = strings.ReplaceAll(tests[i].path, ":", ";")
		}
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePathEnv(tt.path)

			if (err != nil) != tt.wantErr {
				t.Errorf("validatePathEnv() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err != nil && tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("validatePathEnv() error = %v, want error containing %q", err, tt.errMsg)
			}
		})
	}
}

// TestCreateMinimalEnvironment tests creation of minimal safe environment
func TestCreateMinimalEnvironment(t *testing.T) {
	// Set some test environment variables
	testEnv := map[string]string{
		"PATH":             "/usr/bin:/usr/local/bin",
		"HOME":             "/home/testuser",
		"USER":             "testuser",
		"SECRET_KEY":       "should-not-appear",
		"API_TOKEN":        "also-should-not-appear",
		"MALICIOUS_PATH":   ".:../bin",
		"SAFE_BUT_UNKNOWN": "this-should-not-appear",
	}

	// Set test environment
	for k, v := range testEnv {
		os.Setenv(k, v)
		defer os.Unsetenv(k)
	}

	// Create minimal environment
	minEnv := createMinimalEnvironment()

	// Convert to map for easier checking
	envMap := make(map[string]string)
	for _, env := range minEnv {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}

	// Check that only safe variables are included
	safeVars := []string{"PATH", "HOME", "USER"}
	for _, v := range safeVars {
		if _, ok := envMap[v]; !ok {
			t.Errorf("Expected safe variable %s not found in minimal environment", v)
		}
	}

	// Check that unsafe variables are NOT included
	unsafeVars := []string{"SECRET_KEY", "API_TOKEN", "MALICIOUS_PATH", "SAFE_BUT_UNKNOWN"}
	for _, v := range unsafeVars {
		if _, ok := envMap[v]; ok {
			t.Errorf("Unsafe variable %s found in minimal environment", v)
		}
	}

	// Check that values don't contain the unsafe content
	envStr := strings.Join(minEnv, " ")
	unsafeValues := []string{"should-not-appear", "also-should-not-appear", ".:../bin", "this-should-not-appear"}
	for _, v := range unsafeValues {
		if strings.Contains(envStr, v) {
			t.Errorf("Unsafe value %q found in minimal environment", v)
		}
	}
}

// TestEnvironmentSanitization_RealWorldScenarios tests real-world attack scenarios
func TestEnvironmentSanitization_RealWorldScenarios(t *testing.T) {
	scenarios := []struct {
		name        string
		env         []string
		description string
		checkNotIn  []string
	}{
		{
			name: "AWS credential theft attempt",
			env: []string{
				"AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE",
				"AWS_SECRET_ACCESS_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
				"AWS_SESSION_TOKEN=AQoEXAMPLEH4aoAH0gNCAPy...",
				"AWS_DEFAULT_REGION=us-east-1",
			},
			description: "Attempt to steal AWS credentials",
			checkNotIn: []string{
				"AWS_SECRET_ACCESS_KEY",
				"AWS_SESSION_TOKEN",
				"wJalrXUtnFEMI",
				"AQoEXAMPLEH4aoAH0gNCAPy",
			},
		},
		{
			name: "LD_PRELOAD hijacking",
			env: []string{
				"LD_PRELOAD=/tmp/evil.so",
				"LD_LIBRARY_PATH=/tmp:/usr/lib",
				"DYLD_INSERT_LIBRARIES=/tmp/evil.dylib",
			},
			description: "Attempt to hijack library loading",
			checkNotIn: []string{
				"LD_PRELOAD",
				"DYLD_INSERT_LIBRARIES",
				"evil.so",
				"evil.dylib",
			},
		},
		{
			name: "Shell configuration hijacking",
			env: []string{
				"BASH_ENV=/tmp/evil_script.sh",
				"ENV=/tmp/evil_profile",
				"ZDOTDIR=/tmp/evil_zsh",
			},
			description: "Attempt to hijack shell initialization",
			checkNotIn: []string{
				"BASH_ENV",
				"ENV",
				"ZDOTDIR",
				"evil_script.sh",
				"evil_profile",
				"evil_zsh",
			},
		},
		{
			name: "Database credential theft",
			env: []string{
				"DATABASE_URL=postgres://user:password@host:5432/db",
				"MYSQL_ROOT_PASSWORD=root123",
				"REDIS_PASSWORD=redis456",
				"MONGODB_URI=mongodb://admin:pass@cluster.mongodb.net",
			},
			description: "Attempt to steal database credentials",
			checkNotIn: []string{
				"password",
				"root123",
				"redis456",
				"admin:pass",
				"PASSWORD",
			},
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			t.Logf("Testing scenario: %s", scenario.description)

			// Test with inheritance
			sanitized := SanitizeEnvironment(scenario.env, true)
			sanitizedStr := strings.Join(sanitized, " ")

			// Debug: Log what we got for database test
			if scenario.name == "Database credential theft" {
				t.Logf("Sanitized environment: %v", sanitized)
			}

			for _, forbidden := range scenario.checkNotIn {
				if strings.Contains(sanitizedStr, forbidden) {
					t.Errorf("Scenario %q: Found forbidden string %q in sanitized environment",
						scenario.name, forbidden)
				}
			}

			// Test without inheritance (should be even more restrictive)
			minimal := SanitizeEnvironment(scenario.env, false)
			minimalStr := strings.Join(minimal, " ")

			for _, forbidden := range scenario.checkNotIn {
				if strings.Contains(minimalStr, forbidden) {
					t.Errorf("Scenario %q (minimal): Found forbidden string %q in sanitized environment",
						scenario.name, forbidden)
				}
			}
		})
	}
}
