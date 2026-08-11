package database

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
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

func TestCreateDefaultAnimalTags_RespectsDeletedTag(t *testing.T) {
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
		t.Fatalf("expected iso tag to not be deleted after initial creation")
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

	// Re-running default creation must NOT restore a tag that an admin deleted.
	if err := createDefaultAnimalTags(db); err != nil {
		t.Fatalf("createDefaultAnimalTags (second run) failed: %v", err)
	}

	// Exactly one row must exist (the original soft-deleted one), still deleted.
	var rows []models.AnimalTag
	if err := db.Unscoped().Where("group_id = ? AND name = ?", group.ID, "iso").Find(&rows).Error; err != nil {
		t.Fatalf("failed to query iso rows after rerun: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected exactly 1 iso row after rerun, got %d", len(rows))
	}
	if !rows[0].DeletedAt.Valid {
		t.Fatalf("expected iso tag to remain soft-deleted after rerun, but deleted_at was cleared")
	}
}

// TestCreateCustomIndexes_CoverageRequestActiveUnique verifies the partial
// unique index (idx_coverage_request_active_unique) that backstops
// CreateCoverageRequest's app-level check-then-create logic in
// internal/handlers/schedule_coverage.go. A transaction alone does not make
// that check-then-insert atomic under READ COMMITTED (Postgres in
// production), so two concurrent creates for the same (group,
// requested_by_user_id, date, hour) can both pass the app-level check and
// both insert. This test bypasses the app-level check entirely (two direct
// db.Create calls) to isolate and confirm the DB-level backstop actually
// rejects the second insert - and that a cancelled row for the same key
// does NOT block a fresh active one, since the index is scoped to
// status <> 'cancelled'.
func TestCreateCustomIndexes_CoverageRequestActiveUnique(t *testing.T) {
	// Use a per-test-run unique shared-cache DSN so this test's tables don't
	// collide with TestCreateDefaultAnimalTags_RespectsDeletedTag, which
	// uses the same "file::memory:?cache=shared" pattern.
	dsn := fmt.Sprintf("file:coverage_idx_test_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite db: %v", err)
	}

	if err := db.AutoMigrate(&models.User{}, &models.Group{}, &models.ShiftCoverageRequest{}); err != nil {
		t.Fatalf("failed to automigrate: %v", err)
	}
	if err := createCustomIndexes(db); err != nil {
		t.Fatalf("createCustomIndexes failed: %v", err)
	}

	user := models.User{Username: "vol1", Email: "vol1@test.com", Password: "hashed"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}
	group := models.Group{Name: "Dogs"}
	if err := db.Create(&group).Error; err != nil {
		t.Fatalf("failed to create group: %v", err)
	}

	date := time.Date(2026, 8, 18, 0, 0, 0, 0, time.UTC)
	first := models.ShiftCoverageRequest{
		GroupID:           group.ID,
		RequestedByUserID: user.ID,
		Date:              date,
		Hour:              10,
		Status:            models.CoverageRequestOpen,
	}
	if err := db.Create(&first).Error; err != nil {
		t.Fatalf("expected first insert to succeed, got: %v", err)
	}

	second := models.ShiftCoverageRequest{
		GroupID:           group.ID,
		RequestedByUserID: user.ID,
		Date:              date,
		Hour:              10,
		Status:            models.CoverageRequestOpen,
	}
	if err := db.Create(&second).Error; err == nil {
		t.Fatal("expected second insert for the same (group, user, date, hour) to be rejected by the unique index, got no error")
	}

	// A cancelled row for the same key must not block a fresh active one.
	if err := db.Model(&first).Update("status", models.CoverageRequestCancelled).Error; err != nil {
		t.Fatalf("failed to cancel first request: %v", err)
	}
	third := models.ShiftCoverageRequest{
		GroupID:           group.ID,
		RequestedByUserID: user.ID,
		Date:              date,
		Hour:              10,
		Status:            models.CoverageRequestOpen,
	}
	if err := db.Create(&third).Error; err != nil {
		t.Fatalf("expected insert after cancelling the conflicting row to succeed, got: %v", err)
	}
}

// TestCreateCustomIndexes_CoverageRequestClaimedUnique verifies the partial
// unique index (idx_coverage_request_claimed_unique) that backstops
// ClaimCoverageRequest's app-level check-then-write logic in
// internal/handlers/schedule_coverage.go. That check is a plain
// count-then-write across different rows (not a conditional update gated on
// a single row's state, unlike the per-request claim update itself), so
// under READ COMMITTED (Postgres in production) two concurrent claims by the
// SAME user for two DIFFERENT open requests at the same (date, hour) can
// both pass the check and both succeed. This test bypasses the app-level
// check entirely (two direct db.Create calls for claimed rows) to isolate
// and confirm the DB-level backstop actually rejects the second insert -
// and that an open (non-claimed) row for the same claimant/date/hour does
// NOT block it, since the index is scoped to status = 'claimed'.
func TestCreateCustomIndexes_CoverageRequestClaimedUnique(t *testing.T) {
	// Use a per-test-run unique shared-cache DSN so this test's tables don't
	// collide with other tests using the same "file::memory:?cache=shared"
	// pattern.
	dsn := fmt.Sprintf("file:coverage_claimed_idx_test_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite db: %v", err)
	}

	if err := db.AutoMigrate(&models.User{}, &models.Group{}, &models.ShiftCoverageRequest{}); err != nil {
		t.Fatalf("failed to automigrate: %v", err)
	}
	if err := createCustomIndexes(db); err != nil {
		t.Fatalf("createCustomIndexes failed: %v", err)
	}

	alice := models.User{Username: "alice", Email: "alice@test.com", Password: "hashed"}
	if err := db.Create(&alice).Error; err != nil {
		t.Fatalf("failed to create user alice: %v", err)
	}
	bob := models.User{Username: "bob", Email: "bob@test.com", Password: "hashed"}
	if err := db.Create(&bob).Error; err != nil {
		t.Fatalf("failed to create user bob: %v", err)
	}
	claimant := models.User{Username: "claimant", Email: "claimant@test.com", Password: "hashed"}
	if err := db.Create(&claimant).Error; err != nil {
		t.Fatalf("failed to create claimant: %v", err)
	}
	group := models.Group{Name: "Dogs"}
	if err := db.Create(&group).Error; err != nil {
		t.Fatalf("failed to create group: %v", err)
	}

	date := time.Date(2026, 8, 18, 0, 0, 0, 0, time.UTC)

	// Two DIFFERENT requests (different requesters), same claimant, same
	// (date, hour). The first claim should succeed.
	first := models.ShiftCoverageRequest{
		GroupID:           group.ID,
		RequestedByUserID: alice.ID,
		Date:              date,
		Hour:              10,
		Status:            models.CoverageRequestClaimed,
		ClaimedByUserID:   &claimant.ID,
	}
	if err := db.Create(&first).Error; err != nil {
		t.Fatalf("expected first claimed insert to succeed, got: %v", err)
	}

	// A second, still-open request for a different requester at the same
	// (date, hour) must not be blocked by the claimed-only index.
	second := models.ShiftCoverageRequest{
		GroupID:           group.ID,
		RequestedByUserID: bob.ID,
		Date:              date,
		Hour:              10,
		Status:            models.CoverageRequestOpen,
	}
	if err := db.Create(&second).Error; err != nil {
		t.Fatalf("expected second (still-open) insert to succeed, got: %v", err)
	}

	// Now claim the second request with the SAME claimant at the same
	// (date, hour) as the first claim - this is exactly the double-claim
	// race the index defends against, and must be rejected.
	if err := db.Model(&second).Updates(map[string]interface{}{
		"status":             models.CoverageRequestClaimed,
		"claimed_by_user_id": claimant.ID,
	}).Error; err == nil {
		t.Fatal("expected second claim for the same (claimant, date, hour) to be rejected by the unique index, got no error")
	}
}

func TestDBLogLevel_Parsing(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		expected logger.LogLevel
	}{
		{"silent", "silent", logger.Silent},
		{"error", "error", logger.Error},
		{"warn", "warn", logger.Warn},
		{"warning", "warning", logger.Warn},
		{"info", "info", logger.Info},
		{"empty default", "", logger.Warn},
		{"invalid default", "invalid", logger.Warn},
		{"case insensitive uppercase", "WARN", logger.Warn},
		{"case insensitive mixed", "SiLeNt", logger.Silent},
		{"case insensitive info", "INFO", logger.Info},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original env and restore after test
			originalDBLogLevel := os.Getenv("DB_LOG_LEVEL")
			defer func() {
				if originalDBLogLevel != "" {
					os.Setenv("DB_LOG_LEVEL", originalDBLogLevel)
				} else {
					os.Unsetenv("DB_LOG_LEVEL")
				}
			}()

			// Set test env value
			if tt.envValue != "" {
				os.Setenv("DB_LOG_LEVEL", tt.envValue)
			} else {
				os.Unsetenv("DB_LOG_LEVEL")
			}

			// Parse log level using same logic as Initialize()
			var logLevel logger.LogLevel
			switch strings.ToLower(os.Getenv("DB_LOG_LEVEL")) {
			case "silent":
				logLevel = logger.Silent
			case "error":
				logLevel = logger.Error
			case "warn", "warning":
				logLevel = logger.Warn
			case "info":
				logLevel = logger.Info
			default:
				logLevel = logger.Warn
			}

			if logLevel != tt.expected {
				t.Errorf("DB_LOG_LEVEL=%q: expected log level %v, got %v", tt.envValue, tt.expected, logLevel)
			}
		})
	}
}

// TestLocalTimeIsUTC guards this package's init() (see database.go): if
// that line is ever removed, this is the only test in the whole repo that
// would actually catch it. The *_postgres_test.go files that exercise the
// real bug this fixes (a timestamptz round-tripping through pgx comes back
// in time.Local, not UTC) only run in CI's e2e job, which never runs `go
// test`, and even where they do run, GitHub Actions runners default to UTC
// - so they'd pass identically whether or not the fix is in place. This
// test needs no Postgres and no particular process timezone to be
// meaningful, so it runs (and actually enforces something) in every `go
// test ./...` invocation, everywhere.
func TestLocalTimeIsUTC(t *testing.T) {
	// This is a pointer-identity check, not a name/offset check - on an
	// already-UTC host, a deleted init() would still print "time.Local =
	// UTC" (Go's fallback local-zone name), which reads as a passing value
	// even though it's the wrong *Location. Printing both pointers makes a
	// failure unambiguous regardless of the host's own timezone.
	if time.Local != time.UTC {
		t.Fatalf("time.Local (name %v, pointer %p) is not the time.UTC Location (pointer %p) - this package's init() must force time.Local = time.UTC so gorm.io/driver/postgres (via pgx) decodes timestamptz columns as UTC instead of the process's local timezone", time.Local, time.Local, time.UTC)
	}
}

func TestConfigureTracing_RegistersPluginWithoutError(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("failed to open in-memory sqlite db: %v", err)
	}

	if err := configureTracing(db); err != nil {
		t.Fatalf("configureTracing returned error: %v", err)
	}
}
