package middleware

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

// MaxRequestBodySize limits the size of request bodies to prevent DOS attacks
// Default limit is 10MB, but can be overridden per route
func MaxRequestBodySize(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Set max bytes for request body
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)

		c.Next()

		// Check if body size limit was exceeded
		if c.Errors.Last() != nil {
			// Check for common body too large errors
			errMsg := c.Errors.Last().Err.Error()
			if errors.Is(c.Errors.Last().Err, http.ErrHandlerTimeout) || 
			   errMsg == "http: request body too large" {
				c.JSON(http.StatusRequestEntityTooLarge, gin.H{
					"error": "Request body too large",
				})
				c.Abort()
				return
			}
		}
	}
}
