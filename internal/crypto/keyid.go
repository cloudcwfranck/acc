package crypto

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base32"
	"strings"
)

// KeyIDFromPublicKeyEd25519 derives a deterministic keyId from an ed25519 public key.
//
// Algorithm:
//
//	keyId = "ed25519:" + lower(base32(no padding)(sha256(publicKeyBytes)))[:26]
//
// This ensures:
// - Deterministic: same public key always produces same keyId
// - Collision-resistant: uses SHA-256 hash
// - Compact: 26-character identifier
// - URL-safe: base32 encoding with lowercase
func KeyIDFromPublicKeyEd25519(pub ed25519.PublicKey) string {
	// Compute SHA-256 hash of the raw public key bytes
	hash := sha256.Sum256(pub)

	// Encode using RFC 4648 standard base32 without padding
	encoded := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(hash[:])

	// Convert to lowercase and take first 26 characters
	keyID := "ed25519:" + strings.ToLower(encoded)[:26]

	return keyID
}
