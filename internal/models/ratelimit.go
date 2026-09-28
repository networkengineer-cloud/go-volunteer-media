package models

// RateLimitCounter is one fixed-window request counter for the shared rate
// limiter in internal/ratelimit. Keeping the counters in Postgres (rather
// than in process memory) means every replica enforces the same limit.
//
// Key is a SHA-256 hex digest of the limiter scope and its subject (e.g.
// username + client IP), so usernames and addresses are not stored in the
// clear. WindowStart is the Unix time (seconds) the current window began;
// Count is the number of hits recorded in that window. Rows for expired
// windows are deleted by the limiter's sweep.
type RateLimitCounter struct {
	Key         string `gorm:"primaryKey;size:64"`
	WindowStart int64  `gorm:"not null;index"`
	Count       int    `gorm:"not null"`
}
