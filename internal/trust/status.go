package trust

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudcwfranck/acc/internal/ui"
)

// StatusResult represents the trust status output
type StatusResult struct {
	SchemaVersion string            `json:"schemaVersion"`
	ImageRef      string            `json:"imageRef"`
	Status        string            `json:"status"` // pass, fail, unknown
	ProfileUsed   string            `json:"profileUsed,omitempty"`
	Violations    []Violation       `json:"violations"`
	Warnings      []Violation       `json:"warnings"`
	SBOMPresent   bool              `json:"sbomPresent"`
	Attestations  []string          `json:"attestations"`
	Timestamp     string            `json:"timestamp"`
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
func Status(imageRef string, outputJSON bool) (*StatusResult, error) {
	// Load verification state
	state, err := loadVerifyState(imageRef)
	if err != nil {
		if outputJSON {
			// Return minimal result with unknown status
			result := &StatusResult{
				SchemaVersion: "v0.2",
				ImageRef:      imageRef,
				Status:        "unknown",
				Violations:    []Violation{},
				Warnings:      []Violation{},
				Attestations:  []string{},
			}
			return result, nil
		}
		return nil, fmt.Errorf("no verification state found for image: %s\n\nRemediation:\n  - Run 'acc verify %s' first", imageRef, imageRef)
	}

	// Build result from state
	result := &StatusResult{
		SchemaVersion: "v0.2",
		ImageRef:      state.ImageRef,
		Status:        state.Status,
		ProfileUsed:   state.ProfileUsed,
		Timestamp:     state.Timestamp,
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

	// Check SBOM
	if sbomPresent, ok := state.Result["sbomPresent"].(bool); ok {
		result.SBOMPresent = sbomPresent
	}

	// Find attestations
	result.Attestations = findAttestations()

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

	return &state, nil
}

// resolveImageDigest tries to resolve image digest (best effort)
func resolveImageDigest(imageRef string) (string, error) {
	// This is a simplified version - just try to extract from imageRef if it contains @sha256:
	if strings.Contains(imageRef, "@sha256:") {
		parts := strings.Split(imageRef, "@sha256:")
		if len(parts) == 2 {
			return parts[1], nil
		}
	}
	return "", fmt.Errorf("digest not in reference")
}

// findAttestations looks for attestation files
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
// 0 = verified, 1 = not verified, 2 = no state
func (sr *StatusResult) ExitCode() int {
	if sr.Status == "unknown" {
		return 2
	}
	if sr.Status == "pass" {
		return 0
	}
	return 1
}

// getString safely extracts string from map
func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}
