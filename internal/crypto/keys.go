package crypto

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
)

// KeyInfo contains the resolved signing key and metadata
type KeyInfo struct {
	PrivateKey ed25519.PrivateKey
	PublicKey  ed25519.PublicKey
	KeyID      string
}

// ResolveSigningKey resolves the signing key from the following sources in order:
// 1. ACC_SIGNING_KEY environment variable (base64-encoded 64-byte ed25519 private key)
// 2. ACC_SIGNING_KEY_FILE environment variable (path to key file)
// 3. .acc/keys/ed25519.key in the project root
//
// Returns the private key, public key, keyId, and any error.
// Does NOT generate a new key if none found - use EnsureKeyForAttest for that.
func ResolveSigningKey(projectRoot string) (*KeyInfo, error) {
	// 1. Check ACC_SIGNING_KEY environment variable
	if envKey := os.Getenv("ACC_SIGNING_KEY"); envKey != "" {
		privBytes, err := base64.StdEncoding.DecodeString(envKey)
		if err != nil {
			return nil, fmt.Errorf("ACC_SIGNING_KEY is not valid base64: %w", err)
		}
		if len(privBytes) != ed25519.PrivateKeySize {
			return nil, fmt.Errorf("ACC_SIGNING_KEY has invalid length: expected %d bytes, got %d", ed25519.PrivateKeySize, len(privBytes))
		}

		priv := ed25519.PrivateKey(privBytes)
		pub := priv.Public().(ed25519.PublicKey)
		keyID := KeyIDFromPublicKeyEd25519(pub)

		return &KeyInfo{
			PrivateKey: priv,
			PublicKey:  pub,
			KeyID:      keyID,
		}, nil
	}

	// 2. Check ACC_SIGNING_KEY_FILE environment variable
	if envKeyFile := os.Getenv("ACC_SIGNING_KEY_FILE"); envKeyFile != "" {
		return loadKeyFromFile(envKeyFile)
	}

	// 3. Check .acc/keys/ed25519.key in project root
	keyPath := filepath.Join(projectRoot, ".acc", "keys", "ed25519.key")
	if _, err := os.Stat(keyPath); err == nil {
		return loadKeyFromFile(keyPath)
	}

	return nil, fmt.Errorf("no signing key found (checked ACC_SIGNING_KEY, ACC_SIGNING_KEY_FILE, and %s)", keyPath)
}

// EnsureKeyForAttest resolves or generates a signing key for attestation creation.
// If no key exists, it generates a new ed25519 keypair and writes it to .acc/keys/ed25519.key.
func EnsureKeyForAttest(projectRoot string) (*KeyInfo, error) {
	// Try to resolve existing key first
	keyInfo, err := ResolveSigningKey(projectRoot)
	if err == nil {
		return keyInfo, nil
	}

	// No key found - generate new one
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate ed25519 keypair: %w", err)
	}

	keyID := KeyIDFromPublicKeyEd25519(pub)

	// Write to .acc/keys/ed25519.key
	keyPath := filepath.Join(projectRoot, ".acc", "keys", "ed25519.key")
	if err := writeKeyToFile(keyPath, priv); err != nil {
		return nil, fmt.Errorf("failed to write signing key: %w", err)
	}

	return &KeyInfo{
		PrivateKey: priv,
		PublicKey:  pub,
		KeyID:      keyID,
	}, nil
}

// loadKeyFromFile loads an ed25519 private key from a file.
// The file can contain either:
// - Raw 64 bytes (ed25519.PrivateKeySize)
// - Base64-encoded 64 bytes
func loadKeyFromFile(path string) (*KeyInfo, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read key file %s: %w", path, err)
	}

	var privBytes []byte

	// Try raw bytes first
	if len(data) == ed25519.PrivateKeySize {
		privBytes = data
	} else {
		// Try base64 decoding
		decoded, err := base64.StdEncoding.DecodeString(string(data))
		if err != nil {
			return nil, fmt.Errorf("key file %s is not raw bytes or valid base64: %w", path, err)
		}
		if len(decoded) != ed25519.PrivateKeySize {
			return nil, fmt.Errorf("decoded key from %s has invalid length: expected %d bytes, got %d", path, ed25519.PrivateKeySize, len(decoded))
		}
		privBytes = decoded
	}

	priv := ed25519.PrivateKey(privBytes)
	pub := priv.Public().(ed25519.PublicKey)
	keyID := KeyIDFromPublicKeyEd25519(pub)

	return &KeyInfo{
		PrivateKey: priv,
		PublicKey:  pub,
		KeyID:      keyID,
	}, nil
}

// writeKeyToFile writes an ed25519 private key to a file with 0600 permissions.
// The key is stored as raw 64 bytes.
func writeKeyToFile(path string, priv ed25519.PrivateKey) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create key directory: %w", err)
	}

	// Write key with restrictive permissions
	if err := os.WriteFile(path, priv, 0600); err != nil {
		return fmt.Errorf("failed to write key file: %w", err)
	}

	return nil
}
