package inventory

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
)

// GenerateEnrollmentSecret make random secret (show only on creation)
func GenerateEnrollmentSecret() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("rand: %w", err)
	}

	// URL-safe, no padding - easy to copy/paste
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// HashToken save secret in SHA-256 hash
func HashToken(secret string) string {
	sum := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(sum[:])
}