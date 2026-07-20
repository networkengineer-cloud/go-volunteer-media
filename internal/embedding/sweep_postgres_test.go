package embedding

import (
	"context"
	"fmt"
	"net"
	"os"
	"runtime"
	"sync/atomic"
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

func TestSweepComments_ExcludesCommentsOnDeletedAnimal(t *testing.T) {
	db := openSweepTestPostgres(t)
	tx := db.Begin()
	defer tx.Rollback()

	group := models.Group{Name: fmt.Sprintf("SweepTest-%d", time.Now().UnixNano())}
	if err := tx.Create(&group).Error; err != nil {
		t.Fatalf("create group: %v", err)
	}
	user := models.User{Username: fmt.Sprintf("sweeptest-%d", time.Now().UnixNano()), Email: fmt.Sprintf("sweeptest-%d@example.com", time.Now().UnixNano()), Password: "x"}
	if err := tx.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	keptAnimal := models.Animal{GroupID: group.ID, Name: "Buddy", Species: "Dog", Status: "available"}
	if err := tx.Create(&keptAnimal).Error; err != nil {
		t.Fatalf("create kept animal: %v", err)
	}
	deletedAnimal := models.Animal{GroupID: group.ID, Name: "Ghost", Species: "Dog", Status: "available"}
	if err := tx.Create(&deletedAnimal).Error; err != nil {
		t.Fatalf("create animal to be deleted: %v", err)
	}

	keptComment := models.AnimalComment{AnimalID: keptAnimal.ID, UserID: user.ID, Content: "Great playgroup session today."}
	if err := tx.Create(&keptComment).Error; err != nil {
		t.Fatalf("create comment on kept animal: %v", err)
	}
	orphanedComment := models.AnimalComment{AnimalID: deletedAnimal.ID, UserID: user.ID, Content: "Great playgroup session today too."}
	if err := tx.Create(&orphanedComment).Error; err != nil {
		t.Fatalf("create comment on animal to be deleted: %v", err)
	}

	// Soft-delete, exactly as DeleteAnimal does.
	if err := tx.Delete(&deletedAnimal).Error; err != nil {
		t.Fatalf("soft-delete animal: %v", err)
	}

	sweepComments(tx, &StubEmbedder{})

	var keptEmbeddingIsNull bool
	assert.NoError(t, tx.Raw("SELECT embedding IS NULL FROM animal_comments WHERE id = ?", keptComment.ID).Scan(&keptEmbeddingIsNull).Error)
	assert.False(t, keptEmbeddingIsNull, "expected the comment on the kept animal to be embedded")

	var orphanedEmbeddingIsNull bool
	assert.NoError(t, tx.Raw("SELECT embedding IS NULL FROM animal_comments WHERE id = ?", orphanedComment.ID).Scan(&orphanedEmbeddingIsNull).Error)
	assert.True(t, orphanedEmbeddingIsNull, "expected the comment on the deleted animal to be excluded from the sweep, not embedded")
}

func TestSweepUpdates_EmbedsRowsWithNullEmbedding(t *testing.T) {
	db := openSweepTestPostgres(t)
	tx := db.Begin()
	defer tx.Rollback()

	group := models.Group{Name: fmt.Sprintf("SweepTest-%d", time.Now().UnixNano())}
	if err := tx.Create(&group).Error; err != nil {
		t.Fatalf("create group: %v", err)
	}
	user := models.User{Username: "sweeptest", Email: "sweeptest@example.com", Password: "x"}
	if err := tx.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	update := models.Update{GroupID: group.ID, UserID: user.ID, Title: "Playgroup Saturday", Content: "10am at the field."}
	if err := tx.Create(&update).Error; err != nil {
		t.Fatalf("create update: %v", err)
	}

	sweepUpdates(tx, &StubEmbedder{})

	var embeddingIsNull bool
	assert.NoError(t, tx.Raw("SELECT embedding IS NULL FROM updates WHERE id = ?", update.ID).Scan(&embeddingIsNull).Error)
	assert.False(t, embeddingIsNull, "expected sweepUpdates to populate the embedding column")
}

// countingEmbedder wraps another Embedder and atomically counts calls to
// EmbedDocuments, so tests can observe how many sweep ticks actually did
// embedding work.
type countingEmbedder struct {
	Embedder
	calls int64
}

func (c *countingEmbedder) EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error) {
	atomic.AddInt64(&c.calls, 1)
	return c.Embedder.EmbedDocuments(ctx, texts)
}

func (c *countingEmbedder) Calls() int64 {
	return atomic.LoadInt64(&c.calls)
}

// TestStartReconciliationSweep_StopDoesNotLeakGoroutine guards against a
// regression where the sweep's background goroutine keeps running (and its
// ticker keeps firing) after stop() is called — e.g. if the select loop's
// done case were dropped, or if ticker.Stop() were omitted.
//
// It runs against a rolled-back transaction (not the raw db) so the sweep's
// real UPDATEs against animals/animal_comments never escape the test — a
// prior version of this test passed the raw db and permanently overwrote
// live dev-database rows with StubEmbedder junk vectors once the sweep
// goroutine fired for real.
//
// To prove the ticker itself stops (not just that the goroutine exits,
// which NumGoroutine alone can't distinguish from "ticker leaked but the
// goroutine happened to return anyway"), the test seeds a row that stays
// perpetually stale — its updated_at is pinned far in the future, so every
// sweep tick's "embedding_updated_at = now()" write is still older than
// updated_at and the row is re-embedded on every single tick. A counting
// embedder records how many times embedding actually ran. After stop(),
// the count must not increase even after waiting several more tick
// intervals — if it did, the ticker would still be firing.
func TestStartReconciliationSweep_StopDoesNotLeakGoroutine(t *testing.T) {
	db := openSweepTestPostgres(t)
	tx := db.Begin()
	defer tx.Rollback()

	group := models.Group{Name: fmt.Sprintf("SweepLeakTest-%d", time.Now().UnixNano())}
	if err := tx.Create(&group).Error; err != nil {
		t.Fatalf("create group: %v", err)
	}
	animal := models.Animal{GroupID: group.ID, Name: "Rex", Species: "Dog", Status: "available"}
	if err := tx.Create(&animal).Error; err != nil {
		t.Fatalf("create animal: %v", err)
	}
	// Pin updated_at far in the future so the row is always considered
	// stale (embedding_updated_at, set to now() on every sweep tick, is
	// always earlier than this), guaranteeing the embedder is invoked on
	// every tick until the sweep is actually stopped.
	if err := tx.Exec(
		"UPDATE animals SET updated_at = ? WHERE id = ?",
		time.Now().Add(24*time.Hour), animal.ID,
	).Error; err != nil {
		t.Fatalf("failed to pin updated_at: %v", err)
	}

	runtime.GC()
	before := runtime.NumGoroutine()

	embedder := &countingEmbedder{Embedder: &StubEmbedder{}}
	interval := 5 * time.Millisecond
	stop := StartReconciliationSweep(tx, embedder, interval)

	// Wait for at least one real tick to have run the embedder, so we know
	// the sweep is actually ticking before we try to stop it.
	tickDeadline := time.Now().Add(500 * time.Millisecond)
	for embedder.Calls() == 0 {
		if time.Now().After(tickDeadline) {
			t.Fatalf("sweep never invoked the embedder before stop() — test setup didn't produce a stale row")
		}
		time.Sleep(interval)
	}

	stop()

	// Let any tick that was already in flight when stop() was called
	// finish, then snapshot the call count.
	time.Sleep(interval * 2)
	countAtStop := embedder.Calls()

	// Wait past several more tick intervals. If the ticker were still
	// firing (e.g. ticker.Stop() were omitted or the done case dropped),
	// the always-stale row would keep incrementing the counter.
	time.Sleep(interval * 10)
	assert.Equal(t, countAtStop, embedder.Calls(), "embedder was invoked again after stop() — the ticker did not actually stop")

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
