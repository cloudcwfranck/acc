package policy

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudcwfranck/acc/internal/ui"
	"github.com/cloudcwfranck/acc/internal/waivers"
)

// VerifyState represents the persisted verification state
type VerifyState struct {
	ImageRef  string                 `json:"imageRef"`
	Status    string                 `json:"status"`
	Timestamp string                 `json:"timestamp"`
	Result    map[string]interface{} `json:"result"`
}

// Explain loads and displays the last verification decision
func Explain(outputJSON bool) error {
	stateFile := filepath.Join(".acc", "state", "last_verify.json")

	data, err := os.ReadFile(stateFile)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("no verification history found\n\nHint: Run 'acc verify' first to generate verification results")
		}
		return fmt.Errorf("failed to read verification state: %w", err)
	}

	var state VerifyState
	if err := json.Unmarshal(data, &state); err != nil {
		return fmt.Errorf("failed to parse verification state: %w", err)
	}

	if outputJSON {
		// Ensure .result.input exists for contract stability
		if state.Result != nil {
			// Check if input field exists
			if _, hasInput := state.Result["input"]; !hasInput {
				// Add empty input object if missing
				state.Result["input"] = map[string]interface{}{}
			}
		}

		// Output raw state as JSON
		formatted, _ := json.MarshalIndent(state, "", "  ")
		fmt.Println(string(formatted))
		return nil
	}

	// Human-readable explanation
	printExplanation(&state)
	return nil
}

// printExplanation prints a developer-friendly explanation of the last verification
func printExplanation(state *VerifyState) {
	ui.PrintTrust("Last Verification Decision")
	fmt.Println()

	// Basic info
	fmt.Printf("Image:      %s\n", state.ImageRef)
	fmt.Printf("Time:       %s\n", state.Timestamp)

	// Status with color/symbol
	statusIcon := "❓"
	switch state.Status {
	case "pass":
		statusIcon = ui.SymbolSuccess
	case "fail":
		statusIcon = ui.SymbolFailure
	case "warn":
		statusIcon = ui.SymbolWarning
	}
	fmt.Printf("Decision:   %s %s\n", statusIcon, strings.ToUpper(state.Status))
	fmt.Println()

	// Extract result details
	result := state.Result
	if result == nil {
		ui.PrintWarning("No detailed results available")
		return
	}

	// SBOM presence
	if sbomPresent, ok := result["sbomPresent"].(bool); ok {
		if sbomPresent {
			ui.PrintSuccess("SBOM: Present")
		} else {
			ui.PrintError("SBOM: Missing")
		}
	}

	// Attestations
	if attestations, ok := result["attestations"].([]interface{}); ok {
		if len(attestations) > 0 {
			ui.PrintSuccess(fmt.Sprintf("Attestations: %d found", len(attestations)))
		} else {
			ui.PrintWarning("Attestations: None")
		}
	}

	fmt.Println()

	// Violations
	if violations, ok := result["violations"].([]interface{}); ok && len(violations) > 0 {
		fmt.Println("Policy Violations:")
		for i, v := range violations {
			if violation, ok := v.(map[string]interface{}); ok {
				rule := violation["rule"]
				severity := violation["severity"]
				message := violation["message"]
				fmt.Printf("  %d. [%s] %s\n", i+1, severity, rule)
				fmt.Printf("     %s\n", message)
			}
		}
		fmt.Println()

		// Remediation hints
		fmt.Println("Remediation:")
		fmt.Println("  - Review policy rules in .acc/policy/")
		fmt.Println("  - Fix violations in Dockerfile or build process")
		fmt.Println("  - Re-run 'acc build' and 'acc verify'")
		fmt.Println()
	}

	// Policy result summary
	if policyResult, ok := result["policyResult"].(map[string]interface{}); ok {
		if allow, ok := policyResult["allow"].(bool); ok {
			if allow {
				ui.PrintSuccess("Policy: Allowed")
			} else {
				ui.PrintError("Policy: Denied")
			}
		}

		// Warnings
		if warnings, ok := policyResult["warnings"].([]interface{}); ok && len(warnings) > 0 {
			fmt.Println()
			fmt.Println("Warnings:")
			for i, w := range warnings {
				if warning, ok := w.(map[string]interface{}); ok {
					rule := warning["rule"]
					message := warning["message"]
					fmt.Printf("  %d. %s: %s\n", i+1, rule, message)
				}
			}
		}
	}

	// Load and display waivers
	loadedWaivers, err := waivers.LoadWaivers()
	if err == nil && len(loadedWaivers) > 0 {
		fmt.Println()
		fmt.Println("Policy Waivers:")
		for i, w := range loadedWaivers {
			expiredStr := ""
			if w.IsExpired() {
				expiredStr = " ⚠️  EXPIRED"
			}
			fmt.Printf("  %d. %s%s\n", i+1, w.RuleID, expiredStr)
			fmt.Printf("     Justification: %s\n", w.Justification)
			if w.Expiry != "" {
				fmt.Printf("     Expires: %s\n", w.Expiry)
			}
			if w.ApprovedBy != "" {
				fmt.Printf("     Approved by: %s\n", w.ApprovedBy)
			}
		}
	}

	fmt.Println()
	fmt.Println("To see the full verification output:")
	fmt.Printf("  acc verify %s\n", state.ImageRef)
}
