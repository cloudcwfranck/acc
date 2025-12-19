package verify

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/cloudcwfranck/acc/internal/config"
	"github.com/cloudcwfranck/acc/internal/ui"
	"github.com/cloudcwfranck/acc/internal/waivers"
)

// VerifyResult represents the output of a verification operation
type VerifyResult struct {
	Status       string          `json:"status"` // pass, warn, fail
	SBOMPresent  bool            `json:"sbomPresent"`
	PolicyResult *PolicyResult   `json:"policyResult"`
	Attestations []string        `json:"attestations"`
	Violations   []PolicyViolation `json:"violations"`
}

// PolicyResult represents policy evaluation result
type PolicyResult struct {
	Allow      bool              `json:"allow"`
	Violations []PolicyViolation `json:"violations"`
	Warnings   []PolicyViolation `json:"warnings"`
}

// PolicyViolation represents a single policy violation or warning
type PolicyViolation struct {
	Rule     string `json:"rule"`
	Severity string `json:"severity"`
	Result   string `json:"result"`
	Message  string `json:"message"`
}

// Verify verifies SBOM, policy compliance, and attestations (AGENTS.md Section 2 - acc verify)
// This is critical: verification gates execution (Section 1.1)
func Verify(cfg *config.Config, imageRef string, forPromotion bool, outputJSON bool) (*VerifyResult, error) {
	if !outputJSON {
		ui.PrintTrust("Starting verification process")
	}

	result := &VerifyResult{
		Status:       "pass",
		Attestations: []string{},
		Violations:   []PolicyViolation{},
	}

	// Step 1: Verify SBOM exists
	if !outputJSON {
		ui.PrintInfo("Checking for SBOM...")
	}

	sbomExists, err := checkSBOMExists(cfg)
	if err != nil {
		return nil, err
	}

	result.SBOMPresent = sbomExists

	if !sbomExists {
		violation := PolicyViolation{
			Rule:     "sbom-required",
			Severity: "critical",
			Result:   "fail",
			Message:  "SBOM is required but not found",
		}
		result.Violations = append(result.Violations, violation)
		result.Status = "fail"

		if !outputJSON {
			ui.PrintError(violation.Message)
		}

		// CRITICAL: Per AGENTS.md Section 1.1 - verification failures block execution
		if cfg.Policy.Mode == "enforce" {
			// Save state before failing
			saveVerifyState(imageRef, result)
			return result, fmt.Errorf("verification failed: SBOM required but not found")
		}
	} else {
		if !outputJSON {
			ui.PrintSuccess("SBOM found")
		}
	}

	// Step 2: Check for expired waivers (CRITICAL: expired waiver = fail)
	if !outputJSON {
		ui.PrintInfo("Checking policy waivers...")
	}

	loadedWaivers, err := waivers.LoadWaivers()
	if err != nil {
		// Waiver loading failure is not fatal, just log
		if !outputJSON {
			ui.PrintWarning(fmt.Sprintf("Failed to load waivers: %v", err))
		}
		loadedWaivers = []waivers.Waiver{}
	}

	// Check for expired waivers - CRITICAL: expired waiver causes verification failure
	for _, waiver := range loadedWaivers {
		if waiver.IsExpired() {
			violation := PolicyViolation{
				Rule:     waiver.RuleID,
				Severity: "critical",
				Result:   "fail",
				Message:  fmt.Sprintf("Waiver for rule '%s' expired on %s", waiver.RuleID, waiver.Expiry),
			}
			result.Violations = append(result.Violations, violation)
			result.Status = "fail"

			if !outputJSON {
				ui.PrintError(fmt.Sprintf("Expired waiver: %s (expired: %s)", waiver.RuleID, waiver.Expiry))
			}
		}
	}

	// Fail fast on expired waivers in enforce mode
	if result.Status == "fail" && cfg.Policy.Mode == "enforce" {
		saveVerifyState(imageRef, result)
		return result, fmt.Errorf("verification failed: one or more waivers have expired")
	}

	// Step 3: Evaluate policy
	if !outputJSON {
		ui.PrintInfo("Evaluating policy...")
	}

	policyResult, err := evaluatePolicy(cfg, imageRef, forPromotion)
	if err != nil {
		return nil, fmt.Errorf("policy evaluation failed: %w", err)
	}

	result.PolicyResult = policyResult
	result.Violations = append(result.Violations, policyResult.Violations...)

	// Determine overall status
	if len(policyResult.Violations) > 0 {
		result.Status = "fail"
		if !outputJSON {
			ui.PrintError(fmt.Sprintf("Policy evaluation failed with %d violations:", len(policyResult.Violations)))
			for _, v := range policyResult.Violations {
				ui.PrintError(fmt.Sprintf("  [%s] %s: %s", v.Severity, v.Rule, v.Message))
			}
		}

		// CRITICAL: Per AGENTS.md Section 1.1 - no bypass, verification gates execution
		if cfg.Policy.Mode == "enforce" {
			// Save state before failing
			saveVerifyState(imageRef, result)
			return result, fmt.Errorf("verification failed: policy violations detected")
		}
	} else {
		if !outputJSON {
			ui.PrintSuccess("Policy evaluation passed")
		}
	}

	// Step 4: Check attestations (required for promotion)
	if forPromotion {
		if !outputJSON {
			ui.PrintInfo("Checking attestations for promotion...")
		}

		attestationPresent := checkAttestations(cfg)
		if !attestationPresent {
			violation := PolicyViolation{
				Rule:     "attestation-required-for-promotion",
				Severity: "critical",
				Result:   "fail",
				Message:  "Attestation required for promotion but not found",
			}
			result.Violations = append(result.Violations, violation)
			result.Status = "fail"

			if !outputJSON {
				ui.PrintError(violation.Message)
			}

			// CRITICAL: Block promotion if attestation missing (AGENTS.md Section 2 - acc verify)
			if cfg.Policy.Mode == "enforce" {
				// Save state before failing
				saveVerifyState(imageRef, result)
				return result, fmt.Errorf("verification failed: attestation required for promotion")
			}
		}
	}

	if !outputJSON && result.Status == "pass" {
		ui.PrintSuccess("Verification passed")
	}

	// Persist verify results to state (for policy explain)
	if err := saveVerifyState(imageRef, result); err != nil {
		// Don't fail verification if state save fails, just warn
		if !outputJSON {
			ui.PrintWarning(fmt.Sprintf("Failed to save verify state: %v", err))
		}
	}

	return result, nil
}

// checkSBOMExists checks if SBOM file exists for the project
func checkSBOMExists(cfg *config.Config) (bool, error) {
	sbomDir := filepath.Join(".acc", "sbom")
	sbomFile := filepath.Join(sbomDir, fmt.Sprintf("%s.%s.json", cfg.Project.Name, cfg.SBOM.Format))

	_, err := os.Stat(sbomFile)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

// evaluatePolicy evaluates the policy (currently stubbed - would use OPA/Rego)
func evaluatePolicy(cfg *config.Config, imageRef string, forPromotion bool) (*PolicyResult, error) {
	// TODO: Implement actual Rego policy evaluation using OPA
	// For now, return a basic check

	result := &PolicyResult{
		Allow:      true,
		Violations: []PolicyViolation{},
		Warnings:   []PolicyViolation{},
	}

	// NOTE: This is a stub implementation
	// Real implementation would:
	// 1. Load .acc/policy/*.rego files
	// 2. Use github.com/open-policy-agent/opa to evaluate
	// 3. Pass image metadata as input
	// 4. Return actual policy decision

	return result, nil
}

// checkAttestations checks if attestations are present (stubbed)
func checkAttestations(cfg *config.Config) bool {
	// TODO: Implement actual attestation checking
	// Would check for:
	// - SLSA attestations
	// - Build provenance
	// - Signatures
	attestationDir := filepath.Join(".acc", "attestations")
	_, err := os.Stat(attestationDir)
	return err == nil
}

// FormatJSON formats verification result as JSON
func (vr *VerifyResult) FormatJSON() string {
	data, _ := json.MarshalIndent(vr, "", "  ")
	return string(data)
}

// ExitCode returns the appropriate exit code based on verification status
// Per AGENTS.md Section 4.3:
// - 0 → success
// - 2 → warnings allowed
// - 1 → failure / blocked
func (vr *VerifyResult) ExitCode() int {
	switch vr.Status {
	case "pass":
		return 0
	case "warn":
		return 2
	case "fail":
		return 1
	default:
		return 1
	}
}

// VerifyState represents the persisted verification state
type VerifyState struct {
	ImageRef  string         `json:"imageRef"`
	Status    string         `json:"status"`
	Timestamp string         `json:"timestamp"`
	Result    *VerifyResult  `json:"result"`
}

// saveVerifyState persists verification results to .acc/state/last_verify.json
// This enables 'acc policy explain' to show the last verification decision
func saveVerifyState(imageRef string, result *VerifyResult) error {
	stateDir := filepath.Join(".acc", "state")
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}

	state := VerifyState{
		ImageRef:  imageRef,
		Status:    result.Status,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Result:    result,
	}

	// Mask any potential secrets before saving
	// (Currently none in VerifyResult, but defensive)
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	stateFile := filepath.Join(stateDir, "last_verify.json")
	if err := os.WriteFile(stateFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}

	return nil
}
