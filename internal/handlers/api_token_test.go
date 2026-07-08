package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
)

// setupAPITokenTestContext creates a Gin context with an authenticated user,
// mirroring the pattern used in user_admin_test.go.
func setupAPITokenTestContext(userID uint, isAdmin bool) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("user_id", userID)
	c.Set("is_admin", isAdmin)
	return c, w
}

func TestCreateAPIToken(t *testing.T) {
	tests := []struct {
		name           string
		body           map[string]interface{}
		expectedStatus int
	}{
		{
			name: "valid request creates a token and returns the secret once",
			body: map[string]interface{}{
				"name":       "Zapier integration",
				"expires_at": time.Now().Add(30 * 24 * time.Hour).Format(time.RFC3339),
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "missing name is rejected",
			body: map[string]interface{}{
				"expires_at": time.Now().Add(30 * 24 * time.Hour).Format(time.RFC3339),
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "expires_at in the past is rejected",
			body: map[string]interface{}{
				"name":       "bad token",
				"expires_at": time.Now().Add(-24 * time.Hour).Format(time.RFC3339),
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "expires_at more than 1 year out is rejected",
			body: map[string]interface{}{
				"name":       "too-long token",
				"expires_at": time.Now().Add(400 * 24 * time.Hour).Format(time.RFC3339),
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := SetupTestDB(t)
			admin := CreateTestUser(t, db, "admin", "admin@example.com", "password123", true)

			bodyBytes, _ := json.Marshal(tt.body)
			c, w := setupAPITokenTestContext(admin.ID, true)
			c.Request = httptest.NewRequest("POST", "/api/admin/api-tokens", bytes.NewReader(bodyBytes))
			c.Request.Header.Set("Content-Type", "application/json")

			CreateAPIToken(db)(c)

			if w.Code != tt.expectedStatus {
				t.Fatalf("status = %d, want %d, body = %s", w.Code, tt.expectedStatus, w.Body.String())
			}

			if tt.expectedStatus == http.StatusCreated {
				var resp map[string]interface{}
				if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				token, _ := resp["token"].(string)
				if token == "" {
					t.Error("expected a non-empty one-time token value in the response")
				}

				var stored models.APIToken
				if err := db.First(&stored).Error; err != nil {
					t.Fatalf("expected an APIToken row to be created: %v", err)
				}
				if stored.UserID != admin.ID {
					t.Errorf("stored.UserID = %d, want %d", stored.UserID, admin.ID)
				}
				if stored.TokenHash == "" || stored.TokenHash == token {
					t.Error("stored.TokenHash should be a hash, not the plaintext token")
				}
			}
		})
	}
}

func TestListMyAPITokens(t *testing.T) {
	db := SetupTestDB(t)
	admin := CreateTestUser(t, db, "admin", "admin@example.com", "password123", true)
	otherAdmin := CreateTestUser(t, db, "other-admin", "other@example.com", "password123", true)

	mine := &models.APIToken{UserID: admin.ID, Name: "mine", TokenHash: "hash-1", TokenPrefix: "pat_aaaaaaaa", ExpiresAt: time.Now().Add(24 * time.Hour)}
	if err := db.Create(mine).Error; err != nil {
		t.Fatalf("failed to create token: %v", err)
	}
	notMine := &models.APIToken{UserID: otherAdmin.ID, Name: "not mine", TokenHash: "hash-2", TokenPrefix: "pat_bbbbbbbb", ExpiresAt: time.Now().Add(24 * time.Hour)}
	if err := db.Create(notMine).Error; err != nil {
		t.Fatalf("failed to create token: %v", err)
	}

	c, w := setupAPITokenTestContext(admin.ID, true)
	c.Request = httptest.NewRequest("GET", "/api/admin/api-tokens", nil)

	ListMyAPITokens(db)(c)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body = %s", w.Code, w.Body.String())
	}

	var tokens []map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &tokens); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if len(tokens) != 1 {
		t.Fatalf("len(tokens) = %d, want 1 (scoped to the caller)", len(tokens))
	}
	if tokens[0]["name"] != "mine" {
		t.Errorf("tokens[0].name = %v, want %q", tokens[0]["name"], "mine")
	}
	if _, hasHash := tokens[0]["token_hash"]; hasHash {
		t.Error("response should never include token_hash")
	}
}

func TestRevokeAPIToken(t *testing.T) {
	db := SetupTestDB(t)
	admin := CreateTestUser(t, db, "admin", "admin@example.com", "password123", true)
	otherAdmin := CreateTestUser(t, db, "other-admin", "other@example.com", "password123", true)

	mine := &models.APIToken{UserID: admin.ID, Name: "mine", TokenHash: "hash-1", TokenPrefix: "pat_aaaaaaaa", ExpiresAt: time.Now().Add(24 * time.Hour)}
	if err := db.Create(mine).Error; err != nil {
		t.Fatalf("failed to create token: %v", err)
	}
	notMine := &models.APIToken{UserID: otherAdmin.ID, Name: "not mine", TokenHash: "hash-2", TokenPrefix: "pat_bbbbbbbb", ExpiresAt: time.Now().Add(24 * time.Hour)}
	if err := db.Create(notMine).Error; err != nil {
		t.Fatalf("failed to create token: %v", err)
	}

	t.Run("revoking my own token succeeds", func(t *testing.T) {
		c, w := setupAPITokenTestContext(admin.ID, true)
		c.Params = gin.Params{{Key: "tokenId", Value: fmt.Sprintf("%d", mine.ID)}}
		c.Request = httptest.NewRequest("DELETE", "/api/admin/api-tokens/"+fmt.Sprintf("%d", mine.ID), nil)

		RevokeAPIToken(db)(c)

		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200, body = %s", w.Code, w.Body.String())
		}
		var count int64
		db.Model(&models.APIToken{}).Where("id = ?", mine.ID).Count(&count)
		if count != 0 {
			t.Error("token should be soft-deleted and excluded from default-scoped queries")
		}
	})

	t.Run("revoking another admin's token returns 404", func(t *testing.T) {
		c, w := setupAPITokenTestContext(admin.ID, true)
		c.Params = gin.Params{{Key: "tokenId", Value: fmt.Sprintf("%d", notMine.ID)}}
		c.Request = httptest.NewRequest("DELETE", "/api/admin/api-tokens/"+fmt.Sprintf("%d", notMine.ID), nil)

		RevokeAPIToken(db)(c)

		if w.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404, body = %s", w.Code, w.Body.String())
		}
		var count int64
		db.Model(&models.APIToken{}).Where("id = ?", notMine.ID).Count(&count)
		if count != 1 {
			t.Error("another admin's token should not have been revoked")
		}
	})

	t.Run("revoking a nonexistent token returns 404", func(t *testing.T) {
		c, w := setupAPITokenTestContext(admin.ID, true)
		c.Params = gin.Params{{Key: "tokenId", Value: "999999"}}
		c.Request = httptest.NewRequest("DELETE", "/api/admin/api-tokens/999999", nil)

		RevokeAPIToken(db)(c)

		if w.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404, body = %s", w.Code, w.Body.String())
		}
	})
}
