package handlers

import (
	"context"
	"fmt"

	"github.com/networkengineer-cloud/go-volunteer-media/internal/embedding"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/logging"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
	"github.com/pgvector/pgvector-go"
	"gorm.io/gorm"
)

// animalEmbeddingText builds the same searchable text used by the
// search_vector generated column (name/species/breed/description/
// trainer_notes), so keyword and semantic search index the same content.
func animalEmbeddingText(animal models.Animal) string {
	return animal.Name + " " + animal.Species + " " + animal.Breed + " " + animal.Description + " " + animal.TrainerNotes
}

// embedAnimalAsync embeds an animal's searchable text and persists it in a
// detached goroutine that outlives the request. rawDB must be the unscoped
// *gorm.DB (not middleware.GetDB(c, db)) since the request context is
// canceled the instant the handler returns — see the same pattern in
// animal_crud.go's sendQuarantineNotificationEmail usage. Failures are
// logged and left for the reconciliation sweep to retry; never surfaced to
// the write request.
func embedAnimalAsync(rawDB *gorm.DB, embedder embedding.Embedder, animal models.Animal) {
	if !embedding.Usable(embedder) {
		return
	}
	go func() {
		if err := embedAnimalNow(rawDB, embedder, animal); err != nil {
			logging.WithField("error", err.Error()).Warn("Failed to embed animal on write; reconciliation sweep will retry")
		}
	}()
}

// embedAnimalNow is embedAnimalAsync's synchronous core, factored out so
// tests can call it directly instead of racing a goroutine.
//
// The UPDATE's "AND updated_at = ?" guard is an optimistic-concurrency check
// against the row version this specific text was captured from: two edits
// to the same animal in quick succession each spawn their own goroutine, and
// nothing else serializes them, so the earlier edit's (slower) embed call
// can complete after the later edit's (faster) one. Without this guard, the
// earlier goroutine's write would silently overwrite the newer embedding
// with a stale one while still stamping embedding_updated_at = now() —
// making the row look "freshly embedded" to the sweep's staleness check
// (embedding_updated_at < updated_at) even though it holds outdated
// content. With the guard, a write whose captured version no longer matches
// the row's current updated_at simply matches zero rows and is silently
// skipped; the sweep's normal staleness check picks the row up again later
// exactly as it would for any other stale row.
func embedAnimalNow(rawDB *gorm.DB, embedder embedding.Embedder, animal models.Animal) error {
	vec, err := embedder.EmbedDocument(context.Background(), animalEmbeddingText(animal))
	if err != nil {
		return fmt.Errorf("failed to embed animal %d: %w", animal.ID, err)
	}
	if err := rawDB.Exec(
		"UPDATE animals SET embedding = ?, embedding_updated_at = now() WHERE id = ? AND updated_at = ?",
		pgvector.NewVector(vec), animal.ID, animal.UpdatedAt,
	).Error; err != nil {
		return fmt.Errorf("failed to persist embedding for animal %d: %w", animal.ID, err)
	}
	return nil
}

// embedCommentAsync mirrors embedAnimalAsync for comments.
func embedCommentAsync(rawDB *gorm.DB, embedder embedding.Embedder, comment models.AnimalComment) {
	if !embedding.Usable(embedder) {
		return
	}
	go func() {
		if err := embedCommentNow(rawDB, embedder, comment); err != nil {
			logging.WithField("error", err.Error()).Warn("Failed to embed comment on write; reconciliation sweep will retry")
		}
	}()
}

// embedCommentNow is embedCommentAsync's synchronous core. See
// embedAnimalNow's comment for why the UPDATE guards on the row's captured
// updated_at.
func embedCommentNow(rawDB *gorm.DB, embedder embedding.Embedder, comment models.AnimalComment) error {
	vec, err := embedder.EmbedDocument(context.Background(), comment.Content)
	if err != nil {
		return fmt.Errorf("failed to embed comment %d: %w", comment.ID, err)
	}
	if err := rawDB.Exec(
		"UPDATE animal_comments SET embedding = ?, embedding_updated_at = now() WHERE id = ? AND updated_at = ?",
		pgvector.NewVector(vec), comment.ID, comment.UpdatedAt,
	).Error; err != nil {
		return fmt.Errorf("failed to persist embedding for comment %d: %w", comment.ID, err)
	}
	return nil
}

// updateEmbeddingText builds the same searchable text used by updates'
// search_vector generated column (title + content).
func updateEmbeddingText(update models.Update) string {
	return update.Title + " " + update.Content
}

// embedUpdateAsync mirrors embedAnimalAsync for updates. Update has no edit
// endpoint (create/delete only), so this is only ever called from CreateUpdate.
func embedUpdateAsync(rawDB *gorm.DB, embedder embedding.Embedder, update models.Update) {
	if !embedding.Usable(embedder) {
		return
	}
	go func() {
		if err := embedUpdateNow(rawDB, embedder, update); err != nil {
			logging.WithField("error", err.Error()).Warn("Failed to embed update on write; reconciliation sweep will retry")
		}
	}()
}

// embedUpdateNow is embedUpdateAsync's synchronous core. See
// embedAnimalNow's comment for why the UPDATE guards on the row's captured
// updated_at. (Update has no edit endpoint, so this race is theoretical for
// updates specifically today — kept consistent with the other two resources
// anyway, since a future edit endpoint would otherwise silently reintroduce
// the gap this guard closes.)
func embedUpdateNow(rawDB *gorm.DB, embedder embedding.Embedder, update models.Update) error {
	vec, err := embedder.EmbedDocument(context.Background(), updateEmbeddingText(update))
	if err != nil {
		return fmt.Errorf("failed to embed update %d: %w", update.ID, err)
	}
	if err := rawDB.Exec(
		"UPDATE updates SET embedding = ?, embedding_updated_at = now() WHERE id = ? AND updated_at = ?",
		pgvector.NewVector(vec), update.ID, update.UpdatedAt,
	).Error; err != nil {
		return fmt.Errorf("failed to persist embedding for update %d: %w", update.ID, err)
	}
	return nil
}
