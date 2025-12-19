package verify

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cloudcwfranck/acc/internal/config"
)

// TestPolicyDenyEnforcement tests that deny rules cause verification failure
// This test would FAIL on v0.1.0 (deny rules were ignored)
// This test should PASS after the fix
func TestPolicyDenyEnforcement(t *testing.T) {
	// Create temporary directory for test
	tmpDir, err := os.MkdirTemp("", "acc-policy-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Change to temp directory
	originalDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change dir: %v", err)
	}
	defer os.Chdir(originalDir)

	// Configure project
	cfg := &config.Config{
		Project: config.ProjectConfig{
			Name: "test-project",
		},
		SBOM: config.SBOMConfig{
			Format: "spdx",
		},
		Policy: config.PolicyConfig{
			Mode: "enforce",
		},
	}

	// Create SBOM so verification gets to policy evaluation
	sbomDir := filepath.Join(".acc", "sbom")
	if err := os.MkdirAll(sbomDir, 0755); err != nil {
		t.Fatalf("failed to create sbom dir: %v", err)
	}
	// Use proper naming: {project}.{format}.json
	sbomFile := filepath.Join(sbomDir, "test-project.spdx.json")
	if err := os.WriteFile(sbomFile, []byte(`{"spdxVersion": "SPDX-2.3"}`), 0644); err != nil {
		t.Fatalf("failed to write SBOM file: %v", err)
	}

	// Create policy directory
	policyDir := filepath.Join(".acc", "policy")
	if err := os.MkdirAll(policyDir, 0755); err != nil {
		t.Fatalf("failed to create policy dir: %v", err)
	}

	// Create policy file with unconditional deny
	policyContent := `package acc.policy

# Unconditional deny for testing
deny["test failure - deny rule triggered"] { true }
`
	policyFile := filepath.Join(policyDir, "test.rego")
	if err := os.WriteFile(policyFile, []byte(policyContent), 0644); err != nil {
		t.Fatalf("failed to write policy file: %v", err)
	}

	// Run verification
	result, err := Verify(cfg, "test:latest", false, false)

	// CRITICAL: Verification MUST fail when deny rules are present
	if err == nil {
		t.Fatal("Expected verification to fail with deny rule, but it passed")
	}

	if result.Status != "fail" {
		t.Errorf("Expected status 'fail', got '%s'", result.Status)
	}

	if result.PolicyResult == nil {
		t.Fatal("Expected PolicyResult to be set")
	}

	if result.PolicyResult.Allow {
		t.Error("Expected PolicyResult.Allow to be false when deny rules exist")
	}

	if len(result.PolicyResult.Violations) == 0 {
		t.Error("Expected violations to be populated with deny message")
	}

	// Verify deny message is present
	found := false
	for _, v := range result.PolicyResult.Violations {
		if v.Rule == "policy-deny" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected policy-deny violation in results")
	}
}

// TestPolicyDenyJSONOutput tests that JSON output reflects deny correctly
// This test would FAIL on v0.1.0 (JSON showed allow:true even with denies)
// This test should PASS after the fix
func TestPolicyDenyJSONOutput(t *testing.T) {
	// Create temporary directory for test
	tmpDir, err := os.MkdirTemp("", "acc-json-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Change to temp directory
	originalDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change dir: %v", err)
	}
	defer os.Chdir(originalDir)

	// Configure project
	cfg := &config.Config{
		Project: config.ProjectConfig{
			Name: "json-test",
		},
		SBOM: config.SBOMConfig{
			Format: "spdx",
		},
		Policy: config.PolicyConfig{
			Mode: "enforce",
		},
	}

	// Create SBOM
	sbomDir := filepath.Join(".acc", "sbom")
	if err := os.MkdirAll(sbomDir, 0755); err != nil {
		t.Fatalf("failed to create sbom dir: %v", err)
	}
	// Use proper naming: {project}.{format}.json
	sbomFile := filepath.Join(sbomDir, "json-test.spdx.json")
	if err := os.WriteFile(sbomFile, []byte(`{"spdxVersion": "SPDX-2.3"}`), 0644); err != nil {
		t.Fatalf("failed to write SBOM file: %v", err)
	}

	// Create policy with deny
	policyDir := filepath.Join(".acc", "policy")
	if err := os.MkdirAll(policyDir, 0755); err != nil {
		t.Fatalf("failed to create policy dir: %v", err)
	}

	policyContent := `package acc.policy
deny["JSON test deny"] { true }
`
	policyFile := filepath.Join(policyDir, "test.rego")
	if err := os.WriteFile(policyFile, []byte(policyContent), 0644); err != nil {
		t.Fatalf("failed to write policy file: %v", err)
	}

	// Run verification with JSON output
	result, _ := Verify(cfg, "test:latest", false, true)

	// Verify JSON structure
	if result == nil {
		t.Fatal("Expected result to be returned even on failure")
	}

	if result.PolicyResult == nil {
		t.Fatal("Expected PolicyResult to be set")
	}

	if result.PolicyResult.Allow {
		t.Error("JSON output: policyResult.allow must be false when deny exists")
	}

	if len(result.PolicyResult.Violations) == 0 {
		t.Error("JSON output: policyResult.violations must be populated")
	}

	if result.Status != "fail" {
		t.Errorf("JSON output: top-level status must be 'fail', got '%s'", result.Status)
	}
}

// TestExtractDenyRules tests the deny rule parser
func TestExtractDenyRules(t *testing.T) {
	tests := []struct {
		name     string
		policy   string
		expected int
	}{
		{
			name: "deny with message",
			policy: `package acc
deny["test message"] { true }`,
			expected: 1,
		},
		{
			name: "multiple denies",
			policy: `package acc
deny["first"] { true }
deny["second"] { true }`,
			expected: 2,
		},
		{
			name: "deny without message",
			policy: `package acc
deny { true }`,
			expected: 1,
		},
		{
			name: "no denies",
			policy: `package acc
allow { true }`,
			expected: 0,
		},
		{
			name: "deny with equals",
			policy: `package acc
deny = "equal sign message" { true }`,
			expected: 1,
		},
		{
			name: "commented deny",
			policy: `package acc
# deny["should be ignored"] { true }`,
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			denies := extractDenyRules(tt.policy)
			if len(denies) != tt.expected {
				t.Errorf("Expected %d denies, got %d: %v", tt.expected, len(denies), denies)
			}
		})
	}
}

// TestPolicyWithNoDenyPasses tests that policies without deny rules pass
func TestPolicyWithNoDenyPasses(t *testing.T) {
	// Create temporary directory for test
	tmpDir, err := os.MkdirTemp("", "acc-allow-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Change to temp directory
	originalDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change dir: %v", err)
	}
	defer os.Chdir(originalDir)

	// Configure project
	cfg := &config.Config{
		Project: config.ProjectConfig{
			Name: "allow-test",
		},
		SBOM: config.SBOMConfig{
			Format: "spdx",
		},
		Policy: config.PolicyConfig{
			Mode: "enforce",
		},
	}

	// Create SBOM
	sbomDir := filepath.Join(".acc", "sbom")
	if err := os.MkdirAll(sbomDir, 0755); err != nil {
		t.Fatalf("failed to create sbom dir: %v", err)
	}
	// Use proper naming: {project}.{format}.json
	sbomFile := filepath.Join(sbomDir, "allow-test.spdx.json")
	if err := os.WriteFile(sbomFile, []byte(`{"spdxVersion": "SPDX-2.3"}`), 0644); err != nil {
		t.Fatalf("failed to write SBOM file: %v", err)
	}

	// Create policy with NO deny (only allow)
	policyDir := filepath.Join(".acc", "policy")
	if err := os.MkdirAll(policyDir, 0755); err != nil {
		t.Fatalf("failed to create policy dir: %v", err)
	}

	policyContent := `package acc.policy
allow { true }
`
	policyFile := filepath.Join(policyDir, "test.rego")
	if err := os.WriteFile(policyFile, []byte(policyContent), 0644); err != nil {
		t.Fatalf("failed to write policy file: %v", err)
	}

	// Run verification
	result, err := Verify(cfg, "test:latest", false, false)

	// Should pass when no deny rules
	if err != nil {
		t.Fatalf("Expected verification to pass without deny rules, got error: %v", err)
	}

	if result.Status != "pass" {
		t.Errorf("Expected status 'pass', got '%s'", result.Status)
	}

	if result.PolicyResult != nil && !result.PolicyResult.Allow {
		t.Error("Expected PolicyResult.Allow to be true when no deny rules")
	}
}
