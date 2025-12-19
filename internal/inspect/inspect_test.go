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
