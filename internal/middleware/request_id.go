package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const RequestIDKey = "X-Request-ID"

// RequestID adds a unique request ID to each request for tracing
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if request ID already exists in headers
		requestID := c.GetHeader(RequestIDKey)
		
		// Generate new UUID if not provided
		if requestID == "" {
			requestID = uuid.New().String()
		}
		
		// Set request ID in context for use in handlers
		c.Set("request_id", requestID)
		
		// Add request ID to response headers
		c.Header(RequestIDKey, requestID)
		
		c.Next()
	}
}
