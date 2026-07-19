package embedding

import (
	"fmt"
	"net"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/networkengineer-cloud/go-volunteer-media/internal/database"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
	"github.com/pgvector/pgvector-go"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func openSweepTestPostgres(t *testing.T) *gorm.DB {
	t.Helper()

	host := envOrDefault("DB_HOST", "localhost")
	port := envOrDefault("DB_PORT", "5432")

	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, port), 2*time.Second)
	if err != nil {
		t.Skipf("skipping: no Postgres reachable at %s:%s", host, port)
	}
	_ = conn.Close()

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s connect_timeout=5",
		host, port,
		envOrDefault("DB_USER", "postgres"),
		envOrDefault("DB_PASSWORD", "postgres"),
		envOrDefault("DB_NAME", "volunteer_media_test"),
		envOrDefault("DB_SSLMODE", "disable"),
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	if err != nil {
		t.Skipf("skipping: could not connect to postgres (%v)", err)
	}
	if err := database.RunMigrations(db); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}
	return db
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func TestSweepAnimals_EmbedsRowsWithNullEmbedding(t *testing.T) {
	db := openSweepTestPostgres(t)
	tx := db.Begin()
	defer tx.Rollback()

	group := models.Group{Name: fmt.Sprintf("SweepTest-%d", time.Now().UnixNano())}
	if err := tx.Create(&group).Error; err != nil {
		t.Fatalf("create group: %v", err)
	}
	animal := models.Animal{GroupID: group.ID, Name: "Rex", Species: "Dog", Status: "available", Description: "Loves belly rubs."}
	if err := tx.Create(&animal).Error; err != nil {
		t.Fatalf("create animal: %v", err)
	}

	sweepAnimals(tx, &StubEmbedder{})

	var embeddingIsNull bool
	assert.NoError(t, tx.Raw("SELECT embedding IS NULL FROM animals WHERE id = ?", animal.ID).Scan(&embeddingIsNull).Error)
	assert.False(t, embeddingIsNull, "expected sweepAnimals to populate the embedding column")
}

func TestSweepAnimals_ReEmbedsStaleRows(t *testing.T) {
	db := openSweepTestPostgres(t)
	tx := db.Begin()
	defer tx.Rollback()

	group := models.Group{Name: fmt.Sprintf("SweepTest-%d", time.Now().UnixNano())}
	if err := tx.Create(&group).Error; err != nil {
		t.Fatalf("create group: %v", err)
	}
	animal := models.Animal{GroupID: group.ID, Name: "Rex", Species: "Dog", Status: "available"}
	if err := tx.Create(&animal).Error; err != nil {
		t.Fatalf("create animal: %v", err)
	}

	// Simulate an embedding computed before the row's last edit: set
	// embedding_updated_at to a time before updated_at.
	if err := tx.Exec(
		"UPDATE animals SET embedding = ?, embedding_updated_at = ?, updated_at = ? WHERE id = ?",
		pgvector.NewVector(make([]float32, Dimension)), time.Now().Add(-time.Hour), time.Now(), animal.ID,
	).Error; err != nil {
		t.Fatalf("failed to simulate stale embedding: %v", err)
	}

	var embeddedAtBefore time.Time
	assert.NoError(t, tx.Raw("SELECT embedding_updated_at FROM animals WHERE id = ?", animal.ID).Scan(&embeddedAtBefore).Error)

	sweepAnimals(tx, &StubEmbedder{})

	var embeddedAtAfter time.Time
	assert.NoError(t, tx.Raw("SELECT embedding_updated_at FROM animals WHERE id = ?", animal.ID).Scan(&embeddedAtAfter).Error)
	assert.True(t, embeddedAtAfter.After(embeddedAtBefore), "expected the stale embedding to be refreshed")
}

func TestSweepAnimals_RespectsBatchSizeCap(t *testing.T) {
	db := openSweepTestPostgres(t)
	tx := db.Begin()
	defer tx.Rollback()

	group := models.Group{Name: fmt.Sprintf("SweepTest-%d", time.Now().UnixNano())}
	if err := tx.Create(&group).Error; err != nil {
		t.Fatalf("create group: %v", err)
	}
	for i := 0; i < sweepBatchSize+10; i++ {
		a := models.Animal{GroupID: group.ID, Name: fmt.Sprintf("Animal%d", i), Species: "Dog", Status: "available"}
		if err := tx.Create(&a).Error; err != nil {
			t.Fatalf("create animal %d: %v", i, err)
		}
	}

	sweepAnimals(tx, &StubEmbedder{})

	var embeddedCount int64
	assert.NoError(t, tx.Model(&models.Animal{}).Where("group_id = ? AND embedding IS NOT NULL", group.ID).Count(&embeddedCount).Error)
	assert.Equal(t, int64(sweepBatchSize), embeddedCount, "expected exactly one batch's worth of animals to be embedded per sweep call")
}

// TestStartReconciliationSweep_StopDoesNotLeakGoroutine guards against a
// regression where the sweep's background goroutine keeps running (and its
// ticker keeps firing) after stop() is called — e.g. if the select loop's
// done case were dropped, or if ticker.Stop() were omitted.
func TestStartReconciliationSweep_StopDoesNotLeakGoroutine(t *testing.T) {
	db := openSweepTestPostgres(t)

	runtime.GC()
	before := runtime.NumGoroutine()

	stop := StartReconciliationSweep(db, &StubEmbedder{}, 5*time.Millisecond)
	time.Sleep(20 * time.Millisecond) // let the ticker fire at least once
	stop()

	deadline := time.Now().Add(500 * time.Millisecond)
	for {
		runtime.GC()
		after := runtime.NumGoroutine()
		if after <= before {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("goroutine count did not return to baseline after stop(): before=%d after=%d", before, after)
		}
		time.Sleep(10 * time.Millisecond)
	}
}
