package middleware

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// DBMiddleware makes a request-scoped *gorm.DB available via GetDB, binding
// GORM to the request's context so the OTel GORM plugin nests per-query
// spans under the request's trace. Scoping happens once per request here,
// stored on that request's own gin.Context, mirroring how LoggingMiddleware
// exposes the per-request logger via GetLogger.
//
// This replaces the earlier per-handler pattern of reassigning the `db`
// closure parameter (`db = db.WithContext(ctx)`) inside the returned
// gin.HandlerFunc: that parameter is shared by every concurrent request to
// the route (the outer handler function runs once, at route registration),
// so reassigning it from multiple request goroutines was a data race —
// confirmed by `go test -race`, which showed one request's query
// observing a different request's context. Storing the scoped *gorm.DB on
// the per-request gin.Context instead of a shared closure variable removes
// the race entirely.
func DBMiddleware(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("db", db.WithContext(c.Request.Context()))
		c.Next()
	}
}

// GetDB retrieves the request-scoped *gorm.DB set by DBMiddleware. Callers
// must bind the result with `:=`, e.g. `db := middleware.GetDB(c, db)` —
// never reassign the handler's `db` parameter with `=`. That parameter is
// shared by every concurrent request to the route (the closure is created
// once, at route registration), so writing to it from request goroutines is
// the exact data race this middleware exists to avoid; `:=` shadows it with
// a fresh local scoped to this one request.
//
// If DBMiddleware didn't run (e.g. a test that calls the handler directly
// against a bare gin.Context), falls back to scoping the given db itself,
// mirroring GetLogger's fallback for the same missing-from-context case.
func GetDB(c *gin.Context, db *gorm.DB) *gorm.DB {
	if v, exists := c.Get("db"); exists {
		if scoped, ok := v.(*gorm.DB); ok {
			return scoped
		}
	}
	return db.WithContext(c.Request.Context())
}
