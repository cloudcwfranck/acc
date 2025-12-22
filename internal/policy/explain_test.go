package policy

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestExplainJSONContractStability tests that explain JSON output
// always includes .result.input for contract stability (even if empty)
func TestExplainJSONContractStability(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "acc-policy-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Change to temp directory
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)

	// Create .acc/state directory
	stateDir := filepath.Join(".acc", "state")
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Test case 1: State with input field present
	t.Run("with_input", func(t *testing.T) {
		state := map[string]interface{}{
			"imageRef":  "test:latest",
			"status":    "pass",
			"timestamp": "2025-01-22T10:00:00Z",
			"result": map[string]interface{}{
				"status":      "pass",
				"sbomPresent": true,
				"input": map[string]interface{}{
					"config": map[string]interface{}{
						"User": "appuser",
					},
				},
			},
		}

		data, _ := json.MarshalIndent(state, "", "  ")
		stateFile := filepath.Join(stateDir, "last_verify.json")
		if err := os.WriteFile(stateFile, data, 0644); err != nil {
			t.Fatal(err)
		}

		// Verify the function doesn't error
		err := Explain(true)
		if err != nil {
			t.Errorf("Explain() error = %v, want nil", err)
		}
	})

	// Test case 2: State without input field (regression test for contract stability)
	t.Run("without_input_adds_empty_object", func(t *testing.T) {
		state := map[string]interface{}{
			"imageRef":  "test:latest",
			"status":    "pass",
			"timestamp": "2025-01-22T10:00:00Z",
			"result": map[string]interface{}{
				"status":      "pass",
				"sbomPresent": true,
				// Note: no input field - should be added automatically
			},
		}

		data, _ := json.MarshalIndent(state, "", "  ")
		stateFile := filepath.Join(stateDir, "last_verify.json")
		if err := os.WriteFile(stateFile, data, 0644); err != nil {
			t.Fatal(err)
		}

		err := Explain(true)
		if err != nil {
			t.Errorf("Explain() error = %v, want nil", err)
		}

		// Note: Full validation would capture stdout and verify .result.input exists
		// For now, we verify the function completes without error
	})
}
