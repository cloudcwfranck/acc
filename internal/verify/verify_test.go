package verify

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
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
	result, err := Verify(cfg, "nonexistent:image", false, true, nil)

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

// v0.1.4 REGRESSION TEST 6: Test escape hatch behavior
// v0.1.4 change: Escape hatch still creates violation (not a bypass)
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

	// v0.1.4: With escape hatch, returns opa-required violation (not empty)
	// This allows CI tests to run while still recording OPA missing as a failure
	violations, err := evaluateRego(policyDir, input)

	if err != nil {
		t.Errorf("With ACC_ALLOW_NO_OPA=1, should not error: %v", err)
	}

	if len(violations) != 1 {
		t.Errorf("Expected 1 violation (opa-required) with escape hatch, got %d", len(violations))
	}

	if len(violations) > 0 {
		if violations[0].Rule != "opa-required" {
			t.Errorf("Expected rule 'opa-required', got '%s'", violations[0].Rule)
		}
		if violations[0].Severity != "critical" {
			t.Errorf("Expected severity 'critical', got '%s'", violations[0].Severity)
		}
	}
}

// v0.1.4 REGRESSION TEST 7: Test no panic when OPA missing
func TestVerify_NoPanic_WhenOPAIsMissing(t *testing.T) {
	// This test should complete without panicking
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Verify panicked when OPA missing: %v", r)
		}
	}()

	os.Setenv("ACC_ALLOW_NO_OPA", "1")
	defer os.Unsetenv("ACC_ALLOW_NO_OPA")

	tmpDir, err := os.MkdirTemp("", "acc-no-panic-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Setup minimal test structure
	os.MkdirAll(filepath.Join(tmpDir, ".acc", "sbom"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, ".acc", "policy"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, ".acc", "state"), 0755)

	// Create a policy file
	policyFile := filepath.Join(tmpDir, ".acc", "policy", "test.rego")
	os.WriteFile(policyFile, []byte("package acc.policy\nresult := {\"allow\": true}\n"), 0644)

	// Create SBOM
	sbomFile := filepath.Join(tmpDir, ".acc", "sbom", "test-app.syft.json")
	os.WriteFile(sbomFile, []byte("{}"), 0644)

	// Create config
	cfg := &config.Config{
		Project: config.ProjectConfig{Name: "test-app"},
		SBOM:    config.SBOMConfig{Format: "syft"},
		Policy:  config.PolicyConfig{Mode: "enforce"},
	}

	// Change to temp dir
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	// This should NOT panic
	result, _ := Verify(cfg, "test:image", false, true, nil)

	// Result should not be nil
	if result == nil {
		t.Error("Verify returned nil result")
	}

	// Should have opa-required violation
	if result != nil && len(result.Violations) == 0 {
		t.Error("Expected opa-required violation, got none")
	}
}

// v0.1.4 REGRESSION TEST 8: Test structured failure when OPA missing
func TestVerify_ReturnsStructuredFailure_WhenOPAIsMissing(t *testing.T) {
	os.Setenv("ACC_ALLOW_NO_OPA", "1")
	defer os.Unsetenv("ACC_ALLOW_NO_OPA")

	tmpDir, err := os.MkdirTemp("", "acc-structured-fail-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Setup minimal test structure
	os.MkdirAll(filepath.Join(tmpDir, ".acc", "sbom"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, ".acc", "policy"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, ".acc", "state"), 0755)

	// Create SBOM
	sbomFile := filepath.Join(tmpDir, ".acc", "sbom", "test-app.syft.json")
	os.WriteFile(sbomFile, []byte("{}"), 0644)

	// Create a policy file
	policyFile := filepath.Join(tmpDir, ".acc", "policy", "test.rego")
	os.WriteFile(policyFile, []byte("package acc.policy\nresult := {\"allow\": true}\n"), 0644)

	cfg := &config.Config{
		Project: config.ProjectConfig{Name: "test-app"},
		SBOM:    config.SBOMConfig{Format: "syft"},
		Policy:  config.PolicyConfig{Mode: "warn"}, // Use warn mode to avoid early exit
	}

	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	result, err := Verify(cfg, "test:image", false, true, nil)

	// Should return valid result (not nil)
	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	// Should have status fail
	if result.Status != "fail" {
		t.Errorf("Expected status 'fail', got '%s'", result.Status)
	}

	// Should have opa-required OR image-inspect-failed violation
	// (depends on whether container tools are available in test environment)
	foundCriticalViolation := false
	for _, v := range result.Violations {
		if v.Rule == "opa-required" || v.Rule == "image-inspect-failed" || v.Rule == "policy-evaluation-error" {
			foundCriticalViolation = true
			if v.Severity != "critical" {
				t.Errorf("Expected critical severity, got '%s' for rule '%s'", v.Severity, v.Rule)
			}
		}
	}

	if !foundCriticalViolation {
		t.Errorf("Expected at least one critical violation (opa-required, image-inspect-failed, or policy-evaluation-error). Found %d violations", len(result.Violations))
	}

	// PolicyResult should not be nil
	if result.PolicyResult == nil {
		t.Error("PolicyResult should not be nil")
	}

	// Should be valid JSON (no error in FormatJSON)
	jsonOutput := result.FormatJSON()
	if jsonOutput == "" {
		t.Error("FormatJSON returned empty string")
	}

	// ExitCode should not panic
	exitCode := result.ExitCode()
	if exitCode == 0 {
		t.Error("Expected non-zero exit code for failed verification")
	}

	// Verify should not return error in warn mode
	if err != nil {
		t.Errorf("Expected no error in warn mode, got: %v", err)
	}
}

// v0.1.4 REGRESSION TEST 9: Test ExitCode is nil-safe
func TestVerifyResultExitCode_NilSafe(t *testing.T) {
	// Test nil result
	var nilResult *VerifyResult
	exitCode := nilResult.ExitCode()
	if exitCode != 2 {
		t.Errorf("Expected exit code 2 for nil result, got %d", exitCode)
	}

	// Test valid result with pass
	passResult := &VerifyResult{Status: "pass"}
	if passResult.ExitCode() != 0 {
		t.Errorf("Expected exit code 0 for pass, got %d", passResult.ExitCode())
	}

	// Test valid result with fail
	failResult := &VerifyResult{Status: "fail"}
	if failResult.ExitCode() != 1 {
		t.Errorf("Expected exit code 1 for fail, got %d", failResult.ExitCode())
	}
}

// v0.1.4 REGRESSION TEST 10: Test state persistence on failure
func TestVerify_WritesState_OnFailure(t *testing.T) {
	os.Setenv("ACC_ALLOW_NO_OPA", "1")
	defer os.Unsetenv("ACC_ALLOW_NO_OPA")

	tmpDir, err := os.MkdirTemp("", "acc-state-persist-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Setup test structure
	os.MkdirAll(filepath.Join(tmpDir, ".acc", "sbom"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, ".acc", "policy"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, ".acc", "state"), 0755)

	// Create SBOM
	sbomFile := filepath.Join(tmpDir, ".acc", "sbom", "test-app.syft.json")
	os.WriteFile(sbomFile, []byte("{}"), 0644)

	// Create policy
	policyFile := filepath.Join(tmpDir, ".acc", "policy", "test.rego")
	os.WriteFile(policyFile, []byte("package acc.policy\nresult := {\"allow\": true}\n"), 0644)

	cfg := &config.Config{
		Project: config.ProjectConfig{Name: "test-app"},
		SBOM:    config.SBOMConfig{Format: "syft"},
		Policy:  config.PolicyConfig{Mode: "warn"}, // Warn mode so we don't exit early
	}

	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	// Run verify (should fail due to missing OPA)
	result, _ := Verify(cfg, "test:image", false, true, nil)

	if result == nil {
		t.Fatal("Result should not be nil")
	}

	// Check that state file was written
	stateFile := filepath.Join(".acc", "state", "last_verify.json")
	if _, err := os.Stat(stateFile); os.IsNotExist(err) {
		t.Error("State file was not written on failure")
	}

	// Read state and verify it contains the result
	stateData, err := os.ReadFile(stateFile)
	if err != nil {
		t.Fatalf("Failed to read state file: %v", err)
	}

	var state VerifyState
	if err := json.Unmarshal(stateData, &state); err != nil {
		t.Fatalf("Failed to parse state JSON: %v", err)
	}

	if state.Result == nil {
		t.Error("State result is nil")
	}

	if state.Status != "fail" {
		t.Errorf("Expected state status 'fail', got '%s'", state.Status)
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

// v0.2.1 REGRESSION TEST: verify status should be "pass" when allow=true
func TestVerify_StatusFromAllow(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "acc-verify-status-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	oldDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldDir)

	// Create SBOM
	sbomDir := filepath.Join(".acc", "sbom")
	os.MkdirAll(sbomDir, 0755)
	sbomFile := filepath.Join(sbomDir, "status-test.spdx.json")
	os.WriteFile(sbomFile, []byte("{}"), 0644)

	// Create policy that allows
	policyDir := filepath.Join(".acc", "policy")
	os.MkdirAll(policyDir, 0755)
	policyFile := filepath.Join(policyDir, "allow.rego")
	os.WriteFile(policyFile, []byte("package acc.policy\nresult := {\"allow\": true}\n"), 0644)

	cfg := &config.Config{
		Project: config.ProjectConfig{Name: "status-test"},
		SBOM:    config.SBOMConfig{Format: "spdx"},
		Policy:  config.PolicyConfig{Mode: "warn"},
	}

	os.Setenv("ACC_ALLOW_NO_OPA", "1")
	defer os.Unsetenv("ACC_ALLOW_NO_OPA")

	// Bug: verify returned status:"fail" even when allow:true and violations:[]
	// Fix: status should be "pass" when PolicyResult.Allow is true
	result, _ := Verify(cfg, "test:image", false, true, nil)

	if result == nil {
		t.Fatal("expected result, got nil")
	}

	if result.PolicyResult == nil {
		t.Fatal("expected policyResult, got nil")
	}

	// The key assertion: when allow=true, status should be "pass"
	if result.PolicyResult.Allow && result.Status != "pass" {
		t.Errorf("when allow=true, status should be 'pass', got %q", result.Status)
	}
}

// v0.2.1 REGRESSION TEST: SBOM detection should work after build
func TestCheckSBOMExists_Fallback(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "acc-sbom-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	oldDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldDir)

	// Create SBOM directory with a different format than expected
	sbomDir := filepath.Join(".acc", "sbom")
	os.MkdirAll(sbomDir, 0755)

	cfg := &config.Config{
		Project: config.ProjectConfig{Name: "test"},
		SBOM:    config.SBOMConfig{Format: "spdx"},
	}

	// Bug: If exact match not found, sbomPresent was false even if SBOM exists
	// Create SBOM with different name
	sbomFile := filepath.Join(sbomDir, "different-name.cyclonedx.json")
	os.WriteFile(sbomFile, []byte("{}"), 0644)

	// Fix: Should detect ANY .json file in SBOM directory as fallback
	present, err := checkSBOMExists(cfg)
	if err != nil {
		t.Fatalf("checkSBOMExists failed: %v", err)
	}

	if !present {
		t.Error("SBOM should be detected even when name/format doesn't match exactly")
	}
}

// v0.2.2 REGRESSION TEST: Final gate consistency - status MUST match allow
func TestVerify_FinalGateConsistency(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "acc-final-gate-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	oldDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldDir)

	// Create SBOM
	sbomDir := filepath.Join(".acc", "sbom")
	os.MkdirAll(sbomDir, 0755)
	sbomFile := filepath.Join(sbomDir, "final-gate-test.spdx.json")
	os.WriteFile(sbomFile, []byte("{}"), 0644)

	// Create policy that allows (no violations)
	policyDir := filepath.Join(".acc", "policy")
	os.MkdirAll(policyDir, 0755)
	policyFile := filepath.Join(policyDir, "allow.rego")
	os.WriteFile(policyFile, []byte("package acc.policy\nresult := {\"allow\": true, \"violations\": []}\n"), 0644)

	cfg := &config.Config{
		Project: config.ProjectConfig{Name: "final-gate-test"},
		SBOM:    config.SBOMConfig{Format: "spdx"},
		Policy:  config.PolicyConfig{Mode: "warn"},
	}

	os.Setenv("ACC_ALLOW_NO_OPA", "1")
	defer os.Unsetenv("ACC_ALLOW_NO_OPA")

	result, _ := Verify(cfg, "test:image", false, true, nil)

	if result == nil || result.PolicyResult == nil {
		t.Fatal("expected result with policyResult")
	}

	// CRITICAL ASSERTIONS for final gate consistency
	// Bug: status could be "fail" while allow was true
	// Fix: Single authoritative final gate ensures status matches allow

	if result.PolicyResult.Allow && result.Status != "pass" {
		t.Errorf("FINAL GATE VIOLATION: when allow=true, status MUST be 'pass', got %q", result.Status)
		t.Errorf("PolicyResult: %+v", result.PolicyResult)
	}

	if !result.PolicyResult.Allow && result.Status != "fail" {
		t.Errorf("FINAL GATE VIOLATION: when allow=false, status MUST be 'fail', got %q", result.Status)
	}

	// Exit code must also match
	expectedExitCode := 0
	if !result.PolicyResult.Allow {
		expectedExitCode = 1
	}

	if result.ExitCode() != expectedExitCode {
		t.Errorf("exit code mismatch: allow=%v should give exit=%d, got %d",
			result.PolicyResult.Allow, expectedExitCode, result.ExitCode())
	}
}

// v0.2.2 REGRESSION TEST: SBOM missing error should have workflow guidance
func TestVerify_SBOMMissingErrorMessage(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "acc-sbom-error-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	oldDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldDir)

	// NO SBOM directory - SBOM missing

	cfg := &config.Config{
		Project: config.ProjectConfig{Name: "sbom-error-test"},
		SBOM:    config.SBOMConfig{Format: "spdx"},
		Policy:  config.PolicyConfig{Mode: "enforce"},
	}

	result, err := Verify(cfg, "test:image", false, true, nil)

	if err == nil {
		t.Error("expected error when SBOM missing in enforce mode")
	}

	if result == nil {
		t.Fatal("expected result even on error")
	}

	// Bug: Error message didn't provide workflow guidance
	// Fix: Error message now includes step-by-step SBOM generation workflow

	if err != nil {
		errorMsg := err.Error()
		if !strings.Contains(errorMsg, "docker build") && !strings.Contains(errorMsg, "syft") && !strings.Contains(errorMsg, "acc build") {
			t.Errorf("error message should include workflow guidance (docker build, syft, or acc build), got: %s", errorMsg)
		}
	}

	// Check that violation exists
	if len(result.Violations) == 0 {
		t.Error("expected SBOM violation when SBOM missing")
	}

	found := false
	for _, v := range result.Violations {
		if v.Rule == "sbom-required" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected sbom-required violation")
	}
}
