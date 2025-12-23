package trust

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestTrustStatusJSONGolden tests JSON output against golden files
func TestTrustStatusJSONGolden(t *testing.T) {
	scenarios := []struct {
		name       string
		stateFile  string
		goldenFile string
		imageRef   string
	}{
		{
			name:       "pass",
			stateFile:  "pass-state.json",
			goldenFile: "pass.json",
			imageRef:   "demo-app:ok",
		},
		{
			name:       "fail-with-violations",
			stateFile:  "fail-state.json",
			goldenFile: "fail-with-violations.json",
			imageRef:   "demo-app:root",
		},
		{
			name:       "unknown",
			stateFile:  "",
			goldenFile: "unknown.json",
			imageRef:   "never-verified:latest",
		},
	}

	for _, tc := range scenarios {
		t.Run(tc.name, func(t *testing.T) {
			// Create temporary directory
			tmpDir, err := os.MkdirTemp("", "acc-test-*")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(tmpDir)

			// Change to temp directory
			oldWd, _ := os.Getwd()
			defer os.Chdir(oldWd)
			os.Chdir(tmpDir)

			// Setup state if specified
			if tc.stateFile != "" {
				stateDir := filepath.Join(".acc", "state")
				if err := os.MkdirAll(stateDir, 0755); err != nil {
					t.Fatal(err)
				}

				var stateData map[string]interface{}
				switch tc.stateFile {
				case "pass-state.json":
					stateData = map[string]interface{}{
						"imageRef":  tc.imageRef,
						"status":    "pass",
						"timestamp": "2025-01-15T10:00:00Z",
						"result": map[string]interface{}{
							"sbomPresent": true,
							"policyResult": map[string]interface{}{
								"allow":      true,
								"violations": []interface{}{},
								"warnings":   []interface{}{},
							},
						},
					}
				case "fail-state.json":
					stateData = map[string]interface{}{
						"imageRef":  tc.imageRef,
						"status":    "fail",
						"timestamp": "2025-01-15T10:00:00Z",
						"result": map[string]interface{}{
							"sbomPresent": true,
							"policyResult": map[string]interface{}{
								"allow": false,
								"violations": []interface{}{
									map[string]interface{}{
										"rule":     "no-root-user",
										"severity": "high",
										"result":   "fail",
										"message":  "Image runs as root user",
									},
								},
								"warnings": []interface{}{},
							},
						},
					}
				}

				data, _ := json.MarshalIndent(stateData, "", "  ")
				if err := os.WriteFile(filepath.Join(stateDir, "last_verify.json"), data, 0644); err != nil {
					t.Fatal(err)
				}
			}

			// Get status
			result, err := Status(tc.imageRef, false, true)
			if err != nil {
				t.Errorf("Status() error = %v, want nil", err)
			}

			// Load golden file (from project root)
			projectRoot := filepath.Join(oldWd, "..", "..")
			goldenPath := filepath.Join(projectRoot, "testdata", "golden", "trust", tc.goldenFile)
			goldenData, err := os.ReadFile(goldenPath)
			if err != nil {
				t.Fatalf("Failed to read golden file %s: %v", goldenPath, err)
			}

			// Parse golden JSON
			var golden StatusResult
			if err := json.Unmarshal(goldenData, &golden); err != nil {
				t.Fatalf("Failed to parse golden JSON: %v", err)
			}

			// Compare fields (excluding timestamp which may vary in real scenarios)
			if result.SchemaVersion != golden.SchemaVersion {
				t.Errorf("SchemaVersion = %s, want %s", result.SchemaVersion, golden.SchemaVersion)
			}
			if result.ImageRef != golden.ImageRef {
				t.Errorf("ImageRef = %s, want %s", result.ImageRef, golden.ImageRef)
			}
			if result.Status != golden.Status {
				t.Errorf("Status = %s, want %s", result.Status, golden.Status)
			}
			if result.SBOMPresent != golden.SBOMPresent {
				t.Errorf("SBOMPresent = %v, want %v", result.SBOMPresent, golden.SBOMPresent)
			}
			if len(result.Violations) != len(golden.Violations) {
				t.Errorf("len(Violations) = %d, want %d", len(result.Violations), len(golden.Violations))
			}
			if len(result.Warnings) != len(golden.Warnings) {
				t.Errorf("len(Warnings) = %d, want %d", len(result.Warnings), len(golden.Warnings))
			}

			// Verify violations match
			for i, v := range result.Violations {
				if i < len(golden.Violations) {
					if v.Rule != golden.Violations[i].Rule {
						t.Errorf("Violation[%d].Rule = %s, want %s", i, v.Rule, golden.Violations[i].Rule)
					}
					if v.Severity != golden.Violations[i].Severity {
						t.Errorf("Violation[%d].Severity = %s, want %s", i, v.Severity, golden.Violations[i].Severity)
					}
				}
			}
		})
	}
}

// TestTrustStatusJSONFieldOrdering tests that JSON fields are consistently ordered
func TestTrustStatusJSONFieldOrdering(t *testing.T) {
	result := &StatusResult{
		SchemaVersion: "v0.2",
		ImageRef:      "test:latest",
		Status:        "pass",
		SBOMPresent:   true,
		Violations:    []Violation{},
		Warnings:      []Violation{},
		Attestations:  []string{},
		Timestamp:     "2025-01-15T10:00:00Z",
	}

	json1 := result.FormatJSON()
	json2 := result.FormatJSON()

	if json1 != json2 {
		t.Error("JSON output is not deterministic")
	}

	// Verify field order (Go json.Marshal guarantees struct field order)
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(json1), &parsed); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	// All required fields should exist
	required := []string{"schemaVersion", "imageRef", "status", "sbomPresent", "violations", "warnings", "attestations", "timestamp"}
	for _, field := range required {
		if _, exists := parsed[field]; !exists {
			t.Errorf("Required field %s missing from JSON", field)
		}
	}
}

// TestTrustStatusJSONSchemaVersion tests schema version stability
func TestTrustStatusJSONSchemaVersion(t *testing.T) {
	result := &StatusResult{
		SchemaVersion: "v0.2",
		ImageRef:      "test:latest",
		Status:        "pass",
		SBOMPresent:   true,
		Violations:    []Violation{},
		Warnings:      []Violation{},
		Attestations:  []string{},
		Timestamp:     "2025-01-15T10:00:00Z",
	}

	jsonStr := result.FormatJSON()
	var parsed map[string]interface{}
	json.Unmarshal([]byte(jsonStr), &parsed)

	if version, ok := parsed["schemaVersion"].(string); !ok || version != "v0.2" {
		t.Errorf("schemaVersion = %v, want v0.2", parsed["schemaVersion"])
	}
}

// TestTrustStatusJSONSchemaDrift detects unintended schema changes
func TestTrustStatusJSONSchemaDrift(t *testing.T) {
	result := &StatusResult{
		SchemaVersion: "v0.2",
		ImageRef:      "test:latest",
		Status:        "pass",
		ProfileUsed:   "default",
		SBOMPresent:   true,
		Violations:    []Violation{},
		Warnings:      []Violation{},
		Attestations:  []string{},
		Timestamp:     "2025-01-15T10:00:00Z",
	}

	jsonStr := result.FormatJSON()
	var parsed map[string]interface{}
	json.Unmarshal([]byte(jsonStr), &parsed)

	// Define the expected schema
	expectedFields := map[string]string{
		"schemaVersion": "string",
		"imageRef":      "string",
		"status":        "string",
		"sbomPresent":   "bool",
		"violations":    "array",
		"warnings":      "array",
		"attestations":  "array",
		"timestamp":     "string",
	}

	// Optional fields
	optionalFields := map[string]string{
		"profileUsed": "string",
	}

	// Check required fields
	for field, expectedType := range expectedFields {
		value, exists := parsed[field]
		if !exists {
			t.Errorf("Required field %s missing from JSON", field)
			continue
		}

		actualType := getJSONType(value)
		if actualType != expectedType {
			t.Errorf("Field %s has type %s, want %s", field, actualType, expectedType)
		}
	}

	// Check optional fields (if present)
	for field, expectedType := range optionalFields {
		if value, exists := parsed[field]; exists {
			actualType := getJSONType(value)
			if actualType != expectedType {
				t.Errorf("Optional field %s has type %s, want %s", field, actualType, expectedType)
			}
		}
	}

	// Detect unexpected fields
	allExpected := make(map[string]bool)
	for k := range expectedFields {
		allExpected[k] = true
	}
	for k := range optionalFields {
		allExpected[k] = true
	}

	for field := range parsed {
		if !allExpected[field] {
			t.Errorf("Unexpected field in JSON: %s (possible schema drift)", field)
		}
	}
}

func getJSONType(value interface{}) string {
	switch value.(type) {
	case string:
		return "string"
	case bool:
		return "bool"
	case float64:
		return "number"
	case []interface{}:
		return "array"
	case map[string]interface{}:
		return "object"
	default:
		return "unknown"
	}
}
