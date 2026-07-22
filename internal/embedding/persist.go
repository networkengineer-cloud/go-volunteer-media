package embedding

import (
	"fmt"
	"time"

	"github.com/pgvector/pgvector-go"
	"gorm.io/gorm"
)

// PersistEmbedding writes a single row's embedding vector, used by both the
// write-path embed helpers (internal/handlers/search_embed.go) and the
// reconciliation sweep (sweep.go) — the single source of truth for how an
// embedding gets persisted, so a fix to that logic only needs to be made
// once instead of being kept in sync by hand across both call sites.
//
// table is always one of this package's callers' hardcoded constants
// ("animals", "animal_comments", "updates") — never user input — so
// building it into the UPDATE statement via fmt.Sprintf is safe.
//
// The "AND updated_at = ?" clause is an optimistic-concurrency guard against
// the row version this vector's text was captured from: if the row was
// edited again since (by the user, or by another embed attempt that won a
// race), the captured updatedAt no longer matches, the UPDATE matches zero
// rows, and this write is silently discarded rather than overwriting newer
// content with an outdated vector. The row remains (or becomes) stale under
// the normal "embedding_updated_at < updated_at" check, so whatever next
// re-embeds it — the write path's next attempt, or the sweep — retries
// against the row's current state.
func PersistEmbedding(db *gorm.DB, table string, id uint, updatedAt time.Time, vec []float32) error {
	updateSQL := fmt.Sprintf("UPDATE %s SET embedding = ?, embedding_updated_at = now() WHERE id = ? AND updated_at = ?", table)
	return db.Exec(updateSQL, pgvector.NewVector(vec), id, updatedAt).Error
}

// TouchEmbeddingTimestamp marks a row's embedding as fresh as of updatedAt
// without re-embedding it, for callers that determined the embeddable text
// hasn't actually changed since the last real embed (e.g. a status-only
// animal edit). Without this, the row's embedding_updated_at stays behind
// its now-bumped updated_at, and the reconciliation sweep's staleness check
// ("embedding_updated_at < updated_at") re-embeds the identical text on its
// very next tick — spending exactly the Voyage API call the caller skipped.
//
// Deliberately scoped to "AND embedding IS NOT NULL": a row that was never
// embedded still needs a real embed, not just a timestamp bump, so it's left
// alone here and stays visible to the sweep even if its current text
// happens to match some pre-embedding state. The "AND updated_at = ?" guard
// is the same optimistic-concurrency check PersistEmbedding uses.
func TouchEmbeddingTimestamp(db *gorm.DB, table string, id uint, updatedAt time.Time) error {
	updateSQL := fmt.Sprintf("UPDATE %s SET embedding_updated_at = now() WHERE id = ? AND updated_at = ? AND embedding IS NOT NULL", table)
	return db.Exec(updateSQL, id, updatedAt).Error
}
