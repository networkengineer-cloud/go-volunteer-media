package middleware

import (
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

// originalBodyCtxKey is the context key used to store the raw request body before
// any MaxRequestBodySize wrapper has been applied.
const originalBodyCtxKey = "_mw_original_body"

// MaxRequestBodySize limits the size of request bodies to prevent DOS attacks.
// When applied multiple times (e.g. a global default and a per-route override),
// the per-route call replaces the global limit rather than nesting inside it.
// The original unwrapped body is saved in the Gin context on the first call so
// that subsequent calls can wrap it directly at the requested size.
func MaxRequestBodySize(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Use the original, unwrapped body so that nested calls (e.g. a per-route
		// override applied after the global middleware) replace the limit instead
		// of layering a larger limit inside a smaller one.
		body := c.Request.Body
		if orig, exists := c.Get(originalBodyCtxKey); exists {
			rc, ok := orig.(io.ReadCloser)
			if ok {
				body = rc
			}
		} else {
			c.Set(originalBodyCtxKey, body)
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, body, maxBytes)

		c.Next()

		// Check if body size limit was exceeded
		if c.Errors.Last() != nil {
			var mbe *http.MaxBytesError
			if errors.Is(c.Errors.Last().Err, http.ErrHandlerTimeout) ||
				errors.As(c.Errors.Last().Err, &mbe) {
				c.JSON(http.StatusRequestEntityTooLarge, gin.H{
					"error": "Request body too large",
				})
				c.Abort()
				return
			}
		}
	}
}
