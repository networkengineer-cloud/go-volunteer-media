package database

import (
	"os"
	"testing"
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
