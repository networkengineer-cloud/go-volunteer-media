package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/networkengineer-cloud/go-volunteer-media/internal/middleware"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/storage"
)

// statusClientClosedRequest is nginx's non-standard 499: the client hung up
// before the response was written. It is a 4xx, so LoggingMiddleware logs
// it at warn level rather than error.
const statusClientClosedRequest = 499

// respondBlobFetchError writes the response for a failed storage-provider
// read (GetImage / GetDocument):
//
//   - storage.ErrNotFound       → 404 notFoundMsg
//   - context.Canceled          → 499, no body: the client cancelled (closed
//     the tab, scrolled an <img> away). Not a server fault, so no 5xx and no
//     error-level log — this was the source of a prod 5xx alert.
//   - context.DeadlineExceeded  → 504: a real timeout talking to storage.
//   - anything else             → 500 failMsg
//
// The request context is also checked so a provider that returns its own
// error after the client has gone (rather than wrapping ctx.Err()) is still
// treated as a cancellation.
func respondBlobFetchError(c *gin.Context, err error, notFoundMsg, failMsg string) {
	switch {
	case errors.Is(err, storage.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": notFoundMsg})
	case errors.Is(err, context.Canceled) || errors.Is(c.Request.Context().Err(), context.Canceled):
		middleware.GetLogger(c).WithField("error", err.Error()).Info("Client closed request during blob download")
		c.AbortWithStatus(statusClientClosedRequest)
	case errors.Is(err, context.DeadlineExceeded):
		c.JSON(http.StatusGatewayTimeout, gin.H{"error": failMsg})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": failMsg})
	}
}
