package middleware

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// DBMiddleware makes the request's *gorm.DB available via GetDB, which binds
// it to the request's context (so the OTel GORM plugin nests per-query spans
// under the request's trace) the first time a handler actually asks for it.
// The middleware itself just stores the unscoped db — it runs on every
// request, including routes that never touch the database (health checks,
// static assets), so it must not pay for a context-scoped clone up front.
//
// This replaces the earlier per-handler pattern of reassigning the `db`
// closure parameter (`db = db.WithContext(ctx)`) inside the returned
// gin.HandlerFunc: that parameter is shared by every concurrent request to
// the route (the outer handler function runs once, at route registration),
// so reassigning it from multiple request goroutines was a data race —
// confirmed by `go test -race`, which showed one request's query
// observing a different request's context. Storing the (unscoped) *gorm.DB
// on the per-request gin.Context instead of a shared closure variable
// removes the race entirely; GetDB does the actual context-scoping.
func DBMiddleware(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("db", db)
		c.Next()
	}
}

// GetDB returns a *gorm.DB scoped to the request's context. Callers must
// bind the result with `:=`, e.g. `db := middleware.GetDB(c, db)` — never
// reassign the handler's `db` parameter with `=`. That parameter is shared
// by every concurrent request to the route (the closure is created once, at
// route registration), so writing to it from request goroutines is the
// exact data race this middleware exists to avoid; `:=` shadows it with a
// fresh local scoped to this one request.
//
// Reads the unscoped *gorm.DB DBMiddleware stored on the request context and
// scopes it here, on demand, so requests that never call GetDB (health
// checks, static assets) never pay for the clone. If DBMiddleware didn't run
// (e.g. a test that calls the handler directly against a bare gin.Context),
// falls back to scoping the given db instead, mirroring GetLogger's fallback
// for the same missing-from-context case.
func GetDB(c *gin.Context, db *gorm.DB) *gorm.DB {
	if v, exists := c.Get("db"); exists {
		if raw, ok := v.(*gorm.DB); ok {
			db = raw
		}
	}
	return db.WithContext(c.Request.Context())
}
