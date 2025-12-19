package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig("test-project")

	if cfg.Project.Name != "test-project" {
		t.Errorf("expected project name 'test-project', got '%s'", cfg.Project.Name)
	}

	if cfg.Build.Context != "." {
		t.Errorf("expected build context '.', got '%s'", cfg.Build.Context)
	}

	if cfg.Policy.Mode != "enforce" {
		t.Errorf("expected policy mode 'enforce', got '%s'", cfg.Policy.Mode)
	}

	if cfg.Signing.Mode != "keyless" {
		t.Errorf("expected signing mode 'keyless', got '%s'", cfg.Signing.Mode)
	}

	if cfg.SBOM.Format != "spdx" {
		t.Errorf("expected SBOM format 'spdx', got '%s'", cfg.SBOM.Format)
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid config",
			cfg:     DefaultConfig("test"),
			wantErr: false,
		},
		{
			name: "missing project name",
			cfg: &Config{
				Project:  ProjectConfig{Name: ""},
				Build:    BuildConfig{Context: ".", DefaultTag: "latest"},
				Registry: RegistryConfig{Default: "localhost:5000"},
				Policy:   PolicyConfig{Mode: "enforce"},
				Signing:  SigningConfig{Mode: "keyless"},
				SBOM:     SBOMConfig{Format: "spdx"},
			},
			wantErr: true,
			errMsg:  "project.name is required",
		},
		{
			name: "invalid policy mode",
			cfg: &Config{
				Project:  ProjectConfig{Name: "test"},
				Build:    BuildConfig{Context: ".", DefaultTag: "latest"},
				Registry: RegistryConfig{Default: "localhost:5000"},
				Policy:   PolicyConfig{Mode: "invalid"},
				Signing:  SigningConfig{Mode: "keyless"},
				SBOM:     SBOMConfig{Format: "spdx"},
			},
			wantErr: true,
			errMsg:  "policy.mode must be 'enforce' or 'warn'",
		},
		{
			name: "invalid signing mode",
			cfg: &Config{
				Project:  ProjectConfig{Name: "test"},
				Build:    BuildConfig{Context: ".", DefaultTag: "latest"},
				Registry: RegistryConfig{Default: "localhost:5000"},
				Policy:   PolicyConfig{Mode: "enforce"},
				Signing:  SigningConfig{Mode: "invalid"},
				SBOM:     SBOMConfig{Format: "spdx"},
			},
			wantErr: true,
			errMsg:  "signing.mode must be 'keyless' or 'key'",
		},
		{
			name: "invalid sbom format",
			cfg: &Config{
				Project:  ProjectConfig{Name: "test"},
				Build:    BuildConfig{Context: ".", DefaultTag: "latest"},
				Registry: RegistryConfig{Default: "localhost:5000"},
				Policy:   PolicyConfig{Mode: "enforce"},
				Signing:  SigningConfig{Mode: "keyless"},
				SBOM:     SBOMConfig{Format: "invalid"},
			},
			wantErr: true,
			errMsg:  "sbom.format must be 'spdx' or 'cyclonedx'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err.Error() != tt.errMsg {
				t.Errorf("Validate() error message = %v, want %v", err.Error(), tt.errMsg)
			}
		})
	}
}

func TestInit(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "acc-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Change to temp directory
	originalDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp dir: %v", err)
	}
	defer os.Chdir(originalDir)

	// Test init
	err = Init("test-project", true) // Use JSON output to suppress UI
	if err != nil {
		t.Fatalf("Init() failed: %v", err)
	}

	// Check that acc.yaml was created
	if _, err := os.Stat("acc.yaml"); os.IsNotExist(err) {
		t.Error("acc.yaml was not created")
	}

	// Check that .acc directory was created
	if _, err := os.Stat(".acc"); os.IsNotExist(err) {
		t.Error(".acc directory was not created")
	}

	// Check that policy file was created
	policyPath := filepath.Join(".acc", "policy", "default.rego")
	if _, err := os.Stat(policyPath); os.IsNotExist(err) {
		t.Error("default policy was not created")
	}

	// Test that init fails if acc.yaml already exists
	err = Init("test-project", true)
	if err == nil {
		t.Error("Init() should fail when acc.yaml already exists")
	}
}

func TestToYAML(t *testing.T) {
	cfg := DefaultConfig("test-project")
	yaml := cfg.ToYAML()

	// Check that YAML contains expected values
	expectedStrings := []string{
		"project:",
		"name: test-project",
		"build:",
		"context: .",
		"defaultTag: latest",
		"policy:",
		"mode: enforce",
		"signing:",
		"mode: keyless",
		"sbom:",
		"format: spdx",
	}

	for _, expected := range expectedStrings {
		if !contains(yaml, expected) {
			t.Errorf("YAML does not contain expected string: %s", expected)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || contains(s[1:], substr)))
}
