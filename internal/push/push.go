package push

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/cloudcwfranck/acc/internal/config"
	"github.com/cloudcwfranck/acc/internal/trust"
	"github.com/cloudcwfranck/acc/internal/ui"
)

// PushResult represents the result of a push operation
type PushResult struct {
	SchemaVersion      string `json:"schemaVersion"`
	Command            string `json:"command"`
	ImageRef           string `json:"imageRef"`
	ImageDigest        string `json:"imageDigest"`
	VerificationStatus string `json:"verificationStatus"`
	Pushed             bool   `json:"pushed"`
	Timestamp          string `json:"timestamp"`
	AttestationRef     string `json:"attestationRef,omitempty"`
}

// VerifyState represents the persisted verification state
type VerifyState struct {
	ImageRef  string                 `json:"imageRef"`
	Status    string                 `json:"status"`
	Timestamp string                 `json:"timestamp"`
	Result    map[string]interface{} `json:"result"`
}

// AttestationPointer represents the last attestation pointer
type AttestationPointer struct {
	OutputPath string `json:"outputPath"`
	Timestamp  string `json:"timestamp"`
}

// Push pushes a verified image to a registry (AGENTS.md - verify gates execution)
// CRITICAL: Must verify before push - no bypass flags allowed
func Push(cfg *config.Config, imageRef string, outputJSON bool) (*PushResult, error) {
	if imageRef == "" {
		return nil, fmt.Errorf("image reference required\n\nUsage: acc push <image>")
	}

	// CRITICAL: Load and validate verification state (AGENTS.md Section 1.1)
	if !outputJSON {
		ui.PrintTrust("Checking verification status...")
	}

	state, err := loadVerifyState()
	if err != nil {
		return nil, fmt.Errorf("verification state not found: %w\n\nRemediation:\n  Run 'acc verify %s' first to create verification state", err, imageRef)
	}

	// Verify status is not "fail"
	if state.Status == "fail" {
		return nil, fmt.Errorf("verification failed - push BLOCKED\n\nLast verification status: %s\nVerified at: %s\n\nRemediation:\n  Fix policy violations and re-run: acc verify %s", state.Status, state.Timestamp, imageRef)
	}

	if !outputJSON {
		ui.PrintSuccess(fmt.Sprintf("Verification confirmed (status: %s)", state.Status))
	}

	// Ensure imageRef matches last verified image (by digest)
	if err := validateImageMatch(imageRef, state); err != nil {
		return nil, err
	}

	// v0.3.1: Optional attestation enforcement
	if cfg.Policy.RequireAttestation {
		if !outputJSON {
			ui.PrintTrust("Checking attestation requirement...")
		}

		attestResult, err := trust.VerifyAttestations(imageRef, outputJSON)
		if err != nil || attestResult.VerificationStatus != "verified" {
			// Attestation enforcement blocks push (same exit code as verification gate)
			if !outputJSON {
				ui.PrintError("Attestation requirement not met - push BLOCKED")
				fmt.Fprintf(os.Stderr, "\nRemediation:\n")
				fmt.Fprintf(os.Stderr, "  1. Verify the workload: acc verify %s\n", imageRef)
				fmt.Fprintf(os.Stderr, "  2. Create attestation: acc attest %s\n", imageRef)
				fmt.Fprintf(os.Stderr, "  3. Re-push: acc push %s\n", imageRef)
			}
			return nil, fmt.Errorf("attestation requirement not met: %s", attestResult.VerificationStatus)
		}

		if !outputJSON {
			ui.PrintSuccess(fmt.Sprintf("Attestation verified (%d found)", attestResult.AttestationCount))
		}
	}

	// Resolve digest
	digest, err := resolveDigest(imageRef)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve digest: %w\n\nRemediation:\n  - Ensure image exists locally: docker pull %s\n  - Or build the image first: acc build", err, imageRef)
	}

	if !outputJSON {
		ui.PrintInfo(fmt.Sprintf("Pushing %s", imageRef))
		ui.PrintInfo(fmt.Sprintf("Digest: sha256:%s", digest[:12]))
	}

	// Push using available tool
	if err := pushImage(imageRef, outputJSON); err != nil {
		return nil, err
	}

	if !outputJSON {
		ui.PrintSuccess("Image pushed")
	}

	// Check for attestation reference
	attestationRef := ""
	if lastAtt := loadLastAttestation(); lastAtt != nil {
		attestationRef = lastAtt.OutputPath
		if !outputJSON {
			ui.PrintInfo(fmt.Sprintf("Attestation available: %s", attestationRef))
		}
	}

	result := &PushResult{
		SchemaVersion:      "v0.1",
		Command:            "push",
		ImageRef:           imageRef,
		ImageDigest:        digest,
		VerificationStatus: state.Status,
		Pushed:             true,
		Timestamp:          time.Now().UTC().Format(time.RFC3339),
		AttestationRef:     attestationRef,
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

// loadLastAttestation loads the last attestation pointer if it exists
func loadLastAttestation() *AttestationPointer {
	stateFile := filepath.Join(".acc", "state", "last_attestation.json")
	data, err := os.ReadFile(stateFile)
	if err != nil {
		return nil
	}

	var pointer AttestationPointer
	if err := json.Unmarshal(data, &pointer); err != nil {
		return nil
	}

	return &pointer
}

// validateImageMatch ensures imageRef matches the last verified image
func validateImageMatch(imageRef string, state *VerifyState) error {
	// If image refs match exactly, we're good
	if imageRef == state.ImageRef {
		return nil
	}

	// Otherwise, check if digests match
	currentDigest, err1 := resolveDigest(imageRef)
	stateDigest, err2 := resolveDigest(state.ImageRef)

	if err1 == nil && err2 == nil && currentDigest == stateDigest {
		return nil
	}

	return fmt.Errorf("image mismatch: attempting to push '%s' but last verified image was '%s'\n\nRemediation:\n  Run 'acc verify %s' first", imageRef, state.ImageRef, imageRef)
}

// resolveDigest attempts to resolve the digest for an image reference
func resolveDigest(imageRef string) (string, error) {
	tools := []struct {
		name string
		args []string
	}{
		{"nerdctl", []string{"inspect", "--format={{.Id}}", imageRef}},
		{"docker", []string{"inspect", "--format={{.Id}}", imageRef}},
		{"podman", []string{"inspect", "--format={{.Id}}", imageRef}},
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

	return "", fmt.Errorf("could not resolve digest using available tools")
}

// pushImage pushes the image using available tools
func pushImage(imageRef string, quiet bool) error {
	// Try tools in order: nerdctl, docker, oras
	tools := []string{"nerdctl", "docker", "oras"}

	for _, tool := range tools {
		if _, err := exec.LookPath(tool); err == nil {
			var cmd *exec.Cmd

			if tool == "oras" {
				// ORAS uses different syntax
				cmd = exec.Command(tool, "push", imageRef)
			} else {
				cmd = exec.Command(tool, "push", imageRef)
			}

			// Show output unless quiet
			if !quiet {
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
			}

			if err := cmd.Run(); err != nil {
				// Try next tool on error
				continue
			}

			return nil
		}
	}

	return fmt.Errorf("push not possible: no supported tool found\n\nRemediation:\n  - Install nerdctl: https://github.com/containerd/nerdctl\n  - Or install Docker: https://docs.docker.com/get-docker/\n  - Or install ORAS: https://oras.land/docs/installation\n\nNote: acc push requires a container registry client")
}

// FormatJSON formats push result as JSON
func (pr *PushResult) FormatJSON() string {
	data, _ := json.MarshalIndent(pr, "", "  ")
	return string(data)
}
