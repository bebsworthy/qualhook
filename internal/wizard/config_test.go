//go:build unit

package wizard

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	pkgconfig "github.com/bebsworthy/qualhook/pkg/config"
)

func TestNewConfigWizard(t *testing.T) {
	t.Parallel()
	wizard, err := NewConfigWizard()
	if err != nil {
		t.Fatalf("NewConfigWizard() failed: %v", err)
	}
	if wizard == nil {
		t.Fatal("NewConfigWizard returned nil")
	}
	if wizard.projectDetector == nil {
		t.Error("wizard.projectDetector is nil")
	}
	if wizard.defaults == nil {
		t.Error("wizard.defaults is nil")
	}
}

func TestRun_ExistingConfig(t *testing.T) {
	t.Parallel()
	// Create temp directory with existing config
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "qualhook.json")

	existingConfig := &pkgconfig.Config{
		Version: "1.0",
		Commands: map[string]*pkgconfig.CommandConfig{
			"lint": {Command: "npm"},
		},
	}
	configData, _ := pkgconfig.SaveConfig(existingConfig)
	os.WriteFile(configPath, configData, 0644)

	wizard, err := NewConfigWizard()
	if err != nil {
		t.Fatalf("NewConfigWizard() failed: %v", err)
	}

	// Run without force flag
	err = wizard.Run(configPath, false)
	if err == nil {
		t.Error("Expected error when config exists and force=false")
	}
	// In test mode, survey returns EOF when trying to read from stdin
	// This is expected behavior when running tests without interactive input
	if err.Error() != "EOF" && !strings.Contains(err.Error(), "already exists") {
		t.Errorf("Expected EOF or 'already exists' error, got: %v", err)
	}
}

func TestValidateConfig(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		config  *pkgconfig.Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &pkgconfig.Config{
				Version: "1.0",
				Commands: map[string]*pkgconfig.CommandConfig{
					"lint": {
						Command:   "npm",
						Args:      []string{"run", "lint"},
						ExitCodes: []int{1},
						ErrorPatterns: []*pkgconfig.RegexPattern{
							{Pattern: "error"},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid config - missing version",
			config: &pkgconfig.Config{
				Commands: map[string]*pkgconfig.CommandConfig{
					"lint": {Command: "npm"},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid config - empty commands",
			config: &pkgconfig.Config{
				Version: "1.0",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
