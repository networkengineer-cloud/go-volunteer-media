package handlers

import (
	"fmt"
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

// updateSearchResult is a group update match with its relevance rank.
type updateSearchResult struct {
	models.Update
	Rank float64 `json:"rank"`
}

// Search performs hybrid keyword + semantic search over a group's animals,
// comments, and updates. Keyword matching uses Postgres full-text search
// (tsvector/websearch_to_tsquery, unchanged from the original keyword-search
// feature); semantic matching uses pgvector cosine similarity against
// embeddings populated by internal/embedding's write-path goroutine and
// reconciliation sweep. When semantic search is available for this request,
// the two ranked candidate lists are merged via Reciprocal Rank Fusion (see
// search_rank.go) before the client's limit/offset is applied; when it
// isn't, each resource is queried and paginated directly by Postgres exactly
// as the original keyword-only feature did, preserving real ts_rank values
// and unlimited-depth pagination. Semantic search is a ranking enhancement,
// never a hard dependency: if SEMANTIC_SEARCH_ENABLED is off, the embedder
// isn't configured, the query-embedding call fails, or a row has no
// embedding yet, results degrade to keyword-only ranking rather than
// failing the request.
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
		validTypes := map[string]bool{"all": true, "animals": true, "comments": true, "updates": true}
		if !validTypes[searchType] {
			c.JSON(http.StatusBadRequest, gin.H{"error": "type must be one of: all, animals, comments, updates"})
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
		if embedding.Usable(embedder) {
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

			keywordAnimalsQuery := applyPageOrPool(db.Model(&models.Animal{}).
				Select("animals.*, ts_rank(search_vector, websearch_to_tsquery('english', ?)) AS rank", query).
				Where("group_id = ? AND search_vector @@ websearch_to_tsquery('english', ?)", groupID, query).
				Order("rank DESC"), semanticAvailable, pool, limit, offset)
			var keywordRows []animalSearchResult
			if err := keywordAnimalsQuery.Find(&keywordRows).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to search animals"})
				return
			}

			if !semanticAvailable {
				response["animals"] = keywordRows
				response["total_animals"] = totalAnimals
			} else {
				semanticQuery := db.Model(&models.Animal{}).
					Select("animals.*, 0::float8 AS rank").
					Where("group_id = ? AND embedding IS NOT NULL", groupID).
					Clauses(clause.OrderBy{Expression: clause.Expr{SQL: "embedding <=> ?", Vars: []interface{}{queryVector}}})

				// See cappedTotal's doc comment: totalAnimals alone (a
				// keyword-only Count()) would undercount once semantic-only
				// matches are in the results, and an uncapped total would let
				// "Load more" promise results pagination can never reach.
				animals, total := finishSemanticSearch("animals", keywordRows, semanticQuery, pool, fuseAnimalResults, totalAnimals, offset, limit)
				response["animals"] = animals
				response["total_animals"] = total
			}
		}

		if searchType == "all" || searchType == "comments" {
			var totalComments int64

			// models.NonDeletedAnimalCommentsQuery joins animals and excludes
			// soft-deleted ones — GORM's soft-delete scope only auto-applies to
			// the query's primary model (animal_comments), and this condition is
			// shared with internal/embedding/sweep.go's sweepComments so the two
			// call sites can't drift out of sync.
			keywordBase := models.NonDeletedAnimalCommentsQuery(db).
				Where("animals.group_id = ? AND animal_comments.search_vector @@ websearch_to_tsquery('english', ?)", groupID, query)

			if err := keywordBase.Session(&gorm.Session{}).Count(&totalComments).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count matching comments"})
				return
			}

			keywordCommentsQuery := applyPageOrPool(keywordBase.Session(&gorm.Session{}).
				Select("animal_comments.*, animals.name AS animal_name, ts_rank(animal_comments.search_vector, websearch_to_tsquery('english', ?)) AS rank", query).
				Order("rank DESC"), semanticAvailable, pool, limit, offset)
			var keywordRows []commentSearchResult
			if err := keywordCommentsQuery.Find(&keywordRows).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to search comments"})
				return
			}

			if !semanticAvailable {
				response["comments"] = keywordRows
				response["total_comments"] = totalComments
			} else {
				semanticQuery := models.NonDeletedAnimalCommentsQuery(db).
					Select("animal_comments.*, animals.name AS animal_name, 0::float8 AS rank").
					Where("animals.group_id = ? AND animal_comments.embedding IS NOT NULL", groupID).
					Clauses(clause.OrderBy{Expression: clause.Expr{SQL: "animal_comments.embedding <=> ?", Vars: []interface{}{queryVector}}})

				comments, total := finishSemanticSearch("comments", keywordRows, semanticQuery, pool, fuseCommentResults, totalComments, offset, limit)
				response["comments"] = comments
				response["total_comments"] = total
			}
		}

		if searchType == "all" || searchType == "updates" {
			var totalUpdates int64
			if err := db.Model(&models.Update{}).
				Where("group_id = ? AND search_vector @@ websearch_to_tsquery('english', ?)", groupID, query).
				Count(&totalUpdates).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count matching updates"})
				return
			}

			keywordUpdatesQuery := applyPageOrPool(db.Model(&models.Update{}).
				Select("updates.*, ts_rank(search_vector, websearch_to_tsquery('english', ?)) AS rank", query).
				Where("group_id = ? AND search_vector @@ websearch_to_tsquery('english', ?)", groupID, query).
				Order("rank DESC"), semanticAvailable, pool, limit, offset)
			var keywordUpdateRows []updateSearchResult
			if err := keywordUpdatesQuery.Find(&keywordUpdateRows).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to search updates"})
				return
			}

			if !semanticAvailable {
				response["updates"] = keywordUpdateRows
				response["total_updates"] = totalUpdates
			} else {
				semanticQuery := db.Model(&models.Update{}).
					Select("updates.*, 0::float8 AS rank").
					Where("group_id = ? AND embedding IS NOT NULL", groupID).
					Clauses(clause.OrderBy{Expression: clause.Expr{SQL: "embedding <=> ?", Vars: []interface{}{queryVector}}})

				updates, total := finishSemanticSearch("updates", keywordUpdateRows, semanticQuery, pool, fuseUpdateResults, totalUpdates, offset, limit)
				response["updates"] = updates
				response["total_updates"] = total
			}
		}

		c.JSON(http.StatusOK, response)
	}
}

// fuseResults is the shared core of fuseAnimalResults/fuseCommentResults/
// fuseUpdateResults: merges keyword and semantic matches of any row type via
// Reciprocal Rank Fusion and returns the requested page, plus the size of
// the full fused candidate set (before pagination) — callers use this to
// correct total_* for semantic-only matches a keyword-only Count() would
// miss. A row present in both lists is deduplicated (its full data is taken
// from whichever list is checked first — identical either way, since it's
// the same database row); only its combined score changes.
func fuseResults[T any](keyword, semantic []T, getID func(T) uint, setRank func(*T, float64), offset, limit int) ([]T, int) {
	rows := make(map[uint]T, len(keyword)+len(semantic))
	keywordIDs := make([]uint, len(keyword))
	for i, r := range keyword {
		id := getID(r)
		keywordIDs[i] = id
		rows[id] = r
	}
	semanticIDs := make([]uint, len(semantic))
	for i, r := range semantic {
		id := getID(r)
		semanticIDs[i] = id
		if _, ok := rows[id]; !ok {
			rows[id] = r
		}
	}

	ordered, scores := fuseRankedIDs(keywordIDs, semanticIDs)
	page := paginateIDs(ordered, offset, limit)

	result := make([]T, 0, len(page))
	for _, id := range page {
		row := rows[id]
		setRank(&row, scores[id])
		result = append(result, row)
	}
	return result, len(ordered)
}

func fuseAnimalResults(keyword, semantic []animalSearchResult, offset, limit int) ([]animalSearchResult, int) {
	return fuseResults(keyword, semantic,
		func(r animalSearchResult) uint { return r.ID },
		func(r *animalSearchResult, rank float64) { r.Rank = rank },
		offset, limit,
	)
}

func fuseCommentResults(keyword, semantic []commentSearchResult, offset, limit int) ([]commentSearchResult, int) {
	return fuseResults(keyword, semantic,
		func(r commentSearchResult) uint { return r.ID },
		func(r *commentSearchResult, rank float64) { r.Rank = rank },
		offset, limit,
	)
}

func fuseUpdateResults(keyword, semantic []updateSearchResult, offset, limit int) ([]updateSearchResult, int) {
	return fuseResults(keyword, semantic,
		func(r updateSearchResult) uint { return r.ID },
		func(r *updateSearchResult, rank float64) { r.Rank = rank },
		offset, limit,
	)
}

// finishSemanticSearch runs the shared tail of every resource type's
// semantic-search branch in Search: query the vector index (bounded by
// pool), degrade to keyword-only on failure, fuse with the already-fetched
// keyword rows via Reciprocal Rank Fusion, and compute the corrected total.
// This is the one piece of animals/comments/updates' otherwise-parallel
// search blocks that was purely mechanical (identical shape, only the
// semantic query and fuse function differ), so it's factored out here
// rather than kept as three copies that could drift out of sync — e.g. a
// fix to the degrade-on-error behavior only needs to be made once.
func finishSemanticSearch[T any](resourceName string, keywordRows []T, semanticQuery *gorm.DB, pool int, fuse func(keyword, semantic []T, offset, limit int) ([]T, int), keywordCount int64, offset, limit int) ([]T, int64) {
	var semanticRows []T
	if err := semanticQuery.Limit(pool).Find(&semanticRows).Error; err != nil {
		// Semantic search is a ranking enhancement, never a hard dependency
		// (see Search's doc comment): degrade to keyword-only ranking
		// instead of discarding the keyword results already in hand. fuse
		// with a nil semantic list preserves keywordRows' order, and
		// cappedTotal still applies its usual pool-aware capping.
		logging.WithField("error", err.Error()).Warn(fmt.Sprintf("Failed to semantically search %s; degrading to keyword-only results", resourceName))
		semanticRows = nil
	}

	rows, fusedTotal := fuse(keywordRows, semanticRows, offset, limit)
	return rows, cappedTotal(keywordCount, fusedTotal)
}

// applyPageOrPool applies either the client's exact requested page
// (Limit(limit).Offset(offset)) when semantic search isn't available for
// this request, or a wider candidate pool (Limit(pool)) when it is — RRF
// fusion needs a real keyword-ranked candidate list to merge with the
// semantic one before the actual page is sliced out afterward.
func applyPageOrPool(q *gorm.DB, semanticAvailable bool, pool, limit, offset int) *gorm.DB {
	if semanticAvailable {
		return q.Limit(pool)
	}
	return q.Limit(limit).Offset(offset)
}

// cappedTotal folds a keyword-only Count() together with the fused
// candidate set's size, so total_* reflects semantic-only matches a
// keyword-only Count() would miss.
//
// fusedTotal is the size of the actual fused candidate set (fuseResults'
// `ordered`) — exactly what paginateIDs can serve for this request, so it's
// reported as-is even when it exceeds maxCandidatePool (each of the keyword
// and semantic queries can independently return up to `pool` rows, so a
// largely non-overlapping pair of top-`pool` lists can fuse into a set
// bigger than pool itself; capping it here would hide real, reachable
// results behind "Load more" for no reason). This holds even when fusedTotal
// is smaller than keywordCount but still exceeds maxCandidatePool: e.g.
// keywordCount=1000 (many keyword matches overall) and fusedTotal=550 (the
// actual fetchable fused set once a common query saturates both pools) must
// report 550, not the 500 keywordCount alone would clamp to — the fused set
// is exactly what's reachable.
//
// keywordCount, on the other hand, is an uncapped full-table Count() that
// can vastly exceed what a bounded candidate pool will ever fetch — only
// that side of the max() gets clamped to maxCandidatePool, since promising
// a total driven by keywordCount alone would let "Load more" promise depth
// pagination can never actually reach.
func cappedTotal(keywordCount int64, fusedTotal int) int64 {
	return max(int64(fusedTotal), min(keywordCount, int64(maxCandidatePool)))
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
