package profile

import (
	"strings"
)

// Violation represents a policy violation (mirrors verify.PolicyViolation)
// We use this interface to avoid circular dependencies
type Violation struct {
	Rule     string `json:"rule"`
	Severity string `json:"severity"`
	Result   string `json:"result"`
	Message  string `json:"message"`
}

// ResolutionResult represents the result of profile-based violation filtering
type ResolutionResult struct {
	Violations []Violation // Violations that cause failure
	Warnings   []Violation // Violations that are warnings only
	Allow      bool        // Final decision: true if no blocking violations remain
}

// ResolveViolations applies profile filtering to violations
// This is post-evaluation gating only - it does NOT affect policy execution
func ResolveViolations(profile *Profile, violations []Violation) *ResolutionResult {
	result := &ResolutionResult{
		Violations: []Violation{},
		Warnings:   []Violation{},
		Allow:      true,
	}

	// If no profile, all violations are blocking
	if profile == nil {
		result.Violations = violations
		result.Allow = len(violations) == 0
		return result
	}

	// Build allow and ignore sets for fast lookup
	allowSet := make(map[string]bool)
	for _, rule := range profile.Policies.Allow {
		allowSet[rule] = true
	}

	ignoreSet := make(map[string]bool)
	for _, item := range profile.Violations.Ignore {
		ignoreSet[strings.ToLower(item)] = true
	}

	// Filter violations
	for _, v := range violations {
		// Check if this rule is allowed (if allow list exists)
		if len(profile.Policies.Allow) > 0 && !allowSet[v.Rule] {
			// Rule not in allow list, skip (treat as if it doesn't exist)
			continue
		}

		// Check if this violation should be ignored
		if shouldIgnore(v, ignoreSet) {
			// Ignored violations become warnings if warnings are shown
			if profile.Warnings.Show {
				result.Warnings = append(result.Warnings, v)
			}
			continue
		}

		// This violation is blocking
		result.Violations = append(result.Violations, v)
	}

	// Final decision: allow if no blocking violations
	result.Allow = len(result.Violations) == 0

	return result
}

// shouldIgnore checks if a violation should be ignored based on the ignore set
// The ignore set can contain rule names or severity levels
func shouldIgnore(v Violation, ignoreSet map[string]bool) bool {
	// Check if rule name is in ignore list
	if ignoreSet[v.Rule] {
		return true
	}

	// Check if severity is in ignore list
	if ignoreSet[strings.ToLower(v.Severity)] {
		return true
	}

	return false
}
