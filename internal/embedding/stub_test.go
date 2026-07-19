package embedding

import (
	"context"
	"errors"
	"testing"
)

func TestStubEmbedder_DeterministicPerText(t *testing.T) {
	s := &StubEmbedder{}
	ctx := context.Background()

	v1, err := s.EmbedDocument(ctx, "resource guarding around food")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	v2, err := s.EmbedDocument(ctx, "resource guarding around food")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(v1) != Dimension {
		t.Fatalf("expected vector of length %d, got %d", Dimension, len(v1))
	}
	for i := range v1 {
		if v1[i] != v2[i] {
			t.Fatalf("expected identical text to produce identical vectors at index %d: %v != %v", i, v1[i], v2[i])
		}
	}
}

func TestStubEmbedder_DifferentTextDiffersVector(t *testing.T) {
	s := &StubEmbedder{}
	ctx := context.Background()

	v1, _ := s.EmbedDocument(ctx, "resource guarding around food")
	v2, _ := s.EmbedDocument(ctx, "loves belly rubs")

	same := true
	for i := range v1 {
		if v1[i] != v2[i] {
			same = false
			break
		}
	}
	if same {
		t.Fatal("expected different input text to produce different vectors")
	}
}

func TestStubEmbedder_EmptyTextDoesNotPanic(t *testing.T) {
	s := &StubEmbedder{}
	vec, err := s.EmbedDocument(context.Background(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(vec) != Dimension {
		t.Fatalf("expected vector of length %d for empty input, got %d", Dimension, len(vec))
	}
}

func TestStubEmbedder_ErrPropagates(t *testing.T) {
	wantErr := errors.New("boom")
	s := &StubEmbedder{Err: wantErr}
	ctx := context.Background()

	if _, err := s.EmbedDocument(ctx, "text"); err != wantErr {
		t.Fatalf("EmbedDocument: expected %v, got %v", wantErr, err)
	}
	if _, err := s.EmbedQuery(ctx, "text"); err != wantErr {
		t.Fatalf("EmbedQuery: expected %v, got %v", wantErr, err)
	}
	if _, err := s.EmbedDocuments(ctx, []string{"text"}); err != wantErr {
		t.Fatalf("EmbedDocuments: expected %v, got %v", wantErr, err)
	}
}

func TestStubEmbedder_EmbedDocumentsBatchesInOrder(t *testing.T) {
	s := &StubEmbedder{}
	ctx := context.Background()

	texts := []string{"alpha", "bravo", "charlie"}
	batch, err := s.EmbedDocuments(ctx, texts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(batch) != len(texts) {
		t.Fatalf("expected %d vectors, got %d", len(texts), len(batch))
	}
	for i, text := range texts {
		single, _ := s.EmbedDocument(ctx, text)
		for j := range single {
			if batch[i][j] != single[j] {
				t.Fatalf("batch[%d] does not match single EmbedDocument(%q) at index %d", i, text, j)
			}
		}
	}
}
