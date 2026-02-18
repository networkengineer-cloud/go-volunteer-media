package handlers

import "time"

// Authentication and account lockout
const (
	MaxFailedLoginAttempts = 5
	AccountLockoutDuration = 30 * time.Minute
)

// Token expiry durations
const (
	PasswordResetTokenExpiry = 1 * time.Hour
	SetupTokenExpiry         = 7 * 24 * time.Hour
)

// TokenLookupPrefixLength is the number of plaintext token characters stored for
// indexed lookups. Must be <= the length of a token produced by generateSecureToken (64).
const TokenLookupPrefixLength = 16
