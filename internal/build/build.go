package build

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/cloudcwfranck/acc/internal/config"
	"github.com/cloudcwfranck/acc/internal/ui"
)

// BuildResult represents the output of a build operation
type BuildResult struct {
	ImageDigest string   `json:"imageDigest"`
	ImageTag    string   `json:"imageTag"`
	SBOMPath    string   `json:"sbomPath"`
	Attestations []string `json:"attestations"`
}

// Build builds an OCI image and generates SBOM (AGENTS.md Section 2 - acc build)
func Build(cfg *config.Config, tag string, outputJSON bool) (*BuildResult, error) {
	if !outputJSON {
		ui.PrintInfo(fmt.Sprintf("Building image for project '%s'", cfg.Project.Name))
	}

	// Determine which build tool to use (docker, podman, or buildah)
	buildTool, err := detectBuildTool()
	if err != nil {
		return nil, err
	}

	if !outputJSON {
		ui.PrintInfo(fmt.Sprintf("Using build tool: %s", buildTool))
	}

	// Build the image
	imageTag := tag
	if imageTag == "" {
		imageTag = fmt.Sprintf("%s/%s:%s", cfg.Registry.Default, cfg.Project.Name, cfg.Build.DefaultTag)
	}

	buildCmd := exec.Command(buildTool, "build", "-t", imageTag, cfg.Build.Context)
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr

	if !outputJSON {
		ui.PrintInfo(fmt.Sprintf("Running: %s build -t %s %s", buildTool, imageTag, cfg.Build.Context))
	}

	if err := buildCmd.Run(); err != nil {
		return nil, fmt.Errorf("build failed: %w", err)
	}

	if !outputJSON {
		ui.PrintSuccess("Image built successfully")
	}

	// Get image digest
	digest, err := getImageDigest(buildTool, imageTag)
	if err != nil {
		return nil, fmt.Errorf("failed to get image digest: %w", err)
	}

	if !outputJSON {
		ui.PrintInfo(fmt.Sprintf("Image digest: %s", digest))
	}

	// Generate SBOM
	sbomPath, err := generateSBOM(cfg, imageTag, digest)
	if err != nil {
		return nil, fmt.Errorf("failed to generate SBOM: %w", err)
	}

	if !outputJSON {
		ui.PrintSuccess(fmt.Sprintf("SBOM generated: %s", sbomPath))
	}

	result := &BuildResult{
		ImageDigest:  digest,
		ImageTag:     imageTag,
		SBOMPath:     sbomPath,
		Attestations: []string{},
	}

	return result, nil
}

// detectBuildTool detects which OCI build tool is available
func detectBuildTool() (string, error) {
	tools := []string{"docker", "podman", "buildah"}
	for _, tool := range tools {
		if _, err := exec.LookPath(tool); err == nil {
			return tool, nil
		}
	}
	return "", fmt.Errorf("no OCI build tool found (tried: docker, podman, buildah)\n\nRemediation:\n  - Install Docker: https://docs.docker.com/get-docker/\n  - Or install Podman: https://podman.io/getting-started/installation\n  - Or install Buildah: https://github.com/containers/buildah/blob/main/install.md")
}

// getImageDigest retrieves the digest of a built image
func getImageDigest(buildTool, imageTag string) (string, error) {
	cmd := exec.Command(buildTool, "inspect", "--format={{.Id}}", imageTag)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	digest := strings.TrimSpace(string(output))
	// Remove 'sha256:' prefix if present
	digest = strings.TrimPrefix(digest, "sha256:")
	return digest, nil
}

// generateSBOM generates an SBOM for the image
func generateSBOM(cfg *config.Config, imageTag, digest string) (string, error) {
	// Check for syft (SBOM generator)
	if _, err := exec.LookPath("syft"); err != nil {
		return "", fmt.Errorf("syft not found - required for SBOM generation\n\nRemediation:\n  - Install syft: https://github.com/anchore/syft#installation\n  - Or use: curl -sSfL https://raw.githubusercontent.com/anchore/syft/main/install.sh | sh -s -- -b /usr/local/bin")
	}

	// Create .acc/sbom directory if it doesn't exist
	sbomDir := filepath.Join(".acc", "sbom")
	if err := os.MkdirAll(sbomDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create SBOM directory: %w", err)
	}

	// Generate SBOM filename
	sbomFormat := cfg.SBOM.Format
	sbomFile := filepath.Join(sbomDir, fmt.Sprintf("%s.%s.json", cfg.Project.Name, sbomFormat))

	// Run syft to generate SBOM
	var formatArg string
	switch sbomFormat {
	case "spdx":
		formatArg = "spdx-json"
	case "cyclonedx":
		formatArg = "cyclonedx-json"
	default:
		formatArg = "spdx-json"
	}

	cmd := exec.Command("syft", imageTag, "-o", fmt.Sprintf("%s=%s", formatArg, sbomFile))
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("syft failed: %w\nOutput: %s", err, string(output))
	}

	return sbomFile, nil
}

// FormatJSON formats build result as JSON
func (br *BuildResult) FormatJSON() string {
	data, _ := json.MarshalIndent(br, "", "  ")
	return string(data)
}
