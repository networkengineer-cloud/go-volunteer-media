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

	// Age is auto-computed from EstimatedBirthDate when set; the test animal is ~2 years old
	if updatedAnimal.Age != 2 {
		t.Errorf("Expected age 2 (auto-computed from birth date), got %d", updatedAnimal.Age)
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
		Name:    "Rex",
		Species: "Dog",
		Status:  "bite_quarantine",
		QuarantineStartDate: NullableTime{
			Time:  &customDate,
			Valid: true,
		},
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

// TestBulkUpdateAnimals_StatusUpdate tests bulk status update
func TestBulkUpdateAnimals_StatusUpdate(t *testing.T) {
	db := setupAnimalTestDB(t)
	user, group := createAnimalTestUser(t, db, "testuser", "test@example.com", true) // Admin user

	// Create multiple test animals
	animal1 := createTestAnimal(t, db, group.ID, "Rex", "Dog")
	animal2 := createTestAnimal(t, db, group.ID, "Fluffy", "Cat")
	animal3 := createTestAnimal(t, db, group.ID, "Max", "Dog")

	newStatus := "foster"
	bulkReq := BulkUpdateAnimalsRequest{
		AnimalIDs: []uint{animal1.ID, animal2.ID, animal3.ID},
		Status:    &newStatus,
	}

	jsonData, _ := json.Marshal(bulkReq)

	c, w := setupAnimalTestContext(user.ID, true)
	c.Request = httptest.NewRequest("PATCH", "/api/v1/admin/animals/bulk", bytes.NewBuffer(jsonData))
	c.Request.Header.Set("Content-Type", "application/json")

	handler := BulkUpdateAnimals(db)
	handler(c)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response["count"].(float64) != 3 {
		t.Errorf("Expected count 3, got %v", response["count"])
	}

	// Verify animals were updated
	var animals []models.Animal
	db.Where("id IN ?", []uint{animal1.ID, animal2.ID, animal3.ID}).Find(&animals)

	for _, animal := range animals {
		if animal.Status != "foster" {
			t.Errorf("Expected animal %s to have status 'foster', got '%s'", animal.Name, animal.Status)
		}
	}
}

// TestBulkUpdateAnimals_GroupUpdate tests bulk group update
func TestBulkUpdateAnimals_GroupUpdate(t *testing.T) {
	db := setupAnimalTestDB(t)
	user, group1 := createAnimalTestUser(t, db, "testuser", "test@example.com", true)

	// Create second group
	group2 := &models.Group{
		Name:        "Group 2",
		Description: "Test group 2",
	}
	db.Create(group2)

	// Create animals in group1
	animal1 := createTestAnimal(t, db, group1.ID, "Rex", "Dog")
	animal2 := createTestAnimal(t, db, group1.ID, "Fluffy", "Cat")

	bulkReq := BulkUpdateAnimalsRequest{
		AnimalIDs: []uint{animal1.ID, animal2.ID},
		GroupID:   &group2.ID,
	}

	jsonData, _ := json.Marshal(bulkReq)

	c, w := setupAnimalTestContext(user.ID, true)
	c.Request = httptest.NewRequest("PATCH", "/api/v1/admin/animals/bulk", bytes.NewBuffer(jsonData))
	c.Request.Header.Set("Content-Type", "application/json")

	handler := BulkUpdateAnimals(db)
	handler(c)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	// Verify animals were moved to group2
	var animals []models.Animal
	db.Where("id IN ?", []uint{animal1.ID, animal2.ID}).Find(&animals)

	for _, animal := range animals {
		if animal.GroupID != group2.ID {
			t.Errorf("Expected animal %s to be in group %d, got group %d", animal.Name, group2.ID, animal.GroupID)
		}
	}
}

// TestBulkUpdateAnimals_BothUpdates tests updating both status and group
func TestBulkUpdateAnimals_BothUpdates(t *testing.T) {
	db := setupAnimalTestDB(t)
	user, group1 := createAnimalTestUser(t, db, "testuser", "test@example.com", true)

	// Create second group
	group2 := &models.Group{
		Name:        "Group 2",
		Description: "Test group 2",
	}
	db.Create(group2)

	animal := createTestAnimal(t, db, group1.ID, "Rex", "Dog")

	newStatus := "foster"
	bulkReq := BulkUpdateAnimalsRequest{
		AnimalIDs: []uint{animal.ID},
		GroupID:   &group2.ID,
		Status:    &newStatus,
	}

	jsonData, _ := json.Marshal(bulkReq)

	c, w := setupAnimalTestContext(user.ID, true)
	c.Request = httptest.NewRequest("PATCH", "/api/v1/admin/animals/bulk", bytes.NewBuffer(jsonData))
	c.Request.Header.Set("Content-Type", "application/json")

	handler := BulkUpdateAnimals(db)
	handler(c)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	// Verify both updates applied
	var updatedAnimal models.Animal
	db.First(&updatedAnimal, animal.ID)

	if updatedAnimal.GroupID != group2.ID {
		t.Errorf("Expected animal to be in group %d, got group %d", group2.ID, updatedAnimal.GroupID)
	}

	if updatedAnimal.Status != "foster" {
		t.Errorf("Expected status 'foster', got '%s'", updatedAnimal.Status)
	}
}

// TestBulkUpdateAnimals_EmptyAnimalIDs tests validation for empty animal IDs
func TestBulkUpdateAnimals_EmptyAnimalIDs(t *testing.T) {
	db := setupAnimalTestDB(t)
	user, _ := createAnimalTestUser(t, db, "testuser", "test@example.com", true)

	newStatus := "foster"
	bulkReq := BulkUpdateAnimalsRequest{
		AnimalIDs: []uint{},
		Status:    &newStatus,
	}

	jsonData, _ := json.Marshal(bulkReq)

	c, w := setupAnimalTestContext(user.ID, true)
	c.Request = httptest.NewRequest("PATCH", "/api/v1/admin/animals/bulk", bytes.NewBuffer(jsonData))
	c.Request.Header.Set("Content-Type", "application/json")

	handler := BulkUpdateAnimals(db)
	handler(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	var response map[string]string
	json.Unmarshal(w.Body.Bytes(), &response)
	if response["error"] != "No animal IDs provided" {
		t.Errorf("Expected error 'No animal IDs provided', got '%s'", response["error"])
	}
}

// TestBulkUpdateAnimals_NoUpdates tests validation for no updates provided
func TestBulkUpdateAnimals_NoUpdates(t *testing.T) {
	db := setupAnimalTestDB(t)
	user, group := createAnimalTestUser(t, db, "testuser", "test@example.com", true)

	animal := createTestAnimal(t, db, group.ID, "Rex", "Dog")

	bulkReq := BulkUpdateAnimalsRequest{
		AnimalIDs: []uint{animal.ID},
		// No status or group_id provided
	}

	jsonData, _ := json.Marshal(bulkReq)

	c, w := setupAnimalTestContext(user.ID, true)
	c.Request = httptest.NewRequest("PATCH", "/api/v1/admin/animals/bulk", bytes.NewBuffer(jsonData))
	c.Request.Header.Set("Content-Type", "application/json")

	handler := BulkUpdateAnimals(db)
	handler(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	var response map[string]string
	json.Unmarshal(w.Body.Bytes(), &response)
	if response["error"] != "No updates provided" {
		t.Errorf("Expected error 'No updates provided', got '%s'", response["error"])
	}
}

// TestBulkUpdateAnimals_ValidationError tests validation on malformed request
func TestBulkUpdateAnimals_ValidationError(t *testing.T) {
	db := setupAnimalTestDB(t)
	user, _ := createAnimalTestUser(t, db, "testuser", "test@example.com", true)

	// Missing required animal_ids field
	jsonData := []byte(`{"status": "foster"}`)

	c, w := setupAnimalTestContext(user.ID, true)
	c.Request = httptest.NewRequest("PATCH", "/api/v1/admin/animals/bulk", bytes.NewBuffer(jsonData))
	c.Request.Header.Set("Content-Type", "application/json")

	handler := BulkUpdateAnimals(db)
	handler(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

// TestBulkUpdateAnimals_NonExistentAnimals tests bulk update with non-existent IDs
func TestBulkUpdateAnimals_NonExistentAnimals(t *testing.T) {
	db := setupAnimalTestDB(t)
	user, _ := createAnimalTestUser(t, db, "testuser", "test@example.com", true)

	newStatus := "foster"
	bulkReq := BulkUpdateAnimalsRequest{
		AnimalIDs: []uint{99999, 88888, 77777},
		Status:    &newStatus,
	}

	jsonData, _ := json.Marshal(bulkReq)

	c, w := setupAnimalTestContext(user.ID, true)
	c.Request = httptest.NewRequest("PATCH", "/api/v1/admin/animals/bulk", bytes.NewBuffer(jsonData))
	c.Request.Header.Set("Content-Type", "application/json")

	handler := BulkUpdateAnimals(db)
	handler(c)

	// The handler doesn't check if animals exist - it returns success even if no rows affected
	// This is acceptable behavior for bulk operations
	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
}

// TestBulkUpdateAnimals_PartialSuccess tests bulk update with mix of valid and invalid IDs
func TestBulkUpdateAnimals_PartialSuccess(t *testing.T) {
	db := setupAnimalTestDB(t)
	user, group := createAnimalTestUser(t, db, "testuser", "test@example.com", true)

	animal1 := createTestAnimal(t, db, group.ID, "Rex", "Dog")
	animal2 := createTestAnimal(t, db, group.ID, "Fluffy", "Cat")

	newStatus := "foster"
	bulkReq := BulkUpdateAnimalsRequest{
		AnimalIDs: []uint{animal1.ID, animal2.ID, 99999}, // Mix of valid and invalid
		Status:    &newStatus,
	}

	jsonData, _ := json.Marshal(bulkReq)

	c, w := setupAnimalTestContext(user.ID, true)
	c.Request = httptest.NewRequest("PATCH", "/api/v1/admin/animals/bulk", bytes.NewBuffer(jsonData))
	c.Request.Header.Set("Content-Type", "application/json")

	handler := BulkUpdateAnimals(db)
	handler(c)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Verify valid animals were updated
	var animal1Updated, animal2Updated models.Animal
	db.First(&animal1Updated, animal1.ID)
	db.First(&animal2Updated, animal2.ID)

	if animal1Updated.Status != "foster" {
		t.Errorf("Expected animal1 status 'foster', got '%s'", animal1Updated.Status)
	}

	if animal2Updated.Status != "foster" {
		t.Errorf("Expected animal2 status 'foster', got '%s'", animal2Updated.Status)
	}
}

// TestUpdateAnimal_EmptyQuarantineDateString tests that empty string for quarantine_start_date doesn't cause parsing error
// This reproduces the bug where frontend sends "" instead of null
func TestUpdateAnimal_EmptyQuarantineDateString(t *testing.T) {
	db := setupAnimalTestDB(t)
	user, group := createAnimalTestUser(t, db, "testuser", "test@example.com", false)

	animal := createTestAnimal(t, db, group.ID, "Daisy", "Dog")

	tests := []struct {
		name       string
		jsonBody   string
		wantStatus int
		wantError  bool
	}{
		{
			name:       "empty string for quarantine_start_date",
			jsonBody:   `{"name":"Daisy","species":"Dog","breed":"Border Collie","age":3,"description":"Smart dog","status":"foster","quarantine_start_date":""}`,
			wantStatus: http.StatusOK,
			wantError:  false,
		},
		{
			name:       "null for quarantine_start_date",
			jsonBody:   `{"name":"Daisy","species":"Dog","breed":"Border Collie","age":3,"description":"Smart dog","status":"available","quarantine_start_date":null}`,
			wantStatus: http.StatusOK,
			wantError:  false,
		},
		{
			name:       "omitted quarantine_start_date",
			jsonBody:   `{"name":"Daisy","species":"Dog","breed":"Border Collie","age":3,"description":"Smart dog","status":"available"}`,
			wantStatus: http.StatusOK,
			wantError:  false,
		},
		{
			name:       "valid quarantine_start_date with quarantine status",
			jsonBody:   `{"name":"Daisy","species":"Dog","breed":"Border Collie","age":3,"description":"Smart dog","status":"bite_quarantine","quarantine_start_date":"2024-01-15T10:00:00Z"}`,
			wantStatus: http.StatusOK,
			wantError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, w := setupAnimalTestContext(user.ID, false)
			c.Params = gin.Params{
				{Key: "id", Value: fmt.Sprintf("%d", group.ID)},
				{Key: "animalId", Value: fmt.Sprintf("%d", animal.ID)},
			}
			c.Request = httptest.NewRequest("PUT", fmt.Sprintf("/api/v1/groups/%d/animals/%d", group.ID, animal.ID), bytes.NewBufferString(tt.jsonBody))
			c.Request.Header.Set("Content-Type", "application/json")

			handler := UpdateAnimal(db)
			handler(c)

			if w.Code != tt.wantStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.wantStatus, w.Code, w.Body.String())
			}

			if tt.wantError && w.Code == http.StatusOK {
				t.Error("Expected an error but got success")
			}

			if !tt.wantError && w.Code != http.StatusOK {
				t.Errorf("Expected success but got error: %s", w.Body.String())
			}
		})
	}
}

// TestUpdateAnimal_NameHistory tests that name changes are tracked
func TestUpdateAnimal_NameHistory(t *testing.T) {
	db := setupAnimalTestDB(t)

	// Migrate AnimalNameHistory model
	if err := db.AutoMigrate(&models.AnimalNameHistory{}); err != nil {
		t.Fatalf("Failed to migrate AnimalNameHistory: %v", err)
	}

	user, group := createAnimalTestUser(t, db, "testuser", "test@example.com", false)
	animal := createTestAnimal(t, db, group.ID, "OriginalName", "Dog")

	// Update animal name
	updateReq := map[string]interface{}{
		"name":    "NewName",
		"species": "Dog",
		"breed":   "Labrador",
		"age":     3,
		"status":  "available",
	}
	body, _ := json.Marshal(updateReq)

	c, w := setupAnimalTestContext(user.ID, false)
	c.Params = gin.Params{
		{Key: "id", Value: fmt.Sprintf("%d", group.ID)},
		{Key: "animalId", Value: fmt.Sprintf("%d", animal.ID)},
	}
	c.Request = httptest.NewRequest("PUT", fmt.Sprintf("/api/v1/groups/%d/animals/%d", group.ID, animal.ID), bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler := UpdateAnimal(db)
	handler(c)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	// Check that name history was recorded
	var history []models.AnimalNameHistory
	if err := db.Where("animal_id = ?", animal.ID).Find(&history).Error; err != nil {
		t.Fatalf("Failed to query name history: %v", err)
	}

	if len(history) != 1 {
		t.Errorf("Expected 1 name history record, got %d", len(history))
	}

	if len(history) > 0 {
		if history[0].OldName != "OriginalName" {
			t.Errorf("Expected old name 'OriginalName', got '%s'", history[0].OldName)
		}
		if history[0].NewName != "NewName" {
			t.Errorf("Expected new name 'NewName', got '%s'", history[0].NewName)
		}
		if history[0].ChangedBy != user.ID {
			t.Errorf("Expected changed_by %d, got %d", user.ID, history[0].ChangedBy)
		}
	}
}

// TestUpdateAnimal_IsReturned tests the is_returned flag functionality
func TestUpdateAnimal_IsReturned(t *testing.T) {
	db := setupAnimalTestDB(t)
	user, group := createAnimalTestUser(t, db, "testuser", "test@example.com", false)
	animal := createTestAnimal(t, db, group.ID, "TestAnimal", "Dog")

	tests := []struct {
		name       string
		status     string
		isReturned *bool
		wantValue  bool
	}{
		{
			name:       "archived with is_returned true",
			status:     "archived",
			isReturned: boolPtr(true),
			wantValue:  true,
		},
		{
			name:       "archived with is_returned false",
			status:     "archived",
			isReturned: boolPtr(false),
			wantValue:  false,
		},
		{
			name:       "archived without is_returned (defaults to false)",
			status:     "archived",
			isReturned: nil,
			wantValue:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updateReq := map[string]interface{}{
				"name":    "TestAnimal",
				"species": "Dog",
				"breed":   "Mixed",
				"age":     3,
				"status":  tt.status,
			}
			if tt.isReturned != nil {
				updateReq["is_returned"] = *tt.isReturned
			}
			body, _ := json.Marshal(updateReq)

			c, w := setupAnimalTestContext(user.ID, false)
			c.Params = gin.Params{
				{Key: "id", Value: fmt.Sprintf("%d", group.ID)},
				{Key: "animalId", Value: fmt.Sprintf("%d", animal.ID)},
			}
			c.Request = httptest.NewRequest("PUT", fmt.Sprintf("/api/v1/groups/%d/animals/%d", group.ID, animal.ID), bytes.NewBuffer(body))
			c.Request.Header.Set("Content-Type", "application/json")

			handler := UpdateAnimal(db)
			handler(c)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
				return
			}

			// Verify is_returned value
			var updatedAnimal models.Animal
			if err := db.First(&updatedAnimal, animal.ID).Error; err != nil {
				t.Fatalf("Failed to query updated animal: %v", err)
			}

			if updatedAnimal.IsReturned != tt.wantValue {
				t.Errorf("Expected is_returned %v, got %v", tt.wantValue, updatedAnimal.IsReturned)
			}
		})
	}
}

// Helper function to create bool pointer
func boolPtr(b bool) *bool {
	return &b
}
