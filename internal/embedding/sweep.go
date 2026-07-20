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

// embedAndPersist is the shared core of every sweep function: given the
// stale rows' IDs, the row's updated_at at the time it was queried (see the
// UPDATE guard below), and pre-built embedding text (which genuinely
// differs per resource — animals concatenate five fields, comments and
// updates fewer — so that part stays in each resource's own sweep
// function), it embeds the batch and writes each vector back, one UPDATE
// per row. Factored out so a fix to retry/logging/vector-count-mismatch
// handling only needs to be made once instead of risking the three
// resources' sweep functions drifting out of sync with each other. `table`
// is always one of this package's own hardcoded constants ("animals",
// "animal_comments", "updates") — never user input — so building it into
// the UPDATE statement is safe.
//
// The "AND updated_at = ?" guard is an optimistic-concurrency check: if the
// row was edited again between this sweep tick's query and this write (by
// the user, or by a write-path embed goroutine that won the race), the
// captured updated_at no longer matches, the UPDATE matches zero rows, and
// this stale embed attempt is silently discarded rather than overwriting
// newer content with an outdated vector. The row stays (or becomes) stale
// under the normal embedding_updated_at < updated_at check, so the next
// sweep tick retries it against whatever the row looks like by then.
func embedAndPersist(ctx context.Context, span trace.Span, db *gorm.DB, embedder Embedder, table string, ids []uint, updatedAts []time.Time, texts []string) {
	if len(texts) == 0 {
		return
	}

	vectors, err := embedder.EmbedDocuments(ctx, texts)
	if err != nil {
		telemetry.Fail(span, fmt.Errorf("failed to embed %s: %w", table, err), "embed failed")
		return
	}
	if len(vectors) != len(ids) {
		telemetry.Fail(span, fmt.Errorf("embedder returned %d vectors for %d %s", len(vectors), len(ids), table), "vector count mismatch")
		return
	}

	updateSQL := fmt.Sprintf("UPDATE %s SET embedding = ?, embedding_updated_at = now() WHERE id = ? AND updated_at = ?", table)
	for i, id := range ids {
		if err := db.Exec(updateSQL, pgvector.NewVector(vectors[i]), id, updatedAts[i]).Error; err != nil {
			logging.WithField("error", err.Error()).Warn(fmt.Sprintf("Failed to persist %s embedding during sweep", table))
		}
	}
}

type staleAnimal struct {
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

	var rows []staleAnimal
	if err := db.Model(&models.Animal{}).
		Select("id, updated_at, name, species, breed, description, trainer_notes").
		Where("embedding IS NULL OR embedding_updated_at < updated_at").
		Limit(sweepBatchSize).
		Find(&rows).Error; err != nil {
		telemetry.Fail(span, fmt.Errorf("failed to query stale animal embeddings: %w", err), "query failed")
		return
	}
	if len(rows) == 0 {
		return
	}

	ids := make([]uint, len(rows))
	updatedAts := make([]time.Time, len(rows))
	texts := make([]string, len(rows))
	for i, r := range rows {
		ids[i] = r.ID
		updatedAts[i] = r.UpdatedAt
		texts[i] = r.Name + " " + r.Species + " " + r.Breed + " " + r.Description + " " + r.TrainerNotes
	}
	embedAndPersist(ctx, span, db, embedder, "animals", ids, updatedAts, texts)
}

type staleComment struct {
	ID        uint
	UpdatedAt time.Time
	Content   string
}

func sweepComments(db *gorm.DB, embedder Embedder) {
	ctx, span := tracer.Start(context.Background(), "embedding.sweep.comments")
	defer span.End()

	var rows []staleComment
	if err := db.Model(&models.AnimalComment{}).
		Select("id, updated_at, content").
		Where("embedding IS NULL OR embedding_updated_at < updated_at").
		Limit(sweepBatchSize).
		Find(&rows).Error; err != nil {
		telemetry.Fail(span, fmt.Errorf("failed to query stale comment embeddings: %w", err), "query failed")
		return
	}
	if len(rows) == 0 {
		return
	}

	ids := make([]uint, len(rows))
	updatedAts := make([]time.Time, len(rows))
	texts := make([]string, len(rows))
	for i, r := range rows {
		ids[i] = r.ID
		updatedAts[i] = r.UpdatedAt
		texts[i] = r.Content
	}
	embedAndPersist(ctx, span, db, embedder, "animal_comments", ids, updatedAts, texts)
}

type staleUpdate struct {
	ID        uint
	UpdatedAt time.Time
	Title     string
	Content   string
}

func sweepUpdates(db *gorm.DB, embedder Embedder) {
	ctx, span := tracer.Start(context.Background(), "embedding.sweep.updates")
	defer span.End()

	var rows []staleUpdate
	if err := db.Model(&models.Update{}).
		Select("id, updated_at, title, content").
		Where("embedding IS NULL OR embedding_updated_at < updated_at").
		Limit(sweepBatchSize).
		Find(&rows).Error; err != nil {
		telemetry.Fail(span, fmt.Errorf("failed to query stale update embeddings: %w", err), "query failed")
		return
	}
	if len(rows) == 0 {
		return
	}

	ids := make([]uint, len(rows))
	updatedAts := make([]time.Time, len(rows))
	texts := make([]string, len(rows))
	for i, r := range rows {
		ids[i] = r.ID
		updatedAts[i] = r.UpdatedAt
		texts[i] = r.Title + " " + r.Content
	}
	embedAndPersist(ctx, span, db, embedder, "updates", ids, updatedAts, texts)
}
