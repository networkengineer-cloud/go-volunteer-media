package embedding

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

type voyageTestRequest struct {
	Input           interface{} `json:"input"`
	Model           string      `json:"model"`
	InputType       string      `json:"input_type"`
	OutputDimension int         `json:"output_dimension"`
}

type voyageTestResponseItem struct {
	Embedding []float32 `json:"embedding"`
	Index     int       `json:"index"`
}

type voyageTestResponse struct {
	Data []voyageTestResponseItem `json:"data"`
}

func TestVoyageEmbedder_EmbedDocument_SendsDocumentInputType(t *testing.T) {
	var captured voyageTestRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("expected Authorization header \"Bearer test-key\", got %q", got)
		}
		resp := voyageTestResponse{Data: []voyageTestResponseItem{{Embedding: make([]float32, Dimension), Index: 0}}}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	v := NewVoyageEmbedder()
	v.apiKey = "test-key"
	v.apiURL = server.URL

	vec, err := v.EmbedDocument(context.Background(), "resource guarding around food bowls")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(vec) != Dimension {
		t.Fatalf("expected vector of length %d, got %d", Dimension, len(vec))
	}
	if captured.InputType != "document" {
		t.Fatalf("expected input_type \"document\", got %q", captured.InputType)
	}
	if captured.OutputDimension != Dimension {
		t.Fatalf("expected output_dimension %d, got %d", Dimension, captured.OutputDimension)
	}
}

func TestVoyageEmbedder_EmbedQuery_SendsQueryInputType(t *testing.T) {
	var captured voyageTestRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&captured)
		resp := voyageTestResponse{Data: []voyageTestResponseItem{{Embedding: make([]float32, Dimension), Index: 0}}}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	v := NewVoyageEmbedder()
	v.apiKey = "test-key"
	v.apiURL = server.URL

	if _, err := v.EmbedQuery(context.Background(), "resource guarding"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if captured.InputType != "query" {
		t.Fatalf("expected input_type \"query\", got %q", captured.InputType)
	}
}

func TestVoyageEmbedder_EmbedDocuments_SendsBatchAndReturnsInOrder(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req voyageTestRequest
		json.NewDecoder(r.Body).Decode(&req)
		inputs, _ := req.Input.([]interface{})
		items := make([]voyageTestResponseItem, len(inputs))
		for i := range inputs {
			vec := make([]float32, Dimension)
			vec[0] = float32(i) // distinguishable per index
			items[i] = voyageTestResponseItem{Embedding: vec, Index: i}
		}
		resp := voyageTestResponse{Data: items}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	v := NewVoyageEmbedder()
	v.apiKey = "test-key"
	v.apiURL = server.URL

	vecs, err := v.EmbedDocuments(context.Background(), []string{"a", "b", "c"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(vecs) != 3 {
		t.Fatalf("expected 3 vectors, got %d", len(vecs))
	}
	for i := range vecs {
		if vecs[i][0] != float32(i) {
			t.Fatalf("expected vecs[%d][0] == %d, got %v", i, i, vecs[i][0])
		}
	}
}

func TestVoyageEmbedder_EmbedDocuments_ShortResponseReturnsError(t *testing.T) {
	// Simulates Voyage returning fewer embeddings than requested (e.g. a
	// partial batch failure that still responds 200 with a shorter data
	// array) — must fail loudly rather than silently returning an
	// undersized slice a caller could index past the end of.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		items := []voyageTestResponseItem{
			{Embedding: make([]float32, Dimension), Index: 0},
			{Embedding: make([]float32, Dimension), Index: 1},
		}
		resp := voyageTestResponse{Data: items}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	v := NewVoyageEmbedder()
	v.apiKey = "test-key"
	v.apiURL = server.URL

	if _, err := v.EmbedDocuments(context.Background(), []string{"a", "b", "c"}); err == nil {
		t.Fatal("expected an error when Voyage returns fewer embeddings than requested texts")
	}
}

func TestVoyageEmbedder_EmbedDocument_WrongDimensionReturnsError(t *testing.T) {
	// Simulates Voyage returning a vector shorter than the configured
	// Dimension (e.g. a model/config mismatch) — must fail fast here
	// rather than let a wrong-width vector reach PersistEmbedding, where it
	// would fail against the fixed-width vector(Dimension) column and leave
	// the reconciliation sweep retrying (and re-failing) forever.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := voyageTestResponse{Data: []voyageTestResponseItem{{Embedding: make([]float32, Dimension/2), Index: 0}}}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	v := NewVoyageEmbedder()
	v.apiKey = "test-key"
	v.apiURL = server.URL

	if _, err := v.EmbedDocument(context.Background(), "resource guarding"); err == nil {
		t.Fatal("expected an error when Voyage returns a wrong-dimension embedding")
	}
}

func TestVoyageEmbedder_EmbedDocuments_WrongDimensionReturnsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		items := []voyageTestResponseItem{
			{Embedding: make([]float32, Dimension), Index: 0},
			{Embedding: make([]float32, Dimension/2), Index: 1}, // wrong dimension
		}
		resp := voyageTestResponse{Data: items}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	v := NewVoyageEmbedder()
	v.apiKey = "test-key"
	v.apiURL = server.URL

	if _, err := v.EmbedDocuments(context.Background(), []string{"a", "b"}); err == nil {
		t.Fatal("expected an error when Voyage returns a wrong-dimension embedding in a batch")
	}
}

func TestVoyageEmbedder_NonOKResponse_ReturnsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error":"rate limited"}`))
	}))
	defer server.Close()

	v := NewVoyageEmbedder()
	v.apiKey = "test-key"
	v.apiURL = server.URL

	if _, err := v.EmbedDocument(context.Background(), "text"); err == nil {
		t.Fatal("expected an error for a non-200 response, got nil")
	}
}

func TestVoyageEmbedder_IsConfigured(t *testing.T) {
	v := NewVoyageEmbedder()
	v.apiKey = ""
	if v.IsConfigured() {
		t.Fatal("expected IsConfigured to be false with no API key")
	}
	v.apiKey = "test-key"
	if !v.IsConfigured() {
		t.Fatal("expected IsConfigured to be true with an API key set")
	}
}
