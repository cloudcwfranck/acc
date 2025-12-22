package trust

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestVerifyResultExitCodes(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		expected int
	}{
		{"verified returns 0", "verified", 0},
		{"unverified returns 1", "unverified", 1},
		{"unknown returns 2", "unknown", 2},
		{"invalid status defaults to 2", "invalid", 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &VerifyResult{VerificationStatus: tt.status}
			if got := result.ExitCode(); got != tt.expected {
				t.Errorf("ExitCode() = %d, want %d", got, tt.expected)
			}
		})
	}
}

func TestVerifyResultJSONContract(t *testing.T) {
	// Test all required fields are present and never null
	result := &VerifyResult{
		SchemaVersion:      "v0.3",
		ImageRef:           "test:latest",
		ImageDigest:        "abc123def456",
		VerificationStatus: "verified",
		AttestationCount:   1,
		Attestations: []AttestationDetail{
			{
				Path:                    ".acc/attestations/abc123/20250101-120000-attestation.json",
				Timestamp:               "2025-01-01T12:00:00Z",
				VerificationStatus:      "pass",
				VerificationResultsHash: "sha256:abc123",
				ValidSchema:             true,
				DigestMatch:             true,
			},
		},
		Errors: []string{},
	}

	jsonStr := result.FormatJSON()
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	// Verify required fields exist
	requiredFields := []string{
		"schemaVersion", "imageRef", "imageDigest",
		"verificationStatus", "attestationCount",
		"attestations", "errors",
	}

	for _, field := range requiredFields {
		if _, ok := parsed[field]; !ok {
			t.Errorf("JSON missing required field: %s", field)
		}
	}

	// Verify field types
	if schemaVersion, ok := parsed["schemaVersion"].(string); !ok || schemaVersion != "v0.3" {
		t.Errorf("schemaVersion should be 'v0.3', got %v", parsed["schemaVersion"])
	}

	if attestations, ok := parsed["attestations"].([]interface{}); !ok {
		t.Errorf("attestations should be array, got %T", parsed["attestations"])
	} else if len(attestations) != 1 {
		t.Errorf("attestations should have 1 element, got %d", len(attestations))
	}

	if errors, ok := parsed["errors"].([]interface{}); !ok {
		t.Errorf("errors should be array, got %T", parsed["errors"])
	} else if len(errors) != 0 {
		t.Errorf("errors should be empty array, got %d elements", len(errors))
	}
}

func TestVerifyResultEmptyArraysNeverNull(t *testing.T) {
	// Test that empty arrays are never null (contract requirement)
	result := &VerifyResult{
		SchemaVersion:      "v0.3",
		ImageRef:           "test:latest",
		ImageDigest:        "",
		VerificationStatus: "unknown",
		AttestationCount:   0,
		Attestations:       []AttestationDetail{}, // Empty array, not null
		Errors:             []string{},            // Empty array, not null
	}

	jsonStr := result.FormatJSON()
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	// Verify attestations is an empty array, not null
	if attestations, ok := parsed["attestations"].([]interface{}); !ok {
		t.Errorf("attestations should be array (even if empty), got %T", parsed["attestations"])
	} else if attestations == nil {
		t.Errorf("attestations should not be null, should be empty array")
	}

	// Verify errors is an empty array, not null
	if errors, ok := parsed["errors"].([]interface{}); !ok {
		t.Errorf("errors should be array (even if empty), got %T", parsed["errors"])
	} else if errors == nil {
		t.Errorf("errors should not be null, should be empty array")
	}
}

func TestValidateAttestation(t *testing.T) {
	// Create a temporary directory for test attestations
	tmpDir := t.TempDir()

	// Test case 1: Valid attestation
	t.Run("valid attestation", func(t *testing.T) {
		validAttest := map[string]interface{}{
			"schemaVersion": "v0.1",
			"timestamp":     "2025-01-01T12:00:00Z",
			"subject": map[string]interface{}{
				"imageRef":    "test:latest",
				"imageDigest": "abc123def456",
			},
			"evidence": map[string]interface{}{
				"verificationStatus":      "pass",
				"verificationResultsHash": "sha256:xyz789",
			},
		}

		attestPath := filepath.Join(tmpDir, "valid.json")
		data, _ := json.MarshalIndent(validAttest, "", "  ")
		if err := os.WriteFile(attestPath, data, 0644); err != nil {
			t.Fatalf("Failed to write test attestation: %v", err)
		}

		detail := validateAttestation(attestPath, "abc123def456")

		if !detail.ValidSchema {
			t.Errorf("Expected ValidSchema=true, got false")
		}
		if !detail.DigestMatch {
			t.Errorf("Expected DigestMatch=true, got false")
		}
		if detail.Timestamp != "2025-01-01T12:00:00Z" {
			t.Errorf("Expected timestamp='2025-01-01T12:00:00Z', got '%s'", detail.Timestamp)
		}
		if detail.VerificationStatus != "pass" {
			t.Errorf("Expected verificationStatus='pass', got '%s'", detail.VerificationStatus)
		}
	})

	// Test case 2: Digest mismatch
	t.Run("digest mismatch", func(t *testing.T) {
		mismatchAttest := map[string]interface{}{
			"schemaVersion": "v0.1",
			"timestamp":     "2025-01-01T12:00:00Z",
			"subject": map[string]interface{}{
				"imageRef":    "test:latest",
				"imageDigest": "wrongdigest",
			},
			"evidence": map[string]interface{}{
				"verificationStatus":      "pass",
				"verificationResultsHash": "sha256:xyz789",
			},
		}

		attestPath := filepath.Join(tmpDir, "mismatch.json")
		data, _ := json.MarshalIndent(mismatchAttest, "", "  ")
		if err := os.WriteFile(attestPath, data, 0644); err != nil {
			t.Fatalf("Failed to write test attestation: %v", err)
		}

		detail := validateAttestation(attestPath, "abc123def456")

		if !detail.ValidSchema {
			t.Errorf("Expected ValidSchema=true, got false")
		}
		if detail.DigestMatch {
			t.Errorf("Expected DigestMatch=false (digest mismatch), got true")
		}
	})

	// Test case 3: Invalid schema (missing required fields)
	t.Run("invalid schema", func(t *testing.T) {
		invalidAttest := map[string]interface{}{
			"schemaVersion": "v0.1",
			// Missing timestamp, subject, evidence
		}

		attestPath := filepath.Join(tmpDir, "invalid.json")
		data, _ := json.MarshalIndent(invalidAttest, "", "  ")
		if err := os.WriteFile(attestPath, data, 0644); err != nil {
			t.Fatalf("Failed to write test attestation: %v", err)
		}

		detail := validateAttestation(attestPath, "abc123def456")

		if detail.ValidSchema {
			t.Errorf("Expected ValidSchema=false (missing required fields), got true")
		}
	})

	// Test case 4: File not found
	t.Run("file not found", func(t *testing.T) {
		detail := validateAttestation("/nonexistent/path.json", "abc123def456")

		if detail.ValidSchema {
			t.Errorf("Expected ValidSchema=false (file not found), got true")
		}
		if detail.DigestMatch {
			t.Errorf("Expected DigestMatch=false (file not found), got true")
		}
	})
}

func TestAttestationDetailJSONFields(t *testing.T) {
	// Verify all fields in AttestationDetail are serialized correctly
	detail := AttestationDetail{
		Path:                    ".acc/attestations/abc123/test.json",
		Timestamp:               "2025-01-01T12:00:00Z",
		VerificationStatus:      "pass",
		VerificationResultsHash: "sha256:abc123",
		ValidSchema:             true,
		DigestMatch:             true,
	}

	data, err := json.Marshal(detail)
	if err != nil {
		t.Fatalf("Failed to marshal AttestationDetail: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal AttestationDetail: %v", err)
	}

	// Check all expected fields
	expectedFields := []string{
		"path", "timestamp", "verificationStatus",
		"verificationResultsHash", "validSchema", "digestMatch",
	}

	for _, field := range expectedFields {
		if _, ok := parsed[field]; !ok {
			t.Errorf("AttestationDetail JSON missing field: %s", field)
		}
	}

	// Check boolean field types
	if validSchema, ok := parsed["validSchema"].(bool); !ok || !validSchema {
		t.Errorf("validSchema should be bool true, got %v", parsed["validSchema"])
	}

	if digestMatch, ok := parsed["digestMatch"].(bool); !ok || !digestMatch {
		t.Errorf("digestMatch should be bool true, got %v", parsed["digestMatch"])
	}
}
