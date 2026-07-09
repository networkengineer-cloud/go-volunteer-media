package auth

import (
	"strings"
	"testing"
)

func TestGenerateAPIToken(t *testing.T) {
	got, err := GenerateAPIToken()
	if err != nil {
		t.Fatalf("GenerateAPIToken() error = %v", err)
	}

	if !strings.HasPrefix(got.Token, APITokenPrefix) {
		t.Errorf("Token = %q, want prefix %q", got.Token, APITokenPrefix)
	}

	wantLen := len(APITokenPrefix) + 64 // 32 random bytes, hex-encoded
	if len(got.Token) != wantLen {
		t.Errorf("len(Token) = %d, want %d", len(got.Token), wantLen)
	}

	if got.Hash != HashAPIToken(got.Token) {
		t.Errorf("Hash = %q, want %q", got.Hash, HashAPIToken(got.Token))
	}

	wantPrefix := APITokenPrefix + got.Token[len(APITokenPrefix):len(APITokenPrefix)+8]
	if got.DisplayPrefix != wantPrefix {
		t.Errorf("DisplayPrefix = %q, want %q", got.DisplayPrefix, wantPrefix)
	}
}

func TestGenerateAPIToken_Unique(t *testing.T) {
	first, err := GenerateAPIToken()
	if err != nil {
		t.Fatalf("GenerateAPIToken() error = %v", err)
	}
	second, err := GenerateAPIToken()
	if err != nil {
		t.Fatalf("GenerateAPIToken() error = %v", err)
	}

	if first.Token == second.Token {
		t.Error("GenerateAPIToken() produced the same token twice")
	}
}

func TestHashAPIToken(t *testing.T) {
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
			gotEqual := HashAPIToken(tt.a) == HashAPIToken(tt.b)
			if gotEqual != tt.equal {
				t.Errorf("HashAPIToken(%q) == HashAPIToken(%q) = %v, want %v", tt.a, tt.b, gotEqual, tt.equal)
			}
		})
	}

	hash := HashAPIToken("pat_abc123")
	if len(hash) != 64 { // sha256 hex-encoded
		t.Errorf("len(HashAPIToken(...)) = %d, want 64", len(hash))
	}
}

func TestIsAPIToken(t *testing.T) {
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
			if got := IsAPIToken(tt.token); got != tt.want {
				t.Errorf("IsAPIToken(%q) = %v, want %v", tt.token, got, tt.want)
			}
		})
	}
}
