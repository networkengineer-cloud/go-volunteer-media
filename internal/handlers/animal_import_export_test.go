package handlers

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
)

// TestExportAnimalsCSV_Success tests successful CSV export
func TestExportAnimalsCSV_Success(t *testing.T) {
	db := setupAnimalTestDB(t)
	user, group := createAnimalTestUser(t, db, "admin", "admin@example.com", true)

	// Create test animals
	createTestAnimal(t, db, group.ID, "Rex", "Dog")
	createTestAnimal(t, db, group.ID, "Fluffy", "Cat")

	c, w := setupAnimalTestContext(user.ID, true)
	c.Request = httptest.NewRequest("GET", "/api/v1/admin/animals/export-csv", nil)

	handler := ExportAnimalsCSV(db)
	handler(c)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Check content type
	contentType := w.Header().Get("Content-Type")
	if contentType != "text/csv" {
		t.Errorf("Expected Content-Type 'text/csv', got '%s'", contentType)
	}

	// Check content disposition
	contentDisposition := w.Header().Get("Content-Disposition")
	if !strings.Contains(contentDisposition, "attachment") || !strings.Contains(contentDisposition, "animals.csv") {
		t.Errorf("Expected Content-Disposition with attachment and animals.csv, got '%s'", contentDisposition)
	}

	// Parse CSV
	reader := csv.NewReader(w.Body)
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("Failed to parse CSV: %v", err)
	}

	// Should have header + 2 data rows
	if len(records) != 3 {
		t.Errorf("Expected 3 CSV rows (header + 2 animals), got %d", len(records))
	}

	// Check header
	expectedHeader := []string{"id", "group_id", "name", "species", "breed", "age", "description", "status", "image_url"}
	if len(records[0]) != len(expectedHeader) {
		t.Errorf("Expected %d header columns, got %d", len(expectedHeader), len(records[0]))
	}
}

// TestExportAnimalsCSV_WithGroupFilter tests filtering by group
func TestExportAnimalsCSV_WithGroupFilter(t *testing.T) {
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
	createTestAnimal(t, db, group2.ID, "Fluffy", "Cat")

	c, w := setupAnimalTestContext(user.ID, true)
	c.Request = httptest.NewRequest("GET", fmt.Sprintf("/api/v1/admin/animals/export-csv?group_id=%d", group1.ID), nil)

	handler := ExportAnimalsCSV(db)
	handler(c)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Parse CSV
	reader := csv.NewReader(w.Body)
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("Failed to parse CSV: %v", err)
	}

	// Should have header + 1 data row (only group1 animals)
	if len(records) != 2 {
		t.Errorf("Expected 2 CSV rows (header + 1 animal), got %d", len(records))
	}
}

// TestImportAnimalsCSV_Success tests successful CSV import
func TestImportAnimalsCSV_Success(t *testing.T) {
	db := setupAnimalTestDB(t)
	user, group := createAnimalTestUser(t, db, "admin", "admin@example.com", true)

	// Create CSV content
	csvContent := fmt.Sprintf(`group_id,name,species,breed,age,description,status,image_url
%d,Rex,Dog,Golden Retriever,3,Friendly dog,available,/uploads/rex.jpg
%d,Fluffy,Cat,Persian,2,Sweet cat,available,/uploads/fluffy.jpg`, group.ID, group.ID)

	// Create multipart form
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "animals.csv")
	if err != nil {
		t.Fatalf("Failed to create form file: %v", err)
	}
	part.Write([]byte(csvContent))
	writer.Close()

	c, w := setupAnimalTestContext(user.ID, true)
	c.Request = httptest.NewRequest("POST", "/api/v1/admin/animals/import-csv", body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())

	handler := ImportAnimalsCSV(db)
	handler(c)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response["count"].(float64) != 2 {
		t.Errorf("Expected count 2, got %v", response["count"])
	}

	// Verify animals were created
	var animals []models.Animal
	db.Where("group_id = ?", group.ID).Find(&animals)

	if len(animals) != 2 {
		t.Errorf("Expected 2 animals in database, got %d", len(animals))
	}
}

// TestImportAnimalsCSV_InvalidFile tests importing non-CSV file
func TestImportAnimalsCSV_InvalidFile(t *testing.T) {
	db := setupAnimalTestDB(t)
	user, _ := createAnimalTestUser(t, db, "admin", "admin@example.com", true)

	// Create multipart form with non-CSV file
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "animals.txt")
	part.Write([]byte("not a csv"))
	writer.Close()

	c, w := setupAnimalTestContext(user.ID, true)
	c.Request = httptest.NewRequest("POST", "/api/v1/admin/animals/import-csv", body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())

	handler := ImportAnimalsCSV(db)
	handler(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	var response map[string]string
	json.Unmarshal(w.Body.Bytes(), &response)
	if !strings.Contains(response["error"], "CSV") {
		t.Errorf("Expected error about CSV, got '%s'", response["error"])
	}
}

// TestImportAnimalsCSV_MissingRequiredColumn tests CSV with missing required columns
func TestImportAnimalsCSV_MissingRequiredColumn(t *testing.T) {
	db := setupAnimalTestDB(t)
	user, _ := createAnimalTestUser(t, db, "admin", "admin@example.com", true)

	// CSV without required 'name' column
	csvContent := `group_id,species,breed
1,Dog,Golden Retriever`

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "animals.csv")
	part.Write([]byte(csvContent))
	writer.Close()

	c, w := setupAnimalTestContext(user.ID, true)
	c.Request = httptest.NewRequest("POST", "/api/v1/admin/animals/import-csv", body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())

	handler := ImportAnimalsCSV(db)
	handler(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	var response map[string]string
	json.Unmarshal(w.Body.Bytes(), &response)
	if !strings.Contains(response["error"], "name") {
		t.Errorf("Expected error about missing 'name' column, got '%s'", response["error"])
	}
}

// TestImportAnimalsCSV_InvalidData tests CSV with invalid data
func TestImportAnimalsCSV_InvalidData(t *testing.T) {
	db := setupAnimalTestDB(t)
	user, group := createAnimalTestUser(t, db, "admin", "admin@example.com", true)

	// CSV with invalid group_id and missing name
	csvContent := fmt.Sprintf(`group_id,name,species
invalid,Rex,Dog
%d,,Cat
%d,Fluffy,Cat`, group.ID, group.ID)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "animals.csv")
	part.Write([]byte(csvContent))
	writer.Close()

	c, w := setupAnimalTestContext(user.ID, true)
	c.Request = httptest.NewRequest("POST", "/api/v1/admin/animals/import-csv", body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())

	handler := ImportAnimalsCSV(db)
	handler(c)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	// Should import 1 valid animal
	if response["count"].(float64) != 1 {
		t.Errorf("Expected count 1, got %v", response["count"])
	}

	// Should have warnings for invalid rows
	if warnings, ok := response["warnings"]; ok {
		warningList := warnings.([]interface{})
		if len(warningList) != 2 {
			t.Errorf("Expected 2 warnings, got %d", len(warningList))
		}
	} else {
		t.Error("Expected warnings in response")
	}
}

// TestImportAnimalsCSV_NoFile tests import without uploading a file
func TestImportAnimalsCSV_NoFile(t *testing.T) {
	db := setupAnimalTestDB(t)
	user, _ := createAnimalTestUser(t, db, "admin", "admin@example.com", true)

	c, w := setupAnimalTestContext(user.ID, true)
	c.Request = httptest.NewRequest("POST", "/api/v1/admin/animals/import-csv", nil)

	handler := ImportAnimalsCSV(db)
	handler(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	var response map[string]string
	json.Unmarshal(w.Body.Bytes(), &response)
	if response["error"] != "No file uploaded" {
		t.Errorf("Expected error 'No file uploaded', got '%s'", response["error"])
	}
}

// TestExportAnimalCommentsCSV_Success tests successful comment export
func TestExportAnimalCommentsCSV_Success(t *testing.T) {
	db := setupAnimalTestDB(t)

	// Need to add CommentTag and AnimalComment to migrations
	db.AutoMigrate(&models.CommentTag{}, &models.AnimalComment{})

	user, group := createAnimalTestUser(t, db, "admin", "admin@example.com", true)

	// Create animal
	animal := createTestAnimal(t, db, group.ID, "Rex", "Dog")

	// Create comment
	comment := &models.AnimalComment{
		AnimalID: animal.ID,
		UserID:   user.ID,
		Content:  "Test comment",
	}
	db.Create(comment)

	c, w := setupAnimalTestContext(user.ID, true)
	c.Request = httptest.NewRequest("GET", "/api/v1/admin/animals/export-comments-csv", nil)

	handler := ExportAnimalCommentsCSV(db)
	handler(c)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	// Check content type
	contentType := w.Header().Get("Content-Type")
	if contentType != "text/csv" {
		t.Errorf("Expected Content-Type 'text/csv', got '%s'", contentType)
	}

	// Parse CSV
	reader := csv.NewReader(w.Body)
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("Failed to parse CSV: %v", err)
	}

	// Should have header + 1 comment row
	if len(records) != 2 {
		t.Errorf("Expected 2 CSV rows (header + 1 comment), got %d", len(records))
	}

	// Check that comment content is in the CSV
	found := false
	for _, record := range records[1:] {
		for _, field := range record {
			if strings.Contains(field, "Test comment") {
				found = true
				break
			}
		}
	}
	if !found {
		t.Error("Expected to find 'Test comment' in CSV output")
	}
}

// TestExportAnimalCommentsCSV_WithGroupFilter tests filtering comments by group
func TestExportAnimalCommentsCSV_WithGroupFilter(t *testing.T) {
	db := setupAnimalTestDB(t)
	db.AutoMigrate(&models.CommentTag{}, &models.AnimalComment{})

	user, group1 := createAnimalTestUser(t, db, "admin", "admin@example.com", true)

	// Create second group
	group2 := &models.Group{
		Name:        "Group 2",
		Description: "Test group 2",
	}
	db.Create(group2)

	// Create animals in both groups
	animal1 := createTestAnimal(t, db, group1.ID, "Rex", "Dog")
	animal2 := createTestAnimal(t, db, group2.ID, "Fluffy", "Cat")

	// Create comments for both animals
	comment1 := &models.AnimalComment{
		AnimalID: animal1.ID,
		UserID:   user.ID,
		Content:  "Comment for Rex",
	}
	db.Create(comment1)

	comment2 := &models.AnimalComment{
		AnimalID: animal2.ID,
		UserID:   user.ID,
		Content:  "Comment for Fluffy",
	}
	db.Create(comment2)

	c, w := setupAnimalTestContext(user.ID, true)
	c.Request = httptest.NewRequest("GET", fmt.Sprintf("/api/v1/admin/animals/export-comments-csv?group_id=%d", group1.ID), nil)

	handler := ExportAnimalCommentsCSV(db)
	handler(c)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Parse CSV
	reader := csv.NewReader(w.Body)
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("Failed to parse CSV: %v", err)
	}

	// Should have header + 1 comment row (only group1)
	if len(records) != 2 {
		t.Errorf("Expected 2 CSV rows (header + 1 comment), got %d", len(records))
	}
}

// TestExportAnimalCommentsCSV_WithAnimalFilter tests filtering comments by animal
func TestExportAnimalCommentsCSV_WithAnimalFilter(t *testing.T) {
	db := setupAnimalTestDB(t)
	db.AutoMigrate(&models.CommentTag{}, &models.AnimalComment{})

	user, group := createAnimalTestUser(t, db, "admin", "admin@example.com", true)

	// Create two animals
	animal1 := createTestAnimal(t, db, group.ID, "Rex", "Dog")
	animal2 := createTestAnimal(t, db, group.ID, "Fluffy", "Cat")

	// Create comments for both
	comment1 := &models.AnimalComment{
		AnimalID: animal1.ID,
		UserID:   user.ID,
		Content:  "Comment for Rex",
	}
	db.Create(comment1)

	comment2 := &models.AnimalComment{
		AnimalID: animal2.ID,
		UserID:   user.ID,
		Content:  "Comment for Fluffy",
	}
	db.Create(comment2)

	c, w := setupAnimalTestContext(user.ID, true)
	c.Request = httptest.NewRequest("GET", fmt.Sprintf("/api/v1/admin/animals/export-comments-csv?animal_id=%d", animal1.ID), nil)

	handler := ExportAnimalCommentsCSV(db)
	handler(c)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Parse CSV
	reader := csv.NewReader(w.Body)
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("Failed to parse CSV: %v", err)
	}

	// Should have header + 1 comment row (only animal1)
	if len(records) != 2 {
		t.Errorf("Expected 2 CSV rows (header + 1 comment), got %d", len(records))
	}

	// Verify it's the correct comment
	if !strings.Contains(records[1][8], "Comment for Rex") {
		t.Error("Expected comment for Rex in output")
	}
}
