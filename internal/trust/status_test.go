package trust

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestStatusResultExitCode tests the exit code logic
func TestStatusResultExitCode(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		expected int
	}{
		{"pass returns 0", "pass", 0},
		{"fail returns 1", "fail", 1},
		{"unknown returns 2", "unknown", 2},
		{"warn returns 1", "warn", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &StatusResult{Status: tt.status}
			if got := result.ExitCode(); got != tt.expected {
				t.Errorf("ExitCode() = %d, want %d", got, tt.expected)
			}
		})
	}
}

// TestStatusUnknown tests that Status returns unknown when no state found
func TestStatusUnknown(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "acc-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Change to temp directory
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)

	// Status for non-existent image should return unknown
	result, err := Status("never-verified:latest", false, true)
	if err != nil {
		t.Errorf("Status() error = %v, want nil", err)
	}

	if result.Status != "unknown" {
		t.Errorf("Status = %s, want unknown", result.Status)
	}

	if result.ExitCode() != 2 {
		t.Errorf("ExitCode() = %d, want 2", result.ExitCode())
	}

	// Verify JSON structure
	if result.SchemaVersion != "v0.2" {
		t.Errorf("SchemaVersion = %s, want v0.2", result.SchemaVersion)
	}

	if result.Violations == nil || len(result.Violations) != 0 {
		t.Errorf("Violations should be empty array, got %v", result.Violations)
	}

	if result.Warnings == nil || len(result.Warnings) != 0 {
		t.Errorf("Warnings should be empty array, got %v", result.Warnings)
	}

	if result.Attestations == nil || len(result.Attestations) != 0 {
		t.Errorf("Attestations should be empty array, got %v", result.Attestations)
	}

	// v0.2.7: SBOMPresent should always be set as boolean
	if result.SBOMPresent {
		t.Errorf("SBOMPresent = %v, want false", result.SBOMPresent)
	}

	// Regression test: Verify all required fields for JSON contract stability
	jsonData := result.FormatJSON()
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(jsonData), &parsed); err != nil {
		t.Errorf("Failed to parse JSON: %v", err)
	}

	// Verify all required fields exist (even for unknown status)
	requiredFields := []string{"schemaVersion", "imageRef", "status", "sbomPresent", "violations", "warnings", "attestations", "timestamp"}
	for _, field := range requiredFields {
		if _, ok := parsed[field]; !ok {
			t.Errorf("JSON missing required field for unknown status: %s", field)
		}
	}

	// Verify sbomPresent is explicitly false (not null/missing)
	if sbom, ok := parsed["sbomPresent"].(bool); !ok || sbom {
		t.Errorf("sbomPresent should be false for unknown status, got %v", parsed["sbomPresent"])
	}
}

// TestStatusWithVerifyState tests that Status correctly reads verify state
func TestStatusWithVerifyState(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "acc-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Change to temp directory
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)

	// Create .acc/state directory
	stateDir := filepath.Join(".acc", "state")
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create verify state
	state := map[string]interface{}{
		"imageRef":  "test:latest",
		"status":    "pass",
		"timestamp": "2025-01-15T10:00:00Z",
		"result": map[string]interface{}{
			"sbomPresent": true,
			"policyResult": map[string]interface{}{
				"allow":      true,
				"violations": []interface{}{},
				"warnings":   []interface{}{},
			},
		},
	}

	data, _ := json.MarshalIndent(state, "", "  ")
	stateFile := filepath.Join(stateDir, "last_verify.json")
	if err := os.WriteFile(stateFile, data, 0644); err != nil {
		t.Fatal(err)
	}

	// Load status
	result, err := Status("test:latest", false, true)
	if err != nil {
		t.Errorf("Status() error = %v, want nil", err)
	}

	if result.Status != "pass" {
		t.Errorf("Status = %s, want pass", result.Status)
	}

	if result.ExitCode() != 0 {
		t.Errorf("ExitCode() = %d, want 0", result.ExitCode())
	}

	if !result.SBOMPresent {
		t.Errorf("SBOMPresent = %v, want true", result.SBOMPresent)
	}

	if result.Timestamp != "2025-01-15T10:00:00Z" {
		t.Errorf("Timestamp = %s, want 2025-01-15T10:00:00Z", result.Timestamp)
	}
}

// TestStatusWithViolations tests that Status correctly extracts violations
func TestStatusWithViolations(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "acc-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Change to temp directory
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)

	// Create .acc/state directory
	stateDir := filepath.Join(".acc", "state")
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create verify state with violations
	state := map[string]interface{}{
		"imageRef":  "test:root",
		"status":    "fail",
		"timestamp": "2025-01-15T10:00:00Z",
		"result": map[string]interface{}{
			"sbomPresent": true,
			"policyResult": map[string]interface{}{
				"allow": false,
				"violations": []interface{}{
					map[string]interface{}{
						"rule":     "no-root-user",
						"severity": "high",
						"result":   "fail",
						"message":  "Image runs as root user",
					},
				},
				"warnings": []interface{}{},
			},
		},
	}

	data, _ := json.MarshalIndent(state, "", "  ")
	stateFile := filepath.Join(stateDir, "last_verify.json")
	if err := os.WriteFile(stateFile, data, 0644); err != nil {
		t.Fatal(err)
	}

	// Load status
	result, err := Status("test:root", false, true)
	if err != nil {
		t.Errorf("Status() error = %v, want nil", err)
	}

	if result.Status != "fail" {
		t.Errorf("Status = %s, want fail", result.Status)
	}

	// fail status returns exit code 1
	if result.ExitCode() != 1 {
		t.Errorf("ExitCode() = %d, want 1", result.ExitCode())
	}

	if len(result.Violations) != 1 {
		t.Errorf("len(Violations) = %d, want 1", len(result.Violations))
	}

	if len(result.Violations) > 0 {
		v := result.Violations[0]
		if v.Rule != "no-root-user" {
			t.Errorf("Violation.Rule = %s, want no-root-user", v.Rule)
		}
		if v.Severity != "high" {
			t.Errorf("Violation.Severity = %s, want high", v.Severity)
		}
	}
}

// TestFindAttestationsForImage tests per-image attestation discovery
func TestFindAttestationsForImage(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "acc-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Change to temp directory
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)

	// Create attestation directories for different images
	digest1 := "abcdef123456"
	digest2 := "fedcba654321"

	attestDir1 := filepath.Join(".acc", "attestations", digest1)
	attestDir2 := filepath.Join(".acc", "attestations", digest2)

	os.MkdirAll(attestDir1, 0755)
	os.MkdirAll(attestDir2, 0755)

	// Create attestation files
	os.WriteFile(filepath.Join(attestDir1, "20250115-100000-attestation.json"), []byte("{}"), 0644)
	os.WriteFile(filepath.Join(attestDir1, "20250115-110000-attestation.json"), []byte("{}"), 0644)
	os.WriteFile(filepath.Join(attestDir2, "20250115-120000-attestation.json"), []byte("{}"), 0644)

	// Test finding attestations for digest1
	attestations1 := findAttestationsForImage(digest1)
	if len(attestations1) != 2 {
		t.Errorf("findAttestationsForImage(%s) returned %d attestations, want 2", digest1, len(attestations1))
	}

	// Test finding attestations for digest2
	attestations2 := findAttestationsForImage(digest2)
	if len(attestations2) != 1 {
		t.Errorf("findAttestationsForImage(%s) returned %d attestations, want 1", digest2, len(attestations2))
	}

	// Test finding attestations for non-existent digest
	attestations3 := findAttestationsForImage("nonexistent")
	if len(attestations3) != 0 {
		t.Errorf("findAttestationsForImage(nonexistent) returned %d attestations, want 0", len(attestations3))
	}

	// Test with longer digest (should use first 12 chars)
	longDigest := digest1 + "extracharacters"
	attestations4 := findAttestationsForImage(longDigest)
	if len(attestations4) != 2 {
		t.Errorf("findAttestationsForImage(%s) returned %d attestations, want 2", longDigest, len(attestations4))
	}
}

// TestStatusJSONSchema tests the JSON output schema
func TestStatusJSONSchema(t *testing.T) {
	result := &StatusResult{
		SchemaVersion: "v0.2",
		ImageRef:      "test:latest",
		Status:        "pass",
		SBOMPresent:   true,
		Violations:    []Violation{},
		Warnings:      []Violation{},
		Attestations:  []string{},
		Timestamp:     "2025-01-15T10:00:00Z",
	}

	jsonData := result.FormatJSON()
	if jsonData == "" {
		t.Error("FormatJSON() returned empty string")
	}

	// Verify JSON can be parsed
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(jsonData), &parsed); err != nil {
		t.Errorf("Failed to parse JSON: %v", err)
	}

	// Verify required fields exist
	requiredFields := []string{"schemaVersion", "imageRef", "status", "sbomPresent", "violations", "warnings", "attestations", "timestamp"}
	for _, field := range requiredFields {
		if _, ok := parsed[field]; !ok {
			t.Errorf("JSON missing required field: %s", field)
		}
	}

	// Verify empty arrays are present (not null)
	if parsed["violations"] == nil {
		t.Error("violations should be [], not null")
	}
	if parsed["warnings"] == nil {
		t.Error("warnings should be [], not null")
	}
	if parsed["attestations"] == nil {
		t.Error("attestations should be [], not null")
	}

	// Verify sbomPresent is boolean
	if sbom, ok := parsed["sbomPresent"].(bool); !ok {
		t.Errorf("sbomPresent should be boolean, got %T", parsed["sbomPresent"])
	} else if !sbom {
		t.Error("sbomPresent should be true")
	}
}

// TestGetString tests the getString helper function
func TestGetString(t *testing.T) {
	m := map[string]interface{}{
		"stringVal": "hello",
		"intVal":    42,
		"boolVal":   true,
	}

	tests := []struct {
		key      string
		expected string
	}{
		{"stringVal", "hello"},
		{"intVal", ""},  // Not a string
		{"boolVal", ""}, // Not a string
		{"missing", ""}, // Missing key
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			got := getString(m, tt.key)
			if got != tt.expected {
				t.Errorf("getString(%s) = %q, want %q", tt.key, got, tt.expected)
			}
		})
	}
}
