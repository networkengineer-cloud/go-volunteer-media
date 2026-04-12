package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupGroupDocumentTestDB creates an in-memory SQLite DB with all models needed for group document tests.
func setupGroupDocumentTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	os.Setenv("JWT_SECRET", "aB3dE5fG7hI9jK1lM3nO5pQ7rS9tU1vW3xY5zA7bC9dE1fG3hI5jK7lM9nO1pQ3")

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("Failed to get database instance: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)

	if err := db.AutoMigrate(
		&models.User{},
		&models.Group{},
		&models.UserGroup{},
		&models.GroupDocument{},
	); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	return db
}

// newGroupDocTestContext creates a Gin test context with user auth set.
func newGroupDocTestContext(userID uint, isAdmin bool) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("user_id", userID)
	c.Set("is_admin", isAdmin)
	return c, w
}

// buildDocumentMultipartRequest builds a multipart/form-data POST with a document file and optional form fields.
func buildDocumentMultipartRequest(t *testing.T, fields map[string]string, fileFieldName, filename string, fileContent []byte) *http.Request {
	t.Helper()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Write text fields
	for key, val := range fields {
		if err := writer.WriteField(key, val); err != nil {
			t.Fatalf("Failed to write field %q: %v", key, err)
		}
	}

	// Write file (optional – pass nil content to skip)
	if fileContent != nil {
		part, err := writer.CreateFormFile(fileFieldName, filename)
		if err != nil {
			t.Fatalf("Failed to create form file: %v", err)
		}
		if _, err := part.Write(fileContent); err != nil {
			t.Fatalf("Failed to write file content: %v", err)
		}
	}

	writer.Close()
	req := httptest.NewRequest(http.MethodPost, "/test", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}

// insertGroupDocument inserts a GroupDocument directly into the DB for serving tests.
func insertGroupDocument(t *testing.T, db *gorm.DB, groupID uint, uploaderID uint, blobID, filename string, fileData []byte) *models.GroupDocument {
	t.Helper()
	doc := &models.GroupDocument{
		GroupID:              groupID,
		Title:                "Test Document",
		Description:          "A test document",
		OrderIndex:           0,
		FileURL:              fmt.Sprintf("/api/group-documents/%s", blobID),
		FileName:             filename,
		FileType:             "application/pdf",
		FileSize:             len(fileData),
		FileProvider:         "postgres",
		FileBlobIdentifier:   blobID,
		FileBlobExtension:    ".pdf",
		FileData:             fileData,
		FileUploadedByUserID: &uploaderID,
	}
	if err := db.Create(doc).Error; err != nil {
		t.Fatalf("Failed to insert group document: %v", err)
	}
	return doc
}

// ───────────────────────────────────────────────────────────────────────────────
// GetGroupDocuments
// ───────────────────────────────────────────────────────────────────────────────

func TestGetGroupDocuments(t *testing.T) {
	tests := []struct {
		name           string
		setupUser      func(db *gorm.DB, group *models.Group) (userID uint, isAdmin bool)
		insertDocs     int // number of docs to insert before the request
		expectedStatus int
		expectedCount  *int // nil means don't check count
	}{
		{
			name: "non-member cannot list (403)",
			setupUser: func(db *gorm.DB, group *models.Group) (uint, bool) {
				user := &models.User{Username: "nonmember", Email: "nm@test.com", Password: "x"}
				db.Create(user)
				return user.ID, false
			},
			insertDocs:     1,
			expectedStatus: http.StatusForbidden,
		},
		{
			name: "group member can list (200)",
			setupUser: func(db *gorm.DB, group *models.Group) (uint, bool) {
				user := &models.User{Username: "member", Email: "member@test.com", Password: "x"}
				db.Create(user)
				addUserToGroupForDocTest(db, user.ID, group.ID, false)
				return user.ID, false
			},
			insertDocs:     2,
			expectedStatus: http.StatusOK,
			expectedCount:  intPtr(2),
		},
		{
			name: "site admin can list (200)",
			setupUser: func(db *gorm.DB, group *models.Group) (uint, bool) {
				user := &models.User{Username: "siteadmin", Email: "admin@test.com", Password: "x", IsAdmin: true}
				db.Create(user)
				return user.ID, true
			},
			insertDocs:     1,
			expectedStatus: http.StatusOK,
			expectedCount:  intPtr(1),
		},
		{
			name: "returns empty array when no documents exist (200)",
			setupUser: func(db *gorm.DB, group *models.Group) (uint, bool) {
				user := &models.User{Username: "emptymember", Email: "empty@test.com", Password: "x"}
				db.Create(user)
				addUserToGroupForDocTest(db, user.ID, group.ID, false)
				return user.ID, false
			},
			insertDocs:     0,
			expectedStatus: http.StatusOK,
			expectedCount:  intPtr(0),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupGroupDocumentTestDB(t)

			group := &models.Group{Name: "Test Group " + tt.name, Description: "desc"}
			if err := db.Create(group).Error; err != nil {
				t.Fatalf("Failed to create group: %v", err)
			}

			userID, isAdmin := tt.setupUser(db, group)

			// Insert docs
			uploaderID := userID
			for i := 0; i < tt.insertDocs; i++ {
				insertGroupDocument(t, db, group.ID, uploaderID,
					fmt.Sprintf("blob-%d-%d", group.ID, i),
					fmt.Sprintf("doc%d.pdf", i),
					minimalPDF,
				)
			}

			c, w := newGroupDocTestContext(userID, isAdmin)
			c.Request = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/groups/%d/documents", group.ID), nil)
			c.Params = gin.Params{{Key: "id", Value: fmt.Sprintf("%d", group.ID)}}

			GetGroupDocuments(db)(c)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d; body: %s", tt.expectedStatus, w.Code, w.Body.String())
			}
			if tt.expectedCount != nil && w.Code == http.StatusOK {
				var docs []models.GroupDocument
				if err := json.Unmarshal(w.Body.Bytes(), &docs); err != nil {
					t.Fatalf("Failed to unmarshal response: %v; body: %s", err, w.Body.String())
				}
				if len(docs) != *tt.expectedCount {
					t.Errorf("expected %d docs, got %d", *tt.expectedCount, len(docs))
				}
			}
		})
	}
}

// intPtr is a helper to take the address of an int literal.
func intPtr(i int) *int { return &i }

// AddUserToGroupWithAdmin adapter that doesn't need *testing.T (re-uses the test_helpers version when t is not nil).
// When t is nil we call it from a closure where we don't have a t reference; use a dummy instead.
func addUserToGroupForDocTest(db *gorm.DB, userID, groupID uint, isGroupAdmin bool) {
	ug := &models.UserGroup{UserID: userID, GroupID: groupID, IsGroupAdmin: isGroupAdmin}
	db.Create(ug)
}

// ───────────────────────────────────────────────────────────────────────────────
// UploadGroupDocument
// ───────────────────────────────────────────────────────────────────────────────

func TestUploadGroupDocument(t *testing.T) {
	tests := []struct {
		name           string
		setupUser      func(db *gorm.DB, group *models.Group) (userID uint, isAdmin bool)
		fields         map[string]string
		fileFieldName  string
		filename       string
		fileContent      []byte // nil = don't include a file
		converterOverride *mockConverter // nil = use default &mockConverter{}
		expectedStatus int
	}{
		{
			name: "non-member cannot upload (403)",
			setupUser: func(db *gorm.DB, group *models.Group) (uint, bool) {
				u := &models.User{Username: "upnm", Email: "upnm@test.com", Password: "x"}
				db.Create(u)
				return u.ID, false
			},
			fields:         map[string]string{"title": "My Doc"},
			fileFieldName:  "file",
			filename:       "doc.pdf",
			fileContent:    minimalPDF,
			expectedStatus: http.StatusForbidden,
		},
		{
			name: "group member (non-admin) cannot upload (403)",
			setupUser: func(db *gorm.DB, group *models.Group) (uint, bool) {
				u := &models.User{Username: "upmember", Email: "upmember@test.com", Password: "x"}
				db.Create(u)
				addUserToGroupForDocTest(db, u.ID, group.ID, false)
				return u.ID, false
			},
			fields:         map[string]string{"title": "My Doc"},
			fileFieldName:  "file",
			filename:       "doc.pdf",
			fileContent:    minimalPDF,
			expectedStatus: http.StatusForbidden,
		},
		{
			name: "group admin can upload valid PDF (201)",
			setupUser: func(db *gorm.DB, group *models.Group) (uint, bool) {
				u := &models.User{Username: "upgadmin", Email: "upgadmin@test.com", Password: "x"}
				db.Create(u)
				addUserToGroupForDocTest(db, u.ID, group.ID, true)
				return u.ID, false
			},
			fields:         map[string]string{"title": "Valid Doc", "description": "A description"},
			fileFieldName:  "file",
			filename:       "protocol.pdf",
			fileContent:    minimalPDF,
			expectedStatus: http.StatusCreated,
		},
		{
			name: "missing file returns 400",
			setupUser: func(db *gorm.DB, group *models.Group) (uint, bool) {
				u := &models.User{Username: "upnofile", Email: "upnofile@test.com", Password: "x"}
				db.Create(u)
				addUserToGroupForDocTest(db, u.ID, group.ID, true)
				return u.ID, false
			},
			fields:         map[string]string{"title": "Doc Without File"},
			fileFieldName:  "file",
			filename:       "",
			fileContent:    nil, // no file
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "invalid file type returns 400",
			setupUser: func(db *gorm.DB, group *models.Group) (uint, bool) {
				u := &models.User{Username: "upbadtype", Email: "upbadtype@test.com", Password: "x"}
				db.Create(u)
				addUserToGroupForDocTest(db, u.ID, group.ID, true)
				return u.ID, false
			},
			fields:         map[string]string{"title": "Bad Type Doc"},
			fileFieldName:  "file",
			filename:       "script.exe",
			fileContent:    []byte("MZthis is an exe"),
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "title too short (<2 chars) returns 400",
			setupUser: func(db *gorm.DB, group *models.Group) (uint, bool) {
				u := &models.User{Username: "upshorttitle", Email: "upshorttitle@test.com", Password: "x"}
				db.Create(u)
				addUserToGroupForDocTest(db, u.ID, group.ID, true)
				return u.ID, false
			},
			fields:         map[string]string{"title": "X"}, // 1 char — too short
			fileFieldName:  "file",
			filename:       "doc.pdf",
			fileContent:    minimalPDF,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "DOCX is converted to PDF on upload (201)",
			setupUser: func(db *gorm.DB, group *models.Group) (uint, bool) {
				u := &models.User{Username: "convadmin", Email: "convadmin@test.com", Password: "x"}
				db.Create(u)
				addUserToGroupForDocTest(db, u.ID, group.ID, true)
				return u.ID, false
			},
			fields:        map[string]string{"title": "Converted Doc"},
			fileFieldName: "file",
			filename:      "report.docx",
			// ZIP magic bytes — passes ValidateDocumentUpload for .docx
			fileContent:    append([]byte{0x50, 0x4B, 0x03, 0x04}, make([]byte, 60)...),
			expectedStatus: http.StatusCreated,
		},
		{
			name: "conversion failure returns 422",
			setupUser: func(db *gorm.DB, group *models.Group) (uint, bool) {
				u := &models.User{Username: "convfailadmin", Email: "convfail@test.com", Password: "x"}
				db.Create(u)
				addUserToGroupForDocTest(db, u.ID, group.ID, true)
				return u.ID, false
			},
			fields:        map[string]string{"title": "Bad Doc"},
			fileFieldName: "file",
			filename:      "broken.docx",
			fileContent:   append([]byte{0x50, 0x4B, 0x03, 0x04}, make([]byte, 60)...),
			converterOverride: &mockConverter{
				ConvertErr: fmt.Errorf("libreoffice conversion failed"),
			},
			expectedStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "PDF upload skips conversion even when converter would fail",
			setupUser: func(db *gorm.DB, group *models.Group) (uint, bool) {
				u := &models.User{Username: "pdfskipadmin", Email: "pdfskip@test.com", Password: "x"}
				db.Create(u)
				addUserToGroupForDocTest(db, u.ID, group.ID, true)
				return u.ID, false
			},
			fields:        map[string]string{"title": "PDF Doc"},
			fileFieldName: "file",
			filename:      "direct.pdf",
			fileContent:   minimalPDF,
			// ConvertErr is set — if the handler calls the converter for a PDF the upload
			// would return 422. Getting 201 proves the converter was NOT called.
			converterOverride: &mockConverter{ConvertErr: fmt.Errorf("should not be called")},
			expectedStatus:    http.StatusCreated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupGroupDocumentTestDB(t)

			group := &models.Group{Name: "UploadGroup-" + tt.name, Description: "desc"}
			if err := db.Create(group).Error; err != nil {
				t.Fatalf("Failed to create group: %v", err)
			}

			userID, isAdmin := tt.setupUser(db, group)

			storageMock := &mockStorageProvider{}
			conv := tt.converterOverride
			if conv == nil {
				conv = &mockConverter{}
			}
			req := buildDocumentMultipartRequest(t, tt.fields, tt.fileFieldName, tt.filename, tt.fileContent)

			c, w := newGroupDocTestContext(userID, isAdmin)
			c.Request = req
			c.Params = gin.Params{{Key: "id", Value: fmt.Sprintf("%d", group.ID)}}

			UploadGroupDocument(db, storageMock, conv)(c)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d; body: %s", tt.expectedStatus, w.Code, w.Body.String())
			}

			if tt.expectedStatus == http.StatusCreated {
				var doc models.GroupDocument
				if err := json.Unmarshal(w.Body.Bytes(), &doc); err != nil {
					t.Fatalf("Failed to unmarshal 201 response: %v; body: %s", err, w.Body.String())
				}
				if doc.ID == 0 {
					t.Error("Expected created document to have a non-zero ID")
				}
				if doc.GroupID != group.ID {
					t.Errorf("Expected GroupID %d, got %d", group.ID, doc.GroupID)
				}
			}
		})
	}
}

// ───────────────────────────────────────────────────────────────────────────────
// UploadGroupDocument — postgres fallback
// ───────────────────────────────────────────────────────────────────────────────

func TestUploadGroupDocument_PostgresFallback(t *testing.T) {
	db := setupGroupDocumentTestDB(t)

	group := &models.Group{Name: "FallbackGroup", Description: "desc"}
	if err := db.Create(group).Error; err != nil {
		t.Fatalf("Failed to create group: %v", err)
	}
	admin := &models.User{Username: "fbadmin", Email: "fbadmin@test.com", Password: "x"}
	if err := db.Create(admin).Error; err != nil {
		t.Fatalf("Failed to create admin user: %v", err)
	}
	addUserToGroupForDocTest(db, admin.ID, group.ID, true)

	// Simulate a storage provider failure — handler must fall back to postgres
	storageMock := &mockStorageProvider{UploadDocumentErr: fmt.Errorf("storage unavailable")}

	req := buildDocumentMultipartRequest(t,
		map[string]string{"title": "Fallback Doc"},
		"file", "fallback.pdf", minimalPDF,
	)

	c, w := newGroupDocTestContext(admin.ID, false)
	c.Request = req
	c.Params = gin.Params{{Key: "id", Value: fmt.Sprintf("%d", group.ID)}}

	UploadGroupDocument(db, storageMock, &mockConverter{})(c)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d; body: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}
	docID, ok := resp["id"].(float64)
	if !ok || docID == 0 {
		t.Fatalf("expected non-zero id in response, got %v", resp["id"])
	}
	if fileURL, _ := resp["file_url"].(string); fileURL == "" {
		t.Error("expected file_url to be set on fallback document")
	}

	// FileProvider and FileData are json:"-"; verify via DB fetch
	var saved models.GroupDocument
	if err := db.First(&saved, uint(docID)).Error; err != nil {
		t.Fatalf("Failed to fetch saved document: %v", err)
	}
	if saved.FileProvider != "postgres" {
		t.Errorf("expected file_provider=postgres on fallback, got %q", saved.FileProvider)
	}
	if len(saved.FileData) == 0 {
		t.Error("expected file_data to be stored in postgres on fallback")
	}
}

// ───────────────────────────────────────────────────────────────────────────────
// DeleteGroupDocument
// ───────────────────────────────────────────────────────────────────────────────

func TestDeleteGroupDocument(t *testing.T) {
	tests := []struct {
		name           string
		setupUser      func(db *gorm.DB, group *models.Group) (userID uint, isAdmin bool)
		docGroupSame   bool   // if false, doc belongs to a different group
		docBlobID      string // blob identifier for the inserted doc; empty = don't insert
		paramDocID     string // docId param; "0" = use an ID that won't exist
		expectedStatus int
	}{
		{
			name: "group member cannot delete (403)",
			setupUser: func(db *gorm.DB, group *models.Group) (uint, bool) {
				u := &models.User{Username: "delmember", Email: "delmember@test.com", Password: "x"}
				db.Create(u)
				addUserToGroupForDocTest(db, u.ID, group.ID, false)
				return u.ID, false
			},
			docGroupSame:   true,
			docBlobID:      "blob-del-member",
			expectedStatus: http.StatusForbidden,
		},
		{
			name: "group admin can delete own group's document (200)",
			setupUser: func(db *gorm.DB, group *models.Group) (uint, bool) {
				u := &models.User{Username: "deladmin", Email: "deladmin@test.com", Password: "x"}
				db.Create(u)
				addUserToGroupForDocTest(db, u.ID, group.ID, true)
				return u.ID, false
			},
			docGroupSame:   true,
			docBlobID:      "blob-del-admin",
			expectedStatus: http.StatusOK,
		},
		{
			name: "group admin cannot delete another group's document (404)",
			setupUser: func(db *gorm.DB, group *models.Group) (uint, bool) {
				u := &models.User{Username: "delother", Email: "delother@test.com", Password: "x"}
				db.Create(u)
				addUserToGroupForDocTest(db, u.ID, group.ID, true)
				return u.ID, false
			},
			docGroupSame:   false, // doc in a different group
			docBlobID:      "blob-del-other",
			expectedStatus: http.StatusNotFound,
		},
		{
			name: "deleting non-existent document returns 404",
			setupUser: func(db *gorm.DB, group *models.Group) (uint, bool) {
				u := &models.User{Username: "delnotexist", Email: "delne@test.com", Password: "x"}
				db.Create(u)
				addUserToGroupForDocTest(db, u.ID, group.ID, true)
				return u.ID, false
			},
			docGroupSame:   true,
			docBlobID:      "", // no doc inserted
			paramDocID:     "99999",
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupGroupDocumentTestDB(t)

			group := &models.Group{Name: "DelGroup-" + tt.name, Description: "desc"}
			if err := db.Create(group).Error; err != nil {
				t.Fatalf("Failed to create group: %v", err)
			}

			userID, isAdmin := tt.setupUser(db, group)

			var docID string
			if tt.docBlobID != "" {
				targetGroupID := group.ID
				if !tt.docGroupSame {
					// Create a second group and put the doc there
					otherGroup := &models.Group{Name: "OtherGroup-" + tt.name, Description: "other"}
					if err := db.Create(otherGroup).Error; err != nil {
						t.Fatalf("Failed to create other group: %v", err)
					}
					targetGroupID = otherGroup.ID
				}
				doc := insertGroupDocument(t, db, targetGroupID, userID, tt.docBlobID, "file.pdf", minimalPDF)
				docID = fmt.Sprintf("%d", doc.ID)
			} else {
				docID = tt.paramDocID
			}

			storageMock := &mockStorageProvider{}

			c, w := newGroupDocTestContext(userID, isAdmin)
			c.Request = httptest.NewRequest(http.MethodDelete,
				fmt.Sprintf("/api/v1/groups/%d/documents/%s", group.ID, docID), nil)
			c.Params = gin.Params{
				{Key: "id", Value: fmt.Sprintf("%d", group.ID)},
				{Key: "docId", Value: docID},
			}

			DeleteGroupDocument(db, storageMock)(c)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d; body: %s", tt.expectedStatus, w.Code, w.Body.String())
			}

			if tt.expectedStatus == http.StatusOK {
				var resp map[string]interface{}
				if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
					t.Fatalf("Failed to unmarshal 200 response: %v", err)
				}
				if msg, ok := resp["message"].(string); !ok || msg == "" {
					t.Error("Expected non-empty 'message' field in 200 response")
				}
			}
		})
	}
}

// ───────────────────────────────────────────────────────────────────────────────
// ServeGroupDocument
// ───────────────────────────────────────────────────────────────────────────────

func TestServeGroupDocument(t *testing.T) {
	const blobID = "serve-test-blob-123"
	const fileName = "protocol.pdf"
	fileData := append([]byte(nil), minimalPDF...)

	tests := []struct {
		name              string
		setupAuth         func(c *gin.Context, groupID, userID uint)
		insertDocForGroup bool   // if true, insert doc in the primary group
		lookupBlobID      string // the uuid param passed to the handler
		expectedStatus    int
		checkHeader       bool // verify Content-Disposition is set on 200
		useMemberUser     bool // if true, authenticate as member; if false, authenticate as nonMember
		addMemberToGroup  bool // if true, add member to the group before calling the handler
	}{
		{
			name: "unauthenticated request returns 401",
			setupAuth: func(c *gin.Context, groupID, userID uint) {
				// don't set user_id at all
			},
			insertDocForGroup: true,
			lookupBlobID:      blobID,
			expectedStatus:    http.StatusUnauthorized,
			useMemberUser:     false,
			addMemberToGroup:  false,
		},
		{
			name: "non-member returns 403",
			setupAuth: func(c *gin.Context, groupID, userID uint) {
				c.Set("user_id", userID)
				c.Set("is_admin", false)
			},
			insertDocForGroup: true,
			lookupBlobID:      blobID,
			expectedStatus:    http.StatusForbidden,
			useMemberUser:     false,
			addMemberToGroup:  false,
		},
		{
			name: "group member can serve (200)",
			setupAuth: func(c *gin.Context, groupID, userID uint) {
				c.Set("user_id", userID)
				c.Set("is_admin", false)
			},
			insertDocForGroup: true,
			lookupBlobID:      blobID,
			expectedStatus:    http.StatusOK,
			checkHeader:       true,
			useMemberUser:     true,
			addMemberToGroup:  true,
		},
		{
			name: "non-existent document returns 404",
			setupAuth: func(c *gin.Context, groupID, userID uint) {
				c.Set("user_id", userID)
				c.Set("is_admin", false)
			},
			insertDocForGroup: true,
			lookupBlobID:      "does-not-exist-uuid",
			expectedStatus:    http.StatusNotFound,
			useMemberUser:     false,
			addMemberToGroup:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupGroupDocumentTestDB(t)

			group := &models.Group{Name: "ServeGroup-" + tt.name, Description: "desc"}
			if err := db.Create(group).Error; err != nil {
				t.Fatalf("Failed to create group: %v", err)
			}

			// Create a "member" user and a "non-member" user for each test
			member := &models.User{Username: "servemember-" + tt.name, Email: "sm" + tt.name + "@test.com", Password: "x"}
			if err := db.Create(member).Error; err != nil {
				t.Fatalf("Failed to create member user: %v", err)
			}
			nonMember := &models.User{Username: "servenm-" + tt.name, Email: "snm" + tt.name + "@test.com", Password: "x"}
			if err := db.Create(nonMember).Error; err != nil {
				t.Fatalf("Failed to create non-member user: %v", err)
			}

			if tt.addMemberToGroup {
				addUserToGroupForDocTest(db, member.ID, group.ID, false)
			}

			if tt.insertDocForGroup {
				insertGroupDocument(t, db, group.ID, member.ID, blobID, fileName, fileData)
			}

			storageMock := &mockStorageProvider{}

			var authUserID uint
			if tt.useMemberUser {
				authUserID = member.ID
			} else {
				authUserID = nonMember.ID
			}

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet,
				fmt.Sprintf("/api/group-documents/%s", tt.lookupBlobID), nil)
			c.Params = gin.Params{{Key: "uuid", Value: tt.lookupBlobID}}

			tt.setupAuth(c, group.ID, authUserID)

			ServeGroupDocument(db, storageMock)(c)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d; body: %s", tt.expectedStatus, w.Code, w.Body.String())
			}

			if tt.expectedStatus == http.StatusOK {
				body, err := io.ReadAll(w.Body)
				if err != nil {
					t.Fatalf("Failed to read response body: %v", err)
				}
				if !bytes.Equal(body, fileData) {
					t.Errorf("Response body mismatch: got %q, want %q", body, fileData)
				}

				if tt.checkHeader {
					disp := w.Header().Get("Content-Disposition")
					if disp == "" {
						t.Error("Expected Content-Disposition header to be set")
					}
					if disp != "" && len(disp) > 0 {
						// Should contain the filename somewhere
						if !bytes.Contains([]byte(disp), []byte(fileName)) &&
							!bytes.Contains([]byte(disp), []byte("protocol")) {
							t.Errorf("Content-Disposition %q does not reference filename", disp)
						}
					}
				}
			}
		})
	}
}
