package profile

import (
	"testing"
)

// TestResolveViolations_NoProfile tests that without a profile, all violations are blocking
func TestResolveViolations_NoProfile(t *testing.T) {
	violations := []Violation{
		{Rule: "no-root-user", Severity: "high", Message: "runs as root"},
		{Rule: "no-latest-tag", Severity: "medium", Message: "uses latest tag"},
	}

	result := ResolveViolations(nil, violations)

	if len(result.Violations) != 2 {
		t.Errorf("violations = %d, want 2", len(result.Violations))
	}
	if len(result.Warnings) != 0 {
		t.Errorf("warnings = %d, want 0", len(result.Warnings))
	}
	if result.Allow {
		t.Error("allow = true, want false")
	}
}

// TestResolveViolations_AllowList tests filtering by policies.allow
func TestResolveViolations_AllowList(t *testing.T) {
	profile := &Profile{
		SchemaVersion: 1,
		Name:          "test",
		Description:   "Test profile",
		Policies: PolicyConfig{
			Allow: []string{"no-root-user"},
		},
		Warnings: WarningConfig{Show: true},
	}

	violations := []Violation{
		{Rule: "no-root-user", Severity: "high", Message: "runs as root"},
		{Rule: "no-latest-tag", Severity: "medium", Message: "uses latest tag"},
		{Rule: "other-rule", Severity: "low", Message: "other issue"},
	}

	result := ResolveViolations(profile, violations)

	// Only no-root-user should be included (it's in allow list)
	if len(result.Violations) != 1 {
		t.Errorf("violations = %d, want 1", len(result.Violations))
	}
	if len(result.Violations) > 0 && result.Violations[0].Rule != "no-root-user" {
		t.Errorf("violation rule = %q, want %q", result.Violations[0].Rule, "no-root-user")
	}
	if result.Allow {
		t.Error("allow = true, want false (blocking violation present)")
	}
}

// TestResolveViolations_IgnoreByRule tests filtering by violation rule name
func TestResolveViolations_IgnoreByRule(t *testing.T) {
	profile := &Profile{
		SchemaVersion: 1,
		Name:          "test",
		Description:   "Test profile",
		Violations: ViolationConfig{
			Ignore: []string{"no-latest-tag"},
		},
		Warnings: WarningConfig{Show: true},
	}

	violations := []Violation{
		{Rule: "no-root-user", Severity: "high", Message: "runs as root"},
		{Rule: "no-latest-tag", Severity: "medium", Message: "uses latest tag"},
	}

	result := ResolveViolations(profile, violations)

	// no-latest-tag should be ignored, only no-root-user blocks
	if len(result.Violations) != 1 {
		t.Errorf("violations = %d, want 1", len(result.Violations))
	}
	if len(result.Violations) > 0 && result.Violations[0].Rule != "no-root-user" {
		t.Errorf("violation rule = %q, want %q", result.Violations[0].Rule, "no-root-user")
	}
	if len(result.Warnings) != 1 {
		t.Errorf("warnings = %d, want 1", len(result.Warnings))
	}
	if len(result.Warnings) > 0 && result.Warnings[0].Rule != "no-latest-tag" {
		t.Errorf("warning rule = %q, want %q", result.Warnings[0].Rule, "no-latest-tag")
	}
}

// TestResolveViolations_IgnoreBySeverity tests filtering by severity level
func TestResolveViolations_IgnoreBySeverity(t *testing.T) {
	profile := &Profile{
		SchemaVersion: 1,
		Name:          "test",
		Description:   "Test profile",
		Violations: ViolationConfig{
			Ignore: []string{"low", "informational"},
		},
		Warnings: WarningConfig{Show: true},
	}

	violations := []Violation{
		{Rule: "rule1", Severity: "critical", Message: "critical issue"},
		{Rule: "rule2", Severity: "high", Message: "high issue"},
		{Rule: "rule3", Severity: "medium", Message: "medium issue"},
		{Rule: "rule4", Severity: "low", Message: "low issue"},
		{Rule: "rule5", Severity: "informational", Message: "info"},
	}

	result := ResolveViolations(profile, violations)

	// Only critical, high, medium should block (low and informational ignored)
	if len(result.Violations) != 3 {
		t.Errorf("violations = %d, want 3", len(result.Violations))
	}
	if len(result.Warnings) != 2 {
		t.Errorf("warnings = %d, want 2", len(result.Warnings))
	}
	if result.Allow {
		t.Error("allow = true, want false (blocking violations present)")
	}
}

// TestResolveViolations_AllowListAndIgnore tests combination of allow and ignore
func TestResolveViolations_AllowListAndIgnore(t *testing.T) {
	profile := &Profile{
		SchemaVersion: 1,
		Name:          "test",
		Description:   "Test profile",
		Policies: PolicyConfig{
			Allow: []string{"no-root-user", "no-latest-tag"},
		},
		Violations: ViolationConfig{
			Ignore: []string{"medium"},
		},
		Warnings: WarningConfig{Show: true},
	}

	violations := []Violation{
		{Rule: "no-root-user", Severity: "high", Message: "runs as root"},
		{Rule: "no-latest-tag", Severity: "medium", Message: "uses latest tag"},
		{Rule: "other-rule", Severity: "critical", Message: "other issue"},
	}

	result := ResolveViolations(profile, violations)

	// other-rule filtered by allow list
	// no-latest-tag filtered by severity (medium ignored)
	// Only no-root-user should block
	if len(result.Violations) != 1 {
		t.Errorf("violations = %d, want 1", len(result.Violations))
	}
	if len(result.Violations) > 0 && result.Violations[0].Rule != "no-root-user" {
		t.Errorf("violation rule = %q, want %q", result.Violations[0].Rule, "no-root-user")
	}
	if len(result.Warnings) != 1 {
		t.Errorf("warnings = %d, want 1", len(result.Warnings))
	}
}

// TestResolveViolations_AllViolationsIgnored tests that allow=true when all violations ignored
func TestResolveViolations_AllViolationsIgnored(t *testing.T) {
	profile := &Profile{
		SchemaVersion: 1,
		Name:          "test",
		Description:   "Test profile",
		Violations: ViolationConfig{
			Ignore: []string{"low", "medium", "high"},
		},
		Warnings: WarningConfig{Show: true},
	}

	violations := []Violation{
		{Rule: "rule1", Severity: "low", Message: "low issue"},
		{Rule: "rule2", Severity: "medium", Message: "medium issue"},
	}

	result := ResolveViolations(profile, violations)

	if len(result.Violations) != 0 {
		t.Errorf("violations = %d, want 0", len(result.Violations))
	}
	if len(result.Warnings) != 2 {
		t.Errorf("warnings = %d, want 2", len(result.Warnings))
	}
	if !result.Allow {
		t.Error("allow = false, want true (all violations ignored)")
	}
}

// TestResolveViolations_NoViolations tests empty violations list
func TestResolveViolations_NoViolations(t *testing.T) {
	profile := &Profile{
		SchemaVersion: 1,
		Name:          "test",
		Description:   "Test profile",
	}

	result := ResolveViolations(profile, []Violation{})

	if len(result.Violations) != 0 {
		t.Errorf("violations = %d, want 0", len(result.Violations))
	}
	if !result.Allow {
		t.Error("allow = false, want true (no violations)")
	}
}

// TestResolveViolations_WarningsDisabled tests that warnings are not shown when disabled
func TestResolveViolations_WarningsDisabled(t *testing.T) {
	profile := &Profile{
		SchemaVersion: 1,
		Name:          "test",
		Description:   "Test profile",
		Violations: ViolationConfig{
			Ignore: []string{"low"},
		},
		Warnings: WarningConfig{Show: false},
	}

	violations := []Violation{
		{Rule: "rule1", Severity: "low", Message: "low issue"},
		{Rule: "rule2", Severity: "high", Message: "high issue"},
	}

	result := ResolveViolations(profile, violations)

	if len(result.Warnings) != 0 {
		t.Errorf("warnings = %d, want 0 (warnings disabled)", len(result.Warnings))
	}
	if len(result.Violations) != 1 {
		t.Errorf("violations = %d, want 1", len(result.Violations))
	}
}

// TestResolveViolations_CaseInsensitiveSeverity tests case-insensitive severity matching
func TestResolveViolations_CaseInsensitiveSeverity(t *testing.T) {
	profile := &Profile{
		SchemaVersion: 1,
		Name:          "test",
		Description:   "Test profile",
		Violations: ViolationConfig{
			Ignore: []string{"Low", "MEDIUM", "InFoRmAtIoNaL"},
		},
		Warnings: WarningConfig{Show: true},
	}

	violations := []Violation{
		{Rule: "rule1", Severity: "low", Message: "low issue"},
		{Rule: "rule2", Severity: "medium", Message: "medium issue"},
		{Rule: "rule3", Severity: "informational", Message: "info"},
		{Rule: "rule4", Severity: "high", Message: "high issue"},
	}

	result := ResolveViolations(profile, violations)

	// Only high should block
	if len(result.Violations) != 1 {
		t.Errorf("violations = %d, want 1", len(result.Violations))
	}
	if len(result.Warnings) != 3 {
		t.Errorf("warnings = %d, want 3", len(result.Warnings))
	}
}
