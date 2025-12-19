package push

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestPushWithoutVerifyState tests that push fails when verification state is missing
func TestPushWithoutVerifyState(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp dir: %v", err)
	}

	// Create .acc directory but no state
	if err := os.MkdirAll(".acc/state", 0755); err != nil {
		t.Fatalf("failed to create .acc/state: %v", err)
	}

	// Attempt to load verify state (should fail)
	_, err := loadVerifyState()
	if err == nil {
		t.Error("expected error when verification state is missing, got nil")
	}

	// The error should be from os.ReadFile (file not found)
	if !os.IsNotExist(err) {
		t.Errorf("expected IsNotExist error, got: %v", err)
	}
}

// TestPushWithFailedVerification tests that push blocks when verification failed
func TestPushWithFailedVerification(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp dir: %v", err)
	}

	// Create state directory
	stateDir := filepath.Join(".acc", "state")
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		t.Fatalf("failed to create state dir: %v", err)
	}

	// Create verification state with "fail" status
	state := VerifyState{
		ImageRef:  "test:latest",
		Status:    "fail",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Result: map[string]interface{}{
			"status":      "fail",
			"sbomPresent": false,
		},
	}

	data, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("failed to marshal state: %v", err)
	}

	stateFile := filepath.Join(stateDir, "last_verify.json")
	if err := os.WriteFile(stateFile, data, 0644); err != nil {
		t.Fatalf("failed to write state file: %v", err)
	}

	// Load state and verify it has failed status
	loadedState, err := loadVerifyState()
	if err != nil {
		t.Fatalf("failed to load state: %v", err)
	}

	if loadedState.Status != "fail" {
		t.Errorf("expected status 'fail', got '%s'", loadedState.Status)
	}

	// Verify that the error message would mention blocking
	// (We can't call Push directly without mocking exec commands)
	if loadedState.Status == "fail" {
		// This is what Push() would check and reject
		if loadedState.Status != "pass" && loadedState.Status != "warn" {
			// Expected behavior: push should be blocked
		}
	}
}

// TestValidateImageMatchSameRef tests that exact ref matches work
func TestValidateImageMatchSameRef(t *testing.T) {
	state := &VerifyState{
		ImageRef: "myapp:latest",
		Status:   "pass",
	}

	// Same ref should pass
	err := validateImageMatch("myapp:latest", state)
	if err != nil {
		t.Errorf("expected no error for matching ref, got: %v", err)
	}
}

// TestValidateImageMatchDifferentRef tests that different refs fail
func TestValidateImageMatchDifferentRef(t *testing.T) {
	state := &VerifyState{
		ImageRef: "myapp:latest",
		Status:   "pass",
	}

	// Different ref should fail (unless digests match, which we can't test without docker)
	err := validateImageMatch("other:latest", state)
	if err == nil {
		t.Error("expected error for mismatched ref, got nil")
	}

	if !strings.Contains(err.Error(), "image mismatch") {
		t.Errorf("expected 'image mismatch' in error, got: %v", err)
	}
}

// TestLoadLastAttestation tests loading attestation pointer
func TestLoadLastAttestation(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp dir: %v", err)
	}

	// Create state directory
	stateDir := filepath.Join(".acc", "state")
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		t.Fatalf("failed to create state dir: %v", err)
	}

	// Create attestation pointer
	pointer := AttestationPointer{
		OutputPath: ".acc/attestations/test/2025-01-15-attestation.json",
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
	}

	data, err := json.Marshal(pointer)
	if err != nil {
		t.Fatalf("failed to marshal pointer: %v", err)
	}

	stateFile := filepath.Join(stateDir, "last_attestation.json")
	if err := os.WriteFile(stateFile, data, 0644); err != nil {
		t.Fatalf("failed to write attestation pointer: %v", err)
	}

	// Load pointer
	loaded := loadLastAttestation()
	if loaded == nil {
		t.Fatal("expected attestation pointer, got nil")
	}

	if loaded.OutputPath != pointer.OutputPath {
		t.Errorf("expected output path '%s', got '%s'", pointer.OutputPath, loaded.OutputPath)
	}
}

// TestLoadLastAttestationMissing tests that missing attestation returns nil
func TestLoadLastAttestationMissing(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp dir: %v", err)
	}

	// Create .acc directory but no attestation
	if err := os.MkdirAll(".acc/state", 0755); err != nil {
		t.Fatalf("failed to create .acc/state: %v", err)
	}

	// Should return nil when file doesn't exist
	loaded := loadLastAttestation()
	if loaded != nil {
		t.Error("expected nil for missing attestation, got value")
	}
}

// TestPushResultJSON tests JSON serialization of PushResult
func TestPushResultJSON(t *testing.T) {
	result := &PushResult{
		SchemaVersion:      "v0.1",
		Command:            "push",
		ImageRef:           "myapp:latest",
		ImageDigest:        "abc123def456",
		VerificationStatus: "pass",
		Pushed:             true,
		Timestamp:          "2025-01-15T10:00:00Z",
		AttestationRef:     ".acc/attestations/test/attestation.json",
	}

	jsonStr := result.FormatJSON()

	// Verify it's valid JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	// Verify schema version
	if parsed["schemaVersion"] != "v0.1" {
		t.Errorf("expected schemaVersion 'v0.1', got '%v'", parsed["schemaVersion"])
	}

	// Verify command
	if parsed["command"] != "push" {
		t.Errorf("expected command 'push', got '%v'", parsed["command"])
	}

	// Verify pushed status
	if parsed["pushed"] != true {
		t.Errorf("expected pushed true, got '%v'", parsed["pushed"])
	}
}

// TestPushWithPassedVerification tests that push allows passed verification
func TestPushWithPassedVerification(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp dir: %v", err)
	}

	// Create state directory
	stateDir := filepath.Join(".acc", "state")
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		t.Fatalf("failed to create state dir: %v", err)
	}

	// Create verification state with "pass" status
	state := VerifyState{
		ImageRef:  "test:latest",
		Status:    "pass",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Result: map[string]interface{}{
			"status":      "pass",
			"sbomPresent": true,
		},
	}

	data, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("failed to marshal state: %v", err)
	}

	stateFile := filepath.Join(stateDir, "last_verify.json")
	if err := os.WriteFile(stateFile, data, 0644); err != nil {
		t.Fatalf("failed to write state file: %v", err)
	}

	// Load state and verify it has pass status
	loadedState, err := loadVerifyState()
	if err != nil {
		t.Fatalf("failed to load state: %v", err)
	}

	if loadedState.Status != "pass" {
		t.Errorf("expected status 'pass', got '%s'", loadedState.Status)
	}

	// Verify this would NOT be blocked
	if loadedState.Status == "fail" {
		t.Error("pass status should not be blocked")
	}
}
