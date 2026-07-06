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
	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
)

func TestBuildQuarantineEmailBody(t *testing.T) {
	start := time.Date(2026, 6, 22, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC)
	a := &models.Animal{
		Name:                      "Rex",
		QuarantineStartDate:       &start,
		QuarantineEndDate:         &end,
		QuarantineIncidentDetails: "Bit a volunteer on the hand during leashing.",
	}
	title, body := buildQuarantineEmail(a)
	if title != "🚨 Bite Quarantine: Rex" {
		t.Errorf("unexpected title: %q", title)
	}
	if !strings.Contains(body, "Rex has been placed in bite quarantine") {
		t.Errorf("body missing intro: %q", body)
	}
	if !strings.Contains(body, "June 22, 2026") {
		t.Errorf("body missing formatted start date: %q", body)
	}
	if !strings.Contains(body, "July 2, 2026") {
		t.Errorf("body missing formatted end date: %q", body)
	}
	if !strings.Contains(body, "Bit a volunteer on the hand during leashing.") {
		t.Errorf("body missing incident details: %q", body)
	}
}

func TestSendQuarantineNotificationEmail_NilServiceNoPanic(t *testing.T) {
	// nil email service must be a safe no-op
	sendQuarantineNotificationEmail(nil, nil, &models.Animal{Name: "Rex"})
}

// TestUpdateAnimal_EnterQuarantine_StoresIncidentAndKeepsItOnEdit verifies that
// entering bite_quarantine stores the incident details, and that a subsequent
// edit while still in bite_quarantine (without resending incident details)
// does not clear the previously stored details.
func TestUpdateAnimal_EnterQuarantine_StoresIncidentAndKeepsItOnEdit(t *testing.T) {
	db := setupAnimalTestDB(t)
	user, group := createAnimalTestUser(t, db, "testuser", "test@example.com", false)
	animal := createTestAnimal(t, db, group.ID, "Rex", "Dog")

	// Transition into bite quarantine with incident details
	details := "Bit a volunteer."
	updateReq := AnimalRequest{
		Name:                      "Rex",
		Species:                   "Dog",
		Status:                    "bite_quarantine",
		QuarantineIncidentDetails: &details,
	}
	jsonData, _ := json.Marshal(updateReq)

	c, w := setupAnimalTestContext(user.ID, false)
	c.Params = gin.Params{
		{Key: "id", Value: fmt.Sprintf("%d", group.ID)},
		{Key: "animalId", Value: fmt.Sprintf("%d", animal.ID)},
	}
	c.Request = httptest.NewRequest("PUT", fmt.Sprintf("/api/v1/groups/%d/animals/%d", group.ID, animal.ID), bytes.NewBuffer(jsonData))
	c.Request.Header.Set("Content-Type", "application/json")

	handler := UpdateAnimal(db, nil)
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

	// Editing another field while already in BQ (no incident details in payload)
	// must NOT clear the previously stored details.
	updateReq2 := AnimalRequest{
		Name:    "Rex Updated",
		Species: "Dog",
		Status:  "bite_quarantine",
	}
	jsonData2, _ := json.Marshal(updateReq2)

	c2, w2 := setupAnimalTestContext(user.ID, false)
	c2.Params = gin.Params{
		{Key: "id", Value: fmt.Sprintf("%d", group.ID)},
		{Key: "animalId", Value: fmt.Sprintf("%d", animal.ID)},
	}
	c2.Request = httptest.NewRequest("PUT", fmt.Sprintf("/api/v1/groups/%d/animals/%d", group.ID, animal.ID), bytes.NewBuffer(jsonData2))
	c2.Request.Header.Set("Content-Type", "application/json")

	handler2 := UpdateAnimal(db, nil)
	handler2(c2)

	if w2.Code != http.StatusOK {
		t.Fatalf("Expected status %d, got %d. Body: %s", http.StatusOK, w2.Code, w2.Body.String())
	}

	if err := db.First(&got, animal.ID).Error; err != nil {
		t.Fatalf("Failed to reload animal: %v", err)
	}
	if got.QuarantineIncidentDetails != "Bit a volunteer." {
		t.Errorf("incident wrongly cleared on edit: %q", got.QuarantineIncidentDetails)
	}
}

// TestUpdateAnimal_LeaveQuarantine_ClearsIncident verifies that leaving
// bite_quarantine clears the stored incident details.
func TestUpdateAnimal_LeaveQuarantine_ClearsIncident(t *testing.T) {
	db := setupAnimalTestDB(t)
	user, group := createAnimalTestUser(t, db, "testuser", "test@example.com", false)
	animal := createTestAnimal(t, db, group.ID, "Rex", "Dog")

	// Seed an animal already in BQ with details
	if err := db.Model(animal).Updates(map[string]interface{}{
		"status":                      "bite_quarantine",
		"quarantine_incident_details": "Bit a volunteer.",
	}).Error; err != nil {
		t.Fatalf("Failed to seed animal into quarantine: %v", err)
	}

	updateReq := AnimalRequest{
		Name:    "Rex",
		Species: "Dog",
		Status:  "available",
	}
	jsonData, _ := json.Marshal(updateReq)

	c, w := setupAnimalTestContext(user.ID, false)
	c.Params = gin.Params{
		{Key: "id", Value: fmt.Sprintf("%d", group.ID)},
		{Key: "animalId", Value: fmt.Sprintf("%d", animal.ID)},
	}
	c.Request = httptest.NewRequest("PUT", fmt.Sprintf("/api/v1/groups/%d/animals/%d", group.ID, animal.ID), bytes.NewBuffer(jsonData))
	c.Request.Header.Set("Content-Type", "application/json")

	handler := UpdateAnimal(db, nil)
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

// TestCreateAnimal_BQEntry_CreatesIncidentRow verifies that creating an animal
// directly in bite_quarantine writes an AnimalBQIncident row.
func TestCreateAnimal_BQEntry_CreatesIncidentRow(t *testing.T) {
	db := setupAnimalTestDB(t)
	user, group := createAnimalTestUser(t, db, "admin", "admin@example.com", true)

	details := "Bit a volunteer."
	reqBody := AnimalRequest{
		Name:                      "Rex",
		Species:                   "Dog",
		Status:                    "bite_quarantine",
		QuarantineIncidentDetails: &details,
	}
	jsonData, _ := json.Marshal(reqBody)

	c, w := setupAnimalTestContext(user.ID, true)
	c.Params = gin.Params{{Key: "id", Value: fmt.Sprintf("%d", group.ID)}}
	c.Request = httptest.NewRequest("POST", fmt.Sprintf("/api/groups/%d/animals", group.ID), bytes.NewBuffer(jsonData))
	c.Request.Header.Set("Content-Type", "application/json")

	handler := CreateAnimal(db, nil)
	handler(c)

	if w.Code != http.StatusCreated {
		t.Fatalf("Expected 201, got %d. Body: %s", w.Code, w.Body.String())
	}

	var incident models.AnimalBQIncident
	if err := db.Where("end_date IS NULL").First(&incident).Error; err != nil {
		t.Fatalf("Expected an active AnimalBQIncident row, got none: %v", err)
	}
	if incident.IncidentDetails != "Bit a volunteer." {
		t.Errorf("IncidentDetails = %q, want %q", incident.IncidentDetails, "Bit a volunteer.")
	}
}

// TestUpdateAnimal_BQEntry_CreatesIncidentRow verifies that transitioning an
// animal to bite_quarantine writes an AnimalBQIncident row.
func TestUpdateAnimal_BQEntry_CreatesIncidentRow(t *testing.T) {
	db := setupAnimalTestDB(t)
	user, group := createAnimalTestUser(t, db, "admin", "admin@example.com", true)
	animal := createTestAnimal(t, db, group.ID, "Rex", "Dog")

	details := "Bit a volunteer."
	reqBody := AnimalRequest{
		Name:                      "Rex",
		Status:                    "bite_quarantine",
		QuarantineIncidentDetails: &details,
	}
	jsonData, _ := json.Marshal(reqBody)

	c, w := setupAnimalTestContext(user.ID, true)
	c.Params = gin.Params{
		{Key: "id", Value: fmt.Sprintf("%d", group.ID)},
		{Key: "animalId", Value: fmt.Sprintf("%d", animal.ID)},
	}
	c.Request = httptest.NewRequest("PUT", "/", bytes.NewBuffer(jsonData))
	c.Request.Header.Set("Content-Type", "application/json")

	handler := UpdateAnimal(db, nil)
	handler(c)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	var incident models.AnimalBQIncident
	if err := db.Where("animal_id = ? AND end_date IS NULL", animal.ID).First(&incident).Error; err != nil {
		t.Fatalf("Expected an active AnimalBQIncident row: %v", err)
	}
	if incident.IncidentDetails != "Bit a volunteer." {
		t.Errorf("IncidentDetails = %q, want %q", incident.IncidentDetails, "Bit a volunteer.")
	}
}

// TestUpdateAnimal_BQExit_StampsEndDate verifies that leaving bite_quarantine
// stamps EndDate on the active AnimalBQIncident row.
func TestUpdateAnimal_BQExit_StampsEndDate(t *testing.T) {
	db := setupAnimalTestDB(t)
	user, group := createAnimalTestUser(t, db, "admin", "admin@example.com", true)
	animal := createTestAnimal(t, db, group.ID, "Rex", "Dog")

	// Seed animal in BQ with an active incident row.
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

	reqBody := AnimalRequest{Name: "Rex", Status: "available"}
	jsonData, _ := json.Marshal(reqBody)

	c, w := setupAnimalTestContext(user.ID, true)
	c.Params = gin.Params{
		{Key: "id", Value: fmt.Sprintf("%d", group.ID)},
		{Key: "animalId", Value: fmt.Sprintf("%d", animal.ID)},
	}
	c.Request = httptest.NewRequest("PUT", "/", bytes.NewBuffer(jsonData))
	c.Request.Header.Set("Content-Type", "application/json")

	handler := UpdateAnimal(db, nil)
	handler(c)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	var incident models.AnimalBQIncident
	if err := db.Where("animal_id = ?", animal.ID).First(&incident).Error; err != nil {
		t.Fatalf("reload incident row: %v", err)
	}
	if incident.EndDate == nil {
		t.Error("Expected EndDate to be stamped after leaving BQ, got nil")
	}
}

// TestUpdateAnimal_MidBQ_UpdatesIncidentDetails verifies that editing incident
// details while already in bite_quarantine updates the active incident row.
func TestUpdateAnimal_MidBQ_UpdatesIncidentDetails(t *testing.T) {
	db := setupAnimalTestDB(t)
	user, group := createAnimalTestUser(t, db, "admin", "admin@example.com", true)
	animal := createTestAnimal(t, db, group.ID, "Rex", "Dog")

	// Seed animal in BQ with an active incident row.
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
	reqBody := AnimalRequest{
		Name:                      "Rex",
		Status:                    "bite_quarantine",
		QuarantineIncidentDetails: &updated,
	}
	jsonData, _ := json.Marshal(reqBody)

	c, w := setupAnimalTestContext(user.ID, true)
	c.Params = gin.Params{
		{Key: "id", Value: fmt.Sprintf("%d", group.ID)},
		{Key: "animalId", Value: fmt.Sprintf("%d", animal.ID)},
	}
	c.Request = httptest.NewRequest("PUT", "/", bytes.NewBuffer(jsonData))
	c.Request.Header.Set("Content-Type", "application/json")

	handler := UpdateAnimal(db, nil)
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

// TestCreateAnimal_BiteQuarantine_StoresIncidentDetails verifies that creating
// an animal directly with status "bite_quarantine" stores the provided
// incident details.
func TestCreateAnimal_BiteQuarantine_StoresIncidentDetails(t *testing.T) {
	db := setupAnimalTestDB(t)
	user, group := createAnimalTestUser(t, db, "testuser", "test@example.com", false)

	details := "Bit a volunteer."
	animalReq := AnimalRequest{
		Name:                      "Rex",
		Species:                   "Dog",
		Status:                    "bite_quarantine",
		QuarantineIncidentDetails: &details,
	}

	jsonData, _ := json.Marshal(animalReq)

	c, w := setupAnimalTestContext(user.ID, false)
	c.Params = gin.Params{{Key: "id", Value: fmt.Sprintf("%d", group.ID)}}
	c.Request = httptest.NewRequest("POST", fmt.Sprintf("/api/v1/groups/%d/animals", group.ID), bytes.NewBuffer(jsonData))
	c.Request.Header.Set("Content-Type", "application/json")

	handler := CreateAnimal(db, nil)
	handler(c)

	if w.Code != http.StatusCreated {
		t.Fatalf("Expected status %d, got %d. Body: %s", http.StatusCreated, w.Code, w.Body.String())
	}

	var created models.Animal
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	var got models.Animal
	if err := db.First(&got, created.ID).Error; err != nil {
		t.Fatalf("Failed to reload animal: %v", err)
	}
	if got.QuarantineIncidentDetails != "Bit a volunteer." {
		t.Errorf("incident not stored on create: %q", got.QuarantineIncidentDetails)
	}
}

// TestGetAnimal_ReturnsBQHistory verifies that ended BQ episodes are
// returned in the bq_incidents field of the animal detail response.
func TestGetAnimal_ReturnsBQHistory(t *testing.T) {
	db := setupAnimalTestDB(t)
	user, group := createAnimalTestUser(t, db, "admin", "admin@example.com", true)
	animal := createTestAnimal(t, db, group.ID, "Rex", "Dog")

	// Create an ended BQ incident.
	endDate := animal.CreatedAt.Add(10 * 24 * time.Hour)
	if err := db.Create(&models.AnimalBQIncident{
		AnimalID:        animal.ID,
		IncidentDetails: "Bit a volunteer.",
		StartDate:       animal.CreatedAt,
		EndDate:         &endDate,
	}).Error; err != nil {
		t.Fatalf("seed ended incident: %v", err)
	}
	// Also create an active one that must NOT appear.
	if err := db.Create(&models.AnimalBQIncident{
		AnimalID:        animal.ID,
		IncidentDetails: "Active, no end.",
		StartDate:       endDate,
	}).Error; err != nil {
		t.Fatalf("seed active incident: %v", err)
	}

	c, w := setupAnimalTestContext(user.ID, false)
	c.Params = gin.Params{
		{Key: "id", Value: fmt.Sprintf("%d", group.ID)},
		{Key: "animalId", Value: fmt.Sprintf("%d", animal.ID)},
	}
	c.Request = httptest.NewRequest("GET", "/", nil)

	handler := GetAnimal(db)
	handler(c)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	var got models.Animal
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(got.BQIncidents) != 1 {
		t.Fatalf("Expected 1 ended BQ incident in response, got %d", len(got.BQIncidents))
	}
	if got.BQIncidents[0].IncidentDetails != "Bit a volunteer." {
		t.Errorf("IncidentDetails = %q, want %q", got.BQIncidents[0].IncidentDetails, "Bit a volunteer.")
	}
}

// TestUpdateAnimal_MidBQ_EditStartDate_SyncsIncidentRow verifies that editing
// the quarantine start date while still in BQ updates the active incident
// row's StartDate, so the permanent history stays accurate.
func TestUpdateAnimal_MidBQ_EditStartDate_SyncsIncidentRow(t *testing.T) {
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
	reqBody := AnimalRequest{
		Name:   "Rex",
		Status: "bite_quarantine",
		QuarantineStartDate: NullableTime{
			Time:  &correctedStart,
			Valid: true,
		},
	}
	jsonData, _ := json.Marshal(reqBody)

	c, w := setupAnimalTestContext(user.ID, true)
	c.Params = gin.Params{
		{Key: "id", Value: fmt.Sprintf("%d", group.ID)},
		{Key: "animalId", Value: fmt.Sprintf("%d", animal.ID)},
	}
	c.Request = httptest.NewRequest("PUT", "/", bytes.NewBuffer(jsonData))
	c.Request.Header.Set("Content-Type", "application/json")

	handler := UpdateAnimal(db, nil)
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
