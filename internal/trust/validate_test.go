package trust

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestValidateAttestationWithResultsHash tests results hash validation
func TestValidateAttestationWithResultsHash(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)

	tests := []struct {
		name                    string
		attestation             map[string]interface{}
		expectedDigest          string
		expectedResultsHash     string
		expectedValidSchema     bool
		expectedDigestMatch     bool
		expectedResultsHashMatch bool
		expectedReason          string
	}{
		{
			name: "valid attestation with matching results hash",
			attestation: map[string]interface{}{
				"schemaVersion": "v0.1",
				"timestamp":     "2025-01-01T00:00:00Z",
				"subject": map[string]interface{}{
					"imageRef":    "test:latest",
					"imageDigest": "abc123",
				},
				"evidence": map[string]interface{}{
					"verificationStatus":      "pass",
					"verificationResultsHash": "sha256:def456",
				},
			},
			expectedDigest:          "abc123",
			expectedResultsHash:     "sha256:def456",
			expectedValidSchema:     true,
			expectedDigestMatch:     true,
			expectedResultsHashMatch: true,
			expectedReason:          "",
		},
		{
			name: "attestation with mismatched results hash",
			attestation: map[string]interface{}{
				"schemaVersion": "v0.1",
				"timestamp":     "2025-01-01T00:00:00Z",
				"subject": map[string]interface{}{
					"imageRef":    "test:latest",
					"imageDigest": "abc123",
				},
				"evidence": map[string]interface{}{
					"verificationStatus":      "pass",
					"verificationResultsHash": "sha256:wrong",
				},
			},
			expectedDigest:          "abc123",
			expectedResultsHash:     "sha256:def456",
			expectedValidSchema:     true,
			expectedDigestMatch:     true,
			expectedResultsHashMatch: false,
			expectedReason:          "results hash mismatch",
		},
		{
			name: "attestation with missing results hash",
			attestation: map[string]interface{}{
				"schemaVersion": "v0.1",
				"timestamp":     "2025-01-01T00:00:00Z",
				"subject": map[string]interface{}{
					"imageRef":    "test:latest",
					"imageDigest": "abc123",
				},
				"evidence": map[string]interface{}{
					"verificationStatus": "pass",
				},
			},
			expectedDigest:          "abc123",
			expectedResultsHash:     "sha256:def456",
			expectedValidSchema:     true,
			expectedDigestMatch:     true,
			expectedResultsHashMatch: false,
			expectedReason:          "missing results hash",
		},
		{
			name: "attestation with mismatched digest",
			attestation: map[string]interface{}{
				"schemaVersion": "v0.1",
				"timestamp":     "2025-01-01T00:00:00Z",
				"subject": map[string]interface{}{
					"imageRef":    "test:latest",
					"imageDigest": "wrong",
				},
				"evidence": map[string]interface{}{
					"verificationStatus":      "pass",
					"verificationResultsHash": "sha256:def456",
				},
			},
			expectedDigest:          "abc123",
			expectedResultsHash:     "sha256:def456",
			expectedValidSchema:     true,
			expectedDigestMatch:     false,
			expectedResultsHashMatch: true,
			expectedReason:          "digest mismatch",
		},
		{
			name: "attestation with invalid schema (missing timestamp)",
			attestation: map[string]interface{}{
				"schemaVersion": "v0.1",
				"subject": map[string]interface{}{
					"imageRef":    "test:latest",
					"imageDigest": "abc123",
				},
				"evidence": map[string]interface{}{
					"verificationStatus":      "pass",
					"verificationResultsHash": "sha256:def456",
				},
			},
			expectedDigest:          "abc123",
			expectedResultsHash:     "sha256:def456",
			expectedValidSchema:     false,
			expectedDigestMatch:     true,
			expectedResultsHashMatch: true,
			expectedReason:          "invalid schema",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Write attestation to temp file
			attPath := filepath.Join(tmpDir, "test_attestation.json")
			data, _ := json.Marshal(tt.attestation)
			os.WriteFile(attPath, data, 0644)

			// Validate
			detail := ValidateAttestationWithHash(attPath, tt.expectedDigest, tt.expectedResultsHash)

			if detail.ValidSchema != tt.expectedValidSchema {
				t.Errorf("ValidSchema = %v, want %v", detail.ValidSchema, tt.expectedValidSchema)
			}
			if detail.DigestMatch != tt.expectedDigestMatch {
				t.Errorf("DigestMatch = %v, want %v", detail.DigestMatch, tt.expectedDigestMatch)
			}
			if detail.ResultsHashMatch != tt.expectedResultsHashMatch {
				t.Errorf("ResultsHashMatch = %v, want %v", detail.ResultsHashMatch, tt.expectedResultsHashMatch)
			}
			if tt.expectedReason != "" && detail.InvalidReason != tt.expectedReason {
				t.Errorf("InvalidReason = %q, want %q", detail.InvalidReason, tt.expectedReason)
			}
		})
	}
}

// TestMultipleAttestationsWithCounts tests handling multiple attestations
func TestMultipleAttestationsWithCounts(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)

	// Create .acc structure
	os.MkdirAll(filepath.Join(".acc", "attestations", "abc123", "local"), 0755)

	// Create 3 attestations: 2 valid, 1 invalid
	attestations := []struct {
		filename string
		data     map[string]interface{}
	}{
		{
			filename: "valid1.json",
			data: map[string]interface{}{
				"schemaVersion": "v0.1",
				"timestamp":     "2025-01-01T00:00:00Z",
				"subject": map[string]interface{}{
					"imageRef":    "test:latest",
					"imageDigest": "abc123",
				},
				"evidence": map[string]interface{}{
					"verificationStatus":      "pass",
					"verificationResultsHash": "sha256:correct",
				},
			},
		},
		{
			filename: "valid2.json",
			data: map[string]interface{}{
				"schemaVersion": "v0.1",
				"timestamp":     "2025-01-01T01:00:00Z",
				"subject": map[string]interface{}{
					"imageRef":    "test:latest",
					"imageDigest": "abc123",
				},
				"evidence": map[string]interface{}{
					"verificationStatus":      "pass",
					"verificationResultsHash": "sha256:correct",
				},
			},
		},
		{
			filename: "invalid.json",
			data: map[string]interface{}{
				"schemaVersion": "v0.1",
				"timestamp":     "2025-01-01T02:00:00Z",
				"subject": map[string]interface{}{
					"imageRef":    "test:latest",
					"imageDigest": "wrong",
				},
				"evidence": map[string]interface{}{
					"verificationStatus":      "pass",
					"verificationResultsHash": "sha256:correct",
				},
			},
		},
	}

	for _, att := range attestations {
		path := filepath.Join(".acc", "attestations", "abc123", "local", att.filename)
		data, _ := json.Marshal(att.data)
		os.WriteFile(path, data, 0644)
	}

	// Test evaluation
	result, err := EvaluateAttestations("abc123", "sha256:correct", nil)
	if err != nil {
		t.Fatalf("EvaluateAttestations failed: %v", err)
	}

	if result.TotalCount != 3 {
		t.Errorf("TotalCount = %d, want 3", result.TotalCount)
	}
	if result.ValidCount != 2 {
		t.Errorf("ValidCount = %d, want 2", result.ValidCount)
	}
	if result.InvalidCount != 1 {
		t.Errorf("InvalidCount = %d, want 1", result.InvalidCount)
	}
}
