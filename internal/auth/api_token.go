package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

// APITokenPrefix identifies bearer tokens as opaque API tokens (as opposed to
// JWTs), so AuthRequired can branch on it before attempting JWT parsing.
const APITokenPrefix = "pat_"

// GeneratedAPIToken holds a newly created token's one-time plaintext value
// alongside the values that get persisted to the database.
type GeneratedAPIToken struct {
	Token         string // full secret, e.g. "pat_<64 hex chars>" — return to the caller once, never store
	Hash          string // sha256 hex digest of Token — store this
	DisplayPrefix string // e.g. "pat_ab12cd34" — store this, safe to display in a token list
}

// GenerateAPIToken creates a new high-entropy API token (32 random bytes,
// hex-encoded, prefixed with APITokenPrefix).
func GenerateAPIToken() (*GeneratedAPIToken, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return nil, fmt.Errorf("failed to generate random token: %w", err)
	}

	secret := hex.EncodeToString(raw)
	token := APITokenPrefix + secret

	return &GeneratedAPIToken{
		Token:         token,
		Hash:          HashAPIToken(token),
		DisplayPrefix: APITokenPrefix + secret[:8],
	}, nil
}

// HashAPIToken returns the SHA-256 hex digest of a full token value. Used
// both when persisting a newly generated token and when looking up a
// presented token during authentication. Unlike a user-chosen password, the
// token is already high-entropy (256 bits), so there's no offline
// brute-force risk to slow down with bcrypt — a fast, deterministic hash is
// the right tool here.
func HashAPIToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// IsAPIToken reports whether a bearer token looks like an API token (vs a
// JWT), based on its prefix.
func IsAPIToken(token string) bool {
	return len(token) > len(APITokenPrefix) && strings.HasPrefix(token, APITokenPrefix)
}
