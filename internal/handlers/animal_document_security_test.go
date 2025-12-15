package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
)

// TestServeAnimalProtocolDocument_Security tests security aspects of document serving
func TestServeAnimalProtocolDocument_Security(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	// Auto-migrate
	err = db.AutoMigrate(&models.User{}, &models.Group{}, &models.UserGroup{}, &models.Animal{})
	assert.NoError(t, err)

	// Create test data
	group1 := models.Group{Name: "Dogs", Description: "Dog volunteers"}
	assert.NoError(t, db.Create(&group1).Error)

	group2 := models.Group{Name: "Cats", Description: "Cat volunteers"}
	assert.NoError(t, db.Create(&group2).Error)

	// Create users
	user1 := models.User{
		Username: "dogvolunteer",
		Email:    "dog@example.com",
		Password: "hashedpassword",
		IsAdmin:  false,
	}
	assert.NoError(t, db.Create(&user1).Error)
	assert.NoError(t, db.Model(&user1).Association("Groups").Append(&group1))

	user2 := models.User{
		Username: "catvolunteer",
		Email:    "cat@example.com",
		Password: "hashedpassword",
		IsAdmin:  false,
	}
	assert.NoError(t, db.Create(&user2).Error)
	assert.NoError(t, db.Model(&user2).Association("Groups").Append(&group2))

	adminUser := models.User{
		Username: "admin",
		Email:    "admin@example.com",
		Password: "hashedpassword",
		IsAdmin:  true,
	}
	assert.NoError(t, db.Create(&adminUser).Error)

	// Create animal with protocol document in group1
	documentData := []byte("PDF document content for sensitive medical info")
	documentUUID := "test-uuid-123"
	documentURL := fmt.Sprintf("/api/documents/%s", documentUUID)

	animal := models.Animal{
		Name:                   "Rex",
		Species:                "Dog",
		GroupID:                group1.ID,
		Status:                 "available",
		ProtocolDocumentURL:    documentURL,
		ProtocolDocumentName:   "rex-protocol.pdf",
		ProtocolDocumentData:   documentData,
		ProtocolDocumentType:   "application/pdf",
		ProtocolDocumentSize:   len(documentData),
		ProtocolDocumentUserID: &user1.ID,
	}
	assert.NoError(t, db.Create(&animal).Error)

	tests := []struct {
		name           string
		setupAuth      func(*gin.Context)
		documentUUID   string
		expectedStatus int
		expectedError  string
	}{
		{
			name: "unauthenticated request denied",
			setupAuth: func(c *gin.Context) {
				// No auth context
			},
			documentUUID:   documentUUID,
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "Unauthorized",
		},
		{
			name: "authorized group member can access",
			setupAuth: func(c *gin.Context) {
				c.Set("user_id", user1.ID)
				c.Set("is_admin", false)
			},
			documentUUID:   documentUUID,
			expectedStatus: http.StatusOK,
		},
		{
			name: "unauthorized group member denied",
			setupAuth: func(c *gin.Context) {
				c.Set("user_id", user2.ID) // Cat volunteer trying to access dog document
				c.Set("is_admin", false)
			},
			documentUUID:   documentUUID,
			expectedStatus: http.StatusForbidden,
			expectedError:  "Access denied",
		},
		{
			name: "admin can access any document",
			setupAuth: func(c *gin.Context) {
				c.Set("user_id", adminUser.ID)
				c.Set("is_admin", true)
			},
			documentUUID:   documentUUID,
			expectedStatus: http.StatusOK,
		},
		{
			name: "non-existent document returns 404",
			setupAuth: func(c *gin.Context) {
				c.Set("user_id", user1.ID)
				c.Set("is_admin", false)
			},
			documentUUID:   "non-existent-uuid",
			expectedStatus: http.StatusNotFound,
			expectedError:  "Document not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test context
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			// Setup request
			c.Request = httptest.NewRequest("GET", fmt.Sprintf("/api/documents/%s", tt.documentUUID), nil)
			c.Params = gin.Params{
				{Key: "uuid", Value: tt.documentUUID},
			}

			// Setup authentication
			tt.setupAuth(c)

			// Call handler
			handler := ServeAnimalProtocolDocument(db)
			handler(c)

			// Assert status code
			assert.Equal(t, tt.expectedStatus, w.Code, "Status code mismatch")

			// Assert response
			if tt.expectedStatus == http.StatusOK {
				// Verify document content is served
				body, err := io.ReadAll(w.Body)
				assert.NoError(t, err)
				assert.Equal(t, documentData, body, "Document content mismatch")

				// Verify headers
				assert.Equal(t, "application/pdf", w.Header().Get("Content-Type"))
				assert.Contains(t, w.Header().Get("Content-Disposition"), "rex-protocol.pdf")
			} else if tt.expectedError != "" {
				// Verify error message
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Contains(t, response["error"], tt.expectedError)
			}
		})
	}
}

// TestServeAnimalProtocolDocument_UUIDEnumeration tests protection against UUID enumeration
func TestServeAnimalProtocolDocument_UUIDEnumeration(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	err = db.AutoMigrate(&models.User{}, &models.Group{}, &models.UserGroup{}, &models.Animal{})
	assert.NoError(t, err)

	// Create test group and user
	group := models.Group{Name: "Dogs", Description: "Dog volunteers"}
	assert.NoError(t, db.Create(&group).Error)

	user := models.User{
		Username: "volunteer",
		Email:    "volunteer@example.com",
		Password: "hashedpassword",
		IsAdmin:  false,
	}
	assert.NoError(t, db.Create(&user).Error)
	assert.NoError(t, db.Model(&user).Association("Groups").Append(&group))

	// Create multiple animals with documents
	for i := 1; i <= 10; i++ {
		documentUUID := fmt.Sprintf("uuid-%d", i)
		documentURL := fmt.Sprintf("/api/documents/%s", documentUUID)

		animal := models.Animal{
			Name:                   fmt.Sprintf("Animal%d", i),
			Species:                "Dog",
			GroupID:                group.ID,
			Status:                 "available",
			ProtocolDocumentURL:    documentURL,
			ProtocolDocumentName:   fmt.Sprintf("protocol-%d.pdf", i),
			ProtocolDocumentData:   []byte(fmt.Sprintf("Document %d content", i)),
			ProtocolDocumentType:   "application/pdf",
			ProtocolDocumentSize:   20,
			ProtocolDocumentUserID: &user.ID,
		}
		assert.NoError(t, db.Create(&animal).Error)
	}

	// Test that unauthorized user cannot enumerate UUIDs
	unauthorizedUser := models.User{
		Username: "unauthorized",
		Email:    "unauth@example.com",
		Password: "hashedpassword",
		IsAdmin:  false,
	}
	assert.NoError(t, db.Create(&unauthorizedUser).Error)

	// Try to access documents with sequential UUIDs
	for i := 1; i <= 10; i++ {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		documentUUID := fmt.Sprintf("uuid-%d", i)
		c.Request = httptest.NewRequest("GET", fmt.Sprintf("/api/documents/%s", documentUUID), nil)
		c.Params = gin.Params{{Key: "uuid", Value: documentUUID}}
		c.Set("user_id", unauthorizedUser.ID)
		c.Set("is_admin", false)

		handler := ServeAnimalProtocolDocument(db)
		handler(c)

		// All requests should be forbidden (403), not 404
		// This prevents information disclosure about document existence
		assert.Equal(t, http.StatusForbidden, w.Code,
			"Unauthorized user should get 403 for existing documents to prevent enumeration")
	}
}

// TestServeAnimalProtocolDocument_GroupMembershipValidation tests group membership logic
func TestServeAnimalProtocolDocument_GroupMembershipValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	err = db.AutoMigrate(&models.User{}, &models.Group{}, &models.UserGroup{}, &models.Animal{})
	assert.NoError(t, err)

	// Create multiple groups
	dogGroup := models.Group{Name: "Dogs", Description: "Dog volunteers"}
	assert.NoError(t, db.Create(&dogGroup).Error)

	catGroup := models.Group{Name: "Cats", Description: "Cat volunteers"}
	assert.NoError(t, db.Create(&catGroup).Error)

	// Create user who is member of BOTH groups
	multiGroupUser := models.User{
		Username: "multigroup",
		Email:    "multi@example.com",
		Password: "hashedpassword",
		IsAdmin:  false,
	}
	assert.NoError(t, db.Create(&multiGroupUser).Error)
	assert.NoError(t, db.Model(&multiGroupUser).Association("Groups").Append(&dogGroup, &catGroup))

	// Create documents in both groups
	dogDocUUID := "dog-doc-uuid"
	dogAnimal := models.Animal{
		Name:                 "Rex",
		Species:              "Dog",
		GroupID:              dogGroup.ID,
		Status:               "available",
		ProtocolDocumentURL:  fmt.Sprintf("/api/documents/%s", dogDocUUID),
		ProtocolDocumentName: "dog-protocol.pdf",
		ProtocolDocumentData: []byte("Dog document"),
		ProtocolDocumentType: "application/pdf",
	}
	assert.NoError(t, db.Create(&dogAnimal).Error)

	catDocUUID := "cat-doc-uuid"
	catAnimal := models.Animal{
		Name:                 "Whiskers",
		Species:              "Cat",
		GroupID:              catGroup.ID,
		Status:               "available",
		ProtocolDocumentURL:  fmt.Sprintf("/api/documents/%s", catDocUUID),
		ProtocolDocumentName: "cat-protocol.pdf",
		ProtocolDocumentData: []byte("Cat document"),
		ProtocolDocumentType: "application/pdf",
	}
	assert.NoError(t, db.Create(&catAnimal).Error)

	// User should be able to access both documents
	testCases := []struct {
		name string
		uuid string
	}{
		{"dog document", dogDocUUID},
		{"cat document", catDocUUID},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			c.Request = httptest.NewRequest("GET", fmt.Sprintf("/api/documents/%s", tc.uuid), nil)
			c.Params = gin.Params{{Key: "uuid", Value: tc.uuid}}
			c.Set("user_id", multiGroupUser.ID)
			c.Set("is_admin", false)

			handler := ServeAnimalProtocolDocument(db)
			handler(c)

			assert.Equal(t, http.StatusOK, w.Code,
				"Multi-group user should access documents from all their groups")
		})
	}
}

// TestServeAnimalProtocolDocument_EmptyDocumentData tests handling of missing document data
func TestServeAnimalProtocolDocument_EmptyDocumentData(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	err = db.AutoMigrate(&models.User{}, &models.Group{}, &models.UserGroup{}, &models.Animal{})
	assert.NoError(t, err)

	group := models.Group{Name: "Dogs", Description: "Dog volunteers"}
	assert.NoError(t, db.Create(&group).Error)

	user := models.User{
		Username: "volunteer",
		Email:    "volunteer@example.com",
		Password: "hashedpassword",
	}
	assert.NoError(t, db.Create(&user).Error)
	assert.NoError(t, db.Model(&user).Association("Groups").Append(&group))

	// Create animal with document URL but no data
	documentUUID := "empty-doc-uuid"
	documentURL := fmt.Sprintf("/api/documents/%s", documentUUID)

	animal := models.Animal{
		Name:                 "Rex",
		Species:              "Dog",
		GroupID:              group.ID,
		Status:               "available",
		ProtocolDocumentURL:  documentURL,
		ProtocolDocumentName: "protocol.pdf",
		ProtocolDocumentData: []byte{}, // Empty data
		ProtocolDocumentType: "application/pdf",
	}
	assert.NoError(t, db.Create(&animal).Error)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Request = httptest.NewRequest("GET", fmt.Sprintf("/api/documents/%s", documentUUID), nil)
	c.Params = gin.Params{{Key: "uuid", Value: documentUUID}}
	c.Set("user_id", user.ID)
	c.Set("is_admin", false)

	handler := ServeAnimalProtocolDocument(db)
	handler(c)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["error"], "Document data not available")
}
