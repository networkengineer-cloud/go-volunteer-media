package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/auth"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/logging"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/middleware"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
	"gorm.io/gorm"
)

// maxAPITokenLifetime caps how far out an admin can set a token's expiry,
// so "required expiration" can't be defeated with an effectively-infinite date.
const maxAPITokenLifetime = 365 * 24 * time.Hour

// maxAPITokensPerUser caps how many live (non-revoked) tokens a single admin
// can hold at once, so an admin can't accumulate an unbounded number of tokens.
const maxAPITokensPerUser = 20

// apiTokenResponse is what ListMyAPITokens/CreateAPIToken return — it never
// includes the token hash or, after creation, the plaintext secret.
type apiTokenResponse struct {
	ID          uint       `json:"id"`
	Name        string     `json:"name"`
	TokenPrefix string     `json:"token_prefix"`
	CreatedAt   time.Time  `json:"created_at"`
	ExpiresAt   time.Time  `json:"expires_at"`
	LastUsedAt  *time.Time `json:"last_used_at"`
}

func toAPITokenResponse(t models.APIToken) apiTokenResponse {
	return apiTokenResponse{
		ID:          t.ID,
		Name:        t.Name,
		TokenPrefix: t.TokenPrefix,
		CreatedAt:   t.CreatedAt,
		ExpiresAt:   t.ExpiresAt,
		LastUsedAt:  t.LastUsedAt,
	}
}

// ListMyAPITokens returns the calling admin's own API tokens.
func ListMyAPITokens(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		db := middleware.GetDB(c, db)
		userID, _ := middleware.GetUserID(c)

		var tokens []models.APIToken
		if err := db.WithContext(ctx).
			Where("user_id = ?", userID).
			Order("created_at desc").
			Find(&tokens).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch API tokens"})
			return
		}

		resp := make([]apiTokenResponse, len(tokens))
		for i, t := range tokens {
			resp[i] = toAPITokenResponse(t)
		}
		c.JSON(http.StatusOK, resp)
	}
}

// createAPITokenRequest is the CreateAPIToken request body.
type createAPITokenRequest struct {
	Name      string    `json:"name" binding:"required,max=100"`
	ExpiresAt time.Time `json:"expires_at" binding:"required"`
}

// CreateAPIToken generates a new API token for the calling admin. The full
// token value is returned exactly once, in this response, under "token".
func CreateAPIToken(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		db := middleware.GetDB(c, db)
		userID, _ := middleware.GetUserID(c)

		var req createAPITokenRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": formatValidationError(err)})
			return
		}

		now := time.Now()
		if !req.ExpiresAt.After(now) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "expires_at must be in the future"})
			return
		}
		if req.ExpiresAt.After(now.Add(maxAPITokenLifetime)) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "expires_at cannot be more than 1 year out"})
			return
		}

		var tokenCount int64
		if err := db.WithContext(ctx).Model(&models.APIToken{}).
			Where("user_id = ?", userID).
			Count(&tokenCount).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create API token"})
			return
		}
		if tokenCount >= maxAPITokensPerUser {
			c.JSON(http.StatusBadRequest, gin.H{"error": "You have reached the maximum number of API tokens (20). Revoke an existing token before creating a new one."})
			return
		}

		generated, err := auth.GenerateAPIToken()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate API token"})
			return
		}

		apiToken := models.APIToken{
			UserID:      userID,
			Name:        req.Name,
			TokenHash:   generated.Hash,
			TokenPrefix: generated.DisplayPrefix,
			ExpiresAt:   req.ExpiresAt,
		}
		if err := db.WithContext(ctx).Create(&apiToken).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create API token"})
			return
		}

		logging.LogAdminAction(ctx, logging.AuditEventAPITokenCreated, userID, map[string]interface{}{
			"token_id":   apiToken.ID,
			"token_name": apiToken.Name,
		})

		resp := struct {
			apiTokenResponse
			Token string `json:"token"`
		}{
			apiTokenResponse: toAPITokenResponse(apiToken),
			Token:            generated.Token,
		}
		c.JSON(http.StatusCreated, resp)
	}
}

// RevokeAPIToken soft-deletes one of the calling admin's own API tokens.
// A token that doesn't exist or belongs to someone else returns 404 rather
// than 403, so a caller can't use this endpoint to confirm another admin's
// token ID exists.
func RevokeAPIToken(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		db := middleware.GetDB(c, db)
		userID, _ := middleware.GetUserID(c)
		tokenID, err := strconv.ParseUint(c.Param("tokenId"), 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid token ID"})
			return
		}

		var apiToken models.APIToken
		if err := db.WithContext(ctx).
			Where("id = ? AND user_id = ?", tokenID, userID).
			First(&apiToken).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "API token not found"})
			return
		}

		if err := db.WithContext(ctx).Delete(&apiToken).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to revoke API token"})
			return
		}

		logging.LogAdminAction(ctx, logging.AuditEventAPITokenRevoked, userID, map[string]interface{}{
			"token_id":   apiToken.ID,
			"token_name": apiToken.Name,
		})

		c.JSON(http.StatusOK, gin.H{"message": "API token revoked"})
	}
}
