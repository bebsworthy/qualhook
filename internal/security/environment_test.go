//go:build unit

package security

import (
	"fmt"
	"strings"
	"testing"
)

func TestSanitizeEnvironment(t *testing.T) {
	tests := []struct {
		name         string
		env          []string
		allowInherit bool
		wantContains []string
		wantExclude  []string
	}{
		{
			name:         "minimal environment",
			env:          nil,
			allowInherit: false,
			wantContains: []string{}, // PATH is only included if it exists in parent environment
			wantExclude:  []string{"SECRET", "TOKEN", "PASSWORD"},
		},
		{
			name: "filter sensitive variables",
			env: []string{
				"PATH=/usr/bin:/bin",
				"USER=test",
				"AWS_SECRET_ACCESS_KEY=secret123",
				"GITHUB_TOKEN=ghp_123",
				"HOME=/home/test",
			},
			allowInherit: true,
			wantContains: []string{"PATH=", "USER=", "HOME="},
			wantExclude:  []string{"AWS_SECRET_ACCESS_KEY", "GITHUB_TOKEN"},
		},
		{
			name: "filter variables with sensitive patterns",
			env: []string{
				"MY_SECRET_KEY=value",
				"DB_PASSWORD=pass123",
				"API_TOKEN=token123",
				"NORMAL_VAR=value",
			},
			allowInherit: true,
			wantContains: []string{"NORMAL_VAR="},
			wantExclude:  []string{"SECRET", "PASSWORD", "TOKEN"},
		},
		{
			name: "filter dangerous system variables",
			env: []string{
				"PATH=/usr/bin",
				"LD_PRELOAD=/evil/lib.so",
				"DYLD_INSERT_LIBRARIES=/evil/lib.dylib",
				"BASH_ENV=/evil/script.sh",
			},
			allowInherit: true,
			wantContains: []string{"PATH="},
			wantExclude:  []string{"LD_PRELOAD", "DYLD_INSERT_LIBRARIES", "BASH_ENV"},
		},
		{
			name: "filter command injection attempts",
			env: []string{
				"NORMAL=value",
				"INJECTED=$(whoami)",
				"BACKTICK=`id`",
				"PIPE=value|cat",
				"SEMICOLON=value;ls",
			},
			allowInherit: true,
			wantContains: []string{"NORMAL="},
			wantExclude:  []string{"INJECTED", "BACKTICK", "PIPE", "SEMICOLON"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeEnvironment(tt.env, tt.allowInherit)

			// Check wanted variables are present
			for _, want := range tt.wantContains {
				found := false
				for _, env := range result {
					if strings.Contains(env, want) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("SanitizeEnvironment() missing expected variable containing %q", want)
				}
			}

			// Check excluded variables are not present
			for _, exclude := range tt.wantExclude {
				for _, env := range result {
					if strings.Contains(env, exclude) {
						t.Errorf("SanitizeEnvironment() included excluded pattern %q in %q", exclude, env)
					}
				}
			}
		})
	}
}

func TestValidateEnvValue(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		value   string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid value",
			key:     "NORMAL_VAR",
			value:   "normal value",
			wantErr: false,
		},
		{
			name:    "null byte",
			key:     "VAR",
			value:   "value\x00null",
			wantErr: true,
			errMsg:  "null byte",
		},
		{
			name:    "command substitution",
			key:     "VAR",
			value:   "$(whoami)",
			wantErr: true,
			errMsg:  "dangerous pattern",
		},
		{
			name:    "backtick substitution",
			key:     "VAR",
			value:   "`id`",
			wantErr: true,
			errMsg:  "dangerous pattern",
		},
		{
			name:    "pipe character",
			key:     "VAR",
			value:   "value | cat",
			wantErr: true,
			errMsg:  "dangerous pattern",
		},
		{
			name:    "PATH with relative entry",
			key:     "PATH",
			value:   "/usr/bin:./bin:/bin",
			wantErr: true,
			errMsg:  "relative path",
		},
		{
			name:    "PATH with parent directory",
			key:     "PATH",
			value:   "/usr/bin:../bin:/bin",
			wantErr: true,
			errMsg:  "parent directory",
		},
		{
			name:    "valid PATH",
			key:     "PATH",
			value:   "/usr/local/bin:/usr/bin:/bin",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateEnvValue(tt.key, tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateEnvValue() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("validateEnvValue() error = %v, want error containing %q", err, tt.errMsg)
			}
		})
	}
}

func TestMergeEnvironment(t *testing.T) {
	tests := []struct {
		name    string
		base    []string
		custom  []string
		want    map[string]string
		wantErr bool
	}{
		{
			name:   "simple merge",
			base:   []string{"VAR1=base1", "VAR2=base2"},
			custom: []string{"VAR2=custom2", "VAR3=custom3"},
			want: map[string]string{
				"VAR1": "base1",
				"VAR2": "custom2",
				"VAR3": "custom3",
			},
			wantErr: false,
		},
		{
			name:    "invalid custom format",
			base:    []string{"VAR1=base1"},
			custom:  []string{"INVALID"},
			wantErr: true,
		},
		{
			name:    "dangerous custom value",
			base:    []string{"VAR1=base1"},
			custom:  []string{"VAR2=$(whoami)"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := MergeEnvironment(tt.base, tt.custom)
			if (err != nil) != tt.wantErr {
				t.Errorf("MergeEnvironment() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && err == nil {
				// Convert result to map for comparison
				resultMap := make(map[string]string)
				for _, env := range result {
					parts := strings.SplitN(env, "=", 2)
					if len(parts) == 2 {
						resultMap[parts[0]] = parts[1]
					}
				}

				// Check all expected values
				for k, v := range tt.want {
					if resultMap[k] != v {
						t.Errorf("MergeEnvironment() %s = %q, want %q", k, resultMap[k], v)
					}
				}

				// Check no extra values
				if len(resultMap) != len(tt.want) {
					t.Errorf("MergeEnvironment() returned %d vars, want %d", len(resultMap), len(tt.want))
				}
			}
		})
	}
}

func TestContainsSensitivePattern(t *testing.T) {
	tests := []struct {
		key  string
		want bool
	}{
		{"NORMAL_VAR", false},
		{"DB_PASSWORD", true},
		{"API_SECRET", true},
		{"AUTH_TOKEN", true},
		{"PRIVATE_KEY", true},
		{"ACCESS_KEY_ID", true},
		{"MY_CREDENTIAL", true},
		{"password_hash", true},
		{"SECRET_VALUE", true},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			if got := containsSensitivePattern(tt.key); got != tt.want {
				t.Errorf("containsSensitivePattern(%q) = %v, want %v", tt.key, got, tt.want)
			}
		})
	}
}

func ExampleSanitizeEnvironment() {
	// Example of sanitizing environment for subprocess
	env := []string{
		"PATH=/usr/bin:/bin",
		"USER=alice",
		"HOME=/home/alice",
		"AWS_SECRET_ACCESS_KEY=secret123",
		"GITHUB_TOKEN=ghp_123",
		"NORMAL_VAR=value",
	}

	sanitized := SanitizeEnvironment(env, true)

	// The sanitized environment will exclude sensitive variables
	for _, e := range sanitized {
		if strings.Contains(e, "SECRET") || strings.Contains(e, "TOKEN") {
			panic("Sensitive variable leaked!")
		}
		fmt.Println(e)
	}
}
