package waivers

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestWaiverIsExpired tests the expiry checking logic
func TestWaiverIsExpired(t *testing.T) {
	tests := []struct {
		name     string
		waiver   Waiver
		expected bool
	}{
		{
			name: "not-expired",
			waiver: Waiver{
				RuleID:        "test-rule",
				Justification: "test",
				Expiry:        "2099-12-31T23:59:59Z", // Far future
			},
			expected: false,
		},
		{
			name: "expired",
			waiver: Waiver{
				RuleID:        "test-rule",
				Justification: "test",
				Expiry:        "2020-01-01T00:00:00Z", // Past
			},
			expected: true,
		},
		{
			name: "no-expiry",
			waiver: Waiver{
				RuleID:        "test-rule",
				Justification: "test",
				Expiry:        "",
			},
			expected: false,
		},
		{
			name: "invalid-expiry",
			waiver: Waiver{
				RuleID:        "test-rule",
				Justification: "test",
				Expiry:        "invalid-date",
			},
			expected: true, // Invalid dates are treated as expired for safety
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.waiver.IsExpired()
			if result != tt.expected {
				t.Errorf("IsExpired() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestWaiverExpiryEdgeCase tests waiver that expires "now"
func TestWaiverExpiryEdgeCase(t *testing.T) {
	// Create a waiver that expires in 1 second
	futureTime := time.Now().UTC().Add(1 * time.Second)
	waiver := Waiver{
		RuleID:        "test-rule",
		Justification: "test",
		Expiry:        futureTime.Format(time.RFC3339),
	}

	// Should not be expired yet
	if waiver.IsExpired() {
		t.Error("waiver should not be expired yet")
	}

	// Wait for it to expire
	time.Sleep(2 * time.Second)

	// Should now be expired
	if !waiver.IsExpired() {
		t.Error("waiver should be expired now")
	}
}

// TestLoadWaivers tests loading waivers from file
func TestLoadWaivers(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp dir: %v", err)
	}

	// Test with no waivers file
	waivers, err := LoadWaivers()
	if err != nil {
		t.Errorf("LoadWaivers() should not error when file missing, got: %v", err)
	}
	if len(waivers) != 0 {
		t.Errorf("expected 0 waivers, got %d", len(waivers))
	}

	// Create .acc directory
	if err := os.MkdirAll(".acc", 0755); err != nil {
		t.Fatalf("failed to create .acc dir: %v", err)
	}

	// Create waivers file
	waiversYAML := `waivers:
  - ruleId: no-root-user
    justification: Legacy container requires root
    expiry: "2099-12-31T23:59:59Z"
    approvedBy: security-team@example.com
  - ruleId: deprecated-base-image
    justification: Migration planned for Q2
    expiry: "2025-06-30T23:59:59Z"
`

	waiversPath := filepath.Join(".acc", "waivers.yaml")
	if err := os.WriteFile(waiversPath, []byte(waiversYAML), 0644); err != nil {
		t.Fatalf("failed to write waivers file: %v", err)
	}

	// Load waivers
	waivers, err = LoadWaivers()
	if err != nil {
		t.Fatalf("LoadWaivers() failed: %v", err)
	}

	if len(waivers) != 2 {
		t.Fatalf("expected 2 waivers, got %d", len(waivers))
	}

	// Check first waiver
	if waivers[0].RuleID != "no-root-user" {
		t.Errorf("expected ruleId 'no-root-user', got '%s'", waivers[0].RuleID)
	}
	if waivers[0].Justification != "Legacy container requires root" {
		t.Errorf("unexpected justification: %s", waivers[0].Justification)
	}
	if waivers[0].ApprovedBy != "security-team@example.com" {
		t.Errorf("unexpected approvedBy: %s", waivers[0].ApprovedBy)
	}

	// Check second waiver
	if waivers[1].RuleID != "deprecated-base-image" {
		t.Errorf("expected ruleId 'deprecated-base-image', got '%s'", waivers[1].RuleID)
	}
}

// TestLoadWaiversInvalidYAML tests handling of invalid YAML
func TestLoadWaiversInvalidYAML(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp dir: %v", err)
	}

	// Create .acc directory
	if err := os.MkdirAll(".acc", 0755); err != nil {
		t.Fatalf("failed to create .acc dir: %v", err)
	}

	// Create invalid waivers file
	invalidYAML := `waivers:
  - ruleId: test
    invalid yaml here [[[
`

	waiversPath := filepath.Join(".acc", "waivers.yaml")
	if err := os.WriteFile(waiversPath, []byte(invalidYAML), 0644); err != nil {
		t.Fatalf("failed to write waivers file: %v", err)
	}

	// Load waivers - should error
	_, err := LoadWaivers()
	if err == nil {
		t.Error("LoadWaivers() should error on invalid YAML")
	}
}

// TestGetWaiverForRule tests looking up waivers by rule ID
func TestGetWaiverForRule(t *testing.T) {
	waivers := []Waiver{
		{
			RuleID:        "rule-1",
			Justification: "test 1",
			Expiry:        "2099-12-31T23:59:59Z",
		},
		{
			RuleID:        "rule-2",
			Justification: "test 2",
			Expiry:        "2099-12-31T23:59:59Z",
		},
	}

	// Test finding existing waiver
	waiver := GetWaiverForRule(waivers, "rule-1")
	if waiver == nil {
		t.Fatal("expected to find waiver for rule-1")
	}
	if waiver.RuleID != "rule-1" {
		t.Errorf("expected ruleId 'rule-1', got '%s'", waiver.RuleID)
	}

	// Test finding second waiver
	waiver = GetWaiverForRule(waivers, "rule-2")
	if waiver == nil {
		t.Fatal("expected to find waiver for rule-2")
	}
	if waiver.RuleID != "rule-2" {
		t.Errorf("expected ruleId 'rule-2', got '%s'", waiver.RuleID)
	}

	// Test not finding non-existent waiver
	waiver = GetWaiverForRule(waivers, "rule-3")
	if waiver != nil {
		t.Error("should not find waiver for rule-3")
	}
}
