package trust

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

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
func VerifyAttestations(imageRef string, outputJSON bool) (*VerifyResult, error) {
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

	// Step 2: Find attestations for this digest
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

	// Parse JSON
	var attest map[string]interface{}
	if err := json.Unmarshal(data, &attest); err != nil {
		return detail
	}

	// Extract fields
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

	// Validate schema (basic check for required fields)
	requiredFields := []string{"schemaVersion", "timestamp", "subject", "evidence"}
	detail.ValidSchema = true
	for _, field := range requiredFields {
		if _, ok := attest[field]; !ok {
			detail.ValidSchema = false
			break
		}
	}

	// Validate digest match
	if subject, ok := attest["subject"].(map[string]interface{}); ok {
		if attestDigest, ok := subject["imageDigest"].(string); ok {
			// Both digests should match (without sha256: prefix if present)
			detail.DigestMatch = (attestDigest == expectedDigest)
		}
	}

	return detail
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
