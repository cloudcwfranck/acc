package verify

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cloudcwfranck/acc/internal/config"
)

// TestPolicyDenyEnforcement tests that deny rules cause verification failure
// This test would FAIL on v0.1.0 (deny rules were ignored)
// Updated for v0.1.2 to verify structured deny objects are preserved
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
	sbomFile := filepath.Join(sbomDir, "test-project.spdx.json")
	if err := os.WriteFile(sbomFile, []byte(`{"spdxVersion": "SPDX-2.3"}`), 0644); err != nil {
		t.Fatalf("failed to write SBOM file: %v", err)
	}

	// Create policy directory
	policyDir := filepath.Join(".acc", "policy")
	if err := os.MkdirAll(policyDir, 0755); err != nil {
		t.Fatalf("failed to create policy dir: %v", err)
	}

	// v0.1.2: Use structured deny object
	policyContent := `package acc.policy

deny contains {
	"rule": "test-deny-rule",
	"severity": "critical",
	"message": "Test failure - deny rule triggered"
}
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

	// v0.1.2: Verify deny object fields are preserved verbatim
	violation := result.PolicyResult.Violations[0]
	if violation.Rule != "test-deny-rule" {
		t.Errorf("Expected rule 'test-deny-rule', got '%s'", violation.Rule)
	}
	if violation.Severity != "critical" {
		t.Errorf("Expected severity 'critical', got '%s'", violation.Severity)
	}
	if violation.Message != "Test failure - deny rule triggered" {
		t.Errorf("Expected exact message, got '%s'", violation.Message)
	}
}

// TestPolicyDenyJSONOutput tests that JSON output reflects deny correctly
// This test would FAIL on v0.1.0 (JSON showed allow:true even with denies)
// Updated for v0.1.2 to verify structured deny propagation
func TestPolicyDenyJSONOutput(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "acc-json-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	originalDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change dir: %v", err)
	}
	defer os.Chdir(originalDir)

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

	sbomDir := filepath.Join(".acc", "sbom")
	if err := os.MkdirAll(sbomDir, 0755); err != nil {
		t.Fatalf("failed to create sbom dir: %v", err)
	}
	sbomFile := filepath.Join(sbomDir, "json-test.spdx.json")
	if err := os.WriteFile(sbomFile, []byte(`{"spdxVersion": "SPDX-2.3"}`), 0644); err != nil {
		t.Fatalf("failed to write SBOM file: %v", err)
	}

	policyDir := filepath.Join(".acc", "policy")
	if err := os.MkdirAll(policyDir, 0755); err != nil {
		t.Fatalf("failed to create policy dir: %v", err)
	}

	// v0.1.2: Use structured deny object
	policyContent := `package acc.policy

deny contains {
	"rule": "json-test-rule",
	"severity": "high",
	"message": "JSON test deny"
}
`
	policyFile := filepath.Join(policyDir, "test.rego")
	if err := os.WriteFile(policyFile, []byte(policyContent), 0644); err != nil {
		t.Fatalf("failed to write policy file: %v", err)
	}

	result, _ := Verify(cfg, "test:latest", false, true)

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

	// v0.1.2: Verify structured fields are preserved
	violation := result.PolicyResult.Violations[0]
	if violation.Rule != "json-test-rule" {
		t.Errorf("Expected rule 'json-test-rule', got '%s'", violation.Rule)
	}
	if violation.Severity != "high" {
		t.Errorf("Expected severity 'high', got '%s'", violation.Severity)
	}
}

// TestPolicyWithNoDenyPasses tests that policies without deny rules pass
func TestPolicyWithNoDenyPasses(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "acc-allow-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	originalDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change dir: %v", err)
	}
	defer os.Chdir(originalDir)

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

	sbomDir := filepath.Join(".acc", "sbom")
	if err := os.MkdirAll(sbomDir, 0755); err != nil {
		t.Fatalf("failed to create sbom dir: %v", err)
	}
	sbomFile := filepath.Join(sbomDir, "allow-test.spdx.json")
	if err := os.WriteFile(sbomFile, []byte(`{"spdxVersion": "SPDX-2.3"}`), 0644); err != nil {
		t.Fatalf("failed to write SBOM file: %v", err)
	}

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

	result, err := Verify(cfg, "test:latest", false, false)

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

// v0.1.2 MANDATORY TEST 1: Single deny rule with exact field preservation
// This test MUST FAIL on v0.1.1 (synthetic violations with rule="policy-deny")
// This test MUST PASS on v0.1.2 (verbatim deny object propagation)
func TestSingleDenyRuleVerbatim(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "acc-v012-test1-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	originalDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change dir: %v", err)
	}
	defer os.Chdir(originalDir)

	cfg := &config.Config{
		Project: config.ProjectConfig{Name: "smoke-test"},
		SBOM:    config.SBOMConfig{Format: "spdx"},
		Policy:  config.PolicyConfig{Mode: "enforce"},
	}

	sbomDir := filepath.Join(".acc", "sbom")
	if err := os.MkdirAll(sbomDir, 0755); err != nil {
		t.Fatalf("failed to create sbom dir: %v", err)
	}
	sbomFile := filepath.Join(sbomDir, "smoke-test.spdx.json")
	if err := os.WriteFile(sbomFile, []byte(`{"spdxVersion": "SPDX-2.3"}`), 0644); err != nil {
		t.Fatalf("failed to write SBOM: %v", err)
	}

	policyDir := filepath.Join(".acc", "policy")
	if err := os.MkdirAll(policyDir, 0755); err != nil {
		t.Fatalf("failed to create policy dir: %v", err)
	}

	// Exact policy from requirements
	policyContent := `package acc.policy

deny contains {
	"rule": "smoke-deny",
	"message": "SMOKE TEST DENY",
	"severity": "critical"
}
`
	policyFile := filepath.Join(policyDir, "smoke.rego")
	if err := os.WriteFile(policyFile, []byte(policyContent), 0644); err != nil {
		t.Fatalf("failed to write policy: %v", err)
	}

	result, err := Verify(cfg, "test:latest", false, false)

	// Must fail
	if err == nil {
		t.Fatal("Expected verification to fail with deny rule")
	}

	if result.PolicyResult == nil {
		t.Fatal("Expected PolicyResult to be set")
	}

	// Exactly 1 violation
	if len(result.PolicyResult.Violations) != 1 {
		t.Fatalf("Expected exactly 1 violation, got %d", len(result.PolicyResult.Violations))
	}

	v := result.PolicyResult.Violations[0]

	// Rule name must be exact
	if v.Rule != "smoke-deny" {
		t.Errorf("Expected rule 'smoke-deny', got '%s' (FAILS on v0.1.1: shows 'policy-deny')", v.Rule)
	}

	// Message must be exact
	if v.Message != "SMOKE TEST DENY" {
		t.Errorf("Expected message 'SMOKE TEST DENY', got '%s'", v.Message)
	}

	// Severity must be exact
	if v.Severity != "critical" {
		t.Errorf("Expected severity 'critical', got '%s'", v.Severity)
	}

	// allow must be false
	if result.PolicyResult.Allow {
		t.Error("Expected allow=false when deny exists")
	}
}

// v0.1.2 MANDATORY TEST 2: Multiple deny rules
// This test MUST FAIL on v0.1.1 (duplicate synthetic violations)
// This test MUST PASS on v0.1.2 (exactly 3 distinct violations)
func TestMultipleDenyRules(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "acc-v012-test2-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	originalDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change dir: %v", err)
	}
	defer os.Chdir(originalDir)

	cfg := &config.Config{
		Project: config.ProjectConfig{Name: "multi-test"},
		SBOM:    config.SBOMConfig{Format: "spdx"},
		Policy:  config.PolicyConfig{Mode: "enforce"},
	}

	sbomDir := filepath.Join(".acc", "sbom")
	os.MkdirAll(sbomDir, 0755)
	sbomFile := filepath.Join(sbomDir, "multi-test.spdx.json")
	os.WriteFile(sbomFile, []byte(`{"spdxVersion": "SPDX-2.3"}`), 0644)

	policyDir := filepath.Join(".acc", "policy")
	os.MkdirAll(policyDir, 0755)

	// Policy with 3 different deny rules
	policyContent := `package acc.policy

deny contains {
	"rule": "no-root",
	"severity": "high",
	"message": "Container runs as root"
}

deny contains {
	"rule": "missing-health-check",
	"severity": "medium",
	"message": "No health check defined"
}

deny contains {
	"rule": "insecure-port",
	"severity": "critical",
	"message": "Exposes insecure port 80"
}
`
	policyFile := filepath.Join(policyDir, "multi.rego")
	os.WriteFile(policyFile, []byte(policyContent), 0644)

	result, err := Verify(cfg, "test:latest", false, false)

	if err == nil {
		t.Fatal("Expected verification to fail with deny rules")
	}

	if result.PolicyResult == nil {
		t.Fatal("Expected PolicyResult to be set")
	}

	// Exactly 3 violations (NOT duplicates!)
	if len(result.PolicyResult.Violations) != 3 {
		t.Fatalf("Expected exactly 3 violations, got %d (FAILS on v0.1.1: may show duplicates)", len(result.PolicyResult.Violations))
	}

	// Verify all 3 distinct rules are present
	rules := make(map[string]bool)
	for _, v := range result.PolicyResult.Violations {
		rules[v.Rule] = true
	}

	expected := []string{"no-root", "missing-health-check", "insecure-port"}
	for _, rule := range expected {
		if !rules[rule] {
			t.Errorf("Expected rule '%s' not found in violations", rule)
		}
	}

	// Verify no synthetic "policy-deny" violations
	for _, v := range result.PolicyResult.Violations {
		if v.Rule == "policy-deny" {
			t.Error("Found synthetic 'policy-deny' violation (FAILS on v0.1.1)")
		}
	}
}

// v0.1.2 MANDATORY TEST 3: Allow-all policy (no denies)
func TestAllowAllPolicy(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "acc-v012-test3-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	originalDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(originalDir)

	cfg := &config.Config{
		Project: config.ProjectConfig{Name: "allow-all-test"},
		SBOM:    config.SBOMConfig{Format: "spdx"},
		Policy:  config.PolicyConfig{Mode: "enforce"},
	}

	sbomDir := filepath.Join(".acc", "sbom")
	os.MkdirAll(sbomDir, 0755)
	sbomFile := filepath.Join(sbomDir, "allow-all-test.spdx.json")
	os.WriteFile(sbomFile, []byte(`{"spdxVersion": "SPDX-2.3"}`), 0644)

	policyDir := filepath.Join(".acc", "policy")
	os.MkdirAll(policyDir, 0755)

	// Policy with only allow rules, no deny
	policyContent := `package acc.policy

default allow := true

deny := []
`
	policyFile := filepath.Join(policyDir, "allow.rego")
	os.WriteFile(policyFile, []byte(policyContent), 0644)

	result, err := Verify(cfg, "test:latest", false, false)

	// Must pass
	if err != nil {
		t.Fatalf("Expected verification to pass with allow-all policy, got error: %v", err)
	}

	if result.Status != "pass" {
		t.Errorf("Expected status 'pass', got '%s'", result.Status)
	}

	// No violations
	if result.PolicyResult != nil && len(result.PolicyResult.Violations) > 0 {
		t.Errorf("Expected no violations, got %d", len(result.PolicyResult.Violations))
	}

	// allow must be true
	if result.PolicyResult != nil && !result.PolicyResult.Allow {
		t.Error("Expected allow=true for allow-all policy")
	}
}

// TestParseDenyObjects tests the parser directly
func TestParseDenyObjects(t *testing.T) {
	tests := []struct {
		name     string
		policy   string
		expected int
		rules    []string
	}{
		{
			name: "single structured deny",
			policy: `deny contains {
  "rule": "test-rule",
  "severity": "high",
  "message": "test message"
}`,
			expected: 1,
			rules:    []string{"test-rule"},
		},
		{
			name: "multiple structured denies",
			policy: `deny contains {
  "rule": "first",
  "severity": "high",
  "message": "first message"
}

deny contains {
  "rule": "second",
  "severity": "low",
  "message": "second message"
}`,
			expected: 2,
			rules:    []string{"first", "second"},
		},
		{
			name:     "no denies",
			policy:   `allow { true }`,
			expected: 0,
			rules:    []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			violations := parseDenyObjects(tt.policy)
			if len(violations) != tt.expected {
				t.Errorf("Expected %d violations, got %d", tt.expected, len(violations))
			}

			for i, expectedRule := range tt.rules {
				if i >= len(violations) {
					t.Errorf("Missing violation for rule '%s'", expectedRule)
					continue
				}
				if violations[i].Rule != expectedRule {
					t.Errorf("Expected rule '%s', got '%s'", expectedRule, violations[i].Rule)
				}
			}
		})
	}
}
