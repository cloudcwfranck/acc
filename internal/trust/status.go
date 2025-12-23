package trust

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/cloudcwfranck/acc/internal/ui"
)

// StatusResult represents the trust status output
type StatusResult struct {
	SchemaVersion string      `json:"schemaVersion"`
	ImageRef      string      `json:"imageRef"`
	Status        string      `json:"status"` // pass, fail, unknown
	ProfileUsed   string      `json:"profileUsed,omitempty"`
	Violations    []Violation `json:"violations"`
	Warnings      []Violation `json:"warnings"`
	SBOMPresent   bool        `json:"sbomPresent"`
	Attestations  []string    `json:"attestations"`
	Timestamp     string      `json:"timestamp"`
}

// Violation represents a policy violation
type Violation struct {
	Rule     string `json:"rule"`
	Severity string `json:"severity"`
	Result   string `json:"result"`
	Message  string `json:"message"`
}

// VerifyState represents persisted verification state
type VerifyState struct {
	ImageRef    string                 `json:"imageRef"`
	Status      string                 `json:"status"`
	Timestamp   string                 `json:"timestamp"`
	ProfileUsed string                 `json:"profileUsed,omitempty"`
	Result      map[string]interface{} `json:"result"`
}

// Status loads and displays the trust status for an image
// v0.3.2: optionally fetch attestations from remote registry when remote=true
func Status(imageRef string, remote, outputJSON bool) (*StatusResult, error) {
	// Load verification state
	state, err := loadVerifyState(imageRef)
	if err != nil {
		// v0.2.7: Always return result with status "unknown" (exit code 2)
		// instead of error (exit code 1) when state not found
		result := &StatusResult{
			SchemaVersion: "v0.2",
			ImageRef:      imageRef,
			Status:        "unknown",
			SBOMPresent:   false, // v0.2.7: Ensure always set
			Violations:    []Violation{},
			Warnings:      []Violation{},
			Attestations:  []string{},
			Timestamp:     "", // v0.2.7: Empty string for unknown state
		}

		if !outputJSON {
			fmt.Fprintf(os.Stderr, "Warning: No verification state found for image: %s\n", imageRef)
			fmt.Fprintf(os.Stderr, "Remediation: Run 'acc verify %s' first\n\n", imageRef)
		}

		return result, nil
	}

	// Resolve digest for per-image attestation lookup
	digest, _ := resolveImageDigest(imageRef)

	// Build result from state (v0.2.7: ensure all fields initialized)
	result := &StatusResult{
		SchemaVersion: "v0.2",
		ImageRef:      state.ImageRef,
		Status:        state.Status,
		ProfileUsed:   state.ProfileUsed,
		Timestamp:     state.Timestamp,
		SBOMPresent:   false, // Default, will be set below
		Violations:    []Violation{},
		Warnings:      []Violation{},
		Attestations:  []string{},
	}

	// Extract violations and warnings from result
	if stateResult, ok := state.Result["policyResult"].(map[string]interface{}); ok {
		if violations, ok := stateResult["violations"].([]interface{}); ok {
			for _, v := range violations {
				if vm, ok := v.(map[string]interface{}); ok {
					result.Violations = append(result.Violations, Violation{
						Rule:     getString(vm, "rule"),
						Severity: getString(vm, "severity"),
						Result:   getString(vm, "result"),
						Message:  getString(vm, "message"),
					})
				}
			}
		}
		if warnings, ok := stateResult["warnings"].([]interface{}); ok {
			for _, w := range warnings {
				if wm, ok := w.(map[string]interface{}); ok {
					result.Warnings = append(result.Warnings, Violation{
						Rule:     getString(wm, "rule"),
						Severity: getString(wm, "severity"),
						Result:   getString(wm, "result"),
						Message:  getString(wm, "message"),
					})
				}
			}
		}
	}

	// Check SBOM (v0.2.7: ensure always set as boolean)
	if sbomPresent, ok := state.Result["sbomPresent"].(bool); ok {
		result.SBOMPresent = sbomPresent
	} else {
		result.SBOMPresent = false
	}

	// v0.3.2: Optionally fetch remote attestations before finding local ones
	if remote && digest != "" {
		if err := fetchRemoteAttestations(imageRef, digest, outputJSON); err != nil {
			// Remote fetch failed - log warning but don't fail
			// This preserves local-only workflow when network unavailable
			if !outputJSON {
				fmt.Fprintf(os.Stderr, "Warning: Failed to fetch remote attestations: %v\n", err)
			}
		}
	}

	// v0.2.7: Find attestations for this specific image (per-image isolation)
	// v0.3.2: This now includes both local and remote-cached attestations
	result.Attestations = findAttestationsForImage(digest)

	// Output results
	if outputJSON {
		return result, nil
	}

	// Human-readable output
	printHumanStatus(result)
	return result, nil
}

// loadVerifyState loads verification state for an image
// First tries digest-scoped state, falls back to global state
func loadVerifyState(imageRef string) (*VerifyState, error) {
	// Try to resolve digest for digest-scoped lookup
	digest, _ := resolveImageDigest(imageRef)
	if digest != "" {
		digestFile := filepath.Join(".acc", "state", "verify", digest+".json")
		if data, err := os.ReadFile(digestFile); err == nil {
			var state VerifyState
			if err := json.Unmarshal(data, &state); err == nil {
				return &state, nil
			}
		}
	}

	// Fall back to global last_verify.json
	stateFile := filepath.Join(".acc", "state", "last_verify.json")
	data, err := os.ReadFile(stateFile)
	if err != nil {
		return nil, err
	}

	var state VerifyState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}

	// CRITICAL: Verify the state is for the requested image
	// Without this check, different images can share state (cross-image leakage)
	if state.ImageRef != imageRef {
		return nil, fmt.Errorf("state mismatch: found state for %s, requested %s", state.ImageRef, imageRef)
	}

	return &state, nil
}

// resolveImageDigest tries to resolve image digest using container tools
// v0.2.1: Fixed to properly query Docker/Podman/nerdctl for digest
func resolveImageDigest(imageRef string) (string, error) {
	// First, try to extract from imageRef if it contains @sha256:
	if strings.Contains(imageRef, "@sha256:") {
		parts := strings.Split(imageRef, "@sha256:")
		if len(parts) == 2 {
			return parts[1], nil
		}
	}

	// Try different container tools to get the digest
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

	return "", fmt.Errorf("could not resolve digest for %s", imageRef)
}

// findAttestations looks for attestation files (all images)
func findAttestations() []string {
	attestDir := filepath.Join(".acc", "attestations")
	if _, err := os.Stat(attestDir); os.IsNotExist(err) {
		return []string{}
	}

	var attestations []string
	filepath.Walk(attestDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && filepath.Ext(path) == ".json" {
			attestations = append(attestations, path)
		}
		return nil
	})

	return attestations
}

// findAttestationsForImage looks for attestation files for a specific image digest
// v0.2.7: Per-image attestation isolation - only returns attestations for this digest
func findAttestationsForImage(digest string) []string {
	if digest == "" {
		// If digest not available, return all attestations as fallback
		return findAttestations()
	}

	// Use first 12 chars of digest to match directory structure
	digestPrefix := digest
	if len(digest) > 12 {
		digestPrefix = digest[:12]
	}

	attestDir := filepath.Join(".acc", "attestations", digestPrefix)
	if _, err := os.Stat(attestDir); os.IsNotExist(err) {
		return []string{}
	}

	var attestations []string
	filepath.Walk(attestDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && filepath.Ext(path) == ".json" {
			attestations = append(attestations, path)
		}
		return nil
	})

	return attestations
}

// printHumanStatus prints human-readable trust status
func printHumanStatus(result *StatusResult) {
	ui.PrintTrust("Trust Status")
	fmt.Println()

	// Image information
	fmt.Printf("Image:          %s\n", result.ImageRef)
	fmt.Printf("Last Verified:  %s\n", result.Timestamp)
	fmt.Println()

	// Verification status
	statusIcon := "❓"
	switch result.Status {
	case "pass":
		statusIcon = ui.SymbolSuccess
	case "fail":
		statusIcon = ui.SymbolFailure
	case "warn":
		statusIcon = ui.SymbolWarning
	}
	fmt.Printf("Status:         %s %s\n", statusIcon, strings.ToUpper(result.Status))

	// v0.2.0: Show profile if used
	if result.ProfileUsed != "" {
		fmt.Printf("Profile:        %s\n", result.ProfileUsed)
	}
	fmt.Println()

	// Artifacts
	fmt.Println("Artifacts:")
	if result.SBOMPresent {
		ui.PrintSuccess("  SBOM:         present")
	} else {
		ui.PrintWarning("  SBOM:         not found")
	}

	if len(result.Attestations) > 0 {
		ui.PrintSuccess(fmt.Sprintf("  Attestations: %d found", len(result.Attestations)))
	} else {
		ui.PrintWarning("  Attestations: none")
	}
	fmt.Println()

	// v0.2.0: Show violations and warnings separately
	if len(result.Violations) > 0 {
		fmt.Println("Policy Violations:")
		for _, v := range result.Violations {
			ui.PrintError(fmt.Sprintf("  [%s] %s: %s", v.Severity, v.Rule, v.Message))
		}
		fmt.Println()
	}

	if len(result.Warnings) > 0 {
		fmt.Println(fmt.Sprintf("Warnings (%d ignored):", len(result.Warnings)))
		for _, w := range result.Warnings {
			ui.PrintWarning(fmt.Sprintf("  [%s] %s: %s", w.Severity, w.Rule, w.Message))
		}
		fmt.Println()
	}

	if len(result.Violations) == 0 && len(result.Warnings) == 0 && result.Status == "pass" {
		ui.PrintSuccess("No policy violations")
	}
}

// FormatJSON formats status result as JSON
func (sr *StatusResult) FormatJSON() string {
	data, _ := json.MarshalIndent(sr, "", "  ")
	return string(data)
}

// ExitCode returns appropriate exit code
// PRESERVED BEHAVIOR: 0=pass, 1=fail/warn, 2=unknown
func (sr *StatusResult) ExitCode() int {
	if sr.Status == "unknown" {
		return 2
	}
	if sr.Status == "pass" {
		return 0
	}
	// fail or warn
	return 1
}

// getString safely extracts string from map
func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

// fetchRemoteAttestations fetches attestations from a remote OCI registry and caches them locally
// v0.3.2: Remote attestation fetching
func fetchRemoteAttestations(imageRef, digest string, outputJSON bool) error {
	// v0.3.2: Remote attestation fetching implementation
	// TODO: Implement OCI artifact pull using oras-go or go-containerregistry
	//
	// Design:
	// 1. Resolve registry and repository from imageRef
	// 2. Query for attestation artifacts with matching digest
	//    - Use OCI referrers API or tag naming convention
	//    - Media type: application/vnd.acc.attestation.v1+json
	// 3. Pull attestation artifacts using standard Docker auth (~/.docker/config.json)
	// 4. Cache to: .acc/attestations/<digest-prefix>/remote/<source>/<timestamp-or-hash>.json
	// 5. Merge with existing local attestations (findAttestationsForImage will find both)
	//
	// For now, return not implemented error to allow compilation and testing

	return fmt.Errorf("remote attestation fetching not yet implemented (v0.3.2 TODO)")
}
