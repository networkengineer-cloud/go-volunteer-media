// Package embedding provides semantic-search embedding generation, behind a
// provider-agnostic Embedder interface, and the SEMANTIC_SEARCH_ENABLED
// feature flag that gates every Voyage API call in the system (write-path
// embedding, the reconciliation sweep, and query-time search embedding).
package embedding

import (
	"context"
	"os"
)

// Dimension is the embedding vector size used everywhere: the Voyage model
// configuration, the animals/animal_comments.embedding pgvector columns, and
// the HNSW indexes on them. All four must agree — changing this requires a
// schema migration and a full re-embed of the corpus, not just a config edit.
const Dimension = 1024

// Embedder generates embedding vectors for search indexing and querying.
// Document and query embedding are separate methods (not the same call)
// because Voyage's retrieval-tuned models produce better results when told
// which side of a search a piece of text is on ("document" being indexed vs.
// "query" being searched for) — see VoyageEmbedder's input_type usage.
type Embedder interface {
	// EmbedDocument embeds text being indexed (an animal's searchable
	// fields, or a comment's content) for storage.
	EmbedDocument(ctx context.Context, text string) ([]float32, error)
	// EmbedDocuments batches EmbedDocument for multiple texts in one API
	// call — used by the reconciliation sweep to embed a batch of stale
	// rows at once instead of one request per row.
	EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error)
	// EmbedQuery embeds a user's search string at query time.
	EmbedQuery(ctx context.Context, text string) ([]float32, error)
}

// SemanticSearchEnabled is the single source of truth for the
// SEMANTIC_SEARCH_ENABLED flag's parsing, so its three call sites (the
// write-path embed goroutine, the reconciliation sweep, and the search
// handler) can't disagree on what "enabled" means. Matches the existing
// EMAIL_ENABLED convention in internal/email/provider.go: unset or any value
// other than "false"/"0" means enabled.
func SemanticSearchEnabled() bool {
	v := os.Getenv("SEMANTIC_SEARCH_ENABLED")
	return v != "false" && v != "0"
}
