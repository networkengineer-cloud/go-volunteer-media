package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
)

// TestUpdateAnimalAdmin_Success tests successful admin update
func TestUpdateAnimalAdmin_Success(t *testing.T) {
	db := setupAnimalTestDB(t)
	user, group := createAnimalTestUser(t, db, "admin", "admin@example.com", true)

	animal := createTestAnimal(t, db, group.ID, "Rex", "Dog")

	updateReq := AnimalRequest{
		Name:        "Rex Updated",
		Species:     "Dog",
		Breed:       "Labrador",
		Age:         5,
		Description: "Updated by admin",
		Status:      "foster",
	}

	jsonData, _ := json.Marshal(updateReq)

	c, w := setupAnimalTestContext(user.ID, true)
	c.Params = gin.Params{{Key: "animalId", Value: fmt.Sprintf("%d", animal.ID)}}
	c.Request = httptest.NewRequest("PUT", fmt.Sprintf("/api/v1/admin/animals/%d", animal.ID), bytes.NewBuffer(jsonData))
	c.Request.Header.Set("Content-Type", "application/json")

	handler := UpdateAnimalAdmin(db)
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

	if updatedAnimal.Age != 5 {
		t.Errorf("Expected age 5, got %d", updatedAnimal.Age)
	}

	if updatedAnimal.Status != "foster" {
		t.Errorf("Expected status 'foster', got '%s'", updatedAnimal.Status)
	}
}

// TestUpdateAnimalAdmin_PartialUpdate tests partial field updates
func TestUpdateAnimalAdmin_PartialUpdate(t *testing.T) {
	db := setupAnimalTestDB(t)
	user, group := createAnimalTestUser(t, db, "admin", "admin@example.com", true)

	animal := createTestAnimal(t, db, group.ID, "Rex", "Dog")
	originalBreed := animal.Breed

	// Only update name
	updateReq := AnimalRequest{
		Name: "Rex Updated",
	}

	jsonData, _ := json.Marshal(updateReq)

	c, w := setupAnimalTestContext(user.ID, true)
	c.Params = gin.Params{{Key: "animalId", Value: fmt.Sprintf("%d", animal.ID)}}
	c.Request = httptest.NewRequest("PUT", fmt.Sprintf("/api/v1/admin/animals/%d", animal.ID), bytes.NewBuffer(jsonData))
	c.Request.Header.Set("Content-Type", "application/json")

	handler := UpdateAnimalAdmin(db)
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

	if updatedAnimal.Breed != originalBreed {
		t.Errorf("Expected breed to remain '%s', got '%s'", originalBreed, updatedAnimal.Breed)
	}
}

// TestUpdateAnimalAdmin_StatusTransition tests status change tracking
func TestUpdateAnimalAdmin_StatusTransition(t *testing.T) {
	db := setupAnimalTestDB(t)
	user, group := createAnimalTestUser(t, db, "admin", "admin@example.com", true)

	animal := createTestAnimal(t, db, group.ID, "Rex", "Dog")

	updateReq := AnimalRequest{
		Name:   "Rex",
		Status: "bite_quarantine",
	}

	jsonData, _ := json.Marshal(updateReq)

	c, w := setupAnimalTestContext(user.ID, true)
	c.Params = gin.Params{{Key: "animalId", Value: fmt.Sprintf("%d", animal.ID)}}
	c.Request = httptest.NewRequest("PUT", fmt.Sprintf("/api/v1/admin/animals/%d", animal.ID), bytes.NewBuffer(jsonData))
	c.Request.Header.Set("Content-Type", "application/json")

	handler := UpdateAnimalAdmin(db)
	handler(c)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var updatedAnimal models.Animal
	if err := json.Unmarshal(w.Body.Bytes(), &updatedAnimal); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if updatedAnimal.Status != "bite_quarantine" {
		t.Errorf("Expected status 'bite_quarantine', got '%s'", updatedAnimal.Status)
	}

	if updatedAnimal.QuarantineStartDate == nil {
		t.Error("Expected QuarantineStartDate to be set")
	}
}

// TestUpdateAnimalAdmin_MoveGroup tests moving animal to different group
func TestUpdateAnimalAdmin_MoveGroup(t *testing.T) {
	db := setupAnimalTestDB(t)
	user, group1 := createAnimalTestUser(t, db, "admin", "admin@example.com", true)

	// Create second group
	group2 := &models.Group{
		Name:        "Group 2",
		Description: "Test group 2",
	}
	db.Create(group2)

	animal := createTestAnimal(t, db, group1.ID, "Rex", "Dog")

	updateReq := AnimalRequest{
		Name:    "Rex",
		GroupID: group2.ID,
	}

	jsonData, _ := json.Marshal(updateReq)

	c, w := setupAnimalTestContext(user.ID, true)
	c.Params = gin.Params{{Key: "animalId", Value: fmt.Sprintf("%d", animal.ID)}}
	c.Request = httptest.NewRequest("PUT", fmt.Sprintf("/api/v1/admin/animals/%d", animal.ID), bytes.NewBuffer(jsonData))
	c.Request.Header.Set("Content-Type", "application/json")

	handler := UpdateAnimalAdmin(db)
	handler(c)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var updatedAnimal models.Animal
	if err := json.Unmarshal(w.Body.Bytes(), &updatedAnimal); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if updatedAnimal.GroupID != group2.ID {
		t.Errorf("Expected GroupID %d, got %d", group2.ID, updatedAnimal.GroupID)
	}
}

// TestUpdateAnimalAdmin_NotFound tests updating non-existent animal
func TestUpdateAnimalAdmin_NotFound(t *testing.T) {
	db := setupAnimalTestDB(t)
	user, _ := createAnimalTestUser(t, db, "admin", "admin@example.com", true)

	updateReq := AnimalRequest{
		Name: "Rex",
	}

	jsonData, _ := json.Marshal(updateReq)

	c, w := setupAnimalTestContext(user.ID, true)
	c.Params = gin.Params{{Key: "animalId", Value: "99999"}}
	c.Request = httptest.NewRequest("PUT", "/api/v1/admin/animals/99999", bytes.NewBuffer(jsonData))
	c.Request.Header.Set("Content-Type", "application/json")

	handler := UpdateAnimalAdmin(db)
	handler(c)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

// TestUpdateAnimalAdmin_NoUpdates tests request that doesn't change any values
func TestUpdateAnimalAdmin_NoUpdates(t *testing.T) {
	db := setupAnimalTestDB(t)
	user, group := createAnimalTestUser(t, db, "admin", "admin@example.com", true)

	animal := createTestAnimal(t, db, group.ID, "Rex", "Dog")

	// Request with Name (required) but same value - technically an update even though value doesn't change
	// The UpdateAnimalAdmin handler treats any non-zero field as an update
	updateReq := AnimalRequest{
		Name: "Rex", // Same name
	}

	jsonData, _ := json.Marshal(updateReq)

	c, w := setupAnimalTestContext(user.ID, true)
	c.Params = gin.Params{{Key: "animalId", Value: fmt.Sprintf("%d", animal.ID)}}
	c.Request = httptest.NewRequest("PUT", fmt.Sprintf("/api/v1/admin/animals/%d", animal.ID), bytes.NewBuffer(jsonData))
	c.Request.Header.Set("Content-Type", "application/json")

	handler := UpdateAnimalAdmin(db)
	handler(c)

	// The handler will accept this as an update (even though the value is the same)
	// This is acceptable behavior - it updates the field with the same value
	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
	}
}

// TestUpdateAnimalAdmin_ValidationError tests validation errors
func TestUpdateAnimalAdmin_ValidationError(t *testing.T) {
	db := setupAnimalTestDB(t)
	user, group := createAnimalTestUser(t, db, "admin", "admin@example.com", true)

	animal := createTestAnimal(t, db, group.ID, "Rex", "Dog")

	// Empty request (missing required Name field)
	updateReq := AnimalRequest{}

	jsonData, _ := json.Marshal(updateReq)

	c, w := setupAnimalTestContext(user.ID, true)
	c.Params = gin.Params{{Key: "animalId", Value: fmt.Sprintf("%d", animal.ID)}}
	c.Request = httptest.NewRequest("PUT", fmt.Sprintf("/api/v1/admin/animals/%d", animal.ID), bytes.NewBuffer(jsonData))
	c.Request.Header.Set("Content-Type", "application/json")

	handler := UpdateAnimalAdmin(db)
	handler(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	var response map[string]string
	json.Unmarshal(w.Body.Bytes(), &response)
	if !strings.Contains(response["error"], "Name") {
		t.Errorf("Expected error about Name field, got '%s'", response["error"])
	}
}

// TestGetAllAnimals_Success tests successful retrieval of all animals
func TestGetAllAnimals_Success(t *testing.T) {
	db := setupAnimalTestDB(t)
	user, group1 := createAnimalTestUser(t, db, "admin", "admin@example.com", true)

	// Create second group
	group2 := &models.Group{
		Name:        "Group 2",
		Description: "Test group 2",
	}
	db.Create(group2)

	// Create animals in both groups
	createTestAnimal(t, db, group1.ID, "Rex", "Dog")
	createTestAnimal(t, db, group1.ID, "Fluffy", "Cat")
	createTestAnimal(t, db, group2.ID, "Max", "Dog")

	c, w := setupAnimalTestContext(user.ID, true)
	c.Request = httptest.NewRequest("GET", "/api/v1/admin/animals", nil)

	handler := GetAllAnimals(db)
	handler(c)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var animals []models.Animal
	if err := json.Unmarshal(w.Body.Bytes(), &animals); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if len(animals) != 3 {
		t.Errorf("Expected 3 animals, got %d", len(animals))
	}
}

// TestGetAllAnimals_WithFilters tests filtering all animals
func TestGetAllAnimals_WithFilters(t *testing.T) {
	db := setupAnimalTestDB(t)
	user, group1 := createAnimalTestUser(t, db, "admin", "admin@example.com", true)

	// Create animals with different statuses
	animal1 := createTestAnimal(t, db, group1.ID, "Rex", "Dog")
	animal1.Status = "available"
	db.Save(animal1)

	animal2 := createTestAnimal(t, db, group1.ID, "Fluffy", "Cat")
	animal2.Status = "foster"
	db.Save(animal2)

	tests := []struct {
		name          string
		query         string
		expectedCount int
	}{
		{
			name:          "filter by status available",
			query:         "?status=available",
			expectedCount: 1,
		},
		{
			name:          "filter by status foster",
			query:         "?status=foster",
			expectedCount: 1,
		},
		{
			name:          "filter by group",
			query:         fmt.Sprintf("?group_id=%d", group1.ID),
			expectedCount: 2,
		},
		{
			name:          "filter by name",
			query:         "?name=rex",
			expectedCount: 1,
		},
		{
			name:          "all animals",
			query:         "",
			expectedCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, w := setupAnimalTestContext(user.ID, true)
			c.Request = httptest.NewRequest("GET", "/api/v1/admin/animals"+tt.query, nil)

			handler := GetAllAnimals(db)
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

// TestGetAllAnimals_OrderedByGroupAndName tests ordering
func TestGetAllAnimals_OrderedByGroupAndName(t *testing.T) {
	db := setupAnimalTestDB(t)
	user, group1 := createAnimalTestUser(t, db, "admin", "admin@example.com", true)

	// Create second group
	group2 := &models.Group{
		Name:        "Group 2",
		Description: "Test group 2",
	}
	db.Create(group2)

	// Create animals in specific order to test sorting
	createTestAnimal(t, db, group2.ID, "Zebra", "Dog")
	createTestAnimal(t, db, group1.ID, "Rex", "Dog")
	createTestAnimal(t, db, group1.ID, "Alpha", "Cat")

	c, w := setupAnimalTestContext(user.ID, true)
	c.Request = httptest.NewRequest("GET", "/api/v1/admin/animals", nil)

	handler := GetAllAnimals(db)
	handler(c)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var animals []models.Animal
	if err := json.Unmarshal(w.Body.Bytes(), &animals); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if len(animals) != 3 {
		t.Fatalf("Expected 3 animals, got %d", len(animals))
	}

	// Check ordering: should be by group_id, then by name
	// Group 1 animals should come first (Alpha, Rex), then Group 2 (Zebra)
	if animals[0].Name != "Alpha" {
		t.Errorf("Expected first animal to be 'Alpha', got '%s'", animals[0].Name)
	}

	if animals[1].Name != "Rex" {
		t.Errorf("Expected second animal to be 'Rex', got '%s'", animals[1].Name)
	}

	if animals[2].Name != "Zebra" {
		t.Errorf("Expected third animal to be 'Zebra', got '%s'", animals[2].Name)
	}
}
