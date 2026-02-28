package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/email"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/groupme"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupUpdateTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	// Migrate models
	err = db.AutoMigrate(
		&models.User{},
		&models.Group{},
		&models.Update{},
	)
	if err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	// Create test data
	user := models.User{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "hashedpassword",
		IsAdmin:  false,
	}
	db.Create(&user)

	group := models.Group{
		Name:        "Test Group",
		Description: "Test group description",
	}
	db.Create(&group)

	// Add user to group
	db.Model(&user).Association("Groups").Append(&group)

	update := models.Update{
		GroupID:  group.ID,
		UserID:   user.ID,
		Title:    "Test Update",
		Content:  "Test update content",
		ImageURL: "http://example.com/image.jpg",
	}
	db.Create(&update)

	return db
}

func TestGetUpdates(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		setupContext   func(*gin.Context)
		expectedStatus int
		expectedError  string
	}{
		{
			name: "successful retrieval of updates",
			setupContext: func(c *gin.Context) {
				c.Set("user_id", uint(1))
				c.Set("is_admin", false)
				c.Params = gin.Params{{Key: "id", Value: "1"}}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "forbidden when no group access",
			setupContext: func(c *gin.Context) {
				c.Set("user_id", uint(999))
				c.Set("is_admin", false)
				c.Params = gin.Params{{Key: "id", Value: "1"}}
			},
			expectedStatus: http.StatusForbidden,
			expectedError:  "Access denied",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			db := setupUpdateTestDB(t)
			defer func() {
				sqlDB, _ := db.DB()
				sqlDB.Close()
			}()

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/groups/1/updates", nil)
			tt.setupContext(c)

			// Execute
			handler := GetUpdates(db)
			handler(c)

			// Assert
			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedError != "" {
				assert.Contains(t, w.Body.String(), tt.expectedError)
			}
		})
	}
}

func TestCreateUpdate(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		setupContext   func(*gin.Context)
		requestBody    interface{}
		expectedStatus int
		expectedError  string
	}{
		{
			name: "successful update creation",
			setupContext: func(c *gin.Context) {
				c.Set("user_id", uint(1))
				c.Set("is_admin", false)
				c.Params = gin.Params{{Key: "id", Value: "1"}}
			},
			requestBody: UpdateRequest{
				Title:    "New Update",
				Content:  "New update content",
				ImageURL: "http://example.com/new.jpg",
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "forbidden when no group access",
			setupContext: func(c *gin.Context) {
				c.Set("user_id", uint(999))
				c.Set("is_admin", false)
				c.Params = gin.Params{{Key: "id", Value: "1"}}
			},
			requestBody: UpdateRequest{
				Title:   "New Update",
				Content: "New update content",
			},
			expectedStatus: http.StatusForbidden,
			expectedError:  "Access denied",
		},
		{
			name: "bad request when title is missing",
			setupContext: func(c *gin.Context) {
				c.Set("user_id", uint(1))
				c.Set("is_admin", false)
				c.Params = gin.Params{{Key: "id", Value: "1"}}
			},
			requestBody: UpdateRequest{
				Content: "Content without title",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "bad request when content is missing",
			setupContext: func(c *gin.Context) {
				c.Set("user_id", uint(1))
				c.Set("is_admin", false)
				c.Params = gin.Params{{Key: "id", Value: "1"}}
			},
			requestBody: UpdateRequest{
				Title: "Title without content",
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			db := setupUpdateTestDB(t)
			defer func() {
				sqlDB, _ := db.DB()
				sqlDB.Close()
			}()

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			bodyBytes, _ := json.Marshal(tt.requestBody)
			c.Request = httptest.NewRequest("POST", "/groups/1/updates", bytes.NewBuffer(bodyBytes))
			c.Request.Header.Set("Content-Type", "application/json")
			tt.setupContext(c)

			// Execute
			handler := CreateUpdate(db, email.NewService(db), groupme.NewService())
			handler(c)

			// Assert
			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedError != "" {
				assert.Contains(t, w.Body.String(), tt.expectedError)
			}
		})
	}
}

func TestCreateUpdateWithSendEmailFlagForNonAdminIsForcedFalse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := setupUpdateTestDB(t)
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	reqBody := UpdateRequest{
		Title:       "Email Enabled Update",
		Content:     "This update should persist send_email flag",
		SendEmail:   true,
		SendGroupMe: false,
	}
	bodyBytes, _ := json.Marshal(reqBody)
	c.Request = httptest.NewRequest("POST", "/groups/1/updates", bytes.NewBuffer(bodyBytes))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("user_id", uint(1))
	c.Set("is_admin", false)
	c.Params = gin.Params{{Key: "id", Value: "1"}}

	handler := CreateUpdate(db, email.NewService(db), groupme.NewService())
	handler(c)

	assert.Equal(t, http.StatusCreated, w.Code)

	var created models.Update
	err := json.Unmarshal(w.Body.Bytes(), &created)
	require.NoError(t, err)
	assert.Equal(t, "Email Enabled Update", created.Title)
	assert.False(t, created.SendEmail)
}

func TestCreateUpdateWithSendEmailFlagForSiteAdminIsAllowed(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := setupUpdateTestDB(t)
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	reqBody := UpdateRequest{
		Title:       "Admin Email Enabled Update",
		Content:     "This update should keep send_email true for admins",
		SendEmail:   true,
		SendGroupMe: false,
	}
	bodyBytes, _ := json.Marshal(reqBody)
	c.Request = httptest.NewRequest("POST", "/groups/1/updates", bytes.NewBuffer(bodyBytes))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("user_id", uint(1))
	c.Set("is_admin", true)
	c.Params = gin.Params{{Key: "id", Value: "1"}}

	handler := CreateUpdate(db, email.NewService(db), groupme.NewService())
	handler(c)

	assert.Equal(t, http.StatusCreated, w.Code)

	var created models.Update
	err := json.Unmarshal(w.Body.Bytes(), &created)
	require.NoError(t, err)
	assert.Equal(t, "Admin Email Enabled Update", created.Title)
	assert.True(t, created.SendEmail)
}

func setupDeleteUpdateTestDB(t *testing.T) (*gorm.DB, models.User, models.User, models.User, models.Group, models.Update) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(
		&models.User{},
		&models.Group{},
		&models.Update{},
		&models.UserGroup{},
	)
	require.NoError(t, err)

	// Site admin user
	siteAdmin := models.User{
		Username: "siteadmin",
		Email:    "admin@example.com",
		Password: "hashedpassword",
		IsAdmin:  true,
	}
	require.NoError(t, db.Create(&siteAdmin).Error)

	// Group admin user
	groupAdmin := models.User{
		Username: "groupadmin",
		Email:    "gadmin@example.com",
		Password: "hashedpassword",
		IsAdmin:  false,
	}
	require.NoError(t, db.Create(&groupAdmin).Error)

	// Regular member
	member := models.User{
		Username: "member",
		Email:    "member@example.com",
		Password: "hashedpassword",
		IsAdmin:  false,
	}
	require.NoError(t, db.Create(&member).Error)

	group := models.Group{
		Name:        "Test Group",
		Description: "Test group description",
	}
	require.NoError(t, db.Create(&group).Error)

	// Add group admin to group as admin
	require.NoError(t, db.Create(&models.UserGroup{UserID: groupAdmin.ID, GroupID: group.ID, IsGroupAdmin: true}).Error)
	// Add regular member to group
	require.NoError(t, db.Create(&models.UserGroup{UserID: member.ID, GroupID: group.ID, IsGroupAdmin: false}).Error)

	update := models.Update{
		GroupID: group.ID,
		UserID:  groupAdmin.ID,
		Title:   "Test Announcement",
		Content: "Test announcement content",
	}
	require.NoError(t, db.Create(&update).Error)

	return db, siteAdmin, groupAdmin, member, group, update
}

func TestDeleteUpdate(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		setupContext   func(*gin.Context, models.User, models.User, models.User, models.Group, models.Update)
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "successful delete as group admin",
			setupContext: func(c *gin.Context, _ models.User, groupAdmin models.User, _ models.User, group models.Group, update models.Update) {
				c.Set("user_id", groupAdmin.ID)
				c.Set("is_admin", false)
				c.Params = gin.Params{
					{Key: "id", Value: strconv.FormatUint(uint64(group.ID), 10)},
					{Key: "updateId", Value: strconv.FormatUint(uint64(update.ID), 10)},
				}
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "Group announcement deleted successfully",
		},
		{
			name: "successful delete as site admin",
			setupContext: func(c *gin.Context, siteAdmin models.User, _ models.User, _ models.User, group models.Group, update models.Update) {
				c.Set("user_id", siteAdmin.ID)
				c.Set("is_admin", true)
				c.Params = gin.Params{
					{Key: "id", Value: strconv.FormatUint(uint64(group.ID), 10)},
					{Key: "updateId", Value: strconv.FormatUint(uint64(update.ID), 10)},
				}
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "Group announcement deleted successfully",
		},
		{
			name: "forbidden for regular member",
			setupContext: func(c *gin.Context, _ models.User, _ models.User, member models.User, group models.Group, update models.Update) {
				c.Set("user_id", member.ID)
				c.Set("is_admin", false)
				c.Params = gin.Params{
					{Key: "id", Value: strconv.FormatUint(uint64(group.ID), 10)},
					{Key: "updateId", Value: strconv.FormatUint(uint64(update.ID), 10)},
				}
			},
			expectedStatus: http.StatusForbidden,
			expectedBody:   "Only group admins can delete group announcements",
		},
		{
			name: "bad request for invalid updateId",
			setupContext: func(c *gin.Context, _ models.User, groupAdmin models.User, _ models.User, group models.Group, _ models.Update) {
				c.Set("user_id", groupAdmin.ID)
				c.Set("is_admin", false)
				c.Params = gin.Params{
					{Key: "id", Value: strconv.FormatUint(uint64(group.ID), 10)},
					{Key: "updateId", Value: "not-a-number"},
				}
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid update ID",
		},
		{
			name: "not found when update belongs to different group",
			setupContext: func(c *gin.Context, siteAdmin models.User, _ models.User, _ models.User, _ models.Group, update models.Update) {
				c.Set("user_id", siteAdmin.ID)
				c.Set("is_admin", true)
				c.Params = gin.Params{
					{Key: "id", Value: "9999"},
					{Key: "updateId", Value: strconv.FormatUint(uint64(update.ID), 10)},
				}
			},
			expectedStatus: http.StatusNotFound,
			expectedBody:   "Update not found",
		},
		{
			name: "not found when update does not exist",
			setupContext: func(c *gin.Context, siteAdmin models.User, _ models.User, _ models.User, group models.Group, _ models.Update) {
				c.Set("user_id", siteAdmin.ID)
				c.Set("is_admin", true)
				c.Params = gin.Params{
					{Key: "id", Value: strconv.FormatUint(uint64(group.ID), 10)},
					{Key: "updateId", Value: "9999"},
				}
			},
			expectedStatus: http.StatusNotFound,
			expectedBody:   "Update not found",
		},
		{
			name: "bad request for invalid groupId as site admin",
			setupContext: func(c *gin.Context, siteAdmin models.User, _ models.User, _ models.User, _ models.Group, _ models.Update) {
				c.Set("user_id", siteAdmin.ID)
				c.Set("is_admin", true)
				c.Params = gin.Params{
					{Key: "id", Value: "not-a-number"},
					{Key: "updateId", Value: "1"},
				}
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid group ID",
		},
		{
			name: "bad request for invalid groupId as group admin",
			setupContext: func(c *gin.Context, _ models.User, groupAdmin models.User, _ models.User, _ models.Group, _ models.Update) {
				c.Set("user_id", groupAdmin.ID)
				c.Set("is_admin", false)
				c.Params = gin.Params{
					{Key: "id", Value: "not-a-number"},
					{Key: "updateId", Value: "1"},
				}
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid group ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, siteAdmin, groupAdmin, member, group, update := setupDeleteUpdateTestDB(t)
			defer func() {
				sqlDB, _ := db.DB()
				sqlDB.Close()
			}()

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("DELETE", "/groups/1/updates/1", nil)
			tt.setupContext(c, siteAdmin, groupAdmin, member, group, update)

			handler := DeleteUpdate(db)
			handler(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedBody != "" {
				assert.Contains(t, w.Body.String(), tt.expectedBody)
			}
		})
	}
}
