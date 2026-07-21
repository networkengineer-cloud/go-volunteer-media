package embedding

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/networkengineer-cloud/go-volunteer-media/internal/telemetry"
)

const (
	defaultVoyageAPIURL = "https://api.voyageai.com/v1/embeddings"
	defaultVoyageModel  = "voyage-4"
)

var tracer = telemetry.Tracer("internal/embedding")

// VoyageEmbedder implements Embedder using Voyage AI's embeddings API.
type VoyageEmbedder struct {
	apiKey string
	model  string
	apiURL string // configurable for testing
	client *http.Client
}

// NewVoyageEmbedder creates a VoyageEmbedder from environment variables
// (VOYAGE_API_KEY required; VOYAGE_MODEL optional, defaults to "voyage-4").
func NewVoyageEmbedder() *VoyageEmbedder {
	model := os.Getenv("VOYAGE_MODEL")
	if model == "" {
		model = defaultVoyageModel
	}
	return &VoyageEmbedder{
		apiKey: os.Getenv("VOYAGE_API_KEY"),
		model:  model,
		apiURL: defaultVoyageAPIURL,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// IsConfigured reports whether an API key is present. Nil-receiver-safe so
// Usable(embedder) can call it even when embedder is a typed-nil
// *VoyageEmbedder held in the Embedder interface (interface != nil in that
// case, so Usable's own nil check can't catch it).
func (v *VoyageEmbedder) IsConfigured() bool {
	return v != nil && v.apiKey != ""
}

type voyageRequest struct {
	Input           interface{} `json:"input"`
	Model           string      `json:"model"`
	InputType       string      `json:"input_type"`
	OutputDimension int         `json:"output_dimension"`
}

type voyageResponseItem struct {
	Embedding []float32 `json:"embedding"`
	Index     int       `json:"index"`
}

type voyageResponse struct {
	Data []voyageResponseItem `json:"data"`
}

func (v *VoyageEmbedder) embed(ctx context.Context, spanName string, input interface{}, inputType string) ([]voyageResponseItem, error) {
	if !v.IsConfigured() {
		return nil, fmt.Errorf("Voyage embedder is not configured (VOYAGE_API_KEY not set)")
	}

	ctx, span := tracer.Start(ctx, spanName)
	defer span.End()

	reqBody := voyageRequest{
		Input:           input,
		Model:           v.model,
		InputType:       inputType,
		OutputDimension: Dimension,
	}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, telemetry.Fail(span, fmt.Errorf("failed to marshal Voyage request: %w", err), "marshal failed")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, v.apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, telemetry.Fail(span, fmt.Errorf("failed to create Voyage request: %w", err), "request creation failed")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+v.apiKey)

	resp, err := v.client.Do(req)
	if err != nil {
		return nil, telemetry.Fail(span, fmt.Errorf("failed to send Voyage request: %w", err), "request failed")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, telemetry.Fail(span, fmt.Errorf("failed to read Voyage response: %w", err), "read failed")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, telemetry.Fail(span, fmt.Errorf("Voyage API error: status %d, body: %s", resp.StatusCode, string(body)), "non-200 response")
	}

	var voyageResp voyageResponse
	if err := json.Unmarshal(body, &voyageResp); err != nil {
		return nil, telemetry.Fail(span, fmt.Errorf("failed to unmarshal Voyage response: %w", err), "unmarshal failed")
	}

	return voyageResp.Data, nil
}

// firstEmbedding extracts and validates the single embedding from a
// single-item Voyage response, applying the same index-integrity check
// EmbedDocuments applies to its batch responses — a single-item request
// should only ever get back index 0, so anything else means the API
// response doesn't correspond to the request we sent.
func firstEmbedding(items []voyageResponseItem) ([]float32, error) {
	if len(items) == 0 {
		return nil, fmt.Errorf("Voyage API returned no embeddings")
	}
	if items[0].Index != 0 {
		return nil, fmt.Errorf("Voyage API returned unexpected index %d for a single-item request", items[0].Index)
	}
	return items[0].Embedding, nil
}

// EmbedDocument embeds a single piece of indexed text (input_type "document").
func (v *VoyageEmbedder) EmbedDocument(ctx context.Context, text string) ([]float32, error) {
	items, err := v.embed(ctx, "embedding.voyage.embed_document", text, "document")
	if err != nil {
		return nil, err
	}
	return firstEmbedding(items)
}

// EmbedQuery embeds a user's search string (input_type "query") — Voyage's
// retrieval-tuned models perform better when query text is distinguished
// from document text at embed time.
func (v *VoyageEmbedder) EmbedQuery(ctx context.Context, text string) ([]float32, error) {
	items, err := v.embed(ctx, "embedding.voyage.embed_query", text, "query")
	if err != nil {
		return nil, err
	}
	return firstEmbedding(items)
}

// EmbedDocuments batches multiple documents into a single Voyage API call.
func (v *VoyageEmbedder) EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error) {
	inputs := make([]interface{}, len(texts))
	for i, t := range texts {
		inputs[i] = t
	}
	items, err := v.embed(ctx, "embedding.voyage.embed_documents", inputs, "document")
	if err != nil {
		return nil, err
	}
	// Size and validate against the request length (len(texts)), not the
	// response length (len(items)) — a short response (fewer items than
	// requested) must fail loudly here rather than silently returning an
	// undersized slice that a caller indexing by request position could
	// misinterpret as a shorter-but-complete result.
	if len(items) != len(texts) {
		return nil, fmt.Errorf("Voyage API returned %d embeddings for %d inputs", len(items), len(texts))
	}
	out := make([][]float32, len(texts))
	for _, item := range items {
		if item.Index < 0 || item.Index >= len(out) {
			return nil, fmt.Errorf("Voyage API returned out-of-range index %d for %d inputs", item.Index, len(out))
		}
		out[item.Index] = item.Embedding
	}
	// A response with a duplicate index (e.g. two items both claiming index
	// 3) passes the length and range checks above while leaving another
	// index's slot never written — catch that here instead of letting a nil
	// embedding reach PersistEmbedding indistinguishably from a normal
	// transient DB error.
	for i, vec := range out {
		if vec == nil {
			return nil, fmt.Errorf("Voyage API response never populated index %d for %d inputs (duplicate index in response)", i, len(out))
		}
	}
	return out, nil
}
