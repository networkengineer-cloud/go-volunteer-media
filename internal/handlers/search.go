package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/middleware"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
	"gorm.io/gorm"
)

// animalSearchResult is an animal match with its full-text relevance rank.
type animalSearchResult struct {
	models.Animal
	Rank float64 `json:"rank"`
}

// commentSearchResult is a comment match with its parent animal's name/id
// (comments are meaningless out of the context of which animal they're on)
// and its full-text relevance rank.
type commentSearchResult struct {
	models.AnimalComment
	AnimalName string  `json:"animal_name"`
	Rank       float64 `json:"rank"`
}

// Search performs keyword/phrase search over a group's animals and comments
// using Postgres full-text search (tsvector columns populated by generated
// columns — see createSearchIndexes). Scoped to the requesting user's group
// access, matching every other /groups/:id route in this file set.
func Search(db *gorm.DB) gin.HandlerFunc {
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

		response := gin.H{}

		if searchType == "all" || searchType == "animals" {
			var animals []animalSearchResult
			var totalAnimals int64

			db.Model(&models.Animal{}).
				Select("animals.*, ts_rank(search_vector, websearch_to_tsquery('english', ?)) AS rank", query).
				Where("group_id = ? AND search_vector @@ websearch_to_tsquery('english', ?)", groupID, query).
				Count(&totalAnimals)

			if err := db.Model(&models.Animal{}).
				Select("animals.*, ts_rank(search_vector, websearch_to_tsquery('english', ?)) AS rank", query).
				Where("group_id = ? AND search_vector @@ websearch_to_tsquery('english', ?)", groupID, query).
				Order("rank DESC").
				Limit(limit).
				Offset(offset).
				Find(&animals).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to search animals"})
				return
			}

			response["animals"] = animals
			response["total_animals"] = totalAnimals
		}

		if searchType == "all" || searchType == "comments" {
			var comments []commentSearchResult
			var totalComments int64

			base := db.Model(&models.AnimalComment{}).
				Joins("JOIN animals ON animals.id = animal_comments.animal_id").
				Where("animals.group_id = ? AND animal_comments.search_vector @@ websearch_to_tsquery('english', ?)", groupID, query)

			base.Session(&gorm.Session{}).Count(&totalComments)

			if err := base.Session(&gorm.Session{}).
				Select("animal_comments.*, animals.name AS animal_name, ts_rank(animal_comments.search_vector, websearch_to_tsquery('english', ?)) AS rank", query).
				Order("rank DESC").
				Limit(limit).
				Offset(offset).
				Find(&comments).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to search comments"})
				return
			}

			response["comments"] = comments
			response["total_comments"] = totalComments
		}

		c.JSON(http.StatusOK, response)
	}
}
