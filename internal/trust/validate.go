package trust

import (
	"encoding/json"
	"os"
	"sort"
)

// AttestationValidityDetail extends AttestationDetail with v0.3.3 validity fields
type AttestationValidityDetail struct {
	AttestationDetail
	ResultsHashMatch bool   `json:"resultsHashMatch"` // v0.3.3: verification results hash matches
	InvalidReason    string `json:"invalidReason"`    // v0.3.3: reason if invalid
}

// AttestationEvaluationResult represents the evaluation of all attestations
type AttestationEvaluationResult struct {
	TotalCount   int                         `json:"totalCount"`
	ValidCount   int                         `json:"validCount"`
	InvalidCount int                         `json:"invalidCount"`
	Attestations []AttestationValidityDetail `json:"attestations"`
}

// ValidateAttestationWithHash validates a single attestation with results hash checking
// v0.3.3: Adds verification results hash validation for integrity binding
func ValidateAttestationWithHash(path, expectedDigest, expectedResultsHash string) AttestationValidityDetail {
	detail := AttestationValidityDetail{
		AttestationDetail: AttestationDetail{
			Path:        path,
			ValidSchema: false,
			DigestMatch: false,
		},
		ResultsHashMatch: false,
		InvalidReason:    "",
	}

	// Read attestation file
	data, err := os.ReadFile(path)
	if err != nil {
		detail.InvalidReason = "cannot read file"
		return detail
	}

	// Parse JSON
	var attest map[string]interface{}
	if err := json.Unmarshal(data, &attest); err != nil {
		detail.InvalidReason = "invalid JSON"
		return detail
	}

	// Extract timestamp
	if timestamp, ok := attest["timestamp"].(string); ok {
		detail.Timestamp = timestamp
	}

	// Extract evidence fields
	if evidence, ok := attest["evidence"].(map[string]interface{}); ok {
		if status, ok := evidence["verificationStatus"].(string); ok {
			detail.VerificationStatus = status
		}
		if hash, ok := evidence["verificationResultsHash"].(string); ok {
			detail.VerificationResultsHash = hash
		}
	}

	// Validate schema (required fields)
	requiredFields := []string{"schemaVersion", "timestamp", "subject", "evidence"}
	detail.ValidSchema = true
	for _, field := range requiredFields {
		if _, ok := attest[field]; !ok {
			detail.ValidSchema = false
			detail.InvalidReason = "invalid schema"
			break
		}
	}

	// Validate digest match
	if subject, ok := attest["subject"].(map[string]interface{}); ok {
		if attestDigest, ok := subject["imageDigest"].(string); ok {
			detail.DigestMatch = (attestDigest == expectedDigest)
			if !detail.DigestMatch && detail.InvalidReason == "" {
				detail.InvalidReason = "digest mismatch"
			}
		}
	}

	// v0.3.3: Validate results hash match (integrity binding)
	if detail.VerificationResultsHash == "" {
		detail.ResultsHashMatch = false
		if detail.InvalidReason == "" {
			detail.InvalidReason = "missing results hash"
		}
	} else if expectedResultsHash != "" {
		detail.ResultsHashMatch = (detail.VerificationResultsHash == expectedResultsHash)
		if !detail.ResultsHashMatch && detail.InvalidReason == "" {
			detail.InvalidReason = "results hash mismatch"
		}
	} else {
		// No expected hash provided - cannot validate
		detail.ResultsHashMatch = false
	}

	return detail
}

// EvaluateAttestations evaluates all attestations for an image with validity checks
// v0.3.3: Returns counts of valid vs invalid attestations
func EvaluateAttestations(digest, expectedResultsHash string, attestPaths []string) (*AttestationEvaluationResult, error) {
	result := &AttestationEvaluationResult{
		TotalCount:   0,
		ValidCount:   0,
		InvalidCount: 0,
		Attestations: []AttestationValidityDetail{},
	}

	// If no paths provided, find them
	if attestPaths == nil {
		attestPaths = findAttestationsForImage(digest)
	}

	result.TotalCount = len(attestPaths)

	// Evaluate each attestation
	for _, path := range attestPaths {
		detail := ValidateAttestationWithHash(path, digest, expectedResultsHash)
		result.Attestations = append(result.Attestations, detail)

		// Count as valid only if all checks pass
		if detail.ValidSchema && detail.DigestMatch && detail.ResultsHashMatch {
			result.ValidCount++
		} else {
			result.InvalidCount++
		}
	}

	// Sort attestations deterministically (by timestamp, then path)
	sort.Slice(result.Attestations, func(i, j int) bool {
		if result.Attestations[i].Timestamp != result.Attestations[j].Timestamp {
			return result.Attestations[i].Timestamp < result.Attestations[j].Timestamp
		}
		return result.Attestations[i].Path < result.Attestations[j].Path
	})

	return result, nil
}

// IsAttestationValid checks if an attestation passes all validity checks
func IsAttestationValid(detail AttestationValidityDetail) bool {
	return detail.ValidSchema && detail.DigestMatch && detail.ResultsHashMatch
}
