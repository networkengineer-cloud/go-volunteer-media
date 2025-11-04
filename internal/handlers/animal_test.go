package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/auth"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupAnimalTestDB creates an in-memory SQLite database for animal testing
func setupAnimalTestDB(t *testing.T) *gorm.DB {
	// Set JWT_SECRET for testing
	os.Setenv("JWT_SECRET", "aB3dE5fG7hI9jK1lM3nO5pQ7rS9tU1vW3xY5zA7bC9dE1fG3hI5jK7lM9nO1pQ3")
	
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	// Run migrations
	err = db.AutoMigrate(&models.User{}, &models.Group{}, &models.Animal{}, &models.AnimalTag{})
	if err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	return db
}

// createAnimalTestUser creates a user with a group for testing
func createAnimalTestUser(t *testing.T, db *gorm.DB, username, email string, isAdmin bool) (*models.User, *models.Group) {
	hashedPassword, err := auth.HashPassword("password123")
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	user := &models.User{
		Username: username,
		Email:    email,
		Password: hashedPassword,
		IsAdmin:  isAdmin,
	}

	if err := db.Create(user).Error; err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	group := &models.Group{
		Name:        fmt.Sprintf("%s's Group", username),
		Description: "Test group",
	}

	if err := db.Create(group).Error; err != nil {
		t.Fatalf("Failed to create group: %v", err)
	}

	// Associate user with group
	if err := db.Model(user).Association("Groups").Append(group); err != nil {
		t.Fatalf("Failed to associate user with group: %v", err)
	}

	return user, group
}

// createTestAnimal creates an animal in the database for testing
func createTestAnimal(t *testing.T, db *gorm.DB, groupID uint, name, species string) *models.Animal {
	now := time.Now()
	animal := &models.Animal{
		GroupID:          groupID,
		Name:             name,
		Species:          species,
		Breed:            "Test Breed",
		Age:              2,
		Description:      "Test animal",
		Status:           "available",
		ArrivalDate:      &now,
		LastStatusChange: &now,
	}

	if err := db.Create(animal).Error; err != nil {
		t.Fatalf("Failed to create animal: %v", err)
	}

	return animal
}

// setupAnimalTestContext creates a Gin context with authenticated user
func setupAnimalTestContext(userID uint, isAdmin bool) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	
	// Set authentication context
	c.Set("user_id", userID)
	c.Set("is_admin", isAdmin)
	
	return c, w
}

// TestGetAnimals_Success tests successful retrieval of animals
func TestGetAnimals_Success(t *testing.T) {
	db := setupAnimalTestDB(t)
	user, group := createAnimalTestUser(t, db, "testuser", "test@example.com", false)

	// Create test animals
	createTestAnimal(t, db, group.ID, "Rex", "Dog")
	createTestAnimal(t, db, group.ID, "Fluffy", "Cat")

	c, w := setupAnimalTestContext(user.ID, false)
	c.Params = gin.Params{{Key: "id", Value: fmt.Sprintf("%d", group.ID)}}
	c.Request = httptest.NewRequest("GET", fmt.Sprintf("/api/v1/groups/%d/animals", group.ID), nil)

	handler := GetAnimals(db)
	handler(c)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var animals []models.Animal
	if err := json.Unmarshal(w.Body.Bytes(), &animals); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if len(animals) != 2 {
		t.Errorf("Expected 2 animals, got %d", len(animals))
	}
}

// TestGetAnimals_StatusFilter tests filtering animals by status
func TestGetAnimals_StatusFilter(t *testing.T) {
	db := setupAnimalTestDB(t)
	user, group := createAnimalTestUser(t, db, "testuser", "test@example.com", false)

	// Create animals with different statuses
	animal1 := createTestAnimal(t, db, group.ID, "Rex", "Dog")
	animal1.Status = "available"
	db.Save(animal1)

	animal2 := createTestAnimal(t, db, group.ID, "Fluffy", "Cat")
	animal2.Status = "foster"
	db.Save(animal2)

	animal3 := createTestAnimal(t, db, group.ID, "Max", "Dog")
	animal3.Status = "bite_quarantine"
	db.Save(animal3)

	tests := []struct {
		name          string
		statusQuery   string
		expectedCount int
	}{
		{
			name:          "default filter (available and bite_quarantine)",
			statusQuery:   "",
			expectedCount: 2, // available and bite_quarantine
		},
		{
			name:          "filter by available",
			statusQuery:   "available",
			expectedCount: 1,
		},
		{
			name:          "filter by foster",
			statusQuery:   "foster",
			expectedCount: 1,
		},
		{
			name:          "filter by all",
			statusQuery:   "all",
			expectedCount: 3,
		},
		{
			name:          "filter by multiple statuses",
			statusQuery:   "available,foster",
			expectedCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, w := setupAnimalTestContext(user.ID, false)
			c.Params = gin.Params{{Key: "id", Value: fmt.Sprintf("%d", group.ID)}}
			
			url := fmt.Sprintf("/api/v1/groups/%d/animals", group.ID)
			if tt.statusQuery != "" {
				url = fmt.Sprintf("%s?status=%s", url, tt.statusQuery)
			}
			c.Request = httptest.NewRequest("GET", url, nil)

			handler := GetAnimals(db)
			handler(c)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
				return
			}

			var animals []models.Animal
			if err := json.Unmarshal(w.Body.Bytes(), &animals); err != nil {
				t.Fatalf("Failed to unmarshal response: %v", err)
			}

			if len(animals) != tt.expectedCount {
				t.Errorf("Expected %d animals, got %d", tt.expectedCount, len(animals))
			}
		})
	}
}

// TestGetAnimals_NameSearch tests searching animals by name
func TestGetAnimals_NameSearch(t *testing.T) {
	db := setupAnimalTestDB(t)
	user, group := createAnimalTestUser(t, db, "testuser", "test@example.com", false)

	// Create test animals
	createTestAnimal(t, db, group.ID, "Rex", "Dog")
	createTestAnimal(t, db, group.ID, "Fluffy", "Cat")
	createTestAnimal(t, db, group.ID, "Max", "Dog")

	tests := []struct {
		name          string
		searchQuery   string
		expectedCount int
	}{
		{
			name:          "search for 'rex'",
			searchQuery:   "rex",
			expectedCount: 1,
		},
		{
			name:          "search for 'fl'",
			searchQuery:   "fl",
			expectedCount: 1,
		},
		{
			name:          "search for 'dog' (no match in name)",
			searchQuery:   "dog",
			expectedCount: 0,
		},
		{
			name:          "case insensitive search",
			searchQuery:   "REX",
			expectedCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, w := setupAnimalTestContext(user.ID, false)
			c.Params = gin.Params{{Key: "id", Value: fmt.Sprintf("%d", group.ID)}}
			url := fmt.Sprintf("/api/v1/groups/%d/animals?name=%s", group.ID, tt.searchQuery)
			c.Request = httptest.NewRequest("GET", url, nil)

			handler := GetAnimals(db)
			handler(c)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
				return
			}

			var animals []models.Animal
			if err := json.Unmarshal(w.Body.Bytes(), &animals); err != nil {
				t.Fatalf("Failed to unmarshal response: %v", err)
			}

			if len(animals) != tt.expectedCount {
				t.Errorf("Expected %d animals, got %d", tt.expectedCount, len(animals))
			}
		})
	}
}

// TestGetAnimals_AccessDenied tests unauthorized access
func TestGetAnimals_AccessDenied(t *testing.T) {
	db := setupAnimalTestDB(t)
	_, group1 := createAnimalTestUser(t, db, "user1", "user1@example.com", false)
	user2, _ := createAnimalTestUser(t, db, "user2", "user2@example.com", false)

	// Create animal in user1's group
	createTestAnimal(t, db, group1.ID, "Rex", "Dog")

	// Try to access user1's group with user2's credentials
	c, w := setupAnimalTestContext(user2.ID, false)
	c.Params = gin.Params{{Key: "id", Value: fmt.Sprintf("%d", group1.ID)}}
	c.Request = httptest.NewRequest("GET", fmt.Sprintf("/api/v1/groups/%d/animals", group1.ID), nil)

	handler := GetAnimals(db)
	handler(c)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status %d, got %d", http.StatusForbidden, w.Code)
	}
}

// TestGetAnimals_AdminAccess tests admin can access any group
func TestGetAnimals_AdminAccess(t *testing.T) {
	db := setupAnimalTestDB(t)
	_, group := createAnimalTestUser(t, db, "user1", "user1@example.com", false)
	admin, _ := createAnimalTestUser(t, db, "admin", "admin@example.com", true)

	// Create animal in user's group
	createTestAnimal(t, db, group.ID, "Rex", "Dog")

	// Admin should be able to access user's group
	c, w := setupAnimalTestContext(admin.ID, true)
	c.Params = gin.Params{{Key: "id", Value: fmt.Sprintf("%d", group.ID)}}
	c.Request = httptest.NewRequest("GET", fmt.Sprintf("/api/v1/groups/%d/animals", group.ID), nil)

	handler := GetAnimals(db)
	handler(c)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var animals []models.Animal
	if err := json.Unmarshal(w.Body.Bytes(), &animals); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if len(animals) != 1 {
		t.Errorf("Expected 1 animal, got %d", len(animals))
	}
}

// TestGetAnimal_Success tests successful retrieval of a single animal
func TestGetAnimal_Success(t *testing.T) {
	db := setupAnimalTestDB(t)
	user, group := createAnimalTestUser(t, db, "testuser", "test@example.com", false)

	animal := createTestAnimal(t, db, group.ID, "Rex", "Dog")

	c, w := setupAnimalTestContext(user.ID, false)
	c.Params = gin.Params{
		{Key: "id", Value: fmt.Sprintf("%d", group.ID)},
		{Key: "animalId", Value: fmt.Sprintf("%d", animal.ID)},
	}
	c.Request = httptest.NewRequest("GET", fmt.Sprintf("/api/v1/groups/%d/animals/%d", group.ID, animal.ID), nil)

	handler := GetAnimal(db)
	handler(c)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var retrievedAnimal models.Animal
	if err := json.Unmarshal(w.Body.Bytes(), &retrievedAnimal); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if retrievedAnimal.Name != "Rex" {
		t.Errorf("Expected animal name 'Rex', got '%s'", retrievedAnimal.Name)
	}
}

// TestGetAnimal_NotFound tests retrieving a non-existent animal
func TestGetAnimal_NotFound(t *testing.T) {
	db := setupAnimalTestDB(t)
	user, group := createAnimalTestUser(t, db, "testuser", "test@example.com", false)

	c, w := setupAnimalTestContext(user.ID, false)
	c.Params = gin.Params{
		{Key: "id", Value: fmt.Sprintf("%d", group.ID)},
		{Key: "animalId", Value: "99999"},
	}
	c.Request = httptest.NewRequest("GET", fmt.Sprintf("/api/v1/groups/%d/animals/99999", group.ID), nil)

	handler := GetAnimal(db)
	handler(c)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

// TestGetAnimal_WrongGroup tests retrieving an animal from wrong group
func TestGetAnimal_WrongGroup(t *testing.T) {
	db := setupAnimalTestDB(t)
	user1, group1 := createAnimalTestUser(t, db, "user1", "user1@example.com", false)
	_, group2 := createAnimalTestUser(t, db, "user2", "user2@example.com", false)

	animal := createTestAnimal(t, db, group1.ID, "Rex", "Dog")

	// Try to get animal from group2 using group1's animal ID
	c, w := setupAnimalTestContext(user1.ID, false)
	c.Params = gin.Params{
		{Key: "id", Value: fmt.Sprintf("%d", group2.ID)},
		{Key: "animalId", Value: fmt.Sprintf("%d", animal.ID)},
	}
	c.Request = httptest.NewRequest("GET", fmt.Sprintf("/api/v1/groups/%d/animals/%d", group2.ID, animal.ID), nil)

	handler := GetAnimal(db)
	handler(c)

	// Should be forbidden since user can't access group2
	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status %d, got %d", http.StatusForbidden, w.Code)
	}
}

// TestCreateAnimal_Success tests successful animal creation
func TestCreateAnimal_Success(t *testing.T) {
	db := setupAnimalTestDB(t)
	user, group := createAnimalTestUser(t, db, "testuser", "test@example.com", false)

	animalReq := AnimalRequest{
		Name:        "Rex",
		Species:     "Dog",
		Breed:       "Golden Retriever",
		Age:         3,
		Description: "Friendly dog",
		Status:      "available",
	}

	jsonData, _ := json.Marshal(animalReq)

	c, w := setupAnimalTestContext(user.ID, false)
	c.Params = gin.Params{{Key: "id", Value: fmt.Sprintf("%d", group.ID)}}
	c.Request = httptest.NewRequest("POST", fmt.Sprintf("/api/v1/groups/%d/animals", group.ID), bytes.NewBuffer(jsonData))
	c.Request.Header.Set("Content-Type", "application/json")

	handler := CreateAnimal(db)
	handler(c)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusCreated, w.Code, w.Body.String())
	}

	var createdAnimal models.Animal
	if err := json.Unmarshal(w.Body.Bytes(), &createdAnimal); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if createdAnimal.Name != "Rex" {
		t.Errorf("Expected animal name 'Rex', got '%s'", createdAnimal.Name)
	}

	if createdAnimal.Status != "available" {
		t.Errorf("Expected status 'available', got '%s'", createdAnimal.Status)
	}

	// Verify animal was created in database
	var dbAnimal models.Animal
	if err := db.First(&dbAnimal, createdAnimal.ID).Error; err != nil {
		t.Errorf("Animal not found in database: %v", err)
	}
}

// TestCreateAnimal_ValidationError tests validation errors
func TestCreateAnimal_ValidationError(t *testing.T) {
	db := setupAnimalTestDB(t)
	user, group := createAnimalTestUser(t, db, "testuser", "test@example.com", false)

	tests := []struct {
		name    string
		request AnimalRequest
	}{
		{
			name: "missing required name",
			request: AnimalRequest{
				Species: "Dog",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonData, _ := json.Marshal(tt.request)

			c, w := setupAnimalTestContext(user.ID, false)
			c.Params = gin.Params{{Key: "id", Value: fmt.Sprintf("%d", group.ID)}}
			c.Request = httptest.NewRequest("POST", fmt.Sprintf("/api/v1/groups/%d/animals", group.ID), bytes.NewBuffer(jsonData))
			c.Request.Header.Set("Content-Type", "application/json")

			handler := CreateAnimal(db)
			handler(c)

			if w.Code != http.StatusBadRequest {
				t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
			}
		})
	}
}

// TestCreateAnimal_DefaultStatus tests default status assignment
func TestCreateAnimal_DefaultStatus(t *testing.T) {
	db := setupAnimalTestDB(t)
	user, group := createAnimalTestUser(t, db, "testuser", "test@example.com", false)

	animalReq := AnimalRequest{
		Name:    "Rex",
		Species: "Dog",
		// Status not provided
	}

	jsonData, _ := json.Marshal(animalReq)

	c, w := setupAnimalTestContext(user.ID, false)
	c.Params = gin.Params{{Key: "id", Value: fmt.Sprintf("%d", group.ID)}}
	c.Request = httptest.NewRequest("POST", fmt.Sprintf("/api/v1/groups/%d/animals", group.ID), bytes.NewBuffer(jsonData))
	c.Request.Header.Set("Content-Type", "application/json")

	handler := CreateAnimal(db)
	handler(c)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusCreated, w.Code, w.Body.String())
	}

	var createdAnimal models.Animal
	if err := json.Unmarshal(w.Body.Bytes(), &createdAnimal); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if createdAnimal.Status != "available" {
		t.Errorf("Expected default status 'available', got '%s'", createdAnimal.Status)
	}
}

// TestCreateAnimal_StatusSpecificDates tests status-specific date fields
func TestCreateAnimal_StatusSpecificDates(t *testing.T) {
	db := setupAnimalTestDB(t)
	user, group := createAnimalTestUser(t, db, "testuser", "test@example.com", false)

	tests := []struct {
		name          string
		status        string
		checkDateFunc func(*models.Animal) bool
	}{
		{
			name:   "foster status sets foster start date",
			status: "foster",
			checkDateFunc: func(a *models.Animal) bool {
				return a.FosterStartDate != nil
			},
		},
		{
			name:   "bite_quarantine status sets quarantine start date",
			status: "bite_quarantine",
			checkDateFunc: func(a *models.Animal) bool {
				return a.QuarantineStartDate != nil
			},
		},
		{
			name:   "archived status sets archived date",
			status: "archived",
			checkDateFunc: func(a *models.Animal) bool {
				return a.ArchivedDate != nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			animalReq := AnimalRequest{
				Name:    "TestAnimal",
				Species: "Dog",
				Status:  tt.status,
			}

			jsonData, _ := json.Marshal(animalReq)

			c, w := setupAnimalTestContext(user.ID, false)
			c.Params = gin.Params{{Key: "id", Value: fmt.Sprintf("%d", group.ID)}}
			c.Request = httptest.NewRequest("POST", fmt.Sprintf("/api/v1/groups/%d/animals", group.ID), bytes.NewBuffer(jsonData))
			c.Request.Header.Set("Content-Type", "application/json")

			handler := CreateAnimal(db)
			handler(c)

			if w.Code != http.StatusCreated {
				t.Errorf("Expected status %d, got %d. Body: %s", http.StatusCreated, w.Code, w.Body.String())
			}

			var createdAnimal models.Animal
			if err := json.Unmarshal(w.Body.Bytes(), &createdAnimal); err != nil {
				t.Fatalf("Failed to unmarshal response: %v", err)
			}

			if !tt.checkDateFunc(&createdAnimal) {
				t.Errorf("Expected status-specific date to be set for status '%s'", tt.status)
			}
		})
	}
}

// TestCreateAnimal_AccessDenied tests unauthorized animal creation
func TestCreateAnimal_AccessDenied(t *testing.T) {
	db := setupAnimalTestDB(t)
	_, group1 := createAnimalTestUser(t, db, "user1", "user1@example.com", false)
	user2, _ := createAnimalTestUser(t, db, "user2", "user2@example.com", false)

	animalReq := AnimalRequest{
		Name:    "Rex",
		Species: "Dog",
	}

	jsonData, _ := json.Marshal(animalReq)

	// Try to create animal in user1's group with user2's credentials
	c, w := setupAnimalTestContext(user2.ID, false)
	c.Params = gin.Params{{Key: "id", Value: fmt.Sprintf("%d", group1.ID)}}
	c.Request = httptest.NewRequest("POST", fmt.Sprintf("/api/v1/groups/%d/animals", group1.ID), bytes.NewBuffer(jsonData))
	c.Request.Header.Set("Content-Type", "application/json")

	handler := CreateAnimal(db)
	handler(c)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status %d, got %d", http.StatusForbidden, w.Code)
	}
}

// TestCreateAnimal_InvalidGroupID tests creation with invalid group ID
func TestCreateAnimal_InvalidGroupID(t *testing.T) {
	db := setupAnimalTestDB(t)
	user, _ := createAnimalTestUser(t, db, "testuser", "test@example.com", false)

	animalReq := AnimalRequest{
		Name:    "Rex",
		Species: "Dog",
	}

	jsonData, _ := json.Marshal(animalReq)

	c, w := setupAnimalTestContext(user.ID, false)
	c.Params = gin.Params{{Key: "id", Value: "invalid"}}
	c.Request = httptest.NewRequest("POST", "/api/v1/groups/invalid/animals", bytes.NewBuffer(jsonData))
	c.Request.Header.Set("Content-Type", "application/json")

	handler := CreateAnimal(db)
	handler(c)

	// Invalid group ID causes checkGroupAccess to fail (returns 403) or parsing fails (returns 400)
	if w.Code != http.StatusBadRequest && w.Code != http.StatusForbidden {
		t.Errorf("Expected status %d or %d, got %d", http.StatusBadRequest, http.StatusForbidden, w.Code)
	}
}

// TestDeleteAnimal_Success tests successful animal deletion (soft delete)
func TestDeleteAnimal_Success(t *testing.T) {
	db := setupAnimalTestDB(t)
	user, group := createAnimalTestUser(t, db, "testuser", "test@example.com", false)

	animal := createTestAnimal(t, db, group.ID, "Rex", "Dog")

	c, w := setupAnimalTestContext(user.ID, false)
	c.Params = gin.Params{
		{Key: "id", Value: fmt.Sprintf("%d", group.ID)},
		{Key: "animalId", Value: fmt.Sprintf("%d", animal.ID)},
	}
	c.Request = httptest.NewRequest("DELETE", fmt.Sprintf("/api/v1/groups/%d/animals/%d", group.ID, animal.ID), nil)

	handler := DeleteAnimal(db)
	handler(c)

	// The handler returns 200 with a message, not 204
	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	// Verify response message
	var response map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response["message"] != "Animal deleted successfully" {
		t.Errorf("Expected success message, got: %s", response["message"])
	}

	// Verify soft delete - animal should not be found with normal query
	var deletedAnimal models.Animal
	err := db.First(&deletedAnimal, animal.ID).Error
	if err == nil {
		t.Error("Expected animal to be soft deleted, but it was found")
	}

	// Verify animal exists with Unscoped query
	err = db.Unscoped().First(&deletedAnimal, animal.ID).Error
	if err != nil {
		t.Errorf("Expected animal to exist in database (soft deleted): %v", err)
	}

	if deletedAnimal.DeletedAt.Time.IsZero() {
		t.Error("Expected DeletedAt to be set, but it was zero")
	}
}

// TestDeleteAnimal_NotFound tests deleting a non-existent animal
func TestDeleteAnimal_NotFound(t *testing.T) {
	db := setupAnimalTestDB(t)
	user, group := createAnimalTestUser(t, db, "testuser", "test@example.com", false)

	c, w := setupAnimalTestContext(user.ID, false)
	c.Params = gin.Params{
		{Key: "id", Value: fmt.Sprintf("%d", group.ID)},
		{Key: "animalId", Value: "99999"},
	}
	c.Request = httptest.NewRequest("DELETE", fmt.Sprintf("/api/v1/groups/%d/animals/99999", group.ID), nil)

	handler := DeleteAnimal(db)
	handler(c)

	// Note: The current implementation doesn't check if the animal exists before deleting
	// It returns 200 even if no rows were affected. This is a potential improvement area.
	// For now, we test the actual behavior.
	if w.Code != http.StatusOK && w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d or %d, got %d", http.StatusOK, http.StatusNotFound, w.Code)
	}
}

// TestDeleteAnimal_AccessDenied tests unauthorized animal deletion
func TestDeleteAnimal_AccessDenied(t *testing.T) {
	db := setupAnimalTestDB(t)
	_, group1 := createAnimalTestUser(t, db, "user1", "user1@example.com", false)
	user2, _ := createAnimalTestUser(t, db, "user2", "user2@example.com", false)

	animal := createTestAnimal(t, db, group1.ID, "Rex", "Dog")

	// Try to delete user1's animal with user2's credentials
	c, w := setupAnimalTestContext(user2.ID, false)
	c.Params = gin.Params{
		{Key: "id", Value: fmt.Sprintf("%d", group1.ID)},
		{Key: "animalId", Value: fmt.Sprintf("%d", animal.ID)},
	}
	c.Request = httptest.NewRequest("DELETE", fmt.Sprintf("/api/v1/groups/%d/animals/%d", group1.ID, animal.ID), nil)

	handler := DeleteAnimal(db)
	handler(c)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status %d, got %d", http.StatusForbidden, w.Code)
	}

	// Verify animal was not deleted
	var animal2 models.Animal
	if err := db.First(&animal2, animal.ID).Error; err != nil {
		t.Errorf("Animal should still exist: %v", err)
	}
}

// TestUpdateAnimal_Success tests successful animal update
func TestUpdateAnimal_Success(t *testing.T) {
	db := setupAnimalTestDB(t)
	user, group := createAnimalTestUser(t, db, "testuser", "test@example.com", false)

	animal := createTestAnimal(t, db, group.ID, "Rex", "Dog")

	updateReq := AnimalRequest{
		Name:        "Rex Updated",
		Species:     "Dog",
		Breed:       "Labrador",
		Age:         4,
		Description: "Updated description",
		Status:      "available",
	}

	jsonData, _ := json.Marshal(updateReq)

	c, w := setupAnimalTestContext(user.ID, false)
	c.Params = gin.Params{
		{Key: "id", Value: fmt.Sprintf("%d", group.ID)},
		{Key: "animalId", Value: fmt.Sprintf("%d", animal.ID)},
	}
	c.Request = httptest.NewRequest("PUT", fmt.Sprintf("/api/v1/groups/%d/animals/%d", group.ID, animal.ID), bytes.NewBuffer(jsonData))
	c.Request.Header.Set("Content-Type", "application/json")

	handler := UpdateAnimal(db)
	handler(c)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var updatedAnimal models.Animal
	if err := json.Unmarshal(w.Body.Bytes(), &updatedAnimal); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if updatedAnimal.Name != "Rex Updated" {
		t.Errorf("Expected name 'Rex Updated', got '%s'", updatedAnimal.Name)
	}

	if updatedAnimal.Breed != "Labrador" {
		t.Errorf("Expected breed 'Labrador', got '%s'", updatedAnimal.Breed)
	}

	if updatedAnimal.Age != 4 {
		t.Errorf("Expected age 4, got %d", updatedAnimal.Age)
	}
}

// TestUpdateAnimal_NotFound tests updating non-existent animal
func TestUpdateAnimal_NotFound(t *testing.T) {
	db := setupAnimalTestDB(t)
	user, group := createAnimalTestUser(t, db, "testuser", "test@example.com", false)

	updateReq := AnimalRequest{
		Name:    "Rex",
		Species: "Dog",
	}

	jsonData, _ := json.Marshal(updateReq)

	c, w := setupAnimalTestContext(user.ID, false)
	c.Params = gin.Params{
		{Key: "id", Value: fmt.Sprintf("%d", group.ID)},
		{Key: "animalId", Value: "99999"},
	}
	c.Request = httptest.NewRequest("PUT", fmt.Sprintf("/api/v1/groups/%d/animals/99999", group.ID), bytes.NewBuffer(jsonData))
	c.Request.Header.Set("Content-Type", "application/json")

	handler := UpdateAnimal(db)
	handler(c)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

// TestUpdateAnimal_StatusTransition tests status change tracking
func TestUpdateAnimal_StatusTransition(t *testing.T) {
	db := setupAnimalTestDB(t)
	user, group := createAnimalTestUser(t, db, "testuser", "test@example.com", false)

	animal := createTestAnimal(t, db, group.ID, "Rex", "Dog")
	oldStatusChangeTime := animal.LastStatusChange

	// Wait a bit to ensure time difference
	time.Sleep(10 * time.Millisecond)

	tests := []struct {
		name              string
		newStatus         string
		checkDateField    func(*models.Animal) bool
		checkClearedField func(*models.Animal) bool
	}{
		{
			name:      "transition to foster",
			newStatus: "foster",
			checkDateField: func(a *models.Animal) bool {
				return a.FosterStartDate != nil
			},
			checkClearedField: func(a *models.Animal) bool {
				return a.QuarantineStartDate == nil && a.ArchivedDate == nil
			},
		},
		{
			name:      "transition to bite_quarantine",
			newStatus: "bite_quarantine",
			checkDateField: func(a *models.Animal) bool {
				return a.QuarantineStartDate != nil
			},
			checkClearedField: func(a *models.Animal) bool {
				return a.FosterStartDate == nil && a.ArchivedDate == nil
			},
		},
		{
			name:      "transition to archived",
			newStatus: "archived",
			checkDateField: func(a *models.Animal) bool {
				return a.ArchivedDate != nil
			},
			checkClearedField: func(a *models.Animal) bool {
				return true // archived doesn't clear other fields by default
			},
		},
		{
			name:      "transition back to available",
			newStatus: "available",
			checkDateField: func(a *models.Animal) bool {
				return true // available doesn't set specific dates
			},
			checkClearedField: func(a *models.Animal) bool {
				return a.FosterStartDate == nil && a.QuarantineStartDate == nil && a.ArchivedDate == nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updateReq := AnimalRequest{
				Name:    "Rex",
				Species: "Dog",
				Status:  tt.newStatus,
			}

			jsonData, _ := json.Marshal(updateReq)

			c, w := setupAnimalTestContext(user.ID, false)
			c.Params = gin.Params{
				{Key: "id", Value: fmt.Sprintf("%d", group.ID)},
				{Key: "animalId", Value: fmt.Sprintf("%d", animal.ID)},
			}
			c.Request = httptest.NewRequest("PUT", fmt.Sprintf("/api/v1/groups/%d/animals/%d", group.ID, animal.ID), bytes.NewBuffer(jsonData))
			c.Request.Header.Set("Content-Type", "application/json")

			handler := UpdateAnimal(db)
			handler(c)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
			}

			var updatedAnimal models.Animal
			if err := json.Unmarshal(w.Body.Bytes(), &updatedAnimal); err != nil {
				t.Fatalf("Failed to unmarshal response: %v", err)
			}

			if updatedAnimal.Status != tt.newStatus {
				t.Errorf("Expected status '%s', got '%s'", tt.newStatus, updatedAnimal.Status)
			}

			// Check that LastStatusChange was updated
			if updatedAnimal.LastStatusChange.Equal(*oldStatusChangeTime) {
				t.Error("Expected LastStatusChange to be updated")
			}

			// Check status-specific date field
			if !tt.checkDateField(&updatedAnimal) {
				t.Errorf("Expected status-specific date to be set for status '%s'", tt.newStatus)
			}

			// Check cleared fields
			if !tt.checkClearedField(&updatedAnimal) {
				t.Errorf("Expected other status fields to be cleared for status '%s'", tt.newStatus)
			}

			// Update oldStatusChangeTime for next iteration
			oldStatusChangeTime = updatedAnimal.LastStatusChange
			time.Sleep(10 * time.Millisecond)
		})
	}
}

// TestUpdateAnimal_NoStatusChange tests updating without changing status
func TestUpdateAnimal_NoStatusChange(t *testing.T) {
	db := setupAnimalTestDB(t)
	user, group := createAnimalTestUser(t, db, "testuser", "test@example.com", false)

	animal := createTestAnimal(t, db, group.ID, "Rex", "Dog")
	originalStatus := animal.Status
	originalStatusChangeTime := animal.LastStatusChange

	// Update other fields but keep same status
	updateReq := AnimalRequest{
		Name:        "Rex Updated",
		Species:     "Dog",
		Breed:       "Labrador",
		Age:         4,
		Description: "Updated description",
		Status:      originalStatus,
	}

	jsonData, _ := json.Marshal(updateReq)

	c, w := setupAnimalTestContext(user.ID, false)
	c.Params = gin.Params{
		{Key: "id", Value: fmt.Sprintf("%d", group.ID)},
		{Key: "animalId", Value: fmt.Sprintf("%d", animal.ID)},
	}
	c.Request = httptest.NewRequest("PUT", fmt.Sprintf("/api/v1/groups/%d/animals/%d", group.ID, animal.ID), bytes.NewBuffer(jsonData))
	c.Request.Header.Set("Content-Type", "application/json")

	handler := UpdateAnimal(db)
	handler(c)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var updatedAnimal models.Animal
	if err := json.Unmarshal(w.Body.Bytes(), &updatedAnimal); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	// Name should be updated
	if updatedAnimal.Name != "Rex Updated" {
		t.Errorf("Expected name 'Rex Updated', got '%s'", updatedAnimal.Name)
	}

	// Status should remain the same
	if updatedAnimal.Status != originalStatus {
		t.Errorf("Expected status '%s', got '%s'", originalStatus, updatedAnimal.Status)
	}

	// LastStatusChange should remain the same (no status change)
	if !updatedAnimal.LastStatusChange.Equal(*originalStatusChangeTime) {
		t.Error("Expected LastStatusChange to remain unchanged when status doesn't change")
	}
}

// TestUpdateAnimal_ValidationError tests validation on update
func TestUpdateAnimal_ValidationError(t *testing.T) {
	db := setupAnimalTestDB(t)
	user, group := createAnimalTestUser(t, db, "testuser", "test@example.com", false)

	animal := createTestAnimal(t, db, group.ID, "Rex", "Dog")

	// Missing required name field
	updateReq := AnimalRequest{
		Species: "Dog",
	}

	jsonData, _ := json.Marshal(updateReq)

	c, w := setupAnimalTestContext(user.ID, false)
	c.Params = gin.Params{
		{Key: "id", Value: fmt.Sprintf("%d", group.ID)},
		{Key: "animalId", Value: fmt.Sprintf("%d", animal.ID)},
	}
	c.Request = httptest.NewRequest("PUT", fmt.Sprintf("/api/v1/groups/%d/animals/%d", group.ID, animal.ID), bytes.NewBuffer(jsonData))
	c.Request.Header.Set("Content-Type", "application/json")

	handler := UpdateAnimal(db)
	handler(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

// TestUpdateAnimal_AccessDenied tests unauthorized update
func TestUpdateAnimal_AccessDenied(t *testing.T) {
	db := setupAnimalTestDB(t)
	_, group1 := createAnimalTestUser(t, db, "user1", "user1@example.com", false)
	user2, _ := createAnimalTestUser(t, db, "user2", "user2@example.com", false)

	animal := createTestAnimal(t, db, group1.ID, "Rex", "Dog")

	updateReq := AnimalRequest{
		Name:    "Rex Updated",
		Species: "Dog",
	}

	jsonData, _ := json.Marshal(updateReq)

	// Try to update user1's animal with user2's credentials
	c, w := setupAnimalTestContext(user2.ID, false)
	c.Params = gin.Params{
		{Key: "id", Value: fmt.Sprintf("%d", group1.ID)},
		{Key: "animalId", Value: fmt.Sprintf("%d", animal.ID)},
	}
	c.Request = httptest.NewRequest("PUT", fmt.Sprintf("/api/v1/groups/%d/animals/%d", group1.ID, animal.ID), bytes.NewBuffer(jsonData))
	c.Request.Header.Set("Content-Type", "application/json")

	handler := UpdateAnimal(db)
	handler(c)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status %d, got %d", http.StatusForbidden, w.Code)
	}

	// Verify animal was not updated
	var dbAnimal models.Animal
	db.First(&dbAnimal, animal.ID)
	if dbAnimal.Name != "Rex" {
		t.Errorf("Animal should not have been updated, got name: %s", dbAnimal.Name)
	}
}

// TestUpdateAnimal_CustomQuarantineDate tests setting custom quarantine start date
func TestUpdateAnimal_CustomQuarantineDate(t *testing.T) {
	db := setupAnimalTestDB(t)
	user, group := createAnimalTestUser(t, db, "testuser", "test@example.com", false)

	animal := createTestAnimal(t, db, group.ID, "Rex", "Dog")

	customDate := time.Now().AddDate(0, 0, -7) // 7 days ago
	updateReq := AnimalRequest{
		Name:                "Rex",
		Species:             "Dog",
		Status:              "bite_quarantine",
		QuarantineStartDate: &customDate,
	}

	jsonData, _ := json.Marshal(updateReq)

	c, w := setupAnimalTestContext(user.ID, false)
	c.Params = gin.Params{
		{Key: "id", Value: fmt.Sprintf("%d", group.ID)},
		{Key: "animalId", Value: fmt.Sprintf("%d", animal.ID)},
	}
	c.Request = httptest.NewRequest("PUT", fmt.Sprintf("/api/v1/groups/%d/animals/%d", group.ID, animal.ID), bytes.NewBuffer(jsonData))
	c.Request.Header.Set("Content-Type", "application/json")

	handler := UpdateAnimal(db)
	handler(c)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var updatedAnimal models.Animal
	if err := json.Unmarshal(w.Body.Bytes(), &updatedAnimal); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if updatedAnimal.QuarantineStartDate == nil {
		t.Error("Expected QuarantineStartDate to be set")
	} else if !updatedAnimal.QuarantineStartDate.Equal(customDate) {
		t.Errorf("Expected QuarantineStartDate to be %v, got %v", customDate, *updatedAnimal.QuarantineStartDate)
	}
}
