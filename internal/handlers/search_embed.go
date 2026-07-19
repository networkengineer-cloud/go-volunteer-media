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
	if !embedding.SemanticSearchEnabled() {
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
func embedAnimalNow(rawDB *gorm.DB, embedder embedding.Embedder, animal models.Animal) error {
	vec, err := embedder.EmbedDocument(context.Background(), animalEmbeddingText(animal))
	if err != nil {
		return fmt.Errorf("failed to embed animal %d: %w", animal.ID, err)
	}
	if err := rawDB.Exec(
		"UPDATE animals SET embedding = ?, embedding_updated_at = now() WHERE id = ?",
		pgvector.NewVector(vec), animal.ID,
	).Error; err != nil {
		return fmt.Errorf("failed to persist embedding for animal %d: %w", animal.ID, err)
	}
	return nil
}

// embedCommentAsync mirrors embedAnimalAsync for comments.
func embedCommentAsync(rawDB *gorm.DB, embedder embedding.Embedder, comment models.AnimalComment) {
	if !embedding.SemanticSearchEnabled() {
		return
	}
	go func() {
		if err := embedCommentNow(rawDB, embedder, comment); err != nil {
			logging.WithField("error", err.Error()).Warn("Failed to embed comment on write; reconciliation sweep will retry")
		}
	}()
}

// embedCommentNow is embedCommentAsync's synchronous core.
func embedCommentNow(rawDB *gorm.DB, embedder embedding.Embedder, comment models.AnimalComment) error {
	vec, err := embedder.EmbedDocument(context.Background(), comment.Content)
	if err != nil {
		return fmt.Errorf("failed to embed comment %d: %w", comment.ID, err)
	}
	if err := rawDB.Exec(
		"UPDATE animal_comments SET embedding = ?, embedding_updated_at = now() WHERE id = ?",
		pgvector.NewVector(vec), comment.ID,
	).Error; err != nil {
		return fmt.Errorf("failed to persist embedding for comment %d: %w", comment.ID, err)
	}
	return nil
}
