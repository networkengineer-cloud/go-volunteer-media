package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/embedding"
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

	handler := UpdateAnimalAdmin(db, nil, &embedding.StubEmbedder{})
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

	handler := UpdateAnimalAdmin(db, nil, &embedding.StubEmbedder{})
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

	handler := UpdateAnimalAdmin(db, nil, &embedding.StubEmbedder{})
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

// TestUpdateAnimalAdmin_EnterQuarantine_StoresIncidentDetails verifies that
// entering bite_quarantine via the admin handler stores the incident details.
func TestUpdateAnimalAdmin_EnterQuarantine_StoresIncidentDetails(t *testing.T) {
	db := setupAnimalTestDB(t)
	user, group := createAnimalTestUser(t, db, "admin", "admin@example.com", true)

	animal := createTestAnimal(t, db, group.ID, "Rex", "Dog")

	details := "Bit a volunteer."
	updateReq := AnimalRequest{
		Name:                      "Rex",
		Status:                    "bite_quarantine",
		QuarantineIncidentDetails: &details,
	}

	jsonData, _ := json.Marshal(updateReq)

	c, w := setupAnimalTestContext(user.ID, true)
	c.Params = gin.Params{{Key: "animalId", Value: fmt.Sprintf("%d", animal.ID)}}
	c.Request = httptest.NewRequest("PUT", fmt.Sprintf("/api/v1/admin/animals/%d", animal.ID), bytes.NewBuffer(jsonData))
	c.Request.Header.Set("Content-Type", "application/json")

	handler := UpdateAnimalAdmin(db, nil, &embedding.StubEmbedder{})
	handler(c)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var got models.Animal
	if err := db.First(&got, animal.ID).Error; err != nil {
		t.Fatalf("Failed to reload animal: %v", err)
	}
	if got.QuarantineIncidentDetails != "Bit a volunteer." {
		t.Errorf("incident not stored: %q", got.QuarantineIncidentDetails)
	}
}

// TestUpdateAnimalAdmin_LeaveQuarantine_ClearsIncidentDetails verifies that
// leaving bite_quarantine via the admin handler clears the stored incident details.
func TestUpdateAnimalAdmin_LeaveQuarantine_ClearsIncidentDetails(t *testing.T) {
	db := setupAnimalTestDB(t)
	user, group := createAnimalTestUser(t, db, "admin", "admin@example.com", true)

	animal := createTestAnimal(t, db, group.ID, "Rex", "Dog")

	// Seed an animal already in BQ with details
	if err := db.Model(animal).Updates(map[string]interface{}{
		"status":                      "bite_quarantine",
		"quarantine_incident_details": "Bit a volunteer.",
	}).Error; err != nil {
		t.Fatalf("Failed to seed animal into quarantine: %v", err)
	}

	updateReq := AnimalRequest{
		Name:   "Rex",
		Status: "available",
	}

	jsonData, _ := json.Marshal(updateReq)

	c, w := setupAnimalTestContext(user.ID, true)
	c.Params = gin.Params{{Key: "animalId", Value: fmt.Sprintf("%d", animal.ID)}}
	c.Request = httptest.NewRequest("PUT", fmt.Sprintf("/api/v1/admin/animals/%d", animal.ID), bytes.NewBuffer(jsonData))
	c.Request.Header.Set("Content-Type", "application/json")

	handler := UpdateAnimalAdmin(db, nil, &embedding.StubEmbedder{})
	handler(c)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var got models.Animal
	if err := db.First(&got, animal.ID).Error; err != nil {
		t.Fatalf("Failed to reload animal: %v", err)
	}
	if got.QuarantineIncidentDetails != "" {
		t.Errorf("incident not cleared on leaving BQ: %q", got.QuarantineIncidentDetails)
	}
}

// TestUpdateAnimalAdmin_UnderVetCareTransition tests transitioning to under_vet_care clears other status fields
func TestUpdateAnimalAdmin_UnderVetCareTransition(t *testing.T) {
	db := setupAnimalTestDB(t)
	user, group := createAnimalTestUser(t, db, "admin", "admin@example.com", true)

	animal := createTestAnimal(t, db, group.ID, "Rex", "Dog")
	now := time.Now()
	animal.Status = "archived"
	animal.ArchivedDate = &now
	db.Save(animal)

	updateReq := AnimalRequest{
		Name:   "Rex",
		Status: "under_vet_care",
	}

	jsonData, _ := json.Marshal(updateReq)

	c, w := setupAnimalTestContext(user.ID, true)
	c.Params = gin.Params{{Key: "animalId", Value: fmt.Sprintf("%d", animal.ID)}}
	c.Request = httptest.NewRequest("PUT", fmt.Sprintf("/api/v1/admin/animals/%d", animal.ID), bytes.NewBuffer(jsonData))
	c.Request.Header.Set("Content-Type", "application/json")

	handler := UpdateAnimalAdmin(db, nil, &embedding.StubEmbedder{})
	handler(c)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var updatedAnimal models.Animal
	if err := json.Unmarshal(w.Body.Bytes(), &updatedAnimal); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if updatedAnimal.Status != "under_vet_care" {
		t.Errorf("Expected status 'under_vet_care', got '%s'", updatedAnimal.Status)
	}

	if updatedAnimal.ArchivedDate != nil {
		t.Error("Expected ArchivedDate to be cleared when transitioning to under_vet_care")
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

	handler := UpdateAnimalAdmin(db, nil, &embedding.StubEmbedder{})
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

	handler := UpdateAnimalAdmin(db, nil, &embedding.StubEmbedder{})
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

	handler := UpdateAnimalAdmin(db, nil, &embedding.StubEmbedder{})
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

	handler := UpdateAnimalAdmin(db, nil, &embedding.StubEmbedder{})
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

// --- Quarantine approval status admin tests ---

// TestUpdateAnimalAdmin_ApprovalStatusSet verifies admin path sets approval status and stamps the date.
func TestUpdateAnimalAdmin_ApprovalStatusSet(t *testing.T) {
	db := setupAnimalTestDB(t)
	user, group := createAnimalTestUser(t, db, "adminapproval1", "adminapproval1@example.com", true)

	animal := createTestAnimal(t, db, group.ID, "Gus", "Dog")
	animal.Status = "bite_quarantine"
	db.Save(animal)

	newStatus := "granted"
	req := AnimalRequest{
		Name:                     "Gus",
		Status:                   "bite_quarantine",
		QuarantineApprovalStatus: &newStatus,
	}
	body, _ := json.Marshal(req)

	c, w := setupAnimalTestContext(user.ID, true)
	c.Params = gin.Params{{Key: "animalId", Value: fmt.Sprintf("%d", animal.ID)}}
	c.Request = httptest.NewRequest("PUT", "/", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")

	UpdateAnimalAdmin(db, nil, &embedding.StubEmbedder{})(c)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var updated models.Animal
	db.First(&updated, animal.ID)
	if updated.QuarantineApprovalStatus != "granted" {
		t.Errorf("Expected approval_status 'granted', got %q", updated.QuarantineApprovalStatus)
	}
	if updated.QuarantineApprovalDate == nil {
		t.Error("Expected approval_date to be set, got nil")
	}
}

// TestUpdateAnimalAdmin_ApprovalClearedOnTransitionToAvailable verifies admin path clears approval on exit.
func TestUpdateAnimalAdmin_ApprovalClearedOnTransitionToAvailable(t *testing.T) {
	db := setupAnimalTestDB(t)
	user, group := createAnimalTestUser(t, db, "adminapproval2", "adminapproval2@example.com", true)

	animal := createTestAnimal(t, db, group.ID, "Hank", "Cat")
	animal.Status = "bite_quarantine"
	animal.QuarantineApprovalStatus = "requested"
	db.Save(animal)

	req := AnimalRequest{
		Name:   "Hank",
		Status: "available",
	}
	body, _ := json.Marshal(req)

	c, w := setupAnimalTestContext(user.ID, true)
	c.Params = gin.Params{{Key: "animalId", Value: fmt.Sprintf("%d", animal.ID)}}
	c.Request = httptest.NewRequest("PUT", "/", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")

	UpdateAnimalAdmin(db, nil, &embedding.StubEmbedder{})(c)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var updated models.Animal
	db.First(&updated, animal.ID)
	if updated.QuarantineApprovalStatus != "" {
		t.Errorf("Expected approval_status '', got %q", updated.QuarantineApprovalStatus)
	}
	if updated.QuarantineApprovalDate != nil {
		t.Error("Expected approval_date to be nil after leaving quarantine, got non-nil")
	}
}

func TestUpdateAnimalAdmin_BiteQuarantine_DefaultEndDate(t *testing.T) {
	db := setupAnimalTestDB(t)
	user, group := createAnimalTestUser(t, db, "admin", "admin@example.com", true)
	animal := createTestAnimal(t, db, group.ID, "Rex", "Dog")

	startDate := time.Date(2025, 11, 3, 0, 0, 0, 0, time.UTC) // Monday
	updateReq := AnimalRequest{
		Name:   "Rex",
		Status: "bite_quarantine",
		QuarantineStartDate: NullableTime{
			Time:  &startDate,
			Valid: true,
		},
	}
	jsonData, _ := json.Marshal(updateReq)

	c, w := setupAnimalTestContext(user.ID, true)
	c.Params = gin.Params{{Key: "animalId", Value: fmt.Sprintf("%d", animal.ID)}}
	c.Request = httptest.NewRequest("PUT", fmt.Sprintf("/api/v1/admin/animals/%d", animal.ID), bytes.NewBuffer(jsonData))
	c.Request.Header.Set("Content-Type", "application/json")

	handler := UpdateAnimalAdmin(db, nil, &embedding.StubEmbedder{})
	handler(c)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var got models.Animal
	if err := db.First(&got, animal.ID).Error; err != nil {
		t.Fatalf("Failed to reload animal: %v", err)
	}
	expectedEnd := time.Date(2025, 11, 13, 0, 0, 0, 0, time.UTC)
	if got.QuarantineEndDate == nil || !got.QuarantineEndDate.Equal(expectedEnd) {
		t.Errorf("Expected QuarantineEndDate %v, got %v", expectedEnd, got.QuarantineEndDate)
	}
}

func TestUpdateAnimalAdmin_BiteQuarantine_EndDateBeforeStartDate(t *testing.T) {
	db := setupAnimalTestDB(t)
	user, group := createAnimalTestUser(t, db, "admin", "admin@example.com", true)
	animal := createTestAnimal(t, db, group.ID, "Rex", "Dog")

	startDate := time.Date(2025, 11, 10, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2025, 11, 5, 0, 0, 0, 0, time.UTC)
	updateReq := AnimalRequest{
		Name:   "Rex",
		Status: "bite_quarantine",
		QuarantineStartDate: NullableTime{
			Time:  &startDate,
			Valid: true,
		},
		QuarantineEndDate: NullableTime{
			Time:  &endDate,
			Valid: true,
		},
	}
	jsonData, _ := json.Marshal(updateReq)

	c, w := setupAnimalTestContext(user.ID, true)
	c.Params = gin.Params{{Key: "animalId", Value: fmt.Sprintf("%d", animal.ID)}}
	c.Request = httptest.NewRequest("PUT", fmt.Sprintf("/api/v1/admin/animals/%d", animal.ID), bytes.NewBuffer(jsonData))
	c.Request.Header.Set("Content-Type", "application/json")

	handler := UpdateAnimalAdmin(db, nil, &embedding.StubEmbedder{})
	handler(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestUpdateAnimalAdmin_EditEndDateOnly_WhileInQuarantine(t *testing.T) {
	db := setupAnimalTestDB(t)
	user, group := createAnimalTestUser(t, db, "admin", "admin@example.com", true)
	animal := createTestAnimal(t, db, group.ID, "Rex", "Dog")

	startDate := time.Date(2025, 11, 3, 0, 0, 0, 0, time.UTC)
	defaultEnd := time.Date(2025, 11, 13, 0, 0, 0, 0, time.UTC)
	if err := db.Model(animal).Updates(map[string]interface{}{
		"status":                "bite_quarantine",
		"quarantine_start_date": startDate,
		"quarantine_end_date":   defaultEnd,
	}).Error; err != nil {
		t.Fatalf("Failed to seed animal into quarantine: %v", err)
	}

	overrideEnd := time.Date(2025, 11, 24, 0, 0, 0, 0, time.UTC)
	updateReq := AnimalRequest{
		Name:   "Rex",
		Status: "bite_quarantine",
		QuarantineEndDate: NullableTime{
			Time:  &overrideEnd,
			Valid: true,
		},
	}
	jsonData, _ := json.Marshal(updateReq)

	c, w := setupAnimalTestContext(user.ID, true)
	c.Params = gin.Params{{Key: "animalId", Value: fmt.Sprintf("%d", animal.ID)}}
	c.Request = httptest.NewRequest("PUT", fmt.Sprintf("/api/v1/admin/animals/%d", animal.ID), bytes.NewBuffer(jsonData))
	c.Request.Header.Set("Content-Type", "application/json")

	handler := UpdateAnimalAdmin(db, nil, &embedding.StubEmbedder{})
	handler(c)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var got models.Animal
	if err := db.First(&got, animal.ID).Error; err != nil {
		t.Fatalf("Failed to reload animal: %v", err)
	}
	if got.QuarantineEndDate == nil || !got.QuarantineEndDate.Equal(overrideEnd) {
		t.Errorf("Expected QuarantineEndDate override %v, got %v", overrideEnd, got.QuarantineEndDate)
	}
	if got.QuarantineStartDate == nil || !got.QuarantineStartDate.Equal(startDate) {
		t.Errorf("Expected QuarantineStartDate to remain %v, got %v", startDate, got.QuarantineStartDate)
	}
}

func TestUpdateAnimalAdmin_LeaveQuarantine_ClearsEndDate(t *testing.T) {
	db := setupAnimalTestDB(t)
	user, group := createAnimalTestUser(t, db, "admin", "admin@example.com", true)
	animal := createTestAnimal(t, db, group.ID, "Rex", "Dog")

	startDate := time.Date(2025, 11, 3, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2025, 11, 13, 0, 0, 0, 0, time.UTC)
	if err := db.Model(animal).Updates(map[string]interface{}{
		"status":                "bite_quarantine",
		"quarantine_start_date": startDate,
		"quarantine_end_date":   endDate,
	}).Error; err != nil {
		t.Fatalf("Failed to seed animal into quarantine: %v", err)
	}

	updateReq := AnimalRequest{
		Name:   "Rex",
		Status: "available",
	}
	jsonData, _ := json.Marshal(updateReq)

	c, w := setupAnimalTestContext(user.ID, true)
	c.Params = gin.Params{{Key: "animalId", Value: fmt.Sprintf("%d", animal.ID)}}
	c.Request = httptest.NewRequest("PUT", fmt.Sprintf("/api/v1/admin/animals/%d", animal.ID), bytes.NewBuffer(jsonData))
	c.Request.Header.Set("Content-Type", "application/json")

	handler := UpdateAnimalAdmin(db, nil, &embedding.StubEmbedder{})
	handler(c)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var got models.Animal
	if err := db.First(&got, animal.ID).Error; err != nil {
		t.Fatalf("Failed to reload animal: %v", err)
	}
	if got.QuarantineEndDate != nil {
		t.Errorf("Expected QuarantineEndDate to be cleared, got %v", got.QuarantineEndDate)
	}
}

// TestUpdateAnimalAdmin_BQEntry_CreatesIncidentRow verifies that the admin
// handler creates an AnimalBQIncident row when transitioning to bite_quarantine.
func TestUpdateAnimalAdmin_BQEntry_CreatesIncidentRow(t *testing.T) {
	db := setupAnimalTestDB(t)
	user, group := createAnimalTestUser(t, db, "admin", "admin@example.com", true)
	animal := createTestAnimal(t, db, group.ID, "Rex", "Dog")

	details := "Bit a volunteer."
	updateReq := AnimalRequest{
		Name:                      "Rex",
		Status:                    "bite_quarantine",
		QuarantineIncidentDetails: &details,
	}
	jsonData, _ := json.Marshal(updateReq)

	c, w := setupAnimalTestContext(user.ID, true)
	c.Params = gin.Params{{Key: "animalId", Value: fmt.Sprintf("%d", animal.ID)}}
	c.Request = httptest.NewRequest("PUT", fmt.Sprintf("/api/v1/admin/animals/%d", animal.ID), bytes.NewBuffer(jsonData))
	c.Request.Header.Set("Content-Type", "application/json")

	handler := UpdateAnimalAdmin(db, nil, &embedding.StubEmbedder{})
	handler(c)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	var incident models.AnimalBQIncident
	if err := db.Where("animal_id = ? AND end_date IS NULL", animal.ID).First(&incident).Error; err != nil {
		t.Fatalf("Expected active AnimalBQIncident row: %v", err)
	}
	if incident.IncidentDetails != "Bit a volunteer." {
		t.Errorf("IncidentDetails = %q, want %q", incident.IncidentDetails, "Bit a volunteer.")
	}
}

// TestUpdateAnimalAdmin_BQExit_StampsEndDate verifies that the admin handler
// stamps EndDate on the active incident row when the animal leaves BQ.
func TestUpdateAnimalAdmin_BQExit_StampsEndDate(t *testing.T) {
	db := setupAnimalTestDB(t)
	user, group := createAnimalTestUser(t, db, "admin", "admin@example.com", true)
	animal := createTestAnimal(t, db, group.ID, "Rex", "Dog")

	if err := db.Model(animal).Updates(map[string]interface{}{
		"status":                      "bite_quarantine",
		"quarantine_incident_details": "Bit a volunteer.",
	}).Error; err != nil {
		t.Fatalf("seed BQ status: %v", err)
	}
	if err := db.Create(&models.AnimalBQIncident{
		AnimalID:        animal.ID,
		IncidentDetails: "Bit a volunteer.",
		StartDate:       animal.CreatedAt,
	}).Error; err != nil {
		t.Fatalf("seed incident row: %v", err)
	}

	updateReq := AnimalRequest{Name: "Rex", Status: "available"}
	jsonData, _ := json.Marshal(updateReq)

	c, w := setupAnimalTestContext(user.ID, true)
	c.Params = gin.Params{{Key: "animalId", Value: fmt.Sprintf("%d", animal.ID)}}
	c.Request = httptest.NewRequest("PUT", fmt.Sprintf("/api/v1/admin/animals/%d", animal.ID), bytes.NewBuffer(jsonData))
	c.Request.Header.Set("Content-Type", "application/json")

	handler := UpdateAnimalAdmin(db, nil, &embedding.StubEmbedder{})
	handler(c)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	var incident models.AnimalBQIncident
	if err := db.Where("animal_id = ?", animal.ID).First(&incident).Error; err != nil {
		t.Fatalf("reload incident row: %v", err)
	}
	if incident.EndDate == nil {
		t.Error("Expected EndDate to be stamped, got nil")
	}
}

// TestUpdateAnimalAdmin_BQExit_ExplicitEndDate_UsesProvidedValue verifies that when
// the request provides an explicit quarantine_end_date while leaving bite_quarantine
// (the modal-confirmed path), it's used to close the incident verbatim instead of
// the animal's previously stored end date.
func TestUpdateAnimalAdmin_BQExit_ExplicitEndDate_UsesProvidedValue(t *testing.T) {
	db := setupAnimalTestDB(t)
	user, group := createAnimalTestUser(t, db, "admin", "admin@example.com", true)
	animal := createTestAnimal(t, db, group.ID, "Rex", "Dog")

	startDate := time.Date(2025, 11, 3, 0, 0, 0, 0, time.UTC)
	storedEndDate := time.Date(2025, 11, 13, 0, 0, 0, 0, time.UTC)
	confirmedEndDate := time.Date(2025, 11, 6, 0, 0, 0, 0, time.UTC) // vet-cleared early

	if err := db.Model(animal).Updates(map[string]interface{}{
		"status":                "bite_quarantine",
		"quarantine_start_date": startDate,
		"quarantine_end_date":   storedEndDate,
	}).Error; err != nil {
		t.Fatalf("seed BQ status: %v", err)
	}
	if err := db.Create(&models.AnimalBQIncident{
		AnimalID:  animal.ID,
		StartDate: startDate,
	}).Error; err != nil {
		t.Fatalf("seed incident row: %v", err)
	}

	updateReq := AnimalRequest{
		Name:   "Rex",
		Status: "available",
		QuarantineEndDate: NullableTime{
			Time:  &confirmedEndDate,
			Valid: true,
		},
	}
	jsonData, _ := json.Marshal(updateReq)

	c, w := setupAnimalTestContext(user.ID, true)
	c.Params = gin.Params{{Key: "animalId", Value: fmt.Sprintf("%d", animal.ID)}}
	c.Request = httptest.NewRequest("PUT", fmt.Sprintf("/api/v1/admin/animals/%d", animal.ID), bytes.NewBuffer(jsonData))
	c.Request.Header.Set("Content-Type", "application/json")

	handler := UpdateAnimalAdmin(db, nil, &embedding.StubEmbedder{})
	handler(c)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	var incident models.AnimalBQIncident
	if err := db.Where("animal_id = ?", animal.ID).First(&incident).Error; err != nil {
		t.Fatalf("reload incident row: %v", err)
	}
	if incident.EndDate == nil || !incident.EndDate.Equal(confirmedEndDate) {
		t.Errorf("Expected EndDate %v, got %v", confirmedEndDate, incident.EndDate)
	}
}

// TestUpdateAnimalAdmin_BQExit_ExplicitEndDate_NoStoredStartDate_Succeeds verifies
// that an animal which reached bite_quarantine status without a QuarantineStartDate
// on record can still leave bite_quarantine when the exit modal sends an explicit
// confirmed end date — there's no start date to validate against, so the exit must
// not be blocked. Deliberately seeds no AnimalBQIncident row: a real CSV-imported
// animal (the actual motivating case for a nil start date — see
// animal_import_export.go, which sets status directly without ever creating an
// incident row) has none, so this asserts the request succeeds even when there's no
// incident to close, not just that a pre-existing incident's EndDate resolves
// correctly.
func TestUpdateAnimalAdmin_BQExit_ExplicitEndDate_NoStoredStartDate_Succeeds(t *testing.T) {
	db := setupAnimalTestDB(t)
	user, group := createAnimalTestUser(t, db, "admin", "admin@example.com", true)
	animal := createTestAnimal(t, db, group.ID, "Rex", "Dog")

	confirmedEndDate := time.Date(2025, 11, 6, 0, 0, 0, 0, time.UTC)

	if err := db.Model(animal).Updates(map[string]interface{}{
		"status": "bite_quarantine",
		// quarantine_start_date intentionally left unset, matching an animal
		// imported via CSV directly into bite_quarantine status.
	}).Error; err != nil {
		t.Fatalf("seed BQ status: %v", err)
	}

	updateReq := AnimalRequest{
		Name:   "Rex",
		Status: "available",
		QuarantineEndDate: NullableTime{
			Time:  &confirmedEndDate,
			Valid: true,
		},
	}
	jsonData, _ := json.Marshal(updateReq)

	c, w := setupAnimalTestContext(user.ID, true)
	c.Params = gin.Params{{Key: "animalId", Value: fmt.Sprintf("%d", animal.ID)}}
	c.Request = httptest.NewRequest("PUT", fmt.Sprintf("/api/v1/admin/animals/%d", animal.ID), bytes.NewBuffer(jsonData))
	c.Request.Header.Set("Content-Type", "application/json")

	handler := UpdateAnimalAdmin(db, nil, &embedding.StubEmbedder{})
	handler(c)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	var reloaded models.Animal
	if err := db.First(&reloaded, animal.ID).Error; err != nil {
		t.Fatalf("reload animal: %v", err)
	}
	if reloaded.Status != "available" {
		t.Errorf("Expected status to change to available, got %q", reloaded.Status)
	}
}

// TestUpdateAnimalAdmin_BQExit_InvalidExplicitEndDate_RejectsBeforeSaving verifies
// that an explicit exit end date before the quarantine start date is rejected with a
// 400 and that the animal's status is NOT changed — validation must happen before
// the animal record is updated, not after.
func TestUpdateAnimalAdmin_BQExit_InvalidExplicitEndDate_RejectsBeforeSaving(t *testing.T) {
	db := setupAnimalTestDB(t)
	user, group := createAnimalTestUser(t, db, "admin", "admin@example.com", true)
	animal := createTestAnimal(t, db, group.ID, "Rex", "Dog")

	startDate := time.Date(2025, 11, 3, 0, 0, 0, 0, time.UTC)
	storedEndDate := time.Date(2025, 11, 13, 0, 0, 0, 0, time.UTC)
	invalidEndDate := time.Date(2025, 11, 1, 0, 0, 0, 0, time.UTC) // before start date

	if err := db.Model(animal).Updates(map[string]interface{}{
		"status":                "bite_quarantine",
		"quarantine_start_date": startDate,
		"quarantine_end_date":   storedEndDate,
	}).Error; err != nil {
		t.Fatalf("seed BQ status: %v", err)
	}
	if err := db.Create(&models.AnimalBQIncident{
		AnimalID:  animal.ID,
		StartDate: startDate,
	}).Error; err != nil {
		t.Fatalf("seed incident row: %v", err)
	}

	updateReq := AnimalRequest{
		Name:   "Rex",
		Status: "available",
		QuarantineEndDate: NullableTime{
			Time:  &invalidEndDate,
			Valid: true,
		},
	}
	jsonData, _ := json.Marshal(updateReq)

	c, w := setupAnimalTestContext(user.ID, true)
	c.Params = gin.Params{{Key: "animalId", Value: fmt.Sprintf("%d", animal.ID)}}
	c.Request = httptest.NewRequest("PUT", fmt.Sprintf("/api/v1/admin/animals/%d", animal.ID), bytes.NewBuffer(jsonData))
	c.Request.Header.Set("Content-Type", "application/json")

	handler := UpdateAnimalAdmin(db, nil, &embedding.StubEmbedder{})
	handler(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("Expected 400, got %d. Body: %s", w.Code, w.Body.String())
	}

	var reloaded models.Animal
	if err := db.First(&reloaded, animal.ID).Error; err != nil {
		t.Fatalf("reload animal: %v", err)
	}
	if reloaded.Status != "bite_quarantine" {
		t.Errorf("Expected status to remain bite_quarantine after a rejected request, got %q", reloaded.Status)
	}

	var incident models.AnimalBQIncident
	if err := db.Where("animal_id = ?", animal.ID).First(&incident).Error; err != nil {
		t.Fatalf("reload incident row: %v", err)
	}
	if incident.EndDate != nil {
		t.Errorf("Expected incident to remain open after a rejected request, got EndDate %v", incident.EndDate)
	}
}

// TestUpdateAnimalAdmin_LeaveQuarantine_EarlyExit_CapsEndDateAtNow verifies that when
// an animal leaves bite_quarantine before its stored end date has arrived (no
// explicit end date provided), the closed incident is stamped with now, not the
// future stored date.
func TestUpdateAnimalAdmin_LeaveQuarantine_EarlyExit_CapsEndDateAtNow(t *testing.T) {
	db := setupAnimalTestDB(t)
	user, group := createAnimalTestUser(t, db, "admin", "admin@example.com", true)
	animal := createTestAnimal(t, db, group.ID, "Rex", "Dog")

	startDate := time.Now().Add(-2 * 24 * time.Hour)
	futureEndDate := time.Now().Add(7 * 24 * time.Hour) // stored end date still a week out
	if err := db.Model(animal).Updates(map[string]interface{}{
		"status":                "bite_quarantine",
		"quarantine_start_date": startDate,
		"quarantine_end_date":   futureEndDate,
	}).Error; err != nil {
		t.Fatalf("Failed to seed animal into quarantine: %v", err)
	}
	if err := db.Create(&models.AnimalBQIncident{
		AnimalID:  animal.ID,
		StartDate: startDate,
	}).Error; err != nil {
		t.Fatalf("seed incident row: %v", err)
	}

	beforeRequest := time.Now()
	updateReq := AnimalRequest{Name: "Rex", Status: "available"}
	jsonData, _ := json.Marshal(updateReq)

	c, w := setupAnimalTestContext(user.ID, true)
	c.Params = gin.Params{{Key: "animalId", Value: fmt.Sprintf("%d", animal.ID)}}
	c.Request = httptest.NewRequest("PUT", fmt.Sprintf("/api/v1/admin/animals/%d", animal.ID), bytes.NewBuffer(jsonData))
	c.Request.Header.Set("Content-Type", "application/json")

	handler := UpdateAnimalAdmin(db, nil, &embedding.StubEmbedder{})
	handler(c)
	afterRequest := time.Now()

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	var incident models.AnimalBQIncident
	if err := db.Where("animal_id = ?", animal.ID).First(&incident).Error; err != nil {
		t.Fatalf("reload incident row: %v", err)
	}
	if incident.EndDate == nil {
		t.Fatal("Expected EndDate to be stamped, got nil")
	}
	if incident.EndDate.Before(beforeRequest) || incident.EndDate.After(afterRequest) {
		t.Errorf("Expected EndDate to be capped at now (between %v and %v), got %v (stored future end date was %v)", beforeRequest, afterRequest, *incident.EndDate, futureEndDate)
	}
}

// TestUpdateAnimalAdmin_MidBQ_UpdatesIncidentDetails verifies that editing
// incident details while in BQ via the admin handler updates the incident row.
func TestUpdateAnimalAdmin_MidBQ_UpdatesIncidentDetails(t *testing.T) {
	db := setupAnimalTestDB(t)
	user, group := createAnimalTestUser(t, db, "admin", "admin@example.com", true)
	animal := createTestAnimal(t, db, group.ID, "Rex", "Dog")

	if err := db.Model(animal).Updates(map[string]interface{}{
		"status":                      "bite_quarantine",
		"quarantine_incident_details": "Original details.",
	}).Error; err != nil {
		t.Fatalf("seed BQ status: %v", err)
	}
	if err := db.Create(&models.AnimalBQIncident{
		AnimalID:        animal.ID,
		IncidentDetails: "Original details.",
		StartDate:       animal.CreatedAt,
	}).Error; err != nil {
		t.Fatalf("seed incident row: %v", err)
	}

	updated := "Updated details."
	updateReq := AnimalRequest{
		Name:                      "Rex",
		QuarantineIncidentDetails: &updated,
	}
	jsonData, _ := json.Marshal(updateReq)

	c, w := setupAnimalTestContext(user.ID, true)
	c.Params = gin.Params{{Key: "animalId", Value: fmt.Sprintf("%d", animal.ID)}}
	c.Request = httptest.NewRequest("PUT", fmt.Sprintf("/api/v1/admin/animals/%d", animal.ID), bytes.NewBuffer(jsonData))
	c.Request.Header.Set("Content-Type", "application/json")

	handler := UpdateAnimalAdmin(db, nil, &embedding.StubEmbedder{})
	handler(c)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	var incident models.AnimalBQIncident
	if err := db.Where("animal_id = ? AND end_date IS NULL", animal.ID).First(&incident).Error; err != nil {
		t.Fatalf("reload incident row: %v", err)
	}
	if incident.IncidentDetails != "Updated details." {
		t.Errorf("IncidentDetails = %q, want %q", incident.IncidentDetails, "Updated details.")
	}
}

// TestUpdateAnimalAdmin_MidBQ_EditStartDate_SyncsIncidentRow verifies that
// editing the quarantine start date while still in BQ updates the active
// incident row's StartDate, so the permanent history stays accurate.
func TestUpdateAnimalAdmin_MidBQ_EditStartDate_SyncsIncidentRow(t *testing.T) {
	db := setupAnimalTestDB(t)
	user, group := createAnimalTestUser(t, db, "admin", "admin@example.com", true)
	animal := createTestAnimal(t, db, group.ID, "Rex", "Dog")

	originalStart := time.Date(2025, 11, 3, 0, 0, 0, 0, time.UTC)
	if err := db.Model(animal).Updates(map[string]interface{}{
		"status":                "bite_quarantine",
		"quarantine_start_date": originalStart,
	}).Error; err != nil {
		t.Fatalf("seed BQ status: %v", err)
	}
	if err := db.Create(&models.AnimalBQIncident{
		AnimalID:        animal.ID,
		IncidentDetails: "Original details.",
		StartDate:       originalStart,
	}).Error; err != nil {
		t.Fatalf("seed incident row: %v", err)
	}

	correctedStart := time.Date(2025, 11, 1, 0, 0, 0, 0, time.UTC)
	updateReq := AnimalRequest{
		Name:   "Rex",
		Status: "bite_quarantine",
		QuarantineStartDate: NullableTime{
			Time:  &correctedStart,
			Valid: true,
		},
	}
	jsonData, _ := json.Marshal(updateReq)

	c, w := setupAnimalTestContext(user.ID, true)
	c.Params = gin.Params{{Key: "animalId", Value: fmt.Sprintf("%d", animal.ID)}}
	c.Request = httptest.NewRequest("PUT", fmt.Sprintf("/api/v1/admin/animals/%d", animal.ID), bytes.NewBuffer(jsonData))
	c.Request.Header.Set("Content-Type", "application/json")

	handler := UpdateAnimalAdmin(db, nil, &embedding.StubEmbedder{})
	handler(c)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	var incident models.AnimalBQIncident
	if err := db.Where("animal_id = ? AND end_date IS NULL", animal.ID).First(&incident).Error; err != nil {
		t.Fatalf("reload incident row: %v", err)
	}
	if !incident.StartDate.Equal(correctedStart) {
		t.Errorf("StartDate = %v, want %v", incident.StartDate, correctedStart)
	}
}
