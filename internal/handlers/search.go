package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/embedding"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/logging"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/middleware"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
	"github.com/pgvector/pgvector-go"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// animalSearchResult is an animal match with its full-text/semantic
// relevance rank (the Reciprocal Rank Fusion score once fused; see
// fuseAnimalResults).
type animalSearchResult struct {
	models.Animal
	Rank float64 `json:"rank"`
}

// commentSearchResult is a comment match with its parent animal's name/id
// (comments are meaningless out of the context of which animal they're on)
// and its relevance rank.
type commentSearchResult struct {
	models.AnimalComment
	AnimalName string  `json:"animal_name"`
	Rank       float64 `json:"rank"`
}

// Search performs hybrid keyword + semantic search over a group's animals
// and comments. Keyword matching uses Postgres full-text search
// (tsvector/websearch_to_tsquery, unchanged from the original keyword-search
// feature); semantic matching uses pgvector cosine similarity against
// embeddings populated by internal/embedding's write-path goroutine and
// reconciliation sweep. The two ranked candidate lists are merged via
// Reciprocal Rank Fusion (see search_rank.go) before the client's
// limit/offset is applied. Semantic search is a ranking enhancement, never a
// hard dependency: if SEMANTIC_SEARCH_ENABLED is off, the query-embedding
// call fails, or a row has no embedding yet, results degrade to keyword-only
// ranking rather than failing the request.
func Search(db *gorm.DB, embedder embedding.Embedder) gin.HandlerFunc {
	return func(c *gin.Context) {
		db := middleware.GetDB(c, db)
		groupID := c.Param("id")
		userID, _ := c.Get("user_id")
		isAdmin, _ := c.Get("is_admin")

		if !checkGroupAccess(db, userID, isAdmin, groupID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			return
		}

		query := c.Query("q")
		if query == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "q parameter is required"})
			return
		}

		searchType := c.DefaultQuery("type", "all")
		if searchType != "all" && searchType != "animals" && searchType != "comments" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "type must be one of: all, animals, comments"})
			return
		}

		limit := 20
		if limitParam := c.Query("limit"); limitParam != "" {
			if parsed, err := strconv.Atoi(limitParam); err == nil && parsed > 0 {
				limit = parsed
				if limit > 100 {
					limit = 100
				}
			}
		}

		offset := 0
		if offsetParam := c.Query("offset"); offsetParam != "" {
			if parsed, err := strconv.Atoi(offsetParam); err == nil && parsed >= 0 {
				offset = parsed
			}
		}

		pool := candidatePoolSize(offset, limit)

		var queryVector pgvector.Vector
		semanticAvailable := false
		if embedding.SemanticSearchEnabled() {
			if vec, err := embedder.EmbedQuery(c.Request.Context(), query); err == nil {
				queryVector = pgvector.NewVector(vec)
				semanticAvailable = true
			} else {
				logging.WithField("error", err.Error()).Warn("Failed to embed search query; falling back to keyword-only results")
			}
		}

		response := gin.H{}

		if searchType == "all" || searchType == "animals" {
			var totalAnimals int64
			if err := db.Model(&models.Animal{}).
				Where("group_id = ? AND search_vector @@ websearch_to_tsquery('english', ?)", groupID, query).
				Count(&totalAnimals).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count matching animals"})
				return
			}

			var keywordRows []animalSearchResult
			if err := db.Model(&models.Animal{}).
				Select("animals.*, ts_rank(search_vector, websearch_to_tsquery('english', ?)) AS rank", query).
				Where("group_id = ? AND search_vector @@ websearch_to_tsquery('english', ?)", groupID, query).
				Order("rank DESC").
				Limit(pool).
				Find(&keywordRows).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to search animals"})
				return
			}

			var semanticRows []animalSearchResult
			if semanticAvailable {
				if err := db.Model(&models.Animal{}).
					Select("animals.*, 0::float8 AS rank").
					Where("group_id = ? AND embedding IS NOT NULL", groupID).
					Clauses(clause.OrderBy{Expression: clause.Expr{SQL: "embedding <=> ?", Vars: []interface{}{queryVector}}}).
					Limit(pool).
					Find(&semanticRows).Error; err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to semantically search animals"})
					return
				}
			}

			response["animals"] = fuseAnimalResults(keywordRows, semanticRows, offset, limit)
			response["total_animals"] = totalAnimals
		}

		if searchType == "all" || searchType == "comments" {
			var totalComments int64

			// GORM's soft-delete scope only auto-applies to the query's primary
			// model (animal_comments) — the joined animals table needs an
			// explicit deleted_at check, or comments on a deleted animal leak
			// through search.
			keywordBase := db.Model(&models.AnimalComment{}).
				Joins("JOIN animals ON animals.id = animal_comments.animal_id").
				Where("animals.group_id = ? AND animals.deleted_at IS NULL AND animal_comments.search_vector @@ websearch_to_tsquery('english', ?)", groupID, query)

			if err := keywordBase.Session(&gorm.Session{}).Count(&totalComments).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count matching comments"})
				return
			}

			var keywordRows []commentSearchResult
			if err := keywordBase.Session(&gorm.Session{}).
				Select("animal_comments.*, animals.name AS animal_name, ts_rank(animal_comments.search_vector, websearch_to_tsquery('english', ?)) AS rank", query).
				Order("rank DESC").
				Limit(pool).
				Find(&keywordRows).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to search comments"})
				return
			}

			var semanticRows []commentSearchResult
			if semanticAvailable {
				semanticBase := db.Model(&models.AnimalComment{}).
					Joins("JOIN animals ON animals.id = animal_comments.animal_id").
					Where("animals.group_id = ? AND animals.deleted_at IS NULL AND animal_comments.embedding IS NOT NULL", groupID)

				if err := semanticBase.
					Select("animal_comments.*, animals.name AS animal_name, 0::float8 AS rank").
					Clauses(clause.OrderBy{Expression: clause.Expr{SQL: "animal_comments.embedding <=> ?", Vars: []interface{}{queryVector}}}).
					Limit(pool).
					Find(&semanticRows).Error; err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to semantically search comments"})
					return
				}
			}

			response["comments"] = fuseCommentResults(keywordRows, semanticRows, offset, limit)
			response["total_comments"] = totalComments
		}

		c.JSON(http.StatusOK, response)
	}
}

// fuseAnimalResults merges keyword and semantic animal matches via
// Reciprocal Rank Fusion and returns the requested page. A row present in
// both lists is deduplicated (its full data is taken from whichever list is
// checked first — identical either way, since it's the same database row);
// only its combined score changes.
func fuseAnimalResults(keyword, semantic []animalSearchResult, offset, limit int) []animalSearchResult {
	rows := make(map[uint]animalSearchResult, len(keyword)+len(semantic))
	keywordIDs := make([]uint, len(keyword))
	for i, r := range keyword {
		keywordIDs[i] = r.ID
		rows[r.ID] = r
	}
	semanticIDs := make([]uint, len(semantic))
	for i, r := range semantic {
		semanticIDs[i] = r.ID
		if _, ok := rows[r.ID]; !ok {
			rows[r.ID] = r
		}
	}

	ordered, scores := fuseRankedIDs(keywordIDs, semanticIDs)
	page := paginateIDs(ordered, offset, limit)

	result := make([]animalSearchResult, 0, len(page))
	for _, id := range page {
		row := rows[id]
		row.Rank = scores[id]
		result = append(result, row)
	}
	return result
}

// fuseCommentResults mirrors fuseAnimalResults for commentSearchResult.
func fuseCommentResults(keyword, semantic []commentSearchResult, offset, limit int) []commentSearchResult {
	rows := make(map[uint]commentSearchResult, len(keyword)+len(semantic))
	keywordIDs := make([]uint, len(keyword))
	for i, r := range keyword {
		keywordIDs[i] = r.ID
		rows[r.ID] = r
	}
	semanticIDs := make([]uint, len(semantic))
	for i, r := range semantic {
		semanticIDs[i] = r.ID
		if _, ok := rows[r.ID]; !ok {
			rows[r.ID] = r
		}
	}

	ordered, scores := fuseRankedIDs(keywordIDs, semanticIDs)
	page := paginateIDs(ordered, offset, limit)

	result := make([]commentSearchResult, 0, len(page))
	for _, id := range page {
		row := rows[id]
		row.Rank = scores[id]
		result = append(result, row)
	}
	return result
}

// paginateIDs slices a fused ID order to the client's requested page,
// clamping so an offset beyond the end of the fused candidates returns an
// empty page instead of panicking.
func paginateIDs(ordered []uint, offset, limit int) []uint {
	start := offset
	if start > len(ordered) {
		start = len(ordered)
	}
	end := start + limit
	if end > len(ordered) {
		end = len(ordered)
	}
	return ordered[start:end]
}
