package verify

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/cloudcwfranck/acc/internal/config"
	"github.com/cloudcwfranck/acc/internal/ui"
	"github.com/cloudcwfranck/acc/internal/waivers"
)

// VerifyResult represents the output of a verification operation
type VerifyResult struct {
	Status       string            `json:"status"` // pass, warn, fail
	SBOMPresent  bool              `json:"sbomPresent"`
	PolicyResult *PolicyResult     `json:"policyResult"`
	Attestations []string          `json:"attestations"`
	Violations   []PolicyViolation `json:"violations"`
	Input        *RegoInput        `json:"input,omitempty"` // v0.1.3: Rego input document
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

	if result.Status == "fail" && len(result.Violations) > 0 && cfg.Policy.Mode == "enforce" {
		saveVerifyState(imageRef, result)
		return result, fmt.Errorf("verification failed: one or more waivers have expired")
	}

	// Step 3: Evaluate policy
	if !outputJSON {
		ui.PrintInfo("Evaluating policy...")
	}

	// Build Rego input for policy evaluation
	regoInput, err := buildRegoInput(cfg, imageRef, forPromotion)
	if err != nil {
		// v0.1.3: Image inspection failure is a CRITICAL violation
		violation := PolicyViolation{
			Rule:     "image-inspect-failed",
			Severity: "critical",
			Result:   "fail",
			Message:  fmt.Sprintf("Unable to inspect image config: %v", err),
		}
		result.Violations = append(result.Violations, violation)
		result.Status = "fail"

		if !outputJSON {
			ui.PrintError(violation.Message)
		}

		if cfg.Policy.Mode == "enforce" {
			saveVerifyState(imageRef, result)
			return result, fmt.Errorf("verification failed: %s", violation.Message)
		}
	} else {
		// Store input in result for policy explain
		result.Input = regoInput
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

	// Step 4: Check attestations (optional for now)
	attestPresent := checkAttestations(cfg)
	result.Attestations = []string{}
	if attestPresent {
		result.Attestations = append(result.Attestations, "present")
	}

	// Final status
	if result.Status != "fail" {
		result.Status = "pass"
		if !outputJSON {
			ui.PrintSuccess("Verification passed")
		}
	}

	// Save verification state
	saveVerifyState(imageRef, result)

	return result, nil
}

// VerifyState represents the persisted verification state for policy explain
type VerifyState struct {
	ImageRef  string        `json:"imageRef"`
	Status    string        `json:"status"`
	Timestamp string        `json:"timestamp"`
	Result    *VerifyResult `json:"result"`
}

// FormatJSON returns JSON representation
func (r *VerifyResult) FormatJSON() string {
	data, _ := json.MarshalIndent(r, "", "  ")
	return string(data)
}

// ExitCode returns the appropriate exit code for this result
func (r *VerifyResult) ExitCode() int {
	if r.Status == "pass" {
		return 0
	}
	return 1
}

// checkSBOMExists verifies SBOM file presence
func checkSBOMExists(cfg *config.Config) (bool, error) {
	sbomDir := filepath.Join(".acc", "sbom")

	// SBOM must match pattern: {project}.{format}.json
	sbomFile := filepath.Join(sbomDir, fmt.Sprintf("%s.%s.json", cfg.Project.Name, cfg.SBOM.Format))

	if _, err := os.Stat(sbomFile); os.IsNotExist(err) {
		return false, nil
	}

	return true, nil
}

// RegoInput represents the input document passed to Rego policy evaluation
type RegoInput struct {
	Config      ImageConfig      `json:"config"`
	SBOM        SBOMInfo         `json:"sbom"`
	Attestation AttestationInfo  `json:"attestation"`
	Promotion   bool             `json:"promotion"`
}

// ImageConfig contains image configuration fields
type ImageConfig struct {
	User   string            `json:"User"`
	Labels map[string]string `json:"Labels"`
}

// SBOMInfo contains SBOM presence information
type SBOMInfo struct {
	Present bool `json:"present"`
}

// AttestationInfo contains attestation presence information
type AttestationInfo struct {
	Present bool `json:"present"`
}

// inspectImageConfig inspects an image and returns its config
// v0.1.3: Returns error if inspection fails (no silent fallback)
func inspectImageConfig(imageRef string) (*ImageConfig, error) {
	// Try docker/podman/nerdctl to inspect image
	tools := []string{"docker", "podman", "nerdctl"}

	var lastErr error
	for _, tool := range tools {
		if _, err := exec.LookPath(tool); err == nil {
			// Use docker inspect to get full config as JSON
			cmd := exec.Command(tool, "inspect", imageRef)
			output, err := cmd.Output()
			if err != nil {
				lastErr = err
				continue
			}

			// Parse JSON output
			var inspectOutput []struct {
				Config struct {
					User   string            `json:"User"`
					Labels map[string]string `json:"Labels"`
				} `json:"Config"`
			}

			if err := json.Unmarshal(output, &inspectOutput); err != nil {
				lastErr = err
				continue
			}

			if len(inspectOutput) > 0 {
				labels := inspectOutput[0].Config.Labels
				if labels == nil {
					labels = make(map[string]string)
				}
				return &ImageConfig{
					User:   inspectOutput[0].Config.User,
					Labels: labels,
				}, nil
			}
		}
	}

	// v0.1.3: No tools found or all failed - return error
	if lastErr != nil {
		return nil, fmt.Errorf("failed to inspect image: %w (tried docker/podman/nerdctl)", lastErr)
	}
	return nil, fmt.Errorf("no container tools found (docker/podman/nerdctl required)")
}

// buildRegoInput constructs the input document for Rego evaluation
func buildRegoInput(cfg *config.Config, imageRef string, forPromotion bool) (*RegoInput, error) {
	// Get image configuration - v0.1.3: hard fail if this fails
	imageConfig, err := inspectImageConfig(imageRef)
	if err != nil {
		return nil, err
	}

	// Check for SBOM
	sbomPresent, _ := checkSBOMExists(cfg)

	// Check for attestations
	attestationPresent := checkAttestations(cfg)

	return &RegoInput{
		Config:      *imageConfig,
		SBOM:        SBOMInfo{Present: sbomPresent},
		Attestation: AttestationInfo{Present: attestationPresent},
		Promotion:   forPromotion,
	}, nil
}

// evaluateRego runs OPA evaluation and returns violations
// v0.1.3: OPA is REQUIRED - no text parsing fallback
func evaluateRego(policyDir string, input *RegoInput) ([]PolicyViolation, error) {
	// Check if opa is available
	opaPath, err := exec.LookPath("opa")
	if err != nil {
		// OPA is REQUIRED for security decisions
		// Check for escape hatch (dev/testing only)
		if os.Getenv("ACC_ALLOW_NO_OPA") == "1" {
			return []PolicyViolation{}, nil
		}
		return nil, fmt.Errorf("opa command not found: policy evaluation requires OPA\n\nInstall OPA: https://www.openpolicyagent.org/docs/latest/#running-opa\nOr set ACC_ALLOW_NO_OPA=1 (development only, NOT for production)")
	}

	// Marshal input to JSON
	inputJSON, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal input: %w", err)
	}

	// Write input to temp file
	inputFile, err := os.CreateTemp("", "acc-rego-input-*.json")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(inputFile.Name())
	defer inputFile.Close()

	if _, err := inputFile.Write(inputJSON); err != nil {
		return nil, fmt.Errorf("failed to write input: %w", err)
	}
	inputFile.Close()

	// Run OPA eval - evaluate data.acc.policy.result (not just deny)
	// This allows policies to build complete result objects
	cmd := exec.Command(opaPath, "eval",
		"--data", policyDir,
		"--input", inputFile.Name(),
		"--format", "json",
		"data.acc.policy.result")

	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("OPA evaluation failed: %s", string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("OPA evaluation failed: %w", err)
	}

	// Parse OPA output
	var opaResult struct {
		Result []struct {
			Expressions []struct {
				Value map[string]interface{} `json:"value"`
			} `json:"expressions"`
		} `json:"result"`
	}

	if err := json.Unmarshal(output, &opaResult); err != nil {
		return nil, fmt.Errorf("failed to parse OPA output: %w", err)
	}

	// Extract violations from policy result
	var violations []PolicyViolation
	if len(opaResult.Result) > 0 && len(opaResult.Result[0].Expressions) > 0 {
		value := opaResult.Result[0].Expressions[0].Value
		if value != nil {
			// Extract violations from result.violations
			if viols, ok := value["violations"].([]interface{}); ok {
				for _, item := range viols {
					if violation := parseViolationObject(item); violation != nil {
						violations = append(violations, *violation)
					}
				}
			}

			// Also check for deny set (backwards compatibility)
			if denies, ok := value["deny"].([]interface{}); ok {
				for _, item := range denies {
					if violation := parseViolationObject(item); violation != nil {
						violations = append(violations, *violation)
					}
				}
			}
		}
	}

	return violations, nil
}

// parseViolationObject parses a single violation object
func parseViolationObject(obj interface{}) *PolicyViolation {
	m, ok := obj.(map[string]interface{})
	if !ok {
		return nil
	}

	violation := &PolicyViolation{
		Rule:     getString(m, "rule", "policy-violation"),
		Severity: getString(m, "severity", "error"),
		Result:   getString(m, "result", "fail"),
		Message:  getString(m, "message", "Policy deny rule triggered"),
	}

	return violation
}

// getString safely extracts a string from a map
func getString(m map[string]interface{}, key, defaultValue string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return defaultValue
}

// evaluatePolicy evaluates the policy by running Rego with proper input
func evaluatePolicy(cfg *config.Config, imageRef string, forPromotion bool) (*PolicyResult, error) {
	result := &PolicyResult{
		Allow:      true,
		Violations: []PolicyViolation{},
		Warnings:   []PolicyViolation{},
	}

	// Load policy files from .acc/policy/
	policyDir := ".acc/policy"

	// Check if policy directory exists
	if _, err := os.Stat(policyDir); os.IsNotExist(err) {
		// No policy directory - allow by default
		return result, nil
	}

	// Read all .rego files in policy directory
	files, err := filepath.Glob(filepath.Join(policyDir, "*.rego"))
	if err != nil {
		return nil, fmt.Errorf("failed to read policy files: %w", err)
	}

	if len(files) == 0 {
		// No policy files - allow by default
		return result, nil
	}

	// Build Rego input document
	regoInput, err := buildRegoInput(cfg, imageRef, forPromotion)
	if err != nil {
		return nil, fmt.Errorf("failed to build Rego input: %w", err)
	}

	// Evaluate policy with OPA
	violations, err := evaluateRego(policyDir, regoInput)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate policy: %w", err)
	}

	if len(violations) > 0 {
		// If ANY deny violations exist, policy fails
		// Deny is authoritative
		result.Allow = false
		result.Violations = violations
	}

	return result, nil
}

// checkAttestations checks if attestations are present (stubbed)
func checkAttestations(cfg *config.Config) bool {
	// TODO: Implement actual attestation checking
	// For now, check if .acc/attestations directory has any files
	attestDir := filepath.Join(".acc", "attestations")
	if _, err := os.Stat(attestDir); os.IsNotExist(err) {
		return false
	}
	return true
}

// saveVerifyState persists verification results for policy explain
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
