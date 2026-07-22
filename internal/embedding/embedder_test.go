package embedding

import (
	"testing"
)

func TestSemanticSearchEnabled_DefaultsFalse(t *testing.T) {
	t.Setenv("SEMANTIC_SEARCH_ENABLED", "")
	if SemanticSearchEnabled() {
		t.Fatal("expected SemanticSearchEnabled to default to false when unset (opt-in, unlike EMAIL_ENABLED)")
	}
}

func TestSemanticSearchEnabled_TrueEnables(t *testing.T) {
	t.Setenv("SEMANTIC_SEARCH_ENABLED", "true")
	if !SemanticSearchEnabled() {
		t.Fatal("expected SemanticSearchEnabled to be true when set to \"true\"")
	}
}

func TestSemanticSearchEnabled_OneEnables(t *testing.T) {
	t.Setenv("SEMANTIC_SEARCH_ENABLED", "1")
	if !SemanticSearchEnabled() {
		t.Fatal("expected SemanticSearchEnabled to be true when set to \"1\"")
	}
}

func TestSemanticSearchEnabled_OtherValuesDisable(t *testing.T) {
	for _, v := range []string{"false", "0", "yes", "TRUE", "enabled"} {
		t.Setenv("SEMANTIC_SEARCH_ENABLED", v)
		if SemanticSearchEnabled() {
			t.Fatalf("expected SemanticSearchEnabled to be false when set to %q — only exactly \"true\" or \"1\" enable", v)
		}
	}
}

func TestUsable_RequiresBothFlagAndConfigured(t *testing.T) {
	t.Setenv("SEMANTIC_SEARCH_ENABLED", "true")

	if !Usable(&StubEmbedder{}) {
		t.Fatal("expected Usable to be true when the flag is enabled and the embedder is configured")
	}
	if Usable(&StubEmbedder{Unconfigured: true}) {
		t.Fatal("expected Usable to be false when the embedder is not configured, even though the flag is enabled — this is the exact gap that let an unconfigured-but-enabled Voyage embedder retry and fail indefinitely on every write and every sweep tick")
	}

	t.Setenv("SEMANTIC_SEARCH_ENABLED", "false")
	if Usable(&StubEmbedder{}) {
		t.Fatal("expected Usable to be false when the flag is disabled, even though the embedder is configured")
	}

	if Usable(nil) {
		t.Fatal("expected Usable to be false for a nil embedder")
	}
}
