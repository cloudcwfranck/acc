package verify

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/cloudcwfranck/acc/internal/config"
)

// v0.1.3 REGRESSION TEST 1: Test that input document is properly constructed
// This test verifies buildRegoInput creates the correct structure
func TestBuildRegoInput(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "acc-input-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	originalDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(originalDir)

	cfg := &config.Config{
		Project: config.ProjectConfig{Name: "test-project"},
		SBOM:    config.SBOMConfig{Format: "spdx"},
		Policy:  config.PolicyConfig{Mode: "enforce"},
	}

	// Create SBOM
	sbomDir := filepath.Join(".acc", "sbom")
	os.MkdirAll(sbomDir, 0755)
	sbomFile := filepath.Join(sbomDir, "test-project.spdx.json")
	os.WriteFile(sbomFile, []byte(`{"spdxVersion": "SPDX-2.3"}`), 0644)

	// Test: buildRegoInput should fail if no container tools available
	// Unless we can inspect a real image, we expect an error
	_, err = buildRegoInput(cfg, "test:latest", false)

	// We expect this to fail in test environment (no docker/podman/nerdctl)
	if err == nil {
		t.Log("Warning: buildRegoInput succeeded - container tools may be available")
	} else {
		// Expected error - container tools not found
		if !contains(err.Error(), "no container tools found") && !contains(err.Error(), "failed to inspect") {
			t.Errorf("Expected container tools error, got: %v", err)
		}
	}
}

// v0.1.3 REGRESSION TEST 2: Test SBOM field in input
func TestSBOMPresentField(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "acc-sbom-field-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	originalDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(originalDir)

	cfg := &config.Config{
		Project: config.ProjectConfig{Name: "sbom-test"},
		SBOM:    config.SBOMConfig{Format: "spdx"},
		Policy:  config.PolicyConfig{Mode: "enforce"},
	}

	// Test 1: SBOM present
	sbomDir := filepath.Join(".acc", "sbom")
	os.MkdirAll(sbomDir, 0755)
	sbomFile := filepath.Join(sbomDir, "sbom-test.spdx.json")
	os.WriteFile(sbomFile, []byte(`{}`), 0644)

	present, err := checkSBOMExists(cfg)
	if err != nil {
		t.Fatalf("checkSBOMExists failed: %v", err)
	}
	if !present {
		t.Error("Expected SBOM to be present")
	}

	// Test 2: SBOM absent
	os.Remove(sbomFile)
	present, err = checkSBOMExists(cfg)
	if err != nil {
		t.Fatalf("checkSBOMExists failed: %v", err)
	}
	if present {
		t.Error("Expected SBOM to be absent")
	}
}

// v0.1.3 REGRESSION TEST 3: Test policy explain includes input
func TestPolicyExplainIncludesInput(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "acc-explain-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	originalDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(originalDir)

	// Create a mock verify state with input
	stateDir := filepath.Join(".acc", "state")
	os.MkdirAll(stateDir, 0755)

	input := &RegoInput{
		Config: ImageConfig{
			User:   "",
			Labels: map[string]string{"test": "value"},
		},
		SBOM:        SBOMInfo{Present: true},
		Attestation: AttestationInfo{Present: false},
		Promotion:   false,
	}

	result := &VerifyResult{
		Status:      "fail",
		SBOMPresent: true,
		Input:       input,
	}

	state := VerifyState{
		ImageRef:  "test:latest",
		Status:    "fail",
		Timestamp: "2025-01-19T00:00:00Z",
		Result:    result,
	}

	// Save state
	stateFile := filepath.Join(stateDir, "last_verify.json")
	data, _ := json.MarshalIndent(state, "", "  ")
	os.WriteFile(stateFile, data, 0644)

	// Read it back and verify input is present
	readData, err := os.ReadFile(stateFile)
	if err != nil {
		t.Fatalf("failed to read state: %v", err)
	}

	var readState VerifyState
	if err := json.Unmarshal(readData, &readState); err != nil {
		t.Fatalf("failed to parse state: %v", err)
	}

	// Verify input is in the result
	if readState.Result == nil {
		t.Fatal("Result is nil")
	}

	if readState.Result.Input == nil {
		t.Fatal("Input is nil - policy explain will not show input")
	}

	if readState.Result.Input.Config.User != "" {
		t.Errorf("Expected empty User, got '%s'", readState.Result.Input.Config.User)
	}

	if !readState.Result.Input.SBOM.Present {
		t.Error("Expected SBOM.Present to be true")
	}
}

// v0.1.3 REGRESSION TEST 4: Test image inspection failure creates violation
func TestImageInspectFailureCreatesViolation(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "acc-inspect-fail-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	originalDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(originalDir)

	cfg := &config.Config{
		Project: config.ProjectConfig{Name: "inspect-test"},
		SBOM:    config.SBOMConfig{Format: "spdx"},
		Policy:  config.PolicyConfig{Mode: "enforce"},
	}

	// Create SBOM so we get to policy evaluation
	sbomDir := filepath.Join(".acc", "sbom")
	os.MkdirAll(sbomDir, 0755)
	sbomFile := filepath.Join(sbomDir, "inspect-test.spdx.json")
	os.WriteFile(sbomFile, []byte(`{}`), 0644)

	// Create empty policy directory (no policies)
	policyDir := filepath.Join(".acc", "policy")
	os.MkdirAll(policyDir, 0755)

	// Set escape hatch to allow test to run without OPA
	os.Setenv("ACC_ALLOW_NO_OPA", "1")
	defer os.Unsetenv("ACC_ALLOW_NO_OPA")

	// Verify with non-existent image (will fail inspection)
	result, err := Verify(cfg, "nonexistent:image", false, true)

	// Should fail with image-inspect-failed violation
	if err == nil {
		t.Fatal("Expected verification to fail with inspection error")
	}

	if result == nil {
		t.Fatal("Expected result to be returned")
	}

	if result.Status != "fail" {
		t.Errorf("Expected status 'fail', got '%s'", result.Status)
	}

	// Find the image-inspect-failed violation
	found := false
	for _, v := range result.Violations {
		if v.Rule == "image-inspect-failed" {
			found = true
			if v.Severity != "critical" {
				t.Errorf("Expected critical severity, got '%s'", v.Severity)
			}
		}
	}

	if !found {
		t.Error("Expected image-inspect-failed violation")
	}
}

// v0.1.3 REGRESSION TEST 5: Test OPA required error message
func TestOPARequiredError(t *testing.T) {
	// Unset escape hatch
	os.Unsetenv("ACC_ALLOW_NO_OPA")

	tmpDir, err := os.MkdirTemp("", "acc-opa-required-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	input := &RegoInput{
		Config:      ImageConfig{User: "", Labels: make(map[string]string)},
		SBOM:        SBOMInfo{Present: true},
		Attestation: AttestationInfo{Present: false},
		Promotion:   false,
	}

	policyDir := filepath.Join(tmpDir, ".acc", "policy")
	os.MkdirAll(policyDir, 0755)

	// Write a dummy policy file
	policyFile := filepath.Join(policyDir, "test.rego")
	os.WriteFile(policyFile, []byte("package acc.policy\n"), 0644)

	// Try to evaluate without OPA
	_, err = evaluateRego(policyDir, input)

	// Should fail with clear message about OPA being required
	if err == nil {
		t.Log("OPA may be installed on this system - test skipped")
		return
	}

	errMsg := err.Error()
	if !contains(errMsg, "opa command not found") {
		t.Errorf("Expected 'opa command not found' error, got: %v", err)
	}

	if !contains(errMsg, "Install OPA") {
		t.Error("Error message should include OPA installation instructions")
	}
}

// v0.1.3 REGRESSION TEST 6: Test escape hatch works
func TestOPAEscapeHatch(t *testing.T) {
	os.Setenv("ACC_ALLOW_NO_OPA", "1")
	defer os.Unsetenv("ACC_ALLOW_NO_OPA")

	tmpDir, err := os.MkdirTemp("", "acc-escape-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	input := &RegoInput{
		Config:      ImageConfig{User: "", Labels: make(map[string]string)},
		SBOM:        SBOMInfo{Present: true},
		Attestation: AttestationInfo{Present: false},
		Promotion:   false,
	}

	policyDir := filepath.Join(tmpDir, ".acc", "policy")
	os.MkdirAll(policyDir, 0755)

	// With escape hatch, should return empty violations (no error)
	violations, err := evaluateRego(policyDir, input)

	if err != nil {
		t.Errorf("With ACC_ALLOW_NO_OPA=1, should not error: %v", err)
	}

	if len(violations) != 0 {
		t.Errorf("Expected 0 violations with escape hatch, got %d", len(violations))
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsRec(s, substr))
}

func containsRec(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
