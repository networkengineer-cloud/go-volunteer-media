// Package ratelimit is a fixed-window rate limiter whose counters live in
// the database (models.RateLimitCounter), so the limit holds across every
// replica instead of per process (roadmap item AR-4).
//
// Limits are keyed on a subject chosen per rule - for login, the username
// together with the client IP - so a whole shelter behind one public IP is
// not throttled as if it were a single client. A looser per-IP rule still
// caps password spraying across many usernames from one address, and the
// per-account lockout in the login handler covers distributed guessing
// against one account.
//
// Each rule has a name that scopes its counters ("login:user_ip",
// "login:ip", ...). A new client type - the check-in kiosk, for example -
// gets its own rules rather than sharing the volunteer login budget.
package ratelimit

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/logging"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/middleware"
	"gorm.io/gorm"
)

// Rule is one named limit: at most Limit hits per Window for each subject.
type Rule struct {
	Name   string
	Limit  int
	Window time.Duration
}

// Limiter records hits in the database.
type Limiter struct {
	db  *gorm.DB
	now func() time.Time
}

// New returns a Limiter storing its counters through db.
func New(db *gorm.DB) *Limiter {
	return &Limiter{db: db, now: time.Now}
}

// upsertSQL records one hit and returns the window's count. The row lock
// taken by ON CONFLICT DO UPDATE makes concurrent hits from different
// replicas serialize on the key. A hit stamped with an older window than
// the row's (clock skew between replicas) counts toward the newer window
// rather than resetting it.
const upsertSQL = `INSERT INTO rate_limit_counters (key, window_start, count) VALUES (?, ?, 1)
ON CONFLICT (key) DO UPDATE SET
	count = CASE WHEN excluded.window_start > rate_limit_counters.window_start
		THEN 1 ELSE rate_limit_counters.count + 1 END,
	window_start = CASE WHEN excluded.window_start > rate_limit_counters.window_start
		THEN excluded.window_start ELSE rate_limit_counters.window_start END
RETURNING count, window_start`

// Hit records one hit for subject under rule. It reports whether the hit is
// within the limit and, when it is not, how long until the window resets.
func (l *Limiter) Hit(ctx context.Context, rule Rule, subject string) (allowed bool, retryAfter time.Duration, err error) {
	now := l.now()
	windowSecs := int64(rule.Window / time.Second)
	if windowSecs < 1 {
		windowSecs = 1
	}
	windowStart := now.Unix() - now.Unix()%windowSecs

	var row struct {
		Count       int
		WindowStart int64
	}
	if err := l.db.WithContext(ctx).Raw(upsertSQL, counterKey(rule.Name, subject), windowStart).Scan(&row).Error; err != nil {
		return false, 0, fmt.Errorf("ratelimit: record hit: %w", err)
	}
	if row.Count <= rule.Limit {
		return true, 0, nil
	}
	reset := time.Unix(row.WindowStart+windowSecs, 0)
	retryAfter = reset.Sub(now)
	if retryAfter < time.Second {
		retryAfter = time.Second
	}
	// A replica with a skewed clock can have written a window_start in the
	// future (see the upsertSQL comment above); without this cap that skew
	// leaks straight into the client-facing Retry-After header as a wait far
	// longer than the rule's own window.
	if retryAfter > rule.Window {
		retryAfter = rule.Window
	}
	return false, retryAfter, nil
}

// counterKey hashes the rule name and subject so usernames and IPs are not
// stored in the clear and every key has a fixed length.
func counterKey(rule, subject string) string {
	sum := sha256.Sum256([]byte(rule + "\x00" + subject))
	return hex.EncodeToString(sum[:])
}

// Sweep deletes counters whose window started more than maxWindow ago; no
// rule can still be counting them. It is idempotent, so every replica may
// run it.
func (l *Limiter) Sweep(ctx context.Context, maxWindow time.Duration) (int64, error) {
	cutoff := l.now().Add(-maxWindow).Unix()
	res := l.db.WithContext(ctx).Exec("DELETE FROM rate_limit_counters WHERE window_start < ?", cutoff)
	return res.RowsAffected, res.Error
}

// StartSweep runs Sweep every interval until stop is called. maxWindow must
// be at least the longest Rule.Window in use.
func (l *Limiter) StartSweep(interval, maxWindow time.Duration) (stop func()) {
	ticker := time.NewTicker(interval)
	done := make(chan struct{})
	finished := make(chan struct{})
	go func() {
		defer close(finished)
		for {
			select {
			case <-ticker.C:
				if _, err := l.Sweep(context.Background(), maxWindow); err != nil {
					logging.WithField("error", err.Error()).Warn("Rate limit counter sweep failed")
				}
			case <-done:
				ticker.Stop()
				return
			}
		}
	}()
	var once sync.Once
	return func() {
		once.Do(func() { close(done) })
		<-finished
	}
}

// Check pairs a Rule with the subject it limits. With Field empty the
// subject is the client IP. With Field set (e.g. "username") the subject is
// that JSON body field, trimmed and lower-cased, joined with the client IP;
// a request without the field skips this check (pair it with an IP-only
// check so it is still limited).
type Check struct {
	Rule  Rule
	Field string
}

// bodyPeekCap is the largest body Middleware accepts on routes with a
// Check.Field; larger bodies get 413. The whole body must be parsed to find
// the field the handler will see, and reading the global 10MB cap for every
// unauthenticated request would be an amplification path.
const bodyPeekCap = 64 * 1024 // 64 KiB

// Middleware applies every check to the request and rejects it with 429 and
// a Retry-After header when any is over its limit. All checks are counted,
// even after one fails, so a client cannot probe which limit it tripped.
//
// A database error fails open (the request proceeds, the error is logged):
// the endpoints this guards need the database anyway, and the login
// handler's per-account lockout still applies.
func Middleware(l *Limiter, checks ...Check) gin.HandlerFunc {
	needsBody := false
	for _, ch := range checks {
		if ch.Field != "" {
			needsBody = true
		}
	}

	return func(c *gin.Context) {
		ctx := c.Request.Context()
		ip := c.ClientIP()

		var prefix []byte
		if needsBody && c.Request.Body != nil {
			// Read one byte past the cap so an oversized body is detected.
			// A body over the cap is rejected outright: a truncated prefix
			// can't be parsed, so the per-account check would silently
			// drop to IP-only while the handler still read the full body.
			// The auth bodies this guards are a few hundred bytes. Read
			// errors (e.g. an upstream MaxBytesReader) are treated the same
			// way - no check is ever skipped because of the body.
			var err error
			prefix, err = io.ReadAll(io.LimitReader(c.Request.Body, bodyPeekCap+1))
			if err != nil || len(prefix) > bodyPeekCap {
				c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, gin.H{
					"error": "Request body too large",
				})
				return
			}
			c.Request.Body = io.NopCloser(bytes.NewReader(prefix))
		}

		var longestWait time.Duration
		limited := false
		for _, ch := range checks {
			subject := ip
			if ch.Field != "" {
				value := extractField(prefix, ch.Field)
				if value == "" {
					continue
				}
				subject = value + "|" + ip
			}
			allowed, wait, err := l.Hit(ctx, ch.Rule, subject)
			if err != nil {
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					// The client went away or its deadline passed; this is
					// not a limiter or database problem worth paging on.
					middleware.GetLogger(c).Debug("Rate limit check skipped: request context ended")
				} else {
					middleware.GetLogger(c).Error("Rate limit check failed; allowing request", err)
				}
				continue
			}
			if !allowed {
				limited = true
				if wait > longestWait {
					longestWait = wait
				}
			}
		}

		if limited {
			middleware.GetLogger(c).WithFields(map[string]interface{}{
				"ip":       ip,
				"endpoint": c.Request.URL.Path,
				"method":   c.Request.Method,
			}).Warn("Rate limit exceeded")
			logging.LogRateLimitExceeded(ctx, ip, c.Request.URL.Path)
			c.Header("Retry-After", strconv.Itoa(int(math.Ceil(longestWait.Seconds()))))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "Too many requests. Please try again later.",
			})
			return
		}
		c.Next()
	}
}

// extractField returns body's JSON field named name as a trimmed, lower-cased
// string, or "" when it is missing, not a string, or the body doesn't parse.
//
// It decodes into a struct with a single field tagged `json:"name"`, rather
// than doing an exact-key lookup against map[string]json.RawMessage, because
// handlers bind the same body into a struct and encoding/json matches struct
// fields case-insensitively, with the last occurrence of a duplicate key
// winning. An exact-match map lookup let both of those slip past the
// per-account check: {"USERNAME":"victim"} matched no "username" key at all,
// and {"username":"decoy","Username":"victim"} charged the decoy instead of
// the account the handler actually resolves to. Decoding with the same
// semantics the handler uses closes both bypasses.
func extractField(body []byte, name string) string {
	fieldType := reflect.StructOf([]reflect.StructField{
		{
			Name: "Value",
			Type: reflect.TypeOf(""),
			Tag:  reflect.StructTag(fmt.Sprintf(`json:%q`, name)),
		},
	})
	target := reflect.New(fieldType)
	if err := json.Unmarshal(body, target.Interface()); err != nil {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(target.Elem().Field(0).String()))
}
