package inspect

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/cloudcwfranck/acc/internal/config"
)

func TestInspect(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "acc-inspect-test-*")
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

	// Create config
	cfg := config.DefaultConfig("test-project")

	// Test inspect with no artifacts
	result, err := Inspect(cfg, "test:latest", true)
	if err != nil {
		t.Fatalf("Inspect() failed: %v", err)
	}

	if result.ImageRef != "test:latest" {
		t.Errorf("expected imageRef 'test:latest', got '%s'", result.ImageRef)
	}

	if result.Status != "unknown" {
		t.Errorf("expected status 'unknown', got '%s'", result.Status)
	}

	if result.SchemaVersion != "v0.1" {
		t.Errorf("expected schemaVersion 'v0.1', got '%s'", result.SchemaVersion)
	}

	if result.Policy.Mode != "enforce" {
		t.Errorf("expected policy mode 'enforce', got '%s'", result.Policy.Mode)
	}
}

func TestFindSBOM(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "acc-sbom-test-*")
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

	cfg := config.DefaultConfig("test-project")

	// Test with no SBOM directory
	path, format := findSBOM(cfg)
	if path != "" {
		t.Errorf("expected empty path, got '%s'", path)
	}
	if format != "" {
		t.Errorf("expected empty format, got '%s'", format)
	}

	// Create SBOM directory and file
	sbomDir := filepath.Join(".acc", "sbom")
	if err := os.MkdirAll(sbomDir, 0755); err != nil {
		t.Fatalf("failed to create sbom dir: %v", err)
	}

	sbomFile := filepath.Join(sbomDir, "test-project.spdx.json")
	if err := os.WriteFile(sbomFile, []byte("{}"), 0644); err != nil {
		t.Fatalf("failed to write sbom file: %v", err)
	}

	// Test with SBOM file
	path, format = findSBOM(cfg)
	if path == "" {
		t.Error("expected path to be set")
	}
	if format != "spdx" {
		t.Errorf("expected format 'spdx', got '%s'", format)
	}
}

func TestFindAttestations(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "acc-attest-test-*")
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

	// Test with no attestations directory
	attestations := findAttestations()
	if len(attestations) != 0 {
		t.Errorf("expected 0 attestations, got %d", len(attestations))
	}

	// Create attestations directory and files
	attestDir := filepath.Join(".acc", "attestations")
	if err := os.MkdirAll(attestDir, 0755); err != nil {
		t.Fatalf("failed to create attestations dir: %v", err)
	}

	attestFile1 := filepath.Join(attestDir, "attest1.json")
	if err := os.WriteFile(attestFile1, []byte("{}"), 0644); err != nil {
		t.Fatalf("failed to write attestation file: %v", err)
	}

	attestFile2 := filepath.Join(attestDir, "attest2.json")
	if err := os.WriteFile(attestFile2, []byte("{}"), 0644); err != nil {
		t.Fatalf("failed to write attestation file: %v", err)
	}

	// Test with attestation files
	attestations = findAttestations()
	if len(attestations) != 2 {
		t.Errorf("expected 2 attestations, got %d", len(attestations))
	}
}

func TestLoadLastVerifyStatus(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "acc-verify-state-test-*")
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

	// Test with no state file
	status := loadLastVerifyStatus()
	if status != nil {
		t.Error("expected nil status when no state file exists")
	}

	// Create state directory and file
	stateDir := filepath.Join(".acc", "state")
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		t.Fatalf("failed to create state dir: %v", err)
	}

	verifyStatus := LastVerifyStatus{
		Status:    "pass",
		Timestamp: "2025-01-01T00:00:00Z",
		ImageRef:  "test:latest",
	}

	data, _ := json.Marshal(verifyStatus)
	stateFile := filepath.Join(stateDir, "last_verify.json")
	if err := os.WriteFile(stateFile, data, 0644); err != nil {
		t.Fatalf("failed to write state file: %v", err)
	}

	// Test with state file
	status = loadLastVerifyStatus()
	if status == nil {
		t.Fatal("expected status to be loaded")
	}

	if status.Status != "pass" {
		t.Errorf("expected status 'pass', got '%s'", status.Status)
	}

	if status.ImageRef != "test:latest" {
		t.Errorf("expected imageRef 'test:latest', got '%s'", status.ImageRef)
	}
}

func TestFormatJSON(t *testing.T) {
	result := &InspectResult{
		SchemaVersion: "v0.1",
		ImageRef:      "test:latest",
		Status:        "pass",
		Artifacts: ArtifactInfo{
			SBOMPath:     ".acc/sbom/test.json",
			SBOMFormat:   "spdx",
			Attestations: []string{},
		},
		Policy: PolicyInfo{
			Mode:       "enforce",
			PolicyPack: ".acc/policy",
			Waivers:    []Waiver{},
		},
		Metadata:  map[string]string{"test": "value"},
		Timestamp: "2025-01-01T00:00:00Z",
	}

	jsonStr := result.FormatJSON()
	if jsonStr == "" {
		t.Error("expected non-empty JSON string")
	}

	// Verify it's valid JSON
	var decoded InspectResult
	if err := json.Unmarshal([]byte(jsonStr), &decoded); err != nil {
		t.Errorf("failed to decode JSON: %v", err)
	}

	if decoded.ImageRef != "test:latest" {
		t.Errorf("expected imageRef 'test:latest', got '%s'", decoded.ImageRef)
	}
}

// TestFindAttestationsInSubdirectories tests that attestations in subdirectories are discovered
// This test would FAIL on v0.1.0 (only looked at top-level files)
// This test should PASS after the fix
func TestFindAttestationsInSubdirectories(t *testing.T) {
	// Create temporary directory for test
	tmpDir, err := os.MkdirTemp("", "acc-attest-subdir-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Change to temp directory
	originalDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change dir: %v", err)
	}
	defer os.Chdir(originalDir)

	// Create attestations in subdirectories (as attest command does)
	// .acc/attestations/<digest>/YYYYMMDD-HHMMSS-attestation.json
	attestDir1 := filepath.Join(".acc", "attestations", "abc123def456")
	if err := os.MkdirAll(attestDir1, 0755); err != nil {
		t.Fatalf("failed to create attestation subdir 1: %v", err)
	}

	attestDir2 := filepath.Join(".acc", "attestations", "def456abc789")
	if err := os.MkdirAll(attestDir2, 0755); err != nil {
		t.Fatalf("failed to create attestation subdir 2: %v", err)
	}

	// Create attestation files in subdirectories
	attestFile1 := filepath.Join(attestDir1, "20250115-100000-attestation.json")
	if err := os.WriteFile(attestFile1, []byte(`{"schemaVersion":"v0.1"}`), 0644); err != nil {
		t.Fatalf("failed to write attestation file 1: %v", err)
	}

	attestFile2 := filepath.Join(attestDir2, "20250115-110000-attestation.json")
	if err := os.WriteFile(attestFile2, []byte(`{"schemaVersion":"v0.1"}`), 0644); err != nil {
		t.Fatalf("failed to write attestation file 2: %v", err)
	}

	attestFile3 := filepath.Join(attestDir2, "20250115-120000-attestation.json")
	if err := os.WriteFile(attestFile3, []byte(`{"schemaVersion":"v0.1"}`), 0644); err != nil {
		t.Fatalf("failed to write attestation file 3: %v", err)
	}

	// Test attestation discovery
	attestations := findAttestations()

	// Should find ALL 3 attestations in subdirectories
	if len(attestations) != 3 {
		t.Errorf("Expected 3 attestations in subdirectories, got %d", len(attestations))
		t.Logf("Found attestations: %v", attestations)
	}

	// Verify paths are correct
	foundPaths := make(map[string]bool)
	for _, path := range attestations {
		foundPaths[path] = true
	}

	if !foundPaths[attestFile1] {
		t.Errorf("Expected to find %s", attestFile1)
	}
	if !foundPaths[attestFile2] {
		t.Errorf("Expected to find %s", attestFile2)
	}
	if !foundPaths[attestFile3] {
		t.Errorf("Expected to find %s", attestFile3)
	}
}

// v0.1.5 REGRESSION TEST 1: Test per-image verification state
func TestInspect_PerImageVerificationState(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "acc-inspect-perimage-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	// Setup directories
	os.MkdirAll(".acc/state/verify", 0755)

	// Create two different digest-scoped states
	digest1 := "abc123def456"
	digest2 := "xyz789uvw012"

	state1 := map[string]interface{}{
		"imageRef":  "test:v1",
		"status":    "pass",
		"timestamp": "2025-01-01T00:00:00Z",
	}

	state2 := map[string]interface{}{
		"imageRef":  "test:v2",
		"status":    "fail",
		"timestamp": "2025-01-01T01:00:00Z",
	}

	data1, _ := json.MarshalIndent(state1, "", "  ")
	data2, _ := json.MarshalIndent(state2, "", "  ")

	os.WriteFile(filepath.Join(".acc/state/verify", digest1+".json"), data1, 0644)
	os.WriteFile(filepath.Join(".acc/state/verify", digest2+".json"), data2, 0644)

	// Test that loadVerifyStatusForImage loads the correct state for each digest
	status1 := loadVerifyStatusForImage(digest1)
	if status1 == nil {
		t.Fatal("Expected non-nil status for digest1")
	}
	if status1.Status != "pass" {
		t.Errorf("Expected status 'pass' for digest1, got '%s'", status1.Status)
	}

	status2 := loadVerifyStatusForImage(digest2)
	if status2 == nil {
		t.Fatal("Expected non-nil status for digest2")
	}
	if status2.Status != "fail" {
		t.Errorf("Expected status 'fail' for digest2, got '%s'", status2.Status)
	}
}

// v0.1.5 REGRESSION TEST 2: Test inspect doesn't leak last verify
func TestInspect_DoesNotLeakLastVerify(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "acc-inspect-noleak-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	// Setup directories
	os.MkdirAll(".acc/state/verify", 0755)

	// Create a global last_verify.json with status "fail"
	globalState := map[string]interface{}{
		"imageRef":  "other:image",
		"status":    "fail",
		"timestamp": "2025-01-01T00:00:00Z",
	}
	globalData, _ := json.MarshalIndent(globalState, "", "  ")
	os.WriteFile(".acc/state/last_verify.json", globalData, 0644)

	// Create a digest-scoped state for our image with status "pass"
	digest := "abc123def456"
	digestState := map[string]interface{}{
		"imageRef":  "test:image",
		"status":    "pass",
		"timestamp": "2025-01-01T01:00:00Z",
	}
	digestData, _ := json.MarshalIndent(digestState, "", "  ")
	os.WriteFile(filepath.Join(".acc/state/verify", digest+".json"), digestData, 0644)

	// The bug: inspect would show global "fail" instead of digest-scoped "pass"
	// After fix: inspect should show digest-scoped "pass"
	status := loadVerifyStatusForImage(digest)
	if status == nil {
		t.Fatal("Expected non-nil status")
	}

	if status.Status != "pass" {
		t.Errorf("Expected digest-scoped status 'pass', got '%s' (leaked from global last_verify.json)", status.Status)
	}

	if status.ImageRef != "test:image" {
		t.Errorf("Expected imageRef 'test:image', got '%s'", status.ImageRef)
	}
}
