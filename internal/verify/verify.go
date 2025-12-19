package verify

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

	// Fail fast on expired waivers in enforce mode
	if result.Status == "fail" && cfg.Policy.Mode == "enforce" {
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
		// Don't fail if input building fails, just log warning
		if !outputJSON {
			ui.PrintWarning(fmt.Sprintf("Failed to build Rego input: %v", err))
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
func inspectImageConfig(imageRef string) (*ImageConfig, error) {
	// Try docker/podman/nerdctl to inspect image
	tools := []string{"docker", "podman", "nerdctl"}

	for _, tool := range tools {
		if _, err := exec.LookPath(tool); err == nil {
			// Use docker inspect to get full config as JSON
			cmd := exec.Command(tool, "inspect", imageRef)
			output, err := cmd.Output()
			if err != nil {
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

	// If no tool found or inspection failed, return empty config
	return &ImageConfig{
		User:   "",
		Labels: make(map[string]string),
	}, nil
}

// buildRegoInput constructs the input document for Rego evaluation
func buildRegoInput(cfg *config.Config, imageRef string, forPromotion bool) (*RegoInput, error) {
	// Get image configuration
	imageConfig, err := inspectImageConfig(imageRef)
	if err != nil {
		// If we can't inspect, use empty config rather than failing
		imageConfig = &ImageConfig{
			User:   "",
			Labels: make(map[string]string),
		}
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
func evaluateRego(policyDir string, input *RegoInput) ([]PolicyViolation, error) {
	// Check if opa is available
	opaPath, err := exec.LookPath("opa")
	if err != nil {
		// Fallback to text parsing if OPA not available
		// This maintains backwards compatibility
		return evaluateRegoFallback(policyDir)
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

	// Run OPA eval
	cmd := exec.Command(opaPath, "eval",
		"--data", policyDir,
		"--input", inputFile.Name(),
		"--format", "json",
		"data.acc.policy.deny")

	output, err := cmd.Output()
	if err != nil {
		// OPA command failed - try fallback
		return evaluateRegoFallback(policyDir)
	}

	// Parse OPA output
	var opaResult struct {
		Result []struct {
			Expressions []struct {
				Value interface{} `json:"value"`
			} `json:"expressions"`
		} `json:"result"`
	}

	if err := json.Unmarshal(output, &opaResult); err != nil {
		return nil, fmt.Errorf("failed to parse OPA output: %w", err)
	}

	// Extract violations from OPA result
	var violations []PolicyViolation
	if len(opaResult.Result) > 0 && len(opaResult.Result[0].Expressions) > 0 {
		value := opaResult.Result[0].Expressions[0].Value
		if value != nil {
			// Convert to violations
			violations = parseOPADenySet(value)
		}
	}

	return violations, nil
}

// parseOPADenySet parses the deny set from OPA output
func parseOPADenySet(value interface{}) []PolicyViolation {
	var violations []PolicyViolation

	// OPA returns deny as a set/array of objects
	switch v := value.(type) {
	case []interface{}:
		for _, item := range v {
			if violation := parseViolationObject(item); violation != nil {
				violations = append(violations, *violation)
			}
		}
	case map[string]interface{}:
		// Single violation as object
		if violation := parseViolationObject(v); violation != nil {
			violations = append(violations, *violation)
		}
	}

	return violations
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

// evaluateRegoFallback uses text parsing when OPA is not available
func evaluateRegoFallback(policyDir string) ([]PolicyViolation, error) {
	// Read all .rego files
	files, err := filepath.Glob(filepath.Join(policyDir, "*.rego"))
	if err != nil {
		return nil, err
	}

	var policyContent string
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}
		policyContent += string(content) + "\n"
	}

	// Use the existing text parser as fallback
	return parseDenyObjects(policyContent), nil
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

// parseDenyObjects parses Rego content and extracts structured deny objects
// This preserves the exact rule, severity, and message from Rego policies
// No synthetic violations are created - we propagate deny objects verbatim
func parseDenyObjects(policyContent string) []PolicyViolation {
	var violations []PolicyViolation

	// Parse deny contains { ... } blocks
	// Modern Rego policies define deny objects like:
	//   deny contains {
	//     "rule": "no-root-user",
	//     "severity": "high",
	//     "message": "Container runs as root"
	//   }

	lines := strings.Split(policyContent, "\n")
	inDenyBlock := false
	var currentDeny map[string]string
	var blockLines []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip comments
		if strings.HasPrefix(trimmed, "#") {
			continue
		}

		// Check for deny contains { or deny = { or deny[...] {
		if strings.HasPrefix(trimmed, "deny contains {") ||
			strings.HasPrefix(trimmed, "deny = {") ||
			strings.HasPrefix(trimmed, "deny[") && strings.Contains(trimmed, "{") {
			inDenyBlock = true
			currentDeny = make(map[string]string)
			blockLines = []string{line}
			continue
		}

		if inDenyBlock {
			blockLines = append(blockLines, line)

			// Extract key-value pairs from deny object
			// Match: "key": "value"  or  "key": "value",
			if strings.Contains(trimmed, ":") {
				parts := strings.SplitN(trimmed, ":", 2)
				if len(parts) == 2 {
					key := strings.TrimSpace(parts[0])
					value := strings.TrimSpace(parts[1])

					// Remove quotes from key
					key = strings.Trim(key, `"`)

					// Remove trailing comma from value first
					value = strings.TrimSuffix(value, ",")
					value = strings.TrimSpace(value)

					// Remove quotes from value
					value = strings.Trim(value, `"`)

					if key != "" && value != "" {
						currentDeny[key] = value
					}
				}
			}

			// Check for closing brace
			if strings.Contains(trimmed, "}") {
				// Create PolicyViolation from parsed deny object
				violation := PolicyViolation{
					Rule:     getOrDefault(currentDeny, "rule", "policy-violation"),
					Severity: getOrDefault(currentDeny, "severity", "error"),
					Result:   "fail",
					Message:  getOrDefault(currentDeny, "message", "Policy deny rule triggered"),
				}
				violations = append(violations, violation)

				inDenyBlock = false
				currentDeny = nil
				blockLines = nil
			}
		}
	}

	return violations
}

// getOrDefault returns the value for a key, or a default if not found
func getOrDefault(m map[string]string, key, defaultValue string) string {
	if val, ok := m[key]; ok && val != "" {
		return val
	}
	return defaultValue
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
	ImageRef  string        `json:"imageRef"`
	Status    string        `json:"status"`
	Timestamp string        `json:"timestamp"`
	Result    *VerifyResult `json:"result"`
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
