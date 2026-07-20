package embedding

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/networkengineer-cloud/go-volunteer-media/internal/logging"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/telemetry"
	"go.opentelemetry.io/otel/trace"
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
				if !Usable(embedder) {
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

// staleRow is one row awaiting embedding: its ID, its updated_at at query
// time (used as PersistEmbedding's optimistic-concurrency guard), and the
// text to embed — kept together in one struct, rather than three parallel
// slices, so they can't desync by index if a future sweep function's
// row-building loop changes shape.
type staleRow struct {
	ID        uint
	UpdatedAt time.Time
	Text      string
}

// embedAndPersist is the shared core of every sweep function: embeds a
// batch of stale rows' text and persists each result via PersistEmbedding
// (the same function the write-path embed helpers in
// internal/handlers/search_embed.go use), so a fix to
// retry/logging/vector-count-mismatch/persistence handling only needs to be
// made in one place instead of risking the three resources' sweep functions
// drifting out of sync with each other. table is the literal SQL table
// name ("animals", "animal_comments", "updates"); resourceName is the
// human-readable word used in log/telemetry messages ("animals",
// "comments", "updates") — kept separate from table because they diverge
// for comments, and an operator's saved log search for "comments" shouldn't
// silently break because the underlying table is called "animal_comments".
func embedAndPersist(ctx context.Context, span trace.Span, db *gorm.DB, embedder Embedder, table, resourceName string, rows []staleRow) {
	if len(rows) == 0 {
		return
	}

	texts := make([]string, len(rows))
	for i, r := range rows {
		texts[i] = r.Text
	}

	vectors, err := embedder.EmbedDocuments(ctx, texts)
	if err != nil {
		telemetry.Fail(span, fmt.Errorf("failed to embed %s: %w", resourceName, err), "embed failed")
		return
	}
	if len(vectors) != len(rows) {
		telemetry.Fail(span, fmt.Errorf("embedder returned %d vectors for %d %s", len(vectors), len(rows), resourceName), "vector count mismatch")
		return
	}

	for i, r := range rows {
		if err := PersistEmbedding(db, table, r.ID, r.UpdatedAt, vectors[i]); err != nil {
			logging.WithField("error", err.Error()).Warn(fmt.Sprintf("Failed to persist %s embedding during sweep", resourceName))
		}
	}
}

type staleAnimalRow struct {
	ID           uint
	UpdatedAt    time.Time
	Name         string
	Species      string
	Breed        string
	Description  string
	TrainerNotes string
}

func sweepAnimals(db *gorm.DB, embedder Embedder) {
	ctx, span := tracer.Start(context.Background(), "embedding.sweep.animals")
	defer span.End()

	var dbRows []staleAnimalRow
	if err := db.Model(&models.Animal{}).
		Select("id, updated_at, name, species, breed, description, trainer_notes").
		Where("embedding IS NULL OR embedding_updated_at < updated_at").
		Limit(sweepBatchSize).
		Find(&dbRows).Error; err != nil {
		telemetry.Fail(span, fmt.Errorf("failed to query stale animal embeddings: %w", err), "query failed")
		return
	}

	rows := make([]staleRow, len(dbRows))
	for i, r := range dbRows {
		rows[i] = staleRow{
			ID:        r.ID,
			UpdatedAt: r.UpdatedAt,
			Text:      AnimalEmbeddingText(r.Name, r.Species, r.Breed, r.Description, r.TrainerNotes),
		}
	}
	embedAndPersist(ctx, span, db, embedder, "animals", "animals", rows)
}

type staleCommentRow struct {
	ID        uint
	UpdatedAt time.Time
	Content   string
}

func sweepComments(db *gorm.DB, embedder Embedder) {
	ctx, span := tracer.Start(context.Background(), "embedding.sweep.comments")
	defer span.End()

	// Joins animals and excludes soft-deleted ones for the same reason
	// internal/handlers/search.go's comment search does: without this,
	// comments belonging to a deleted animal are perpetually re-selected as
	// "stale" (their embedding_updated_at can never catch up to anything
	// meaningful, since they're never actually re-edited) and burn sweep
	// batch slots and Voyage calls embedding content that can never surface
	// in search results.
	var dbRows []staleCommentRow
	if err := db.Model(&models.AnimalComment{}).
		Joins("JOIN animals ON animals.id = animal_comments.animal_id").
		Select("animal_comments.id, animal_comments.updated_at, animal_comments.content").
		Where("animals.deleted_at IS NULL AND (animal_comments.embedding IS NULL OR animal_comments.embedding_updated_at < animal_comments.updated_at)").
		Limit(sweepBatchSize).
		Find(&dbRows).Error; err != nil {
		telemetry.Fail(span, fmt.Errorf("failed to query stale comment embeddings: %w", err), "query failed")
		return
	}

	rows := make([]staleRow, len(dbRows))
	for i, r := range dbRows {
		rows[i] = staleRow{ID: r.ID, UpdatedAt: r.UpdatedAt, Text: r.Content}
	}
	embedAndPersist(ctx, span, db, embedder, "animal_comments", "comments", rows)
}

type staleUpdateRow struct {
	ID        uint
	UpdatedAt time.Time
	Title     string
	Content   string
}

func sweepUpdates(db *gorm.DB, embedder Embedder) {
	ctx, span := tracer.Start(context.Background(), "embedding.sweep.updates")
	defer span.End()

	var dbRows []staleUpdateRow
	if err := db.Model(&models.Update{}).
		Select("id, updated_at, title, content").
		Where("embedding IS NULL OR embedding_updated_at < updated_at").
		Limit(sweepBatchSize).
		Find(&dbRows).Error; err != nil {
		telemetry.Fail(span, fmt.Errorf("failed to query stale update embeddings: %w", err), "query failed")
		return
	}

	rows := make([]staleRow, len(dbRows))
	for i, r := range dbRows {
		rows[i] = staleRow{ID: r.ID, UpdatedAt: r.UpdatedAt, Text: UpdateEmbeddingText(r.Title, r.Content)}
	}
	embedAndPersist(ctx, span, db, embedder, "updates", "updates", rows)
}
