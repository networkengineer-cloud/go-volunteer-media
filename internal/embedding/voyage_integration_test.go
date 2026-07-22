package embedding

import (
	"context"
	"os"
	"testing"
)

// A thin real-Voyage integration test, gated behind a real VOYAGE_API_KEY
// (skipped otherwise — same spirit as the Postgres-gated tests elsewhere in
// this codebase) to catch adapter-level breakage (request shape, response
// parsing) without needing a live key in every CI run.
func TestVoyageEmbedder_Integration_EmbedDocument(t *testing.T) {
	if os.Getenv("VOYAGE_API_KEY") == "" {
		t.Skip("skipping: VOYAGE_API_KEY not set")
	}

	v := NewVoyageEmbedder()
	vec, err := v.EmbedDocument(context.Background(), "Loves belly rubs and playing fetch.")
	if err != nil {
		t.Fatalf("EmbedDocument failed: %v", err)
	}
	if len(vec) != Dimension {
		t.Fatalf("expected vector of length %d, got %d", Dimension, len(vec))
	}
}
