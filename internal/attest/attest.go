package attest

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/cloudcwfranck/acc/internal/config"
	"github.com/cloudcwfranck/acc/internal/ui"
)

// Attestation represents a minimal v0 attestation
type Attestation struct {
	SchemaVersion string            `json:"schemaVersion"`
	Type          string            `json:"type"`
	ImageRef      string            `json:"imageRef"`
	Digest        string            `json:"digest,omitempty"`
	Timestamp     string            `json:"timestamp"`
	BuildMetadata BuildMetadata     `json:"buildMetadata,omitempty"`
	PolicyHash    string            `json:"policyHash,omitempty"`
	Metadata      map[string]string `json:"metadata"`
}

// BuildMetadata contains build-related metadata
type BuildMetadata struct {
	BuildTool string            `json:"buildTool,omitempty"`
	BuildTime string            `json:"buildTime,omitempty"`
	Builder   string            `json:"builder,omitempty"`
	Env       map[string]string `json:"env,omitempty"`
}

// AttestResult represents the result of attestation creation
type AttestResult struct {
	AttestationPath string      `json:"attestationPath"`
	Attestation     Attestation `json:"attestation"`
}

// Attest creates an attestation for an image
func Attest(cfg *config.Config, imageRef string, outputJSON bool) (*AttestResult, error) {
	if imageRef == "" {
		return nil, fmt.Errorf("image reference required")
	}

	if !outputJSON {
		ui.PrintInfo(fmt.Sprintf("Creating attestation for %s", imageRef))
	}

	// Resolve digest
	digest, err := resolveDigest(imageRef)
	if err != nil {
		if !outputJSON {
			ui.PrintWarning(fmt.Sprintf("Could not resolve digest: %v", err))
		}
	}

	// Load last verify results to get policy hash
	policyHash := loadPolicyHash()

	// Create attestation
	attestation := Attestation{
		SchemaVersion: "v0.1",
		Type:          "acc.build.v0",
		ImageRef:      imageRef,
		Digest:        digest,
		Timestamp:     time.Now().UTC().Format(time.RFC3339),
		BuildMetadata: getBuildMetadata(),
		PolicyHash:    policyHash,
		Metadata:      make(map[string]string),
	}

	// Add project metadata
	attestation.Metadata["project"] = cfg.Project.Name
	attestation.Metadata["policyMode"] = cfg.Policy.Mode
	attestation.Metadata["sbomFormat"] = cfg.SBOM.Format

	// Ensure attestations directory exists
	attestDir := filepath.Join(".acc", "attestations")
	if err := os.MkdirAll(attestDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create attestations directory: %w", err)
	}

	// Generate filename with timestamp and truncated digest
	filename := fmt.Sprintf("attest_%s", time.Now().UTC().Format("20060102_150405"))
	if digest != "" {
		filename = fmt.Sprintf("attest_%s_%s", time.Now().UTC().Format("20060102_150405"), digest[:12])
	}
	filename += ".json"

	attestPath := filepath.Join(attestDir, filename)

	// Write attestation with deterministic ordering (json.MarshalIndent ensures key order)
	data, err := json.MarshalIndent(attestation, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal attestation: %w", err)
	}

	if err := os.WriteFile(attestPath, data, 0644); err != nil {
		return nil, fmt.Errorf("failed to write attestation: %w", err)
	}

	if !outputJSON {
		ui.PrintSuccess(fmt.Sprintf("Attestation created: %s", attestPath))
		fmt.Printf("\nAttestation Details:\n")
		fmt.Printf("  Type:      %s\n", attestation.Type)
		fmt.Printf("  Image:     %s\n", attestation.ImageRef)
		if attestation.Digest != "" {
			fmt.Printf("  Digest:    sha256:%s\n", attestation.Digest)
		}
		fmt.Printf("  Timestamp: %s\n", attestation.Timestamp)
		if attestation.PolicyHash != "" {
			fmt.Printf("  Policy:    %s (hash)\n", attestation.PolicyHash[:16])
		}
	}

	result := &AttestResult{
		AttestationPath: attestPath,
		Attestation:     attestation,
	}

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

	return "", fmt.Errorf("could not resolve digest")
}

// getBuildMetadata gathers build metadata from environment
func getBuildMetadata() BuildMetadata {
	metadata := BuildMetadata{
		BuildTime: time.Now().UTC().Format(time.RFC3339),
		Env:       make(map[string]string),
	}

	// Detect build tool
	tools := []string{"docker", "podman", "buildah", "nerdctl"}
	for _, tool := range tools {
		if _, err := exec.LookPath(tool); err == nil {
			metadata.BuildTool = tool
			break
		}
	}

	// Get builder info (hostname)
	if hostname, err := os.Hostname(); err == nil {
		metadata.Builder = hostname
	}

	// Add safe environment variables (no secrets)
	safeEnvVars := []string{"USER", "CI", "GITHUB_ACTIONS", "GITLAB_CI"}
	for _, key := range safeEnvVars {
		if val := os.Getenv(key); val != "" {
			metadata.Env[key] = val
		}
	}

	return metadata
}

// loadPolicyHash loads the policy hash from last verify results
func loadPolicyHash() string {
	stateFile := filepath.Join(".acc", "state", "last_verify.json")
	data, err := os.ReadFile(stateFile)
	if err != nil {
		return ""
	}

	// Hash the verify results to create a policy decision hash
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// FormatJSON formats attestation result as JSON
func (ar *AttestResult) FormatJSON() string {
	data, _ := json.MarshalIndent(ar, "", "  ")
	return string(data)
}
