package profile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestLoad_ValidProfile tests loading a valid profile
func TestLoad_ValidProfile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "acc-profile-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create .acc/profiles/ directory
	profilesDir := filepath.Join(tmpDir, ".acc", "profiles")
	if err := os.MkdirAll(profilesDir, 0755); err != nil {
		t.Fatalf("failed to create profiles dir: %v", err)
	}

	// Create valid profile
	profileContent := `schemaVersion: 1
name: baseline
description: Baseline enforcement profile
policies:
  allow:
    - no-root-user
    - no-latest-tag
violations:
  ignore:
    - informational
    - low
warnings:
  show: true
`
	profilePath := filepath.Join(profilesDir, "baseline.yaml")
	if err := os.WriteFile(profilePath, []byte(profileContent), 0644); err != nil {
		t.Fatalf("failed to write profile: %v", err)
	}

	// Change to temp dir to test relative path resolution
	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	// Load profile
	profile, err := Load("baseline")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Validate fields
	if profile.SchemaVersion != 1 {
		t.Errorf("schemaVersion = %d, want 1", profile.SchemaVersion)
	}
	if profile.Name != "baseline" {
		t.Errorf("name = %q, want %q", profile.Name, "baseline")
	}
	if profile.Description != "Baseline enforcement profile" {
		t.Errorf("description = %q, want %q", profile.Description, "Baseline enforcement profile")
	}
	if len(profile.Policies.Allow) != 2 {
		t.Errorf("len(policies.allow) = %d, want 2", len(profile.Policies.Allow))
	}
	if len(profile.Violations.Ignore) != 2 {
		t.Errorf("len(violations.ignore) = %d, want 2", len(profile.Violations.Ignore))
	}
	if !profile.Warnings.Show {
		t.Error("warnings.show = false, want true")
	}
}

// TestLoad_ExplicitPath tests loading a profile from an explicit path
func TestLoad_ExplicitPath(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "acc-profile-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	profileContent := `schemaVersion: 1
name: custom
description: Custom profile
`
	profilePath := filepath.Join(tmpDir, "custom.yaml")
	if err := os.WriteFile(profilePath, []byte(profileContent), 0644); err != nil {
		t.Fatalf("failed to write profile: %v", err)
	}

	// Load with explicit path
	profile, err := Load(profilePath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if profile.Name != "custom" {
		t.Errorf("name = %q, want %q", profile.Name, "custom")
	}
}

// TestLoad_ProfileNotFound tests error when profile doesn't exist
func TestLoad_ProfileNotFound(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "acc-profile-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	_, err = Load("nonexistent")
	if err == nil {
		t.Fatal("expected error for non-existent profile, got nil")
	}

	if !strings.Contains(err.Error(), "profile not found") {
		t.Errorf("expected 'profile not found' error, got: %v", err)
	}
}

// TestLoad_InvalidYAML tests error on invalid YAML
func TestLoad_InvalidYAML(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "acc-profile-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	profilesDir := filepath.Join(tmpDir, ".acc", "profiles")
	if err := os.MkdirAll(profilesDir, 0755); err != nil {
		t.Fatalf("failed to create profiles dir: %v", err)
	}

	// Invalid YAML
	profilePath := filepath.Join(profilesDir, "invalid.yaml")
	if err := os.WriteFile(profilePath, []byte("invalid: [yaml: syntax"), 0644); err != nil {
		t.Fatalf("failed to write profile: %v", err)
	}

	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	_, err = Load("invalid")
	if err == nil {
		t.Fatal("expected error for invalid YAML, got nil")
	}

	if !strings.Contains(err.Error(), "failed to parse") {
		t.Errorf("expected 'failed to parse' error, got: %v", err)
	}
}

// TestLoad_UnknownFields tests that unknown fields are rejected
func TestLoad_UnknownFields(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "acc-profile-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	profilesDir := filepath.Join(tmpDir, ".acc", "profiles")
	if err := os.MkdirAll(profilesDir, 0755); err != nil {
		t.Fatalf("failed to create profiles dir: %v", err)
	}

	// Profile with unknown field
	profileContent := `schemaVersion: 1
name: test
description: Test profile
unknownField: value
`
	profilePath := filepath.Join(profilesDir, "unknown.yaml")
	if err := os.WriteFile(profilePath, []byte(profileContent), 0644); err != nil {
		t.Fatalf("failed to write profile: %v", err)
	}

	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	_, err = Load("unknown")
	if err == nil {
		t.Fatal("expected error for unknown field, got nil")
	}

	if !strings.Contains(err.Error(), "field unknownField not found") {
		t.Errorf("expected 'field not found' error, got: %v", err)
	}
}

// TestValidate_ValidProfile tests validation of a valid profile
func TestValidate_ValidProfile(t *testing.T) {
	profile := &Profile{
		SchemaVersion: 1,
		Name:          "test",
		Description:   "Test profile",
		Policies: PolicyConfig{
			Allow: []string{"rule1", "rule2"},
		},
		Violations: ViolationConfig{
			Ignore: []string{"low", "informational"},
		},
		Warnings: WarningConfig{
			Show: true,
		},
	}

	if err := Validate(profile); err != nil {
		t.Errorf("Validate failed for valid profile: %v", err)
	}
}

// TestValidate_InvalidSchemaVersion tests rejection of invalid schema version
func TestValidate_InvalidSchemaVersion(t *testing.T) {
	tests := []struct {
		name          string
		schemaVersion int
	}{
		{"zero", 0},
		{"negative", -1},
		{"future version", 2},
		{"large version", 999},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profile := &Profile{
				SchemaVersion: tt.schemaVersion,
				Name:          "test",
				Description:   "Test profile",
			}

			err := Validate(profile)
			if err == nil {
				t.Fatal("expected error for invalid schema version, got nil")
			}

			if !strings.Contains(err.Error(), "unsupported schemaVersion") {
				t.Errorf("expected 'unsupported schemaVersion' error, got: %v", err)
			}
		})
	}
}

// TestValidate_MissingRequiredFields tests validation of missing required fields
func TestValidate_MissingRequiredFields(t *testing.T) {
	tests := []struct {
		name        string
		profile     *Profile
		expectedErr string
	}{
		{
			name: "missing name",
			profile: &Profile{
				SchemaVersion: 1,
				Description:   "Test",
			},
			expectedErr: "name is required",
		},
		{
			name: "missing description",
			profile: &Profile{
				SchemaVersion: 1,
				Name:          "test",
			},
			expectedErr: "description is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.profile)
			if err == nil {
				t.Fatal("expected error for missing required field, got nil")
			}

			if !strings.Contains(err.Error(), tt.expectedErr) {
				t.Errorf("expected error containing %q, got: %v", tt.expectedErr, err)
			}
		})
	}
}

// TestValidate_EmptyRuleName tests rejection of empty rule names
func TestValidate_EmptyRuleName(t *testing.T) {
	profile := &Profile{
		SchemaVersion: 1,
		Name:          "test",
		Description:   "Test profile",
		Policies: PolicyConfig{
			Allow: []string{"rule1", "", "rule2"},
		},
	}

	err := Validate(profile)
	if err == nil {
		t.Fatal("expected error for empty rule name, got nil")
	}

	if !strings.Contains(err.Error(), "empty rule name not allowed") {
		t.Errorf("expected 'empty rule name not allowed' error, got: %v", err)
	}
}

// TestValidate_EmptyIgnoreValue tests rejection of empty ignore values
func TestValidate_EmptyIgnoreValue(t *testing.T) {
	profile := &Profile{
		SchemaVersion: 1,
		Name:          "test",
		Description:   "Test profile",
		Violations: ViolationConfig{
			Ignore: []string{"low", " ", "medium"},
		},
	}

	err := Validate(profile)
	if err == nil {
		t.Fatal("expected error for empty ignore value, got nil")
	}

	if !strings.Contains(err.Error(), "empty value not allowed") {
		t.Errorf("expected 'empty value not allowed' error, got: %v", err)
	}
}

// TestValidate_MinimalProfile tests validation of minimal valid profile
func TestValidate_MinimalProfile(t *testing.T) {
	profile := &Profile{
		SchemaVersion: 1,
		Name:          "minimal",
		Description:   "Minimal profile",
	}

	if err := Validate(profile); err != nil {
		t.Errorf("Validate failed for minimal profile: %v", err)
	}
}
