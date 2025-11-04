package auth

import (
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// resetJWTSecret resets the JWT secret for testing
func resetJWTSecret() {
	jwtSecretOnce = sync.Once{}
	jwtSecret = nil
}

func TestHashPassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{
			name:     "valid password",
			password: "ValidPassword123",
			wantErr:  false,
		},
		{
			name:     "empty password",
			password: "",
			wantErr:  false, // bcrypt can hash empty strings
		},
		{
			name:     "long password within limits (72 bytes)",
			password: strings.Repeat("a", 72),
			wantErr:  false,
		},
		{
			name:     "password exceeding bcrypt limit (>72 bytes)",
			password: strings.Repeat("a", 100),
			wantErr:  true, // bcrypt has a 72 byte limit
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := HashPassword(tt.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("HashPassword() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if len(hash) == 0 {
					t.Errorf("HashPassword() returned empty hash")
				}
				// Hash should start with bcrypt identifier
				if !strings.HasPrefix(hash, "$2a$") && !strings.HasPrefix(hash, "$2b$") {
					t.Errorf("HashPassword() returned invalid bcrypt hash format")
				}
			}
		})
	}
}

func TestCheckPassword(t *testing.T) {
	validPassword := "MySecurePassword123"
	validHash, _ := HashPassword(validPassword)

	tests := []struct {
		name           string
		hashedPassword string
		password       string
		wantErr        bool
	}{
		{
			name:           "correct password",
			hashedPassword: validHash,
			password:       validPassword,
			wantErr:        false,
		},
		{
			name:           "incorrect password",
			hashedPassword: validHash,
			password:       "WrongPassword",
			wantErr:        true,
		},
		{
			name:           "empty password against hash",
			hashedPassword: validHash,
			password:       "",
			wantErr:        true,
		},
		{
			name:           "invalid hash format",
			hashedPassword: "not-a-valid-hash",
			password:       validPassword,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckPassword(tt.hashedPassword, tt.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckPassword() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCheckSecretEntropy(t *testing.T) {
	tests := []struct {
		name    string
		secret  string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid random secret (32+ chars)",
			secret:  "aB3!xZ7$pQ9#mN4@kL8&hG2%fD6*jS5^",
			wantErr: false,
		},
		{
			name:    "all same character",
			secret:  strings.Repeat("1", 32),
			wantErr: true,
			errMsg:  "insufficient entropy",
		},
		{
			name:    "insufficient variety",
			secret:  "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaabb",
			wantErr: true,
			errMsg:  "insufficient character variety",
		},
		{
			name:    "contains 'change'",
			secret:  "please-change-this-secret-value-now!",
			wantErr: true,
			errMsg:  "default/example value",
		},
		{
			name:    "contains 'example'",
			secret:  "example-jwt-secret-32-characters!",
			wantErr: true,
			errMsg:  "default/example value",
		},
		{
			name:    "contains 'test'",
			secret:  "test-secret-value-for-testing-app!",
			wantErr: true,
			errMsg:  "default/example value",
		},
		{
			name:    "contains 'default'",
			secret:  "default-jwt-secret-32-chars-long!",
			wantErr: true,
			errMsg:  "default/example value",
		},
		{
			name:    "openssl generated secret (base64)",
			secret:  "Zx9$pLm3!qWe7@rTy4#uI8&oP2%aS6*dFg5^hJ1",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkSecretEntropy(tt.secret)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkSecretEntropy() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("checkSecretEntropy() error message = %v, want to contain %v", err.Error(), tt.errMsg)
			}
		})
	}
}

func TestGenerateToken(t *testing.T) {
	// Set up valid JWT_SECRET for testing (generated with openssl rand -base64 32)
	os.Setenv("JWT_SECRET", "L5WTt6D+6R55YfKzwqPRAEX5bR0bkNo4i58jYKL0wsk=")
	defer os.Unsetenv("JWT_SECRET")
	defer resetJWTSecret()
	resetJWTSecret()

	tests := []struct {
		name    string
		userID  uint
		isAdmin bool
		wantErr bool
	}{
		{
			name:    "generate token for regular user",
			userID:  1,
			isAdmin: false,
			wantErr: false,
		},
		{
			name:    "generate token for admin user",
			userID:  2,
			isAdmin: true,
			wantErr: false,
		},
		{
			name:    "generate token for user ID 0",
			userID:  0,
			isAdmin: false,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := GenerateToken(tt.userID, tt.isAdmin)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if len(token) == 0 {
					t.Errorf("GenerateToken() returned empty token")
				}
				// Token should have three parts (header.payload.signature)
				parts := strings.Split(token, ".")
				if len(parts) != 3 {
					t.Errorf("GenerateToken() returned invalid JWT format, got %d parts, want 3", len(parts))
				}
			}
		})
	}
}

func TestValidateToken(t *testing.T) {
	// Set up valid JWT_SECRET for testing (generated with openssl rand -base64 32)
	testSecret := "L5WTt6D+6R55YfKzwqPRAEX5bR0bkNo4i58jYKL0wsk="
	os.Setenv("JWT_SECRET", testSecret)
	defer os.Unsetenv("JWT_SECRET")
	defer resetJWTSecret()
	resetJWTSecret()

	// Generate a valid token
	validToken, _ := GenerateToken(1, false)
	adminToken, _ := GenerateToken(2, true)

	// Generate an expired token
	claims := Claims{
		UserID:  3,
		IsAdmin: false,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		},
	}
	expiredToken, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(testSecret))

	// Generate token with wrong signature
	wrongToken, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		UserID:  4,
		IsAdmin: false,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}).SignedString([]byte("wrong-secret-key-different-from-env"))

	tests := []struct {
		name        string
		tokenString string
		wantErr     bool
		wantUserID  uint
		wantIsAdmin bool
	}{
		{
			name:        "valid regular user token",
			tokenString: validToken,
			wantErr:     false,
			wantUserID:  1,
			wantIsAdmin: false,
		},
		{
			name:        "valid admin token",
			tokenString: adminToken,
			wantErr:     false,
			wantUserID:  2,
			wantIsAdmin: true,
		},
		{
			name:        "expired token",
			tokenString: expiredToken,
			wantErr:     true,
		},
		{
			name:        "token with wrong signature",
			tokenString: wrongToken,
			wantErr:     true,
		},
		{
			name:        "malformed token",
			tokenString: "not.a.valid.jwt.token",
			wantErr:     true,
		},
		{
			name:        "empty token",
			tokenString: "",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := ValidateToken(tt.tokenString)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if claims.UserID != tt.wantUserID {
					t.Errorf("ValidateToken() UserID = %v, want %v", claims.UserID, tt.wantUserID)
				}
				if claims.IsAdmin != tt.wantIsAdmin {
					t.Errorf("ValidateToken() IsAdmin = %v, want %v", claims.IsAdmin, tt.wantIsAdmin)
				}
			}
		})
	}
}

func TestGetJWTSecret(t *testing.T) {
	tests := []struct {
		name      string
		envSecret string
		wantErr   bool
		setup     func()
		cleanup   func()
	}{
		{
			name:      "valid secret",
			envSecret: "L5WTt6D+6R55YfKzwqPRAEX5bR0bkNo4i58jYKL0wsk=",
			wantErr:   false,
			setup: func() {
				resetJWTSecret()
				os.Setenv("JWT_SECRET", "L5WTt6D+6R55YfKzwqPRAEX5bR0bkNo4i58jYKL0wsk=")
			},
			cleanup: func() {
				os.Unsetenv("JWT_SECRET")
				resetJWTSecret()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}
			if tt.cleanup != nil {
				defer tt.cleanup()
			}

			secret, err := getJWTSecret()
			if (err != nil) != tt.wantErr {
				t.Errorf("getJWTSecret() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if len(secret) == 0 {
					t.Errorf("getJWTSecret() returned empty secret")
				}
				if string(secret) != tt.envSecret {
					t.Errorf("getJWTSecret() = %v, want %v", string(secret), tt.envSecret)
				}
			}
		})
	}
}

func TestPasswordHashingRoundTrip(t *testing.T) {
	passwords := []string{
		"SimplePassword",
		"C0mpl3x!P@ssw0rd",
		"very-long-password-with-many-characters-to-test-limits",
		"password with spaces",
		"пароль",        // Cyrillic
		"密码",           // Chinese
		"🔒secure123",   // Emoji
	}

	for _, password := range passwords {
		t.Run(password, func(t *testing.T) {
			// Hash the password
			hash, err := HashPassword(password)
			if err != nil {
				t.Fatalf("HashPassword() failed: %v", err)
			}

			// Verify the password matches
			if err := CheckPassword(hash, password); err != nil {
				t.Errorf("CheckPassword() failed for correct password: %v", err)
			}

			// Verify a wrong password fails
			wrongPassword := password + "wrong"
			if err := CheckPassword(hash, wrongPassword); err == nil {
				t.Errorf("CheckPassword() succeeded for wrong password, should have failed")
			}
		})
	}
}

func TestTokenGenerateAndValidateRoundTrip(t *testing.T) {
	// Set up valid JWT_SECRET for testing (generated with openssl rand -base64 32)
	os.Setenv("JWT_SECRET", "L5WTt6D+6R55YfKzwqPRAEX5bR0bkNo4i58jYKL0wsk=")
	defer os.Unsetenv("JWT_SECRET")
	defer resetJWTSecret()
	resetJWTSecret()

	testCases := []struct {
		userID  uint
		isAdmin bool
	}{
		{userID: 1, isAdmin: false},
		{userID: 100, isAdmin: true},
		{userID: 999, isAdmin: false},
	}

	for _, tc := range testCases {
		t.Run("roundtrip", func(t *testing.T) {
			// Generate token
			token, err := GenerateToken(tc.userID, tc.isAdmin)
			if err != nil {
				t.Fatalf("GenerateToken() failed: %v", err)
			}

			// Validate token
			claims, err := ValidateToken(token)
			if err != nil {
				t.Fatalf("ValidateToken() failed: %v", err)
			}

			// Verify claims
			if claims.UserID != tc.userID {
				t.Errorf("UserID = %v, want %v", claims.UserID, tc.userID)
			}
			if claims.IsAdmin != tc.isAdmin {
				t.Errorf("IsAdmin = %v, want %v", claims.IsAdmin, tc.isAdmin)
			}

			// Verify token expiration is in the future
			if claims.ExpiresAt.Before(time.Now()) {
				t.Errorf("Token already expired")
			}

			// Verify token was issued in the past (or now)
			if claims.IssuedAt.After(time.Now().Add(1 * time.Second)) {
				t.Errorf("Token issued in the future")
			}
		})
	}
}
