package database

import (
	"os"
	"testing"
	"time"

	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestInitialize_EnvDefaults(t *testing.T) {
	// Clear environment variables to test defaults
	os.Clearenv()

	// Initialize should not panic with default values
	db, err := Initialize()
	if err != nil {
		// Error is expected if database is not available
		// We're just testing that the function doesn't panic
		// and returns an error properly
		t.Logf("Database initialization failed (expected in test): %v", err)
		return
	}

	// If DB is available, verify it's not nil
	if db == nil {
		t.Fatal("Database should not be nil when initialization succeeds")
	}
}

func TestInitialize_CustomEnv(t *testing.T) {
	// Set custom environment variables
	os.Setenv("DB_HOST", "testhost")
	os.Setenv("DB_PORT", "5433")
	os.Setenv("DB_USER", "testuser")
	os.Setenv("DB_PASSWORD", "testpass")
	os.Setenv("DB_NAME", "testdb")
	os.Setenv("DB_SSLMODE", "require")
	defer os.Clearenv()

	// Initialize with custom environment - will likely fail in test
	// but we're testing environment variable handling
	_, err := Initialize()
	if err != nil {
		t.Logf("Database initialization failed with custom env (expected): %v", err)
		return
	}
}

func TestInitialize_InvalidSSLMode(t *testing.T) {
	// Set invalid SSL mode
	os.Clearenv()
	os.Setenv("DB_SSLMODE", "invalid_mode")
	defer os.Clearenv()

	_, err := Initialize()
	if err == nil {
		t.Error("Expected error for invalid SSL mode")
	}

	// Check that the error contains "invalid SSL mode"
	if err != nil {
		expected := "invalid SSL mode: invalid_mode"
		if len(err.Error()) < len(expected) || err.Error()[:len(expected)] != expected {
			t.Errorf("Expected error to start with '%s', got: %v", expected, err)
		}
	}
}

func TestCreateDefaultAnimalTags_RestoresSoftDeletedTag(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite db: %v", err)
	}

	if err := db.AutoMigrate(&models.Group{}, &models.AnimalTag{}); err != nil {
		t.Fatalf("failed to automigrate: %v", err)
	}

	group := models.Group{Name: "modsquad"}
	if err := db.Create(&group).Error; err != nil {
		t.Fatalf("failed to create group: %v", err)
	}

	if err := createDefaultAnimalTags(db); err != nil {
		t.Fatalf("createDefaultAnimalTags failed: %v", err)
	}

	var iso models.AnimalTag
	if err := db.Where("group_id = ? AND name = ?", group.ID, "iso").First(&iso).Error; err != nil {
		t.Fatalf("failed to find iso tag: %v", err)
	}

	if iso.DeletedAt.Valid {
		t.Fatalf("expected iso tag to be not deleted")
	}

	if err := db.Delete(&iso).Error; err != nil {
		t.Fatalf("failed to soft-delete iso tag: %v", err)
	}

	// Ensure the soft-delete took effect.
	var softDeleted models.AnimalTag
	if err := db.Unscoped().Where("group_id = ? AND name = ?", group.ID, "iso").First(&softDeleted).Error; err != nil {
		t.Fatalf("failed to find soft-deleted iso tag: %v", err)
	}
	if !softDeleted.DeletedAt.Valid {
		t.Fatalf("expected iso tag to be soft-deleted")
	}

	// Guard against any clock precision edge cases.
	if softDeleted.DeletedAt.Time.After(time.Now().Add(5 * time.Second)) {
		t.Fatalf("unexpected deleted_at in the future: %v", softDeleted.DeletedAt.Time)
	}

	// Re-running default creation should restore (undelete) the tag instead of failing on uniqueness.
	if err := createDefaultAnimalTags(db); err != nil {
		t.Fatalf("createDefaultAnimalTags (second run) failed: %v", err)
	}

	var restored models.AnimalTag
	if err := db.Unscoped().Where("group_id = ? AND name = ?", group.ID, "iso").First(&restored).Error; err != nil {
		t.Fatalf("failed to find restored iso tag: %v", err)
	}
	if restored.DeletedAt.Valid {
		t.Fatalf("expected iso tag to be restored (deleted_at cleared)")
	}
}
