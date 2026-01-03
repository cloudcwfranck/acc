package trust

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudcwfranck/acc/internal/crypto"
	"github.com/cloudcwfranck/acc/internal/ui"
)

// VerifyResult represents attestation verification output
type VerifyResult struct {
	SchemaVersion      string              `json:"schemaVersion"`
	ImageRef           string              `json:"imageRef"`
	ImageDigest        string              `json:"imageDigest"`
	VerificationStatus string              `json:"verificationStatus"` // verified, unverified, unknown
	AttestationCount   int                 `json:"attestationCount"`
	Attestations       []AttestationDetail `json:"attestations"`
	Errors             []string            `json:"errors"`
}

// AttestationDetail represents details about a single attestation
type AttestationDetail struct {
	Path                    string `json:"path"`
	Timestamp               string `json:"timestamp"`
	VerificationStatus      string `json:"verificationStatus"`
	VerificationResultsHash string `json:"verificationResultsHash"`
	ValidSchema             bool   `json:"validSchema"`
	DigestMatch             bool   `json:"digestMatch"`
}

// VerifyAttestations verifies attestations for an image
// v0.3.0: Local-only, read-only attestation verification
// v0.3.2: optionally fetch from remote registry when remote=true
func VerifyAttestations(imageRef string, remote, outputJSON bool) (*VerifyResult, error) {
	result := &VerifyResult{
		SchemaVersion:      "v0.3",
		ImageRef:           imageRef,
		ImageDigest:        "",
		VerificationStatus: "unknown",
		AttestationCount:   0,
		Attestations:       []AttestationDetail{},
		Errors:             []string{},
	}

	// Step 1: Resolve image digest
	digest, err := resolveImageDigest(imageRef)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Cannot resolve digest: %v", err))
		result.VerificationStatus = "unknown"
		if !outputJSON {
			fmt.Fprintf(os.Stderr, "Error: Cannot resolve digest for %s\n", imageRef)
			fmt.Fprintf(os.Stderr, "Remediation: Ensure image exists locally (docker pull %s)\n", imageRef)
		}
		return result, fmt.Errorf("cannot resolve digest")
	}
	result.ImageDigest = digest

	// v0.3.2: Optionally fetch remote attestations before finding local ones
	if remote {
		if err := fetchRemoteAttestations(imageRef, digest, outputJSON); err != nil {
			// Remote fetch failed - log warning but don't fail
			// This preserves local-only workflow when network unavailable
			if !outputJSON {
				fmt.Fprintf(os.Stderr, "Warning: Failed to fetch remote attestations: %v\n", err)
			}
		}
	}

	// Step 2: Find attestations for this digest
	// v0.3.2: This now includes both local and remote-cached attestations
	attestPaths := findAttestationsForImage(digest)
	result.AttestationCount = len(attestPaths)

	if len(attestPaths) == 0 {
		result.VerificationStatus = "unverified"
		result.Errors = append(result.Errors, "No attestations found")
		if !outputJSON {
			fmt.Fprintf(os.Stderr, "No attestations found for %s\n", imageRef)
			fmt.Fprintf(os.Stderr, "Remediation: Run 'acc verify %s && acc attest %s'\n", imageRef, imageRef)
		}
		return result, fmt.Errorf("no attestations found")
	}

	// Step 3: Validate each attestation
	allValid := true
	for _, path := range attestPaths {
		detail := validateAttestation(path, digest)
		result.Attestations = append(result.Attestations, detail)

		if !detail.ValidSchema || !detail.DigestMatch {
			allValid = false
			result.Errors = append(result.Errors,
				fmt.Sprintf("Invalid attestation: %s (schema=%t, digest=%t)",
					filepath.Base(path), detail.ValidSchema, detail.DigestMatch))
		}
	}

	// Step 4: Determine overall status
	if allValid {
		result.VerificationStatus = "verified"
	} else {
		result.VerificationStatus = "unverified"
		return result, fmt.Errorf("attestation validation failed")
	}

	if !outputJSON {
		printHumanVerifyResult(result)
	}

	return result, nil
}

// validateAttestation validates a single attestation file
// Supports both legacy (v0.1) and envelope (v0.3.3) formats
func validateAttestation(path, expectedDigest string) AttestationDetail {
	detail := AttestationDetail{
		Path:        path,
		ValidSchema: false,
		DigestMatch: false,
	}

	// Read attestation file
	data, err := os.ReadFile(path)
	if err != nil {
		return detail
	}

	// Parse JSON to detect format
	var topLevel map[string]interface{}
	if err := json.Unmarshal(data, &topLevel); err != nil {
		return detail
	}

	// Check if this is envelope format (has "attestation" and "envelope" fields)
	var attest map[string]interface{}
	var envelope map[string]interface{}

	if attestField, hasAttestation := topLevel["attestation"]; hasAttestation {
		if envelopeField, hasEnvelope := topLevel["envelope"]; hasEnvelope {
			// v0.3.3 envelope format
			if attestMap, ok := attestField.(map[string]interface{}); ok {
				attest = attestMap
			} else {
				return detail
			}
			if envMap, ok := envelopeField.(map[string]interface{}); ok {
				envelope = envMap
			} else {
				return detail
			}
		} else {
			// Has "attestation" but no "envelope" - invalid
			return detail
		}
	} else {
		// Legacy format - top level IS the attestation
		attest = topLevel
	}

	// Extract fields from attestation object
	if timestamp, ok := attest["timestamp"].(string); ok {
		detail.Timestamp = timestamp
	}

	if evidence, ok := attest["evidence"].(map[string]interface{}); ok {
		if status, ok := evidence["verificationStatus"].(string); ok {
			detail.VerificationStatus = status
		}
		if hash, ok := evidence["verificationResultsHash"].(string); ok {
			detail.VerificationResultsHash = hash
		}
	}

	// Validate schema (basic check for required fields in attestation object)
	requiredFields := []string{"schemaVersion", "timestamp", "subject", "evidence"}
	detail.ValidSchema = true
	for _, field := range requiredFields {
		if _, ok := attest[field]; !ok {
			detail.ValidSchema = false
			break
		}
	}

	// Validate digest match with normalization
	if subject, ok := attest["subject"].(map[string]interface{}); ok {
		if attestDigest, ok := subject["imageDigest"].(string); ok {
			// Normalize both digests for comparison (handles sha256: prefix variations)
			detail.DigestMatch = (normalizeDigest(attestDigest) == normalizeDigest(expectedDigest))
		}
	}

	// If envelope exists, verify signature
	if envelope != nil {
		if !verifyEnvelopeSignature(attest, envelope) {
			// Signature verification failed - mark as invalid
			detail.ValidSchema = false
			return detail
		}
		// Signature verified - attestation is valid
	}

	return detail
}

// verifyEnvelopeSignature verifies the envelope signature for v0.3.3 attestations
func verifyEnvelopeSignature(attestation, envelope map[string]interface{}) bool {
	// Extract envelope fields
	alg, _ := envelope["alg"].(string)
	keyID, _ := envelope["keyId"].(string)
	publicKeyB64, _ := envelope["publicKey"].(string)
	canon, _ := envelope["canon"].(string)
	payloadHash, _ := envelope["payloadHash"].(string)
	signatureB64, _ := envelope["signature"].(string)

	// Validate required fields
	if alg != "ed25519" {
		return false
	}
	if canon != "jcs" {
		return false
	}
	if publicKeyB64 == "" || signatureB64 == "" || payloadHash == "" || keyID == "" {
		return false
	}

	// Decode public key
	publicKeyBytes, err := base64.StdEncoding.DecodeString(publicKeyB64)
	if err != nil || len(publicKeyBytes) != ed25519.PublicKeySize {
		return false
	}
	publicKey := ed25519.PublicKey(publicKeyBytes)

	// Verify keyId matches public key
	expectedKeyID := crypto.KeyIDFromPublicKeyEd25519(publicKey)
	if keyID != expectedKeyID {
		return false
	}

	// Canonicalize attestation object
	canonicalPayload, err := crypto.CanonicalizeJCS(attestation)
	if err != nil {
		return false
	}

	// Verify payload hash
	payloadHashBytes := sha256.Sum256(canonicalPayload)
	expectedPayloadHash := fmt.Sprintf("sha256:%x", payloadHashBytes)
	if payloadHash != expectedPayloadHash {
		return false
	}

	// Decode signature
	signatureBytes, err := base64.StdEncoding.DecodeString(signatureB64)
	if err != nil || len(signatureBytes) != ed25519.SignatureSize {
		return false
	}

	// Verify signature
	if !ed25519.Verify(publicKey, canonicalPayload, signatureBytes) {
		return false
	}

	// All checks passed
	return true
}

// normalizeDigest normalizes a digest string for comparison
// - Trims whitespace
// - Lowercases
// - Strips optional "sha256:" prefix
func normalizeDigest(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ToLower(s)
	s = strings.TrimPrefix(s, "sha256:")
	return s
}

// printHumanVerifyResult prints human-readable verification result
func printHumanVerifyResult(result *VerifyResult) {
	ui.PrintTrust("Attestation Verification")
	fmt.Println()

	// Image information
	fmt.Printf("Image:       %s\n", result.ImageRef)
	if len(result.ImageDigest) >= 12 {
		fmt.Printf("Digest:      sha256:%s...\n", result.ImageDigest[:12])
	} else {
		fmt.Printf("Digest:      %s\n", result.ImageDigest)
	}
	fmt.Println()

	// Verification status with icon
	statusIcon := "❓"
	switch result.VerificationStatus {
	case "verified":
		statusIcon = ui.SymbolSuccess
	case "unverified":
		statusIcon = ui.SymbolFailure
	case "unknown":
		statusIcon = ui.SymbolWarning
	}
	fmt.Printf("Status:      %s %s\n", statusIcon, result.VerificationStatus)
	fmt.Printf("Attestations: %d found\n", result.AttestationCount)
	fmt.Println()

	// Attestation details
	if len(result.Attestations) > 0 {
		fmt.Println("Details:")
		for i, att := range result.Attestations {
			fmt.Printf("\n  [%d] %s\n", i+1, filepath.Base(att.Path))
			fmt.Printf("      Timestamp:   %s\n", att.Timestamp)
			fmt.Printf("      Status:      %s\n", att.VerificationStatus)
			if att.ValidSchema && att.DigestMatch {
				ui.PrintSuccess(fmt.Sprintf("      Valid:       ✓ (schema=%t, digest=%t)",
					att.ValidSchema, att.DigestMatch))
			} else {
				ui.PrintError(fmt.Sprintf("      Valid:       ✗ (schema=%t, digest=%t)",
					att.ValidSchema, att.DigestMatch))
			}
		}
		fmt.Println()
	}

	// Errors
	if len(result.Errors) > 0 {
		fmt.Println("Errors:")
		for _, err := range result.Errors {
			ui.PrintError(fmt.Sprintf("  - %s", err))
		}
		fmt.Println()
	}
}

// FormatJSON formats verify result as JSON
func (vr *VerifyResult) FormatJSON() string {
	data, _ := json.MarshalIndent(vr, "", "  ")
	return string(data)
}

// ExitCode returns appropriate exit code
// v0.3.0: 0=verified, 1=unverified, 2=unknown
func (vr *VerifyResult) ExitCode() int {
	switch vr.VerificationStatus {
	case "verified":
		return 0 // Success
	case "unverified":
		return 1 // Failure
	case "unknown":
		return 2 // Cannot complete
	default:
		return 2
	}
}
