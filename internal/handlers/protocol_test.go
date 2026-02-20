package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// TestUploadProtocolImage tests the protocol image upload handler, which requires
// either site-admin or group-admin access and uploads to the storage provider.
func TestUploadProtocolImage(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		provider       *mockStorageProvider
		isAdmin        bool
		makeGroupAdmin bool // add the test user as a group admin
		request        func(*testing.T) *http.Request
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "site admin uploads valid PNG successfully",
			provider:       &mockStorageProvider{},
			isAdmin:        true,
			makeGroupAdmin: false,
			request: func(t *testing.T) *http.Request {
				return createImageMultipartRequest(t, "image", "protocol.png", minimalPNG)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "/api/images/test-uuid",
		},
		{
			name:           "group admin uploads valid PNG successfully",
			provider:       &mockStorageProvider{},
			isAdmin:        false,
			makeGroupAdmin: true,
			request: func(t *testing.T) *http.Request {
				return createImageMultipartRequest(t, "image", "protocol.png", minimalPNG)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "/api/images/test-uuid",
		},
		{
			name:           "non-admin non-group-admin returns 403",
			provider:       &mockStorageProvider{},
			isAdmin:        false,
			makeGroupAdmin: false,
			request: func(t *testing.T) *http.Request {
				return createImageMultipartRequest(t, "image", "protocol.png", minimalPNG)
			},
			expectedStatus: http.StatusForbidden,
			expectedBody:   "Admin access required",
		},
		{
			name:           "storage provider error returns 500",
			provider:       &mockStorageProvider{UploadImageErr: errors.New("blob unavailable")},
			isAdmin:        true,
			makeGroupAdmin: false,
			request: func(t *testing.T) *http.Request {
				return createImageMultipartRequest(t, "image", "protocol.png", minimalPNG)
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "Failed to upload image",
		},
		{
			name:           "missing file field returns 400",
			provider:       &mockStorageProvider{},
			isAdmin:        true,
			makeGroupAdmin: false,
			request: func(t *testing.T) *http.Request {
				return httptest.NewRequest(http.MethodPost, "/test", nil)
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "No file uploaded",
		},
		{
			name:           "disallowed file type returns 400",
			provider:       &mockStorageProvider{},
			isAdmin:        true,
			makeGroupAdmin: false,
			request: func(t *testing.T) *http.Request {
				return createImageMultipartRequest(t, "image", "protocol.txt", []byte("not an image"))
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := SetupTestDB(t)
			defer func() {
				sqlDB, _ := db.DB()
				sqlDB.Close()
			}()

			group := CreateTestGroup(t, db, "Test Group", "Description")

			user := CreateTestUser(t, db, "testuser", "user@example.com", "pass1234", false)
			if tt.makeGroupAdmin {
				AddUserToGroupWithAdmin(t, db, user.ID, group.ID, true)
			}

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = tt.request(t)
			c.Set("user_id", user.ID)
			c.Set("is_admin", tt.isAdmin)
			c.Params = gin.Params{{Key: "id", Value: fmt.Sprintf("%d", group.ID)}}

			handler := UploadProtocolImage(db, tt.provider)
			handler(c)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectedStatus, w.Code, w.Body.String())
			}
			if !strings.Contains(w.Body.String(), tt.expectedBody) {
				t.Errorf("Expected body to contain %q, got %s", tt.expectedBody, w.Body.String())
			}
		})
	}
}
