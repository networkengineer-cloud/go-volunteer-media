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
	// IsConfigured reports whether this Embedder can actually serve
	// requests (e.g. an API key is present). Combined with
	// SemanticSearchEnabled via Usable — an operator-enabled-but-unconfigured
	// embedder must not be treated as usable, or every write-path embed
	// attempt and every sweep tick would retry and fail forever.
	IsConfigured() bool
}

// SemanticSearchEnabled is the single source of truth for the
// SEMANTIC_SEARCH_ENABLED flag's parsing, so its three call sites (the
// write-path embed goroutine, the reconciliation sweep, and the search
// handler) can't disagree on what "enabled" means.
//
// Deliberately opt-in — unset or any value other than "true"/"1" means
// disabled — unlike EMAIL_ENABLED's opt-out convention in
// internal/email/provider.go. Email is a low-cost, well-understood default;
// Voyage is a paid, usage-billed API, so an operator who sets
// VOYAGE_API_KEY but never separately considers this flag should not
// silently start incurring real outbound API calls.
func SemanticSearchEnabled() bool {
	v := os.Getenv("SEMANTIC_SEARCH_ENABLED")
	return v == "true" || v == "1"
}

// Usable combines the SEMANTIC_SEARCH_ENABLED flag with the embedder's own
// IsConfigured check into the single question every call site actually
// cares about: should this attempt semantic work at all? Neither check
// alone is sufficient — an enabled-but-unconfigured embedder (flag on, no
// API key) or a configured-but-disabled one (API key set, flag left at its
// opt-in default) would each retry and fail indefinitely on every write and
// every sweep tick if only one side were checked. This is the single source
// of truth all five call sites (three write-path embed helpers, the sweep,
// and the search handler's query-time embed) must use so none of them can
// drift out of sync with the others.
func Usable(embedder Embedder) bool {
	return SemanticSearchEnabled() && embedder != nil && embedder.IsConfigured()
}
