package auth

import (
	"strings"
	"testing"
)

func TestGenerateApiToken(t *testing.T) {
	got, err := GenerateApiToken()
	if err != nil {
		t.Fatalf("GenerateApiToken() error = %v", err)
	}

	if !strings.HasPrefix(got.Token, ApiTokenPrefix) {
		t.Errorf("Token = %q, want prefix %q", got.Token, ApiTokenPrefix)
	}

	wantLen := len(ApiTokenPrefix) + 64 // 32 random bytes, hex-encoded
	if len(got.Token) != wantLen {
		t.Errorf("len(Token) = %d, want %d", len(got.Token), wantLen)
	}

	if got.Hash != HashApiToken(got.Token) {
		t.Errorf("Hash = %q, want %q", got.Hash, HashApiToken(got.Token))
	}

	wantPrefix := ApiTokenPrefix + got.Token[len(ApiTokenPrefix):len(ApiTokenPrefix)+8]
	if got.DisplayPrefix != wantPrefix {
		t.Errorf("DisplayPrefix = %q, want %q", got.DisplayPrefix, wantPrefix)
	}
}

func TestGenerateApiToken_Unique(t *testing.T) {
	first, err := GenerateApiToken()
	if err != nil {
		t.Fatalf("GenerateApiToken() error = %v", err)
	}
	second, err := GenerateApiToken()
	if err != nil {
		t.Fatalf("GenerateApiToken() error = %v", err)
	}

	if first.Token == second.Token {
		t.Error("GenerateApiToken() produced the same token twice")
	}
}

func TestHashApiToken(t *testing.T) {
	tests := []struct {
		name  string
		a, b  string
		equal bool
	}{
		{name: "same input, same hash", a: "pat_abc123", b: "pat_abc123", equal: true},
		{name: "different input, different hash", a: "pat_abc123", b: "pat_abc124", equal: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotEqual := HashApiToken(tt.a) == HashApiToken(tt.b)
			if gotEqual != tt.equal {
				t.Errorf("HashApiToken(%q) == HashApiToken(%q) = %v, want %v", tt.a, tt.b, gotEqual, tt.equal)
			}
		})
	}

	hash := HashApiToken("pat_abc123")
	if len(hash) != 64 { // sha256 hex-encoded
		t.Errorf("len(HashApiToken(...)) = %d, want 64", len(hash))
	}
}

func TestIsApiToken(t *testing.T) {
	tests := []struct {
		name  string
		token string
		want  bool
	}{
		{name: "api token", token: "pat_abc123", want: true},
		{name: "jwt-looking token", token: "eyJhbGciOiJIUzI1NiIs.eyJ1c2VyX2lkIjoxfQ.sig", want: false},
		{name: "empty string", token: "", want: false},
		{name: "just the prefix, no secret", token: "pat_", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsApiToken(tt.token); got != tt.want {
				t.Errorf("IsApiToken(%q) = %v, want %v", tt.token, got, tt.want)
			}
		})
	}
}
