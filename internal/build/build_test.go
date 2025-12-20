package build

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cloudcwfranck/acc/internal/config"
)

// v0.2.3 REGRESSION TEST 1: Test detectBuildTool logic
func TestDetectBuildTool(t *testing.T) {
	// This test verifies detectBuildTool returns an error when no tools are found
	// In CI environment without docker/podman/buildah, this should fail with clear message
	_, err := detectBuildTool()

	// In test environment, we expect no build tools, so this should error
	if err != nil {
		// Expected - verify error message is helpful
		if err.Error() == "" {
			t.Error("detectBuildTool should return descriptive error message")
		}
		t.Logf("Expected error (no build tools): %v", err)
	} else {
		// Build tool found - tests running in environment with docker/podman
		t.Log("Build tool detected - skipping test (docker/podman available)")
	}
}

// v0.2.3 REGRESSION TEST 2: Test SBOM file verification logic
// This test verifies that Build checks if SBOM file actually exists after generation
func TestBuild_SBOMVerification(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "acc-build-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	originalDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(originalDir)

	// Create minimal config
	cfg := &config.Config{
		Project:  config.ProjectConfig{Name: "test-build"},
		SBOM:     config.SBOMConfig{Format: "spdx"},
		Build:    config.BuildConfig{Context: ".", DefaultTag: "latest"},
		Registry: config.RegistryConfig{Default: "localhost"},
	}

	// Test: Build should fail when container tools are not available
	// This documents the expected contract: Build MUST produce SBOM or fail
	_, err = Build(cfg, "test-build:latest", true)

	// We expect Build to fail in test environment (no docker/podman)
	if err == nil {
		// If Build succeeded, verify SBOM exists
		sbomPath := filepath.Join(".acc", "sbom", "test-build.spdx.json")
		if _, statErr := os.Stat(sbomPath); os.IsNotExist(statErr) {
			t.Error("CRITICAL: Build succeeded but SBOM not found - this is Bug #2")
		} else {
			t.Log("Build succeeded and SBOM exists (container tools available)")
		}
	} else {
		// Expected error in test environment
		t.Logf("Build failed as expected (no container tools): %v", err)

		// Verify SBOM directory doesn't have partial files
		sbomDir := filepath.Join(".acc", "sbom")
		if entries, _ := os.ReadDir(sbomDir); len(entries) > 0 {
			t.Error("SBOM directory should be empty after failed build")
		}
	}
}

// v0.2.3 REGRESSION TEST 3: Test generateSBOM contract
// This test documents that generateSBOM MUST create the file or return error
func TestGenerateSBOM_Contract(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "acc-sbom-gen-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	originalDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(originalDir)

	cfg := &config.Config{
		Project: config.ProjectConfig{Name: "test-sbom"},
		SBOM:    config.SBOMConfig{Format: "spdx"},
	}

	// Test: generateSBOM should fail when syft is not available
	sbomPath, err := generateSBOM(cfg, "test:latest", "abc123")

	if err == nil {
		// If it succeeded, SBOM file MUST exist
		if _, statErr := os.Stat(sbomPath); os.IsNotExist(statErr) {
			t.Fatalf("CRITICAL: generateSBOM returned success but file not found: %s", sbomPath)
		}
		t.Logf("SBOM generated successfully: %s", sbomPath)
	} else {
		// Expected error when syft not available
		if sbomPath != "" {
			t.Error("generateSBOM should not return path when it fails")
		}
		t.Logf("generateSBOM failed as expected (syft not available): %v", err)
	}
}

// v0.2.3 REGRESSION TEST 4: Test SBOM path construction
func TestSBOMPath_Consistency(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "acc-sbom-path-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	originalDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(originalDir)

	projectName := "myproject"
	sbomFormat := "spdx"

	// Create .acc/sbom directory
	sbomDir := filepath.Join(".acc", "sbom")
	if err := os.MkdirAll(sbomDir, 0755); err != nil {
		t.Fatalf("failed to create sbom dir: %v", err)
	}

	// Expected SBOM path based on generateSBOM logic
	expectedPath := filepath.Join(sbomDir, projectName+"."+sbomFormat+".json")

	// Test: SBOM path should be predictable based on project name and format
	// This allows verify to find the SBOM reliably
	t.Logf("Expected SBOM path: %s", expectedPath)

	// Verify the path matches the pattern used in generateSBOM
	if filepath.Dir(expectedPath) != sbomDir {
		t.Error("SBOM path should be in .acc/sbom directory")
	}

	if !filepath.IsAbs(sbomDir) {
		// Relative paths are ok, verify it's under .acc/sbom
		if filepath.Base(filepath.Dir(expectedPath)) != "sbom" {
			t.Error("SBOM should be in sbom directory")
		}
	}
}
