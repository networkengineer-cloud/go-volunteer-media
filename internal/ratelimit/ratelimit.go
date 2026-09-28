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
	"fmt"
	"io"
	"math"
	"net/http"
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

		var fields map[string]json.RawMessage
		if needsBody && c.Request.Body != nil {
			body, err := io.ReadAll(c.Request.Body)
			if err != nil {
				// Most likely the body-size cap; let the handler report it.
				c.Request.Body = io.NopCloser(bytes.NewReader(body))
				c.Next()
				return
			}
			c.Request.Body = io.NopCloser(bytes.NewReader(body))
			_ = json.Unmarshal(body, &fields)
		}

		var longestWait time.Duration
		limited := false
		for _, ch := range checks {
			subject := ip
			if ch.Field != "" {
				value := stringField(fields, ch.Field)
				if value == "" {
					continue
				}
				subject = value + "|" + ip
			}
			allowed, wait, err := l.Hit(ctx, ch.Rule, subject)
			if err != nil {
				middleware.GetLogger(c).Error("Rate limit check failed; allowing request", err)
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

// stringField returns fields[name] as a trimmed, lower-cased string, or ""
// when it is missing or not a string.
func stringField(fields map[string]json.RawMessage, name string) string {
	raw, ok := fields[name]
	if !ok {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(s))
}
