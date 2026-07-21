package handlers

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/networkengineer-cloud/go-volunteer-media/internal/embedding"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/logging"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
	"gorm.io/gorm"
)

// embedWriteWG tracks every in-flight write-path embed goroutine spawned by
// embedAsync, so WaitForPendingEmbeds can drain them during graceful
// shutdown — mirroring the reconciliation sweep's own stop/drain mechanism
// (internal/embedding/sweep.go's StartReconciliationSweep). Without this, a
// goroutine spawned by a request that returns just before shutdown could
// still be calling embedding.PersistEmbedding after cmd/api/main.go closes
// the DB connection pool.
var embedWriteWG sync.WaitGroup

// embedWriteDrainTimeout bounds how long WaitForPendingEmbeds waits for
// in-flight embed goroutines to finish before giving up, mirroring
// sweepStopTimeout's bounded wait in internal/embedding/sweep.go.
const embedWriteDrainTimeout = 10 * time.Second

// WaitForPendingEmbeds blocks (up to embedWriteDrainTimeout) until every
// write-path embed goroutine spawned so far has finished. Call during
// graceful shutdown, after the HTTP server has stopped accepting new
// requests but before closing the DB connection pool.
func WaitForPendingEmbeds() {
	done := make(chan struct{})
	go func() {
		embedWriteWG.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(embedWriteDrainTimeout):
		logging.Warn(fmt.Sprintf("Write-path embed goroutines did not finish within %s of shutdown signal; proceeding with shutdown anyway", embedWriteDrainTimeout))
	}
}

// animalEmbeddingText builds the same searchable text used by the
// search_vector generated column (name/species/breed/description/
// trainer_notes), so keyword and semantic search index the same content.
// Delegates to embedding.AnimalEmbeddingText, the single source of truth
// for the formula shared with the reconciliation sweep.
func animalEmbeddingText(animal models.Animal) string {
	return embedding.AnimalEmbeddingText(animal.Name, animal.Species, animal.Breed, animal.Description, animal.TrainerNotes)
}

// updateEmbeddingText builds the same searchable text used by updates'
// search_vector generated column (title + content). Delegates to
// embedding.UpdateEmbeddingText, the single source of truth for the formula
// shared with the reconciliation sweep.
func updateEmbeddingText(update models.Update) string {
	return embedding.UpdateEmbeddingText(update.Title, update.Content)
}

// embedWriteConcurrencyLimit bounds how many write-path embed goroutines can
// be mid-flight at once, mirroring the reconciliation sweep's sweepBatchSize
// discipline (internal/embedding/sweep.go) — without a cap, a burst of
// concurrent writes (e.g. many users commenting at once) would fire as many
// unbounded concurrent Voyage API requests as there are writes, with no
// backpressure.
const embedWriteConcurrencyLimit = 10

var embedWriteSemaphore = make(chan struct{}, embedWriteConcurrencyLimit)

// embedNow is the shared synchronous core of embedAnimalNow/embedCommentNow/
// embedUpdateNow: embeds the given text and persists it via
// embedding.PersistEmbedding (the same function the reconciliation sweep
// uses, see internal/embedding/sweep.go's embedAndPersist), so a fix to
// embed/persist handling only needs to be made in one place instead of
// risking the three resources' write-path helpers drifting out of sync.
//
// embedding.PersistEmbedding's "AND updated_at = ?" guard is an
// optimistic-concurrency check against the row version this specific text
// was captured from: two edits to the same row in quick succession each
// spawn their own goroutine, and nothing else serializes them, so the
// earlier edit's (slower) embed call can complete after the later edit's
// (faster) one. Without this guard, the earlier goroutine's write would
// silently overwrite the newer embedding with a stale one while still
// stamping embedding_updated_at = now() — making the row look "freshly
// embedded" to the sweep's staleness check (embedding_updated_at <
// updated_at) even though it holds outdated content. With the guard, a
// write whose captured version no longer matches the row's current
// updated_at simply matches zero rows and is silently skipped; the sweep's
// normal staleness check picks the row up again later exactly as it would
// for any other stale row.
func embedNow(rawDB *gorm.DB, embedder embedding.Embedder, table, resourceName string, id uint, updatedAt time.Time, text string) error {
	vec, err := embedder.EmbedDocument(context.Background(), text)
	if err != nil {
		return fmt.Errorf("failed to embed %s %d: %w", resourceName, id, err)
	}
	if err := embedding.PersistEmbedding(rawDB, table, id, updatedAt, vec); err != nil {
		return fmt.Errorf("failed to persist embedding for %s %d: %w", resourceName, id, err)
	}
	return nil
}

// embedAsync runs embedNow in a detached goroutine that outlives the
// request, bounded by embedWriteSemaphore. rawDB must be the unscoped
// *gorm.DB (not middleware.GetDB(c, db)) since the request context is
// canceled the instant the handler returns — see the same pattern in
// animal_crud.go's sendQuarantineNotificationEmail usage. Failures are
// logged and left for the reconciliation sweep to retry; never surfaced to
// the write request.
func embedAsync(rawDB *gorm.DB, embedder embedding.Embedder, table, resourceName string, id uint, updatedAt time.Time, text string) {
	if !embedding.Usable(embedder) {
		return
	}
	embedWriteWG.Go(func() {
		embedWriteSemaphore <- struct{}{}
		defer func() { <-embedWriteSemaphore }()
		if err := embedNow(rawDB, embedder, table, resourceName, id, updatedAt, text); err != nil {
			logging.WithField("error", err.Error()).Warn(fmt.Sprintf("Failed to embed %s on write; reconciliation sweep will retry", resourceName))
		}
	})
}

// embedAnimalAsync embeds an animal's searchable text and persists it
// asynchronously. See embedAsync's doc comment for the concurrency/rawDB
// contract.
func embedAnimalAsync(rawDB *gorm.DB, embedder embedding.Embedder, animal models.Animal) {
	embedAsync(rawDB, embedder, "animals", "animal", animal.ID, animal.UpdatedAt, animalEmbeddingText(animal))
}

// embedAnimalNow is embedAnimalAsync's synchronous core, factored out so
// tests can call it directly instead of racing a goroutine.
func embedAnimalNow(rawDB *gorm.DB, embedder embedding.Embedder, animal models.Animal) error {
	return embedNow(rawDB, embedder, "animals", "animal", animal.ID, animal.UpdatedAt, animalEmbeddingText(animal))
}

// embedCommentAsync mirrors embedAnimalAsync for comments.
func embedCommentAsync(rawDB *gorm.DB, embedder embedding.Embedder, comment models.AnimalComment) {
	embedAsync(rawDB, embedder, "animal_comments", "comment", comment.ID, comment.UpdatedAt, comment.Content)
}

// embedCommentNow is embedCommentAsync's synchronous core.
func embedCommentNow(rawDB *gorm.DB, embedder embedding.Embedder, comment models.AnimalComment) error {
	return embedNow(rawDB, embedder, "animal_comments", "comment", comment.ID, comment.UpdatedAt, comment.Content)
}

// embedUpdateAsync mirrors embedAnimalAsync for updates. Update has no edit
// endpoint (create/delete only), so this is only ever called from CreateUpdate.
func embedUpdateAsync(rawDB *gorm.DB, embedder embedding.Embedder, update models.Update) {
	embedAsync(rawDB, embedder, "updates", "update", update.ID, update.UpdatedAt, updateEmbeddingText(update))
}

// embedUpdateNow is embedUpdateAsync's synchronous core. (Update has no edit
// endpoint, so the optimistic-concurrency race embedNow guards against is
// theoretical for updates specifically today — kept consistent with the
// other two resources anyway, since a future edit endpoint would otherwise
// silently reintroduce the gap this guard closes.)
func embedUpdateNow(rawDB *gorm.DB, embedder embedding.Embedder, update models.Update) error {
	return embedNow(rawDB, embedder, "updates", "update", update.ID, update.UpdatedAt, updateEmbeddingText(update))
}
