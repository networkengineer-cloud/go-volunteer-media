package handlers

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

// GetEnvironment returns the current environment configuration
func GetEnvironment() gin.HandlerFunc {
	return func(c *gin.Context) {
		env := os.Getenv("ENV")
		c.JSON(http.StatusOK, gin.H{
			"environment":     env,
			"is_development":  env == "development",
			"developer_tools": env == "development",
		})
	}
}
