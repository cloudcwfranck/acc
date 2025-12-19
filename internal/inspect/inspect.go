package inspect

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/cloudcwfranck/acc/internal/config"
	"github.com/cloudcwfranck/acc/internal/ui"
	"github.com/cloudcwfranck/acc/internal/waivers"
)

// InspectResult represents the inspection output
type InspectResult struct {
	SchemaVersion string            `json:"schemaVersion"`
	ImageRef      string            `json:"imageRef"`
	Digest        string            `json:"digest,omitempty"`
	Status        string            `json:"status"` // pass, warn, fail, unknown
	Artifacts     ArtifactInfo      `json:"artifacts"`
	Policy        PolicyInfo        `json:"policy"`
	Metadata      map[string]string `json:"metadata"`
	Timestamp     string            `json:"timestamp"`
}

// ArtifactInfo contains artifact-related information
type ArtifactInfo struct {
	SBOMPath     string   `json:"sbomPath,omitempty"`
	SBOMFormat   string   `json:"sbomFormat,omitempty"`
	Attestations []string `json:"attestations"`
}

// PolicyInfo contains policy-related information
type PolicyInfo struct {
	Mode       string   `json:"mode"`
	PolicyPack string   `json:"policyPack"`
	Waivers    []Waiver `json:"waivers"`
}

// Waiver represents a policy waiver
type Waiver struct {
	RuleID        string `json:"ruleId"`
	Justification string `json:"justification"`
	Expiry        string `json:"expiry,omitempty"`
	Expired       bool   `json:"expired"`
}

// Inspect performs inspection of an image and returns trust summary
func Inspect(cfg *config.Config, imageRef string, outputJSON bool) (*InspectResult, error) {
	if imageRef == "" {
		return nil, fmt.Errorf("image reference required")
	}

	result := &InspectResult{
		SchemaVersion: "v0.1",
		ImageRef:      imageRef,
		Status:        "unknown",
		Artifacts: ArtifactInfo{
			Attestations: []string{},
		},
		Policy: PolicyInfo{
			Mode:       cfg.Policy.Mode,
			PolicyPack: ".acc/policy",
			Waivers:    []Waiver{},
		},
		Metadata:  make(map[string]string),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	// Try to resolve digest
	digest, err := resolveDigest(imageRef)
	if err != nil {
		if !outputJSON {
			ui.PrintWarning(fmt.Sprintf("Could not resolve digest: %v", err))
		}
	} else {
		result.Digest = digest
		result.Metadata["digestResolved"] = "true"
	}

	// Check for SBOM
	sbomPath, sbomFormat := findSBOM(cfg)
	if sbomPath != "" {
		result.Artifacts.SBOMPath = sbomPath
		result.Artifacts.SBOMFormat = sbomFormat
	}

	// Check for attestations
	attestations := findAttestations()
	result.Artifacts.Attestations = attestations

	// Load last verification status if available
	lastVerify := loadLastVerifyStatus()
	if lastVerify != nil {
		result.Status = lastVerify.Status
		result.Metadata["lastVerified"] = lastVerify.Timestamp
	}

	// Load waivers and check expiry status
	loadedWaivers, err := waivers.LoadWaivers()
	if err == nil && len(loadedWaivers) > 0 {
		inspectWaivers := make([]Waiver, 0, len(loadedWaivers))
		for _, w := range loadedWaivers {
			inspectWaivers = append(inspectWaivers, Waiver{
				RuleID:        w.RuleID,
				Justification: w.Justification,
				Expiry:        w.Expiry,
				Expired:       w.IsExpired(),
			})
		}
		result.Policy.Waivers = inspectWaivers
	}

	// Output results
	if outputJSON {
		return result, nil
	}

	// Human-readable output
	printHumanInspect(result)
	return result, nil
}

// resolveDigest attempts to resolve the digest for an image reference
func resolveDigest(imageRef string) (string, error) {
	// Try different tools to get the digest
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

	return "", fmt.Errorf("could not resolve digest using available tools\n\nRemediation:\n  - Pull the image first: docker pull %s\n  - Or ensure the image exists locally", imageRef)
}

// findSBOM looks for SBOM files in .acc/sbom/
func findSBOM(cfg *config.Config) (string, string) {
	sbomDir := filepath.Join(".acc", "sbom")
	if _, err := os.Stat(sbomDir); os.IsNotExist(err) {
		return "", ""
	}

	// Look for project-specific SBOM
	sbomFile := filepath.Join(sbomDir, fmt.Sprintf("%s.%s.json", cfg.Project.Name, cfg.SBOM.Format))
	if _, err := os.Stat(sbomFile); err == nil {
		return sbomFile, cfg.SBOM.Format
	}

	// Look for any SBOM files
	files, err := os.ReadDir(sbomDir)
	if err != nil {
		return "", ""
	}

	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".json") {
			path := filepath.Join(sbomDir, file.Name())
			// Detect format from filename
			format := "spdx"
			if strings.Contains(file.Name(), "cyclonedx") {
				format = "cyclonedx"
			}
			return path, format
		}
	}

	return "", ""
}

// findAttestations looks for attestation files in .acc/attestations/
func findAttestations() []string {
	attestDir := filepath.Join(".acc", "attestations")
	if _, err := os.Stat(attestDir); os.IsNotExist(err) {
		return []string{}
	}

	files, err := os.ReadDir(attestDir)
	if err != nil {
		return []string{}
	}

	var attestations []string
	for _, file := range files {
		if !file.IsDir() {
			attestations = append(attestations, filepath.Join(attestDir, file.Name()))
		}
	}

	return attestations
}

// LastVerifyStatus represents persisted verification status
type LastVerifyStatus struct {
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
	ImageRef  string `json:"imageRef"`
}

// loadLastVerifyStatus loads the last verification status from state
func loadLastVerifyStatus() *LastVerifyStatus {
	stateFile := filepath.Join(".acc", "state", "last_verify.json")
	data, err := os.ReadFile(stateFile)
	if err != nil {
		return nil
	}

	var status LastVerifyStatus
	if err := json.Unmarshal(data, &status); err != nil {
		return nil
	}

	return &status
}

// printHumanInspect prints human-readable inspection output
func printHumanInspect(result *InspectResult) {
	ui.PrintTrust("Trust Summary")
	fmt.Println()

	// Image information
	fmt.Printf("Image:          %s\n", result.ImageRef)
	if result.Digest != "" {
		fmt.Printf("Digest:         sha256:%s\n", result.Digest)
	} else {
		ui.PrintWarning("Digest:         (not resolved)")
	}
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
	if lastVerified, ok := result.Metadata["lastVerified"]; ok {
		fmt.Printf("Last Verified:  %s\n", lastVerified)
	}
	fmt.Println()

	// Artifacts
	fmt.Println("Artifacts:")
	if result.Artifacts.SBOMPath != "" {
		ui.PrintSuccess(fmt.Sprintf("  SBOM:         %s (%s)", result.Artifacts.SBOMPath, result.Artifacts.SBOMFormat))
	} else {
		ui.PrintWarning("  SBOM:         (not found)")
	}

	if len(result.Artifacts.Attestations) > 0 {
		ui.PrintSuccess(fmt.Sprintf("  Attestations: %d found", len(result.Artifacts.Attestations)))
		for _, att := range result.Artifacts.Attestations {
			fmt.Printf("    - %s\n", att)
		}
	} else {
		ui.PrintWarning("  Attestations: (none)")
	}
	fmt.Println()

	// Policy
	fmt.Println("Policy:")
	fmt.Printf("  Mode:         %s\n", result.Policy.Mode)
	fmt.Printf("  Pack:         %s\n", result.Policy.PolicyPack)

	if len(result.Policy.Waivers) > 0 {
		fmt.Println("  Waivers:")
		for _, w := range result.Policy.Waivers {
			expiredStr := ""
			if w.Expired {
				expiredStr = " (EXPIRED)"
			}
			fmt.Printf("    - %s: %s%s\n", w.RuleID, w.Justification, expiredStr)
			if w.Expiry != "" {
				fmt.Printf("      Expires: %s\n", w.Expiry)
			}
		}
	}
}

// FormatJSON formats inspection result as JSON
func (ir *InspectResult) FormatJSON() string {
	data, _ := json.MarshalIndent(ir, "", "  ")
	return string(data)
}
