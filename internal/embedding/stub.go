package embedding

import "context"

// StubEmbedder is a deterministic, in-memory Embedder for tests — no
// network calls. Identical input text always produces an identical vector
// (useful for asserting exact nearest-neighbor behavior in tests without a
// real embedding model), and different text produces a different vector.
// Set Err to make every call fail, for exercising fallback/error-handling
// paths (query-embedding failure degrading to keyword-only, etc.). Set
// Unconfigured to make IsConfigured() report false, for exercising the
// Usable()-gated degrade path without needing Err (which fails every call
// outright rather than modeling "not configured").
type StubEmbedder struct {
	// Dim overrides the vector length; defaults to Dimension (1024) when zero.
	Dim          int
	Err          error
	Unconfigured bool
}

// IsConfigured always reports true unless Unconfigured is explicitly set —
// StubEmbedder never needs real credentials, so it's "configured" by
// default in every test that doesn't specifically exercise the
// not-configured path. Nil-receiver-safe for the same reason as
// VoyageEmbedder.IsConfigured: a typed-nil *StubEmbedder held in the
// Embedder interface is not == nil, so Usable() relies on this method
// itself to catch that case.
func (s *StubEmbedder) IsConfigured() bool {
	return s != nil && !s.Unconfigured
}

func (s *StubEmbedder) dimension() int {
	if s.Dim == 0 {
		return Dimension
	}
	return s.Dim
}

func (s *StubEmbedder) vectorFor(text string) []float32 {
	vec := make([]float32, s.dimension())
	if len(text) == 0 {
		return vec
	}
	for i := range vec {
		vec[i] = float32(text[i%len(text)]) / 255.0
	}
	return vec
}

func (s *StubEmbedder) EmbedDocument(_ context.Context, text string) ([]float32, error) {
	if s.Err != nil {
		return nil, s.Err
	}
	return s.vectorFor(text), nil
}

func (s *StubEmbedder) EmbedDocuments(_ context.Context, texts []string) ([][]float32, error) {
	if s.Err != nil {
		return nil, s.Err
	}
	out := make([][]float32, len(texts))
	for i, t := range texts {
		out[i] = s.vectorFor(t)
	}
	return out, nil
}

func (s *StubEmbedder) EmbedQuery(ctx context.Context, text string) ([]float32, error) {
	return s.EmbedDocument(ctx, text)
}
