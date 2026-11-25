package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/networkengineer-cloud/go-volunteer-media/internal/middleware"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/storage"
)

// isClientCancel reports whether a failed operation should be attributed to
// the client having gone away rather than to the server. It checks err and,
// as a fallback, the request context itself: a dependency may return its own
// error (one that does not wrap ctx.Err()) once the client has disconnected.
//
// Only context.Canceled counts. context.DeadlineExceeded means something
// took too long, which is a real failure and must still surface as a 5xx.
func isClientCancel(c *gin.Context, err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(c.Request.Context().Err(), context.Canceled)
}

// respondClientClosed ends the request with 499 and no body. The client is
// gone, so nothing will read the response; the status exists for logs and
// traces, where LoggingMiddleware records it at info level.
func respondClientClosed(c *gin.Context) {
	c.AbortWithStatus(middleware.StatusClientClosedRequest)
}

// respondInternalErrorUnlessCanceled writes a 500 with msg, unless the
// failure was caused by the client disconnecting, in which case it writes
// 499 instead. Use it on the "unexpected error" branch of a handler that
// queries the request-scoped DB, so a client hanging up mid-query does not
// page anyone as a server error.
func respondInternalErrorUnlessCanceled(c *gin.Context, err error, msg string) {
	if isClientCancel(c, err) {
		respondClientClosed(c)
		return
	}
	respondInternalError(c, msg)
}

// respondBlobFetchError writes the response for a failed storage-provider
// read (GetImage / GetDocument):
//
//   - storage.ErrNotFound      → 404 notFoundMsg
//   - client cancellation      → 499, no body (see isClientCancel). Not a
//     server fault, so no 5xx and no error-level log; this was the source
//     of a prod 5xx alert.
//   - context.DeadlineExceeded → 504 failMsg: a real timeout talking to
//     storage.
//   - anything else            → 500 failMsg
func respondBlobFetchError(c *gin.Context, err error, notFoundMsg, failMsg string) {
	switch {
	case errors.Is(err, storage.ErrNotFound):
		respondNotFound(c, notFoundMsg)
	case isClientCancel(c, err):
		respondClientClosed(c)
	case errors.Is(err, context.DeadlineExceeded):
		c.JSON(http.StatusGatewayTimeout, gin.H{"error": failMsg})
	default:
		respondInternalError(c, failMsg)
	}
}
