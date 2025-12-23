package attest

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/cloudcwfranck/acc/internal/config"
	"github.com/cloudcwfranck/acc/internal/ui"
)

// Attestation represents the v0 attestation format
type Attestation struct {
	SchemaVersion string          `json:"schemaVersion"`
	Command       string          `json:"command"`
	Timestamp     string          `json:"timestamp"`
	Subject       Subject         `json:"subject"`
	Evidence      Evidence        `json:"evidence"`
	Metadata      AttestationMeta `json:"metadata"`
}

// Subject identifies what is being attested
type Subject struct {
	ImageRef    string `json:"imageRef"`
	ImageDigest string `json:"imageDigest,omitempty"`
}

// Evidence contains verification evidence
type Evidence struct {
	SBOMRef                 string `json:"sbomRef,omitempty"`
	PolicyPack              string `json:"policyPack"`
	PolicyMode              string `json:"policyMode"`
	VerificationStatus      string `json:"verificationStatus"`
	VerificationResultsHash string `json:"verificationResultsHash"`
}

// AttestationMeta contains tool metadata
type AttestationMeta struct {
	Tool        string `json:"tool"`
	ToolVersion string `json:"toolVersion"`
	GitCommit   string `json:"gitCommit,omitempty"`
}

// AttestResult represents the result of attestation creation
type AttestResult struct {
	OutputPath  string      `json:"outputPath"`
	Attestation Attestation `json:"attestation"`
}

// VerifyState represents the persisted verification state (reused from verify package)
type VerifyState struct {
	ImageRef  string                 `json:"imageRef"`
	Status    string                 `json:"status"`
	Timestamp string                 `json:"timestamp"`
	Result    map[string]interface{} `json:"result"`
}

// Attest creates an attestation for an image
// v0.3.2: optionally publish to remote registry when remote=true
func Attest(cfg *config.Config, imageRef, version, commit string, remote, outputJSON bool) (*AttestResult, error) {
	if imageRef == "" {
		return nil, fmt.Errorf("image reference required")
	}

	// Load last verification state
	verifyState, err := loadVerifyState()
	if err != nil {
		return nil, fmt.Errorf("verification state not found\n\nRemediation:\n  Run 'acc verify %s' first to generate verification results", imageRef)
	}

	// Verify imageRef matches last verified image
	if err := validateImageMatch(imageRef, verifyState); err != nil {
		return nil, err
	}

	// v0.1.5: Only print creation message AFTER validation passes
	if !outputJSON {
		ui.PrintInfo(fmt.Sprintf("Creating attestation for %s", imageRef))
	}

	// Resolve digest
	digest, err := resolveDigest(imageRef)
	if err != nil {
		if !outputJSON {
			ui.PrintWarning(fmt.Sprintf("Could not resolve digest: %v", err))
		}
		digest = ""
	}

	// Compute canonical hash of verification results
	resultsHash, err := computeCanonicalHash(verifyState)
	if err != nil {
		return nil, fmt.Errorf("failed to compute verification hash: %w", err)
	}

	// Get SBOM reference if available
	sbomRef := getSBOMRef(cfg)

	// Create attestation
	attestation := Attestation{
		SchemaVersion: "v0.1",
		Command:       "attest",
		Timestamp:     time.Now().UTC().Format(time.RFC3339),
		Subject: Subject{
			ImageRef:    imageRef,
			ImageDigest: digest,
		},
		Evidence: Evidence{
			SBOMRef:                 sbomRef,
			PolicyPack:              ".acc/policy",
			PolicyMode:              cfg.Policy.Mode,
			VerificationStatus:      verifyState.Status,
			VerificationResultsHash: resultsHash,
		},
		Metadata: AttestationMeta{
			Tool:        "acc",
			ToolVersion: version,
			GitCommit:   commit,
		},
	}

	// Determine output path
	outputPath, err := determineOutputPath(imageRef, digest)
	if err != nil {
		return nil, err
	}

	// Write attestation file
	if err := writeAttestation(outputPath, &attestation); err != nil {
		return nil, err
	}

	// Update last_attestation.json pointer
	if err := updateLastAttestationPointer(&attestation, outputPath); err != nil {
		if !outputJSON {
			ui.PrintWarning(fmt.Sprintf("Failed to update last attestation pointer: %v", err))
		}
	}

	if !outputJSON {
		ui.PrintSuccess("Attestation created")
		fmt.Printf("  Path:    %s\n", outputPath)
		fmt.Printf("  Subject: %s\n", imageRef)
		if digest != "" {
			fmt.Printf("  Digest:  sha256:%s\n", digest[:12])
		}
		fmt.Printf("  Hash:    %s\n", resultsHash[:16])
	}

	result := &AttestResult{
		OutputPath:  outputPath,
		Attestation: attestation,
	}

	// v0.3.2: Optionally publish attestation to remote registry
	if remote {
		if !outputJSON {
			ui.PrintInfo("Publishing attestation to remote registry...")
		}

		// Publish to remote OCI registry
		if err := publishAttestationToRegistry(imageRef, &attestation, outputJSON); err != nil {
			return nil, fmt.Errorf("failed to publish attestation to remote registry: %w", err)
		}

		if !outputJSON {
			ui.PrintSuccess("Attestation published to remote registry")
		}
	}

	return result, nil
}

// loadVerifyState loads the last verification state
func loadVerifyState() (*VerifyState, error) {
	stateFile := filepath.Join(".acc", "state", "last_verify.json")
	data, err := os.ReadFile(stateFile)
	if err != nil {
		return nil, err
	}

	var state VerifyState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to parse verification state: %w", err)
	}

	return &state, nil
}

// validateImageMatch ensures the imageRef matches the last verified image
// Testing Contract: attest MUST fail when target image != last verified image
func validateImageMatch(imageRef string, state *VerifyState) error {
	// CRITICAL: Always resolve digests for authoritative comparison
	// Digest comparison is more reliable than ref comparison (refs can alias)
	currentDigest, err1 := resolveDigest(imageRef)
	stateDigest, err2 := resolveDigest(state.ImageRef)

	// If both digests resolved, use digest comparison (authoritative)
	if err1 == nil && err2 == nil {
		if currentDigest == stateDigest {
			// Same image - allow attestation
			return nil
		}
		// Different digests = different images - MUST fail
		return fmt.Errorf("image mismatch: attempting to attest '%s' (digest: sha256:%s) but last verified image was '%s' (digest: sha256:%s)\n\nRemediation:\n  Run 'acc verify %s' first",
			imageRef, currentDigest[:12], state.ImageRef, stateDigest[:12], imageRef)
	}

	// Fallback: If digest resolution failed, compare refs as strings
	// This handles cases where image might not be in local cache
	if imageRef != state.ImageRef {
		return fmt.Errorf("image mismatch: attempting to attest '%s' but last verified image was '%s'\n\nRemediation:\n  Run 'acc verify %s' first",
			imageRef, state.ImageRef, imageRef)
	}

	// Same ref - allow attestation
	return nil
}

// computeCanonicalHash computes a canonical SHA256 hash of verification results
func computeCanonicalHash(state *VerifyState) (string, error) {
	// Extract violations and waivers from state
	result := state.Result
	if result == nil {
		result = make(map[string]interface{})
	}

	// Build canonical structure for hashing
	canonical := map[string]interface{}{
		"status":       state.Status,
		"violations":   extractAndSortViolations(result),
		"waivers":      extractAndSortWaivers(result),
		"sbomPresent":  result["sbomPresent"],
		"attestations": result["attestations"],
	}

	// Marshal with sorted keys (json.Marshal guarantees map key ordering)
	data, err := json.Marshal(canonical)
	if err != nil {
		return "", err
	}

	// Compute SHA256
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]), nil
}

// extractAndSortViolations extracts violations and sorts them canonically
func extractAndSortViolations(result map[string]interface{}) []map[string]interface{} {
	violations := []map[string]interface{}{}

	if v, ok := result["violations"].([]interface{}); ok {
		for _, item := range v {
			if violation, ok := item.(map[string]interface{}); ok {
				violations = append(violations, violation)
			}
		}
	}

	// Sort by rule, then severity for deterministic ordering
	sort.Slice(violations, func(i, j int) bool {
		ruleI, _ := violations[i]["rule"].(string)
		ruleJ, _ := violations[j]["rule"].(string)
		if ruleI != ruleJ {
			return ruleI < ruleJ
		}
		sevI, _ := violations[i]["severity"].(string)
		sevJ, _ := violations[j]["severity"].(string)
		return sevI < sevJ
	})

	return violations
}

// extractAndSortWaivers extracts waivers and sorts them canonically
func extractAndSortWaivers(result map[string]interface{}) []map[string]interface{} {
	waivers := []map[string]interface{}{}

	if policyResult, ok := result["policyResult"].(map[string]interface{}); ok {
		if w, ok := policyResult["waivers"].([]interface{}); ok {
			for _, item := range w {
				if waiver, ok := item.(map[string]interface{}); ok {
					waivers = append(waivers, waiver)
				}
			}
		}
	}

	// Sort by ruleId for deterministic ordering
	sort.Slice(waivers, func(i, j int) bool {
		ruleI, _ := waivers[i]["ruleId"].(string)
		ruleJ, _ := waivers[j]["ruleId"].(string)
		return ruleI < ruleJ
	})

	return waivers
}

// getSBOMRef returns the SBOM reference if available
func getSBOMRef(cfg *config.Config) string {
	sbomDir := filepath.Join(".acc", "sbom")
	sbomFile := filepath.Join(sbomDir, fmt.Sprintf("%s.%s.json", cfg.Project.Name, cfg.SBOM.Format))

	if _, err := os.Stat(sbomFile); err == nil {
		return sbomFile
	}

	return ""
}

// determineOutputPath determines where to write the attestation
func determineOutputPath(imageRef, digest string) (string, error) {
	// Sanitize imageRef for use as directory name
	sanitized := sanitizeRef(imageRef)

	// Use digest if available, otherwise sanitized ref
	dirName := sanitized
	if digest != "" {
		dirName = digest[:12] // Use first 12 chars of digest
	}

	// Create directory structure
	attestDir := filepath.Join(".acc", "attestations", dirName)
	if err := os.MkdirAll(attestDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create attestation directory: %w", err)
	}

	// Generate filename with timestamp
	timestamp := time.Now().UTC().Format("20060102-150405")
	filename := fmt.Sprintf("%s-attestation.json", timestamp)

	return filepath.Join(attestDir, filename), nil
}

// sanitizeRef sanitizes an image reference for use as a directory name
func sanitizeRef(ref string) string {
	// Remove registry prefix
	parts := strings.Split(ref, "/")
	name := parts[len(parts)-1]

	// Remove tag (split on : and take first part)
	name = strings.Split(name, ":")[0]

	// Replace any invalid chars (including @ and .)
	reg := regexp.MustCompile(`[^a-zA-Z0-9\-_]`)
	return reg.ReplaceAllString(name, "_")
}

// writeAttestation writes the attestation to a file
func writeAttestation(path string, attestation *Attestation) error {
	// Marshal with deterministic ordering
	data, err := json.MarshalIndent(attestation, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal attestation: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write attestation: %w", err)
	}

	return nil
}

// updateLastAttestationPointer updates the last_attestation.json pointer
func updateLastAttestationPointer(attestation *Attestation, path string) error {
	stateDir := filepath.Join(".acc", "state")
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return err
	}

	pointer := map[string]interface{}{
		"attestationPath": path,
		"timestamp":       attestation.Timestamp,
		"imageRef":        attestation.Subject.ImageRef,
		"imageDigest":     attestation.Subject.ImageDigest,
		"status":          attestation.Evidence.VerificationStatus,
	}

	data, err := json.MarshalIndent(pointer, "", "  ")
	if err != nil {
		return err
	}

	pointerFile := filepath.Join(stateDir, "last_attestation.json")
	return os.WriteFile(pointerFile, data, 0644)
}

// resolveDigest attempts to resolve the digest for an image reference
func resolveDigest(imageRef string) (string, error) {
	tools := []struct {
		name string
		args []string
	}{
		{"docker", []string{"inspect", "--format={{.Id}}", imageRef}},
		{"podman", []string{"inspect", "--format={{.Id}}", imageRef}},
		{"nerdctl", []string{"inspect", "--format={{.Id}}", imageRef}},
	}

	for _, tool := range tools {
		if _, err := exec.LookPath(tool.name); err == nil {
			cmd := exec.Command(tool.name, tool.args...)
			output, err := cmd.Output()
			if err == nil {
				digest := strings.TrimSpace(string(output))
				digest = strings.TrimPrefix(digest, "sha256:")
				if digest != "" {
					return digest, nil
				}
			}
		}
	}

	return "", fmt.Errorf("could not resolve digest")
}

// publishAttestationToRegistry publishes an attestation to a remote OCI registry
// v0.3.2: Remote attestation publishing
func publishAttestationToRegistry(imageRef string, attestation *Attestation, outputJSON bool) error {
	// v0.3.2: Remote attestation publishing implementation
	// TODO: Implement OCI artifact push using oras-go or go-containerregistry
	//
	// Design:
	// 1. Convert attestation to JSON
	// 2. Create OCI artifact with media type: application/vnd.acc.attestation.v1+json
	// 3. Tag/reference using image digest
	// 4. Push to registry using standard Docker auth (~/.docker/config.json)
	//
	// For now, return not implemented error to allow compilation and testing

	return fmt.Errorf("remote attestation publishing not yet implemented (v0.3.2 TODO)")
}

// FormatJSON formats attestation result as JSON
func (ar *AttestResult) FormatJSON() string {
	data, _ := json.MarshalIndent(ar, "", "  ")
	return string(data)
}
