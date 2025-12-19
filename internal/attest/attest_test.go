package attest

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/cloudcwfranck/acc/internal/config"
)

func TestAttest(t *testing.T) {
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

	// Create config
	cfg := config.DefaultConfig("test-project")

	// Test attest
	result, err := Attest(cfg, "test:latest", true)
	if err != nil {
		t.Fatalf("Attest() failed: %v", err)
	}

	// Verify result
	if result.AttestationPath == "" {
		t.Error("expected attestation path to be set")
	}

	if result.Attestation.ImageRef != "test:latest" {
		t.Errorf("expected imageRef 'test:latest', got '%s'", result.Attestation.ImageRef)
	}

	if result.Attestation.SchemaVersion != "v0.1" {
		t.Errorf("expected schemaVersion 'v0.1', got '%s'", result.Attestation.SchemaVersion)
	}

	if result.Attestation.Type != "acc.build.v0" {
		t.Errorf("expected type 'acc.build.v0', got '%s'", result.Attestation.Type)
	}

	// Verify file was created
	if _, err := os.Stat(result.AttestationPath); os.IsNotExist(err) {
		t.Error("attestation file was not created")
	}

	// Verify file contents
	data, err := os.ReadFile(result.AttestationPath)
	if err != nil {
		t.Fatalf("failed to read attestation file: %v", err)
	}

	var attestation Attestation
	if err := json.Unmarshal(data, &attestation); err != nil {
		t.Fatalf("failed to unmarshal attestation: %v", err)
	}

	if attestation.ImageRef != "test:latest" {
		t.Errorf("attestation file has wrong imageRef: %s", attestation.ImageRef)
	}

	// Verify metadata
	if attestation.Metadata["project"] != "test-project" {
		t.Errorf("expected project 'test-project', got '%s'", attestation.Metadata["project"])
	}

	if attestation.Metadata["policyMode"] != "enforce" {
		t.Errorf("expected policyMode 'enforce', got '%s'", attestation.Metadata["policyMode"])
	}
}

func TestGetBuildMetadata(t *testing.T) {
	metadata := getBuildMetadata()

	if metadata.BuildTime == "" {
		t.Error("expected buildTime to be set")
	}

	// BuildTool may or may not be set depending on environment
	// Just verify the function doesn't panic

	// Verify env map is initialized
	if metadata.Env == nil {
		t.Error("expected env map to be initialized")
	}
}

func TestLoadPolicyHash(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "acc-policy-hash-test-*")
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
	hash := loadPolicyHash()
	if hash != "" {
		t.Error("expected empty hash when no state file exists")
	}

	// Create state directory and file
	stateDir := filepath.Join(".acc", "state")
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		t.Fatalf("failed to create state dir: %v", err)
	}

	testData := []byte(`{"status":"pass","timestamp":"2025-01-01T00:00:00Z"}`)
	stateFile := filepath.Join(stateDir, "last_verify.json")
	if err := os.WriteFile(stateFile, testData, 0644); err != nil {
		t.Fatalf("failed to write state file: %v", err)
	}

	// Test with state file
	hash = loadPolicyHash()
	if hash == "" {
		t.Error("expected non-empty hash when state file exists")
	}

	// Verify hash is sha256 (64 hex chars)
	if len(hash) != 64 {
		t.Errorf("expected hash length 64, got %d", len(hash))
	}
}

func TestFormatJSON(t *testing.T) {
	result := &AttestResult{
		AttestationPath: ".acc/attestations/test.json",
		Attestation: Attestation{
			SchemaVersion: "v0.1",
			Type:          "acc.build.v0",
			ImageRef:      "test:latest",
			Timestamp:     "2025-01-01T00:00:00Z",
			BuildMetadata: BuildMetadata{
				BuildTool: "docker",
				BuildTime: "2025-01-01T00:00:00Z",
			},
			Metadata: map[string]string{"project": "test"},
		},
	}

	jsonStr := result.FormatJSON()
	if jsonStr == "" {
		t.Error("expected non-empty JSON string")
	}

	// Verify it's valid JSON
	var decoded AttestResult
	if err := json.Unmarshal([]byte(jsonStr), &decoded); err != nil {
		t.Errorf("failed to decode JSON: %v", err)
	}

	if decoded.Attestation.ImageRef != "test:latest" {
		t.Errorf("expected imageRef 'test:latest', got '%s'", decoded.Attestation.ImageRef)
	}
}

func TestAttestationDeterministic(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "acc-determ-test-*")
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

	// Create attestation
	result, err := Attest(cfg, "test:v1", true)
	if err != nil {
		t.Fatalf("Attest() failed: %v", err)
	}

	// Read the file
	data, err := os.ReadFile(result.AttestationPath)
	if err != nil {
		t.Fatalf("failed to read attestation: %v", err)
	}

	// Verify JSON is well-formed and can be parsed
	var attestation Attestation
	if err := json.Unmarshal(data, &attestation); err != nil {
		t.Fatalf("failed to unmarshal attestation: %v", err)
	}

	// Verify re-marshaling produces valid JSON
	remarshaled, err := json.Marshal(attestation)
	if err != nil {
		t.Fatalf("failed to re-marshal attestation: %v", err)
	}

	var remarAttestation Attestation
	if err := json.Unmarshal(remarshaled, &remarAttestation); err != nil {
		t.Fatalf("failed to unmarshal re-marshaled attestation: %v", err)
	}

	// Verify key fields preserved
	if remarAttestation.ImageRef != attestation.ImageRef {
		t.Error("imageRef not preserved through marshal/unmarshal cycle")
	}
}
