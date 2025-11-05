package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestHealthCheck(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		expectedStatus int
	}{
		{
			name:           "health check returns OK",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/health", nil)

			// Execute
			handler := HealthCheck()
			handler(c)

			// Assert
			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Contains(t, w.Body.String(), "healthy")
			assert.Contains(t, w.Body.String(), "time")
		})
	}
}

func TestReadinessCheck(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		setupDB        func() *gorm.DB
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "readiness check returns ready when database is connected",
			setupDB: func() *gorm.DB {
				db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
				if err != nil {
					t.Fatalf("Failed to open database: %v", err)
				}
				return db
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "ready",
		},
		{
			name: "readiness check returns not ready when database is unavailable",
			setupDB: func() *gorm.DB {
				db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
				if err != nil {
					t.Fatalf("Failed to open database: %v", err)
				}
				// Close the database to simulate unavailability
				sqlDB, _ := db.DB()
				sqlDB.Close()
				return db
			},
			expectedStatus: http.StatusServiceUnavailable,
			expectedBody:   "not ready",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			db := tt.setupDB()
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/ready", nil)

			// Execute
			handler := ReadinessCheck(db)
			handler(c)

			// Assert
			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Contains(t, w.Body.String(), tt.expectedBody)

			// Cleanup
			if tt.expectedStatus == http.StatusOK {
				sqlDB, _ := db.DB()
				sqlDB.Close()
			}
		})
	}
}

func TestReadinessCheck_DBConnectionError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create a DB and close it to simulate connection error
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	sqlDB, _ := db.DB()
	sqlDB.Close()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/ready", nil)

	handler := ReadinessCheck(db)
	handler(c)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Contains(t, w.Body.String(), "not ready")
}
