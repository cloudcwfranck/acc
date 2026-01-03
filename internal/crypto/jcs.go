package crypto

import (
	"encoding/json"
	"fmt"

	"github.com/gowebpki/jcs"
)

// CanonicalizeJCS performs RFC 8785 JSON Canonicalization Scheme (JCS) on the given value.
// This produces deterministic, stable canonical bytes suitable for signing.
//
// The input value is first marshaled to JSON, then canonicalized according to RFC 8785.
// This ensures consistent ordering of object keys and deterministic formatting.
func CanonicalizeJCS(v any) ([]byte, error) {
	// First marshal to JSON
	jsonBytes, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal value: %w", err)
	}

	// Apply JCS canonicalization
	canonical, err := jcs.Transform(jsonBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to canonicalize JSON: %w", err)
	}

	return canonical, nil
}
