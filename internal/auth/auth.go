package auth

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/logging"
	"golang.org/x/crypto/bcrypt"
)

var (
	jwtSecret     []byte
	jwtSecretOnce sync.Once
)

// checkSecretEntropy performs basic entropy checks on the JWT secret
func checkSecretEntropy(secret string) error {
	// Check for common weak patterns
	if strings.Repeat("1", 32) == secret[:32] {
		return errors.New("JWT_SECRET appears to be all the same character - insufficient entropy")
	}
	
	// Check if it's sequential or repeated characters
	hasVariety := false
	charMap := make(map[rune]int)
	for _, char := range secret {
		charMap[char]++
		if len(charMap) > 10 {
			hasVariety = true
			break
		}
	}
	
	if !hasVariety && len(secret) >= 32 {
		return errors.New("JWT_SECRET has insufficient character variety - use a cryptographically random secret")
	}
	
	// Warn if it looks like a default value
	lowerSecret := strings.ToLower(secret)
	if strings.Contains(lowerSecret, "change") || 
	   strings.Contains(lowerSecret, "example") || 
	   strings.Contains(lowerSecret, "test") ||
	   strings.Contains(lowerSecret, "default") {
		return errors.New("JWT_SECRET appears to be a default/example value - use a secure random secret")
	}
	
	return nil
}

// initJWTSecret initializes the JWT secret from environment variable
func initJWTSecret() {
	jwtSecretOnce.Do(func() {
		secret := os.Getenv("JWT_SECRET")
		if secret == "" {
			logging.Fatal("JWT_SECRET environment variable is required", nil)
		}
		if len(secret) < 32 {
			logging.Fatal("JWT_SECRET must be at least 32 characters long for security", nil)
		}
		
		// Check secret entropy and quality
		if err := checkSecretEntropy(secret); err != nil {
			logging.Fatal(fmt.Sprintf("JWT_SECRET validation failed: %s. Generate a secure secret with: openssl rand -base64 32", err.Error()), nil)
		}
		
		jwtSecret = []byte(secret)
		logging.Info("JWT secret validated successfully")
	})
}

// getJWTSecret returns the JWT secret, initializing it if necessary
func getJWTSecret() ([]byte, error) {
	initJWTSecret()
	if len(jwtSecret) == 0 {
		return nil, fmt.Errorf("JWT secret not initialized")
	}
	return jwtSecret, nil
}

// Claims represents JWT claims
type Claims struct {
	UserID  uint `json:"user_id"`
	IsAdmin bool `json:"is_admin"`
	jwt.RegisteredClaims
}

// HashPassword hashes a password using bcrypt
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// CheckPassword compares a hashed password with a plain text password
func CheckPassword(hashedPassword, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}

// GenerateToken generates a JWT token for a user
func GenerateToken(userID uint, isAdmin bool) (string, error) {
	secret, err := getJWTSecret()
	if err != nil {
		return "", err
	}

	claims := Claims{
		UserID:  userID,
		IsAdmin: isAdmin,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}

// ValidateToken validates a JWT token and returns the claims
func ValidateToken(tokenString string) (*Claims, error) {
	secret, err := getJWTSecret()
	if err != nil {
		return nil, err
	}

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return secret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}
