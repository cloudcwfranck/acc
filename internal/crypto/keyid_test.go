package crypto

import (
	"crypto/ed25519"
	"encoding/hex"
	"testing"
)

func TestKeyIDFromPublicKeyEd25519_Deterministic(t *testing.T) {
	// Create a fixed test key by decoding known bytes
	pubKeyHex := "d75a980182b10ab7d54bfed3c964073a0ee172f3daa62325af021a68f707511a"
	pubKeyBytes, err := hex.DecodeString(pubKeyHex)
	if err != nil {
		t.Fatalf("Failed to decode test public key: %v", err)
	}

	publicKey := ed25519.PublicKey(pubKeyBytes)

	// Compute keyId twice to ensure determinism
	keyID1 := KeyIDFromPublicKeyEd25519(publicKey)
	keyID2 := KeyIDFromPublicKeyEd25519(publicKey)

	// Should be identical
	if keyID1 != keyID2 {
		t.Errorf("KeyID is not deterministic: %s != %s", keyID1, keyID2)
	}

	// Should have expected format
	if len(keyID1) != len("ed25519:")+26 {
		t.Errorf("KeyID has unexpected length: %d (expected %d)", len(keyID1), len("ed25519:")+26)
	}

	if keyID1[:8] != "ed25519:" {
		t.Errorf("KeyID does not start with 'ed25519:': %s", keyID1)
	}

	// Should be lowercase
	for _, c := range keyID1[8:] {
		if c >= 'A' && c <= 'Z' {
			t.Errorf("KeyID contains uppercase characters: %s", keyID1)
			break
		}
	}

	t.Logf("Generated keyId: %s", keyID1)
}

func TestKeyIDFromPublicKeyEd25519_DifferentKeys(t *testing.T) {
	// Generate two different keys
	pub1, _, err1 := ed25519.GenerateKey(nil)
	pub2, _, err2 := ed25519.GenerateKey(nil)

	if err1 != nil || err2 != nil {
		t.Fatalf("Failed to generate keys: %v, %v", err1, err2)
	}

	keyID1 := KeyIDFromPublicKeyEd25519(pub1)
	keyID2 := KeyIDFromPublicKeyEd25519(pub2)

	// Should be different
	if keyID1 == keyID2 {
		t.Errorf("Different public keys produced same keyId: %s", keyID1)
	}
}

func TestKeyIDFromPublicKeyEd25519_Format(t *testing.T) {
	// Generate a test key
	pub, _, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	keyID := KeyIDFromPublicKeyEd25519(pub)

	// Check format: ed25519:<26 lowercase base32 chars>
	if len(keyID) != 34 { // "ed25519:" (8) + 26 chars
		t.Errorf("KeyID has wrong length: got %d, want 34", len(keyID))
	}

	if keyID[:8] != "ed25519:" {
		t.Errorf("KeyID has wrong prefix: got %s, want 'ed25519:'", keyID[:8])
	}

	// Check that suffix is valid base32 (uppercase letters and digits 2-7)
	suffix := keyID[8:]
	validChars := "abcdefghijklmnopqrstuvwxyz234567"
	for _, c := range suffix {
		valid := false
		for _, vc := range validChars {
			if c == vc {
				valid = true
				break
			}
		}
		if !valid {
			t.Errorf("KeyID contains invalid character: %c (in %s)", c, keyID)
		}
	}
}
