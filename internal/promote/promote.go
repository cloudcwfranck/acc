package promote

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/cloudcwfranck/acc/internal/config"
	"github.com/cloudcwfranck/acc/internal/ui"
	"github.com/cloudcwfranck/acc/internal/verify"
)

// PromoteResult represents the result of a promotion
type PromoteResult struct {
	SourceRef string `json:"sourceRef"`
	TargetRef string `json:"targetRef"`
	Digest    string `json:"digest"`
	Env       string `json:"env"`
	Status    string `json:"status"`
}

// Promote promotes an image to an environment (AGENTS.md Section 2 - acc promote)
// CRITICAL: This MUST call verify internally and block on failure
func Promote(cfg *config.Config, imageRef, targetEnv string, outputJSON bool) (*PromoteResult, error) {
	if imageRef == "" {
		return nil, fmt.Errorf("image reference required")
	}
	if targetEnv == "" {
		return nil, fmt.Errorf("target environment required\n\nUsage: acc promote <image> --to <env>")
	}

	if !outputJSON {
		ui.PrintInfo(fmt.Sprintf("Promoting %s to environment: %s", imageRef, targetEnv))
	}

	// CRITICAL: Verification gates execution (AGENTS.md Section 1.1)
	if !outputJSON {
		ui.PrintTrust("Running verification before promotion...")
	}

	// Get environment-specific policy
	envPolicy := cfg.GetPolicyForEnv(targetEnv)

	// Create a temporary config with environment-specific policy for verification
	tempCfg := *cfg
	tempCfg.Policy = envPolicy

	// Verify with promotion flag set (requires attestations)
	verifyResult, err := verify.Verify(&tempCfg, imageRef, true, outputJSON, nil)
	if err != nil {
		// RED OUTPUT MEANS STOP (AGENTS.md Section 0)
		if !outputJSON {
			ui.PrintError(fmt.Sprintf("Verification failed for environment '%s' - promotion BLOCKED", targetEnv))
		}
		return nil, fmt.Errorf("verification failed: %w", err)
	}

	if verifyResult.Status == "fail" {
		if !outputJSON {
			ui.PrintError(fmt.Sprintf("Verification failed for environment '%s' - promotion BLOCKED", targetEnv))
		}
		return nil, fmt.Errorf("verification failed with status: %s", verifyResult.Status)
	}

	if !outputJSON {
		ui.PrintSuccess("Verification passed - proceeding with promotion")
	}

	// Get environment-specific registry
	envRegistry := cfg.GetRegistryForEnv(targetEnv)

	// Resolve digest
	digest, err := resolveDigest(imageRef)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve digest: %w\n\nRemediation:\n  - Ensure image exists locally: docker pull %s\n  - Or build the image first: acc build", err, imageRef)
	}

	// Determine target reference
	targetRef := buildTargetRef(imageRef, targetEnv, envRegistry.Default)

	if !outputJSON {
		ui.PrintInfo(fmt.Sprintf("Promoting: %s -> %s", imageRef, targetRef))
		ui.PrintInfo(fmt.Sprintf("Digest: sha256:%s", digest))
	}

	// Promote (re-tag without rebuild)
	if err := retagImage(imageRef, targetRef, digest); err != nil {
		return nil, err
	}

	if !outputJSON {
		ui.PrintSuccess(fmt.Sprintf("Promoted to %s", targetRef))
	}

	result := &PromoteResult{
		SourceRef: imageRef,
		TargetRef: targetRef,
		Digest:    digest,
		Env:       targetEnv,
		Status:    "success",
	}

	return result, nil
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

	return "", fmt.Errorf("could not resolve digest using available tools")
}

// buildTargetRef builds the target reference for promotion
func buildTargetRef(sourceRef, env, registry string) string {
	// Extract image name without tag
	parts := strings.Split(sourceRef, ":")
	imageName := parts[0]

	// Remove registry prefix if present
	if strings.Contains(imageName, "/") {
		lastSlash := strings.LastIndex(imageName, "/")
		imageName = imageName[lastSlash+1:]
	}

	// Build target reference with environment tag
	return fmt.Sprintf("%s/%s:%s", registry, imageName, env)
}

// retagImage re-tags an image without rebuild
func retagImage(sourceRef, targetRef, digest string) error {
	// Try different tools for re-tagging
	tools := []string{"docker", "podman", "nerdctl"}

	for _, tool := range tools {
		if _, err := exec.LookPath(tool); err == nil {
			// Tag the image
			cmd := exec.Command(tool, "tag", sourceRef, targetRef)
			if err := cmd.Run(); err != nil {
				continue
			}

			// Verify the tag points to the same digest
			verifyCmd := exec.Command(tool, "inspect", "--format={{.Id}}", targetRef)
			output, err := verifyCmd.Output()
			if err != nil {
				continue
			}

			newDigest := strings.TrimSpace(string(output))
			newDigest = strings.TrimPrefix(newDigest, "sha256:")

			if newDigest != digest {
				return fmt.Errorf("tag verification failed: digest mismatch")
			}

			return nil
		}
	}

	return fmt.Errorf("re-tag not possible: no supported tool found\n\nRemediation:\n  - Install Docker: https://docs.docker.com/get-docker/\n  - Or install Podman: https://podman.io/getting-started/installation\n  - Or install nerdctl: https://github.com/containerd/nerdctl\n\nNote: acc promote requires local image re-tagging capability")
}
