package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/logging"
)

// LoggingMiddleware logs HTTP requests with structured logging. Per-route
// request count/duration metrics are NOT recorded here — otelgin.Middleware
// (registered separately in cmd/api/main.go) already emits the standard
// http.server.request.duration histogram with method/route/status
// attributes via the same MeterProvider; adding a second, differently-named
// metric here would just double-count the same signal in Axiom.
func LoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Start timer
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		// Get request ID from context
		requestID, _ := c.Get("request_id")

		// Create logger with request context. WithContext first so trace_id/
		// span_id from the otelgin span (already on c.Request.Context() by
		// the time this middleware runs) get attached, then layer the
		// request-specific fields on top. Reused below for the post-request
		// log so trace_id/span_id are only derived once per request.
		ctxLogger := logging.WithContext(c.Request.Context())
		logger := ctxLogger.WithFields(map[string]interface{}{
			"request_id": requestID,
			"method":     c.Request.Method,
			"path":       path,
			"query":      query,
			"ip":         c.ClientIP(),
			"user_agent": c.Request.UserAgent(),
		})

		// Add logger to context for use in handlers
		c.Set("logger", logger)

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(start)

		// Get status code and error if any
		status := c.Writer.Status()

		// Log the request with appropriate level
		logFields := map[string]interface{}{
			"request_id": requestID,
			"method":     c.Request.Method,
			"path":       path,
			"query":      query,
			"status":     status,
			"latency_ms": latency.Milliseconds(),
			"ip":         c.ClientIP(),
			"user_agent": c.Request.UserAgent(),
			"bytes_in":   c.Request.ContentLength,
			"bytes_out":  c.Writer.Size(),
		}

		// Add user ID if authenticated
		if userID, exists := c.Get("user_id"); exists {
			logFields["user_id"] = userID
		}

		requestLogger := ctxLogger.WithFields(logFields)

		// Log with appropriate level based on status code
		if status >= 500 {
			requestLogger.Error("Request failed with server error", nil)
		} else if status >= 400 {
			requestLogger.Warn("Request failed with client error")
		} else {
			requestLogger.Info("Request completed successfully")
		}
	}
}

// GetLogger retrieves the logger from gin context
func GetLogger(c *gin.Context) *logging.Logger {
	if logger, exists := c.Get("logger"); exists {
		if l, ok := logger.(*logging.Logger); ok {
			return l
		}
	}
	// Return default logger with request context if not found
	return logging.WithContext(c.Request.Context())
}
