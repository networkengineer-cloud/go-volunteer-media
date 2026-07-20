package embedding

import (
	"testing"
)

func TestSemanticSearchEnabled_DefaultsTrue(t *testing.T) {
	t.Setenv("SEMANTIC_SEARCH_ENABLED", "")
	if !SemanticSearchEnabled() {
		t.Fatal("expected SemanticSearchEnabled to default to true when unset")
	}
}

func TestSemanticSearchEnabled_FalseDisables(t *testing.T) {
	t.Setenv("SEMANTIC_SEARCH_ENABLED", "false")
	if SemanticSearchEnabled() {
		t.Fatal("expected SemanticSearchEnabled to be false when set to \"false\"")
	}
}

func TestSemanticSearchEnabled_ZeroDisables(t *testing.T) {
	t.Setenv("SEMANTIC_SEARCH_ENABLED", "0")
	if SemanticSearchEnabled() {
		t.Fatal("expected SemanticSearchEnabled to be false when set to \"0\"")
	}
}

func TestSemanticSearchEnabled_OtherValuesEnable(t *testing.T) {
	t.Setenv("SEMANTIC_SEARCH_ENABLED", "true")
	if !SemanticSearchEnabled() {
		t.Fatal("expected SemanticSearchEnabled to be true when set to \"true\"")
	}
}

func TestUsable_RequiresBothFlagAndConfigured(t *testing.T) {
	t.Setenv("SEMANTIC_SEARCH_ENABLED", "")

	if !Usable(&StubEmbedder{}) {
		t.Fatal("expected Usable to be true when the flag is enabled and the embedder is configured")
	}
	if Usable(&StubEmbedder{Unconfigured: true}) {
		t.Fatal("expected Usable to be false when the embedder is not configured, even though the flag defaults to enabled — this is the exact gap that let an unconfigured-but-enabled Voyage embedder retry and fail indefinitely on every write and every sweep tick")
	}

	t.Setenv("SEMANTIC_SEARCH_ENABLED", "false")
	if Usable(&StubEmbedder{}) {
		t.Fatal("expected Usable to be false when the flag is disabled, even though the embedder is configured")
	}

	if Usable(nil) {
		t.Fatal("expected Usable to be false for a nil embedder")
	}
}
