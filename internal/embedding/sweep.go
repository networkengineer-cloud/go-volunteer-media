package embedding

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/networkengineer-cloud/go-volunteer-media/internal/logging"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/telemetry"
	"github.com/pgvector/pgvector-go"
	"gorm.io/gorm"
)

// sweepBatchSize bounds how many rows one sweep tick embeds, per resource
// type. A large first-deploy backfill drains gradually over several ticks
// instead of firing one unbounded request.
const sweepBatchSize = 50

// StartReconciliationSweep runs a periodic background pass that embeds any
// animal/comment/update whose embedding is missing or older than the row's
// last edit. One mechanism covers three situations: initial backfill of
// pre-existing rows on first deploy, retrying rows whose async write-path
// embed attempt failed, and catching up on anything created/edited while
// SEMANTIC_SEARCH_ENABLED was false — none of these need special-casing
// differently from one another. Returns a stop function; call it during
// graceful shutdown to stop the ticker.
func StartReconciliationSweep(db *gorm.DB, embedder Embedder, interval time.Duration) (stop func()) {
	ticker := time.NewTicker(interval)
	done := make(chan struct{})

	go func() {
		for {
			select {
			case <-ticker.C:
				if !SemanticSearchEnabled() {
					continue
				}
				sweepAnimals(db, embedder)
				sweepComments(db, embedder)
				sweepUpdates(db, embedder)
			case <-done:
				ticker.Stop()
				return
			}
		}
	}()

	var once sync.Once
	return func() { once.Do(func() { close(done) }) }
}

type staleAnimal struct {
	ID           uint
	Name         string
	Species      string
	Breed        string
	Description  string
	TrainerNotes string
}

func sweepAnimals(db *gorm.DB, embedder Embedder) {
	ctx, span := tracer.Start(context.Background(), "embedding.sweep.animals")
	defer span.End()

	var rows []staleAnimal
	if err := db.Model(&models.Animal{}).
		Select("id, name, species, breed, description, trainer_notes").
		Where("embedding IS NULL OR embedding_updated_at < updated_at").
		Limit(sweepBatchSize).
		Find(&rows).Error; err != nil {
		telemetry.Fail(span, fmt.Errorf("failed to query stale animal embeddings: %w", err), "query failed")
		return
	}
	if len(rows) == 0 {
		return
	}

	texts := make([]string, len(rows))
	for i, r := range rows {
		texts[i] = r.Name + " " + r.Species + " " + r.Breed + " " + r.Description + " " + r.TrainerNotes
	}

	vectors, err := embedder.EmbedDocuments(ctx, texts)
	if err != nil {
		telemetry.Fail(span, fmt.Errorf("failed to embed animals: %w", err), "embed failed")
		return
	}
	if len(vectors) != len(rows) {
		telemetry.Fail(span, fmt.Errorf("embedder returned %d vectors for %d animals", len(vectors), len(rows)), "vector count mismatch")
		return
	}

	for i, r := range rows {
		if err := db.Exec(
			"UPDATE animals SET embedding = ?, embedding_updated_at = now() WHERE id = ?",
			pgvector.NewVector(vectors[i]), r.ID,
		).Error; err != nil {
			logging.WithField("error", err.Error()).Warn("Failed to persist animal embedding during sweep")
		}
	}
}

type staleComment struct {
	ID      uint
	Content string
}

func sweepComments(db *gorm.DB, embedder Embedder) {
	ctx, span := tracer.Start(context.Background(), "embedding.sweep.comments")
	defer span.End()

	var rows []staleComment
	if err := db.Model(&models.AnimalComment{}).
		Select("id, content").
		Where("embedding IS NULL OR embedding_updated_at < updated_at").
		Limit(sweepBatchSize).
		Find(&rows).Error; err != nil {
		telemetry.Fail(span, fmt.Errorf("failed to query stale comment embeddings: %w", err), "query failed")
		return
	}
	if len(rows) == 0 {
		return
	}

	texts := make([]string, len(rows))
	for i, r := range rows {
		texts[i] = r.Content
	}

	vectors, err := embedder.EmbedDocuments(ctx, texts)
	if err != nil {
		telemetry.Fail(span, fmt.Errorf("failed to embed comments: %w", err), "embed failed")
		return
	}
	if len(vectors) != len(rows) {
		telemetry.Fail(span, fmt.Errorf("embedder returned %d vectors for %d comments", len(vectors), len(rows)), "vector count mismatch")
		return
	}

	for i, r := range rows {
		if err := db.Exec(
			"UPDATE animal_comments SET embedding = ?, embedding_updated_at = now() WHERE id = ?",
			pgvector.NewVector(vectors[i]), r.ID,
		).Error; err != nil {
			logging.WithField("error", err.Error()).Warn("Failed to persist comment embedding during sweep")
		}
	}
}

type staleUpdate struct {
	ID      uint
	Title   string
	Content string
}

func sweepUpdates(db *gorm.DB, embedder Embedder) {
	ctx, span := tracer.Start(context.Background(), "embedding.sweep.updates")
	defer span.End()

	var rows []staleUpdate
	if err := db.Model(&models.Update{}).
		Select("id, title, content").
		Where("embedding IS NULL OR embedding_updated_at < updated_at").
		Limit(sweepBatchSize).
		Find(&rows).Error; err != nil {
		telemetry.Fail(span, fmt.Errorf("failed to query stale update embeddings: %w", err), "query failed")
		return
	}
	if len(rows) == 0 {
		return
	}

	texts := make([]string, len(rows))
	for i, r := range rows {
		texts[i] = r.Title + " " + r.Content
	}

	vectors, err := embedder.EmbedDocuments(ctx, texts)
	if err != nil {
		telemetry.Fail(span, fmt.Errorf("failed to embed updates: %w", err), "embed failed")
		return
	}
	if len(vectors) != len(rows) {
		telemetry.Fail(span, fmt.Errorf("embedder returned %d vectors for %d updates", len(vectors), len(rows)), "vector count mismatch")
		return
	}

	for i, r := range rows {
		if err := db.Exec(
			"UPDATE updates SET embedding = ?, embedding_updated_at = now() WHERE id = ?",
			pgvector.NewVector(vectors[i]), r.ID,
		).Error; err != nil {
			logging.WithField("error", err.Error()).Warn("Failed to persist update embedding during sweep")
		}
	}
}
