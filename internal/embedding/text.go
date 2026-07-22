package embedding

// AnimalEmbeddingText builds the canonical searchable text for an animal
// from its name/species/breed/description/trainer_notes fields — matching
// the same field set the search_vector generated column indexes. This is
// the single source of truth for the formula, used by both the write-path
// embed helpers (internal/handlers/search_embed.go) and the reconciliation
// sweep, so keyword search and semantic search always index the same
// content for the same row.
func AnimalEmbeddingText(name, species, breed, description, trainerNotes string) string {
	return name + " " + species + " " + breed + " " + description + " " + trainerNotes
}

// UpdateEmbeddingText builds the canonical searchable text for a group
// update from its title/content fields, matching the search_vector
// generated column. See AnimalEmbeddingText's comment for why this lives
// here rather than being duplicated per call site.
func UpdateEmbeddingText(title, content string) string {
	return title + " " + content
}
