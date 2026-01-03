package crypto

import (
	"testing"
)

func TestCanonicalizeJCS_Deterministic(t *testing.T) {
	// Test that same input produces same output
	input := map[string]interface{}{
		"b": 2,
		"a": 1,
		"c": map[string]interface{}{
			"y": 25,
			"x": 24,
		},
	}

	canonical1, err1 := CanonicalizeJCS(input)
	if err1 != nil {
		t.Fatalf("First canonicalization failed: %v", err1)
	}

	canonical2, err2 := CanonicalizeJCS(input)
	if err2 != nil {
		t.Fatalf("Second canonicalization failed: %v", err2)
	}

	if string(canonical1) != string(canonical2) {
		t.Errorf("Canonicalization is not deterministic:\n  First:  %s\n  Second: %s", canonical1, canonical2)
	}
}

func TestCanonicalizeJCS_KeyOrdering(t *testing.T) {
	// RFC 8785 requires keys to be sorted
	input := map[string]interface{}{
		"zebra": 1,
		"apple": 2,
		"mango": 3,
	}

	canonical, err := CanonicalizeJCS(input)
	if err != nil {
		t.Fatalf("Canonicalization failed: %v", err)
	}

	// Should have keys in sorted order: "apple" before "mango" before "zebra"
	expected := `{"apple":2,"mango":3,"zebra":1}`
	if string(canonical) != expected {
		t.Errorf("Keys not properly ordered:\n  Got:      %s\n  Expected: %s", canonical, expected)
	}
}

func TestCanonicalizeJCS_NestedObjects(t *testing.T) {
	// Test nested objects
	input := map[string]interface{}{
		"outer": map[string]interface{}{
			"inner": map[string]interface{}{
				"z": 3,
				"a": 1,
			},
		},
	}

	canonical, err := CanonicalizeJCS(input)
	if err != nil {
		t.Fatalf("Canonicalization failed: %v", err)
	}

	// Nested keys should also be sorted
	expected := `{"outer":{"inner":{"a":1,"z":3}}}`
	if string(canonical) != expected {
		t.Errorf("Nested canonicalization incorrect:\n  Got:      %s\n  Expected: %s", canonical, expected)
	}
}

func TestCanonicalizeJCS_Arrays(t *testing.T) {
	// Arrays should maintain order
	input := map[string]interface{}{
		"numbers": []int{3, 1, 2},
	}

	canonical, err := CanonicalizeJCS(input)
	if err != nil {
		t.Fatalf("Canonicalization failed: %v", err)
	}

	// Array order should be preserved
	expected := `{"numbers":[3,1,2]}`
	if string(canonical) != expected {
		t.Errorf("Array canonicalization incorrect:\n  Got:      %s\n  Expected: %s", canonical, expected)
	}
}

func TestCanonicalizeJCS_NoWhitespace(t *testing.T) {
	// Canonical form should have no whitespace
	input := map[string]interface{}{
		"a": 1,
		"b": 2,
	}

	canonical, err := CanonicalizeJCS(input)
	if err != nil {
		t.Fatalf("Canonicalization failed: %v", err)
	}

	// Should not contain spaces or newlines
	for _, c := range string(canonical) {
		if c == ' ' || c == '\n' || c == '\t' || c == '\r' {
			t.Errorf("Canonical form contains whitespace: %q", canonical)
			break
		}
	}
}

func TestCanonicalizeJCS_Stability(t *testing.T) {
	// Test that same data in different order produces same canonical form
	input1 := map[string]interface{}{"a": 1, "b": 2, "c": 3}
	input2 := map[string]interface{}{"c": 3, "a": 1, "b": 2}
	input3 := map[string]interface{}{"b": 2, "c": 3, "a": 1}

	canonical1, _ := CanonicalizeJCS(input1)
	canonical2, _ := CanonicalizeJCS(input2)
	canonical3, _ := CanonicalizeJCS(input3)

	if string(canonical1) != string(canonical2) || string(canonical1) != string(canonical3) {
		t.Errorf("Different input orders produced different canonical forms:\n  1: %s\n  2: %s\n  3: %s",
			canonical1, canonical2, canonical3)
	}
}
