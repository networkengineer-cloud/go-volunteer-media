package database

import (
	"net"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// Mirrors internal/handlers/search_postgres_test.go's gating pattern: skip
// entirely if no Postgres is reachable, so `go test ./...` stays green
// without a database.
func openDatabaseTestPostgres(t *testing.T) *gorm.DB {
	t.Helper()

	host := envOrDefault("DB_HOST", "localhost")
	port := envOrDefault("DB_PORT", "5432")

	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, port), 2*time.Second)
	if err != nil {
		t.Skipf("skipping: no Postgres reachable at %s:%s", host, port)
	}
	_ = conn.Close()

	user := envOrDefault("DB_USER", "postgres")
	password := envOrDefault("DB_PASSWORD", "postgres")
	dbname := envOrDefault("DB_NAME", "volunteer_media_test")
	sslmode := envOrDefault("DB_SSLMODE", "disable")

	dsn := "host=" + host + " port=" + port + " user=" + user + " password=" + password +
		" dbname=" + dbname + " sslmode=" + sslmode + " connect_timeout=5"

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err != nil {
		t.Skipf("skipping: could not connect to postgres database %q (%v)", dbname, err)
	}
	return db
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func TestCreateCustomIndexes_CreatesEmbeddingColumnsAndIndexes(t *testing.T) {
	db := openDatabaseTestPostgres(t)

	if err := RunMigrations(db); err != nil {
		t.Fatalf("RunMigrations failed: %v", err)
	}

	var columnExists bool
	assert.NoError(t, db.Raw(`
		SELECT EXISTS (
			SELECT 1 FROM information_schema.columns
			WHERE table_name = 'animals' AND column_name = 'embedding'
		)
	`).Scan(&columnExists).Error)
	assert.True(t, columnExists, "animals.embedding column must exist after migration")

	var commentColumnExists bool
	assert.NoError(t, db.Raw(`
		SELECT EXISTS (
			SELECT 1 FROM information_schema.columns
			WHERE table_name = 'animal_comments' AND column_name = 'embedding'
		)
	`).Scan(&commentColumnExists).Error)
	assert.True(t, commentColumnExists, "animal_comments.embedding column must exist after migration")

	var animalsEmbeddingUpdatedAtExists bool
	assert.NoError(t, db.Raw(`
		SELECT EXISTS (
			SELECT 1 FROM information_schema.columns
			WHERE table_name = 'animals' AND column_name = 'embedding_updated_at'
		)
	`).Scan(&animalsEmbeddingUpdatedAtExists).Error)
	assert.True(t, animalsEmbeddingUpdatedAtExists, "animals.embedding_updated_at column must exist after migration")

	var commentEmbeddingUpdatedAtExists bool
	assert.NoError(t, db.Raw(`
		SELECT EXISTS (
			SELECT 1 FROM information_schema.columns
			WHERE table_name = 'animal_comments' AND column_name = 'embedding_updated_at'
		)
	`).Scan(&commentEmbeddingUpdatedAtExists).Error)
	assert.True(t, commentEmbeddingUpdatedAtExists, "animal_comments.embedding_updated_at column must exist after migration")

	var animalsIndexExists bool
	assert.NoError(t, db.Raw(`
		SELECT EXISTS (SELECT 1 FROM pg_indexes WHERE indexname = 'idx_animals_embedding')
	`).Scan(&animalsIndexExists).Error)
	assert.True(t, animalsIndexExists, "idx_animals_embedding HNSW index must exist")

	var commentsIndexExists bool
	assert.NoError(t, db.Raw(`
		SELECT EXISTS (SELECT 1 FROM pg_indexes WHERE indexname = 'idx_animal_comments_embedding')
	`).Scan(&commentsIndexExists).Error)
	assert.True(t, commentsIndexExists, "idx_animal_comments_embedding HNSW index must exist")

	var updatesSearchVectorExists bool
	assert.NoError(t, db.Raw(`
		SELECT EXISTS (
			SELECT 1 FROM information_schema.columns
			WHERE table_name = 'updates' AND column_name = 'search_vector'
		)
	`).Scan(&updatesSearchVectorExists).Error)
	assert.True(t, updatesSearchVectorExists, "updates.search_vector column must exist after migration")

	var updatesEmbeddingExists bool
	assert.NoError(t, db.Raw(`
		SELECT EXISTS (
			SELECT 1 FROM information_schema.columns
			WHERE table_name = 'updates' AND column_name = 'embedding'
		)
	`).Scan(&updatesEmbeddingExists).Error)
	assert.True(t, updatesEmbeddingExists, "updates.embedding column must exist after migration")

	var updatesEmbeddingIndexExists bool
	assert.NoError(t, db.Raw(`
		SELECT EXISTS (SELECT 1 FROM pg_indexes WHERE indexname = 'idx_updates_embedding')
	`).Scan(&updatesEmbeddingIndexExists).Error)
	assert.True(t, updatesEmbeddingIndexExists, "idx_updates_embedding HNSW index must exist")
}
